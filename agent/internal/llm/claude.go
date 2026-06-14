// Package llm — Claude (Anthropic) provider for the agent binary.
//
// Mirrors backend/internal/platform/llm/claude.go. The Anthropic Messages API
// wire format is independent of the OpenAI-compatible providers:
//   - Endpoint:  POST {baseURL}/v1/messages
//   - Auth:      x-api-key header (not Authorization: Bearer)
//   - Required:  anthropic-version: 2023-06-01
//   - Response:  content[].text  (not choices[].message.content)
//   - Streaming: content_block_delta events
//
// GenerateStream enables token-by-token output during agent iteration passes —
// the same technique used for generation output with other streaming providers.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	agentDefaultAnthropicBaseURL = "https://api.anthropic.com"
	agentAnthropicVersion        = "2023-06-01"
	agentClaudeMaxTokens         = 8192
)

func init() {
	Register(ProviderClaude, newClaudeProvider)
}

func newClaudeProvider(cfg LLMConfig, kr func(string) (string, error)) (LLMProvider, error) {
	return &claudeProvider{
		baseURL:       agentDefaultAnthropicBaseURL,
		model:         cfg.Model,
		credentialRef: cfg.CredentialRef,
		keyResolver:   kr,
		httpClient:    &http.Client{Timeout: agentDefaultHTTPTimeout},
	}, nil
}

// claudeProvider calls the Anthropic Messages API.
type claudeProvider struct {
	baseURL       string
	model         string
	credentialRef string
	keyResolver   func(string) (string, error)
	httpClient    *http.Client
}

// Generate calls POST {baseURL}/v1/messages and returns the full response.
func (p *claudeProvider) Generate(ctx context.Context, req LLMRequest) (LLMResponse, error) {
	apiKey, err := p.keyResolver(p.credentialRef)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("claude.Generate: %w", err)
	}

	body, err := json.Marshal(agentClaudeRequest{
		Model:     p.model,
		MaxTokens: agentClaudeMaxTokens,
		System:    req.SystemPrompt,
		Messages: []agentClaudeMessage{
			{Role: "user", Content: req.UserMessage},
		},
		Temperature: req.Temperature,
		Stream:      false,
	})
	if err != nil {
		return LLMResponse{}, fmt.Errorf("claude.Generate: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("claude.Generate: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", agentAnthropicVersion)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("claude.Generate: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, agentMaxResponseBytes))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("claude.Generate: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return LLMResponse{}, fmt.Errorf("claude.Generate: HTTP %d: %s", resp.StatusCode, respBody)
	}

	var result agentClaudeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return LLMResponse{}, fmt.Errorf("claude.Generate: parse response: %w", err)
	}

	text := ""
	for _, block := range result.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}
	if text == "" {
		return LLMResponse{}, fmt.Errorf("claude.Generate: no text content block in response")
	}

	return LLMResponse{
		Content:      text,
		FinishReason: result.StopReason,
		TokensUsed:   result.Usage.InputTokens + result.Usage.OutputTokens,
	}, nil
}

// GenerateStream calls POST {baseURL}/v1/messages with stream:true.
// content_block_delta events produce TokenChunk values; message_stop closes
// the channel — enabling LLM-like streaming during agent iteration passes.
func (p *claudeProvider) GenerateStream(ctx context.Context, req LLMRequest) (<-chan TokenChunk, error) {
	apiKey, err := p.keyResolver(p.credentialRef)
	if err != nil {
		return nil, fmt.Errorf("claude.GenerateStream: %w", err)
	}

	body, err := json.Marshal(agentClaudeRequest{
		Model:     p.model,
		MaxTokens: agentClaudeMaxTokens,
		System:    req.SystemPrompt,
		Messages: []agentClaudeMessage{
			{Role: "user", Content: req.UserMessage},
		},
		Temperature: req.Temperature,
		Stream:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("claude.GenerateStream: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("claude.GenerateStream: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", agentAnthropicVersion)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("claude.GenerateStream: http: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("claude.GenerateStream: HTTP %d: %s", resp.StatusCode, errBody)
	}

	ch := make(chan TokenChunk, 64)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		p.readClaudeSSEStream(ctx, resp.Body, ch)
	}()
	return ch, nil
}

func (p *claudeProvider) readClaudeSSEStream(ctx context.Context, body io.Reader, ch chan<- TokenChunk) {
	const sseBufSize = 512 * 1024
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, sseBufSize), sseBufSize)
	var eventType string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			sendChunk(ctx, ch, TokenChunk{Err: ctx.Err(), Done: true})
			return
		default:
		}

		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		switch eventType {
		case "content_block_delta":
			var delta agentClaudeStreamDelta
			if err := json.Unmarshal([]byte(data), &delta); err != nil {
				continue
			}
			if delta.Delta.Type == "text_delta" && delta.Delta.Text != "" {
				sendChunk(ctx, ch, TokenChunk{Text: delta.Delta.Text})
			}
		case "message_stop":
			sendChunk(ctx, ch, TokenChunk{Done: true})
			return
		}
	}

	if err := scanner.Err(); err != nil {
		sendChunk(ctx, ch, TokenChunk{Err: fmt.Errorf("claude stream scan: %w", err), Done: true})
	}
}

// ── Wire types ────────────────────────────────────────────────────────────────

type agentClaudeRequest struct {
	Model       string               `json:"model"`
	MaxTokens   int                  `json:"max_tokens"`
	System      string               `json:"system,omitempty"`
	Messages    []agentClaudeMessage `json:"messages"`
	Temperature float64              `json:"temperature"`
	Stream      bool                 `json:"stream"`
}

type agentClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type agentClaudeResponse struct {
	Content    []agentClaudeContentBlock `json:"content"`
	StopReason string                    `json:"stop_reason"`
	Usage      agentClaudeUsage          `json:"usage"`
}

type agentClaudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type agentClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type agentClaudeStreamDelta struct {
	Delta agentClaudeStreamDeltaContent `json:"delta"`
}

type agentClaudeStreamDeltaContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

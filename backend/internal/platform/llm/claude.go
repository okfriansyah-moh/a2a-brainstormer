// Package llm — Claude (Anthropic) provider.
//
// claudeProvider uses the Anthropic Messages API wire format, which is
// incompatible with the OpenAI-compatible providers:
//   - Endpoint:  POST {baseURL}/v1/messages   (not /chat/completions)
//   - Auth:      x-api-key header             (not Authorization: Bearer)
//   - Required:  anthropic-version: 2023-06-01
//   - Response:  content[].text               (not choices[].message.content)
//   - Streaming: content_block_delta events   (not choices[].delta.content)
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
	defaultAnthropicBaseURL  = "https://api.anthropic.com"
	anthropicVersion         = "2023-06-01"
	claudeMaxTokens          = 8192
)

func init() {
	Register(ProviderClaude, newClaudeProvider)
}

func newClaudeProvider(cfg LLMConfig, kr func(string) (string, error)) (LLMProvider, error) {
	return &claudeProvider{
		baseURL:       defaultAnthropicBaseURL,
		model:         cfg.Model,
		credentialRef: cfg.CredentialRef,
		keyResolver:   kr,
		httpClient:    &http.Client{Timeout: defaultHTTPTimeout},
	}, nil
}

// claudeProvider calls the Anthropic Messages API.
// Credentials are resolved at call time — never stored or logged.
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

	body, err := json.Marshal(claudeRequest{
		Model:     p.model,
		MaxTokens: claudeMaxTokens,
		System:    req.SystemPrompt,
		Messages: []claudeMessage{
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
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("claude.Generate: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("claude.Generate: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return LLMResponse{}, fmt.Errorf("claude.Generate: HTTP %d: %s", resp.StatusCode, respBody)
	}

	var result claudeResponse
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
// SSE `content_block_delta` events with `delta.type=="text_delta"` produce
// TokenChunk values; `message_stop` closes the channel with Done:true.
// This gives Claude the same token-by-token streaming during agent iteration
// passes as the OpenAI-compatible providers.
func (p *claudeProvider) GenerateStream(ctx context.Context, req LLMRequest) (<-chan TokenChunk, error) {
	apiKey, err := p.keyResolver(p.credentialRef)
	if err != nil {
		return nil, fmt.Errorf("claude.GenerateStream: %w", err)
	}

	body, err := json.Marshal(claudeRequest{
		Model:     p.model,
		MaxTokens: claudeMaxTokens,
		System:    req.SystemPrompt,
		Messages: []claudeMessage{
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
	httpReq.Header.Set("anthropic-version", anthropicVersion)
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

// readClaudeSSEStream parses Anthropic SSE events.
// content_block_delta with delta.type=="text_delta" → emit text.
// message_stop → emit Done:true and return.
func (p *claudeProvider) readClaudeSSEStream(ctx context.Context, body io.Reader, ch chan<- TokenChunk) {
	scanner := bufio.NewScanner(body)
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
			var delta claudeStreamDelta
			if err := json.Unmarshal([]byte(data), &delta); err != nil {
				continue
			}
			if delta.Delta.Type == "text_delta" && delta.Delta.Text != "" {
				if !sendChunk(ctx, ch, TokenChunk{Text: delta.Delta.Text}) {
					return
				}
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

type claudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system,omitempty"`
	Messages    []claudeMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	Stream      bool            `json:"stream"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content    []claudeContentBlock `json:"content"`
	StopReason string               `json:"stop_reason"`
	Usage      claudeUsage          `json:"usage"`
}

type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type claudeStreamDelta struct {
	Delta claudeStreamDeltaContent `json:"delta"`
}

type claudeStreamDeltaContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

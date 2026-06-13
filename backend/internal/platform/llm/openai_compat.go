// Package llm — OpenAI-compatible provider base.
//
// openAICompatProvider is the shared HTTP implementation for all providers that
// use the OpenAI chat completions wire format: Copilot, OpenAI, and DeepSeek.
// It implements both LLMProvider (blocking) and StreamingLLMProvider (SSE).
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
	"time"
)

const (
	defaultHTTPTimeout = 10 * time.Minute
	maxResponseBytes   = 8 * 1024 * 1024 // 8 MiB
)

// openAICompatProvider posts to {baseURL}/chat/completions using the OpenAI
// wire format. Credentials are resolved at call time via keyResolver.
type openAICompatProvider struct {
	baseURL       string
	model         string
	credentialRef string
	keyResolver   func(string) (string, error)
	httpClient    *http.Client
}

func newOpenAICompatProvider(
	baseURL, model, credRef string,
	httpClient *http.Client,
	kr func(string) (string, error),
) *openAICompatProvider {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &openAICompatProvider{
		baseURL:       baseURL,
		model:         model,
		credentialRef: credRef,
		keyResolver:   kr,
		httpClient:    httpClient,
	}
}

// Generate calls {baseURL}/chat/completions (non-streaming) and returns the
// full LLM response. The API key is resolved from credentialRef at call time.
func (p *openAICompatProvider) Generate(ctx context.Context, req LLMRequest) (LLMResponse, error) {
	apiKey, err := p.keyResolver(p.credentialRef)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("openai_compat.Generate: %w", err)
	}

	body, err := json.Marshal(openAIRequest{
		Model: p.model,
		Messages: []openAIMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserMessage},
		},
		Temperature:    req.Temperature,
		MaxTokens:      8192,
		Stream:         false,
		ResponseFormat: openAIResponseFormatFor(req.ResponseFormat),
	})
	if err != nil {
		return LLMResponse{}, fmt.Errorf("openai_compat.Generate: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("openai_compat.Generate: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("openai_compat.Generate: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("openai_compat.Generate: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return LLMResponse{}, fmt.Errorf("openai_compat.Generate: HTTP %d: %s", resp.StatusCode, respBody)
	}

	var result openAIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return LLMResponse{}, fmt.Errorf("openai_compat.Generate: parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return LLMResponse{}, fmt.Errorf("openai_compat.Generate: API returned zero choices")
	}

	tokensUsed := result.Usage.PromptTokens + result.Usage.CompletionTokens
	if tokensUsed == 0 {
		tokensUsed = result.Usage.TotalTokens
	}

	return LLMResponse{
		Content:      result.Choices[0].Message.Content,
		FinishReason: result.Choices[0].FinishReason,
		TokensUsed:   tokensUsed,
	}, nil
}

// GenerateStream calls {baseURL}/chat/completions with stream:true and returns
// a channel of token chunks. The channel is closed after the terminal Done
// chunk or when ctx is cancelled.
func (p *openAICompatProvider) GenerateStream(ctx context.Context, req LLMRequest) (<-chan TokenChunk, error) {
	apiKey, err := p.keyResolver(p.credentialRef)
	if err != nil {
		return nil, fmt.Errorf("openai_compat.GenerateStream: %w", err)
	}

	body, err := json.Marshal(openAIRequest{
		Model: p.model,
		Messages: []openAIMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserMessage},
		},
		Temperature: req.Temperature,
		MaxTokens:   8192,
		Stream:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("openai_compat.GenerateStream: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai_compat.GenerateStream: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai_compat.GenerateStream: http: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("openai_compat.GenerateStream: HTTP %d: %s", resp.StatusCode, errBody)
	}

	ch := make(chan TokenChunk, 64)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		p.readSSEStream(ctx, resp.Body, ch)
	}()
	return ch, nil
}

// readSSEStream parses `data: {json}` lines from the SSE body.
func (p *openAICompatProvider) readSSEStream(ctx context.Context, body io.Reader, ch chan<- TokenChunk) {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			sendChunk(ctx, ch, TokenChunk{Err: ctx.Err(), Done: true})
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			sendChunk(ctx, ch, TokenChunk{Done: true})
			return
		}

		var delta openAIStreamDelta
		if err := json.Unmarshal([]byte(payload), &delta); err != nil {
			continue
		}
		if len(delta.Choices) == 0 {
			continue
		}
		text := delta.Choices[0].Delta.Content
		if text != "" {
			if !sendChunk(ctx, ch, TokenChunk{Text: text}) {
				return
			}
		}
	}
	if err := scanner.Err(); err != nil {
		sendChunk(ctx, ch, TokenChunk{Err: fmt.Errorf("openai_compat stream scan: %w", err), Done: true})
	}
}

// ── Wire types ────────────────────────────────────────────────────────────────

type openAIRequest struct {
	Model          string                `json:"model"`
	Messages       []openAIMessage       `json:"messages"`
	Temperature    float64               `json:"temperature"`
	MaxTokens      int                   `json:"max_tokens,omitempty"`
	Stream         bool                  `json:"stream"`
	ResponseFormat *openAIResponseFormat `json:"response_format,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseFormat struct {
	Type string `json:"type"`
}

func openAIResponseFormatFor(hint string) *openAIResponseFormat {
	if hint == "json_object" {
		return &openAIResponseFormat{Type: "json_object"}
	}
	return nil
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIStreamDelta struct {
	Choices []openAIStreamChoice `json:"choices"`
}

type openAIStreamChoice struct {
	Delta openAIStreamDeltaContent `json:"delta"`
}

type openAIStreamDeltaContent struct {
	Content string `json:"content"`
}

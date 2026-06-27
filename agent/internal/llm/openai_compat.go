// Package llm — OpenAI-compatible provider base for the agent binary.
//
// Mirrors backend/internal/platform/llm/openai_compat.go. Implements both
// LLMProvider (blocking) and StreamingLLMProvider (SSE) so the agent executor
// receives token-by-token output during iteration passes — the same streaming
// technique used for generation output.
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
	agentDefaultHTTPTimeout = 10 * time.Minute
	agentMaxResponseBytes   = 8 * 1024 * 1024 // 8 MiB
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
		httpClient = &http.Client{Timeout: agentDefaultHTTPTimeout}
	}
	return &openAICompatProvider{
		baseURL:       baseURL,
		model:         model,
		credentialRef: credRef,
		keyResolver:   kr,
		httpClient:    httpClient,
	}
}

// Generate calls {baseURL}/chat/completions (blocking) and returns the full response.
func (p *openAICompatProvider) Generate(ctx context.Context, req LLMRequest) (LLMResponse, error) {
	apiKey, err := p.keyResolver(p.credentialRef)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("openai_compat.Generate: %w", err)
	}

	body, err := json.Marshal(p.buildOpenAIRequest(req, false))
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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, agentMaxResponseBytes))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("openai_compat.Generate: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return LLMResponse{}, fmt.Errorf("openai_compat.Generate: HTTP %d: %s", resp.StatusCode, respBody)
	}

	var result agentOpenAIResponse
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
		Content:          result.Choices[0].Message.Content,
		FinishReason:     result.Choices[0].FinishReason,
		TokensUsed:       tokensUsed,
		CacheReadTokens:  result.Usage.PromptTokensDetails.CachedTokens,
	}, nil
}

// GenerateStream calls {baseURL}/chat/completions with stream:true.
// Each SSE token is forwarded as a TokenChunk so the executor can emit it as
// a Working status update, producing LLM-like progress during agent iterations.
func (p *openAICompatProvider) GenerateStream(ctx context.Context, req LLMRequest) (<-chan TokenChunk, error) {
	apiKey, err := p.keyResolver(p.credentialRef)
	if err != nil {
		return nil, fmt.Errorf("openai_compat.GenerateStream: %w", err)
	}

	body, err := json.Marshal(p.buildOpenAIRequest(req, true))
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

func (p *openAICompatProvider) readSSEStream(ctx context.Context, body io.Reader, ch chan<- TokenChunk) {
	const sseBufSize = 512 * 1024
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, sseBufSize), sseBufSize)
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

		var delta agentOpenAIStreamDelta
		if err := json.Unmarshal([]byte(payload), &delta); err != nil {
			continue
		}
		if len(delta.Choices) == 0 {
			continue
		}
		text := delta.Choices[0].Delta.Content
		if text != "" {
			sendChunk(ctx, ch, TokenChunk{Text: text})
		}
		if reason := delta.Choices[0].FinishReason; reason != "" {
			sendChunk(ctx, ch, TokenChunk{Done: true, FinishReason: reason})
			return
		}
	}
	if err := scanner.Err(); err != nil {
		sendChunk(ctx, ch, TokenChunk{Err: fmt.Errorf("openai_compat stream scan: %w", err), Done: true})
	}
}

// ── Wire types ────────────────────────────────────────────────────────────────

func (p *openAICompatProvider) buildOpenAIRequest(req LLMRequest, stream bool) agentOpenAIRequest {
	blocks := WireMessages(req)
	msgs := make([]agentOpenAIMessage, 0, len(blocks))
	for _, b := range blocks {
		msgs = append(msgs, agentOpenAIMessage{Role: b.Role, Content: b.Content})
	}
	if len(msgs) == 0 {
		sys, user := ResolvePromptFields(req)
		if sys != "" {
			msgs = append(msgs, agentOpenAIMessage{Role: "system", Content: sys})
		}
		if user != "" {
			msgs = append(msgs, agentOpenAIMessage{Role: "user", Content: user})
		}
	}
	out := agentOpenAIRequest{
		Model:       p.model,
		Messages:    msgs,
		Temperature: req.Temperature,
		MaxTokens:   8192,
		Stream:      stream,
	}
	if key := PromptCacheKey(req); key != "" {
		out.PromptCacheKey = key
	}
	return out
}

type agentOpenAIRequest struct {
	Model          string                `json:"model"`
	Messages       []agentOpenAIMessage  `json:"messages"`
	Temperature    float64               `json:"temperature"`
	MaxTokens      int                   `json:"max_tokens,omitempty"`
	Stream         bool                  `json:"stream"`
	PromptCacheKey string                `json:"prompt_cache_key,omitempty"`
}

type agentOpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type agentOpenAIResponse struct {
	Choices []agentOpenAIChoice `json:"choices"`
	Usage   agentOpenAIUsage    `json:"usage"`
}

type agentOpenAIChoice struct {
	Message      agentOpenAIMessage `json:"message"`
	FinishReason string             `json:"finish_reason"`
}

type agentOpenAIUsage struct {
	PromptTokens        int                        `json:"prompt_tokens"`
	CompletionTokens    int                        `json:"completion_tokens"`
	TotalTokens         int                        `json:"total_tokens"`
	PromptTokensDetails agentOpenAIPromptTokenDetails `json:"prompt_tokens_details"`
}

type agentOpenAIPromptTokenDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type agentOpenAIStreamDelta struct {
	Choices []agentOpenAIStreamChoice `json:"choices"`
}

type agentOpenAIStreamChoice struct {
	Delta        agentOpenAIStreamDeltaContent `json:"delta"`
	FinishReason string                      `json:"finish_reason"`
}

type agentOpenAIStreamDeltaContent struct {
	Content string `json:"content"`
}

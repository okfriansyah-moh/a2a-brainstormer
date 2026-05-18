// Package llm provides the LLMProvider interface and the CopilotProvider
// implementation for the agent binary. The interface shape mirrors
// backend/internal/platform/llm but is declared independently — the agent
// binary must not import any backend package.
//
// Security invariant: os.Getenv is never called here. The API key is resolved
// via the resolveKey function injected at construction time, which must be
// config.GetLLMAPIKey from agent/internal/config. This keeps all env reads
// confined to config/config.go.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMProvider is the agent binary's abstraction over LLM backends.
// All LLM calls go through this interface; implementations must never call
// Copilot/Claude SDKs directly from business logic.
type LLMProvider interface {
	Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)
}

// LLMRequest contains everything a provider needs to make an LLM call.
type LLMRequest struct {
	SystemPrompt string
	UserMessage  string
	Temperature  float64
}

// LLMResponse contains the raw output from an LLM call.
type LLMResponse struct {
	Content      string
	FinishReason string
	TokensUsed   int
}

// DefaultCopilotEndpoint is the GitHub Copilot OpenAI-compatible chat
// completions URL.
const DefaultCopilotEndpoint = "https://api.githubcopilot.com/chat/completions"

const (
	defaultHTTPTimeout = 90 * time.Second
	maxResponseBytes   = 1 << 20 // 1 MiB
)

// CopilotProvider implements LLMProvider using the GitHub Copilot REST API.
// The API key is resolved at each Generate call via resolveKey — never stored
// on the struct or logged.
type CopilotProvider struct {
	model         string
	credentialRef string
	endpoint      string
	client        *http.Client
	resolveKey    func(credentialRef string) (string, error)
}

// NewCopilotProvider constructs a CopilotProvider.
//
//   - model: LLM model name (e.g. "gpt-4o").
//   - credentialRef: env var name holding the API key (e.g. "COPILOT_API_KEY").
//   - endpoint: Copilot API URL; defaults to DefaultCopilotEndpoint when empty.
//   - httpClient: reusable HTTP client; a default 90s-timeout client is used when nil.
//   - resolveKey: must be config.GetLLMAPIKey — keeps os.Getenv confined to config.go.
func NewCopilotProvider(
	model, credentialRef, endpoint string,
	httpClient *http.Client,
	resolveKey func(string) (string, error),
) *CopilotProvider {
	if endpoint == "" {
		endpoint = DefaultCopilotEndpoint
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &CopilotProvider{
		model:         model,
		credentialRef: credentialRef,
		endpoint:      endpoint,
		client:        httpClient,
		resolveKey:    resolveKey,
	}
}

// Generate calls the Copilot chat completions API and returns the LLM output.
// The API key is resolved from credentialRef at call time — never cached.
func (p *CopilotProvider) Generate(ctx context.Context, req LLMRequest) (LLMResponse, error) {
	// Resolve credential at call time; error if absent (no silent fallback).
	apiKey, err := p.resolveKey(p.credentialRef)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: resolve key: %w", err)
	}

	body, err := json.Marshal(copilotRequest{
		Model: p.model,
		Messages: []copilotMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserMessage},
		},
		Temperature:    req.Temperature,
		ResponseFormat: &copilotResponseFormat{Type: "json_object"},
	})
	if err != nil {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey) // key used here only; never persisted

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Error body is safe to include; it does not contain the API key.
		return LLMResponse{}, fmt.Errorf("copilot.Generate: API returned HTTP %d: %s", resp.StatusCode, respBody)
	}

	var result copilotResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: API returned zero choices")
	}

	return LLMResponse{
		Content:      result.Choices[0].Message.Content,
		FinishReason: result.Choices[0].FinishReason,
		TokensUsed:   result.Usage.TotalTokens,
	}, nil
}

// ── OpenAI-compatible wire types ──────────────────────────────────────────────

type copilotRequest struct {
	Model          string                 `json:"model"`
	Messages       []copilotMessage       `json:"messages"`
	Temperature    float64                `json:"temperature"`
	ResponseFormat *copilotResponseFormat `json:"response_format,omitempty"`
}

type copilotMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// copilotResponseFormat requests structured JSON output from the model.
type copilotResponseFormat struct {
	Type string `json:"type"` // "json_object"
}

type copilotResponse struct {
	Choices []copilotChoice `json:"choices"`
	Usage   copilotUsage    `json:"usage"`
}

type copilotChoice struct {
	Message      copilotMessage `json:"message"`
	FinishReason string         `json:"finish_reason"`
}

type copilotUsage struct {
	TotalTokens int `json:"total_tokens"`
}

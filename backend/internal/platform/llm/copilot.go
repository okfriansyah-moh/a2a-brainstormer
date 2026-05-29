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

// DefaultCopilotEndpoint is the GitHub Copilot OpenAI-compatible chat
// completions URL. Pass an alternative to NewCopilotProvider when using a
// proxy, local model, or test server.
const DefaultCopilotEndpoint = "https://api.githubcopilot.com/chat/completions"

// defaultHTTPTimeout guards against hanging LLM calls.
// Long-form document generation (≥1000 lines of Markdown) routinely
// takes 3–8 minutes on `gpt-4.1`, so the timeout is set well above the
// agent-side default. Override per-call via context if a tighter bound
// is needed.
const defaultHTTPTimeout = 10 * time.Minute

// maxResponseBytes limits how many bytes we read from the LLM API response
// body to prevent unbounded memory growth. 8 MiB comfortably covers a
// ≥1000-line Markdown document (~80 chars/line on average).
const maxResponseBytes = 8 << 20 // 8 MiB

// CopilotProvider implements LLMProvider using the GitHub Copilot
// OpenAI-compatible REST API. It never imports the Copilot SDK directly —
// all communication happens over net/http.
//
// Security: the API key is resolved from cfg.CredentialRef (an env var name)
// at each Generate call via ResolveKey. It is never stored on the struct or
// logged anywhere.
type CopilotProvider struct {
	cfg      LLMConfig
	endpoint string
	client   *http.Client
}

// NewCopilotProvider creates a CopilotProvider.
//   - endpoint: Copilot API URL; defaults to DefaultCopilotEndpoint when empty.
//   - httpClient: reusable HTTP client; a default with 90 s timeout is used when nil.
func NewCopilotProvider(cfg LLMConfig, endpoint string, httpClient *http.Client) *CopilotProvider {
	if endpoint == "" {
		endpoint = DefaultCopilotEndpoint
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &CopilotProvider{cfg: cfg, endpoint: endpoint, client: httpClient}
}

// Generate calls the Copilot chat completions API and returns the LLM output.
// The API key is resolved from CredentialRef at call time — never cached.
func (p *CopilotProvider) Generate(ctx context.Context, req LLMRequest) (LLMResponse, error) {
	// Resolve credential at call time; error if absent (no silent fallback).
	apiKey, err := ResolveKey(p.cfg.CredentialRef)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: %w", err)
	}

	body, err := json.Marshal(copilotRequest{
		Model: p.cfg.Model,
		Messages: []copilotMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserMessage},
		},
		Temperature:    req.Temperature,
		ResponseFormat: responseFormatFor(req.ResponseFormat),
	})
	if err != nil {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey) // key used here, never persisted

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("copilot.Generate: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Error body is safe to include; it does not contain our API key.
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

// responseFormatFor converts the provider-agnostic LLMRequest.ResponseFormat
// hint into the Copilot wire field. Empty and "text" produce free-form output
// (no response_format header sent — required for Markdown / prose tasks).
// Only "json_object" enables structured JSON mode.
func responseFormatFor(hint string) *copilotResponseFormat {
	if hint == "json_object" {
		return &copilotResponseFormat{Type: "json_object"}
	}
	return nil
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

// Package llm provides the LLMProvider abstraction for the agent binary.
// All LLM calls must go through the LLMProvider interface — no provider SDK
// may be imported directly from business logic.
//
// Security invariant: os.Getenv is never called here. The API key is resolved
// via the resolveKey function injected at construction time, which must be
// config.GetLLMAPIKey from agent/internal/config. This keeps all env reads
// confined to config/config.go.
package llm

import (
	"context"
	"net/http"
)

// LLMProvider is the agent binary's abstraction over LLM backends.
type LLMProvider interface {
	Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)
}

// LLMRequest contains everything a provider needs to make an LLM call.
type LLMRequest struct {
	SystemPrompt string
	UserMessage  string
	Temperature  float64
	// Tiered holds cache-aware prompt blocks. When nil, SystemPrompt and
	// UserMessage are used directly.
	Tiered *TieredPrompt
}

// LLMResponse contains the raw output from an LLM call.
type LLMResponse struct {
	Content          string
	FinishReason     string
	TokensUsed       int
	CacheReadTokens  int
	CacheWriteTokens int
}

// DefaultCopilotBaseURL is the GitHub Copilot OpenAI-compatible base URL.
const DefaultCopilotBaseURL = "https://api.githubcopilot.com"

// DefaultCopilotEndpoint is kept for backward compatibility.
const DefaultCopilotEndpoint = DefaultCopilotBaseURL + "/chat/completions"

func init() {
	Register(ProviderCopilot, newCopilotProviderFromConfig)
}

func newCopilotProviderFromConfig(cfg LLMConfig, kr func(string) (string, error)) (LLMProvider, error) {
	return &CopilotProvider{
		openAICompatProvider: newOpenAICompatProvider(DefaultCopilotBaseURL, cfg.Model, cfg.CredentialRef, nil, kr),
	}, nil
}

// CopilotProvider implements LLMProvider using the GitHub Copilot REST API.
// It delegates all HTTP logic to openAICompatProvider which also implements
// StreamingLLMProvider — enabling token-by-token output during agent iterations.
type CopilotProvider struct {
	*openAICompatProvider
}

// NewCopilotProvider constructs a CopilotProvider.
//
//   - model: LLM model name (e.g. "gpt-4o").
//   - credentialRef: env var name holding the API key (e.g. "COPILOT_API_KEY").
//   - endpoint: Copilot API base URL; defaults to DefaultCopilotBaseURL when empty.
//   - httpClient: reusable HTTP client; a default 10-min timeout is used when nil.
//   - resolveKey: must be config.GetLLMAPIKey — keeps os.Getenv confined to config.go.
func NewCopilotProvider(
	model, credentialRef, endpoint string,
	httpClient *http.Client,
	resolveKey func(string) (string, error),
) *CopilotProvider {
	baseURL := endpoint
	if baseURL == "" {
		baseURL = DefaultCopilotBaseURL
	}
	return &CopilotProvider{
		openAICompatProvider: newOpenAICompatProvider(baseURL, model, credentialRef, httpClient, resolveKey),
	}
}

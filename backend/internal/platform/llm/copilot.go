package llm

import "net/http"

// DefaultCopilotEndpoint is the GitHub Copilot OpenAI-compatible base URL.
// Kept for backward compatibility — new code should use the registry via llm.New().
const DefaultCopilotEndpoint = "https://api.githubcopilot.com"

func init() {
	Register(ProviderCopilot, newCopilotProviderFromConfig)
}

func newCopilotProviderFromConfig(cfg LLMConfig, kr func(string) (string, error)) (LLMProvider, error) {
	return &CopilotProvider{
		openAICompatProvider: newOpenAICompatProvider(DefaultCopilotEndpoint, cfg.Model, cfg.CredentialRef, nil, kr),
	}, nil
}

// CopilotProvider implements LLMProvider using the GitHub Copilot
// OpenAI-compatible REST API. It delegates all HTTP logic to openAICompatProvider.
//
// Security: the API key is resolved from cfg.CredentialRef (an env var name)
// at each Generate call via ResolveKey. It is never stored on the struct or
// logged anywhere.
type CopilotProvider struct {
	*openAICompatProvider
}

// NewCopilotProvider creates a CopilotProvider.
//   - endpoint: Copilot API base URL; defaults to DefaultCopilotEndpoint when empty.
//   - httpClient: reusable HTTP client; a default with 10 min timeout is used when nil.
func NewCopilotProvider(cfg LLMConfig, endpoint string, httpClient *http.Client) *CopilotProvider {
	baseURL := endpoint
	if baseURL == "" {
		baseURL = DefaultCopilotEndpoint
	}
	return &CopilotProvider{
		openAICompatProvider: newOpenAICompatProvider(baseURL, cfg.Model, cfg.CredentialRef, httpClient, ResolveKey),
	}
}

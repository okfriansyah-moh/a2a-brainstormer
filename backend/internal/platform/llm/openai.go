package llm

import "net/http"

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

func init() {
	Register(ProviderOpenAI, newOpenAIProvider)
}

func newOpenAIProvider(cfg LLMConfig, kr func(string) (string, error)) (LLMProvider, error) {
	return newOpenAICompatProvider(defaultOpenAIBaseURL, cfg.Model, cfg.CredentialRef, (*http.Client)(nil), kr), nil
}

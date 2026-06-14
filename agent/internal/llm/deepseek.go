package llm

import "net/http"

const defaultDeepSeekBaseURL = "https://api.deepseek.com"

func init() {
	Register(ProviderDeepSeek, newDeepSeekProvider)
}

func newDeepSeekProvider(cfg LLMConfig, kr func(string) (string, error)) (LLMProvider, error) {
	return newOpenAICompatProvider(defaultDeepSeekBaseURL, cfg.Model, cfg.CredentialRef, (*http.Client)(nil), kr), nil
}

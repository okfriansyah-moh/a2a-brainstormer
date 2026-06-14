// Package llm — provider registry for the agent binary.
//
// Mirrors backend/internal/platform/llm/registry.go. The agent binary is a
// separate Go module and must not import any backend package — this is an
// independent copy of the same registry contract.
package llm

import (
	"fmt"
	"strings"
	"sync"
)

// LLMConfig carries provider, model, and credential reference for one LLM dispatch.
// CredentialRef holds the env var NAME only — never the raw key value.
type LLMConfig struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	CredentialRef string `json:"credential_ref"`
}

// ProviderKind identifies an LLM provider implementation.
type ProviderKind string

const (
	ProviderCopilot  ProviderKind = "copilot"
	ProviderOpenCode ProviderKind = "opencode"
	ProviderOpenAI   ProviderKind = "openai"
	ProviderClaude   ProviderKind = "claude"
	ProviderDeepSeek ProviderKind = "deepseek"
)

// AllProviderKinds is the canonical ordered list of provider identifiers.
var AllProviderKinds = []ProviderKind{
	ProviderCopilot, ProviderOpenCode, ProviderOpenAI, ProviderClaude, ProviderDeepSeek,
}

// ProviderFactory constructs an LLMProvider from config.
type ProviderFactory func(cfg LLMConfig, keyResolver func(string) (string, error)) (LLMProvider, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[ProviderKind]ProviderFactory)
)

// Register adds a factory to the global registry.
// Called from init() in each provider file.
func Register(kind ProviderKind, f ProviderFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[kind] = f
}

// New instantiates the provider for the given config.
// Returns a descriptive error on unknown provider — never silently falls back.
func New(cfg LLMConfig, keyResolver func(string) (string, error)) (LLMProvider, error) {
	registryMu.RLock()
	f, ok := registry[ProviderKind(cfg.Provider)]
	registryMu.RUnlock()
	if !ok {
		kinds := make([]string, len(AllProviderKinds))
		for i, k := range AllProviderKinds {
			kinds[i] = string(k)
		}
		return nil, fmt.Errorf("unknown LLM provider %q — valid: %s", cfg.Provider, strings.Join(kinds, ", "))
	}
	return f(cfg, keyResolver)
}

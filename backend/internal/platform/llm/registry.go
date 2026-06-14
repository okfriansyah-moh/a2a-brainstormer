// Package llm — provider registry.
//
// Registry decouples provider selection from all call sites. Adding a new
// provider requires one new file with an init() registration — no changes to
// main.go, handlers, or business logic.
package llm

import (
	"fmt"
	"strings"
	"sync"
)

// ProviderKind identifies an LLM provider implementation.
type ProviderKind string

const (
	ProviderCopilot  ProviderKind = "copilot"
	ProviderOpenCode ProviderKind = "opencode"
	ProviderOpenAI   ProviderKind = "openai"
	ProviderClaude   ProviderKind = "claude"
	ProviderDeepSeek ProviderKind = "deepseek"
)

// AllProviderKinds is the canonical ordered list for API and UI enumeration.
var AllProviderKinds = []ProviderKind{
	ProviderCopilot, ProviderOpenCode, ProviderOpenAI, ProviderClaude, ProviderDeepSeek,
}

// ProviderFactory constructs an LLMProvider from a resolved config.
// keyResolver resolves a CredentialRef (env var name) to the actual key at
// call time — it must be config.GetLLMAPIKey to keep os.Getenv confined to
// the config package.
type ProviderFactory func(cfg LLMConfig, keyResolver func(string) (string, error)) (LLMProvider, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[ProviderKind]ProviderFactory)
)

// Register adds a factory to the global registry.
// Called from init() in each provider file — never from main.go or business logic.
func Register(kind ProviderKind, f ProviderFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[kind] = f
}

// New instantiates the provider for the given config.
// Returns a descriptive error listing valid kinds when cfg.Provider is unknown.
// Never silently falls back to a default provider.
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

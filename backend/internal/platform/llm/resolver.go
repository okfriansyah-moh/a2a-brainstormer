package llm

import (
	"a2a-brainstorm/backend/internal/platform/config"
)

// Resolve applies the tiered LLM config override (highest priority first):
//
//	session override → agent-level → global default
//
// Only non-zero fields from a higher-priority config overwrite lower-priority
// fields. This enables partial overrides — e.g. a session override that sets
// only Model will inherit Provider and CredentialRef from the agent-level or
// global config.
//
// All three pointers may be nil; a nil global produces a zero LLMConfig.
func Resolve(global, agentLevel, sessionOverride *LLMConfig) LLMConfig {
	if global == nil {
		global = &LLMConfig{}
	}
	result := *global

	if agentLevel != nil {
		if agentLevel.Provider != "" {
			result.Provider = agentLevel.Provider
		}
		if agentLevel.Model != "" {
			result.Model = agentLevel.Model
		}
		if agentLevel.CredentialRef != "" {
			result.CredentialRef = agentLevel.CredentialRef
		}
	}

	if sessionOverride != nil {
		if sessionOverride.Provider != "" {
			result.Provider = sessionOverride.Provider
		}
		if sessionOverride.Model != "" {
			result.Model = sessionOverride.Model
		}
		if sessionOverride.CredentialRef != "" {
			result.CredentialRef = sessionOverride.CredentialRef
		}
	}

	return result
}

// ResolveKey resolves a CredentialRef (env var name) to its actual API key
// at runtime. Returns an error if the env var is absent or empty — there is
// no silent fallback to another provider.
//
// All os.Getenv calls are delegated to config.GetLLMAPIKey to maintain the
// architecture invariant that os.Getenv is confined to
// backend/internal/platform/config/config.go.
func ResolveKey(credentialRef string) (string, error) {
	return config.GetLLMAPIKey(credentialRef)
}

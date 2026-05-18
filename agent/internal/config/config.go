// Package config centralises ALL os.Getenv calls for the agent binary.
// No other file in the agent binary may call os.Getenv directly.
// This invariant is enforced architecturally — any new env var access must be
// added here as a typed getter, never inline in business logic.
package config

import (
	"errors"
	"fmt"
	"os"
)

// ── Server ────────────────────────────────────────────────────────────────────

// GetPort returns the TCP port the agent HTTP server listens on.
// Reads AGENT_PORT; defaults to "9090" when unset.
func GetPort() string {
	if v := os.Getenv("AGENT_PORT"); v != "" {
		return v
	}
	return "9090"
}

// ── LLM Provider ──────────────────────────────────────────────────────────────

// GetLLMProvider returns the LLM provider identifier (e.g. "copilot", "claude").
// Reads AGENT_LLM_PROVIDER; defaults to "copilot" when unset.
func GetLLMProvider() string {
	if v := os.Getenv("AGENT_LLM_PROVIDER"); v != "" {
		return v
	}
	return "copilot"
}

// GetLLMModel returns the LLM model identifier (e.g. "gpt-4o", "claude-opus-4").
// Reads AGENT_LLM_MODEL; defaults to "gpt-4o" when unset.
func GetLLMModel() string {
	if v := os.Getenv("AGENT_LLM_MODEL"); v != "" {
		return v
	}
	return "gpt-4o"
}

// GetLLMCredentialRef returns the env var NAME that holds the LLM API key.
// Reads AGENT_LLM_CREDENTIAL_REF; defaults to "COPILOT_API_KEY" when unset.
//
// Security invariant: this function returns the env var *name*, never its value.
// To resolve the actual key call GetLLMAPIKey(GetLLMCredentialRef()).
func GetLLMCredentialRef() string {
	if v := os.Getenv("AGENT_LLM_CREDENTIAL_REF"); v != "" {
		return v
	}
	return "COPILOT_API_KEY"
}

// GetLLMAPIKey resolves the actual API key stored in the env var whose name is
// credentialRef. Returns an error if the env var is absent or empty.
//
// Security rules:
//   - The returned string must never be logged.
//   - It must not be stored on any struct field.
//   - It must only be used as a Bearer token in an HTTP request and discarded.
func GetLLMAPIKey(credentialRef string) (string, error) {
	if credentialRef == "" {
		return "", errors.New("credential ref is empty")
	}
	key := os.Getenv(credentialRef)
	if key == "" {
		return "", fmt.Errorf("credential env var %q is not set or empty — agent unavailable", credentialRef)
	}
	return key, nil
}

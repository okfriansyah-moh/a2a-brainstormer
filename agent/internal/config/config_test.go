// Package config_test tests the agent configuration getters.
// All tests are isolated via t.Setenv so they do not affect the real environment.
package config_test

import (
	"testing"

	"a2a-brainstorm/agent/internal/config"
)

// ── GetPort ───────────────────────────────────────────────────────────────────

func TestGetPort_Default(t *testing.T) {
	t.Setenv("AGENT_PORT", "")
	if got := config.GetPort(); got != "9090" {
		t.Errorf("GetPort() = %q, want %q", got, "9090")
	}
}

func TestGetPort_Override(t *testing.T) {
	t.Setenv("AGENT_PORT", "8888")
	if got := config.GetPort(); got != "8888" {
		t.Errorf("GetPort() = %q, want %q", got, "8888")
	}
}

// ── GetLLMProvider ────────────────────────────────────────────────────────────

func TestGetLLMProvider_Default(t *testing.T) {
	t.Setenv("AGENT_LLM_PROVIDER", "")
	if got := config.GetLLMProvider(); got != "copilot" {
		t.Errorf("GetLLMProvider() = %q, want %q", got, "copilot")
	}
}

func TestGetLLMProvider_Override(t *testing.T) {
	t.Setenv("AGENT_LLM_PROVIDER", "claude")
	if got := config.GetLLMProvider(); got != "claude" {
		t.Errorf("GetLLMProvider() = %q, want %q", got, "claude")
	}
}

// ── GetLLMModel ───────────────────────────────────────────────────────────────

func TestGetLLMModel_Default(t *testing.T) {
	t.Setenv("AGENT_LLM_MODEL", "")
	if got := config.GetLLMModel(); got != "gpt-4o" {
		t.Errorf("GetLLMModel() = %q, want %q", got, "gpt-4o")
	}
}

func TestGetLLMModel_Override(t *testing.T) {
	t.Setenv("AGENT_LLM_MODEL", "claude-opus-4")
	if got := config.GetLLMModel(); got != "claude-opus-4" {
		t.Errorf("GetLLMModel() = %q, want %q", got, "claude-opus-4")
	}
}

// ── GetLLMCredentialRef ───────────────────────────────────────────────────────

func TestGetLLMCredentialRef_Default(t *testing.T) {
	t.Setenv("AGENT_LLM_CREDENTIAL_REF", "")
	if got := config.GetLLMCredentialRef(); got != "COPILOT_API_KEY" {
		t.Errorf("GetLLMCredentialRef() = %q, want %q", got, "COPILOT_API_KEY")
	}
}

func TestGetLLMCredentialRef_Override(t *testing.T) {
	t.Setenv("AGENT_LLM_CREDENTIAL_REF", "CLAUDE_API_KEY")
	if got := config.GetLLMCredentialRef(); got != "CLAUDE_API_KEY" {
		t.Errorf("GetLLMCredentialRef() = %q, want %q", got, "CLAUDE_API_KEY")
	}
}

// ── GetLLMAPIKey ──────────────────────────────────────────────────────────────

func TestGetLLMAPIKey_Present(t *testing.T) {
	t.Setenv("MY_TEST_KEY", "supersecret")
	key, err := config.GetLLMAPIKey("MY_TEST_KEY")
	if err != nil {
		t.Fatalf("GetLLMAPIKey: unexpected error: %v", err)
	}
	if key != "supersecret" {
		t.Errorf("GetLLMAPIKey = %q, want %q", key, "supersecret")
	}
}

func TestGetLLMAPIKey_Missing(t *testing.T) {
	t.Setenv("MISSING_KEY", "")
	_, err := config.GetLLMAPIKey("MISSING_KEY")
	if err == nil {
		t.Fatal("expected error for missing credential, got nil")
	}
}

func TestGetLLMAPIKey_EmptyRef(t *testing.T) {
	_, err := config.GetLLMAPIKey("")
	if err == nil {
		t.Fatal("expected error for empty credentialRef, got nil")
	}
}

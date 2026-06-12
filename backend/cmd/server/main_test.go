package main

import (
	"testing"

	"a2a-brainstorm/backend/internal/platform/llm"
)

func TestNewGlobalLLMProvider_UsesOpenCodeWhenConfigured(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "opencode")
	t.Setenv("GLOBAL_OPENCODE_BASE_URL", "http://opencode:4096")
	t.Setenv("GLOBAL_OPENCODE_MODEL", "github-copilot/claude-sonnet-4.6")
	t.Setenv("OPENCODE_USERNAME_REF", "TEST_OPENCODE_USERNAME")
	t.Setenv("OPENCODE_PASSWORD_REF", "TEST_OPENCODE_PASSWORD")
	t.Setenv("TEST_OPENCODE_USERNAME", "opencode")
	t.Setenv("TEST_OPENCODE_PASSWORD", "secret")

	provider := newGlobalLLMProvider()
	if _, ok := provider.(*llm.OpenCodeProvider); !ok {
		t.Fatalf("newGlobalLLMProvider() type = %T, want *llm.OpenCodeProvider", provider)
	}
}

func TestNewGlobalLLMProvider_FallsBackToCopilot(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "copilot")
	t.Setenv("GLOBAL_LLM_MODEL", "gpt-4o")
	t.Setenv("GLOBAL_LLM_CREDENTIAL_REF", "TEST_COPILOT_KEY")
	t.Setenv("TEST_COPILOT_KEY", "test-key")

	provider := newGlobalLLMProvider()
	if _, ok := provider.(*llm.CopilotProvider); !ok {
		t.Fatalf("newGlobalLLMProvider() type = %T, want *llm.CopilotProvider", provider)
	}
}

func TestHasDiscoveryHintsCredentials_OpenCodeMissingUsername(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "opencode")
	t.Setenv("OPENCODE_USERNAME_REF", "TEST_OPENCODE_USERNAME")
	t.Setenv("OPENCODE_PASSWORD_REF", "TEST_OPENCODE_PASSWORD")
	// Intentionally omit TEST_OPENCODE_USERNAME.
	t.Setenv("TEST_OPENCODE_PASSWORD", "secret")

	if got := hasDiscoveryHintsCredentials("opencode"); got {
		t.Fatalf("hasDiscoveryHintsCredentials(opencode) = %v, want false when username is missing", got)
	}
}

func TestHasDiscoveryHintsCredentials_OpenCodePresent(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "opencode")
	t.Setenv("OPENCODE_USERNAME_REF", "TEST_OPENCODE_USERNAME")
	t.Setenv("OPENCODE_PASSWORD_REF", "TEST_OPENCODE_PASSWORD")
	t.Setenv("TEST_OPENCODE_USERNAME", "opencode")
	t.Setenv("TEST_OPENCODE_PASSWORD", "secret")

	if got := hasDiscoveryHintsCredentials("opencode"); !got {
		t.Fatalf("hasDiscoveryHintsCredentials(opencode) = %v, want true when both credentials are present", got)
	}
}

func TestHasDiscoveryHintsCredentials_CopilotMissingCredential(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "copilot")
	t.Setenv("GLOBAL_LLM_CREDENTIAL_REF", "TEST_COPILOT_KEY")
	// Intentionally omit TEST_COPILOT_KEY.

	if got := hasDiscoveryHintsCredentials("copilot"); got {
		t.Fatalf("hasDiscoveryHintsCredentials(copilot) = %v, want false when credential is missing", got)
	}
}

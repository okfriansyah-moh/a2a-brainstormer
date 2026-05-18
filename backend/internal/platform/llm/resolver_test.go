package llm_test

import (
	"testing"

	"a2a-brainstorm/backend/internal/platform/llm"
)

func TestResolve_GlobalOnly(t *testing.T) {
	global := &llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "COPILOT_API_KEY"}
	got := llm.Resolve(global, nil, nil)
	if got.Provider != "copilot" || got.Model != "gpt-4o" || got.CredentialRef != "COPILOT_API_KEY" {
		t.Errorf("expected global config unchanged; got %+v", got)
	}
}

func TestResolve_NilGlobal(t *testing.T) {
	got := llm.Resolve(nil, nil, nil)
	if got.Provider != "" || got.Model != "" || got.CredentialRef != "" {
		t.Errorf("expected zero config for nil global; got %+v", got)
	}
}

func TestResolve_AgentLevelOverridesGlobal(t *testing.T) {
	global := &llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "COPILOT_API_KEY"}
	agentLevel := &llm.LLMConfig{Provider: "claude", Model: "claude-opus-4", CredentialRef: "CLAUDE_API_KEY"}
	got := llm.Resolve(global, agentLevel, nil)
	if got.Provider != "claude" {
		t.Errorf("expected provider=claude; got %s", got.Provider)
	}
	if got.Model != "claude-opus-4" {
		t.Errorf("expected model=claude-opus-4; got %s", got.Model)
	}
	if got.CredentialRef != "CLAUDE_API_KEY" {
		t.Errorf("expected credentialRef=CLAUDE_API_KEY; got %s", got.CredentialRef)
	}
}

func TestResolve_SessionOverridesModelOnly(t *testing.T) {
	// Partial session override: only Model set; Provider + CredentialRef
	// should come from agentLevel.
	global := &llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "COPILOT_API_KEY"}
	agentLevel := &llm.LLMConfig{Provider: "claude", Model: "claude-opus-4", CredentialRef: "CLAUDE_API_KEY"}
	sessionOverride := &llm.LLMConfig{Model: "claude-sonnet-4"} // partial

	got := llm.Resolve(global, agentLevel, sessionOverride)
	if got.Provider != "claude" {
		t.Errorf("expected provider=claude; got %s", got.Provider)
	}
	if got.Model != "claude-sonnet-4" {
		t.Errorf("expected model=claude-sonnet-4; got %s", got.Model)
	}
	if got.CredentialRef != "CLAUDE_API_KEY" {
		t.Errorf("expected credentialRef=CLAUDE_API_KEY; got %s", got.CredentialRef)
	}
}

func TestResolve_SessionOverridesAll(t *testing.T) {
	global := &llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "COPILOT_API_KEY"}
	session := &llm.LLMConfig{Provider: "claude", Model: "claude-opus-4", CredentialRef: "CLAUDE_API_KEY"}
	got := llm.Resolve(global, nil, session)
	if got.Provider != "claude" || got.Model != "claude-opus-4" || got.CredentialRef != "CLAUDE_API_KEY" {
		t.Errorf("expected full session override; got %+v", got)
	}
}

func TestResolveKey_Missing(t *testing.T) {
	_, err := llm.ResolveKey("NONEXISTENT_LLM_CRED_XYZ_999")
	if err == nil {
		t.Error("expected error for absent env var; got nil")
	}
}

func TestResolveKey_Empty(t *testing.T) {
	_, err := llm.ResolveKey("")
	if err == nil {
		t.Error("expected error for empty credentialRef; got nil")
	}
}

func TestResolveKey_Present(t *testing.T) {
	t.Setenv("TEST_LLM_KEY_RESOLVER", "my-test-key")
	got, err := llm.ResolveKey("TEST_LLM_KEY_RESOLVER")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "my-test-key" {
		t.Errorf("expected my-test-key; got %s", got)
	}
}

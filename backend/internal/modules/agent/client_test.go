// Package agent_test covers LLM config tier resolution in the Dispatch function
// without requiring a live A2A endpoint.
//
// Task 15 requirements:
//   - Assert Dispatch resolves tiered LLM config: session override > agent-level > global (§8.12)
//
// BuildSystemPrompt tests are in agent_service_test.go (Task 7).
package agent_test

import (
	"testing"

	"a2a-brainstorm/backend/internal/platform/llm"
)

// ── LLM config tier resolution (§8.12) ───────────────────────────────────────
//
// The Dispatch function resolves via llm.Resolve(global, agentLevel, sessionOverride).
// We test the resolver directly since Dispatch requires a live A2A endpoint.

func TestLLMResolve_SessionOverrideWins(t *testing.T) {
	global := &llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "COPILOT_API_KEY"}
	agentLevel := &llm.LLMConfig{Provider: "claude", Model: "claude-3-sonnet", CredentialRef: "CLAUDE_API_KEY"}
	session := &llm.LLMConfig{Provider: "claude", Model: "claude-opus-4", CredentialRef: "CLAUDE_API_KEY"}

	resolved := llm.Resolve(global, agentLevel, session)
	if resolved.Model != "claude-opus-4" {
		t.Errorf("session override must win; got model %q", resolved.Model)
	}
	if resolved.Provider != "claude" {
		t.Errorf("session override must win; got provider %q", resolved.Provider)
	}
}

func TestLLMResolve_AgentLevelOverridesGlobal(t *testing.T) {
	global := &llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "COPILOT_API_KEY"}
	agentLevel := &llm.LLMConfig{Provider: "claude", Model: "claude-3-haiku", CredentialRef: "CLAUDE_API_KEY"}

	resolved := llm.Resolve(global, agentLevel, nil)
	if resolved.Model != "claude-3-haiku" {
		t.Errorf("agent-level must override global; got model %q", resolved.Model)
	}
}

func TestLLMResolve_GlobalFallback(t *testing.T) {
	global := &llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "COPILOT_API_KEY"}

	resolved := llm.Resolve(global, nil, nil)
	if resolved.Model != "gpt-4o" {
		t.Errorf("global fallback; got model %q", resolved.Model)
	}
	if resolved.Provider != "copilot" {
		t.Errorf("global fallback; got provider %q", resolved.Provider)
	}
}

func TestLLMResolve_PartialSessionOverride_MergesFromAgent(t *testing.T) {
	// A session override of only {model} (no provider/credential) must pull
	// provider and credential from the agent-level config.
	global := &llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "COPILOT_API_KEY"}
	agentLevel := &llm.LLMConfig{Provider: "claude", Model: "claude-3-haiku", CredentialRef: "CLAUDE_API_KEY"}
	session := &llm.LLMConfig{Model: "claude-opus-4"} // provider and credential_ref are empty

	resolved := llm.Resolve(global, agentLevel, session)
	if resolved.Model != "claude-opus-4" {
		t.Errorf("session model override; got model %q", resolved.Model)
	}
	// Provider must fall back to agent-level since session left it empty.
	if resolved.Provider != "claude" {
		t.Errorf("provider must come from agent-level when session leaves it empty; got %q", resolved.Provider)
	}
}

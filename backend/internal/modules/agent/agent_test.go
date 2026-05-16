// Package agent_test verifies the pure-logic components of the agent package:
// DefaultRoles, ValidRole, and JSON serialisation of Agent/Skill.
// Repository tests require a live database and are deferred to integration tests.
package agent_test

import (
	"encoding/json"
	"testing"
	"time"

	"a2a-brainstorm/backend/internal/modules/agent"
	"a2a-brainstorm/backend/internal/platform/llm"
)

// ── DefaultRoles ──────────────────────────────────────────────────────────────

func TestDefaultRoles_Two(t *testing.T) {
	roles := agent.DefaultRoles(2)
	if len(roles) != 2 {
		t.Fatalf("want 2 roles, got %d", len(roles))
	}
	if roles[0] != agent.RoleBuilder {
		t.Errorf("roles[0]: want %q, got %q", agent.RoleBuilder, roles[0])
	}
	if roles[1] != agent.RoleReviewer {
		t.Errorf("roles[1]: want %q, got %q", agent.RoleReviewer, roles[1])
	}
}

func TestDefaultRoles_Three(t *testing.T) {
	roles := agent.DefaultRoles(3)
	if len(roles) != 3 {
		t.Fatalf("want 3 roles, got %d", len(roles))
	}
	want := []agent.Role{agent.RoleBuilder, agent.RoleReviewer, agent.RoleRefiner}
	for i, w := range want {
		if roles[i] != w {
			t.Errorf("roles[%d]: want %q, got %q", i, w, roles[i])
		}
	}
}

func TestDefaultRoles_Four(t *testing.T) {
	roles := agent.DefaultRoles(4)
	if len(roles) != 4 {
		t.Fatalf("want 4 roles, got %d", len(roles))
	}
	want := []agent.Role{
		agent.RoleBuilder, agent.RoleReviewer,
		agent.RoleRefiner, agent.RoleDevilsAdvocate,
	}
	for i, w := range want {
		if roles[i] != w {
			t.Errorf("roles[%d]: want %q, got %q", i, w, roles[i])
		}
	}
}

func TestDefaultRoles_Five(t *testing.T) {
	// 5th agent is beyond catalogue; must be assigned "review".
	roles := agent.DefaultRoles(5)
	if len(roles) != 5 {
		t.Fatalf("want 5 roles, got %d", len(roles))
	}
	if roles[4] != agent.RoleReviewer {
		t.Errorf("roles[4]: want %q (extra agent), got %q", agent.RoleReviewer, roles[4])
	}
}

func TestDefaultRoles_BelowMinimum(t *testing.T) {
	if agent.DefaultRoles(1) != nil {
		t.Error("expected nil for agentCount < 2")
	}
	if agent.DefaultRoles(0) != nil {
		t.Error("expected nil for agentCount = 0")
	}
}

// ── ValidRole ─────────────────────────────────────────────────────────────────

func TestValidRole_AllValid(t *testing.T) {
	valid := []agent.Role{
		agent.RoleBuilder,
		agent.RoleReviewer,
		agent.RoleRefiner,
		agent.RoleDevilsAdvocate,
	}
	for _, r := range valid {
		if !agent.ValidRole(r) {
			t.Errorf("ValidRole(%q) = false, want true", r)
		}
	}
}

func TestValidRole_Invalid(t *testing.T) {
	invalid := []agent.Role{"", "admin", "build_and_review", "BUILD"}
	for _, r := range invalid {
		if agent.ValidRole(r) {
			t.Errorf("ValidRole(%q) = true, want false", r)
		}
	}
}

// ── JSON serialisation ────────────────────────────────────────────────────────

func TestAgent_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond).UTC()
	cfg := &llm.LLMConfig{
		Provider:      "claude",
		Model:         "claude-opus-4",
		CredentialRef: "CLAUDE_API_KEY",
	}
	original := agent.Agent{
		ID:           "00000000-0000-0000-0000-000000000001",
		Name:         "architect",
		Description:  "Designs the system",
		DefaultRole:  agent.RoleBuilder,
		SystemPrompt: "You design systems.",
		LLMConfig:    cfg,
		Endpoint:     "http://localhost:9090",
		CreatedAt:    now,
		Skills:       []agent.Skill{{ID: "skill-1", Name: "brainstorm", Prompt: "Think deeply."}},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded agent.Agent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID: want %q, got %q", original.ID, decoded.ID)
	}
	if decoded.DefaultRole != agent.RoleBuilder {
		t.Errorf("DefaultRole: want %q, got %q", agent.RoleBuilder, decoded.DefaultRole)
	}
	if decoded.LLMConfig == nil {
		t.Fatal("LLMConfig: want non-nil, got nil")
	}
	if decoded.LLMConfig.CredentialRef != "CLAUDE_API_KEY" {
		t.Errorf("CredentialRef: want %q, got %q", "CLAUDE_API_KEY", decoded.LLMConfig.CredentialRef)
	}
	if len(decoded.Skills) != 1 {
		t.Errorf("Skills: want 1, got %d", len(decoded.Skills))
	}
}

func TestAgent_JSONFieldNames(t *testing.T) {
	// Verify canonical JSON field names — downstream consumers depend on these.
	a := agent.Agent{
		ID:          "id-1",
		Name:        "n",
		DefaultRole: agent.RoleReviewer,
		Endpoint:    "http://x",
	}
	data, _ := json.Marshal(a)
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "name", "default_role", "endpoint", "created_at"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON field %q missing", key)
		}
	}
}

func TestSkill_JSONRoundTrip(t *testing.T) {
	s := agent.Skill{
		ID:          "skill-abc",
		Name:        "convergence",
		Description: "Detects convergence.",
		Prompt:      "Evaluate if the ideas have converged.",
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded agent.Skill
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Name != s.Name {
		t.Errorf("Name: want %q, got %q", s.Name, decoded.Name)
	}
	if decoded.Prompt != s.Prompt {
		t.Errorf("Prompt: want %q, got %q", s.Prompt, decoded.Prompt)
	}
}

func TestRoleConstants_Values(t *testing.T) {
	// Guard against accidental renames — downstream agent expects exact strings.
	cases := map[agent.Role]string{
		agent.RoleBuilder:        "build",
		agent.RoleReviewer:       "review",
		agent.RoleRefiner:        "refine",
		agent.RoleDevilsAdvocate: "devils_advocate",
	}
	for r, want := range cases {
		if string(r) != want {
			t.Errorf("Role constant %q: want value %q, got %q", r, want, string(r))
		}
	}
}

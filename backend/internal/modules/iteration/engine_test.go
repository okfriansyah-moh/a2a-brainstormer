// Package iteration — engine tests.
//
// Tests use in-package stubs (no live DB, no live A2A endpoint) to verify:
//   - Convergence is detected when all three §8.6 conditions hold simultaneously.
//   - Agents are dispatched in ascending Position order within each pass (§8.4).
package iteration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	agentpkg "a2a-brainstorm/backend/internal/modules/agent"
	"a2a-brainstorm/backend/internal/modules/session"
	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/llm"
)

// ─── stubs ────────────────────────────────────────────────────────────────────

// stubAgentProvider returns pre-configured agents and records the order of
// GetAgent calls so pipeline ordering can be asserted.
type stubAgentProvider struct {
	agents map[string]agentpkg.Agent
	order  []string // AgentID of each GetAgent call, in call order
}

func (s *stubAgentProvider) GetAgent(_ context.Context, id string) (agentpkg.Agent, error) {
	s.order = append(s.order, id)
	if a, ok := s.agents[id]; ok {
		return a, nil
	}
	return agentpkg.Agent{}, fmt.Errorf("stub: agent not found: %s", id)
}

func (s *stubAgentProvider) ResolveActiveSkills(_ context.Context, _ string, _ *[]string) ([]agentpkg.Skill, error) {
	return nil, nil
}

// stubSessionStore captures UpdateState and UpdateStatus calls in memory.
type stubSessionStore struct {
	states   []state.CanonicalState
	statuses []string
}

func (s *stubSessionStore) UpdateState(_ context.Context, _ string, cs *state.CanonicalState) error {
	cp := *cs
	s.states = append(s.states, cp)
	return nil
}

func (s *stubSessionStore) UpdateStatus(_ context.Context, _ string, status string) error {
	s.statuses = append(s.statuses, status)
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// twoAgentSession creates a minimal session with two agents at positions 0 and 1.
func twoAgentSession(sessID, agentAID, agentBID string, maxIter int) session.Session {
	return session.Session{
		ID:            sessID,
		Idea:          "test product idea",
		Status:        session.StatusActive,
		MaxIterations: maxIter,
		Agents: []session.SessionAgent{
			{SessionID: sessID, AgentID: agentAID, Position: 0, Role: string(agentpkg.RoleBuilder)},
			{SessionID: sessID, AgentID: agentBID, Position: 1, Role: string(agentpkg.RoleReviewer)},
		},
	}
}

// completePlanStep returns a plan step whose description exceeds the 10-word
// minimum required by state.Merge and IsExecutionPlanComplete.
func completePlanStep() state.Step {
	return state.Step{
		Title:       "Implement core API",
		Description: "Design and implement the REST API endpoints for all core business operations",
	}
}

// ─── tests ────────────────────────────────────────────────────────────────────

// TestEngineConvergence verifies that the engine stops early when all three
// quality-convergence conditions from §8.6 are met before maxIter is reached.
//
// Confidence sequence (2 agents per pass, 10 max iterations):
//
//	Pass 1: 0.5,  0.5  → delta = |0.50 - 0.00| = 0.50  → NOT converged
//	Pass 2: 0.52, 0.52 → delta = |0.52 - 0.50| = 0.02  → NOT converged (equal, not strictly less)
//	Pass 3: 0.525,0.525→ delta = |0.525- 0.52| = 0.005 → converged  ✓
func TestEngineConvergence(t *testing.T) {
	t.Setenv("CONVERGENCE_THRESHOLD", "0.02")

	const (
		sessID   = "11111111-1111-1111-1111-111111111111"
		agentAID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		agentBID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	)

	agentProv := &stubAgentProvider{
		agents: map[string]agentpkg.Agent{
			agentAID: {ID: agentAID, Name: "Builder", DefaultRole: agentpkg.RoleBuilder, Endpoint: "http://agent-a"},
			agentBID: {ID: agentBID, Name: "Reviewer", DefaultRole: agentpkg.RoleReviewer, Endpoint: "http://agent-b"},
		},
	}
	store := &stubSessionStore{}

	// Confidence per dispatch call (6 calls = 2 agents × 3 passes).
	callConfidences := []float64{0.5, 0.5, 0.52, 0.52, 0.525, 0.525}
	callIdx := 0

	mockDispatch := func(
		_ context.Context,
		_ agentpkg.Agent,
		_ agentpkg.Role,
		_ []agentpkg.Skill,
		_ *llm.LLMConfig,
		current state.CanonicalState,
	) (state.CanonicalState, error) {
		out := current
		if callIdx < len(callConfidences) {
			out.Metrics.Confidence = callConfidences[callIdx]
		}
		// Provide a complete execution plan so IsExecutionPlanComplete → true.
		out.ExecutionPlan = []state.Step{completePlanStep()}
		callIdx++
		return out, nil
	}

	eng := NewEngine(mockDispatch, agentProv, store, testLogger())
	sess := twoAgentSession(sessID, agentAID, agentBID, 10)

	initial := state.CanonicalState{
		Idea: map[string]any{"text": "test idea"},
	}

	result, err := eng.Run(context.Background(), sess, initial)
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	// Convergence must trigger at iteration 3.
	if result.Meta.Iteration != 3 {
		t.Errorf("expected convergence at iteration 3, got iteration %d", result.Meta.Iteration)
	}

	// Session status must be updated to "converged".
	if len(store.statuses) == 0 || store.statuses[len(store.statuses)-1] != session.StatusConverged {
		t.Errorf("expected final status %q, got statuses %v", session.StatusConverged, store.statuses)
	}

	// Exactly 6 dispatch calls: 2 agents × 3 passes.
	if callIdx != 6 {
		t.Errorf("expected 6 dispatch calls, got %d", callIdx)
	}

	// State must have been persisted after each pass (3 times).
	if len(store.states) != 3 {
		t.Errorf("expected 3 persisted states, got %d", len(store.states))
	}
}

// TestEnginePipelineOrder verifies that within each iteration pass agents are
// dispatched in ascending Position order (§8.4: "ordered by position ASC").
//
// The mock dispatch returns confidence = 0.999 on every call, which means:
//
//	Pass 1: delta = |0.999 - 0.000| = 0.999 → NOT converged
//	Pass 2: delta = |0.999 - 0.999| = 0.000 → converged ✓  (2 passes total)
//
// Total calls: 4 (2 agents × 2 passes). Expected order: A, B, A, B.
func TestEnginePipelineOrder(t *testing.T) {
	t.Setenv("CONVERGENCE_THRESHOLD", "0.02")

	const (
		sessID   = "22222222-2222-2222-2222-222222222222"
		agentAID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		agentBID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	)

	agentProv := &stubAgentProvider{
		agents: map[string]agentpkg.Agent{
			agentAID: {ID: agentAID, Name: "Agent A", DefaultRole: agentpkg.RoleBuilder, Endpoint: "http://a"},
			agentBID: {ID: agentBID, Name: "Agent B", DefaultRole: agentpkg.RoleReviewer, Endpoint: "http://b"},
		},
	}
	store := &stubSessionStore{}

	var callOrder []string
	mockDispatch := func(
		_ context.Context,
		ag agentpkg.Agent,
		_ agentpkg.Role,
		_ []agentpkg.Skill,
		_ *llm.LLMConfig,
		current state.CanonicalState,
	) (state.CanonicalState, error) {
		callOrder = append(callOrder, ag.ID)
		out := current
		out.Metrics.Confidence = 0.999
		out.ExecutionPlan = []state.Step{completePlanStep()}
		return out, nil
	}

	eng := NewEngine(mockDispatch, agentProv, store, testLogger())
	sess := twoAgentSession(sessID, agentAID, agentBID, 5)

	initial := state.CanonicalState{
		Idea: map[string]any{"text": "test idea"},
	}

	_, err := eng.Run(context.Background(), sess, initial)
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	// Must have at least 4 calls (2 passes × 2 agents).
	if len(callOrder) < 4 {
		t.Fatalf("expected at least 4 dispatch calls, got %d: %v", len(callOrder), callOrder)
	}

	// Verify agent A is always called before agent B in each pass.
	for pass := 0; pass+1 < len(callOrder); pass += 2 {
		if callOrder[pass] != agentAID {
			t.Errorf("pass %d: expected agent A (position 0) first, got %s", pass/2+1, callOrder[pass])
		}
		if callOrder[pass+1] != agentBID {
			t.Errorf("pass %d: expected agent B (position 1) second, got %s", pass/2+1, callOrder[pass+1])
		}
	}
}

// TestEngineMaxIterations verifies that the engine stops at maxIter when
// quality convergence is never reached, and marks the session "converged".
func TestEngineMaxIterations(t *testing.T) {
	t.Setenv("CONVERGENCE_THRESHOLD", "0.02")

	const (
		sessID   = "33333333-3333-3333-3333-333333333333"
		agentAID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		agentBID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	)

	agentProv := &stubAgentProvider{
		agents: map[string]agentpkg.Agent{
			agentAID: {ID: agentAID, Name: "A", DefaultRole: agentpkg.RoleBuilder, Endpoint: "http://a"},
			agentBID: {ID: agentBID, Name: "B", DefaultRole: agentpkg.RoleReviewer, Endpoint: "http://b"},
		},
	}
	store := &stubSessionStore{}

	// Dispatch always returns an incomplete state so convergence never fires.
	mockDispatch := func(
		_ context.Context,
		_ agentpkg.Agent,
		_ agentpkg.Role,
		_ []agentpkg.Skill,
		_ *llm.LLMConfig,
		current state.CanonicalState,
	) (state.CanonicalState, error) {
		// No execution plan → IsExecutionPlanComplete = false → never converges.
		return current, nil
	}

	const maxIter = 3
	eng := NewEngine(mockDispatch, agentProv, store, testLogger())
	sess := twoAgentSession(sessID, agentAID, agentBID, maxIter)

	result, err := eng.Run(context.Background(), sess, state.CanonicalState{
		Idea: map[string]any{"text": "test idea"},
	})
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	// Engine ran all maxIter passes.
	if result.Meta.Iteration != maxIter {
		t.Errorf("expected iteration %d at max, got %d", maxIter, result.Meta.Iteration)
	}

	// Session status must be updated to "converged" even when maxIter is reached.
	if len(store.statuses) == 0 || store.statuses[len(store.statuses)-1] != session.StatusConverged {
		t.Errorf("expected final status %q, got statuses %v", session.StatusConverged, store.statuses)
	}

	// State persisted once per pass.
	if len(store.states) != maxIter {
		t.Errorf("expected %d persisted states, got %d", maxIter, len(store.states))
	}
}

// TestEngineMetaAgentsPopulated verifies that the backend is authoritative for
// Meta.Agents — name, role, provider, model, and skills are populated from the
// agent registry, not from whatever the LLM returns.
//
// The mock dispatch deliberately zeroes out meta.agents to simulate an LLM
// that strips or corrupts the field; the engine must restore the correct data.
func TestEngineMetaAgentsPopulated(t *testing.T) {
	t.Setenv("CONVERGENCE_THRESHOLD", "0.02")

	const (
		sessID   = "44444444-4444-4444-4444-444444444444"
		agentAID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		agentBID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	)

	agentProv := &stubAgentProvider{
		agents: map[string]agentpkg.Agent{
			agentAID: {
				ID:          agentAID,
				Name:        "Builder Agent",
				DefaultRole: agentpkg.RoleBuilder,
				Endpoint:    "http://a",
				LLMConfig:   &llm.LLMConfig{Provider: "opencode", Model: "claude-sonnet-4.6"},
			},
			agentBID: {
				ID:          agentBID,
				Name:        "Reviewer Agent",
				DefaultRole: agentpkg.RoleReviewer,
				Endpoint:    "http://b",
				LLMConfig:   &llm.LLMConfig{Provider: "opencode", Model: "claude-sonnet-4.6"},
			},
		},
	}

	// stubAgentProviderWithSkills overrides ResolveActiveSkills to return named skills.
	agentProvWithSkills := &stubAgentProviderWithSkills{
		stubAgentProvider: agentProv,
		skills: map[string][]agentpkg.Skill{
			agentAID: {{ID: "s1", Name: "SkillA"}, {ID: "s2", Name: "SkillB"}},
			agentBID: {{ID: "s3", Name: "SkillC"}},
		},
	}

	store := &stubSessionStore{}

	// Mock dispatch deliberately wipes Meta.Agents to simulate LLM corruption.
	mockDispatch := func(
		_ context.Context,
		_ agentpkg.Agent,
		_ agentpkg.Role,
		_ []agentpkg.Skill,
		_ *llm.LLMConfig,
		current state.CanonicalState,
	) (state.CanonicalState, error) {
		out := current
		out.Metrics.Confidence = 0.999
		out.ExecutionPlan = []state.Step{completePlanStep()}
		out.Meta.Agents = nil // simulate LLM stripping the field
		return out, nil
	}

	eng := NewEngine(mockDispatch, agentProvWithSkills, store, testLogger())
	sess := twoAgentSession(sessID, agentAID, agentBID, 5)

	result, err := eng.Run(context.Background(), sess, state.CanonicalState{
		Idea: map[string]any{"text": "test idea"},
	})
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	if len(result.Meta.Agents) != 2 {
		t.Fatalf("expected 2 AgentMeta entries, got %d", len(result.Meta.Agents))
	}

	a := result.Meta.Agents[0]
	if a.AgentID != agentAID {
		t.Errorf("agent[0].AgentID = %q, want %q", a.AgentID, agentAID)
	}
	if a.Name != "Builder Agent" {
		t.Errorf("agent[0].Name = %q, want %q", a.Name, "Builder Agent")
	}
	if a.Role != string(agentpkg.RoleBuilder) {
		t.Errorf("agent[0].Role = %q, want %q", a.Role, agentpkg.RoleBuilder)
	}
	if a.Provider != "opencode" {
		t.Errorf("agent[0].Provider = %q, want %q", a.Provider, "opencode")
	}
	if a.Model != "claude-sonnet-4.6" {
		t.Errorf("agent[0].Model = %q, want %q", a.Model, "claude-sonnet-4.6")
	}
	if len(a.Skills) != 2 || a.Skills[0] != "SkillA" || a.Skills[1] != "SkillB" {
		t.Errorf("agent[0].Skills = %v, want [SkillA SkillB]", a.Skills)
	}

	b := result.Meta.Agents[1]
	if b.AgentID != agentBID {
		t.Errorf("agent[1].AgentID = %q, want %q", b.AgentID, agentBID)
	}
	if b.Name != "Reviewer Agent" {
		t.Errorf("agent[1].Name = %q, want %q", b.Name, "Reviewer Agent")
	}
	if len(b.Skills) != 1 || b.Skills[0] != "SkillC" {
		t.Errorf("agent[1].Skills = %v, want [SkillC]", b.Skills)
	}
}

// stubAgentProviderWithSkills extends stubAgentProvider with per-agent skills.
type stubAgentProviderWithSkills struct {
	*stubAgentProvider
	skills map[string][]agentpkg.Skill
}

func (s *stubAgentProviderWithSkills) ResolveActiveSkills(_ context.Context, agentID string, _ *[]string) ([]agentpkg.Skill, error) {
	return s.skills[agentID], nil
}

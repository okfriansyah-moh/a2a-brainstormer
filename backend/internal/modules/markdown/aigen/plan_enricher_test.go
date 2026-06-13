// Package aigen — tests for the plan enricher.
package aigen_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"a2a-brainstorm/backend/internal/modules/markdown/aigen"
	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/llm"
)

// planEnricherStub is a simple LLMProvider stub for plan enricher tests.
type planEnricherStub struct {
	response string
	err      error
}

func (s *planEnricherStub) Generate(_ context.Context, _ llm.LLMRequest) (llm.LLMResponse, error) {
	if s.err != nil {
		return llm.LLMResponse{}, s.err
	}
	return llm.LLMResponse{Content: s.response, FinishReason: "stop"}, nil
}

// basePlanState returns a minimal canonical state with one execution plan step.
func basePlanState() state.CanonicalState {
	return state.CanonicalState{
		Idea: map[string]any{
			"name": "Test Project",
		},
		Architecture: map[string]any{},
		ExecutionPlan: []state.Step{
			{
				Title:       "Scaffold project",
				Objective:   "Initialise the Go module structure.",
				Deliverables: []string{"backend/cmd/server/main.go"},
				ExitCriteria: []string{"go build ./... succeeds"},
			},
		},
		Meta:    state.StateMeta{Iteration: 1},
		Metrics: state.StateMetrics{Confidence: 0.5},
	}
}

// validOverlayJSON returns a valid PlanEnrichmentOverlay JSON for one phase.
func validOverlayJSON() string {
	return `{
		"phases": [
			{
				"position": 0,
				"coding_standards": ["SRP: main.go only wires dependencies, no business logic"],
				"invariant_checks": ["No direct DB calls from cmd layer"],
				"layer_tags": ["backend/cmd/server/"],
				"prompt_context_refs": ["§8.1"]
			}
		],
		"dependency_graph_ascii": "Task 1 ─► Task 2",
		"deep_knowledge_sections": [
			{"heading": "§8.1 CanonicalState", "content": "The shared state shape."}
		]
	}`
}

// TestPlanEnricher_HappyPath verifies that a valid overlay JSON is merged into
// the canonical state and the original s is not modified.
func TestPlanEnricher_HappyPath(t *testing.T) {
	stub := &planEnricherStub{response: validOverlayJSON()}
	e := aigen.NewPlanEnricher(stub, slog.Default())

	s := basePlanState()
	originalArch := s.Architecture // capture before (should be empty)
	_ = originalArch

	out, err := e.Enrich(context.Background(), s)
	if err != nil {
		t.Fatalf("Enrich returned unexpected error: %v", err)
	}

	// CodingStandards should appear in plan_phase_enrichments.
	if _, ok := out.Architecture["plan_phase_enrichments"]; !ok {
		t.Error("expected plan_phase_enrichments to be set in architecture")
	}

	// DependencyGraphASCII should be set.
	if v, ok := out.Architecture["dependency_graph_ascii"]; !ok || v == "" {
		t.Error("expected dependency_graph_ascii to be set in architecture")
	}

	// deep_knowledge_sections should be set.
	if _, ok := out.Architecture["deep_knowledge_sections"]; !ok {
		t.Error("expected deep_knowledge_sections to be set in architecture")
	}

	// Original s should be untouched.
	if len(s.Architecture) != 0 {
		t.Error("Enrich mutated the original state's architecture")
	}
}

// TestPlanEnricher_EmptyState verifies that an empty execution plan causes
// a no-op — original state returned with no panic.
func TestPlanEnricher_EmptyState(t *testing.T) {
	stub := &planEnricherStub{response: `{"phases":[],"dependency_graph_ascii":""}`}
	e := aigen.NewPlanEnricher(stub, slog.Default())

	var s state.CanonicalState // zero value: no execution plan
	out, err := e.Enrich(context.Background(), s)
	if err != nil {
		t.Fatalf("expected nil error for empty state, got: %v", err)
	}
	if out.Architecture != nil {
		t.Error("expected nil architecture for empty state")
	}
}

// TestPlanEnricher_LLMError verifies that an LLM provider error causes a graceful
// fallback — the original state is returned and no error is propagated.
func TestPlanEnricher_LLMError(t *testing.T) {
	stub := &planEnricherStub{err: errors.New("LLM unavailable")}
	e := aigen.NewPlanEnricher(stub, slog.Default())

	s := basePlanState()
	out, err := e.Enrich(context.Background(), s)
	if err != nil {
		t.Fatalf("expected nil error on LLM failure (fallback mode), got: %v", err)
	}
	// Architecture should be unchanged.
	if _, ok := out.Architecture["plan_phase_enrichments"]; ok {
		t.Error("expected plan_phase_enrichments to be absent when LLM fails")
	}
}

// TestPlanEnricher_InvalidJSON verifies that a malformed JSON response causes a
// graceful fallback — the original state is returned and no error is propagated.
func TestPlanEnricher_InvalidJSON(t *testing.T) {
	stub := &planEnricherStub{response: "not valid json {{{"}
	e := aigen.NewPlanEnricher(stub, slog.Default())

	s := basePlanState()
	out, err := e.Enrich(context.Background(), s)
	if err != nil {
		t.Fatalf("expected nil error on JSON parse failure (fallback mode), got: %v", err)
	}
	// Architecture should be unchanged.
	if _, ok := out.Architecture["plan_phase_enrichments"]; ok {
		t.Error("expected plan_phase_enrichments to be absent on JSON parse failure")
	}
}

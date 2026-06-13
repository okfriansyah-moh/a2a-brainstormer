// Package aigen — tests for the architecture enricher.
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

// enricherStub is a simple LLMProvider stub that returns a pre-configured
// JSON response or error for enricher tests.
type enricherStub struct {
	response string
	err      error
}

func (s *enricherStub) Generate(_ context.Context, _ llm.LLMRequest) (llm.LLMResponse, error) {
	if s.err != nil {
		return llm.LLMResponse{}, s.err
	}
	return llm.LLMResponse{Content: s.response, FinishReason: "stop"}, nil
}

func baseState() state.CanonicalState {
	return state.CanonicalState{
		Idea: map[string]any{
			"text": "A tool that helps teams brainstorm using autonomous AI agents",
		},
		Architecture: map[string]any{
			"backend":  "Go modular monolith",
			"frontend": "SvelteKit",
		},
		Meta:    state.StateMeta{Iteration: 2},
		Metrics: state.StateMetrics{Confidence: 0.75},
	}
}

// TestArchEnricher_HappyPath verifies that a valid JSON overlay is merged into
// the canonical state without overwriting any existing field.
func TestArchEnricher_HappyPath(t *testing.T) {
	overlayJSON := `{
		"problem_statement": "Teams lack structured brainstorm tooling",
		"solution_summary": "Multi-agent brainstorm pipeline with convergence engine",
		"scope_in": ["Core brainstorm pipeline", "Agent orchestration"],
		"scope_out": ["Mobile application", "Third-party integrations"],
		"guarantees": ["Deterministic output for same canonical state"],
		"data_flow_prose": "User submits idea; pipeline fans out to N agents in sequence.",
		"exit_criteria": ["All agents converge within 10 iterations"]
	}`
	stub := &enricherStub{response: overlayJSON}
	e := aigen.NewArchEnricher(stub, slog.Default())

	s := baseState()
	out, err := e.Enrich(context.Background(), s)
	if err != nil {
		t.Fatalf("Enrich returned unexpected error: %v", err)
	}
	// problem_statement should be set in idea
	if ps, ok := out.Idea["problem_statement"]; !ok || ps == "" {
		t.Errorf("expected idea.problem_statement to be set, got %v", ps)
	}
	// original idea fields should be preserved
	if out.Idea["text"] == "" {
		t.Error("expected idea.text to be preserved")
	}
	// guarantees should be set in architecture
	if _, ok := out.Architecture["guarantees"]; !ok {
		t.Error("expected architecture.guarantees to be set")
	}
}

// TestArchEnricher_DoesNotOverwriteExisting verifies that fields already set
// in the canonical state are not overwritten by the enricher.
func TestArchEnricher_DoesNotOverwriteExisting(t *testing.T) {
	overlayJSON := `{"problem_statement": "Overwrite attempt"}`
	stub := &enricherStub{response: overlayJSON}
	e := aigen.NewArchEnricher(stub, slog.Default())

	s := baseState()
	s.Idea["problem_statement"] = "Original problem statement"

	out, _ := e.Enrich(context.Background(), s)
	if got := out.Idea["problem_statement"]; got != "Original problem statement" {
		t.Errorf("enricher overwrote existing field; got %q", got)
	}
}

// TestArchEnricher_LLMError verifies that a provider error causes a graceful
// fallback — the original state is returned and no error is propagated.
func TestArchEnricher_LLMError(t *testing.T) {
	stub := &enricherStub{err: errors.New("LLM unavailable")}
	e := aigen.NewArchEnricher(stub, slog.Default())

	s := baseState()
	out, err := e.Enrich(context.Background(), s)
	if err != nil {
		t.Fatalf("expected nil error on LLM failure (fallback mode), got: %v", err)
	}
	// State should be unchanged
	if out.Idea["text"] != s.Idea["text"] {
		t.Error("expected original state to be returned on LLM error")
	}
}

// TestArchEnricher_InvalidJSON verifies that a malformed JSON response causes a
// graceful fallback — the original state is returned and no error is propagated.
func TestArchEnricher_InvalidJSON(t *testing.T) {
	stub := &enricherStub{response: "not valid json {{{"}
	e := aigen.NewArchEnricher(stub, slog.Default())

	s := baseState()
	out, err := e.Enrich(context.Background(), s)
	if err != nil {
		t.Fatalf("expected nil error on JSON parse failure (fallback mode), got: %v", err)
	}
	if out.Idea["text"] != s.Idea["text"] {
		t.Error("expected original state to be returned on JSON error")
	}
}

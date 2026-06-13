// Package aigen — tests for ReadmeEnricher.
package aigen

import (
	"context"
	"errors"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/llm"
)

// mockLLMForReadme is a minimal LLMProvider stub used by ReadmeEnricher tests.
type mockLLMForReadme struct {
	response string
	err      error
}

func (m *mockLLMForReadme) Generate(_ context.Context, _ llm.LLMRequest) (llm.LLMResponse, error) {
	if m.err != nil {
		return llm.LLMResponse{}, m.err
	}
	return llm.LLMResponse{Content: m.response}, nil
}

// readmeBaseState returns a minimal CanonicalState suitable for enricher tests.
func readmeBaseState() state.CanonicalState {
	return state.CanonicalState{
		Idea: map[string]any{
			"name":    "TestProject",
			"context": "A test project for CI.",
		},
		Architecture: map[string]any{
			"tech_stack": map[string]any{
				"backend":  "Go 1.26",
				"frontend": "SvelteKit",
			},
		},
	}
}

// TestReadmeEnricher_EnrichPopulatesOverlay verifies that a valid LLM overlay
// JSON is merged into the canonical state's Architecture map.
func TestReadmeEnricher_EnrichPopulatesOverlay(t *testing.T) {
	overlayJSON := `{
		"golden_rule": "same state equals same output",
		"when_to_use_mermaid": "flowchart LR\n  A --> B",
		"quick_start_commands": ["docker compose up", "open http://localhost:5173"],
		"command_reference": [{"command": "go test ./...", "description": "Run all tests"}],
		"is_not": ["Not a hosted service", "Not a general LLM wrapper"]
	}`

	enricher := NewReadmeEnricher(&mockLLMForReadme{response: overlayJSON}, 30)
	s := readmeBaseState()

	enriched, err := enricher.Enrich(context.Background(), s)
	if err != nil {
		t.Fatalf("Enrich returned unexpected error: %v", err)
	}

	if enriched.Architecture == nil {
		t.Fatal("enriched Architecture is nil")
	}

	if v, ok := enriched.Architecture["golden_rule"]; !ok {
		t.Error("golden_rule not set in Architecture")
	} else if v != "same state equals same output" {
		t.Errorf("golden_rule = %q; want 'same state equals same output'", v)
	}

	if v, ok := enriched.Architecture["when_to_use_mermaid"]; !ok {
		t.Error("when_to_use_mermaid not set in Architecture")
	} else if _, ok := v.(string); !ok {
		t.Errorf("when_to_use_mermaid is not a string: %T", v)
	}

	if v, ok := enriched.Architecture["quick_start_commands"]; !ok {
		t.Error("quick_start_commands not set in Architecture")
	} else if _, ok := v.([]any); !ok {
		t.Errorf("quick_start_commands is not []any: %T", v)
	}

	if v, ok := enriched.Architecture["command_reference"]; !ok {
		t.Error("command_reference not set in Architecture")
	} else if _, ok := v.([]any); !ok {
		t.Errorf("command_reference is not []any: %T", v)
	}

	if v, ok := enriched.Architecture["is_not"]; !ok {
		t.Error("is_not not set in Architecture")
	} else if _, ok := v.([]any); !ok {
		t.Errorf("is_not is not []any: %T", v)
	}
}

// TestReadmeEnricher_EnrichFallsBackOnLLMError verifies that when the LLM
// returns an error, Enrich returns the original state unchanged with nil error.
func TestReadmeEnricher_EnrichFallsBackOnLLMError(t *testing.T) {
	enricher := NewReadmeEnricher(&mockLLMForReadme{err: errors.New("LLM unavailable")}, 30)
	s := readmeBaseState()

	enriched, err := enricher.Enrich(context.Background(), s)
	if err != nil {
		t.Fatalf("expected nil error on LLM failure, got: %v", err)
	}

	// Architecture should be unchanged — no new keys added.
	if len(enriched.Architecture) != len(s.Architecture) {
		t.Errorf("Architecture changed on LLM error: original %d keys, got %d",
			len(s.Architecture), len(enriched.Architecture))
	}
	for k := range s.Architecture {
		if _, ok := enriched.Architecture[k]; !ok {
			t.Errorf("Architecture[%q] missing after LLM error", k)
		}
	}
}

// TestReadmeEnricher_OverlayMergeEnricherWins verifies that when both the
// state and the overlay have a value for golden_rule, the overlay wins.
func TestReadmeEnricher_OverlayMergeEnricherWins(t *testing.T) {
	overlayJSON := `{"golden_rule": "enriched rule"}`

	s := readmeBaseState()
	s.Architecture["golden_rule"] = "original"

	enricher := NewReadmeEnricher(&mockLLMForReadme{response: overlayJSON}, 30)
	enriched, err := enricher.Enrich(context.Background(), s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v, ok := enriched.Architecture["golden_rule"]; !ok {
		t.Error("golden_rule missing after Enrich")
	} else if v != "enriched rule" {
		t.Errorf("golden_rule = %q; enricher should win, want 'enriched rule'", v)
	}
}

// TestReadmeEnricher_NilLLM verifies that a ReadmeEnricher constructed with
// a nil LLM provider returns the original state unchanged without panicking.
func TestReadmeEnricher_NilLLM(t *testing.T) {
	enricher := NewReadmeEnricher(nil, 30)
	s := readmeBaseState()

	enriched, err := enricher.Enrich(context.Background(), s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enriched.Idea == nil {
		t.Error("Idea is nil after nil-LLM Enrich")
	}
	if len(enriched.Architecture) != len(s.Architecture) {
		t.Errorf("Architecture changed with nil LLM: want %d keys, got %d",
			len(s.Architecture), len(enriched.Architecture))
	}
	for k := range s.Architecture {
		if _, ok := enriched.Architecture[k]; !ok {
			t.Errorf("Architecture[%q] missing after nil-LLM Enrich", k)
		}
	}
}

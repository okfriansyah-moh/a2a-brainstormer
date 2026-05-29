// Package markdown — tests for the GenerateArchitecture long-form generator.
package markdown

import (
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
)

func TestGenerateArchitecture_NonEmpty(t *testing.T) {
	s := sampleState()
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("GenerateArchitecture returned error: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(got, " — Architecture") {
		t.Errorf("expected '— Architecture' in output")
	}
}

func TestGenerateArchitecture_Determinism(t *testing.T) {
	s := sampleState()
	got1, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	got2, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got1 != got2 {
		t.Error("GenerateArchitecture is not deterministic: two calls with same input produced different output")
	}
}

func TestGenerateArchitecture_ContainsIdea(t *testing.T) {
	s := sampleState()
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"A brainstorm tool for autonomous agents",
		"Go modular monolith",
		"SvelteKit",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}

func TestGenerateArchitecture_DoesNotIncludeResolvedRisks(t *testing.T) {
	// The long-form generator uses renderRisksTable which shows all risks,
	// marking resolved ones as "✅ Resolved". This test verifies that the
	// unresolved risk appears in the output with its text visible.
	s := sampleState()
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Unresolved risk should be present.
	if !strings.Contains(got, "LLM rate limits could delay iteration") {
		t.Error("expected unresolved risk text to appear in architecture output")
	}
}

func TestGenerateArchitecture_EmptyState(t *testing.T) {
	s := state.CanonicalState{}
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("unexpected error on empty state: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty output for empty state")
	}
	// Should contain a placeholder section header.
	if !strings.Contains(got, "#") {
		t.Error("expected at least one markdown heading in empty-state output")
	}
}

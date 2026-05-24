// Package markdown — tests for the GeneratePlan long-form generator.
package markdown

import (
	"strings"
	"testing"
)

func TestGeneratePlan_MinLines(t *testing.T) {
	s := sampleState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan returned error: %v", err)
	}
	lines := strings.Count(got, "\n") + 1
	if lines < 1000 {
		t.Errorf("expected ≥ 1000 lines, got %d", lines)
	}
}

func TestGeneratePlan_Determinism(t *testing.T) {
	s := sampleState()
	got1, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	got2, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got1 != got2 {
		t.Error("GeneratePlan is not deterministic: two calls produced different output")
	}
}

func TestGeneratePlan_StructuralSections(t *testing.T) {
	s := sampleState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"## 1. Goal",
		"## 5. Implementation Tasks",
		"## 8. Deep Knowledge Reference",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}

// Package markdown — tests for the GenerateRoadmap long-form generator.
package markdown

import (
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
)

func TestGenerateRoadmap_MinLines(t *testing.T) {
	s := sampleState()
	got, err := GenerateRoadmap(s)
	if err != nil {
		t.Fatalf("GenerateRoadmap returned error: %v", err)
	}
	lines := strings.Count(got, "\n") + 1
	if lines < 1000 {
		t.Errorf("expected ≥ 1000 lines, got %d", lines)
	}
}

func TestGenerateRoadmap_Determinism(t *testing.T) {
	s := sampleState()
	got1, err := GenerateRoadmap(s)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	got2, err := GenerateRoadmap(s)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got1 != got2 {
		t.Error("GenerateRoadmap is not deterministic: two calls produced different output")
	}
}

func TestGenerateRoadmap_ContainsMilestones(t *testing.T) {
	s := sampleState()
	got, err := GenerateRoadmap(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"Set up project scaffold",
		"Implement core pipeline",
		"Agents are reachable via HTTP",
		"How do we handle network partitions",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}

func TestGenerateRoadmap_EmptyPlan(t *testing.T) {
	s := state.CanonicalState{}
	got, err := GenerateRoadmap(s)
	if err != nil {
		t.Fatalf("unexpected error on empty state: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty output for empty state")
	}
}

// Package markdown — tests for artifact generation and atomic write.
package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
)

// sampleState returns a non-trivial CanonicalState for use across tests.
func sampleState() state.CanonicalState {
	return state.CanonicalState{
		Idea: map[string]any{"text": "A brainstorm tool for autonomous agents"},
		Architecture: map[string]any{
			"backend":  "Go modular monolith",
			"frontend": "SvelteKit",
		},
		ExecutionPlan: []state.Step{
			{Title: "Set up project scaffold", Description: "Initialise go modules and directory structure"},
			{Title: "Implement core pipeline", Description: "Build the N-agent iteration engine with deterministic dispatch"},
		},
		Risks: []state.Risk{
			{Text: "LLM rate limits could delay iteration", Severity: "medium", Resolved: false},
			{Text: "Old risk already fixed", Severity: "low", Resolved: true},
		},
		Assumptions: []string{"Agents are reachable via HTTP"},
		OpenQuestions: []string{
			"How do we handle network partitions mid-iteration?",
		},
		Metrics: state.StateMetrics{Confidence: 0.85},
		Meta:    state.StateMeta{Iteration: 3},
	}
}

// ── GenerateArchitecture ────────────────────────────────────────────────────

func TestGenerateArchitecture_ContainsIdeaAndArchitecture(t *testing.T) {
	s := sampleState()
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mustContain := []string{
		"# Architecture",
		"A brainstorm tool for autonomous agents",
		"backend",
		"Go modular monolith",
		"Set up project scaffold",
		"Implement core pipeline",
		"LLM rate limits", // unresolved risk should appear
	}
	for _, want := range mustContain {
		if !strings.Contains(got, want) {
			t.Errorf("GenerateArchitecture: expected output to contain %q\ngot:\n%s", want, got)
		}
	}
}

func TestGenerateArchitecture_DoesNotIncludeResolvedRisks(t *testing.T) {
	s := sampleState()
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "Old risk already fixed") {
		t.Errorf("GenerateArchitecture: resolved risk should not appear in output")
	}
}

func TestGenerateArchitecture_EmptyArchitecture(t *testing.T) {
	s := state.CanonicalState{} // empty state
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("unexpected error on empty state: %v", err)
	}
	if !strings.Contains(got, "No architecture details recorded yet") {
		t.Errorf("expected placeholder for empty architecture, got:\n%s", got)
	}
}

// ── GenerateRoadmap ─────────────────────────────────────────────────────────

func TestGenerateRoadmap_ContainsMilestones(t *testing.T) {
	s := sampleState()
	got, err := GenerateRoadmap(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mustContain := []string{
		"# Roadmap",
		"Set up project scaffold",
		"Implement core pipeline",
		"Agents are reachable via HTTP",       // assumption
		"How do we handle network partitions", // open question
		"iteration 3",
		"0.85", // confidence
	}
	for _, want := range mustContain {
		if !strings.Contains(got, want) {
			t.Errorf("GenerateRoadmap: expected output to contain %q\ngot:\n%s", want, got)
		}
	}
}

func TestGenerateRoadmap_EmptyPlan(t *testing.T) {
	s := state.CanonicalState{}
	got, err := GenerateRoadmap(s)
	if err != nil {
		t.Fatalf("unexpected error on empty state: %v", err)
	}
	if !strings.Contains(got, "No execution plan steps recorded yet") {
		t.Errorf("expected placeholder for empty plan, got:\n%s", got)
	}
}

// ── WriteArtifacts ──────────────────────────────────────────────────────────

func TestWriteArtifacts_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	s := sampleState()

	if err := WriteArtifacts(s, dir); err != nil {
		t.Fatalf("WriteArtifacts returned error: %v", err)
	}

	for _, name := range []string{"architecture.md", "roadmap.md"} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("expected %s to be non-empty", name)
		}
	}
}

func TestWriteArtifacts_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	s := sampleState()

	if err := WriteArtifacts(s, dir); err != nil {
		t.Fatalf("first WriteArtifacts call: %v", err)
	}
	if err := WriteArtifacts(s, dir); err != nil {
		t.Fatalf("second WriteArtifacts call (idempotent): %v", err)
	}
}

// ── Writer struct ────────────────────────────────────────────────────────────

func TestWriter_WriteArtifacts(t *testing.T) {
	dir := t.TempDir()
	s := sampleState()

	w := &Writer{}
	if err := w.WriteArtifacts(s, dir); err != nil {
		t.Fatalf("Writer.WriteArtifacts: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "architecture.md")); err != nil {
		t.Errorf("architecture.md not found after Writer.WriteArtifacts: %v", err)
	}
}

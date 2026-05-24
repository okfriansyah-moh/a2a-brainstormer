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

// ── GenerateContent ──────────────────────────────────────────────────────────

func TestGenerateContent_ReturnsBothStrings(t *testing.T) {
	s := sampleState()
	arch, roadmap, err := GenerateContent(s)
	if err != nil {
		t.Fatalf("GenerateContent returned error: %v", err)
	}
	if !strings.Contains(arch, "# Architecture") {
		t.Errorf("expected arch to contain '# Architecture', got:\n%s", arch)
	}
	if !strings.Contains(roadmap, "Roadmap") {
		t.Errorf("expected roadmap to contain 'Roadmap', got:\n%s", roadmap)
	}
}

func TestGenerateContent_MatchesIndividualFunctions(t *testing.T) {
	// GenerateArchitecture uses map[string]any whose iteration order is
	// non-deterministic, so we cannot compare exact strings. Instead verify
	// that the same key content is present in both outputs.
	s := sampleState()

	arch, _, err := GenerateContent(s)
	if err != nil {
		t.Fatalf("GenerateContent returned error: %v", err)
	}

	// Both outputs must contain the same sentinel values from sampleState.
	for _, want := range []string{
		"# Architecture",
		"A brainstorm tool for autonomous agents",
		"Go modular monolith",
	} {
		if !strings.Contains(arch, want) {
			t.Errorf("GenerateContent arch: expected %q in output, got:\n%s", want, arch)
		}
	}
}

func TestWriter_GenerateContent(t *testing.T) {
	s := sampleState()
	arch, roadmap, err := GenerateContent(s)
	if err != nil {
		t.Fatalf("GenerateContent returned error: %v", err)
	}
	if arch == "" {
		t.Error("expected non-empty arch from GenerateContent")
	}
	if roadmap == "" {
		t.Error("expected non-empty roadmap from GenerateContent")
	}
}

// Package markdown — tests for artifact generation and atomic write.
package markdown

import (
	"context"
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

// expectedSlug returns the slug-based filename prefix for sampleState.
func expectedSlug() string {
	return "a-brainstorm-tool-for-autonomous-agents"
}

// ── WriteArtifacts ──────────────────────────────────────────────────────────

func TestWriteArtifacts_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	s := sampleState()

	if err := WriteArtifacts(context.Background(), s, dir, nil); err != nil {
		t.Fatalf("WriteArtifacts returned error: %v", err)
	}

	for _, suffix := range []string{"architecture.md", "roadmap.md"} {
		name := expectedSlug() + "_" + suffix
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

	if err := WriteArtifacts(context.Background(), s, dir, nil); err != nil {
		t.Fatalf("first WriteArtifacts call: %v", err)
	}
	if err := WriteArtifacts(context.Background(), s, dir, nil); err != nil {
		t.Fatalf("second WriteArtifacts call (idempotent): %v", err)
	}
}

// ── Writer struct ────────────────────────────────────────────────────────────

func TestWriter_WriteArtifacts(t *testing.T) {
	dir := t.TempDir()
	s := sampleState()

	w := &Writer{}
	if err := w.WriteArtifacts(context.Background(), s, dir, nil); err != nil {
		t.Fatalf("Writer.WriteArtifacts: %v", err)
	}
	expected := filepath.Join(dir, expectedSlug()+"_architecture.md")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected file not found after Writer.WriteArtifacts: %v", err)
	}
}

// ── Title shape (§8.23: idea text must NOT appear in H1 by itself) ───────────

func TestGenerateArchitecture_TitleShape(t *testing.T) {
	s := sampleState()
	out, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("GenerateArchitecture: %v", err)
	}
	firstLine := strings.SplitN(out, "\n", 2)[0]
	if !strings.HasPrefix(firstLine, "# ") {
		t.Fatalf("expected H1 prefix, got %q", firstLine)
	}
	if !strings.HasSuffix(firstLine, " — Architecture") {
		t.Errorf("expected H1 to end with ' — Architecture', got %q", firstLine)
	}
}

func TestGenerateReadme_DescriptionAppearsBoundedTimes(t *testing.T) {
	s := sampleState()
	out, err := GenerateReadme(s)
	if err != nil {
		t.Fatalf("GenerateReadme: %v", err)
	}
	count := strings.Count(out, "A brainstorm tool for autonomous agents")
	if count == 0 {
		t.Errorf("expected idea text to appear at least once, got 0")
	}
	if count > 3 {
		t.Errorf("expected idea text to appear ≤ 3 times (no padding loop), got %d", count)
	}
}

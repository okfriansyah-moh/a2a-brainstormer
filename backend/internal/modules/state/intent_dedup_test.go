package state_test

import (
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
)

func TestMerge_CollapsePhaseAndTaskDuplicates(t *testing.T) {
	t.Parallel()
	foundationDesc := "Set up the monorepo structure, Docker environment, database migrations, CI pipeline, and shared infrastructure with config logging DB pool auth middleware establishing the modular monolith skeleton with empty modules"
	taskDesc := "Set up monorepo, Docker, database, CI, shared infrastructure, and modular monolith skeleton with enough detail to pass validation"

	incoming := state.CanonicalState{
		Idea: map[string]any{"t": "1"},
		ExecutionPlan: []state.Step{
			{Title: "Phase 1 — Foundation & Project Scaffolding", Description: foundationDesc},
			{Title: "Task 1 — Foundation & Project Scaffolding", Description: taskDesc},
			{Title: "Phase 2 — Player & Community Modules", Description: "Implement unified player identity registration profile and community CRUD create join list members with invite codes membership roles and visibility settings"},
			{Title: "Task 2 — Player & Community Modules", Description: "Implement unified player identity registration profile and community CRUD create join list members with invite codes"},
		},
	}

	out := state.Merge(state.CanonicalState{Idea: map[string]any{"t": "1"}}, incoming)
	if len(out.ExecutionPlan) != 2 {
		t.Fatalf("expected 2 execution plan steps after intent dedup, got %d: %+v", len(out.ExecutionPlan), out.ExecutionPlan)
	}
}

func TestMerge_CollapseNearDuplicateOpenQuestions(t *testing.T) {
	t.Parallel()
	short := "What is the expected match submission write throughput at 10k MAU and 100k MAU? The current plan does not include a throughput analysis or benchmark target."
	long := short + " This should be addressed during performance testing in Phase 3."

	out := state.Merge(state.CanonicalState{
		Idea:          map[string]any{"t": "1"},
		OpenQuestions: []string{short},
	}, state.CanonicalState{
		Idea:          map[string]any{"t": "1"},
		OpenQuestions: []string{long},
	})

	if len(out.OpenQuestions) != 1 {
		t.Fatalf("expected 1 open question after dedup, got %d: %v", len(out.OpenQuestions), out.OpenQuestions)
	}
	if !strings.Contains(out.OpenQuestions[0], "performance testing") {
		t.Fatalf("expected longer open question to win, got %q", out.OpenQuestions[0])
	}
}

func TestMerge_CollapseResolvedOpenQuestionVariant(t *testing.T) {
	t.Parallel()
	short := "What concrete rate-limiting threshold and mechanism will be implemented for the anti-fraud suspicious pattern detection (same pair submitting many matches)?"
	long := "What concrete rate-limiting threshold and mechanism will be implemented for anti-fraud suspicious pattern detection? Resolved: per-pair 10 matches/day sliding window, now part of architectural decisions."

	out := state.Merge(state.CanonicalState{
		Idea:          map[string]any{"t": "1"},
		OpenQuestions: []string{short},
	}, state.CanonicalState{
		Idea:          map[string]any{"t": "1"},
		OpenQuestions: []string{long},
	})

	if len(out.OpenQuestions) != 1 {
		t.Fatalf("expected 1 open question after dedup, got %d: %v", len(out.OpenQuestions), out.OpenQuestions)
	}
	if !strings.Contains(strings.ToLower(out.OpenQuestions[0]), "resolved:") {
		t.Fatalf("expected resolved variant to win, got %q", out.OpenQuestions[0])
	}
}

func TestMerge_CollapseSameTopicAssumptions(t *testing.T) {
	t.Parallel()
	older := "The fraud validation logic is assumed to be embeddable in the match module, but the directory layout lists a separate fraud module."
	newer := "Fraud validation logic is now separated into its own module (internal/modules/fraud/), resolving the previous architectural inconsistency with the vertical slice principle."

	out := state.Merge(state.CanonicalState{
		Idea:        map[string]any{"t": "1"},
		Assumptions: []string{older},
	}, state.CanonicalState{
		Idea:        map[string]any{"t": "1"},
		Assumptions: []string{newer},
	})

	if len(out.Assumptions) != 1 {
		t.Fatalf("expected 1 assumption after topic dedup, got %d: %v", len(out.Assumptions), out.Assumptions)
	}
	if !strings.Contains(out.Assumptions[0], "internal/modules/fraud") {
		t.Fatalf("expected newer assumption to win, got %q", out.Assumptions[0])
	}
}

func TestMerge_KeepsDistinctAssumptions(t *testing.T) {
	t.Parallel()
	out := state.Merge(state.CanonicalState{
		Idea: map[string]any{"t": "1"},
		Assumptions: []string{
			"Monthly ranking snapshots are immutable and all-time rankings are live computed values.",
			"GPS venue validation will work reliably for all Jakarta indoor courts.",
		},
	}, state.CanonicalState{Idea: map[string]any{"t": "1"}})

	if len(out.Assumptions) != 2 {
		t.Fatalf("expected distinct assumptions preserved, got %d: %v", len(out.Assumptions), out.Assumptions)
	}
}

// Package markdown — comprehensive tests for the GeneratePlan generator (§8.31).
package markdown

import (
	"regexp"
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
)

// ─── test helpers ─────────────────────────────────────────────────────────────

// planSparseState returns a minimal CanonicalState with one execution plan step.
func planSparseState() state.CanonicalState {
	return state.CanonicalState{
		Idea: map[string]any{"text": "Single phase project"},
		ExecutionPlan: []state.Step{
			{
				Title:        "Scaffold project",
				Description:  "Initialise go modules and directory structure.",
				Deliverables: []string{"backend/cmd/server/main.go"},
				ExitCriteria: []string{"go build ./... succeeds"},
			},
		},
		Metrics: state.StateMetrics{Confidence: 0.5},
		Meta:    state.StateMeta{Iteration: 1},
	}
}

// planFullState returns a CanonicalState with 3 execution plan steps.
func planFullState() state.CanonicalState {
	return state.CanonicalState{
		Idea: map[string]any{
			"text":  "A multi-agent brainstorm engine",
			"goals": []any{"Goal A", "Goal B", "Goal C"},
		},
		Architecture: map[string]any{
			"backend": "Go modular monolith",
		},
		ExecutionPlan: []state.Step{
			{
				Title:             "Scaffold",
				Objective:         "Initialise the project structure.",
				Description:       "Create the Go module skeleton.",
				Deliverables:      []string{"backend/internal/modules/session/service.go"},
				FunctionContracts: []string{"func NewService(repo Repository) *Service"},
				FailureHandling:   "Return error; no panic.",
				ExitCriteria:      []string{"go build ./... succeeds"},
			},
			{
				Title:       "Core pipeline",
				Objective:   "Implement the N-agent iteration engine.",
				Description: "Build the sequential agent dispatch loop.",
				Deliverables: []string{
					"backend/internal/modules/pipeline/engine.go",
					"agent/internal/executor/executor.go",
				},
				ExitCriteria: []string{"Pipeline produces output for 2+ agents"},
			},
			{
				Title:        "Frontend",
				Objective:    "Build the SvelteKit UI.",
				Description:  "Implement pipeline status and state viewer.",
				Deliverables: []string{"frontend/src/routes/+page.svelte"},
				ExitCriteria: []string{"pnpm build succeeds"},
			},
		},
		Risks: []state.Risk{
			{Text: "LLM rate limits", Severity: "medium", Resolved: false},
		},
		Metrics: state.StateMetrics{Confidence: 0.88},
		Meta:    state.StateMeta{Iteration: 3},
	}
}

// ─── tests ────────────────────────────────────────────────────────────────────

// TestGeneratePlan_SparseState verifies that a single-phase sparse state
// renders a valid task block with all required fields.
func TestGeneratePlan_SparseState(t *testing.T) {
	s := planSparseState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan returned error: %v", err)
	}
	if !strings.Contains(got, "**Goal:**") {
		t.Error("expected **Goal:** in output")
	}
	if !strings.Contains(got, "**Validation:**") {
		t.Error("expected **Validation:** in output")
	}
	if !strings.Contains(got, "go build ./...") {
		t.Error("expected go build ./... in Validation")
	}
	if !strings.Contains(got, "**Invariant check:**") {
		t.Error("expected **Invariant check:** in output")
	}
	// 4 canonical invariant items
	canonicals := []string{
		"No file modified outside",
		"No cross-module import introduced",
		"No `os.Getenv` outside `config.go`",
		"All SQL via parameterized queries only",
	}
	for _, c := range canonicals {
		if !strings.Contains(got, c) {
			t.Errorf("expected canonical invariant %q in output", c)
		}
	}
	if strings.Contains(got, "see phase deliverables") {
		t.Error("output must NOT contain 'see phase deliverables'")
	}
}

// TestGeneratePlan_ForbiddenStrings ensures the output never contains forbidden strings.
func TestGeneratePlan_ForbiddenStrings(t *testing.T) {
	for _, tc := range []struct {
		name      string
		s         state.CanonicalState
		forbidden []string
	}{
		{
			name:      "sparse",
			s:         planSparseState(),
			forbidden: []string{"see phase deliverables", "per module test suite", "Phase 0 —", "Phase 1 —"},
		},
		{
			name:      "full",
			s:         planFullState(),
			forbidden: []string{"see phase deliverables", "per module test suite", "Phase 0 —"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GeneratePlan(tc.s)
			if err != nil {
				t.Fatalf("GeneratePlan error: %v", err)
			}
			for _, f := range tc.forbidden {
				if strings.Contains(got, f) {
					t.Errorf("output contains forbidden string %q", f)
				}
			}
		})
	}
}

// TestGeneratePlan_GoalsFromIdea verifies that s.Idea["goals"] appears in §1.
func TestGeneratePlan_GoalsFromIdea(t *testing.T) {
	s := state.CanonicalState{
		Idea: map[string]any{
			"goals": []any{"Goal A", "Goal B"},
		},
		Metrics: state.StateMetrics{Confidence: 0.5},
		Meta:    state.StateMeta{Iteration: 1},
	}
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if !strings.Contains(got, "Goal A") {
		t.Error("expected 'Goal A' in §1 Goals")
	}
	if !strings.Contains(got, "Goal B") {
		t.Error("expected 'Goal B' in §1 Goals")
	}
}

// TestGeneratePlan_GoalsFallback verifies that empty goals shows fallback message.
func TestGeneratePlan_GoalsFallback(t *testing.T) {
	s := state.CanonicalState{
		Idea:    map[string]any{"text": "some project"},
		Metrics: state.StateMetrics{Confidence: 0.5},
		Meta:    state.StateMeta{Iteration: 1},
	}
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if !strings.Contains(got, "_Goals not populated") {
		t.Error("expected '_Goals not populated' fallback in output")
	}
	if strings.Contains(got, "_Goals not yet defined by the agents._") {
		t.Error("output must NOT contain the old fallback text '_Goals not yet defined by the agents._'")
	}
}

// TestGeneratePlan_TaskHeadingFormat verifies that all task headings match
// ### Task N — and no ### Phase headings appear.
func TestGeneratePlan_TaskHeadingFormat(t *testing.T) {
	s := planFullState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	taskRE := regexp.MustCompile(`(?m)^### Task [0-9]+ —`)
	phaseRE := regexp.MustCompile(`(?m)^### Phase [0-9]+ —`)

	if !taskRE.MatchString(got) {
		t.Error("expected at least one '### Task N —' heading")
	}
	if phaseRE.MatchString(got) {
		t.Error("output must NOT contain '### Phase N —' headings")
	}
}

// TestGeneratePlan_ValidationNeverEmitsBadStrings verifies that Validation
// section never contains forbidden strings.
func TestGeneratePlan_ValidationNeverEmitsBadStrings(t *testing.T) {
	s := planFullState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if strings.Contains(got, "per module test suite") {
		t.Error("output must NOT contain 'per module test suite'")
	}
	if strings.Contains(got, "run tests") && !strings.Contains(got, "go test") {
		// "run tests" is vague; acceptable only if concrete go test command present
		t.Error("output contains vague 'run tests' without concrete go test command")
	}
}

// TestGeneratePlan_ValidationAlwaysHasGoBuild verifies every plan output contains
// go build ./...
func TestGeneratePlan_ValidationAlwaysHasGoBuild(t *testing.T) {
	for _, s := range []state.CanonicalState{planSparseState(), planFullState()} {
		got, err := GeneratePlan(s)
		if err != nil {
			t.Fatalf("GeneratePlan error: %v", err)
		}
		if !strings.Contains(got, "go build ./...") {
			t.Error("expected 'go build ./...' in output")
		}
	}
}

// TestGeneratePlan_InvariantCheckCanonicalItems verifies all 4 canonical invariant
// items appear in every task block.
func TestGeneratePlan_InvariantCheckCanonicalItems(t *testing.T) {
	s := planFullState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	canonicals := []string{
		"No file modified outside \"Files to create/modify\" list",
		"No cross-module import introduced",
		"No `os.Getenv` outside `config.go`",
		"All SQL via parameterized queries only (if DB-touching)",
	}
	for _, c := range canonicals {
		if !strings.Contains(got, c) {
			t.Errorf("expected canonical invariant %q in output", c)
		}
	}
}

// TestGeneratePlan_FullState verifies that each task section has all 7 required fields.
func TestGeneratePlan_FullState(t *testing.T) {
	s := planFullState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	requiredFields := []string{
		"**Goal:**",
		"**Layer(s) affected:**",
		"**Files to create:**",
		"**Coding standards:**",
		"**Validation:**",
		"**Invariant check:**",
		"**Prompt context needed:**",
	}
	for _, field := range requiredFields {
		if !strings.Contains(got, field) {
			t.Errorf("expected field %q in output", field)
		}
	}
	// Verify we have 3 task sections.
	taskCount := strings.Count(got, "### Task ")
	if taskCount != 3 {
		t.Errorf("expected 3 task sections, got %d", taskCount)
	}
}

// TestGeneratePlan_DeepKnowledgeSection verifies §8 Deep Knowledge is present.
func TestGeneratePlan_DeepKnowledgeSection(t *testing.T) {
	s := planFullState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if !strings.Contains(got, "## 8. Deep Knowledge Reference") {
		t.Error("expected '## 8. Deep Knowledge Reference' section")
	}
	if !strings.Contains(got, "Domain-specific schemas") {
		t.Error("expected domain-focused deep knowledge intro in §8")
	}
}

// TestGeneratePlan_Determinism verifies same state → same output (two calls).
func TestGeneratePlan_Determinism(t *testing.T) {
	s := planFullState()
	got1, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	got2, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got1 != got2 {
		t.Error("GeneratePlan is not deterministic: two calls with same state produced different output")
	}
}

// TestGeneratePlan_TitleFormat verifies H1 contains " — Implementation Plan".
func TestGeneratePlan_TitleFormat(t *testing.T) {
	s := sampleState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if !strings.Contains(got, " — Implementation Plan") {
		t.Error("expected ' — Implementation Plan' in H1")
	}
	firstLine := strings.SplitN(got, "\n", 2)[0]
	if !strings.HasPrefix(firstLine, "# ") {
		t.Errorf("expected H1 prefix, got %q", firstLine)
	}
}

// TestGeneratePlan_StructuralSections verifies all required top-level sections exist.
func TestGeneratePlan_StructuralSections(t *testing.T) {
	s := sampleState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"## 1. Goals",
		"## 2. Milestones",
		"## 5. Implementation Tasks",
		"## 7. How to Use This Plan",
		"## 8. Deep Knowledge Reference",
		"## For AI Agents",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}

// TestGeneratePlan_FrontendValidation verifies pnpm check appears when frontend deliverables present.
func TestGeneratePlan_FrontendValidation(t *testing.T) {
	s := planFullState() // includes a frontend deliverable in step 3
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if !strings.Contains(got, "pnpm check") {
		t.Error("expected 'pnpm check' when frontend deliverables are present")
	}
}

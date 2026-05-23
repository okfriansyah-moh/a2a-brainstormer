// Package state_test covers the CanonicalState model, merge algorithm, and
// validator for backend/internal/modules/state.
package state_test

import (
	"encoding/json"
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
)

// ── model / JSON round-trip ───────────────────────────────────────────────────

func TestCanonicalState_JSONFieldNames(t *testing.T) {
	s := state.CanonicalState{
		Idea:          map[string]any{"title": "Platform Idea"},
		Architecture:  map[string]any{"pattern": "modular"},
		ExecutionPlan: []state.Step{{Title: "Step 1", Description: "Implement the thing with enough words here"}},
		Risks:         []state.Risk{{Text: "Risk A", Severity: "high", Resolved: false}},
		Assumptions:   []string{"Budget is fixed"},
		OpenQuestions: []string{"Who owns this?"},
		Metrics:       state.StateMetrics{Confidence: 0.75},
		Meta: state.StateMeta{
			Iteration: 2,
			Agents: []state.AgentMeta{
				{AgentID: "aa", Name: "Agent A", Role: "build", Provider: "copilot", Model: "gpt-4o", Skills: []string{"brainstorming"}},
				{AgentID: "bb", Name: "Agent B", Role: "review", Provider: "claude", Model: "claude-opus-4", Skills: []string{"security-audit"}},
			},
		},
	}

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	raw := string(b)

	// §8.1 canonical field names
	for _, key := range []string{
		`"idea"`, `"architecture"`, `"execution_plan"`, `"risks"`,
		`"assumptions"`, `"open_questions"`, `"metrics"`, `"meta"`,
		`"confidence"`, `"iteration"`, `"agents"`,
		`"agent_id"`, `"provider"`, `"model"`, `"skills"`,
		`"title"`, `"description"`,
		`"text"`, `"severity"`, `"resolved"`,
	} {
		if !strings.Contains(raw, key) {
			t.Errorf("JSON missing key %s; got: %s", key, raw)
		}
	}

	// Round-trip
	var decoded state.CanonicalState
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded.Metrics.Confidence != 0.75 {
		t.Errorf("confidence: got %g, want 0.75", decoded.Metrics.Confidence)
	}
	if decoded.Meta.Iteration != 2 {
		t.Errorf("iteration: got %d, want 2", decoded.Meta.Iteration)
	}
	if len(decoded.Meta.Agents) != 2 {
		t.Errorf("agents: got %d, want 2", len(decoded.Meta.Agents))
	}
}

// ── Merge: risks ─────────────────────────────────────────────────────────────

func TestMerge_UnionRisks(t *testing.T) {
	base := state.CanonicalState{
		Idea:  map[string]any{"t": "1"},
		Risks: []state.Risk{{Text: "Risk A", Severity: "high"}},
	}
	incoming := state.CanonicalState{
		Idea:  map[string]any{"t": "1"},
		Risks: []state.Risk{{Text: "Risk B", Severity: "medium"}},
	}

	out := state.Merge(base, incoming)
	if len(out.Risks) != 2 {
		t.Errorf("expected 2 risks after union, got %d: %+v", len(out.Risks), out.Risks)
	}
}

func TestMerge_DeduplicateRisks(t *testing.T) {
	r := state.Risk{Text: "Risk A", Severity: "high"}
	base := state.CanonicalState{
		Idea:  map[string]any{"t": "1"},
		Risks: []state.Risk{r},
	}
	incoming := state.CanonicalState{
		Idea:  map[string]any{"t": "1"},
		Risks: []state.Risk{r, {Text: "  risk a  ", Severity: "low"}}, // normalised same as above
	}

	out := state.Merge(base, incoming)
	if len(out.Risks) != 1 {
		t.Errorf("expected 1 risk after dedup, got %d: %+v", len(out.Risks), out.Risks)
	}
}

func TestMerge_RemoveResolvedRisks(t *testing.T) {
	base := state.CanonicalState{
		Idea:  map[string]any{"t": "1"},
		Risks: []state.Risk{{Text: "Risk A", Severity: "high", Resolved: true}},
	}
	incoming := state.CanonicalState{
		Idea:  map[string]any{"t": "1"},
		Risks: []state.Risk{{Text: "Risk B", Severity: "low"}},
	}

	out := state.Merge(base, incoming)
	for _, r := range out.Risks {
		if r.Text == "Risk A" {
			t.Errorf("resolved risk should have been removed, but found: %+v", r)
		}
	}
	if len(out.Risks) != 1 {
		t.Errorf("expected 1 risk (unresolved only), got %d", len(out.Risks))
	}
}

// ── Merge: execution plan ─────────────────────────────────────────────────────

func TestMerge_RejectVagueSteps(t *testing.T) {
	base := state.CanonicalState{
		Idea: map[string]any{"t": "1"},
		ExecutionPlan: []state.Step{
			{Title: "Step A", Description: "Too short"},
		},
	}
	incoming := state.CanonicalState{
		Idea: map[string]any{"t": "1"},
		ExecutionPlan: []state.Step{
			{Title: "Step B", Description: "This description is long enough to pass the ten word minimum check"},
		},
	}

	out := state.Merge(base, incoming)
	if len(out.ExecutionPlan) != 1 {
		t.Errorf("expected 1 step after rejecting vague, got %d: %+v", len(out.ExecutionPlan), out.ExecutionPlan)
	}
	if out.ExecutionPlan[0].Title != "Step B" {
		t.Errorf("expected Step B, got %s", out.ExecutionPlan[0].Title)
	}
}

func TestMerge_CollapseStepsKeepDetailed(t *testing.T) {
	base := state.CanonicalState{
		Idea: map[string]any{"t": "1"},
		ExecutionPlan: []state.Step{
			{Title: "Deploy", Description: "Deploy to production with monitoring and alerting enabled"},
		},
	}
	incoming := state.CanonicalState{
		Idea: map[string]any{"t": "1"},
		ExecutionPlan: []state.Step{
			{Title: "deploy", Description: "Deploy to production with monitoring alerting rollback strategy and feature flags also canary release"},
		},
	}

	out := state.Merge(base, incoming)
	if len(out.ExecutionPlan) != 1 {
		t.Errorf("expected 1 collapsed step, got %d", len(out.ExecutionPlan))
	}
	// The longer description should win.
	if !strings.Contains(out.ExecutionPlan[0].Description, "canary release") {
		t.Errorf("expected longer description to win, got: %s", out.ExecutionPlan[0].Description)
	}
}

// ── Merge: assumptions / open_questions ──────────────────────────────────────

func TestMerge_UnionAssumptions(t *testing.T) {
	base := state.CanonicalState{
		Idea:        map[string]any{"t": "1"},
		Assumptions: []string{"Budget is fixed"},
	}
	incoming := state.CanonicalState{
		Idea:        map[string]any{"t": "1"},
		Assumptions: []string{"Budget is fixed", "Team size stays constant"},
	}

	out := state.Merge(base, incoming)
	if len(out.Assumptions) != 2 {
		t.Errorf("expected 2 assumptions after union+dedup, got %d: %v", len(out.Assumptions), out.Assumptions)
	}
}

func TestMerge_DeduplicateOpenQuestions(t *testing.T) {
	q := "Who is the product owner?"
	base := state.CanonicalState{
		Idea:          map[string]any{"t": "1"},
		OpenQuestions: []string{q},
	}
	incoming := state.CanonicalState{
		Idea:          map[string]any{"t": "1"},
		OpenQuestions: []string{q, "What is the deployment strategy?"},
	}

	out := state.Merge(base, incoming)
	if len(out.OpenQuestions) != 2 {
		t.Errorf("expected 2 open questions, got %d: %v", len(out.OpenQuestions), out.OpenQuestions)
	}
}

// ── Merge: stability ─────────────────────────────────────────────────────────

func TestMerge_UsesIncomingIdea(t *testing.T) {
	base := state.CanonicalState{
		Idea: map[string]any{"title": "v1"},
	}
	incoming := state.CanonicalState{
		Idea: map[string]any{"title": "v2"},
	}
	out := state.Merge(base, incoming)
	if out.Idea["title"] != "v2" {
		t.Errorf("expected incoming idea to win, got %v", out.Idea["title"])
	}
}

func TestMerge_EmptyIncomingPreservesBase(t *testing.T) {
	base := state.CanonicalState{
		Idea:         map[string]any{"title": "original"},
		Architecture: map[string]any{"pattern": "hexagonal"},
	}
	incoming := state.CanonicalState{
		Idea:         map[string]any{"title": "original"},
		Architecture: nil, // agent did not update architecture
	}
	out := state.Merge(base, incoming)
	if out.Architecture["pattern"] != "hexagonal" {
		t.Errorf("expected base architecture to be preserved when incoming is nil, got %v", out.Architecture)
	}
}

// ── Merge: persistent conflict detection ─────────────────────────────────────

func TestMerge_DetectsPersistentConflictAtIteration3(t *testing.T) {
	base := state.CanonicalState{
		Idea: map[string]any{"title": "version A"},
	}
	incoming := state.CanonicalState{
		Idea: map[string]any{"title": "version B"},
		Meta: state.StateMeta{Iteration: 3},
	}

	out := state.Merge(base, incoming)

	found := false
	for _, q := range out.OpenQuestions {
		if strings.Contains(q, "idea") && strings.Contains(q, "conflict") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected persistent conflict in open_questions at iteration 3, got: %v", out.OpenQuestions)
	}
}

func TestMerge_NoPersistentConflictBeforeIteration3(t *testing.T) {
	base := state.CanonicalState{
		Idea: map[string]any{"title": "version A"},
	}
	incoming := state.CanonicalState{
		Idea: map[string]any{"title": "version B"},
		Meta: state.StateMeta{Iteration: 2},
	}

	out := state.Merge(base, incoming)

	for _, q := range out.OpenQuestions {
		if strings.Contains(q, "conflict") {
			t.Errorf("should not flag conflict before iteration 3; got: %s", q)
		}
	}
}

// ── UnmarshalJSON: LLM string-coercion ───────────────────────────────────────

// TestUnmarshal_RisksAsStrings verifies that a JSON payload where "risks" is
// []string (common LLM shorthand) is coerced into []Risk without error.
func TestUnmarshal_RisksAsStrings(t *testing.T) {
	raw := `{
		"idea": {"text": "test"},
		"risks": ["Performance degradation under load", "High onboarding cost"],
		"metrics": {"confidence": 0.3},
		"meta": {"iteration": 1, "agents": [
			{"agent_id":"a","name":"A","role":"build","provider":"copilot","model":"gpt-4.1","skills":[]},
			{"agent_id":"b","name":"B","role":"review","provider":"copilot","model":"gpt-4.1","skills":[]}
		]}
	}`

	var cs state.CanonicalState
	if err := json.Unmarshal([]byte(raw), &cs); err != nil {
		t.Fatalf("UnmarshalJSON failed on string risks: %v", err)
	}
	if len(cs.Risks) != 2 {
		t.Fatalf("expected 2 risks, got %d: %+v", len(cs.Risks), cs.Risks)
	}
	if cs.Risks[0].Text != "Performance degradation under load" {
		t.Errorf("risk[0].Text: got %q, want %q", cs.Risks[0].Text, "Performance degradation under load")
	}
	if cs.Risks[0].Severity != "medium" {
		t.Errorf("risk[0].Severity: got %q, want \"medium\" (default)", cs.Risks[0].Severity)
	}
	if cs.Risks[0].Resolved {
		t.Errorf("risk[0].Resolved: expected false (default), got true")
	}
}

// TestUnmarshal_StepsAsStrings verifies that a JSON payload where
// "execution_plan" is []string is coerced into []Step without error.
func TestUnmarshal_StepsAsStrings(t *testing.T) {
	raw := `{
		"idea": {"text": "test"},
		"execution_plan": ["Set up CI/CD pipeline", "Deploy to staging"],
		"metrics": {"confidence": 0.2},
		"meta": {"iteration": 1, "agents": [
			{"agent_id":"a","name":"A","role":"build","provider":"copilot","model":"gpt-4.1","skills":[]},
			{"agent_id":"b","name":"B","role":"review","provider":"copilot","model":"gpt-4.1","skills":[]}
		]}
	}`

	var cs state.CanonicalState
	if err := json.Unmarshal([]byte(raw), &cs); err != nil {
		t.Fatalf("UnmarshalJSON failed on string steps: %v", err)
	}
	if len(cs.ExecutionPlan) != 2 {
		t.Fatalf("expected 2 steps, got %d: %+v", len(cs.ExecutionPlan), cs.ExecutionPlan)
	}
	if cs.ExecutionPlan[0].Title != "Set up CI/CD pipeline" {
		t.Errorf("step[0].Title: got %q, want %q", cs.ExecutionPlan[0].Title, "Set up CI/CD pipeline")
	}
}

// TestUnmarshal_MixedRisksAndSteps verifies that mixed arrays (some objects,
// some strings) are handled correctly — real LLM output can mix both forms.
func TestUnmarshal_MixedRisksAndSteps(t *testing.T) {
	raw := `{
		"idea": {"text": "test"},
		"risks": [
			{"text": "Structured risk", "severity": "high", "resolved": false},
			"Plain string risk"
		],
		"execution_plan": [
			{"title": "Structured step", "description": "A properly structured step description"},
			"Plain string step"
		],
		"metrics": {"confidence": 0.5},
		"meta": {"iteration": 1, "agents": [
			{"agent_id":"a","name":"A","role":"build","provider":"copilot","model":"gpt-4.1","skills":[]},
			{"agent_id":"b","name":"B","role":"review","provider":"copilot","model":"gpt-4.1","skills":[]}
		]}
	}`

	var cs state.CanonicalState
	if err := json.Unmarshal([]byte(raw), &cs); err != nil {
		t.Fatalf("UnmarshalJSON failed on mixed risks/steps: %v", err)
	}
	if len(cs.Risks) != 2 {
		t.Errorf("expected 2 risks, got %d", len(cs.Risks))
	}
	if cs.Risks[0].Severity != "high" {
		t.Errorf("structured risk severity: got %q, want \"high\"", cs.Risks[0].Severity)
	}
	if cs.Risks[1].Text != "Plain string risk" {
		t.Errorf("string risk text: got %q", cs.Risks[1].Text)
	}
	if len(cs.ExecutionPlan) != 2 {
		t.Errorf("expected 2 steps, got %d", len(cs.ExecutionPlan))
	}
	if cs.ExecutionPlan[1].Title != "Plain string step" {
		t.Errorf("string step title: got %q", cs.ExecutionPlan[1].Title)
	}
}

// ── Validate ──────────────────────────────────────────────────────────────────

func TestValidate_Valid(t *testing.T) {
	s := state.CanonicalState{
		Idea:    map[string]any{"title": "Something"},
		Metrics: state.StateMetrics{Confidence: 0.5},
		Meta: state.StateMeta{
			Iteration: 1,
			Agents: []state.AgentMeta{
				{AgentID: "a", Name: "A"},
				{AgentID: "b", Name: "B"},
			},
		},
	}
	if err := state.Validate(s); err != nil {
		t.Fatalf("expected no error for valid state, got: %v", err)
	}
}

func TestValidate_EmptyIdea(t *testing.T) {
	s := state.CanonicalState{
		Idea:    map[string]any{},
		Metrics: state.StateMetrics{Confidence: 0.5},
	}
	if err := state.Validate(s); err == nil {
		t.Fatal("expected error for empty idea, got nil")
	}
}

func TestValidate_NilIdea(t *testing.T) {
	s := state.CanonicalState{
		Metrics: state.StateMetrics{Confidence: 0.5},
	}
	if err := state.Validate(s); err == nil {
		t.Fatal("expected error for nil idea, got nil")
	}
}

func TestValidate_ConfidenceTooHigh(t *testing.T) {
	s := state.CanonicalState{
		Idea:    map[string]any{"t": "1"},
		Metrics: state.StateMetrics{Confidence: 1.5},
	}
	if err := state.Validate(s); err == nil {
		t.Fatal("expected error for confidence > 1, got nil")
	}
}

func TestValidate_ConfidenceNegative(t *testing.T) {
	s := state.CanonicalState{
		Idea:    map[string]any{"t": "1"},
		Metrics: state.StateMetrics{Confidence: -0.1},
	}
	if err := state.Validate(s); err == nil {
		t.Fatal("expected error for confidence < 0, got nil")
	}
}

func TestValidate_ConfidenceZeroValid(t *testing.T) {
	s := state.CanonicalState{
		Idea:    map[string]any{"t": "1"},
		Metrics: state.StateMetrics{Confidence: 0.0},
	}
	if err := state.Validate(s); err != nil {
		t.Fatalf("confidence 0.0 should be valid, got: %v", err)
	}
}

func TestValidate_ConfidenceOneValid(t *testing.T) {
	s := state.CanonicalState{
		Idea:    map[string]any{"t": "1"},
		Metrics: state.StateMetrics{Confidence: 1.0},
	}
	if err := state.Validate(s); err != nil {
		t.Fatalf("confidence 1.0 should be valid, got: %v", err)
	}
}

func TestValidate_TooFewAgentsAfterIteration(t *testing.T) {
	s := state.CanonicalState{
		Idea:    map[string]any{"t": "1"},
		Metrics: state.StateMetrics{Confidence: 0.5},
		Meta: state.StateMeta{
			Iteration: 1,
			Agents:    []state.AgentMeta{{AgentID: "a", Name: "A"}}, // only 1
		},
	}
	if err := state.Validate(s); err == nil {
		t.Fatal("expected error for fewer than 2 agents at iteration > 0, got nil")
	}
}

func TestValidate_ZeroIterationAllowsEmptyAgents(t *testing.T) {
	s := state.CanonicalState{
		Idea:    map[string]any{"t": "1"},
		Metrics: state.StateMetrics{Confidence: 0.0},
		Meta:    state.StateMeta{Iteration: 0, Agents: nil},
	}
	if err := state.Validate(s); err != nil {
		t.Fatalf("iteration 0 with no agents should be valid (initial state), got: %v", err)
	}
}

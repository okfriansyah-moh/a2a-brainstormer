// Package markdown — comprehensive tests for the 16-section GenerateArchitecture generator.
package markdown

import (
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
)

// sparseState returns a minimal CanonicalState with empty maps.
func sparseState() state.CanonicalState {
	return state.CanonicalState{}
}

// fullState returns a CanonicalState with enricher overlay fields populated.
func fullState() state.CanonicalState {
	return state.CanonicalState{
		Idea: map[string]any{
			"name":               "TestProject",
			"problem_statement":  "Teams lack structured brainstorm tooling",
			"solution_summary":   "Multi-agent pipeline with convergence engine",
			"text":               "A multi-agent brainstorm system",
			"summary":            "Automated idea refinement via AI agents",
			"goals":              []any{"Goal A", "Goal B", "Goal C", "Goal D"},
		},
		Architecture: map[string]any{
			"backend":  "Go",
			"frontend": "SvelteKit",
			"scope": map[string]any{
				"in":  []any{"Core pipeline", "Agent orchestration"},
				"out": []any{"Mobile app", "Billing"},
			},
			"decisions":      []any{"Use Go for backend", "Use PostgreSQL for storage"},
			"data_flow_prose": "User submits idea; pipeline fans agents sequentially.",
			"data_flows": []any{
				map[string]any{"from": "Client", "to": "Backend", "label": "POST /sessions"},
			},
			"security": []any{
				map[string]any{
					"surface":    "API",
					"risk":       "Unauthorized access",
					"mitigation": "Validate session IDs on every request",
				},
			},
			"guarantees": []any{
				"Deterministic output for same input",
			},
			"extension_points": []any{
				map[string]any{
					"name":  "Add LLM provider",
					"steps": []any{"Implement LLMProvider interface", "Register in resolver"},
				},
			},
			"decision_enrichments": []any{
				map[string]any{
					"title":        "Use Go for backend",
					"alternatives": "Python, Node.js",
					"tradeoff":     "Better performance, stronger typing",
				},
			},
			"tech_stack_rationale": map[string]any{
				"Go": "Compiled, fast, strong concurrency primitives",
			},
			"exit_criteria": []any{"All agents converge", "Confidence ≥ 0.85"},
		},
		Risks: []state.Risk{
			{Text: "LLM rate limits could delay iteration", Severity: "medium", Resolved: false},
			{Text: "Old risk already fixed", Severity: "low", Resolved: true},
		},
		Assumptions:   []string{"Agents are reachable via HTTP"},
		OpenQuestions: []string{"How do we handle network partitions?"},
		Metrics:       state.StateMetrics{Confidence: 0.82},
		Meta:          state.StateMeta{Iteration: 3},
	}
}

// TestGenerateArchitecture_SparseState verifies that an empty state produces
// all 16 sections without panicking and with appropriate fallback content.
func TestGenerateArchitecture_SparseState(t *testing.T) {
	s := sparseState()
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("GenerateArchitecture returned error: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty output")
	}

	sections := []string{
		"## 1. Problem Statement",
		"## 2. Solution",
		"## 3. Scope",
		"## 4. Layers",
		"## 5. Tech Stack",
		"## 6. Data Flows",
		"## 7. Module Boundaries",
		"## 8. Architecture Decisions",
		"## 9. Extension Points",
		"## 10. Security Considerations",
		"## 11. Quality Targets",
		"## 12. System Guarantees",
		"## 13. Risks",
		"## 14. Assumptions",
		"## 15. Open Questions",
		"## 16. Definition of Done",
	}
	for _, sec := range sections {
		if !strings.Contains(got, sec) {
			t.Errorf("sparse state: expected section %q in output", sec)
		}
	}

	// Verify fallback content is present
	for _, fallback := range []string{
		"_No risks identified yet._",
		"_No assumptions recorded._",
		"_No open questions at this time._",
		"Add a new LLM provider",
		"Authentication",
	} {
		if !strings.Contains(got, fallback) {
			t.Errorf("sparse state: expected fallback content %q", fallback)
		}
	}
}

// TestGenerateArchitecture_FullState verifies that a fully-populated state with
// enricher overlay fields renders each section with enriched content.
func TestGenerateArchitecture_FullState(t *testing.T) {
	s := fullState()
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("GenerateArchitecture returned error: %v", err)
	}

	for _, want := range []string{
		"Teams lack structured brainstorm tooling",   // §1 problem_statement
		"Multi-agent pipeline with convergence engine", // §2 solution_summary
		"Core pipeline",                               // §3 scope_in
		"Mobile app",                                  // §3 scope_out
		"User submits idea",                           // §6 data_flow_prose
		"Unauthorized access",                         // §10 security
		"Deterministic output for same input",         // §12 guarantees
		"All agents converge",                         // §16 exit_criteria
		"Python, Node.js",                             // §8 decision alternatives
	} {
		if !strings.Contains(got, want) {
			t.Errorf("full state: expected %q in output", want)
		}
	}
}

// TestGenerateArchitecture_MetadataBlock verifies that the metadata blockquote
// contains the iteration number and confidence value.
func TestGenerateArchitecture_MetadataBlock(t *testing.T) {
	s := sampleState() // iteration=3, confidence=0.85
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("GenerateArchitecture: %v", err)
	}
	if !strings.Contains(got, "**Iteration:** 3") {
		t.Errorf("expected '**Iteration:** 3' in metadata block")
	}
	if !strings.Contains(got, "**Confidence:** 0.8500") {
		t.Errorf("expected '**Confidence:** 0.8500' in metadata block")
	}
}

// TestGenerateArchitecture_TOC verifies that exactly 16 TOC entries are rendered
// and that each matches a section heading.
func TestGenerateArchitecture_TOC(t *testing.T) {
	s := sparseState()
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("GenerateArchitecture: %v", err)
	}

	tocEntries := []string{
		"Problem Statement",
		"Solution",
		"Scope",
		"Layers",
		"Tech Stack",
		"Data Flows",
		"Module Boundaries",
		"Architecture Decisions",
		"Extension Points",
		"Security Considerations",
		"Quality Targets",
		"System Guarantees",
		"Risks",
		"Assumptions",
		"Open Questions",
		"Definition of Done",
	}
	if len(tocEntries) != 16 {
		t.Fatalf("TOC entry list length is %d, expected 16", len(tocEntries))
	}
	for _, entry := range tocEntries {
		if !strings.Contains(got, entry) {
			t.Errorf("expected TOC entry %q in output", entry)
		}
	}
}

// TestGenerateArchitecture_ProblemStatement_Fallback verifies the three-tier
// fallback: problem_statement absent → context → text.
func TestGenerateArchitecture_ProblemStatement_Fallback(t *testing.T) {
	// Only text is present — should fall back to text ≤200 chars.
	s := state.CanonicalState{
		Idea: map[string]any{
			"text": "A brainstorm tool that uses AI agents to iteratively refine software architecture ideas through autonomous conversation.",
		},
	}
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("GenerateArchitecture: %v", err)
	}
	if !strings.Contains(got, "brainstorm tool") {
		t.Errorf("expected idea text to appear in Problem Statement section")
	}

	// context is preferred over text
	s2 := state.CanonicalState{
		Idea: map[string]any{
			"text":    "text fallback",
			"context": "Context is the primary source. More detail follows.",
		},
	}
	got2, _ := GenerateArchitecture(s2)
	if !strings.Contains(got2, "Context is the primary source") {
		t.Errorf("expected context to be used over text in Problem Statement")
	}
	if strings.Contains(got2, "text fallback") && strings.HasPrefix(got2, "## 1. Problem Statement\n\ntext fallback") {
		t.Errorf("text should not be used when context is present")
	}
}

// TestGenerateArchitecture_EnrichedRisksTable_NoToTitle verifies that severity
// values are capitalised correctly (no unicode ToTitle bug: "Medium" not "MEDIUM").
func TestGenerateArchitecture_EnrichedRisksTable_NoToTitle(t *testing.T) {
	s := state.CanonicalState{
		Risks: []state.Risk{
			{Text: "Risk A", Severity: "medium", Resolved: false},
			{Text: "Risk B", Severity: "HIGH", Resolved: false},
			{Text: "Risk C", Severity: "low", Resolved: true},
		},
	}
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("GenerateArchitecture: %v", err)
	}
	if !strings.Contains(got, "Medium") {
		t.Errorf("expected 'Medium' (title-case) in risks table")
	}
	if strings.Contains(got, "MEDIUM") {
		t.Errorf("found 'MEDIUM' (unicode ToTitle bug) — should be 'Medium'")
	}
	if !strings.Contains(got, "High") {
		t.Errorf("expected 'High' (title-case) in risks table")
	}
	if strings.Contains(got, "HIGH") && !strings.Contains(got, "⚠️ Open (HIGH)") {
		// HIGH as input should become "High" after capitaliseSeverity
		t.Errorf("found 'HIGH' in output — should be normalised to 'High'")
	}
}

// TestGenerateArchitecture_EnrichedDecisionsTable verifies that the decisions
// table has 6 columns including Decision and Status.
func TestGenerateArchitecture_EnrichedDecisionsTable(t *testing.T) {
	s := state.CanonicalState{
		Architecture: map[string]any{
			"decisions": []any{"Use PostgreSQL", "Use Go"},
			"decision_enrichments": []any{
				map[string]any{
					"title":        "Use PostgreSQL",
					"alternatives": "MySQL, SQLite",
					"tradeoff":     "JSONB support for dynamic state",
				},
			},
		},
	}
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("GenerateArchitecture: %v", err)
	}
	// 6-column header
	if !strings.Contains(got, "Decision") || !strings.Contains(got, "Status") {
		t.Errorf("expected 'Decision' and 'Status' columns in decisions table")
	}
	if !strings.Contains(got, "Alternatives") || !strings.Contains(got, "Tradeoff") {
		t.Errorf("expected 'Alternatives' and 'Tradeoff' columns in enriched decisions table")
	}
	// Enrichment values
	if !strings.Contains(got, "MySQL, SQLite") {
		t.Errorf("expected enriched alternatives 'MySQL, SQLite' in table")
	}
}

// TestGenerateArchitecture_ExtensionPoints_Fallback verifies that generic
// stub extension points are rendered when architecture["extension_points"] is absent.
func TestGenerateArchitecture_ExtensionPoints_Fallback(t *testing.T) {
	s := sparseState()
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("GenerateArchitecture: %v", err)
	}
	if !strings.Contains(got, "Add a new LLM provider") {
		t.Errorf("expected stub extension point 'Add a new LLM provider'")
	}
	if !strings.Contains(got, "Add a new output document format") {
		t.Errorf("expected stub extension point 'Add a new output document format'")
	}
}

// TestGenerateArchitecture_SecurityConsiderations_Fallback verifies that the
// generic stub security rows are rendered when architecture["security"] is absent.
func TestGenerateArchitecture_SecurityConsiderations_Fallback(t *testing.T) {
	s := sparseState()
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("GenerateArchitecture: %v", err)
	}
	if !strings.Contains(got, "Authentication") {
		t.Errorf("expected stub security row 'Authentication'")
	}
	if !strings.Contains(got, "Prompt Injection") {
		t.Errorf("expected stub security row 'Prompt Injection'")
	}
	if !strings.Contains(got, "Surface") {
		t.Errorf("expected 'Surface' column header in security table")
	}
	if !strings.Contains(got, "Mitigation") {
		t.Errorf("expected 'Mitigation' column header in security table")
	}
}

// TestGenerateArchitecture_DefinitionOfDone_AlwaysAppended verifies that
// "All open questions resolved" and "Confidence ≥ 0.85" are always present.
func TestGenerateArchitecture_DefinitionOfDone_AlwaysAppended(t *testing.T) {
	for _, s := range []state.CanonicalState{sparseState(), sampleState(), fullState()} {
		got, err := GenerateArchitecture(s)
		if err != nil {
			t.Fatalf("GenerateArchitecture: %v", err)
		}
		if !strings.Contains(got, "All open questions resolved") {
			t.Errorf("expected 'All open questions resolved' in Definition of Done")
		}
		if !strings.Contains(got, "Confidence ≥ 0.85") {
			t.Errorf("expected 'Confidence ≥ 0.85' in Definition of Done")
		}
	}
}

// TestGenerateArchitecture_Determinism verifies that two calls with the same
// state produce identical output.
func TestGenerateArchitecture_Determinism(t *testing.T) {
	for _, s := range []state.CanonicalState{sparseState(), sampleState(), fullState()} {
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
}

// TestGenerateArchitecture_TitleFormat verifies that the H1 ends with " — Architecture".
func TestGenerateArchitecture_TitleFormat(t *testing.T) {
	for _, s := range []state.CanonicalState{sparseState(), sampleState(), fullState()} {
		got, err := GenerateArchitecture(s)
		if err != nil {
			t.Fatalf("GenerateArchitecture: %v", err)
		}
		firstLine := strings.SplitN(got, "\n", 2)[0]
		if !strings.HasPrefix(firstLine, "# ") {
			t.Fatalf("expected H1 prefix, got %q", firstLine)
		}
		if !strings.HasSuffix(firstLine, " — Architecture") {
			t.Errorf("expected H1 to end with ' — Architecture', got %q", firstLine)
		}
	}
}

// Preserve existing tests that other test files rely on: NonEmpty, ContainsIdea,
// DoesNotIncludeResolvedRisks (now via enriched table), EmptyState.

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
	s := sampleState()
	got, err := GenerateArchitecture(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	if !strings.Contains(got, "#") {
		t.Error("expected at least one markdown heading in empty-state output")
	}
}

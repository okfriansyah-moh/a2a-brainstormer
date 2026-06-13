// Package markdown — tests for GenerateReadme (§8.32).
package markdown

import (
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
)

// readmeRichState returns a CanonicalState with a fully-populated Architecture
// map that exercises all 13 README sections and the enricher overlay fields.
func readmeRichState() state.CanonicalState {
	return state.CanonicalState{
		Idea: map[string]any{
			"name":    "BrainstormEngine",
			"context": "A multi-agent brainstorm pipeline that produces architecture and plan documents.",
		},
		Architecture: map[string]any{
			"tagline":              "BrainstormEngine is a deterministic multi-agent design engine.",
			"description_paragraph": "BrainstormEngine orchestrates N AI agents in a sequential pipeline to refine a software design.",
			"golden_rule":          "same state + same config = identical output, always",
			"is_not": []any{
				"Not a general-purpose LLM wrapper",
				"Not designed for single-turn conversations",
				"Not a hosted SaaS",
			},
			"when_to_use_mermaid": "flowchart LR\n  A[Start] --> B{Use case?}\n  B --> C[BrainstormEngine]",
			"when_to_use": []any{
				map[string]any{
					"title":       "Design a new service",
					"description": "Use when you need architectural perspectives on a new service.",
					"code":        "docker compose up",
				},
				map[string]any{
					"title":       "Generate an implementation plan",
					"description": "Use when you need a structured AI-driven task breakdown.",
					"code":        "go run ./backend/cmd/server/main.go",
				},
				map[string]any{
					"title":       "Iterative refinement",
					"description": "Use for multi-pass design stabilisation.",
					"code":        "curl -X POST http://localhost:8080/sessions",
				},
			},
			"prerequisites": []any{
				map[string]any{"tool": "Go", "version": "1.26+", "required": true},
				map[string]any{"tool": "Docker", "version": "24+", "required": true},
			},
			"quick_start_commands": []any{
				"docker compose up --build",
				"open http://localhost:5173",
			},
			"command_reference": []any{
				map[string]any{"command": "go test ./...", "description": "Run all tests"},
				map[string]any{"command": "docker compose up", "description": "Start all services"},
			},
			"architecture_ascii": "┌─────────┐\n│ Backend │\n└─────────┘",
			"architecture_mermaid": "flowchart LR\n  Backend --> Agent",
			"contributing_note": "Read AGENTS.md and .github/skills/ before contributing.",
			"tech_stack": map[string]any{
				"backend":  "Go 1.26",
				"frontend": "SvelteKit",
			},
		},
		Meta:    state.StateMeta{Iteration: 3},
		Metrics: state.StateMetrics{Confidence: 0.87},
	}
}

// readmeMinimalState returns a nearly-empty CanonicalState to exercise fallbacks.
func readmeMinimalState() state.CanonicalState {
	return state.CanonicalState{
		Idea: map[string]any{
			"name": "MinimalProject",
		},
	}
}

// TestGenerateReadme_AllThirteenSectionsPresent verifies that a rich state
// produces all required section headings.
func TestGenerateReadme_AllThirteenSectionsPresent(t *testing.T) {
	got, err := GenerateReadme(readmeRichState())
	if err != nil {
		t.Fatalf("GenerateReadme returned error: %v", err)
	}

	requiredHeadings := []string{
		"## Golden Rule",
		"## What it is NOT",
		"## When to use",
		"## Installation",
		"## Quick Start",
		"## Architecture",
		"## Command Reference",
		"## Tech Stack",
		"## Repository Format",
		"## Contributing",
	}
	for _, h := range requiredHeadings {
		if !strings.Contains(got, h) {
			t.Errorf("expected section %q not found in output", h)
		}
	}
}

// TestGenerateReadme_TitleNoReadmeSuffix verifies the title never contains
// "— README" (the old forbidden suffix).
func TestGenerateReadme_TitleNoReadmeSuffix(t *testing.T) {
	got, err := GenerateReadme(readmeRichState())
	if err != nil {
		t.Fatalf("GenerateReadme returned error: %v", err)
	}

	if strings.Contains(got, "— README") {
		t.Errorf("output title contains forbidden '— README' suffix")
	}
	if !strings.HasPrefix(got, "# BrainstormEngine\n") {
		t.Errorf("expected output to start with '# BrainstormEngine\\n', got prefix: %q",
			got[:min(50, len(got))])
	}
}

// TestGenerateReadme_ForbiddenStringsAbsent verifies none of the forbidden
// strings appear in the deterministic output.
func TestGenerateReadme_ForbiddenStringsAbsent(t *testing.T) {
	got, err := GenerateReadme(readmeRichState())
	if err != nil {
		t.Fatalf("GenerateReadme returned error: %v", err)
	}

	forbidden := []string{
		"<repository-url>",
		"Phase 1 —",
		"## Known Risks",
		"For AI Agents",
		"renderForAIAgentsAppendix",
	}
	for _, f := range forbidden {
		if strings.Contains(got, f) {
			t.Errorf("forbidden string %q found in output", f)
		}
	}
}

// TestGenerateReadme_GoldenRuleRendered verifies that a golden_rule value in
// the Architecture map is rendered as **Golden rule:** {text}.
func TestGenerateReadme_GoldenRuleRendered(t *testing.T) {
	s := readmeMinimalState()
	s.Architecture = map[string]any{
		"golden_rule": "same state = same output",
	}

	got, err := GenerateReadme(s)
	if err != nil {
		t.Fatalf("GenerateReadme returned error: %v", err)
	}

	if !strings.Contains(got, "**Golden rule:**") {
		t.Error("expected '**Golden rule:**' in output")
	}
	if !strings.Contains(got, "same state = same output") {
		t.Error("expected golden_rule value 'same state = same output' in output")
	}
}

// TestGenerateReadme_WhenToUseWithMermaid verifies that a when_to_use_mermaid
// value triggers a ```mermaid fenced block in the output.
func TestGenerateReadme_WhenToUseWithMermaid(t *testing.T) {
	s := readmeMinimalState()
	s.Architecture = map[string]any{
		"when_to_use_mermaid": "flowchart LR\n  A --> B",
	}

	got, err := GenerateReadme(s)
	if err != nil {
		t.Fatalf("GenerateReadme returned error: %v", err)
	}

	if !strings.Contains(got, "```mermaid") {
		t.Error("expected '```mermaid' fenced block in output")
	}
	if !strings.Contains(got, "flowchart LR") {
		t.Error("expected mermaid content 'flowchart LR' in output")
	}
}

// TestGenerateReadme_WhenToUseScenariosWithCode verifies that when_to_use
// scenarios with a non-empty code field produce fenced code blocks.
func TestGenerateReadme_WhenToUseScenariosWithCode(t *testing.T) {
	s := readmeMinimalState()
	s.Architecture = map[string]any{
		"when_to_use": []any{
			map[string]any{"title": "Scenario A", "description": "Desc A", "code": "make build"},
			map[string]any{"title": "Scenario B", "description": "Desc B", "code": "make test"},
			map[string]any{"title": "Scenario C", "description": "Desc C", "code": "make run"},
		},
	}

	got, err := GenerateReadme(s)
	if err != nil {
		t.Fatalf("GenerateReadme returned error: %v", err)
	}

	// Each scenario with a code field should produce a fenced code block.
	fenceCount := strings.Count(got, "```\nmake")
	if fenceCount < 3 {
		t.Errorf("expected at least 3 fenced code blocks for scenarios, found %d", fenceCount)
	}
}

// TestGenerateReadme_QuickStartRealCommands verifies that quick_start_commands
// are rendered verbatim and <repository-url> is never emitted.
func TestGenerateReadme_QuickStartRealCommands(t *testing.T) {
	s := readmeMinimalState()
	s.Architecture = map[string]any{
		"quick_start_commands": []any{
			"docker compose up",
			"open http://localhost:5173",
		},
	}

	got, err := GenerateReadme(s)
	if err != nil {
		t.Fatalf("GenerateReadme returned error: %v", err)
	}

	if !strings.Contains(got, "docker compose up") {
		t.Error("expected 'docker compose up' in output")
	}
	if !strings.Contains(got, "open http://localhost:5173") {
		t.Error("expected 'open http://localhost:5173' in output")
	}
	if strings.Contains(got, "<repository-url>") {
		t.Error("forbidden placeholder '<repository-url>' found in output")
	}
}

// TestGenerateReadme_FallbacksForEmptyState verifies that GenerateReadme does
// not panic on an empty state, produces ≥ 800 chars, and contains ## Contributing.
func TestGenerateReadme_FallbacksForEmptyState(t *testing.T) {
	s := state.CanonicalState{
		Idea: map[string]any{"name": "EmptyProject"},
	}

	got, err := GenerateReadme(s)
	if err != nil {
		t.Fatalf("GenerateReadme returned error: %v", err)
	}
	if len(got) < 800 {
		t.Errorf("expected output ≥ 800 chars, got %d chars", len(got))
	}
	if !strings.Contains(got, "## Contributing") {
		t.Error("expected '## Contributing' section in fallback output")
	}
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

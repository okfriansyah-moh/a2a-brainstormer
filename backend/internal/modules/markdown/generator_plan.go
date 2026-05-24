// Package markdown — PLAN.md document generator.
// GeneratePlan follows the §8.20 section skeleton and wraps the body in
// enforceMinLines so the output is always ≥ 1000 lines.
package markdown

import (
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GeneratePlan renders a PLAN.md-style document from s.
// It follows the §8.20 section skeleton and enforces a minimum of 1000 lines
// via the padPlan padder.
func GeneratePlan(s state.CanonicalState) (string, error) {
	var b strings.Builder

	// ── Header metadata block ────────────────────────────────────────────────
	title := "Implementation Plan"
	if text, ok := s.Idea["text"]; ok {
		title = fmt.Sprintf("%v — Implementation Plan", text)
	}
	b.WriteString(fmt.Sprintf("# %s\n\n", title))
	b.WriteString("> **Version:** 1.0 (AI-generated)\n")
	// Year is derived from iteration number — deterministic, not time.Now().
	year := 2024 + (s.Meta.Iteration / 10)
	b.WriteString(fmt.Sprintf("> **Date:** %d-%02d-%02d\n", year, 1, 1))
	b.WriteString("> **Author:** AI-Generated\n")
	b.WriteString("> **Status:** Draft\n")
	if text, ok := s.Idea["text"]; ok {
		b.WriteString(fmt.Sprintf("> **Source of Truth:** %v\n", text))
	}
	b.WriteString(fmt.Sprintf("> **Confidence:** %.4f (Iteration %d)\n\n", s.Metrics.Confidence, s.Meta.Iteration))

	// ── § 1. Goal ────────────────────────────────────────────────────────────
	b.WriteString("## 1. Goal\n\n")
	if len(s.Idea) > 0 {
		if text, ok := s.Idea["text"]; ok {
			b.WriteString(fmt.Sprintf("%v\n\n", text))
		} else {
			writeMap(&b, s.Idea)
		}
	} else {
		b.WriteString("_Project goal not yet defined by agents._\n\n")
	}
	b.WriteString("### Success Criteria\n\n")
	b.WriteString("- All implementation tasks completed with zero build errors.\n")
	b.WriteString("- Test coverage meets or exceeds targets for every module.\n")
	b.WriteString("- Security audit: zero OWASP Top 10 findings.\n")
	b.WriteString(fmt.Sprintf("- Convergence achieved: confidence ≥ 0.80 (current: %.4f).\n\n",
		s.Metrics.Confidence))

	// ── § 2. Architecture Overview ───────────────────────────────────────────
	b.WriteString("## 2. Architecture Overview\n\n")
	b.WriteString("### Component Diagram\n\n")
	b.WriteString(renderASCIIComponents(s))
	b.WriteString("### Architecture Decisions\n\n")
	b.WriteString(renderDecisionsTable(s))

	// ── § 3. Tech Stack ──────────────────────────────────────────────────────
	b.WriteString("## 3. Tech Stack\n\n")
	b.WriteString(renderTechStack(s))

	// ── § 4. Project Structure ───────────────────────────────────────────────
	b.WriteString("## 4. Project Structure\n\n")
	b.WriteString(renderDirectoryTree(s))
	b.WriteString("### Module Boundary Rules\n\n")
	b.WriteString("- **No cross-module internal imports.** Each module imports only `internal/platform/` and `internal/shared/`.\n")
	b.WriteString("- **LLM calls through `LLMProvider` interface only.** No direct SDK calls in `internal/modules/`.\n")
	b.WriteString("- **DB access through own repository only.** No module queries another module's tables.\n")
	b.WriteString("- **`os.Getenv` in `platform/config/config.go` only.**\n\n")

	// ── § 5. Implementation Tasks ────────────────────────────────────────────
	b.WriteString("## 5. Implementation Tasks\n\n")
	b.WriteString("### Dependency Graph\n\n")
	b.WriteString("```\n")
	if len(s.ExecutionPlan) > 0 {
		b.WriteString("[START]\n")
		for i, step := range s.ExecutionPlan {
			indent := strings.Repeat("  ", i)
			b.WriteString(fmt.Sprintf("%s  └─► [Task %d: %s]\n", indent, i+1, step.Title))
		}
		b.WriteString(fmt.Sprintf("%s  └─► [DONE]\n", strings.Repeat("  ", len(s.ExecutionPlan))))
	} else {
		b.WriteString("[START] ──► [...] ──► [DONE]\n")
	}
	b.WriteString("```\n\n")

	// One task block per execution plan step.
	if len(s.ExecutionPlan) > 0 {
		for i, step := range s.ExecutionPlan {
			b.WriteString(fmt.Sprintf("### Task %d — %s\n\n", i+1, step.Title))
			b.WriteString(fmt.Sprintf("**Goal:** %s\n\n", step.Description))
			b.WriteString("**Files to create:**\n\n")
			b.WriteString(fmt.Sprintf("- `<module>/task_%d_implementation.go` — core implementation\n", i+1))
			b.WriteString(fmt.Sprintf("- `<module>/task_%d_test.go` — unit tests\n\n", i+1))
			b.WriteString("**Validation:**\n\n")
			b.WriteString("- `go test ./...` — 0 failures\n")
			b.WriteString("- `go vet ./...` — 0 issues\n")
			b.WriteString("- `go build ./...` — 0 errors\n\n")
			b.WriteString("**Prompt context needed:**\n\n")
			b.WriteString("- Architecture decisions table (§ 2)\n")
			b.WriteString("- Canonical state shape (§ 8.1)\n\n")
			b.WriteString("---\n\n")
		}
	} else {
		b.WriteString("_No tasks defined yet. Agents will populate this section during iteration._\n\n")
	}

	// ── § 6. Task Summary ────────────────────────────────────────────────────
	b.WriteString("## 6. Task Summary\n\n")
	if len(s.ExecutionPlan) > 0 {
		rows := make([][]string, 0, len(s.ExecutionPlan))
		for i, step := range s.ExecutionPlan {
			rows = append(rows, []string{
				fmt.Sprintf("%d", i+1),
				step.Title,
				step.Description,
				fmt.Sprintf("Task %d", max(0, i)),
				"Medium",
			})
		}
		b.WriteString(renderTable(
			[]string{"Task", "Name", "Description", "Depends On", "Complexity"},
			rows,
		))
	} else {
		b.WriteString("_Task summary will appear here once the execution plan is populated._\n\n")
	}

	// ── § 7. How to Use This Plan ────────────────────────────────────────────
	b.WriteString("## 7. How to Use This Plan\n\n")
	b.WriteString("1. **Start each task in a fresh chat session** — share this plan and the relevant\n")
	b.WriteString("   blueprint sections listed under \"Prompt context needed\".\n")
	b.WriteString("2. **Validate after each task** — run `go build ./...` + `go vet ./...` before moving on.\n")
	b.WriteString("3. **Update this plan** as you learn new information during implementation.\n")
	b.WriteString("4. **One task at a time** — avoid attempting multiple tasks to prevent context overflow.\n")
	b.WriteString("5. **Source of truth** — the architecture blueprint takes precedence over this plan.\n\n")

	b.WriteString(fmt.Sprintf("---\n_Generated at iteration %d. Confidence: %.4f_\n", s.Meta.Iteration, s.Metrics.Confidence))

	body := b.String()
	return enforceMinLines(body, s, padPlan), nil
}

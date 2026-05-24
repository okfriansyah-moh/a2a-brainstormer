// Package markdown — roadmap document generator.
// GenerateRoadmap follows the §8.20 eight-section skeleton and wraps the body
// in enforceMinLines so the output is always ≥ 1000 lines.
package markdown

import (
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GenerateRoadmap renders the roadmap.md document from s.
// It follows the §8.20 section skeleton (8 sections) and enforces a minimum
// of 1000 lines via the padRoadmap padder.
func GenerateRoadmap(s state.CanonicalState) (string, error) {
	var b strings.Builder

	// ── Title ────────────────────────────────────────────────────────────────
	title := "Implementation Roadmap"
	if text, ok := s.Idea["text"]; ok {
		title = fmt.Sprintf("%v — Implementation Roadmap", text)
	}
	b.WriteString(fmt.Sprintf("# %s\n\n", title))

	// ── § 1. Goal ────────────────────────────────────────────────────────────
	b.WriteString("## 1. Goal\n\n")
	if len(s.Idea) > 0 {
		if text, ok := s.Idea["text"]; ok {
			b.WriteString(fmt.Sprintf("> %v\n\n", text))
		} else {
			writeMap(&b, s.Idea)
		}
	} else {
		b.WriteString("_Project goal not yet defined._\n\n")
	}
	b.WriteString(fmt.Sprintf("**Current Status:** Iteration %d — Confidence %.4f\n\n",
		s.Meta.Iteration, s.Metrics.Confidence))

	// ── § 2. Milestones ──────────────────────────────────────────────────────
	b.WriteString("## 2. Milestones\n\n")
	if len(s.ExecutionPlan) == 0 {
		b.WriteString("_No execution plan steps recorded yet._\n\n")
	} else {
		b.WriteString("| # | Step | Description | Target Week |\n")
		b.WriteString("|---|------|-------------|-------------|\n")
		for i, step := range s.ExecutionPlan {
			desc := step.Description
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			b.WriteString(fmt.Sprintf("| %d | %s | %s | Week %d |\n", i+1, step.Title, desc, i+1))
		}
		b.WriteString("\n")
	}

	// ── § 3. Phase Breakdown ─────────────────────────────────────────────────
	b.WriteString("## 3. Phase Breakdown\n\n")
	if len(s.ExecutionPlan) > 0 {
		for i, step := range s.ExecutionPlan {
			b.WriteString(fmt.Sprintf("### Phase %d: %s\n\n", i+1, step.Title))
			if step.Description != "" {
				b.WriteString("**Description:** " + step.Description + "\n\n")
			}
			b.WriteString("**Deliverables:**\n\n")
			b.WriteString(fmt.Sprintf("- Completed implementation for phase %d.\n", i+1))
			b.WriteString("- All tests passing with zero failures.\n")
			b.WriteString("- Documentation updated.\n\n")
			b.WriteString("**Exit Criteria:**\n\n")
			b.WriteString("- [ ] Tests: 0 failures.\n")
			b.WriteString("- [ ] Linter: 0 issues.\n")
			b.WriteString("- [ ] Build: 0 errors.\n")
			b.WriteString("- [ ] No open critical risks.\n\n")
		}
	} else {
		b.WriteString("_Phases will appear here as the execution plan is developed._\n\n")
	}

	// ── § 4. Dependencies ────────────────────────────────────────────────────
	b.WriteString("## 4. Dependencies\n\n")
	b.WriteString("### Cross-Phase Dependency Graph\n\n")
	b.WriteString("```\n")
	if len(s.ExecutionPlan) > 0 {
		b.WriteString("[START]\n")
		for i, step := range s.ExecutionPlan {
			indent := strings.Repeat("  ", i)
			b.WriteString(fmt.Sprintf("%s  └─► [Phase %d: %s]\n", indent, i+1, step.Title))
		}
		b.WriteString(fmt.Sprintf("%s  └─► [DONE]\n", strings.Repeat("  ", len(s.ExecutionPlan))))
	} else {
		b.WriteString("[START] ──► [...] ──► [DONE]\n")
	}
	b.WriteString("```\n\n")
	b.WriteString("### External Dependencies\n\n")
	b.WriteString(renderTable(
		[]string{"Dependency", "Type", "Version", "Purpose"},
		[][]string{
			{"PostgreSQL", "Infrastructure", "16+", "Primary data store"},
			{"Go toolchain", "Build", "1.26+", "Backend + agent compilation"},
			{"Node.js", "Build", "20 LTS", "Frontend build toolchain"},
			{"Docker", "Ops", "24+", "Container build + compose"},
			{"a2a-go", "Library", "v2", "A2A protocol implementation"},
		},
	))

	// ── § 5. Risks & Mitigations ─────────────────────────────────────────────
	b.WriteString("## 5. Risks & Mitigations\n\n")
	b.WriteString(renderRisksTable(s))

	// ── § 6. Assumptions ─────────────────────────────────────────────────────
	b.WriteString("## 6. Assumptions\n\n")
	if len(s.Assumptions) > 0 {
		for _, a := range s.Assumptions {
			b.WriteString(fmt.Sprintf("- %s\n", a))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("_No assumptions recorded yet._\n\n")
	}

	// ── § 7. Validation Strategy ─────────────────────────────────────────────
	b.WriteString("## 7. Validation Strategy\n\n")
	b.WriteString("### Per-Phase Quality Gates\n\n")
	b.WriteString("Each phase must pass these gates before the next phase begins:\n\n")
	b.WriteString("| Gate | Command | Pass Condition |\n")
	b.WriteString("|------|---------|----------------|\n")
	b.WriteString("| Unit tests | `go test ./...` | 0 failures |\n")
	b.WriteString("| Frontend tests | `pnpm test` | 0 failures |\n")
	b.WriteString("| Security review | Manual OWASP check | 0 critical findings |\n")
	b.WriteString("| Linter | `go vet ./...` | 0 issues |\n")
	b.WriteString("| Frontend linter | `pnpm lint` | 0 errors |\n")
	b.WriteString("| Build | `go build ./...` | 0 errors |\n")
	b.WriteString("| Frontend build | `pnpm build` | 0 errors |\n\n")
	if len(s.ExecutionPlan) > 0 {
		b.WriteString("### Per-Step Validation\n\n")
		for i, step := range s.ExecutionPlan {
			b.WriteString(fmt.Sprintf("**Step %d — %s:**\n\n", i+1, step.Title))
			b.WriteString("- Unit tests covering all new functions.\n")
			b.WriteString("- Integration test: end-to-end flow with local services.\n")
			b.WriteString("- Determinism test: same input → same output (2+ runs).\n\n")
		}
	}

	// ── § 8. Rollout Plan ────────────────────────────────────────────────────
	b.WriteString("## 8. Rollout Plan\n\n")
	b.WriteString("### Staged Delivery\n\n")
	b.WriteString("| Stage | Target | Criteria |\n")
	b.WriteString("|-------|--------|----------|\n")
	b.WriteString("| Alpha | Internal team | All tests green; core feature complete |\n")
	b.WriteString("| Beta | Invited users | Performance targets met; UX feedback collected |\n")
	b.WriteString("| GA | All users | SLA 99.5%; docs published; rollback tested |\n\n")

	// Open Questions
	if len(s.OpenQuestions) > 0 {
		b.WriteString("## Open Questions\n\n")
		for _, q := range s.OpenQuestions {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", q))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("---\n_Generated at iteration %d. Confidence: %.4f_\n",
		s.Meta.Iteration, s.Metrics.Confidence))

	body := b.String()
	return enforceMinLines(body, s, padRoadmap), nil
}

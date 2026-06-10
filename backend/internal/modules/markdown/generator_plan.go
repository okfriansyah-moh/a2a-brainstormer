// Package markdown — consolidated plan document generator (§8.29.5).
// Roadmap milestones and phase breakdown are absorbed into plan.md.
package markdown

import (
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GeneratePlan renders the consolidated implementation plan from s.
func GeneratePlan(s state.CanonicalState) (string, error) {
	var b strings.Builder
	title := shortTitle(s)

	b.WriteString(fmt.Sprintf("# %s — Implementation Plan\n\n", title))
	b.WriteString(fmt.Sprintf("> %s\n\n", oneLineDescription(s)))
	b.WriteString(fmt.Sprintf("> Iteration **%d** · Confidence **%.4f**\n\n",
		s.Meta.Iteration, s.Metrics.Confidence))

	// ── § 1. Goals ──────────────────────────────────────────────────────────
	b.WriteString("## 1. Goals\n\n")
	if v, ok := s.Idea["goals"]; ok {
		if goals := stringsFromAny(v); len(goals) > 0 {
			for _, g := range goals {
				b.WriteString(fmt.Sprintf("- %s\n", g))
			}
			b.WriteString("\n")
		} else {
			b.WriteString(fmt.Sprintf("%v\n\n", v))
		}
	} else if mvp, ok := s.Idea["mvp_must_haves"]; ok {
		for _, item := range stringsFromAny(mvp) {
			b.WriteString(fmt.Sprintf("- %s\n", item))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("_Goals not yet defined by the agents._\n\n")
	}

	// ── § 2. Milestones ─────────────────────────────────────────────────────
	b.WriteString("## 2. Milestones\n\n")
	if len(s.ExecutionPlan) == 0 {
		b.WriteString("_No execution plan steps recorded yet._\n\n")
	} else {
		rows := make([][]string, 0, len(s.ExecutionPlan))
		for i, step := range s.ExecutionPlan {
			desc := step.Description
			if len(desc) > 100 {
				desc = desc[:97] + "..."
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", i+1),
				step.Title,
				desc,
			})
		}
		b.WriteString(renderTable([]string{"#", "Phase", "Summary"}, rows))
	}

	// ── § 3. Phase Breakdown ────────────────────────────────────────────────
	b.WriteString("## 3. Phase Breakdown\n\n")
	b.WriteString(renderStructuredPhases(s))

	// ── § 4. Cross-phase Dependencies ───────────────────────────────────────
	b.WriteString("## 4. Cross-phase Dependencies\n\n")
	b.WriteString(renderCrossPhaseDependencies(s))

	// ── § 5. Module / Task Breakdown ────────────────────────────────────────
	b.WriteString("## 5. Module Tasks\n\n")
	b.WriteString(renderModuleTaskBreakdown(s))

	// ── § 6. Risks ──────────────────────────────────────────────────────────
	b.WriteString("## 6. Risks\n\n")
	b.WriteString(renderRisksTable(s))

	if len(s.OpenQuestions) > 0 {
		b.WriteString("### Open Questions\n\n")
		for _, q := range s.OpenQuestions {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", q))
		}
		b.WriteString("\n")
	}

	b.WriteString(renderForAIAgentsAppendix(s, "plan"))

	return b.String(), nil
}

func renderCrossPhaseDependencies(s state.CanonicalState) string {
	if len(s.ExecutionPlan) == 0 {
		return "_No cross-phase dependencies recorded yet._\n\n"
	}
	var b strings.Builder
	rows := make([][]string, 0, len(s.ExecutionPlan))
	for i, step := range s.ExecutionPlan {
		deps := "_none_"
		if len(step.BlockingDependencies) > 0 {
			deps = strings.Join(step.BlockingDependencies, ", ")
		}
		rows = append(rows, []string{
			fmt.Sprintf("Phase %d", i+1),
			step.Title,
			deps,
		})
	}
	b.WriteString(renderTable([]string{"Phase", "Title", "Blocking Dependencies"}, rows))
	return b.String()
}

func renderModuleTaskBreakdown(s state.CanonicalState) string {
	if comps, ok := s.Architecture["components"]; ok {
		items := stringsFromAny(comps)
		if len(items) > 0 {
			var b strings.Builder
			for _, c := range items {
				b.WriteString(fmt.Sprintf("### %s\n\n", c))
				b.WriteString("- **Files to create:** _to be refined by agents_\n")
				b.WriteString("- **Validation:** `go test ./...`\n\n")
			}
			return b.String()
		}
	}
	if len(s.ExecutionPlan) > 0 {
		var b strings.Builder
		for i, step := range s.ExecutionPlan {
			b.WriteString(fmt.Sprintf("### Task %d — %s\n\n", i+1, step.Title))
			b.WriteString(fmt.Sprintf("%s\n\n", step.Description))
			b.WriteString("- **Files to create:** _see phase deliverables_\n")
			b.WriteString("- **Validation:** per module test suite\n\n")
		}
		return b.String()
	}
	return "_Module and task breakdown will appear once agents populate the architecture._\n\n"
}

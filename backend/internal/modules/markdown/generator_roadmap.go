// Package markdown — roadmap document generator (§8.23).
package markdown

import (
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GenerateRoadmap renders the roadmap document from s. The title uses the
// project short title, the idea appears once as a blockquote, and each
// execution_plan step becomes a structured phase block.
func GenerateRoadmap(s state.CanonicalState) (string, error) {
	var b strings.Builder
	title := shortTitle(s)

	b.WriteString(fmt.Sprintf("# %s — Roadmap\n\n", title))
	b.WriteString(fmt.Sprintf("> %s\n\n", oneLineDescription(s)))
	b.WriteString(fmt.Sprintf("> Iteration **%d** · Confidence **%.4f**\n\n",
		s.Meta.Iteration, s.Metrics.Confidence))

	// ── § 1. Goal ───────────────────────────────────────────────────────────
	b.WriteString("## 1. Goal\n\n")
	if v, ok := s.Idea["goals"]; ok {
		if goals := stringsFromAny(v); len(goals) > 0 {
			for _, g := range goals {
				b.WriteString(fmt.Sprintf("- %s\n", g))
			}
			b.WriteString("\n")
		} else {
			b.WriteString(fmt.Sprintf("%v\n\n", v))
		}
	} else {
		b.WriteString("_Goals will appear here once the agents populate the idea._\n\n")
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

	// ── § 4. Risks & Mitigations ────────────────────────────────────────────
	b.WriteString("## 4. Risks & Mitigations\n\n")
	b.WriteString(renderRisksTable(s))

	// ── § 5. Assumptions ────────────────────────────────────────────────────
	b.WriteString("## 5. Assumptions\n\n")
	if len(s.Assumptions) > 0 {
		for _, a := range s.Assumptions {
			b.WriteString(fmt.Sprintf("- %s\n", a))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("_No assumptions recorded yet._\n\n")
	}

	// ── § 6. Open Questions ─────────────────────────────────────────────────
	if len(s.OpenQuestions) > 0 {
		b.WriteString("## 6. Open Questions\n\n")
		for _, q := range s.OpenQuestions {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", q))
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

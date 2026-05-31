// Package markdown — implementation plan document generator (§8.23).
package markdown

import (
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GeneratePlan renders the implementation plan document from s. Title uses
// the project short title; the idea appears once as a blockquote.
func GeneratePlan(s state.CanonicalState) (string, error) {
	var b strings.Builder
	title := shortTitle(s)

	b.WriteString(fmt.Sprintf("# %s — Implementation Plan\n\n", title))
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
		b.WriteString("_Goals not yet defined by the agents._\n\n")
	}

	// ── § 2. Architecture Overview ──────────────────────────────────────────
	b.WriteString("## 2. Architecture Overview\n\n")
	if mermaid := renderDataFlowsMermaid(s); mermaid != "" {
		b.WriteString(mermaid)
	} else {
		b.WriteString(renderASCIIComponents(s))
	}
	b.WriteString("### Architecture Decisions\n\n")
	b.WriteString(renderDecisionsTable(s))

	// ── § 3. Tech Stack ─────────────────────────────────────────────────────
	b.WriteString("## 3. Tech Stack\n\n")
	b.WriteString(renderTechStack(s))

	// ── § 4. Project Structure ──────────────────────────────────────────────
	b.WriteString("## 4. Project Structure\n\n")
	b.WriteString(renderDirectoryTree(s))

	// ── § 5. Implementation Tasks ───────────────────────────────────────────
	b.WriteString("## 5. Implementation Tasks\n\n")
	b.WriteString(renderStructuredPhases(s))

	// ── § 6. Risks ──────────────────────────────────────────────────────────
	b.WriteString("## 6. Risks\n\n")
	b.WriteString(renderRisksTable(s))

	// ── § 7. Open Questions ─────────────────────────────────────────────────
	if len(s.OpenQuestions) > 0 {
		b.WriteString("## 7. Open Questions\n\n")
		for _, q := range s.OpenQuestions {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", q))
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

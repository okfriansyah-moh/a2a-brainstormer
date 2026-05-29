// Package markdown — architecture document generator (§8.23).
package markdown

import (
	"fmt"
	"slices"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GenerateArchitecture renders the architecture document from s.
// The title is the project short title (not the full idea text); the idea
// appears exactly once as a blockquote one-line description.
func GenerateArchitecture(s state.CanonicalState) (string, error) {
	var b strings.Builder
	title := shortTitle(s)

	// ── Title + one-line description ────────────────────────────────────────
	b.WriteString(fmt.Sprintf("# %s — Architecture\n\n", title))
	b.WriteString(fmt.Sprintf("> %s\n\n", oneLineDescription(s)))
	b.WriteString(fmt.Sprintf("> Iteration **%d** · Confidence **%.4f**\n\n",
		s.Meta.Iteration, s.Metrics.Confidence))

	// ── § 1. Overview ───────────────────────────────────────────────────────
	b.WriteString("## 1. Overview\n\n")
	if v, ok := s.Idea["context"]; ok {
		b.WriteString(fmt.Sprintf("%v\n\n", v))
	}
	if v, ok := s.Idea["goals"]; ok {
		b.WriteString("**Goals:**\n\n")
		if goals := stringsFromAny(v); len(goals) > 0 {
			for _, g := range goals {
				b.WriteString(fmt.Sprintf("- %s\n", g))
			}
			b.WriteString("\n")
		} else {
			b.WriteString(fmt.Sprintf("%v\n\n", v))
		}
	}

	// ── § 2. Layers ─────────────────────────────────────────────────────────
	b.WriteString("## 2. Layers\n\n")
	if structured := renderArchitectureLayers(s); structured != "" {
		b.WriteString(structured)
	} else if len(s.Architecture) > 0 {
		// Fallback: iterate the architecture map.
		keys := make([]string, 0, len(s.Architecture))
		for k := range s.Architecture {
			if k == "layers" || k == "data_flows" || k == "tech_stack" ||
				k == "decisions" || k == "directory_layout" || k == "config" {
				continue
			}
			keys = append(keys, k)
		}
		slices.Sort(keys)
		if len(keys) == 0 {
			b.WriteString("_Architecture details not yet defined._\n\n")
		}
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("### %s\n\n", k))
			b.WriteString(fmt.Sprintf("%v\n\n", s.Architecture[k]))
		}
	} else {
		b.WriteString("_Architecture details not yet defined._\n\n")
	}

	// ── § 3. Tech Stack ─────────────────────────────────────────────────────
	b.WriteString("## 3. Tech Stack\n\n")
	b.WriteString(renderTechStack(s))

	// ── § 4. Data Flows ─────────────────────────────────────────────────────
	b.WriteString("## 4. Data Flows\n\n")
	if mermaid := renderDataFlowsMermaid(s); mermaid != "" {
		b.WriteString(mermaid)
	} else {
		b.WriteString(renderASCIIComponents(s))
	}

	// ── § 5. Module Boundaries ──────────────────────────────────────────────
	b.WriteString("## 5. Module Boundaries\n\n")
	b.WriteString("### Directory Structure\n\n")
	b.WriteString(renderDirectoryTree(s))

	// ── § 6. Architecture Decisions ─────────────────────────────────────────
	b.WriteString("## 6. Architecture Decisions\n\n")
	b.WriteString(renderDecisionsTable(s))

	// ── § 7. Quality Targets ────────────────────────────────────────────────
	b.WriteString("## 7. Quality Targets\n\n")
	rows := [][]string{
		{"Confidence", fmt.Sprintf("%.4f", s.Metrics.Confidence)},
	}
	if s.Metrics.TestCoverageTarget > 0 {
		rows = append(rows, []string{"Test coverage target", fmt.Sprintf("%.1f%%", s.Metrics.TestCoverageTarget*100)})
	}
	if s.Metrics.LatencyBudgetMs > 0 {
		rows = append(rows, []string{"Latency budget", fmt.Sprintf("%d ms", s.Metrics.LatencyBudgetMs)})
	}
	b.WriteString(renderTable([]string{"Metric", "Value"}, rows))

	// ── § 8. Risks ──────────────────────────────────────────────────────────
	b.WriteString("## 8. Risks\n\n")
	b.WriteString(renderRisksTable(s))

	// ── § 9. Assumptions ────────────────────────────────────────────────────
	b.WriteString("## 9. Assumptions\n\n")
	if len(s.Assumptions) > 0 {
		for _, a := range s.Assumptions {
			b.WriteString(fmt.Sprintf("- %s\n", a))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("_No assumptions recorded._\n\n")
	}

	// ── § 10. Open Questions ────────────────────────────────────────────────
	b.WriteString("## 10. Open Questions\n\n")
	if len(s.OpenQuestions) > 0 {
		for _, q := range s.OpenQuestions {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", q))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("_No open questions at this time._\n\n")
	}

	return b.String(), nil
}

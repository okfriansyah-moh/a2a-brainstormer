// Package markdown — README document generator (§8.23).
package markdown

import (
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GenerateReadme renders the README document from s. The title uses the
// project short title; the project description appears exactly once as a
// blockquote one-line description.
func GenerateReadme(s state.CanonicalState) (string, error) {
	var b strings.Builder
	title := shortTitle(s)

	b.WriteString(fmt.Sprintf("# %s — README\n\n", title))
	b.WriteString(fmt.Sprintf("> %s\n\n", oneLineDescription(s)))

	// ── Overview ────────────────────────────────────────────────────────────
	b.WriteString("## Overview\n\n")
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

	// ── Architecture Summary ────────────────────────────────────────────────
	b.WriteString("## Architecture\n\n")
	if mermaid := renderDataFlowsMermaid(s); mermaid != "" {
		b.WriteString(mermaid)
	} else {
		b.WriteString(renderASCIIComponents(s))
	}

	// ── Tech Stack ──────────────────────────────────────────────────────────
	b.WriteString("## Tech Stack\n\n")
	b.WriteString(renderTechStack(s))

	// ── Project Structure ───────────────────────────────────────────────────
	b.WriteString("## Project Structure\n\n")
	b.WriteString(renderDirectoryTree(s))

	// ── Quick Start ─────────────────────────────────────────────────────────
	b.WriteString("## Quick Start\n\n")
	b.WriteString("```bash\n")
	b.WriteString("# Clone, install dependencies, run.\n")
	b.WriteString("git clone <repository-url> && cd <project>\n")
	b.WriteString("```\n\n")

	// ── Configuration ───────────────────────────────────────────────────────
	b.WriteString("## Configuration\n\n")
	b.WriteString(renderEnvVarList(s))

	// ── Roadmap summary ─────────────────────────────────────────────────────
	b.WriteString("## Roadmap\n\n")
	if len(s.ExecutionPlan) == 0 {
		b.WriteString("_Roadmap will appear here once phases are defined._\n\n")
	} else {
		limit := len(s.ExecutionPlan)
		if limit > 8 {
			limit = 8
		}
		for i := 0; i < limit; i++ {
			step := s.ExecutionPlan[i]
			if step.Description != "" {
				b.WriteString(fmt.Sprintf("- **Phase %d — %s:** %s\n", i+1, step.Title, step.Description))
			} else {
				b.WriteString(fmt.Sprintf("- **Phase %d — %s**\n", i+1, step.Title))
			}
		}
		if len(s.ExecutionPlan) > limit {
			b.WriteString(fmt.Sprintf("- _… and %d more phase(s) — see the full roadmap document._\n",
				len(s.ExecutionPlan)-limit))
		}
		b.WriteString("\n")
	}

	// ── Risks ───────────────────────────────────────────────────────────────
	b.WriteString("## Known Risks\n\n")
	b.WriteString(renderRisksTable(s))

	// ── Status footer ───────────────────────────────────────────────────────
	b.WriteString(fmt.Sprintf("---\n_Iteration %d · Confidence %.4f_\n",
		s.Meta.Iteration, s.Metrics.Confidence))

	return b.String(), nil
}

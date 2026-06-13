// Package markdown — architecture document generator (§8.30).
// Produces a 16-section publication-quality architecture.md from CanonicalState.
package markdown

import (
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GenerateArchitecture renders the 16-section architecture document from s.
// Output is fully deterministic: same state input → identical output.
// The H1 format is always: # {title} — Architecture
func GenerateArchitecture(s state.CanonicalState) (string, error) {
	var b strings.Builder
	title := shortTitle(s)

	// ── H1 + metadata block ──────────────────────────────────────────────────
	b.WriteString(fmt.Sprintf("# %s — Architecture\n\n", title))
	b.WriteString(renderMetadataBlock(s))

	// ── Table of Contents ────────────────────────────────────────────────────
	b.WriteString(renderTableOfContents())

	// ── § 1. Problem Statement ───────────────────────────────────────────────
	b.WriteString("## 1. Problem Statement\n\n")
	b.WriteString(renderProblemStatement(s))

	// ── § 2. Solution ────────────────────────────────────────────────────────
	b.WriteString("## 2. Solution\n\n")
	b.WriteString(renderSolution(s))

	// ── § 3. Scope ───────────────────────────────────────────────────────────
	b.WriteString("## 3. Scope\n\n")
	b.WriteString(renderScope(s))

	// ── § 4. Layers ──────────────────────────────────────────────────────────
	b.WriteString("## 4. Layers\n\n")
	if structured := renderArchitectureLayers(s); structured != "" {
		b.WriteString(structured)
	} else {
		b.WriteString("_Architecture details not yet defined._\n\n")
	}

	// ── § 5. Tech Stack ──────────────────────────────────────────────────────
	b.WriteString("## 5. Tech Stack\n\n")
	b.WriteString(renderEnrichedTechStack(s))

	// ── § 6. Data Flows ──────────────────────────────────────────────────────
	b.WriteString("## 6. Data Flows\n\n")
	b.WriteString(renderDataFlowsWithProse(s))

	// ── § 7. Module Boundaries ───────────────────────────────────────────────
	b.WriteString("## 7. Module Boundaries\n\n")
	b.WriteString("### Directory Structure\n\n")
	b.WriteString(renderDirectoryTree(s))

	// ── § 8. Architecture Decisions ──────────────────────────────────────────
	b.WriteString("## 8. Architecture Decisions\n\n")
	b.WriteString(renderEnrichedDecisionsTable(s))

	// ── § 9. Extension Points ────────────────────────────────────────────────
	b.WriteString("## 9. Extension Points\n\n")
	b.WriteString(renderExtensionPoints(s))

	// ── § 10. Security Considerations ───────────────────────────────────────
	b.WriteString("## 10. Security Considerations\n\n")
	b.WriteString(renderSecurityConsiderations(s))

	// ── § 11. Quality Targets ────────────────────────────────────────────────
	b.WriteString("## 11. Quality Targets\n\n")
	rows := [][]string{
		{"Confidence", fmt.Sprintf("%.4f", s.Metrics.Confidence)},
	}
	if s.Metrics.TestCoverageTarget > 0 {
		rows = append(rows, []string{"Test coverage target", fmt.Sprintf("%.1f%%", s.Metrics.TestCoverageTarget*100)})
	}
	if s.Metrics.LatencyBudgetMs > 0 {
		rows = append(rows, []string{"Latency budget", fmt.Sprintf("%d ms", s.Metrics.LatencyBudgetMs)})
	}
	rows = append(rows, []string{"Coding Standards", "AGENTS.md module boundary rules"})
	b.WriteString(renderTable([]string{"Metric", "Value"}, rows))

	// ── § 12. System Guarantees ──────────────────────────────────────────────
	b.WriteString("## 12. System Guarantees\n\n")
	b.WriteString(renderSystemGuarantees(s))

	// ── § 13. Risks ──────────────────────────────────────────────────────────
	b.WriteString("## 13. Risks\n\n")
	b.WriteString(renderEnrichedRisksTable(s))

	// ── § 14. Assumptions ───────────────────────────────────────────────────
	b.WriteString("## 14. Assumptions\n\n")
	if len(s.Assumptions) > 0 {
		for _, a := range s.Assumptions {
			b.WriteString(fmt.Sprintf("- %s\n", a))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("_No assumptions recorded._\n\n")
	}

	// ── § 15. Open Questions ─────────────────────────────────────────────────
	b.WriteString("## 15. Open Questions\n\n")
	if len(s.OpenQuestions) > 0 {
		for _, q := range s.OpenQuestions {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", q))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("_No open questions at this time._\n\n")
	}

	// ── § 16. Definition of Done ─────────────────────────────────────────────
	b.WriteString("## 16. Definition of Done\n\n")
	b.WriteString(renderDefinitionOfDone(s))

	// ── For AI Agents appendix ───────────────────────────────────────────────
	b.WriteString(renderForAIAgentsAppendix(s, "architecture"))

	return b.String(), nil
}

// Package markdown — shared rendering helpers used by all four document
// generators. All functions are pure and deterministic: same input →
// identical output, no randomness, no time.Now(), no map iteration without
// key sorting.
package markdown

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"a2a-brainstorm/backend/internal/modules/state"
)

// suffixForKey maps an output-doc key to the lowercase filename suffix.
// All suffixes are lowercase per §8.23 (no "PLAN.md" / "README.md" uppercase).
var suffixForKey = map[string]string{
	"architecture": "architecture.md",
	"roadmap":      "roadmap.md",
	"plan":         "plan.md",
	"readme":       "readme.md",
}

// ── title / slug helpers ─────────────────────────────────────────────────────

// shortTitle returns a concise human-readable title for the project.
// Priority: s.Idea["name"] → first sentence of s.Idea["text"] truncated at the
// nearest word boundary up to 60 runes → "Untitled Brainstorm".
func shortTitle(s state.CanonicalState) string {
	if name, ok := s.Idea["name"]; ok {
		if str := strings.TrimSpace(fmt.Sprintf("%v", name)); str != "" {
			return truncateAtWord(str, 60)
		}
	}
	if text, ok := s.Idea["text"]; ok {
		raw := strings.TrimSpace(fmt.Sprintf("%v", text))
		if raw != "" {
			cut := strings.IndexAny(raw, ".!?\n")
			if cut > 0 {
				raw = raw[:cut]
			}
			return truncateAtWord(strings.TrimSpace(raw), 60)
		}
	}
	return "Untitled Brainstorm"
}

// truncateAtWord cuts s at the nearest word boundary ≤ max runes.
// Returns s unchanged if it is already short enough.
func truncateAtWord(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	cut := max
	for cut > 0 && !unicode.IsSpace(runes[cut]) {
		cut--
	}
	if cut == 0 {
		cut = max
	}
	return strings.TrimRight(string(runes[:cut]), " \t,;:-")
}

// oneLineDescription returns a single-line summary suitable for a Markdown
// blockquote / overview paragraph. Priority: s.Idea["summary"] →
// s.Idea["description"] → s.Idea["text"] → "A brainstorm project."
// Any embedded newlines are collapsed to a single space.
// Output is clamped to 150 chars + "…" to prevent paragraph-length blockquotes.
func oneLineDescription(s state.CanonicalState) string {
	for _, key := range []string{"summary", "description", "text"} {
		if v, ok := s.Idea[key]; ok {
			str := strings.TrimSpace(fmt.Sprintf("%v", v))
			if str != "" {
				joined := strings.Join(strings.Fields(str), " ")
				return truncateChars(joined, 150)
			}
		}
	}
	return "A brainstorm project."
}

// slugify converts an arbitrary title to a filesystem-safe slug:
// lowercase, ASCII alphanumerics + '-', collapsed repeats, trimmed, ≤ 50.
// Empty input → "untitled".
func slugify(title string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 50 {
		out = strings.TrimRight(out[:50], "-")
	}
	if out == "" {
		return "untitled"
	}
	return out
}

// buildFilename composes "{slug}_{suffix}" using the given title and key.
// Unknown keys fall back to "{key}.md".
func buildFilename(title, key string) string {
	suffix, ok := suffixForKey[key]
	if !ok {
		suffix = key + ".md"
	}
	return slugify(title) + "_" + suffix
}

// ── shared rendering helpers ─────────────────────────────────────────────────

// writeMap writes the key-value pairs of a map[string]any as Markdown bullet
// points into b. Keys are sorted for deterministic output.
func writeMap(b *strings.Builder, m map[string]any) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("- **%s**: %v\n", k, m[k]))
	}
	b.WriteString("\n")
}

// renderTable returns a Markdown table with the given headers and rows.
// All columns are left-aligned. Rows with fewer columns than headers are
// right-padded with empty strings.
func renderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = "---"
	}
	b.WriteString("| " + strings.Join(sep, " | ") + " |\n")
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i := range cells {
			if i < len(row) {
				cells[i] = row[i]
			}
		}
		b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	}
	b.WriteString("\n")
	return b.String()
}

// renderASCIIComponents produces a simple text box diagram that lists all
// keys in s.Architecture as labelled boxes connected left-to-right.
// When s.Architecture is empty a placeholder is returned.
func renderASCIIComponents(s state.CanonicalState) string {
	if len(s.Architecture) == 0 {
		return "```\n[ No components defined ]\n```\n\n"
	}
	keys := make([]string, 0, len(s.Architecture))
	for k := range s.Architecture {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	var b strings.Builder
	b.WriteString("```\n")
	for i, k := range keys {
		width := len(k) + 2
		top := "+" + strings.Repeat("-", width) + "+"
		mid := "| " + k + " |"
		b.WriteString(top + "\n")
		b.WriteString(mid + "\n")
		b.WriteString(top + "\n")
		if i < len(keys)-1 {
			b.WriteString("       |\n")
			b.WriteString("       v\n")
		}
	}
	b.WriteString("```\n\n")
	return b.String()
}

// renderDirectoryTree renders the value stored under key "directory_layout"
// in s.Architecture as a fenced code block. Falls back to a placeholder.
func renderDirectoryTree(s state.CanonicalState) string {
	if v, ok := s.Architecture["directory_layout"]; ok {
		if m, ok := v.(map[string]any); ok && len(m) > 0 {
			var b strings.Builder
			b.WriteString("```\n")
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for i, k := range keys {
				if i == len(keys)-1 {
					b.WriteString(fmt.Sprintf("└── %s/\n", k))
				} else {
					b.WriteString(fmt.Sprintf("├── %s/\n", k))
				}
				if sub, ok := m[k].(map[string]any); ok {
					subKeys := make([]string, 0, len(sub))
					for sk := range sub {
						subKeys = append(subKeys, sk)
					}
					slices.Sort(subKeys)
					prefix := "│   "
					if i == len(keys)-1 {
						prefix = "    "
					}
					for j, sk := range subKeys {
						if j == len(subKeys)-1 {
							b.WriteString(fmt.Sprintf("%s└── %s\n", prefix, sk))
						} else {
							b.WriteString(fmt.Sprintf("%s├── %s\n", prefix, sk))
						}
					}
				}
			}
			b.WriteString("```\n\n")
			return b.String()
		}
		str := fmt.Sprintf("%v", v)
		if strings.HasPrefix(str, "map[") {
			// fall through to architecture-keys-based tree
		} else {
			return fmt.Sprintf("```\n%s\n```\n\n", str)
		}
	}
	if len(s.Architecture) == 0 {
		return "```\n<directory structure not yet defined>\n```\n\n"
	}
	keys := make([]string, 0, len(s.Architecture))
	for k := range s.Architecture {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	b.WriteString("```\n./\n")
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("├── %s/\n", k))
	}
	b.WriteString("└── docs/\n")
	b.WriteString("```\n\n")
	return b.String()
}

// renderTechStack produces a Markdown section listing all technology entries
// found in s.Architecture["tech_stack"] or falls back to the raw Architecture
// map when no tech_stack key exists.
func renderTechStack(s state.CanonicalState) string {
	var b strings.Builder
	if ts, ok := s.Architecture["tech_stack"]; ok {
		if m, ok := ts.(map[string]any); ok && len(m) > 0 {
			writeMap(&b, m)
		} else if strs := stringsFromAny(ts); len(strs) > 0 {
			for _, item := range strs {
				b.WriteString(fmt.Sprintf("- %s\n", item))
			}
			b.WriteString("\n")
		} else {
			str := fmt.Sprintf("%v", ts)
			if strings.HasPrefix(str, "map[") {
				b.WriteString("_Tech stack not yet defined._\n\n")
			} else {
				b.WriteString(str + "\n\n")
			}
		}
		return b.String()
	}
	if len(s.Architecture) == 0 {
		return "_Tech stack not yet defined._\n\n"
	}
	keys := make([]string, 0, len(s.Architecture))
	for k := range s.Architecture {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, []string{k, fmt.Sprintf("%v", s.Architecture[k]), "—"})
	}
	return renderTable([]string{"Layer", "Technology", "Version"}, rows)
}

// renderDecisionsTable produces a Markdown table of architecture decisions
// stored in s.Architecture["decisions"]. Falls back to a placeholder row.
func renderDecisionsTable(s state.CanonicalState) string {
	if v, ok := s.Architecture["decisions"]; ok {
		if strs := stringsFromAny(v); len(strs) > 0 {
			rows := make([][]string, len(strs))
			for i, d := range strs {
				rows[i] = []string{fmt.Sprintf("ADR-%03d", i+1), d, "Selected for fit", "Accepted"}
			}
			return renderTable([]string{"ID", "Decision", "Rationale", "Status"}, rows)
		}
		if m, ok := v.(map[string]any); ok && len(m) > 0 {
			var b strings.Builder
			writeMap(&b, m)
			return b.String()
		}
		str := fmt.Sprintf("%v", v)
		if !strings.HasPrefix(str, "map[") {
			return str + "\n\n"
		}
		return "_Architecture decisions not yet defined._\n\n"
	}
	if len(s.Architecture) == 0 {
		return renderTable(
			[]string{"ID", "Decision", "Rationale", "Status"},
			[][]string{{"ADR-001", "Architecture not yet defined", "—", "Draft"}},
		)
	}
	keys := make([]string, 0, len(s.Architecture))
	for k := range s.Architecture {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	rows := make([][]string, 0, len(keys))
	for i, k := range keys {
		rows = append(rows, []string{
			fmt.Sprintf("ADR-%03d", i+1),
			fmt.Sprintf("Use %s for %s layer", s.Architecture[k], k),
			"Selected for performance and ecosystem fit",
			"Accepted",
		})
	}
	return renderTable([]string{"ID", "Decision", "Rationale", "Status"}, rows)
}

// capitaliseSeverity title-cases a severity string without the unicode bug of
// strings.ToTitle. e.g. "medium" → "Medium", "HIGH" → "High".
func capitaliseSeverity(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// renderRisksTable produces a Markdown table of all risks in s.Risks.
// Resolved risks are marked with ✅; unresolved ones with ⚠️.
func renderRisksTable(s state.CanonicalState) string {
	if len(s.Risks) == 0 {
		return "_No risks identified yet._\n\n"
	}
	rows := make([][]string, 0, len(s.Risks))
	for i, r := range s.Risks {
		status := "⚠️ Open"
		if r.Resolved {
			status = "✅ Resolved"
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			r.Text,
			capitaliseSeverity(r.Severity),
			status,
		})
	}
	return renderTable([]string{"#", "Risk", "Severity", "Status"}, rows)
}

// renderExecutionPlanList produces a numbered Markdown list of execution plan
// steps with titles and descriptions.
func renderExecutionPlanList(s state.CanonicalState) string {
	if len(s.ExecutionPlan) == 0 {
		return "_No execution plan defined yet._\n\n"
	}
	var b strings.Builder
	for i, step := range s.ExecutionPlan {
		b.WriteString(fmt.Sprintf("%d. **%s**", i+1, step.Title))
		if step.Description != "" {
			b.WriteString(fmt.Sprintf(" — %s", step.Description))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

// renderEnvVarList produces a Markdown list of configuration / environment
// variables derived from s.Architecture["config"]. Falls back to a generic
// placeholder list when the config key is absent.
func renderEnvVarList(s state.CanonicalState) string {
	if v, ok := s.Architecture["config"]; ok {
		if m, ok := v.(map[string]any); ok && len(m) > 0 {
			var b strings.Builder
			b.WriteString("```env\n")
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, k := range keys {
				name := strings.ToUpper(strings.ReplaceAll(k, " ", "_"))
				b.WriteString(fmt.Sprintf("%s=%v\n", name, m[k]))
			}
			b.WriteString("```\n\n")
			return b.String()
		}
		str := fmt.Sprintf("%v", v)
		if strings.HasPrefix(str, "map[") {
			// fall through to generic env-var generation
		} else {
			return fmt.Sprintf("```env\n%s\n```\n\n", str)
		}
	}
	var b strings.Builder
	b.WriteString("```env\n")
	if len(s.Architecture) > 0 {
		keys := make([]string, 0, len(s.Architecture))
		for k := range s.Architecture {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			name := strings.ToUpper(strings.ReplaceAll(k, " ", "_"))
			b.WriteString(fmt.Sprintf("%s_HOST=localhost\n", name))
			b.WriteString(fmt.Sprintf("%s_PORT=8080\n", name))
		}
	} else {
		b.WriteString("APP_HOST=localhost\n")
		b.WriteString("APP_PORT=8080\n")
		b.WriteString("DATABASE_URL=postgres://user:pass@localhost:5432/db\n")
	}
	b.WriteString("```\n\n")
	return b.String()
}

// renderJSONDump serialises s to indented JSON inside a fenced code block.
// Errors in marshalling are silently ignored (the output falls back to empty).
func renderJSONDump(s state.CanonicalState) string {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "```json\n{}\n```\n\n"
	}
	return fmt.Sprintf("```json\n%s\n```\n\n", string(raw))
}

// ── structured-content helpers (§8.23) ──────────────────────────────────────

// stringsFromAny coerces a value into a []string when it is either []string,
// []any of strings, or a single string. Returns nil otherwise.
func stringsFromAny(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			out = append(out, fmt.Sprintf("%v", e))
		}
		return out
	case string:
		if x == "" {
			return nil
		}
		return []string{x}
	}
	return nil
}

// mapsFromAny coerces a value into a []map[string]any slice. Accepts
// []map[string]any or []any of map[string]any. Returns nil otherwise.
func mapsFromAny(v any) []map[string]any {
	switch x := v.(type) {
	case []map[string]any:
		return x
	case []any:
		out := make([]map[string]any, 0, len(x))
		for _, e := range x {
			if m, ok := e.(map[string]any); ok {
				out = append(out, m)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return nil
}

// renderArchitectureLayers renders structured s.Architecture["layers"] entries
// as a sequence of subsections with Responsibility / Technologies /
// Dependencies tables. Returns "" if no structured layers are present.
func renderArchitectureLayers(s state.CanonicalState) string {
	v, ok := s.Architecture["layers"]
	if !ok {
		return ""
	}
	layers := mapsFromAny(v)
	if len(layers) == 0 {
		return ""
	}
	var b strings.Builder
	for _, layer := range layers {
		name := strings.TrimSpace(fmt.Sprintf("%v", layer["name"]))
		if name == "" || name == "<nil>" {
			continue
		}
		b.WriteString(fmt.Sprintf("### %s\n\n", name))
		if desc, ok := layer["description"]; ok {
			str := strings.TrimSpace(fmt.Sprintf("%v", desc))
			if str != "" && str != "<nil>" {
				b.WriteString(str + "\n\n")
			}
		}
		if resp, ok := layer["responsibility"]; ok {
			b.WriteString(fmt.Sprintf("**Responsibility:** %v\n\n", resp))
		}
		if tech := stringsFromAny(layer["technologies"]); len(tech) > 0 {
			b.WriteString("**Technologies:**\n\n")
			for _, t := range tech {
				b.WriteString(fmt.Sprintf("- %s\n", t))
			}
			b.WriteString("\n")
		}
		if deps := stringsFromAny(layer["dependencies"]); len(deps) > 0 {
			b.WriteString("**Dependencies:**\n\n")
			for _, d := range deps {
				b.WriteString(fmt.Sprintf("- %s\n", d))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderDataFlowsMermaid renders s.Architecture["data_flows"] entries as a
// Mermaid `graph LR` block. Each entry must be a map with from/to (and
// optional label) keys. Returns "" if no structured flows are present.
func renderDataFlowsMermaid(s state.CanonicalState) string {
	v, ok := s.Architecture["data_flows"]
	if !ok {
		return ""
	}
	flows := mapsFromAny(v)
	if len(flows) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("```mermaid\n")
	b.WriteString("graph LR\n")
	wrote := false
	for _, f := range flows {
		from := strings.TrimSpace(fmt.Sprintf("%v", f["from"]))
		to := strings.TrimSpace(fmt.Sprintf("%v", f["to"]))
		if from == "" || to == "" || from == "<nil>" || to == "<nil>" {
			continue
		}
		label := ""
		if l, ok := f["label"]; ok {
			label = strings.TrimSpace(fmt.Sprintf("%v", l))
		}
		if label != "" && label != "<nil>" {
			b.WriteString(fmt.Sprintf("  %s -->|%s| %s\n", mermaidID(from), label, mermaidID(to)))
		} else {
			b.WriteString(fmt.Sprintf("  %s --> %s\n", mermaidID(from), mermaidID(to)))
		}
		wrote = true
	}
	b.WriteString("```\n\n")
	if !wrote {
		return ""
	}
	return b.String()
}

// mermaidID returns a Mermaid-safe node id followed by the quoted label.
func mermaidID(label string) string {
	id := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		default:
			return '_'
		}
	}, label)
	if id == "" {
		id = "node"
	}
	return fmt.Sprintf("%s[\"%s\"]", id, label)
}

// renderStructuredPhases renders s.ExecutionPlan entries that carry the
// §8.23 structured fields (objective / blocking_dependencies / scope /
// deliverables / function_contracts / failure_handling / exit_criteria).
// Steps missing these fields fall back to a minimal Title/Description block.
func renderStructuredPhases(s state.CanonicalState) string {
	if len(s.ExecutionPlan) == 0 {
		return "_No execution plan defined yet._\n\n"
	}
	var b strings.Builder
	for i, step := range s.ExecutionPlan {
		b.WriteString(fmt.Sprintf("### Phase %d — %s\n\n", i+1, step.Title))
		if step.Description != "" {
			b.WriteString(fmt.Sprintf("**Description:** %s\n\n", step.Description))
		}
		if step.Objective != "" {
			b.WriteString(fmt.Sprintf("**Objective:** %s\n\n", step.Objective))
		}
		if len(step.BlockingDependencies) > 0 {
			b.WriteString("**Blocking Dependencies:**\n\n")
			for _, d := range step.BlockingDependencies {
				b.WriteString(fmt.Sprintf("- %s\n", d))
			}
			b.WriteString("\n")
		}
		if step.Scope != "" {
			b.WriteString(fmt.Sprintf("**Scope:** %s\n\n", step.Scope))
		}
		if len(step.Deliverables) > 0 {
			b.WriteString("**Deliverables:**\n\n")
			for _, d := range step.Deliverables {
				b.WriteString(fmt.Sprintf("- %s\n", d))
			}
			b.WriteString("\n")
		}
		if len(step.FunctionContracts) > 0 {
			b.WriteString("**Function Contracts:**\n\n")
			for _, c := range step.FunctionContracts {
				b.WriteString(fmt.Sprintf("- `%s`\n", c))
			}
			b.WriteString("\n")
		}
		if step.FailureHandling != "" {
			b.WriteString(fmt.Sprintf("**Failure Handling:** %s\n\n", step.FailureHandling))
		}
		if len(step.ExitCriteria) > 0 {
			b.WriteString("**Exit Criteria:**\n\n")
			for _, c := range step.ExitCriteria {
				b.WriteString(fmt.Sprintf("- [ ] %s\n", c))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// ── string utilities ─────────────────────────────────────────────────────────

// truncateChars clamps s to max runes, appending "…" when truncated.
func truncateChars(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// firstSentence returns the text up to (but not including) the first sentence-
// ending punctuation (.!?), clamped to max chars. Falls back to truncateChars
// when no sentence boundary is found.
func firstSentence(s string, max int) string {
	cut := strings.IndexAny(s, ".!?\n")
	if cut > 0 {
		s = strings.TrimSpace(s[:cut])
	}
	return truncateChars(s, max)
}

// ── §8.30 16-section architecture helpers ────────────────────────────────────

// renderMetadataBlock returns the deterministic status/iteration/confidence
// blockquote that appears immediately after the H1.
func renderMetadataBlock(s state.CanonicalState) string {
	status := "Draft"
	if s.Metrics.Confidence >= 0.85 {
		status = "Converged"
	}
	return fmt.Sprintf("> **Status:** %s\n> **Iteration:** %d\n> **Confidence:** %.4f\n> **Generated:** —\n\n",
		status, s.Meta.Iteration, s.Metrics.Confidence)
}

// renderTableOfContents returns the hardcoded 16-entry TOC for the architecture
// document.
func renderTableOfContents() string {
	entries := []string{
		"1. [Problem Statement](#1-problem-statement)",
		"2. [Solution](#2-solution)",
		"3. [Scope](#3-scope)",
		"4. [Layers](#4-layers)",
		"5. [Tech Stack](#5-tech-stack)",
		"6. [Data Flows](#6-data-flows)",
		"7. [Module Boundaries](#7-module-boundaries)",
		"8. [Architecture Decisions](#8-architecture-decisions)",
		"9. [Extension Points](#9-extension-points)",
		"10. [Security Considerations](#10-security-considerations)",
		"11. [Quality Targets](#11-quality-targets)",
		"12. [System Guarantees](#12-system-guarantees)",
		"13. [Risks](#13-risks)",
		"14. [Assumptions](#14-assumptions)",
		"15. [Open Questions](#15-open-questions)",
		"16. [Definition of Done](#16-definition-of-done)",
	}
	var b strings.Builder
	b.WriteString("**Table of Contents**\n\n")
	for _, e := range entries {
		b.WriteString(e + "\n")
	}
	b.WriteString("\n---\n\n")
	return b.String()
}

// renderProblemStatement renders §1 with three-tier fallback.
// Priority: idea["problem_statement"] → first sentence of idea["context"] ≤200
// → idea["text"] ≤200.
func renderProblemStatement(s state.CanonicalState) string {
	if v, ok := s.Idea["problem_statement"]; ok {
		str := strings.TrimSpace(fmt.Sprintf("%v", v))
		if str != "" {
			return str + "\n\n"
		}
	}
	if v, ok := s.Idea["context"]; ok {
		str := strings.TrimSpace(fmt.Sprintf("%v", v))
		if str != "" {
			return firstSentence(str, 200) + "\n\n"
		}
	}
	if v, ok := s.Idea["text"]; ok {
		str := strings.TrimSpace(fmt.Sprintf("%v", v))
		if str != "" {
			return truncateChars(str, 200) + "\n\n"
		}
	}
	return "_Problem statement not yet defined._\n\n"
}

// renderSolution renders §2 with three-tier fallback.
// Priority: idea["solution_summary"] → idea["summary"] → first 300 chars of
// idea["text"].
func renderSolution(s state.CanonicalState) string {
	if v, ok := s.Idea["solution_summary"]; ok {
		str := strings.TrimSpace(fmt.Sprintf("%v", v))
		if str != "" {
			return str + "\n\n"
		}
	}
	if v, ok := s.Idea["summary"]; ok {
		str := strings.TrimSpace(fmt.Sprintf("%v", v))
		if str != "" {
			return str + "\n\n"
		}
	}
	if v, ok := s.Idea["text"]; ok {
		str := strings.TrimSpace(fmt.Sprintf("%v", v))
		if str != "" {
			return truncateChars(str, 300) + "\n\n"
		}
	}
	return "_Solution not yet defined._\n\n"
}

// renderScope renders §3 with In Scope / Out of Scope tables.
// Primary: s.Architecture["scope"] map with keys "in" and "out".
// Fallback: first 3 goals from idea["goals"]; out = fixed string.
func renderScope(s state.CanonicalState) string {
	var inItems, outItems []string

	if v, ok := s.Architecture["scope"]; ok {
		if m, ok := v.(map[string]any); ok {
			inItems = stringsFromAny(m["in"])
			outItems = stringsFromAny(m["out"])
		}
	}

	if len(inItems) == 0 {
		if v, ok := s.Idea["goals"]; ok {
			goals := stringsFromAny(v)
			if len(goals) > 3 {
				goals = goals[:3]
			}
			inItems = goals
		}
	}
	if len(outItems) == 0 {
		outItems = []string{"Further feature scope — defined in future iterations"}
	}

	var b strings.Builder

	// In Scope table
	b.WriteString("**In Scope**\n\n")
	inRows := make([][]string, len(inItems))
	for i, item := range inItems {
		inRows[i] = []string{fmt.Sprintf("%d", i+1), item}
	}
	b.WriteString(renderTable([]string{"#", "In Scope"}, inRows))

	// Out of Scope table
	b.WriteString("**Out of Scope**\n\n")
	outRows := make([][]string, len(outItems))
	for i, item := range outItems {
		outRows[i] = []string{fmt.Sprintf("%d", i+1), item}
	}
	b.WriteString(renderTable([]string{"#", "Out of Scope"}, outRows))

	return b.String()
}

// renderEnrichedTechStack renders §5 as a 4-column table when tech_stack_rationale
// is present, otherwise falls back to a 2-column table.
func renderEnrichedTechStack(s state.CanonicalState) string {
	var rationale map[string]string

	if v, ok := s.Architecture["tech_stack_rationale"]; ok {
		if m, ok := v.(map[string]any); ok && len(m) > 0 {
			rationale = make(map[string]string, len(m))
			for k, val := range m {
				rationale[k] = fmt.Sprintf("%v", val)
			}
		}
	}

	// Collect tech entries from tech_stack
	type techEntry struct {
		name    string
		role    string
		version string
	}
	var entries []techEntry

	if ts, ok := s.Architecture["tech_stack"]; ok {
		if m, ok := ts.(map[string]any); ok && len(m) > 0 {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, k := range keys {
				entries = append(entries, techEntry{name: k, role: fmt.Sprintf("%v", m[k])})
			}
		} else if strs := stringsFromAny(ts); len(strs) > 0 {
			for _, item := range strs {
				entries = append(entries, techEntry{name: item, role: "—"})
			}
		}
	}

	// Fallback: derive from Architecture keys
	if len(entries) == 0 {
		keys := make([]string, 0, len(s.Architecture))
		for k := range s.Architecture {
			switch k {
			case "layers", "data_flows", "decisions", "directory_layout", "config",
				"scope", "extension_points", "security", "guarantees",
				"decision_enrichments", "tech_stack_rationale", "data_flow_prose",
				"exit_criteria", "forbidden_patterns", "llm_interface_name":
				continue
			}
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			entries = append(entries, techEntry{name: k, role: fmt.Sprintf("%v", s.Architecture[k])})
		}
	}

	if len(entries) == 0 {
		return "_Tech stack not yet defined._\n\n"
	}

	if len(rationale) > 0 {
		// 4-column table
		rows := make([][]string, len(entries))
		for i, e := range entries {
			rat := "—"
			if r, ok := rationale[e.name]; ok {
				rat = r
			}
			ver := e.version
			if ver == "" {
				ver = "—"
			}
			rows[i] = []string{e.name, e.role, rat, ver}
		}
		return renderTable([]string{"Technology", "Role", "Rationale", "Version/Notes"}, rows)
	}

	// 2-column fallback
	rows := make([][]string, len(entries))
	for i, e := range entries {
		rows[i] = []string{e.name, e.role}
	}
	return renderTable([]string{"Technology", "Role"}, rows)
}

// renderDataFlowsWithProse renders §6 as optional prose paragraph followed by
// the Mermaid diagram (or ASCII fallback).
func renderDataFlowsWithProse(s state.CanonicalState) string {
	var b strings.Builder

	if v, ok := s.Architecture["data_flow_prose"]; ok {
		prose := strings.TrimSpace(fmt.Sprintf("%v", v))
		if prose != "" {
			b.WriteString(prose + "\n\n")
		}
	}

	if mermaid := renderDataFlowsMermaid(s); mermaid != "" {
		b.WriteString(mermaid)
	} else {
		b.WriteString(renderASCIIComponents(s))
	}

	return b.String()
}

// renderEnrichedDecisionsTable renders §8 as a 6-column table:
// # | Decision | Choice | Alternatives | Tradeoff | Status.
// Alternatives and Tradeoff come from architecture["decision_enrichments"].
func renderEnrichedDecisionsTable(s state.CanonicalState) string {
	// Build enrichment lookup by title
	enrichments := map[string]struct{ Alternatives, Tradeoff string }{}
	if v, ok := s.Architecture["decision_enrichments"]; ok {
		if entries := mapsFromAny(v); len(entries) > 0 {
			for _, e := range entries {
				title := strings.TrimSpace(fmt.Sprintf("%v", e["title"]))
				if title == "" {
					continue
				}
				enrichments[title] = struct{ Alternatives, Tradeoff string }{
					Alternatives: strings.TrimSpace(fmt.Sprintf("%v", e["alternatives"])),
					Tradeoff:     strings.TrimSpace(fmt.Sprintf("%v", e["tradeoff"])),
				}
			}
		}
	}

	if v, ok := s.Architecture["decisions"]; ok {
		if strs := stringsFromAny(v); len(strs) > 0 {
			rows := make([][]string, len(strs))
			for i, d := range strs {
				enr := enrichments[d]
				alt := enr.Alternatives
				if alt == "" {
					alt = "—"
				}
				trd := enr.Tradeoff
				if trd == "" {
					trd = "—"
				}
				rows[i] = []string{
					fmt.Sprintf("%d", i+1),
					d,
					"Selected for fit",
					alt,
					trd,
					"Accepted",
				}
			}
			return renderTable([]string{"#", "Decision", "Choice", "Alternatives", "Tradeoff", "Status"}, rows)
		}
		if m, ok := v.(map[string]any); ok && len(m) > 0 {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			rows := make([][]string, len(keys))
			for i, k := range keys {
				d := fmt.Sprintf("%v", m[k])
				enr := enrichments[d]
				alt := enr.Alternatives
				if alt == "" {
					alt = "—"
				}
				trd := enr.Tradeoff
				if trd == "" {
					trd = "—"
				}
				rows[i] = []string{
					fmt.Sprintf("%d", i+1),
					d,
					"Selected for fit",
					alt,
					trd,
					"Accepted",
				}
			}
			return renderTable([]string{"#", "Decision", "Choice", "Alternatives", "Tradeoff", "Status"}, rows)
		}
	}

	if len(s.Architecture) == 0 {
		return renderTable(
			[]string{"#", "Decision", "Choice", "Alternatives", "Tradeoff", "Status"},
			[][]string{{"1", "Architecture not yet defined", "—", "—", "—", "Draft"}},
		)
	}

	// Derive from architecture keys
	keys := make([]string, 0, len(s.Architecture))
	for k := range s.Architecture {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	rows := make([][]string, 0, len(keys))
	for i, k := range keys {
		d := fmt.Sprintf("Use %s for %s layer", s.Architecture[k], k)
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			d,
			"Selected for performance and ecosystem fit",
			"—",
			"—",
			"Accepted",
		})
	}
	return renderTable([]string{"#", "Decision", "Choice", "Alternatives", "Tradeoff", "Status"}, rows)
}

// renderExtensionPoints renders §9 from architecture["extension_points"].
// Falls back to 2 generic stub entries.
func renderExtensionPoints(s state.CanonicalState) string {
	type extPoint struct {
		Name  string
		Steps []string
	}

	var points []extPoint

	if v, ok := s.Architecture["extension_points"]; ok {
		if entries := mapsFromAny(v); len(entries) > 0 {
			for _, e := range entries {
				name := strings.TrimSpace(fmt.Sprintf("%v", e["name"]))
				if name == "" || name == "<nil>" {
					continue
				}
				steps := stringsFromAny(e["steps"])
				points = append(points, extPoint{Name: name, Steps: steps})
			}
		}
	}

	if len(points) == 0 {
		points = []extPoint{
			{
				Name: "Add a new LLM provider",
				Steps: []string{
					"Implement the `LLMProvider` interface",
					"Register in `platform/llm/resolver.go`",
					"Add env var to `config.go`",
				},
			},
			{
				Name: "Add a new output document format",
				Steps: []string{
					"Implement `Generate<DocType>(s CanonicalState) (string, error)`",
					"Register key in `markdown/generator.go` `GenerateAll`",
					"Add key to session `output_docs` allowed values",
				},
			},
		}
	}

	var b strings.Builder
	for i, p := range points {
		b.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, p.Name))
		for j, step := range p.Steps {
			b.WriteString(fmt.Sprintf("%d. %s\n", j+1, step))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// renderSecurityConsiderations renders §10 as a 3-column table.
// Reads architecture["security"] array. Falls back to 2 stub rows.
func renderSecurityConsiderations(s state.CanonicalState) string {
	type secRow struct {
		Surface    string
		Risk       string
		Mitigation string
	}
	var rows []secRow

	if v, ok := s.Architecture["security"]; ok {
		if entries := mapsFromAny(v); len(entries) > 0 {
			for _, e := range entries {
				surface := strings.TrimSpace(fmt.Sprintf("%v", e["surface"]))
				risk := strings.TrimSpace(fmt.Sprintf("%v", e["risk"]))
				mitigation := strings.TrimSpace(fmt.Sprintf("%v", e["mitigation"]))
				if surface == "<nil>" {
					surface = ""
				}
				if risk == "<nil>" {
					risk = ""
				}
				if mitigation == "<nil>" {
					mitigation = ""
				}
				if surface != "" {
					rows = append(rows, secRow{Surface: surface, Risk: risk, Mitigation: mitigation})
				}
			}
		}
	}

	if len(rows) == 0 {
		rows = []secRow{
			{
				Surface:    "Authentication",
				Risk:       "Unauthorized access to session data",
				Mitigation: "Validate all session IDs against DB on every request",
			},
			{
				Surface:    "Prompt Injection",
				Risk:       "Malicious user input manipulates LLM output",
				Mitigation: "Validate and sanitise all user-controlled input before LLM prompt assembly",
			},
		}
	}

	tableRows := make([][]string, len(rows))
	for i, r := range rows {
		tableRows[i] = []string{r.Surface, r.Risk, r.Mitigation}
	}
	return renderTable([]string{"Surface", "Risk", "Mitigation"}, tableRows)
}

// renderSystemGuarantees renders §12 as a numbered list.
// Reads architecture["guarantees"] ([]string). Falls back to 3 hardcoded items.
func renderSystemGuarantees(s state.CanonicalState) string {
	var items []string

	if v, ok := s.Architecture["guarantees"]; ok {
		items = stringsFromAny(v)
	}

	if len(items) == 0 {
		items = []string{
			"Deterministic output: same canonical state input produces identical generated documents",
			"Isolated modules: no cross-module direct imports between `internal/modules/` packages",
			"LLM abstracted: all model calls routed through the `LLMProvider` interface — no direct SDK usage in business logic",
		}
	}

	var b strings.Builder
	for i, item := range items {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
	}
	b.WriteString("\n")
	return b.String()
}

// renderEnrichedRisksTable renders §13 as a 6-column table:
// # | Risk | Likelihood | Impact | Mitigation | Status.
// Uses capitaliseSeverity (not strings.ToTitle) to avoid the unicode bug.
func renderEnrichedRisksTable(s state.CanonicalState) string {
	if len(s.Risks) == 0 {
		return "_No risks identified yet._\n\n"
	}
	rows := make([][]string, 0, len(s.Risks))
	for i, r := range s.Risks {
		status := fmt.Sprintf("⚠️ Open (%s)", capitaliseSeverity(r.Severity))
		if r.Resolved {
			status = fmt.Sprintf("✅ Resolved (%s)", capitaliseSeverity(r.Severity))
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			r.Text,
			"—",
			"—",
			"—",
			status,
		})
	}
	return renderTable([]string{"#", "Risk", "Likelihood", "Impact", "Mitigation", "Status"}, rows)
}

// renderDefinitionOfDone renders §16 as a GitHub-flavour checklist.
// Reads architecture["exit_criteria"] ([]string).
// Always appends "All open questions resolved" and "Confidence ≥ 0.85".
// Fallback when exit_criteria absent: 5 generic items.
func renderDefinitionOfDone(s state.CanonicalState) string {
	items := stringsFromAny(s.Architecture["exit_criteria"])

	if len(items) == 0 {
		items = []string{
			"Core feature implemented and manually tested end-to-end",
			"All unit and integration tests passing (0 failures)",
			"`go build ./...` and `go vet ./...` clean",
			"No hardcoded credentials, ports, or model names outside config files",
			"Architecture document generated, reviewed, and committed",
		}
	}

	// Always append these two items unconditionally.
	items = append(items, "All open questions resolved")
	items = append(items, "Confidence ≥ 0.85")

	var b strings.Builder
	for _, item := range items {
		b.WriteString(fmt.Sprintf("- [ ] %s\n", item))
	}
	b.WriteString("\n")
	return b.String()
}

// renderForAIAgentsAppendix appends the dual-audience machine-oriented section (§8.29.6).
// When docKey == "architecture" it uses architecture-specific fields (layer names,
// LLM interface name, forbidden patterns) from the canonical state.
func renderForAIAgentsAppendix(s state.CanonicalState, docKey string) string {
	var b strings.Builder
	b.WriteString("## For AI Agents\n\n")
	b.WriteString("### Stack\n\n")
	if docKey == "architecture" {
		b.WriteString(renderEnrichedTechStack(s))
	} else {
		b.WriteString(renderTechStack(s))
	}
	b.WriteString("### Key Contracts\n\n")

	// LLM interface name from state or canonical default.
	llmIface := "LLMProvider"
	if v, ok := s.Architecture["llm_interface_name"]; ok {
		if str := strings.TrimSpace(fmt.Sprintf("%v", v)); str != "" && str != "<nil>" {
			llmIface = str
		}
	}

	// Layer names from actual layers.
	layerNames := extractLayerNames(s)

	b.WriteString("- Module paths follow vertical-slice layout under `backend/internal/modules/`\n")
	b.WriteString("- Cross-module communication uses exported service interfaces only\n")
	b.WriteString(fmt.Sprintf("- LLM calls go through `%s`; A2A via platform wrapper\n", llmIface))
	if len(layerNames) > 0 {
		b.WriteString(fmt.Sprintf("- Layer boundaries: %s\n", strings.Join(layerNames, " → ")))
	}
	b.WriteString(fmt.Sprintf("- Primary artifact: `%s` — preserve heading order when editing\n\n", docKey))

	b.WriteString("### Implementation Order\n\n")
	if len(s.ExecutionPlan) > 0 {
		for i, step := range s.ExecutionPlan {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, step.Title))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("1. Scaffold project structure\n2. Implement core modules\n3. Wire HTTP handlers and frontend\n\n")
	}

	b.WriteString("### Out of Scope\n\n")

	// Forbidden patterns from state or canonical defaults.
	forbidden := stringsFromAny(s.Architecture["forbidden_patterns"])
	if len(forbidden) == 0 {
		forbidden = []string{
			"Microservices between modules",
			"Direct LLM SDK usage in business modules",
			"Non-deterministic IDs or runtime role alternation",
		}
	}
	for _, f := range forbidden {
		b.WriteString(fmt.Sprintf("- %s\n", f))
	}
	b.WriteString("\n")
	return b.String()
}

// extractLayerNames returns the ordered list of layer names from
// s.Architecture["layers"], or nil when no structured layers are present.
func extractLayerNames(s state.CanonicalState) []string {
	v, ok := s.Architecture["layers"]
	if !ok {
		return nil
	}
	layers := mapsFromAny(v)
	if len(layers) == 0 {
		return nil
	}
	names := make([]string, 0, len(layers))
	for _, layer := range layers {
		name := strings.TrimSpace(fmt.Sprintf("%v", layer["name"]))
		if name != "" && name != "<nil>" {
			names = append(names, name)
		}
	}
	return names
}

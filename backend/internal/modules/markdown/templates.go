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
func oneLineDescription(s state.CanonicalState) string {
	for _, key := range []string{"summary", "description", "text"} {
		if v, ok := s.Idea[key]; ok {
			str := strings.TrimSpace(fmt.Sprintf("%v", v))
			if str != "" {
				return strings.Join(strings.Fields(str), " ")
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
		return fmt.Sprintf("```\n%v\n```\n\n", v)
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
		b.WriteString(fmt.Sprintf("%v\n\n", ts))
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
		return fmt.Sprintf("%v\n\n", v)
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
			strings.ToTitle(r.Severity),
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
		return fmt.Sprintf("```env\n%v\n```\n\n", v)
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

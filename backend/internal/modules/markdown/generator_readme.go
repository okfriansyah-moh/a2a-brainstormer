// Package markdown — README document generator (§8.32).
package markdown

import (
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GenerateReadme renders the README document from s.
// It produces 13 sections: title, tagline, description, golden rule, what-not,
// when-to-use, installation, quick start, architecture, command reference,
// tech stack, repository format, contributing.
//
// Deterministic: same state → same output, no randomness, no time.Now().
func GenerateReadme(s state.CanonicalState) (string, error) {
	var b strings.Builder
	title := shortTitle(s)

	// §1 — Title (NO "— README" suffix per §8.32)
	b.WriteString(fmt.Sprintf("# %s\n\n", title))

	// §2 — Tagline blockquote
	tagline := archString(s, "tagline")
	if tagline == "" {
		tagline = oneLineDescription(s)
	}
	b.WriteString(fmt.Sprintf("> %s\n\n", tagline))

	// §3 — Description paragraph
	desc := archString(s, "description_paragraph")
	if desc == "" {
		if v, ok := s.Idea["context"]; ok {
			desc = fmt.Sprintf("%v", v)
			if len(desc) > 300 {
				desc = desc[:300]
			}
			desc = strings.TrimSpace(desc)
		}
	}
	if desc != "" {
		b.WriteString(desc + "\n\n")
	}

	// §4 — Golden Rule
	b.WriteString("## Golden Rule\n\n")
	if gr := archString(s, "golden_rule"); gr != "" {
		b.WriteString(fmt.Sprintf("**Golden rule:** %s\n\n", gr))
	} else {
		b.WriteString("**Golden rule:** _No golden rule defined._\n\n")
	}

	// §5 — What it is NOT
	b.WriteString("## What it is NOT\n\n")
	b.WriteString(renderIsNot(s))

	// §6 — When to use
	b.WriteString("## When to use\n\n")
	b.WriteString(renderWhenToUse(s))

	// §7 — Installation / Prerequisites
	b.WriteString("## Installation\n\n")
	b.WriteString(renderInstallation(s))

	// §8 — Quick Start
	b.WriteString("## Quick Start\n\n")
	b.WriteString(renderQuickStart(s))

	// §9 — Architecture
	b.WriteString("## Architecture\n\n")
	b.WriteString(renderReadmeArchitecture(s))

	// §10 — Command Reference
	b.WriteString("## Command Reference\n\n")
	b.WriteString(renderCommandReference(s))

	// §11 — Tech Stack
	b.WriteString("## Tech Stack\n\n")
	b.WriteString(renderTechStack(s))

	// §12 — Repository Format
	b.WriteString("## Repository Format\n\n")
	b.WriteString(renderRepositoryFormat(s))

	// §13 — Contributing
	b.WriteString("## Contributing\n\n")
	b.WriteString(renderContributing(s))

	// HTML comment footer (not visible in rendered output)
	b.WriteString(fmt.Sprintf("<!-- Iteration %d · Confidence %.4f -->\n",
		s.Meta.Iteration, s.Metrics.Confidence))

	return b.String(), nil
}

// archString returns s.Architecture[key] as a trimmed string, or "" when
// absent, nil, or not a string value. This is the canonical accessor for all
// ReadmeEnrichmentOverlay fields stored in the architecture map.
func archString(s state.CanonicalState, key string) string {
	if s.Architecture == nil {
		return ""
	}
	v, ok := s.Architecture[key]
	if !ok {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(str)
}

// renderIsNot renders the "What it is NOT" section from s.Architecture["is_not"]
// as a bullet list. Falls back to three generic bullets when absent.
func renderIsNot(s state.CanonicalState) string {
	var b strings.Builder
	if v, ok := s.Architecture["is_not"]; ok {
		items := stringsFromAny(v)
		if len(items) > 0 {
			for _, item := range items {
				b.WriteString(fmt.Sprintf("- %s\n", item))
			}
			b.WriteString("\n")
			return b.String()
		}
	}
	// Fallback: 3 generic scope-boundary bullets
	b.WriteString("- Not a general-purpose LLM wrapper\n")
	b.WriteString("- Not designed for single-agent, single-turn conversations\n")
	b.WriteString("- Not a hosted SaaS — requires local deployment\n")
	b.WriteString("\n")
	return b.String()
}

// renderWhenToUse renders §6 — mermaid flowchart (if present) followed by
// numbered scenarios from s.Architecture["when_to_use"].
func renderWhenToUse(s state.CanonicalState) string {
	var b strings.Builder

	// Mermaid decision flowchart
	if mmd := archString(s, "when_to_use_mermaid"); mmd != "" {
		b.WriteString("```mermaid\n")
		b.WriteString(mmd)
		if !strings.HasSuffix(mmd, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	}

	// Scenarios
	if v, ok := s.Architecture["when_to_use"]; ok {
		scenarios := mapsFromAny(v)
		if len(scenarios) > 0 {
			for i, sc := range scenarios {
				title := strings.TrimSpace(fmt.Sprintf("%v", sc["title"]))
				desc := strings.TrimSpace(fmt.Sprintf("%v", sc["description"]))
				code := strings.TrimSpace(fmt.Sprintf("%v", sc["code"]))
				if title == "<nil>" {
					title = ""
				}
				if desc == "<nil>" {
					desc = ""
				}
				if code == "<nil>" {
					code = ""
				}
				b.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, title))
				if desc != "" {
					b.WriteString(desc + "\n\n")
				}
				if code != "" {
					b.WriteString("```\n")
					b.WriteString(code + "\n")
					b.WriteString("```\n\n")
				}
			}
			return b.String()
		}
	}

	// Fallback: 3 generic scenarios without code blocks (no real commands known)
	if b.Len() == 0 {
		b.WriteString("### 1. Exploring a complex software architecture\n\n")
		b.WriteString("Use when you need multiple perspectives on a system design.\n\n")
		b.WriteString("### 2. Generating implementation-ready plans\n\n")
		b.WriteString("Use when you need an AI-driven breakdown of tasks with clear validation criteria.\n\n")
		b.WriteString("### 3. Iterative refinement with confidence scoring\n\n")
		b.WriteString("Use when a design needs multiple passes before reaching production quality.\n\n")
	} else {
		// mermaid was written but no scenarios: append fallback scenarios
		b.WriteString("### 1. Exploring a complex software architecture\n\n")
		b.WriteString("Use when you need multiple perspectives on a system design.\n\n")
		b.WriteString("### 2. Generating implementation-ready plans\n\n")
		b.WriteString("Use when you need an AI-driven breakdown of tasks with clear validation criteria.\n\n")
		b.WriteString("### 3. Iterative refinement with confidence scoring\n\n")
		b.WriteString("Use when a design needs multiple passes before reaching production quality.\n\n")
	}

	return b.String()
}

// renderInstallation renders §7 — prerequisites table + setup steps.
func renderInstallation(s state.CanonicalState) string {
	var b strings.Builder

	// Prerequisites table from s.Architecture["prerequisites"]
	if v, ok := s.Architecture["prerequisites"]; ok {
		prereqs := mapsFromAny(v)
		if len(prereqs) > 0 {
			rows := make([][]string, 0, len(prereqs))
			for _, p := range prereqs {
				tool := strings.TrimSpace(fmt.Sprintf("%v", p["tool"]))
				version := strings.TrimSpace(fmt.Sprintf("%v", p["version"]))
				required := strings.TrimSpace(fmt.Sprintf("%v", p["required"]))
				if tool == "<nil>" {
					tool = ""
				}
				if version == "<nil>" || version == "false" {
					version = "—"
				}
				if required == "true" {
					required = "Yes"
				} else if required == "false" {
					required = "No"
				} else if required == "<nil>" {
					required = "Yes"
				}
				if tool != "" {
					rows = append(rows, []string{tool, version, required})
				}
			}
			if len(rows) > 0 {
				b.WriteString(renderTable([]string{"Tool", "Version", "Required"}, rows))
			}
		}
	}

	// Setup steps
	b.WriteString("**Setup:**\n\n")
	if v, ok := s.Architecture["quick_start_commands"]; ok {
		cmds := stringsFromAny(v)
		if len(cmds) > 0 && cmds[0] != "" {
			for i, cmd := range cmds {
				b.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, cmd))
			}
			b.WriteString("\n")
			return b.String()
		}
	}
	// Fallback setup steps
	b.WriteString("1. `go install ./cmd/server`\n")
	b.WriteString("2. `docker compose up --build`\n")
	b.WriteString("\n")
	return b.String()
}

// renderQuickStart renders §8 — a fenced bash block with real commands.
// Source: s.Architecture["quick_start_commands"] ([]any of strings).
// Never emits <repository-url> or <project> placeholders.
func renderQuickStart(s state.CanonicalState) string {
	var b strings.Builder
	if v, ok := s.Architecture["quick_start_commands"]; ok {
		cmds := stringsFromAny(v)
		if len(cmds) > 0 {
			b.WriteString("```bash\n")
			for _, cmd := range cmds {
				b.WriteString(cmd + "\n")
			}
			b.WriteString("```\n\n")
			return b.String()
		}
	}
	// Fallback: real commands, never placeholders
	b.WriteString("```bash\n")
	b.WriteString("docker compose up --build\n")
	b.WriteString("go run ./backend/cmd/server/main.go\n")
	b.WriteString("```\n\n")
	return b.String()
}

// renderReadmeArchitecture renders §9 — ASCII diagram + Mermaid diagram.
// Prefers arch-enriched fields over generic fallbacks.
func renderReadmeArchitecture(s state.CanonicalState) string {
	var b strings.Builder

	// ASCII diagram
	asciiDiagram := archString(s, "architecture_ascii")
	if asciiDiagram != "" {
		b.WriteString("```\n")
		b.WriteString(asciiDiagram)
		if !strings.HasSuffix(asciiDiagram, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	} else {
		b.WriteString(renderASCIIComponents(s))
	}

	// Mermaid diagram
	mmdDiagram := archString(s, "architecture_mermaid")
	if mmdDiagram != "" {
		b.WriteString("```mermaid\n")
		b.WriteString(mmdDiagram)
		if !strings.HasSuffix(mmdDiagram, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	} else if mermaid := renderDataFlowsMermaid(s); mermaid != "" {
		b.WriteString(mermaid)
	}

	return b.String()
}

// renderCommandReference renders §10 — a Markdown table of commands.
// Source: s.Architecture["command_reference"] ([]any of {command, description}).
func renderCommandReference(s state.CanonicalState) string {
	if v, ok := s.Architecture["command_reference"]; ok {
		entries := mapsFromAny(v)
		if len(entries) > 0 {
			rows := make([][]string, 0, len(entries))
			for _, e := range entries {
				cmd := strings.TrimSpace(fmt.Sprintf("%v", e["command"]))
				desc := strings.TrimSpace(fmt.Sprintf("%v", e["description"]))
				if cmd == "<nil>" {
					cmd = ""
				}
				if desc == "<nil>" {
					desc = ""
				}
				if cmd != "" {
					rows = append(rows, []string{fmt.Sprintf("`%s`", cmd), desc})
				}
			}
			if len(rows) > 0 {
				return renderTable([]string{"Command", "Description"}, rows)
			}
		}
	}
	// Fallback: one generic row
	return renderTable(
		[]string{"Command", "Description"},
		[][]string{{"`docker compose up`", "Start all services"}},
	)
}

// renderRepositoryFormat renders §12 — directory tree + optional repo format notes.
func renderRepositoryFormat(s state.CanonicalState) string {
	var b strings.Builder
	b.WriteString(renderDirectoryTree(s))

	// Per-entry notes from s.Architecture["repo_format_notes"]
	if v, ok := s.Architecture["repo_format_notes"]; ok {
		if m, ok := v.(map[string]any); ok && len(m) > 0 {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			// Sort for deterministic output
			sortedKeys := keys
			for i := 1; i < len(sortedKeys); i++ {
				for j := i; j > 0 && sortedKeys[j] < sortedKeys[j-1]; j-- {
					sortedKeys[j], sortedKeys[j-1] = sortedKeys[j-1], sortedKeys[j]
				}
			}
			rows := make([][]string, 0, len(sortedKeys))
			for _, k := range sortedKeys {
				rows = append(rows, []string{k, fmt.Sprintf("%v", m[k])})
			}
			b.WriteString(renderTable([]string{"Path", "Description"}, rows))
		}
	}
	return b.String()
}

// renderContributing renders §13 — contributing note.
// Source: s.Architecture["contributing_note"] → fallback to generic message.
func renderContributing(s state.CanonicalState) string {
	if note := archString(s, "contributing_note"); note != "" {
		return note + "\n\n"
	}
	return "Read `AGENTS.md` and `.github/skills/` before changing this codebase.\n\n"
}

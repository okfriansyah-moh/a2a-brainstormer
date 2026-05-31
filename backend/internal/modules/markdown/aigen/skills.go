// Package aigen layers an AI-driven document-generation pass on top of the
// deterministic markdown generators. It is wired in when FINALIZE_MODE is
// "hybrid" or "ai"; in "deterministic" mode the package is bypassed entirely.
//
// See docs/PLAN.md §8.27 for the full contract: skill bundle composition,
// rubric defaults, auto-repair algorithm, fallback semantics.
package aigen

import (
	"fmt"
	"io/fs"
	"strings"
)

// Skill is one prompt fragment loaded from a `.github/skills/<name>/SKILL.md`
// file. Prompt is the verbatim body with the YAML frontmatter stripped.
type Skill struct {
	Name   string
	Path   string
	Prompt string
}

// SkillBundle is the ordered collection of skills injected as a system-prompt
// prefix into every AI generation call. Order is stable and significant —
// earlier skills are composed first.
type SkillBundle struct {
	Skills []Skill
}

// Compose returns the concatenated bundle prompt, with each skill rendered
// under a `## Skill: <name>` heading. Returns an empty string when the bundle
// holds no skills.
func (b SkillBundle) Compose() string {
	if len(b.Skills) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, s := range b.Skills {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString("## Skill: ")
		sb.WriteString(s.Name)
		sb.WriteString("\n\n")
		sb.WriteString(s.Prompt)
	}
	return sb.String()
}

// LoadBundle loads the listed skill files from fsys (rooted at the repo) and
// returns a SkillBundle in the same order. Each file's YAML frontmatter
// (delimited by `---` lines) is stripped before storing the prompt body.
//
// Returns an error if any path is empty, missing, or unreadable — there is no
// silent fallback. The caller is expected to fall back to deterministic mode
// when the bundle fails to load.
func LoadBundle(fsys fs.FS, paths []string) (SkillBundle, error) {
	if len(paths) == 0 {
		return SkillBundle{}, fmt.Errorf("aigen: skill bundle paths is empty")
	}
	skills := make([]Skill, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			return SkillBundle{}, fmt.Errorf("aigen: empty skill path in bundle")
		}
		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			return SkillBundle{}, fmt.Errorf("aigen: read skill %q: %w", p, err)
		}
		skills = append(skills, Skill{
			Name:   skillNameFromPath(p),
			Path:   p,
			Prompt: stripFrontmatter(string(raw)),
		})
	}
	return SkillBundle{Skills: skills}, nil
}

// skillNameFromPath returns the directory name immediately above SKILL.md,
// which by repository convention is the canonical skill identifier.
// Falls back to the full path when the convention does not apply.
func skillNameFromPath(p string) string {
	parts := strings.Split(p, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.EqualFold(parts[i], "SKILL.md") && i > 0 {
			return parts[i-1]
		}
	}
	return p
}

// stripFrontmatter removes a leading YAML frontmatter block delimited by lines
// containing only `---`. When no frontmatter is present, the input is returned
// unchanged with leading whitespace trimmed.
func stripFrontmatter(body string) string {
	trimmed := strings.TrimLeft(body, " \t\r\n")
	if !strings.HasPrefix(trimmed, "---") {
		return trimmed
	}
	// Find the closing `---` on its own line.
	rest := trimmed[3:]
	// Skip the first newline after the opening marker.
	if idx := strings.Index(rest, "\n"); idx >= 0 {
		rest = rest[idx+1:]
	}
	if end := strings.Index(rest, "\n---"); end >= 0 {
		after := rest[end+4:]
		return strings.TrimLeft(after, " \t\r\n")
	}
	return trimmed
}

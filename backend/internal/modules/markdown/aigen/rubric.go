// Package aigen — rubric validator for AI-generated documents.
//
// The rubric is a per-document set of section rules. Each section rule
// specifies a heading that MUST appear in the rendered document, a minimum
// body length in characters, and an optional list of required keywords or
// forbidden placeholder phrases. See docs/PLAN.md §8.27 for defaults.
package aigen

import (
	"fmt"
	"strings"
)

// SectionRule is one rubric line — a single heading the document must contain
// with body-level constraints.
type SectionRule struct {
	Heading            string
	MinChars           int
	RequiredKeywords   []string
	ForbidPlaceholders []string
}

// Rubric is the per-document quality contract evaluated after each AI draft.
type Rubric struct {
	DocKey   string
	Sections []SectionRule
	// MinTotalLines, when > 0, requires the full rendered document to contain
	// at least this many newline-separated lines. Used to enforce thorough,
	// structured output (sub-sections, tables, diagrams) without relying on
	// any single section to carry the weight.
	MinTotalLines int
	// MinTotalChars, when > 0, requires the trimmed full document body to
	// contain at least this many characters. Pairs with MinTotalLines to guard
	// against "many short lines" gaming of the line-count rule.
	MinTotalChars int
}

// RubricFinding is one rule violation in the validated document.
type RubricFinding struct {
	Heading string
	Reason  string
}

// String renders a finding as a single bullet line for the auto-repair prompt.
func (f RubricFinding) String() string {
	return fmt.Sprintf("- §%q: %s", f.Heading, f.Reason)
}

// defaultPlaceholders are forbidden body strings shared by every section rule
// unless the rule overrides ForbidPlaceholders explicitly.
var defaultPlaceholders = []string{"TBD", "TODO", "Lorem ipsum", "placeholder"}

// defaultRubrics enumerates the canonical rubric per output-doc key. See
// docs/PLAN.md §8.27 for the source of these defaults.
//
// MinTotalLines is set to 1000 on every doc — the AI generator must produce
// production-grade, deeply-structured documents. The per-section character
// minimums are sized so the deterministic scaffold can never satisfy the
// rubric on its own: an AI rewrite is mandatory.
var defaultRubrics = map[string]Rubric{
	"architecture": {
		DocKey:        "architecture",
		MinTotalLines: 1000,
		MinTotalChars: 35000,
		Sections: []SectionRule{
			{Heading: "1. Overview", MinChars: 1500},
			{Heading: "2. System Components", MinChars: 4000, RequiredKeywords: []string{"Responsibility", "Technologies", "Dependencies"}},
			{Heading: "3. Data Model", MinChars: 2500},
			{Heading: "4. Data Flow", MinChars: 2000, RequiredKeywords: []string{"```mermaid"}},
			{Heading: "5. Deployment", MinChars: 1500},
		},
	},
	"roadmap": {
		DocKey:        "roadmap",
		MinTotalLines: 1000,
		MinTotalChars: 35000,
		Sections: []SectionRule{
			{Heading: "1. Goals", MinChars: 1200},
			{Heading: "2. Milestones", MinChars: 2500},
			{Heading: "3. Phase Breakdown", MinChars: 5000, RequiredKeywords: []string{"Objective", "Scope", "Deliverables", "Exit Criteria"}},
			{Heading: "4. Risks", MinChars: 1500},
		},
	},
	"plan": {
		DocKey:        "plan",
		MinTotalLines: 1000,
		MinTotalChars: 35000,
		Sections: []SectionRule{
			{Heading: "1. Scope", MinChars: 1200},
			{Heading: "2. Architecture", MinChars: 2000},
			{Heading: "3. Modules", MinChars: 3500},
			{Heading: "4. Tasks", MinChars: 5000, RequiredKeywords: []string{"Files to create", "Validation"}},
		},
	},
	"readme": {
		DocKey:        "readme",
		MinTotalLines: 1000,
		MinTotalChars: 35000,
		Sections: []SectionRule{
			{Heading: "Overview", MinChars: 1500},
			{Heading: "Architecture", MinChars: 1500},
			{Heading: "Roadmap", MinChars: 1500},
			{Heading: "Getting Started", MinChars: 1000},
		},
	},
}

// RubricFor returns the canonical rubric for docKey, or an empty Rubric (no
// sections) when the key is unknown. An empty rubric trivially passes — that
// matches the Task-32 contract of allowing custom doc keys via the Generators
// registry without breaking the AI path.
func RubricFor(docKey string) Rubric {
	if r, ok := defaultRubrics[docKey]; ok {
		return r
	}
	return Rubric{DocKey: docKey}
}

// Validate evaluates content against r and returns one finding per rule
// violation. An empty slice means the document passes the rubric.
//
// The check looks for each heading anywhere in the document (case-sensitive,
// substring match against any line beginning with "#") and treats the body as
// the text from the heading to the next heading at the same or higher level
// (or end-of-document). Document-level rules (MinTotalLines, MinTotalChars)
// are evaluated against the trimmed full content.
func Validate(content string, r Rubric) []RubricFinding {
	var findings []RubricFinding
	trimmed := strings.TrimSpace(content)
	if r.MinTotalLines > 0 {
		lineCount := strings.Count(trimmed, "\n") + 1
		if lineCount < r.MinTotalLines {
			findings = append(findings, RubricFinding{
				Heading: "<document>",
				Reason:  fmt.Sprintf("document has %d lines; minimum is %d — expand every section with sub-headings, tables, diagrams, code samples, and worked examples", lineCount, r.MinTotalLines),
			})
		}
	}
	if r.MinTotalChars > 0 {
		if len(trimmed) < r.MinTotalChars {
			findings = append(findings, RubricFinding{
				Heading: "<document>",
				Reason:  fmt.Sprintf("document has %d chars; minimum is %d — add concrete detail, not filler", len(trimmed), r.MinTotalChars),
			})
		}
	}
	for _, rule := range r.Sections {
		body, ok := extractSectionBody(content, rule.Heading)
		if !ok {
			findings = append(findings, RubricFinding{
				Heading: rule.Heading,
				Reason:  "required section heading not found",
			})
			continue
		}
		bodyLen := len(strings.TrimSpace(body))
		if rule.MinChars > 0 && bodyLen < rule.MinChars {
			findings = append(findings, RubricFinding{
				Heading: rule.Heading,
				Reason:  fmt.Sprintf("section body has %d chars; minimum is %d", bodyLen, rule.MinChars),
			})
		}
		for _, kw := range rule.RequiredKeywords {
			if !strings.Contains(body, kw) {
				findings = append(findings, RubricFinding{
					Heading: rule.Heading,
					Reason:  fmt.Sprintf("required keyword %q absent from section body", kw),
				})
			}
		}
		placeholders := rule.ForbidPlaceholders
		if placeholders == nil {
			placeholders = defaultPlaceholders
		}
		for _, ph := range placeholders {
			if strings.Contains(body, ph) {
				findings = append(findings, RubricFinding{
					Heading: rule.Heading,
					Reason:  fmt.Sprintf("forbidden placeholder %q present in section body", ph),
				})
			}
		}
	}
	return findings
}

// extractSectionBody returns the substring of content that follows the first
// line matching the heading and ends before the next heading line. The second
// return value is false when no matching heading is found.
//
// Heading match: the line, trimmed of leading "#" and whitespace, must contain
// rule.Heading as a substring. This tolerates `## 2. System Components`,
// `### 2. System Components — Details`, and other formatting variations.
func extractSectionBody(content, heading string) (string, bool) {
	lines := strings.Split(content, "\n")
	startIdx := -1
	for i, ln := range lines {
		if !strings.HasPrefix(strings.TrimSpace(ln), "#") {
			continue
		}
		stripped := strings.TrimLeft(strings.TrimSpace(ln), "#")
		stripped = strings.TrimSpace(stripped)
		if strings.Contains(stripped, heading) {
			startIdx = i + 1
			break
		}
	}
	if startIdx < 0 {
		return "", false
	}
	endIdx := len(lines)
	for j := startIdx; j < len(lines); j++ {
		if strings.HasPrefix(strings.TrimSpace(lines[j]), "#") {
			endIdx = j
			break
		}
	}
	return strings.Join(lines[startIdx:endIdx], "\n"), true
}

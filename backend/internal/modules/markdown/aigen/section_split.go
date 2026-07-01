// Package aigen — deterministic scaffold splitting and merging for section-sequential generation.
package aigen

import (
	"fmt"
	"strings"
)

// EnhancedSection is one rubric bucket after enhancement.
type EnhancedSection struct {
	Heading     string // rubric heading key e.g. "4. Layers"
	HeadingLine string // primary ## line for this bucket
	Body        string // merged body for all ## blocks in the bucket (excludes heading lines)
}

// DocHeading is one markdown heading line in document order.
type DocHeading struct {
	LineIndex int
	Line      string
	Title     string // stripped of leading # and whitespace
}

// ListHeadings returns all lines that start with # in document order.
func ListHeadings(content string) []DocHeading {
	lines := strings.Split(content, "\n")
	out := make([]DocHeading, 0)
	for i, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		stripped := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		out = append(out, DocHeading{
			LineIndex: i,
			Line:      ln,
			Title:     stripped,
		})
	}
	return out
}

// ExtractPreamble returns content before the first ## heading that matches
// firstSectionHeading (substring match on stripped title).
func ExtractPreamble(content string, firstSectionHeading string) (string, bool) {
	headings := ListHeadings(content)
	if len(headings) == 0 {
		return "", false
	}
	firstIdx := -1
	for i, h := range headings {
		if headingMatches(h.Title, firstSectionHeading) {
			firstIdx = i
			break
		}
	}
	if firstIdx < 0 {
		return "", false
	}
	lines := strings.Split(content, "\n")
	end := headings[firstIdx].LineIndex
	if end <= 0 {
		return "", true
	}
	return strings.Join(lines[:end], "\n"), true
}

// BucketScaffoldSections splits scaffold into rubric-ordered buckets. Non-rubric
// ## blocks between rubric headings attach to the next rubric bucket.
func BucketScaffoldSections(scaffold string, rubric Rubric) ([]EnhancedSection, error) {
	if len(rubric.Sections) == 0 {
		return nil, fmt.Errorf("bucket sections: empty rubric for %q", rubric.DocKey)
	}
	headings := ListHeadings(scaffold)
	if len(headings) == 0 {
		return nil, fmt.Errorf("bucket sections: no headings in scaffold")
	}

	lines := strings.Split(scaffold, "\n")

	// Map each rubric section to its primary heading index.
	rubricIdx := make([]int, len(rubric.Sections))
	for i, rule := range rubric.Sections {
		idx := -1
		for j, h := range headings {
			if headingMatches(h.Title, rule.Heading) {
				idx = j
				break
			}
		}
		if idx < 0 {
			return nil, fmt.Errorf("bucket sections: rubric heading %q not found", rule.Heading)
		}
		rubricIdx[i] = idx
	}

	out := make([]EnhancedSection, 0, len(rubric.Sections))
	for i, rule := range rubric.Sections {
		endHeading := rubricIdx[i]
		var startHeading int
		if i == 0 {
			startHeading = endHeading
		} else {
			startHeading = rubricIdx[i-1] + 1
			if startHeading > endHeading {
				startHeading = endHeading
			}
		}

		var bodyParts []string
		primaryLine := headings[endHeading].Line
		for j := startHeading; j <= endHeading; j++ {
			h := headings[j]
			sectStart := h.LineIndex + 1
			sectEnd := len(lines)
			if j+1 < len(headings) {
				sectEnd = headings[j+1].LineIndex
			}
			if sectStart < sectEnd {
				bodyParts = append(bodyParts, strings.Join(lines[sectStart:sectEnd], "\n"))
			}
		}
		out = append(out, EnhancedSection{
			Heading:     rule.Heading,
			HeadingLine: primaryLine,
			Body:        strings.TrimSpace(strings.Join(bodyParts, "\n\n")),
		})
	}
	return out, nil
}

// MergeSections rebuilds a document from preamble and enhanced sections.
// Each section is emitted as its primary HeadingLine followed by the body.
func MergeSections(preamble string, sections []EnhancedSection) string {
	var b strings.Builder
	if preamble != "" {
		b.WriteString(strings.TrimRight(preamble, "\n"))
		b.WriteString("\n\n")
	}
	for i, sec := range sections {
		line := sec.HeadingLine
		if line == "" {
			line = "## " + sec.Heading
		}
		b.WriteString(line)
		b.WriteString("\n\n")
		body := strings.Trim(sec.Body, "\n")
		if body != "" {
			b.WriteString(body)
			b.WriteString("\n\n")
		}
		if i < len(sections)-1 && body == "" {
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// SummarizePriorSections produces a deterministic truncated summary of prior
// enhanced sections for LLM context (first paragraph + table header lines).
func SummarizePriorSections(sections []EnhancedSection, maxChars int) string {
	if maxChars <= 0 || len(sections) == 0 {
		return ""
	}
	var b strings.Builder
	for _, sec := range sections {
		if b.Len() >= maxChars {
			break
		}
		chunk := summarizeOneSection(sec)
		if chunk == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n---\n")
		}
		b.WriteString(chunk)
	}
	s := b.String()
	if len(s) > maxChars {
		return s[:maxChars]
	}
	return s
}

func summarizeOneSection(sec EnhancedSection) string {
	var b strings.Builder
	b.WriteString("## ")
	b.WriteString(sec.Heading)
	b.WriteString("\n")
	lines := strings.Split(sec.Body, "\n")
	added := 0
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, "#") || added == 0 {
			b.WriteString(ln)
			b.WriteString("\n")
			added++
		}
		if added >= 4 {
			break
		}
	}
	return strings.TrimSpace(b.String())
}

func headingMatches(title, rubricHeading string) bool {
	return strings.HasPrefix(title, rubricHeading)
}

// ReplaceSectionBody swaps one section body in merged content.
func ReplaceSectionBody(content, heading, newBody string) (string, error) {
	headings := ListHeadings(content)
	target := -1
	for i, h := range headings {
		if headingMatches(h.Title, heading) {
			target = i
			break
		}
	}
	if target < 0 {
		return "", fmt.Errorf("replace section: heading %q not found", heading)
	}
	lines := strings.Split(content, "\n")
	start := headings[target].LineIndex + 1
	end := len(lines)
	if target+1 < len(headings) {
		end = headings[target+1].LineIndex
	}
	var b strings.Builder
	b.WriteString(strings.Join(lines[:start], "\n"))
	if start > 0 {
		b.WriteString("\n")
	}
	b.WriteString(strings.Trim(newBody, "\n"))
	if end < len(lines) {
		b.WriteString("\n")
		b.WriteString(strings.Join(lines[end:], "\n"))
	}
	result := b.String()
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result, nil
}

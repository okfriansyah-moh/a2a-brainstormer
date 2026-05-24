// Package shared contains types used by multiple backend modules.
// These types must not reference any module-specific packages.
package shared

// GeneratedDocument is one rendered output artifact produced by the markdown
// generator when a session is finalized.
//
// Filename is the canonical filename for the document (e.g. "architecture.md").
// Content is the full Markdown text.
// LineCount is the number of lines in Content, for frontend display.
type GeneratedDocument struct {
	Filename  string `json:"filename"`
	Content   string `json:"content"`
	LineCount int    `json:"line_count"`
}

// Package shared contains types used by multiple backend modules.
// These types must not reference any module-specific packages.
package shared

// GeneratedDocument is one rendered output artifact produced by the markdown
// generator when a session is finalized.
//
// Filename is the canonical filename for the document (e.g. "architecture.md").
// Content is the full Markdown text.
// LineCount is the number of lines in Content, for frontend display.
// Source records how the document was produced:
//   - "deterministic" — package-level template generator only.
//   - "ai"            — AI rewrite pass succeeded (hybrid or ai mode).
//   - "ai_fallback"   — AI pass attempted but failed; scaffold returned.
type GeneratedDocument struct {
	Filename  string `json:"filename"`
	Content   string `json:"content"`
	LineCount int    `json:"line_count"`
	Source    string `json:"source,omitempty"`
}

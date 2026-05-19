// Package markdown generates the two output artifacts produced when a
// brainstorm session is finalized: architecture.md and roadmap.md.
//
// Both files are written atomically (write to a .tmp file, then rename)
// so a partial write never leaves a corrupt artifact on disk.
//
// The package has no DB access and no LLM calls — it is a pure text
// transformation from CanonicalState to Markdown.
package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GenerateArchitecture renders the architecture.md document from the
// Architecture map and ExecutionPlan in s.
// Returns an error only if the state is structurally empty.
func GenerateArchitecture(s state.CanonicalState) (string, error) {
	var b strings.Builder

	b.WriteString("# Architecture\n\n")

	// Idea section
	if len(s.Idea) > 0 {
		b.WriteString("## Idea\n\n")
		if text, ok := s.Idea["text"]; ok {
			b.WriteString(fmt.Sprintf("%v\n\n", text))
		} else {
			writeMap(&b, s.Idea)
		}
	}

	// Architecture section
	b.WriteString("## Architecture\n\n")
	if len(s.Architecture) > 0 {
		writeMap(&b, s.Architecture)
	} else {
		b.WriteString("_No architecture details recorded yet._\n\n")
	}

	// Execution Plan summary
	if len(s.ExecutionPlan) > 0 {
		b.WriteString("## Execution Plan\n\n")
		for i, step := range s.ExecutionPlan {
			b.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, step.Title))
			if step.Description != "" {
				b.WriteString(step.Description + "\n\n")
			}
		}
	}

	// Risks
	if len(s.Risks) > 0 {
		b.WriteString("## Risks\n\n")
		for _, r := range s.Risks {
			if r.Resolved {
				continue
			}
			b.WriteString(fmt.Sprintf("- **[%s]** %s\n", strings.ToUpper(r.Severity), r.Text))
		}
		b.WriteString("\n")
	}

	// Metadata footer
	b.WriteString(fmt.Sprintf("---\n_Generated at iteration %d._\n", s.Meta.Iteration))

	return b.String(), nil
}

// GenerateRoadmap renders the roadmap.md document from the ExecutionPlan in s.
// Each step becomes a checklist item with an estimated relative timeline
// derived from the step's position in the plan.
func GenerateRoadmap(s state.CanonicalState) (string, error) {
	var b strings.Builder

	b.WriteString("# Roadmap\n\n")

	if len(s.Idea) > 0 {
		if text, ok := s.Idea["text"]; ok {
			b.WriteString(fmt.Sprintf("> **Idea:** %v\n\n", text))
		}
	}

	if len(s.ExecutionPlan) == 0 {
		b.WriteString("_No execution plan steps recorded yet._\n")
		return b.String(), nil
	}

	now := time.Now()
	b.WriteString("## Milestones\n\n")
	b.WriteString("| # | Step | Description | Target |\n")
	b.WriteString("|---|------|-------------|--------|\n")

	for i, step := range s.ExecutionPlan {
		// Simple relative timeline: each step is estimated at +1 week from now.
		target := now.AddDate(0, 0, (i+1)*7).Format("2006-01-02")
		desc := step.Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		b.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n", i+1, step.Title, desc, target))
	}
	b.WriteString("\n")

	// Assumptions
	if len(s.Assumptions) > 0 {
		b.WriteString("## Assumptions\n\n")
		for _, a := range s.Assumptions {
			b.WriteString(fmt.Sprintf("- %s\n", a))
		}
		b.WriteString("\n")
	}

	// Open Questions
	if len(s.OpenQuestions) > 0 {
		b.WriteString("## Open Questions\n\n")
		for _, q := range s.OpenQuestions {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", q))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("---\n_Generated at iteration %d. Confidence: %.2f_\n",
		s.Meta.Iteration, s.Metrics.Confidence))

	return b.String(), nil
}

// GenerateContent produces both the architecture and roadmap markdown strings
// for a finalized session. It is the single entry-point used by callers that
// need the content in-memory (e.g. the finalize HTTP handler). WriteArtifacts
// calls this function internally so the generation logic lives in one place.
func GenerateContent(s state.CanonicalState) (arch string, roadmap string, err error) {
	arch, err = GenerateArchitecture(s)
	if err != nil {
		return "", "", fmt.Errorf("generate content: architecture: %w", err)
	}
	roadmap, err = GenerateRoadmap(s)
	if err != nil {
		return "", "", fmt.Errorf("generate content: roadmap: %w", err)
	}
	return arch, roadmap, nil
}

// WriteArtifacts writes architecture.md and roadmap.md to outputDir.
// Each file is written atomically: content is first written to a .tmp file,
// then renamed to the final path. If either write fails, the error is returned
// and the other file may or may not have been written.
func WriteArtifacts(s state.CanonicalState, outputDir string) error {
	arch, roadmap, err := GenerateContent(s)
	if err != nil {
		return fmt.Errorf("write artifacts: %w", err)
	}
	if err := writeAtomic(filepath.Join(outputDir, "architecture.md"), arch); err != nil {
		return fmt.Errorf("write artifacts: architecture.md: %w", err)
	}
	if err := writeAtomic(filepath.Join(outputDir, "roadmap.md"), roadmap); err != nil {
		return fmt.Errorf("write artifacts: roadmap.md: %w", err)
	}
	return nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

// Writer is a zero-value struct that implements the markdownWriter interface
// required by session.Handler. It delegates all work to the package-level
// functions so that callers can program to an interface without changing any
// generation logic.
type Writer struct{}

// GenerateContent satisfies the markdownWriter interface used by session.Handler.
func (w *Writer) GenerateContent(s state.CanonicalState) (arch string, roadmap string, err error) {
	return GenerateContent(s)
}

// WriteArtifacts satisfies the markdownWriter interface used by session.Handler.
func (w *Writer) WriteArtifacts(s state.CanonicalState, outputDir string) error {
	return WriteArtifacts(s, outputDir)
}

// writeAtomic writes content to destPath via a .tmp file and an atomic rename.
// This ensures readers never observe a partial write.
func writeAtomic(destPath, content string) error {
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create output dir %q: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".artifact-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file in %q: %w", dir, err)
	}
	tmpName := tmp.Name()

	// Ensure tmp is cleaned up on any failure path.
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, destPath); err != nil {
		return fmt.Errorf("rename %q → %q: %w", tmpName, destPath, err)
	}
	ok = true
	return nil
}

// writeMap writes the key-value pairs of a map[string]any as Markdown
// bullet points into the builder.
func writeMap(b *strings.Builder, m map[string]any) {
	for k, v := range m {
		b.WriteString(fmt.Sprintf("- **%s**: %v\n", k, v))
	}
	b.WriteString("\n")
}

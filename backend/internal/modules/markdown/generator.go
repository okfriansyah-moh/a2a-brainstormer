// Package markdown generates the output artifacts produced when a brainstorm
// session is finalized: architecture.md, roadmap.md, and (in Task 29) plan and
// readme documents.
//
// All files are written atomically (write to a .tmp file, then rename) so a
// partial write never leaves a corrupt artifact on disk.
//
// The package has no DB access and no LLM calls — it is a pure text
// transformation from CanonicalState to Markdown.
package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/shared"
)

// filenameForKey maps a document key to its canonical filename.
var filenameForKey = map[string]string{
	"architecture": "architecture.md",
	"roadmap":      "roadmap.md",
	"plan":         "PLAN.md",
	"readme":       "README.md",
}

// Generators is the registry of per-key generator functions.
// Add new document types by inserting entries here — never hardcode keys in
// service or handler layers.
var Generators = map[string]func(state.CanonicalState) (string, error){
	"architecture": GenerateArchitecture,
	"roadmap":      GenerateRoadmap,
	"plan":         GeneratePlan,
	"readme":       GenerateReadme,
}

// GenerateAll generates documents for each key in keys, using the Generators
// registry. Unknown keys are returned as an error.
// The returned map is keyed by the same keys that were requested.
func GenerateAll(s state.CanonicalState, keys []string) (map[string]shared.GeneratedDocument, error) {
	result := make(map[string]shared.GeneratedDocument, len(keys))
	for _, key := range keys {
		gen, ok := Generators[key]
		if !ok {
			return nil, fmt.Errorf("generate all: unknown document key %q", key)
		}
		content, err := gen(s)
		if err != nil {
			return nil, fmt.Errorf("generate all: key %q: %w", key, err)
		}
		filename := filenameForKey[key]
		result[key] = shared.GeneratedDocument{
			Filename:  filename,
			Content:   content,
			LineCount: strings.Count(content, "\n") + 1,
		}
	}
	return result, nil
}

// GenerateContent produces both the architecture and roadmap markdown strings
// for a finalized session. Kept for use by WriteArtifacts.
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

// GenerateAll satisfies the markdownWriter interface used by session.Handler.
func (w *Writer) GenerateAll(s state.CanonicalState, keys []string) (map[string]shared.GeneratedDocument, error) {
	return GenerateAll(s, keys)
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

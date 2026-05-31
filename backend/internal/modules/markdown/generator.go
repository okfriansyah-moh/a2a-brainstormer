// Package markdown generates the output artifacts produced when a brainstorm
// session is finalized: architecture, roadmap, plan, and readme.
//
// All files are written atomically (write to a .tmp file, then rename) so a
// partial write never leaves a corrupt artifact on disk.
//
// The package has no DB access and no LLM calls — it is a pure text
// transformation from CanonicalState to Markdown.
package markdown

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/shared"
)

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
// registry. Unknown keys are returned as an error. Filenames are derived from
// the canonical state's short title using buildFilename(title, key).
func GenerateAll(s state.CanonicalState, keys []string) (map[string]shared.GeneratedDocument, error) {
	title := shortTitle(s)
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
		result[key] = shared.GeneratedDocument{
			Filename:  buildFilename(title, key),
			Content:   content,
			LineCount: strings.Count(content, "\n") + 1,
			Source:    "deterministic",
		}
	}
	return result, nil
}

// WriteArtifacts writes the selected markdown documents to outputDir.
// Each file is written atomically: content is first written to a .tmp file
// then renamed to the final path. Filenames are derived from the canonical
// state's short title via buildFilename(title, key). The ctx parameter is
// accepted to satisfy the session-handler interface; the deterministic
// generators make no LLM calls and ignore it.
func WriteArtifacts(_ context.Context, s state.CanonicalState, outputDir string, keys []string) error {
	if len(keys) == 0 {
		keys = []string{"architecture", "roadmap"}
	}
	docs, err := GenerateAll(s, keys)
	if err != nil {
		return fmt.Errorf("write artifacts: %w", err)
	}
	for _, key := range keys {
		doc, ok := docs[key]
		if !ok {
			continue
		}
		if err := writeAtomic(filepath.Join(outputDir, doc.Filename), doc.Content); err != nil {
			return fmt.Errorf("write artifacts: %s: %w", doc.Filename, err)
		}
	}
	return nil
}

// Writer is a zero-value struct that implements the markdownWriter interface
// required by session.Handler. It delegates all work to the package-level
// functions so that callers can program to an interface without changing any
// generation logic.
type Writer struct{}

// GenerateAll satisfies the markdownWriter interface used by session.Handler.
// The context is accepted to match the interface but is not used — the
// deterministic generators make no LLM calls.
func (w *Writer) GenerateAll(_ context.Context, s state.CanonicalState, keys []string) (map[string]shared.GeneratedDocument, error) {
	return GenerateAll(s, keys)
}

// WriteArtifacts satisfies the markdownWriter interface used by session.Handler.
func (w *Writer) WriteArtifacts(ctx context.Context, s state.CanonicalState, outputDir string, keys []string) error {
	return WriteArtifacts(ctx, s, outputDir, keys)
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

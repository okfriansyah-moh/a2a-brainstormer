// Package markdown — finalize orchestrator that optionally layers an AI
// rewrite pass on top of the deterministic generators. See docs/PLAN.md §8.27.
package markdown

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"a2a-brainstorm/backend/internal/modules/markdown/aigen"
	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/shared"
)

// FinalizeMode controls the document-generation strategy used at finalize.
type FinalizeMode string

const (
	// FinalizeModeDeterministic uses the package-level Generators only — no
	// LLM calls. This is the historical Task-28/Task-32 behaviour and the
	// only mode that is byte-stable across runs.
	FinalizeModeDeterministic FinalizeMode = "deterministic"
	// FinalizeModeHybrid runs the deterministic generators first to produce a
	// seed scaffold, then asks the AI generator to rewrite each doc. Any AI
	// failure falls back silently to the scaffold for that doc.
	FinalizeModeHybrid FinalizeMode = "hybrid"
	// FinalizeModeAI uses the deterministic generators only as a seed; any
	// unrecoverable AI failure aborts finalize with an error. Use only when
	// the operator explicitly opts in.
	FinalizeModeAI FinalizeMode = "ai"
)

// ParseFinalizeMode normalises a config string into a FinalizeMode value.
// Unknown values fall back to FinalizeModeDeterministic.
func ParseFinalizeMode(raw string) FinalizeMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(FinalizeModeHybrid):
		return FinalizeModeHybrid
	case string(FinalizeModeAI):
		return FinalizeModeAI
	default:
		return FinalizeModeDeterministic
	}
}

// Orchestrator is the markdown-package entry point used by the session handler.
// It satisfies the same interface as Writer but adds an optional AI pass.
//
// The zero value behaves like Writer (deterministic only). Construct via
// NewOrchestrator to enable the AI path.
type Orchestrator struct {
	mode FinalizeMode
	ai   *aigen.Generator
}

// NewOrchestrator returns an Orchestrator. When mode is deterministic, or
// when ai is nil, the AI path is bypassed and behaviour matches Writer.
func NewOrchestrator(mode FinalizeMode, ai *aigen.Generator) *Orchestrator {
	return &Orchestrator{mode: mode, ai: ai}
}

// GenerateAll runs the deterministic pipeline, then (when configured) the AI
// enhancement pass. The returned map always contains every requested key.
// The caller-provided ctx is forwarded to the AI enhance pass so that the
// handler's finalize timeout (GetFinalizeTimeout) can cancel an overrunning
// LLM call instead of letting it run unchecked.
func (o *Orchestrator) GenerateAll(ctx context.Context, s state.CanonicalState, keys []string) (map[string]shared.GeneratedDocument, error) {
	scaffolds, err := GenerateAll(s, keys)
	if err != nil {
		return nil, err
	}
	if o == nil || o.ai == nil || o.mode == FinalizeModeDeterministic {
		return scaffolds, nil
	}
	enhanced, err := o.ai.Enhance(ctx, s, scaffolds)
	if err != nil {
		if o.mode == FinalizeModeHybrid {
			// Safety net: Enhance handles per-key errors internally in hybrid
			// mode, but if an unhandled error escapes (e.g. context already
			// cancelled before the loop, or a future code path), fall back to
			// the deterministic scaffold instead of failing the whole request.
			return scaffolds, nil
		}
		return nil, fmt.Errorf("orchestrator: ai enhance: %w", err)
	}
	return enhanced, nil
}

// WriteArtifacts generates the architecture and roadmap documents via the
// orchestrator's GenerateAll (so they include any AI enhancement) and writes
// each atomically into outputDir. This mirrors the package-level WriteArtifacts
// contract for compatibility with the session handler.
func (o *Orchestrator) WriteArtifacts(s state.CanonicalState, outputDir string) error {
	docs, err := o.GenerateAll(context.Background(), s, []string{"architecture", "roadmap"})
	if err != nil {
		return fmt.Errorf("write artifacts: %w", err)
	}
	for _, key := range []string{"architecture", "roadmap"} {
		doc := docs[key]
		if err := writeAtomic(filepath.Join(outputDir, doc.Filename), doc.Content); err != nil {
			return fmt.Errorf("write artifacts: %s: %w", doc.Filename, err)
		}
	}
	return nil
}

// Package aigen — progress callback types for document generation.
//
// ProgressFunc and TokenFunc are injected by the Orchestrator into
// EnhanceWithProgress so callers can forward SSE events without the
// generator importing the sse package.
package aigen

// DocStep enumerates the progress stages emitted during per-document generation.
type DocStep string

const (
	// DocStepEnricher is emitted before the LLM enricher pre-pass runs.
	DocStepEnricher DocStep = "enricher"
	// DocStepDraft is emitted before the initial AI rewrite call.
	DocStepDraft DocStep = "draft"
	// DocStepRepair is emitted before each rubric repair pass.
	DocStepRepair DocStep = "repair"
	// DocStepComplete is emitted when a document finishes successfully.
	DocStepComplete DocStep = "complete"
)

// ProgressFunc is called at key stages during document generation.
// docKey identifies the document (e.g. "architecture").
// step is the current generation stage.
// detail is a short human-readable description of what is happening.
type ProgressFunc func(docKey string, step DocStep, detail string)

// TokenFunc is called for each LLM token chunk during streaming generation.
// docKey identifies the document being generated.
// token is the raw text fragment from the LLM.
type TokenFunc func(docKey string, token string)

// EnhanceOpts bundles optional callbacks for EnhanceWithProgress.
// Zero value is valid — nil callbacks are silently skipped.
type EnhanceOpts struct {
	ProgressFn ProgressFunc
	TokenFn    TokenFunc
}

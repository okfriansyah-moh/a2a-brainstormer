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
	// DocStepSectionEnhance is emitted before each section enhance LLM call.
	DocStepSectionEnhance DocStep = "section_enhance"
	// DocStepSectionRepair is emitted before each section-scoped repair call.
	DocStepSectionRepair DocStep = "section_repair"
	// DocStepCoherenceAudit is emitted before the coherence audit LLM call.
	DocStepCoherenceAudit DocStep = "coherence_audit"
	// DocStepCoherenceFix is emitted before a coherence micro-fix call.
	DocStepCoherenceFix DocStep = "coherence_fix"
)

// ProgressMeta carries optional section-level SSE fields.
type ProgressMeta struct {
	Section       string
	FindingsCount int
}

// ProgressFunc is called at key stages during document generation.
type ProgressFunc func(docKey string, step DocStep, detail string, meta ProgressMeta)

// TokenFunc is called for each LLM token chunk during streaming generation.
type TokenFunc func(docKey string, token string)

// EnhanceOpts bundles optional callbacks for EnhanceWithProgress.
type EnhanceOpts struct {
	ProgressFn ProgressFunc
	TokenFn    TokenFunc
}

// EmitProgress invokes ProgressFn when set.
func (o EnhanceOpts) EmitProgress(docKey string, step DocStep, detail string, meta ProgressMeta) {
	if o.ProgressFn != nil {
		o.ProgressFn(docKey, step, detail, meta)
	}
}

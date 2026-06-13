// Package aigen — readme enricher: single LLM pre-pass that fills optional
// narrative fields in CanonicalState before the README generator renders.
// See docs/PLAN.md §8.32.
package aigen

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/llm"
)

// ─── Overlay types (§8.32) ────────────────────────────────────────────────────

// ReadmeEnrichmentOverlay holds the structured enrichment data the LLM
// pre-pass returns for the README document. Every field is optional; absent
// fields leave the canonical state unchanged (never overwrite existing values).
type ReadmeEnrichmentOverlay struct {
	Tagline              string            `json:"tagline"`
	DescriptionParagraph string            `json:"description_paragraph"`
	GoldenRule           string            `json:"golden_rule"`
	IsNot                []string          `json:"is_not"`
	WhenToUse            []WhenToUseScenario `json:"when_to_use"`
	WhenToUseMermaid     string            `json:"when_to_use_mermaid"`
	QuickStartCommands   []string          `json:"quick_start_commands"`
	CommandReference     []CommandRef      `json:"command_reference"`
	ArchitectureASCII    string            `json:"architecture_ascii"`
	ArchitectureMermaid  string            `json:"architecture_mermaid"`
	Prerequisites        []Prerequisite    `json:"prerequisites"`
	RepoFormatNotes      map[string]string `json:"repo_format_notes"`
	ContributingNote     string            `json:"contributing_note"`
}

// WhenToUseScenario is one numbered entry in the "When to use" section.
type WhenToUseScenario struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Code        string `json:"code"`
}

// CommandRef is one row in the Command Reference table.
type CommandRef struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// Prerequisite is one row in the Installation prerequisites table.
type Prerequisite struct {
	Tool     string `json:"tool"`
	Version  string `json:"version"`
	Required bool   `json:"required"`
}

// ─── ReadmeEnricher ───────────────────────────────────────────────────────────

// ReadmeEnricher runs a single LLM pre-pass to populate optional README
// narrative fields in the canonical state before the README generator renders.
// It is a pure augmenter: on any error it returns the original state unchanged
// and logs at Warn level — no error is propagated.
type ReadmeEnricher struct {
	llm        llm.LLMProvider
	timeoutSec int
}

// NewReadmeEnricher constructs a ReadmeEnricher. timeoutSec is the LLM call
// deadline in seconds; it should be obtained from config.GetReadmeEnricherTimeoutSec().
func NewReadmeEnricher(l llm.LLMProvider, timeoutSec int) *ReadmeEnricher {
	return &ReadmeEnricher{llm: l, timeoutSec: timeoutSec}
}

// readmeEnricherSystemPrompt is the LLM system prompt per §8.32.3.
const readmeEnricherSystemPrompt = `You are an expert technical writer generating a README enrichment overlay for a software project.

GOAL: Produce content for a beginner-friendly README — short, scannable, product-focused. The reader just found this project on GitHub and needs to decide "is this for me?" in under 5 minutes.

INPUT: A CanonicalState JSON object describing a project.

OUTPUT: A single JSON object conforming to ReadmeEnrichmentOverlay (all fields optional).

RULES:
1. tagline: ≤ 120 chars. One punchy sentence. No marketing fluff. Completes: "{ProductName} is a ..."
2. golden_rule: One sentence — the single most important invariant the project upholds.
3. is_not: 3–5 one-sentence bullets. Scope boundaries and common misconceptions. Start each with "Not a ..." or "Not designed to ...".
4. when_to_use: 3–5 numbered scenarios. Each scenario: title (≤ 8 words) + description (2–3 sentences max) + a real runnable code snippet (not pseudo-code). Scenarios must cover DISTINCT use cases.
5. when_to_use_mermaid: A mermaid flowchart (flowchart LR or TD) of 5–8 nodes showing when to use this product vs alternatives. Keep it simple.
6. quick_start_commands: 3–5 real commands that get the project running from zero (e.g., ` + "`cp .env.example .env`" + `, ` + "`docker compose up --build`" + `). Never use ` + "`<placeholder>`" + ` values.
7. command_reference: 5–8 entries MAX. Core commands only: start, stop, test, build, migrate. NOT a full command encyclopedia.
8. architecture_ascii: 8–12 line ASCII diagram of major components and data flow. Use box-drawing characters. Keep it readable at a glance.
9. architecture_mermaid: Simple mermaid component or flow diagram matching the ASCII above.
10. prerequisites: 3–6 tools with minimum versions. Required only — not a full dependency list.
11. contributing_note: 2–3 sentences: what to read first, how to run tests, PR expectations.

FORBIDDEN in any field:
- Environment variable configuration tables (these go in separate docs)
- Troubleshooting sections
- Database schema or domain model tables
- Internal implementation details
- Any content longer than 5 sentences for a single field

QUALITY BAR: All content must be specific to THIS project. Generic boilerplate is forbidden.`

// Enrich runs the LLM pre-pass against s and returns an updated copy with
// optional README fields populated. The enricher always wins: if the state
// already has a field and the overlay has a non-empty value, the overlay wins.
//
// On any error (LLM failure, timeout, JSON parse error) Enrich returns the
// original s unchanged and logs at slog.Warn. No error is propagated.
func (e *ReadmeEnricher) Enrich(ctx context.Context, s state.CanonicalState) (state.CanonicalState, error) {
	if e == nil || e.llm == nil {
		return s, nil
	}

	input := buildReadmeEnricherInput(s)
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return s, nil
	}

	deadline := time.Duration(e.timeoutSec) * time.Second
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	resp, err := e.llm.Generate(ctx, llm.LLMRequest{
		SystemPrompt:   readmeEnricherSystemPrompt,
		UserMessage:    string(inputJSON),
		Temperature:    0.2,
		ResponseFormat: "json",
	})
	if err != nil {
		slog.WarnContext(ctx, "readme_enricher_failed", slog.Any("error", err))
		return s, nil
	}

	raw := stripJSONFencing(resp.Content)

	var overlay ReadmeEnrichmentOverlay
	if err := json.Unmarshal([]byte(raw), &overlay); err != nil {
		slog.WarnContext(ctx, "readme_enricher_invalid_json", slog.Any("error", err))
		return s, nil
	}

	return mergeReadmeOverlay(s, overlay), nil
}

// buildReadmeEnricherInput builds a compact representation of the canonical
// state for the LLM pre-pass. Only includes fields relevant to README generation.
func buildReadmeEnricherInput(s state.CanonicalState) map[string]any {
	out := map[string]any{
		"idea":        s.Idea,
		"iteration":   s.Meta.Iteration,
		"confidence":  s.Metrics.Confidence,
	}
	// Include a minimal architecture snapshot (omit large enrichment keys)
	if s.Architecture != nil {
		arch := make(map[string]any)
		for _, k := range []string{"tech_stack", "layers", "components", "data_flows", "decisions"} {
			if v, ok := s.Architecture[k]; ok {
				arch[k] = v
			}
		}
		out["architecture_summary"] = arch
	}
	return out
}

// mergeReadmeOverlay applies the enricher overlay to a copy of s.
// The enricher always wins: non-empty overlay fields overwrite whatever is
// currently in s.Architecture. String fields are stored directly; slice/map
// fields go through a JSON roundtrip so they are always stored as []any /
// map[string]any (the type the generators expect).
func mergeReadmeOverlay(s state.CanonicalState, o ReadmeEnrichmentOverlay) state.CanonicalState {
	arch := copyMap(s.Architecture)
	if arch == nil {
		arch = make(map[string]any)
	}

	// String fields: store directly.
	if o.Tagline != "" {
		arch["tagline"] = o.Tagline
	}
	if o.DescriptionParagraph != "" {
		arch["description_paragraph"] = o.DescriptionParagraph
	}
	if o.GoldenRule != "" {
		arch["golden_rule"] = o.GoldenRule
	}
	if o.WhenToUseMermaid != "" {
		arch["when_to_use_mermaid"] = o.WhenToUseMermaid
	}
	if o.ArchitectureASCII != "" {
		arch["architecture_ascii"] = o.ArchitectureASCII
	}
	if o.ArchitectureMermaid != "" {
		arch["architecture_mermaid"] = o.ArchitectureMermaid
	}
	if o.ContributingNote != "" {
		arch["contributing_note"] = o.ContributingNote
	}

	// Slice/map fields: JSON roundtrip so downstream sees []any / map[string]any.
	if len(o.IsNot) > 0 {
		arch["is_not"] = jsonRoundtrip(o.IsNot)
	}
	if len(o.WhenToUse) > 0 {
		arch["when_to_use"] = jsonRoundtrip(o.WhenToUse)
	}
	if len(o.QuickStartCommands) > 0 {
		arch["quick_start_commands"] = jsonRoundtrip(o.QuickStartCommands)
	}
	if len(o.CommandReference) > 0 {
		arch["command_reference"] = jsonRoundtrip(o.CommandReference)
	}
	if len(o.Prerequisites) > 0 {
		arch["prerequisites"] = jsonRoundtrip(o.Prerequisites)
	}
	if len(o.RepoFormatNotes) > 0 {
		arch["repo_format_notes"] = jsonRoundtrip(o.RepoFormatNotes)
	}

	out := s
	out.Architecture = arch
	return out
}

// jsonRoundtrip marshals v to JSON and back into any, so slices and maps are
// always stored as []any / map[string]any — the types the generators expect.
func jsonRoundtrip(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return v
	}
	return out
}

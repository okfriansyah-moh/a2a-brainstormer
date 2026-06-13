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

INPUT: A CanonicalState JSON object describing a project being designed: its idea, architecture decisions, tech stack, modules, and metrics.

OUTPUT: A single JSON object conforming to ReadmeEnrichmentOverlay (all fields optional).

RULES:
1. tagline: ≤ 120 chars. One punchy sentence. No marketing fluff. Should complete: "{ProductName} is a ..."
2. golden_rule: One sentence stating the single most important invariant — the property that must always hold. Example: "same input + same config = identical output, always."
3. is_not: 3-5 short bullets clarifying what the product is NOT (scope boundaries, common misconceptions). Start each with "Not a ..." or "Not designed to ...".
4. when_to_use: 3-6 numbered scenarios. Each must have a realistic ` + "`code`" + ` example — a real command or config snippet, not pseudo-code. Scenarios must be distinct (different use cases, not variations of the same thing).
5. when_to_use_mermaid: A mermaid flowchart (flowchart LR or TD) showing the decision path for when to use this product vs alternatives. 5-9 nodes. Use ` + "`-->`" + ` for arrows and ` + "`[text]`" + ` for rectangles, ` + "`{text}`" + ` for diamonds.
6. quick_start_commands: 2-5 commands. Real commands for the tech stack described in the state (e.g., ` + "`make dev`" + `, ` + "`docker compose up`" + `, ` + "`go run ./cmd/server`" + `). Never use ` + "`<repository-url>`" + ` or ` + "`<project>`" + ` placeholders.
7. command_reference: 3-10 entries. Cover the most important developer-facing commands (build, test, run, deploy, migrate).
8. architecture_ascii: A clean ASCII diagram showing the major system components and their connections. Use box-drawing chars (─ │ ┌ └ ┐ ┘ ├ ┤ ┬ ┴ ┼) for clarity. 8-15 lines.
9. architecture_mermaid: A mermaid diagram (flowchart or graph) of the architecture. Match the ASCII above.
10. prerequisites: List only tools actually required by the tech stack. Include versions where known.
11. contributing_note: 2-3 sentences. What to read before contributing (docs, skill files, test commands).

QUALITY BAR: Content must be specific to THIS project (based on its idea, architecture, tech stack). Generic placeholder text is forbidden.`

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

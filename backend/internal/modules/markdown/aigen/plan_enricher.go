// Package aigen — plan enricher: single LLM pre-pass that adds per-phase
// quality fields (coding standards, invariant checks, layer tags, prompt context
// refs) and a dependency graph to the canonical state before the plan generator
// renders the implementation-plan document. See docs/PLAN.md §8.31.
package aigen

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/config"
	"a2a-brainstorm/backend/internal/platform/llm"
)

// ─── Overlay types (§8.31) ────────────────────────────────────────────────────

// PlanEnrichmentOverlay holds the structured enrichment data the LLM pre-pass
// returns. Every field is optional; absent fields leave the canonical state
// unchanged (never overwrite existing values).
type PlanEnrichmentOverlay struct {
	Phases                []PhaseEnrichment `json:"phases"`
	DependencyGraphASCII  string            `json:"dependency_graph_ascii"`
	DeepKnowledgeSections []DKSection       `json:"deep_knowledge_sections"`
}

// PhaseEnrichment holds AI-agent-readable quality fields for one execution plan
// step. Position is the 0-based index into s.ExecutionPlan.
type PhaseEnrichment struct {
	Position          int      `json:"position"`
	CodingStandards   []string `json:"coding_standards"`
	InvariantChecks   []string `json:"invariant_checks"`
	LayerTags         []string `json:"layer_tags"`
	PromptContextRefs []string `json:"prompt_context_refs"`
}

// DKSection is one entry in the deep-knowledge reference produced by the
// enricher — a §8.N excerpt an AI agent needs when executing the task session.
type DKSection struct {
	Heading string `json:"heading"`
	Content string `json:"content"`
}

// ─── Validation ───────────────────────────────────────────────────────────────

const (
	maxCodingStandardLen  = 200
	maxInvariantCheckLen  = 80
	maxDependencyGraphLen = 3000
	maxDKContentLen       = 5000
	maxSliceItems         = 30
)

// validateOverlay checks the overlay against §8.31 rules.
// Returns a non-nil error describing the first violation found.
func validateOverlay(o PlanEnrichmentOverlay, planLen int) error {
	if len(o.Phases) != planLen {
		return fmt.Errorf("phases length %d does not match execution plan length %d", len(o.Phases), planLen)
	}
	if len(o.DependencyGraphASCII) > maxDependencyGraphLen {
		return fmt.Errorf("dependency_graph_ascii exceeds %d chars", maxDependencyGraphLen)
	}
	if len(o.DeepKnowledgeSections) > maxSliceItems {
		return fmt.Errorf("deep_knowledge_sections has %d items; max %d", len(o.DeepKnowledgeSections), maxSliceItems)
	}
	for i, dks := range o.DeepKnowledgeSections {
		if len(dks.Content) > maxDKContentLen {
			return fmt.Errorf("deep_knowledge_sections[%d].content exceeds %d chars", i, maxDKContentLen)
		}
	}
	for i, ph := range o.Phases {
		if len(ph.CodingStandards) > maxSliceItems {
			return fmt.Errorf("phases[%d].coding_standards has %d items; max %d", i, len(ph.CodingStandards), maxSliceItems)
		}
		for j, cs := range ph.CodingStandards {
			if cs == "" {
				return fmt.Errorf("phases[%d].coding_standards[%d] is empty", i, j)
			}
			if len(cs) > maxCodingStandardLen {
				return fmt.Errorf("phases[%d].coding_standards[%d] exceeds %d chars", i, j, maxCodingStandardLen)
			}
		}
		if len(ph.InvariantChecks) > maxSliceItems {
			return fmt.Errorf("phases[%d].invariant_checks has %d items; max %d", i, len(ph.InvariantChecks), maxSliceItems)
		}
		for j, ic := range ph.InvariantChecks {
			if ic == "" {
				return fmt.Errorf("phases[%d].invariant_checks[%d] is empty", i, j)
			}
			if len(ic) > maxInvariantCheckLen {
				return fmt.Errorf("phases[%d].invariant_checks[%d] exceeds %d chars", i, j, maxInvariantCheckLen)
			}
			if len(ic) > 0 && (ic[0] < 'A' || ic[0] > 'Z') {
				return fmt.Errorf("phases[%d].invariant_checks[%d] does not start with a capital letter", i, j)
			}
		}
		for j, lt := range ph.LayerTags {
			if !strings.HasPrefix(lt, "backend/") && !strings.HasPrefix(lt, "agent/") && !strings.HasPrefix(lt, "frontend/") {
				return fmt.Errorf("phases[%d].layer_tags[%d] %q must start with backend/, agent/, or frontend/", i, j, lt)
			}
		}
	}
	return nil
}

// ─── PlanEnricher ─────────────────────────────────────────────────────────────

// PlanEnricher runs a single LLM pre-pass to populate per-phase quality fields
// and the dependency graph in the canonical state before the plan generator
// renders. It is a pure augmenter: it only fills absent fields; it never
// overwrites values already present. On any error it returns the original state
// unchanged and logs at Warn level — no error is propagated.
type PlanEnricher struct {
	llm    llm.LLMProvider
	logger *slog.Logger
}

// NewPlanEnricher constructs a PlanEnricher. A nil logger is replaced with
// slog.Default().
func NewPlanEnricher(provider llm.LLMProvider, logger *slog.Logger) *PlanEnricher {
	if logger == nil {
		logger = slog.Default()
	}
	return &PlanEnricher{llm: provider, logger: logger}
}

// planEnricherSystemPrompt is the LLM prompt fragment per §8.31.4.
const planEnricherSystemPrompt = `You are a software architecture assistant. You receive a summary of implementation plan phases and return a structured JSON enrichment that improves the AI-agent readability of each task.

For each phase, produce:
- coding_standards: 3-5 SOLID/YAGNI/DRY principles applied to THIS specific phase's code
- invariant_checks: 2-4 task-specific architectural invariants (in addition to the canonical 4)
- layer_tags: module paths affected (backend/internal/modules/X/, agent/internal/Y/, frontend/src/...)
- prompt_context_refs: §8.N references an agent needs to execute this task

Also produce:
- dependency_graph_ascii: ASCII art showing task order and parallelism (use ─►, │, ▼, ┌, ┐, └, ┘)
- deep_knowledge_sections: 2-4 §8 entries (schemas, algorithms, contracts) needed across the task sessions

Return ONLY valid JSON matching the schema. No prose. No markdown fences.

Respond in under 4000 tokens. Prefer concrete specifics over generic platitudes.
For coding_standards: name the actual function/struct, not "SRP: keep it focused".
For invariant_checks: start with a verb ("No X outside Y", "All Z via W only").`

// Enrich runs the LLM pre-pass against s and returns an updated copy with
// optional plan enrichment fields populated. Merge rule: only absent/nil/empty
// fields are filled — existing values are never overwritten.
//
// On any error (LLM failure, timeout, JSON parse error, validation failure)
// Enrich returns the original s unchanged and logs at slog.Warn. No error is
// propagated.
func (e *PlanEnricher) Enrich(ctx context.Context, s state.CanonicalState) (state.CanonicalState, error) {
	if e == nil || e.llm == nil {
		return s, nil
	}
	if len(s.ExecutionPlan) == 0 {
		return s, nil
	}

	timeout := time.Duration(config.GetPlanEnricherTimeoutSec()) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	userPrompt := buildPlanEnricherUserPrompt(s)

	resp, err := e.llm.Generate(ctx, llm.LLMRequest{
		SystemPrompt:   planEnricherSystemPrompt,
		UserMessage:    userPrompt,
		Temperature:    0.1,
		ResponseFormat: "json_object",
	})
	if err != nil {
		e.logger.WarnContext(ctx, "plan_enricher_fallback",
			slog.String("reason", fmt.Sprintf("LLM call failed: %v", err)),
		)
		return s, nil
	}

	raw := strings.TrimSpace(resp.Content)
	if raw == "" {
		e.logger.WarnContext(ctx, "plan_enricher_fallback",
			slog.String("reason", "LLM returned empty response"),
		)
		return s, nil
	}

	raw = stripJSONFencing(raw)

	var overlay PlanEnrichmentOverlay
	if err := json.Unmarshal([]byte(raw), &overlay); err != nil {
		e.logger.WarnContext(ctx, "plan_enricher_fallback",
			slog.String("reason", fmt.Sprintf("JSON parse error: %v", err)),
		)
		return s, nil
	}

	if err := validateOverlay(overlay, len(s.ExecutionPlan)); err != nil {
		e.logger.WarnContext(ctx, "plan_enricher_fallback",
			slog.String("reason", fmt.Sprintf("overlay validation failed: %v", err)),
		)
		return s, nil
	}

	return mergePlanOverlay(s, overlay), nil
}

// buildPlanEnricherUserPrompt serialises the execution plan phases for the LLM
// pre-pass. Includes only the fields relevant for enrichment.
func buildPlanEnricherUserPrompt(s state.CanonicalState) string {
	type phaseInput struct {
		Position    int      `json:"position"`
		Title       string   `json:"title"`
		Objective   string   `json:"objective,omitempty"`
		Deliverable []string `json:"deliverables,omitempty"`
		ExitCrit    []string `json:"exit_criteria,omitempty"`
	}
	phases := make([]phaseInput, len(s.ExecutionPlan))
	for i, step := range s.ExecutionPlan {
		phases[i] = phaseInput{
			Position:    i,
			Title:       step.Title,
			Objective:   step.Objective,
			Deliverable: step.Deliverables,
			ExitCrit:    step.ExitCriteria,
		}
	}
	payload := map[string]any{
		"phases":     phases,
		"idea_title": stringFromMap(s.Idea, "name"),
		"iteration":  s.Meta.Iteration,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

// mergePlanOverlay applies the enricher overlay to a copy of s, filling only
// absent fields. Returns the updated copy.
func mergePlanOverlay(s state.CanonicalState, o PlanEnrichmentOverlay) state.CanonicalState {
	arch := copyMap(s.Architecture)
	if arch == nil {
		arch = make(map[string]any)
	}

	// Only fill dependency_graph_ascii when absent.
	if o.DependencyGraphASCII != "" {
		if _, exists := arch["dependency_graph_ascii"]; !exists {
			arch["dependency_graph_ascii"] = o.DependencyGraphASCII
		}
	}

	// Only fill deep_knowledge_sections when absent.
	if len(o.DeepKnowledgeSections) > 0 {
		if _, exists := arch["deep_knowledge_sections"]; !exists {
			raw, _ := json.Marshal(o.DeepKnowledgeSections)
			var v any
			_ = json.Unmarshal(raw, &v)
			arch["deep_knowledge_sections"] = v
		}
	}

	// Only fill plan_phase_enrichments when absent.
	if len(o.Phases) > 0 {
		if _, exists := arch["plan_phase_enrichments"]; !exists {
			raw, _ := json.Marshal(o.Phases)
			var v any
			_ = json.Unmarshal(raw, &v)
			arch["plan_phase_enrichments"] = v
		}
	}

	out := s
	out.Architecture = arch
	return out
}

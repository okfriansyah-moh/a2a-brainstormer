// Package aigen — architecture enricher: single LLM pre-pass that fills
// optional narrative fields in CanonicalState before the 16-section generator
// renders the architecture document. See docs/PLAN.md §8.30.
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

// ─── Overlay types (§8.30) ───────────────────────────────────────────────────

// ArchEnrichmentOverlay holds the optional narrative fields the LLM pre-pass
// populates. Every field is optional; absent fields leave the canonical state
// unchanged (never overwrite existing values).
type ArchEnrichmentOverlay struct {
	ProblemStatement    string             `json:"problem_statement"`
	SolutionSummary     string             `json:"solution_summary"`
	ScopeIn             []string           `json:"scope_in"`
	ScopeOut            []string           `json:"scope_out"`
	ExtensionPoints     []OverlayExtPoint  `json:"extension_points"`
	Security            []OverlaySecEntry  `json:"security"`
	Guarantees          []string           `json:"guarantees"`
	DecisionEnrichments []OverlayDecEnrich `json:"decision_enrichments"`
	TechStackRationale  map[string]string  `json:"tech_stack_rationale"`
	DataFlowProse       string             `json:"data_flow_prose"`
	ExitCriteria        []string           `json:"exit_criteria"`
}

// OverlayExtPoint is one extension point with an ordered list of implementation steps.
type OverlayExtPoint struct {
	Name  string   `json:"name"`
	Steps []string `json:"steps"`
}

// OverlaySecEntry is one row in the security considerations table.
type OverlaySecEntry struct {
	Surface    string `json:"surface"`
	Risk       string `json:"risk"`
	Mitigation string `json:"mitigation"`
}

// OverlayDecEnrich adds alternatives and tradeoff narrative to an existing
// architecture decision. Title must exactly match an existing decision title.
type OverlayDecEnrich struct {
	Title        string `json:"title"`
	Alternatives string `json:"alternatives"`
	Tradeoff     string `json:"tradeoff"`
}

// ─── ArchEnricher ─────────────────────────────────────────────────────────────

// ArchEnricher runs a single LLM pre-pass to populate optional narrative fields
// in the canonical state before the 16-section architecture generator renders.
// It is a pure augmenter: it only fills absent fields; it never overwrites
// values already present in the state. On any error it returns the original
// state unchanged and logs at Warn level — no error is propagated.
type ArchEnricher struct {
	llm    llm.LLMProvider
	logger *slog.Logger
}

// NewArchEnricher constructs an ArchEnricher. A nil logger is replaced with
// slog.Default().
func NewArchEnricher(provider llm.LLMProvider, logger *slog.Logger) *ArchEnricher {
	if logger == nil {
		logger = slog.Default()
	}
	return &ArchEnricher{llm: provider, logger: logger}
}

const enricherSystemPrompt = `You are an architecture documentation assistant for the USER'S PRODUCT described in the input JSON — not the brainstorm tool that produced the session.

Given a compact JSON description of the product's brainstorm state, return a JSON object that fills in the gaps needed to render a high-quality architecture.md document.

Rules:
- Return ONLY valid JSON matching the output schema. No markdown fencing. No prose outside the JSON.
- Keep all strings concise — respect the per-field character limits in the schema.
- Do NOT invent technical facts not implied by the input. Infer from what is present.
- Focus on the product domain (users, problem, solution, integrations) — never describe A2A protocols, agent pipelines, or this repository's modules unless they are the product.
- If a field cannot be meaningfully populated from the input, omit it entirely.
- ` + "`decision_enrichments[].title`" + ` must EXACTLY match one of the decision titles in the input.

Output schema (all fields optional):
{
  "problem_statement": "string ≤400 chars",
  "solution_summary": "string ≤400 chars",
  "scope_in": ["string ≤80 chars"],
  "scope_out": ["string ≤80 chars"],
  "extension_points": [{"name": "string ≤60 chars", "steps": ["string ≤100 chars"]}],
  "security": [{"surface": "≤60", "risk": "≤120", "mitigation": "≤120"}],
  "guarantees": ["string ≤120 chars"],
  "decision_enrichments": [{"title": "exact match", "alternatives": "≤150", "tradeoff": "≤150"}],
  "tech_stack_rationale": {"tech_name": "rationale ≤100 chars"},
  "data_flow_prose": "string ≤300 chars",
  "exit_criteria": ["string ≤100 chars"]
}`

// Enrich runs the LLM pre-pass against s and returns an updated copy with
// optional fields populated. Merge rule: only absent/nil/empty fields are
// filled — existing values are never overwritten.
//
// On any error (LLM failure, timeout, JSON parse error) Enrich returns the
// original s unchanged and logs at slog.Warn. No error is propagated.
func (e *ArchEnricher) Enrich(ctx context.Context, s state.CanonicalState) (state.CanonicalState, error) {
	if e == nil || e.llm == nil {
		return s, nil
	}

	timeout := time.Duration(config.GetArchEnricherTimeoutSec()) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	userPrompt := buildEnricherUserPrompt(s)

	resp, err := e.llm.Generate(ctx, llm.LLMRequest{
		SystemPrompt:   enricherSystemPrompt,
		UserMessage:    userPrompt,
		Temperature:    0.1,
		ResponseFormat: "json_object",
	})
	if err != nil {
		e.logger.WarnContext(ctx, "arch_enricher_fallback",
			slog.String("reason", fmt.Sprintf("LLM call failed: %v", err)),
		)
		return s, nil
	}

	raw := strings.TrimSpace(resp.Content)
	if raw == "" {
		e.logger.WarnContext(ctx, "arch_enricher_fallback",
			slog.String("reason", "LLM returned empty response"),
		)
		return s, nil
	}

	// Strip markdown fencing if the LLM added it despite instructions.
	raw = stripJSONFencing(raw)

	var overlay ArchEnrichmentOverlay
	if err := json.Unmarshal([]byte(raw), &overlay); err != nil {
		e.logger.WarnContext(ctx, "arch_enricher_fallback",
			slog.String("reason", fmt.Sprintf("JSON parse error: %v", err)),
		)
		return s, nil
	}

	out := mergeOverlay(s, overlay)
	return out, nil
}

// buildEnricherUserPrompt serialises the relevant parts of the canonical state
// for the LLM pre-pass. Keeps token usage minimal — only the fields the
// enricher can act on are included.
func buildEnricherUserPrompt(s state.CanonicalState) string {
	// Collect decision titles for the decision_enrichments constraint.
	decisionTitles := extractDecisionTitles(s)

	payload := map[string]any{
		"idea":            s.Idea,
		"architecture":    s.Architecture,
		"risks_count":     len(s.Risks),
		"assumptions":     s.Assumptions,
		"open_questions":  s.OpenQuestions,
		"decision_titles": decisionTitles,
		"iteration":       s.Meta.Iteration,
		"confidence":      s.Metrics.Confidence,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

// extractDecisionTitles returns the list of decision titles from
// s.Architecture["decisions"] for use as the allowed set in decision_enrichments.
func extractDecisionTitles(s state.CanonicalState) []string {
	v, ok := s.Architecture["decisions"]
	if !ok {
		return nil
	}
	// decisions may be []string or []map[string]any
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		titles := make([]string, 0, len(x))
		for _, e := range x {
			switch t := e.(type) {
			case string:
				titles = append(titles, t)
			case map[string]any:
				if title, ok := t["title"].(string); ok {
					titles = append(titles, title)
				} else if name, ok := t["name"].(string); ok {
					titles = append(titles, name)
				}
			}
		}
		return titles
	}
	return nil
}

// mergeOverlay applies the enricher overlay to a copy of s, filling only
// absent fields. Returns the updated copy.
func mergeOverlay(s state.CanonicalState, o ArchEnrichmentOverlay) state.CanonicalState {
	// Deep-copy the idea and architecture maps to avoid mutating the original.
	idea := copyMap(s.Idea)
	arch := copyMap(s.Architecture)

	// idea fields
	if o.ProblemStatement != "" {
		setIfAbsent(idea, "problem_statement", o.ProblemStatement)
	}
	if o.SolutionSummary != "" {
		setIfAbsent(idea, "solution_summary", o.SolutionSummary)
	}

	// architecture fields
	if len(o.ScopeIn) > 0 || len(o.ScopeOut) > 0 {
		if _, exists := arch["scope"]; !exists {
			scope := map[string]any{
				"in":  sliceToAny(o.ScopeIn),
				"out": sliceToAny(o.ScopeOut),
			}
			arch["scope"] = scope
		}
	}
	if len(o.ExtensionPoints) > 0 {
		if _, exists := arch["extension_points"]; !exists {
			raw, _ := json.Marshal(o.ExtensionPoints)
			var v any
			_ = json.Unmarshal(raw, &v)
			arch["extension_points"] = v
		}
	}
	if len(o.Security) > 0 {
		if _, exists := arch["security"]; !exists {
			raw, _ := json.Marshal(o.Security)
			var v any
			_ = json.Unmarshal(raw, &v)
			arch["security"] = v
		}
	}
	if len(o.Guarantees) > 0 {
		if _, exists := arch["guarantees"]; !exists {
			arch["guarantees"] = sliceToAny(o.Guarantees)
		}
	}
	if len(o.DecisionEnrichments) > 0 {
		if _, exists := arch["decision_enrichments"]; !exists {
			raw, _ := json.Marshal(o.DecisionEnrichments)
			var v any
			_ = json.Unmarshal(raw, &v)
			arch["decision_enrichments"] = v
		}
	}
	if len(o.TechStackRationale) > 0 {
		if _, exists := arch["tech_stack_rationale"]; !exists {
			m := make(map[string]any, len(o.TechStackRationale))
			for k, v := range o.TechStackRationale {
				m[k] = v
			}
			arch["tech_stack_rationale"] = m
		}
	}
	if o.DataFlowProse != "" {
		if _, exists := arch["data_flow_prose"]; !exists {
			arch["data_flow_prose"] = o.DataFlowProse
		}
	}
	if len(o.ExitCriteria) > 0 {
		if _, exists := arch["exit_criteria"]; !exists {
			arch["exit_criteria"] = sliceToAny(o.ExitCriteria)
		}
	}

	out := s
	out.Idea = idea
	out.Architecture = arch
	return out
}

// copyMap returns a shallow copy of m. Returns nil when m is nil.
func copyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// setIfAbsent sets m[key] = value only when m[key] is absent or the empty string.
func setIfAbsent(m map[string]any, key string, value any) {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return
		}
	}
	m[key] = value
}

// sliceToAny converts a []string to []any for map storage.
func sliceToAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// stripJSONFencing removes ```json ... ``` or ``` ... ``` wrappers that some
// LLM providers add despite explicit instructions not to.
func stripJSONFencing(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence (```json or ```)
		end := strings.Index(s, "\n")
		if end >= 0 {
			s = s[end+1:]
		}
		// Remove closing fence
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	return s
}

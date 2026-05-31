// Package state defines the CanonicalState type — the shared brainstorm
// document passed between agents during each iteration pipeline pass.
//
// The JSON field names in this package are canonical: downstream agents and the
// DB JSONB column rely on this exact shape. See §8.1 of docs/PLAN.md.
package state

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// CanonicalState is the immutable snapshot of a brainstorming session at a
// given iteration boundary. All agent Dispatch calls receive and return this type.
type CanonicalState struct {
	Idea          map[string]any `json:"idea"`
	Architecture  map[string]any `json:"architecture"`
	ExecutionPlan []Step         `json:"execution_plan"`
	Risks         []Risk         `json:"risks"`
	Assumptions   []string       `json:"assumptions"`
	OpenQuestions []string       `json:"open_questions"`
	Metrics       StateMetrics   `json:"metrics"`
	Meta          StateMeta      `json:"meta"`
}

// Step is a single item in the execution plan.
//
// The §8.23 structured fields (Objective, BlockingDependencies, Scope,
// Deliverables, FunctionContracts, FailureHandling, ExitCriteria) are
// optional: when present they enable the long-form roadmap / plan / readme
// generators to emit per-phase blocks; when absent the generators fall back
// to the minimal Title / Description form.
type Step struct {
	Title       string `json:"title"`
	Description string `json:"description"`

	Objective            string   `json:"objective,omitempty"`
	BlockingDependencies []string `json:"blocking_dependencies,omitempty"`
	Scope                string   `json:"scope,omitempty"`
	Deliverables         []string `json:"deliverables,omitempty"`
	FunctionContracts    []string `json:"function_contracts,omitempty"`
	FailureHandling      string   `json:"failure_handling,omitempty"`
	ExitCriteria         []string `json:"exit_criteria,omitempty"`
}

// UnmarshalJSON normalises LLM output that uses "name" or "phase_name"
// instead of "title", and "summary" instead of "description". The §8.23
// structured fields are decoded via the same alias struct so they survive
// the custom unmarshaller.
func (s *Step) UnmarshalJSON(data []byte) error {
	type stepAlias struct {
		Title       string `json:"title"`
		Name        string `json:"name"`       // LLM alias
		PhaseName   string `json:"phase_name"` // LLM alias
		Description string `json:"description"`
		Summary     string `json:"summary"` // LLM alias for description

		Objective            string   `json:"objective"`
		BlockingDependencies []string `json:"blocking_dependencies"`
		Scope                string   `json:"scope"`
		Deliverables         []string `json:"deliverables"`
		FunctionContracts    []string `json:"function_contracts"`
		FailureHandling      string   `json:"failure_handling"`
		ExitCriteria         []string `json:"exit_criteria"`
	}
	var a stepAlias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	s.Title = a.Title
	if s.Title == "" {
		switch {
		case a.Name != "":
			s.Title = a.Name
		case a.PhaseName != "":
			s.Title = a.PhaseName
		}
	}
	s.Description = a.Description
	if s.Description == "" && a.Summary != "" {
		s.Description = a.Summary
	}
	s.Objective = a.Objective
	s.BlockingDependencies = a.BlockingDependencies
	s.Scope = a.Scope
	s.Deliverables = a.Deliverables
	s.FunctionContracts = a.FunctionContracts
	s.FailureHandling = a.FailureHandling
	s.ExitCriteria = a.ExitCriteria
	return nil
}

// Risk describes a potential project risk surfaced by an agent.
type Risk struct {
	Text     string `json:"text"`
	Severity string `json:"severity"` // "critical" | "high" | "medium" | "low"
	Resolved bool   `json:"resolved"`
}

// StateMetrics holds session-level quality metrics updated by the convergence
// engine. The §8.23 optional fields (TestCoverageTarget, LatencyBudgetMs) are
// emitted only when set; they let the generators render concrete numbers in
// the architecture / readme documents.
type StateMetrics struct {
	Confidence         float64 `json:"confidence"`
	TestCoverageTarget float64 `json:"test_coverage_target,omitempty"`
	LatencyBudgetMs    int     `json:"latency_budget_ms,omitempty"`
}

// StateMeta holds iteration bookkeeping and the ordered agent roster.
type StateMeta struct {
	Iteration int         `json:"iteration"`
	Agents    []AgentMeta `json:"agents"`
}

// AgentMeta is the observability record for one agent in the pipeline.
// Skills stores skill names only (not prompt fragments).
// This list is populated at session creation and must have ≥ 2 entries.
type AgentMeta struct {
	AgentID  string   `json:"agent_id"`
	Name     string   `json:"name"`
	Role     string   `json:"role"`
	Provider string   `json:"provider"`
	Model    string   `json:"model"`
	Skills   []string `json:"skills"`
}

// UnmarshalJSON implements json.Unmarshaler for CanonicalState.
// It accepts both typed struct formats and plain-string shorthands for the
// `risks` and `execution_plan` fields, which LLMs frequently emit instead of
// full objects. This prevents hard unmarshal failures from LLM output variability.
//
// Coercion rules:
//   - string risk   → Risk{Text: s, Severity: "medium", Resolved: false}
//   - string step   → Step{Title: s, Description: ""}
func (cs *CanonicalState) UnmarshalJSON(b []byte) error {
	// rawState mirrors CanonicalState but holds the polymorphic fields as
	// json.RawMessage so we can inspect and coerce them ourselves.
	type rawState struct {
		Idea          map[string]any  `json:"idea"`
		Architecture  map[string]any  `json:"architecture"`
		ExecutionPlan json.RawMessage `json:"execution_plan"`
		Risks         json.RawMessage `json:"risks"`
		Assumptions   json.RawMessage `json:"assumptions"`
		OpenQuestions json.RawMessage `json:"open_questions"`
		Metrics       StateMetrics    `json:"metrics"`
		Meta          StateMeta       `json:"meta"`
	}

	var raw rawState
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	cs.Idea = raw.Idea
	cs.Architecture = raw.Architecture
	cs.Metrics = raw.Metrics
	cs.Meta = raw.Meta

	var coerceErr error
	cs.Assumptions, coerceErr = coerceStringSlice(raw.Assumptions)
	if coerceErr != nil {
		return fmt.Errorf("assumptions: %w", coerceErr)
	}
	cs.OpenQuestions, coerceErr = coerceStringSlice(raw.OpenQuestions)
	if coerceErr != nil {
		return fmt.Errorf("open_questions: %w", coerceErr)
	}

	// Coerce each risk from either a Risk object or a plain string.
	var coerceRiskErr error
	cs.Risks, coerceRiskErr = coerceRiskSlice(raw.Risks)
	if coerceRiskErr != nil {
		return fmt.Errorf("risks: %w", coerceRiskErr)
	}

	// Coerce each execution_plan item from either a Step object or a plain string.
	var coerceStepErr error
	cs.ExecutionPlan, coerceStepErr = coerceStepSlice(raw.ExecutionPlan)
	if coerceStepErr != nil {
		return fmt.Errorf("execution_plan: %w", coerceStepErr)
	}

	return nil
}

// coerceStringSlice converts a raw JSON value to []string, tolerating the
// various shapes LLMs emit instead of a clean JSON array of strings.
//
// Coercion rules (tried in order):
//  1. null / empty         → []string{}
//  2. []string             → used as-is
//  3. []any                → each element formatted with %v
//  4. map[string]any       → values formatted with %v; keys sorted for determinism
//  5. bare string          → single-element slice
//  6. anything else        → []string{} (never errors; unknown shape is silent-empty)
func coerceStringSlice(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []string{}, nil
	}

	// Preferred: clean JSON array of strings.
	var ss []string
	if err := json.Unmarshal(raw, &ss); err == nil {
		return ss, nil
	}

	// Fallback A: JSON array with mixed types.
	var aa []any
	if err := json.Unmarshal(raw, &aa); err == nil {
		out := make([]string, 0, len(aa))
		for _, v := range aa {
			out = append(out, humanizeValue(v))
		}
		return out, nil
	}

	// Fallback B: JSON object — extract values, keys sorted for determinism.
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err == nil {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]string, 0, len(m))
		for _, k := range keys {
			out = append(out, humanizeValue(m[k]))
		}
		return out, nil
	}

	// Fallback C: bare JSON string.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []string{s}, nil
	}

	// Unknown shape — return empty without propagating an error so the rest of
	// the CanonicalState can still be used.
	return []string{}, nil
}

// humanizeValue renders a JSON-decoded value as a human-readable string for
// display in `assumptions` and `open_questions` lists. LLMs frequently emit
// objects (e.g. `{id, impact, question, resolution, status}`) where the
// canonical schema expects plain strings; rather than show raw Go map syntax
// (`map[id:oq1 impact:high ...]`), we extract the primary text field and
// decorate it with secondary metadata.
//
// Recognised primary fields (in priority order): question, text, statement,
// title, name, description. Recognised metadata fields: impact, severity,
// status, resolution, answer.
func humanizeValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case map[string]any:
		return humanizeMap(t)
	case []any:
		parts := make([]string, 0, len(t))
		for _, e := range t {
			parts = append(parts, humanizeValue(e))
		}
		return strings.Join(parts, "; ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func humanizeMap(m map[string]any) string {
	primaryKeys := []string{"question", "text", "statement", "title", "name", "description"}
	var primary string
	usedKey := ""
	for _, k := range primaryKeys {
		if val, ok := m[k]; ok {
			if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
				primary = strings.TrimSpace(s)
				usedKey = k
				break
			}
		}
	}

	// Collect well-known metadata fields in display order.
	metaOrder := []string{"resolution", "answer", "impact", "severity", "status"}
	var metas []string
	for _, k := range metaOrder {
		if k == usedKey {
			continue
		}
		if val, ok := m[k]; ok {
			if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
				metas = append(metas, k+": "+strings.TrimSpace(s))
			}
		}
	}

	if primary != "" {
		if len(metas) == 0 {
			return primary
		}
		return primary + " (" + strings.Join(metas, " · ") + ")"
	}

	// No recognised primary field — fall back to sorted key:value pairs,
	// skipping noisy IDs.
	keys := make([]string, 0, len(m))
	for k := range m {
		if k == "id" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+": "+fmt.Sprintf("%v", m[k]))
	}
	return strings.Join(parts, " · ")
}

// coerceStepSlice converts a raw JSON value to []Step, tolerating the various
// shapes LLMs emit instead of a clean JSON array of Step objects.
//
// Coercion rules (tried in order):
//  1. null / empty            → []Step{}
//  2. []Step (array)          → used as-is
//  3. []any (array of mixed)  → each element coerced to Step
//  4. map[string]any (object) → values (sorted by key) coerced to Steps
//  5. bare string             → single-element slice with Title=s
func coerceStepSlice(raw json.RawMessage) ([]Step, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []Step{}, nil
	}

	// Preferred: clean JSON array of Step objects.
	var steps []Step
	if err := json.Unmarshal(raw, &steps); err == nil {
		return steps, nil
	}

	// Fallback A: JSON array with mixed types — coerce each element individually.
	var aa []json.RawMessage
	if err := json.Unmarshal(raw, &aa); err == nil {
		out := make([]Step, 0, len(aa))
		for _, item := range aa {
			var step Step
			if err2 := json.Unmarshal(item, &step); err2 == nil {
				out = append(out, step)
				continue
			}
			var title string
			if err2 := json.Unmarshal(item, &title); err2 == nil {
				out = append(out, Step{Title: title})
				continue
			}
			// skip unparseable element
		}
		return out, nil
	}

	// Fallback B: JSON object — extract values sorted by key.
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err == nil {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]Step, 0, len(m))
		for _, k := range keys {
			switch v := m[k].(type) {
			case string:
				out = append(out, Step{Title: v})
			case map[string]any:
				title, _ := v["title"].(string)
				desc, _ := v["description"].(string)
				out = append(out, Step{Title: title, Description: desc})
			default:
				out = append(out, Step{Title: fmt.Sprintf("%v", v)})
			}
		}
		return out, nil
	}

	// Fallback C: bare string.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []Step{{Title: s}}, nil
	}

	return []Step{}, nil
}

// coerceRiskSlice converts a raw JSON value to []Risk, tolerating the various
// shapes LLMs emit instead of a clean JSON array of Risk objects.
//
// Coercion rules (tried in order):
//  1. null / empty            → []Risk{}
//  2. []Risk (array)          → used as-is
//  3. []any (array of mixed)  → each element coerced to Risk
//  4. map[string]any (object) → values (sorted by key) coerced to Risks
//  5. bare string             → single-element slice with Text=s, Severity="medium"
func coerceRiskSlice(raw json.RawMessage) ([]Risk, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []Risk{}, nil
	}

	// Preferred: clean JSON array of Risk objects.
	var risks []Risk
	if err := json.Unmarshal(raw, &risks); err == nil {
		return risks, nil
	}

	// Fallback A: JSON array with mixed types.
	var aa []json.RawMessage
	if err := json.Unmarshal(raw, &aa); err == nil {
		out := make([]Risk, 0, len(aa))
		for _, item := range aa {
			var risk Risk
			if err2 := json.Unmarshal(item, &risk); err2 == nil {
				out = append(out, risk)
				continue
			}
			var text string
			if err2 := json.Unmarshal(item, &text); err2 == nil {
				out = append(out, Risk{Text: text, Severity: "medium"})
				continue
			}
		}
		return out, nil
	}

	// Fallback B: JSON object — extract values sorted by key.
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err == nil {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]Risk, 0, len(m))
		for _, k := range keys {
			switch v := m[k].(type) {
			case string:
				out = append(out, Risk{Text: v, Severity: "medium"})
			case map[string]any:
				text, _ := v["text"].(string)
				severity, _ := v["severity"].(string)
				if severity == "" {
					severity = "medium"
				}
				resolved, _ := v["resolved"].(bool)
				out = append(out, Risk{Text: text, Severity: severity, Resolved: resolved})
			default:
				out = append(out, Risk{Text: fmt.Sprintf("%v", v), Severity: "medium"})
			}
		}
		return out, nil
	}

	// Fallback C: bare string.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []Risk{{Text: s, Severity: "medium"}}, nil
	}

	return []Risk{}, nil
}

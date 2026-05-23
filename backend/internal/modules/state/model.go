// Package state defines the CanonicalState type — the shared brainstorm
// document passed between agents during each iteration pipeline pass.
//
// The JSON field names in this package are canonical: downstream agents and the
// DB JSONB column rely on this exact shape. See §8.1 of docs/PLAN.md.
package state

import (
	"encoding/json"
	"fmt"
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
type Step struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Risk describes a potential project risk surfaced by an agent.
type Risk struct {
	Text     string `json:"text"`
	Severity string `json:"severity"` // "critical" | "high" | "medium" | "low"
	Resolved bool   `json:"resolved"`
}

// StateMetrics holds session-level quality metrics updated by the convergence engine.
type StateMetrics struct {
	Confidence float64 `json:"confidence"`
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
		Idea          map[string]any    `json:"idea"`
		Architecture  map[string]any    `json:"architecture"`
		ExecutionPlan []json.RawMessage `json:"execution_plan"`
		Risks         []json.RawMessage `json:"risks"`
		Assumptions   []string          `json:"assumptions"`
		OpenQuestions []string          `json:"open_questions"`
		Metrics       StateMetrics      `json:"metrics"`
		Meta          StateMeta         `json:"meta"`
	}

	var raw rawState
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	cs.Idea = raw.Idea
	cs.Architecture = raw.Architecture
	cs.Assumptions = raw.Assumptions
	cs.OpenQuestions = raw.OpenQuestions
	cs.Metrics = raw.Metrics
	cs.Meta = raw.Meta

	// Coerce each risk from either a Risk object or a plain string.
	cs.Risks = make([]Risk, 0, len(raw.Risks))
	for i, r := range raw.Risks {
		var risk Risk
		if err := json.Unmarshal(r, &risk); err != nil {
			var s string
			if err2 := json.Unmarshal(r, &s); err2 != nil {
				return fmt.Errorf("risks[%d]: expected object or string: %w", i, err)
			}
			risk = Risk{Text: s, Severity: "medium", Resolved: false}
		}
		cs.Risks = append(cs.Risks, risk)
	}

	// Coerce each execution_plan item from either a Step object or a plain string.
	cs.ExecutionPlan = make([]Step, 0, len(raw.ExecutionPlan))
	for i, s := range raw.ExecutionPlan {
		var step Step
		if err := json.Unmarshal(s, &step); err != nil {
			var title string
			if err2 := json.Unmarshal(s, &title); err2 != nil {
				return fmt.Errorf("execution_plan[%d]: expected object or string: %w", i, err)
			}
			step = Step{Title: title}
		}
		cs.ExecutionPlan = append(cs.ExecutionPlan, step)
	}

	return nil
}

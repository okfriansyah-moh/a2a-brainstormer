// Package state defines the CanonicalState type — the shared brainstorm
// document passed between agents during each iteration pipeline pass.
//
// The JSON field names in this package are canonical: downstream agents and the
// DB JSONB column rely on this exact shape. See §8.1 of docs/PLAN.md.
package state

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

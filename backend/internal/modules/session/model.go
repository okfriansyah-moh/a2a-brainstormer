// Package session defines the Session and SessionAgent domain models and the
// request/response types for the session lifecycle REST API.
//
// A Session ties together:
//   - a product idea (plain text)
//   - an ordered list of agents (SessionAgent, position-indexed)
//   - the running CanonicalState snapshot (updated after each iteration pass)
//   - lifecycle status (active → converged → approved | failed)
//
// Minimum 2 agents per session is enforced by the service layer, which returns
// HTTP 400 for requests violating this constraint.
package session

import (
	"time"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/llm"
	"a2a-brainstorm/backend/internal/shared"
)

// StatusActive is the initial session status set at creation time.
const (
	StatusActive    = "active"
	StatusRunning   = "running" // set while an iteration is in-flight
	StatusConverged = "converged"
	StatusApproved  = "approved"
	StatusFailed    = "failed"
)

// AllowedOutputDocs is the exhaustive set of valid document keys that may be
// requested for a session. Callers validate against this map (O(1) lookup).
// Task 29 will register generator implementations for each key.
var AllowedOutputDocs = map[string]bool{
	"architecture": true,
	"roadmap":      true,
	"plan":         true,
	"readme":       true,
}

// DefaultOutputDocs is the default document selection applied when
// CreateSessionRequest.OutputDocs is nil or empty.
var DefaultOutputDocs = []string{"architecture", "roadmap"}

// Session is the top-level aggregate for a brainstorm run.
// CurrentState is nil until the first iteration pipeline pass completes.
// Agents is populated on single-session GET requests; it is omitted on list responses.
// AgentCount is populated only by ListSessions (via a subquery COUNT); it is
// zero on single-session GET responses (use len(Agents) there instead).
type Session struct {
	ID            string                `json:"id"`
	Idea          string                `json:"idea"`
	Status        string                `json:"status"`
	MaxIterations int                   `json:"max_iterations"`
	OutputDocs    []string              `json:"output_docs"`
	CurrentState  *state.CanonicalState `json:"current_state,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
	Agents        []SessionAgent        `json:"agents,omitempty"`
	AgentCount    int                   `json:"-"` // list-only, not serialised
}

// SessionAgent represents one agent binding within a session.
//
// Position is 0-indexed pipeline order: the iteration engine passes state
// through agents in ascending Position order.
//
// SkillOverrides distinguishes three states:
//   - nil  → use the agent's default attached skills (agent_skills table)
//   - &[]  → disable all skills for this session (empty prompt injection)
//   - &[id1, id2] → use exactly these skill IDs for this session
type SessionAgent struct {
	SessionID      string         `json:"session_id"`
	AgentID        string         `json:"agent_id"`
	Position       int            `json:"position"`
	Role           string         `json:"role"`
	LLMOverride    *llm.LLMConfig `json:"llm_override,omitempty"`
	SkillOverrides *[]string      `json:"skill_overrides"` // nil | [] | [...]
}

// CreateSessionRequest is the validated input body for POST /sessions.
//
// AgentIDs must contain ≥ 2 entries (service rejects with 400 otherwise).
// MaxIterations defaults to 10 when omitted or zero.
//
// RoleOverrides: optional map of agentID → role. When absent, DefaultRoles
// distribution is used. Each role value must pass agent.ValidRole.
//
// LLMOverrides: optional per-agent LLM config override (merged at dispatch time
// according to the tiered resolver: session > agent > global).
//
// SkillOverrides: optional per-agent skill list. Omitted key = use agent
// defaults. Explicit empty slice = disable all. Non-empty = use those IDs.
//
// OutputDocs: optional list of document keys to generate at finalize time.
// Valid keys: architecture, roadmap, plan, readme.
// When nil or empty, defaults to ["architecture","roadmap"].
type CreateSessionRequest struct {
	Idea           string                    `json:"idea"`
	AgentIDs       []string                  `json:"agent_ids"`
	MaxIterations  int                       `json:"max_iterations,omitempty"`
	OutputDocs     []string                  `json:"output_docs,omitempty"`
	RoleOverrides  map[string]string         `json:"role_overrides,omitempty"`
	LLMOverrides   map[string]*llm.LLMConfig `json:"llm_overrides,omitempty"`
	SkillOverrides map[string]*[]string      `json:"skill_overrides,omitempty"`
}

// UpdateOutputDocsRequest is the body for PATCH /sessions/{id}/output-docs.
type UpdateOutputDocsRequest struct {
	OutputDocs []string `json:"output_docs"`
}

// FinalizeInput is the optional body for POST /sessions/{id}/finalize.
// When OutputDocs is non-nil, it overrides the session's stored document
// selection for this finalize call only (persisted before generation).
type FinalizeInput struct {
	OutputDocs []string `json:"output_docs,omitempty"`
}

// SessionListItem is the summary representation of a Session used in list
// responses. Agents are not loaded; Idea is truncated to 120 characters by
// the service layer. Confidence and CurrentIteration are extracted from the
// current_state JSONB field.
type SessionListItem struct {
	ID               string    `json:"id"`
	Idea             string    `json:"idea"` // ≤ 120 chars
	Status           string    `json:"status"`
	MaxIterations    int       `json:"max_iterations"`
	CurrentIteration int       `json:"current_iteration"` // from current_state.meta.iteration
	Confidence       float64   `json:"confidence"`        // from current_state.metrics.confidence
	AgentCount       int       `json:"agent_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ListSessionsResponse is the envelope returned by GET /sessions.
// Total matches len(Sessions) and is included for pagination readiness.
type ListSessionsResponse struct {
	Sessions []SessionListItem `json:"sessions"`
	Total    int               `json:"total"`
}

// GeneratedDocument is an alias for the shared document type, re-exported
// so that callers do not need to import the shared package directly when
// working with FinalizeResponse.
type GeneratedDocument = shared.GeneratedDocument

// FinalizeResponse is the response body for POST /sessions/{id}/finalize.
// Documents is a map keyed by output doc key ("architecture", "roadmap",
// "plan", "readme") to the generated artifact. Only the keys that were
// requested for the session are present.
type FinalizeResponse struct {
	SessionID string                              `json:"session_id"`
	Documents map[string]shared.GeneratedDocument `json:"documents"`
	Status    string                              `json:"status"`
}

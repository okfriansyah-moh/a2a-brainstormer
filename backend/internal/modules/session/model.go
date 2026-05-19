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
)

// StatusActive is the initial session status set at creation time.
const (
	StatusActive    = "active"
	StatusConverged = "converged"
	StatusApproved  = "approved"
	StatusFailed    = "failed"
)

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
type CreateSessionRequest struct {
	Idea           string                    `json:"idea"`
	AgentIDs       []string                  `json:"agent_ids"`
	MaxIterations  int                       `json:"max_iterations,omitempty"`
	RoleOverrides  map[string]string         `json:"role_overrides,omitempty"`
	LLMOverrides   map[string]*llm.LLMConfig `json:"llm_overrides,omitempty"`
	SkillOverrides map[string]*[]string      `json:"skill_overrides,omitempty"`
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

// FinalizeResponse is the response body for POST /sessions/{id}/finalize.
// ArchitectureMarkdown and RoadmapMarkdown contain the rendered artifact
// content so the frontend can offer inline preview and download without a
// separate file-fetch round-trip.
type FinalizeResponse struct {
	SessionID            string `json:"session_id"`
	ArchitectureMarkdown string `json:"architecture_markdown"`
	RoadmapMarkdown      string `json:"roadmap_markdown"`
	Status               string `json:"status"`
}

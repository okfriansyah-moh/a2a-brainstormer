// Package agent defines the Agent and Skill domain models, the Role type,
// and the repository layer for the agents, skills, and agent_skills tables.
//
// LLMConfig is imported from backend/internal/platform/llm — it is not
// duplicated here (see §8.2 of docs/PLAN.md).
package agent

import (
	"time"

	"a2a-brainstorm/backend/internal/platform/llm"
)

// Agent represents a registered brainstorm agent sourced from the agents table.
// The Skills field is populated only on GET requests (not on list queries).
type Agent struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	DefaultRole  Role           `json:"default_role"`
	SystemPrompt string         `json:"system_prompt,omitempty"`
	LLMConfig    *llm.LLMConfig `json:"llm_config,omitempty"`
	Endpoint     string         `json:"endpoint"`
	CreatedAt    time.Time      `json:"created_at"`
	Skills       []Skill        `json:"skills"`
}

// Skill is a prompt-level behaviour fragment injected into an agent's assembled
// system prompt at dispatch time.  The agent binary receives only the assembled
// SystemPrompt string — it has no knowledge of skill names, IDs, or the
// agent_skills table.
type Skill struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Prompt      string    `json:"prompt"`
	CreatedAt   time.Time `json:"created_at"`
}

// Role is the assigned behaviour profile for an agent within a session pipeline.
// Roles are fixed at session creation and must not change between iterations.
// See §8.13 of docs/PLAN.md for the full catalogue and distribution rules.
type Role string

// Role constants — the allowed values for Agent.DefaultRole and the
// session_agents.role column.  ValidRole enforces this allowlist.
const (
	RoleBuilder        Role = "build"
	RoleReviewer       Role = "review"
	RoleRefiner        Role = "refine"
	RoleDevilsAdvocate Role = "devils_advocate"
)

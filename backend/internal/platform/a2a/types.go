// Package a2a provides the backend-side A2A client factory, AgentCard resolver,
// and the BrainstormPayload wire format used for all backend↔agent communication.
// All inter-agent messages are packed as DataPart inside a2a.SendMessageRequest.
package a2a

import "a2a-brainstorm/backend/internal/platform/llm"

// BrainstormPayload is the single wire format for backend→agent A2A messages.
// It is packed as a DataPart inside a2a.NewMessage and unpacked by the agent
// executor. The agent binary operates solely on the assembled SystemPrompt and
// State — it has no knowledge of skill names, DB records, or credential refs.
//
// JSON field names are non-negotiable — downstream agent executors depend on
// exactly this shape as defined in docs/PLAN.md §8.3.
type BrainstormPayload struct {
	// Role is the agent's behavioural role for this dispatch.
	// Allowed values: "build" | "review" | "refine" | "devils_advocate"
	Role string `json:"role"`

	// SystemPrompt is the fully assembled prompt string:
	//   agent.system_prompt + "\n\n" + skill_1.prompt + "\n\n" + skill_2.prompt
	// The agent binary receives only this string; skill names are invisible to it.
	SystemPrompt string `json:"system_prompt"`

	// LLMConfig is the resolved tiered LLM configuration for this dispatch.
	// CredentialRef stores the env var name only — never the raw API key.
	LLMConfig llm.LLMConfig `json:"llm_config"`

	// State is the current CanonicalState passed to the agent as context.
	// The agent must return an updated state via a DataPart artifact.
	State any `json:"state"`
}

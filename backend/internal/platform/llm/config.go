package llm

// LLMConfig carries the provider identifier, model name, and credential
// reference for a single LLM call target.
//
// It is stored in the database as JSONB and passed through the A2A
// BrainstormPayload. See §8.2 and §8.12 of docs/PLAN.md for resolution rules.
//
// Security invariant: CredentialRef stores ONLY the env var NAME
// (e.g. "COPILOT_API_KEY"). The actual key value is resolved at runtime via
// config.GetLLMAPIKey(CredentialRef) — it is never stored here or in the DB.
type LLMConfig struct {
	Provider      string `json:"provider"`       // "copilot" | "claude"
	Model         string `json:"model"`          // e.g. "gpt-4o", "claude-opus-4"
	CredentialRef string `json:"credential_ref"` // env var name — never the raw key value
}

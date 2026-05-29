// Package llm defines the LLMProvider abstraction used by all modules.
// No module may call a Copilot or Claude SDK directly — all LLM calls must
// go through the LLMProvider interface defined here.
package llm

import "context"

// LLMProvider is the single call boundary for all LLM interactions in the
// backend and agent binaries. All business logic that needs an LLM calls this
// interface; concrete SDK usage is confined to the implementation files
// (copilot.go, claude.go, etc.).
type LLMProvider interface {
	Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)
}

// LLMRequest is the provider-agnostic input for a generation call.
type LLMRequest struct {
	SystemPrompt string
	UserMessage  string
	// Temperature controls output randomness.
	// Keep ≤ 0.2 for deterministic pipeline results (recommended: 0.15).
	// Values above 0.5 are not used for production pipeline calls.
	Temperature float64
	// ResponseFormat selects the provider's output mode:
	//   ""            – free-form text (default; recommended for Markdown / prose)
	//   "text"        – explicit free-form text (alias for "")
	//   "json_object" – ask the provider to constrain output to a JSON object
	// Providers that do not support a particular value silently ignore it.
	ResponseFormat string
}

// LLMResponse is the provider-agnostic output from a generation call.
type LLMResponse struct {
	Content      string
	FinishReason string
	TokensUsed   int
}

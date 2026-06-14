// Package llm — streaming extension for LLMProvider.
package llm

import "context"

// TokenChunk is a single token emitted during a streaming LLM generation.
type TokenChunk struct {
	Text         string
	Done         bool
	Err          error
	FinishReason string
}

// StreamingLLMProvider extends LLMProvider with a streaming generation method.
// When the underlying LLM endpoint does not support SSE the implementation
// falls back to a single-chunk stream so callers do not need a special path.
type StreamingLLMProvider interface {
	LLMProvider
	GenerateStream(ctx context.Context, req LLMRequest) (<-chan TokenChunk, error)
}

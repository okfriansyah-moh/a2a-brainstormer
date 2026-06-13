// Package llm — streaming LLM provider interface.
//
// StreamingLLMProvider extends LLMProvider with a GenerateStream method.
// Callers should type-assert to this interface and fall back to Generate
// when the underlying provider does not support streaming.
package llm

import "context"

// TokenChunk is a single piece of streamed text from an LLM response.
// Done marks the terminal chunk; after it the channel is closed.
// When Err is non-nil the chunk is terminal and Done is also true.
type TokenChunk struct {
	Text string
	Done bool
	Err  error
}

// StreamingLLMProvider extends LLMProvider with a streaming generate call.
// Implementations that cannot stream must not implement this interface;
// callers should type-assert and fall back to Generate when absent.
type StreamingLLMProvider interface {
	LLMProvider

	// GenerateStream starts an LLM call and returns a channel of token chunks.
	// The channel is closed after the terminal Done chunk is delivered.
	// The caller must drain the channel or cancel ctx to release resources.
	GenerateStream(ctx context.Context, req LLMRequest) (<-chan TokenChunk, error)
}

// Package llm — tiered prompt types for provider-native prompt caching.
package llm

// CachePolicy marks whether a prompt block may be cached by the provider.
type CachePolicy int

const (
	// CacheEphemeral marks a block for provider ephemeral prompt caching.
	CacheEphemeral CachePolicy = iota
	// CacheNone marks a block that must never be cached.
	CacheNone
)

// PromptBlock is one message fragment with an optional cache policy.
type PromptBlock struct {
	Role        string // "system" | "user"
	Content     string
	CachePolicy CachePolicy
}

// TieredPrompt is the cache-aware prompt assembly passed to providers.
// When nil on LLMRequest, providers use SystemPrompt and UserMessage only.
type TieredPrompt struct {
	// Blocks is the tiered single-turn layout (Option A).
	Blocks []PromptBlock
	// Messages is the multi-turn thread layout (Option B).
	Messages []PromptBlock
	SessionID string
	AgentID   string
	Provider  string
	Model     string
}

// FlattenLegacy returns a classic two-field LLMRequest from tiered blocks.
func (t *TieredPrompt) FlattenLegacy() LLMRequest {
	if t == nil {
		return LLMRequest{}
	}
	if len(t.Messages) > 0 {
		return flattenMessages(t.Messages)
	}
	return flattenBlocks(t.Blocks)
}

func flattenBlocks(blocks []PromptBlock) LLMRequest {
	var sys, user stringsBuilder
	for _, b := range blocks {
		switch b.Role {
		case "system":
			sys.write(b.Content)
		case "user":
			user.write(b.Content)
		}
	}
	return LLMRequest{
		SystemPrompt: sys.String(),
		UserMessage:  user.String(),
	}
}

func flattenMessages(messages []PromptBlock) LLMRequest {
	// Providers that only support system+user flatten thread messages into user.
	var sys stringsBuilder
	var user stringsBuilder
	for _, m := range messages {
		switch m.Role {
		case "system":
			sys.write(m.Content)
		default:
			user.write(m.Content)
		}
	}
	return LLMRequest{
		SystemPrompt: sys.String(),
		UserMessage:  user.String(),
	}
}

type stringsBuilder struct {
	b []byte
}

func (s *stringsBuilder) write(text string) {
	if text == "" {
		return
	}
	if len(s.b) > 0 {
		s.b = append(s.b, '\n')
	}
	s.b = append(s.b, text...)
}

func (s *stringsBuilder) String() string {
	return string(s.b)
}

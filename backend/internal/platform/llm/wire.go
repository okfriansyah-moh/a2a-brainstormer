package llm

// ResolvePromptFields returns system and user strings for providers that only
// support a two-message layout.
func ResolvePromptFields(req LLMRequest) (system, user string) {
	if req.Tiered != nil {
		flat := req.Tiered.FlattenLegacy()
		return flat.SystemPrompt, flat.UserMessage
	}
	return req.SystemPrompt, req.UserMessage
}

// WireMessages returns provider message blocks preserving cache policies when tiered.
func WireMessages(req LLMRequest) []PromptBlock {
	if req.Tiered == nil {
		out := make([]PromptBlock, 0, 2)
		if req.SystemPrompt != "" {
			out = append(out, PromptBlock{Role: "system", Content: req.SystemPrompt, CachePolicy: CacheEphemeral})
		}
		if req.UserMessage != "" {
			out = append(out, PromptBlock{Role: "user", Content: req.UserMessage, CachePolicy: CacheNone})
		}
		return out
	}
	if len(req.Tiered.Messages) > 0 {
		return req.Tiered.Messages
	}
	return req.Tiered.Blocks
}

// PromptCacheKey returns a stable cache key for OpenAI-compatible providers.
func PromptCacheKey(req LLMRequest) string {
	if req.Tiered == nil {
		return ""
	}
	if req.Tiered.SessionID == "" || req.Tiered.AgentID == "" {
		return ""
	}
	return req.Tiered.SessionID + ":" + req.Tiered.AgentID
}

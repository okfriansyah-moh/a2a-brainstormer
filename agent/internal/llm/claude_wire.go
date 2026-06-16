package llm

import "encoding/json"

type claudeCacheControl struct {
	Type string `json:"type"`
}

type claudeTextBlock struct {
	Type         string              `json:"type"`
	Text         string              `json:"text"`
	CacheControl *claudeCacheControl `json:"cache_control,omitempty"`
}

type claudeWireMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type claudeWireRequest struct {
	Model       string              `json:"model"`
	MaxTokens   int                 `json:"max_tokens"`
	System      any                 `json:"system,omitempty"`
	Messages    []claudeWireMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
	Stream      bool                `json:"stream"`
}

func buildClaudeWireRequest(model string, maxTokens int, req LLMRequest, stream bool) ([]byte, error) {
	blocks := WireMessages(req)
	var systemBlocks []claudeTextBlock
	var messages []claudeWireMessage

	for _, b := range blocks {
		block := claudeTextBlock{Type: "text", Text: b.Content}
		if b.CachePolicy == CacheEphemeral {
			block.CacheControl = &claudeCacheControl{Type: "ephemeral"}
		}
		switch b.Role {
		case "system":
			systemBlocks = append(systemBlocks, block)
		default:
			messages = append(messages, claudeWireMessage{
				Role:    b.Role,
				Content: []claudeTextBlock{block},
			})
		}
	}

	if len(systemBlocks) == 0 {
		sys, user := ResolvePromptFields(req)
		if sys != "" {
			systemBlocks = []claudeTextBlock{{Type: "text", Text: sys, CacheControl: &claudeCacheControl{Type: "ephemeral"}}}
		}
		if user != "" {
			messages = append(messages, claudeWireMessage{Role: "user", Content: user})
		}
	}

	var system any
	if len(systemBlocks) == 1 && systemBlocks[0].CacheControl == nil {
		system = systemBlocks[0].Text
	} else if len(systemBlocks) > 0 {
		system = systemBlocks
	}

	return json.Marshal(claudeWireRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		System:      system,
		Messages:    messages,
		Temperature: req.Temperature,
		Stream:      stream,
	})
}

type claudeUsageExtended struct {
	InputTokens      int `json:"input_tokens"`
	OutputTokens     int `json:"output_tokens"`
	CacheReadTokens  int `json:"cache_read_input_tokens"`
	CacheCreateTokens int `json:"cache_creation_input_tokens"`
}

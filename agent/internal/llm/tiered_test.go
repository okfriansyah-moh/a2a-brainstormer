package llm

import (
	"testing"
)

func TestWireMessages_LegacyRequest(t *testing.T) {
	req := LLMRequest{SystemPrompt: "sys", UserMessage: "user"}
	blocks := WireMessages(req)
	if len(blocks) != 2 {
		t.Fatalf("want 2 blocks, got %d", len(blocks))
	}
}

func TestFlattenLegacy_TieredBlocks(t *testing.T) {
	tp := &TieredPrompt{
		Blocks: []PromptBlock{
			{Role: "system", Content: "A"},
			{Role: "user", Content: "B"},
		},
	}
	flat := tp.FlattenLegacy()
	if flat.SystemPrompt != "A" || flat.UserMessage != "B" {
		t.Fatalf("flat = %#v", flat)
	}
}

func TestPromptCacheKey_RequiresIDs(t *testing.T) {
	req := LLMRequest{Tiered: &TieredPrompt{SessionID: "s", AgentID: "a"}}
	if PromptCacheKey(req) != "s:a" {
		t.Fatalf("key = %q", PromptCacheKey(req))
	}
}

func TestBuildClaudeWireRequest_WithCache(t *testing.T) {
	req := LLMRequest{
		Temperature: 0.15,
		Tiered: &TieredPrompt{
			Blocks: []PromptBlock{
				{Role: "system", Content: "static", CachePolicy: CacheEphemeral},
				{Role: "user", Content: "dynamic", CachePolicy: CacheNone},
			},
		},
	}
	body, err := buildClaudeWireRequest("claude-test", 1024, req, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("empty wire body")
	}
}

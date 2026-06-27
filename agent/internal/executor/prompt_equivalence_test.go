package executor

import (
	"strings"
	"testing"

	"a2a-brainstorm/agent/internal/llm"
)

func TestAssertSemanticEquivalent_LegacyVsTiered(t *testing.T) {
	payload := BrainstormPayload{
		Role:         "build",
		SystemPrompt: "You are an architect.",
		OutputDocs:   []string{"architecture", "plan"},
		UserFeedback: "Focus on API design.",
		State: map[string]any{
			"idea":    map[string]any{"text": "agentic commerce"},
			"metrics": map[string]any{"confidence": 0.5},
		},
	}

	legacy := BuildLegacyLLMRequest(payload)
	tiered := BuildTieredBlocks(payload)
	if err := AssertSemanticEquivalent(legacy, tiered); err != nil {
		t.Fatalf("semantic equivalence failed: %v", err)
	}
}

func TestAssertSemanticEquivalent_AllRoles(t *testing.T) {
	roles := []string{"build", "review", "refine", "devils_advocate", "unknown"}
	for _, role := range roles {
		payload := BrainstormPayload{
			Role:         role,
			SystemPrompt: "base",
			State:        map[string]any{"idea": map[string]any{"text": "x"}},
		}
		if err := AssertSemanticEquivalent(BuildLegacyLLMRequest(payload), BuildTieredBlocks(payload)); err != nil {
			t.Fatalf("role %q: %v", role, err)
		}
	}
}

func TestNormalizeStateJSON_StableKeys(t *testing.T) {
	raw := `{"metrics":{"confidence":0.5},"idea":{"text":"x"}}`
	want := `{"idea":{"text":"x"},"metrics":{"confidence":0.5}}`
	got := NormalizeStateJSON(raw)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildLegacyPreservesStateJSON(t *testing.T) {
	payload := BrainstormPayload{
		Role:         "review",
		SystemPrompt: "reviewer",
		State:        map[string]any{"idea": map[string]any{"text": "seed"}},
	}
	legacy := BuildLegacyLLMRequest(payload)
	parts := LegacyPromptParts(legacy)
	if parts.StateJSON != `{"idea":{"text":"seed"}}` {
		t.Fatalf("state = %q", parts.StateJSON)
	}
}

func TestCanonicalRequestHash_UsesSHA256Hex(t *testing.T) {
	req := llm.LLMRequest{
		SystemPrompt: "sys",
		UserMessage:  "msg",
		Temperature:  0.15,
	}
	h := CanonicalRequestHash(req)
	if len(h) != 64 {
		t.Fatalf("expected 64-char sha256 hex, got %d", len(h))
	}
	for _, c := range h {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Fatalf("non-hex char in hash: %q", c)
		}
	}
}

func TestCanonicalRequestHash_DistinguishesConcatenationAliases(t *testing.T) {
	req1 := llm.LLMRequest{Tiered: &llm.TieredPrompt{Blocks: []llm.PromptBlock{{Role: "ab", Content: "c"}}}}
	req2 := llm.LLMRequest{Tiered: &llm.TieredPrompt{Blocks: []llm.PromptBlock{{Role: "a", Content: "bc"}}}}

	h1 := CanonicalRequestHash(req1)
	h2 := CanonicalRequestHash(req2)
	if h1 == h2 {
		t.Fatalf("expected distinct hashes for distinct tiered prompts, got same hash %q", h1)
	}
}

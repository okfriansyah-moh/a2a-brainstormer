package executor

import (
	"os"
	"testing"

	"a2a-brainstorm/agent/internal/config"
)

func TestResponseCache_ExactHashHit(t *testing.T) {
	t.Setenv("PROMPT_CACHE_ENABLED", "true")
	cache := NewResponseCache(4)
	req := BuildLegacyLLMRequest(BrainstormPayload{
		Role:         "build",
		SystemPrompt: "architect",
		State:        map[string]any{"idea": map[string]any{"text": "x"}},
	})

	hash := CanonicalRequestHash(req)
	cache.Put(hash, `{"metrics":{"confidence":0.9}}`, "stop")

	got, ok := cache.Get(hash)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.content == "" {
		t.Fatal("expected cached content")
	}
}

func TestResponseCache_GetRefreshesRecency(t *testing.T) {
	cache := NewResponseCache(2)
	cache.Put("a", "A", "stop")
	cache.Put("b", "B", "stop")

	if _, ok := cache.Get("a"); !ok {
		t.Fatal("expected cache hit for a")
	}

	cache.Put("c", "C", "stop")
	if _, ok := cache.Get("a"); !ok {
		t.Fatal("expected a to remain after recency refresh")
	}
	if _, ok := cache.Get("b"); ok {
		t.Fatal("expected b to be evicted as least recently used")
	}
}

func TestResponseCache_PutExistingRefreshesRecency(t *testing.T) {
	cache := NewResponseCache(2)
	cache.Put("a", "A", "stop")
	cache.Put("b", "B", "stop")

	cache.Put("a", "A2", "stop")
	cache.Put("c", "C", "stop")

	if got, ok := cache.Get("a"); !ok || got.content != "A2" {
		t.Fatalf("expected refreshed entry for a, got ok=%v content=%q", ok, got.content)
	}
	if _, ok := cache.Get("b"); ok {
		t.Fatal("expected b to be evicted after a refresh")
	}
}

func TestAssemblePrompt_LegacyMode(t *testing.T) {
	t.Setenv("PROMPT_CACHE_ENABLED", "true")
	t.Setenv("PROMPT_CACHE_MODE", "legacy")

	payload := BrainstormPayload{Role: "build", SystemPrompt: "base", State: map[string]any{}}
	got := AssemblePrompt(payload, nil)
	if got.Mode != PromptCacheLegacy {
		t.Fatalf("mode = %q", got.Mode)
	}
	if got.Legacy.Tiered != nil {
		t.Fatal("legacy mode should not set tiered")
	}
}

func TestAssemblePrompt_ThreadMode(t *testing.T) {
	t.Setenv("PROMPT_CACHE_ENABLED", "true")
	t.Setenv("PROMPT_CACHE_MODE", "thread")

	defaultThreadStore.Reset()
	payload := BrainstormPayload{
		SessionID: "s1", AgentID: "a1",
		Role: "build", SystemPrompt: "base",
		State: map[string]any{"idea": map[string]any{"text": "x"}},
	}
	got := AssemblePrompt(payload, defaultThreadStore)
	if got.Mode != PromptCacheThread {
		t.Fatalf("mode = %q", got.Mode)
	}
	if got.Tiered == nil || len(got.Tiered.Messages) == 0 {
		t.Fatal("expected thread messages")
	}
}

func TestMain(m *testing.M) {
	// Ensure deterministic defaults for package tests.
	_ = os.Setenv("PROMPT_CACHE_MODE", "legacy")
	_ = config.GetPromptCacheMode()
	os.Exit(m.Run())
}

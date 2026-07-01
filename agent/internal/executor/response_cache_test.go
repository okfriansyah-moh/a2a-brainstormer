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

func TestResponseCache_RejectsInvalidJSONOnLookup(t *testing.T) {
	t.Setenv("PROMPT_CACHE_ENABLED", "true")
	defaultResponseCache.Reset()

	req := BuildLegacyLLMRequest(BrainstormPayload{
		Role:         "build",
		SystemPrompt: "architect",
		State:        map[string]any{"idea": map[string]any{"text": "x"}},
	})
	hash := CanonicalRequestHash(req)
	defaultResponseCache.Put(hash, `{"architecture":{"layers":[{"name":"API"`, "length")

	if _, ok := lookupRetryCache(req); ok {
		t.Fatal("expected cache miss for invalid JSON")
	}
	if _, ok := defaultResponseCache.Get(hash); ok {
		t.Fatal("expected invalid JSON entry to be evicted")
	}
}

func TestStoreRetryCache_SkipsInvalidJSON(t *testing.T) {
	t.Setenv("PROMPT_CACHE_ENABLED", "true")
	defaultResponseCache.Reset()

	req := BuildLegacyLLMRequest(BrainstormPayload{Role: "build", State: map[string]any{}})
	storeRetryCache(req, generatedContent{text: `{"broken":`, finishReason: "length"})

	hash := CanonicalRequestHash(req)
	if _, ok := defaultResponseCache.Get(hash); ok {
		t.Fatal("expected invalid JSON response not to be cached")
	}
}

func TestStoreRetryCache_StoresValidJSON(t *testing.T) {
	t.Setenv("PROMPT_CACHE_ENABLED", "true")
	defaultResponseCache.Reset()

	req := BuildLegacyLLMRequest(BrainstormPayload{Role: "build", State: map[string]any{}})
	storeRetryCache(req, generatedContent{text: `{"metrics":{"confidence":0.5}}`, finishReason: "stop"})

	got, ok := lookupRetryCache(req)
	if !ok {
		t.Fatal("expected cache hit for valid JSON")
	}
	if got.text == "" {
		t.Fatal("expected cached content")
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

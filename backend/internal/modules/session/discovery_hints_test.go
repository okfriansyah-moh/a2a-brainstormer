package session

import (
	"context"
	"encoding/json"
	"testing"

	"a2a-brainstorm/backend/internal/platform/llm"
)

type stubLLM struct {
	content string
	err     error
}

func (s stubLLM) Generate(_ context.Context, _ llm.LLMRequest) (llm.LLMResponse, error) {
	if s.err != nil {
		return llm.LLMResponse{}, s.err
	}
	return llm.LLMResponse{Content: s.content}, nil
}

func TestMergeHints_FillsEmptyTiers(t *testing.T) {
	got := mergeHints(DiscoveryHintsResponse{Q2: []string{"Custom MVP"}})
	if len(got.Q2) != 1 || got.Q2[0] != "Custom MVP" {
		t.Fatalf("q2 = %v", got.Q2)
	}
	if len(got.Q3) == 0 || len(got.Q4) == 0 {
		t.Fatalf("expected static backfill, got %+v", got)
	}
}

func TestDiscoveryHintsService_StaticFallbackOnLLMError(t *testing.T) {
	svc := NewDiscoveryHintsService(stubLLM{err: context.Canceled}, nil)
	resp := svc.Generate(context.Background(), stringsIdea())
	if resp.Q2 == nil || len(resp.Q2) == 0 {
		t.Fatal("expected static q2 hints")
	}
}

func TestDiscoveryHintsService_CachesLLMResult(t *testing.T) {
	payload, _ := json.Marshal(DiscoveryHintsResponse{
		Q2: []string{"A"},
		Q3: []string{"B"},
		Q4: []string{"C"},
	})
	provider := stubLLM{content: string(payload)}
	svc := NewDiscoveryHintsService(provider, nil)
	svc.cache = newHintsLRU(8)

	idea := stringsIdea()
	_ = svc.Generate(context.Background(), idea)
	second := svc.Generate(context.Background(), idea)
	if second.Q2[0] != "A" {
		t.Fatalf("unexpected cached response: %+v", second)
	}
}

func stringsIdea() string {
	return "Build a deterministic multi-agent design workspace for engineers"
}

func TestHintsLRU_EvictsOldest(t *testing.T) {
	c := newHintsLRU(2)
	c.put("a", DiscoveryHintsResponse{Q2: []string{"1"}})
	c.put("b", DiscoveryHintsResponse{Q2: []string{"2"}})
	c.put("c", DiscoveryHintsResponse{Q2: []string{"3"}})
	if _, ok := c.get("a"); ok {
		t.Fatal("expected oldest key evicted")
	}
	if _, ok := c.get("c"); !ok {
		t.Fatal("expected newest key present")
	}
}

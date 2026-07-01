package executor

import (
	"context"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"

	"a2a-brainstorm/agent/internal/llm"
)

type continuationLLM struct {
	calls int
}

func (m *continuationLLM) Generate(_ context.Context, _ llm.LLMRequest) (llm.LLMResponse, error) {
	idx := m.calls
	m.calls++
	content := `{"architecture":{"layers":[{"name":"API"`
	reason := "length"
	if idx > 0 {
		content = `,"responsibility":"Handles HTTP requests for the service"}]},"metrics":{"confidence":0.8}}`
		reason = "stop"
	}
	return llm.LLMResponse{Content: content, FinishReason: reason}, nil
}

func TestGenerateStateContent_ContinuesOnLength(t *testing.T) {
	t.Setenv("PROMPT_CACHE_ENABLED", "false")
	provider := &continuationLLM{}
	exec := New(nil, provider, nil)
	execCtx := &a2asrv.ExecutorContext{TaskID: "task-1"}

	content, err := exec.generateStateContent(
		context.Background(),
		execCtx,
		provider,
		llm.LLMRequest{UserMessage: "return json delta"},
		func(a2a.Event, error) bool { return true },
		BrainstormPayload{Role: "build"},
	)
	if err != nil {
		t.Fatalf("generateStateContent() error = %v", err)
	}
	if provider.calls < 2 {
		t.Fatalf("expected continuation call, got %d calls", provider.calls)
	}
	got, err := extractJSON(content)
	if err != nil {
		t.Fatalf("extractJSON() error = %v", err)
	}
	state, ok := got.(map[string]any)
	if !ok || state["architecture"] == nil {
		t.Fatalf("expected architecture in JSON, got %#v", got)
	}
}

type singleShotLLM struct {
	calls int
}

func (m *singleShotLLM) Generate(_ context.Context, _ llm.LLMRequest) (llm.LLMResponse, error) {
	m.calls++
	return llm.LLMResponse{
		Content:      `{"metrics":{"confidence":0.7}}`,
		FinishReason: "stop",
	}, nil
}

func TestGenerateStateContent_BypassesInvalidCachedJSON(t *testing.T) {
	t.Setenv("PROMPT_CACHE_ENABLED", "true")
	defaultResponseCache.Reset()

	provider := &singleShotLLM{}
	exec := New(nil, provider, nil)
	execCtx := &a2asrv.ExecutorContext{TaskID: "task-cache"}
	req := llm.LLMRequest{UserMessage: "return json delta"}
	payload := BrainstormPayload{Role: "build", State: map[string]any{}}

	hash := CanonicalRequestHash(req)
	defaultResponseCache.Put(hash, `{"architecture":{"layers":[{"name":"API"`, "length")

	content, err := exec.generateStateContent(
		context.Background(),
		execCtx,
		provider,
		req,
		func(a2a.Event, error) bool { return true },
		payload,
	)
	if err != nil {
		t.Fatalf("generateStateContent() error = %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("expected fresh LLM call after invalid cache, got %d calls", provider.calls)
	}
	if _, err := extractJSON(content); err != nil {
		t.Fatalf("extractJSON() error = %v", err)
	}
}

func TestGenerateStateContent_SkipsCacheWhenFeedbackPresent(t *testing.T) {
	t.Setenv("PROMPT_CACHE_ENABLED", "true")
	defaultResponseCache.Reset()

	provider := &singleShotLLM{}
	exec := New(nil, provider, nil)
	execCtx := &a2asrv.ExecutorContext{TaskID: "task-feedback"}
	req := llm.LLMRequest{UserMessage: "return json delta"}
	payload := BrainstormPayload{
		Role:         "build",
		State:        map[string]any{},
		UserFeedback: "Add mobile offline support",
	}

	hash := CanonicalRequestHash(req)
	defaultResponseCache.Put(hash, `{"metrics":{"confidence":0.9}}`, "stop")

	_, err := exec.generateStateContent(
		context.Background(),
		execCtx,
		provider,
		req,
		func(a2a.Event, error) bool { return true },
		payload,
	)
	if err != nil {
		t.Fatalf("generateStateContent() error = %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("expected cache bypass on feedback pass, got %d calls", provider.calls)
	}
}

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
	provider := &continuationLLM{}
	exec := New(nil, provider, nil)
	execCtx := &a2asrv.ExecutorContext{TaskID: "task-1"}

	content, err := exec.generateStateContent(
		context.Background(),
		execCtx,
		provider,
		llm.LLMRequest{UserMessage: "return json delta"},
		func(a2a.Event, error) bool { return true },
		"build",
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

package executor

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"

	"a2a-brainstorm/agent/internal/llm"
)

// ── mock LLM provider ─────────────────────────────────────────────────────────

type mockLLMProvider struct {
	response string
	err      error
}

func (m *mockLLMProvider) Generate(_ context.Context, _ llm.LLMRequest) (llm.LLMResponse, error) {
	return llm.LLMResponse{Content: m.response}, m.err
}

// capturingLLMProvider records the most-recent LLMRequest so tests can inspect it.
type capturingLLMProvider struct {
	response string
	captured llm.LLMRequest
}

func (c *capturingLLMProvider) Generate(_ context.Context, req llm.LLMRequest) (llm.LLMResponse, error) {
	c.captured = req
	return llm.LLMResponse{Content: c.response}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func makeExecCtx(payload BrainstormPayload) *a2asrv.ExecutorContext {
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewDataPart(payload))
	return &a2asrv.ExecutorContext{
		TaskID:  a2a.NewTaskID(),
		Message: msg,
	}
}

func collectEvents(exec *BrainstormExecutor, execCtx *a2asrv.ExecutorContext) ([]a2a.Event, error) {
	var events []a2a.Event
	var lastErr error
	for ev, err := range exec.Execute(context.Background(), execCtx) {
		if err != nil {
			lastErr = err
		}
		if ev != nil {
			events = append(events, ev)
		}
	}
	return events, lastErr
}

func lastStatusState(events []a2a.Event) (a2a.TaskState, bool) {
	for i := len(events) - 1; i >= 0; i-- {
		if su, ok := events[i].(*a2a.TaskStatusUpdateEvent); ok {
			return su.Status.State, true
		}
	}
	return "", false
}

func hasStatusState(events []a2a.Event, want a2a.TaskState) bool {
	for _, ev := range events {
		if su, ok := ev.(*a2a.TaskStatusUpdateEvent); ok && su.Status.State == want {
			return true
		}
	}
	return false
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestExecute_SuccessSequence(t *testing.T) {
	updatedState := map[string]any{
		"idea":    map[string]any{"text": "updated"},
		"metrics": map[string]any{"confidence": 0.8},
	}
	respJSON, err := json.Marshal(updatedState)
	if err != nil {
		t.Fatal(err)
	}

	exec := New(&mockLLMProvider{response: string(respJSON)}, nil)
	payload := BrainstormPayload{
		Role:         "build",
		SystemPrompt: "You are a brainstorm agent.",
		State:        map[string]any{"idea": map[string]any{"text": "original"}},
	}

	events, err := collectEvents(exec, makeExecCtx(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) < 3 {
		t.Fatalf("want >= 3 events, got %d", len(events))
	}

	// Must include a Working status update.
	if !hasStatusState(events, a2a.TaskStateWorking) {
		t.Error("expected TaskStateWorking event")
	}

	// Must include an artifact event.
	hasArtifact := false
	for _, ev := range events {
		if _, ok := ev.(*a2a.TaskArtifactUpdateEvent); ok {
			hasArtifact = true
			break
		}
	}
	if !hasArtifact {
		t.Error("expected a TaskArtifactUpdateEvent")
	}

	// Last status must be Completed.
	state, ok := lastStatusState(events)
	if !ok {
		t.Fatal("no status update event found")
	}
	if state != a2a.TaskStateCompleted {
		t.Errorf("last status state = %v, want %v", state, a2a.TaskStateCompleted)
	}
}

func TestExecute_NewTask_EmitsSubmittedTask(t *testing.T) {
	respJSON, _ := json.Marshal(map[string]any{})
	exec := New(&mockLLMProvider{response: string(respJSON)}, nil)
	payload := BrainstormPayload{Role: "review", SystemPrompt: "Review."}

	execCtx := makeExecCtx(payload)
	// StoredTask nil → new task, must emit SubmittedTask first.
	events, _ := collectEvents(exec, execCtx)

	if len(events) == 0 {
		t.Fatal("expected events, got none")
	}
	if _, ok := events[0].(*a2a.Task); !ok {
		t.Errorf("first event for new task should be *a2a.Task (SubmittedTask), got %T", events[0])
	}
}

func TestExecute_StoredTask_DoesNotEmitSubmittedTask(t *testing.T) {
	respJSON, _ := json.Marshal(map[string]any{})
	exec := New(&mockLLMProvider{response: string(respJSON)}, nil)
	payload := BrainstormPayload{Role: "review", SystemPrompt: "Review."}

	execCtx := makeExecCtx(payload)
	execCtx.StoredTask = &a2a.Task{} // simulate existing task

	events, _ := collectEvents(exec, execCtx)

	for _, ev := range events {
		if _, ok := ev.(*a2a.Task); ok {
			t.Error("should not emit SubmittedTask when StoredTask is present")
		}
	}
}

func TestExecute_LLMError_EmitsFailed(t *testing.T) {
	exec := New(&mockLLMProvider{err: errors.New("LLM unavailable")}, nil)
	payload := BrainstormPayload{
		Role:         "review",
		SystemPrompt: "Review the state.",
	}

	events, _ := collectEvents(exec, makeExecCtx(payload))

	if !hasStatusState(events, a2a.TaskStateFailed) {
		t.Error("expected TaskStateFailed event when LLM returns an error")
	}
}

func TestExecute_NonJSONLLMResponse_EmitsFailed(t *testing.T) {
	exec := New(&mockLLMProvider{response: "not valid json {"}, nil)
	payload := BrainstormPayload{
		Role:         "refine",
		SystemPrompt: "Refine the state.",
		State:        map[string]any{},
	}

	events, _ := collectEvents(exec, makeExecCtx(payload))

	if !hasStatusState(events, a2a.TaskStateFailed) {
		t.Error("expected TaskStateFailed event for non-JSON LLM response")
	}
}

func TestExecute_EmptyMessage_EmitsFailed(t *testing.T) {
	exec := New(&mockLLMProvider{}, nil)
	execCtx := &a2asrv.ExecutorContext{
		TaskID:  a2a.NewTaskID(),
		Message: a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart("no data part here")),
	}

	events, _ := collectEvents(exec, execCtx)

	if !hasStatusState(events, a2a.TaskStateFailed) {
		t.Error("expected TaskStateFailed when message contains no DataPart")
	}
}

func TestCancel_EmitsCanceled(t *testing.T) {
	exec := New(&mockLLMProvider{}, nil)
	execCtx := &a2asrv.ExecutorContext{TaskID: a2a.NewTaskID()}

	var events []a2a.Event
	for ev, _ := range exec.Cancel(context.Background(), execCtx) {
		if ev != nil {
			events = append(events, ev)
		}
	}

	if len(events) != 1 {
		t.Fatalf("want 1 event for Cancel, got %d", len(events))
	}
	su, ok := events[0].(*a2a.TaskStatusUpdateEvent)
	if !ok {
		t.Fatalf("cancel event should be *TaskStatusUpdateEvent, got %T", events[0])
	}
	if su.Status.State != a2a.TaskStateCanceled {
		t.Errorf("cancel event state = %v, want %v", su.Status.State, a2a.TaskStateCanceled)
	}
}

func TestExtractPayload_NilMessage_ReturnsError(t *testing.T) {
	_, err := extractPayload(nil)
	if err == nil {
		t.Error("expected error for nil message")
	}
}

func TestExtractPayload_NoDataPart_ReturnsError(t *testing.T) {
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart("text only"))
	_, err := extractPayload(msg)
	if err == nil {
		t.Error("expected error when message has no DataPart")
	}
}

func TestExtractPayload_ValidPayload_Succeeds(t *testing.T) {
	want := BrainstormPayload{
		Role:         "build",
		SystemPrompt: "system prompt",
	}
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewDataPart(want))
	got, err := extractPayload(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Role != want.Role {
		t.Errorf("role = %q, want %q", got.Role, want.Role)
	}
	if got.SystemPrompt != want.SystemPrompt {
		t.Errorf("system_prompt = %q, want %q", got.SystemPrompt, want.SystemPrompt)
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		input string
		n     int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
	}
	for _, c := range cases {
		got := truncate(c.input, c.n)
		if got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.input, c.n, got, c.want)
		}
	}
}

// TestExecute_UserMessageContainsJSON verifies that the user message sent to
// the LLM always contains the word "json". This satisfies the OpenAI-compatible
// API requirement that at least one message must include "json" when
// response_format=json_object is specified.
func TestExecute_UserMessageContainsJSON(t *testing.T) {
	updatedState := map[string]any{"metrics": map[string]any{"confidence": 0.9}}
	respJSON, _ := json.Marshal(updatedState)

	provider := &capturingLLMProvider{response: string(respJSON)}
	exec := New(provider, nil)
	payload := BrainstormPayload{
		Role:         "build",
		SystemPrompt: "You are an agent.", // intentionally no "json" here
		State:        map[string]any{"idea": map[string]any{"text": "seed"}},
	}

	_, _ = collectEvents(exec, makeExecCtx(payload))

	if !strings.Contains(strings.ToLower(provider.captured.UserMessage), "json") {
		t.Errorf("user message must contain the word 'json' to satisfy response_format=json_object; got: %q",
			provider.captured.UserMessage)
	}
}

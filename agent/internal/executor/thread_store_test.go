package executor

import (
	"strings"
	"testing"
)

func TestThreadStore_ReplacesStateMessage(t *testing.T) {
	ts := NewThreadStore(8)
	payload := BrainstormPayload{
		SessionID:  "sess-1",
		AgentID:    "agent-1",
		Role:       "build",
		SystemPrompt: "architect",
		State:      map[string]any{"idea": map[string]any{"text": "v1"}},
	}

	msgs1 := ts.MessagesFor(payload)
	if len(msgs1) != 3 {
		t.Fatalf("first call: want 3 messages, got %d", len(msgs1))
	}
	if !strings.Contains(msgs1[len(msgs1)-1].Content, `"text":"v1"`) {
		t.Fatal("expected state v1 in last message")
	}

	payload.State = map[string]any{"idea": map[string]any{"text": "v2"}}
	msgs2 := ts.MessagesFor(payload)
	if len(msgs2) != 3 {
		t.Fatalf("second call: want 3 messages, got %d", len(msgs2))
	}
	stateMsgs := 0
	for _, m := range msgs2 {
		if strings.HasPrefix(m.Content, currentStateLabel+":") {
			stateMsgs++
		}
	}
	if stateMsgs != 1 {
		t.Fatalf("expected exactly one CURRENT_STATE message, got %d", stateMsgs)
	}
	if !strings.Contains(msgs2[len(msgs2)-1].Content, `"text":"v2"`) {
		t.Fatal("expected state v2 in last message")
	}
}

func TestThreadStore_SeparateAgents(t *testing.T) {
	ts := NewThreadStore(8)
	p1 := BrainstormPayload{SessionID: "s", AgentID: "a1", Role: "build", SystemPrompt: "x", State: map[string]any{"v": 1}}
	p2 := BrainstormPayload{SessionID: "s", AgentID: "a2", Role: "review", SystemPrompt: "x", State: map[string]any{"v": 2}}
	ts.MessagesFor(p1)
	msgs := ts.MessagesFor(p2)
	if !strings.Contains(msgs[len(msgs)-1].Content, `"v":2`) {
		t.Fatal("agent threads should be isolated")
	}
}

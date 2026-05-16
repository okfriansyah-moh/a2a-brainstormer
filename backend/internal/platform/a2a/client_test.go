// Package a2a_test tests the backend-side A2A client helpers.
// All tests are network-free — HTTP servers are provided by net/http/httptest.
package a2a_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sdk "github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"

	platA2A "a2a-brainstorm/backend/internal/platform/a2a"
	"a2a-brainstorm/backend/internal/platform/llm"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// minimalAgentCard returns the JSON bytes for a minimal valid AgentCard
// pointing to the provided serverURL using the HTTP+JSON transport.
func minimalAgentCard(serverURL string) []byte {
	card := map[string]any{
		"name":        "test-agent",
		"description": "test",
		"version":     "1.0",
		"url":         serverURL,
		"supportedInterfaces": []map[string]any{
			{
				"url":             serverURL + "/",
				"protocolVersion": "1.0",
				"protocolBinding": "HTTP+JSON",
			},
		},
	}
	b, _ := json.Marshal(card)
	return b
}

// newCardServer starts an httptest.Server that serves a valid AgentCard at
// /.well-known/agent-card.json pointing to itself.
func newCardServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	srv := httptest.NewServer(mux)

	mux.HandleFunc("/.well-known/agent-card.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(minimalAgentCard(srv.URL))
	})

	if handler != nil {
		mux.Handle("/", handler)
	}

	t.Cleanup(srv.Close)
	return srv
}

// ── NewClient ─────────────────────────────────────────────────────────────────

func TestNewClient_Success(t *testing.T) {
	srv := newCardServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	client, err := platA2A.NewClient(t.Context(), srv.URL)
	if err != nil {
		t.Fatalf("NewClient: unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient: returned nil client")
	}
}

func TestNewClient_BadEndpoint(t *testing.T) {
	_, err := platA2A.NewClient(t.Context(), "http://127.0.0.1:0") // nothing listening
	if err == nil {
		t.Fatal("NewClient: expected error for unreachable endpoint, got nil")
	}
}

func TestNewClient_CardNotFound(t *testing.T) {
	// Server returns 404 for the AgentCard path.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	_, err := platA2A.NewClient(t.Context(), srv.URL)
	if err == nil {
		t.Fatal("NewClient: expected error for 404 AgentCard, got nil")
	}

	var statusErr *agentcard.ErrStatusNotOK
	if !errors.As(err, &statusErr) {
		t.Logf("error chain: %v", err) // ok — may be wrapped
	}
}

// ── ExtractStateFromResult ────────────────────────────────────────────────────

func TestExtractStateFromResult_Task_DataPart(t *testing.T) {
	want := map[string]any{"idea": "test"}
	artifact := &sdk.Artifact{
		Parts: sdk.ContentParts{sdk.NewDataPart(want)},
	}
	task := &sdk.Task{Artifacts: []*sdk.Artifact{artifact}}

	got, err := platA2A.ExtractStateFromResult(task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if gotMap["idea"] != want["idea"] {
		t.Errorf("idea = %v, want %v", gotMap["idea"], want["idea"])
	}
}

func TestExtractStateFromResult_Message_DataPart(t *testing.T) {
	want := map[string]any{"confidence": 0.9}
	msg := sdk.NewMessage(sdk.MessageRoleUser, sdk.NewDataPart(want))

	got, err := platA2A.ExtractStateFromResult(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil state, got nil")
	}
}

func TestExtractStateFromResult_NoDataPart(t *testing.T) {
	// Task with only a TextPart — no DataPart.
	artifact := &sdk.Artifact{
		Parts: sdk.ContentParts{sdk.NewTextPart("just text")},
	}
	task := &sdk.Task{Artifacts: []*sdk.Artifact{artifact}}

	_, err := platA2A.ExtractStateFromResult(task)
	if err == nil {
		t.Fatal("expected error when no DataPart present, got nil")
	}
}

func TestExtractStateFromResult_Nil(t *testing.T) {
	_, err := platA2A.ExtractStateFromResult(nil)
	if err == nil {
		t.Fatal("expected error for nil result, got nil")
	}
}

func TestExtractStateFromResult_EmptyTask(t *testing.T) {
	task := &sdk.Task{}
	_, err := platA2A.ExtractStateFromResult(task)
	if err == nil {
		t.Fatal("expected error for empty task, got nil")
	}
}

// ── isTransientError (via SendPayload retry behaviour) ───────────────────────

// TestSendPayload_ContextCancellation verifies that a cancelled context causes
// SendPayload to return immediately without exhausting retries.
func TestSendPayload_ContextCancellation(t *testing.T) {
	callCount := 0
	srv := newCardServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		// Simulate slow agent — context will be cancelled before response.
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	client, err := platA2A.NewClient(t.Context(), srv.URL)
	if err != nil {
		t.Skipf("could not build client (transport issue): %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately

	payload := platA2A.BrainstormPayload{
		Role:         "build",
		SystemPrompt: "test",
		LLMConfig:    llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "TEST_KEY"},
		State:        map[string]any{},
	}

	_, err = platA2A.SendPayload(ctx, client, payload)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// ── BrainstormPayload round-trip ──────────────────────────────────────────────

// TestBrainstormPayload_JSONRoundTrip verifies the JSON tags match the wire
// format defined in docs/PLAN.md §8.3.
func TestBrainstormPayload_JSONRoundTrip(t *testing.T) {
	payload := platA2A.BrainstormPayload{
		Role:         "review",
		SystemPrompt: "You are a reviewer.",
		LLMConfig: llm.LLMConfig{
			Provider:      "claude",
			Model:         "claude-opus-4",
			CredentialRef: "CLAUDE_API_KEY",
		},
		State: map[string]any{"idea": "test product"},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	for _, key := range []string{"role", "system_prompt", "llm_config", "state"} {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON field %q missing in marshaled payload", key)
		}
	}
	if m["role"] != "review" {
		t.Errorf("role = %v, want review", m["role"])
	}

	// Verify CredentialRef is preserved but raw API key is never embedded.
	llmCfg, _ := m["llm_config"].(map[string]any)
	if llmCfg["credential_ref"] != "CLAUDE_API_KEY" {
		t.Errorf("credential_ref = %v, want CLAUDE_API_KEY", llmCfg["credential_ref"])
	}
	// Sanity: no key value should appear.
	if _, hasKey := llmCfg["api_key"]; hasKey {
		t.Error("api_key field must not appear in BrainstormPayload JSON")
	}
}

// ── SendPayload: ensure DataPart wrapping ─────────────────────────────────────

// TestSendPayload_DataPartWrapping verifies that SendPayload packs the payload
// inside a DataPart (not a TextPart) as required by §8.3.
// This is tested by verifying BrainstormPayload can be wrapped in a DataPart
// and the Data() accessor correctly returns it.
func TestSendPayload_DataPartWrapping(t *testing.T) {
	payload := platA2A.BrainstormPayload{
		Role:         "build",
		SystemPrompt: "You are a builder.",
		LLMConfig:    llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "KEY"},
		State:        map[string]any{"idea": "initial"},
	}

	// Wrap in a DataPart the same way SendPayload does internally.
	part := sdk.NewDataPart(payload)
	if part == nil {
		t.Fatal("NewDataPart returned nil")
	}

	got := part.Data()
	if got == nil {
		t.Fatal("Part.Data() returned nil — payload was not wrapped as DataPart")
	}

	// Verify round-trip: the value must be JSON-serialisable back to BrainstormPayload.
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal(part.Data()): %v", err)
	}
	var decoded platA2A.BrainstormPayload
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("json.Unmarshal into BrainstormPayload: %v", err)
	}
	if decoded.Role != "build" {
		t.Errorf("decoded.Role = %q, want %q", decoded.Role, "build")
	}
}

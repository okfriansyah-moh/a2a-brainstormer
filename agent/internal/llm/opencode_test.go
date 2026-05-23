package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"a2a-brainstorm/agent/internal/llm"
)

// ── helpers ────────────────────────────────────────────────────────────────────

// mockResolver builds a resolveKey function backed by the supplied env map.
// It mirrors config.GetLLMAPIKey behaviour: returns error when value is absent or empty.
func mockResolver(env map[string]string) func(string) (string, error) {
	return func(ref string) (string, error) {
		v, ok := env[ref]
		if !ok || v == "" {
			return "", fmt.Errorf("credential env var %q is not set or empty — agent unavailable", ref)
		}
		return v, nil
	}
}

// newMockServer builds an httptest.Server that handles POST /session and
// POST /session/:id/message. The sessionHandler and messageHandler funcs
// control per-test behaviour.
func newMockServer(
	t *testing.T,
	sessionHandler func(w http.ResponseWriter, r *http.Request),
	messageHandler func(w http.ResponseWriter, r *http.Request),
) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		sessionHandler(w, r)
	})

	// Match /session/<id>/message — pattern introduced in Go 1.22+ stdlib mux.
	mux.HandleFunc("/session/", func(w http.ResponseWriter, r *http.Request) {
		// Path pattern: /session/{id}/message
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/message") {
			messageHandler(w, r)
			return
		}
		http.NotFound(w, r)
	})

	return httptest.NewServer(mux)
}

// ── Test: Generate returns correct LLMResponse.Content ────────────────────────

func TestOpenCodeProvider_Generate_ReturnsContent(t *testing.T) {
	const wantContent = "Here is the architecture proposal."

	srv := newMockServer(t,
		// POST /session → return session ID
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "sess-abc-123"})
		},
		// POST /session/:id/message → return text parts
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"info": map[string]string{"id": "msg-1", "role": "assistant"},
				"parts": []map[string]string{
					{"type": "text", "text": wantContent},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		},
	)
	defer srv.Close()

	env := map[string]string{
		"TEST_USERNAME": "opencode",
		"TEST_PASSWORD": "secret",
	}
	provider := llm.NewOpenCodeProvider(llm.OpenCodeConfig{
		BaseURL:     srv.URL,
		ProviderID:  "github",
		ModelID:     "gpt-4o",
		UsernameRef: "TEST_USERNAME",
		PasswordRef: "TEST_PASSWORD",
	}, nil, mockResolver(env))

	resp, err := provider.Generate(context.Background(), llm.LLMRequest{
		SystemPrompt: "You are a brainstorm agent.",
		UserMessage:  `{"idea":"test"}`,
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.Content != wantContent {
		t.Errorf("Content = %q; want %q", resp.Content, wantContent)
	}
}

// ── Test: multiple text parts are concatenated ─────────────────────────────────

func TestOpenCodeProvider_Generate_ConcatenatesTextParts(t *testing.T) {
	srv := newMockServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "sess-multi"})
		},
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"parts": []map[string]string{
					{"type": "text", "text": "Part A"},
					{"type": "tool_call", "text": "ignored"},
					{"type": "text", "text": "Part B"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		},
	)
	defer srv.Close()

	env := map[string]string{"U": "user", "P": "pass"}
	provider := llm.NewOpenCodeProvider(llm.OpenCodeConfig{
		BaseURL: srv.URL, ProviderID: "github", ModelID: "gpt-4o",
		UsernameRef: "U", PasswordRef: "P",
	}, nil, mockResolver(env))

	resp, err := provider.Generate(context.Background(), llm.LLMRequest{UserMessage: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	const want = "Part APart B"
	if resp.Content != want {
		t.Errorf("Content = %q; want %q", resp.Content, want)
	}
}

// ── Test: absent password env var → error, no silent fallback ─────────────────

func TestOpenCodeProvider_Generate_AbsentCredential_ReturnsError(t *testing.T) {
	srv := newMockServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "sess-x"})
		},
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)
	defer srv.Close()

	// PASSWORD env var is intentionally absent.
	env := map[string]string{"TEST_USER2": "opencode"}
	provider := llm.NewOpenCodeProvider(llm.OpenCodeConfig{
		BaseURL:     srv.URL,
		ProviderID:  "github",
		ModelID:     "gpt-4o",
		UsernameRef: "TEST_USER2",
		PasswordRef: "MISSING_PASSWORD_VAR", // not in env map
	}, nil, mockResolver(env))

	_, err := provider.Generate(context.Background(), llm.LLMRequest{UserMessage: "hi"})
	if err == nil {
		t.Fatal("expected error for missing password credential, got nil")
	}
	if !strings.Contains(err.Error(), "MISSING_PASSWORD_VAR") {
		t.Errorf("error should mention the missing ref name; got: %v", err)
	}
}

// ── Test: ensureSession is called exactly once across multiple Generate calls ──

func TestOpenCodeProvider_Generate_SessionCreatedOnce(t *testing.T) {
	var sessionCalls atomic.Int32

	srv := newMockServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			sessionCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "sess-once"})
		},
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"parts": []map[string]string{{"type": "text", "text": "ok"}},
			}
			_ = json.NewEncoder(w).Encode(resp)
		},
	)
	defer srv.Close()

	env := map[string]string{"U": "user", "P": "pass"}
	provider := llm.NewOpenCodeProvider(llm.OpenCodeConfig{
		BaseURL: srv.URL, ProviderID: "github", ModelID: "gpt-4o",
		UsernameRef: "U", PasswordRef: "P",
	}, nil, mockResolver(env))

	const calls = 5
	for i := 0; i < calls; i++ {
		if _, err := provider.Generate(context.Background(), llm.LLMRequest{UserMessage: "ping"}); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}

	if got := sessionCalls.Load(); got != 1 {
		t.Errorf("POST /session called %d times; want exactly 1", got)
	}
}

// ── Test: HTTP 401 from OpenCode server is propagated, not retried ─────────────

func TestOpenCodeProvider_Generate_HTTP401_PropagatedNotRetried(t *testing.T) {
	var sessionCalls atomic.Int32

	srv := newMockServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			sessionCalls.Add(1)
			// Simulate wrong password → 401 on session creation.
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		},
		func(w http.ResponseWriter, _ *http.Request) {
			// Should not be reached.
			w.WriteHeader(http.StatusOK)
		},
	)
	defer srv.Close()

	env := map[string]string{"U": "user", "P": "wrong"}
	provider := llm.NewOpenCodeProvider(llm.OpenCodeConfig{
		BaseURL: srv.URL, ProviderID: "github", ModelID: "gpt-4o",
		UsernameRef: "U", PasswordRef: "P",
	}, nil, mockResolver(env))

	_, err := provider.Generate(context.Background(), llm.LLMRequest{UserMessage: "hi"})
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention HTTP 401; got: %v", err)
	}

	// 401 is a 4xx — must not be silently retried.
	// session creation should be attempted only once (sync.Once).
	if got := sessionCalls.Load(); got != 1 {
		t.Errorf("POST /session called %d times after 401; want 1 (no retry)", got)
	}

	// A second Generate call must NOT attempt a new session (sync.Once persists error).
	_, err2 := provider.Generate(context.Background(), llm.LLMRequest{UserMessage: "hi again"})
	if !errors.Is(err2, err) && err2 == nil {
		t.Fatalf("second Generate call: expected same session init error, got nil")
	}
	if got := sessionCalls.Load(); got != 1 {
		t.Errorf("POST /session called again after initial failure; want still 1, got %d", got)
	}
}

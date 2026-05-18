package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/platform/llm"
)

// newCopilotTestServer creates an httptest.Server that returns a valid
// OpenAI-compatible chat completion response.
func newCopilotTestServer(t *testing.T, statusCode int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST; got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("missing Authorization header")
		}
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("Authorization header must be Bearer; got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if body != nil {
			json.NewEncoder(w).Encode(body) //nolint:errcheck
		}
	}))
}

func successBody(content string, tokens int) any {
	return map[string]any{
		"choices": []map[string]any{
			{
				"message":       map[string]any{"role": "assistant", "content": content},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{"total_tokens": tokens},
	}
}

func TestCopilotProvider_Generate_Success(t *testing.T) {
	srv := newCopilotTestServer(t, http.StatusOK, successBody(`{"architecture":{}}`, 42))
	defer srv.Close()

	t.Setenv("TEST_COPILOT_KEY", "test-api-key-value")
	cfg := llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "TEST_COPILOT_KEY"}
	p := llm.NewCopilotProvider(cfg, srv.URL, srv.Client())

	resp, err := p.Generate(context.Background(), llm.LLMRequest{
		SystemPrompt: "You are a design agent.",
		UserMessage:  `{"idea":{}}`,
		Temperature:  0.15,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason=stop; got %q", resp.FinishReason)
	}
	if resp.TokensUsed != 42 {
		t.Errorf("expected TokensUsed=42; got %d", resp.TokensUsed)
	}
}

func TestCopilotProvider_Generate_MissingCredential(t *testing.T) {
	cfg := llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "NONEXISTENT_COPILOT_KEY_XYZ"}
	p := llm.NewCopilotProvider(cfg, "http://localhost:9999", nil)

	_, err := p.Generate(context.Background(), llm.LLMRequest{
		SystemPrompt: "system",
		UserMessage:  "user",
	})
	if err == nil {
		t.Fatal("expected error for missing credential; got nil")
	}
}

func TestCopilotProvider_Generate_HTTPError401(t *testing.T) {
	srv := newCopilotTestServer(t, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
	defer srv.Close()

	t.Setenv("TEST_COPILOT_KEY_401", "bad-key")
	cfg := llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "TEST_COPILOT_KEY_401"}
	p := llm.NewCopilotProvider(cfg, srv.URL, srv.Client())

	_, err := p.Generate(context.Background(), llm.LLMRequest{
		SystemPrompt: "system",
		UserMessage:  "user",
	})
	if err == nil {
		t.Fatal("expected error for 401 response; got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention HTTP 401; got: %v", err)
	}
}

func TestCopilotProvider_Generate_HTTPError500(t *testing.T) {
	srv := newCopilotTestServer(t, http.StatusInternalServerError, map[string]any{"error": "internal"})
	defer srv.Close()

	t.Setenv("TEST_COPILOT_KEY_500", "any-key")
	cfg := llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "TEST_COPILOT_KEY_500"}
	p := llm.NewCopilotProvider(cfg, srv.URL, srv.Client())

	_, err := p.Generate(context.Background(), llm.LLMRequest{
		SystemPrompt: "system",
		UserMessage:  "user",
	})
	if err == nil {
		t.Fatal("expected error for 500 response; got nil")
	}
}

func TestCopilotProvider_Generate_EmptyChoices(t *testing.T) {
	srv := newCopilotTestServer(t, http.StatusOK, map[string]any{
		"choices": []any{},
		"usage":   map[string]any{"total_tokens": 0},
	})
	defer srv.Close()

	t.Setenv("TEST_COPILOT_KEY_EMPTY", "any-key")
	cfg := llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "TEST_COPILOT_KEY_EMPTY"}
	p := llm.NewCopilotProvider(cfg, srv.URL, srv.Client())

	_, err := p.Generate(context.Background(), llm.LLMRequest{
		SystemPrompt: "system",
		UserMessage:  "user",
	})
	if err == nil {
		t.Fatal("expected error for empty choices; got nil")
	}
}

func TestCopilotProvider_Generate_ContextCancellation(t *testing.T) {
	srv := newCopilotTestServer(t, http.StatusOK, successBody(`{}`, 1))
	defer srv.Close()

	t.Setenv("TEST_COPILOT_KEY_CTX", "any-key")
	cfg := llm.LLMConfig{Provider: "copilot", Model: "gpt-4o", CredentialRef: "TEST_COPILOT_KEY_CTX"}
	p := llm.NewCopilotProvider(cfg, srv.URL, srv.Client())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := p.Generate(ctx, llm.LLMRequest{SystemPrompt: "s", UserMessage: "u"})
	if err == nil {
		t.Fatal("expected error for cancelled context; got nil")
	}
}

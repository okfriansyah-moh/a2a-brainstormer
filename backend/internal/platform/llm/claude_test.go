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

// ── helpers ───────────────────────────────────────────────────────────────────

func claudeSuccessBody(text, stopReason string, inTok, outTok int) any {
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"stop_reason": stopReason,
		"usage": map[string]any{
			"input_tokens":  inTok,
			"output_tokens": outTok,
		},
	}
}

func newClaudeTestServer(t *testing.T, statusCode int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST; got %s", r.Method)
		}
		if r.Header.Get("x-api-key") == "" {
			t.Error("missing x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("missing or wrong anthropic-version: %q", r.Header.Get("anthropic-version"))
		}
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("Claude must NOT use Authorization header; got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if body != nil {
			json.NewEncoder(w).Encode(body) //nolint:errcheck
		}
	}))
}

// ── Generate tests ────────────────────────────────────────────────────────────

func TestClaudeProvider_Registry_Resolves(t *testing.T) {
	cfg := llm.LLMConfig{Provider: "claude", Model: "claude-opus-4-8", CredentialRef: "ANTHROPIC_TEST_KEY"}
	p, err := llm.New(cfg, func(string) (string, error) { return "sk-test", nil })
	if err != nil {
		t.Fatalf("registry failed to resolve claude: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestClaudeProvider_Generate_Headers(t *testing.T) {
	srv := newClaudeTestServer(t, http.StatusOK, claudeSuccessBody("hello claude", "end_turn", 10, 5))
	defer srv.Close()

	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)

	p, err := llm.New(
		llm.LLMConfig{Provider: "claude", Model: "claude-opus-4-8", CredentialRef: "CLAUDE_TEST_KEY"},
		func(string) (string, error) { return "test-anthropic-key", nil },
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp, err := p.Generate(context.Background(), llm.LLMRequest{
		SystemPrompt: "system",
		UserMessage:  "user",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Content != "hello claude" {
		t.Fatalf("Content = %q, want %q", resp.Content, "hello claude")
	}
}

func TestClaudeProvider_Generate_ContentExtraction(t *testing.T) {
	resp := claudeSuccessBody("extracted text", "end_turn", 20, 8)
	parsed, _ := json.Marshal(resp)
	var cr map[string]any
	json.Unmarshal(parsed, &cr) //nolint:errcheck

	content := cr["content"].([]any)
	block := content[0].(map[string]any)
	if block["type"] != "text" {
		t.Error("expected content block type=text")
	}
	if block["text"] != "extracted text" {
		t.Errorf("expected text=extracted text; got %v", block["text"])
	}
	stopReason := cr["stop_reason"].(string)
	if stopReason != "end_turn" {
		t.Errorf("expected stop_reason=end_turn; got %q", stopReason)
	}
	usage := cr["usage"].(map[string]any)
	tokensUsed := int(usage["input_tokens"].(float64)) + int(usage["output_tokens"].(float64))
	if tokensUsed != 28 {
		t.Errorf("expected 28 tokens; got %d", tokensUsed)
	}
}

// ── Streaming tests ───────────────────────────────────────────────────────────

func TestClaudeProvider_GenerateStream_Tokens(t *testing.T) {
	sseBody := strings.Join([]string{
		"event: content_block_delta",
		`data: {"delta":{"type":"text_delta","text":"Hello"}}`,
		"",
		"event: content_block_delta",
		`data: {"delta":{"type":"text_delta","text":" world"}}`,
		"",
		"event: message_stop",
		`data: {}`,
		"",
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" {
			t.Error("missing x-api-key on streaming request")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("missing anthropic-version on streaming request: %q", r.Header.Get("anthropic-version"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sseBody))
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)

	p, err := llm.New(
		llm.LLMConfig{Provider: "claude", Model: "claude-opus-4-8", CredentialRef: "CLAUDE_TEST_KEY"},
		func(string) (string, error) { return "test-anthropic-key", nil },
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	streamer, ok := p.(llm.StreamingLLMProvider)
	if !ok {
		t.Fatal("expected StreamingLLMProvider")
	}

	ch, err := streamer.GenerateStream(context.Background(), llm.LLMRequest{
		SystemPrompt: "system",
		UserMessage:  "user",
	})
	if err != nil {
		t.Fatalf("GenerateStream: %v", err)
	}

	var tokens []string
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("stream error: %v", chunk.Err)
		}
		if chunk.Text != "" {
			tokens = append(tokens, chunk.Text)
		}
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens; got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "Hello" || tokens[1] != " world" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestClaudeProvider_Generate_HTTPError(t *testing.T) {
	errBody := map[string]any{"type": "error", "error": map[string]any{"type": "authentication_error", "message": "invalid x-api-key"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errBody) //nolint:errcheck
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)

	p, err := llm.New(
		llm.LLMConfig{Provider: "claude", Model: "claude-opus-4-8", CredentialRef: "CLAUDE_TEST_KEY"},
		func(string) (string, error) { return "bad-key", nil },
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = p.Generate(context.Background(), llm.LLMRequest{UserMessage: "hi"})
	if err == nil {
		t.Fatal("expected error for HTTP 401")
	}
}

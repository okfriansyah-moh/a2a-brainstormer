// Package llm — OpenCodeProvider proxies LLM requests through a running
// OpenCode HTTP server instance.
//
// Session lifecycle:
//   - A single OpenCode chat session is created lazily on the first Generate
//     call and reused for all subsequent calls within the same process lifetime.
//   - sync.Once ensures thread-safe single initialisation.
//
// Security invariants:
//   - os.Getenv is never called here; credentials are resolved via the injected
//     resolveKey function (must be config.GetLLMAPIKey).
//   - Resolved username and password values are never logged or stored on the
//     struct; they are used inline per request and immediately discarded.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	openCodeHTTPTimeout  = 120 * time.Second // LLM calls can be slow
	openCodeMaxRespBytes = 4 << 20           // 4 MiB
	openCodeRetryWait    = 2 * time.Second
)

// OpenCodeConfig holds all configuration for the OpenCodeProvider.
// ProviderID and ModelID are typically obtained by splitting AGENT_OPENCODE_MODEL
// on the first "/" (e.g. "github/gpt-4o" → {ProviderID:"github", ModelID:"gpt-4o"}).
type OpenCodeConfig struct {
	BaseURL     string // OpenCode server base URL, e.g. "http://localhost:4096"
	ProviderID  string // LLM provider, e.g. "github", "anthropic", "openai"
	ModelID     string // Model name, e.g. "gpt-4o", "claude-opus-4-5"
	UsernameRef string // env var NAME holding the Basic-Auth username
	PasswordRef string // env var NAME holding the Basic-Auth password
}

// OpenCodeProvider implements LLMProvider using the OpenCode HTTP server API.
// It maintains a single chat session per process lifetime (lazy initialisation).
type OpenCodeProvider struct {
	cfg        OpenCodeConfig
	client     *http.Client
	resolveKey func(ref string) (string, error)

	// session state — protected by once + mu
	once       sync.Once
	sessionID  string
	sessionErr error
}

// NewOpenCodeProvider constructs an OpenCodeProvider.
//
//   - cfg: OpenCode server configuration.
//   - httpClient: reusable HTTP client; a default 120s-timeout client is used when nil.
//   - resolveKey: must be config.GetLLMAPIKey — keeps os.Getenv confined to config.go.
func NewOpenCodeProvider(
	cfg OpenCodeConfig,
	httpClient *http.Client,
	resolveKey func(string) (string, error),
) *OpenCodeProvider {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: openCodeHTTPTimeout}
	}
	return &OpenCodeProvider{
		cfg:        cfg,
		client:     httpClient,
		resolveKey: resolveKey,
	}
}

// Generate sends LLMRequest to the OpenCode server and returns the response.
// It creates the OpenCode session lazily on the first call (sync.Once).
func (p *OpenCodeProvider) Generate(ctx context.Context, req LLMRequest) (LLMResponse, error) {
	// Ensure we have a session ID before making the message call.
	p.once.Do(func() {
		p.sessionID, p.sessionErr = p.createSession(ctx)
	})
	if p.sessionErr != nil {
		return LLMResponse{}, fmt.Errorf("opencode.Generate: session init: %w", p.sessionErr)
	}

	return p.sendMessage(ctx, req)
}

// createSession calls POST {BaseURL}/session and returns the new session ID.
// Credentials are resolved at call time and never stored.
func (p *OpenCodeProvider) createSession(ctx context.Context) (string, error) {
	username, password, err := p.resolveCredentials()
	if err != nil {
		return "", err
	}

	bodyBytes, err := json.Marshal(map[string]string{"title": "brainstorm"})
	if err != nil {
		return "", fmt.Errorf("marshal session request: %w", err)
	}

	url := p.cfg.BaseURL + "/session"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("build session request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(username, password) // credentials used inline; not persisted

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("POST /session: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, openCodeMaxRespBytes))
	if err != nil {
		return "", fmt.Errorf("read session response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// 4xx: return immediately — no retry.
		// 5xx: also return; retrying at session-creation is unsafe (could create duplicate sessions).
		return "", fmt.Errorf("POST /session returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse session response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("POST /session: response missing id field")
	}
	return result.ID, nil
}

// sendMessage calls POST {BaseURL}/session/{id}/message with one retry on 5xx/timeout.
func (p *OpenCodeProvider) sendMessage(ctx context.Context, req LLMRequest) (LLMResponse, error) {
	resp, err := p.doSendMessage(ctx, req)
	if err == nil {
		return resp, nil
	}

	// Retry once after a brief pause for transient server-side errors.
	// 4xx errors are not retried (permanent failure).
	select {
	case <-ctx.Done():
		return LLMResponse{}, ctx.Err()
	case <-time.After(openCodeRetryWait):
	}

	return p.doSendMessage(ctx, req)
}

// doSendMessage executes a single POST /session/:id/message call.
func (p *OpenCodeProvider) doSendMessage(ctx context.Context, req LLMRequest) (LLMResponse, error) {
	username, password, err := p.resolveCredentials()
	if err != nil {
		return LLMResponse{}, err
	}

	type messagePart struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type modelRef struct {
		ProviderID string `json:"providerID"`
		ModelID    string `json:"modelID"`
	}
	type messageRequest struct {
		Parts  []messagePart `json:"parts"`
		Model  modelRef      `json:"model"`
		System string        `json:"system"`
	}

	body := messageRequest{
		Parts:  []messagePart{{Type: "text", Text: req.UserMessage}},
		Model:  modelRef{ProviderID: p.cfg.ProviderID, ModelID: p.cfg.ModelID},
		System: req.SystemPrompt,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("opencode.sendMessage: marshal: %w", err)
	}

	url := fmt.Sprintf("%s/session/%s/message", p.cfg.BaseURL, p.sessionID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("opencode.sendMessage: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(username, password) // credentials used inline; not persisted

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("opencode.sendMessage: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, openCodeMaxRespBytes))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("opencode.sendMessage: read response: %w", err)
	}

	// 4xx: permanent failure — return immediately, do not retry.
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return LLMResponse{}, fmt.Errorf("opencode.sendMessage: HTTP %d: %s", resp.StatusCode, respBody)
	}
	// 5xx: transient — caller (sendMessage) will retry once.
	if resp.StatusCode >= 500 {
		return LLMResponse{}, fmt.Errorf("opencode.sendMessage: HTTP %d: %s", resp.StatusCode, respBody)
	}

	// Parse the message response.
	type responsePart struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type messageResponse struct {
		Parts []responsePart `json:"parts"`
	}

	var result messageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return LLMResponse{}, fmt.Errorf("opencode.sendMessage: parse response: %w", err)
	}

	// Concatenate all text parts into Content (ignore non-text parts).
	var content string
	for _, part := range result.Parts {
		if part.Type == "text" {
			content += part.Text
		}
	}

	return LLMResponse{
		Content:      content,
		FinishReason: "stop",
	}, nil
}

// resolveCredentials resolves username and password from their env var refs.
// Returns an error if either is absent (no silent fallback).
// The resolved values must never be logged or persisted.
func (p *OpenCodeProvider) resolveCredentials() (username, password string, err error) {
	username, err = p.resolveKey(p.cfg.UsernameRef)
	if err != nil {
		return "", "", fmt.Errorf("opencode: resolve username (%s): %w", p.cfg.UsernameRef, err)
	}
	password, err = p.resolveKey(p.cfg.PasswordRef)
	if err != nil {
		return "", "", fmt.Errorf("opencode: resolve password (%s): %w", p.cfg.PasswordRef, err)
	}
	return username, password, nil
}

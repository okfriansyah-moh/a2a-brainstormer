// Package llm — OpenCodeProvider proxies LLM requests through a running
// OpenCode HTTP server instance. Mirrors the agent-binary implementation so
// that the backend's markdown generator can use the same OpenCode path as the
// brainstorm agents — no direct GitHub Copilot API key required.
//
// Session lifecycle:
//   - A fresh OpenCode chat session is created for every Generate call.
//     This ensures concurrent callers (e.g. parallel document generators)
//     each get an independent session and do not serialise on the server side.
//     Single-turn callers (like the markdown finalizer) have no need for
//     cross-call session history, so the overhead is one extra HTTP round trip.
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
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// openCodeHTTPTimeout is intentionally generous: the OpenCode server
	// proxies to GitHub Copilot / Claude / etc. and a single Claude Sonnet 4.6
	// turn can take 5+ minutes for long-form (markdown) output. The finalize
	// handler enforces its own outer ceiling via GetFinalizeTimeout().
	openCodeHTTPTimeout  = 600 * time.Second
	openCodeMaxRespBytes = 4 << 20 // 4 MiB
	openCodeRetryWait    = 2 * time.Second
)

// OpenCodeConfig holds all configuration for the OpenCodeProvider.
// ProviderID and ModelID are typically obtained by splitting the configured
// model string on the first "/" (e.g. "github-copilot/claude-sonnet-4.6" →
// {ProviderID:"github-copilot", ModelID:"claude-sonnet-4.6"}).
type OpenCodeConfig struct {
	BaseURL     string // OpenCode server base URL, e.g. "http://opencode:4096"
	ProviderID  string // LLM provider, e.g. "github-copilot", "anthropic"
	ModelID     string // Model name, e.g. "claude-sonnet-4.6", "gpt-4.1"
	UsernameRef string // env var NAME holding the Basic-Auth username
	PasswordRef string // env var NAME holding the Basic-Auth password
}

// OpenCodeProvider implements LLMProvider using the OpenCode HTTP server API.
// It maintains a single chat session per process lifetime (lazy initialisation).
type OpenCodeProvider struct {
	cfg        OpenCodeConfig
	client     *http.Client
	resolveKey func(ref string) (string, error)
}

// openCodePermanentError wraps failures that must not be retried (4xx responses,
// credential resolution failures, and marshal/request-build errors).
type openCodePermanentError struct{ err error }

func (e openCodePermanentError) Error() string { return e.err.Error() }
func (e openCodePermanentError) Unwrap() error { return e.err }

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
// A fresh OpenCode session is created for each call so that concurrent
// callers (e.g. parallel document generators) do not share a session and
// therefore do not serialise on the OpenCode server side.
func (p *OpenCodeProvider) Generate(ctx context.Context, req LLMRequest) (LLMResponse, error) {
	sessionID, err := p.createSession(ctx)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("opencode.Generate: session init: %w", err)
	}
	return p.sendMessage(ctx, sessionID, req)
}

// createSession calls POST {BaseURL}/session and returns the new session ID.
func (p *OpenCodeProvider) createSession(ctx context.Context) (string, error) {
	username, password, err := p.resolveCredentials()
	if err != nil {
		return "", err
	}

	bodyBytes, err := json.Marshal(map[string]string{"title": "brainstorm-markdown"})
	if err != nil {
		return "", fmt.Errorf("marshal session request: %w", err)
	}

	url := p.cfg.BaseURL + "/session"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("build session request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(username, password)

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
func (p *OpenCodeProvider) sendMessage(ctx context.Context, sessionID string, req LLMRequest) (LLMResponse, error) {
	resp, err := p.doSendMessage(ctx, sessionID, req)
	if err == nil {
		return resp, nil
	}

	// 4xx and credential errors are permanent — do not retry.
	var pe openCodePermanentError
	if errors.As(err, &pe) {
		return LLMResponse{}, err
	}
	select {
	case <-ctx.Done():
		return LLMResponse{}, ctx.Err()
	case <-time.After(openCodeRetryWait):
	}
	return p.doSendMessage(ctx, sessionID, req)
}

// doSendMessage executes a single POST /session/:id/message call.
func (p *OpenCodeProvider) doSendMessage(ctx context.Context, sessionID string, req LLMRequest) (LLMResponse, error) {
	username, password, err := p.resolveCredentials()
	if err != nil {
		return LLMResponse{}, openCodePermanentError{err}
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
		return LLMResponse{}, openCodePermanentError{fmt.Errorf("opencode.sendMessage: marshal: %w", err)}
	}

	url := fmt.Sprintf("%s/session/%s/message", p.cfg.BaseURL, sessionID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return LLMResponse{}, openCodePermanentError{fmt.Errorf("opencode.sendMessage: build request: %w", err)}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(username, password)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("opencode.sendMessage: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, openCodeMaxRespBytes))
	if err != nil {
		return LLMResponse{}, fmt.Errorf("opencode.sendMessage: read response: %w", err)
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return LLMResponse{}, openCodePermanentError{fmt.Errorf("opencode.sendMessage: HTTP %d: %s", resp.StatusCode, respBody)}
	}
	if resp.StatusCode >= 500 {
		return LLMResponse{}, fmt.Errorf("opencode.sendMessage: HTTP %d: %s", resp.StatusCode, respBody)
	}

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

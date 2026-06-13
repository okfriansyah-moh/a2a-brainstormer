// Package llm — streaming extension for OpenCodeProvider.
//
// GenerateStream attempts SSE streaming on the OpenCode message endpoint.
// If the server returns a non-SSE content type the response is read in full
// and emitted as a single-chunk stream, so callers always see the same
// channel-based interface regardless of server capability.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GenerateStream implements StreamingLLMProvider.
// A fresh OpenCode session is created per call to avoid context-window overflow.
func (p *OpenCodeProvider) GenerateStream(ctx context.Context, req LLMRequest) (<-chan TokenChunk, error) {
	sessionID, err := p.createSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("opencode.GenerateStream: session init: %w", err)
	}

	ch := make(chan TokenChunk, 64)
	go func() {
		defer close(ch)
		if err := p.streamMessage(ctx, sessionID, req, ch); err != nil {
			sendChunk(ctx, ch, TokenChunk{Err: err})
		}
	}()
	return ch, nil
}

// streamMessage sends the request with Accept: text/event-stream.
// Falls back to a full-body read when the server returns a non-SSE content type.
func (p *OpenCodeProvider) streamMessage(ctx context.Context, sessionID string, req LLMRequest, ch chan<- TokenChunk) error {
	username, password, err := p.resolveCredentials()
	if err != nil {
		return permanentError{err}
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
		return fmt.Errorf("opencode.streamMessage: marshal: %w", err)
	}

	url := fmt.Sprintf("%s/session/%s/message", p.cfg.BaseURL, sessionID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("opencode.streamMessage: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.SetBasicAuth(username, password)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("opencode.streamMessage: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return permanentError{fmt.Errorf("opencode.streamMessage: HTTP %d: %s", resp.StatusCode, snippet)}
	}

	// Non-SSE response: read in full and emit as one chunk.
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, openCodeMaxRespBytes))
		if err != nil {
			return fmt.Errorf("opencode.streamMessage: read non-SSE body: %w", err)
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
			return fmt.Errorf("opencode.streamMessage: parse non-SSE response: %w", err)
		}
		for _, part := range result.Parts {
			if part.Type == "text" && part.Text != "" {
				sendChunk(ctx, ch, TokenChunk{Text: part.Text})
			}
		}
		sendChunk(ctx, ch, TokenChunk{Done: true})
		return nil
	}

	// Parse the SSE event stream line by line.
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			sendChunk(ctx, ch, TokenChunk{Done: true})
			return nil
		}
		text, done := parseSSETextDelta(data)
		if done {
			sendChunk(ctx, ch, TokenChunk{Done: true})
			return nil
		}
		if text != "" {
			sendChunk(ctx, ch, TokenChunk{Text: text})
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("opencode.streamMessage: scan SSE: %w", err)
	}
	sendChunk(ctx, ch, TokenChunk{Done: true})
	return nil
}

// parseSSETextDelta extracts text and a done flag from a single SSE data line.
// Handles three common SSE formats:
//   - OpenCode native:   {"type":"text","text":"…"}
//   - AI SDK text-delta: {"type":"text-delta","textDelta":"…"}
//   - OpenAI compat:     {"choices":[{"delta":{"content":"…"}}]}
func parseSSETextDelta(data string) (text string, done bool) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return "", false
	}

	t, _ := raw["type"].(string)

	switch t {
	case "text":
		text, _ = raw["text"].(string)
		return text, false
	case "text-delta":
		text, _ = raw["textDelta"].(string)
		return text, false
	case "finish":
		return "", true
	}

	// OpenAI-compat choices delta
	if choices, ok := raw["choices"].([]any); ok && len(choices) > 0 {
		choice, _ := choices[0].(map[string]any)
		if delta, ok := choice["delta"].(map[string]any); ok {
			content, _ := delta["content"].(string)
			if content != "" {
				return content, false
			}
		}
		if fr, _ := choice["finish_reason"].(string); fr != "" {
			return "", true
		}
	}

	return "", false
}

// sendChunk sends chunk to ch, dropping it silently when the context is done.
func sendChunk(ctx context.Context, ch chan<- TokenChunk, chunk TokenChunk) {
	select {
	case ch <- chunk:
	case <-ctx.Done():
	}
}

// Compile-time assertion.
var _ StreamingLLMProvider = (*OpenCodeProvider)(nil)

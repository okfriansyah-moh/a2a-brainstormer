// Package llm — streaming extension for OpenCodeProvider.
//
// GenerateStream tries to stream tokens from OpenCode by requesting
// Accept: text/event-stream on the message endpoint. When the server
// returns a non-SSE Content-Type the call falls back gracefully: the full
// response is delivered as one terminal TokenChunk so callers always receive
// a consistent channel-based API regardless of server capability.
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

// Compile-time assertion: OpenCodeProvider satisfies StreamingLLMProvider.
var _ StreamingLLMProvider = (*OpenCodeProvider)(nil)

// GenerateStream implements StreamingLLMProvider.
func (p *OpenCodeProvider) GenerateStream(ctx context.Context, req LLMRequest) (<-chan TokenChunk, error) {
	sessID, err := p.createSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("opencode.GenerateStream: session init: %w", err)
	}
	ch := make(chan TokenChunk, 64)
	go func() {
		defer close(ch)
		p.streamMessage(ctx, sessID, req, ch)
	}()
	return ch, nil
}

// streamMessage performs the streaming HTTP call and forwards tokens to ch.
// On error it sends a terminal TokenChunk{Done:true,Err:...}.
func (p *OpenCodeProvider) streamMessage(ctx context.Context, sessID string, req LLMRequest, ch chan<- TokenChunk) {
	resp, err := p.doStreamRequest(ctx, sessID, req)
	if err != nil {
		sendChunk(ctx, ch, TokenChunk{Done: true, Err: err})
		return
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		// Server returned a complete JSON body — deliver as one terminal chunk.
		body, err := io.ReadAll(io.LimitReader(resp.Body, openCodeMaxRespBytes))
		if err != nil {
			sendChunk(ctx, ch, TokenChunk{Done: true, Err: fmt.Errorf("opencode.stream: read body: %w", err)})
			return
		}
		text, err := parseStreamFallbackBody(body)
		if err != nil {
			sendChunk(ctx, ch, TokenChunk{Done: true, Err: err})
			return
		}
		sendChunk(ctx, ch, TokenChunk{Text: text, Done: true})
		return
	}

	// SSE path: scan line-by-line and forward text deltas.
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			break
		}
		text, done := parseSSETextDelta(data)
		if text != "" {
			if !sendChunk(ctx, ch, TokenChunk{Text: text}) {
				return
			}
		}
		if done {
			break
		}
	}
	sendChunk(ctx, ch, TokenChunk{Done: true})
}

// doStreamRequest builds and executes the POST /session/{id}/message with
// Accept: text/event-stream. Returns the open *http.Response; caller must close.
func (p *OpenCodeProvider) doStreamRequest(ctx context.Context, sessID string, req LLMRequest) (*http.Response, error) {
	username, password, err := p.resolveCredentials()
	if err != nil {
		return nil, openCodePermanentError{err}
	}

	type msgPart struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type modelRef struct {
		ProviderID string `json:"providerID"`
		ModelID    string `json:"modelID"`
	}
	type msgReq struct {
		Parts  []msgPart `json:"parts"`
		Model  modelRef  `json:"model"`
		System string    `json:"system"`
	}
	payload := msgReq{
		Parts:  []msgPart{{Type: "text", Text: req.UserMessage}},
		Model:  modelRef{ProviderID: p.cfg.ProviderID, ModelID: p.cfg.ModelID},
		System: req.SystemPrompt,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, openCodePermanentError{fmt.Errorf("opencode.stream: marshal: %w", err)}
	}

	url := fmt.Sprintf("%s/session/%s/message", p.cfg.BaseURL, sessID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, openCodePermanentError{fmt.Errorf("opencode.stream: build request: %w", err)}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.SetBasicAuth(username, password)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("opencode.stream: http: %w", err)
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, openCodePermanentError{fmt.Errorf("opencode.stream: HTTP %d: %s", resp.StatusCode, errBody)}
	}
	if resp.StatusCode >= 500 {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("opencode.stream: HTTP %d: %s", resp.StatusCode, errBody)
	}
	return resp, nil
}

// parseSSETextDelta extracts the text fragment from a single SSE data line.
// Returns (text, done) where done signals end-of-stream.
// Handles common LLM SSE streaming formats (OpenCode native, AI SDK, OpenAI).
func parseSSETextDelta(data string) (text string, done bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return "", false
	}

	// OpenCode / AI SDK: {"type":"text","text":"..."} or {"type":"textDelta","textDelta":"..."}
	if typeRaw, ok := raw["type"]; ok {
		var evtType string
		if json.Unmarshal(typeRaw, &evtType) == nil {
			switch evtType {
			case "text":
				if r, ok := raw["text"]; ok {
					var t string
					if json.Unmarshal(r, &t) == nil {
						return t, false
					}
				}
			case "textDelta", "text-delta":
				for _, k := range []string{"textDelta", "text", "delta"} {
					if r, ok := raw[k]; ok {
						var t string
						if json.Unmarshal(r, &t) == nil && t != "" {
							return t, false
						}
					}
				}
			case "finish", "stop", "done", "message_stop":
				return "", true
			}
		}
	}

	// Simple: {"content":"..."}
	if r, ok := raw["content"]; ok {
		var t string
		if json.Unmarshal(r, &t) == nil && t != "" {
			return t, false
		}
	}

	// OpenAI-compat: {"choices":[{"delta":{"content":"..."},"finish_reason":null}]}
	if r, ok := raw["choices"]; ok {
		var choices []struct {
			Delta        struct{ Content string `json:"content"` } `json:"delta"`
			FinishReason *string                                    `json:"finish_reason"`
		}
		if json.Unmarshal(r, &choices) == nil && len(choices) > 0 {
			if choices[0].FinishReason != nil {
				return "", true
			}
			return choices[0].Delta.Content, false
		}
	}

	return "", false
}

// parseStreamFallbackBody parses a non-streaming OpenCode JSON response body
// into its text content, used when the server ignores Accept: text/event-stream.
func parseStreamFallbackBody(body []byte) (string, error) {
	var result struct {
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("opencode.stream: parse fallback body: %w", err)
	}
	var sb strings.Builder
	for _, pt := range result.Parts {
		if pt.Type == "text" {
			sb.WriteString(pt.Text)
		}
	}
	return sb.String(), nil
}

// sendChunk delivers chunk to ch, returning false if ctx is done.
func sendChunk(ctx context.Context, ch chan<- TokenChunk, chunk TokenChunk) bool {
	select {
	case ch <- chunk:
		return true
	case <-ctx.Done():
		return false
	}
}

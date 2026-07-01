// Package a2a provides the backend-side A2A client factory, AgentCard resolver,
// and helpers for packing/unpacking BrainstormPayload messages over the A2A
// protocol (github.com/a2aproject/a2a-go/v2).
//
// Usage pattern:
//
//	client, err := NewClient(ctx, agentEndpoint)
//	result, err := SendPayload(ctx, client, payload)
//	updatedState, err := ExtractStateFromResult(result)
package a2a

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"a2a-brainstorm/backend/internal/platform/config"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
)

const (
	// maxRetries is the number of times SendPayload retries on transient errors.
	maxRetries = 3
	// retryBaseDelay is the initial backoff delay before the first retry.
	retryBaseDelay = 100 * time.Millisecond
)

// cardResolver resolves AgentCards for A2A clients.
// It is kept behind an interface so tests can swap in a flaky resolver for
// retry-path verification without depending on real network I/O.
type agentCardResolver interface {
	Resolve(ctx context.Context, baseURL string, opts ...agentcard.ResolveOption) (*a2a.AgentCard, error)
}

var cardResolver agentCardResolver = agentcard.DefaultResolver

// NewClient resolves the AgentCard from {agentEndpoint}/.well-known/agent-card.json
// and constructs an a2aclient.Client using the negotiated transport.
//
// The HTTP client used for A2A calls is configured with a timeout sourced from
// config.GetAgentCallTimeout() (default 30 minutes) to accommodate long-running
// LLM inference calls that exceed the SDK's built-in 3-minute default.
//
// The caller owns the returned *Client and must not share it across goroutines
// without synchronisation.
func NewClient(ctx context.Context, agentEndpoint string) (*a2aclient.Client, error) {
	return newClientWithHTTPTimeout(ctx, agentEndpoint, config.GetAgentCallTimeout())
}

// NewStreamingClient is like NewClient but uses an HTTP client with no overall
// request timeout. Streaming A2A calls are bounded by the caller context instead.
func NewStreamingClient(ctx context.Context, agentEndpoint string) (*a2aclient.Client, error) {
	return newClientWithHTTPTimeout(ctx, agentEndpoint, 0)
}

func newClientWithHTTPTimeout(ctx context.Context, agentEndpoint string, httpTimeout time.Duration) (*a2aclient.Client, error) {
	var lastErr error
	delay := retryBaseDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			slog.Default().WarnContext(ctx, "A2A card resolution failed, retrying",
				slog.Int("attempt", attempt),
				slog.Int("max_retries", maxRetries),
				slog.String("error", lastErr.Error()),
				slog.String("next_delay", delay.String()),
			)
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled while retrying A2A client setup: %w", ctx.Err())
			case <-time.After(delay):
			}
			delay *= 2
		}

		card, err := cardResolver.Resolve(ctx, agentEndpoint)
		if err != nil {
			lastErr = err
			if !isTransientError(err) {
				return nil, fmt.Errorf("resolve agent card for %q: %w", agentEndpoint, err)
			}
			continue
		}

		// Override the SDK default 3-minute HTTP timeout. Blocking calls use the
		// configured agent-call timeout; streaming callers pass httpTimeout=0 and
		// rely on context deadlines instead.
		var httpClient *http.Client
		if httpTimeout > 0 {
			httpClient = &http.Client{Timeout: httpTimeout}
		} else {
			httpClient = &http.Client{}
		}

		client, err := a2aclient.NewFromCard(ctx, card,
			a2aclient.WithJSONRPCTransport(httpClient),
			a2aclient.WithRESTTransport(httpClient),
		)
		if err != nil {
			lastErr = err
			if !isTransientError(err) {
				return nil, fmt.Errorf("create a2a client for %q: %w", agentEndpoint, err)
			}
			continue
		}

		return client, nil
	}

	return nil, fmt.Errorf("resolve agent card for %q after %d retries: %w", agentEndpoint, maxRetries, lastErr)
}

// SendPayload packs payload as a DataPart inside an A2A message and sends it
// to the agent via the provided client.
//
// Transient errors (network timeout, a2a.ErrInternalError, a2a.ErrServerError)
// are retried up to maxRetries times with exponential backoff starting at
// retryBaseDelay. Client errors (4xx / application-level a2a errors) are
// returned immediately without retry.
func SendPayload(ctx context.Context, client *a2aclient.Client, payload BrainstormPayload) (a2a.SendMessageResult, error) {
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewDataPart(payload))
	req := &a2a.SendMessageRequest{Message: msg}

	var lastErr error
	delay := retryBaseDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			slog.Default().WarnContext(ctx, "A2A send failed, retrying",
				slog.Int("attempt", attempt),
				slog.Int("max_retries", maxRetries),
				slog.String("error", lastErr.Error()),
				slog.String("next_delay", delay.String()),
			)
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled waiting for retry: %w", ctx.Err())
			case <-time.After(delay):
			}
			delay *= 2
		}

		slog.Default().InfoContext(ctx, "sending A2A message",
			slog.Int("attempt", attempt+1),
			slog.Int("max_attempts", maxRetries+1),
		)

		result, err := client.SendMessage(ctx, req)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !isTransientError(err) {
			return nil, fmt.Errorf("send a2a message: %w", err)
		}
	}

	return nil, fmt.Errorf("send a2a message after %d retries: %w", maxRetries, lastErr)
}

// ExtractStateFromResult walks the SendMessageResult to find the agent's
// output DataPart artifact and returns its value.
//
// The agent executor emits the updated CanonicalState as a DataPart artifact on
// success. Failed tasks must not fall back to request history — the user
// message also carries a DataPart (input state) and would masquerade as a
// successful no-op merge.
func ExtractStateFromResult(result a2a.SendMessageResult) (any, error) {
	if result == nil {
		return nil, errors.New("extract state: nil SendMessageResult")
	}

	switch r := result.(type) {
	case *a2a.Task:
		if r.Status.State == a2a.TaskStateFailed {
			return nil, taskFailureError(r)
		}
		if data, ok := dataPartFromArtifacts(r.Artifacts); ok {
			return data, nil
		}
		return nil, errors.New("extract state: no agent artifact DataPart in task")

	case *a2a.Message:
		for _, part := range r.Parts {
			if part == nil {
				continue
			}
			if d := part.Data(); d != nil {
				return d, nil
			}
		}
	}

	return nil, errors.New("extract state: no DataPart found in SendMessageResult")
}

func dataPartFromArtifacts(artifacts []*a2a.Artifact) (any, bool) {
	for _, artifact := range artifacts {
		if artifact == nil {
			continue
		}
		for _, part := range artifact.Parts {
			if part == nil {
				continue
			}
			if d := part.Data(); d != nil {
				return d, true
			}
		}
	}
	return nil, false
}

func taskFailureError(task *a2a.Task) error {
	if task == nil {
		return errors.New("extract state: agent task failed")
	}
	if task.Status.Message != nil {
		for _, part := range task.Status.Message.Parts {
			if t := part.Text(); t != "" {
				return fmt.Errorf("extract state: agent task failed: %s", t)
			}
		}
	}
	return errors.New("extract state: agent task failed")
}

// isTransientError reports whether err is a transient failure that warrants a retry.
//
// Transient:
//   - Network timeout (net.Error.Timeout() == true)
//   - Temporary network error (url.Error.Temporary())
//   - A2A server-side errors: a2a.ErrInternalError, a2a.ErrServerError
//
// Non-transient (returned immediately):
//   - Application-level errors (4xx class: invalid params, not found, etc.)
//   - Explicit context cancellation (context.Canceled)
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Explicit cancellation is not transient. Transient deadline expiry
	// (for example from a slow first contact with the agent) is still worth
	// retrying when the caller context is still alive.
	if errors.Is(err, context.Canceled) {
		return false
	}

	// Network-level timeout.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// url.Error wraps network errors from the HTTP transport.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		// Temporary() is deprecated but still correct for connection-reset / EOF.
		//nolint:staticcheck
		if urlErr.Temporary() {
			return true
		}
	}

	// A2A protocol-level server errors (maps to HTTP 5xx).
	if errors.Is(err, a2a.ErrInternalError) || errors.Is(err, a2a.ErrServerError) {
		return true
	}

	return false
}

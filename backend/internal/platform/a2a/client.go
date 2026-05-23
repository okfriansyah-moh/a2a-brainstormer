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
	"net/url"
	"time"

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

// cardResolver is the default AgentCard resolver with a 30-second HTTP timeout.
// Exposed as a package-level var so tests can substitute it.
var cardResolver = agentcard.DefaultResolver

// NewClient resolves the AgentCard from {agentEndpoint}/.well-known/agent-card.json
// and constructs an a2aclient.Client using the negotiated transport.
//
// The caller owns the returned *Client and must not share it across goroutines
// without synchronisation.
func NewClient(ctx context.Context, agentEndpoint string) (*a2aclient.Client, error) {
	card, err := cardResolver.Resolve(ctx, agentEndpoint)
	if err != nil {
		return nil, fmt.Errorf("resolve agent card for %q: %w", agentEndpoint, err)
	}

	client, err := a2aclient.NewFromCard(ctx, card)
	if err != nil {
		return nil, fmt.Errorf("create a2a client for %q: %w", agentEndpoint, err)
	}

	return client, nil
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

// ExtractStateFromResult walks the SendMessageResult to find the first DataPart
// across all artifact parts and message parts, returning its value.
//
// The agent executor emits the updated CanonicalState as a DataPart artifact.
// ExtractStateFromResult unwraps it so the iteration engine can pass the result
// to state.Merge.
func ExtractStateFromResult(result a2a.SendMessageResult) (any, error) {
	if result == nil {
		return nil, errors.New("extract state: nil SendMessageResult")
	}

	switch r := result.(type) {
	case *a2a.Task:
		for _, artifact := range r.Artifacts {
			if artifact == nil {
				continue
			}
			for _, part := range artifact.Parts {
				if part == nil {
					continue
				}
				if d := part.Data(); d != nil {
					return d, nil
				}
			}
		}
		// Fall through to check history messages if no artifact DataPart found.
		for _, msg := range r.History {
			if msg == nil {
				continue
			}
			for _, part := range msg.Parts {
				if part == nil {
					continue
				}
				if d := part.Data(); d != nil {
					return d, nil
				}
			}
		}

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

// isTransientError reports whether err is a transient failure that warrants a retry.
//
// Transient:
//   - Network timeout (net.Error.Timeout() == true)
//   - Temporary network error (url.Error.Temporary())
//   - A2A server-side errors: a2a.ErrInternalError, a2a.ErrServerError
//
// Non-transient (returned immediately):
//   - Application-level errors (4xx class: invalid params, not found, etc.)
//   - Context cancellation / deadline exceeded
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Context errors are not transient — caller cancelled the operation.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
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

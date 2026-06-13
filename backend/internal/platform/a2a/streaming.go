// Package a2a — streaming variant of SendPayload.
//
// SendStreamingPayload calls the agent via the A2A streaming protocol
// (client.SendStreamingMessage). For each Working-state status update that
// carries a text message the tokenFn callback is invoked so the caller can
// relay individual tokens to the browser. The final CanonicalState is
// extracted from the first artifact-update event and returned as a synthetic
// *a2a.Message so that the existing ExtractStateFromResult helper works
// unchanged.
//
// When the agent's AgentCard does not advertise streaming capability the SDK
// transparently falls back to blocking SendMessage and yields the complete
// task as a single event — this function handles that case via the *a2a.Task
// and *a2a.Message branches.
package a2a

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
)

// SendStreamingPayload sends payload to the agent and streams events back.
//
// tokenFn is called with each text token emitted by the agent's Working-state
// status updates. Pass nil to disable token forwarding (tokens are still
// consumed but discarded).
//
// Returns an a2a.SendMessageResult containing the agent's DataPart artifact
// so that ExtractStateFromResult works identically to the blocking path.
func SendStreamingPayload(
	ctx context.Context,
	client *a2aclient.Client,
	payload BrainstormPayload,
	tokenFn func(token string),
) (a2a.SendMessageResult, error) {
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewDataPart(payload))
	req := &a2a.SendMessageRequest{Message: msg}

	var artifactData any // DataPart content from the first artifact-update event

	for evt, err := range client.SendStreamingMessage(ctx, req) {
		if err != nil {
			return nil, fmt.Errorf("stream event: %w", err)
		}
		if evt == nil {
			continue
		}

		switch e := evt.(type) {
		case *a2a.Task:
			// Non-streaming fallback: full task returned as single event.
			slog.Default().InfoContext(ctx, "a2a streaming: received full Task (non-streaming fallback)")
			return e, nil

		case *a2a.Message:
			// Final message result (non-streaming fallback path).
			slog.Default().InfoContext(ctx, "a2a streaming: received Message result")
			return e, nil

		case *a2a.TaskStatusUpdateEvent:
			if e.Status.State == a2a.TaskStateWorking && e.Status.Message != nil && tokenFn != nil {
				for _, part := range e.Status.Message.Parts {
					if t := part.Text(); t != "" {
						tokenFn(t)
					}
				}
			}

		case *a2a.TaskArtifactUpdateEvent:
			if e.Artifact != nil && artifactData == nil {
				for _, part := range e.Artifact.Parts {
					if d := part.Data(); d != nil {
						artifactData = d
						break
					}
				}
			}
		}
	}

	// Stream ended. Build a synthetic *a2a.Message containing the artifact
	// DataPart so ExtractStateFromResult finds the state in the message parts.
	if artifactData == nil {
		return nil, fmt.Errorf("stream ended with no artifact DataPart")
	}
	result := a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewDataPart(artifactData))
	return result, nil
}

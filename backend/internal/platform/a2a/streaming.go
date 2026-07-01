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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

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
			if artifactData != nil && isRecoverableStreamClose(err) {
				slog.Default().WarnContext(ctx, "a2a streaming: connection closed after artifact received",
					slog.String("error", err.Error()),
				)
				return messageFromArtifactData(artifactData), nil
			}
			return nil, fmt.Errorf("stream event: %w", err)
		}
		if evt == nil {
			continue
		}

		switch e := evt.(type) {
		case *a2a.Task:
			if result, err, done := terminalTaskResult(e, artifactData); done {
				if err != nil {
					return nil, err
				}
				if e.Status.State == a2a.TaskStateCompleted && artifactData == nil {
					slog.Default().InfoContext(ctx, "a2a streaming: task completed")
				}
				return result, nil
			}
			// In-progress task envelope while streaming — wait for artifacts/tokens.

		case *a2a.Message:
			// Final message result (non-streaming fallback path).
			slog.Default().InfoContext(ctx, "a2a streaming: received Message result")
			return e, nil

		case *a2a.TaskStatusUpdateEvent:
			if e.Status.State == a2a.TaskStateFailed {
				return nil, taskStatusFailureError(e)
			}
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
	return messageFromArtifactData(artifactData), nil
}

func messageFromArtifactData(artifactData any) *a2a.Message {
	return a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewDataPart(artifactData))
}

// isRecoverableStreamClose reports whether the A2A SSE connection dropped after
// the agent artifact was already received (e.g. agent HTTP write timeout).
func isRecoverableStreamClose(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unexpected eof") || strings.Contains(msg, "eof")
}

// terminalTaskResult reports whether a Task event ends the stream and, when it
// does, the SendMessageResult or error to return. Non-terminal tasks (e.g.
// Working right after stream open) must be ignored so the caller keeps reading
// artifact and token events.
func terminalTaskResult(task *a2a.Task, artifactData any) (a2a.SendMessageResult, error, bool) {
	if task == nil {
		return nil, nil, false
	}
	switch task.Status.State {
	case a2a.TaskStateFailed:
		return nil, taskFailureError(task), true
	case a2a.TaskStateCompleted:
		if artifactData != nil {
			return messageFromArtifactData(artifactData), nil, true
		}
		if _, ok := dataPartFromArtifacts(task.Artifacts); ok {
			return task, nil, true
		}
		return nil, fmt.Errorf("extract state: no agent artifact DataPart in task"), true
	default:
		return nil, nil, false
	}
}

func taskStatusFailureError(evt *a2a.TaskStatusUpdateEvent) error {
	if evt == nil {
		return errors.New("extract state: agent task failed")
	}
	if evt.Status.Message != nil {
		for _, part := range evt.Status.Message.Parts {
			if t := part.Text(); t != "" {
				return fmt.Errorf("extract state: agent task failed: %s", t)
			}
		}
	}
	return errors.New("extract state: agent task failed")
}

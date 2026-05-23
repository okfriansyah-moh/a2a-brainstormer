// Package executor implements BrainstormExecutor — the a2asrv.AgentExecutor
// that processes incoming BrainstormPayload messages through an LLMProvider
// and emits an updated CanonicalState as a DataPart artifact.
//
// Event sequence emitted by Execute:
//
//	SubmittedTask (if new) → StatusUpdate(Working) → ArtifactEvent(DataPart) → StatusUpdate(Completed)
//
// On LLM error or parse failure:
//
//	SubmittedTask (if new) → StatusUpdate(Working) → StatusUpdate(Failed)
//
// On Cancel:
//
//	StatusUpdate(Canceled)
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"

	"a2a-brainstorm/agent/internal/llm"
)

// BrainstormPayload mirrors the backend's BrainstormPayload wire format exactly.
// JSON field names are non-negotiable — see docs/PLAN.md §8.3.
// The agent binary does not import the backend module; this is an independent
// copy that must be kept in sync with backend/internal/platform/a2a/types.go.
type BrainstormPayload struct {
	// Role is the agent's behavioural role for this dispatch.
	// Allowed values: "build" | "review" | "refine" | "devils_advocate"
	Role string `json:"role"`

	// SystemPrompt is the fully assembled prompt:
	//   agent.system_prompt + "\n\n" + skill_1.prompt + ...
	SystemPrompt string `json:"system_prompt"`

	// LLMConfig is the resolved tiered configuration for this dispatch.
	// CredentialRef holds the env var name only — never the raw key.
	LLMConfig LLMConfig `json:"llm_config"`

	// State is the current CanonicalState passed to the agent as context.
	State any `json:"state"`
}

// LLMConfig is the per-dispatch LLM configuration embedded in BrainstormPayload.
type LLMConfig struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	CredentialRef string `json:"credential_ref"`
}

// BrainstormExecutor implements a2asrv.AgentExecutor.
// It extracts a BrainstormPayload from the incoming A2A message, calls the
// injected LLMProvider, and emits the updated CanonicalState as a DataPart.
type BrainstormExecutor struct {
	llm    llm.LLMProvider
	logger *slog.Logger
}

// New constructs a BrainstormExecutor. llmProvider must be non-nil.
func New(llmProvider llm.LLMProvider, logger *slog.Logger) *BrainstormExecutor {
	return &BrainstormExecutor{llm: llmProvider, logger: logger}
}

// Compile-time interface assertion.
var _ a2asrv.AgentExecutor = (*BrainstormExecutor)(nil)

// Execute implements a2asrv.AgentExecutor.
// It processes the incoming BrainstormPayload through the LLMProvider and
// emits the standard A2A event sequence.
func (e *BrainstormExecutor) Execute(
	ctx context.Context,
	execCtx *a2asrv.ExecutorContext,
) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		// Emit SubmittedTask only when this is a brand-new task.
		if execCtx.StoredTask == nil {
			if !yield(a2a.NewSubmittedTask(execCtx, execCtx.Message), nil) {
				return
			}
		}

		// Signal that the agent is now working.
		if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil), nil) {
			return
		}

		// Extract BrainstormPayload from the first DataPart in the message.
		payload, err := extractPayload(execCtx.Message)
		if err != nil {
			e.logError(ctx, "extract payload failed", err, slog.String("task_id", string(execCtx.TaskID)))
			errMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart(
				fmt.Sprintf("failed to extract payload: %s", err),
			))
			yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateFailed, errMsg), nil)
			return
		}

		if e.logger != nil {
			e.logger.InfoContext(ctx, "brainstorm task received",
				slog.String("task_id", string(execCtx.TaskID)),
				slog.String("role", payload.Role),
				slog.String("provider", payload.LLMConfig.Provider),
				slog.String("model", payload.LLMConfig.Model),
			)
		}

		// Serialize the current state as the user message for the LLM.
		stateJSON, err := json.Marshal(payload.State)
		if err != nil {
			stateJSON = []byte("{}")
		}

		// The OpenAI-compatible response_format:json_object requires the word
		// "json" to appear in at least one message. We embed the state in a
		// labelled instruction so the constraint is always satisfied regardless
		// of what the system prompt contains.
		userMessage := fmt.Sprintf(
			"Current brainstorm state (JSON):\n%s\n\nRespond with the complete updated JSON state.",
			string(stateJSON),
		)

		// Call the LLM through the LLMProvider interface.
		// Temperature 0.15 enforces near-deterministic output (blueprint §8.4).
		if e.logger != nil {
			e.logger.InfoContext(ctx, "calling LLM",
				slog.String("task_id", string(execCtx.TaskID)),
				slog.String("role", payload.Role),
				slog.String("model", payload.LLMConfig.Model),
				slog.String("provider", payload.LLMConfig.Provider),
			)
		}
		resp, err := e.llm.Generate(ctx, llm.LLMRequest{
			SystemPrompt: payload.SystemPrompt,
			UserMessage:  userMessage,
			Temperature:  0.15,
		})
		if err != nil {
			e.logError(ctx, "LLM generate failed", err, slog.String("role", payload.Role))
			errMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart(
				fmt.Sprintf("LLM generate failed: %s", err),
			))
			yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateFailed, errMsg), nil)
			return
		}

		if e.logger != nil {
			e.logger.InfoContext(ctx, "LLM call complete, parsing state",
				slog.String("task_id", string(execCtx.TaskID)),
				slog.String("role", payload.Role),
				slog.Int("response_bytes", len(resp.Content)),
			)
		}

		// Parse the LLM JSON response as the updated CanonicalState.
		var updatedState any
		if err := json.Unmarshal([]byte(resp.Content), &updatedState); err != nil {
			e.logError(ctx, "parse LLM response as state failed", err,
				slog.String("content_prefix", truncate(resp.Content, 200)),
			)
			errMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart(
				fmt.Sprintf("LLM returned non-JSON: %s", err),
			))
			yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateFailed, errMsg), nil)
			return
		}

		// Emit the updated state as a DataPart artifact.
		if e.logger != nil {
			e.logger.InfoContext(ctx, "state updated, emitting artifact",
				slog.String("task_id", string(execCtx.TaskID)),
				slog.String("role", payload.Role),
			)
		}
		if !yield(a2a.NewArtifactEvent(execCtx, a2a.NewDataPart(updatedState)), nil) {
			return
		}

		// Signal successful completion.
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
	}
}

// Cancel implements a2asrv.AgentExecutor.
// Emits a single TaskStateCanceled status update event.
func (e *BrainstormExecutor) Cancel(
	ctx context.Context,
	execCtx *a2asrv.ExecutorContext,
) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// extractPayload walks the message parts and unmarshals the first DataPart into
// a BrainstormPayload. Returns an error if the message is nil or contains no
// DataPart.
func extractPayload(msg *a2a.Message) (BrainstormPayload, error) {
	if msg == nil {
		return BrainstormPayload{}, fmt.Errorf("message is nil")
	}
	for _, part := range msg.Parts {
		raw := part.Data()
		if raw == nil {
			continue
		}
		// DataPart content arrives as any (typically map[string]any after the
		// JSON round-trip through the A2A wire). Re-encode to JSON then decode
		// into the typed struct.
		b, err := json.Marshal(raw)
		if err != nil {
			continue
		}
		var payload BrainstormPayload
		if err := json.Unmarshal(b, &payload); err != nil {
			continue
		}
		return payload, nil
	}
	return BrainstormPayload{}, fmt.Errorf("no DataPart found in message")
}

// logError logs err at ERROR level when a logger is present. This is a no-op
// when the executor was constructed without a logger (e.g. in unit tests).
func (e *BrainstormExecutor) logError(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	if e.logger == nil {
		return
	}
	args := make([]any, 0, len(attrs)+2)
	args = append(args, slog.Any("error", err))
	for _, a := range attrs {
		args = append(args, a)
	}
	e.logger.ErrorContext(ctx, msg, args...)
}

// truncate returns s truncated to at most n bytes, appending "..." when cut.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

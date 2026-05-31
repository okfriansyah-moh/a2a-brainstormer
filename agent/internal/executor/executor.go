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
	"regexp"
	"strings"

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
// It extracts a BrainstormPayload from the incoming A2A message, selects the
// appropriate LLMProvider based on payload.LLMConfig.Provider, and emits the
// updated CanonicalState as a DataPart artifact.
//
// Provider resolution order:
//  1. Look up payload.LLMConfig.Provider in the providers map.
//  2. If not found (or providers is nil), use fallback.
//
// This allows each agent in the DB to declare its own provider (e.g. opencode)
// while the agent binary gracefully falls back to copilot when that provider
// is unavailable at runtime.
type BrainstormExecutor struct {
	providers map[string]llm.LLMProvider // keyed by provider name, e.g. "copilot", "opencode"
	fallback  llm.LLMProvider            // used when providers is nil or key not found
	logger    *slog.Logger
}

// New constructs a BrainstormExecutor.
//
// providers maps provider names to LLMProvider implementations; may be nil.
// fallback is used when the payload's provider is not in the map; must be non-nil.
func New(providers map[string]llm.LLMProvider, fallback llm.LLMProvider, logger *slog.Logger) *BrainstormExecutor {
	return &BrainstormExecutor{providers: providers, fallback: fallback, logger: logger}
}

// resolveProvider returns the LLMProvider to use for a given provider name,
// and a bool indicating whether the resolution was strict (an exact match
// in the providers map). When name is empty the fallback is returned and
// strict=true (the caller did not request a specific provider). When name is
// non-empty but missing from the providers map, strict=false signals that the
// agent binary must fail fast rather than silently use the fallback — this
// enforces the security invariant that a session/agent configured for a
// specific provider must never be transparently downgraded to another one.
func (e *BrainstormExecutor) resolveProvider(name string) (llm.LLMProvider, bool) {
	if name == "" {
		return e.fallback, true
	}
	if e.providers != nil {
		if p, ok := e.providers[name]; ok {
			return p, true
		}
	}
	return nil, false
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

		// Construct the user message with a strict JSON-only instruction.
		// Claude may still add prose unless we are very explicit; we also apply
		// JSON extraction as a fallback in parsing (see extractJSON below).
		userMessage := fmt.Sprintf(
			"CRITICAL INSTRUCTION: You MUST respond with ONLY a valid JSON object.\n"+
				"Do NOT include any explanation, commentary, markdown, or text outside the JSON.\n"+
				"Your entire response must start with { and end with }.\n"+
				"No prose before or after. No ```json fences. Pure JSON only.\n\n"+
				"Current brainstorm state (JSON):\n%s\n\n"+
				"Return the complete updated canonical state as a single JSON object.",
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
		// Select provider from payload config. When the payload requests a
		// specific provider that is not available in this agent binary, fail
		// fast — a silent fallback would let a session configured for one
		// provider run on a different one, violating the project security
		// invariant (see AGENTS.md §Security invariants).
		activeLLM, ok := e.resolveProvider(payload.LLMConfig.Provider)
		if !ok {
			e.logError(ctx, "requested LLM provider not registered",
				fmt.Errorf("provider %q is not configured on this agent", payload.LLMConfig.Provider),
				slog.String("role", payload.Role),
			)
			errMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart(
				fmt.Sprintf("requested LLM provider %q is not configured on this agent", payload.LLMConfig.Provider),
			))
			yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateFailed, errMsg), nil)
			return
		}
		resp, err := activeLLM.Generate(ctx, llm.LLMRequest{
			SystemPrompt: payload.SystemPrompt + requiredOutputStructurePrompt,
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
		// extractJSON handles responses where Claude wraps the JSON in prose or
		// markdown code fences despite the explicit instruction.
		updatedState, err := extractJSON(resp.Content)
		if err != nil {
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
			// Extract key metrics from the parsed state map for observability.
			confidence := 0.0
			openQCount := 0
			risksCount := 0
			planSteps := 0
			if stateMap, ok := updatedState.(map[string]any); ok {
				if m, ok := stateMap["metrics"].(map[string]any); ok {
					if c, ok := m["confidence"].(float64); ok {
						confidence = c
					}
				}
				if oq, ok := stateMap["open_questions"].([]any); ok {
					openQCount = len(oq)
				}
				if r, ok := stateMap["risks"].([]any); ok {
					risksCount = len(r)
				}
				if p, ok := stateMap["execution_plan"].([]any); ok {
					planSteps = len(p)
				}
			}
			e.logger.InfoContext(ctx, "state updated, emitting artifact",
				slog.String("task_id", string(execCtx.TaskID)),
				slog.String("role", payload.Role),
				slog.Float64("confidence", confidence),
				slog.Int("execution_plan_steps", planSteps),
				slog.Int("risks_count", risksCount),
				slog.Int("open_questions_count", openQCount),
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

// jsonCodeFenceRE matches a ```json ... ``` or ``` ... ``` fenced code block.
var jsonCodeFenceRE = regexp.MustCompile(`(?s)` + "```" + `(?:json)?\s*([\s\S]*?)` + "```")

// extractJSON attempts to parse a valid JSON object from raw, which may be:
//  1. Pure JSON (the expected case).
//  2. JSON wrapped in ```json … ``` or ``` … ``` markdown fences.
//  3. A JSON object embedded anywhere in prose text.
//
// Returns an error only when no valid JSON object can be found.
func extractJSON(raw string) (any, error) {
	raw = strings.TrimSpace(raw)

	// Fast path: raw content is already valid JSON.
	var out any
	if err := json.Unmarshal([]byte(raw), &out); err == nil {
		return out, nil
	}

	// Try extracting from a ```json … ``` or ``` … ``` code fence.
	if m := jsonCodeFenceRE.FindStringSubmatch(raw); len(m) == 2 {
		candidate := strings.TrimSpace(m[1])
		if err := json.Unmarshal([]byte(candidate), &out); err == nil {
			return out, nil
		}
	}

	// Last resort: find the first '{' and the last '}' and try that substring.
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		candidate := raw[start : end+1]
		if err := json.Unmarshal([]byte(candidate), &out); err == nil {
			return out, nil
		}
	}

	return nil, fmt.Errorf("response contains no valid JSON object (first 200 bytes: %s)", truncate(raw, 200))
}

// Package iteration — event type constants and EventEmitter alias.
//
// Event types are used as the SSE `event:` field and as the `evtType` argument
// to EventEmitter.Emit. Constants are shared between the engine (producer) and
// the handler (SSE dispatcher / consumer).
//
// The EventEmitter interface lives in platform/sse so that the session module
// can also import it without creating a circular dependency. This file exposes
// a type alias so iteration-package code can reference it without qualifying
// the import.
package iteration

import "a2a-brainstorm/backend/internal/platform/sse"

// EventEmitter is the interface satisfied by *sse.Broadcaster and
// sse.NoopEmitter. It is used by the Engine to publish lifecycle events.
type EventEmitter = sse.EventEmitter

// NoopEmitter is a convenience alias for sse.NoopEmitter. It is used in tests
// and any context that does not need live SSE events. All Emit calls are
// discarded.
type NoopEmitter = sse.NoopEmitter

// Event type constants for the SSE stream (§8.22 of docs/PLAN.md).
const (
	// EventIterationStart is emitted once before the agent loop begins for a
	// given iteration pass.
	EventIterationStart = "iteration.start"

	// EventAgentStarted is emitted immediately before the A2A dispatch call
	// for each agent in the pipeline.
	EventAgentStarted = "agent.started"

	// EventAgentComplete is emitted after each agent's output has been merged
	// into the cumulative state. Includes a confidence delta.
	EventAgentComplete = "agent.complete"

	// EventAgentError is emitted when a dispatch call returns an error. The
	// engine aborts the current iteration pass on this event.
	EventAgentError = "agent.error"

	// EventIterationComplete is emitted once after all agents have completed a
	// full pass and before the convergence check.
	EventIterationComplete = "iteration.complete"

	// EventSessionFinalized is emitted by session.Service after a session is
	// successfully approved / finalized.
	EventSessionFinalized = "session.finalized"
)

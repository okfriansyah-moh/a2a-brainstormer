// Package iteration provides the HTTP handler for the iteration endpoint.
//
// Routes registered by RegisterRoutes:
//
//	POST   /sessions/{id}/iterate
//	POST   /sessions/{id}/agents/{agent_id}/preview
//	POST   /sessions/{id}/agents/{agent_id}/apply
//	DELETE /sessions/{id}/agents/{agent_id}/preview
//
// Input validation:
//   - Session IDs and agent IDs are validated as UUID format before any service call.
//
// Error mapping:
//
//	session.ErrNotFound      → 404
//	ErrPreviewNotFound       → 404
//	ErrSessionTerminal       → 409 Conflict
//	ErrIterationInFlight     → 409 Conflict
//	ErrPreviewIDMismatch     → 412 Precondition Failed
//	validation errors        → 400
//	other errors             → 500 (detail never exposed to caller)
package iteration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"

	"a2a-brainstorm/backend/internal/modules/session"
	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/config"
	"a2a-brainstorm/backend/internal/platform/sse"
)

// uuidRE matches UUID v4 format used for session IDs.
var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// iterationSvc is the subset of *Service required by the Handler.
// Using an interface enables test stubs without a live DB.
type iterationSvc interface {
	TriggerIteration(ctx context.Context, sessionID string) (IterationResult, error)
	Preview(ctx context.Context, sessionID, agentID string) (PreviewResponse, error)
	Apply(ctx context.Context, sessionID, agentID, previewID string) (state.CanonicalState, error)
	DiscardPreview(ctx context.Context, sessionID, agentID string) error
	CheckSessionExists(ctx context.Context, sessionID string) error
}

// eventSubscriber is the minimal interface the SSE handler needs from the
// Broadcaster. Using an interface allows injecting a test double.
type eventSubscriber interface {
	Subscribe(sessionID uuid.UUID, lastEventID uint64) (<-chan sse.Event, func())
}

// Handler provides the HTTP handler for the iteration endpoint.
type Handler struct {
	svc    iterationSvc
	events eventSubscriber // nil when SSE is disabled
	logger *slog.Logger
}

// NewHandler constructs an iteration Handler backed by the given Service and
// broadcaster. broadcaster may be nil — in that case the SSE endpoint returns
// 503 Service Unavailable.
func NewHandler(svc *Service, broadcaster eventSubscriber, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, events: broadcaster, logger: logger}
}

// NewHandlerWithService constructs a Handler from any iterationSvc implementation.
// Used in tests to inject a stub. SSE is disabled (no broadcaster).
func NewHandlerWithService(svc iterationSvc, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes registers the iteration HTTP routes on mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /sessions/{id}/iterate", h.handleIterate)
	mux.HandleFunc("POST /sessions/{id}/agents/{agent_id}/preview", h.handlePreview)
	mux.HandleFunc("POST /sessions/{id}/agents/{agent_id}/apply", h.handleApply)
	mux.HandleFunc("DELETE /sessions/{id}/agents/{agent_id}/preview", h.handleDiscardPreview)
	mux.HandleFunc("GET /sessions/{id}/events", h.handleEvents)
}

// handleIterate handles POST /sessions/{id}/iterate.
//
// Triggers the full iteration engine loop for the session and returns an
// IterationResult JSON envelope matching the IterateResponse shape (§8.7).
func (h *Handler) handleIterate(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if !uuidRE.MatchString(sessionID) {
		writeIterError(w, http.StatusBadRequest, "invalid session ID format")
		return
	}

	h.logger.InfoContext(r.Context(), "iteration requested",
		slog.String("session_id", sessionID),
	)

	// Detach the pipeline context from the HTTP request context. A client
	// disconnect or the server's WriteTimeout would otherwise cancel r.Context()
	// mid-pipeline, aborting the in-flight LLM calls. context.WithoutCancel
	// (Go 1.21+) copies values but ignores the parent's cancellation signal.
	// A separate deadline provides the upper-bound safety net.
	iterCtx, cancel := context.WithTimeout(
		context.WithoutCancel(r.Context()),
		config.GetIterationTimeout(),
	)
	defer cancel()

	result, err := h.svc.TriggerIteration(iterCtx, sessionID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "trigger iteration failed",
			slog.String("session_id", sessionID),
			slog.String("error", err.Error()),
		)
		switch {
		case errors.Is(err, session.ErrNotFound):
			writeIterError(w, http.StatusNotFound, "session not found")
		case errors.Is(err, ErrSessionTerminal):
			writeIterError(w, http.StatusConflict, "session is already approved")
		case errors.Is(err, ErrIterationInFlight):
			writeIterError(w, http.StatusConflict, "iteration already in progress")
		default:
			writeIterError(w, http.StatusInternalServerError, "iteration failed")
		}
		return
	}

	writeIterJSON(w, http.StatusOK, result)
}

// handlePreview handles POST /sessions/{id}/agents/{agent_id}/preview.
//
// Dispatches a single agent against the session's current state and returns
// the agent's output without merging or persisting it. Response matches §8.21.
func (h *Handler) handlePreview(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	agentID := r.PathValue("agent_id")

	if !uuidRE.MatchString(sessionID) {
		writeIterError(w, http.StatusBadRequest, "invalid session ID format")
		return
	}
	if !uuidRE.MatchString(agentID) {
		writeIterError(w, http.StatusBadRequest, "invalid agent ID format")
		return
	}

	h.logger.InfoContext(r.Context(), "preview requested",
		slog.String("session_id", sessionID),
		slog.String("agent_id", agentID),
	)

	previewCtx, cancel := context.WithTimeout(
		context.WithoutCancel(r.Context()),
		config.GetIterationTimeout(),
	)
	defer cancel()

	resp, err := h.svc.Preview(previewCtx, sessionID, agentID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "preview failed",
			slog.String("session_id", sessionID),
			slog.String("agent_id", agentID),
			slog.String("error", err.Error()),
		)
		switch {
		case errors.Is(err, session.ErrNotFound):
			writeIterError(w, http.StatusNotFound, "session not found")
		case errors.Is(err, ErrIterationInFlight):
			writeIterError(w, http.StatusConflict, "iteration already in progress")
		default:
			writeIterError(w, http.StatusInternalServerError, "preview failed")
		}
		return
	}

	writeIterJSON(w, http.StatusOK, resp)
}

// applyRequest is the optional JSON body for POST .../apply.
type applyRequest struct {
	PreviewID string `json:"preview_id"`
}

// handleApply handles POST /sessions/{id}/agents/{agent_id}/apply.
//
// Merges the stored preview output into the session's live canonical state and
// persists it. Body is optional: { "preview_id": "<uuid>" } for optimistic
// concurrency. On mismatch returns 412. On no-preview returns 404. (§8.21)
func (h *Handler) handleApply(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	agentID := r.PathValue("agent_id")

	if !uuidRE.MatchString(sessionID) {
		writeIterError(w, http.StatusBadRequest, "invalid session ID format")
		return
	}
	if !uuidRE.MatchString(agentID) {
		writeIterError(w, http.StatusBadRequest, "invalid agent ID format")
		return
	}

	var req applyRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeIterError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	h.logger.InfoContext(r.Context(), "preview apply requested",
		slog.String("session_id", sessionID),
		slog.String("agent_id", agentID),
		slog.String("preview_id", req.PreviewID),
	)

	newState, err := h.svc.Apply(r.Context(), sessionID, agentID, req.PreviewID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "apply failed",
			slog.String("session_id", sessionID),
			slog.String("agent_id", agentID),
			slog.String("error", err.Error()),
		)
		switch {
		case errors.Is(err, session.ErrNotFound):
			writeIterError(w, http.StatusNotFound, "session not found")
		case errors.Is(err, ErrPreviewNotFound):
			writeIterError(w, http.StatusNotFound, "no preview found for this agent")
		case errors.Is(err, ErrPreviewIDMismatch):
			writeIterError(w, http.StatusPreconditionFailed, "preview_id mismatch")
		case errors.Is(err, ErrIterationInFlight):
			writeIterError(w, http.StatusConflict, "iteration already in progress")
		default:
			writeIterError(w, http.StatusInternalServerError, "apply failed")
		}
		return
	}

	writeIterJSON(w, http.StatusOK, newState)
}

// handleDiscardPreview handles DELETE /sessions/{id}/agents/{agent_id}/preview.
//
// Removes any stored preview for the agent. Idempotent — always returns 204.
func (h *Handler) handleDiscardPreview(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	agentID := r.PathValue("agent_id")

	if !uuidRE.MatchString(sessionID) {
		writeIterError(w, http.StatusBadRequest, "invalid session ID format")
		return
	}
	if !uuidRE.MatchString(agentID) {
		writeIterError(w, http.StatusBadRequest, "invalid agent ID format")
		return
	}

	if err := h.svc.DiscardPreview(r.Context(), sessionID, agentID); err != nil {
		h.logger.ErrorContext(r.Context(), "discard preview failed",
			slog.String("session_id", sessionID),
			slog.String("agent_id", agentID),
			slog.String("error", err.Error()),
		)
		writeIterError(w, http.StatusInternalServerError, "discard preview failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleEvents handles GET /sessions/{id}/events.
//
// Opens a server-sent events stream for the session. The client receives
// real-time agent lifecycle events as they occur. Optionally honours the
// Last-Event-ID request header to replay missed events from the ring buffer.
//
// Error responses (before SSE headers are sent):
//   - 400 Bad Request  — invalid UUID format
//   - 404 Not Found    — session does not exist
//   - 429 Too Many Requests — subscriber limit reached (10 per session)
//   - 503 Service Unavailable — SSE broadcaster not initialised
func (h *Handler) handleEvents(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if !uuidRE.MatchString(sessionID) {
		writeIterError(w, http.StatusBadRequest, "invalid session ID format")
		return
	}

	if h.events == nil {
		writeIterError(w, http.StatusServiceUnavailable, "SSE not available")
		return
	}

	// Validate session exists before opening the stream.
	if err := h.svc.CheckSessionExists(r.Context(), sessionID); err != nil {
		if errors.Is(err, session.ErrNotFound) {
			writeIterError(w, http.StatusNotFound, "session not found")
		} else {
			writeIterError(w, http.StatusInternalServerError, "failed to verify session")
		}
		return
	}

	// Parse Last-Event-ID for replay support.
	// The frontend manually manages reconnects via EventSource (sse.ts) and
	// encodes the last received ID as a query parameter (?lastEventId=N) because
	// the native EventSource API does not allow setting custom request headers.
	// Native browser auto-reconnect sends it as the Last-Event-ID header instead.
	// Check the query param first; fall back to the standard header.
	var lastEventID uint64
	if raw := r.URL.Query().Get("lastEventId"); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, 64); err == nil {
			lastEventID = parsed
		}
	} else if raw := r.Header.Get("Last-Event-ID"); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, 64); err == nil {
			lastEventID = parsed
		}
	}

	sessUUID, err := uuid.Parse(sessionID)
	if err != nil {
		writeIterError(w, http.StatusBadRequest, "invalid session ID format")
		return
	}

	ch, unsubscribe := h.events.Subscribe(sessUUID, lastEventID)
	if ch == nil {
		writeIterError(w, http.StatusTooManyRequests, "subscriber limit reached for this session")
		return
	}
	defer unsubscribe()

	// Set SSE response headers. Must be done before the first Write.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	// Keep-alive ticker: LLM calls can take minutes with no SSE data flowing.
	// Without a heartbeat the browser (or any intermediate proxy) may close an
	// idle SSE connection, forcing a reconnect that briefly loses the agent
	// "running" status in the UI.
	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				// Channel closed by broadcaster (slow consumer drop or server shutdown).
				return
			}
			if err := writeSSEEvent(w, evt); err != nil {
				// Client disconnected.
				return
			}
			if canFlush {
				flusher.Flush()
			}
		case <-keepalive.C:
			// SSE comment line — not parsed as an event by the browser;
			// prevents idle-timeout disconnects during long LLM calls.
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return
			}
			if canFlush {
				flusher.Flush()
			}
		case <-r.Context().Done():
			// Client disconnected or server shutting down.
			return
		}
	}
}

// writeSSEEvent formats an Event as an SSE message and writes it to w.
// Returns an error if the write fails (client disconnected).
func writeSSEEvent(w http.ResponseWriter, evt sse.Event) error {
	dataBytes, err := json.Marshal(evt.Data)
	if err != nil {
		// Non-serialisable data — skip the event rather than breaking the stream.
		return nil
	}
	_, werr := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n",
		evt.ID, evt.Type, dataBytes)
	return werr
}

// ── JSON helpers ──────────────────────────────────────────────────────────────

func writeIterJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeIterError(w http.ResponseWriter, status int, msg string) {
	writeIterJSON(w, status, map[string]string{"error": msg})
}

// Package iteration provides the HTTP handler for the iteration endpoint.
//
// Routes registered by RegisterRoutes:
//
//	POST /sessions/{id}/iterate
//
// Input validation:
//   - Session IDs are validated as UUID format before any service call.
//
// Error mapping:
//
//	session.ErrNotFound      → 404
//	ErrSessionTerminal       → 409 Conflict
//	validation errors        → 400
//	other errors             → 500 (detail never exposed to caller)
package iteration

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"a2a-brainstorm/backend/internal/modules/session"
	"a2a-brainstorm/backend/internal/platform/config"
)

// uuidRE matches UUID v4 format used for session IDs.
var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// iterationSvc is the subset of *Service required by the Handler.
// Using an interface enables test stubs without a live DB.
type iterationSvc interface {
	TriggerIteration(ctx context.Context, sessionID string) (IterationResult, error)
}

// Handler provides the HTTP handler for the iteration endpoint.
type Handler struct {
	svc    iterationSvc
	logger *slog.Logger
}

// NewHandler constructs an iteration Handler backed by the given Service.
func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// NewHandlerWithService constructs a Handler from any iterationSvc implementation.
// Used in tests to inject a stub.
func NewHandlerWithService(svc iterationSvc, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes registers the iteration HTTP routes on mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /sessions/{id}/iterate", h.handleIterate)
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
		default:
			writeIterError(w, http.StatusInternalServerError, "iteration failed")
		}
		return
	}

	writeIterJSON(w, http.StatusOK, result)
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

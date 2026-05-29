// Package session provides the HTTP handler layer for the session lifecycle API.
//
// Routes registered:
//
//	POST   /sessions
//	GET    /sessions
//	GET    /sessions/{id}
//	POST   /sessions/{id}/finalize
//	PATCH  /sessions/{id}/output-docs
//
// Input validation:
//   - Session IDs are validated as UUID v4 format before any DB call.
//   - Request bodies are size-limited to 1 MiB.
//   - All required fields are checked; missing fields return HTTP 400.
//
// Error mapping:
//
//	ErrNotFound              → 404
//	ErrConflict              → 409
//	validation errors        → 400
//	other errors             → 500 (detail never exposed to caller)
package session

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/config"
	"a2a-brainstorm/backend/internal/shared"
)

// uuidRE matches UUID v4 format used for session and agent IDs.
var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// sessionService is the subset of *Service required by the Handler.
// Using an interface enables test stubs without importing a live DB.
type sessionService interface {
	CreateSession(ctx context.Context, req CreateSessionRequest) (Session, error)
	GetSession(ctx context.Context, id string) (Session, error)
	ListSessions(ctx context.Context) (ListSessionsResponse, error)
	FinalizeSession(ctx context.Context, id string, input FinalizeInput) (Session, error)
	UpdateOutputDocs(ctx context.Context, id string, docs []string) error
}

// markdownWriter is the subset of the markdown package required by the Handler.
// Injecting an interface keeps the markdown package out of the import graph
// for unit tests that do not need file I/O.
type markdownWriter interface {
	GenerateAll(ctx context.Context, s state.CanonicalState, keys []string) (map[string]shared.GeneratedDocument, error)
	WriteArtifacts(s state.CanonicalState, outputDir string) error
}

// MarkdownWriter is the exported alias of the handler's markdown dependency.
// It exists so the cmd/server wiring can hold the same interface type without
// duplicating the contract. The unexported markdownWriter remains the canonical
// name used internally.
type MarkdownWriter = markdownWriter

// Handler implements the HTTP layer for the session API.
type Handler struct {
	svc       sessionService
	markdown  markdownWriter
	outputDir string
	logger    *slog.Logger
}

// NewHandler constructs a Handler backed by the given Service.
// md may be nil — if nil, markdown generation is skipped on finalize.
// outputDir is the directory where artifacts are written; ignored when md is nil.
func NewHandler(svc *Service, md markdownWriter, outputDir string, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, markdown: md, outputDir: outputDir, logger: logger}
}

// NewHandlerWithService constructs a Handler from any sessionService implementation.
// This is used in tests to inject a stub.
func NewHandlerWithService(svc sessionService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes registers all session HTTP routes on mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /sessions", h.createSession)
	mux.HandleFunc("GET /sessions", h.listSessions)
	mux.HandleFunc("GET /sessions/{id}", h.getSession)
	mux.HandleFunc("POST /sessions/{id}/finalize", h.finalizeSession)
	mux.HandleFunc("PATCH /sessions/{id}/output-docs", h.updateOutputDocs)
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Idea == "" {
		writeError(w, http.StatusBadRequest, "idea is required")
		return
	}
	if len(req.AgentIDs) < 2 {
		writeError(w, http.StatusBadRequest, "at least 2 agent IDs are required")
		return
	}
	for _, id := range req.AgentIDs {
		if !uuidRE.MatchString(id) {
			writeError(w, http.StatusBadRequest, "invalid agent ID format: "+id)
			return
		}
	}

	sess, err := h.svc.CreateSession(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, sess)
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidRE.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid session ID format")
		return
	}

	sess, err := h.svc.GetSession(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListSessions(r.Context())
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	if resp.Sessions == nil {
		resp.Sessions = []SessionListItem{}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) finalizeSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidRE.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid session ID format")
		return
	}

	// Parse optional body — an empty body is treated as FinalizeInput{}.
	var input FinalizeInput
	if r.ContentLength > 0 {
		if err := readJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	sess, err := h.svc.FinalizeSession(r.Context(), id, input)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	// Generate documents for the response body. Document keys come from the
	// persisted session.OutputDocs (which FinalizeSession may have updated
	// from the input override before returning).
	documents := make(map[string]shared.GeneratedDocument)
	if h.markdown != nil && sess.CurrentState != nil {
		keys := sess.OutputDocs
		if len(keys) == 0 {
			keys = DefaultOutputDocs
		}

		// Clear the server-level WriteTimeout — AI markdown generation can
		// exceed the default 300 s limit. The generation is bounded instead
		// by GetFinalizeTimeout(), which defaults to 10 minutes.
		_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

		genCtx, genCancel := context.WithTimeout(r.Context(), config.GetFinalizeTimeout())
		defer genCancel()

		docs, merr := h.markdown.GenerateAll(genCtx, *sess.CurrentState, keys)
		if merr != nil {
			if h.logger != nil {
				h.logger.ErrorContext(r.Context(), "markdown generation failed",
					slog.String("session_id", id),
					slog.Any("error", merr))
			}
			writeError(w, http.StatusInternalServerError, "markdown generation failed")
			return
		}
		documents = docs

		if h.outputDir != "" {
			if werr := h.markdown.WriteArtifacts(*sess.CurrentState, h.outputDir); werr != nil {
				if h.logger != nil {
					h.logger.ErrorContext(r.Context(), "markdown artifact write failed",
						slog.String("session_id", id),
						slog.Any("error", werr))
				}
				// Write failure is non-fatal — session remains approved.
			}
		}
	}

	writeJSON(w, http.StatusOK, FinalizeResponse{
		SessionID: sess.ID,
		Documents: documents,
		Status:    sess.Status,
	})
}

func (h *Handler) updateOutputDocs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidRE.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid session ID format")
		return
	}

	var req UpdateOutputDocsRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.OutputDocs) == 0 {
		writeError(w, http.StatusBadRequest, "output_docs must contain at least one document key")
		return
	}

	if err := h.svc.UpdateOutputDocs(r.Context(), id, req.OutputDocs); err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Error helpers ─────────────────────────────────────────────────────────────

func (h *Handler) handleServiceError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "operation not permitted in current session state")
		return
	}
	if errors.Is(err, ErrStateNotReady) {
		reason := strings.TrimPrefix(err.Error(), "finalize session: load: ")
		reason = strings.TrimPrefix(reason, "finalize session: ")
		reason = strings.TrimPrefix(reason, "state not ready for finalize: ")
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error":  "state_not_ready",
			"reason": reason,
		})
		return
	}
	// Surface validation errors (produced by service layer) as 400.
	if isValidationError(err) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.logger != nil {
		h.logger.ErrorContext(r.Context(), "session handler error", slog.Any("error", err))
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}

// isValidationError returns true for errors that originated from input
// validation in the service layer (e.g., missing fields, invalid roles,
// agent unavailability).  These are safe to surface in the response body.
func isValidationError(err error) bool {
	// All service-layer validation errors are plain errors without DB context.
	// We check for known prefix patterns rather than wrapping every error in a
	// custom type to keep the service layer simple.
	msg := err.Error()
	prefixes := []string{
		"idea is required",
		"at least 2",
		"invalid role",
		"agent ",
		"output_docs",
		"invalid output doc key",
		"duplicate output doc key",
	}
	for _, p := range prefixes {
		if len(msg) >= len(p) && msg[:len(p)] == p {
			return true
		}
	}
	return false
}

// ── JSON helpers ──────────────────────────────────────────────────────────────

func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

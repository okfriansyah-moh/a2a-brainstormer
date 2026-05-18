// Package agent provides the HTTP handler layer for the agent and skill
// REST API endpoints defined in §8.7 of docs/PLAN.md.
//
// All handlers:
//   - Validate UUID path parameters and required body fields.
//   - Return 400 on validation failure, 404 on not-found, 409 on name conflict.
//   - Never leak raw DB error messages to the HTTP response body.
//   - Use application/json for all request and response bodies.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

// uuidRE is used to validate UUID v4 path parameters before they reach the DB.
var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// agentService is the minimal interface satisfied by *Service.  The Handler
// depends on this interface rather than the concrete type so that unit tests
// can inject stubs without a live database.
type agentService interface {
	RegisterAgent(ctx context.Context, req RegisterAgentRequest) (Agent, error)
	GetAgent(ctx context.Context, id string) (Agent, error)
	ListAgents(ctx context.Context) ([]Agent, error)
	UpdateAgent(ctx context.Context, a Agent) (Agent, error)
	DeleteAgent(ctx context.Context, id string) error
	CreateSkill(ctx context.Context, sk Skill) (Skill, error)
	GetSkill(ctx context.Context, id string) (Skill, error)
	ListSkills(ctx context.Context) ([]Skill, error)
	UpdateSkill(ctx context.Context, sk Skill) (Skill, error)
	DeleteSkill(ctx context.Context, id string) error
	AttachSkill(ctx context.Context, agentID, skillID string) error
	DetachSkill(ctx context.Context, agentID, skillID string) error
	GetAgentSkills(ctx context.Context, agentID string) ([]Skill, error)
}

// Handler exposes the agent and skill domain as HTTP endpoints.
// It delegates all business logic to the agentService interface.
type Handler struct {
	svc    agentService
	logger *slog.Logger
}

// NewHandler constructs a Handler backed by the given Service.
func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// NewHandlerWithService constructs a Handler from any agentService implementation.
// This is primarily used in tests to inject a stub service.
func NewHandlerWithService(svc agentService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes binds all agent and skill REST routes to mux.
// Route patterns use Go 1.22+ method-prefixed path syntax.
//
// Agents:
//
//	POST   /agents
//	GET    /agents
//	GET    /agents/{id}
//	PUT    /agents/{id}
//	DELETE /agents/{id}
//	GET    /agents/{id}/skills
//	POST   /agents/{id}/skills/{skill_id}
//	DELETE /agents/{id}/skills/{skill_id}
//
// Skills:
//
//	POST   /skills
//	GET    /skills
//	GET    /skills/{id}
//	PUT    /skills/{id}
//	DELETE /skills/{id}
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /agents", h.createAgent)
	mux.HandleFunc("GET /agents", h.listAgents)
	mux.HandleFunc("GET /agents/{id}", h.getAgent)
	mux.HandleFunc("PUT /agents/{id}", h.updateAgent)
	mux.HandleFunc("DELETE /agents/{id}", h.deleteAgent)
	mux.HandleFunc("GET /agents/{id}/skills", h.getAgentSkills)
	mux.HandleFunc("POST /agents/{id}/skills/{skill_id}", h.attachSkill)
	mux.HandleFunc("DELETE /agents/{id}/skills/{skill_id}", h.detachSkill)

	mux.HandleFunc("POST /skills", h.createSkill)
	mux.HandleFunc("GET /skills", h.listSkills)
	mux.HandleFunc("GET /skills/{id}", h.getSkill)
	mux.HandleFunc("PUT /skills/{id}", h.updateSkill)
	mux.HandleFunc("DELETE /skills/{id}", h.deleteSkill)
}

// ── Agent handlers ────────────────────────────────────────────────────────────

func (h *Handler) createAgent(w http.ResponseWriter, r *http.Request) {
	var req RegisterAgentRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Input validation — delegate deeper checks to service.RegisterAgent.
	req.Name = strings.TrimSpace(req.Name)
	req.Endpoint = strings.TrimSpace(req.Endpoint)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Endpoint == "" {
		writeError(w, http.StatusBadRequest, "endpoint is required")
		return
	}
	if !ValidRole(req.DefaultRole) {
		writeError(w, http.StatusBadRequest, "invalid default_role")
		return
	}

	agent, err := h.svc.RegisterAgent(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

func (h *Handler) listAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := h.svc.ListAgents(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list agents")
		return
	}
	writeJSON(w, http.StatusOK, agents)
}

func (h *Handler) getAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !isValidUUID(id) {
		writeError(w, http.StatusBadRequest, "invalid agent id")
		return
	}
	agent, err := h.svc.GetAgent(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (h *Handler) updateAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !isValidUUID(id) {
		writeError(w, http.StatusBadRequest, "invalid agent id")
		return
	}

	var a Agent
	if err := readJSON(r, &a); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	a.ID = id

	a.Name = strings.TrimSpace(a.Name)
	if a.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !ValidRole(a.DefaultRole) {
		writeError(w, http.StatusBadRequest, "invalid default_role")
		return
	}

	updated, err := h.svc.UpdateAgent(r.Context(), a)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !isValidUUID(id) {
		writeError(w, http.StatusBadRequest, "invalid agent id")
		return
	}
	if err := h.svc.DeleteAgent(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getAgentSkills(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	if !isValidUUID(agentID) {
		writeError(w, http.StatusBadRequest, "invalid agent id")
		return
	}
	skills, err := h.svc.GetAgentSkills(r.Context(), agentID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

func (h *Handler) attachSkill(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	skillID := r.PathValue("skill_id")
	if !isValidUUID(agentID) {
		writeError(w, http.StatusBadRequest, "invalid agent id")
		return
	}
	if !isValidUUID(skillID) {
		writeError(w, http.StatusBadRequest, "invalid skill id")
		return
	}
	if err := h.svc.AttachSkill(r.Context(), agentID, skillID); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) detachSkill(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	skillID := r.PathValue("skill_id")
	if !isValidUUID(agentID) {
		writeError(w, http.StatusBadRequest, "invalid agent id")
		return
	}
	if !isValidUUID(skillID) {
		writeError(w, http.StatusBadRequest, "invalid skill id")
		return
	}
	if err := h.svc.DetachSkill(r.Context(), agentID, skillID); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Skill handlers ────────────────────────────────────────────────────────────

func (h *Handler) createSkill(w http.ResponseWriter, r *http.Request) {
	var sk Skill
	if err := readJSON(r, &sk); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	sk.Name = strings.TrimSpace(sk.Name)
	sk.Prompt = strings.TrimSpace(sk.Prompt)
	if sk.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if sk.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}
	created, err := h.svc.CreateSkill(r.Context(), sk)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) listSkills(w http.ResponseWriter, r *http.Request) {
	skills, err := h.svc.ListSkills(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list skills")
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

func (h *Handler) getSkill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !isValidUUID(id) {
		writeError(w, http.StatusBadRequest, "invalid skill id")
		return
	}
	sk, err := h.svc.GetSkill(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sk)
}

func (h *Handler) updateSkill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !isValidUUID(id) {
		writeError(w, http.StatusBadRequest, "invalid skill id")
		return
	}
	var sk Skill
	if err := readJSON(r, &sk); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	sk.ID = id
	sk.Name = strings.TrimSpace(sk.Name)
	sk.Prompt = strings.TrimSpace(sk.Prompt)
	if sk.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if sk.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}
	updated, err := h.svc.UpdateSkill(r.Context(), sk)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteSkill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !isValidUUID(id) {
		writeError(w, http.StatusBadRequest, "invalid skill id")
		return
	}
	if err := h.svc.DeleteSkill(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// handleServiceError maps service-layer errors to HTTP status codes.
// Raw DB errors are never returned to the caller; only a safe message is sent.
func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	// Detect UNIQUE constraint violation (name conflict) by string heuristic on
	// the pgx error message — avoids importing pgconn in the handler layer.
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE") || strings.Contains(msg, "unique") ||
		strings.Contains(msg, "duplicate") || strings.Contains(msg, "_unique") {
		writeError(w, http.StatusConflict, "name already exists")
		return
	}
	h.logger.Error("service error", slog.String("error", msg))
	writeError(w, http.StatusInternalServerError, "internal error")
}

// isValidUUID returns true if s matches the canonical UUID format.
func isValidUUID(s string) bool {
	return uuidRE.MatchString(s)
}

// writeJSON serialises v to JSON and writes it with the given HTTP status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Headers already sent; log only.
		_ = err
	}
}

// writeError sends a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// readJSON decodes the request body into v.
// Returns an error if the body is malformed or exceeds 1 MiB.
func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

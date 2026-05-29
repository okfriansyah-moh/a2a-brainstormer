// Package session_test provides unit tests for the session module.
//
// Coverage:
//   - Service.CreateSession — validation rules (missing idea, <2 agents,
//     duplicate agents, invalid role override, agent unavailability)
//   - Handler input validation — UUID format, missing required fields
//
// Repository-backed paths require a live DB and are covered by integration
// tests (Task 15).
package session_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"a2a-brainstorm/backend/internal/modules/agent"
	"a2a-brainstorm/backend/internal/modules/session"
)

// ── stub agentProvider ────────────────────────────────────────────────────────

// stubAgentProvider implements the agentProvider interface used by Service.
// By default GetAgent returns a minimal Agent and CheckAvailability succeeds.
type stubAgentProvider struct {
	getAgentErr       error
	checkAvailability error
}

func (s stubAgentProvider) GetAgent(_ context.Context, id string) (agent.Agent, error) {
	if s.getAgentErr != nil {
		return agent.Agent{}, s.getAgentErr
	}
	return agent.Agent{ID: id, Name: "stub-" + id, DefaultRole: agent.RoleBuilder, Endpoint: "http://agent"}, nil
}

func (s stubAgentProvider) CheckAvailability(_ context.Context, _ agent.Agent) error {
	return s.checkAvailability
}

// ── Service unit tests ────────────────────────────────────────────────────────

// createSvc builds a Service with nil repository and the given stub provider.
// The nil repo means these tests must not exercise any path that calls repo.
// We use a testRepository shim for paths that call CreateSession / CreateSessionAgents.
func newTestService(ap stubAgentProvider) *session.Service {
	return session.NewServiceWithDeps(nil, ap, nil)
}

func TestCreateSession_MissingIdea(t *testing.T) {
	svc := newTestService(stubAgentProvider{})
	_, err := svc.CreateSession(context.Background(), session.CreateSessionRequest{
		AgentIDs: []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"},
	})
	if err == nil {
		t.Fatal("expected error for missing idea")
	}
}

func TestCreateSession_TooFewAgents(t *testing.T) {
	svc := newTestService(stubAgentProvider{})
	_, err := svc.CreateSession(context.Background(), session.CreateSessionRequest{
		Idea:     "test idea",
		AgentIDs: []string{"00000000-0000-0000-0000-000000000001"},
	})
	if err == nil {
		t.Fatal("expected error for < 2 agents")
	}
}

func TestCreateSession_DuplicateAgents(t *testing.T) {
	svc := newTestService(stubAgentProvider{})
	id := "00000000-0000-0000-0000-000000000001"
	_, err := svc.CreateSession(context.Background(), session.CreateSessionRequest{
		Idea:     "test idea",
		AgentIDs: []string{id, id},
	})
	if err == nil {
		t.Fatal("expected error for duplicate agents")
	}
}

func TestCreateSession_InvalidRoleOverride(t *testing.T) {
	svc := newTestService(stubAgentProvider{})
	id1 := "00000000-0000-0000-0000-000000000001"
	id2 := "00000000-0000-0000-0000-000000000002"
	_, err := svc.CreateSession(context.Background(), session.CreateSessionRequest{
		Idea:          "test idea",
		AgentIDs:      []string{id1, id2},
		RoleOverrides: map[string]string{id1: "invalid_role"},
	})
	if err == nil {
		t.Fatal("expected error for invalid role override")
	}
}

func TestCreateSession_AgentNotFound(t *testing.T) {
	svc := newTestService(stubAgentProvider{getAgentErr: agent.ErrNotFound})
	_, err := svc.CreateSession(context.Background(), session.CreateSessionRequest{
		Idea:     "test idea",
		AgentIDs: []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"},
	})
	if err == nil {
		t.Fatal("expected error when agent not found")
	}
}

func TestCreateSession_AgentUnavailable(t *testing.T) {
	svc := newTestService(stubAgentProvider{checkAvailability: errUnavailable})
	_, err := svc.CreateSession(context.Background(), session.CreateSessionRequest{
		Idea:     "test idea",
		AgentIDs: []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"},
	})
	if err == nil {
		t.Fatal("expected error when agent unavailable")
	}
}

var errUnavailable = agent.ErrNotFound // reuse sentinel for availability stub

// ── Handler validation tests ──────────────────────────────────────────────────

// stubSessionService implements the sessionService interface for handler tests.
type stubSessionService struct{}

func (stubSessionService) CreateSession(_ context.Context, _ session.CreateSessionRequest) (session.Session, error) {
	return session.Session{}, session.ErrNotFound
}
func (stubSessionService) GetSession(_ context.Context, _ string) (session.Session, error) {
	return session.Session{}, session.ErrNotFound
}
func (stubSessionService) ListSessions(_ context.Context) (session.ListSessionsResponse, error) {
	return session.ListSessionsResponse{Sessions: []session.SessionListItem{}, Total: 0}, nil
}
func (stubSessionService) FinalizeSession(_ context.Context, _ string, _ session.FinalizeInput) (session.Session, error) {
	return session.Session{}, session.ErrNotFound
}
func (stubSessionService) UpdateOutputDocs(_ context.Context, _ string, _ []string) error {
	return session.ErrNotFound
}

func buildTestMux() *http.ServeMux {
	mux := http.NewServeMux()
	session.NewHandlerWithService(stubSessionService{}, nil).RegisterRoutes(mux)
	return mux
}

func TestHandler_GetSession_InvalidUUID(t *testing.T) {
	mux := buildTestMux()
	req := httptest.NewRequest(http.MethodGet, "/sessions/not-a-uuid", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_GetSession_ValidUUID_ReturnsNotFound(t *testing.T) {
	mux := buildTestMux()
	req := httptest.NewRequest(http.MethodGet, "/sessions/00000000-0000-0000-0000-000000000001", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandler_CreateSession_MissingIdea(t *testing.T) {
	mux := buildTestMux()
	body, _ := json.Marshal(map[string]any{
		"agent_ids": []string{
			"00000000-0000-0000-0000-000000000001",
			"00000000-0000-0000-0000-000000000002",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_CreateSession_TooFewAgents(t *testing.T) {
	mux := buildTestMux()
	body, _ := json.Marshal(map[string]any{
		"idea":      "test",
		"agent_ids": []string{"00000000-0000-0000-0000-000000000001"},
	})
	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_CreateSession_InvalidAgentIDFormat(t *testing.T) {
	mux := buildTestMux()
	body, _ := json.Marshal(map[string]any{
		"idea":      "test",
		"agent_ids": []string{"not-a-uuid", "also-not-a-uuid"},
	})
	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_FinalizeSession_InvalidUUID(t *testing.T) {
	mux := buildTestMux()
	req := httptest.NewRequest(http.MethodPost, "/sessions/bad-id/finalize", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_ListSessions_ReturnsEmptyEnvelope(t *testing.T) {
	mux := buildTestMux()
	req := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result session.ListSessionsResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("expected ListSessionsResponse JSON, got: %s", w.Body.String())
	}
	if result.Sessions == nil {
		t.Fatal("sessions field must be a non-nil array")
	}
	if len(result.Sessions) != 0 {
		t.Fatalf("expected empty sessions array, got %d items", len(result.Sessions))
	}
	if result.Total != 0 {
		t.Fatalf("expected total=0, got %d", result.Total)
	}
}

// ── FinalizeSession FinalizeResponse tests ────────────────────────────────────

// stubFinalizeService returns a valid Session with approved status for finalize.
type stubFinalizeService struct{ stubSessionService }

func (stubFinalizeService) FinalizeSession(_ context.Context, id string, _ session.FinalizeInput) (session.Session, error) {
	return session.Session{
		ID:     id,
		Status: "approved",
	}, nil
}

func TestHandler_FinalizeSession_ReturnsFinalizeResponse(t *testing.T) {
	mux := http.NewServeMux()
	session.NewHandlerWithService(stubFinalizeService{}, nil).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/sessions/00000000-0000-0000-0000-000000000001/finalize", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp session.FinalizeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("expected FinalizeResponse JSON, got: %s", w.Body.String())
	}
	if resp.SessionID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("expected session_id to match, got %q", resp.SessionID)
	}
	if resp.Status != "approved" {
		t.Errorf("expected status=approved, got %q", resp.Status)
	}
}

func TestHandler_FinalizeSession_ValidUUID_NotFound(t *testing.T) {
	mux := buildTestMux()
	req := httptest.NewRequest(http.MethodPost, "/sessions/00000000-0000-0000-0000-000000000001/finalize", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// stubNotReadyService returns ErrStateNotReady wrapped with a reason,
// simulating the §8.23 readiness gate rejecting a premature finalize.
type stubNotReadyService struct{ stubSessionService }

func (stubNotReadyService) FinalizeSession(_ context.Context, _ string, _ session.FinalizeInput) (session.Session, error) {
	return session.Session{}, fmt.Errorf("%w: confidence 0.1000 is below threshold 0.5", session.ErrStateNotReady)
}

func TestHandler_FinalizeSession_StateNotReady_Returns422(t *testing.T) {
	mux := http.NewServeMux()
	session.NewHandlerWithService(stubNotReadyService{}, nil).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/sessions/00000000-0000-0000-0000-000000000001/finalize", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d (body=%s)", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "state_not_ready" {
		t.Errorf("error field = %q, want state_not_ready", body["error"])
	}
	if body["reason"] == "" {
		t.Errorf("expected non-empty reason field, got: %v", body)
	}
}

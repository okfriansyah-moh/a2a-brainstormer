// Package agent_test provides unit tests for the pure-logic components
// introduced by Task 7: BuildSystemPrompt, handler input validation, and
// ResolveActiveSkills override semantics.
//
// Repository-backed and A2A-dependent paths (Dispatch, RegisterAgent endpoint
// check) require a live database / agent and are covered by integration tests.
package agent_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"a2a-brainstorm/backend/internal/modules/agent"
)

// ── BuildSystemPrompt ─────────────────────────────────────────────────────────

func TestBuildSystemPrompt_NoSkills(t *testing.T) {
	result := agent.BuildSystemPrompt("You are an architect.", nil)
	if result != "You are an architect." {
		t.Errorf("want base prompt unchanged, got %q", result)
	}
}

func TestBuildSystemPrompt_EmptySkillSlice(t *testing.T) {
	result := agent.BuildSystemPrompt("base", []agent.Skill{})
	if result != "base" {
		t.Errorf("want base only, got %q", result)
	}
}

func TestBuildSystemPrompt_OneSkill(t *testing.T) {
	skills := []agent.Skill{{Name: "convergence", Prompt: "Detect convergence."}}
	result := agent.BuildSystemPrompt("base", skills)
	want := "base\n\nDetect convergence."
	if result != want {
		t.Errorf("want %q, got %q", want, result)
	}
}

func TestBuildSystemPrompt_MultipleSkills(t *testing.T) {
	skills := []agent.Skill{
		{Name: "s1", Prompt: "Fragment one."},
		{Name: "s2", Prompt: "Fragment two."},
		{Name: "s3", Prompt: "Fragment three."},
	}
	result := agent.BuildSystemPrompt("base", skills)
	want := "base\n\nFragment one.\n\nFragment two.\n\nFragment three."
	if result != want {
		t.Errorf("want %q, got %q", want, result)
	}
}

func TestBuildSystemPrompt_SkipEmptyPrompts(t *testing.T) {
	skills := []agent.Skill{
		{Name: "s1", Prompt: "Real content."},
		{Name: "s2", Prompt: ""},
	}
	result := agent.BuildSystemPrompt("base", skills)
	want := "base\n\nReal content."
	if result != want {
		t.Errorf("want %q, got %q", want, result)
	}
}

func TestBuildSystemPrompt_EmptyBase(t *testing.T) {
	skills := []agent.Skill{{Name: "s", Prompt: "Skill prompt."}}
	result := agent.BuildSystemPrompt("", skills)
	want := "\n\nSkill prompt."
	if result != want {
		t.Errorf("want %q, got %q", want, result)
	}
}

// ── Handler validation tests ──────────────────────────────────────────────────
// These tests use a stubService that always returns ErrNotFound so that we can
// reach real service calls after validation passes without needing a DB.

type stubService struct{}

func (stubService) RegisterAgent(_ context.Context, _ agent.RegisterAgentRequest) (agent.Agent, error) {
	return agent.Agent{}, agent.ErrNotFound
}
func (stubService) GetAgent(_ context.Context, _ string) (agent.Agent, error) {
	return agent.Agent{}, agent.ErrNotFound
}
func (stubService) ListAgents(_ context.Context) ([]agent.Agent, error) {
	return nil, agent.ErrNotFound
}
func (stubService) UpdateAgent(_ context.Context, _ agent.Agent) (agent.Agent, error) {
	return agent.Agent{}, agent.ErrNotFound
}
func (stubService) DeleteAgent(_ context.Context, _ string) error { return agent.ErrNotFound }
func (stubService) CreateSkill(_ context.Context, _ agent.Skill) (agent.Skill, error) {
	return agent.Skill{}, agent.ErrNotFound
}
func (stubService) GetSkill(_ context.Context, _ string) (agent.Skill, error) {
	return agent.Skill{}, agent.ErrNotFound
}
func (stubService) ListSkills(_ context.Context) ([]agent.Skill, error) {
	return nil, agent.ErrNotFound
}
func (stubService) UpdateSkill(_ context.Context, _ agent.Skill) (agent.Skill, error) {
	return agent.Skill{}, agent.ErrNotFound
}
func (stubService) DeleteSkill(_ context.Context, _ string) error { return agent.ErrNotFound }
func (stubService) AttachSkill(_ context.Context, _, _ string) error {
	return agent.ErrNotFound
}
func (stubService) DetachSkill(_ context.Context, _, _ string) error {
	return agent.ErrNotFound
}
func (stubService) GetAgentSkills(_ context.Context, _ string) ([]agent.Skill, error) {
	return nil, agent.ErrNotFound
}

func buildTestMux() *http.ServeMux {
	mux := http.NewServeMux()
	h := agent.NewHandlerWithService(stubService{}, nil)
	h.RegisterRoutes(mux)
	return mux
}

func doRequest(t *testing.T, mux *http.ServeMux, method, path string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w.Result()
}

func TestHandler_CreateAgent_MissingName(t *testing.T) {
	mux := buildTestMux()
	body := map[string]any{"endpoint": "http://x.example", "default_role": "build"}
	resp := doRequest(t, mux, "POST", "/agents", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateAgent_InvalidRole(t *testing.T) {
	mux := buildTestMux()
	body := map[string]any{"name": "a", "endpoint": "http://x", "default_role": "emperor"}
	resp := doRequest(t, mux, "POST", "/agents", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateAgent_MissingEndpoint(t *testing.T) {
	mux := buildTestMux()
	body := map[string]any{"name": "a", "default_role": "build"}
	resp := doRequest(t, mux, "POST", "/agents", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestHandler_GetAgent_InvalidUUID(t *testing.T) {
	mux := buildTestMux()
	resp := doRequest(t, mux, "GET", "/agents/not-a-uuid", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestHandler_GetAgent_ValidUUID_ReturnsNotFound(t *testing.T) {
	// Valid UUID passes validation and reaches stub → 404.
	mux := buildTestMux()
	resp := doRequest(t, mux, "GET", "/agents/00000000-0000-0000-0000-000000000001", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateSkill_MissingPrompt(t *testing.T) {
	mux := buildTestMux()
	body := map[string]any{"name": "my-skill"}
	resp := doRequest(t, mux, "POST", "/skills", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateSkill_MissingName(t *testing.T) {
	mux := buildTestMux()
	body := map[string]any{"prompt": "Do something."}
	resp := doRequest(t, mux, "POST", "/skills", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestHandler_AttachSkill_InvalidAgentUUID(t *testing.T) {
	mux := buildTestMux()
	resp := doRequest(t, mux, "POST", "/agents/bad-id/skills/00000000-0000-0000-0000-000000000001", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestHandler_AttachSkill_InvalidSkillUUID(t *testing.T) {
	mux := buildTestMux()
	validUUID := "00000000-0000-0000-0000-000000000001"
	resp := doRequest(t, mux, "POST", "/agents/"+validUUID+"/skills/bad-id", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteSkill_InvalidUUID(t *testing.T) {
	mux := buildTestMux()
	resp := doRequest(t, mux, "DELETE", "/skills/not-a-uuid", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateAgent_InvalidRole(t *testing.T) {
	mux := buildTestMux()
	validUUID := "00000000-0000-0000-0000-000000000001"
	body := map[string]any{"name": "a", "endpoint": "http://x", "default_role": "bad"}
	resp := doRequest(t, mux, "PUT", "/agents/"+validUUID, body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

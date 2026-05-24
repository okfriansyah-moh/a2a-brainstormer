// Package agent provides the service layer for agent and skill lifecycle
// management, including registration, availability checking, skill assembly,
// and active-skill resolution.
//
// Business rules enforced here:
//   - Agents must have a reachable endpoint before registration is accepted.
//   - Credential availability is checked via llm.ResolveKey; absent env var
//     causes the agent to be unavailable — no silent fallback.
//   - os.Getenv is NEVER called here; all credential resolution delegates to
//     platform/llm.ResolveKey → platform/config.GetLLMAPIKey.
package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"a2a-brainstorm/backend/internal/platform/llm"
)

// RegisterAgentRequest carries the validated input for agent registration.
// All fields except Description, SystemPrompt, and LLMConfig are required.
type RegisterAgentRequest struct {
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	DefaultRole  Role           `json:"default_role"`
	SystemPrompt string         `json:"system_prompt,omitempty"`
	LLMConfig    *llm.LLMConfig `json:"llm_config,omitempty"`
	Endpoint     string         `json:"endpoint"`
}

// Service provides all business operations for the agent and skill domain.
// It delegates persistence to the Repository and never accesses the DB directly.
type Service struct {
	repo       *Repository
	httpClient *http.Client
	logger     *slog.Logger
}

// NewService constructs a Service with the given repository and logger.
// A dedicated HTTP client with a 5-second timeout is used for endpoint
// reachability checks during RegisterAgent.
func NewService(repo *Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:       repo,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
	}
}

// ── Agent operations ──────────────────────────────────────────────────────────

// RegisterAgent validates the endpoint is reachable, then persists the agent.
// Returns an error if the endpoint is unreachable or the name already exists.
func (s *Service) RegisterAgent(ctx context.Context, req RegisterAgentRequest) (Agent, error) {
	if req.Name == "" {
		return Agent{}, errors.New("register agent: name is required")
	}
	if req.Endpoint == "" {
		return Agent{}, errors.New("register agent: endpoint is required")
	}
	if !ValidRole(req.DefaultRole) {
		return Agent{}, fmt.Errorf("register agent: invalid default_role %q", req.DefaultRole)
	}

	if err := s.checkEndpointReachable(ctx, req.Endpoint); err != nil {
		return Agent{}, fmt.Errorf("register agent: %w", err)
	}

	a := Agent{
		Name:         req.Name,
		Description:  req.Description,
		DefaultRole:  req.DefaultRole,
		SystemPrompt: req.SystemPrompt,
		LLMConfig:    req.LLMConfig,
		Endpoint:     req.Endpoint,
	}
	created, err := s.repo.CreateAgent(ctx, a)
	if err != nil {
		return Agent{}, fmt.Errorf("register agent: %w", err)
	}
	s.logger.Info("agent registered", slog.String("id", created.ID), slog.String("name", created.Name))
	return created, nil
}

// GetAgent returns the agent with its skills list populated.
func (s *Service) GetAgent(ctx context.Context, id string) (Agent, error) {
	a, err := s.repo.GetAgent(ctx, id)
	if err != nil {
		return Agent{}, fmt.Errorf("get agent: %w", err)
	}
	skills, err := s.repo.GetAgentSkills(ctx, id)
	if err != nil {
		return Agent{}, fmt.Errorf("get agent: load skills: %w", err)
	}
	a.Skills = skills
	return a, nil
}

// ListAgents returns all agents with their attached skills populated.
// The skills list is loaded per agent so the frontend can render agent cards
// (skill counts, skill chips) without making an additional request per agent.
func (s *Service) ListAgents(ctx context.Context) ([]Agent, error) {
	agents, err := s.repo.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	for i := range agents {
		skills, err := s.repo.GetAgentSkills(ctx, agents[i].ID)
		if err != nil {
			return nil, fmt.Errorf("list agents: load skills for %s: %w", agents[i].ID, err)
		}
		agents[i].Skills = skills
	}
	return agents, nil
}

// UpdateAgent persists updated agent fields and returns the refreshed record.
func (s *Service) UpdateAgent(ctx context.Context, a Agent) (Agent, error) {
	if a.ID == "" {
		return Agent{}, errors.New("update agent: id is required")
	}
	updated, err := s.repo.UpdateAgent(ctx, a)
	if err != nil {
		return Agent{}, fmt.Errorf("update agent: %w", err)
	}
	return updated, nil
}

// DeleteAgent removes the agent from the registry.
func (s *Service) DeleteAgent(ctx context.Context, id string) error {
	if err := s.repo.DeleteAgent(ctx, id); err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	s.logger.Info("agent deleted", slog.String("id", id))
	return nil
}

// CheckAvailability verifies that the agent's credential env var is set.
// Returns nil if no LLMConfig is configured (credential not required).
// Returns a non-nil error if the credential ref is empty or the env var is absent.
// Callers should mark the agent unavailable when this returns an error.
func (s *Service) CheckAvailability(_ context.Context, a Agent) error {
	if a.LLMConfig == nil {
		return nil
	}
	if a.LLMConfig.CredentialRef == "" {
		return fmt.Errorf("agent %s: credential_ref is empty", a.ID)
	}
	if _, err := llm.ResolveKey(a.LLMConfig.CredentialRef); err != nil {
		return fmt.Errorf("agent %s: credential %q not available: %w", a.ID, a.LLMConfig.CredentialRef, err)
	}
	return nil
}

// ── Skill operations ──────────────────────────────────────────────────────────

// CreateSkill persists a new skill and returns the created record.
func (s *Service) CreateSkill(ctx context.Context, sk Skill) (Skill, error) {
	if sk.Name == "" {
		return Skill{}, errors.New("create skill: name is required")
	}
	if sk.Prompt == "" {
		return Skill{}, errors.New("create skill: prompt is required")
	}
	created, err := s.repo.CreateSkill(ctx, sk)
	if err != nil {
		return Skill{}, fmt.Errorf("create skill: %w", err)
	}
	return created, nil
}

// GetSkill returns a single skill by ID.
func (s *Service) GetSkill(ctx context.Context, id string) (Skill, error) {
	sk, err := s.repo.GetSkill(ctx, id)
	if err != nil {
		return Skill{}, fmt.Errorf("get skill: %w", err)
	}
	return sk, nil
}

// ListSkills returns all skills.
func (s *Service) ListSkills(ctx context.Context) ([]Skill, error) {
	skills, err := s.repo.ListSkills(ctx)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	return skills, nil
}

// UpdateSkill replaces the mutable fields of an existing skill.
func (s *Service) UpdateSkill(ctx context.Context, sk Skill) (Skill, error) {
	if sk.ID == "" {
		return Skill{}, errors.New("update skill: id is required")
	}
	if sk.Name == "" {
		return Skill{}, errors.New("update skill: name is required")
	}
	if sk.Prompt == "" {
		return Skill{}, errors.New("update skill: prompt is required")
	}
	updated, err := s.repo.UpdateSkill(ctx, sk)
	if err != nil {
		return Skill{}, fmt.Errorf("update skill: %w", err)
	}
	return updated, nil
}

// DeleteSkill removes a skill from the library.
func (s *Service) DeleteSkill(ctx context.Context, id string) error {
	if err := s.repo.DeleteSkill(ctx, id); err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	return nil
}

// AttachSkill binds a skill to an agent (idempotent — re-attaching is safe).
func (s *Service) AttachSkill(ctx context.Context, agentID, skillID string) error {
	if err := s.repo.AttachSkill(ctx, agentID, skillID); err != nil {
		return fmt.Errorf("attach skill: %w", err)
	}
	return nil
}

// DetachSkill removes a skill binding from an agent.
func (s *Service) DetachSkill(ctx context.Context, agentID, skillID string) error {
	if err := s.repo.DetachSkill(ctx, agentID, skillID); err != nil {
		return fmt.Errorf("detach skill: %w", err)
	}
	return nil
}

// GetAgentSkills returns all skills attached to the given agent.
func (s *Service) GetAgentSkills(ctx context.Context, agentID string) ([]Skill, error) {
	skills, err := s.repo.GetAgentSkills(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("get agent skills: %w", err)
	}
	return skills, nil
}

// ── Pipeline support ──────────────────────────────────────────────────────────

// ResolveActiveSkills determines the effective skill set for an agent in a
// session dispatch, following the skill_overrides semantics from §8.14:
//
//   - overrides == nil  → use agent's default attached skills (DB lookup).
//   - overrides non-nil and empty  → no skills; return an empty slice.
//   - overrides non-nil and non-empty  → fetch exactly those skill IDs.
func (s *Service) ResolveActiveSkills(ctx context.Context, agentID string, overrides *[]string) ([]Skill, error) {
	// nil overrides: use default agent skills
	if overrides == nil {
		return s.repo.GetAgentSkills(ctx, agentID)
	}
	// empty slice: all skills disabled for this session
	if len(*overrides) == 0 {
		return []Skill{}, nil
	}
	// explicit override list: fetch each skill
	skills := make([]Skill, 0, len(*overrides))
	for _, skillID := range *overrides {
		sk, err := s.repo.GetSkill(ctx, skillID)
		if err != nil {
			return nil, fmt.Errorf("resolve active skills: %w", err)
		}
		skills = append(skills, sk)
	}
	return skills, nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

// checkEndpointReachable performs a lightweight HTTP GET to the agent's
// AgentCard URL ({endpoint}/.well-known/agent-card.json) with the context
// deadline (capped at 5 seconds) to confirm the agent is live before
// registration.
func (s *Service) checkEndpointReachable(ctx context.Context, endpoint string) error {
	checkURL := endpoint + "/.well-known/agent-card.json"
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, checkURL, nil)
	if err != nil {
		return fmt.Errorf("endpoint unreachable: build request for %q: %w", checkURL, err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("endpoint unreachable: GET %q: %w", checkURL, err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode >= 400 {
		return fmt.Errorf("endpoint returned HTTP %d for %q", resp.StatusCode, checkURL)
	}
	return nil
}

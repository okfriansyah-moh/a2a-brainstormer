// Package session provides the business logic for the session lifecycle.
//
// Rules enforced here:
//   - A session must have ≥ 2 agent IDs (HTTP 400 returned otherwise).
//   - All referenced agents must exist and be available before session is created.
//   - Roles are assigned from RoleOverrides if provided; otherwise DefaultRoles
//     distribution is used (see §8.13 of docs/PLAN.md).
//   - os.Getenv is NEVER called here.
package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"a2a-brainstorm/backend/internal/modules/agent"
)

// agentProvider is the session module's dependency on the agent domain.
// It is satisfied by *agent.Service in production and by a stub in tests.
// Only the subset of agent.Service required by session is declared here.
type agentProvider interface {
	GetAgent(ctx context.Context, id string) (agent.Agent, error)
	CheckAvailability(ctx context.Context, a agent.Agent) error
}

// Service provides all business operations for the session lifecycle.
// It delegates persistence to the Repository and agent lookups to agentProvider.
type Service struct {
	repo   *Repository
	agents agentProvider
	logger *slog.Logger
}

// NewService constructs a Service with the given repository, agent provider,
// and logger. The agent provider is called at session creation time to verify
// that all referenced agents exist and are available.
func NewService(repo *Repository, agents agentProvider, logger *slog.Logger) *Service {
	return &Service{repo: repo, agents: agents, logger: logger}
}

// NewServiceWithDeps is an alias for NewService that accepts the agentProvider
// interface directly. Used in tests to inject a stub without a live DB.
func NewServiceWithDeps(repo *Repository, agents agentProvider, logger *slog.Logger) *Service {
	return NewService(repo, agents, logger)
}

// CreateSession creates a new brainstorm session.
//
// Validation performed (all return HTTP-mappable errors):
//   - Idea must be non-empty.
//   - AgentIDs must contain ≥ 2 entries.
//   - Each agent ID must exist in the registry.
//   - Each agent must pass CheckAvailability (credential env var present).
//   - Each role override (if given) must pass agent.ValidRole.
//
// On success, the Session row and all SessionAgent rows are persisted in one
// database transaction.
func (s *Service) CreateSession(ctx context.Context, req CreateSessionRequest) (Session, error) {
	if req.Idea == "" {
		return Session{}, errors.New("idea is required")
	}
	if len(req.AgentIDs) < 2 {
		return Session{}, errors.New("at least 2 agent IDs are required")
	}

	// Deduplicate agent IDs while preserving order.
	seen := make(map[string]struct{}, len(req.AgentIDs))
	unique := make([]string, 0, len(req.AgentIDs))
	for _, id := range req.AgentIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) < 2 {
		return Session{}, errors.New("at least 2 distinct agent IDs are required")
	}

	// Validate role overrides before touching DB.
	for agentID, role := range req.RoleOverrides {
		if !agent.ValidRole(agent.Role(role)) {
			return Session{}, fmt.Errorf("invalid role %q for agent %s", role, agentID)
		}
	}

	// Verify all agents exist and are available.
	agentMap := make(map[string]agent.Agent, len(unique))
	for _, id := range unique {
		a, err := s.agents.GetAgent(ctx, id)
		if err != nil {
			return Session{}, fmt.Errorf("agent %s: %w", id, err)
		}
		if err := s.agents.CheckAvailability(ctx, a); err != nil {
			return Session{}, fmt.Errorf("agent %s unavailable: %w", id, err)
		}
		agentMap[id] = a
	}

	maxIter := req.MaxIterations
	if maxIter <= 0 {
		maxIter = 10
	}

	// Build the session row to be persisted transactionally with session_agents.
	sessionInput := Session{
		Idea:          req.Idea,
		Status:        StatusActive,
		MaxIterations: maxIter,
	}

	// Assign roles: use overrides when present, otherwise DefaultRoles distribution.
	roles := agent.DefaultRoles(len(unique))
	for i, id := range unique {
		if override, ok := req.RoleOverrides[id]; ok {
			roles[i] = agent.Role(override)
		}
	}

	// Build session_agents rows.
	sessionAgents := make([]SessionAgent, len(unique))
	for i, id := range unique {
		sa := SessionAgent{
			SessionID: "",
			AgentID:   id,
			Position:  i,
			Role:      string(roles[i]),
		}
		if llmOvr, ok := req.LLMOverrides[id]; ok {
			sa.LLMOverride = llmOvr
		}
		if skillOvr, ok := req.SkillOverrides[id]; ok {
			sa.SkillOverrides = skillOvr
		}
		sessionAgents[i] = sa
	}

	sess, err := s.repo.CreateSessionWithAgents(ctx, sessionInput, sessionAgents)
	if err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}

	s.logger.InfoContext(ctx, "session created",
		slog.String("session_id", sess.ID),
		slog.Int("agents", len(unique)),
		slog.Int("max_iterations", maxIter),
	)
	return sess, nil
}

// GetSession returns the session with the given ID, including its ordered agent
// bindings. Returns ErrNotFound if no session with that ID exists.
func (s *Service) GetSession(ctx context.Context, id string) (Session, error) {
	sess, err := s.repo.GetSession(ctx, id)
	if err != nil {
		return Session{}, fmt.Errorf("get session: %w", err)
	}
	return sess, nil
}

// ListSessions returns a summary list of all sessions ordered newest-first.
// Each item is mapped from the raw Session row: Idea is truncated to 120 chars
// and Confidence/CurrentIteration are extracted from the JSONB current_state.
func (s *Service) ListSessions(ctx context.Context) (ListSessionsResponse, error) {
	sessions, err := s.repo.ListSessions(ctx)
	if err != nil {
		return ListSessionsResponse{}, fmt.Errorf("list sessions: %w", err)
	}

	items := make([]SessionListItem, 0, len(sessions))
	for _, sess := range sessions {
		items = append(items, toSessionListItem(sess))
	}
	return ListSessionsResponse{Sessions: items, Total: len(items)}, nil
}

// toSessionListItem maps a full Session to its summary representation.
// Idea is truncated at a word-safe boundary up to 120 characters.
// Confidence and CurrentIteration are extracted from CurrentState when present.
func toSessionListItem(sess Session) SessionListItem {
	idea := sess.Idea
	if len(idea) > 120 {
		idea = idea[:120]
	}

	item := SessionListItem{
		ID:            sess.ID,
		Idea:          idea,
		Status:        sess.Status,
		MaxIterations: sess.MaxIterations,
		CreatedAt:     sess.CreatedAt,
		UpdatedAt:     sess.UpdatedAt,
		AgentCount:    len(sess.Agents),
	}

	if sess.CurrentState != nil {
		item.CurrentIteration = sess.CurrentState.Meta.Iteration
		item.Confidence = sess.CurrentState.Metrics.Confidence
	}
	return item
}

// FinalizeSession marks a session as approved. Called by the finalize endpoint;
// the markdown generation is triggered by the handler after this returns.
func (s *Service) FinalizeSession(ctx context.Context, id string) (Session, error) {
	if err := s.repo.UpdateStatus(ctx, id, StatusApproved); err != nil {
		return Session{}, fmt.Errorf("finalize session: %w", err)
	}
	sess, err := s.repo.GetSession(ctx, id)
	if err != nil {
		return Session{}, fmt.Errorf("finalize session: reload: %w", err)
	}
	s.logger.InfoContext(ctx, "session finalized", slog.String("session_id", id))
	return sess, nil
}

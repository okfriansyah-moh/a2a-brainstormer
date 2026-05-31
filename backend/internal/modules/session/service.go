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
	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/sse"
)

// ErrStateNotReady is returned by FinalizeSession when the canonical state
// does not yet satisfy the readiness gate: either CurrentState is nil (no
// iteration has completed) or the Idea field is empty.
// The wrapped error message contains the specific reason.
var ErrStateNotReady = errors.New("state not ready for finalize")

// isStateReadyForFinalize returns whether s satisfies the §8.23 readiness
// gate, plus a human-readable reason describing the first failing check.
// Confidence threshold is intentionally low (0.1) so that a converged session
// with partial agreement can still be finalized — the user has already seen
// the canonical state and opted in by clicking Finalize.
//
// Architecture and execution_plan are NOT required: the markdown generator can
// produce useful output from a partial state; blocking finalization on those
// fields prevents document generation when agents ran but produced incomplete
// output (e.g. after provider failures).
//
// Confidence is NOT checked here: if CurrentState is non-nil (enforced by the
// caller) then at least one iteration completed. A failed-agent session may
// have confidence=0 but the user should still be able to generate documents.
func isStateReadyForFinalize(s state.CanonicalState) (bool, string) {
	if len(s.Idea) == 0 {
		return false, "idea is empty; run at least one iteration before finalizing"
	}
	return true, ""
}

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
	repo    *Repository
	agents  agentProvider
	emitter sse.EventEmitter
	logger  *slog.Logger
}

// NewService constructs a Service with the given repository, agent provider,
// and logger. The agent provider is called at session creation time to verify
// that all referenced agents exist and are available.
func NewService(repo *Repository, agents agentProvider, logger *slog.Logger) *Service {
	return &Service{repo: repo, agents: agents, emitter: sse.NoopEmitter{}, logger: logger}
}

// NewServiceWithDeps is an alias for NewService that accepts the agentProvider
// interface directly. Used in tests to inject a stub without a live DB.
func NewServiceWithDeps(repo *Repository, agents agentProvider, logger *slog.Logger) *Service {
	return NewService(repo, agents, logger)
}

// SetEmitter configures the EventEmitter used to publish SSE lifecycle events.
// Call this after NewService when the SSE broadcaster is available. The zero
// value (NoopEmitter) is safe — SetEmitter is optional.
func (s *Service) SetEmitter(emitter sse.EventEmitter) {
	if emitter == nil {
		emitter = sse.NoopEmitter{}
	}
	s.emitter = emitter
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

	// deduplicate and validate OutputDocs.
	outputDocs := req.OutputDocs
	if len(outputDocs) == 0 {
		outputDocs = DefaultOutputDocs
	} else {
		if err := validateOutputDocs(outputDocs); err != nil {
			return Session{}, err
		}
	}

	// Build the session row to be persisted transactionally with session_agents.
	sessionInput := Session{
		Idea:          req.Idea,
		Status:        StatusActive,
		MaxIterations: maxIter,
		OutputDocs:    outputDocs,
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
// Idea is truncated at a rune-safe boundary up to 120 runes.
// Confidence and CurrentIteration are extracted from CurrentState when present.
func toSessionListItem(sess Session) SessionListItem {
	idea := sess.Idea
	if runes := []rune(idea); len(runes) > 120 {
		idea = string(runes[:120])
	}

	item := SessionListItem{
		ID:            sess.ID,
		Idea:          idea,
		Status:        sess.Status,
		MaxIterations: sess.MaxIterations,
		CreatedAt:     sess.CreatedAt,
		UpdatedAt:     sess.UpdatedAt,
		AgentCount:    sess.AgentCount,
	}

	if sess.CurrentState != nil {
		item.CurrentIteration = sess.CurrentState.Meta.Iteration
		item.Confidence = sess.CurrentState.Metrics.Confidence
	}
	return item
}

// FinalizeSession marks a session as approved. Called by the finalize endpoint;
// the markdown generation is triggered by the handler after this returns.
// If input.OutputDocs is non-nil, it overrides the session's stored document
// selection before the status transition (the override is persisted).
//
// Readiness gate: the session's current canonical state must contain a
// non-empty idea. Architecture and execution_plan are not required —
// documents can be generated from partial state. Returns ErrStateNotReady
// wrapped with the human-readable reason.
func (s *Service) FinalizeSession(ctx context.Context, id string, input FinalizeInput) (Session, error) {
	// Load the current session first so we can run the readiness gate before
	// mutating any state.
	current, err := s.repo.GetSession(ctx, id)
	if err != nil {
		return Session{}, fmt.Errorf("finalize session: load: %w", err)
	}
	if current.CurrentState == nil {
		return Session{}, fmt.Errorf("%w: canonical state has not been produced yet", ErrStateNotReady)
	}

	// For already-approved sessions (regeneration path), skip the readiness
	// gate and status transition — the state was validated at first-finalize and
	// the user is simply re-running document generation with a different doc
	// selection. For all other statuses, run the gate as normal.
	if current.Status != StatusApproved {
		if ready, reason := isStateReadyForFinalize(*current.CurrentState); !ready {
			return Session{}, fmt.Errorf("%w: %s", ErrStateNotReady, reason)
		}
	}

	if len(input.OutputDocs) > 0 {
		// Return validation errors unwrapped so the handler's
		// isValidationError prefix check maps them to HTTP 400 instead of 500.
		if err := validateOutputDocs(input.OutputDocs); err != nil {
			return Session{}, err
		}
		if err := s.repo.UpdateOutputDocs(ctx, id, input.OutputDocs); err != nil {
			return Session{}, fmt.Errorf("finalize session: update output docs: %w", err)
		}
	}

	if current.Status != StatusApproved {
		if err := s.repo.UpdateStatus(ctx, id, StatusApproved); err != nil {
			return Session{}, fmt.Errorf("finalize session: %w", err)
		}
	}

	sess, err := s.repo.GetSession(ctx, id)
	if err != nil {
		return Session{}, fmt.Errorf("finalize session: reload: %w", err)
	}
	s.logger.InfoContext(ctx, "session finalized", slog.String("session_id", id))

	// Emit SSE event so connected clients know the session is approved.
	s.emitter.Emit(id, "session.finalized", map[string]any{
		"documents": sess.OutputDocs,
	})

	return sess, nil
}

// UpdateOutputDocs replaces the output_docs for a session.
// Validation: len ≥ 1, all keys in AllowedOutputDocs, no duplicates.
// Returns ErrConflict if the session is already in the approved state.
func (s *Service) UpdateOutputDocs(ctx context.Context, id string, docs []string) error {
	if err := validateOutputDocs(docs); err != nil {
		return err
	}

	// Guard: reject updates on non-active sessions. UpdateOutputDocs is only
	// permitted while the session is in StatusActive — allowing changes during
	// "running" risks mid-iteration document-selection drift, and "approved"
	// is terminal for this field.
	sess, err := s.repo.GetSession(ctx, id)
	if err != nil {
		return fmt.Errorf("update output docs: %w", err)
	}
	if sess.Status != StatusActive {
		return ErrConflict
	}

	if err := s.repo.UpdateOutputDocs(ctx, id, docs); err != nil {
		return fmt.Errorf("update output docs: %w", err)
	}
	s.logger.InfoContext(ctx, "output docs updated",
		slog.String("session_id", id),
		slog.Any("docs", docs),
	)
	return nil
}

// validateOutputDocs enforces the shared rules:
//   - at least one key
//   - every key is in AllowedOutputDocs
//   - no duplicate keys
func validateOutputDocs(docs []string) error {
	if len(docs) == 0 {
		return errors.New("output_docs must contain at least one document key")
	}
	seen := make(map[string]struct{}, len(docs))
	for _, key := range docs {
		if !AllowedOutputDocs[key] {
			return fmt.Errorf("invalid output doc key %q: must be one of architecture, roadmap, plan, readme", key)
		}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate output doc key %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// Package iteration provides the service layer for triggering the N-agent
// iteration pipeline against a brainstorm session.
package iteration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"a2a-brainstorm/backend/internal/modules/session"
	"a2a-brainstorm/backend/internal/modules/state"
)

// Sentinel errors returned by service methods.
var (
	// ErrSessionTerminal is returned by TriggerIteration when the session is
	// already in a terminal state that prevents further iteration (approved).
	ErrSessionTerminal = errors.New("session is in a terminal state")

	// ErrIterationInFlight is returned when a concurrent Iterate, Preview, or
	// Apply call is already holding the per-session lock.
	ErrIterationInFlight = errors.New("iteration is already in flight for this session")

	// ErrPreviewNotFound is returned by Apply when no preview exists for the
	// given session/agent pair.
	ErrPreviewNotFound = errors.New("no preview found for this agent")

	// ErrPreviewIDMismatch is returned by Apply when the caller supplies a
	// preview_id that does not match the stored preview.
	ErrPreviewIDMismatch = errors.New("preview_id does not match the stored preview")

	// ErrAgentNotInSession is returned by Preview when the supplied agentID is
	// not a member of the session's agent roster. Mapped to HTTP 409 by the
	// handler per §8.21.
	ErrAgentNotInSession = errors.New("agent is not a member of the session")
)

// sessionProvider is the service-layer interface for reading session data.
// Satisfied by *session.Service in production.
type sessionProvider interface {
	GetSession(ctx context.Context, id string) (session.Session, error)
}

// stateWriter is the service-layer interface for persisting canonical state.
// Satisfied by *session.Repository in production.
type stateWriter interface {
	UpdateState(ctx context.Context, id string, cs *state.CanonicalState) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

// sessionLockMap is an append-only map of per-session mutexes.
// It allows TryLock-based concurrency control without a global lock on
// the whole service — each session is independently protected.
type sessionLockMap struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func newSessionLockMap() *sessionLockMap {
	return &sessionLockMap{locks: make(map[string]*sync.Mutex)}
}

// getLock returns the mutex for the given sessionID, creating it on first use.
func (m *sessionLockMap) getLock(id string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	lock, ok := m.locks[id]
	if !ok {
		lock = new(sync.Mutex)
		m.locks[id] = lock
	}
	return lock
}

// Service orchestrates triggering a full iteration run for a brainstorm session.
type Service struct {
	engine       *Engine
	sessions     sessionProvider
	store        stateWriter
	previews     *PreviewStore
	sessionLocks *sessionLockMap
	logger       *slog.Logger
}

// NewService constructs an iteration Service.
//
// store must satisfy stateWriter — in production pass *session.Repository.
func NewService(engine *Engine, sessions sessionProvider, store stateWriter, logger *slog.Logger) *Service {
	return &Service{
		engine:       engine,
		sessions:     sessions,
		store:        store,
		previews:     NewPreviewStore(),
		sessionLocks: newSessionLockMap(),
		logger:       logger,
	}
}

// IterationResult is returned by TriggerIteration and matches the
// IterateResponse JSON shape expected by the frontend (§8.7 of docs/PLAN.md).
type IterationResult struct {
	SessionID string               `json:"session_id"`
	Iteration int                  `json:"iteration"`
	State     state.CanonicalState `json:"state"`
	Converged bool                 `json:"converged"`
}

// PreviewResponse is returned by Preview — matches §8.21 API contract.
type PreviewResponse struct {
	SessionID string               `json:"session_id"`
	AgentID   string               `json:"agent_id"`
	PreviewID string               `json:"preview_id"`
	Output    state.CanonicalState `json:"output"`
	CreatedAt string               `json:"created_at"`
}

// TriggerIteration loads the session, seeds or continues the canonical state,
// and runs the full iteration engine loop (which repeats until quality
// convergence or the maxIter cap). It returns an IterationResult containing
// the final CanonicalState and convergence status.
//
// Returns ErrSessionTerminal if the session status is "approved".
// Returns ErrIterationInFlight if another operation holds the session lock.
// Returns a wrapped session.ErrNotFound if the session does not exist.
func (s *Service) TriggerIteration(ctx context.Context, sessionID string) (IterationResult, error) {
	lock := s.sessionLocks.getLock(sessionID)
	if !lock.TryLock() {
		return IterationResult{}, ErrIterationInFlight
	}
	defer lock.Unlock()

	s.logger.InfoContext(ctx, "iteration trigger received",
		slog.String("session_id", sessionID),
	)

	sess, err := s.sessions.GetSession(ctx, sessionID)
	if err != nil {
		return IterationResult{}, fmt.Errorf("trigger iteration: load session %s: %w", sessionID, err)
	}

	s.logger.InfoContext(ctx, "session loaded for iteration",
		slog.String("session_id", sessionID),
		slog.String("status", sess.Status),
		slog.Int("agent_count", len(sess.Agents)),
		slog.Int("max_iterations", sess.MaxIterations),
	)

	// Guard: do not re-trigger on explicitly approved sessions.
	if sess.Status == session.StatusApproved {
		return IterationResult{}, fmt.Errorf("trigger iteration: session %s is already approved: %w",
			sessionID, ErrSessionTerminal)
	}

	// Seed the initial state from existing progress or the session idea.
	initial := state.CanonicalState{}
	if sess.CurrentState != nil {
		initial = *sess.CurrentState
	} else {
		// First run: populate the idea field so agents have context.
		initial.Idea = map[string]any{"text": sess.Idea}
	}

	// Ensure Meta.Agents carries observability info from session agents if not
	// yet populated. Agent names/providers are filled in by agent.Dispatch; we
	// seed the IDs and roles here so downstream observers see the roster early.
	if len(initial.Meta.Agents) == 0 && len(sess.Agents) > 0 {
		initial.Meta.Agents = make([]state.AgentMeta, 0, len(sess.Agents))
		for _, sa := range sess.Agents {
			initial.Meta.Agents = append(initial.Meta.Agents, state.AgentMeta{
				AgentID: sa.AgentID,
				Role:    sa.Role,
			})
		}
	}

	// Invalidate stale previews for this session — a full run supersedes all
	// single-agent previews.
	s.previews.Clear(sessionID)

	// Mark session as running so clients that reload can detect the in-flight
	// iteration and enter watching mode instead of triggering a duplicate run.
	if statusErr := s.store.UpdateStatus(ctx, sessionID, session.StatusRunning); statusErr != nil {
		s.logger.WarnContext(ctx, "failed to set session status to running",
			slog.String("session_id", sessionID),
			slog.String("error", statusErr.Error()),
		)
	}

	result, err := s.engine.Run(ctx, sess, initial)
	if err != nil {
		// Reset to active so the session can be retried.
		_ = s.store.UpdateStatus(context.Background(), sessionID, session.StatusActive)
		return IterationResult{}, fmt.Errorf("trigger iteration: run engine: %w", err)
	}

	s.logger.InfoContext(ctx, "iteration completed",
		slog.String("session_id", sessionID),
		slog.Int("iteration", result.Meta.Iteration),
		slog.Float64("confidence", result.Metrics.Confidence),
	)

	// The engine always marks the session "converged" on a successful run
	// (either by quality convergence or by exhausting max iterations).
	return IterationResult{
		SessionID: sessionID,
		Iteration: result.Meta.Iteration,
		State:     result,
		Converged: true,
	}, nil
}

// Preview dispatches a single agent against the session's current canonical
// state and stores the result in the in-memory preview store. The result is
// NOT persisted or merged into the session state — call Apply to commit.
//
// Returns ErrIterationInFlight if TriggerIteration, Preview, or Apply is
// already in progress for this session. The caller receives HTTP 409.
// Returns a wrapped session.ErrNotFound if the session does not exist.
// Returns an error if agentID is not a member of the session.
func (s *Service) Preview(ctx context.Context, sessionID, agentID string) (PreviewResponse, error) {
	lock := s.sessionLocks.getLock(sessionID)
	if !lock.TryLock() {
		return PreviewResponse{}, ErrIterationInFlight
	}
	defer lock.Unlock()

	sess, err := s.sessions.GetSession(ctx, sessionID)
	if err != nil {
		return PreviewResponse{}, fmt.Errorf("preview: load session %s: %w", sessionID, err)
	}

	current := state.CanonicalState{}
	if sess.CurrentState != nil {
		current = *sess.CurrentState
	} else {
		current.Idea = map[string]any{"text": sess.Idea}
	}

	out, err := s.engine.RunSingleAgent(ctx, sess, current, agentID)
	if err != nil {
		return PreviewResponse{}, fmt.Errorf("preview: run single agent: %w", err)
	}

	result := PreviewResult{
		PreviewID: uuid.New().String(),
		AgentID:   agentID,
		Output:    out,
		CreatedAt: time.Now().UTC(),
	}
	s.previews.Set(sessionID, agentID, result)

	s.logger.InfoContext(ctx, "preview stored",
		slog.String("session_id", sessionID),
		slog.String("agent_id", agentID),
		slog.String("preview_id", result.PreviewID),
	)

	return PreviewResponse{
		SessionID: sessionID,
		AgentID:   agentID,
		PreviewID: result.PreviewID,
		Output:    result.Output,
		CreatedAt: result.CreatedAt.Format(time.RFC3339),
	}, nil
}

// Apply merges a stored preview into the session's live canonical state and
// persists it. The in-memory preview is cleared after a successful apply.
//
// previewID is optional. When non-empty, it must match the stored preview's ID
// (optimistic concurrency guard); on mismatch HTTP 412 is returned. When
// empty the guard is skipped.
//
// Returns ErrIterationInFlight if another operation holds the session lock.
// Returns ErrPreviewNotFound if no preview exists for the session/agent pair.
// Returns ErrPreviewIDMismatch if previewID is supplied and does not match.
func (s *Service) Apply(ctx context.Context, sessionID, agentID, previewID string) (state.CanonicalState, error) {
	lock := s.sessionLocks.getLock(sessionID)
	if !lock.TryLock() {
		return state.CanonicalState{}, ErrIterationInFlight
	}
	defer lock.Unlock()

	stored, ok := s.previews.Get(sessionID, agentID)
	if !ok {
		return state.CanonicalState{}, ErrPreviewNotFound
	}

	if previewID != "" && previewID != stored.PreviewID {
		return state.CanonicalState{}, ErrPreviewIDMismatch
	}

	sess, err := s.sessions.GetSession(ctx, sessionID)
	if err != nil {
		return state.CanonicalState{}, fmt.Errorf("apply: load session %s: %w", sessionID, err)
	}

	current := state.CanonicalState{}
	if sess.CurrentState != nil {
		current = *sess.CurrentState
	} else {
		current.Idea = map[string]any{"text": sess.Idea}
	}

	// Merge the preview output into the live state and increment the counter.
	merged := state.Merge(current, stored.Output)
	merged.Meta.Iteration = current.Meta.Iteration + 1

	if err := s.store.UpdateState(ctx, sessionID, &merged); err != nil {
		return state.CanonicalState{}, fmt.Errorf("apply: persist state: %w", err)
	}

	// Clear the now-applied preview.
	s.previews.Delete(sessionID, agentID)

	s.logger.InfoContext(ctx, "preview applied",
		slog.String("session_id", sessionID),
		slog.String("agent_id", agentID),
		slog.Int("iteration", merged.Meta.Iteration),
	)

	return merged, nil
}

// DiscardPreview removes any stored preview for the given session/agent pair.
// It is idempotent — calling it when no preview exists is not an error.
// Does NOT require the session lock because it only writes to the in-memory
// preview store, not to persistent state.
func (s *Service) DiscardPreview(ctx context.Context, sessionID, agentID string) error {
	s.previews.Delete(sessionID, agentID)
	s.logger.InfoContext(ctx, "preview discarded",
		slog.String("session_id", sessionID),
		slog.String("agent_id", agentID),
	)
	return nil
}

// CheckSessionExists validates that a session with the given ID exists.
// Returns session.ErrNotFound (wrapped) when the session does not exist.
// Used by the SSE handler to reject subscriptions for non-existent sessions.
func (s *Service) CheckSessionExists(ctx context.Context, sessionID string) error {
	_, err := s.sessions.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("check session exists: %w", err)
	}
	return nil
}

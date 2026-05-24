// Package iteration provides the service layer for triggering the N-agent
// iteration pipeline against a brainstorm session.
package iteration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"a2a-brainstorm/backend/internal/modules/session"
	"a2a-brainstorm/backend/internal/modules/state"
)

// ErrSessionTerminal is returned by TriggerIteration when the session is
// already in a terminal state that prevents further iteration (approved).
var ErrSessionTerminal = errors.New("session is in a terminal state")

// sessionProvider is the service-layer interface for reading session data.
// Satisfied by *session.Service in production.
type sessionProvider interface {
	GetSession(ctx context.Context, id string) (session.Session, error)
}

// Service orchestrates triggering a full iteration run for a brainstorm session.
type Service struct {
	engine   *Engine
	sessions sessionProvider
	logger   *slog.Logger
}

// NewService constructs an iteration Service.
func NewService(engine *Engine, sessions sessionProvider, logger *slog.Logger) *Service {
	return &Service{
		engine:   engine,
		sessions: sessions,
		logger:   logger,
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

// TriggerIteration loads the session, seeds or continues the canonical state,
// and runs the full iteration engine loop (which repeats until quality
// convergence or the maxIter cap). It returns an IterationResult containing
// the final CanonicalState and convergence status.
//
// Returns ErrSessionTerminal if the session status is "approved".
// Returns a wrapped session.ErrNotFound if the session does not exist.
func (s *Service) TriggerIteration(ctx context.Context, sessionID string) (IterationResult, error) {
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

	result, err := s.engine.Run(ctx, sess, initial)
	if err != nil {
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

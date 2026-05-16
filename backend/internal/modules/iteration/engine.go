// Package iteration implements the deterministic N-agent iteration pipeline
// for the brainstorm system.
//
// The Engine drives the ordered pipeline defined in §8.4 of docs/PLAN.md:
// each iteration pass sends canonical state through every session agent in
// ascending position order, merges the pipeline output back into the state,
// then checks quality-based convergence. The loop repeats until convergence is
// detected (§8.6) or the session's max-iterations cap is reached.
//
// Roles are fixed at session creation and are NEVER modified by the engine.
package iteration

import (
	"context"
	"fmt"
	"log/slog"

	agentpkg "a2a-brainstorm/backend/internal/modules/agent"
	"a2a-brainstorm/backend/internal/modules/convergence"
	"a2a-brainstorm/backend/internal/modules/session"
	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/config"
	"a2a-brainstorm/backend/internal/platform/llm"
)

// DispatchFunc is the function signature used to send canonical state to an
// agent over A2A and receive the updated state.
//
// Using a function type rather than an interface keeps the engine lean and
// makes test injection trivial: tests pass a closure; production passes
// agentpkg.Dispatch directly.
type DispatchFunc func(
	ctx context.Context,
	ag agentpkg.Agent,
	role agentpkg.Role,
	activeSkills []agentpkg.Skill,
	llmOverride *llm.LLMConfig,
	current state.CanonicalState,
) (state.CanonicalState, error)

// agentProvider is the iteration engine's narrow view of the agent domain.
// Satisfied by *agentpkg.Service in production.
type agentProvider interface {
	GetAgent(ctx context.Context, id string) (agentpkg.Agent, error)
	ResolveActiveSkills(ctx context.Context, agentID string, overrides *[]string) ([]agentpkg.Skill, error)
}

// sessionStore is the iteration engine's narrow persistence interface.
// Satisfied by *session.Repository in production; the interface is kept
// minimal so tests can use a trivial in-memory stub.
type sessionStore interface {
	UpdateState(ctx context.Context, id string, cs *state.CanonicalState) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

// Engine executes the ordered N-agent iteration pipeline.
type Engine struct {
	dispatch DispatchFunc
	agents   agentProvider
	store    sessionStore
	logger   *slog.Logger
}

// NewEngine constructs an Engine with the given dependencies.
//
// dispatch must be agentpkg.Dispatch in production. It is a parameter so that
// tests can inject a closure without requiring a live A2A endpoint.
func NewEngine(dispatch DispatchFunc, agents agentProvider, store sessionStore, logger *slog.Logger) *Engine {
	return &Engine{
		dispatch: dispatch,
		agents:   agents,
		store:    store,
		logger:   logger,
	}
}

// Run executes the full iteration loop for the given session, starting from
// initialState, and returns the final CanonicalState regardless of the stop
// reason (convergence, maxIter cap, or a fatal dispatch error).
//
// Algorithm (§8.4 of docs/PLAN.md):
//
//  1. For i = 1 … maxIter:
//     a. Pass state through all session agents in ascending Position order.
//     Each agent receives the cumulative output of the previous agent.
//     b. Merge pipeline output with the pre-pass state (state.Merge).
//     c. Set Meta.Iteration = i on the merged state.
//     d. Persist the merged state (sessionStore.UpdateState).
//     e. Evaluate convergence.Check(prev, merged). Break if converged.
//  2. On convergence or maxIter: update session status to "converged".
//
// Roles are read from sess.Agents[i].Role and are NEVER modified here.
func (e *Engine) Run(ctx context.Context, sess session.Session, initialState state.CanonicalState) (state.CanonicalState, error) {
	if len(sess.Agents) < 2 {
		return initialState, fmt.Errorf("iteration engine: session %s requires at least 2 agents, got %d",
			sess.ID, len(sess.Agents))
	}

	maxIter := sess.MaxIterations
	if maxIter <= 0 {
		maxIter = config.GetMaxIterations()
	}

	current := initialState

	for i := 1; i <= maxIter; i++ {
		pipelineOut, err := e.runPipelinePass(ctx, sess, current, i)
		if err != nil {
			return current, fmt.Errorf("iteration %d: pipeline pass: %w", i, err)
		}

		// Merge pipeline output with the pre-pass state (§8.5).
		merged := state.Merge(current, pipelineOut)
		merged.Meta.Iteration = i

		// Persist after each full pass — not per-agent within a pass (§8.4).
		if err := e.store.UpdateState(ctx, sess.ID, &merged); err != nil {
			return merged, fmt.Errorf("iteration %d: persist state: %w", i, err)
		}

		e.logger.InfoContext(ctx, "iteration pass complete",
			slog.String("session_id", sess.ID),
			slog.Int("iteration", i),
			slog.Float64("confidence", merged.Metrics.Confidence),
		)

		// Quality convergence check (§8.6 conditions 1–3).
		if convergence.Check(current, merged) {
			e.logger.InfoContext(ctx, "convergence detected",
				slog.String("session_id", sess.ID),
				slog.Int("iteration", i),
			)
			if err := e.store.UpdateStatus(ctx, sess.ID, session.StatusConverged); err != nil {
				// Log the failure but do not mask the successful convergence result.
				e.logger.WarnContext(ctx, "failed to update session status to converged",
					slog.String("session_id", sess.ID),
					slog.String("error", err.Error()),
				)
			}
			return merged, nil
		}

		current = merged
	}

	// Max-iterations cap reached (§8.6 condition 5). Transition to "converged"
	// so the user can still review and approve the final state.
	e.logger.InfoContext(ctx, "max iterations reached without quality convergence",
		slog.String("session_id", sess.ID),
		slog.Int("max_iterations", maxIter),
	)
	if err := e.store.UpdateStatus(ctx, sess.ID, session.StatusConverged); err != nil {
		e.logger.WarnContext(ctx, "failed to update session status after max iterations",
			slog.String("session_id", sess.ID),
			slog.String("error", err.Error()),
		)
	}
	return current, nil
}

// runPipelinePass executes one ordered pass through all session agents.
// Each agent in the pipeline receives the cumulative output of the previous
// agent (§8.4: "each agent receives the output of the previous").
func (e *Engine) runPipelinePass(
	ctx context.Context,
	sess session.Session,
	initial state.CanonicalState,
	iterNum int,
) (state.CanonicalState, error) {
	current := initial

	for _, sa := range sess.Agents {
		ag, err := e.agents.GetAgent(ctx, sa.AgentID)
		if err != nil {
			return current, fmt.Errorf("get agent %s: %w", sa.AgentID, err)
		}

		activeSkills, err := e.agents.ResolveActiveSkills(ctx, sa.AgentID, sa.SkillOverrides)
		if err != nil {
			return current, fmt.Errorf("resolve skills for agent %s: %w", sa.AgentID, err)
		}

		out, err := e.dispatch(ctx, ag, agentpkg.Role(sa.Role), activeSkills, sa.LLMOverride, current)
		if err != nil {
			return current, fmt.Errorf("dispatch agent %s (iter %d): %w", sa.AgentID, iterNum, err)
		}

		e.logger.DebugContext(ctx, "agent pass complete",
			slog.String("session_id", sess.ID),
			slog.String("agent_id", sa.AgentID),
			slog.String("role", sa.Role),
			slog.Int("iteration", iterNum),
		)

		current = out
	}

	return current, nil
}

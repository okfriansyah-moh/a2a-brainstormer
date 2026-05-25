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
	"time"

	agentpkg "a2a-brainstorm/backend/internal/modules/agent"
	"a2a-brainstorm/backend/internal/modules/convergence"
	"a2a-brainstorm/backend/internal/modules/session"
	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/config"
	"a2a-brainstorm/backend/internal/platform/llm"
	"a2a-brainstorm/backend/internal/platform/sse"
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
	emitter  sse.EventEmitter
	logger   *slog.Logger
}

// NewEngine constructs an Engine with the given dependencies.
//
// dispatch must be agentpkg.Dispatch in production. It is a parameter so that
// tests can inject a closure without requiring a live A2A endpoint.
//
// emitter receives SSE lifecycle events. Pass sse.NoopEmitter{} in tests or
// when SSE is not required.
func NewEngine(dispatch DispatchFunc, agents agentProvider, store sessionStore, emitter sse.EventEmitter, logger *slog.Logger) *Engine {
	if emitter == nil {
		emitter = sse.NoopEmitter{}
	}
	return &Engine{
		dispatch: dispatch,
		agents:   agents,
		store:    store,
		emitter:  emitter,
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

	e.logger.InfoContext(ctx, "pipeline starting",
		slog.String("session_id", sess.ID),
		slog.Int("agent_count", len(sess.Agents)),
		slog.Int("max_iterations", maxIter),
		slog.Int("resume_from_iteration", initialState.Meta.Iteration),
	)

	current := initialState

	for i := 1; i <= maxIter; i++ {
		// Build the agents list for the iteration.start event.
		agentMetas := make([]map[string]any, len(sess.Agents))
		for j, sa := range sess.Agents {
			agentMetas[j] = map[string]any{
				"agent_id": sa.AgentID,
				"role":     sa.Role,
				"position": sa.Position,
			}
		}
		e.emitter.Emit(sess.ID, EventIterationStart, map[string]any{
			"iteration": i,
			"agents":    agentMetas,
		})

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
			slog.Int("execution_plan_steps", len(merged.ExecutionPlan)),
			slog.Int("risks_count", len(merged.Risks)),
			slog.Int("open_questions_count", len(merged.OpenQuestions)),
		)

		converged := convergence.Check(current, merged)
		e.emitter.Emit(sess.ID, EventIterationComplete, map[string]any{
			"iteration":  i,
			"converged":  converged,
			"confidence": merged.Metrics.Confidence,
			"state":      merged, // embed state so the frontend updates in real-time
		})

		// Quality convergence check (§8.6 conditions 1–3).
		if converged {
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
//
// The backend is authoritative for Meta.Agents — the LLM must never own it.
// The roster is built from live agent data as we iterate and is force-applied
// to the state both before each dispatch (so the LLM sees correct data) and
// after each dispatch (to prevent LLM drift).
func (e *Engine) runPipelinePass(
	ctx context.Context,
	sess session.Session,
	initial state.CanonicalState,
	iterNum int,
) (state.CanonicalState, error) {
	current := initial

	// roster accumulates authoritative AgentMeta entries as we fetch each agent.
	roster := make([]state.AgentMeta, 0, len(sess.Agents))

	for _, sa := range sess.Agents {
		ag, err := e.agents.GetAgent(ctx, sa.AgentID)
		if err != nil {
			return current, fmt.Errorf("get agent %s: %w", sa.AgentID, err)
		}

		activeSkills, err := e.agents.ResolveActiveSkills(ctx, sa.AgentID, sa.SkillOverrides)
		if err != nil {
			return current, fmt.Errorf("resolve skills for agent %s: %w", sa.AgentID, err)
		}

		// Resolve effective provider/model for observability, mirroring the
		// tiered priority used by Dispatch (session override → agent → global).
		provider, model := resolveProviderModel(ag, sa.LLMOverride)

		// Build skill-name list for the observability record.
		skillNames := make([]string, len(activeSkills))
		for i, sk := range activeSkills {
			skillNames[i] = sk.Name
		}

		roster = append(roster, state.AgentMeta{
			AgentID:  sa.AgentID,
			Name:     ag.Name,
			Role:     sa.Role,
			Provider: provider,
			Model:    model,
			Skills:   skillNames,
		})

		// Inject the authoritative roster into the state before dispatch so
		// the LLM receives correct meta context.
		current.Meta.Agents = cloneAgentMetas(roster)

		e.logger.InfoContext(ctx, "dispatching to agent",
			slog.String("session_id", sess.ID),
			slog.String("agent_id", sa.AgentID),
			slog.String("agent_name", ag.Name),
			slog.String("role", sa.Role),
			slog.Int("iteration", iterNum),
			slog.Int("skill_count", len(activeSkills)),
		)

		e.emitter.Emit(sess.ID, EventAgentStarted, map[string]any{
			"iteration": iterNum,
			"agent_id":  sa.AgentID,
			"role":      sa.Role,
			"position":  sa.Position,
		})

		confBefore := current.Metrics.Confidence
		dispatchStart := time.Now()
		out, err := e.dispatch(ctx, ag, agentpkg.Role(sa.Role), activeSkills, sa.LLMOverride, current)
		if err != nil {
			e.emitter.Emit(sess.ID, EventAgentError, map[string]any{
				"iteration": iterNum,
				"agent_id":  sa.AgentID,
				"error":     err.Error(),
			})
			return current, fmt.Errorf("dispatch agent %s (iter %d): %w", sa.AgentID, iterNum, err)
		}

		// Force-overwrite meta.agents in the returned state — the LLM must
		// not be the source of truth for agent observability data.
		out.Meta.Agents = cloneAgentMetas(roster)

		confAfter := out.Metrics.Confidence
		e.logger.InfoContext(ctx, "agent pass complete",
			slog.String("session_id", sess.ID),
			slog.String("agent_id", sa.AgentID),
			slog.String("agent_name", ag.Name),
			slog.String("role", sa.Role),
			slog.Int("iteration", iterNum),
			slog.Int64("duration_ms", time.Since(dispatchStart).Milliseconds()),
			slog.Float64("confidence_before", confBefore),
			slog.Float64("confidence_after", confAfter),
			slog.Float64("confidence_delta", confAfter-confBefore),
			slog.Int("execution_plan_steps", len(out.ExecutionPlan)),
			slog.Int("risks_count", len(out.Risks)),
			slog.Int("open_questions_count", len(out.OpenQuestions)),
		)

		e.emitter.Emit(sess.ID, EventAgentComplete, map[string]any{
			"iteration":        iterNum,
			"agent_id":         sa.AgentID,
			"confidence_delta": confAfter - confBefore,
			"output":           out, // included so the frontend can render per-agent output
		})

		current = out
	}

	return current, nil
}

// resolveProviderModel returns the effective provider and model for a dispatch,
// mirroring the priority order of llm.Resolve: session override → agent-level.
// Global defaults are not checked here because this function is for
// observability only — the actual LLM call uses the full llm.Resolve chain.
func resolveProviderModel(ag agentpkg.Agent, sessionOverride *llm.LLMConfig) (provider, model string) {
	if sessionOverride != nil {
		if sessionOverride.Provider != "" {
			provider = sessionOverride.Provider
		}
		if sessionOverride.Model != "" {
			model = sessionOverride.Model
		}
	}
	if provider == "" && ag.LLMConfig != nil {
		provider = ag.LLMConfig.Provider
	}
	if model == "" && ag.LLMConfig != nil {
		model = ag.LLMConfig.Model
	}
	return provider, model
}

// RunSingleAgent dispatches one specific agent against currentState and
// returns the agent's output WITHOUT merging or persisting it. It is the
// compute step behind the Preview endpoint (§8.21 of docs/PLAN.md).
//
// The caller is responsible for holding the per-session mutex before invoking
// this method and releasing it afterwards — RunSingleAgent itself does not
// acquire any lock.
//
// Returns an error if agentID is not a member of sess.Agents.
func (e *Engine) RunSingleAgent(
	ctx context.Context,
	sess session.Session,
	currentState state.CanonicalState,
	agentID string,
) (state.CanonicalState, error) {
	// Find the session-agent slot for this agentID.
	var sa session.SessionAgent
	found := false
	for _, slot := range sess.Agents {
		if slot.AgentID == agentID {
			sa = slot
			found = true
			break
		}
	}
	if !found {
		return currentState, fmt.Errorf("run single agent: agent %s is not a member of session %s",
			agentID, sess.ID)
	}

	ag, err := e.agents.GetAgent(ctx, sa.AgentID)
	if err != nil {
		return currentState, fmt.Errorf("run single agent: get agent %s: %w", sa.AgentID, err)
	}

	activeSkills, err := e.agents.ResolveActiveSkills(ctx, sa.AgentID, sa.SkillOverrides)
	if err != nil {
		return currentState, fmt.Errorf("run single agent: resolve skills for agent %s: %w", sa.AgentID, err)
	}

	provider, model := resolveProviderModel(ag, sa.LLMOverride)
	skillNames := make([]string, len(activeSkills))
	for i, sk := range activeSkills {
		skillNames[i] = sk.Name
	}

	// Build a single-entry roster so the agent sees its own meta context.
	roster := []state.AgentMeta{{
		AgentID:  sa.AgentID,
		Name:     ag.Name,
		Role:     sa.Role,
		Provider: provider,
		Model:    model,
		Skills:   skillNames,
	}}
	currentState.Meta.Agents = cloneAgentMetas(roster)

	e.logger.InfoContext(ctx, "single-agent preview dispatch",
		slog.String("session_id", sess.ID),
		slog.String("agent_id", sa.AgentID),
		slog.String("agent_name", ag.Name),
		slog.String("role", sa.Role),
	)

	e.emitter.Emit(sess.ID, EventAgentStarted, map[string]any{
		"iteration": currentState.Meta.Iteration,
		"agent_id":  sa.AgentID,
		"role":      sa.Role,
		"position":  sa.Position,
	})

	confBefore := currentState.Metrics.Confidence
	out, err := e.dispatch(ctx, ag, agentpkg.Role(sa.Role), activeSkills, sa.LLMOverride, currentState)
	if err != nil {
		e.emitter.Emit(sess.ID, EventAgentError, map[string]any{
			"iteration": currentState.Meta.Iteration,
			"agent_id":  sa.AgentID,
			"error":     err.Error(),
		})
		return currentState, fmt.Errorf("run single agent: dispatch agent %s: %w", sa.AgentID, err)
	}

	// Force-overwrite meta.agents — the LLM must not be the source of truth.
	out.Meta.Agents = cloneAgentMetas(roster)

	e.emitter.Emit(sess.ID, EventAgentComplete, map[string]any{
		"iteration":        currentState.Meta.Iteration,
		"agent_id":         sa.AgentID,
		"confidence_delta": out.Metrics.Confidence - confBefore,
	})

	return out, nil
}

// cloneAgentMetas returns a deep copy of the slice so mutations to the
// original roster do not affect the state embedded in any prior snapshot.
func cloneAgentMetas(src []state.AgentMeta) []state.AgentMeta {
	out := make([]state.AgentMeta, len(src))
	for i, m := range src {
		out[i] = m
		if m.Skills != nil {
			skills := make([]string, len(m.Skills))
			copy(skills, m.Skills)
			out[i].Skills = skills
		}
	}
	return out
}

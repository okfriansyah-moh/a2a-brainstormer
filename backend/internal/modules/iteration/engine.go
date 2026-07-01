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
	"a2a-brainstorm/backend/internal/platform/ctxidle"
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
	userFeedback string,
	dispatchCtx agentpkg.DispatchContext,
) (state.CanonicalState, error)

// StreamDispatchFunc extends DispatchFunc with a tokenFn callback that is
// called for each text token the agent emits during its Working phase.
// Production code sets this via Engine.SetStreamDispatch; tests leave it nil
// so DispatchFunc is used instead (no signature change to DispatchFunc).
type StreamDispatchFunc func(
	ctx context.Context,
	ag agentpkg.Agent,
	role agentpkg.Role,
	activeSkills []agentpkg.Skill,
	llmOverride *llm.LLMConfig,
	current state.CanonicalState,
	userFeedback string,
	dispatchCtx agentpkg.DispatchContext,
	tokenFn func(token string),
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
	GetStatus(ctx context.Context, id string) (string, error)
	UpdateState(ctx context.Context, id string, cs *state.CanonicalState) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

// Engine executes the ordered N-agent iteration pipeline.
type Engine struct {
	dispatch       DispatchFunc
	streamDispatch StreamDispatchFunc // optional; set via SetStreamDispatch
	agents         agentProvider
	store          sessionStore
	emitter        sse.EventEmitter
	logger         *slog.Logger
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

// SetStreamDispatch registers a streaming dispatch function. When set, the
// engine uses streamDispatch instead of dispatch so that per-token SSE events
// are forwarded to the browser. Call this once during server initialisation
// after NewEngine returns.
func (e *Engine) SetStreamDispatch(fn StreamDispatchFunc) {
	e.streamDispatch = fn
}

// Run executes exactly one iteration pass for the given session, starting from
// initialState, and returns the merged CanonicalState plus whether the session
// has reached a terminal state (quality convergence or max-iterations cap).
//
// Each POST /sessions/{id}/iterate triggers one call to Run. The next pass
// number is initialState.Meta.Iteration + 1. When the pass completes without
// terminal convergence the session status is reset to "active" so the user can
// review results and trigger the next pass manually.
//
// Algorithm (§8.4 of docs/PLAN.md) — single pass per invocation:
//  1. Pass state through all session agents in ascending Position order.
//  2. Merge pipeline output with the pre-pass state (state.Merge).
//  3. Set Meta.Iteration = pass number on the merged state.
//  4. Persist the merged state (sessionStore.UpdateState).
//  5. Evaluate convergence.Check(prev, merged) and max-iterations cap.
//
// Roles are read from sess.Agents[i].Role and are NEVER modified here.
func (e *Engine) Run(ctx context.Context, sess session.Session, initialState state.CanonicalState, userFeedback string) (state.CanonicalState, bool, error) {
	if len(sess.Agents) < 2 {
		return initialState, false, fmt.Errorf("iteration engine: session %s requires at least 2 agents, got %d",
			sess.ID, len(sess.Agents))
	}

	maxIter := sess.MaxIterations
	if maxIter <= 0 {
		maxIter = config.GetMaxIterations()
	}

	current := state.Compact(initialState)
	passNum := current.Meta.Iteration + 1
	if passNum < 1 {
		passNum = 1
	}
	if passNum > maxIter {
		return current, true, fmt.Errorf("iteration engine: session %s already reached max iterations (%d)",
			sess.ID, maxIter)
	}

	e.logger.InfoContext(ctx, "pipeline pass starting",
		slog.String("session_id", sess.ID),
		slog.Int("agent_count", len(sess.Agents)),
		slog.Int("max_iterations", maxIter),
		slog.Int("pass", passNum),
		slog.Int("completed_passes", current.Meta.Iteration),
	)

	// Check if the session was finalized (approved) from another request.
	statusCtx, statusCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	liveStatus, statusErr := e.store.GetStatus(statusCtx, sess.ID)
	statusCancel()
	if statusErr == nil && liveStatus == session.StatusApproved {
		e.logger.InfoContext(ctx, "iteration aborted: session approved mid-run",
			slog.String("session_id", sess.ID),
			slog.Int("aborted_at_iteration", passNum),
		)
		return current, true, nil
	}

	agentMetas := make([]map[string]any, len(sess.Agents))
	for j, sa := range sess.Agents {
		agentMetas[j] = map[string]any{
			"agent_id": sa.AgentID,
			"role":     sa.Role,
			"position": sa.Position,
		}
	}
	e.emitter.Emit(sess.ID, EventIterationStart, map[string]any{
		"iteration": passNum,
		"agents":    agentMetas,
	})

	pipelineOut, successCount, err := e.runPipelinePass(ctx, sess, current, passNum, userFeedback)
	if err != nil {
		return current, false, fmt.Errorf("iteration %d: pipeline pass: %w", passNum, err)
	}

	merged := state.Merge(current, pipelineOut)
	merged = state.Compact(merged)
	merged.Meta.Iteration = passNum

	persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	persistErr := e.store.UpdateState(persistCtx, sess.ID, &merged)
	persistCancel()
	if persistErr != nil {
		return merged, false, fmt.Errorf("iteration %d: persist state: %w", passNum, persistErr)
	}

	e.logger.InfoContext(ctx, "iteration pass complete",
		slog.String("session_id", sess.ID),
		slog.Int("iteration", passNum),
		slog.Float64("confidence", merged.Metrics.Confidence),
		slog.Int("execution_plan_steps", len(merged.ExecutionPlan)),
		slog.Int("risks_count", len(merged.Risks)),
		slog.Int("open_questions_count", len(merged.OpenQuestions)),
		slog.Int("agents_succeeded", successCount),
	)

	qualityConverged := convergence.Check(current, merged)
	atCap := passNum >= maxIter
	terminal := qualityConverged || atCap

	e.emitter.Emit(sess.ID, EventIterationComplete, map[string]any{
		"iteration":  passNum,
		"converged":  terminal,
		"confidence": merged.Metrics.Confidence,
		"state":      merged,
	})

	statusCtx2, statusCancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer statusCancel2()

	switch {
	case qualityConverged:
		e.logger.InfoContext(ctx, "convergence detected",
			slog.String("session_id", sess.ID),
			slog.Int("iteration", passNum),
		)
		if err := e.store.UpdateStatus(statusCtx2, sess.ID, session.StatusConverged); err != nil {
			e.logger.WarnContext(ctx, "failed to update session status to converged",
				slog.String("session_id", sess.ID),
				slog.String("error", err.Error()),
			)
		}
	case atCap:
		e.logger.InfoContext(ctx, "max iterations reached without quality convergence",
			slog.String("session_id", sess.ID),
			slog.Int("max_iterations", maxIter),
		)
		if err := e.store.UpdateStatus(statusCtx2, sess.ID, session.StatusConverged); err != nil {
			e.logger.WarnContext(ctx, "failed to update session status after max iterations",
				slog.String("session_id", sess.ID),
				slog.String("error", err.Error()),
			)
		}
	default:
		if successCount == 0 {
			e.logger.WarnContext(ctx, "pipeline pass finished with no successful agents",
				slog.String("session_id", sess.ID),
				slog.Int("iteration", passNum),
			)
		}
		if err := e.store.UpdateStatus(statusCtx2, sess.ID, session.StatusActive); err != nil {
			e.logger.WarnContext(ctx, "failed to reset session status to active",
				slog.String("session_id", sess.ID),
				slog.String("error", err.Error()),
			)
		}
	}

	return merged, terminal, nil
}

// runPipelinePass executes one ordered pass through all session agents.
// Each agent in the pipeline receives the cumulative output of the previous
// agent (§8.4: "each agent receives the output of the previous").
//
// The backend is authoritative for Meta.Agents — the LLM must never own it.
// The roster is built from live agent data as we iterate and is force-applied
// to the state both before each dispatch (so the LLM sees correct data) and
// after each dispatch (to prevent LLM drift).
//
// Agent dispatch errors (JSON parse failure, failed A2A task, unreachable LLM)
// abort the pass immediately after emitting EventAgentError so later agents do
// not run on stale or empty state. Fatal errors (agent not found, session
// misconfiguration) are still returned as errors.
// The returned int is the number of agents that dispatched successfully.
func (e *Engine) runPipelinePass(
	ctx context.Context,
	sess session.Session,
	initial state.CanonicalState,
	iterNum int,
	userFeedback string,
) (state.CanonicalState, int, error) {
	current := initial
	successCount := 0

	// roster accumulates authoritative AgentMeta entries as we fetch each agent.
	roster := make([]state.AgentMeta, 0, len(sess.Agents))

	for _, sa := range sess.Agents {
		// Use a short-lived context for DB/metadata lookups so that an
		// exhausted iterCtx budget (from long LLM calls) does not prevent
		// infrastructure calls from succeeding.
		dbCtx, dbCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		ag, err := e.agents.GetAgent(dbCtx, sa.AgentID)
		if err != nil {
			dbCancel()
			return current, successCount, fmt.Errorf("get agent %s: %w", sa.AgentID, err)
		}

		activeSkills, err := e.agents.ResolveActiveSkills(dbCtx, sa.AgentID, sa.SkillOverrides)
		dbCancel()
		if err != nil {
			return current, successCount, fmt.Errorf("resolve skills for agent %s: %w", sa.AgentID, err)
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
		e.emitter.Emit(sess.ID, EventAgentPhase, map[string]any{
			"iteration": iterNum,
			"agent_id":  sa.AgentID,
			"role":      sa.Role,
			"phase":     "generating",
			"detail":    "Waiting for model response…",
		})

		confBefore := current.Metrics.Confidence
		dispatchStart := time.Now()
		dispatchCtx := agentpkg.DispatchContext{
			SessionID:  sess.ID,
			AgentID:    sa.AgentID,
			OutputDocs: sess.OutputDocs,
		}
		// Give each agent call an independent per-call timeout that is NOT
		// inherited from the parent iterCtx budget. Without WithoutCancel, a
		// long first-agent LLM call exhausts iterCtx and leaves every subsequent
		// agent with a zero-or-negative deadline — causing immediate deadline
		// exceeded errors on iteration 2+.
		//
		// Trade-off: context.WithoutCancel also drops parent *cancellation*
		// signals (client disconnect, server shutdown). An in-flight agent call
		// will therefore not abort on those signals; it runs to its own
		// agentCallTimeout deadline. This is intentional: the per-agent timeout
		// (config.GetAgentCallTimeout) is the effective upper bound, and
		// interrupting a long LLM call mid-stream produces no useful output.
		agentCallTimeout := config.GetAgentCallTimeout()
		totalCtx, totalCancel := context.WithTimeout(context.WithoutCancel(ctx), agentCallTimeout)

		var out state.CanonicalState
		var agentCtx context.Context
		var agentCancel func()
		if e.streamDispatch != nil {
			// Streaming calls may run longer than a fixed HTTP client timeout when
			// tokens keep arriving. A startup deadline aborts stalled models that
			// never emit the first token; idle timeout arms on the first token.
			streamCtx, bumpIdle := ctxidle.WithStartupAndIdleTimeout(
				totalCtx,
				config.GetAgentFirstTokenTimeout(),
				config.GetAgentStreamIdleTimeout(),
			)
			agentCtx = streamCtx
			agentCancel = totalCancel
			// Capture loop variables for the token closure.
			agentID := sa.AgentID
			iterN := iterNum
			batcher := newAgentTokenBatcher()
			phaseThrottle := &streamPhaseThrottler{}
			streamChars := 0
			emitTokens := func(text string) {
				if text == "" {
					return
				}
				streamChars += len(text)
				e.emitter.Emit(sess.ID, EventAgentToken, map[string]any{
					"iteration": iterN,
					"agent_id":  agentID,
					"token":     text,
				})
				if phaseThrottle.shouldEmit(streamChars) {
					phaseThrottle.markEmitted(streamChars)
					e.emitter.Emit(sess.ID, EventAgentPhase, map[string]any{
						"iteration": iterN,
						"agent_id":  agentID,
						"role":      sa.Role,
						"phase":     "generating",
						"detail":    fmt.Sprintf("Streaming model output… %d characters received", streamChars),
					})
				}
			}
			tokenFn := func(token string) {
				bumpIdle()
				if batched := batcher.append(token); batched != "" {
					emitTokens(batched)
				}
			}
			out, err = e.streamDispatch(agentCtx, ag, agentpkg.Role(sa.Role), activeSkills, sa.LLMOverride, current, userFeedback, dispatchCtx, tokenFn)
			if flushed := batcher.flush(); flushed != "" {
				emitTokens(flushed)
			}
		} else {
			agentCtx = totalCtx
			agentCancel = totalCancel
			out, err = e.dispatch(agentCtx, ag, agentpkg.Role(sa.Role), activeSkills, sa.LLMOverride, current, userFeedback, dispatchCtx)
		}
		agentCancel()
		if err != nil {
			e.logger.WarnContext(ctx, "agent dispatch error; aborting pipeline pass",
				slog.String("session_id", sess.ID),
				slog.String("agent_id", sa.AgentID),
				slog.String("role", sa.Role),
				slog.Int("iteration", iterNum),
				slog.String("error", err.Error()),
			)
			e.emitter.Emit(sess.ID, EventAgentError, map[string]any{
				"iteration": iterNum,
				"agent_id":  sa.AgentID,
				"error":     err.Error(),
			})
			return current, successCount, fmt.Errorf("agent %s dispatch failed: %w", sa.AgentID, err)
		}
		successCount++

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
			"iteration":            iterNum,
			"agent_id":             sa.AgentID,
			"confidence_delta":     confAfter - confBefore,
			"execution_plan_steps": len(out.ExecutionPlan),
			"risks_count":          len(out.Risks),
			"open_questions_count": len(out.OpenQuestions),
		})

		current = out
	}

	return current, successCount, nil
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
		return currentState, fmt.Errorf("run single agent: %w: agent %s, session %s",
			ErrAgentNotInSession, agentID, sess.ID)
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
	out, err := e.dispatch(ctx, ag, agentpkg.Role(sa.Role), activeSkills, sa.LLMOverride, currentState, "", agentpkg.DispatchContext{
		SessionID:  sess.ID,
		AgentID:    sa.AgentID,
		OutputDocs: sess.OutputDocs,
	})
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

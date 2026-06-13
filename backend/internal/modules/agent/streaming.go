// Package agent — streaming dispatch variant.
package agent

import (
	"context"
	"fmt"
	"log/slog"

	"a2a-brainstorm/backend/internal/modules/state"
	platA2A "a2a-brainstorm/backend/internal/platform/a2a"
	"a2a-brainstorm/backend/internal/platform/config"
	"a2a-brainstorm/backend/internal/platform/llm"
)

// DispatchWithTokens is a streaming variant of Dispatch that accepts a
// tokenFn callback invoked for each text token the agent emits during its
// Working phase.
//
// The function signature intentionally differs from DispatchFunc so that the
// iteration engine can hold both a blocking dispatch and a streaming dispatch
// without changing the DispatchFunc type (which tests depend on).
//
// tokenFn may be nil — in that case tokens are consumed and discarded.
func DispatchWithTokens(
	ctx context.Context,
	agent Agent,
	role Role,
	activeSkills []Skill,
	sessionLLMOverride *llm.LLMConfig,
	currentState state.CanonicalState,
	userFeedback string,
	tokenFn func(token string),
) (state.CanonicalState, error) {
	// 1. Resolve tiered LLM config.
	globalCfg := &llm.LLMConfig{
		Provider:      config.GetGlobalLLMProvider(),
		Model:         config.GetGlobalLLMModel(),
		CredentialRef: config.GetGlobalLLMCredentialRef(),
	}
	effectiveCfg := llm.Resolve(globalCfg, agent.LLMConfig, sessionLLMOverride)

	// 2. Assemble system prompt.
	systemPrompt := BuildSystemPrompt(agent.SystemPrompt, activeSkills)

	// 3. Pack payload and open streaming A2A connection.
	payload := platA2A.BrainstormPayload{
		Role:         string(role),
		SystemPrompt: systemPrompt,
		LLMConfig:    effectiveCfg,
		State:        currentState,
		UserFeedback: userFeedback,
	}

	slog.Default().InfoContext(ctx, "resolving A2A agent endpoint (streaming)",
		slog.String("agent_id", agent.ID),
		slog.String("agent_name", agent.Name),
		slog.String("role", string(role)),
		slog.String("endpoint", agent.Endpoint),
		slog.Int("skill_count", len(activeSkills)),
	)

	client, err := platA2A.NewStreamingClient(ctx, agent.Endpoint)
	if err != nil {
		return state.CanonicalState{}, fmt.Errorf("dispatch agent %s (stream): new client: %w", agent.ID, err)
	}

	slog.Default().InfoContext(ctx, "sending streaming payload to agent via A2A",
		slog.String("agent_id", agent.ID),
		slog.String("role", string(role)),
		slog.String("provider", effectiveCfg.Provider),
		slog.String("model", effectiveCfg.Model),
	)

	result, err := platA2A.SendStreamingPayload(ctx, client, payload, tokenFn)
	if err != nil {
		return state.CanonicalState{}, fmt.Errorf("dispatch agent %s (stream): send payload: %w", agent.ID, err)
	}

	// 4. Extract updated state from the artifact DataPart.
	stateAny, err := platA2A.ExtractStateFromResult(result)
	if err != nil {
		return state.CanonicalState{}, fmt.Errorf("dispatch agent %s (stream): extract state: %w", agent.ID, err)
	}

	updated, err := convertToCanonicalState(stateAny)
	if err != nil {
		return state.CanonicalState{}, fmt.Errorf("dispatch agent %s (stream): convert state: %w", agent.ID, err)
	}
	merged := state.MergeAgentDelta(currentState, updated)

	slog.Default().InfoContext(ctx, "agent state extracted (streaming)",
		slog.String("agent_id", agent.ID),
		slog.String("agent_name", agent.Name),
		slog.String("role", string(role)),
		slog.Float64("confidence", merged.Metrics.Confidence),
		slog.Int("execution_plan_steps", len(merged.ExecutionPlan)),
	)

	return merged, nil
}

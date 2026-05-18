// Package agent provides the A2A dispatch function and system prompt assembly
// for the brainstorm pipeline.
//
// Dispatch is the primary call site used by the iteration engine: it resolves
// the tiered LLM config, assembles the effective system prompt from the agent's
// base prompt and active skills, packs everything into a BrainstormPayload, and
// sends it to the remote agent over the A2A protocol.
//
// All LLM config resolution follows the priority order defined in §8.12:
//
//	session override → agent-level → global default
//
// All A2A communication delegates to backend/internal/platform/a2a.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
	platA2A "a2a-brainstorm/backend/internal/platform/a2a"
	"a2a-brainstorm/backend/internal/platform/config"
	"a2a-brainstorm/backend/internal/platform/llm"
)

// Dispatch executes one pipeline pass for a single agent:
//
//  1. Resolves the effective LLM config (tiered: session override → agent-level → global).
//  2. Assembles the effective system prompt (base + active skill fragments).
//  3. Packs BrainstormPayload as a DataPart and sends it to the agent via A2A.
//  4. Extracts the updated CanonicalState from the agent's artifact response.
//
// The returned CanonicalState is the raw agent output. Callers (iteration engine)
// are responsible for running state.Merge and persisting the result.
func Dispatch(
	ctx context.Context,
	agent Agent,
	role Role,
	activeSkills []Skill,
	sessionLLMOverride *llm.LLMConfig,
	currentState state.CanonicalState,
) (state.CanonicalState, error) {
	// 1. Resolve tiered LLM config.
	globalCfg := &llm.LLMConfig{
		Provider:      config.GetGlobalLLMProvider(),
		Model:         config.GetGlobalLLMModel(),
		CredentialRef: config.GetGlobalLLMCredentialRef(),
	}
	effectiveCfg := llm.Resolve(globalCfg, agent.LLMConfig, sessionLLMOverride)

	// 2. Assemble system prompt from agent base + skill fragments.
	systemPrompt := BuildSystemPrompt(agent.SystemPrompt, activeSkills)

	// 3. Pack the payload and send via A2A.
	payload := platA2A.BrainstormPayload{
		Role:         string(role),
		SystemPrompt: systemPrompt,
		LLMConfig:    effectiveCfg,
		State:        currentState,
	}

	client, err := platA2A.NewClient(ctx, agent.Endpoint)
	if err != nil {
		return state.CanonicalState{}, fmt.Errorf("dispatch agent %s: new client: %w", agent.ID, err)
	}

	result, err := platA2A.SendPayload(ctx, client, payload)
	if err != nil {
		return state.CanonicalState{}, fmt.Errorf("dispatch agent %s: send payload: %w", agent.ID, err)
	}

	// 4. Extract the updated state from the agent's DataPart artifact.
	stateAny, err := platA2A.ExtractStateFromResult(result)
	if err != nil {
		return state.CanonicalState{}, fmt.Errorf("dispatch agent %s: extract state: %w", agent.ID, err)
	}

	updated, err := convertToCanonicalState(stateAny)
	if err != nil {
		return state.CanonicalState{}, fmt.Errorf("dispatch agent %s: convert state: %w", agent.ID, err)
	}
	return updated, nil
}

// BuildSystemPrompt assembles the effective system prompt by concatenating the
// agent's base prompt with each active skill's prompt fragment, separated by
// double newlines.
//
// This is the canonical injection point for skills — the agent binary receives
// only the assembled string and has no knowledge of skill names or IDs.
//
// Assembly follows §8.14:
//
//	effective_prompt = base
//	               + "\n\n" + skill_1.prompt
//	               + "\n\n" + skill_2.prompt
//	               + ...
func BuildSystemPrompt(base string, skills []Skill) string {
	if len(skills) == 0 {
		return base
	}
	parts := make([]string, 0, 1+len(skills))
	parts = append(parts, base)
	for _, sk := range skills {
		if sk.Prompt != "" {
			parts = append(parts, sk.Prompt)
		}
	}
	return strings.Join(parts, "\n\n")
}

// convertToCanonicalState coerces the raw value returned by
// ExtractStateFromResult (a2a DataPart content — typically a map[string]any or
// already-typed struct) into a CanonicalState by round-tripping through JSON.
func convertToCanonicalState(v any) (state.CanonicalState, error) {
	if v == nil {
		return state.CanonicalState{}, fmt.Errorf("agent returned nil state")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return state.CanonicalState{}, fmt.Errorf("marshal extracted state: %w", err)
	}
	var s state.CanonicalState
	if err := json.Unmarshal(b, &s); err != nil {
		return state.CanonicalState{}, fmt.Errorf("unmarshal to CanonicalState: %w", err)
	}
	return s, nil
}

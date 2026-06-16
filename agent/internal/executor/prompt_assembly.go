package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"a2a-brainstorm/agent/internal/config"
	"a2a-brainstorm/agent/internal/executor/prompts"
	"a2a-brainstorm/agent/internal/llm"
)

const (
	stateSectionPrefix = "Current brainstorm state (read-only context — do NOT echo unchanged fields):\n"
	stateSectionSuffix = "\n\nReturn your role-scoped JSON delta now."
	sessionAnchorRule  = "Only the latest CURRENT_STATE block is authoritative."
	currentStateLabel  = "CURRENT_STATE"
)

// PromptCacheMode selects how prompts are assembled before LLM calls.
type PromptCacheMode string

const (
	PromptCacheLegacy PromptCacheMode = "legacy"
	PromptCacheTiered PromptCacheMode = "tiered"
	PromptCacheThread PromptCacheMode = "thread"
)

// AssembledPrompt holds the result of prompt assembly for one dispatch.
type AssembledPrompt struct {
	Legacy llm.LLMRequest
	Tiered *llm.TieredPrompt
	Mode   PromptCacheMode
}

// marshalStateJSON serializes payload.State for the authoritative state block.
func marshalStateJSON(payload BrainstormPayload) string {
	stateJSON, err := json.Marshal(payload.State)
	if err != nil {
		return "{}"
	}
	return string(stateJSON)
}

// buildInjectedSystemPrompt returns the legacy system prompt with plan/readme injections.
func buildInjectedSystemPrompt(payload BrainstormPayload) string {
	systemPrompt := prompts.InjectIfPlanOutput(payload.OutputDocs, payload.SystemPrompt)
	systemPrompt = prompts.InjectIfReadmeOutput(systemPrompt, payload.OutputDocs)
	return systemPrompt + requiredOutputStructurePrompt
}

// feedbackSection returns the optional user feedback block.
func feedbackSection(userFeedback string) string {
	if userFeedback == "" {
		return ""
	}
	return fmt.Sprintf(
		"\n\nUSER FEEDBACK (highest priority — address this in your response):\n%s\n",
		userFeedback,
	)
}

// buildDynamicUserContent is Tier 3 — always includes full authoritative state.
func buildDynamicUserContent(payload BrainstormPayload, stateJSON string) string {
	return buildDynamicUserContentWithLabel(payload, stateJSON, false)
}

// buildThreadStateContent labels the state block for multi-turn thread mode.
func buildThreadStateContent(payload BrainstormPayload, stateJSON string) string {
	return buildDynamicUserContentWithLabel(payload, stateJSON, true)
}

func buildDynamicUserContentWithLabel(payload BrainstormPayload, stateJSON string, labelState bool) string {
	prefix := ""
	if labelState {
		prefix = currentStateLabel + ":\n"
	}
	return fmt.Sprintf(
		"%s%s%s%s%s",
		feedbackSection(payload.UserFeedback),
		prefix,
		stateSectionPrefix,
		stateJSON,
		stateSectionSuffix,
	)
}

// buildSessionAnchor returns Tier 2 session-stable metadata (no state duplication).
func buildSessionAnchor(outputDocs []string) string {
	var b strings.Builder
	b.WriteString("SESSION_ANCHOR:\n")
	if len(outputDocs) > 0 {
		b.WriteString("OUTPUT_DOCS: ")
		b.WriteString(strings.Join(outputDocs, ", "))
		b.WriteByte('\n')
	}
	b.WriteString(sessionAnchorRule)
	return b.String()
}

// BuildLegacyLLMRequest reproduces the pre-cache prompt assembly exactly.
func BuildLegacyLLMRequest(payload BrainstormPayload) llm.LLMRequest {
	stateJSON := marshalStateJSON(payload)
	userMessage := fmt.Sprintf(
		"%s\n%s\n%s\n"+
			"Current brainstorm state (read-only context — do NOT echo unchanged fields):\n%s\n\n"+
			"Return your role-scoped JSON delta now.",
		deltaOutputPreamble,
		roleDeltaInstruction(payload.Role),
		feedbackSection(payload.UserFeedback),
		stateJSON,
	)
	return llm.LLMRequest{
		SystemPrompt: buildInjectedSystemPrompt(payload),
		UserMessage:  userMessage,
		Temperature:  0.15,
	}
}

// BuildTieredBlocks returns Option A cache tiers for one dispatch.
func BuildTieredBlocks(payload BrainstormPayload) []llm.PromptBlock {
	stateJSON := marshalStateJSON(payload)
	tier1 := buildInjectedSystemPrompt(payload) + "\n\n" +
		deltaOutputPreamble + "\n" + roleDeltaInstruction(payload.Role)

	blocks := []llm.PromptBlock{
		{Role: "system", Content: tier1, CachePolicy: llm.CacheEphemeral},
		{Role: "user", Content: buildSessionAnchor(payload.OutputDocs), CachePolicy: llm.CacheEphemeral},
		{Role: "user", Content: buildDynamicUserContent(payload, stateJSON), CachePolicy: llm.CacheNone},
	}
	return blocks
}

// AssemblePrompt builds the LLM request for the configured cache mode.
func AssemblePrompt(payload BrainstormPayload, threads *ThreadStore) AssembledPrompt {
	if !config.GetPromptCacheEnabled() {
		legacy := BuildLegacyLLMRequest(payload)
		return AssembledPrompt{Legacy: legacy, Mode: PromptCacheLegacy}
	}

	mode := PromptCacheMode(config.GetPromptCacheMode())
	switch mode {
	case PromptCacheTiered:
		tiered := &llm.TieredPrompt{
			Blocks:    BuildTieredBlocks(payload),
			SessionID: payload.SessionID,
			AgentID:   payload.AgentID,
			Provider:  payload.LLMConfig.Provider,
			Model:     payload.LLMConfig.Model,
		}
		legacy := tiered.FlattenLegacy()
		legacy.Temperature = 0.15
		legacy.Tiered = tiered
		return AssembledPrompt{Legacy: legacy, Tiered: tiered, Mode: PromptCacheTiered}

	case PromptCacheThread:
		if threads == nil {
			threads = defaultThreadStore
		}
		messages := threads.MessagesFor(payload)
		tiered := &llm.TieredPrompt{
			Messages:  messages,
			SessionID: payload.SessionID,
			AgentID:   payload.AgentID,
			Provider:  payload.LLMConfig.Provider,
			Model:     payload.LLMConfig.Model,
		}
		legacy := tiered.FlattenLegacy()
		legacy.Temperature = 0.15
		legacy.Tiered = tiered
		return AssembledPrompt{Legacy: legacy, Tiered: tiered, Mode: PromptCacheThread}

	default:
		legacy := BuildLegacyLLMRequest(payload)
		return AssembledPrompt{Legacy: legacy, Mode: PromptCacheLegacy}
	}
}

// RunShadowEquivalenceCheck logs a warning when tiered assembly drops legacy content.
func RunShadowEquivalenceCheck(ctx context.Context, payload BrainstormPayload, logger *slog.Logger) {
	if !config.GetPromptCacheShadowMode() {
		return
	}
	legacy := BuildLegacyLLMRequest(payload)
	tiered := BuildTieredBlocks(payload)
	if err := AssertSemanticEquivalent(legacy, tiered); err != nil {
		if logger != nil {
			logger.WarnContext(ctx, "prompt cache shadow equivalence mismatch",
				slog.String("error", err.Error()),
				slog.String("session_id", payload.SessionID),
				slog.String("agent_id", payload.AgentID),
			)
		}
	}
}

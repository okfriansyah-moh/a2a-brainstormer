package executor

import (
	"testing"
)

func TestAssertSemanticEquivalent_LegacyVsTiered(t *testing.T) {
	payload := BrainstormPayload{
		Role:         "build",
		SystemPrompt: "You are an architect.",
		OutputDocs:   []string{"architecture", "plan"},
		UserFeedback: "Focus on API design.",
		State: map[string]any{
			"idea": map[string]any{"text": "agentic commerce"},
			"metrics": map[string]any{"confidence": 0.5},
		},
	}

	legacy := BuildLegacyLLMRequest(payload)
	tiered := BuildTieredBlocks(payload)
	if err := AssertSemanticEquivalent(legacy, tiered); err != nil {
		t.Fatalf("semantic equivalence failed: %v", err)
	}
}

func TestAssertSemanticEquivalent_AllRoles(t *testing.T) {
	roles := []string{"build", "review", "refine", "devils_advocate", "unknown"}
	for _, role := range roles {
		payload := BrainstormPayload{
			Role:         role,
			SystemPrompt: "base",
			State:        map[string]any{"idea": map[string]any{"text": "x"}},
		}
		if err := AssertSemanticEquivalent(BuildLegacyLLMRequest(payload), BuildTieredBlocks(payload)); err != nil {
			t.Fatalf("role %q: %v", role, err)
		}
	}
}

func TestNormalizeStateJSON_StableKeys(t *testing.T) {
	raw := `{"metrics":{"confidence":0.5},"idea":{"text":"x"}}`
	want := `{"idea":{"text":"x"},"metrics":{"confidence":0.5}}`
	got := NormalizeStateJSON(raw)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildLegacyPreservesStateJSON(t *testing.T) {
	payload := BrainstormPayload{
		Role:         "review",
		SystemPrompt: "reviewer",
		State:        map[string]any{"idea": map[string]any{"text": "seed"}},
	}
	legacy := BuildLegacyLLMRequest(payload)
	parts := LegacyPromptParts(legacy)
	if parts.StateJSON != `{"idea":{"text":"seed"}}` {
		t.Fatalf("state = %q", parts.StateJSON)
	}
}

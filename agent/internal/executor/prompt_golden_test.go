package executor

import (
	"encoding/json"
	"testing"
)

// goldenDelta is a minimal agent JSON delta used in replay fixtures.
var goldenBuildDelta = map[string]any{
	"architecture": map[string]any{
		"layers": []any{
			map[string]any{"name": "API", "responsibility": "HTTP handlers"},
		},
	},
	"metrics": map[string]any{"confidence": 0.85},
}

func TestGoldenReplay_LegacyVsThreadStateJSON(t *testing.T) {
	t.Setenv("PROMPT_CACHE_MODE", "legacy")
	payload := BrainstormPayload{
		SessionID: "golden-sess", AgentID: "golden-agent",
		Role: "build", SystemPrompt: "architect",
		OutputDocs: []string{"architecture", "plan"},
		State: map[string]any{"idea": map[string]any{"text": "commerce platform"}},
	}
	legacyParts := LegacyPromptParts(BuildLegacyLLMRequest(payload))

	defaultThreadStore.Reset()
	t.Setenv("PROMPT_CACHE_MODE", "thread")
	threadParts := TieredPromptParts(defaultThreadStore.MessagesFor(payload))

	if legacyParts.StateJSON != threadParts.StateJSON {
		t.Fatalf("state mismatch:\nlegacy=%s\nthread=%s", legacyParts.StateJSON, threadParts.StateJSON)
	}
}

func TestGoldenReplay_MergedStateFieldParity(t *testing.T) {
	base := map[string]any{
		"idea":    map[string]any{"text": "commerce"},
		"metrics": map[string]any{"confidence": 0.5},
	}
	merged := mergeGoldenDelta(base, goldenBuildDelta)
	metrics, ok := merged["metrics"].(map[string]any)
	if !ok || metrics["confidence"] != 0.85 {
		t.Fatalf("metrics = %#v", merged["metrics"])
	}
	if merged["architecture"] == nil {
		t.Fatal("expected architecture from delta")
	}
}

func mergeGoldenDelta(base, delta map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(delta))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range delta {
		out[k] = v
	}
	return out
}

func TestGoldenReplay_FixtureRoundTrip(t *testing.T) {
	raw := `{"role":"build","system_prompt":"architect","state":{"idea":{"text":"x"}}}`
	var payload BrainstormPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	legacy := BuildLegacyLLMRequest(payload)
	if legacy.Temperature != 0.15 {
		t.Fatalf("temperature = %v", legacy.Temperature)
	}
}

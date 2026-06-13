package config

import "testing"

func TestGetGlobalLLMProvider_DefaultsToDeepSeek(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "")
	if got := GetGlobalLLMProvider(); got != "deepseek" {
		t.Fatalf("GetGlobalLLMProvider() = %q, want %q", got, "deepseek")
	}
}

func TestGetGlobalLLMModel_DefaultsToDeepSeekV4Flash(t *testing.T) {
	t.Setenv("GLOBAL_LLM_MODEL", "")
	if got := GetGlobalLLMModel(); got != "deepseek-v4-flash" {
		t.Fatalf("GetGlobalLLMModel() = %q, want %q", got, "deepseek-v4-flash")
	}
}

func TestGetGlobalLLMCredentialRef_DefaultsToDeepSeekKey(t *testing.T) {
	t.Setenv("GLOBAL_LLM_CREDENTIAL_REF", "")
	if got := GetGlobalLLMCredentialRef(); got != "DEEPSEEK_API_KEY" {
		t.Fatalf("GetGlobalLLMCredentialRef() = %q, want %q", got, "DEEPSEEK_API_KEY")
	}
}
func TestGetMinConfidenceFloor_ClampsBounds(t *testing.T) {
	t.Setenv("MIN_CONFIDENCE_FLOOR", "-0.5")
	if got := GetMinConfidenceFloor(); got != 0 {
		t.Fatalf("expected clamped floor 0, got %v", got)
	}

	t.Setenv("MIN_CONFIDENCE_FLOOR", "1.5")
	if got := GetMinConfidenceFloor(); got != 1 {
		t.Fatalf("expected clamped floor 1, got %v", got)
	}
}

func TestGetMinConfidenceFloor_UsesDefaultWhenUnset(t *testing.T) {
	t.Setenv("MIN_CONFIDENCE_FLOOR", "")
	if got := GetMinConfidenceFloor(); got != 0.90 {
		t.Fatalf("expected default floor 0.90, got %v", got)
	}
}

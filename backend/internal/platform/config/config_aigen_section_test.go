package config

import "testing"

func TestIsSectionSequentialDoc_Defaults(t *testing.T) {
	t.Setenv("AIGEN_SECTION_SEQUENTIAL", "")
	if !IsSectionSequentialDoc("architecture") {
		t.Fatal("expected architecture in default section-sequential set")
	}
	if !IsSectionSequentialDoc("plan") {
		t.Fatal("expected plan in default section-sequential set")
	}
	if IsSectionSequentialDoc("readme") {
		t.Fatal("readme should not be section-sequential by default")
	}
}

func TestIsSectionSequentialDoc_CustomList(t *testing.T) {
	t.Setenv("AIGEN_SECTION_SEQUENTIAL", "plan")
	if IsSectionSequentialDoc("architecture") {
		t.Fatal("architecture should be excluded")
	}
	if !IsSectionSequentialDoc("plan") {
		t.Fatal("plan should be included")
	}
}

func TestGetAIGenCoherenceMinRatio_Clamps(t *testing.T) {
	t.Setenv("AIGEN_COHERENCE_MIN_RATIO", "0.1")
	if got := GetAIGenCoherenceMinRatio(); got != 0.5 {
		t.Fatalf("expected clamp 0.5, got %v", got)
	}
	t.Setenv("AIGEN_COHERENCE_MIN_RATIO", "2")
	if got := GetAIGenCoherenceMinRatio(); got != 1.0 {
		t.Fatalf("expected clamp 1.0, got %v", got)
	}
}

func TestGetAIGenCoherenceEnabled_ParseBool(t *testing.T) {
	t.Setenv("AIGEN_COHERENCE_ENABLED", "false")
	if GetAIGenCoherenceEnabled() {
		t.Fatal("expected false")
	}
}

func TestGetAIGenPriorSectionMaxChars_Clamps(t *testing.T) {
	t.Setenv("AIGEN_PRIOR_SECTION_MAX_CHARS", "100")
	if got := GetAIGenPriorSectionMaxChars(); got != 500 {
		t.Fatalf("expected clamp 500, got %d", got)
	}
}

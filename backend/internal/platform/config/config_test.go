package config

import "testing"

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

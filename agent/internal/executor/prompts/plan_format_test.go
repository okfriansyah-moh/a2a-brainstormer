// Package prompts — tests for plan format injection.
package prompts

import (
	"strings"
	"testing"
)

// TestInjectIfPlanOutput_InjectsWhenPlanPresent verifies that when "plan" is in
// outputDocs, the PlanTaskFormat is appended to the base prompt.
func TestInjectIfPlanOutput_InjectsWhenPlanPresent(t *testing.T) {
	base := "You are a software architect."
	result := InjectIfPlanOutput([]string{"plan"}, base)

	if !strings.HasPrefix(result, base) {
		t.Error("expected result to start with base prompt")
	}
	if !strings.Contains(result, "PLAN.md Output Format Requirements") {
		t.Error("expected PlanTaskFormat to be injected when 'plan' is in outputDocs")
	}
	if !strings.Contains(result, "**Goal:**") {
		t.Error("expected **Goal:** in injected format")
	}
	if !strings.Contains(result, "go build ./...") {
		t.Error("expected go build ./... in injected format")
	}
}

// TestInjectIfPlanOutput_SkipsWhenNoPlan verifies that when "plan" is NOT in
// outputDocs, the base prompt is returned unchanged.
func TestInjectIfPlanOutput_SkipsWhenNoPlan(t *testing.T) {
	base := "You are a software architect."
	result := InjectIfPlanOutput([]string{"readme"}, base)

	if result != base {
		t.Errorf("expected base prompt unchanged when 'plan' not in outputDocs, got: %q", result)
	}
}

// TestInjectIfPlanOutput_EmptyDocs verifies that an empty outputDocs slice
// returns the base prompt unchanged.
func TestInjectIfPlanOutput_EmptyDocs(t *testing.T) {
	base := "base prompt"
	result := InjectIfPlanOutput(nil, base)
	if result != base {
		t.Errorf("expected base prompt unchanged for nil outputDocs, got: %q", result)
	}

	result2 := InjectIfPlanOutput([]string{}, base)
	if result2 != base {
		t.Errorf("expected base prompt unchanged for empty outputDocs, got: %q", result2)
	}
}

// TestPlanTaskFormat_Length verifies the constant is within the 2500 char limit.
func TestPlanTaskFormat_Length(t *testing.T) {
	if len(PlanTaskFormat) > 2500 {
		t.Errorf("PlanTaskFormat exceeds 2500 chars: got %d", len(PlanTaskFormat))
	}
}

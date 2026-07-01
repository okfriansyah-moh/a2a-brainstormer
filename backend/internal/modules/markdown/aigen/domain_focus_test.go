package aigen

import (
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
)

func TestDomainFocusRules_ForbidsBrainstormTool(t *testing.T) {
	s := state.CanonicalState{
		Idea: map[string]any{
			"title": "Match Point",
			"text":  "Cross-community player reputation for padel and tennis",
		},
	}
	rules := domainFocusRules(s)
	for _, needle := range []string{
		"USER'S PRODUCT",
		"A2A Brainstorm",
		"Match Point",
	} {
		if !strings.Contains(rules, needle) {
			t.Errorf("domainFocusRules missing %q", needle)
		}
	}
}

func TestSummariseState_IncludesExecutionPlan(t *testing.T) {
	s := state.CanonicalState{
		Idea: map[string]any{"title": "Match Point"},
		ExecutionPlan: []state.Step{
			{Title: "Reputation engine", Objective: "Build scoring core"},
		},
	}
	got := summariseState(s)
	if !strings.Contains(got, "Reputation engine") {
		t.Errorf("summariseState missing execution plan title: %q", got)
	}
}

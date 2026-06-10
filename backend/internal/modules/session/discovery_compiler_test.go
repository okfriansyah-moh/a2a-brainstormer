package session

import (
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/shared"
)

func TestCompileEnrichedIdea_Deterministic(t *testing.T) {
	answers := DiscoveryAnswers{
		Q1: "Backend engineers using spreadsheets",
		Q2: []string{"API contracts", "Auth"},
		Q3: []string{"Zero data loss"},
		Q4: []string{"Saves hours per week"},
	}
	constraints := shared.TechConstraints{
		AgentsDecide: false,
		MustUse:      []string{"Go", "PostgreSQL"},
	}

	got1 := CompileEnrichedIdea("Build a pricing orchestrator", answers, constraints)
	got2 := CompileEnrichedIdea("Build a pricing orchestrator", answers, constraints)
	if got1 != got2 {
		t.Fatal("CompileEnrichedIdea is not deterministic")
	}
	for _, want := range []string{
		"## Product Idea",
		"Build a pricing orchestrator",
		"### Persona & current state",
		"Backend engineers using spreadsheets",
		"### MVP must-haves",
		"- API contracts",
		"Must use: Go, PostgreSQL",
	} {
		if !strings.Contains(got1, want) {
			t.Errorf("missing %q in enriched idea", want)
		}
	}
}

func TestSeedInitialState_MapsDiscoveryFields(t *testing.T) {
	answers := DiscoveryAnswers{
		Q1: "Ops teams",
		Q2: []string{"Core data model"},
		Q3: []string{"Sub-100ms latency"},
		Q4: []string{"Cheaper to run"},
	}
	constraints := shared.TechConstraints{AgentsDecide: true}

	cs := SeedInitialState("Idea text", answers, constraints)
	if err := state.Validate(cs); err != nil {
		t.Fatalf("seeded state invalid: %v", err)
	}
	if cs.Idea["persona"] != "Ops teams" {
		t.Errorf("persona = %v", cs.Idea["persona"])
	}
	mvp, ok := cs.Idea["mvp_must_haves"].([]string)
	if !ok || len(mvp) != 1 || mvp[0] != "Core data model" {
		t.Errorf("mvp_must_haves = %v", cs.Idea["mvp_must_haves"])
	}
	if len(cs.ExecutionPlan) != 1 || cs.ExecutionPlan[0].Title != "Core data model" {
		t.Fatalf("execution_plan seed = %+v", cs.ExecutionPlan)
	}
	if len(cs.Assumptions) != 1 || !strings.HasPrefix(cs.Assumptions[0], "Non-negotiable:") {
		t.Fatalf("assumptions = %v", cs.Assumptions)
	}
}

func TestSeedInitialState_WhitespaceOnlyQ2DoesNotPanic(t *testing.T) {
	answers := DiscoveryAnswers{Q2: []string{"  ", "\t", ""}}
	constraints := shared.TechConstraints{AgentsDecide: true}

	cs := SeedInitialState("Idea text", answers, constraints)
	if err := state.Validate(cs); err != nil {
		t.Fatalf("seeded state invalid: %v", err)
	}
	if _, ok := cs.Idea["mvp_must_haves"]; ok {
		t.Fatalf("expected no mvp_must_haves for whitespace-only Q2, got %v", cs.Idea["mvp_must_haves"])
	}
	if len(cs.ExecutionPlan) != 0 {
		t.Fatalf("expected empty execution_plan, got %+v", cs.ExecutionPlan)
	}
}

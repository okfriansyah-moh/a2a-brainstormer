package session

import (
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/shared"
)

// DiscoveryAnswers holds optional guided-onboarding responses (§8.29.2).
type DiscoveryAnswers struct {
	Q1 string   `json:"q1,omitempty"`
	Q2 []string `json:"q2,omitempty"`
	Q3 []string `json:"q3,omitempty"`
	Q4 []string `json:"q4,omitempty"`
}

// CompileEnrichedIdea assembles deterministic markdown from user inputs (no LLM).
func CompileEnrichedIdea(idea string, answers DiscoveryAnswers, constraints shared.TechConstraints) string {
	var b strings.Builder
	b.WriteString("## Product Idea\n\n")
	b.WriteString(strings.TrimSpace(idea))
	b.WriteString("\n\n## Discovery\n\n")

	b.WriteString("### Persona & current state\n\n")
	if strings.TrimSpace(answers.Q1) != "" {
		b.WriteString(strings.TrimSpace(answers.Q1))
	} else {
		b.WriteString("_skipped_")
	}
	b.WriteString("\n\n")

	writeChipSection(&b, "### MVP must-haves\n\n", answers.Q2)
	writeChipSection(&b, "### Non-negotiables\n\n", answers.Q3)
	writeChipSection(&b, "### Value proposition\n\n", answers.Q4)

	b.WriteString("## Tech Constraints\n\n")
	if constraints.AgentsDecide {
		b.WriteString("_Agents will decide stack_")
	} else {
		for _, line := range constraints.ToAssumptions() {
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		if len(constraints.ToAssumptions()) == 0 {
			b.WriteString("_Agents will decide stack_")
		}
	}
	b.WriteString("\n")
	return b.String()
}

func writeChipSection(b *strings.Builder, heading string, items []string) {
	b.WriteString(heading)
	if len(items) == 0 {
		b.WriteString("_skipped_\n\n")
		return
	}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

// SeedInitialState maps discovery answers into canonical JSON before iteration 1.
func SeedInitialState(idea string, answers DiscoveryAnswers, constraints shared.TechConstraints) state.CanonicalState {
	cs := state.CanonicalState{
		Idea: map[string]any{
			"text": strings.TrimSpace(idea),
		},
		Architecture:  map[string]any{},
		ExecutionPlan: nil,
		Assumptions:   nil,
		OpenQuestions: nil,
		Metrics:       state.StateMetrics{Confidence: 0.0},
		Meta:          state.StateMeta{Iteration: 0},
	}

	q1 := strings.TrimSpace(answers.Q1)
	if q1 != "" {
		cs.Idea["persona"] = q1
		cs.Idea["context"] = q1
	}

	if len(answers.Q2) > 0 {
		mvp := dedupeStrings(answers.Q2)
		cs.Idea["mvp_must_haves"] = mvp
		first := mvp[0]
		cs.ExecutionPlan = []state.Step{{
			Title:       first,
			Description: fmt.Sprintf("MVP must-have from discovery: %s", first),
		}}
	}

	for _, item := range dedupeStrings(answers.Q3) {
		cs.Assumptions = append(cs.Assumptions, "Non-negotiable: "+item)
	}
	cs.Assumptions = append(cs.Assumptions, constraints.ToAssumptions()...)

	if len(answers.Q4) > 0 {
		cs.Idea["value_props"] = dedupeStrings(answers.Q4)
	}

	return cs
}

func dedupeStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

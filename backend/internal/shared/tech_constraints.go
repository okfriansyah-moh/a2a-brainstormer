package shared

import "strings"

// TechConstraints captures optional stack preferences from guided onboarding.
// When AgentsDecide is true the three tier arrays are ignored.
type TechConstraints struct {
	AgentsDecide    bool     `json:"agents_decide"`
	MustUse         []string `json:"must_use,omitempty"`
	ComfortableWith []string `json:"comfortable_with,omitempty"`
	AvoidIfPossible []string `json:"avoid_if_possible,omitempty"`
}

// DefaultTechConstraints is the onboarding default: agents pick the stack.
func DefaultTechConstraints() TechConstraints {
	return TechConstraints{AgentsDecide: true}
}

// ToAssumptions formats non-empty constraint tiers into canonical assumptions[].
// Returns nil when agents_decide is true.
func (t TechConstraints) ToAssumptions() []string {
	if t.AgentsDecide {
		return nil
	}
	var out []string
	if len(t.MustUse) > 0 {
		out = append(out, "Must use: "+joinUnique(t.MustUse))
	}
	if len(t.ComfortableWith) > 0 {
		out = append(out, "Comfortable with: "+joinUnique(t.ComfortableWith))
	}
	if len(t.AvoidIfPossible) > 0 {
		out = append(out, "Avoid if possible: "+joinUnique(t.AvoidIfPossible))
	}
	return out
}

func joinUnique(items []string) string {
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
	return strings.Join(out, ", ")
}

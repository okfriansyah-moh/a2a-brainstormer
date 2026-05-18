// Package state provides Validate, which checks a CanonicalState for
// structural correctness before it is persisted or dispatched to agents.
package state

import "fmt"

// Validate returns an error if s is structurally invalid.
//
// Enforced invariants:
//   - Idea must be non-nil and contain at least one key (§8.1).
//   - Metrics.Confidence must be in the closed interval [0.0, 1.0].
//   - When Meta.Iteration > 0, Meta.Agents must contain at least 2 entries
//     (the minimum pipeline size required by the session service).
func Validate(s CanonicalState) error {
	if len(s.Idea) == 0 {
		return fmt.Errorf("state validation: idea must not be empty")
	}
	if s.Metrics.Confidence < 0.0 || s.Metrics.Confidence > 1.0 {
		return fmt.Errorf("state validation: confidence %g is outside [0, 1]", s.Metrics.Confidence)
	}
	if s.Meta.Iteration > 0 && len(s.Meta.Agents) < 2 {
		return fmt.Errorf("state validation: meta.agents must have at least 2 entries when iteration > 0, got %d", len(s.Meta.Agents))
	}
	return nil
}

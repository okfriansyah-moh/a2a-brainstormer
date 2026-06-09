// Package convergence implements the multi-condition convergence detection
// algorithm for the brainstorm iteration pipeline.
//
// Stop conditions (§8.6 of docs/PLAN.md):
//
// ALL of the following must hold for quality convergence:
//  1. No new critical risks appeared between iterations.
//  2. The execution plan is complete (all steps have descriptions; none
//     referenced in open_questions).
//  3. The absolute confidence delta between iterations is below the configured
//     threshold (default 0.02, set via CONVERGENCE_THRESHOLD env var).
//
// OR any of the following triggers a forced stop:
//  4. User explicitly approved via POST /sessions/{id}/finalize.
//  5. The iteration count reached the configured maximum (enforced by the
//     iteration engine, not this package).
package convergence

import (
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/config"
)

// Check returns true when the pipeline has converged based on quality
// conditions 1–3 from §8.6, plus the minimum confidence floor. All four
// conditions must hold simultaneously.
//
// The iteration engine is responsible for enforcing the maxIter cap (condition
// 5); this function only evaluates quality-based convergence.
func Check(prev, next state.CanonicalState) bool {
	threshold := config.GetConvergenceThreshold()
	floor := config.GetMinConfidenceFloor()
	return next.Metrics.Confidence >= floor &&
		!HasNewCriticalRisks(prev, next) &&
		IsExecutionPlanComplete(next) &&
		ConfidenceDelta(prev, next) < threshold
}

// ConfidenceDelta returns the absolute change in confidence score between two
// successive canonical states: |next.Metrics.Confidence - prev.Metrics.Confidence|.
func ConfidenceDelta(prev, next state.CanonicalState) float64 {
	d := next.Metrics.Confidence - prev.Metrics.Confidence
	if d < 0 {
		d = -d
	}
	return d
}

// HasNewCriticalRisks returns true if next contains at least one
// critical-severity risk whose normalised text was not present in prev.
//
// Normalisation is: lowercase, trimmed, whitespace collapsed to single spaces.
// This catches semantically duplicate entries that differ only in casing or spacing.
func HasNewCriticalRisks(prev, next state.CanonicalState) bool {
	prevCritical := make(map[string]struct{}, len(prev.Risks))
	for _, r := range prev.Risks {
		if r.Severity == "critical" {
			prevCritical[normaliseText(r.Text)] = struct{}{}
		}
	}
	for _, r := range next.Risks {
		if r.Severity == "critical" {
			if _, found := prevCritical[normaliseText(r.Text)]; !found {
				return true
			}
		}
	}
	return false
}

// IsExecutionPlanComplete returns true when the plan satisfies the completion
// heuristic from §8.6:
//
//   - The plan contains at least one step (an empty plan is never "complete").
//   - Every step has a non-empty description.
//   - No step title appears as a substring in any open question (which would
//     indicate the step is still under debate).
func IsExecutionPlanComplete(s state.CanonicalState) bool {
	if len(s.ExecutionPlan) == 0 {
		return false
	}
	for _, step := range s.ExecutionPlan {
		if strings.TrimSpace(step.Description) == "" {
			return false
		}
		stepLower := strings.ToLower(step.Title)
		for _, q := range s.OpenQuestions {
			if strings.Contains(strings.ToLower(q), stepLower) {
				return false
			}
		}
	}
	return true
}

// normaliseText returns a canonical deduplication key:
// lowercase, trimmed, internal whitespace collapsed to a single space.
func normaliseText(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

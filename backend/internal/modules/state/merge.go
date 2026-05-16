// Package state provides the merge algorithm that combines two CanonicalState
// snapshots into a single unified state after a pipeline pass.
//
// Merge rules (§8.5 of docs/PLAN.md):
//  1. Union risks: deduplicate by normalised text; keep unique risks from both.
//  2. Remove resolved: risks with Resolved=true are dropped.
//  3. Collapse duplicate plan steps: steps sharing a title are merged; the
//     longer description wins.
//  4. Reject vague steps: steps whose description is fewer than 10 words are
//     discarded.
//  5. Stability rule: if base and incoming agree on a field, it is not changed.
//  6. Persistent conflict: if an item has appeared in open_questions with a
//     conflict marker before, it is not duplicated.
package state

import (
	"fmt"
	"strings"
)

// Merge combines the base state (before the pipeline pass) with the incoming
// state (the cumulative output of all agents in the pass) and returns a new
// CanonicalState. Neither base nor incoming is mutated.
//
// The caller (iteration engine) is responsible for setting Meta.Iteration on
// the returned state.
func Merge(base, incoming CanonicalState) CanonicalState {
	out := CanonicalState{
		Idea:          mergeMap(base.Idea, incoming.Idea),
		Architecture:  mergeMap(base.Architecture, incoming.Architecture),
		ExecutionPlan: mergeSteps(base.ExecutionPlan, incoming.ExecutionPlan),
		Risks:         mergeRisks(base.Risks, incoming.Risks),
		Assumptions:   mergeStrings(base.Assumptions, incoming.Assumptions),
		OpenQuestions: mergeStrings(base.OpenQuestions, incoming.OpenQuestions),
		Metrics:       incoming.Metrics,
		Meta:          incoming.Meta,
	}

	// Stability rule (§8.5 rule 5): detect persistent conflicts and surface them.
	out.OpenQuestions = detectConflicts(base, incoming, out.OpenQuestions)

	return out
}

// ── internal helpers ──────────────────────────────────────────────────────────

// mergeMap merges two map[string]any values. If both are non-empty and equal,
// the base is returned (stability). Otherwise the incoming value wins.
func mergeMap(base, incoming map[string]any) map[string]any {
	if len(incoming) == 0 {
		return cloneMap(base)
	}
	return cloneMap(incoming)
}

// cloneMap returns a shallow copy of m; returns nil if m is nil.
func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// normaliseText returns a canonical form used for deduplication:
// lowercase, trimmed, whitespace collapsed.
func normaliseText(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

// mergeRisks applies §8.5 rules 1–2:
//   - Union by normalised text (dedup)
//   - Drop resolved risks
func mergeRisks(base, incoming []Risk) []Risk {
	seen := make(map[string]struct{})
	var out []Risk

	add := func(r Risk) {
		if r.Resolved {
			return // §8.5 rule 2: remove resolved
		}
		key := normaliseText(r.Text)
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}

	for _, r := range base {
		add(r)
	}
	for _, r := range incoming {
		add(r)
	}
	return out
}

// wordCount returns the number of whitespace-separated words in s.
func wordCount(s string) int {
	return len(strings.Fields(s))
}

// mergeSteps applies §8.5 rules 3–4:
//   - Collapse steps with identical titles (keep the more detailed description)
//   - Reject steps with description < 10 words
func mergeSteps(base, incoming []Step) []Step {
	byTitle := make(map[string]Step)
	order := make([]string, 0)

	record := func(s Step) {
		key := normaliseText(s.Title)
		if key == "" {
			return
		}
		if existing, ok := byTitle[key]; ok {
			// Keep the more detailed description.
			if wordCount(s.Description) > wordCount(existing.Description) {
				byTitle[key] = s
			}
		} else {
			byTitle[key] = s
			order = append(order, key)
		}
	}

	for _, s := range base {
		record(s)
	}
	for _, s := range incoming {
		record(s)
	}

	out := make([]Step, 0, len(order))
	for _, key := range order {
		s := byTitle[key]
		if wordCount(s.Description) < 10 {
			continue // §8.5 rule 4: reject vague steps
		}
		out = append(out, s)
	}
	return out
}

// mergeStrings unions two string slices, deduplicating by normalised form.
func mergeStrings(base, incoming []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range append(append([]string{}, base...), incoming...) {
		key := normaliseText(s)
		if key == "" {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}

// detectConflicts implements §8.5 rule 6: if the iteration count indicates a
// persistent conflict (iteration ≥ 3) and the idea or architecture fields
// differ between base and incoming, add an open question for human resolution.
// Already-present conflict markers are not duplicated.
func detectConflicts(base, incoming CanonicalState, existing []string) []string {
	if incoming.Meta.Iteration < 3 {
		return existing
	}

	existingSet := make(map[string]struct{}, len(existing))
	for _, q := range existing {
		existingSet[normaliseText(q)] = struct{}{}
	}

	candidate := func(msg string) {
		key := normaliseText(msg)
		if _, dup := existingSet[key]; !dup {
			existing = append(existing, msg)
			existingSet[key] = struct{}{}
		}
	}

	if !mapsEqual(base.Idea, incoming.Idea) && len(base.Idea) > 0 && len(incoming.Idea) > 0 {
		candidate(fmt.Sprintf("Persistent conflict in 'idea' field at iteration %d — requires human resolution", incoming.Meta.Iteration))
	}
	if !mapsEqual(base.Architecture, incoming.Architecture) && len(base.Architecture) > 0 && len(incoming.Architecture) > 0 {
		candidate(fmt.Sprintf("Persistent conflict in 'architecture' field at iteration %d — requires human resolution", incoming.Meta.Iteration))
	}

	return existing
}

// mapsEqual does a shallow key-value comparison of two map[string]any values.
func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		// Shallow comparison using fmt.Sprintf as a quick string representation.
		if fmt.Sprintf("%v", va) != fmt.Sprintf("%v", vb) {
			return false
		}
	}
	return true
}

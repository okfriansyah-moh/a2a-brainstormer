---
name: convergence-engine-patterns
type: skill
description: "Enforces correct multi-condition convergence detection, delta-confidence threshold evaluation, and stop logic in backend/internal/modules/convergence/engine.go."
---

## Purpose

This skill governs `backend/internal/modules/convergence/engine.go`. Convergence is the termination mechanism for the brainstorm pipeline. It is a **pure function module** — no DB access, no side effects — that evaluates whether the canonical state has stabilized enough to stop iterating. The rules come from `docs/PLAN.md §8.6` and `docs/A2A-agent-Brainstorm.md §11`.

## Rules

1. **`convergence.Check` is a pure function.** It must accept `(prev, next CanonicalState)` and return `bool`. No DB calls, no HTTP calls, no global state. The `convergence/` module has no repository and no service — only `engine.go` (see `docs/PLAN.md §8.8`).

2. **All three content-based conditions must hold together for convergence.** Return `true` only when ALL of: (a) no new critical risks, (b) execution plan complete, (c) `ConfidenceDelta < convergenceThreshold`. These are AND-conditions, not OR (see `docs/PLAN.md §8.6`).

3. **Early-exit conditions are OR-conditions that bypass content checks.** Return `true` immediately (regardless of content conditions) if: session status is `approved` OR `iteration >= maxIter`. Inject these as parameters — do not read from DB inside `Check`.

4. **`ConfidenceDelta` formula is exact.** Implement as `math.Abs(next.Metrics.Confidence - prev.Metrics.Confidence)`. The default threshold is `0.02`, read from `platform/config.GetConvergenceThreshold()`.

5. **`HasNewCriticalRisks` compares by risk text hash, not object identity.** A risk is "new critical" if its normalized text hash appears in `next.Risks` with `severity == "critical"` but not in `prev.Risks`. Use the same deduplication hash as `state/merge.go`.

6. **`IsExecutionPlanComplete` uses a two-part heuristic.** A plan is complete when: all steps have a non-empty description AND no step title appears in `state.OpenQuestions`. This mirrors the merge validator logic in `internal/modules/state/validator.go`.

7. **Persistent oscillation is surfaced via `open_questions`, not by forcing convergence.** If the same field has been toggled for 3+ iterations, `state/merge.go` adds it to `OpenQuestions`. `convergence.Check` then naturally fails condition (b) until the user resolves it.

## Anti-Patterns

- **Do NOT access the database inside `convergence/engine.go`.** The module must remain a pure computation. Pass all required data as function parameters from `iteration/engine.go`.

- **Do NOT treat a single stop condition as sufficient for convergence.** The system must meet all three content conditions simultaneously (no new critical risks + plan complete + low confidence delta) before declaring convergence — OR hit a user-approval/max-iter early exit.

- **Do NOT use floating-point equality (`==`) for confidence comparison.** Always compute the absolute delta and compare to the threshold. Identical-looking float values may not be bit-equal.

- **Do NOT short-circuit convergence on the first iteration.** Iteration 1 transitions from an empty baseline; content checks will not pass. The engine naturally handles this since `prev` and `next` will differ significantly, but code must not special-case `iteration == 1`.

## Checklist

```
[ ] convergence.Check(prev, next CanonicalState) bool — pure function, no DB/HTTP
[ ] All three AND-conditions checked: no new critical risks + plan complete + ConfidenceDelta < threshold
[ ] Early-exit OR-conditions (approved status, maxIter) injected as parameters
[ ] ConfidenceDelta = math.Abs(next.Metrics.Confidence - prev.Metrics.Confidence)
[ ] Threshold read from platform/config.GetConvergenceThreshold() (default 0.02)
[ ] HasNewCriticalRisks uses normalized text hash, not object pointer comparison
[ ] IsExecutionPlanComplete checks non-empty description AND no OpenQuestions reference
[ ] No convergence.Check import of internal/modules/state/repository or any DB package
```

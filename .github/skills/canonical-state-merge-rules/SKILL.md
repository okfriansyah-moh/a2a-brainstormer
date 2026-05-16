---
name: canonical-state-merge-rules
type: skill
description: "Enforces the union-dedup, stability-lock, and vague-output-rejection rules for CanonicalState merges in backend/modules/state/merge.go."
---

## Purpose

This skill governs `backend/modules/state/merge.go` and `backend/modules/state/validator.go`. The canonical state is the shared JSON object that flows through the entire pipeline — every agent reads it and writes an updated version. Incorrect merge logic causes risk loss, duplicate plan steps, oscillation, or silent data corruption. Rules come from `docs/PLAN.md §8.1`, `§8.5`, and `docs/A2A-agent-Brainstorm.md §10`.

## Rules

1. **`CanonicalState` struct fields must exactly match `docs/PLAN.md §8.1` JSON shape.** All `json` tags must match the schema: `idea`, `architecture`, `execution_plan`, `risks`, `assumptions`, `open_questions`, `metrics.confidence`, `meta.iteration`, `meta.agents`. Fixed keys `agentA`/`agentB` do NOT exist in `meta` — use the `agents []AgentMeta` slice.

2. **Risks are merged by union-dedup using normalized text hash.** In `merge.go:Merge`, combine `base.Risks` and `incoming.Risks`; deduplicate by hashing the lowercased, whitespace-normalized risk text. Never drop a risk that appears in either list unless it is marked `resolved: true`.

3. **Resolved risks are removed after dedup.** After union, filter out any risk with `resolved == true`. This ensures resolved issues do not resurface in subsequent iterations.

4. **Duplicate execution plan steps are collapsed.** Steps with identical titles (case-insensitive, trimmed) are merged — keep the one with the longer/more detailed description. Do not keep both.

5. **Vague plan steps are rejected.** Any step whose `description` contains fewer than 10 words (split on whitespace) is dropped from `execution_plan` during merge. This is enforced in `merge.go`, not just in `validator.go`.

6. **Stability lock prevents overwriting agreed-upon fields.** If `prev.Architecture` and `next.Architecture` hold the exact same value (deep equal), the merge output keeps the locked value and does not overwrite it with any incoming variant. This applies to `architecture`, `assumptions`, and individual `execution_plan` step descriptions.

7. **Persistent conflict escalates to `open_questions`.** If the same top-level field (e.g., `architecture`) has been modified in opposing directions for 3+ consecutive iterations, `merge.go` must append a descriptive entry to `state.OpenQuestions` flagging it for user resolution.

8. **`validator.go:Validate` is called by the iteration engine after every merge.** It rejects: empty `idea`, `confidence` outside `[0.0, 1.0]`, `meta.agents` slice with fewer than 1 entry. Return a typed error; do not panic.

## Anti-Patterns

- **Do NOT merge by replacing `base` with `incoming` wholesale.** A naive `state = agentOutput` discards risk history, locked fields, and open questions accumulated in previous iterations.

- **Do NOT use pointer equality or `==` for risk deduplication.** Two `Risk` structs with identical text are duplicates regardless of their Go object identity. Use normalized text hash (`strings.ToLower` + `strings.Fields` join).

- **Do NOT store skill prompt fragments in `AgentMeta.Skills`.** The `skills` field in `meta.agents` stores skill **names only** (e.g., `"Security Review"`) for observability. Prompt text belongs in `agent.BuildSystemPrompt`, never in state.

- **Do NOT call `merge.go` functions from outside `modules/state/` and `modules/iteration/`.** Other modules (e.g., `convergence/`) receive `CanonicalState` as a value; they must not mutate or re-merge it.

## Checklist

```
[ ] CanonicalState json tags match docs/PLAN.md §8.1 exactly; no agentA/agentB keys
[ ] Risks merged as union by normalized text hash; no loss of unique risks
[ ] resolved:true risks removed after union dedup
[ ] Duplicate plan steps collapsed (keep more detailed); not duplicated
[ ] Plan steps with description < 10 words dropped during merge
[ ] Stability lock: identical prev/next field values preserved unchanged
[ ] 3+ iteration oscillation on same field → appended to open_questions
[ ] validator.go:Validate called after every merge; rejects empty idea, confidence out of range
[ ] AgentMeta.Skills stores names only, not prompt fragments
```

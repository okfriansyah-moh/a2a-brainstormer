# Section-Sequential AI Document Generation Design

Date: 2026-06-26
Status: Proposed

## Objective

Improve finalize-time document quality in hybrid/AI mode by replacing the current monolithic per-document LLM rewrite with a **section-sequential enhancement pipeline**: enhance one rubric section at a time, merge deterministically, repair only failing sections, then run an **always-on coherence audit** with finding-first micro-fixes protected by hard guardrails.

**v1 rollout:** `architecture` and `plan` only. `readme` retains the current monolithic `enhanceOne` path until v2.

## Problem Statement

Today `aigen.Generator.enhanceOneWithOpts` sends the **entire** deterministic scaffold to the LLM in one call and asks for a 600–1000+ line rewrite. The rubric validator (`rubric.go`) already checks quality **per section**, but generation and repair operate on the **whole document**. This mismatch causes:

- Shallow middle sections when the model loses focus across 16 architecture headings
- Expensive whole-doc repair passes that re-touch good sections
- Partial alignment with `docs/PLAN.md` §8.27 original intent ("per-document multi-section LLM pass")

The deterministic generators (`generator_architecture.go`, `generator_plan.go`) already build documents section-by-section. The AI layer should mirror that granularity.

## Locked Design Decisions

| # | Decision |
|---|----------|
| 1 | **Scope:** AI finalize only — changes confined to `markdown/aigen/` and orchestrator wiring. Brainstorm agents and canonical state merge are untouched. |
| 2 | **Order:** Sequential — one section per LLM call; each call receives prior enhanced sections as context. |
| 3 | **Post-merge:** Deterministic merge + section-scoped rubric repair. |
| 4 | **Coherence:** Always-on audit call per doc; micro-fixes only when audit returns findings. |
| 5 | **Coherence guardrails:** Structural freeze, length floor, rubric non-regression, no-expand-without-finding, fix scope cap. |
| 6 | **v1 rollout:** `architecture` + `plan` only; `readme` unchanged. |

## Current State

Relevant files:

- `backend/internal/modules/markdown/aigen/generator.go` — monolithic `enhanceOneWithOpts`
- `backend/internal/modules/markdown/aigen/rubric.go` — `SectionRule`, `extractSectionBody`, `Validate`
- `backend/internal/modules/markdown/aigen/progress.go` — `DocStep` enum
- `backend/internal/modules/markdown/orchestrator.go` — `GenerateAll` / `GenerateAllWithProgress`
- `backend/internal/modules/markdown/generator_architecture.go` — 16 deterministic sections
- `backend/internal/modules/markdown/generator_plan.go` — 8 deterministic sections
- `backend/internal/modules/session/handler.go` — finalize endpoint, SSE `doc.phase` / `doc.token`

Flow today:

```
GenerateAll (deterministic scaffolds)
  → enricher pre-pass (one LLM call per doc type)
  → enhanceOne (one LLM call rewrites entire doc)
  → rubric validate (per-section)
  → repair loop (rewrites entire doc)
```

## Proposed Flow

```
GenerateAll (deterministic scaffolds)
  → enricher pre-pass (unchanged)
  → sectionSequentialEnhance (NEW — per doc key in v1 scope)
      for each SectionRule in RubricFor(docKey).Sections (in order):
        extract scaffold slice
        LLM enhance section (prior enhanced sections as context)
        validate SectionRule → section-scoped repair if needed
        on unrecoverable failure → fallback to deterministic slice
  → deterministic merge (preamble + enhanced sections)
  → full-doc rubric validate
  → section-scoped repair for any remaining failures
  → coherence audit (always — structured JSON findings only)
  → for each finding → guarded micro-fix on that section only
  → guardrail validation + final rubric
  → return merged document
```

`readme` and unknown doc keys continue through the existing `enhanceOneWithOpts` path.

## Module Responsibility

### New: `aigen/section_enhancer.go`

Owns the section-sequential loop for a single document key:

- Input: `docKey`, scaffold `GeneratedDocument`, `CanonicalState`, `EnhanceOpts`
- Output: enhanced `GeneratedDocument`
- Uses `RubricFor(docKey).Sections` as the ordered section registry
- Delegates per-section LLM calls to existing prompt builders (extended, not replaced)

### New: `aigen/section_split.go`

Pure functions — no LLM, no DB:

- `ExtractPreamble(content string, firstSectionHeading string) (preamble, body string)`
- `ExtractSection(scaffold, heading string) (headingLine, body string, ok bool)` — wraps `extractSectionBody`
- `MergeSections(preamble string, sections []EnhancedSection) string`
- `SummarizePriorSections(sections []EnhancedSection, maxTokens int) string` — token budget for context window

### New: `aigen/coherence.go`

Owns the always-on audit + guarded micro-fix pipeline:

- `Audit(docKey, mergedContent string) ([]CoherenceFinding, error)` — JSON-only LLM call
- `MicroFix(sectionHeading, sectionBody, finding CoherenceFinding) (string, error)` — section-scoped
- `ApplyCoherenceGuardrails(before, after string, rubric Rubric) (string, []GuardrailViolation)` — structural freeze, length floor, rubric non-regression

### Modified: `aigen/generator.go`

- `enhanceOneWithOpts` dispatches by doc key:
  - `architecture`, `plan` → `sectionSequentialEnhance`
  - `readme`, others → existing monolithic path (unchanged)
- Repair loop refactored: `repairSection` replaces whole-doc repair for v1 docs

### Modified: `aigen/progress.go`

Extend progress callbacks:

```go
DocStepSectionEnhance DocStep = "section_enhance"
DocStepSectionRepair  DocStep = "section_repair"
DocStepCoherenceAudit DocStep = "coherence_audit"
DocStepCoherenceFix   DocStep = "coherence_fix"
```

SSE payload extension (backward-compatible):

```json
{ "doc_key": "architecture", "step": "section_enhance", "section": "4. Layers", "detail": "..." }
{ "doc_key": "architecture", "step": "coherence_audit", "findings_count": 2 }
```

### Unchanged

- Deterministic generators (`generator_*.go`)
- Enricher pre-passes (`enricher.go`, `plan_enricher.go`, `readme_enricher.go`)
- `orchestrator.go` interface — no handler changes required
- `readme` monolithic enhance path

## Section Registry

**Source of truth:** `RubricFor(docKey).Sections` in `rubric.go`. No duplicate registry.

### Architecture (16 rubric sections)

`1. Problem Statement` through `16. Definition of Done`, plus `For AI Agents`.

### Plan (5 rubric sections)

`1. Goals`, `5. Implementation Tasks`, `7. How to Use This Plan`, `8. Deep Knowledge Reference`, `For AI Agents`.

Note: plan deterministic generator has additional sections (Milestones, Assumptions, Dependency Graph, Task Summary, Risks) that are not individually rubric-gated. v1 treats only rubric-listed headings as AI-enhance targets; other `##` blocks between rubric sections are included in the scaffold slice of the **next** rubric section or passed through deterministically. Implementation detail: split scaffold on all `##` headings, map each to nearest rubric section bucket.

### Preamble (deterministic only)

Never sent to LLM for enhancement or coherence:

- H1 title line
- Metadata block (`> ...` lines)
- Table of Contents (architecture only)

Extracted once from scaffold; re-attached at merge time.

## Per-Section Enhance Contract

### Input to each section LLM call

```
System: existing buildSystemPrompt(docKey) + section-specific instruction
User:
  - Canonical state summary (summariseState)
  - Prior enhanced sections (summarized if > token budget)
  - This section's deterministic scaffold slice
  - SectionRule constraints (MinChars, RequiredKeywords)
```

### Output

- Section body only (no `##` heading in LLM output — heading re-inserted at merge)
- Validated against `SectionRule` before advancing
- On validation failure: up to `maxRepairs` section-scoped repair calls
- On unrecoverable failure: deterministic scaffold slice for that section; log `aigen_section_fallback`

### Prior-section context budget

Config: `AIGEN_PRIOR_SECTION_MAX_CHARS` (default: 4000).

When prior sections exceed budget, summarize via deterministic extraction of first paragraph + table headers per section (no LLM summarization — preserves determinism of context assembly).

## Coherence Pass (Always Audit)

Runs after merge + section-scoped repair for `architecture` and `plan`.

### Step 1 — Audit (always, one LLM call)

Prompt contract: auditor role, JSON-only output.

```json
{
  "findings": [
    {
      "section_heading": "5. Tech Stack",
      "issue": "PostgreSQL listed in §4 Layers but MySQL in §5",
      "fix_type": "terminology",
      "confidence": "high"
    }
  ]
}
```

`fix_type` enum: `terminology` | `cross_reference` | `contradiction` | `naming`

Empty findings → coherence complete, zero additional LLM cost.

Audit response validation: must parse as JSON with `findings` array. On parse failure: retry once; on second failure skip coherence (keep pre-coherence doc).

### Step 2 — Guarded micro-fix (per finding only)

Only sections referenced in findings are sent to LLM. Sections with no findings are **never touched**.

### Guardrails (non-negotiable)

| Guardrail | Rule | On violation |
|-----------|------|--------------|
| Structural freeze | Identical `##` headings, same order, preamble unchanged | Discard entire coherence output |
| Length floor | Section body ≥ 95% pre-coherence chars (`AIGEN_COHERENCE_MIN_RATIO`, default 0.95) | Revert that section |
| Rubric non-regression | Section passing rubric pre-coherence must pass post-fix | Revert that section |
| No-expand-without-finding | Micro-fix only for sections with audit findings | N/A — enforced by dispatch |
| Fix scope cap | Edited section chars change ≤ 10% (`AIGEN_COHERENCE_MAX_EDIT_RATIO`, default 0.10) unless finding is `contradiction` | Revert that section |
| Immutable sections | `For AI Agents` — audit may flag; micro-fix skipped | Finding logged, no LLM call |

### Step 3 — Post-coherence validation

- Full rubric `Validate` on merged doc
- Per-section rubric compare before/after coherence
- Any regression → revert affected sections to pre-coherence versions

## Error Handling

| Failure | Hybrid mode behavior |
|---------|---------------------|
| Section enhance LLM error | Fallback to deterministic slice for that section |
| Section repair exhausted | Fallback to deterministic slice; `source: "ai_fallback"` on doc |
| Coherence audit parse failure (after retry) | Skip coherence; return pre-coherence doc |
| Coherence micro-fix guardrail violation | Revert that section only |
| Structural freeze failure | Discard all coherence changes |
| Context cancelled | Return best-effort merged doc assembled so far |

No change to `FinalizeModeAI` strict semantics — failures propagate as today.

## Configuration

New env vars (read via `backend/internal/platform/config/config.go` only):

| Var | Default | Purpose |
|-----|---------|---------|
| `AIGEN_SECTION_SEQUENTIAL` | `architecture,plan` | Comma-separated doc keys using section-sequential path |
| `AIGEN_COHERENCE_ENABLED` | `true` | Master switch for coherence pass |
| `AIGEN_COHERENCE_MIN_RATIO` | `0.95` | Length floor per section |
| `AIGEN_COHERENCE_MAX_EDIT_RATIO` | `0.10` | Max relative edit size per micro-fix |
| `AIGEN_PRIOR_SECTION_MAX_CHARS` | `4000` | Context budget for prior sections |

Existing vars unchanged: `FINALIZE_MODE`, `AIGEN_MAX_REPAIRS`, enricher toggles.

## Testability

Unit tests (no network, mock `LLMProvider`):

- `section_split_test.go` — preamble extraction, section merge, heading order preservation
- `section_enhancer_test.go` — sequential loop calls N times for N sections; prior context included; per-section fallback
- `coherence_test.go` — audit JSON parsing; guardrail violations trigger revert; immutable sections skipped; empty findings skip micro-fix
- `generator_test.go` — `architecture`/`plan` dispatch to section path; `readme` stays monolithic

Integration test:

- Full enhance of fixture scaffold → merged doc passes rubric; coherence audit with mock findings → only flagged sections change

## DTO / API Impact

**None.** `GeneratedDocument` shape unchanged. Finalize endpoint and SSE event types are backward-compatible (new optional `section` field on `doc.phase` events).

## Alternatives Considered

### Option 1: Section-sequential + merge only (no coherence)

Pros: simplest, fewest LLM calls.
Cons: cross-section drift (tech stack vs layers) uncorrected.
**Rejected** — user requires always-audit coherence with guardrails.

### Option 2: Parallel section batches (2 at a time)

Pros: ~2× faster per doc.
Cons: narrative inconsistency between parallel sections.
**Rejected** — user chose sequential.

### Option 3: Full-doc coherence rewrite after section merge

Pros: easy to implement.
Cons: high regression risk; undoes section-level depth gains.
**Rejected** — user explicitly requires guardrails against rewriting good content.

### Option 4 (selected): Section-sequential + finding-first coherence

Pros: focused LLM attention per section; cross-section fixes without touching clean sections; aligns with §8.27 intent.
Cons: more LLM calls per doc (~16–20 for architecture vs 1–3 today); longer finalize latency.

## Non-Goals (v1)

- Section-sequential for `readme` (deferred v2)
- Brainstorm pipeline agent changes (canonical state production)
- New output doc types
- Frontend UI changes (SSE events are backward-compatible extensions)
- Reducing rubric `MinTotalLines` / `MinChars` thresholds

## Success Criteria

- Architecture and plan docs in hybrid mode show measurably fewer rubric repair passes (target: ≤ 2 whole-doc repair equivalents via section-scoped repair)
- No section that passed rubric pre-coherence fails post-coherence (guardrail enforcement in tests)
- `readme` generation behavior identical to pre-change (regression test)
- Deterministic mode (`FINALIZE_MODE=deterministic`) byte-identical output (no code path changes)

## Implementation Plan (high level — not executed until spec approval)

1. `section_split.go` + tests — pure split/merge utilities
2. `section_enhancer.go` + tests — sequential loop with mock LLM
3. Refactor `generator.go` dispatch + section-scoped repair
4. `coherence.go` + tests — audit + guarded micro-fix
5. Progress/SSE extensions
6. Config getters
7. Integration test with fixture state
8. Manual smoke: finalize session with `architecture` + `plan` in hybrid mode

## Open Questions

None — all decisions locked in brainstorming session 2026-06-26.

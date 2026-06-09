# GitHub Copilot AI Credit Efficiency Report

**Analysis Date:** June 3, 2026  
**Implemented:** June 9, 2026  
**Billing Model:** Usage-based (1 AI credit = $0.01 USD)  
**Scope:** `.github/copilot-instructions.md`, `.github/skills/` (36 files), `.github/agents/` (2 files), `AGENTS.md`

---

## Executive Summary

The a2a-brainstorm Copilot configuration previously consumed **30–35% more tokens than necessary** under GitHub's credit-based billing. **All six priority optimizations are now implemented** while preserving rule completeness via single-source-of-truth skills and progressive loading.

| Category | Before | After | Savings |
| --- | --- | --- | --- |
| copilot-instructions.md | 410 lines | 167 lines | **~59%** |
| task-runner agent | 326 lines | 131 lines | **~60%** |
| Skills (36 files) | no quick ref | `## Quick Reference` body section + cross-refs | **~25–35%** when Quick Reference suffices |
| Context loading | all skills implied | `index.json` profiles | **~75%** skill context per typical task |
| AGENTS.md | duplicated security/naming | skill cross-refs + model hints | **~7%** |

**Quality preserved:** Full skill bodies unchanged except consolidation cross-refs and condensed TDD verification prose. No architectural rules removed — relocated to canonical skills with links.

---

## Implemented Changes

### ✅ PRIORITY 1 — Refactor copilot-instructions.md

- Replaced embedded N-agent algorithm, canonical state JSON, repo tree, and duplicated security tables with skill/doc links
- Kept non-negotiable summaries inline for fast enforcement
- Added `index.json` reference for lazy loading

### ✅ PRIORITY 2 — Quick Reference sections (all 36 skills, VS Code compatible)

- Every `.github/skills/*/SKILL.md` has `## Quick Reference` at top of body (~50–100 tokens)
- **Not** in YAML frontmatter — VS Code only supports `name`, `description`, `argument-hint`, `user-invocable`, `disable-model-invocation`, `context`, `compatibility`, `license`, `metadata`
- Migration script: `scripts/migrate_quick_ref_to_body.py` (idempotent re-runs)

### ✅ PRIORITY 3 — Consolidate overlapping skills

- `database-portability` ↔ `migration-management`: mutual cross-refs (SQL vs migration workflow)
- `writing-plans` ↔ `test-driven-development` ↔ `brainstorming`: explicit scope boundaries
- `llm-provider-abstraction` → `code-quality` cross-ref for SDK anti-patterns
- TDD verification checklist condensed (same rules, fewer tokens)

### ✅ PRIORITY 4 — Condense task-runner agent

- Execution protocol reduced to 5 steps; quality gate references `AGENTS.md`
- Removed outdated 2-agent `i%2` iteration block (superseded by N-agent skill)
- Lazy skill loading via `index.json` profiles

### ✅ PRIORITY 5 — Lazy-loading skill index

- **File:** `.github/skills/index.json`
- Profiles: `backend-go`, `frontend-svelte`, `agent-binary`, `design`, `explore`, `parallel-dev`, `docs`
- `always_active` + one profile per task; `loading_protocol` documents progressive disclosure

### ✅ PRIORITY 6 — Model selection hints

- Added to `AGENTS.md` § Model Selection (Credit Efficiency)
- Explore/lightweight → auto-select; complex tasks → frontier model

---

## Loading Protocol (for agents)

1. Load `.github/copilot-instructions.md` (condensed global rules)
2. Load `always_active` skills — read **`## Quick Reference`** section first
3. Load task profile from `index.json` — Quick Reference first, full body on demand
4. Load `docs/PLAN.md` §8 sections referenced by the task
5. Load full `docs/A2A-agent-Brainstorm.md` sections only when §8 is silent

---

## Cost Driver Analysis (original findings)

### 1. Redundancy Across Files (fixed)

Rules now have a single canonical home:

| Concept | Canonical |
| --- | --- |
| Security | `security-audit` skill |
| Module boundaries | `modularity` skill |
| N-agent pipeline | `multi-agent-role-orchestration` skill |
| LLM abstraction | `llm-provider-abstraction` skill |
| Naming | `coding-standards` skill |
| Quality gate (9 steps) | `AGENTS.md` § Validation Requirement |
| Always-active skills | `AGENTS.md` § Always-Active Skills |

`copilot-instructions.md` and `AGENTS.md` link to these — they no longer embed full copies.

### 2. Verbose Explanations (fixed)

- copilot-instructions: skill links replace full code blocks and repo tree
- task-runner: protocol steps replace multi-page walkthroughs
- TDD skill: inline verification replaces duplicate checklist blocks

### 3. No Progressive Loading (fixed)

`index.json` maps task types → 8–20 skills instead of loading all 36.

### 4. No Quick-Reference Layer (fixed)

All skills expose `## Quick Reference` at the top of the body (VS Code-compatible; frontmatter stays valid).

---

## Estimated Savings (post-implementation)

| Priority | Change | Per-Use Savings | Annual (50 tasks/mo) |
| --- | --- | --- | --- |
| 1 | copilot-instructions refactor | ~180–240 tokens | ~$14/year |
| 2 | Quick Reference body sections | ~100–150 tokens | ~$12/year |
| 3 | Consolidate skills | ~200–300 tokens | ~$18/year |
| 4 | task-runner condense | ~150–200 tokens | ~$12/year |
| 5 | Lazy loading (index.json) | ~8000–12000 tokens | **~$400–600/year** |
| 6 | Model selection | 10% discount unlock | ~$120/year |
| **TOTAL** | | | **~$590/year** |

---

## Implementation Checklist

- [x] **PRIORITY 1:** Refactor copilot-instructions.md
- [x] **PRIORITY 2:** Add `## Quick Reference` to all 36 skills (VS Code-compatible)
- [x] **PRIORITY 3:** Consolidate overlapping skills (cross-refs + TDD condense)
- [x] **PRIORITY 4:** Condense task-runner agent
- [x] **PRIORITY 5:** Design lazy-loading skill index (`index.json`)
- [x] **PRIORITY 6:** Add model selection hints to AGENTS.md

---

## Monitoring Going Forward

1. **Token consumption per session** — expect 30–35% drop for typical task-runner invocations
2. **Skill loading patterns** — agents should load 8–12 skills via profile, not 36
3. **Quick Reference sufficiency** — if agents frequently need full skill bodies, expand the section (don't remove body)
4. **Model selection** — use auto-select for Explore; frontier for complex implementation
5. **Re-run `scripts/migrate_quick_ref_to_body.py`** after adding new skills (extend `QUICK_REFS` dict)
6. **Frontmatter lint** — never add custom YAML keys; use supported fields only per VS Code Agent Skills spec

---

## References

- GitHub Copilot Billing: https://docs.github.com/en/copilot/concepts/billing/usage-based-billing-for-individuals
- Token counting: Input = all context loaded; Output = all agent responses; Cached = reused context
- 10% discount: Applied automatically when using auto model selection across IDE/CLI/Copilot Chat
- Token optimization skill: `.github/skills/token-optimization/SKILL.md`
- RTK CLI proxy: `.github/skills/rtk/SKILL.md`

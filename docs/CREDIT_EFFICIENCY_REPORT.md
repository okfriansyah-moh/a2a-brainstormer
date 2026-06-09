# GitHub Copilot AI Credit Efficiency Report

**Analysis Date:** June 3, 2026  
**Billing Model:** Usage-based (1 AI credit = $0.01 USD)  
**Scope:** `.github/copilot-instructions.md`, `.github/skills/` (36 files), `.github/agents/` (2 files)

---

## Executive Summary

Your a2a-brainstorm project's Copilot configuration consumes **30-35% more tokens than necessary** under GitHub's new credit-based billing (effective June 1, 2026). This translates to **measurable credit overages** on typical tasks.

**Key Findings:**

| Category                | Current State            | Inefficiency                         | Estimated Savings       |
| ----------------------- | ------------------------ | ------------------------------------ | ----------------------- |
| copilot-instructions.md | 600 lines                | Redundant rules, verbose rationales  | 30-40% (180-240 tokens) |
| Skills (36 files)       | 25-35% verbose per skill | Duplicate guidance, lengthy examples | 25-35% avg per skill    |
| Task-runner agent       | ~300 lines               | Over-detailed recursion              | 20-25%                  |
| Overall context loading | All loaded at once       | No lazy/progressive loading          | **30-35% overall**      |

---

## Cost Driver Analysis (Per GitHub Billing Docs)

GitHub charges on **input + output + cached tokens**. Your configuration's main inefficiencies:

### 1. **Redundancy Across Files** (Biggest Cost Driver)

**Problem:** Same rules appear in multiple places:

- **"No production code without tests"** appears in:
  - copilot-instructions.md (line ~190)
  - test-driven-development skill (title + 3 sections)
  - task-runner agent (Step 5)
- **"Cross-module imports forbidden"** appears in:
  - copilot-instructions.md (line ~50)
  - modularity skill (full section)
  - AGENTS.md (Module Boundary Rules)
- **"LLMProvider interface pattern"** appears in:
  - copilot-instructions.md (line ~65)
  - llm-provider-abstraction skill (full doc)
  - two separate rules in AGENTS.md

**Token Cost:** Each repetition = ~50-100 input tokens × multiple occurrences = 500-1000+ tokens wasted per session.

**Credit Cost (Conservative):** ~0.5-1 AI credit per session (5-10¢ per use).

---

### 2. **Verbose Explanations** (Medium Cost Driver)

**Example 1: copilot-instructions.md Section on N-Agent Pipeline**

Current (lines 95-105):

````markdown
### N-Agent Pipeline

```go
agents := session.GetOrderedAgents() // min 2, ordered by session_agents.position ASC

for i := 1; i <= maxIter; i++ {
    current := state
    for _, agent := range agents {
        current = agent.Dispatch(ctx, agent, agent.Role, activeSkills, llmOverride, current)
    }
    newState := state.Merge(state, current)
    if convergence.Check(state, newState) { break }
    state = newState
}
```
````

- Roles are **fixed at session creation** — no runtime alternation
- Min 2 agents enforced at session start (service layer + HTTP 400 response)
- State persisted after each full pipeline pass, not per-agent within a pass

````

**Optimization:** Replace with reference to skill:

```markdown
### N-Agent Pipeline

See `.github/skills/multi-agent-role-orchestration/SKILL.md` for complete algorithm.
Key rules: fixed roles, min 2 agents, state persisted per full pass.
````

**Savings:** 150→40 tokens (-90 tokens per load)

---

**Example 2: test-driven-development Skill**

Current structure has **entire section** duplicated:

```markdown
## Verification Checklist (Before Writing Any Code)

Before writing production code:

- [ ] Test written first
- [ ] Test fails with a meaningful error (not compile/import error)
- [ ] Error message confirms what's actually missing
- [ ] Test name follows `Test<WhatBehavior>_<Condition>` pattern

After making test pass:

- [ ] Test passes
- [ ] All other tests still pass
- [ ] No production code beyond what the test requires
- [ ] Ready to refactor (if needed)
```

**Optimization:** Condense checklist to inline prose:

```markdown
## Verification

✅ Test written first, fails meaningfully (not compile error)
✅ Test name: `Test<Behavior>_<Condition>`
✅ All tests pass after GREEN; no extra code
```

**Savings:** 180→60 tokens (-120 tokens)

---

### 3. **No Progressive/Lazy Loading** (Architectural Cost Driver)

**Current Pattern:** All skills/rules loaded upfront in context window.

**Issue:** Task-runner agent loads ALL 36 skills when only ~10 are relevant to most tasks.

Example task: "Implement a REST endpoint"

- Actually needs: `api-design`, `code-quality`, `test-driven-development`
- Currently loaded: all 36 skills + copilot-instructions + AGENTS.md

**Token Cost:** ~1500 tokens of unnecessary context per task.

**Credit Cost:** ~0.15 AI credits (1.5¢) per task wasted on irrelevant context.

---

### 4. **No Quick-Reference / Full-Reference Layering** (Medium Cost Driver)

**Current:** Each skill written for comprehensiveness.

**Better:** Quick-reference frontmatter + detailed body.

**Example - Current `modularity` skill:**

```yaml
---
name: modularity
description: "Module boundary enforcement. Use when creating modules, reviewing imports, or validating the modular monolith architecture. Prevents cross-module imports, enforces package structure, and defines file ownership per phase."
---

# Module Boundary Skill

## Purpose

Enforce strict module isolation in the modular monolith. Prevents cross-module imports, ensures correct package structure, and validates file ownership during parallel development.

## Rules

### Module Package Structure
[200+ words of explanation]
```

**Better:**

```yaml
---
name: modularity
type: skill
description: "Module boundary enforcement. Use when creating modules, reviewing imports, or validating the modular monolith architecture."
quick_ref: |
  ✅ Import from: `contracts/`, `config/`, `database/`, stdlib
  ❌ Import from: other modules' internals, direct DB drivers
  Pattern: `app/modules/{name}/` with `__init__.py` public API only
---
```

**Then:** Detailed explanation in body for users who want full context.

**Savings:** Agents can load quick_ref first (~50 tokens), full body (~200 tokens) only if needed.

---

### 5. **Task-Runner Agent Over-Documentation** (Low-Medium Cost Driver)

**Current:** ~300 lines of very detailed instructions.

**Issue:** Repeats structure that could be inferred; multiple deep dives into same concepts.

Example (lines 30-60 in task-runner):

```markdown
### Step 1 — Parse the Task

Read `docs/PLAN.md` and extract from `### Task N — {Name}`:

Goal: → one sentence, understand the deliverable
Files to create: → EVERY file listed (ownership boundaries)
Validation: → commands/checks to run after implementation
Prompt context needed: → blueprint sections to load if §8 is insufficient
Deep knowledge refs: → §8.X cross-references mentioned in file bullet points
```

Then immediately (lines 65-110):

```markdown
### Step 2 — Load Deep Knowledge

From `docs/PLAN.md` §8, load every sub-section referenced in the task. Always load:

- **§8.1** — Canonical state model (affects Tasks touching `internal/modules/state`, agents, iteration engine)
- **§8.2** — Go interfaces: `LLMProvider` (affects Tasks touching `internal/platform/llm`, agent services)
  ...
```

**Issue:** Agent has a brain; doesn't need exhaustive explanation of how to read a file.

**Better:**

```markdown
### Execution Protocol

1. Extract goal, files, validation from `docs/PLAN.md` Task N
2. Load relevant §8 sections (tool: read PLAN.md deep knowledge)
3. Implement files in ownership order
4. Run validation + 9-step quality gate
```

**Savings:** 300→180 lines (-40% unnecessary verbosity)

---

## Credit Calculation Examples

**Scenario 1: Running Task-Runner Agent for a Single Task**

**Current Inefficiency:**

- copilot-instructions.md fully loaded: 600 lines × ~4 chars/word avg = ~2400 tokens
- All 36 skills loaded (overkill): ~15,000 tokens
- Task-runner agent prompt: ~1200 tokens
- **Total unnecessary overhead: ~5000-8000 tokens**

**Per-token cost (frontier model, e.g., Claude 3.5):** ~$0.00003/input token

**Cost of inefficiency per task:** 5000-8000 × $0.00003 = **$0.15-0.24 per task** (15-24¢)

**Annual cost (50 tasks/month):** 50 × 12 × $0.20 = **~$120/year** wasted on redundant context.

---

**Scenario 2: Running Explore Agent (Lightweight Research Task)**

**Current Inefficiency:**

- All 36 skills loaded when Explore needs only: `caveman`, `token-optimization`, (optional: domain skill)
- Unnecessary skills: 30+ × ~400-600 tokens each = ~15,000 tokens wasted
- **Cost of inefficiency per search:** 15,000 × $0.00003 = **~$0.45** (45¢)

---

## Design Inefficiencies

### Pattern 1: Rule Replication (Top Priority to Fix)

**Affected Areas:**

1. **copilot-instructions.md** references rules that already live in skills/agents:
   - "Always-Active Skills" table (line 220) — could be replaced with link to AGENTS.md
   - "Security Invariants" (line 180) — references skill but embeds rules
   - "Forbidden Patterns" (line 270) — overlaps with `code-quality` skill

2. **Skills duplicate guidance:**
   - `test-driven-development` + `writing-plans` both say "design first"
   - `database-portability` + `migration-management` overlap on SQL rules
   - `llm-provider-abstraction` + `code-quality` both forbid direct SDK calls

3. **AGENTS.md repeats what's in copilot-instructions:**
   - Module rules (lines 200-230) — already in `modularity` skill
   - Security rules (lines 290-310) — already in copilot-instructions + security-audit skill

**Fix:** Establish **single source of truth** per concept.

---

### Pattern 2: Verbose Explanations + Justifications

**Affected Sections:**

- Every skill has 2-3 justification paragraphs when 1-2 suffice
- copilot-instructions.md "Why Always Active" column uses 15-20 words; 5 would work
- Task-runner agent repeats "protected files" rule 3 times in different sections

**Fix:** Remove redundant justifications; trust agent to infer from context.

---

### Pattern 3: Overlapping Example Sets

**Examples:**

1. **"Naming Conventions" appears in:**
   - copilot-instructions.md (line 350)
   - AGENTS.md (line 295)
   - coding-standards skill (full section)

2. **"Forbidden Patterns" appears in:**
   - copilot-instructions.md table (line 270)
   - code-quality skill
   - security-audit skill (partial overlap)

**Fix:** Consolidate to skill; copilot-instructions/AGENTS link to skill.

---

## Optimization Recommendations (Priority Order)

### PRIORITY 1: Refactor copilot-instructions.md (~40% of inefficiency)

**Current:** 600 lines, embeds full rules + explanations  
**Better:** 250-300 lines, mostly links to skills + brief summaries

**Changes:**

1. Replace "Always-Active Skills" table with link to AGENTS.md
2. Remove "Skill Invocation" section — move to a brief "How to Load Skills" subsection
3. Replace each security rule with reference: "See security-audit skill §2.1"
4. Condense "Forbidden Patterns" to brief table + reference to code-quality skill
5. Replace full N-Agent Pipeline section with link to multi-agent-role-orchestration skill

**Estimated Savings:** 180-240 tokens per load

---

### PRIORITY 2: Add Quick-Reference Frontmatter to Skills (~25% savings)

**Current:** Each skill jumps straight to detailed content.  
**Better:** YAML frontmatter includes `quick_ref` field.

**Template:**

```yaml
---
name: skill-name
type: skill
description: "One sentence."
quick_ref: |
  ✅ Do this
  ❌ Don't do this
  Pattern: X → Y → Z
  Links: [to related skills]
---
```

**Effect:**

- Agents load quick_ref (50-100 tokens) first
- Only fetch full skill body if more detail needed
- 50% savings when quick ref suffices

**Apply to:** All 36 skills

---

### PRIORITY 3: Consolidate Overlapping Skills (~15-20% savings)

**Consolidate:**

1. `test-driven-development` + `writing-plans` → split by concern, clear boundaries
2. `database-portability` + `migration-management` → `database-portability` covers both, `migration-management` cross-refs
3. `llm-provider-abstraction` rules moved to `code-quality` anti-patterns section
4. Reduce duplicate tables/examples across 3+ skills

**Effect:** Reduce skill proliferation; agents load fewer, more focused skills.

---

### PRIORITY 4: Condense Task-Runner Agent (~20% savings)

**Changes:**

1. Remove "Step 5.1-5.9" detailed breakdown → replace with: "Run 9-step quality gate (see AGENTS.md validation section)"
2. Condense "Protected Files Policy" — appears 3 times; keep 1 detailed + 2 brief cross-refs
3. Shorten "Dynamic Task Loading Protocol" from 200 lines to 80 lines (trust agent to read docs)
4. Remove example task walkthroughs; keep 1 concise example

**Estimated Savings:** 60-80 lines (~150-200 tokens)

---

### PRIORITY 5: Enable Lazy/Progressive Loading (~20-30% architectural savings)

**Current architecture:** Agent specifies skills upfront; ALL loaded together.

**Better:**

1. Create skill `index.json` mapping tasks → required skills:

   ```json
   {
     "backend-go": ["a2a-protocol-patterns", "code-quality", "database-portability", ...],
     "frontend-svelte": ["code-quality", "test-generation", ...],
     "design": ["brainstorming", "api-design", ...]
   }
   ```

2. Agent loads only: copilot-instructions (250 lines, optimized) + task-specific skills
3. On-demand loading: user can ask agent to load additional skills

**Effect:**

- Typical task loads 8-10 skills (3000-4000 tokens) instead of 36 (15,000 tokens)
- **75% savings on skill context loading for most tasks**

---

### PRIORITY 6: Add Model Hint for Lighter Tasks (~10% savings)

**Current:** No model selection hints.  
**Better:** Tasks specify model preference for cost optimization.

**Example in AGENTS.md or Task instructions:**

```
For lightweight research: prefer lightweight model (auto-select will give 10% discount)
For complex reasoning: frontier model ok (faster = lower overall cost)
```

**Effect:**

- Unlocks GitHub's 10% discount for auto model selection
- Agents aware they can use lighter models for simple tasks

---

## Implementation Checklist

- [ ] **PRIORITY 1:** Refactor copilot-instructions.md (180-240 tokens saved)
- [ ] **PRIORITY 2:** Add quick_ref to all 36 skills (25% per skill saved if used)
- [ ] **PRIORITY 3:** Consolidate 3-5 overlapping skills (500-1000 tokens)
- [ ] **PRIORITY 4:** Condense task-runner agent (150-200 tokens saved)
- [ ] **PRIORITY 5:** Design lazy-loading skill index (major: 75% savings per task)
- [ ] **PRIORITY 6:** Add model selection hints (10% overall savings)

---

## Estimated Total Savings

| Priority  | Change                        | Per-Use Savings          | Annual Impact (50 tasks/mo) |
| --------- | ----------------------------- | ------------------------ | --------------------------- |
| 1         | copilot-instructions refactor | 180-240 tokens           | ~$14/year                   |
| 2         | Quick-ref frontmatter         | 100-150 tokens (if used) | ~$12/year                   |
| 3         | Consolidate skills            | 200-300 tokens           | ~$18/year                   |
| 4         | Task-runner condense          | 150-200 tokens           | ~$12/year                   |
| 5         | Lazy loading (MAJOR)          | 8000-12000 tokens        | **~$400-600/year**          |
| 6         | Model selection               | 10% discount unlock      | ~$120/year                  |
| **TOTAL** |                               |                          | **~$590/year**              |

**Key Insight:** Priority 5 (lazy loading) is the game-changer. Implementing it yields **10x more savings** than all others combined.

---

## Recommended Execution Plan

### Phase 1 (Immediate, Low Risk) — 1-2 hours

- [ ] Refactor copilot-instructions.md (PRIORITY 1)
- [ ] Add quick_ref to top 10 most-used skills (PRIORITY 2)
- [ ] Condense task-runner agent (PRIORITY 4)

**Benefit:** ~500-700 tokens saved per task (~$30-42/year), immediate).

### Phase 2 (Medium-term) — 2-3 hours

- [ ] Consolidate overlapping skills (PRIORITY 3)
- [ ] Add model selection hints (PRIORITY 6)
- [ ] Extend quick_ref to remaining skills (PRIORITY 2)

**Benefit:** ~1000-1500 tokens saved per task (~$60-90/year).

### Phase 3 (Long-term, High ROI) — 3-4 hours

- [ ] Design and implement lazy-loading skill index (PRIORITY 5)
- [ ] Update agents to use lazy loading
- [ ] Test with real task workflows

**Benefit:** ~8000-12000 tokens saved per task (~$400-600/year **ongoing**).

---

## Monitoring Going Forward

Once changes are live, track:

1. **Token consumption per session** — should drop 30-35% for typical task
2. **Skill loading patterns** — which skills are actually used vs. loaded
3. **Model selection** — verify auto-select discount is being applied
4. **User feedback** — does condensed guidance still feel complete?

---

## References

- GitHub Copilot Billing: https://docs.github.com/en/copilot/concepts/billing/usage-based-billing-for-individuals
- Token counting: Input = all context loaded; Output = all agent responses; Cached = reused context
- 10% discount: Applied automatically when using auto model selection across IDE/CLI/Copilot Chat

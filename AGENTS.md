# AGENTS.md — Agent & Skill Governance

> **Scope:** All agents, skills, and AI-assisted workflows in the `a2a-brainstorm` codebase.
> **Source of Truth:** `docs/A2A-agent-Brainstorm.md` (architecture) + `docs/PLAN.md` (tasks)

---

## Overview

This document governs how GitHub Copilot agents and skills are structured, loaded, and composed in this project. It is the primary reference for anyone creating a new agent, skill, or AI-assisted workflow.

---

## Reference Documents

| Document                          | Purpose                                                                                          |
| --------------------------------- | ------------------------------------------------------------------------------------------------ |
| `docs/A2A-agent-Brainstorm.md`    | **Single source of truth.** Architecture, modules, API, data flows, A2A interaction model.       |
| `docs/PLAN.md`                    | Implementation plan — 15 tasks with exact files, validation steps, and deep knowledge reference. |
| `.github/copilot-instructions.md` | Copilot global rules enforced on every session — invariants, stack, forbidden patterns.          |
| `.github/agents/`                 | Agent definitions (`.agent.md` files). Each file is one deployable agent mode.                   |
| `.github/skills/`                 | Skill definitions (`SKILL.md` files). Skills are pre-digested knowledge packages.                |

---

## Agent Registry

Agents are defined in `.github/agents/`. Each `.agent.md` file declares one agent mode with a YAML frontmatter header and a Markdown body.

### Current Agents

| Agent         | File                                  | Purpose                                                                                         |
| ------------- | ------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `task-runner` | `.github/agents/task-runner.agent.md` | Implements tasks from `docs/PLAN.md`. Production-ready code per task. Non-interactive.          |
| `Explore`     | `.github/agents/Explore.agent.md`     | Read-only codebase explorer. Finds patterns, traces data flows, answers questions. Never edits. |

### Agent Frontmatter Schema

Every agent file **must** include this frontmatter:

```yaml
---
name: <agent-name> # machine-readable, kebab-case
description: "..." # one sentence; appears in agent picker
argument-hint: "..." # example invocation shown to the user
tools: [...] # explicit tool allowlist
---
```

### Adding a New Agent

1. Create `.github/agents/<name>.agent.md`
2. Define frontmatter (name, description, argument-hint, tools)
3. Write the agent body:
   - `## Role` — what this agent does and its constraints
   - `## Skills Used` — list of skill file paths this agent loads
   - `## Subagents Used` — downstream agents it may invoke via `runSubagent`
   - `## Execution Model` — step-by-step workflow
4. Register the agent in the **Current Agents** table above
5. Add it to the **Agent–Skill Composition** matrix in this file

### Agent Rules

- **One responsibility per agent** — agents must not overlap in scope
- **Never modify docs/** — agents may read `docs/` files but never write to them
- **`docs/PLAN.md` is the task source** — agents do not invent tasks; they implement from the plan
- **Non-interactive by default** — agents must complete their assigned work without mid-session questions unless the agent is explicitly interactive (e.g., `Explore`)
- **Load skills before reasoning** — always load relevant skill files first; skills are cheaper context than re-reading raw docs

---

## Skill Registry

Skills are defined in `.github/skills/<name>/SKILL.md`. Skills are pre-digested knowledge packages — compact, opinionated, and reusable across agents.

### Current Skills

| Skill                            | Path                                             | Domain                                                             |
| -------------------------------- | ------------------------------------------------ | ------------------------------------------------------------------ |
| `a2a-protocol-patterns`          | `.github/skills/a2a-protocol-patterns/`          | Correct usage of `a2a-go/v2` — client, server, executor, DataPart  |
| `api-design`                     | `.github/skills/api-design/`                     | REST endpoint design, request/response contracts, versioning       |
| `brainstorming`                  | `.github/skills/brainstorming/`                  | Design-first gate before any implementation work                   |
| `canonical-state-merge-rules`    | `.github/skills/canonical-state-merge-rules/`    | Union-dedup, stability-lock, and vague-output-rejection for state  |
| `caveman`                        | `.github/skills/caveman/`                        | Ultra-compressed output (~65-75% fewer tokens)                     |
| `code-quality`                   | `.github/skills/code-quality/`                   | Type annotations, structured logging, anti-patterns                |
| `coding-standards`               | `.github/skills/coding-standards/`               | Naming, function design, Go idioms                                 |
| `config-validation`              | `.github/skills/config-validation/`              | YAML config, no hardcoded values, config-driven parameters         |
| `conflict-resolution`            | `.github/skills/conflict-resolution/`            | Git merge conflict resolution for parallel branches                |
| `convergence-engine-patterns`    | `.github/skills/convergence-engine-patterns/`    | Multi-condition convergence detection, delta-confidence threshold  |
| `database-portability`           | `.github/skills/database-portability/`           | Portable SQL (pgx/sqlc), no ORM, cross-engine compatibility        |
| `dependency-analysis`            | `.github/skills/dependency-analysis/`            | Import graph, circular dependency detection, coupling metrics      |
| `determinism`                    | `.github/skills/determinism/`                    | Same input + config = identical output; no randomness              |
| `docs-sync`                      | `.github/skills/docs-sync/`                      | Detect drift between code and documentation                        |
| `dto`                            | `.github/skills/dto/`                            | DTO registry, immutability, producer/consumer rules                |
| `failure`                        | `.github/skills/failure/`                        | Retry logic, abort thresholds, graceful degradation                |
| `idempotency`                    | `.github/skills/idempotency/`                    | `ON CONFLICT DO NOTHING`, content-addressable IDs                  |
| `llm-provider-abstraction`       | `.github/skills/llm-provider-abstraction/`       | `LLMProvider` interface, tiered config resolver, env-ref creds     |
| `migration-management`           | `.github/skills/migration-management/`           | Portable, reversible, sequential SQL migrations                    |
| `modularity`                     | `.github/skills/modularity/`                     | Module boundary enforcement, vertical slice, no cross-imports      |
| `multi-agent-role-orchestration` | `.github/skills/multi-agent-role-orchestration/` | N-agent ordered pipeline, role assignment, sequential state thread |
| `parallel-dev`                   | `.github/skills/parallel-dev/`                   | Parallel development orchestration, phase grouping                 |
| `performance-optimization`       | `.github/skills/performance-optimization/`       | Query optimization, memory reduction, throughput                   |
| `pipeline`                       | `.github/skills/pipeline/`                       | Stage ordering, DTO flow map, parallelism matrix                   |
| `plan-management`                | `.github/skills/plan-management/`                | PLAN.md creation, task structure, deep knowledge reference         |
| `project-scaffold`               | `.github/skills/project-scaffold/`               | Project initialization, directory structure, boilerplate           |
| `roadmap-spec`                   | `.github/skills/roadmap-spec/`                   | Phase spec structure: Objective, Scope, Function Contracts         |
| `rtk`                            | `.github/skills/rtk/`                            | Token-efficient CLI proxy (60-90% token savings)                   |
| `running-prompt`                 | `.github/skills/running-prompt/`                 | Structured task execution: plan → implement → verify               |
| `security-audit`                 | `.github/skills/security-audit/`                 | OWASP auditing, injection prevention, secrets management           |
| `subagent-driven-development`    | `.github/skills/subagent-driven-development/`    | Fresh subagent per task with two-stage spec + quality review       |
| `test-driven-development`        | `.github/skills/test-driven-development/`        | RED-GREEN-REFACTOR; no production code without a failing test      |
| `test-generation`                | `.github/skills/test-generation/`                | Unit/integration test patterns, mocking, coverage requirements     |
| `token-optimization`             | `.github/skills/token-optimization/`             | Progressive context loading, skill-first, no redundant reads       |
| `vertical-slice`                 | `.github/skills/vertical-slice/`                 | Feature-per-folder: handler/service/repository/model per module    |
| `writing-plans`                  | `.github/skills/writing-plans/`                  | Convert approved design into detailed per-task implementation plan |

### Skill File Schema

Every `SKILL.md` **must** begin with YAML frontmatter:

```yaml
---
name: <skill-name> # kebab-case, matches directory name
type: skill
description: >
  One paragraph. Describes when to load this skill.
---
```

The body must contain at minimum:

- `## Purpose` — what this skill enforces
- `## Rules` — numbered, enforceable constraints
- `## Checklist` — testable exit criteria

### Adding a New Skill

1. Create `.github/skills/<name>/SKILL.md`
2. Write frontmatter (name, type, description)
3. Write body (Purpose, Rules, Checklist)
4. Register the skill in the **Current Skills** table above
5. Add it to any relevant agent in the **Agent–Skill Composition Matrix** below

### Skill Loading Rules

- **Load skills before reading raw docs** — skills are pre-digested and consume fewer tokens
- **Reference, do not repeat** — say "per `a2a-protocol-patterns` skill" instead of restating its rules
- **Progressive disclosure**: skill → doc section → full doc (load only what is needed)
- Each agent declares its skills explicitly in a `## Skills Used` section

---

## Always-Active Skills

These skills apply to **every agent and every task** without explicit loading. All agents must honour them.

| Skill                         | Reason Always Active                                                         |
| ----------------------------- | ---------------------------------------------------------------------------- |
| `brainstorming`               | Design-first gate — NEVER write code before presenting a design and approval |
| `writing-plans`               | After approval, break work into 2–5 min tasks before implementing            |
| `subagent-driven-development` | Dispatch fresh subagent per task with 2-stage spec + quality review          |
| `test-driven-development`     | No production code without a failing test first (RED-GREEN-REFACTOR)         |
| `caveman`                     | Compress output ~75% when user requests it — no filler, full accuracy        |
| `rtk`                         | Use `rtk <cmd>` for terminal output (60-90% token savings)                   |

> **Superpowers shorthand:** `brainstorming` + `writing-plans` + `subagent-driven-development` + `test-driven-development` are collectively the **superpowers** and are always active.

---

## Agent–Skill Composition Matrix

Cells marked ✅ mean the agent explicitly loads that skill.

### Implementation Agents

| Skill                            | `task-runner` | `Explore` |
| -------------------------------- | :-----------: | :-------: |
| `a2a-protocol-patterns`          |      ✅       |           |
| `api-design`                     |      ✅       |           |
| `brainstorming`                  |      ✅       |           |
| `canonical-state-merge-rules`    |      ✅       |           |
| `caveman`                        |      ✅       |    ✅     |
| `code-quality`                   |      ✅       |           |
| `coding-standards`               |      ✅       |           |
| `config-validation`              |      ✅       |           |
| `convergence-engine-patterns`    |      ✅       |           |
| `database-portability`           |      ✅       |           |
| `determinism`                    |      ✅       |           |
| `dto`                            |      ✅       |           |
| `failure`                        |      ✅       |           |
| `idempotency`                    |      ✅       |           |
| `llm-provider-abstraction`       |      ✅       |           |
| `migration-management`           |      ✅       |           |
| `modularity`                     |      ✅       |           |
| `multi-agent-role-orchestration` |      ✅       |           |
| `plan-management`                |      ✅       |           |
| `rtk`                            |      ✅       |    ✅     |
| `security-audit`                 |      ✅       |           |
| `subagent-driven-development`    |      ✅       |           |
| `test-driven-development`        |      ✅       |           |
| `test-generation`                |      ✅       |           |
| `token-optimization`             |      ✅       |    ✅     |
| `vertical-slice`                 |      ✅       |           |
| `writing-plans`                  |      ✅       |           |

---

## SubAgent Delegation Map

| Caller Agent  | Delegates To | Purpose                                |
| ------------- | ------------ | -------------------------------------- |
| `task-runner` | `Explore`    | Read-only research before writing code |

---

## Protected Files Policy

These files have strict modification rules. All agents must honour them.

| Path                           | Rule                                                                                  |
| ------------------------------ | ------------------------------------------------------------------------------------- |
| `docs/A2A-agent-Brainstorm.md` | **Read-only.** The source blueprint. Never modified after design is locked.           |
| `docs/PLAN.md`                 | **Task progress comments only** (`✅ Task N completed`). Never rewrite task bodies.   |
| `docs/PLAN.md §8`              | **Read-only during execution.** Deep knowledge reference; never edited by agents.     |
| `.github/skills/*/SKILL.md`    | **Read-only during task execution.** Skills are updated only in dedicated sessions.   |
| `.github/agents/*.agent.md`    | **Read-only during task execution.** Agents are updated only in dedicated sessions.   |
| `migrations/*.sql`             | **Append-only.** New migration files may be added; existing files are never modified. |
| `contracts/`                   | **Additive only.** New types allowed; existing types never modified.                  |

---

## File Ownership Rule

Each task in `docs/PLAN.md` lists exactly the files it owns under `### Files to create`. **An agent must only write to files owned by the currently assigned task.** If a compile error requires touching a file owned by a different task, stop and fix it within the current task's own files (compatibility shim or interface boundary).

---

## Validation Requirement

Every task and every agent session must end with a passing validation step:

| Layer        | Command                                             |
| ------------ | --------------------------------------------------- |
| Backend      | `go build ./...` + `go vet ./...`                   |
| Agent binary | `go build ./...` + `go vet ./...`                   |
| Frontend     | `pnpm check` + `pnpm build`                         |
| Tests        | `go test ./...` (backend) + `go test ./...` (agent) |

No task is complete until its validation passes.

---

## Security Invariants (All Agents Must Enforce)

These rules apply to every file any agent produces. They are not negotiable.

1. **API keys are never stored in source code, config files, or logs.** `CredentialRef` stores the env var _name_ only (e.g. `"CLAUDE_API_KEY"`); the actual key is resolved at runtime via `os.Getenv(credentialRef)`.
2. **All `os.Getenv()` calls are confined to `backend/internal/platform/config/config.go` (backend) and `agent/internal/config/config.go` (agent binary).** No other file may call `os.Getenv()`.
3. **`llm_config` JSONB column stores only `{provider, model, credential_ref}`.** The key value must never appear in the DB.
4. **Absent credential env var at startup → agent marked unavailable.** No silent fallback to another provider.
5. **No SQL string interpolation.** All queries use parameterized statements (pgx named params).
6. **Input validation on all HTTP handlers** — UUID format, non-empty required fields, bounded numeric inputs. Return `400` on violation; never pass raw user input to SQL or LLM prompts.
7. See `.github/skills/security-audit/SKILL.md` for the full OWASP checklist.

---

## Naming Conventions

Source files are named after the **domain concept or behavior** they implement — never after sprint tasks, phase labels, or ticket numbers.

| Bad (opaque)    | Good (functional)        | Reason                           |
| --------------- | ------------------------ | -------------------------------- |
| `phase4.go`     | `convergence_engine.go`  | Names the domain concept         |
| `task3_impl.go` | `llm_resolver.go`        | Describes what the code does     |
| `helpers.go`    | `prompt_assembly.go`     | Disambiguates the specific logic |
| `b2_test.go`    | `merge_strategy_test.go` | Names the behavior under test    |

---

## Module Boundary Rules

See `.github/skills/modularity/SKILL.md` for full rules. Summary:

- `backend/modules/<name>/` — vertical slice: `handler.go`, `service.go`, `repository.go`, `model.go`
- Modules communicate only via types from `backend/internal/shared/` or their own exported service interface
- No module imports another module's internal packages (no `modules/session` → `modules/agent/repository`)
- All DB access goes through the module's own repository, not another module's repository
- `backend/internal/platform/` is shared infrastructure; any module may import it

---

## Skill Invocation

To invoke a skill from a chat session or agent, reference it by path:

```
#file:.github/skills/brainstorming/SKILL.md
```

Or reference it from copilot-instructions for automatic loading:

```
See `.github/skills/<name>/SKILL.md` for rules.
```

Skills are loaded on-demand, not pre-loaded into every context window. Prefer skills over re-reading full docs.

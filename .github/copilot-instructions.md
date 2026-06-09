# Copilot Instructions — a2a-brainstorm

> Architectural constraints for `a2a-brainstorm`. Violations are not acceptable.
> **Governance:** `AGENTS.md` · **Lazy skill loading:** `.github/skills/index.json`

---

## Reference Documents

| Document | Purpose |
| --- | --- |
| `docs/A2A-agent-Brainstorm.md` | Architecture SSOT — modules, A2A, API, canonical state |
| `docs/PLAN.md` | 31-task plan + §8 deep knowledge (schemas, algorithms) |
| `AGENTS.md` | Agent/skill registry, composition matrix, protected files |
| `.github/skills/index.json` | Task-type → required skills (load only these) |
| `.github/skills/` | Pre-digested knowledge — **load before raw docs** |

**Progressive disclosure:** skill `description` → `## Quick Reference` section → skill body → doc section → full doc.

---

## Architecture Invariants

### Stack

- **Backend:** Go 1.26, modular monolith, vertical slice (`handler.go + service.go + repository.go + model.go`)
- **Agent binary:** Go 1.26, `github.com/a2aproject/a2a-go/v2`
- **Frontend:** SvelteKit, TypeScript, TailwindCSS, Svelte stores
- **Database:** PostgreSQL 16, pgx/v5, sqlc — no ORM
- **Deploy:** Docker + docker-compose, single shared agent image

### Module Communication

- Modules export service interfaces only — no cross-module internal imports
- No `map[string]any` across module boundaries — typed structs
- `backend/internal/platform/` = shared infra; `backend/internal/shared/` = shared types
- Entry: `backend/cmd/server/main.go` — single deployable, no inter-module RPC

### Domain Rules (load skill for detail)

| Topic | Skill / Doc |
| --- | --- |
| A2A wire format | `.github/skills/a2a-protocol-patterns/SKILL.md` · `docs/PLAN.md §8.3` |
| LLM calls | `.github/skills/llm-provider-abstraction/SKILL.md` · `§8.12` |
| Database / SQL | `.github/skills/database-portability/SKILL.md` |
| Determinism | `.github/skills/determinism/SKILL.md` |
| Idempotency | `.github/skills/idempotency/SKILL.md` |
| N-agent pipeline | `.github/skills/multi-agent-role-orchestration/SKILL.md` · `§8.4` |
| Canonical state | `docs/PLAN.md §8.1` — shape is non-negotiable |
| State merge | `.github/skills/canonical-state-merge-rules/SKILL.md` · `§8.5` |
| Convergence | `.github/skills/convergence-engine-patterns/SKILL.md` · `§8.6` |
| Skill injection | Server-side prompt assembly in `agent/client.go` — agent binary sees assembled `SystemPrompt` only |

### N-Agent Pipeline (summary)

Fixed roles at session creation · min 2 agents · ordered by `session_agents.position ASC` · each agent receives previous agent's output · state persisted once per full pass · no runtime role alternation.

Full algorithm: `.github/skills/multi-agent-role-orchestration/SKILL.md`

### Session / Output / Preview / SSE (v1.3)

- `POST /sessions`: `agent_ids` ≥ 2 (400 otherwise); `skill_overrides` / `role_overrides` / `output_docs` per `§8` and Task 28
- Output docs: `GenerateAll(state, keys)` registry in `markdown/generator.go` — keys: `architecture`, `roadmap`, `plan`, `readme`
- Preview/apply: `iteration/preview.go` — `POST/GET /sessions/{id}/preview/{agentID}`, `POST .../apply`
- SSE: `platform/sse/broadcaster.go` — `GET /sessions/{id}/events`; native `EventSource` in frontend; fire-and-forget

---

## Security Invariants

**Full checklist:** `.github/skills/security-audit/SKILL.md`

1. API keys never in source/config/logs — `CredentialRef` = env var **name** only
2. `os.Getenv()` only in `backend/internal/platform/config/config.go` and `agent/internal/config/config.go`
3. `llm_config` JSONB: `{provider, model, credential_ref}` only
4. Missing credential at startup → agent unavailable (no silent fallback)
5. HTTP handlers validate input (UUID, bounds) → 400 on violation
6. Parameterized SQL only — no string interpolation
7. Never log resolved credential values

---

## Always-Active Skills

See `AGENTS.md` § Always-Active Skills: `brainstorming`, `writing-plans`, `subagent-driven-development`, `test-driven-development`, `caveman`, `rtk`.

Load via `#file:.github/skills/<name>/SKILL.md` or task-specific set from `index.json`.

---

## Development Rules

1. Design before code — `brainstorming` skill
2. Vertical slice per module — `.github/skills/vertical-slice/SKILL.md`
3. No cross-module internal imports — `.github/skills/modularity/SKILL.md`
4. DB access via module `repository.go` only
5. LLM via `LLMProvider` interface only — `platform/llm/`
6. Config via env vars — getters in `config.go` only
7. Structured logging — `log/slog`; no `fmt.Println`
8. Tests without network — mocks/fakes
9. One canonical location per concept

---

## Forbidden Patterns

**Full table:** `.github/skills/code-quality/SKILL.md` · `.github/skills/security-audit/SKILL.md`

| Category | Forbidden |
| --- | --- |
| Architecture | Microservices between modules, inter-module RPC, shared mutable globals |
| Database | ORM (`gorm`, `ent`), SQL concat, driver imports in `internal/modules/` |
| LLM | Direct SDK calls in `internal/modules/` or `agent/internal/executor/` |
| Config | Hardcoded keys/ports/models; `os.Getenv` outside config files |
| State | Non-deterministic IDs from timestamps; per-agent mutable globals |
| Naming | Task-code filenames (`phase4.go`), single-letter files |
| Go format | Stray `package` line before doc comment on line 1 — check every new `.go` file |
| UI (v1.1+) | Hardcoded hex colors; new `/agents`/`/skills` pages; deprecated `AgentPanel`/`StateView`/`Timeline`; WebSocket for SSE; preview UI outside `PipelineStage` |

---

## Repository Layout (top-level)

```
backend/  agent/  frontend/  migrations/  docs/  .github/
```

Module map and file ownership: `docs/PLAN.md` tasks · `AGENTS.md` File Ownership Rule.

---

## File Naming

See `.github/skills/coding-standards/SKILL.md` — name files after domain concept (`convergence_engine.go` not `phase4.go`).

---

## Protected Files

| Path | Rule |
| --- | --- |
| `docs/A2A-agent-Brainstorm.md` | Read-only after design lock |
| `docs/PLAN.md` | Task progress comments only; §8 read-only during execution |
| `migrations/*.sql` | Append-only |
| `.github/skills/*/SKILL.md` | Read-only during task execution |
| `.github/agents/*.agent.md` | Read-only during task execution |

---

## Validation Gates

**9-step quality gate** (ordered, zero findings each): tests → security → linter → build.

Commands and todo requirements: `AGENTS.md` § Validation Requirement.

Use `rtk` for verbose command output: `.github/skills/rtk/SKILL.md`

---

## Documentation Ownership

| Topic | Canonical |
| --- | --- |
| Architecture | `docs/A2A-agent-Brainstorm.md` |
| Tasks + schemas | `docs/PLAN.md` / §8 |
| Agent/skill governance | `AGENTS.md` |
| Global Copilot rules | `.github/copilot-instructions.md` (this file) |

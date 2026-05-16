# Copilot Instructions — a2a-brainstorm

> These instructions enforce the architectural constraints for the `a2a-brainstorm` project.
> Violations are not acceptable and must not be introduced, even partially.
> **Always check `AGENTS.md` for agent/skill governance rules.**

---

## Reference Documents

| Document                       | Purpose                                                                                                               |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------- |
| `docs/A2A-agent-Brainstorm.md` | **Single source of truth.** Architecture, modules, A2A interaction model, API endpoints, canonical state, data flows. |
| `docs/PLAN.md`                 | 15-task implementation plan with exact files, validation steps, and deep knowledge reference (§8).                    |
| `AGENTS.md`                    | Agent & skill governance — registry, composition matrix, protected files policy, naming rules.                        |
| `.github/agents/`              | Agent definitions (`.agent.md`). Each file is one deployable Copilot agent mode.                                      |
| `.github/skills/`              | Skill definitions (`SKILL.md`). Pre-digested knowledge packages loaded on demand.                                     |

When generating code, refer to these documents for exact schemas, interfaces, and algorithms. Do not invent structures that contradict them.

---

## Architecture Invariants

### Stack

- **Backend:** Go 1.26, modular monolith, vertical slice per module
- **Agent binary:** Go 1.26, `github.com/a2aproject/a2a-go/v2` (requires Go ≥ 1.24.4)
- **Frontend:** SvelteKit (latest stable), TypeScript, TailwindCSS, Svelte stores
- **Database:** PostgreSQL 16, pgx/v5, sqlc — no heavy ORM
- **Deployment:** Docker + docker-compose, single shared agent image

### Modular Monolith (Backend)

- Single deployable, single repo, single database
- Entry point: `backend/cmd/server/main.go`
- No microservices, no inter-process RPC between backend modules
- Module structure: `backend/modules/<name>/handler.go + service.go + repository.go + model.go`

### Module Communication

- Modules communicate only through their own exported service interfaces
- No module imports another module's internal packages (`modules/session` must not import `modules/agent/repository`)
- No raw `map[string]any` crossing module boundaries — use typed structs
- Shared platform infrastructure lives in `backend/internal/platform/` — any module may import it
- Shared types (used by multiple modules) live in `backend/internal/shared/`

### A2A Layer

- SDK: `github.com/a2aproject/a2a-go/v2` — packages: `a2a`, `a2asrv`, `a2aclient`, `a2agrpc`
- Communication is **message-based** — no custom task JSON schema
- Domain context is packed as `a2a.NewDataPart(BrainstormPayload{})` inside `a2a.SendMessageRequest`
- Backend sends via `a2aclient.NewFromCard(ctx, card)` → `client.SendMessage()`
- Agent receives via `a2asrv.AgentExecutor.Execute(ctx, execCtx *a2asrv.ExecutorContext)`
- `BrainstormPayload` is the **only** backend↔agent wire format — see `docs/PLAN.md §8.3`

### LLM Provider Abstraction (Enforced)

- All LLM calls go through the `LLMProvider` interface — **never** call Copilot/Claude SDK directly from business logic
- Interface: `Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)`
- Tiered config resolver: session override → agent-level → global default (see `docs/PLAN.md §8.12`)
- `LLMConfig.CredentialRef` stores the **env var name only** (e.g. `"CLAUDE_API_KEY"`) — never the key value
- See `.github/skills/llm-provider-abstraction/SKILL.md`

### Database Rules

- All DB access goes through the module's own `repository.go` — no module queries another module's tables directly
- Only pgx parameterized queries — no string interpolation in SQL
- All schema changes go through `migrations/*.sql` files (numbered, sequential)
- Migrations are **append-only** — never modify an existing migration file
- Use `ON CONFLICT DO NOTHING` (not engine-specific variants)
- See `.github/skills/database-portability/SKILL.md`

### Determinism

- Same input + same config = identical output. Always.
- The iteration engine produces deterministic results: ordered pipeline, fixed roles per session
- No `rand.New()`, no `time.Now()` used for state transitions, no non-deterministic sources
- See `.github/skills/determinism/SKILL.md`

### Idempotency

- Running the iteration pipeline twice on the same input produces no duplicates
- All SQL writes use `ON CONFLICT DO NOTHING` or `ON CONFLICT DO UPDATE` semantics
- See `.github/skills/idempotency/SKILL.md`

---

## Core Business Rules (Do Not Violate)

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

- Roles are **fixed at session creation** — no runtime alternation
- Min 2 agents enforced at session start (service layer + HTTP 400 response)
- State persisted after each full pipeline pass, not per-agent within a pass

### Canonical State Shape

See `docs/PLAN.md §8.1`. The shape is non-negotiable — agents depend on it:

```json
{
  "idea": {},
  "architecture": {},
  "execution_plan": [],
  "risks": [],
  "assumptions": [],
  "open_questions": [],
  "metrics": { "confidence": 0.0 },
  "meta": {
    "iteration": 0,
    "agents": [
      {
        "agent_id": "",
        "name": "",
        "role": "",
        "provider": "",
        "model": "",
        "skills": []
      }
    ]
  }
}
```

### Skill Injection

Skills are prompt-level text fragments — not tool calls. Assembly happens server-side in `agent/client.go`:

```
effective_system_prompt = agent.system_prompt + "\n\n" + skill_1.prompt + "\n\n" + skill_2.prompt
```

The agent binary receives only the assembled `SystemPrompt` string. It has no knowledge of skill names, IDs, or the `agent_skills` table.

### Session Creation Rules

- `POST /sessions` requires `agent_ids` with ≥ 2 entries — reject with `400` otherwise
- `skill_overrides`: omitted = use agent defaults; empty array `[]` = disable all skills for that agent
- `role_overrides`: optional; if absent, use `agent.DefaultRoles(agentCount)` distribution

---

## Security Invariants (Non-Negotiable)

1. **API keys never in source, config files, or logs.** `CredentialRef` is the env var name only.
2. **`os.Getenv()` confined to one file per binary:**
   - Backend: `backend/internal/platform/config/config.go`
   - Agent: `agent/internal/config/config.go`
   - Nowhere else.
3. **`llm_config` JSONB stores only `{provider, model, credential_ref}`.** Never the key value.
4. **Absent credential at startup → agent unavailable.** No silent fallback.
5. **All HTTP handlers validate input** — UUID format, non-empty fields, bounded integers. Return `400` on violation.
6. **No SQL string interpolation.** Parameterized queries only (pgx named params).
7. **Secrets never logged.** Structured logger must not emit `CredentialRef` resolved values.
8. See `.github/skills/security-audit/SKILL.md` for the full OWASP checklist.

---

## Always-Active Skills (Apply to Every Session)

| Skill                         | File                                                  | Why Always Active                                               |
| ----------------------------- | ----------------------------------------------------- | --------------------------------------------------------------- |
| `brainstorming`               | `.github/skills/brainstorming/SKILL.md`               | Design-first gate — NEVER write code before presenting a design |
| `writing-plans`               | `.github/skills/writing-plans/SKILL.md`               | Break approved work into 2–5 min tasks before implementing      |
| `subagent-driven-development` | `.github/skills/subagent-driven-development/SKILL.md` | Fresh subagent per task with 2-stage spec + quality review      |
| `test-driven-development`     | `.github/skills/test-driven-development/SKILL.md`     | No production code without a failing test first                 |
| `caveman`                     | `.github/skills/caveman/SKILL.md`                     | Compress output ~75% on request — no filler, full accuracy      |
| `rtk`                         | `.github/skills/rtk/SKILL.md`                         | Use `rtk <cmd>` for terminal output (60-90% token savings)      |

---

## Skill Invocation

Reference skills by path in any prompt:

```
#file:.github/skills/a2a-protocol-patterns/SKILL.md
#file:.github/skills/modularity/SKILL.md
```

**Load skills before raw docs.** Skills are pre-digested; they cost far fewer tokens than re-reading `docs/A2A-agent-Brainstorm.md` in full.

**Progressive disclosure:** skill → doc section → full doc.

---

## Development Rules

1. **No production code before design.** Present a design, get approval, then implement. (`brainstorming` skill)
2. **Vertical slice per module.** `handler.go + service.go + repository.go + model.go` per module under `backend/modules/`.
3. **No cross-module internal imports.** Modules may only import `backend/internal/platform/`, `backend/internal/shared/`, and their own internal packages.
4. **All DB access through the module's own repository.** No raw SQL outside `repository.go` files.
5. **LLMProvider interface is the only LLM call boundary.** No direct SDK imports outside `platform/llm/`.
6. **Config via env vars only** — no hardcoded values, thresholds, or magic numbers. All getters in `config.go`.
7. **Structured logging only** — use `log/slog` (stdlib); no `fmt.Println`; no unstructured console output.
8. **Tests without network.** All tests must run without real DB, real LLM, or real agent endpoints. Use mocks/fakes.
9. **One canonical location per concept.** No duplicate types, no copied SQL schemas, no parallel utility files.

---

## Forbidden Patterns

| Category     | Forbidden                                                                                                                  |
| ------------ | -------------------------------------------------------------------------------------------------------------------------- |
| Architecture | Microservices between backend modules, inter-module RPC, shared mutable global state                                       |
| Database     | ORM frameworks (`gorm`, `ent`), direct driver imports in `modules/`, SQL string concat                                     |
| LLM          | Direct Copilot/Claude SDK calls in `modules/` or `agent/internal/executor/`                                                |
| Config       | Hardcoded API keys, hardcoded ports, hardcoded model names, `os.Getenv` outside config                                     |
| Credentials  | Storing raw API keys anywhere other than environment variables                                                             |
| State        | Per-agent mutable global state; non-deterministic ID generation (UUID v4 for new IDs is fine; never use timestamps as IDs) |
| Naming       | Task codes as file names (`phase4.go`, `b3_test.go`), single-letter files (`h.go`)                                         |

---

## Repository Structure

```
a2a-brainstorm/
├── AGENTS.md                        ← agent & skill governance (this context)
├── go.work                          ← Go workspace (backend + agent)
├── docker-compose.yml
├── Makefile
├── docs/
│   ├── A2A-agent-Brainstorm.md      ← architecture blueprint (read-only)
│   └── PLAN.md                      ← implementation plan (task progress only)
├── backend/
│   ├── go.mod
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── platform/
│   │   │   ├── config/              ← ALL os.Getenv() calls live here
│   │   │   ├── db/                  ← pgx pool, migration runner
│   │   │   ├── logger/              ← log/slog wrapper
│   │   │   ├── http/                ← net/http server, CORS, middleware
│   │   │   ├── llm/                 ← LLMProvider interface + impls + resolver
│   │   │   └── a2a/                 ← a2aclient factory, AgentCard resolver
│   │   └── shared/                  ← types shared across modules
│   └── modules/
│       ├── session/
│       ├── iteration/
│       ├── agent/
│       ├── state/
│       ├── convergence/
│       └── markdown/
├── agent/
│   ├── go.mod
│   ├── agentcard.go
│   ├── cmd/server/main.go
│   └── internal/
│       ├── executor/                ← BrainstormExecutor implements a2asrv.AgentExecutor
│       ├── llm/                     ← LLMProvider implementations (Copilot, Claude)
│       └── config/                  ← ALL os.Getenv() calls for agent binary live here
├── frontend/
│   └── src/
│       ├── routes/
│       └── lib/
│           ├── components/
│           ├── stores/
│           └── services/api.ts
├── migrations/                      ← SQL migration files (numbered, append-only)
└── .github/
    ├── copilot-instructions.md      ← this file
    ├── AGENTS.md                    → root AGENTS.md (canonical)
    ├── agents/                      ← agent definitions
    └── skills/                      ← skill definitions
```

---

## File Naming Standards

Name source files after the **domain concept or behavior** they implement.

| Bad (opaque)    | Good (functional)        | Reason                           |
| --------------- | ------------------------ | -------------------------------- |
| `phase4.go`     | `convergence_engine.go`  | Names the domain concept         |
| `task3_impl.go` | `llm_resolver.go`        | Describes what the code does     |
| `helpers.go`    | `prompt_assembly.go`     | Disambiguates the specific logic |
| `b2_test.go`    | `merge_strategy_test.go` | Names the behavior under test    |
| `h.go`          | `session_handler.go`     | Full word, no abbreviation       |

---

## Protected Files

| Path                           | Rule                                                                    |
| ------------------------------ | ----------------------------------------------------------------------- |
| `docs/A2A-agent-Brainstorm.md` | **Read-only.** Never modified after design lock.                        |
| `docs/PLAN.md`                 | **Task progress comments only.** Never rewrite task bodies or §8.       |
| `migrations/*.sql`             | **Append-only.** New files may be added; existing files never modified. |
| `.github/skills/*/SKILL.md`    | **Read-only during task execution.**                                    |
| `.github/agents/*.agent.md`    | **Read-only during task execution.**                                    |

---

## Validation Gates

Every implementation session must end with all applicable gates passing:

| Layer        | Gate Command                      |
| ------------ | --------------------------------- |
| Backend      | `go build ./...` + `go vet ./...` |
| Agent binary | `go build ./...` + `go vet ./...` |
| Frontend     | `pnpm check` + `pnpm build`       |
| Tests        | `go test ./...`                   |

---

## Documentation Ownership

Each concept has one canonical document. Use cross-references rather than duplicating content.

| Topic                            | Canonical Document                |
| -------------------------------- | --------------------------------- |
| System architecture & data flows | `docs/A2A-agent-Brainstorm.md`    |
| Implementation tasks             | `docs/PLAN.md`                    |
| Deep knowledge (schemas/algos)   | `docs/PLAN.md §8`                 |
| Agent & skill governance         | `AGENTS.md`                       |
| Copilot global rules             | `.github/copilot-instructions.md` |

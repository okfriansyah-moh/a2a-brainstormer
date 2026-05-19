# PLAN.md — a2a-brainstorm Implementation Plan

> **Version:** 1.1
> **Date:** 2026-05-12 (updated with UI redesign tasks)
> **Author:** Core, Data and AI Team
> **Status:** Ready for Implementation
> **Source of Truth:** `docs/A2A-agent-Brainstorm.md`
> **Change in v1.1:** Added Tasks 16–25 — Polished UI redesign matching `frontend/mockups/future-polished-mockup.html`. New routes: `/settings`, `/history`, `/session/[id]/finalize`. New components: `PipelineStage`, `ConfidenceBar`, `CanonicalStatePanel`, `RiskBoard`, `WarningModal`. Backend addition: `GET /sessions` list endpoint + markdown content return.

---

## 1. Goal

Build a **deterministic multi-agent design system** — not a chatbot — that takes a product idea, runs it through an ordered pipeline of N specialized agents (min 2), and converges to a pair of output artifacts (`architecture.md` + `roadmap.md`). Each agent is a separate Go service communicating over the A2A protocol (`github.com/a2aproject/a2a-go/v2`). The backend is a Go 1.26 modular monolith orchestrating agent dispatch, iteration, convergence detection, and state management. The frontend is a SvelteKit structured workspace that shows each agent's output side-by-side per iteration — not a chat interface.

**Why:** Engineers waste hours writing design documents manually. This system accelerates architecture decisions by having multiple AI agents with distinct roles (builder, reviewer, refiner, devil's advocate) challenge and refine an idea until it converges.

---

## 2. Architecture Overview

```
frontend/ (SvelteKit)
       ↓  REST API
backend/ (Go 1.26 modular monolith)
       ↓  a2a-go/v2 (SendMessage / AgentCard)
┌──────────────────────────────────────────────────┐
│  Agent 1        Agent 2        Agent N            │
│  (a2a-go/v2)   (a2a-go/v2)   (a2a-go/v2)         │
│  Role: build   Role: review   Role: refine        │
│  LLM: Copilot  LLM: Claude   LLM: any            │
└──────────────────────────────────────────────────┘
       ↓
PostgreSQL (canonical state + agent registry + skills)
       ↓
Markdown Generator → architecture.md + roadmap.md
```

**Key architectural decisions (non-negotiable):**

| Decision                              | Rationale                                                                                    |
| ------------------------------------- | -------------------------------------------------------------------------------------------- |
| Modular monolith (backend)            | Single deployable; avoids distributed complexity at MVP                                      |
| Vertical slice per module             | Each module owns handler + service + repository + model                                      |
| Ordered N-agent pipeline              | Min 2 agents; roles fixed at session creation; no runtime alternation                        |
| `LLMProvider` interface               | Decouples Copilot/Claude from all business logic                                             |
| Tiered LLM config resolver            | session override → agent-level → global default; resolved at call time                       |
| Credentials as env var refs only      | `CredentialRef` stores env var _name_ only; key resolved via `os.Getenv()` at runtime        |
| `a2a-go/v2` SDK, message-based        | No custom task JSON schema; domain context packed as `DataPart` in `SendMessageRequest`      |
| Skills = prompt injection             | Skills are text fragments appended to the system prompt server-side; agent binary is unaware |
| Svelte stores (no external state lib) | SvelteKit-native; avoids bundle bloat                                                        |
| pgx / sqlc (no heavy ORM)             | Type-safe generated queries; idiomatic Go                                                    |

---

## 3. Tech Stack

**Backend + Agent (Go 1.26):**

```
github.com/a2aproject/a2a-go/v2        A2A protocol (client + server)
github.com/jackc/pgx/v5               PostgreSQL driver
github.com/sqlc-dev/sqlc               Query generation (dev tool)
github.com/google/uuid                 UUID generation
net/http (stdlib)                      HTTP server (backend)
```

**Frontend:**

```
SvelteKit (latest stable)
TypeScript
TailwindCSS
@tanstack/svelte-query                 Server state / data fetching
Svelte stores (built-in)               Client state
```

**Infrastructure:**

```
PostgreSQL 16
Docker + docker-compose
```

---

## 4. Project Structure

```
a2a-brainstorm/
├── go.work                          ← Go workspace (backend + agent modules)
├── docker-compose.yml
├── Makefile
├── docs/
│   ├── A2A-agent-Brainstorm.md      ← source of truth (never modify)
│   └── PLAN.md                      ← this file
│
├── backend/
│   ├── go.mod
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── platform/
│   │   │   ├── config/              ← env var getters
│   │   │   ├── db/                  ← pgx pool, migration runner
│   │   │   ├── logger/
│   │   │   ├── http/                ← server setup, middleware
│   │   │   ├── llm/                 ← LLMProvider interface + implementations + resolver
│   │   │   └── a2a/                 ← a2aclient factory + AgentCard resolution
│   │   └── shared/                  ← shared domain types across modules
│   └── modules/
│       ├── session/                 ← handler, service, repository, model
│       ├── iteration/               ← handler, service, engine
│       ├── agent/                   ← handler, service, repository, model, client, role
│       ├── state/                   ← model, merge, validator
│       ├── convergence/             ← engine
│       └── markdown/                ← generator
│
├── agent/
│   ├── go.mod
│   ├── agentcard.go                 ← declares a2a.AgentCard
│   ├── cmd/server/main.go
│   └── internal/
│       ├── executor/                ← implements a2asrv.AgentExecutor
│       ├── llm/                     ← LLMProvider implementations
│       └── config/                  ← env var config for agent binary
│
├── frontend/
│   ├── package.json
│   ├── svelte.config.js
│   ├── tailwind.config.ts
│   └── src/
│       ├── routes/
│       │   ├── +page.svelte         ← home (session creation)
│       │   ├── session/[id]/+page.svelte
│       │   ├── agents/+page.svelte  ← agent registry management
│       │   └── skills/+page.svelte  ← skill library management
│       └── lib/
│           ├── components/
│           │   ├── AgentPanel.svelte
│           │   ├── AgentSelector.svelte
│           │   ├── SkillManager.svelte
│           │   ├── ControlPanel.svelte
│           │   ├── StateView.svelte
│           │   └── Timeline.svelte
│           ├── stores/
│           │   ├── sessionStore.ts
│           │   └── agentRegistryStore.ts
│           └── services/
│               └── api.ts
│
└── migrations/                      ← SQL migration files (numbered, sequential)
    ├── 001_agents.sql
    ├── 002_skills.sql
    ├── 003_sessions.sql
    └── 004_iterations.sql
```

---

## 5. Implementation Tasks

### Dependency Graph

```
Task 1 (Project Scaffold) ──────────────────────────────────────────────────────┐
    │                                                                              │
    ▼                                                                              │
Task 2 (Platform: Config + DB + Logger)                                           │
    │                                                                              │
    ├──────────────────────────────┐                                              │
    ▼                              ▼                                              │
Task 3 (Platform: LLM)       Task 4 (Platform: A2A)                              │
    │                              │                                              │
    └──────────────┬───────────────┘                                              │
                   ▼                                                               │
             Task 5 (State Module)                                                │
                   │                                                               │
                   ├────────────────────────────────────────────────────────────┐ │
                   ▼                                                              │ │
             Task 6 (Agent Module: Models + DB Schema) ◀────────────────────────┘─┘
                   │
                   ▼
             Task 7 (Agent Module: Service + Handler + Dispatch)
                   │
                   ▼
             Task 8 (Session Module)
                   │
                   ▼
             Task 9 (Iteration Engine + Convergence)
                   │
                   ▼
             Task 10 (Markdown + Backend Wire-up)
                   │
          ┌────────┴──────────────────────────────────────┐
          ▼                                                 ▼
    Task 11 (Agent Service Binary)          Task 12 (Frontend: Scaffold + Stores + API)
                                                            │
                                                ┌───────────┴────────────────┐
                                                ▼                            ▼
                                     Task 13 (Session Workspace)  Task 16 (Design System)
                                                │                            │
                                                ▼                 ┌──────────┼──────────┐
                                     Task 14 (Agent Registry)     ▼          ▼          ▼
                                                │         Task 17 (Home) Task 20  Task 23
                                                │         (Home redesign) (Settings) (History)
                                                │                 │
                                                ▼                 ▼
                                     Task 15 (Integration)  Task 18 (Session Pipeline)
                                                                  │
                                                     ┌────────────┴───────────────────────────┐
                                                     ▼                                         ▼
                                          Task 19 (BE: List + Artifact)           Task 22 (Roles+Modal)
                                                     │                                         │
                                                     ├──────────────────────────┐              │
                                                     ▼                          ▼              ▼
                                          Task 20 (Settings)           Task 23 (History)  Task 24 (Finalize)
                                                     │
                                                     ▼
                                          Task 21 (Agent+Skill Forms)
                                                     │
                                              All Tasks 16–24
                                                     │
                                                     ▼
                                          Task 25 (Navigation + Final Validation)
```

---

### Task 1 — Project Scaffold <!-- ✅ Task 1 completed -->

**Goal:** Initialize the Go workspace, both Go modules (backend + agent), SvelteKit frontend shell, docker-compose, and the Makefile.

**Files to create:**

- `go.work` — Go workspace referencing `./backend` and `./agent`
- `backend/go.mod` — module `a2a-brainstorm/backend`, Go 1.26; add initial deps: `a2a-go/v2`, `pgx/v5`, `uuid`
- `agent/go.mod` — module `a2a-brainstorm/agent`, Go 1.26; add initial deps: `a2a-go/v2`
- `docker-compose.yml` — services: `backend` (port 8080), `agent` (port 9090, `--scale agent=N` friendly), `postgres` (port 5432, image `postgres:16`)
  - `agent` service uses a single shared image; role is injected at runtime per A2A request
  - health checks for all services
- `Makefile` — targets: `build`, `build-agent`, `up`, `down`, `migrate`, `test`, `frontend`, `lint`
- All backend directory stubs (empty `.gitkeep` files): `cmd/server/`, `internal/platform/`, `modules/`
- All agent directory stubs: `cmd/server/`, `internal/executor/`, `internal/llm/`, `internal/config/`
- SvelteKit scaffold: run `pnpm create svelte@latest frontend` (TypeScript strict, no example files), then install TailwindCSS and `@tanstack/svelte-query`
- `migrations/` directory with numbered `.sql` stubs

**Validation:**

- `go work sync` in repo root: zero errors
- `cd backend && go build ./...`: zero errors (no source files yet)
- `cd agent && go build ./...`: zero errors
- `cd frontend && pnpm install`: zero errors
- `docker-compose config`: valid YAML, all services present

**Prompt context needed:** Blueprint §3 (Backend structure), §7 (Agent structure), §12 (Frontend structure), §18 (Deployment)

---

### Task 2 — Platform: Config + DB + Logger

<!-- ✅ Task 2 completed -->

**Goal:** Build the foundational platform services that every module imports — environment config, PostgreSQL connection pool, migration runner, and structured logger.

**Files to create:**

- `backend/internal/platform/config/config.go` — all env var getters; see §8.12 for full list
  - `GetDatabaseURL()` — required, throws descriptive error if absent
  - `GetMaxIterations()` (default `10`), `GetConvergenceThreshold()` (default `0.02`)
  - `GetGlobalLLMProvider()`, `GetGlobalLLMModel()`, `GetGlobalLLMCredentialRef()`
  - `GetAgentEndpoints()` — comma-separated list of agent base URLs (for dev)
  - **Never** use `os.Getenv()` inline anywhere outside this file
- `backend/internal/platform/db/db.go`
  - `NewPool(ctx, cfg) (*pgxpool.Pool, error)` — opens pgx connection pool
  - `RunMigrations(ctx, pool, migrationsDir) error` — sequential SQL file runner (reads `migrations/*.sql` ordered by filename)
  - Uses `GetDatabaseURL()` from config; never accepts raw connection string from caller
- `backend/internal/platform/logger/logger.go`
  - Structured logger wrapping `log/slog` (stdlib, Go 1.21+)
  - `Info`, `Warn`, `Error`, `Debug` helpers; context-aware
  - Never logs credential values; accepts `maskCredentials(msg)` helper

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues

**Prompt context needed:** Blueprint §5 (Platform layer), §5 Credential Security Rules

---

### Task 3 — Platform: LLM Abstraction

<!-- ✅ Task 3 completed -->

**Goal:** Define the `LLMProvider` interface, the tiered config resolver, and the Copilot provider implementation that all agents and future providers implement.

**Files to create:**

- `backend/internal/platform/llm/provider.go` — see §8.2 for exact types
  - `LLMProvider` interface: `Generate(ctx, LLMRequest) (LLMResponse, error)`
  - `LLMRequest` struct: `SystemPrompt string`, `UserMessage string`, `Temperature float64`
  - `LLMResponse` struct: `Content string`, `FinishReason string`, `TokensUsed int`
- `backend/internal/platform/llm/config.go` — see §8.2
  - `LLMConfig` struct: `Provider string`, `Model string`, `CredentialRef string`
  - `CredentialRef` must be an env var name, never a raw key
- `backend/internal/platform/llm/resolver.go`
  - `Resolve(global, agentLevel, sessionOverride *LLMConfig) LLMConfig` — see §8.2 for tiered priority
  - `ResolveKey(credentialRef string) (string, error)` — calls `os.Getenv(credentialRef)`; returns error if empty (no silent fallback)
- `backend/internal/platform/llm/copilot.go`
  - `CopilotProvider` implements `LLMProvider`
  - Reads API key via `ResolveKey(cfg.CredentialRef)` at call time
  - Uses structured JSON schema prompt format; low temperature for determinism

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- `LLMProvider` interface has no direct import of Copilot or Claude SDK — only the implementation files do

**Prompt context needed:** Blueprint §5 (LLM Abstraction), §5 (LLM Config Tiered Resolver), §5 (Credential Security Rules), §8.2 in this PLAN

---

### Task 4 — Platform: A2A Layer

**Goal:** Build the backend-side A2A client factory (resolves `AgentCard`, creates `a2aclient`) and the agent-side server setup helper (`a2asrv`).

**Files to create:**

- `backend/internal/platform/a2a/client.go`
  - `NewClient(ctx, agentEndpoint string) (a2aclient.Client, error)` — resolves `AgentCard` from `{endpoint}/.well-known/agent.json` then calls `a2aclient.NewFromCard()`
  - `SendPayload(ctx, client, payload BrainstormPayload) (any, error)` — wraps payload in `a2a.NewDataPart`, creates `a2a.Message`, calls `client.SendMessage()`; see §8.3 for payload shape
  - `ExtractStateFromResult(result a2a.SendMessageResult) (any, error)` — walks `Artifact.Parts`, extracts `DataPart` content
  - Retries on transient errors (5xx, timeout); immediate failure on 4xx
- `backend/internal/platform/a2a/types.go`
  - `BrainstormPayload` struct: `Role string`, `SystemPrompt string`, `LLMConfig LLMConfig`, `State any` — this is the `DataPart` content shape; see §8.3
- `agent/internal/config/config.go` — same pattern as backend config; reads `AGENT_PORT`, `COPILOT_API_KEY`, `CLAUDE_API_KEY` etc.

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd agent && go build ./...`: zero errors
- `BrainstormPayload` is the single source of truth for backend↔agent wire format

**Prompt context needed:** Blueprint §5 (A2A Layer), §7 (A2A Interaction Model), §8.3 in this PLAN

<!-- ✅ Task 4 completed -->

### Task 5 — State Module

**Goal:** Define the canonical state type, the merge algorithm, and the state validator that all iteration and agent modules depend on.

**Files to create:**

- `backend/internal/modules/state/model.go` — see §8.1 for exact JSON structure
  - `CanonicalState` struct with all fields: `Idea`, `Architecture`, `ExecutionPlan []Step`, `Risks []Risk`, `Assumptions []string`, `OpenQuestions []string`, `Metrics StateMetrics`, `Meta StateMeta`
  - `StateMeta` includes `Iteration int`, `Agents []AgentMeta` (not fixed `agentA`/`agentB`)
  - `AgentMeta` includes `AgentID`, `Name`, `Role`, `Provider`, `Model`, `Skills []string` (names only)
  - All `json` tags must match §8.1 exactly — downstream agents depend on this shape
- `backend/internal/modules/state/merge.go` — see §8.5
  - `Merge(base, incoming CanonicalState) CanonicalState`
  - Rules: union risks (deduplicate by text hash), remove resolved risks, collapse duplicate plan steps, reject steps with vague text (< 10 words)
  - Stability rule: if both agree on a field value → lock it (do not overwrite with identical content)
- `backend/internal/modules/state/validator.go`
  - `Validate(s CanonicalState) error` — rejects malformed state; enforces non-empty idea, confidence in [0,1]

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues

**Prompt context needed:** Blueprint §8 (Canonical State), §10 (Merge Strategy), §8.1 and §8.5 in this PLAN

<!-- ✅ Task 5 completed -->

### Task 6 — Agent Module: Models, Repository, and DB Schema <!-- ✅ Task 6 completed -->

**Goal:** Define the Agent and Skill domain models, create all DB migration files for the agent registry, and implement the repository layer.

**Files to create:**

- `migrations/001_agents.sql` — see §8.11 for exact DDL
  - `CREATE TABLE agents (id, name, description, default_role, system_prompt, llm_config JSONB, endpoint, created_at)`
  - `CREATE TABLE skills (id, name, description, prompt, created_at)`
  - `CREATE TABLE agent_skills (agent_id, skill_id, PRIMARY KEY(agent_id, skill_id))`
- `backend/internal/modules/agent/model.go` — see §8.13 for Role constants
  - `Agent` struct: all fields matching `agents` table + `Skills []Skill` (loaded on GET)
  - `Skill` struct: `ID`, `Name`, `Description`, `Prompt`, `CreatedAt` — see §8.14
  - `Role` type (`string`) + constants: `RoleBuilder`, `RoleReviewer`, `RoleRefiner`, `RoleDevilsAdvocate` — see §8.13
  - `LLMConfig` struct: imported from `internal/platform/llm` — do not duplicate
- `backend/internal/modules/agent/role.go`
  - `DefaultRoles(agentCount int) []Role` — distributes roles by count; see §8.13 distribution table
  - `ValidRole(r Role) bool` — allowlist check
- `backend/internal/modules/agent/repository.go`
  - `CreateAgent`, `GetAgent`, `ListAgents`, `UpdateAgent`, `DeleteAgent`
  - `CreateSkill`, `GetSkill`, `ListSkills`, `DeleteSkill`
  - `AttachSkill(agentID, skillID)`, `DetachSkill(agentID, skillID)`, `GetAgentSkills(agentID) []Skill`
  - Uses pgx directly; no ORM; queries are verbatim SQL (sqlc-generated in future)

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- SQL migration `001_agents.sql` applies cleanly: `psql $DATABASE_URL -f migrations/001_agents.sql`

**Prompt context needed:** Blueprint §6 (agent/ module), §6 DB Tables, §8.11 and §8.13 in this PLAN

---

### Task 7 — Agent Module: Service, Handler, and A2A Dispatch <!-- ✅ Task 7 completed -->

**Goal:** Implement the agent service (business logic + skill assembly + A2A dispatch), the HTTP handler (CRUD REST API for agents and skills), and `client.go` (the A2A dispatch function).

**Files to create:**

- `backend/internal/modules/agent/service.go`
  - `RegisterAgent(ctx, req) (Agent, error)` — validates endpoint reachable via `/health` or AgentCard fetch
  - `GetAgent`, `ListAgents`, `DeleteAgent`, `UpdateAgent`
  - `CreateSkill`, `ListSkills`, `DeleteSkill`, `AttachSkill`, `DetachSkill`, `GetAgentSkills`
  - `ResolveActiveSkills(agentID uuid, overrides []uuid) []Skill` — if overrides present use them; empty override = disable all; absent = use default attached skills
  - `CheckAvailability(agent Agent) error` — validates credential ref env var is set; marks agent unavailable otherwise
- `backend/internal/modules/agent/client.go` — see §8.3 for dispatch pseudocode
  - `Dispatch(ctx, agent Agent, role Role, activeSkills []Skill, sessionLLMOverride *LLMConfig, state CanonicalState) (CanonicalState, error)`
  - Internally: resolves tiered LLM config → assembles system prompt → builds `BrainstormPayload` → calls `platform/a2a.SendPayload()` → extracts updated state
  - `BuildSystemPrompt(base string, skills []Skill) string` — concatenates skill `.Prompt` fragments; see §8.14
- `backend/internal/modules/agent/handler.go`
  - REST handlers for all agent + skill endpoints; see §8.7 for full route list
  - Input validation on all IDs (valid UUID), names (non-empty), prompts (non-empty)
  - Returns `400` on validation failure, `404` on not-found, `409` on name conflict

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues

**Prompt context needed:** Blueprint §6 (agent/ responsibilities), §6 (Skill Injection at Runtime), §8.3, §8.7, §8.13, §8.14 in this PLAN

---

### Task 8 — Session Module <!-- ✅ Task 8 completed -->

**Goal:** Implement the session lifecycle — create session, bind agents, store idea, manage status — with full DB schema.

**Files to create:**

- `migrations/003_sessions.sql` — see §8.11
  - `CREATE TABLE sessions (id, idea TEXT, status TEXT, max_iterations INT, created_at, updated_at)`
  - `CREATE TABLE session_agents (session_id, agent_id, position INT, role TEXT, llm_override JSONB, skill_overrides JSONB, PRIMARY KEY(session_id, agent_id))`
- `backend/internal/modules/session/model.go`
  - `Session` struct; `SessionAgent` struct (includes `Position`, `Role`, `LLMOverride`, `SkillOverrides`)
  - `CreateSessionRequest` — validated input shape; see §8.7 for `POST /sessions` body
  - Minimum 2 agents enforced in request validation
- `backend/internal/modules/session/repository.go`
  - `CreateSession`, `GetSession`, `ListSessions`
  - `CreateSessionAgents(sessionID, agents []SessionAgent)`
  - `GetOrderedAgents(sessionID) []SessionAgent` — ordered by `position ASC`
- `backend/internal/modules/session/service.go`
  - `CreateSession(ctx, req CreateSessionRequest) (Session, error)`
    - Validates ≥ 2 agent IDs
    - Assigns roles: uses `req.RoleOverrides` if provided, otherwise `agent.DefaultRoles(len(agentIDs))`
    - Validates all agent IDs exist and are available
  - `GetSession(ctx, id) (Session, error)`
- `backend/internal/modules/session/handler.go`
  - `POST /sessions`, `GET /sessions/{id}`, `POST /sessions/{id}/finalize`

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- Migration `003_sessions.sql` applies cleanly

**Prompt context needed:** Blueprint §6 (session/ module), §6 (Session-Agent Binding), §8.7, §8.11 in this PLAN

---

### Task 9 — Iteration Engine + Convergence <!-- ✅ Task 9 completed -->

**Goal:** Implement the deterministic N-agent iteration pipeline and the convergence detection engine.

**Files to create:**

- `backend/internal/modules/convergence/engine.go` — see §8.6
  - `Check(prev, next CanonicalState) bool` — returns true (converged) when all stop conditions met; see §8.6
  - `ConfidenceDelta(prev, next CanonicalState) float64` — `|next.Metrics.Confidence - prev.Metrics.Confidence|`
  - `HasNewCriticalRisks(prev, next CanonicalState) bool`
  - `IsExecutionPlanComplete(s CanonicalState) bool` — heuristic: all steps have non-empty description and no open questions reference them
- `backend/internal/modules/iteration/engine.go` — see §8.4 for exact algorithm
  - `Run(ctx, session Session, initialState CanonicalState) (CanonicalState, error)`
  - Ordered pipeline: for each iteration, pass state through every ordered agent sequentially; each agent receives the output of the previous
  - Calls `agent.Dispatch()` for each agent; aggregates via `state.Merge()`
  - Calls `convergence.Check()` after each full pipeline pass; breaks when true
  - Updates `state.Meta.Iteration` each pass
  - Persists state after each full pipeline pass (not per-agent)
- `backend/internal/modules/iteration/service.go`
  - `TriggerIteration(ctx, sessionID uuid) (CanonicalState, error)` — loads session + state, calls engine, persists result
- `backend/internal/modules/iteration/handler.go`
  - `POST /sessions/{id}/iterate` → triggers one iteration and returns updated state

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- Unit test: `engine_test.go` — mock 2 agents, run 3 iterations, assert convergence detected when `ConfidenceDelta < threshold`

**Prompt context needed:** Blueprint §9 (Iteration Engine), §11 (Convergence), §10 (Merge Strategy), §8.4, §8.5, §8.6 in this PLAN

---

### Task 10 — Markdown Generator + Backend Wire-up <!-- ✅ Task 10 completed -->

**Goal:** Implement the markdown output generator and wire all modules into `cmd/server/main.go` with the HTTP router.

**Files to create:**

- `backend/internal/modules/markdown/generator.go`
  - `GenerateArchitecture(s CanonicalState) (string, error)` — renders `architecture.md` from `s.Architecture` + `s.ExecutionPlan`
  - `GenerateRoadmap(s CanonicalState) (string, error)` — renders `roadmap.md` from `s.ExecutionPlan` + timeline
  - `WriteArtifacts(s CanonicalState, outputDir string) error` — writes both files atomically (tmp → rename)
- `backend/cmd/server/main.go` — wire-up:
  - Init: read config, open DB pool, run migrations, init all module services
  - Register all HTTP routes (see §8.7 for full endpoint list)
  - Graceful shutdown on `SIGTERM`/`SIGINT`
- `backend/internal/platform/http/router.go`
  - `NewRouter(deps) http.Handler` — `net/http` with route groups: `/sessions`, `/agents`, `/skills`
  - CORS headers for SvelteKit dev origin
  - Request logging middleware
- `POST /sessions/{id}/finalize` handler in `session/handler.go`
  - Triggers `markdown.WriteArtifacts()` on finalized session state

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- `go run ./backend/cmd/server` starts without panicking (DB not required for this validation — check startup log)

**Prompt context needed:** Blueprint §16 (API Integration), §17 (Output Artifacts), §8.7 in this PLAN

---

### Task 11 — Agent Service Binary <!-- ✅ Task 11 completed -->

**Goal:** Build the standalone agent binary — `agentcard.go` declaration, `BrainstormExecutor` implementing `a2asrv.AgentExecutor`, LLM provider, and HTTP server wiring.

**Files to create:**

- `agent/agentcard.go`
  - `NewAgentCard(port int) *a2a.AgentCard` — name: `"brainstorm-agent"`, description, capabilities (`Streaming: false`)
  - Declares `AgentSkill` entries matching role catalog (build, review, refine, devils_advocate) — these are for discovery only
  - Uses `a2asrv.NewRESTHandler` transport
- `agent/internal/executor/executor.go` — see §8.3 for exact `Execute` implementation template
  - `BrainstormExecutor` implements `a2asrv.AgentExecutor`
  - `Execute(ctx, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error]`
    - Extracts `BrainstormPayload` from `execCtx.Message.Parts` DataPart
    - Calls `e.llm.Generate(ctx, LLMRequest{SystemPrompt: payload.SystemPrompt, UserMessage: marshalState(payload.State)})`
    - Emits: `NewSubmittedTask` → `NewStatusUpdateEvent(Working)` → `NewArtifactEvent(DataPart(updatedState))` → `NewStatusUpdateEvent(Completed)`
  - `Cancel(ctx, execCtx) iter.Seq2[a2a.Event, error]` — emits `TaskStateCanceled`
  - LLM must be called through `LLMProvider` interface — never inline Copilot/Claude SDK
- `agent/internal/llm/copilot.go`
  - Same `LLMProvider` interface as backend (copy the interface definition — do not import from backend module)
  - `CopilotProvider` reads key via `os.Getenv(config.GetLLMCredentialRef())`
- `agent/internal/config/config.go`
  - `GetPort()`, `GetLLMProvider()`, `GetLLMModel()`, `GetLLMCredentialRef()`
- `agent/cmd/server/main.go`
  - Setup: read config, build `AgentCard`, create `BrainstormExecutor`, create `a2asrv.NewHandler`, wrap with `a2asrv.NewRESTHandler`, `http.ListenAndServe`

**Validation:**

- `cd agent && go build ./...`: zero errors
- `cd agent && go vet ./...`: zero issues
- `go run ./agent/cmd/server` starts and serves `/.well-known/agent.json` (curl confirms `200 + valid AgentCard JSON`)

**Prompt context needed:** Blueprint §7 (Agent structure), §7 (A2A Interaction Model), §8.3 in this PLAN

---

### Task 12 — Frontend: Scaffold, Stores, and API Client <!-- ✅ Task 12 completed -->

**Goal:** Set up the SvelteKit project with TypeScript types, all Svelte stores, and the API service layer that all pages import.

**Files to create:**

- `frontend/src/lib/types.ts`
  - `SessionAgent` (id, name, role, provider, model, skills: string[], output?: any)
  - `Agent` (id, name, description, defaultRole, systemPrompt, llmConfig, endpoint, skills: Skill[])
  - `Skill` (id, name, description, prompt)
  - `CanonicalState` — TypeScript equivalent of §8.1 JSON shape
  - `CreateSessionRequest`, `CreateSessionResponse`, `IterateResponse`
- `frontend/src/lib/stores/sessionStore.ts`
  - `sessionStore` writable: `{ session_id, idea, state: CanonicalState | null, iteration, agents: SessionAgent[], loading }` — see §8.9
  - Actions: `setSession`, `setAgents`, `updateState`, `setLoading`
- `frontend/src/lib/stores/agentRegistryStore.ts`
  - `agentRegistryStore` writable: `{ agents: Agent[], skills: Skill[], loading }` — see §8.9
  - Actions: `setAgents`, `setSkills`, `addAgent`, `removeAgent`, `addSkill`, `removeSkill`
- `frontend/src/lib/services/api.ts`
  - All API calls against backend; see §8.7 for full endpoint list
  - Functions: `createSession`, `getSession`, `iterate`, `finalizeSession`
  - `getAgents`, `createAgent`, `updateAgent`, `deleteAgent`
  - `getSkills`, `createSkill`, `updateSkill`, `deleteSkill`
  - `attachSkill(agentId, skillId)`, `detachSkill(agentId, skillId)`, `getAgentSkills(agentId)`
  - Uses `fetch` with typed responses; throws on non-2xx

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean build

**Prompt context needed:** Blueprint §15 (Frontend State), §16 (API Integration), §8.7, §8.9 in this PLAN

---

### Task 13 — Frontend: Session Workspace <!-- ✅ Task 13 completed -->

**Goal:** Build the main session workspace — agent panels, control panel, state viewer, and iteration timeline.

**Files to create:**

- `frontend/src/routes/+page.svelte` — home page; renders idea input + `AgentSelector` for session creation; on submit calls `createSession` and navigates to `/session/{id}`
- `frontend/src/routes/session/[id]/+page.svelte`
  - Loads session on mount; subscribes to `sessionStore`
  - Layout: horizontal agent panels (scrollable when N ≥ 4), control panel, state view, timeline
  - Calls `iterate` on "Next Iteration" button; calls `finalizeSession` on "Approve"
- `frontend/src/lib/components/AgentPanel.svelte`
  - Props: `agent: SessionAgent` (id, name, role, provider, model, skills[], output)
  - Shows: name, role badge, LLM provider + model label, skill tags, structured output, diff highlight vs previous iteration (use simple string diff)
- `frontend/src/lib/components/ControlPanel.svelte`
  - Buttons: Start (disabled after start), Next Iteration (disabled while loading), Approve, Inject Feedback (textarea)
  - Binds to `sessionStore.loading` for disabled state
- `frontend/src/lib/components/StateView.svelte`
  - Renders `sessionStore.state`: Architecture section, Execution Plan accordion, Risks list with severity badges
- `frontend/src/lib/components/Timeline.svelte`
  - Renders iteration history from `sessionStore`; shows per-agent role per iteration as a horizontal timeline

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean build

**Prompt context needed:** Blueprint §13 (UI Layout), §14 (Frontend Components), §15 (Frontend State), §8.9 in this PLAN

---

### Task 14 — Frontend: Agent Registry and Skill Manager <!-- ✅ Task 14 completed -->

**Goal:** Build the agent registry management page and the skill library manager page.

**Files to create:**

- `frontend/src/routes/agents/+page.svelte`
  - Lists all agents from `agentRegistryStore`; shows each agent's name, role, LLM, skill count
  - Allows: create agent (form), edit agent, delete agent
  - Loads `agentRegistryStore` on mount via `getAgents()` + `getSkills()`
- `frontend/src/lib/components/AgentSelector.svelte`
  - Used on home page (`+page.svelte`) during session creation
  - Shows agent registry; allows picking agents (min 2 enforced in UI)
  - Per-selected-agent: role override dropdown, optional LLM model override input
  - Per-selected-agent: skill toggle list (defaults to agent's attached skills; can deselect individual skills for this session)
- `frontend/src/routes/skills/+page.svelte`
  - Loads skill library from `agentRegistryStore`
  - Create/edit/delete skills
  - Per-skill: shows which agents it is attached to
- `frontend/src/lib/components/SkillManager.svelte`
  - Reusable component used in `/skills` route
  - Skill form: name, description, prompt fragment (textarea)
  - Agent attachment: checkbox list of agents; calls `attachSkill` / `detachSkill`

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean build

**Prompt context needed:** Blueprint §14 (AgentSelector, SkillManager), §16 (API endpoints), §8.7, §8.9 in this PLAN

---

### Task 15 — Integration Tests, Documentation, and Final Validation <!-- ✅ Task 15 completed -->

**Goal:** End-to-end integration tests covering the full iteration pipeline, documentation, and the Definition of Done checklist.

**Files to create:**

- `backend/internal/modules/iteration/engine_test.go`
  - Mock `AgentExecutor`: accepts `BrainstormPayload`, returns incremented confidence each call
  - Test: 2-agent session, run engine for 5 iterations, assert convergence triggers before `maxIter`
  - Test: verify ordered pipeline — agent at `position=0` is always called before `position=1`
- `backend/internal/modules/agent/client_test.go`
  - Mock a2aclient; assert `BuildSystemPrompt` concatenates base + skill fragments in correct order
  - Assert `Dispatch` resolves tiered LLM config (session override > agent-level > global)
- `backend/internal/modules/state/merge_test.go`
  - Test: deduplication of risks, collapse of duplicate plan steps, rejection of vague output
- `agent/internal/executor/executor_test.go`
  - Mock `LLMProvider`; assert `Execute` emits `Submitted → Working → ArtifactUpdate → Completed` event sequence
  - Assert extracted `BrainstormPayload` correctly reads from `DataPart`
- `frontend/src/lib/services/api.test.ts`
  - Mock `fetch`; assert all API functions handle `400`/`404`/`500` responses correctly
- `README.md` (repo root)
  - System overview (not a chatbot — deterministic design IDE)
  - Prerequisites: Go 1.26, Node 20+, Docker
  - Quick start: `make up && make migrate && go run ./backend/cmd/server`
  - Agent setup and scaling guide
  - Frontend dev: `cd frontend && pnpm dev`
  - Environment variables table
  - Architecture diagram (text-based, referencing `docs/A2A-agent-Brainstorm.md`)

**Final Validation Checklist:**

- [ ] `cd backend && go build ./...` — zero errors
- [ ] `cd backend && go vet ./...` — zero issues
- [ ] `cd agent && go build ./...` — zero errors
- [ ] `cd agent && go vet ./...` — zero issues
- [ ] `cd backend && go test ./...` — all tests pass
- [ ] `cd agent && go test ./...` — all tests pass
- [ ] `cd frontend && pnpm check` — zero svelte-check errors
- [ ] `cd frontend && pnpm build` — clean production build
- [ ] `docker-compose up` brings up postgres + backend + agent without errors
- [ ] `POST /sessions` with 2 agents → `POST /sessions/{id}/iterate` → `GET /sessions/{id}` returns updated state
- [ ] Agent binary serves valid `AgentCard` at `/.well-known/agent.json`
- [ ] Credential env vars absent → agent marked unavailable, no silent fallback

**Prompt context needed:** All blueprint sections; attach full `docs/A2A-agent-Brainstorm.md`

---

---

### Task 16 — Frontend: Design System Foundation <!-- ✅ Task 16 completed -->

**Goal:** Establish the visual design system — CSS custom properties, Google Fonts, global gradient background, glassmorphism panel/card primitives, button styles, and artboard layout — that all subsequent UI tasks depend on. This is a pure style layer; no functional logic changes.

**Files to create / modify:**

- `frontend/src/app.css` — replace gray Tailwind palette with warm/blue CSS custom properties:
  - `--bg-0: #f5efe4`, `--bg-1: #e8ecf7`, `--ink-900: #151b2f`, `--ink-700: #2d3655`, `--ink-500: #5a6282`, `--ink-300: #a8aec7`
  - `--accent: #0bb6d9`, `--accent-2: #1f7ae0`, `--ok: #1b9f66`, `--warn: #d48806`, `--danger: #ce3158`
  - Full-page background: `radial-gradient(1200px 600px at 10% 10%, #fff8ec, transparent), radial-gradient(900px 500px at 90% 10%, #e8f7ff, transparent), linear-gradient(135deg, #f5efe4, #e8ecf7)`
  - `.artboard`: `min(1300px, 94vw)` centered, `margin: 28px auto`
  - `.topbar`: `background: rgba(255,255,255,0.85)`, `backdrop-filter: blur(12px)`, sticky
  - `.panel`: `background: rgba(255,255,255,0.72)`, `backdrop-filter: blur(8px)`, `border-radius: 18px`, `box-shadow: 0 10px 30px rgba(35,46,82,0.1)`
  - `.card`: same as `.panel` with `border-radius: 14px`
  - Heading font: Space Grotesk; body font: IBM Plex Sans; mono font: IBM Plex Mono
  - Button base classes: `.btn-primary` (gradient `--accent→--accent-2`), `.btn-ghost`, `.btn-danger`
  - Role badge classes: `.badge-build`, `.badge-review`, `.badge-refine`, `.badge-devils-advocate`
  - Status chip classes: `.chip-live`, `.chip-ok`, `.chip-warn`, `.chip-danger`
- `frontend/src/routes/+layout.svelte` — add Google Fonts `<link>` preconnect + stylesheet for IBM Plex Sans (300,400,500), IBM Plex Mono (400), Space Grotesk (500,700); add `<div class="topbar">` wrapper with logo + nav slots
- `frontend/tailwind.config.ts` — extend theme colors with the CSS token names so Tailwind utility classes map to them: `colors: { accent: 'var(--accent)', 'accent-2': 'var(--accent-2)', ok: 'var(--ok)', warn: 'var(--warn)', danger: 'var(--danger)', 'bg-0': 'var(--bg-0)', 'bg-1': 'var(--bg-1)' }`

**Design system spec:** see §8.16

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean build
- Visual smoke: page background is warm-to-blue gradient; fonts render as IBM Plex Sans

**Prompt context needed:** §8.16 in this PLAN, `docs/A2A-agent-Brainstorm.md §20`

---

### Task 17 — Frontend: Home View Redesign <!-- ✅ Task 17 completed -->

**Goal:** Redesign the session-creation home page to match the mockup exactly — topbar, hero panel, 2-column grid (iterations left, agent pool right with inline checkbox rows), gradient CTA button, and estimated-runtime hint.

**Files to modify:**

- `frontend/src/routes/+page.svelte` — full redesign:
  - Topbar: `<header class="topbar">` with "A2A Brainstorm" logo + nav links ("Session History" → `/history`, "⚙ Settings" → `/settings`) + animated Live chip (green pulsing dot)
  - Hero `.panel` centered in `.artboard`, max-width `920px`
  - Idea textarea with char count (no hard limit; show chars used)
  - 2-col grid below textarea:
    - Left col: "Max Iterations" `<input type="number" min="1" max="20">` — defaults to 5
    - Right col: "Agent Pool" — inline checkbox rows, one per agent from `agentRegistryStore.agents`; each row shows agent name, role badge, provider/model label; min-2 enforcement (disable Start if < 2 checked)
  - "Start Session" `<button class="btn-primary">` with gradient; disabled + spinner while loading
  - "Estimated runtime: ~N min" computed hint: `N = checkedAgentCount * iterations * 0.5` minutes; shown below button
  - On submit: call `createSession` → navigate to `/session/{id}`
  - Inline validation: highlight if < 2 agents selected (soft red border on pool + tooltip)
- `frontend/src/lib/components/AgentSelector.svelte` — keep file but replace implementation with the inline pool layout to remain compatible with any code that imports it; it can be a thin wrapper rendering the checkbox rows

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean build
- Renders correctly with 0, 1, and 3+ agents in registry

**Prompt context needed:** §8.7 (POST /sessions body), §8.9 (agentRegistryStore shape), §8.16, `docs/A2A-agent-Brainstorm.md §20`

---

### Task 18 — Frontend: Session View + Sequential Pipeline Components <!-- ✅ Task 18 completed -->

**Goal:** Redesign the session workspace to show a sequential N-agent pipeline with pass summary bar (Pipeline Pass N/M, confidence %), per-stage done/running/waiting states with mono log blocks and summaries, inline canonical state panel, and risk board.

**Files to create / modify:**

- `frontend/src/routes/session/[id]/+page.svelte` — full redesign:
  - Pass summary bar (sticky top): "Pipeline Pass N / M" label + agent count chip + `<ConfidenceBar>` showing `state.metrics.confidence * 100`% + animated shimmer while loading
  - Vertical sequential pipeline panel (`.panel`): one `<PipelineStage>` per agent, separated by connector lines (solid for done→running, dashed for running→waiting)
  - After pipeline panel: 2-col bottom row — left 2/3 `<CanonicalStatePanel>`, right 1/3 `<RiskBoard>`
  - Control bar (sticky bottom): "Run Next Iteration" button (disabled while loading or converged), "Inject Feedback" button (opens inline textarea), "Finalize Session" button → navigates to `/session/{id}/finalize`
  - Stage state derivation from `sessionStore.state`:
    - After a completed iterate call: all agents show `stage-done` with their output derived from `state.meta.agents`
    - During loading (iterate in flight): last agent shows `stage-running`, others show `stage-done`; agents not yet called show `stage-waiting`
    - Injected feedback textarea: plain text, sent as additional context in next iterate call (append to idea)
  - Subscribe to `sessionStore`; call `loadSession` on mount
- `frontend/src/lib/components/PipelineStage.svelte` — **new** (replaces `AgentPanel.svelte`):
  - Props: `agent: SessionAgent`, `status: 'done' | 'running' | 'waiting'`, `output?: string`, `summary?: string`
  - CSS class applied to root: `.stage-done`, `.stage-running`, `.stage-waiting`
  - Done: green check icon, mono log block (dark bg `#1a1d2e`, IBM Plex Mono text), green summary block with `<CheckCircle>` icon
  - Running: animated dots (three dots CSS keyframe blink), mono log block with blinking cursor
  - Waiting: dimmed opacity 0.5, dashed border
  - Role badge at top-right: `.badge-{role}` class
- `frontend/src/lib/components/ConfidenceBar.svelte` — **new**:
  - Props: `value: number` (0–100), `animating: boolean`
  - Segmented progress bar: green fill, animating shimmer when `animating=true`
  - Label shows "Confidence N%"
- `frontend/src/lib/components/CanonicalStatePanel.svelte` — **new** (replaces `StateView.svelte`):
  - Props: `state: CanonicalState | null`
  - Sections as mini-cards: Idea, Architecture, Execution Plan (accordion), Assumptions, Open Questions
  - Uses `.card` class for each section
- `frontend/src/lib/components/RiskBoard.svelte` — **new**:
  - Props: `risks: Risk[]`
  - Shows risk title + severity badge (`.chip-danger` / `.chip-warn`) + description
  - Empty state: "No risks identified" with shield icon
- **Deprecate** (keep files but add `@deprecated` comment + redirect to new components in comments):
  - `frontend/src/lib/components/AgentPanel.svelte` — deprecated; use `PipelineStage.svelte`
  - `frontend/src/lib/components/ControlPanel.svelte` — deprecated; logic inlined into session page
  - `frontend/src/lib/components/StateView.svelte` — deprecated; replaced by `CanonicalStatePanel.svelte`
  - `frontend/src/lib/components/Timeline.svelte` — deprecated; replaced by pass summary bar in session page

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean build
- Session page renders with 0-agent state (loading), 2-agent done state, 2-agent in-progress state

**Prompt context needed:** §8.1 (CanonicalState shape), §8.9 (sessionStore shape), §8.13 (role constants), §8.16, `docs/A2A-agent-Brainstorm.md §20`

---

### Task 19 — Backend: Session List Endpoint + Artifact Content Return <!-- ✅ Task 19 completed -->

**Goal:** Add the missing `GET /sessions` list endpoint (required by history view) and update `POST /sessions/{id}/finalize` to return the generated markdown content in the response body (required by the finalize/export view download buttons). Neither change breaks the existing iteration flow.

**Files to modify:**

- `backend/internal/modules/session/model.go`
  - Add `SessionListItem` struct: `ID`, `Idea` (truncated to 120 chars in service), `Status`, `MaxIterations`, `CurrentIteration int` (from `current_state.meta.iteration`), `Confidence float64` (from `current_state.metrics.confidence`), `AgentCount int`, `CreatedAt`, `UpdatedAt`
  - Add `ListSessionsResponse` struct: `Sessions []SessionListItem`, `Total int`
  - Add `FinalizeResponse` struct: `SessionID`, `ArchitectureMarkdown string`, `RoadmapMarkdown string`, `Status string`
- `backend/internal/modules/session/repository.go`
  - Add `ListSessions(ctx) ([]Session, error)` — `SELECT id, idea, status, max_iterations, current_state, created_at, updated_at FROM sessions ORDER BY created_at DESC`
- `backend/internal/modules/session/service.go`
  - Add `ListSessions(ctx) (ListSessionsResponse, error)` — maps DB rows → `SessionListItem` (extracts confidence + iteration from JSONB `current_state`); truncates idea to 120 chars
  - Update `FinalizeSession(ctx, id) (FinalizeResponse, error)` — call `markdown.GenerateContent(state)` (see below) and include returned strings in response
- `backend/internal/modules/session/handler.go`
  - Add `GET /sessions` handler: calls `service.ListSessions`; returns `200 + ListSessionsResponse`; no auth (same as all other endpoints)
  - Update `POST /sessions/{id}/finalize` handler: returns `FinalizeResponse` JSON (previously returned `204`)
- `backend/internal/modules/markdown/generator.go`
  - Add `GenerateContent(s CanonicalState) (arch string, roadmap string, error)` — same logic as `WriteArtifacts` but returns strings instead of writing files; `WriteArtifacts` calls this internally
- `backend/internal/platform/http/router.go`
  - Register `GET /sessions` route
- `frontend/src/lib/types.ts`
  - Add `SessionListItem` interface matching `SessionListItem` Go struct
  - Update `FinalizeResponse` interface to include `architecture_markdown` and `roadmap_markdown`
- `frontend/src/lib/services/api.ts`
  - Add `listSessions(): Promise<SessionListItem[]>`
  - Update `finalizeSession` return type to `FinalizeResponse`

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- `curl -s http://localhost:8080/sessions | jq .` returns `{"sessions":[], "total":0}` when DB is empty
- `cd frontend && pnpm check`: zero errors

**Prompt context needed:** §8.7 (endpoint definitions), §8.11 (DB schema), §8.16, Task 10 (markdown generator)

---

### Task 20 — Frontend: Settings View — Agents + Skills Tabs <!-- ✅ Task 20 completed -->

**Goal:** Build the unified `/settings` page with tabbed navigation replacing the separate `/agents` and `/skills` routes. The agents tab shows the full agent table (name, role, provider/model, skill count, status, actions). The skills tab shows the skill library table. Old routes redirect to the new page.

**Files to create / modify:**

- `frontend/src/routes/settings/+page.svelte` — **new**:
  - Topbar nav with back-link to `/`
  - Tab bar: "Agents" | "Skills" | "Roles" (3 tabs; roles tab implemented in Task 22)
  - **Agents tab**: table rows — Name, Default Role (badge), Provider/Model, Skills count, Status chip (`.chip-ok` / `.chip-warn`), Edit → `/settings/agent/{id}`, Delete (shows `WarningModal`)
  - **Skills tab**: table rows — Name, Domain (derived from first word of description), Description (truncated 80 chars), Used By (N agents chip), Edit → `/settings/skill/{id}`, Delete
  - Load data on mount: `getAgents()` + `getSkills()` → write to `agentRegistryStore`
  - Empty states: "No agents registered yet. Add one →" link button; same for skills
  - Preserve existing `SkillManager.svelte` usage by keeping the component but wrapping it inside the tab (or deprecate and inline)
- `frontend/src/routes/agents/+page.svelte` — replace full content with `<script>import { goto } from '$app/navigation'; goto('/settings?tab=agents', { replaceState: true });</script>`
- `frontend/src/routes/skills/+page.svelte` — replace full content with `<script>import { goto } from '$app/navigation'; goto('/settings?tab=skills', { replaceState: true });</script>`

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean build
- Navigating to `/agents` or `/skills` redirects to `/settings?tab=agents` or `/settings?tab=skills`

**Prompt context needed:** §8.7 (agent/skill API), §8.9 (agentRegistryStore), §8.16, Task 14 (original agent/skill pages)

---

### Task 21 — Frontend: Agent Form + Skill Form Views <!-- ✅ Task 21 completed -->

**Goal:** Build the agent creation/edit form view and skill creation/edit form view, matching the mockup — card-based forms with all fields, skill assignment pool for agents, and save/cancel navigation.

**Files to create:**

- `frontend/src/routes/settings/agent/new/+page.svelte` — **new**:
  - Form fields: Name (text), Role (select from role constants), Provider (select: copilot / claude), Model (text), Endpoint URL (text), System Prompt (textarea), Description (text)
  - "Assign Skills" section: checkbox list from `agentRegistryStore.skills`; pre-checked defaults empty (none)
  - On submit: call `createAgent(req)` → `attachSkill(agentId, skillId)` for each checked skill → navigate to `/settings?tab=agents`
  - On cancel: navigate back to `/settings?tab=agents`
- `frontend/src/routes/settings/agent/[id]/+page.svelte` — **new**:
  - Same form pre-populated; on load: `getAgent(id)` + `getAgentSkills(id)` to get current attachment
  - On submit: `updateAgent` + diff skill attachments (call `attachSkill`/`detachSkill` for changes)
  - Shows "Delete Agent" button (`.btn-danger`); confirms with `WarningModal`
- `frontend/src/routes/settings/skill/new/+page.svelte` — **new**:
  - Form fields: Name (text), Description (text), Prompt (textarea, labeled "Prompt Fragment — this text is appended to the agent's system prompt when the skill is active")
  - On submit: `createSkill(req)` → navigate to `/settings?tab=skills`
- `frontend/src/routes/settings/skill/[id]/+page.svelte` — **new**:
  - Pre-populated form; `updateSkill` on submit; delete with `WarningModal`
  - "Attached Agents" read-only info: lists agents that have this skill

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean build
- Form validation: name required, prompt required, skill-less submit shows inline error

**Prompt context needed:** §8.7 (agent/skill API endpoints), §8.9 (agentRegistryStore), §8.13 (role catalog), §8.14 (skill injection), §8.16

---

### Task 22 — Frontend: Roles Tab + Warning Modal <!-- ✅ Task 22 completed -->

**Goal:** Add the Roles tab to the Settings view (displaying all four built-in roles as read-only reference cards — no custom role CRUD yet) and implement the reusable `WarningModal` component used by agent/skill deletion flows and the "discard changes?" navigation guard.

**Files to create / modify:**

- `frontend/src/lib/components/WarningModal.svelte` — **new**:
  - Props: `open: boolean`, `title: string`, `body: string`, `confirmLabel: string`, `confirmDanger: boolean`, `onConfirm: () => void`, `onDismiss: () => void`
  - Renders semi-transparent overlay (`rgba(0,0,0,0.35)`) + centered `.panel` modal (max-width 480px)
  - Icon: warning triangle (amber) or danger circle (red) depending on `confirmDanger`
  - Footer: "Dismiss" (`.btn-ghost`) + confirmLabel (`.btn-primary` or `.btn-danger`)
  - Keyboard: `Escape` key triggers `onDismiss`; focus-trap inside modal
- `frontend/src/lib/stores/uiStore.ts` — **new**:
  - `uiStore` writable: `{ modalOpen: boolean, modalTitle: string, modalBody: string, modalConfirmLabel: string, modalConfirmDanger: boolean, onModalConfirm: (() => void) | null }`
  - Actions: `openModal(opts)`, `closeModal()`
- `frontend/src/routes/settings/+page.svelte` — update to add **Roles tab**:
  - Four read-only role cards: BUILD, REVIEW, REFINE, DEVILS ADVOCATE
  - Each card shows: role badge, behavior description (from §8.13), "System Role" chip (`.chip-ok`)
  - "Custom roles coming soon" info callout at bottom of tab
  - Import and use `<WarningModal>` for delete confirmations on Agents and Skills tabs
- `frontend/src/routes/+layout.svelte` — mount `<WarningModal>` at top level, bound to `uiStore`

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean build
- Modal opens/closes correctly; Escape key dismisses; confirm triggers callback

**Prompt context needed:** §8.13 (role catalog + behavior), §8.16

---

### Task 23 — Frontend: Session History View <!-- ✅ Task 23 completed -->

**Goal:** Build the `/history` route — 4 stat cards (sessions completed, avg confidence, docs generated, avg iterations) + searchable/filterable session table linking to the finalize/export view.

**Files to create:**

- `frontend/src/routes/history/+page.svelte` — **new**:
  - Topbar with back-link to `/`
  - 4 stat cards (`.card` class) in a horizontal row:
    - "Sessions Completed" — count of sessions with `status: 'approved' | 'converged'`
    - "Avg Confidence" — mean of `confidence` across all sessions
    - "Docs Generated" — count of sessions with `status: 'approved'`
    - "Avg Iterations" — mean of `current_iteration` across all sessions
  - Live search `<input>` — filters the session table by idea text client-side (no debounce needed)
  - Sessions table columns: Title (idea truncated), Date (`created_at` formatted), Iterations, Confidence (pill: green ≥ 0.8, amber ≥ 0.5, red < 0.5), Agents (count chip), Status chip, "View →" link → `/session/{id}/finalize` for approved, `/session/{id}` otherwise
  - Load on mount: `listSessions()` → compute stats client-side
  - Empty state: "No sessions yet. Start one on the home page" with link

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean build
- Search filters rows reactively; stats re-render on filter (show filtered count vs total)

**Prompt context needed:** §8.7 (GET /sessions), `SessionListItem` type from Task 19, §8.16

---

### Task 24 — Frontend: Finalize/Export View <!-- ✅ Task 24 completed -->

**Goal:** Build the `/session/{id}/finalize` route — animated markdown generation log panel, output file cards with Pending → Running → Done state transitions, preview panes, copy-to-clipboard, and download buttons.

**Files to create:**

- `frontend/src/routes/session/[id]/finalize/+page.svelte` — **new**:
  - On mount: check `sessionStore.session_id`; if not matching `params.id`, call `getSession(id)` to reload
  - "Finalize Session" header with session idea subtitle
  - "Generate Documents" button (`.btn-primary`) — triggers finalize flow; disabled while in progress or already done
  - Markdown Generator log panel (`.panel` with dark background `#1a1d2e`, monospace text):
    - Simulated streaming log lines using `setTimeout` intervals (no real SSE needed): "Analyzing canonical state...", "Extracting architecture decisions...", "Generating architecture.md...", "Generating roadmap.md...", "Writing artifacts... Done ✓"
    - Each line appends every 400ms until complete; shows animated blinking cursor while in progress
    - Green "DONE" badge (`.chip-ok`) appears when all lines shown
  - Two output cards side by side after generation completes:
    - **architecture.md card**: title + "Architecture Document" description + preview pane (textarea `readonly`, pre-populated from `FinalizeResponse.architecture_markdown`) + "Copy" button (clipboard API) + "Download" button (creates `Blob` → `URL.createObjectURL` → `<a download>` click)
    - **roadmap.md card**: same structure for `FinalizeResponse.roadmap_markdown`
  - Done bar at bottom: "Download All" button (triggers both downloads) + "New Session" button → navigate to `/`
  - If session is already `status: 'approved'`: skip generation step, show cards with previously generated content (requires store to cache `FinalizeResponse`); show "Already finalized" chip

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean build
- Log panel streams correctly; download creates valid `.md` file; clipboard copy works

**Prompt context needed:** `FinalizeResponse` type from Task 19, §8.16, Task 18 (session view flow)

---

### Task 25 — Frontend: Navigation Wiring + Final UI Validation <!-- ✅ Task 25 completed -->

**Goal:** Wire all views together with consistent topbar navigation, update `+layout.svelte` with the global nav and modal mount, run all frontend tests, confirm zero linter/type errors, and update documentation.

**Files to modify:**

- `frontend/src/routes/+layout.svelte` — final version:
  - Global topbar: "A2A Brainstorm" logo → `/`, nav links: "Session History" → `/history`, "⚙ Settings" → `/settings`; active link highlight via `$page.url.pathname`
  - Mount `<WarningModal>` bound to `uiStore` (from Task 22)
  - Import global CSS (already imported in existing layout)
- `frontend/src/routes/session/[id]/+page.svelte` — add "← Sessions" back-link in pass summary bar
- `frontend/src/lib/services/api.test.ts` — add test cases for `listSessions` (mock empty + populated response) and `finalizeSession` (mock `FinalizeResponse` with markdown content)
- `README.md` — update Frontend section:
  - New route table: `/` (Home), `/session/{id}` (Session workspace), `/session/{id}/finalize` (Export), `/settings` (Agents + Skills + Roles), `/history` (Session history)
  - Note: `/agents` and `/skills` redirect to `/settings`
  - List new components: `PipelineStage`, `ConfidenceBar`, `CanonicalStatePanel`, `RiskBoard`, `WarningModal`

**Final UI Validation Checklist:**

- [ ] `cd frontend && pnpm check` — zero svelte-check errors
- [ ] `cd frontend && pnpm build` — clean production build
- [ ] `cd frontend && pnpm test` — all API service tests pass
- [ ] `cd backend && go build ./...` — zero errors (Task 19 additions)
- [ ] `cd backend && go vet ./...` — zero issues
- [ ] `cd backend && go test ./...` — all tests pass
- [ ] Navigate `/agents` → redirects to `/settings?tab=agents`
- [ ] Navigate `/skills` → redirects to `/settings?tab=skills`
- [ ] Create session → session workspace shows pipeline stages
- [ ] Session history renders stat cards from `GET /sessions`
- [ ] Finalize flow → log streams → download buttons create `.md` files

**Prompt context needed:** All Tasks 16–24, §8.7, §8.16, §8.17

---

## 6. Task Summary

| Task | Name                                         | Key Files                                                                                                                     | Depends On       | Complexity |
| ---- | -------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ---------------- | ---------- |
| 1    | Project Scaffold                             | `go.work`, `go.mod` ×2, `docker-compose.yml`, `Makefile`, FE scaffold                                                         | —                | Low        |
| 2    | Platform: Config + DB + Logger               | `platform/config/`, `platform/db/`, `platform/logger/`                                                                        | Task 1           | Low        |
| 3    | Platform: LLM Abstraction                    | `platform/llm/provider.go`, `resolver.go`, `copilot.go`                                                                       | Task 2           | Medium     |
| 4    | Platform: A2A Layer                          | `platform/a2a/client.go`, `types.go`, `agent/internal/config/`                                                                | Task 2           | Medium     |
| 5    | State Module                                 | `modules/state/model.go`, `merge.go`, `validator.go`                                                                          | Tasks 3, 4       | Medium     |
| 6    | Agent Module: Models + DB Schema             | `modules/agent/model.go`, `repository.go`, `role.go`, `001_agents.sql`                                                        | Tasks 1, 5       | Medium     |
| 7    | Agent Module: Service + Handler + Dispatch   | `modules/agent/service.go`, `handler.go`, `client.go`                                                                         | Tasks 6, 3, 4    | High       |
| 8    | Session Module                               | `modules/session/*`, `003_sessions.sql`                                                                                       | Task 7           | Medium     |
| 9    | Iteration Engine + Convergence               | `iteration/engine.go`, `convergence/engine.go`                                                                                | Tasks 5, 7, 8    | High       |
| 10   | Markdown + Backend Wire-up                   | `markdown/generator.go`, `cmd/server/main.go`, `platform/http/router.go`                                                      | Tasks 9, 8       | Medium     |
| 11   | Agent Service Binary                         | `agent/agentcard.go`, `executor/executor.go`, `agent/cmd/server/main.go`                                                      | Tasks 3, 4       | High       |
| 12   | Frontend: Scaffold + Stores + API Client     | `lib/types.ts`, `stores/*.ts`, `services/api.ts`                                                                              | Task 1           | Medium     |
| 13   | Frontend: Session Workspace                  | `AgentPanel.svelte`, `ControlPanel.svelte`, `StateView.svelte`, `Timeline.svelte`                                             | Task 12          | Medium     |
| 14   | Frontend: Agent Registry + Skills            | `AgentSelector.svelte`, `SkillManager.svelte`, routes                                                                         | Task 12          | Medium     |
| 15   | Integration Tests + Docs                     | `*_test.go` files, `README.md`                                                                                                | Tasks 11, 13, 14 | Medium     |
| 16   | Frontend: Design System Foundation           | `app.css`, `+layout.svelte`, `tailwind.config.ts`                                                                             | Task 12          | Low        |
| 17   | Frontend: Home View Redesign                 | `routes/+page.svelte`, `AgentSelector.svelte`                                                                                 | Task 16          | Medium     |
| 18   | Frontend: Session View + Pipeline Components | `session/[id]/+page.svelte`, `PipelineStage.svelte`, `ConfidenceBar.svelte`, `RiskBoard.svelte`, `CanonicalStatePanel.svelte` | Tasks 16, 17     | High       |
| 19   | Backend: Session List + Artifact Content     | `session/model.go`, `repository.go`, `service.go`, `handler.go`, `markdown/generator.go`, `api.ts`, `types.ts`                | Tasks 10, 12     | Medium     |
| 20   | Frontend: Settings View (Agents+Skills Tabs) | `routes/settings/+page.svelte`, redirect `/agents`, redirect `/skills`                                                        | Tasks 16, 19     | Medium     |
| 21   | Frontend: Agent Form + Skill Form Views      | `settings/agent/new`, `settings/agent/[id]`, `settings/skill/new`, `settings/skill/[id]`                                      | Task 20          | Medium     |
| 22   | Frontend: Roles Tab + Warning Modal          | `WarningModal.svelte`, `uiStore.ts`, settings Roles tab                                                                       | Task 20          | Medium     |
| 23   | Frontend: Session History View               | `routes/history/+page.svelte`                                                                                                 | Tasks 16, 19     | Medium     |
| 24   | Frontend: Finalize/Export View               | `routes/session/[id]/finalize/+page.svelte`                                                                                   | Tasks 19, 22     | Medium     |
| 25   | Frontend: Navigation Wiring + Final UI Val   | `+layout.svelte`, `api.test.ts`, `README.md`                                                                                  | Tasks 16–24      | Medium     |

---

## 7. How to Use This Plan

1. **Start each task in a fresh chat session** — share this `PLAN.md` + the relevant blueprint sections listed under "Prompt context needed"
2. **Validate after each task** — run `go build ./...` + `go vet ./...` (backend/agent) or `pnpm check` + `pnpm build` (frontend) before moving to the next task
3. **Update this plan** as you learn new information during implementation
4. **One task at a time** — do not attempt multiple tasks in a single session to avoid context overflow
5. **Source of truth** — always refer to `docs/A2A-agent-Brainstorm.md` for exact design decisions. This `PLAN.md` is the breakdown strategy; the blueprint is the specification.

---

## 8. Deep Knowledge Reference

This section contains complete schemas, business rules, algorithms, and data flows from `docs/A2A-agent-Brainstorm.md`. Attach the relevant sub-sections to each task session.

---

### 8.1 Canonical State Model

```json
{
  "idea": {},
  "architecture": {},
  "execution_plan": [],
  "risks": [],
  "assumptions": [],
  "open_questions": [],
  "metrics": {
    "confidence": 0.0
  },
  "meta": {
    "iteration": 0,
    "agents": [
      {
        "agent_id": "uuid",
        "name": "Agent Alpha",
        "role": "build",
        "provider": "claude",
        "model": "claude-opus-4",
        "skills": ["Security Review", "Cost Optimization"]
      }
    ]
  }
}
```

Rules:

- `meta.agents` is populated from `session_agents` at session creation — length ≥ 2
- `skills` in `AgentMeta` stores names only (not prompt fragments) — for observability
- Fixed keys `agentA`/`agentB` do **not** exist; the list is dynamic

---

### 8.2 Go Interfaces

```go
// LLMProvider — all LLM calls go through this interface; never call Copilot/Claude SDK directly
type LLMProvider interface {
    Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)
}

type LLMRequest struct {
    SystemPrompt string
    UserMessage  string
    Temperature  float64
}

type LLMResponse struct {
    Content      string
    FinishReason string
    TokensUsed   int
}

// LLMConfig — stored in DB and passed through A2A; CredentialRef is an env var name, never the key value
type LLMConfig struct {
    Provider      string // "copilot" | "claude"
    Model         string // e.g. "claude-opus-4", "gpt-4o"
    CredentialRef string // env var name, e.g. "CLAUDE_API_KEY"
}

// Tiered resolver — session override wins, then agent-level, then global default
func Resolve(global, agentLevel, sessionOverride *LLMConfig) LLMConfig

// Credential security rules:
// 1. API keys never stored in DB or config files
// 2. CredentialRef holds only the env var name
// 3. Actual key resolved at runtime: os.Getenv(credentialRef)
// 4. Absent env var at startup → agent marked unavailable; no silent fallback
// 5. llm_config JSONB column stores only {provider, model, credential_ref}
```

---

### 8.3 A2A Interaction Model

The SDK (`github.com/a2aproject/a2a-go/v2`) is **message-based** — no custom task schema. Domain context is packed as a `DataPart` inside `a2a.SendMessageRequest`.

**Wire format (backend → agent):**

```go
type BrainstormPayload struct {
    Role         string    `json:"role"`          // "build" | "review" | "refine" | "devils_advocate"
    SystemPrompt string    `json:"system_prompt"` // assembled: agent base prompt + skill fragments
    LLMConfig    LLMConfig `json:"llm_config"`    // resolved tiered config (no raw key)
    State        any       `json:"state"`         // CanonicalState
}
```

**Backend dispatch (`client.go`):**

```go
// 1. Resolve tiered LLM config
llmCfg := resolver.Resolve(globalCfg, agentCfg, sessionOverride)

// 2. Assemble skill prompt fragments
systemPrompt := BuildSystemPrompt(agent.SystemPrompt, activeSkills)

// 3. Pack as DataPart in A2A message
msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewDataPart(BrainstormPayload{...}))

// 4. Resolve AgentCard and send
card, _ := agentcard.DefaultResolver.Resolve(ctx, agent.Endpoint)
client, _ := a2aclient.NewFromCard(ctx, card)
result, _ := client.SendMessage(ctx, &a2a.SendMessageRequest{Message: msg})

// 5. Extract updated state from artifact DataPart
updatedState := extractStateFromResult(result)
```

**Agent executor (`executor.go`):**

```go
func (e *BrainstormExecutor) Execute(
    ctx context.Context,
    execCtx *a2asrv.ExecutorContext,
) iter.Seq2[a2a.Event, error] {
    return func(yield func(a2a.Event, error) bool) {
        // 1. Extract payload from DataPart
        var payload BrainstormPayload
        for _, part := range execCtx.Message.Parts {
            if d := part.Data(); d != nil { /* unmarshal into payload */ }
        }

        // 2. Call LLM through LLMProvider interface
        resp, _ := e.llm.Generate(ctx, LLMRequest{
            SystemPrompt: payload.SystemPrompt,
            UserMessage:  marshalState(payload.State),
        })

        // 3. Emit A2A event sequence
        yield(a2a.NewSubmittedTask(execCtx, nil), nil)
        yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil), nil)
        yield(a2a.NewArtifactEvent(execCtx, a2a.NewDataPart(updatedState)), nil)
        yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
    }
}
```

The agent binary **does not** know about skill names, DB records, or credential refs. It receives the fully assembled `SystemPrompt` and operates on `State`.

---

### 8.4 Iteration Engine Algorithm

```go
agents := session.GetOrderedAgents() // min 2, ordered by session_agents.position ASC

for i := 1; i <= maxIter; i++ {
    current := state

    // Ordered pipeline: each agent receives the previous agent's output
    for _, agent := range agents {
        activeSkills := resolveActiveSkills(agent, session.SkillOverrides[agent.ID])
        out, err := agent.Dispatch(ctx, agent, agent.Role, activeSkills, session.LLMOverride[agent.ID], current)
        if err != nil { /* handle */ }
        current = out
    }

    newState := state.Merge(state, current)
    newState.Meta.Iteration = i

    // Persist state after each full pipeline pass
    persistState(ctx, session.ID, newState)

    if convergence.Check(state, newState) {
        break
    }

    state = newState
}
```

Key rules:

- Roles are **fixed at session creation** — no runtime alternation
- Each agent in the pipeline receives the cumulative output of the previous, not the original state
- State is persisted after each full pipeline pass (not per-agent within a pass)
- Max iterations cap prevents infinite loop

---

### 8.5 Merge Strategy Rules

1. **Union risks** — deduplicate by normalized text hash; do not drop unique risks
2. **Remove resolved** — risks marked `resolved: true` are removed from the next iteration's state
3. **Collapse duplicate plan steps** — steps with identical titles are merged (keep the more detailed one)
4. **Reject vague outputs** — plan steps with description < 10 words are dropped
5. **Stability rule** — if prev and next agree on a field value (exact match), lock it; do not overwrite
6. **Persistent conflict** — if the same field has been toggled back-and-forth for 3+ iterations → flag for user resolution (set `open_questions` entry)

---

### 8.6 Convergence Stop Conditions

Stop (return `true` from `convergence.Check`) when **all** of the following hold:

1. No new critical risks appeared (risks not in `prev` but in `next` with severity = `critical`)
2. Execution plan is "complete" — all steps have a non-empty description and no step is referenced in `open_questions`
3. `|next.Metrics.Confidence - prev.Metrics.Confidence| < convergenceThreshold` (default `0.02`)

OR stop when **any** of the following hold:

4. User explicitly approves (session status set to `approved` via `POST /sessions/{id}/finalize`)
5. `iteration >= maxIter` (default `10`, configurable via `MAX_ITERATIONS` env var)

---

### 8.7 API Endpoint Definitions

**Skills:**

```
POST   /skills                         create skill
GET    /skills                         list all skills
GET    /skills/{id}
PUT    /skills/{id}
DELETE /skills/{id}
POST   /agents/{id}/skills/{skill_id}  attach skill to agent
DELETE /agents/{id}/skills/{skill_id}  detach skill from agent
GET    /agents/{id}/skills             list skills for agent
```

**Agents:**

```
POST   /agents                         register agent
GET    /agents                         list all agents (includes skills[])
GET    /agents/{id}
PUT    /agents/{id}
DELETE /agents/{id}
```

**Sessions:**

```
POST   /sessions                       create session
GET    /sessions/{id}
POST   /sessions/{id}/iterate          trigger one iteration
POST   /sessions/{id}/finalize         approve + write .md artifacts
```

**`POST /sessions` request body:**

```json
{
  "idea": "...",
  "agent_ids": ["uuid-1", "uuid-2"],
  "max_iterations": 10,
  "role_overrides": { "uuid-1": "build", "uuid-2": "review" },
  "llm_overrides": {
    "uuid-1": { "model": "claude-opus-4", "credential_ref": "CLAUDE_API_KEY" }
  },
  "skill_overrides": { "uuid-1": ["skill-uuid-a"], "uuid-2": [] }
}
```

`skill_overrides`: optional. Omitted = use agent's default attached skills. Empty array `[]` = disable all skills for that agent in this session.

---

### 8.8 Module Responsibilities Summary

| Module            | Owns                                                           |
| ----------------- | -------------------------------------------------------------- |
| `session/`        | Session lifecycle, session-agent bindings, idea storage        |
| `iteration/`      | Iteration loop trigger, engine invocation, state persistence   |
| `agent/`          | Agent registry, skill registry, A2A dispatch, role assignment  |
| `state/`          | Canonical state type, merge algorithm, validator               |
| `convergence/`    | Convergence detection — pure function, no DB access            |
| `markdown/`       | `architecture.md` and `roadmap.md` generation                  |
| `platform/llm`    | LLMProvider interface, tiered resolver, Copilot + Claude impls |
| `platform/a2a`    | a2aclient factory, AgentCard resolution, payload wrapper       |
| `platform/db`     | pgx pool, migration runner                                     |
| `platform/config` | All env var access (single file, nowhere else)                 |

---

### 8.9 Frontend Component Tree + Svelte Store Shapes

**Component tree:**

```
routes/+page.svelte
  └── AgentSelector.svelte            (session creation — pick agents, set roles/skills)

routes/session/[id]/+page.svelte
  ├── AgentPanel.svelte × N           (one per active session agent)
  ├── ControlPanel.svelte             (Next Iteration, Approve, Inject Feedback)
  ├── StateView.svelte                (Architecture, Execution Plan, Risks)
  └── Timeline.svelte                 (iteration history)

routes/agents/+page.svelte
  └── (inline agent CRUD + AgentSelector preview)

routes/skills/+page.svelte
  └── SkillManager.svelte             (skill library + agent attachment)
```

**Svelte store shapes:**

```ts
// sessionStore
{
  session_id: string | null;
  idea: string;
  state: CanonicalState | null;
  iteration: number;
  agents: SessionAgent[];    // ordered list for active session, includes skills[]
  loading: boolean;
}

// agentRegistryStore
{
  agents: Agent[];           // full registry; each agent includes skills[]
  skills: Skill[];           // full skill library
  loading: boolean;
}
```

---

### 8.10 Failure Modes and Mitigations

| Failure           | Symptom                                                                 | Mitigation                                                                                  |
| ----------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| Oscillation       | State alternates between two values; `ConfidenceDelta` stays high       | Stability bias in merge (lock agreed fields); user override via `/finalize`                 |
| Weak critique     | Reviewer returns same state; confidence climbs without real improvement | Strong role-specific system prompt contracts; enforce minimum diff in `validator.go`        |
| Schema drift      | Agent returns malformed state; merge panics                             | `state.Validate()` on every agent response; reject and retry (max 2 retries)                |
| LLM inconsistency | Copilot returns variable JSON structure                                 | Low temperature; strict JSON schema in system prompt; structured output mode if available   |
| Credential absent | Agent marks itself unavailable at startup                               | `ResolveKey()` returns error; `session.CreateSession` rejects unavailable agents with `400` |

---

### 8.11 DB Schema

```sql
-- agents
CREATE TABLE agents (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL UNIQUE,
    description    TEXT,
    default_role   TEXT NOT NULL,
    system_prompt  TEXT,
    llm_config     JSONB,      -- {provider, model, credential_ref} ONLY — never raw key
    endpoint       TEXT NOT NULL,
    created_at     TIMESTAMPTZ DEFAULT now()
);

-- skills
CREATE TABLE skills (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    prompt      TEXT NOT NULL,   -- injected into system prompt when skill is active
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- agent_skills (many-to-many)
CREATE TABLE agent_skills (
    agent_id   UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    skill_id   UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (agent_id, skill_id)
);

-- sessions
CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idea            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active',  -- active | converged | approved | failed
    max_iterations  INT NOT NULL DEFAULT 10,
    current_state   JSONB,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- session_agents (binds agents to a session with position + role + overrides)
CREATE TABLE session_agents (
    session_id      UUID NOT NULL REFERENCES sessions(id),
    agent_id        UUID NOT NULL REFERENCES agents(id),
    position        INT  NOT NULL,      -- pipeline order (0-indexed)
    role            TEXT NOT NULL,      -- role assigned for this session
    llm_override    JSONB,              -- optional per-session LLM config
    skill_overrides JSONB,              -- null = use defaults; [] = disable all
    PRIMARY KEY (session_id, agent_id)
);
```

---

### 8.12 LLM Config: Tiered Resolver + Credential Security

**Resolution order (highest priority first):**

```
session_agents.llm_override  →  agents.llm_config  →  global default (env vars)
```

The `Resolve(global, agentLevel, sessionOverride *LLMConfig) LLMConfig` function applies the override only for non-zero fields — a session override of `{model: "claude-opus-4"}` (no provider or credential) merges with the agent-level provider and credential.

**Credential security rules (non-negotiable):**

1. API keys are **never stored in the DB, config files, or logs**
2. `CredentialRef` stores only the **env var name** (e.g. `"CLAUDE_API_KEY"`)
3. Actual key resolved at runtime: `os.Getenv(config.CredentialRef)`
4. If env var is absent at startup → `CheckAvailability()` marks agent unavailable; no silent fallback to another provider
5. `llm_config` JSONB stores only `{provider, model, credential_ref}` — auditable, never sensitive

---

### 8.13 Role Catalog and Distribution

```go
type Role string

const (
    RoleBuilder        Role = "build"
    RoleReviewer       Role = "review"
    RoleRefiner        Role = "refine"
    RoleDevilsAdvocate Role = "devils_advocate"
)
```

**Behavior per role:**

| Role              | Behavior                                           |
| ----------------- | -------------------------------------------------- |
| `build`           | Proposes / expands architecture and execution plan |
| `review`          | Critiques output, identifies risks and gaps        |
| `refine`          | Synthesizes prior outputs, removes contradictions  |
| `devils_advocate` | Challenges assumptions, surfaces edge cases        |

**Default distribution by agent count:**

| Agents | Role assignment                                          |
| ------ | -------------------------------------------------------- |
| 2      | build, review                                            |
| 3      | build, review, refine                                    |
| 4      | build, review, refine, devils_advocate                   |
| 5+     | cycles catalog from position 0; extras assigned `review` |

User may override any agent's role at session creation via `role_overrides` map.

---

### 8.14 Skill Injection Logic

Skills are **prompt-level behaviors** — not external tool calls. They are assembled server-side before dispatch.

**Assembly (`BuildSystemPrompt` in `agent/client.go`):**

```
effective_prompt = agent.system_prompt
                 + "\n\n" + skill_1.prompt
                 + "\n\n" + skill_2.prompt
                 + ...
```

**Active skill resolution at dispatch time:**

1. If `session.skill_overrides[agent_id]` is **absent** → use `agent_skills` table (agent defaults)
2. If `session.skill_overrides[agent_id]` is **present (non-nil)** → use that list (may be empty)
3. Empty list `[]` → no skill prompts injected; only base `system_prompt` used

The agent binary receives the final assembled `SystemPrompt` string. It does not know about skill names, IDs, or the `agent_skills` table.

---

### 8.15 Definition of Done

A task session is "done" when:

- [ ] All listed files are created and contain non-stub implementation
- [ ] `go build ./...` passes (backend/agent) or `pnpm check` + `pnpm build` passes (frontend)
- [ ] `go vet ./...` reports zero issues
- [ ] No `LLMProvider` implementation is called directly from business logic (interface only)
- [ ] No raw API key appears anywhere in source, test fixtures, or config files
- [ ] No `os.Getenv()` call appears outside `platform/config/config.go` (backend) or `internal/config/config.go` (agent)
- [ ] All cross-module calls go through service interfaces, not repositories (modules do not import each other's repositories)

---

### 8.16 Frontend Design System Specification

All UI tasks (Tasks 16–25) must use the following design tokens and component classes. Never hard-code color values inline; always reference the CSS custom property.

**Color tokens (defined in `frontend/src/app.css` `:root`):**

```css
:root {
  --bg-0: #f5efe4; /* warm cream — page background base */
  --bg-1: #e8ecf7; /* cool blue-grey — page background accent */
  --ink-900: #151b2f; /* near-black — primary text */
  --ink-700: #2d3655; /* dark — secondary headings */
  --ink-500: #5a6282; /* mid — secondary text */
  --ink-300: #a8aec7; /* light — placeholders, borders */
  --accent: #0bb6d9; /* cyan — primary interactive */
  --accent-2: #1f7ae0; /* blue — gradient end, links */
  --ok: #1b9f66; /* green — success, done state */
  --warn: #d48806; /* amber — warning, review state */
  --danger: #ce3158; /* red — error, delete action */
  --surface: rgba(255, 255, 255, 0.72); /* glassmorphism card fill */
  --blur: blur(8px); /* backdrop blur */
  --shadow-md: 0 10px 30px rgba(35, 46, 82, 0.1);
}
```

**Page background (set on `<body>` or `<main>`):**

```css
background:
  radial-gradient(1200px 600px at 10% 10%, #fff8ec, transparent),
  radial-gradient(900px 500px at 90% 10%, #e8f7ff, transparent),
  linear-gradient(135deg, #f5efe4, #e8ecf7);
min-height: 100vh;
```

**Artboard (page-width container):**

```css
.artboard {
  width: min(1300px, 94vw);
  margin: 28px auto;
}
```

**Panel / Card primitives:**

```css
.panel {
  background: var(--surface);
  backdrop-filter: var(--blur);
  border-radius: 18px;
  box-shadow: var(--shadow-md);
  border: 1px solid rgba(255, 255, 255, 0.6);
  padding: 28px;
}
.card {
  background: var(--surface);
  backdrop-filter: var(--blur);
  border-radius: 14px;
  box-shadow: 0 4px 16px rgba(35, 46, 82, 0.07);
  border: 1px solid rgba(255, 255, 255, 0.6);
  padding: 20px;
}
```

**Topbar:**

```css
.topbar {
  position: sticky;
  top: 0;
  z-index: 100;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(168, 174, 199, 0.3);
  padding: 0 40px;
  height: 56px;
  display: flex;
  align-items: center;
  gap: 24px;
}
```

**Button classes:**

```css
.btn-primary {
  background: linear-gradient(135deg, var(--accent), var(--accent-2));
  color: #fff;
  border: none;
  border-radius: 10px;
  padding: 10px 24px;
  font-weight: 600;
  cursor: pointer;
}
.btn-ghost {
  background: transparent;
  color: var(--ink-700);
  border: 1.5px solid var(--ink-300);
  border-radius: 10px;
  padding: 9px 20px;
  cursor: pointer;
}
.btn-danger {
  background: var(--danger);
  color: #fff;
  border: none;
  border-radius: 10px;
  padding: 9px 20px;
  font-weight: 600;
  cursor: pointer;
}
```

**Role badges:**

```css
.badge-build {
  background: #dbeafe;
  color: #1d4ed8;
}
.badge-review {
  background: #fef3c7;
  color: #92400e;
}
.badge-refine {
  background: #d1fae5;
  color: #065f46;
}
.badge-devils-advocate {
  background: #ede9fe;
  color: #5b21b6;
}
/* common: border-radius 6px, padding 2px 8px, font-size 0.72rem, font-weight 600 */
```

**Status / info chips:**

```css
.chip-live {
  background: #d1fae5;
  color: var(--ok);
}
.chip-ok {
  background: #d1fae5;
  color: var(--ok);
}
.chip-warn {
  background: #fef3c7;
  color: var(--warn);
}
.chip-danger {
  background: #fee2e2;
  color: var(--danger);
}
/* common: border-radius 20px, padding 3px 10px, font-size 0.75rem, font-weight 600 */
```

**Pipeline stage states:**

```css
.stage-done {
  border-left: 3px solid var(--ok);
  opacity: 1;
}
.stage-running {
  border-left: 3px solid var(--accent);
  opacity: 1;
}
.stage-waiting {
  border-left: 3px solid var(--ink-300);
  opacity: 0.5;
}
```

**Mono log block (inside PipelineStage and finalize view):**

```css
.log-block {
  background: #1a1d2e;
  border-radius: 8px;
  padding: 14px 18px;
  font-family: "IBM Plex Mono", monospace;
  font-size: 0.78rem;
  color: #a8d8ea;
  white-space: pre-wrap;
  line-height: 1.6;
}
```

**Typography:**

```css
body {
  font-family: "IBM Plex Sans", sans-serif;
  color: var(--ink-900);
}
h1,
h2,
h3 {
  font-family: "Space Grotesk", sans-serif;
  color: var(--ink-900);
}
code,
pre,
kbd {
  font-family: "IBM Plex Mono", monospace;
}
```

**Google Fonts import (in `<head>`):**

```html
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link
  href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400&family=IBM+Plex+Sans:wght@300;400;500&family=Space+Grotesk:wght@500;700&display=swap"
  rel="stylesheet"
/>
```

---

### 8.17 Frontend Route Map (v1.1)

Complete route structure after Tasks 16–25. All routes are SvelteKit `+page.svelte` files under `frontend/src/routes/`.

| Route                    | File                                        | Purpose                                       | New in v1.1 |
| ------------------------ | ------------------------------------------- | --------------------------------------------- | ----------- |
| `/`                      | `routes/+page.svelte`                       | Session creation — idea input + agent pool    | Redesigned  |
| `/session/[id]`          | `routes/session/[id]/+page.svelte`          | Session workspace — sequential pipeline view  | Redesigned  |
| `/session/[id]/finalize` | `routes/session/[id]/finalize/+page.svelte` | Export view — generation log + download cards | **New**     |
| `/settings`              | `routes/settings/+page.svelte`              | Unified agents + skills + roles management    | **New**     |
| `/settings/agent/new`    | `routes/settings/agent/new/+page.svelte`    | Create agent form                             | **New**     |
| `/settings/agent/[id]`   | `routes/settings/agent/[id]/+page.svelte`   | Edit agent form                               | **New**     |
| `/settings/skill/new`    | `routes/settings/skill/new/+page.svelte`    | Create skill form                             | **New**     |
| `/settings/skill/[id]`   | `routes/settings/skill/[id]/+page.svelte`   | Edit skill form                               | **New**     |
| `/history`               | `routes/history/+page.svelte`               | Session history — stats + searchable table    | **New**     |
| `/agents`                | `routes/agents/+page.svelte`                | Redirect → `/settings?tab=agents`             | Redirect    |
| `/skills`                | `routes/skills/+page.svelte`                | Redirect → `/settings?tab=skills`             | Redirect    |

**Component tree (v1.1):**

```
routes/+layout.svelte
  └── <WarningModal>                       (global modal, from uiStore)
  └── <slot />

routes/+page.svelte (Home)
  └── inline agent pool (AgentSelector.svelte — simplified)

routes/session/[id]/+page.svelte (Session)
  ├── <ConfidenceBar>                      (pass summary bar)
  ├── <PipelineStage> × N                  (replaces AgentPanel)
  ├── <CanonicalStatePanel>                (replaces StateView)
  └── <RiskBoard>

routes/session/[id]/finalize/+page.svelte (Export)
  └── (log panel + output cards — self-contained)

routes/settings/+page.svelte (Settings)
  └── (Agents tab / Skills tab / Roles tab — self-contained)

routes/history/+page.svelte (History)
  └── (stat cards + table — self-contained)
```

**Deprecated components (kept for build compatibility, marked `@deprecated`):**

| Component             | Replaced By                      |
| --------------------- | -------------------------------- |
| `AgentPanel.svelte`   | `PipelineStage.svelte`           |
| `ControlPanel.svelte` | Inline in session page           |
| `StateView.svelte`    | `CanonicalStatePanel.svelte`     |
| `Timeline.svelte`     | Pass summary bar in session page |

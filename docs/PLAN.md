# PLAN.md — a2a-brainstorm Implementation Plan

> **Version:** 1.0
> **Date:** 2026-05-12
> **Author:** Core, Data and AI Team
> **Status:** Ready for Implementation
> **Source of Truth:** `docs/A2A-agent-Brainstorm.md`

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
                                                            ▼
                                            Task 13 (Frontend: Session Workspace)
                                                            │
                                                            ▼
                                            Task 14 (Frontend: Agent Registry + Skills)
                                                            │
                                          ┌─────────────────┴──────────────┐
                                          │                                  │
                                      Task 11                           Task 15
                                   (must complete               (Integration Tests + Docs)
                                   before Task 15)
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

## 6. Task Summary

| Task | Name                                       | Key Files                                                                         | Depends On       | Complexity |
| ---- | ------------------------------------------ | --------------------------------------------------------------------------------- | ---------------- | ---------- |
| 1    | Project Scaffold                           | `go.work`, `go.mod` ×2, `docker-compose.yml`, `Makefile`, FE scaffold             | —                | Low        |
| 2    | Platform: Config + DB + Logger             | `platform/config/`, `platform/db/`, `platform/logger/`                            | Task 1           | Low        |
| 3    | Platform: LLM Abstraction                  | `platform/llm/provider.go`, `resolver.go`, `copilot.go`                           | Task 2           | Medium     |
| 4    | Platform: A2A Layer                        | `platform/a2a/client.go`, `types.go`, `agent/internal/config/`                    | Task 2           | Medium     |
| 5    | State Module                               | `modules/state/model.go`, `merge.go`, `validator.go`                              | Tasks 3, 4       | Medium     |
| 6    | Agent Module: Models + DB Schema           | `modules/agent/model.go`, `repository.go`, `role.go`, `001_agents.sql`            | Tasks 1, 5       | Medium     |
| 7    | Agent Module: Service + Handler + Dispatch | `modules/agent/service.go`, `handler.go`, `client.go`                             | Tasks 6, 3, 4    | High       |
| 8    | Session Module                             | `modules/session/*`, `003_sessions.sql`                                           | Task 7           | Medium     |
| 9    | Iteration Engine + Convergence             | `iteration/engine.go`, `convergence/engine.go`                                    | Tasks 5, 7, 8    | High       |
| 10   | Markdown + Backend Wire-up                 | `markdown/generator.go`, `cmd/server/main.go`, `platform/http/router.go`          | Tasks 9, 8       | Medium     |
| 11   | Agent Service Binary                       | `agent/agentcard.go`, `executor/executor.go`, `agent/cmd/server/main.go`          | Tasks 3, 4       | High       |
| 12   | Frontend: Scaffold + Stores + API Client   | `lib/types.ts`, `stores/*.ts`, `services/api.ts`                                  | Task 1           | Medium     |
| 13   | Frontend: Session Workspace                | `AgentPanel.svelte`, `ControlPanel.svelte`, `StateView.svelte`, `Timeline.svelte` | Task 12          | Medium     |
| 14   | Frontend: Agent Registry + Skills          | `AgentSelector.svelte`, `SkillManager.svelte`, routes                             | Task 12          | Medium     |
| 15   | Integration Tests + Docs                   | `*_test.go` files, `README.md`                                                    | Tasks 11, 13, 14 | Medium     |

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

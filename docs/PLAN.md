# PLAN.md — a2a-brainstorm Implementation Plan

> **Version:** 3.0
> **Date:** 2026-06-12 (inserted Task 37 — README.md Generation Quality Overhaul; renumbered original Tasks 37–47 to Tasks 38–48)
> **Change in v3.0:** Inserted Task 37 — README.md: Generation Quality Overhaul (Enricher + Correct Section Structure). Rewrites `generator_readme.go` to produce all 13 required sections (title, tagline, description paragraph, golden rule, "What it is NOT", "When to use" with Mermaid flowchart + numbered scenarios + code, Prerequisites/Installation, Quick Start, Architecture with ASCII and Mermaid, Command Reference table, Tech Stack, Repository Format, Contributing); removes wrong sections (Known Risks top-level, For AI Agents appendix, Phase-N roadmap). Adds `markdown/aigen/readme_enricher.go` (Approach B) LLM post-pass with `ReadmeEnrichmentOverlay` schema filling golden_rule, when_to_use scenarios, Mermaid flowcharts, quick_start_commands, command_reference, contributing_note, prerequisites. Adds `agent/internal/executor/prompts/readme_format.go` (Approach C) agent prompt constant injected when `readme` is in `output_docs`. Updates rubric ForbiddenStrings + RequiredPatterns. Original Tasks 37–47 renumbered 38–48. New §8.32 documents the README quality standard, enricher schema, enricher LLM prompt, Approach C fragment, and rubric assertions.
> **Change in v2.0:** Inserted Task 36 — PLAN.md Generation Quality Overhaul (Enricher + Correct Task Format). Adds a new `markdown/aigen/plan_enricher.go` single-call LLM post-pass (Approach B) that restructures generated PLAN.md: merges §3 Phase Breakdown into properly-formatted §5 Implementation Tasks, rewrites thin "see phase deliverables" stubs into full `**Goal:**`/`**Files to create:**`/`**Validation:**`/`**Prompt context needed:**` specs, adds `**Invariant check:**` and `**Layer(s) affected:**` per task, deduplicates the Risks section, auto-generates §8 Deep Knowledge from canonical state, and emits an ASCII dependency graph. Adds a new `agent/internal/executor/prompts/plan_format.go` constant (Approach C) injected into agent system prompts when `plan` is in `output_docs`, directing agents to write tasks in the correct spec format (not "Phase N") with runnable validation commands. Fixes §1 Goals always-empty bug. Original Tasks 36–46 renumbered 37–47. New §8.31 documents the plan quality standard, enricher schema, and Approach C prompt spec.
> **Author:** Core, Data and AI Team
> **Status:** Ready for Implementation
> **Source of Truth:** `docs/A2A-agent-Brainstorm.md`
> **Change in v1.1:** Added Tasks 16–25 — Polished UI redesign matching `frontend/mockups/future-polished-mockup.html`. New routes: `/settings`, `/history`, `/session/[id]/finalize`. New components: `PipelineStage`, `ConfidenceBar`, `CanonicalStatePanel`, `RiskBoard`, `WarningModal`. Backend addition: `GET /sessions` list endpoint + markdown content return.
> **Change in v1.2:** Added Tasks 26–27 — OpenCode server integration as an optional LLM provider for the agent binary. New `OpenCodeProvider` implementation, provider selection switch in `agent/cmd/server/main.go`, Docker Compose profile-based service, and full startup documentation.
> **Change in v1.3:** Added Tasks 28–31 — four feature enhancements per blueprint §22: (28) Selectable Output Documents with edit-at-finalize support; (29) `PLAN.md` and `README.md` generators producing ≥1000 lines each; (30) Per-Agent Run button using Option A preview/apply flow; (31) Real-time SSE agent progress stream replacing simulated progress.
> **Change in v1.4:** Added Tasks 33–38 — MCP (Model Context Protocol) tool integration: (33) DB schema for MCP server registry; (34) Backend MCP server CRUD module; (35) Agent–MCP server association + `BrainstormPayload` extension; (36) Agent MCP client package (stdio + HTTP transports, JSON-RPC 2.0); (37) LLM `GenerateWithTools` interface + multi-turn tool-use executor loop; (38) Frontend MCP settings tab, smart JSON config import (Claude Desktop / VS Code / Cursor / Zed / Windsurf / canonical), and agent assignment UI.
> **Change in v1.5:** Inserted Task 32 — Generated Document Quality Overhaul. Fixes title bug (full idea text used as H1), eliminates idea duplication, deletes `enforceMinLines` padding, blocks finalize when state is sparse (HTTP 422), introduces deterministic `{slug}_{kind}.md` filename pattern (e.g. `match-point_architecture.md`), and enriches the canonical state schema + agent role prompts so generators produce depth from real content instead of boilerplate. Original v1.4 MCP tasks renumbered 32–37 → 33–38; deep knowledge sections renumbered §8.23 → §8.24, §8.24 → §8.25, §8.25 → §8.26. New §8.23 documents the doc-quality standard.
> **Change in v1.6:** Inserted Task 33 — AI-Driven Hybrid Document Generator with skill bundle injection. Adds a new `markdown/aigen/` sub-package that wraps the deterministic generators with per-document AI passes orchestrated by an injectable `LLMProvider` and a `SkillBundle` (modular monolith skill, vertical-slice skill, api-design skill, roadmap-spec skill, plan-management skill — each loaded as a prompt fragment). Introduces a section-level rubric validator with auto-repair loop and a `finalize_mode` config switch (`deterministic` | `hybrid` | `ai`) that defaults to `hybrid`. Deterministic generators retained as fallback. Original v1.5 MCP tasks renumbered 33–38 → 34–39. New §8.27 documents the AI-doc-gen contract.
> **Change in v1.8:** Inserted Task 34 — Guided Onboarding v2 (tech constraints + discovery clarify flow per `frontend/mockups/v2.html`), roadmap→plan output consolidation, dual-audience document quality standard v2, and DB-backed artifact persistence for finalized sessions + history UX. Original v1.7 attachment tasks renumbered 34–44 → 35–45; attachment migrations shifted to `009`/`010`; MCP migrations shifted to `011`/`012`. New §8.29 documents the onboarding flow, discovery-hints contract, tech-constraint injection, plan-only output model, and artifact cache schema.
> **Change in v1.9:** Inserted Task 35 — Architecture.md State Enricher + High-Quality Section Generator. Adds a 16-section deterministic generator rewrite for `architecture.md` (Problem Statement, Solution, Scope, Layers, Tech Stack, Data Flows, Module Boundaries, Architecture Decisions, Extension Points, Security Considerations, Quality Targets, System Guarantees, Risks, Assumptions, Open Questions, Definition of Done) plus a new `aigen/enricher.go` single-call LLM pre-pass that populates optional narrative state fields before the deterministic render. Fixes `strings.ToTitle` severity bug. Adds document metadata block and Table of Contents. Original Tasks 35–45 renumbered 36–46. New §8.30 documents the section contract, `ArchEnrichmentOverlay` schema, enricher LLM prompt, rubric targets, and fallback specification.
> **Change in v1.7:** Inserted Tasks 34–38 — Hierarchical Attachment Context system (ChatGPT-style `+` upload UX). Adds a new `modules/attachment/` vertical slice + `platform/extractor/`, `platform/embeddings/`, and `platform/blobstore/` (MinIO) infrastructure. Supports four input types — file (PDF/DOCX/MD/TXT), image (vision-described), URL (server-fetched), raw text paste. Hybrid scope model (session / iteration / agent) drives lifecycle. RAG-lite retrieval via pgvector cosine similarity injects top-K chunks into each agent dispatch through a new `AttachmentRetriever` interface threaded into the iteration engine. `BrainstormPayload` gains an optional `Attachments []AttachmentChunk` field. Frontend: ChatGPT-style attachment menu mounted on home page, session page, and `PipelineStage`. Original v1.6 MCP tasks renumbered 34–39 → 39–44; their migration numbers shift 006/007 → 008/009. New §8.28 documents the attachment system contract.

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
                                                     │
                                              All Tasks 1–25
                                                     │
                                                     ▼
                                          Task 26 (Agent: OpenCode LLM Provider)
                                                     │
                                                     ▼
                                          Task 27 (Infrastructure: OpenCode Service Wiring)
                                                     │
                                                     ▼
                                          Task 28 (Backend: Selectable Output Documents)
                                                     │
                                                     ▼
                                          Task 29 (Backend: PLAN.md + README.md Generators)
                                                     │
                                                     ▼
                                          Task 30 (Per-Agent Preview/Apply — Backend + Frontend)
                                                     │
                                                     ▼
                                          Task 31 (SSE Real-time Progress — Backend + Frontend)
                                                     │
                                                     ▼
                                          Task 32 (Generated Document Quality Overhaul)
                                                     │
                                                     ▼
                                          Task 33 (AI-Driven Hybrid Doc Generator + Skill Bundle)
                                                     │
                                                     ▼
                                          Task 34 (Guided Onboarding v2 + Plan Consolidation + Artifact Persistence)
                                                     │
                                                     ▼
                                          Task 35 (Architecture.md: State Enricher + High-Quality Section Generator)
                                                     │
                                                     ▼
                                          Task 36 (PLAN.md: Generation Quality Overhaul — Enricher + Correct Task Format)
                                                     │
                                                     ▼
                                          Task 37 (README.md: Generation Quality Overhaul — Enricher + Correct Section Structure)
                                                     │
                                                     ▼
                                          Task 38 (DB: Attachments + Chunks Schema — pgvector)
                                                     │
                                                     ▼
                                          Task 39 (Platform: Extractor + Embeddings + Blobstore)
                                                     │
                                                     ▼
                                          Task 40 (Backend: Attachment Module — CRUD + Upload Pipeline)
                                                     │
                                                     ▼
                                          Task 41 (Backend: AttachmentRetriever + Payload Extension + Engine Wiring)
                                                     │
                                                     ▼
                                          Task 42 (Frontend: Attachment Menu + Upload Modal + Scope UX)
                                                     │
                                                     ▼
                                          Task 43 (DB: MCP Server Registry Schema)
                                                     │
                                                     ▼
                                          Task 44 (Backend: MCP Server Module — CRUD)
                                                     │
                              ┌──────────────────────┴──────────────────────┐
                              ▼                                              ▼
                  Task 45 (Backend: Agent–MCP                    Task 46 (Agent: MCP
                  Association + Payload Extension)                Client Package)
                              │                                              │
                              └──────────────────────┬──────────────────────┘
                                                     ▼
                                          Task 47 (Agent: LLM Tool-Use + Executor Loop)
                                                     │
                                                     ▼
                                          Task 48 (Frontend: MCP Settings + Smart Import)
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

### Task 26 — Agent: OpenCode LLM Provider <!-- ✅ Task 26 completed -->

**Goal:** Add `OpenCodeProvider` to the agent binary — a new `LLMProvider` implementation that proxies requests through a running OpenCode server instance (which itself is authenticated to GitHub Copilot or any other OpenCode-supported provider). The agent lazily creates one OpenCode session per process lifetime and reuses it for all subsequent `Generate` calls.

**Files to create / modify:**

- `agent/internal/llm/opencode.go` — **new**
  - `OpenCodeProvider` implements `LLMProvider` (same interface defined in this package)
  - `OpenCodeConfig` struct: `BaseURL string`, `ProviderID string`, `ModelID string`, `UsernameRef string`, `PasswordRef string`
    - `ProviderID` + `ModelID` are parsed from `AGENT_OPENCODE_MODEL` by splitting on `/` (e.g. `"github/gpt-4o"` → `{ProviderID: "github", ModelID: "gpt-4o"}`)
  - `NewOpenCodeProvider(cfg OpenCodeConfig, httpClient *http.Client, resolveKey func(string)(string,error)) *OpenCodeProvider`
    - `resolveKey` must be `config.GetLLMAPIKey` — keeps `os.Getenv` confined to `config/config.go`
    - If `httpClient` is nil, use a default 120s-timeout client (LLM calls can be slow)
  - Session management:
    - `sessionID string` field (protected by `sync.Once`); populated via `ensureSession(ctx) (string, error)` on first `Generate` call
    - `ensureSession`: `POST {BaseURL}/session` body `{"title":"brainstorm"}` with Basic Auth header; extracts `session.id` from response JSON; see §8.18 for request/response shape
    - Credentials resolved at each call: `resolveKey(cfg.UsernameRef)` → username, `resolveKey(cfg.PasswordRef)` → password; return error if either is empty (no silent fallback)
  - `Generate(ctx, req LLMRequest) (LLMResponse, error)`:
    - Calls `ensureSession` first
    - `POST {BaseURL}/session/{sessionID}/message` with Basic Auth and JSON body:
      ```json
      {
        "parts": [{ "type": "text", "text": "<UserMessage>" }],
        "model": { "providerID": "<ProviderID>", "modelID": "<ModelID>" },
        "system": "<SystemPrompt>"
      }
      ```
    - Response: `{"info": {...}, "parts": [{"type":"text","text":"..."},...]}` — extract all `type=text` parts, concatenate, return as `LLMResponse.Content`; see §8.18 for full response shape
    - On HTTP 4xx from OpenCode server → return error immediately (no retry)
    - On HTTP 5xx or timeout → retry once with exponential backoff
  - Security: never log `UsernameRef` resolved value or `PasswordRef` resolved value
- `agent/internal/llm/opencode_test.go` — **new**
  - `httptest.NewServer` mock of OpenCode endpoints: `POST /session` + `POST /session/:id/message`
  - Test: `Generate` returns correct `LLMResponse.Content` extracted from text parts
  - Test: absent `OPENCODE_SERVER_PASSWORD` env var → `Generate` returns error (no silent fallback)
  - Test: `ensureSession` is called exactly once across multiple `Generate` calls (use call counter)
  - Test: HTTP 401 from OpenCode server → error propagated, not silently retried
- `agent/internal/config/config.go` — **modify** (add getters only; do not change existing getters):
  - `GetOpenCodeBaseURL() string` — reads `AGENT_OPENCODE_BASE_URL`; default `"http://localhost:4096"`
  - `GetOpenCodeModel() string` — reads `AGENT_OPENCODE_MODEL`; default `"github/gpt-4o"` (format: `providerID/modelID`)
  - `GetOpenCodeUsernameRef() string` — reads `AGENT_OPENCODE_USERNAME_REF`; default `"OPENCODE_SERVER_USERNAME"` (stores the env var name that holds the actual username)
  - `GetOpenCodePasswordRef() string` — reads `AGENT_OPENCODE_PASSWORD_REF`; default `"OPENCODE_SERVER_PASSWORD"` (stores the env var name that holds the actual password)
- `agent/cmd/server/main.go` — **modify**: extract provider construction into a local `buildLLMProvider` helper and add the `"opencode"` case:

  ```go
  func buildLLMProvider(logger *slog.Logger) llm.LLMProvider {
      switch config.GetLLMProvider() {
      case "opencode":
          model := config.GetOpenCodeModel()
          parts := strings.SplitN(model, "/", 2)
          providerID, modelID := parts[0], parts[1] // validated below
          if len(parts) != 2 || providerID == "" || modelID == "" {
              logger.Warn("AGENT_OPENCODE_MODEL must be 'providerID/modelID'; falling back to github/gpt-4o")
              providerID, modelID = "github", "gpt-4o"
          }
          return llm.NewOpenCodeProvider(llm.OpenCodeConfig{
              BaseURL:     config.GetOpenCodeBaseURL(),
              ProviderID:  providerID,
              ModelID:     modelID,
              UsernameRef: config.GetOpenCodeUsernameRef(),
              PasswordRef: config.GetOpenCodePasswordRef(),
          }, nil, config.GetLLMAPIKey)
      default: // "copilot" and any unrecognised value
          return llm.NewCopilotProvider(
              config.GetLLMModel(),
              config.GetLLMCredentialRef(),
              "", nil, config.GetLLMAPIKey,
          )
      }
  }
  ```

  - Call `buildLLMProvider(logger)` in `run()` in place of the existing `llm.NewCopilotProvider(...)` line
  - Add `"strings"` to imports

**Validation:**

- `cd agent && go build ./...`: zero errors
- `cd agent && go vet ./...`: zero issues
- `cd agent && go test ./internal/llm/...`: all three `opencode_test.go` tests pass (no real OpenCode server needed)
- Startup smoke: `AGENT_LLM_PROVIDER=opencode AGENT_OPENCODE_BASE_URL=http://localhost:4096 go run ./agent/cmd/server` starts and logs `"LLM provider: opencode"` (or equivalent) without panicking

**Prompt context needed:** §8.2 (LLMProvider interface + security), §8.3 (A2A interaction model), §8.12 (credential security rules), §8.18 (OpenCode server API reference, new in this task)

---

### Task 27 — Infrastructure: OpenCode Service Wiring <!-- ✅ Task 27 completed -->

**Goal:** Wire the OpenCode server into `docker-compose.yml` as a Docker Compose profile-based optional service, add all required env vars to `.env.example`, add Makefile convenience targets for the OpenCode workflow (start, one-time auth, status check), and document the end-to-end OpenCode setup in `docs/STARTUP_GUIDE.md`.

**Files to modify:**

- `docker-compose.yml` — add `opencode` service under `profiles: [opencode]` so it is **opt-in** and does not affect the default `docker-compose up` workflow:

  ```yaml
  opencode:
    image: node:22-slim
    profiles: [opencode]
    working_dir: /app
    entrypoint: >
      sh -c "npm install -g opencode-ai && opencode serve
             --hostname 0.0.0.0
             --port 4096
             --cors http://localhost:5173"
    ports:
      - "4096:4096"
    environment:
      - OPENCODE_SERVER_USERNAME=${OPENCODE_SERVER_USERNAME:-opencode}
      - OPENCODE_SERVER_PASSWORD=${OPENCODE_SERVER_PASSWORD}
    volumes:
      - opencode-auth:/root/.local/share/opencode
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:4096/global/health"]
      interval: 15s
      timeout: 5s
      retries: 5
      start_period: 30s
  ```

  - Add `opencode-auth:` entry under the top-level `volumes:` key to persist Copilot OAuth tokens across container restarts
  - Also add `opencode` as a dependency for the `agent` service when running in OpenCode mode (note in comments; avoid hard dep so default profile still works)

- `.env.example` — add new section below the existing LLM vars:

  ```dotenv
  # ── OpenCode Server (only required when AGENT_LLM_PROVIDER=opencode) ──────────
  # Switch the agent binary to route LLM calls through an OpenCode server instance.
  # The OpenCode server must be running and authenticated to a provider (e.g. Copilot).
  #
  # AGENT_LLM_PROVIDER=opencode

  # URL where the OpenCode server listens (service name inside Docker, localhost outside)
  AGENT_OPENCODE_BASE_URL=http://opencode:4096

  # model in "providerID/modelID" format understood by OpenCode
  # Examples: github/gpt-4o  |  anthropic/claude-sonnet-4-5  |  openai/gpt-4o
  AGENT_OPENCODE_MODEL=github/gpt-4o

  # Env var NAME that holds the OpenCode server HTTP Basic auth username
  AGENT_OPENCODE_USERNAME_REF=OPENCODE_SERVER_USERNAME

  # Env var NAME that holds the OpenCode server HTTP Basic auth password
  AGENT_OPENCODE_PASSWORD_REF=OPENCODE_SERVER_PASSWORD

  # Actual OpenCode server credentials (referenced by the _REF vars above)
  OPENCODE_SERVER_USERNAME=opencode
  OPENCODE_SERVER_PASSWORD=change-me-to-a-strong-password
  ```

- `Makefile` — add targets below the existing targets (do not modify existing targets):

  ```makefile
  ## opencode-up: Start the OpenCode server container (requires Docker profile)
  opencode-up:
  	docker compose --profile opencode up -d opencode

  ## opencode-auth: One-time GitHub Copilot OAuth inside the OpenCode container
  ## Run this once after first `make opencode-up`. Follow the device flow URL printed to stdout.
  opencode-auth:
  	docker compose exec opencode opencode /provider/github/oauth/authorize

  ## opencode-status: Check whether the OpenCode server is healthy
  opencode-status:
  	curl -sf http://localhost:4096/global/health | jq .
  ```

- `docs/STARTUP_GUIDE.md` — add a new section "**Running with OpenCode Server (optional)**" with:
  - When to use: when you want GitHub Copilot (or any OpenCode-supported provider) to handle LLM calls through the OpenCode layer, avoiding direct Copilot API key distribution to each agent container
  - Step 1 — set env vars: copy the OpenCode block from `.env.example` into `.env`; set `AGENT_LLM_PROVIDER=opencode`; choose a strong `OPENCODE_SERVER_PASSWORD`
  - Step 2 — start OpenCode: `make opencode-up` (waits for health check)
  - Step 3 — one-time Copilot auth: `make opencode-auth` → follow the device flow URL printed; tokens are persisted in the `opencode-auth` Docker volume
  - Step 4 — start all services: `make up` (backend + agent + postgres; OpenCode remains running from step 2)
  - Step 5 — verify: `make opencode-status` should return `{"healthy":true,...}`
  - Credential flow diagram (text-based):
    ```
    Agent binary
      → POST /session/:id/message (HTTP Basic auth)
    OpenCode server (port 4096)
      → GitHub Copilot API (OAuth token stored in volume)
    GitHub Copilot
      → LLM response
    ```
  - Note: the `opencode-auth` Docker volume (`opencode-auth`) persists the OAuth token across container restarts; re-run `make opencode-auth` only if the token expires or the volume is deleted
  - Troubleshooting table: common errors (401 from OpenCode server → wrong password, 503 → OpenCode not started, 403 from Copilot → re-run `make opencode-auth`)

**Validation:**

- `docker-compose config --profiles opencode`: shows `opencode` service with correct env + volume
- `docker-compose config` (no profile flag): `opencode` service absent from output (opt-in confirmed)
- `make opencode-up` starts the container; `make opencode-status` returns `{"healthy":true}`
- `.env.example` diff: only additions, no existing lines removed
- `cd agent && go build ./...`: zero errors (no Go changes in this task — infra only)

**Prompt context needed:** Task 26 (OpenCode provider config), §8.12 (credential security rules), §8.18 (OpenCode server API)

---

### Task 28 — Backend + Frontend: Selectable Output Documents <!-- ✅ Task 28 completed -->

**Goal:** Let users choose which artifacts (`architecture`, `roadmap`, `plan`, `readme`) are generated, both at session creation and at finalize time. Selection is editable while the session is `active`. See blueprint §22.1.

**Files to create / modify**

- `migrations/005_session_output_docs.sql` — new column `output_docs TEXT[] NOT NULL DEFAULT ARRAY['architecture','roadmap']` on `sessions`; backfill existing rows.
- `backend/internal/modules/session/model.go` — add `OutputDocs []string` field to `Session` struct + `CreateSessionInput` + `FinalizeInput`.
- `backend/internal/modules/session/repository.go` — read/write `output_docs` in SELECT/INSERT/UPDATE statements; add `UpdateOutputDocs(ctx, id, docs)`.
- `backend/internal/modules/session/service.go` — validate input (non-empty, no duplicates, only known keys: `architecture|roadmap|plan|readme`); reject with 409 if status `finalized`.
- `backend/internal/modules/session/handler.go` — wire `POST /sessions` body field; new `PATCH /sessions/{id}/output-docs`; accept optional `output_docs` override in `POST /sessions/{id}/finalize` body (persists override before generating).
- `backend/internal/modules/markdown/generator.go` — change `GenerateContent` signature to `GenerateContent(state, keys []string) (map[string]GeneratedDocument, error)`; iterate `keys`, dispatch to registry (Task 29 fills it).
- `backend/internal/platform/http/router.go` — register the new PATCH route.
- `frontend/src/lib/types.ts` — add `outputDocs: string[]` to `Session` + `CreateSessionRequest` + `FinalizeRequest`; new `FinalizeResponse.documents: Record<string, {filename, content, lineCount}>`.
- `frontend/src/lib/services/api.ts` — extend `createSession`, add `updateOutputDocs`, extend `finalizeSession` to accept optional override.
- `frontend/src/routes/+page.svelte` — add a checkbox group "Documents to generate" in the create-session form (default checked: architecture + roadmap).
- `frontend/src/routes/session/[id]/finalize/+page.svelte` — add the same checkbox group above the "Generate Artifacts" button; selection seeds from session, override sent in finalize body.

**Validation**

- `go test ./backend/internal/modules/session/...` — covers validation rules and PATCH endpoint.
- `migrate up` then `migrate down` on a clean DB — schema reversible.
- Manual: create a session with `["plan"]`, change to `["plan","readme"]` via PATCH, finalize and confirm response keys match.

**Prompt context needed:** §8.19 (output doc selection schema, new in this task), §22.1 of `A2A-agent-Brainstorm.md`, Task 10 (markdown wire-up), Task 19 (existing finalize response shape).

---

### Task 29 — Backend: Long-form Generators for All Output Documents <!-- ✅ Task 29 completed -->

**Goal:** Bring **every** output document — `architecture.md`, `roadmap.md`, `PLAN.md`, `README.md` — to the same long-form quality bar (≥ 1000 lines per document, individually) via a single generator registry. Refactor the two existing generators (`GenerateArchitecture`, `GenerateRoadmap`) to use the shared template helpers + line-count enforcer, and add the two new generators (`GeneratePlan`, `GenerateReadme`). See blueprint §22.2.

**Files to create / modify**

- `backend/internal/modules/markdown/templates.go` — **new**. Shared helpers used by all four generators:
  - `renderASCIIComponents(state) string` — ASCII data-flow diagram from `state.architecture.components`.
  - `renderTable(headers []string, rows [][]string) string` — markdown table renderer with column alignment.
  - `renderDirectoryTree(layout) string` — directory tree from `state.architecture.directory_layout`.
  - `renderEnvVarList(config) string`, `renderTechStack(...)`, `renderDecisionsTable(...)`, `renderRisksTable(...)`, `renderExecutionPlanList(...)`.
  - `enforceMinLines(body, state, padFn) string` — deterministic padding loop (see §8.20).
  - `padArchitecture / padRoadmap / padPlan / padReadme` — per-generator padders that emit per-component deep-dive sub-sections (data flow, failure modes, observability, deployment notes) and per-execution-plan-item elaboration (assumptions, risks, mitigations, validation matrix). Last-resort padder: full canonical state JSON dump in a fenced block.
- `backend/internal/modules/markdown/generator_architecture.go` — **move + refactor** the existing `GenerateArchitecture` here. Replace ad-hoc string building with shared helpers; wrap final body in `enforceMinLines(body, state, padArchitecture)` so output is ≥ 1000 lines. Determinism preserved (no maps iterated without sorting, no `time.Now()`).
- `backend/internal/modules/markdown/generator_roadmap.go` — **move + refactor** the existing `GenerateRoadmap` the same way; wrap in `enforceMinLines(body, state, padRoadmap)`.
- `backend/internal/modules/markdown/generator_plan.go` — **new**. `GeneratePlan(state CanonicalState) (string, error)` per §8.20 section skeleton; wrap in `enforceMinLines(body, state, padPlan)`.
- `backend/internal/modules/markdown/generator_readme.go` — **new**. `GenerateReadme(state CanonicalState) (string, error)` per §8.20 section skeleton; wrap in `enforceMinLines(body, state, padReadme)`.
- `backend/internal/modules/markdown/generator.go` — **modify**:
  - Remove the inline `GenerateArchitecture` and `GenerateRoadmap` function bodies (they now live in their own files).
  - Define the `Generators` registry: `var Generators = map[string]func(state.CanonicalState) (string, error){ "architecture": GenerateArchitecture, "roadmap": GenerateRoadmap, "plan": GeneratePlan, "readme": GenerateReadme }`.
  - Replace `GenerateContent(s)` with `GenerateAll(state, keys []string) (map[string]GeneratedDocument, error)` (Task 28 already changed the call site). Unknown key → return error (caller returns 400).
  - Each entry in the response map carries `{filename, content, line_count}` where `filename` is fixed per key: `architecture` → `architecture.md`, `roadmap` → `roadmap.md`, `plan` → `PLAN.md`, `readme` → `README.md`.
- `backend/internal/modules/markdown/generator_architecture_test.go` — **modify** (or rename from existing test): assert output ≥ 1000 lines AND byte-identical determinism (same input twice → same bytes).
- `backend/internal/modules/markdown/generator_roadmap_test.go` — same as above.
- `backend/internal/modules/markdown/generator_plan_test.go` — **new**: 1000-line minimum + determinism + structural assertion (must contain `## 1. Goal`, `## 5. Implementation Tasks`, `## 8. Deep Knowledge Reference`).
- `backend/internal/modules/markdown/generator_readme_test.go` — **new**: 1000-line minimum + determinism + structural assertion (must contain `## Overview`, `## Quick Start`, `## Configuration`, `## License`).
- `backend/internal/modules/markdown/generator_registry_test.go` — **new**: covers `GenerateAll` with all four keys, unknown-key error, ordering preserved, every returned `GeneratedDocument.LineCount >= 1000`.

**Validation**

- `go test ./backend/internal/modules/markdown/... -run Generate` — all four generators pass line-count + determinism + structural tests.
- `go test ./backend/internal/modules/markdown/... -run Registry` — registry test passes.
- `go vet ./backend/internal/modules/markdown/...` — clean.
- `go build ./backend/...` — zero errors (Task 28's `GenerateContent` → `GenerateAll` migration still compiles end-to-end).
- Manual: feed the `matchpoint` seed state through each of the four keys, eyeball each markdown for structural correctness and that no two consecutive runs produce diff output (`diff <(run1) <(run2)` empty).

**Prompt context needed:** §8.20 (generator template skeletons for all four documents + padding rule, expanded in this task), §22.2 of `A2A-agent-Brainstorm.md`, §8.1 (canonical state shape), Task 28 (registry slot + new `GenerateAll` signature), existing `generator.go` (current `GenerateArchitecture` / `GenerateRoadmap` bodies to migrate).

---

### Task 30 — Per-Agent Run Button (Preview / Apply) <!-- ✅ Task 30 completed -->

**Goal:** Add per-agent preview and apply endpoints + frontend buttons on each `PipelineStage`. Coexists with the existing full-iteration run. See blueprint §22.3.

**Files to create / modify**

- `backend/internal/modules/iteration/preview.go` — new `PreviewStore` (in-memory, keyed by `sessionID → agentID → PreviewResult`); methods `Set`, `Get`, `Delete`, `Clear`.
- `backend/internal/modules/iteration/engine.go` — add `RunSingleAgent(ctx, sessionID, agentID) (CanonicalState, error)` that loads state, dispatches one agent via the existing A2A client, returns the agent output without persisting. Reuses prompt assembly path.
- `backend/internal/modules/iteration/service.go` — `Preview`, `Apply`, `DiscardPreview`. `Apply` merges using the standard merge strategy, increments iteration counter, persists, clears the preview slot. All three reject with 409 when a full iteration is in flight (check engine's per-session mutex).
- `backend/internal/modules/iteration/handler.go` — new handlers for the three endpoints.
- `backend/internal/platform/http/router.go` — register `POST /sessions/{id}/agents/{agent_id}/preview`, `POST .../apply`, `DELETE .../preview`.
- `frontend/src/lib/services/api.ts` — `previewAgent`, `applyAgentPreview`, `discardAgentPreview`.
- `frontend/src/lib/types.ts` — add `PreviewResult` type + `pipelineStage.preview?: PreviewResult` field.
- `frontend/src/lib/components/PipelineStage.svelte` — render two new buttons: **Run This Agent** (calls preview) and **Apply** (disabled until preview exists); show a yellow `chip-warn` banner "Preview — not committed" above the preview log block.
- `frontend/src/routes/session/[id]/+page.svelte` — wire button handlers; refresh canonical state on apply.

**Validation**

- `go test ./backend/internal/modules/iteration/... -run Preview` — covers preview store, 409 conflict during in-flight iteration, apply increments counter.
- `pnpm check` — clean.
- Manual: preview an agent, observe state unchanged; apply, observe state advanced; second iteration still works.

**Prompt context needed:** §8.21 (preview/apply API contract, new in this task), §22.3 of `A2A-agent-Brainstorm.md`, Task 9 (iteration engine), Task 18 (PipelineStage component), Task 19 (session refresh path).

---

### Task 31 — SSE Real-time Agent Progress <!-- ✅ Task 31 completed -->

**Goal:** Replace the simulated progress bar with a live SSE stream of agent lifecycle events. See blueprint §22.4.

**Files to create / modify**

- `backend/internal/platform/sse/broadcaster.go` — `Broadcaster` with `Subscribe(sessionID) (<-chan Event, unsubscribe)`, `Publish(sessionID, evt)`, per-session ring buffer (cap 100) for `Last-Event-ID` replay. Bounded subscriber count.
- `backend/internal/modules/iteration/events.go` — `Event` struct (`ID, Type, Data`), event-type constants (`iteration.start`, `agent.started`, `agent.complete`, `agent.error`, `iteration.complete`, `session.finalized`); `EventEmitter` interface.
- `backend/internal/modules/iteration/engine.go` — accept an `EventEmitter` via constructor; emit `iteration.start` before the loop, `agent.started` before each dispatch, `agent.complete` after each merge (with confidence delta), `iteration.complete` after each pass, `agent.error` on dispatch failure. Same for `RunSingleAgent` (`agent.started` + `agent.complete`/`error`).
- `backend/internal/modules/iteration/handler.go` — new `GET /sessions/{id}/events` SSE handler: validate session exists, set `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`; honor `Last-Event-ID` header by replaying ring buffer; flush after each event; close on client disconnect.
- `backend/internal/modules/session/service.go` — emit `session.finalized` after successful finalize.
- `backend/internal/platform/http/router.go` — register the SSE route.
- `backend/cmd/server/main.go` — instantiate `Broadcaster`, inject into iteration + session services.
- `frontend/src/lib/services/sse.ts` — small wrapper around `EventSource` with auto-reconnect + `Last-Event-ID` handling.
- `frontend/src/routes/session/[id]/+page.svelte` — open the stream on mount, dispatch events to `sessionStore`; update each `PipelineStage` status (`waiting → running → done`) live; remove simulated progress code.
- `frontend/src/lib/stores/sessionStore.ts` — add `applyEvent(evt)` reducer.

**Validation**

- `go test ./backend/internal/platform/sse/... ./backend/internal/modules/iteration/...` — covers broadcaster fan-out, ring-buffer replay, engine event emission ordering.
- Manual: run an iteration with two agents, confirm 1× `iteration.start`, 2× `agent.started` + `agent.complete`, 1× `iteration.complete` in DevTools network tab; reconnect mid-iteration with `Last-Event-ID` and confirm replay.
- `pnpm check` and `pnpm build` — clean.

**Prompt context needed:** §8.22 (SSE event schema, new in this task), §22.4 of `A2A-agent-Brainstorm.md`, Task 9 (iteration engine hook points), Task 18 (PipelineStage status field), Task 30 (preview also emits events).

---

### Task 32 — Generated Document Quality Overhaul <!-- ✅ Task 32 completed -->

**Goal:** Replace the current broken output generators (idea text used as H1, idea body repeated 3–5× per file, empty roadmap sections, repetitive padding from `enforceMinLines`, single hardcoded filename per kind) with a deterministic quality pipeline: short title extraction, one-line description extraction, sparse-state finalize block (HTTP 422), removal of all line-count padding, per-session `{slug}_{kind}.md` filename pattern, an enriched canonical state schema, and agent role prompts updated to populate the new fields so the four output documents (`architecture`, `roadmap`, `plan`, `readme`) finally reach the depth and structure of the reference docs (`MD-AME-ARCHITECTURE.md`, `MULTI_MARKET_ROADMAP_MERGED.md`, `IMPLEMENTATION_ROADMAP.md`, `README.md`). See §8.23 for the full standard.

**Files to create / modify:**

- `backend/internal/modules/markdown/templates.go` — **modify**:
  - **Delete** `enforceMinLines`, `padArchitecture`, `padRoadmap`, `padPlan`, `padReadme` and every related constant. Line count is no longer a quality signal.
  - **Add** `shortTitle(s state.CanonicalState) string` — picks the first non-empty of: `s.Idea["name"]`, the first sentence of `s.Idea["text"]` truncated to 60 chars at a word boundary, or `"Untitled Brainstorm"`. Strips Markdown, newlines, and multi-space runs. See §8.23.
  - **Add** `oneLineDescription(s state.CanonicalState) string` — single sentence ≤ 200 chars; never emits the full idea body verbatim.
  - **Add** `slugify(title string) string` — lowercase ASCII, spaces → `-`, drop punctuation, collapse repeats, trim leading/trailing `-`, max 50 chars. Empty result → first 8 chars of session ID (passed via new optional arg in a follow-up if needed; for this task, use `"untitled"` fallback).
- `backend/internal/modules/markdown/generator.go` — **modify**:
  - Replace the static `filenameForKey` map with `buildFilename(title string, key string) string` returning `slugify(title) + "_" + suffixForKey[key]` where `suffixForKey = {architecture: "architecture.md", roadmap: "roadmap.md", plan: "plan.md", readme: "readme.md"}` (all lowercase — beginner-friendly, no Windows-case-sensitivity surprises).
  - `GenerateAll` computes the slug once from `shortTitle(s)` and passes it to every per-key generator via context, so all four files for a session share the same prefix.
  - `WriteArtifacts` uses the new filenames directly; existing atomic write logic (`.tmp` → rename) unchanged.
- `backend/internal/modules/markdown/generator_architecture.go` — **modify**:
  - Header line becomes `# {shortTitle} — Architecture` (never the full idea body).
  - One-line summary blockquote uses `oneLineDescription`. Drop the existing duplicated `writeMap(&b, s.Idea)` block in §1.
  - § 2 (System Components) iterates `s.Architecture["layers"]` (new structured field — see §8.23) and renders one sub-section per layer with a Responsibility / Technologies / Dependencies table. Fall back to the old map-iteration only when `layers` is absent.
  - § 4 (Data Flow) renders a fenced Mermaid block from `s.Architecture["data_flows"]` when present.
  - Drop the trailing `enforceMinLines` call and its import.
- `backend/internal/modules/markdown/generator_roadmap.go` — **modify**:
  - Header becomes `# {shortTitle} — Roadmap`.
  - § 2 (Milestones) and § 3 (Phase Breakdown) iterate `s.ExecutionPlan` entries; each entry renders the canonical 7-field block (Objective / Blocking Dependencies / Scope / Deliverables / Function Contracts / Failure Handling / Exit Criteria) when those structured fields are present on the entry; otherwise renders a minimal `{name} — {description}` block.
  - Drop the trailing `enforceMinLines` call.
- `backend/internal/modules/markdown/generator_plan.go` — **modify**: same header + dedup pattern; iterate `s.ExecutionPlan` for §5 tasks; drop padding.
- `backend/internal/modules/markdown/generator_readme.go` — **modify**: same header + dedup pattern; Overview prints `oneLineDescription` once (not three times); Roadmap section summarises the first N phases as bullet points; drop padding.
- `backend/internal/modules/state/model.go` — **modify**: extend `CanonicalState` with optional sub-fields. All additions are JSON-omitempty and backward-compatible with sessions created before this task. See §8.23:
  - `Architecture` already `map[string]any` — document the standard keys: `layers []Layer`, `data_flows []DataFlow`, `tech_stack map[string]string`, `directory_layout []string`.
  - `ExecutionPlan` entries now expected (not enforced) to carry: `phase string`, `objective string`, `deliverables []string`, `exit_criteria []string`, `blocking_dependencies []string`.
  - `Risks` entries: `likelihood string`, `impact string`, `mitigation string`.
  - `Assumptions` entries: `rationale string`, `validation_method string`.
  - `Metrics`: `test_coverage_target float64`, `latency_budget_ms int`.
- `backend/internal/modules/session/service.go` — **modify**: `Finalize(ctx, sessionID)` calls new private helper `isStateReadyForFinalize(s state.CanonicalState) (ready bool, reason string)`. When `ready == false`, handler returns HTTP 422 `{"error":"state_not_ready","reason":"..."}` instead of generating empty documents. Readiness rule: `len(s.Idea) > 0 AND len(s.Architecture) > 0 AND len(s.ExecutionPlan) > 0 AND s.Metrics.Confidence >= 0.5`. See §8.23.
- `backend/internal/modules/session/handler.go` — **modify**: surface 422 as a typed error; existing 200/4xx paths unchanged.
- `backend/internal/modules/markdown/generator_*_test.go` — **modify all four**:
  - Replace any `assert lineCount >= 1000` assertions with `assert title is short (< 80 chars)`, `assert idea body appears at most once`, `assert filename matches slug pattern`, `assert §2 sub-sections present when state.architecture.layers is populated`.
- `backend/internal/modules/markdown/generator_registry_test.go` — **modify**: assert filenames follow the `{slug}_{kind}.md` pattern.
- `agent/internal/executor/executor.go` — **modify role-prompt assembly only**: when building the system prompt, include the §8.23 enriched-state schema fragment as a "Required Output Structure" section. This instructs every agent (regardless of role) to populate `architecture.layers`, `execution_plan[].objective`, etc. Existing agent role prompts (Architect / Engineer / Reviewer system prompts injected by backend) updated in the same vein — values stored in `agents.system_prompt` column via a one-line `UPDATE agents SET system_prompt = system_prompt || '\n\n' || $1 WHERE name IN (...)` migration entry recorded in this task's notes, not a new SQL migration file.

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- `go test ./backend/internal/modules/markdown/...`: all new assertions pass; padding is gone; titles are short; filenames slug-based
- `go test ./backend/internal/modules/session/...`: new `isStateReadyForFinalize` test covers all four blocking conditions (empty idea, empty architecture, empty execution_plan, confidence < 0.5)
- Manual smoke: finalize the Match Point seed session → output files are named `match-point_architecture.md`, `match-point_roadmap.md`, `match-point_plan.md`, `match-point_readme.md`; each file's H1 is the short title; idea text appears at most once per file; no `_No execution plan steps recorded yet._` stubs (HTTP 422 returned instead when state is sparse)
- Manual smoke: re-run the brainstorm with updated agent prompts → `state.architecture.layers` populated with ≥ 3 entries; output `architecture.md` § 2 contains one sub-section per layer with a real Responsibility / Tech / Dependencies table — depth comparable to `MD-AME-ARCHITECTURE.md` § 2

**Prompt context needed:** §8.23 (Generated Document Quality Standard — title/description extraction, slug rules, sparse-state readiness check, enriched CanonicalState schema, agent prompt fragment), §8.1 (canonical state shape), §8.20 (section skeletons from Task 29 — now superseded by §8.23 for depth source), Task 28 (selectable output docs — `GenerateAll` registry is the surface this task changes), Task 29 (long-form generators — `enforceMinLines` is deleted by this task)

---

### Task 33 — AI-Driven Hybrid Document Generator + Skill Bundle <!-- ✅ Task 33 completed -->

**Goal:** Layer an AI-driven generation pass on top of the deterministic Task-32 generators so finalize output reaches the depth and consistency of the reference docs (`docs/A2A-agent-Brainstorm.md`, `docs/architecture.md`, `docs/implementation_roadmap.md`, `README.md`). Add a `markdown/aigen/` sub-package that (a) loads a curated **SkillBundle** of prompt fragments — `modular-monolith`, `vertical-slice`, `api-design`, `roadmap-spec`, `plan-management` — drawn from `.github/skills/`, (b) drives a per-document multi-section LLM pass that rewrites each section using the deterministic output as a scaffold, (c) runs a section-level **rubric validator** with bounded auto-repair, and (d) is selected at runtime via a `FINALIZE_MODE` config switch (`deterministic` | `hybrid` | `ai`, default `hybrid`). Deterministic generators remain the safe fallback when AI calls fail, exceed budget, or fail rubric after repair attempts. See §8.27 for the full contract.

**Files to create / modify:**

- `backend/internal/modules/markdown/aigen/skills.go` — **new**:
  - `Skill struct { Name string; Path string; Prompt string }` — `Prompt` is the verbatim contents of the linked `.github/skills/<name>/SKILL.md` minus YAML frontmatter (stripped at load time).
  - `SkillBundle struct { Skills []Skill }` with method `Compose() string` that concatenates each skill prompt under a `## Skill: <name>` heading, separated by blank lines.
  - `LoadDefaultBundle(fs fs.FS) (SkillBundle, error)` — loads the five canonical skills listed above from an injected `fs.FS` (production: `os.DirFS(".")` rooted at repo). Missing skill files return error — no silent fallback. See §8.27 for the canonical skill list and load order.
  - All file paths are config-driven via `GetSkillBundlePaths() []string` (see config change below). No path constants live in this file.
- `backend/internal/modules/markdown/aigen/rubric.go` — **new**:
  - `Rubric struct { DocKey string; Sections []SectionRule }`.
  - `SectionRule struct { Heading string; MinChars int; RequiredKeywords []string; ForbidPlaceholders []string }`. Defaults: `MinChars=400`, `ForbidPlaceholders=["TBD","TODO","Lorem ipsum","placeholder"]`.
  - `RubricFor(docKey string) Rubric` — returns the canonical rubric per doc key (`architecture`, `roadmap`, `plan`, `readme`), all values config-driven via §8.27 defaults.
  - `Validate(content string, r Rubric) []RubricFinding` — returns one `RubricFinding{Heading, Reason}` per failing section. Empty slice = pass.
- `backend/internal/modules/markdown/aigen/generator.go` — **new**:
  - `Generator struct { llm llm.LLMProvider; bundle SkillBundle; maxRepairs int; logger *slog.Logger }`.
  - `New(llm llm.LLMProvider, bundle SkillBundle, logger *slog.Logger) *Generator` — `maxRepairs` read from config (§8.27); never hardcoded.
  - `GenerateAll(ctx context.Context, s state.CanonicalState, keys []string, scaffolds map[string]shared.GeneratedDocument) (map[string]shared.GeneratedDocument, error)`:
    1. For each key in `keys`, build a `LLMRequest` whose `SystemPrompt = bundle.Compose() + "\n\n" + §8.27 doc-style contract for that key`.
    2. `UserMessage` = `"## Scaffold (deterministic draft)\n\n" + scaffolds[key].Content + "\n\n## CanonicalState (JSON)\n\n" + jsonDump(s)`.
    3. Call `llm.Generate(ctx, req)` once → `draft`.
    4. Run `Validate(draft, RubricFor(key))`; if non-empty, run **auto-repair** up to `maxRepairs` times: new `UserMessage` includes the previous draft + a bullet list of findings; ask the model to emit the full revised document only. After `maxRepairs`, return the deterministic scaffold for that key (do NOT return a half-broken AI draft).
    5. Build `shared.GeneratedDocument{Filename: scaffolds[key].Filename, Content: draft, LineCount: countLines(draft)}` and copy into the result map. Filename always comes from the scaffold so Task 32 slug rules are preserved.
  - Deterministic on retry: temperature passed in `LLMRequest` is read from config (default `0.2`), never hardcoded. The skill bundle is composed once per call and reused across keys — no per-key bundle mutation.
- `backend/internal/modules/markdown/generator.go` — **modify**:
  - Add `FinalizeMode` enum: `ModeDeterministic`, `ModeHybrid`, `ModeAI`.
  - Add `Orchestrator struct { det *Writer; ai *aigen.Generator; mode FinalizeMode; logger *slog.Logger }`.
  - `Orchestrator.GenerateAll(ctx, s, keys) (map[string]shared.GeneratedDocument, error)`:
    - `ModeDeterministic` → returns `det.GenerateAll(s, keys)` exactly as today.
    - `ModeHybrid` → calls `det.GenerateAll` first; passes scaffolds + state into `ai.GenerateAll`; on AI error logs `slog.Warn` with reason and returns the deterministic scaffolds unchanged (zero-regression fallback).
    - `ModeAI` → same as hybrid but on AI error returns the wrapped error (no fallback). Reserved for testing / explicit opt-in.
  - Existing `Writer.WriteArtifacts` unchanged — Task 32 atomic temp+rename logic preserved.
- `backend/internal/modules/session/service.go` — **modify**:
  - `FinalizeSession` now calls the orchestrator obtained from `NewService` instead of `Writer.GenerateAll` directly. No new validation logic — readiness gate from Task 32 still runs first.
- `backend/internal/platform/config/config.go` — **modify**:
  - Add `GetFinalizeMode() FinalizeMode` — reads `FINALIZE_MODE` env var; accepts `deterministic` / `hybrid` / `ai` (case-insensitive); default `hybrid`; any other value → error at startup.
  - Add `GetSkillBundlePaths() []string` — reads `SKILL_BUNDLE_PATHS` env var (comma-separated repo-relative paths); default `.github/skills/modularity/SKILL.md,.github/skills/vertical-slice/SKILL.md,.github/skills/api-design/SKILL.md,.github/skills/roadmap-spec/SKILL.md,.github/skills/plan-management/SKILL.md`.
  - Add `GetAIDocMaxRepairs() int` — reads `AIGEN_MAX_REPAIRS`; default `2`; clamp `[0, 5]`.
  - Add `GetAIDocTemperature() float64` — reads `AIGEN_TEMPERATURE`; default `0.2`; clamp `[0.0, 1.0]`.
- `backend/cmd/server/main.go` — **modify**:
  - During wiring, construct `Orchestrator` per `GetFinalizeMode()`; when mode is `hybrid` or `ai`, load `SkillBundle` via `aigen.LoadDefaultBundle(os.DirFS(repoRoot))` using `GetSkillBundlePaths()`. Pass the same `LLMProvider` already constructed for backend↔agent dispatch (see §8.27 for why backend reuses the existing provider instead of dialing the agent binary).
- `backend/internal/modules/markdown/aigen/aigen_test.go` — **new**:
  - `TestSkillBundle_ComposeOrder` — load bundle from in-memory `fstest.MapFS`; assert composed prompt contains skill headings in declared order with no duplicates.
  - `TestRubric_FailsOnShortSection` — section content with < `MinChars` flagged.
  - `TestRubric_FailsOnForbiddenPlaceholder` — content containing `TBD` flagged.
  - `TestGenerator_Hybrid_FallsBackOnLLMError` — stub `LLMProvider` returns error; `Orchestrator.GenerateAll` returns deterministic scaffolds unchanged and emits a warning log.
  - `TestGenerator_AutoRepairThenAccept` — stub returns short draft on round 1, full draft on round 2 → final document is the round-2 draft; `maxRepairs` decremented exactly once.
  - `TestGenerator_RepairExhausted_UsesScaffold` — stub always returns short draft; after `maxRepairs` attempts, returned document equals deterministic scaffold byte-for-byte.
- `docs/STARTUP_GUIDE.md` — **modify**: append a one-paragraph note documenting `FINALIZE_MODE`, `SKILL_BUNDLE_PATHS`, `AIGEN_MAX_REPAIRS`, `AIGEN_TEMPERATURE` and their defaults.

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- `go test ./backend/internal/modules/markdown/aigen/...`: all five new tests pass without a real LLM (stub provider)
- `go test ./backend/internal/modules/markdown/...`: Task-32 deterministic tests still pass unchanged
- `go test ./backend/internal/modules/session/...`: finalize tests still pass; orchestrator wired in test setup with `ModeDeterministic` so behaviour is unchanged for existing tests
- Manual smoke: `FINALIZE_MODE=hybrid make start`; finalize the Match Point seed session → output `match-point_architecture.md` § 2 contains layered component descriptions with concrete responsibilities, no `TBD`, ≥ 400 chars per section; `match-point_roadmap.md` § 3 contains one phase block per `execution_plan` entry with all seven §8.23 fields filled in; total file depth comparable to `docs/architecture.md` and `docs/implementation_roadmap.md`
- Manual smoke: `FINALIZE_MODE=deterministic` → behaviour identical to post-Task-32; same byte-for-byte output
- Manual smoke: stub broken LLM (`COPILOT_API_KEY=invalid FINALIZE_MODE=hybrid`) → finalize still succeeds with deterministic scaffolds; backend logs a `slog.Warn` with the LLM error

**Prompt context needed:** §8.27 (AI-Driven Document Generator contract — SkillBundle composition, rubric defaults, auto-repair algorithm, FinalizeMode semantics, fallback policy), §8.23 (deterministic scaffolds — this task wraps them, never replaces them), §8.2 (`LLMProvider` interface — the only LLM call surface), §8.12 (LLM credential security), AGENTS.md skill registry (canonical paths for the five bundled skills)

---

### Task 34 — Guided Onboarding v2, Plan Consolidation, Doc Quality v2, and Artifact Persistence <!-- ✅ Task 34 completed -->

**Goal:** Ship the `frontend/mockups/v2.html` onboarding experience and close four product gaps identified in the Ngoding Pake AI competitive review: (1) tech-constraint picker on the new-session flow, (2) user-first discovery questions with optional LLM-generated chip hints, (3) deprecate `roadmap.md` in favour of a single consolidated `plan.md` output, (4) raise the dual-audience quality bar for all generated documents (human-readable **and** AI-coding-agent ready), and (5) persist finalized document bodies in PostgreSQL so history and revisit views load instantly without re-running the generator. See §8.29 for the full contract.

**Discovery flow decision (locked):** **User answers first.** The LLM is used only for optional chip-hint generation after the user has entered an idea — never to pre-fill answers or skip the clarify step. Session creation compiles user inputs into canonical state before the first iteration.

**Sub-goals:**

| #    | Area                 | Outcome                                                                                                                                       |
| ---- | -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| 34.1 | Tech constraints UI  | Home page block from v2 mockup: "Let agents decide" toggle (default ON) + three tiers (must / comfortable / avoid) with searchable hint chips |
| 34.2 | Discovery clarify UI | Second step (`/start/clarify` or in-page view): 4 optional questions, skip-all, progress counter; chip hints loaded asynchronously            |
| 34.3 | Roadmap → plan       | Remove `roadmap` output key; merge milestone + phase content into `plan.md` only                                                              |
| 34.4 | Doc quality v2       | Expand rubrics + generator sections; every doc passes human scan **and** agent-ingestion checks (see §8.29.5)                                 |
| 34.5 | Artifact persistence | `session_artifacts` table; finalize UPSERT; history/finalize views read from DB; no "Back to Session" on approved sessions                    |

**Files to create / modify:**

- `migrations/007_session_discovery.sql` — **new** (append-only; repo `006` is `generic_seed_data`):
  - `sessions.discovery_answers JSONB NOT NULL DEFAULT '{}'`
  - `sessions.tech_constraints JSONB NOT NULL DEFAULT '{"agents_decide":true}'` — shape per §8.29.3
  - `sessions.enriched_idea TEXT NOT NULL DEFAULT ''` — compiled markdown block sent to agents
- `migrations/008_session_artifacts.sql` — **new**:
  - `session_artifacts` table: `id UUID PK`, `session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE`, `doc_key TEXT NOT NULL`, `filename TEXT NOT NULL`, `content TEXT NOT NULL`, `line_count INT NOT NULL`, `source TEXT NOT NULL` (`deterministic` \| `hybrid` \| `ai`), `generated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
  - `UNIQUE (session_id, doc_key)` — idempotent finalize re-runs update in place
  - Index: `CREATE INDEX idx_session_artifacts_session ON session_artifacts (session_id)`
- `migrations/009_migrate_roadmap_to_plan.sql` — **new**:
  - Data migration replacing `roadmap` with `plan` in `sessions.output_docs` arrays (dedupe `plan` when both were selected)
- `backend/internal/modules/session/model.go` — **modify**:
  - Add `DiscoveryAnswers`, `TechConstraints`, `EnrichedIdea` to `Session`
  - Extend `CreateSessionRequest` with optional `discovery_answers`, `tech_constraints`
  - Add `SessionArtifact` struct + list response types
  - Remove `"roadmap"` from `AllowedOutputDocs`; update `DefaultOutputDocs` to `["architecture", "plan"]`
- `backend/internal/shared/tech_constraints.go` — **new** — `TechConstraints` + `ToAssumptions()` per §8.29.3
- `backend/internal/modules/session/discovery_compiler.go` — **new**:
  - `CompileEnrichedIdea(...)` — deterministic markdown assembly (no LLM)
  - `SeedInitialState(...)` — pre-populates canonical state before iteration 1
- `backend/internal/modules/session/discovery_hints.go` — **new**:
  - `POST /sessions/discovery-hints` backing service — LLM generates chip labels for Q2–Q4 only; LRU cache keyed by `sha256(idea)`; static v2 defaults on LLM failure
- `backend/internal/modules/session/handler.go` — **modify** — discovery-hints endpoint, artifacts list endpoint, enriched session create
- `backend/internal/modules/session/repository.go` — **modify** — new columns + `UpsertArtifacts` / `ListArtifacts`
- `backend/internal/modules/session/service.go` — **modify** — finalize persists artifacts; `GetArtifacts` for revisit path
- `backend/internal/modules/markdown/generator_plan.go` — **modify** — absorb roadmap milestones + phase breakdown sections
- `backend/internal/modules/markdown/generator_roadmap.go` — **modify** — remove from registry; export `ErrOutputKeyDeprecated` if called directly
- `backend/internal/modules/markdown/generator.go` — **modify** — drop `roadmap` from `Generators` map
- `backend/internal/modules/markdown/aigen/rubric.go` — **modify** — remove `roadmap` rubric; expand `architecture` + `plan` per §8.29.5
- `backend/internal/modules/markdown/aigen/generator.go` — **modify** — dual-audience style contracts
- `backend/internal/platform/config/config.go` — **modify** — `GetDiscoveryHintsCacheSize()`, `GetDiscoveryHintsTemperature()`
- `frontend/src/lib/components/TechConstraints.svelte` — **new**
- `frontend/src/lib/components/DiscoveryClarify.svelte` — **new**
- `frontend/src/lib/components/DiscoveryChipHints.ts` — **new** — static defaults + `mergeHints()`
- `frontend/src/routes/+page.svelte` — **modify** — two-step home → clarify flow; remove roadmap checkbox
- `frontend/src/routes/session/[id]/finalize/+page.svelte` — **modify** — load artifacts from DB; hide "Back to Session" when approved
- `frontend/src/routes/history/+page.svelte` — **modify** — `arch.md` + `plan.md` chips; View → finalize page
- `frontend/src/lib/services/api.ts`, `frontend/src/lib/types.ts` — **modify**
- Tests: `discovery_compiler_test.go`, `discovery_hints_test.go`, `artifacts_test.go`, `generator_plan_test.go`, `api.test.ts`

**Validation:**

- `make migrate` — migrations `007`–`009` apply cleanly
- `cd backend && go build ./...` && `go vet ./...` && `go test ./backend/internal/modules/session/...` && `go test ./backend/internal/modules/markdown/...`
- `cd frontend && pnpm check && pnpm build && pnpm test`
- Manual smoke — onboarding: tech constraints + clarify answers appear in `enriched_idea` and seeded `assumptions[]`
- Manual smoke — hints: static chips immediate; domain chips arrive async; no pre-filled answers
- Manual smoke — plan only: finalize produces `*_plan.md` with milestones + phases; no `*_roadmap.md`
- Manual smoke — persistence: approved session finalize page loads from DB instantly; no "Back to Session"

**Prompt context needed:** §8.29, §8.1, §8.19, §8.23, §8.27, `frontend/mockups/v2.html`, Tasks 17, 19, 24, 33

---

### Task 35 — Architecture.md: State Enricher + High-Quality Section Generator

**Goal:** Produce a publication-quality `architecture.md` with 16 ordered sections (plus document metadata block, Table of Contents, and an AI agents appendix) by implementing two complementary layers: (A) a full rewrite of `generator_architecture.go` that renders all 16 sections from the canonical state with graceful deterministic fallbacks for each section, and (B) a new `aigen/enricher.go` pre-pass that makes a single targeted LLM call to populate optional narrative state fields (`problem_statement`, `solution_summary`, `scope`, `extension_points`, `security`, `guarantees`, decision alternatives/tradeoffs, risk mitigations, tech stack rationale) before the deterministic render runs. When the enricher is disabled or the LLM call fails, each section renders from raw state or a generic stub. See §8.30 for the full section contract, enricher input/output schema, and rubric targets.

**Files to create / modify:**

- `backend/internal/modules/markdown/generator_architecture.go` — **rewrite** — 16-section generator (see §8.30.1 for the section sequence and heading constants):
  - `renderMetadataBlock(s)` — deterministic; outputs `> **Status:** Draft | Converged` based on `s.Meta.Iteration`, `> **Iteration:** N`, `> **Confidence:** X.XXXX`, `> **Generated:** {current date stub from build-time constant}`; no external calls
  - `renderTableOfContents()` — deterministic; 16 numbered TOC entries hardcoded to match section heading constants; always identical
  - `renderProblemStatement(s)` — reads `s.Idea["problem_statement"]` (string from enricher); falls back to first sentence of `s.Idea["context"]` ≤200 chars, then `s.Idea["text"]`; renders as paragraph under `## 1. Problem Statement`
  - `renderSolution(s)` — reads `s.Idea["solution_summary"]` (string from enricher); falls back to `s.Idea["summary"]` then first 300 chars of `s.Idea["text"]`; renders as paragraph + optional 3-bullet approach list if `s.Architecture["approach"]` is present
  - `renderScope(s)` — reads `s.Architecture["scope"]` map with keys `"in"` ([]string) and `"out"` ([]string) from enricher; renders two markdown tables under **In Scope** / **Out of Scope** headings; fallback: In-scope = first 3 goals from `s.Idea["goals"]`, Out-of-scope = "Further feature scope — defined in future iterations"
  - `renderArchitectureLayers(s)` — existing function retained; passes `layer.description` (string) through if present in the enricher overlay
  - `renderEnrichedTechStack(s)` — 4-column table: Technology | Role | Rationale | Version/Notes; reads `s.Architecture["tech_stack_rationale"]` map (key=tech name, value=rationale string) from enricher; falls back to 2-column table (technology, role) when rationale absent; adds a **Not Used (and Why)** sub-table if `s.Architecture["tech_stack_not_used"]` is present
  - `renderDataFlowsWithProse(s)` — renders numbered prose walkthrough paragraph before the Mermaid diagram; prose text from `s.Architecture["data_flow_prose"]` (enricher); falls back to generic "See diagram below" line; Mermaid diagram unchanged from existing `renderDataFlowsMermaid`
  - `renderDirectoryTree(s)` — existing function, kept
  - `renderEnrichedDecisionsTable(s)` — 6-column table: # | Decision | Choice | Alternatives | Tradeoff | Status; `Alternatives` and `Tradeoff` columns populated from `s.Architecture["decision_enrichments"]` array (see §8.30.2); existing decisions show "—" when enrichment absent
  - `renderExtensionPoints(s)` — reads `s.Architecture["extension_points"]` array from enricher (each entry: `name` string + `steps` []string); renders as `### N. Name` sub-sections with numbered steps; generic stub fallback: "Add a new LLM provider" and "Add a new output document format" each with 3 placeholder steps
  - `renderSecurityConsiderations(s)` — reads `s.Architecture["security"]` array from enricher (each entry: `surface`, `risk`, `mitigation` strings); renders as 3-column table; generic stub fallback: authentication surface + prompt injection surface
  - `renderQualityTargets(s)` — existing logic expanded; second table for non-functional targets (`TestCoverageTarget`, `LatencyBudgetMs`) always rendered with "—" for unset values; adds a "Coding Standards" row pointing to AGENTS.md module boundary rules
  - `renderSystemGuarantees(s)` — reads `s.Architecture["guarantees"]` array ([]string from enricher); renders as numbered list; fallback: 3 derived generic guarantees — "Deterministic output: same state input produces identical generated documents", "Isolated modules: no cross-module direct imports", "LLM abstracted: all model calls routed through LLMProvider interface"
  - `renderEnrichedRisksTable(s)` — 6-column table: # | Risk | Likelihood | Impact | Mitigation | Status; reads `s.Risks[].Likelihood`, `s.Risks[].Impact`, `s.Risks[].Mitigation` fields (see §8.23 enriched schema); fixes `strings.ToTitle` bug (replace with `strings.ToUpper(s[:1]) + strings.ToLower(s[1:])` with empty-string guard)
  - `renderDefinitionOfDone(s)` — reads `s.Architecture["exit_criteria"]` ([]string from enricher); renders as GitHub-flavour `- [ ]` checklist; always appends "All open questions resolved" and "Confidence ≥ 0.85" items unconditionally; fallback: 5 generic architectural DoD items
  - `renderForAIAgentsAppendix(s, "architecture")` — rewritten to derive forbidden patterns from `s.Architecture["forbidden_patterns"]` ([]string from enricher) OR hardcoded canonical list from package-level constants; canonical LLM interface name from `s.Architecture["llm_interface_name"]` (fallback: `"LLMProvider"`); layer names from actual `s.Architecture["layers"]` layer names rather than hardcoded strings

- `backend/internal/modules/markdown/templates.go` — **modify**:
  - Add `renderMetadataBlock`, `renderTableOfContents`, `renderProblemStatement`, `renderSolution`, `renderScope`, `renderEnrichedTechStack`, `renderDataFlowsWithProse`, `renderEnrichedDecisionsTable`, `renderExtensionPoints`, `renderSecurityConsiderations`, `renderSystemGuarantees`, `renderEnrichedRisksTable`, `renderDefinitionOfDone` helper functions
  - Fix `strings.ToTitle(r.Severity)` bug in `renderRisksTable` — replace with `strings.ToUpper(r.Severity[:1]) + strings.ToLower(r.Severity[1:])` (guard for empty string)
  - Clamp `oneLineDescription` output to 150 chars + ellipsis to prevent paragraph-length blockquotes
  - Pass `description` field through from enricher overlay in `renderArchitectureLayers`

- `backend/internal/modules/markdown/aigen/enricher.go` — **new**:
  - `ArchEnricher` struct with `llm LLMProvider` dependency
  - `NewArchEnricher(llm LLMProvider) *ArchEnricher`
  - `Enrich(ctx context.Context, s state.CanonicalState) (state.CanonicalState, error)`:
    - Returns original `s` unchanged on any error (LLM failure, invalid JSON, timeout) — non-fatal by design; logs at `slog.Warn`
    - Builds a compact prompt input JSON: `{idea_text, idea_context, idea_goals, tech_stack_keys, layer_names, decision_titles, risk_texts}` — strips raw long text to minimize token count (see §8.30.2 for exact input schema)
    - Single `llm.Generate` call with `GetArchEnricherTimeoutSec()` deadline and the enricher system prompt from §8.30.5
    - Parses response as `ArchEnrichmentOverlay` struct (see §8.30.2 for full field list)
    - Merges overlay into a shallow copy of `s`: only writes fields where the existing value is nil/empty (enricher fills gaps only; never overwrites human-curated state)
    - Returns enriched copy; does not mutate the input `s`
  - `ArchEnrichmentOverlay` struct (see §8.30.2): all fields optional/pointer; JSON-unmarshal from LLM response; validated with `validateOverlay()` (rejects empty strings, rejects slices with >20 items)

- `backend/internal/modules/markdown/aigen/enricher_test.go` — **new**:
  - Test: `Enrich` with mock LLM returning valid overlay JSON → all new fields set on returned state; original `s` untouched
  - Test: `Enrich` with empty canonical state → returns same zero-value state; no panic
  - Test: `Enrich` with LLM returning malformed JSON → returns original state unchanged; no error propagated to caller
  - Test: `Enrich` with LLM returning partial overlay (only `problem_statement`) → only that field set; all other new fields absent

- `backend/internal/modules/markdown/aigen/generator.go` — **modify**:
  - In `GenerateDocument` for key `"architecture"`: when `mode != "deterministic"` and `GetArchEnricherEnabled()` is true, call `enricher.Enrich(ctx, s)` before the deterministic scaffold and replace `s` with the enriched copy
  - Log result: `slog.Info("arch enricher", "enriched_fields", countNonEmpty(overlay))`
  - Handle nil enricher gracefully (enricher is optional; nil = enrichment skipped silently)
  - Constructor `New(...)` gains an optional `enricher *ArchEnricher` parameter (match existing constructor style)

- `backend/internal/modules/markdown/aigen/rubric.go` — **modify**:
  - Update `architecture` rubric entry: 16 required sections per §8.30.3; new `RequiredKeywords` for sections Problem Statement, Solution, Scope, Extension Points, Security Considerations, System Guarantees, Definition of Done; update `MinChars` targets per §8.30.3; total document `MinChars` raised to 1500

- `backend/internal/platform/config/config.go` — **modify**:
  - `GetArchEnricherEnabled() bool` — reads `ARCH_ENRICHER_ENABLED` env var; default `true`
  - `GetArchEnricherTimeoutSec() int` — reads `ARCH_ENRICHER_TIMEOUT_SEC` env var; default `30`; clamp `[5, 120]`

- `backend/internal/modules/markdown/generator_architecture_test.go` — **rewrite**:
  - Test with sparse state (empty maps): all 16 sections rendered; no panic; fallback content present for each section
  - Test with fully populated state (including enricher overlay fields): each section contains the enriched content
  - Test metadata block: contains `s.Meta.Iteration` number and `s.Metrics.Confidence`
  - Test TOC: always has exactly 16 entries matching section headings
  - Test problem statement: falls back through `context` → `text` when `problem_statement` absent
  - Test enriched risks table: `strings.ToTitle` bug absent; "Medium" renders as "Medium" not "MEDIUM"
  - Test enriched decisions table: 6-column output with Alternatives and Tradeoff columns
  - Test extension points: generic stub fallback present when `extension_points` absent
  - Test security considerations: generic stub fallback present when `security` absent
  - Test definition of done: always contains "All open questions resolved" and "Confidence ≥ 0.85" items

**Validation:**

- `cd backend && go build ./...`: zero build errors
- `cd backend && go vet ./...`: zero vet issues
- `cd backend && go test ./internal/modules/markdown/...`: all existing tests pass + all new enricher and generator tests pass; zero failures
- `cd backend && go test ./internal/platform/config/...`: new config getters covered
- Manual smoke: create a brainstorming session, run 2 iterations, call `POST /sessions/{id}/finalize`; download `architecture.md` → file contains exactly 16 numbered sections, a metadata blockquote, and a Table of Contents; Risks table has Mitigation column; no UPPERCASE severity from `strings.ToTitle` bug
- Manual smoke with `ARCH_ENRICHER_ENABLED=false`: same finalize → `architecture.md` contains all 16 sections with deterministic fallback content; no LLM call made for enrichment

**Prompt context needed:** §8.30 (Architecture.md Enrichment — section contract, `ArchEnrichmentOverlay` schema, enricher LLM prompt, rubric targets, fallback specification — new in this task), §8.23 (Generated Document Quality Standard — `shortTitle`, `oneLineDescription`, `slugify`, `buildFilename`), §8.27 (AI-Driven Hybrid Generator — `finalize_mode`, rubric validator, auto-repair loop — enricher wires into this flow), `backend/internal/modules/markdown/generator_architecture.go` (current 10-section implementation to be replaced), `backend/internal/modules/markdown/templates.go` (all helper functions to be extended), `backend/internal/modules/markdown/aigen/` (rubric + generator to be modified)

---

### Task 36 — PLAN.md: Generation Quality Overhaul — Enricher + Correct Task Format

**Goal:** Produce a publication-quality, AI-agent-executable `plan.md` by combining two complementary layers: (A) a full rewrite of `generator_plan.go` that renders §5 Implementation Tasks in the canonical `**Goal:**`/`**Layer(s) affected:**`/`**Files to create:**`/`**Coding standards:**`/`**Validation:**`/`**Invariant check:**`/`**Prompt context needed:**` format — eliminating the thin "see phase deliverables" stubs and the redundant §3 Phase Breakdown split — plus a deterministic ASCII dependency graph and a §8 Deep Knowledge section assembled from canonical state; and (B) a new `markdown/aigen/plan_enricher.go` single-call LLM post-pass that populates the per-task quality fields (`coding_standards`, `invariant_checks`, `layer_tags`, `deep_knowledge_entries`) before the deterministic render runs. Additionally, a new `agent/internal/executor/prompts/plan_format.go` constant (Approach C) is injected into agent system prompts when `plan` is in `output_docs`, directing agents to write tasks using the correct spec format — not "Phase N" — with explicit runnable validation commands and Go function signatures per file. See §8.31 for the full section contract, enricher schema, Approach C prompt spec, and quality assertion rules.

**Layer(s) affected:** `backend/internal/modules/markdown/`, `backend/internal/modules/markdown/aigen/`, `backend/internal/platform/config/`, `agent/internal/executor/prompts/`

**Files to create / modify:**

- `backend/internal/modules/markdown/generator_plan.go` — **rewrite** (see §8.31.1 for full section sequence and field constants):
  - Remove §3 Phase Breakdown and §5 Module Tasks thin-stub sections entirely
  - Introduce single `## 5. Implementation Tasks` section with one `### Task N — {Name}` block per plan phase in canonical state, using these per-task helper functions:
    - `renderTaskGoal(phase)` — reads `phase.Objective` (first sentence ≤ 120 chars); falls back to first sentence of `phase.Description`; renders as `**Goal:** {text}`
    - `renderLayersAffected(phase)` — derives module paths from `phase.Deliverables` file paths (extracts unique `backend/internal/modules/X/`, `agent/internal/Y/`, `frontend/src/...` prefixes); renders as `**Layer(s) affected:** \`path1\`, \`path2\``; falls back to `backend/internal/modules/` if derivation returns empty
    - `renderFilesToCreate(phase)` — converts `phase.Deliverables` list into markdown bullet tree; each deliverable bullet annotated with its `FunctionContracts` entries as nested sub-bullets; includes `phase.FailureHandling` as a sub-bullet under the primary service file with label `**Failure handling:**`
    - `renderCodingStandards(phase)` — renders only for `phase.Complexity` = `"Medium"` or `"High"`; reads `phase.CodingStandards` slice from enricher overlay (§8.31.2); fallback: 3 generic standards (SRP per file, DIP where interfaces exist, no `os.Getenv` outside config)
    - `renderValidation(phase)` — converts `phase.ExitCriteria` checkboxes into `go build`/`go vet`/`go test`/`pnpm check` runnable commands; always appends `cd backend && go build ./...` and `cd backend && go vet ./...` for any Go-touching task; appends `cd frontend && pnpm check && pnpm build` for any frontend-touching task; never emits "per module test suite" string
    - `renderInvariantCheck(phase)` — reads `phase.InvariantChecks` slice from enricher overlay (§8.31.2); appends 4 canonical non-negotiable items unconditionally: `- [ ] No file modified outside "Files to create/modify" list`, `- [ ] No cross-module import introduced`, `- [ ] No \`os.Getenv\` outside \`config.go\``, `- [ ] All SQL via parameterized queries only (if DB-touching)`; falls back to these 4 items only when enricher produces nothing
    - `renderPromptContext(phase)` — reads `phase.PromptContextRefs` slice from enricher overlay; appends standard refs: §8.1 (canonical state), §8.4 (iteration engine) for engine-touching tasks; falls back to generic `§8.1 (CanonicalState), §8.31 (PLAN.md quality standard)`
  - Introduce `## 2. Milestones` table from `s.Plan["milestones"]` (short phase-name + one-sentence summary); remove `## 3. Phase Breakdown` section
  - Fix `## 1. Goals` section: read from `s.Idea["goals"]` ([]string); render as bullet list; fallback: `_Goals not populated — run at least one iteration before finalizing_`; never emit `_Goals not yet defined by the agents._`
  - Add `## 8. Deep Knowledge Reference` section assembled deterministically from canonical state fields: §8.1 inlined from `s` canonical state shape (Go struct skeleton), §8.N entries for each tech stack item with rationale from `s.Architecture["tech_stack_rationale"]`, module responsibility table from layer names, API endpoint table from `s.Architecture["api_endpoints"]` if present
  - Add ASCII dependency graph (from `s.Plan["dependency_graph_ascii"]` enricher field; fallback: generate a simple linear chain from `s.Plan["phases"]` order)
  - Add `## 7. How to Use This Plan` section with Task Execution Protocol block: "A task is NOT complete until: (1) all **Validation** commands produce zero output, (2) all **Invariant check** boxes are ticked, (3) no file outside **Files to create/modify** was touched"
  - Add `## 6. Task Summary` table (task number, name, primary files, depends-on, complexity) derived from `s.Plan["phases"]`
  - Risks deduplication: group `s.Risks` by `Title` similarity; keep highest-detail entry per group; emit deduplicated `## 6.N Risks` sub-section

- `backend/internal/modules/markdown/aigen/plan_enricher.go` — **new**:
  - `PlanEnricher` struct with `llm LLMProvider` dependency
  - `NewPlanEnricher(llm LLMProvider) *PlanEnricher`
  - `Enrich(ctx context.Context, s state.CanonicalState) (state.CanonicalState, error)`:
    - Returns original `s` unchanged on any error (LLM failure, invalid JSON, timeout) — non-fatal; logs at `slog.Warn`
    - Builds a compact prompt input: `{phases: [{name, complexity, deliverables_summary, exit_criteria_count}], idea_text, tech_stack_keys}` — trims content to minimize tokens (see §8.31.2 for exact input schema)
    - Single `llm.Generate` call with `GetPlanEnricherTimeoutSec()` deadline and the enricher system prompt from §8.31.4
    - Parses response as `PlanEnrichmentOverlay` struct (§8.31.2): `Phases []PhaseEnrichment` (indexed by position), `DependencyGraphASCII string`, `DeepKnowledgeSections []DKSection`
    - Each `PhaseEnrichment`: `CodingStandards []string`, `InvariantChecks []string`, `LayerTags []string`, `PromptContextRefs []string` — all optional; empty slice = use deterministic fallback
    - Merges overlay into a shallow copy of `s.Plan` phases: only writes fields where existing value is nil/empty; never overwrites human-curated state
    - Returns enriched copy; does not mutate input `s`
  - `PlanEnrichmentOverlay`, `PhaseEnrichment`, `DKSection` structs with JSON tags; `validateOverlay()` rejects empty strings, slices with >30 items per phase

- `backend/internal/modules/markdown/aigen/plan_enricher_test.go` — **new**:
  - Test: `Enrich` with mock LLM returning valid overlay JSON → `CodingStandards` + `InvariantChecks` + `LayerTags` + `DependencyGraphASCII` set on returned state; original `s` untouched
  - Test: `Enrich` with empty canonical state → returns same zero-value state; no panic
  - Test: `Enrich` with LLM returning malformed JSON → returns original state unchanged; no error propagated to caller
  - Test: `Enrich` with LLM returning partial overlay (only `DependencyGraphASCII`) → only that field set; per-phase fields use deterministic fallback

- `backend/internal/modules/markdown/aigen/generator.go` — **modify**:
  - For key `"plan"`: when `mode != "deterministic"` and `GetPlanEnricherEnabled()` is true, call `planEnricher.Enrich(ctx, s)` before the deterministic scaffold and replace `s` with the enriched copy
  - Log result: `slog.Info("plan enricher", "phases_enriched", countEnrichedPhases(overlay))`
  - Handle nil enricher gracefully (nil = enrichment skipped silently)
  - Constructor `New(...)` gains an optional `planEnricher *PlanEnricher` parameter alongside existing `enricher *ArchEnricher`

- `backend/internal/modules/markdown/aigen/rubric.go` — **modify**:
  - Update `plan` rubric entry: required sections now include `## 5. Implementation Tasks`, `## 7. How to Use This Plan`, `## 8. Deep Knowledge Reference`
  - Add `ForbiddenStrings` field to rubric: `["see phase deliverables", "per module test suite", "Phase 0 —", "Phase 1 —"]` — auto-repair loop must flag and reject any generated content containing these strings
  - Raise `plan` rubric `MinChars` to 3000 (real task specs are substantially longer than thin stubs)
  - Add `RequiredPatterns` field: `["\*\*Goal:\*\*", "\*\*Validation:\*\*", "\*\*Invariant check:\*\*"]` — at least one occurrence required per `### Task` block

- `backend/internal/platform/config/config.go` — **modify**:
  - `GetPlanEnricherEnabled() bool` — reads `PLAN_ENRICHER_ENABLED` env var; default `true`
  - `GetPlanEnricherTimeoutSec() int` — reads `PLAN_ENRICHER_TIMEOUT_SEC` env var; default `45`; clamp `[10, 120]`

- `backend/internal/modules/markdown/generator_plan_test.go` — **new**:
  - Test with sparse state (1 phase, minimal fields): §5 task block rendered; `**Goal:**` present; `**Validation:**` contains `go build ./...`; `**Invariant check:**` contains 4 canonical items; no "see phase deliverables" string anywhere
  - Test with fully populated state (3 phases, enricher overlay set): each task has `**Coding standards:**` (for High/Medium), `**Layer(s) affected:**`, `**Invariant check:**` with task-specific items
  - Test §1 Goals: present and non-empty when `s.Idea["goals"]` is a non-empty slice
  - Test §1 Goals fallback: emits `_Goals not populated_` message (NOT `_Goals not yet defined by the agents._`)
  - Test task heading format: all task headings match `### Task [0-9]+ —` regex; no `### Phase` headings present
  - Test `**Validation:**` field: never contains string `"per module test suite"`; always contains `go build ./...` for Go tasks
  - Test Risks deduplication: duplicate-title risks collapsed to single entry per group
  - Test §8 Deep Knowledge: section present and non-empty when `s.Architecture` has content

- `agent/internal/executor/prompts/plan_format.go` — **new** (Approach C — format injection into agent system prompts):
  - Package `prompts`
  - `PlanTaskFormat` constant string: the prompt fragment injected into builder/reviewer/refiner agent system prompts whenever `plan` is in the session's `output_docs` list (see §8.31.3 for the full fragment text)
  - Content of `PlanTaskFormat`:
    - Instructs agents to write each deliverable as `### Task N — {Name}` (not `### Phase N`)
    - Requires `**Goal:**` (one sentence, ≤ 120 chars), `**Files to create:**` (explicit file paths with Go function signatures nested), `**Validation:**` (runnable shell commands — `go build ./...`, `go vet ./...`, `go test ./path/...`, `pnpm check`; never "per module test suite"), `**Blocking Dependencies:**` (task numbers only)
    - Forbids vague language: "see phase deliverables", "per module test suite", "TBD", "as needed"
    - Instructs agents to include `Function Contracts` as nested sub-bullets under the primary service/handler file (not as a separate top-level section)
    - Instructs agents to include `Failure Handling` as a sub-bullet under the primary service file (not as a separate top-level section)
    - Instructs agents to include `Exit Criteria` inline as the `**Validation:**` checkboxes (behavioral assertions) alongside runnable commands
    - Maximum 2500 chars to stay within agent context budget
  - `InjectIfPlanOutput(outputDocs []string, basePrompt string) string` — appends `PlanTaskFormat` to `basePrompt` when `"plan"` is in `outputDocs`; otherwise returns `basePrompt` unchanged

- `agent/internal/executor/executor.go` — **modify**:
  - In `Execute`, when assembling the system prompt, call `prompts.InjectIfPlanOutput(payload.OutputDocs, basePrompt)` before passing to `llm.Generate`
  - `BrainstormPayload` gains `OutputDocs []string \`json:"output_docs,omitempty"\`` field (backward-compatible, omitempty); backend `DispatchFunc` populates it from `session.output_docs`
  - No other changes to executor logic

**Coding standards:**

- SRP: `PlanEnricher` has exactly one job — post-pass enrichment; `generator_plan.go` has exactly one job — deterministic render from state
- DIP: `PlanEnricher` depends on `LLMProvider` interface, not on any concrete LLM implementation
- OCP: new per-task render helpers (`renderCodingStandards`, `renderInvariantCheck`) are additive; existing `renderValidation` is extended, not replaced
- YAGNI: no caching layer for enrichment — plan generation is a one-time finalize call; no per-phase LLM calls — single batch call covers all phases
- No `fmt.Println`, no `os.Getenv` outside `config.go`, no cross-module imports

**Validation:**

- `cd backend && go build ./...`: zero build errors
- `cd backend && go vet ./...`: zero vet issues
- `cd agent && go build ./...`: zero build errors (executor + prompts package)
- `cd backend && go test ./internal/modules/markdown/...`: all existing tests pass + all new plan generator and enricher tests pass; zero failures
- `cd backend && go test ./internal/platform/config/...`: new `GetPlanEnricherEnabled` and `GetPlanEnricherTimeoutSec` getters covered
- `cd agent && go test ./internal/executor/...`: `InjectIfPlanOutput` tests pass
- Manual smoke — generate plan with `PLAN_ENRICHER_ENABLED=true`: finalize → download `*_plan.md`; verify §5 contains `### Task N —` headings (not `### Phase N`); verify `**Goal:**`, `**Validation:**` with `go build ./...`, `**Invariant check:**` with canonical items; verify §1 Goals non-empty; verify no "see phase deliverables" or "per module test suite" strings in output
- Manual smoke — generate plan with `PLAN_ENRICHER_ENABLED=false`: same assertions hold (deterministic fallbacks); no LLM call made for enrichment
- Manual smoke — `OutputDocs` containing `"plan"`: agent system prompt includes `PlanTaskFormat` fragment; agent output written in task spec format

**Invariant check:**

- [ ] No file modified outside the "Files to create/modify" list above
- [ ] No cross-module import introduced (`markdown/` does not import `modules/attachment/` or any other feature module)
- [ ] No `os.Getenv` outside `config.go`
- [ ] `PlanEnricher` returns original state unchanged on any LLM error — non-fatal contract never broken
- [ ] `generator_plan.go` never emits `"see phase deliverables"` or `"per module test suite"` strings — enforced by `TestPlanGeneratorForbiddenStrings` unit test
- [ ] `agent/internal/executor/prompts/` does not import any `backend/internal/` packages
- [ ] `BrainstormPayload.OutputDocs` field is `omitempty` — no breaking change to existing agent wire format

**Prompt context needed:** §8.31 (PLAN.md quality standard, `PlanEnrichmentOverlay` schema, enricher LLM prompt, Approach C prompt fragment spec, rubric assertions — new in this task), §8.27 (AI-Driven Hybrid Generator — `finalize_mode`, rubric validator, auto-repair loop — enricher wires into this flow), §8.23 (Generated Document Quality Standard), §8.30 (Architecture.md enricher — same pattern reused here), Task 35 (arch enricher — structural model for plan enricher), Task 34 (`generator_plan.go` current baseline this task rewrites)

---

### Task 37 — README.md: Generation Quality Overhaul — Enricher + Correct Section Structure

**Goal:** Rewrite `generator_readme.go` to produce all 13 required README sections, add a new `ReadmeEnricher` LLM post-pass (Approach B) that fills content-heavy sections (golden rule, "When to use" scenarios with code examples, Mermaid flowcharts, Quick Start commands, Command Reference table, Contributing note, Prerequisites), and add a `ReadmeFormat` agent prompt constant (Approach C) that directs the LLM agent to write README-appropriate content — not roadmap phases — when `readme` is in `output_docs`. Removes the three incorrect sections currently hard-coded in the generator (`## Known Risks` top-level, `## Roadmap` with "Phase N —" bullets, For AI Agents appendix).

**Files to create:**

- `backend/internal/modules/markdown/generator_readme.go` — full rewrite; see §8.32.1 for the 13-section contract:
  - Section 1 `# {ShortTitle}` — no `— README` suffix; same `shortTitle(s)` helper as arch generator
  - Section 2 `> {tagline}` — from `s.Architecture["tagline"]` (enricher) → `oneLineDescription(s)` fallback
  - Section 3 **Description paragraph** — from `s.Architecture["description_paragraph"]` (enricher) → `s.Idea["context"]` fallback; always 2-3 sentences
  - Section 4 **Golden Rule** — from `s.Architecture["golden_rule"]` (enricher); `**Golden rule:** {text}` format; fallback: `_No golden rule defined._`
  - Section 5 **What it is NOT** — from `s.Architecture["is_not"]` (enricher); bullet list of 3-5 scope boundaries; fallback: 3 generic anti-patterns based on tech stack
  - Section 6 **When to use** — Mermaid flowchart from `s.Architecture["when_to_use_mermaid"]` (enricher); numbered scenario list from `s.Architecture["when_to_use"]` (enricher); each scenario: `### N. {title}` + description + fenced code block; fallback: 3 generic "use when building X" scenarios without code
  - Section 7 **Installation / Prerequisites** — prerequisites table from `s.Architecture["prerequisites"]` (enricher); setup steps from `s.Architecture["quick_start_commands"]` (enricher) first item; fallback: generic `go install` + `docker compose up` steps
  - Section 8 **Quick Start** — fenced bash block, commands from `s.Architecture["quick_start_commands"]` (enricher); fallback: 3 generic make commands
  - Section 9 **Architecture** — ASCII diagram from `s.Architecture["architecture_ascii"]` (enricher) → `renderASCIIComponents(s)` fallback; Mermaid architecture diagram from `s.Architecture["architecture_mermaid"]` (enricher) → `renderDataFlowsMermaid(s)` fallback; both always rendered if available
  - Section 10 **Command Reference** — table (`| Command | Description |`) from `s.Architecture["command_reference"]` (enricher); fallback: single row placeholder
  - Section 11 **Tech Stack** — from `renderTechStack(s)` (existing helper); enricher does not override, deterministic only
  - Section 12 **Repository Format** — directory tree from `renderDirectoryTree(s)` (existing helper) with enriched per-entry description from `s.Architecture["repo_format_notes"]` (enricher); fallback: tree with generic notes
  - Section 13 **Contributing** — from `s.Architecture["contributing_note"]` (enricher); fallback: "Read `AGENTS.md` and `.github/skills/` before changing this codebase."
  - **Remove**: `## Known Risks` section, For AI Agents appendix (`renderForAIAgentsAppendix`), `## Roadmap` Phase-N bullets; status footer `_Iteration N · Confidence N_` moves to HTML comment
  - `go build ./...` zero errors; no `renderForAIAgentsAppendix` call for readme doc type

- `backend/internal/modules/markdown/aigen/readme_enricher.go` — new `ReadmeEnricher` struct; see §8.32.2 for `ReadmeEnrichmentOverlay` schema:
  - `type ReadmeEnricher struct { llm platform_llm.LLMProvider; timeoutSec int }`
  - `func NewReadmeEnricher(llm platform_llm.LLMProvider, timeoutSec int) *ReadmeEnricher`
  - `func (e *ReadmeEnricher) Enrich(ctx context.Context, s state.CanonicalState) (state.CanonicalState, error)`:
    - Builds a single LLM prompt from `s` (see §8.32.3 for the enricher LLM system prompt); requests JSON response conforming to `ReadmeEnrichmentOverlay`
    - Merges overlay fields into `s.Architecture` map using the key names in §8.32.2 (`tagline`, `description_paragraph`, `golden_rule`, `is_not`, `when_to_use`, `when_to_use_mermaid`, `quick_start_commands`, `command_reference`, `architecture_ascii`, `architecture_mermaid`, `repo_format_notes`, `contributing_note`, `prerequisites`)
    - On any LLM error: returns original `s` unchanged — non-fatal, generator falls back to deterministic stubs
    - Timeout from `ctx` + `e.timeoutSec` deadline; no retry (single call)
    - Structured `slog` logging on success and failure; never logs API keys

- `backend/internal/modules/markdown/aigen/readme_enricher_test.go` — four test cases:
  - `TestReadmeEnricher_EnrichPopulatesOverlay`: golden rule, when_to_use, Mermaid flowchart, quick_start_commands, command_reference all present in returned state after a successful mock LLM call
  - `TestReadmeEnricher_EnrichFallsBackOnLLMError`: LLM provider returns error → `Enrich` returns original state unchanged, no error propagated
  - `TestReadmeEnricher_OverlayMergeNoOverwrite`: if `s.Architecture["golden_rule"]` was already set before enrichment, enricher MUST overwrite it with the LLM value (enricher always wins over pre-existing deterministic values)
  - `TestReadmeEnricher_NilLLM`: `NewReadmeEnricher(nil, 30)` then `Enrich` returns original state, no panic

- `backend/internal/modules/markdown/aigen/generator.go` — modified to wire `ReadmeEnricher`:
  - Add `ReadmeEnricher *ReadmeEnricher` field to `Generator` struct (alongside existing `Enricher` and `PlanEnricher`)
  - In `Generate(ctx, s, docKey)`: when `docKey == "readme"` and `g.ReadmeEnricher != nil`, call `g.ReadmeEnricher.Enrich(ctx, s)` and replace `s` before passing to `markdown.GenerateReadme(enrichedState)`
  - Rubric repair loop (existing) runs AFTER enricher, on the generated markdown string
  - No change to existing `architecture` or `plan` enricher paths

- `backend/internal/modules/markdown/aigen/rubric.go` — add README rubric entry:
  - `"readme"` key in the rubric map (same `DocRubric` struct used by `architecture` and `plan`)
  - `MinChars: 1500`
  - `ForbiddenStrings: []string{"<repository-url>", "Phase 1 —", "Phase 2 —", "Phase 3 —", "## Known Risks\n", "For AI Agents", "## Roadmap\n\n_Roadmap"}`
  - `RequiredPatterns: []string{"` + "```" + `mermaid", "## When to use", "## Contributing", "Golden rule"}`
  - `MaxRepairPasses: 3` — auto-repair loop retries up to 3 times with a correction prompt that lists which required patterns are missing or which forbidden strings are present

- `backend/internal/platform/config/config.go` — add two new getters (modified):
  - `GetReadmeEnricherEnabled() bool` — env `README_ENRICHER_ENABLED`, default `true`; returns false when `FINALIZE_MODE=deterministic`
  - `GetReadmeEnricherTimeoutSec() int` — env `README_ENRICHER_TIMEOUT_SEC`, default `45`, clamped `[10, 120]`
  - See §8.32.4 for the full config table entry

- `backend/internal/modules/markdown/generator_readme_test.go` — eight test cases:
  - `TestGenerateReadme_AllThirteenSectionsPresent`: state with rich `Architecture` map → output contains all 13 H2 headings
  - `TestGenerateReadme_TitleNoReadmeSuffix`: `# {ShortTitle}` — never contains `— README`
  - `TestGenerateReadme_ForbiddenStringsAbsent`: output never contains `"<repository-url>"`, `"Phase 1 —"`, `"## Known Risks\n"`, `"For AI Agents"`, `"renderForAIAgentsAppendix"` output sentinel
  - `TestGenerateReadme_GoldenRuleRendered`: `s.Architecture["golden_rule"] = "…"` → `**Golden rule:**` line present in output
  - `TestGenerateReadme_WhenToUseWithMermaid`: `s.Architecture["when_to_use_mermaid"]` set → output contains ` ``` mermaid` fence
  - `TestGenerateReadme_WhenToUseScenariosWithCode`: `s.Architecture["when_to_use"]` with 3 scenarios each having `"code"` field → 3 fenced code blocks present
  - `TestGenerateReadme_QuickStartRealCommands`: `s.Architecture["quick_start_commands"] = ["make dev", "open http://localhost:5173"]` → exact strings in output, not `git clone <repository-url>`
  - `TestGenerateReadme_FallbacksForEmptyState`: `CanonicalState{}` with minimal idea → output ≥ 800 chars, all 13 headings present, no panics

- `agent/internal/executor/prompts/readme_format.go` — new file (Approach C); see §8.32.5 for the full constant text:
  - `const ReadmeFormat = "..."` — multi-line string that instructs the LLM agent to write README-appropriate content when `readme` is in `output_docs`: include golden rule as a one-sentence invariant, write `## When to use` with numbered scenarios and code examples (not roadmap phases), never use `"Phase N —"` bullets, write `## Contributing` section, Quick Start must use real runnable commands specific to the described tech stack
  - `func InjectIfReadmeOutput(base string, outputDocs []string) string` — appends `ReadmeFormat` to `base` when `"readme"` is in `outputDocs`; no-op otherwise; same pattern as `InjectIfPlanOutput` from Task 36

**Validation:**

- [ ] `go build ./...`: zero build errors in both `backend/` and `agent/`
- [ ] `go vet ./...`: zero vet issues in both `backend/` and `agent/`
- [ ] `go test ./backend/internal/modules/markdown/...`: all 8 generator tests pass, all 4 enricher tests pass
- [ ] `go test ./backend/internal/modules/markdown/aigen/...`: rubric test for `"readme"` key passes; `TestRubricReadme_RequiredPatternsEnforced` confirms repair triggers when `## When to use` is missing
- [ ] `go test ./agent/internal/executor/...`: `TestInjectIfReadmeOutput_InjectsWhenPresent` and `TestInjectIfReadmeOutput_NoopWhenAbsent` pass
- [ ] Manual spot-check: `GenerateReadme(richState)` output contains `**Golden rule:**`, ` ``` mermaid` fence, `## When to use`, `## Contributing`; does NOT contain `<repository-url>`, `## Known Risks`, `For AI Agents`

**Invariant check:**

- [ ] `generator_readme.go` never calls `renderForAIAgentsAppendix` — README is human-facing; use `grep -r "renderForAIAgentsAppendix" backend/internal/modules/markdown/generator_readme.go` → zero results
- [ ] `readme_enricher.go` never imports `os` or calls `os.Getenv` — config access only via `platform/config`
- [ ] `readme_format.go` in `agent/internal/executor/prompts/` does not import any `backend/internal/` package — zero cross-binary imports

**Layer(s) affected:** `backend/internal/modules/markdown/` (generator + enricher + rubric + aigen wiring), `backend/internal/platform/config/` (two new getters), `agent/internal/executor/prompts/` (Approach C constant)

**Prompt context needed:** §8.32 (README quality standard, `ReadmeEnrichmentOverlay` schema, enricher LLM prompt, Approach C prompt fragment, rubric assertions — new in this task), §8.27 (AI-Driven Hybrid Generator — `finalize_mode`, rubric validator, auto-repair loop — enricher wires into this flow), §8.23 (Generated Document Quality Standard), §8.30 (Architecture.md enricher — structural model for readme enricher), §8.31 (PLAN.md enricher — Approach C pattern reused here), Task 36 (plan enricher + `InjectIfPlanOutput` — README enricher mirrors this), Task 35 (arch enricher — `ReadmeEnricher` reuses the same Enrich/merge pattern)

---

### Task 38 — DB: Attachments + Chunks Schema (pgvector)

**Goal:** Create the two database migrations that introduce the `attachments` table (per-upload metadata) and the `attachment_chunks` table (RAG-lite chunks with pgvector embeddings). Defines the `attachment_scope` and `attachment_kind` enums used by all subsequent attachment tasks. No Go or frontend code in this task — schema + enum only.

**Files to create:**

- `migrations/009_attachments.sql` — see §8.28 for exact DDL:
  - `CREATE EXTENSION IF NOT EXISTS vector` — pgvector required for cosine similarity
  - `CREATE TYPE attachment_scope AS ENUM ('session','iteration','agent')` — lifecycle binding
  - `CREATE TYPE attachment_kind AS ENUM ('file','image','url','text')` — input modality
  - `attachments` table: `id UUID PK DEFAULT gen_random_uuid()`, `session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE`, `scope attachment_scope NOT NULL`, `scope_ref TEXT` (nullable; `iteration_number` as string when scope=iteration, `agent_id` UUID string when scope=agent, NULL when scope=session), `kind attachment_kind NOT NULL`, `display_name TEXT NOT NULL`, `mime_type TEXT NOT NULL DEFAULT ''`, `byte_size BIGINT NOT NULL DEFAULT 0`, `source_url TEXT` (set when kind=url), `blob_key TEXT` (object-storage key when kind in (file,image); NULL otherwise), `extracted_text TEXT NOT NULL DEFAULT ''` (full extracted/cleaned text), `summary TEXT NOT NULL DEFAULT ''` (≤ 500 chars, used when chunk retrieval returns nothing), `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
  - Scope-consistency CHECK constraint: `(scope = 'session' AND scope_ref IS NULL) OR (scope IN ('iteration','agent') AND scope_ref IS NOT NULL)`
  - Index: `CREATE INDEX idx_attachments_session_scope ON attachments (session_id, scope, scope_ref)` — supports retrieval queries
- `migrations/010_attachment_chunks.sql`:
  - `attachment_chunks` table: `id UUID PK DEFAULT gen_random_uuid()`, `attachment_id UUID NOT NULL REFERENCES attachments(id) ON DELETE CASCADE`, `chunk_index INT NOT NULL`, `content TEXT NOT NULL`, `embedding VECTOR(1536) NOT NULL` (dimension matches `EMBEDDING_DIM` config; default 1536 for OpenAI `text-embedding-3-small`), `tokens INT NOT NULL DEFAULT 0`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
  - `UNIQUE (attachment_id, chunk_index)` — idempotent re-upload semantics
  - IVF-Flat index: `CREATE INDEX idx_attachment_chunks_embedding ON attachment_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)` — cosine similarity search

**Validation:**

- `docker compose run --rm backend ./migrate up` — both migrations apply cleanly; pgvector extension created
- `psql -c "\dT+ attachment_scope"` — shows enum with three values; same for `attachment_kind`
- `psql -c "\d attachments"` — shows all 12 columns + scope-consistency CHECK + cascade FK
- `psql -c "\d attachment_chunks"` — shows VECTOR(1536) column + unique constraint + ivfflat index
- `psql -c "INSERT INTO attachments (session_id, scope, scope_ref, kind, display_name) VALUES ('00000000-0000-0000-0000-000000000000', 'iteration', NULL, 'text', 'x')"` — fails due to scope CHECK; same INSERT with `scope_ref = '1'` succeeds (provided session exists)
- `docker compose run --rm backend ./migrate down` — rolls back both cleanly; extension preserved (other tables may use it)

**Prompt context needed:** §8.28 (Attachment system schema + scope semantics — new in this task), AGENTS.md migration rules (append-only, numbered sequentially), pgvector docs for IVF-Flat tuning

---

### Task 39 — Platform: Extractor + Embeddings + Blobstore Infrastructure

**Goal:** Build the three platform-layer infrastructure packages every attachment upload depends on: `extractor/` (turns any input modality into clean UTF-8 text), `embeddings/` (turns text into vectors via `LLMProvider`-style interface), and `blobstore/` (MinIO/S3 object storage for original file bytes). These are pure infrastructure — they own no domain logic and are reused only by `modules/attachment/`. Also extends `docker-compose.yml` with the MinIO service and the `config/` package with all related env getters.

**Files to create:**

- `backend/internal/platform/extractor/extractor.go`:
  - `Extractor` interface: `Extract(ctx context.Context, input ExtractInput) (ExtractResult, error)`
  - `ExtractInput` struct: `Kind string` (`file`|`image`|`url`|`text`), `Reader io.Reader` (nil for url/text), `URL string`, `Text string`, `MimeType string`, `DisplayName string`
  - `ExtractResult` struct: `Text string`, `MimeType string`, `ByteSize int64`, `SourceURL string`
  - `Registry` type: maps kind → `Extractor`; `Resolve(kind) Extractor`; returns error on unknown kind
- `backend/internal/platform/extractor/plaintext.go`:
  - Handles `text`, `file` with mime `text/*` or `application/json` or `application/markdown`, also `text/plain` URL responses
  - Normalises line endings, strips control chars, returns text verbatim (no truncation)
- `backend/internal/platform/extractor/pdf.go`:
  - Uses `github.com/ledongthuc/pdf` (or `github.com/dslipak/pdf`) — pure-Go, no CGO, no external binaries
  - Page-by-page extraction; concatenates with `\n\n` separators; falls back to empty text on parse failure (logged warn, not error)
- `backend/internal/platform/extractor/url.go`:
  - `net/http` GET with 15s timeout; allowlist HTTP/HTTPS only (rejects `file://`, `ftp://`, `data:` — SSRF guard)
  - Honours `Content-Type`; routes HTML through `golang.org/x/net/html` text extraction (strips `<script>`, `<style>`, `<nav>`, `<footer>`); routes PDF responses through `pdf.go`; routes JSON / plain through `plaintext.go`
  - Body cap: `GetAttachmentMaxBytes()` from config (default 10 MB); rejects oversize with descriptive error
  - User-Agent header includes `a2a-brainstorm/<version>`
- `backend/internal/platform/extractor/image.go`:
  - Calls `LLMProvider.Generate` with a fixed system prompt requesting a dense factual description (≤ 400 words) of the image; the image is sent as a base64 data URL in the user message
  - Requires the active provider to support vision (Copilot vision, Claude vision, OpenCode with vision-capable model); on unsupported provider returns sentinel error `ErrVisionUnsupported` so service layer can degrade gracefully
  - Provider is injected into the extractor at construction; no `os.Getenv` here
- `backend/internal/platform/embeddings/embeddings.go`:
  - `EmbeddingsProvider` interface: `Embed(ctx context.Context, texts []string) ([][]float32, error)`, `Dimension() int`
  - Mirrors the `LLMProvider` pattern; credential resolution goes through `config.GetEmbeddingsCredentialRef()`
- `backend/internal/platform/embeddings/openai.go`:
  - `OpenAIEmbeddingsProvider` implements the interface using OpenAI `text-embedding-3-small` (1536 dim) by default; configurable via `EMBEDDINGS_MODEL` env
  - Batches up to 100 texts per HTTP call; retries on 429/5xx with exponential backoff (max 3 attempts)
- `backend/internal/platform/blobstore/blobstore.go`:
  - `Blobstore` interface: `Put(ctx, key string, r io.Reader, size int64, contentType string) error`, `Get(ctx, key string) (io.ReadCloser, error)`, `Delete(ctx, key string) error`
  - Deterministic key shape: `attachments/{sessionID}/{attachmentID}/{slug(displayName)}`
- `backend/internal/platform/blobstore/minio.go`:
  - `MinioBlobstore` implements the interface using `github.com/minio/minio-go/v7`
  - Bucket name from `BLOBSTORE_BUCKET` (default `a2a-attachments`); auto-creates bucket on first `Put` if missing
- `backend/internal/platform/config/config.go` — add getters (every `os.Getenv` call lives here, never elsewhere):
  - `GetEmbeddingsProvider()`, `GetEmbeddingsModel()`, `GetEmbeddingsCredentialRef()`, `GetEmbeddingsDimension()` (default 1536, validates ≥ 64)
  - `GetBlobstoreEndpoint()`, `GetBlobstoreAccessKeyRef()`, `GetBlobstoreSecretKeyRef()`, `GetBlobstoreBucket()`, `GetBlobstoreUseSSL()`
  - `GetAttachmentMaxBytes()` (default 10_485_760 = 10 MB), `GetAttachmentChunkSize()` (default 1000 tokens), `GetAttachmentChunkOverlap()` (default 150 tokens), `GetAttachmentRetrievalTopK()` (default 5)
- `docker-compose.yml` — add `minio` service (`minio/minio:latest`, port 9000 API + 9001 console, `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` env, healthcheck on `/minio/health/live`, volume `minio-data`); add to `backend` `depends_on`
- `.env.example` — append all new env vars with placeholder values and inline comments

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- `cd backend && go test ./internal/platform/extractor/...`: covers plaintext passthrough, PDF byte stream → non-empty text, URL with mock HTTP server (HTML → text-only, JSON → verbatim), SSRF guard rejects `file://`, oversize-body rejection
- `cd backend && go test ./internal/platform/embeddings/...`: covers happy path with `httptest` mock, batch chunking ≤ 100, retry-after-429
- `cd backend && go test ./internal/platform/blobstore/...`: skipped if `BLOBSTORE_ENDPOINT` not set; otherwise round-trips Put/Get/Delete
- `docker compose up minio -d` — service healthy; console reachable on `:9001`
- No `os.Getenv` call anywhere outside `platform/config/config.go` (grep check)

**Prompt context needed:** §8.28 (extraction pipeline + embedding dimension contract — new in this task), §8.12 (credential security — env-var-name-only rule), §5 of `.github/copilot-instructions.md` (`os.Getenv` confined to config file)

---

### Task 40 — Backend: Attachment Module (CRUD + Upload Pipeline)

**Goal:** Implement the full `modules/attachment/` vertical slice — `model.go`, `repository.go`, `service.go`, `handler.go`. The service orchestrates the upload pipeline: extract text → chunk → embed → persist blob (if applicable) → persist attachment + chunks atomically. Exposes REST endpoints for creating attachments via all four input kinds (file multipart, image multipart, URL JSON, raw-text JSON), listing by scope, deleting, and an internal-only retrieval endpoint used by the iteration engine.

**Files to create:**

- `backend/internal/modules/attachment/model.go` — see §8.28 for full schema:
  - `Attachment` struct: matches `attachments` row 1-to-1 + computed `BlobURL string` (presigned URL when `Kind in (file,image)`, empty otherwise)
  - `AttachmentChunk` struct: `ID uuid.UUID`, `AttachmentID uuid.UUID`, `Index int`, `Content string`, `Tokens int`, `Score float32` (populated only by retrieval queries)
  - `Scope` constants: `ScopeSession`, `ScopeIteration`, `ScopeAgent` — typed string aliases
  - `Kind` constants: `KindFile`, `KindImage`, `KindURL`, `KindText`
  - `CreateAttachmentInput` struct: `SessionID uuid.UUID`, `Scope Scope`, `ScopeRef *string`, `Kind Kind`, `DisplayName string`, `MimeType string`, `Reader io.Reader` (for file/image), `URL string` (for url), `Text string` (for text)
- `backend/internal/modules/attachment/repository.go`:
  - `Create(ctx, tx pgx.Tx, att Attachment) (Attachment, error)` — INSERT row
  - `CreateChunks(ctx, tx pgx.Tx, attachmentID uuid.UUID, chunks []AttachmentChunk) error` — batched COPY-style INSERT
  - `GetByID(ctx, id) (Attachment, error)`
  - `ListBySession(ctx, sessionID uuid.UUID, scopeFilter *Scope, scopeRefFilter *string) ([]Attachment, error)` — ordered by `created_at ASC`
  - `Delete(ctx, id uuid.UUID) error` — cascade removes chunks
  - `DeleteByScope(ctx, sessionID uuid.UUID, scope Scope, scopeRef string) error` — used by lifecycle cleanup (Task 40)
  - `SearchChunks(ctx, sessionID uuid.UUID, scopes []ScopeMatch, queryEmbedding []float32, topK int) ([]AttachmentChunk, error)` — pgvector cosine similarity (`embedding <=> $1::vector`) filtered to attachments whose `(scope, scope_ref)` matches any entry in `scopes`; returns top-K ordered by ascending distance with `Score = 1 - distance`
  - `ScopeMatch` struct: `Scope Scope`, `ScopeRef *string` (NULL for session-scope match)
- `backend/internal/modules/attachment/service.go`:
  - Constructor: `NewService(repo, blob blobstore.Blobstore, extractors extractor.Registry, embeddings embeddings.EmbeddingsProvider, db *pgxpool.Pool, cfg ServiceConfig) *Service`
  - `Create(ctx, input CreateAttachmentInput) (Attachment, error)`:
    1. Validate input per kind (file/image require non-nil reader + display name; url requires HTTP/HTTPS URL; text requires non-empty Text)
    2. Validate scope/scope_ref consistency: session ⇒ ref nil; iteration ⇒ ref is integer string; agent ⇒ ref is UUID string referencing an agent assigned to the session
    3. Call extractor → get `ExtractResult.Text`; reject if text length < `MinExtractedChars` (default 16) — guards against empty PDFs / failed scrapes
    4. For `kind in (file,image)`: stream original bytes to `blobstore.Put` under the deterministic key (see §8.28 key shape); on failure, do not insert DB row
    5. Chunk text via `chunkText(text, size, overlap)` helper (token-aware splitting, paragraph-respecting; see §8.28 chunking algorithm)
    6. Embed all chunks in a single `embeddings.Embed` call
    7. Generate `summary` via one short `LLMProvider.Generate` call (system prompt: "Summarise this document in ≤ 500 chars for retrieval fallback purposes"); on LLM failure, summary is empty (non-fatal)
    8. Wrap insert in a single transaction: `repo.Create(tx, attachment)` → `repo.CreateChunks(tx, attachmentID, chunks)` → commit; on rollback, also delete the blob (best-effort cleanup; log warn on failure)
    9. Emit log: `slog.Info("attachment created", attachment_id, scope, scope_ref, kind, chunk_count, byte_size)`
  - `List(ctx, sessionID, filter)`, `GetByID`, `Delete` — straight repository delegation; `Delete` also removes blob best-effort
  - `Retrieve(ctx, sessionID uuid.UUID, scopes []ScopeMatch, queryText string, topK int) ([]AttachmentChunk, error)` — embeds queryText (single call), delegates to `repo.SearchChunks`; used by iteration engine in Task 40
- `backend/internal/modules/attachment/handler.go`:
  - `POST   /sessions/{sessionID}/attachments` — Content-Type-driven multiplexer:
    - `multipart/form-data` (fields `scope`, `scope_ref`, `kind`, `file`, `display_name?`) → file or image upload
    - `application/json` body with `{scope, scope_ref, kind: "url", url, display_name?}` → URL ingest
    - `application/json` body with `{scope, scope_ref, kind: "text", text, display_name}` → raw text paste
    - Returns 201 + `Attachment`; 400 on validation; 413 on oversize; 415 on unsupported MIME; 422 on extraction failure (e.g. password-protected PDF)
  - `GET    /sessions/{sessionID}/attachments?scope=&scope_ref=` → 200 + `[]Attachment`
  - `GET    /sessions/{sessionID}/attachments/{id}` → 200 + `Attachment`; 404 if not found
  - `GET    /sessions/{sessionID}/attachments/{id}/content` → 302 redirect to presigned blob URL (only for kind in file/image); 404 otherwise
  - `DELETE /sessions/{sessionID}/attachments/{id}` → 204; 404 if not found
  - Server-side caps: `http.MaxBytesReader(r.Body, GetAttachmentMaxBytes())` on every POST; reject MIME types not in `AllowedMimeTypes` (PDF, DOCX, MD, TXT, JSON, PNG, JPG, JPEG, WEBP)
- `backend/internal/platform/http/router.go` — register all 5 routes under `attachment.NewHandler`; group prefix `/sessions/{sessionID}/attachments`

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- `cd backend && go test ./internal/modules/attachment/...`: covers happy path for all four kinds (with fake blobstore + fake embeddings + fake extractor), scope/scope_ref validation, extraction-failure 422, transaction rollback also removes blob, retrieval returns top-K ordered by score
- Manual smoke:
  - `curl -F scope=session -F kind=file -F file=@docs/A2A-agent-Brainstorm.md http://localhost:8080/sessions/{id}/attachments` → 201 with non-empty `extracted_text`
  - `curl -d '{"scope":"session","kind":"url","url":"https://example.com"}' -H 'Content-Type: application/json' http://localhost:8080/sessions/{id}/attachments` → 201
  - `curl -d '{"scope":"agent","scope_ref":"<agent-uuid>","kind":"text","text":"Use Postgres 16","display_name":"db-pick"}' -H 'Content-Type: application/json' http://localhost:8080/sessions/{id}/attachments` → 201
  - `curl -d '{"scope":"agent","kind":"text","text":"x","display_name":"y"}' -H 'Content-Type: application/json' http://localhost:8080/sessions/{id}/attachments` → 400 (missing scope_ref)
  - SSRF: URL `file:///etc/passwd` → 400

**Prompt context needed:** §8.28 (attachment domain model + upload pipeline algorithm + chunking algorithm), Task 39 (extractor / embeddings / blobstore interfaces), AGENTS.md vertical-slice rules, security invariants 5/6 (parameterized queries, input validation)

---

### Task 41 — Backend: AttachmentRetriever + Payload Extension + Iteration Engine Wiring

**Goal:** Thread attachments into the dispatch path. Introduce a narrow `AttachmentRetriever` interface owned by the iteration engine (same pattern as `agentProvider` and `sessionStore`). Before dispatching each agent, the engine resolves the active scope set (session ∪ current iteration ∪ current agent), retrieves top-K chunks via cosine similarity against the canonical state's `idea + open_questions`, and appends them to `BrainstormPayload` as a new `Attachments []AttachmentChunkRef` field. The agent executor injects the chunks into the assembled system prompt under a dedicated `# Attached Context` section. Iteration- and agent-scoped attachments are deleted after their owning iteration completes via a lifecycle cleanup pass.

**Files to modify:**

- `backend/internal/modules/iteration/engine.go`:
  - Add `attachmentRetriever` interface field:
    ```go
    type attachmentRetriever interface {
        Retrieve(ctx context.Context, sessionID uuid.UUID, scopes []attachment.ScopeMatch, queryText string, topK int) ([]attachment.AttachmentChunk, error)
        DeleteByScope(ctx context.Context, sessionID uuid.UUID, scope attachment.Scope, scopeRef string) error
    }
    ```
  - Extend `NewEngine` constructor with `retriever attachmentRetriever` argument (nullable; nil = attachments disabled, no behaviour change)
  - In `runPipelinePass`, before each agent dispatch:
    1. Build query text: `current.Idea + "\n\n" + strings.Join(current.OpenQuestions, "\n")`
    2. Build scope match list: always include `{Scope: session, ScopeRef: nil}`; if iteration > 0 include `{Scope: iteration, ScopeRef: &strconv.Itoa(i)}`; always include `{Scope: agent, ScopeRef: &agent.ID}`
    3. Call `retriever.Retrieve(ctx, sessID, scopes, query, GetAttachmentRetrievalTopK())` → `chunks`
    4. Convert to `[]platA2A.AttachmentChunkRef` (drop DB IDs; keep content, score, scope, display name) — agent binary only needs prompt-relevant fields
    5. Pass through `DispatchFunc` via a new optional argument `attachments []platA2A.AttachmentChunkRef`
  - After the iteration's `UpdateState` succeeds, call `retriever.DeleteByScope(ctx, sessID, ScopeIteration, strconv.Itoa(i))` to expire iteration-scoped attachments; `ScopeAgent` cleanup runs after each agent's individual dispatch returns
- `backend/internal/platform/a2a/types.go`:
  - Add `AttachmentChunkRef` struct: `Scope string \`json:"scope"\``, `ScopeRef string \`json:"scope_ref,omitempty"\``, `DisplayName string \`json:"display_name"\``, `Content string \`json:"content"\``, `Score float32 \`json:"score"\``
  - Extend `BrainstormPayload` with `Attachments []AttachmentChunkRef \`json:"attachments,omitempty"\`` — backward compatible (omitempty)
- `backend/internal/modules/agent/client.go`:
  - Extend `Dispatch` signature with `attachments []platA2A.AttachmentChunkRef` argument (after `currentState`)
  - Pass through to `payload.Attachments`
  - Pre-flight log: `slog.Info("dispatch with attachments", agent_id, chunk_count, total_chars)`
- `agent/internal/executor/executor.go`:
  - Add `AttachmentChunkRef` struct (mirror, copied verbatim — agent binary must not import backend packages)
  - Extend `BrainstormPayload` with `Attachments []AttachmentChunkRef`
  - In `Execute`, after assembling the base system prompt, append a dedicated section when `len(payload.Attachments) > 0`:

    ```
    # Attached Context

    The following snippets were retrieved from user-attached artifacts for this dispatch. Treat them as authoritative context for the brainstorm but do not echo them verbatim.

    ## [scope: {Scope} | source: {DisplayName} | relevance: {Score:.2f}]
    {Content}
    --- (repeated per chunk)
    ```

  - Order chunks by descending `Score` for prompt priority

- `backend/internal/modules/iteration/engine_test.go`:
  - Add test: nil retriever → engine runs unchanged (zero-regression guarantee)
  - Add test: retriever returns 2 chunks → `DispatchFunc` receives 2 `AttachmentChunkRef`s in score-descending order
  - Add test: after each iteration, `DeleteByScope(ScopeIteration, "i")` invoked exactly once with the correct iteration number
- `agent/internal/executor/executor_test.go`:
  - Add test: payload with 3 attachments → assembled system prompt contains "# Attached Context" header and all 3 contents in descending score order
  - Add test: payload with empty attachments → no `# Attached Context` section appears (byte-identical to existing behaviour)

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- `cd agent && go build ./...`: zero errors
- `cd backend && go test ./internal/modules/iteration/...`: all existing tests still pass + 3 new tests pass
- `cd agent && go test ./internal/executor/...`: all existing tests still pass + 2 new tests pass
- Manual smoke: create session, attach a Markdown doc at session scope, run iterate → agent logs show `dispatch with attachments chunk_count=N>0`; converged state cites attachment content
- Manual smoke: attach a text snippet at iteration scope (`scope_ref=1`), run 2 iterations → after iteration 1 the snippet is deleted; iteration 2 dispatch has chunk_count for that snippet = 0

**Prompt context needed:** §8.28 (AttachmentChunkRef wire format + scope resolution algorithm + system-prompt injection format), §8.3 (BrainstormPayload contract), §8.4 (iteration engine algorithm — extends step 1a), Task 40 (Service.Retrieve + Service.DeleteByScope), Task 9 (engine architecture)

---

### Task 42 — Frontend: Attachment Menu + Upload Modal + Scope-Aware Mount Points

**Goal:** Build the ChatGPT-style `+` attachment menu UX and mount it at three scope-bound locations: home page (session-scope, set during creation), session page (iteration-scope, added between iterations), and `PipelineStage` per-agent header (agent-scope, narrows the next dispatch). Includes the modal with four input kinds (file picker, image picker, URL paste, raw text paste), a sticky list of active attachments per scope, and the API client layer.

**Files to create / modify:**

- `frontend/src/lib/types.ts` — add:
  - `AttachmentScope` type: `'session' | 'iteration' | 'agent'`
  - `AttachmentKind` type: `'file' | 'image' | 'url' | 'text'`
  - `Attachment` interface: `id`, `session_id`, `scope`, `scope_ref`, `kind`, `display_name`, `mime_type`, `byte_size`, `source_url`, `summary`, `created_at`, `blob_url?: string`
  - `CreateAttachmentInput` discriminated union per kind
- `frontend/src/lib/services/api.ts` — add:
  - `listAttachments(sessionId: string, scope?: AttachmentScope, scopeRef?: string): Promise<Attachment[]>`
  - `uploadFileAttachment(sessionId, scope, scopeRef, file: File): Promise<Attachment>` — uses `FormData` multipart
  - `uploadImageAttachment(sessionId, scope, scopeRef, image: File): Promise<Attachment>` — same, kind=image
  - `uploadURLAttachment(sessionId, scope, scopeRef, url: string, displayName?): Promise<Attachment>` — JSON body
  - `uploadTextAttachment(sessionId, scope, scopeRef, text: string, displayName: string): Promise<Attachment>` — JSON body
  - `deleteAttachment(sessionId, id): Promise<void>`
- `frontend/src/lib/stores/attachmentStore.ts` — **new**:
  - `attachmentStore` writable: `{ items: Attachment[], loading: boolean, error: string|null }`
  - Actions: `load(sessionId, scope?, scopeRef?)`, `add(attachment)`, `remove(id)`, `clear()`
  - Derived: `bySessionScope`, `byIterationScope(n)`, `byAgentScope(agentId)`
- `frontend/src/lib/components/AttachmentMenu.svelte` — **new** (the `+` button popover, ChatGPT-style):
  - Props: `sessionId: string`, `scope: AttachmentScope`, `scopeRef?: string`, `disabled?: boolean`
  - Renders a circular `+` button (`.btn-icon`) → opens popover with four menu items, each invoking the same `AttachmentUploadModal` with a preset kind:
    - "Add files" (📎) → kind=file
    - "Add image" (🖼) → kind=image
    - "Add URL" (🌐) → kind=url
    - "Paste text" (✏) → kind=text
  - Keyboard shortcut: `⌘U` opens the file picker directly (mirrors ChatGPT UX)
  - Closes on outside-click; ARIA: `role="menu"`, focus trap, `aria-expanded`
- `frontend/src/lib/components/AttachmentUploadModal.svelte` — **new**:
  - Props: `sessionId`, `scope`, `scopeRef?`, `kind: AttachmentKind`, `onClose: () => void`
  - Per-kind input rendering:
    - `file` → `<input type="file" accept=".pdf,.docx,.md,.txt,.json">`; drag-and-drop zone; live size validation against client-side limit (matches `GetAttachmentMaxBytes`)
    - `image` → `<input type="file" accept="image/png,image/jpeg,image/webp">`; thumbnail preview
    - `url` → URL input with HTTP/HTTPS validation; optional display name override
    - `text` → required display name + multiline textarea (max-height with scroll); character count
  - Submit calls matching `api.upload*Attachment`; on success: `attachmentStore.add(result)`, `onClose()`; on failure: inline error with status-code-aware message (413 → "File too large", 422 → "Could not extract text from this file", 415 → "Unsupported file type")
  - Loading state: button disabled, spinner inline
- `frontend/src/lib/components/AttachmentList.svelte` — **new**:
  - Props: `attachments: Attachment[]`, `onDelete?: (id) => void`, `compact?: boolean`
  - Renders horizontal chip row (compact) or vertical list (full): each chip shows kind icon + display_name (truncated 30 chars) + byte_size formatted + delete-X button
  - Hovering a chip shows a tooltip with `summary` text
  - File/image chips link to `blob_url` (new tab); URL chips link to `source_url`
- `frontend/src/routes/+page.svelte` — **modify** (home page, session creation):
  - Below the idea textarea, render `<AttachmentMenu sessionId={null} scope="session" />` — but since session does not exist yet, store pending uploads in a local array and POST them after `createSession` returns the ID, in a single `await Promise.all` batch
  - Below the menu, render `<AttachmentList attachments={pendingAttachments} compact />`
  - Visually anchored to the input frame (matches the ChatGPT mockup the user referenced)
- `frontend/src/routes/session/[id]/+page.svelte` — **modify**:
  - Above the "Run Next Iteration" button, render `<AttachmentMenu sessionId={id} scope="iteration" scopeRef={String(nextIteration)} />` with a small caption "Add context for next iteration only (auto-removed after this pass)"
  - Render `<AttachmentList attachments={$attachmentStore.byIterationScope(nextIteration)} compact />`
  - Persistent session-scope list rendered in a collapsible sidebar block titled "Session context" — uses `<AttachmentList attachments={$attachmentStore.bySessionScope} />`
- `frontend/src/lib/components/PipelineStage.svelte` — **modify**:
  - In the agent header (next to the Run / Apply buttons added in Task 30), mount `<AttachmentMenu sessionId={sessionId} scope="agent" scopeRef={agent.id} />` with tooltip "Add context for this agent only"
  - Below the stage body, render `<AttachmentList attachments={$attachmentStore.byAgentScope(agent.id)} compact />`
- `frontend/src/app.css` — **modify**: add design-token classes `.btn-icon`, `.attachment-chip`, `.attachment-chip-file`, `.attachment-chip-image`, `.attachment-chip-url`, `.attachment-chip-text` using existing CSS custom properties (`var(--accent)`, `var(--surface)`, etc.); no hard-coded hex values per AGENTS.md
- `frontend/src/lib/services/api.test.ts` — **modify**: add unit tests for the five new `api.ts` functions using `vi.fn()` for `fetch`

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm lint`: zero warnings
- `cd frontend && pnpm test`: existing + 5 new API tests pass
- `cd frontend && pnpm build`: clean production build
- Manual:
  - Home page: click `+` → menu opens with four items; "Paste text" opens modal with textarea; submit before session exists → upload deferred until session ID known; both upload after `createSession`
  - Session page: between iterations, attach an URL at iteration scope → visible in iteration list, runs the next iteration, then auto-disappears (matches Task 40 lifecycle cleanup)
  - PipelineStage: attach a text snippet to agent A → agent B's stage shows no chips; agent A's stage shows the snippet
  - Keyboard: `⌘U` while focused in the home page opens file picker directly
  - Large file (> 10 MB) → modal shows "File too large" before sending
  - PDF upload → after 1-2s shows summary tooltip on the chip; deletion removes both the chip and the blob (verified by re-listing attachments)

**Prompt context needed:** §8.28 (attachment kinds + scope semantics + display contract), §8.16 (design system CSS classes), §8.9 (Svelte store conventions), Task 40 (REST API contract), Task 30 (PipelineStage agent-header layout reference), `frontend/mockups/future-polished-mockup.html` (visual reference for the `+` menu)

---

### Task 43 — DB: MCP Server Registry Schema

**Goal:** Create the two database migrations that introduce the `mcp_servers` table (MCP server registry) and the `agent_mcp_servers` join table (agent ↔ MCP server many-to-many). No Go or frontend code changes in this task — schema only.

**Files to create:**

- `migrations/011_mcp_servers.sql` — see §8.24 for exact DDL:
  - `mcp_servers` table: `id UUID PK DEFAULT gen_random_uuid()`, `name TEXT UNIQUE NOT NULL`, `description TEXT NOT NULL DEFAULT ''`, `transport TEXT NOT NULL CHECK (transport IN ('stdio','http'))`, `command TEXT`, `url TEXT`, `env_refs JSONB NOT NULL DEFAULT '{}'`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
  - Transport consistency check constraint: `(transport = 'stdio' AND command IS NOT NULL AND url IS NULL) OR (transport = 'http' AND url IS NOT NULL AND command IS NULL)` — exactly one of command/url must be set per transport type
- `migrations/012_agent_mcp_servers.sql`:
  - `agent_mcp_servers` join table: `agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE`, `mcp_server_id UUID NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE`, `position INT NOT NULL DEFAULT 0`, `PRIMARY KEY (agent_id, mcp_server_id)`
  - Index: `CREATE INDEX idx_agent_mcp_servers_agent_id ON agent_mcp_servers (agent_id)`

**Validation:**

- `docker compose run --rm backend ./migrate up` — both migrations apply cleanly; no constraint errors
- `docker compose run --rm backend ./migrate down` — rolls back both migrations cleanly; schema identical to post-005 state
- `psql -c "\d mcp_servers"` — shows all 8 columns with correct types and constraints
- `psql -c "\d agent_mcp_servers"` — shows composite PK and both FK constraints with `ON DELETE CASCADE`

**Prompt context needed:** §8.24 (MCP server registry schema, new in this task), migration rules in AGENTS.md (append-only, numbered sequentially)

---

### Task 44 — Backend: MCP Server Module (CRUD)

**Goal:** Implement the full vertical slice for the MCP server registry — `model.go`, `repository.go`, `service.go`, `handler.go`. Expose five REST endpoints (`GET /mcp-servers`, `POST /mcp-servers`, `GET /mcp-servers/{id}`, `PUT /mcp-servers/{id}`, `DELETE /mcp-servers/{id}`). Validation enforces transport-type field consistency and rejects raw secret values in `env_refs`.

**Files to create / modify:**

- `backend/internal/modules/mcpserver/model.go` — see §8.24:
  - `MCPServer` struct: `ID uuid.UUID`, `Name string`, `Description string`, `Transport string`, `Command *string`, `URL *string`, `EnvRefs map[string]string`, `CreatedAt time.Time`
  - `CreateMCPServerInput` struct: same fields minus `ID` and `CreatedAt`; `Transport` required
  - `UpdateMCPServerInput` struct: same as `CreateMCPServerInput` (full replace semantics)
- `backend/internal/modules/mcpserver/repository.go`:
  - `Create(ctx, input) (MCPServer, error)` — INSERT with `ON CONFLICT (name) DO NOTHING`; return 409 if name already taken
  - `GetByID(ctx, id) (MCPServer, error)` — SELECT by PK; return 404 error if not found
  - `List(ctx) ([]MCPServer, error)` — SELECT all, ORDER BY name ASC (deterministic)
  - `Update(ctx, id, input) (MCPServer, error)` — UPDATE; return 404 if row missing
  - `Delete(ctx, id) error` — DELETE; cascade removes `agent_mcp_servers` rows automatically
- `backend/internal/modules/mcpserver/service.go`:
  - `validateInput(input) error`: transport must be `"stdio"` or `"http"`; `"stdio"` requires non-nil non-empty `Command` and nil `URL`; `"http"` requires non-nil non-empty `URL` and nil `Command`; `EnvRefs` values must not contain `=` or whitespace (guards against raw secret values being stored)
  - `Create`, `GetByID`, `List`, `Update`, `Delete` — call repository; `Create` and `Update` call `validateInput` first
- `backend/internal/modules/mcpserver/handler.go`:
  - `GET    /mcp-servers` → 200 + `[]MCPServer`
  - `POST   /mcp-servers` → 201 + `MCPServer`; 400 on validation error; 409 on name conflict
  - `GET    /mcp-servers/{id}` → 200 + `MCPServer`; 404 if not found
  - `PUT    /mcp-servers/{id}` → 200 + `MCPServer`; 400 on validation error; 404 if not found
  - `DELETE /mcp-servers/{id}` → 204; 404 if not found
- `backend/internal/platform/http/router.go` — register all 5 new routes under `mcpserver.NewHandler`

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- `go test ./backend/internal/modules/mcpserver/...`: covers Create/List/Get/Update/Delete happy paths, name-conflict 409, invalid-transport 400, env-ref-with-value 400 (e.g. `"BRAVE_API_KEY": "sk-1234"` → rejected)
- Manual: `curl -X POST http://localhost:8080/mcp-servers -d '{"name":"brave-search","transport":"stdio","command":"npx -y @modelcontextprotocol/server-brave-search","env_refs":{"BRAVE_API_KEY":"BRAVE_API_KEY"}}'` → 201

**Prompt context needed:** §8.24 (MCP server registry schema + env_refs security rule), AGENTS.md module boundary rules (vertical slice pattern), security invariant §5 (env_refs values are var names only)

---

### Task 45 — Backend: Agent–MCP Association + Payload Extension

**Goal:** Extend the agent module to load and persist `agent_mcp_servers` join rows. Extend `BrainstormPayload` with `MCPServers []MCPServerRef` so the iteration engine includes each agent's configured MCP servers in the dispatch payload, giving the agent binary the connection details it needs to dial those servers at runtime.

**Files to modify:**

- `backend/internal/modules/agent/model.go`:
  - Add `MCPServers []MCPServerRef` field to the `Agent` response struct (populated in `GetWithDetails`)
  - Add `MCPServerIDs []uuid.UUID` to `UpdateAgentInput` (optional; nil = leave associations unchanged, empty slice = clear all)
- `backend/internal/modules/agent/repository.go`:
  - `GetWithDetails(ctx, id) (Agent, error)` — extend the existing query to LEFT JOIN `agent_mcp_servers` + `mcp_servers`; aggregate into `Agent.MCPServers`
  - Add `SetMCPServers(ctx, tx pgx.Tx, agentID uuid.UUID, serverIDs []uuid.UUID) error` — DELETE existing rows for agent, INSERT new ones in the same transaction; `ON CONFLICT DO NOTHING`
  - Add `GetMCPServersForAgent(ctx, agentID uuid.UUID) ([]MCPServerRef, error)` — SELECT join for iteration engine use
- `backend/internal/modules/agent/service.go`:
  - `Update(ctx, id, input) (Agent, error)` — if `input.MCPServerIDs != nil`, validate all IDs exist in `mcp_servers`, then call `SetMCPServers` within the same transaction as the agent UPDATE
  - `GetMCPServersForAgent(ctx, agentID) ([]MCPServerRef, error)` — delegates to repository
- `backend/internal/modules/agent/handler.go`:
  - `PUT /agents/{id}` — accept optional `mcp_server_ids []string` in request body; validate UUID format; pass to `UpdateAgentInput`
  - `GET /agents/{id}` — response now includes `mcp_servers []MCPServerRef` array
- `backend/internal/platform/a2a/types.go`:
  - Add `MCPServerRef` struct (see §8.24 for full field list)
  - Extend `BrainstormPayload`: add `MCPServers []MCPServerRef \`json:"mcp_servers,omitempty"\``
- `agent/internal/executor/executor.go` (agent binary — mirror type manually, do not import backend packages):
  - Add `MCPServerRef` struct matching the backend type (copy, not import)
  - Extend `BrainstormPayload` with `MCPServers []MCPServerRef`
- `backend/internal/modules/iteration/engine.go`:
  - Before dispatching each agent, call `agentSvc.GetMCPServersForAgent(ctx, agentID)` → set `payload.MCPServers`
  - `EnvRefs` values pass through verbatim from DB (they are env var names); the agent binary resolves actual values at runtime

**Validation:**

- `cd backend && go build ./...`: zero errors
- `cd backend && go vet ./...`: zero issues
- `go test ./backend/internal/modules/agent/...`: `SetMCPServers` transaction test (assign 2 servers, update to 1 — first server removed; update to nil — associations unchanged)
- `go test ./backend/internal/modules/iteration/...`: mock `agentSvc.GetMCPServersForAgent` returns 1 server → dispatch payload includes 1 `MCPServerRef`; agent with no assignments → `MCPServers` is empty slice (not nil)

**Prompt context needed:** §8.24 (MCPServerRef wire format + security rules), §8.3 (BrainstormPayload contract), Task 44 (MCPServer model + service), Task 7 (agent repository patterns), Task 9 (iteration engine dispatch path)

---

### Task 46 — Agent: MCP Client Package

**Goal:** Build `agent/internal/mcp/` — the package that dials MCP servers, lists their tools, and executes tool calls. Two transports: `stdio` (spawn subprocess, JSON-RPC 2.0 over stdin/stdout) and `http` (POST JSON-RPC 2.0 to a URL). `MCPPool` fans out across all servers assigned to an agent, deduplicates tools, and routes `Call` to the correct server.

**Files to create:**

- `agent/internal/mcp/types.go` — see §8.25:
  - `ToolDef` struct: `Name string`, `Description string`, `InputSchema json.RawMessage`
  - `ToolCall` struct: `ID string`, `Name string`, `Arguments json.RawMessage`
  - `ToolResult` struct: `CallID string`, `Content string`, `IsError bool`
  - `ServerRef` struct: mirrors `executor.MCPServerRef` (copied manually — agent binary must not import backend packages)
- `agent/internal/mcp/client.go`:
  - `MCPClient` interface: `ListTools(ctx context.Context) ([]ToolDef, error)`, `Call(ctx context.Context, toolName string, args json.RawMessage) (ToolResult, error)`, `Close() error`
  - `NewStdioClient(ctx context.Context, serverRef ServerRef, resolveEnv func(string) (string, error)) (MCPClient, error)`:
    - Resolves all `serverRef.EnvRefs` values via `resolveEnv` before setting subprocess env vars; returns error if any env var is absent (no silent fallback)
    - Spawns subprocess with the command from `serverRef.Command`; communicates via stdin/stdout using newline-delimited JSON-RPC 2.0
    - Sends `initialize` handshake (see §8.25) and validates server response before returning
    - Security: `resolveEnv` must be `config.GetLLMAPIKey` — keeps `os.Getenv` confined to `config/config.go`
  - `NewHTTPClient(ctx context.Context, serverRef ServerRef, httpClient *http.Client, resolveEnv func(string) (string, error)) (MCPClient, error)`:
    - Sends JSON-RPC 2.0 messages as HTTP POST to `serverRef.URL`
    - Resolves any auth headers from `EnvRefs` via `resolveEnv`
    - Uses default 30s timeout client if `httpClient` is nil
  - Both clients: `ListTools` sends `{"method":"tools/list"}`; `Call` sends `{"method":"tools/call","params":{"name":...,"arguments":...}}`; see §8.25 for full request/response shapes
- `agent/internal/mcp/pool.go`:
  - `MCPPool` struct: holds map of server name → `MCPClient`; maps tool name → server name (for routing)
  - `NewPool(ctx context.Context, servers []ServerRef, resolveEnv func(string)(string,error), httpClient *http.Client) (*MCPPool, error)` — dials all servers in parallel (goroutines + `errgroup`); fails fast if any dial fails; closes already-dialed servers on error
  - `ListAllTools(ctx context.Context) ([]ToolDef, error)` — calls `ListTools` on all clients, deduplicates by `Name` (last-server-wins in `servers` order), returns sorted by name (determinism guarantee)
  - `Call(ctx context.Context, toolName string, args json.RawMessage) (ToolResult, error)` — looks up which server owns `toolName`; routes call; returns error if tool not found
  - `Close() error` — closes all clients; idempotent
- `agent/internal/mcp/client_test.go`:
  - HTTP transport test: `httptest.NewServer` mock implementing `tools/list` (returns 2 tools) + `tools/call` (returns text result); assert `ListTools` returns correct `[]ToolDef`; assert `Call` sends correct JSON-RPC body and returns `ToolResult`
  - Pool test: 2-server pool each with 2 unique tools; `ListAllTools` returns 4 tools sorted alphabetically; `Call("tool-from-server-2", ...)` routes to server 2 only
  - Env resolution test: `EnvRefs` entry references an env var not in the environment → `NewStdioClient` returns error; `NewPool` fails fast and closes already-dialed servers

**Validation:**

- `cd agent && go build ./...`: zero errors
- `cd agent && go vet ./...`: zero issues
- `cd agent && go test ./internal/mcp/...`: all tests pass (no real MCP server required)

**Prompt context needed:** §8.25 (JSON-RPC 2.0 MCP protocol — initialize handshake, tools/list, tools/call message shapes for both transports), §8.12 (credential security — resolveEnv = config.GetLLMAPIKey), §8.24 (ServerRef / MCPServerRef shape)

---

### Task 47 — Agent: LLM Tool-Use Interface + Executor Loop

**Goal:** Add `GenerateWithTools` to the `LLMProvider` interface and implement it in `CopilotProvider` and `OpenCodeProvider` using OpenAI function-calling format. Replace the single-shot `llm.Generate` call in `BrainstormExecutor.Execute` with a configurable multi-turn tool-use loop: build MCP pool → list tools → call LLM with tools → execute any tool calls via pool → re-call LLM with results → repeat until no tool calls or max rounds reached.

**Files to modify:**

- `agent/internal/llm/copilot.go`:
  - Extend `LLMProvider` interface (defined here): add `GenerateWithTools(ctx context.Context, req LLMRequest, tools []ToolDef) (LLMResponseWithTools, error)`
  - Add types to this package (do not import `agent/internal/mcp` — copy only what's needed):
    - `ToolDef` struct: `Name string`, `Description string`, `InputSchema json.RawMessage`
    - `ToolCallRequest` struct: `ID string`, `Name string`, `Arguments json.RawMessage`
    - `LLMResponseWithTools` struct: `Content string`, `FinishReason string`, `TokensUsed int`, `ToolCalls []ToolCallRequest`
    - `Message` struct: `Role string`, `Content string`, `ToolCalls []ToolCallRequest` (optional), `ToolCallID string` (optional, for role `"tool"`)
  - `CopilotProvider.GenerateWithTools`: converts `[]ToolDef` to OpenAI `tools` array format (`[{"type":"function","function":{"name":...,"description":...,"parameters":<InputSchema>}}]`); sends to Copilot API with full message history; parses `tool_calls` from response when `finish_reason == "tool_calls"`; see §8.25 for wire format
  - `Generate` remains: calls `GenerateWithTools(ctx, req, nil)` internally — no callers broken, zero regression
- `agent/internal/llm/opencode.go`:
  - Implement `GenerateWithTools` — same interface; encodes tools as JSON and appends to system prompt (OpenCode adaptation since it does not have a native function-calling field); parses tool calls from LLM response text using the agreed format in §8.25
- `agent/internal/executor/executor.go`:
  - Replace the single `e.llm.Generate(ctx, req)` call with the multi-turn tool-use loop (see §8.25 for the canonical algorithm):
    1. Build `MCPPool` from `payload.MCPServers` (skip if empty — zero regression path)
    2. `ListAllTools` from pool → `tools []llm.ToolDef`
    3. Initialize `messages []llm.Message` with system + user message
    4. Loop `for round := 0; round < maxRounds; round++`: call `GenerateWithTools(ctx, req, tools)` with message history; if `len(resp.ToolCalls) == 0` → `finalContent = resp.Content; break`; append assistant tool-call message; execute each call via pool; append `"tool"` result messages; continue loop
    5. After loop: if `finalContent == ""` and loop exhausted, log warning and use last `resp.Content`
    6. Parse `finalContent` as `CanonicalState` JSON (existing logic unchanged)
  - When `payload.MCPServers` is empty: pool is nil, `tools` is empty slice, loop runs exactly once — behaviour identical to current implementation
- `agent/internal/config/config.go`:
  - Add `GetMCPMaxToolRounds() int` — reads `AGENT_MCP_MAX_TOOL_ROUNDS`; default `5`; clamp to range `[1, 20]`
- `agent/internal/executor/executor_test.go`:
  - Add test: mock `GenerateWithTools` returns 1 tool call on round 1 (`finish_reason: "tool_calls"`), then final JSON on round 2 (`finish_reason: "stop"`); assert final `CanonicalState` parsed correctly; assert pool `Call` invoked once
  - Add test: max rounds exhausted (mock always returns tool calls) → executor uses last `resp.Content` and emits `Completed` event (no panic)
  - Add test: `payload.MCPServers` empty → loop runs once with `tools = nil`; `GenerateWithTools` called with nil tools; output identical to existing `Generate` tests

**Validation:**

- `cd agent && go build ./...`: zero errors
- `cd agent && go vet ./...`: zero issues
- `cd agent && go test ./internal/...`: all existing tests still pass; 3 new tool-use tests pass
- Smoke: start agent binary with `AGENT_MCP_MAX_TOOL_ROUNDS=3`; no panic; config getter returns 3

**Prompt context needed:** §8.25 (tool-use loop algorithm + `GenerateWithTools` OpenAI wire format + OpenCode adaptation + message history threading), §8.2 (LLMProvider interface), §8.24 (MCPServerRef), Task 46 (MCPPool API), §8.12 (credential security)

---

### Task 48 — Frontend: MCP Server Settings + Agent Assignment + Smart Import

**Goal:** Add the "MCP Servers" tab to `/settings`, build new/edit forms for MCP servers with a "Test Connection" flow, implement the smart JSON config import modal that normalises Claude Desktop / VS Code / Cursor / Zed / Windsurf / canonical JSON formats, and extend the agent edit form with an MCP server multi-select section.

**Files to create / modify:**

- `frontend/src/lib/types.ts` — add:
  - `MCPServer` interface: `id: string`, `name: string`, `description: string`, `transport: 'stdio' | 'http'`, `command?: string`, `url?: string`, `env_refs: Record<string, string>`, `created_at: string`
  - `CreateMCPServerInput` interface: same without `id` and `created_at`
  - `TestConnectionResult` interface: `connected: boolean`, `tool_count: number`, `tool_names: string[]`, `error?: string`
  - Extend `Agent` with `mcp_servers: MCPServer[]`
- `frontend/src/lib/services/api.ts` — add:
  - `listMCPServers(): Promise<MCPServer[]>`
  - `createMCPServer(input: CreateMCPServerInput): Promise<MCPServer>`
  - `getMCPServer(id: string): Promise<MCPServer>`
  - `updateMCPServer(id: string, input: CreateMCPServerInput): Promise<MCPServer>`
  - `deleteMCPServer(id: string): Promise<void>`
  - `testMCPConnection(id: string): Promise<TestConnectionResult>` → `POST /mcp-servers/{id}/test` (backend adds this lightweight endpoint in the handler that dials the server and returns the tool list)
- `frontend/src/routes/settings/mcp/new/+page.svelte` — **new**:
  - Form fields: Name (text), Description (text), Transport radio toggle (`stdio` | `http`)
  - Conditional: when `stdio` → Command textarea (placeholder `npx -y @modelcontextprotocol/server-brave-search`); when `http` → URL input (placeholder `http://localhost:3100`)
  - Env Refs section: dynamic key-value pair list; both columns labeled "Env Var Name" (not key/value) to reinforce that raw secrets are never stored; add/remove row buttons
  - "Test Connection" button (`.btn-ghost`): saves server first, then calls `testMCPConnection`; shows result inline — `.chip-ok` with tool count on success, `.chip-danger` with error on failure
  - On submit: `createMCPServer(input)` → navigate to `/settings?tab=mcp`
- `frontend/src/routes/settings/mcp/[id]/+page.svelte` — **new**:
  - Pre-populated edit form; `updateMCPServer` on submit; delete with `WarningModal`
  - Inline "Test Connection" — same as new form; shows current tool list when connected
- `frontend/src/routes/settings/+page.svelte` — **modify**:
  - Add "MCP Servers" as fourth tab (after Agents, Skills, Roles)
  - MCP Servers tab content: table rows — Name, Transport badge (`.badge-build` for `stdio`, `.badge-review` for `http`), Command/URL (truncated 60 chars), Edit → `/settings/mcp/{id}`, Delete (WarningModal)
  - "Import from Config" button (`.btn-ghost`) → opens smart import modal (inline component below)
  - "Add Manually" button (`.btn-primary`) → navigates to `/settings/mcp/new`
  - Smart import modal (inline in this tab):
    - Textarea labeled "Paste your MCP config JSON"; "Parse" button → calls `parseMCPConfig(raw: string): ParsedMCPServer[]` normaliser (see §8.25 for logic)
    - Preview table: one row per detected server — Name, Transport, Command/URL; checkbox per row (all pre-checked)
    - Security warning banner (`.chip-warn`) shown when any env value in the pasted config looks like a raw secret (contains non-alphanumeric/underscore chars): "⚠ API key values were stripped from env fields. Set the shown env var names on the machine running the agent binary."
    - "Import Selected" button — calls `createMCPServer` for each checked row sequentially; shows per-row progress chip (`pending → importing → done / error`); errors shown inline, import continues for remaining rows
- `frontend/src/routes/settings/agent/[id]/+page.svelte` — **modify**:
  - Add "MCP Servers" section after the existing "Skills" section
  - Renders a checkbox list of all registered MCP servers (loaded from store); pre-checked = agent's current `mcp_servers` array
  - Each row shows: server name, transport badge, tool count (if last test result cached, else "–")
  - On submit: include `mcp_server_ids: string[]` in the `updateAgent` payload

**Validation:**

- `cd frontend && pnpm check`: zero svelte-check errors
- `cd frontend && pnpm build`: clean production build
- `cd frontend && pnpm test`: existing API service tests still pass
- Smart import: paste Claude Desktop config JSON → preview shows correct server names; any env value with non-identifier chars is stripped and warning shown; "Import Selected" creates all servers via API
- Agent edit: assign 2 MCP servers → save → reload → both remain checked
- Test Connection: mock backend returns 3 tools → chip shows "3 tools"; mock error → chip shows error message

**Prompt context needed:** §8.24 (MCPServer model + env_refs rule), §8.26 (smart import normaliser — supported config formats, stripping policy), §8.16 (design system CSS classes), Task 44 (MCP server REST endpoints), Task 45 (agent `mcp_server_ids` field), Task 21 (agent form patterns)

---

## 6. Task Summary

| Task | Name                                          | Key Files                                                                                                                                                                                                                      | Depends On             | Complexity |
| ---- | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------- | ---------- |
| 1    | Project Scaffold                              | `go.work`, `go.mod` ×2, `docker-compose.yml`, `Makefile`, FE scaffold                                                                                                                                                          | —                      | Low        |
| 2    | Platform: Config + DB + Logger                | `platform/config/`, `platform/db/`, `platform/logger/`                                                                                                                                                                         | Task 1                 | Low        |
| 3    | Platform: LLM Abstraction                     | `platform/llm/provider.go`, `resolver.go`, `copilot.go`                                                                                                                                                                        | Task 2                 | Medium     |
| 4    | Platform: A2A Layer                           | `platform/a2a/client.go`, `types.go`, `agent/internal/config/`                                                                                                                                                                 | Task 2                 | Medium     |
| 5    | State Module                                  | `modules/state/model.go`, `merge.go`, `validator.go`                                                                                                                                                                           | Tasks 3, 4             | Medium     |
| 6    | Agent Module: Models + DB Schema              | `modules/agent/model.go`, `repository.go`, `role.go`, `001_agents.sql`                                                                                                                                                         | Tasks 1, 5             | Medium     |
| 7    | Agent Module: Service + Handler + Dispatch    | `modules/agent/service.go`, `handler.go`, `client.go`                                                                                                                                                                          | Tasks 6, 3, 4          | High       |
| 8    | Session Module                                | `modules/session/*`, `003_sessions.sql`                                                                                                                                                                                        | Task 7                 | Medium     |
| 9    | Iteration Engine + Convergence                | `iteration/engine.go`, `convergence/engine.go`                                                                                                                                                                                 | Tasks 5, 7, 8          | High       |
| 10   | Markdown + Backend Wire-up                    | `markdown/generator.go`, `cmd/server/main.go`, `platform/http/router.go`                                                                                                                                                       | Tasks 9, 8             | Medium     |
| 11   | Agent Service Binary                          | `agent/agentcard.go`, `executor/executor.go`, `agent/cmd/server/main.go`                                                                                                                                                       | Tasks 3, 4             | High       |
| 12   | Frontend: Scaffold + Stores + API Client      | `lib/types.ts`, `stores/*.ts`, `services/api.ts`                                                                                                                                                                               | Task 1                 | Medium     |
| 13   | Frontend: Session Workspace                   | `AgentPanel.svelte`, `ControlPanel.svelte`, `StateView.svelte`, `Timeline.svelte`                                                                                                                                              | Task 12                | Medium     |
| 14   | Frontend: Agent Registry + Skills             | `AgentSelector.svelte`, `SkillManager.svelte`, routes                                                                                                                                                                          | Task 12                | Medium     |
| 15   | Integration Tests + Docs                      | `*_test.go` files, `README.md`                                                                                                                                                                                                 | Tasks 11, 13, 14       | Medium     |
| 16   | Frontend: Design System Foundation            | `app.css`, `+layout.svelte`, `tailwind.config.ts`                                                                                                                                                                              | Task 12                | Low        |
| 17   | Frontend: Home View Redesign                  | `routes/+page.svelte`, `AgentSelector.svelte`                                                                                                                                                                                  | Task 16                | Medium     |
| 18   | Frontend: Session View + Pipeline Components  | `session/[id]/+page.svelte`, `PipelineStage.svelte`, `ConfidenceBar.svelte`, `RiskBoard.svelte`, `CanonicalStatePanel.svelte`                                                                                                  | Tasks 16, 17           | High       |
| 19   | Backend: Session List + Artifact Content      | `session/model.go`, `repository.go`, `service.go`, `handler.go`, `markdown/generator.go`, `api.ts`, `types.ts`                                                                                                                 | Tasks 10, 12           | Medium     |
| 20   | Frontend: Settings View (Agents+Skills Tabs)  | `routes/settings/+page.svelte`, redirect `/agents`, redirect `/skills`                                                                                                                                                         | Tasks 16, 19           | Medium     |
| 21   | Frontend: Agent Form + Skill Form Views       | `settings/agent/new`, `settings/agent/[id]`, `settings/skill/new`, `settings/skill/[id]`                                                                                                                                       | Task 20                | Medium     |
| 22   | Frontend: Roles Tab + Warning Modal           | `WarningModal.svelte`, `uiStore.ts`, settings Roles tab                                                                                                                                                                        | Task 20                | Medium     |
| 23   | Frontend: Session History View                | `routes/history/+page.svelte`                                                                                                                                                                                                  | Tasks 16, 19           | Medium     |
| 24   | Frontend: Finalize/Export View                | `routes/session/[id]/finalize/+page.svelte`                                                                                                                                                                                    | Tasks 19, 22           | Medium     |
| 25   | Frontend: Navigation Wiring + Final UI Val    | `+layout.svelte`, `api.test.ts`, `README.md`                                                                                                                                                                                   | Tasks 16–24            | Medium     |
| 26   | Agent: OpenCode LLM Provider                  | `agent/internal/llm/opencode.go`, `opencode_test.go`, `agent/internal/config/config.go` (modified), `agent/cmd/server/main.go` (modified)                                                                                      | Task 11 (agent binary) | Medium     |
| 27   | Infrastructure: OpenCode Service Wiring       | `docker-compose.yml`, `.env.example`, `Makefile`, `docs/STARTUP_GUIDE.md`                                                                                                                                                      | Task 26                | Low        |
| 28   | Backend + FE: Selectable Output Documents     | `005_session_output_docs.sql`, `session/*` (modified), `markdown/generator.go`, `+page.svelte`, `finalize/+page.svelte`                                                                                                        | Tasks 10, 19, 24       | Medium     |
| 29   | Long-form Generators for All Output Docs      | `markdown/templates.go`, `generator_architecture.go`, `generator_roadmap.go`, `generator_plan.go`, `generator_readme.go`, paired `_test.go`, registry test                                                                     | Task 28                | High       |
| 30   | Per-Agent Preview/Apply (Backend + Frontend)  | `iteration/preview.go`, `engine.go` (modified), `service.go`, `handler.go`, `PipelineStage.svelte`, `api.ts`                                                                                                                   | Tasks 9, 18, 19        | High       |
| 31   | SSE Real-time Agent Progress                  | `platform/sse/broadcaster.go`, `iteration/events.go`, `engine.go` + `handler.go` (modified), `sse.ts`, `session/[id]/+page.svelte`                                                                                             | Tasks 9, 18, 30        | High       |
| 32   | Generated Document Quality Overhaul           | `markdown/templates.go`, `generator.go`, `generator_architecture.go`, `generator_roadmap.go`, `generator_plan.go`, `generator_readme.go`, `state/model.go`, `session/service.go`, `session/handler.go`, `executor/executor.go` | Tasks 28, 29           | High       |
| 33   | AI-Driven Hybrid Doc Generator + Skill Bundle | `markdown/aigen/skills.go`, `rubric.go`, `generator.go`, `aigen_test.go`, `markdown/generator.go` (modified), `session/service.go` (modified), `platform/config/config.go` (modified), `cmd/server/main.go` (modified)         | Task 32                | High       |
| 34   | Guided Onboarding v2 + Plan + Artifacts       | `007_session_discovery.sql`, `008_session_artifacts.sql`, `TechConstraints.svelte`, `DiscoveryClarify.svelte`, `discovery_compiler.go`, `generator_plan.go`, `finalize/+page.svelte`                                           | Task 33                | High       |
| 35   | Architecture.md: State Enricher + Section Generator | `markdown/generator_architecture.go` (rewrite), `markdown/templates.go` (modified), `markdown/aigen/enricher.go` (new), `markdown/aigen/enricher_test.go` (new), `markdown/aigen/generator.go` (modified), `markdown/aigen/rubric.go` (modified), `platform/config/config.go` (modified), `generator_architecture_test.go` (rewrite) | Task 34                | High       |
| 36   | PLAN.md: Generation Quality Overhaul          | `markdown/generator_plan.go` (rewrite), `markdown/aigen/plan_enricher.go` (new), `markdown/aigen/plan_enricher_test.go` (new), `markdown/aigen/generator.go` (modified), `markdown/aigen/rubric.go` (modified), `platform/config/config.go` (modified), `generator_plan_test.go` (new), `agent/internal/executor/prompts/plan_format.go` (new) | Task 35                | High       |
| 37   | README.md: Generation Quality Overhaul        | `markdown/generator_readme.go` (rewrite), `markdown/aigen/readme_enricher.go` (new), `markdown/aigen/readme_enricher_test.go` (new), `markdown/aigen/generator.go` (modified), `markdown/aigen/rubric.go` (modified), `platform/config/config.go` (modified), `generator_readme_test.go` (new), `agent/internal/executor/prompts/readme_format.go` (new) | Task 36                | High       |
| 38   | DB: Attachments + Chunks Schema (pgvector)    | `migrations/009_attachments.sql`, `migrations/010_attachment_chunks.sql`                                                                                                                                                       | Task 34                | Low        |
| 39   | Platform: Extractor + Embeddings + Blobstore  | `platform/extractor/*`, `platform/embeddings/*`, `platform/blobstore/*`, `platform/config/config.go` (modified), `docker-compose.yml` (modified)                                                                               | Task 38                | High       |
| 40   | Backend: Attachment Module (CRUD + Pipeline)  | `modules/attachment/model.go`, `repository.go`, `service.go`, `handler.go`, `platform/http/router.go` (modified)                                                                                                               | Task 39                | High       |
| 41   | Backend: AttachmentRetriever + Engine Wiring  | `iteration/engine.go` (modified), `agent/client.go` (modified), `platform/a2a/types.go` (modified), `executor/executor.go` (modified), `engine_test.go`, `executor_test.go`                                                    | Tasks 40, 9, 11        | High       |
| 42   | Frontend: Attachment Menu + Upload Modal      | `AttachmentMenu.svelte`, `AttachmentUploadModal.svelte`, `AttachmentList.svelte`, `attachmentStore.ts`, `+page.svelte`, `session/[id]/+page.svelte`, `PipelineStage.svelte` (modified), `api.ts`, `types.ts`, `app.css`        | Tasks 40, 18, 30       | High       |
| 43   | DB: MCP Server Registry Schema                | `migrations/011_mcp_servers.sql`, `migrations/012_agent_mcp_servers.sql`                                                                                                                                                       | Task 38                | Low        |
| 44   | Backend: MCP Server Module (CRUD)             | `modules/mcpserver/model.go`, `repository.go`, `service.go`, `handler.go`, `platform/http/router.go`                                                                                                                           | Task 43                | Medium     |
| 45   | Backend: Agent–MCP Association + Payload      | `modules/agent/model.go`, `repository.go`, `service.go`, `handler.go` (modified), `platform/a2a/types.go`, `iteration/engine.go`, `executor/executor.go`                                                                       | Tasks 44, 9            | Medium     |
| 46   | Agent: MCP Client Package                     | `agent/internal/mcp/types.go`, `client.go`, `pool.go`, `client_test.go`                                                                                                                                                        | Task 43                | High       |
| 47   | Agent: LLM Tool-Use + Executor Loop           | `agent/internal/llm/copilot.go`, `opencode.go` (modified), `executor/executor.go`, `config/config.go`, `executor_test.go`                                                                                                      | Tasks 45, 46           | High       |
| 48   | Frontend: MCP Settings + Smart Import         | `settings/mcp/new/+page.svelte`, `settings/mcp/[id]/+page.svelte`, `settings/+page.svelte`, `settings/agent/[id]/+page.svelte`, `lib/types.ts`, `api.ts`                                                                       | Tasks 44, 45, 21       | High       |

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

---

### 8.18 OpenCode Server API Reference

OpenCode runs a headless HTTP server (default port `4096`) reachable via REST. The `OpenCodeProvider` in `agent/internal/llm/opencode.go` uses three endpoints.

**Base URL:** configured via `AGENT_OPENCODE_BASE_URL` (default `http://localhost:4096`).

**Authentication:** HTTP Basic auth. The OpenCode server must be started with `OPENCODE_SERVER_PASSWORD` set. The username defaults to `"opencode"` (override with `OPENCODE_SERVER_USERNAME`).

#### Endpoints used by `OpenCodeProvider`

| Method | Path                   | Purpose                                               |
| ------ | ---------------------- | ----------------------------------------------------- |
| `GET`  | `/global/health`       | Liveness — returns `{"healthy":true,"version":"..."}` |
| `POST` | `/session`             | Create a new chat session                             |
| `POST` | `/session/:id/message` | Send a message and block until the AI responds        |

---

#### `POST /session` — Create session

**Request body:**

```json
{ "title": "brainstorm" }
```

**Response (relevant fields):**

```json
{ "id": "session-uuid-..." }
```

The session ID is stored in memory (`OpenCodeProvider.sessionID`) and reused for all subsequent `Generate` calls within the same process lifetime.

---

#### `POST /session/:id/message` — Send message

**Request headers:**

```
Authorization: Basic base64(username:password)
Content-Type: application/json
```

**Request body:**

```json
{
  "parts": [
    {
      "type": "text",
      "text": "<UserMessage — CanonicalState JSON from LLMRequest.UserMessage>"
    }
  ],
  "model": {
    "providerID": "github",
    "modelID": "gpt-4o"
  },
  "system": "<assembled system prompt from LLMRequest.SystemPrompt>"
}
```

**Model field format:** `providerID/modelID` → split on first `/`:

- GitHub Copilot: `providerID = "github"`, e.g. `modelID = "gpt-4o"` or `"claude-sonnet-4-5"`
- Anthropic: `providerID = "anthropic"`, e.g. `modelID = "claude-opus-4-5"`
- OpenAI: `providerID = "openai"`, e.g. `modelID = "gpt-4o"`

**Response shape:**

```json
{
  "info": {
    "id": "msg-uuid",
    "role": "assistant",
    "sessionID": "session-uuid"
  },
  "parts": [{ "type": "text", "text": "<full LLM response content>" }]
}
```

**Parsing rule:** Iterate `response.parts`; concatenate all parts where `type == "text"` into `LLMResponse.Content`. Ignore non-text parts (tool calls, etc.).

---

#### Session management strategy in `OpenCodeProvider`

- Session is created lazily on first `Generate` call via `ensureSession(ctx)`
- `sync.Once` wraps `ensureSession` so only one session creation attempt is made per process lifetime (thread-safe)
- A new process = new session (no session persistence across agent restarts)
- Sessions accumulate conversation context; if stateless per-call behaviour is required, move `ensureSession` into `Generate` (new session per call) — acceptable trade-off if token cost allows

#### `LLMConfig` row for `provider = "opencode"` (stored in `agents.llm_config` JSONB)

```json
{
  "provider": "opencode",
  "model": "github/gpt-4o",
  "credential_ref": "OPENCODE_SERVER_PASSWORD"
}
```

The `credential_ref` field holds the env var **name** for the OpenCode server password. The actual password value is never stored in the database.

#### OpenCode container auth (GitHub Copilot, one-time)

The OpenCode server must authenticate to the underlying provider (e.g. GitHub Copilot) separately from the agent binary. Steps:

1. `make opencode-up` — start the container
2. `make opencode-auth` — runs `opencode /provider/github/oauth/authorize` inside the container; prints a device flow URL
3. User visits the URL in a browser, authorises the GitHub OAuth app
4. Token is saved inside the container at `/root/.local/share/opencode/` — persisted in the `opencode-auth` Docker volume

The `opencode-auth` Docker volume survives container restarts; re-authentication is only needed if the volume is deleted or the OAuth token expires.

---

### 8.19 Output Document Selection (v1.3)

**Available keys** (extensible registry — see Task 29):

```go
var AllowedOutputDocs = map[string]bool{
    "architecture": true,
    "roadmap":      true,
    "plan":         true,
    "readme":       true,
}
```

**Default:** `["architecture", "roadmap"]` — applied when a `POST /sessions` call omits the field.

**DB schema (migration 005):**

```sql
ALTER TABLE sessions
    ADD COLUMN output_docs TEXT[] NOT NULL DEFAULT ARRAY['architecture','roadmap'];

UPDATE sessions
    SET output_docs = ARRAY['architecture','roadmap']
    WHERE output_docs IS NULL;
```

**Validation rules (service layer, both `Create` and `UpdateOutputDocs`):**

1. `len(docs) >= 1` — must select at least one.
2. Every entry must be in `AllowedOutputDocs` — unknown key → 400.
3. No duplicates — case-sensitive uniqueness check → 400.
4. `UpdateOutputDocs` rejects with 409 when `session.status == "finalized"`.
5. `Finalize(input)` — if `input.OutputDocs != nil`, call `UpdateOutputDocs` first (subject to rule 4), then proceed.

**Response shape from `POST /sessions/{id}/finalize`** (replaces the v1.1 fixed pair):

```json
{
  "session_id": "uuid",
  "documents": {
    "architecture": {
      "filename": "architecture.md",
      "content": "...",
      "line_count": 1247
    },
    "plan": { "filename": "PLAN.md", "content": "...", "line_count": 1083 }
  }
}
```

`GeneratedDocument` struct:

```go
type GeneratedDocument struct {
    Filename  string `json:"filename"`
    Content   string `json:"content"`
    LineCount int    `json:"line_count"`
}
```

---

### 8.20 Long-form Generator Templates — All Output Documents (v1.3)

**All four generators** (`GenerateArchitecture`, `GenerateRoadmap`, `GeneratePlan`, `GenerateReadme`) are pure functions with the same signature: `func(CanonicalState) (string, error)`. Same input must produce byte-identical output (enforced by determinism tests). Every generator wraps its body in `enforceMinLines` so the rendered document is **≥ 1000 lines, per document, individually** — not combined.

**`GenerateArchitecture` section skeleton** — emits in this fixed order:

```
# <Title>                          ← from state.idea
## 1. Overview                     ← state.idea + architecture summary
## 2. System Components            ← per-component deep dive from state.architecture.components
## 3. Data Flow                    ← ASCII diagram + textual walkthrough
## 4. Tech Stack                   ← from state.architecture.tech_stack
## 5. Module Boundaries            ← from state.architecture.modules
## 6. Key Architecture Decisions   ← decisions table from state.architecture.decisions
## 7. Data Model                   ← entities + relationships from state.architecture.data_model
## 8. API Surface                  ← endpoints from state.architecture.api
## 9. Failure Modes                ← from state.risks
## 10. Observability               ← from state.architecture.observability
## 11. Security Model              ← from state.architecture.security
## 12. Open Questions              ← from state.open_questions
```

**`GenerateRoadmap` section skeleton** — emits in this fixed order:

```
# <Title> — Implementation Roadmap
## 1. Goal                         ← from state.idea
## 2. Milestones                   ← grouped from state.execution_plan
## 3. Phase Breakdown              ← one section per phase with deliverables + exit criteria
## 4. Dependencies                 ← cross-phase dependency graph
## 5. Risks & Mitigations          ← from state.risks
## 6. Assumptions                  ← from state.assumptions
## 7. Validation Strategy          ← derived from execution_plan[*].validation
## 8. Rollout Plan                 ← staged delivery from execution_plan
```

**`GeneratePlan` section skeleton** — emits in this fixed order:

```
> Version / Date / Author / Status / Source of Truth
## 1. Goal                          ← from state.idea
## 2. Architecture Overview         ← components diagram + decisions table
## 3. Tech Stack                    ← from state.architecture.tech_stack
## 4. Project Structure             ← from state.architecture.directory_layout
## 5. Implementation Tasks
###   Dependency Graph
###   Task N — <name>               ← one per state.execution_plan[*]
## 6. Task Summary                  ← table
## 7. How to Use This Plan          ← static boilerplate
## 8. Deep Knowledge Reference
###   8.1 Canonical State Shape
###   8.2 ...                       ← derived from architecture + assumptions + risks
```

**`GenerateReadme` section skeleton** — emits in this fixed order:

```
# <Title>
> <one-line description>           ← from state.idea
[badges row — placeholder]
## Table of Contents
## Overview                        ← state.idea + architecture summary
## System Architecture             ← ASCII diagram + per-component description
## Repository Structure            ← directory tree
## Prerequisites
## Quick Start
## Configuration                   ← env var list
## Testing                         ← from execution_plan[*].validation
## Risk & Assumptions
## Roadmap                         ← execution_plan summary
## Documentation
## Contributing
## License
```

**Line-count enforcement (≥ 1000 lines per doc, individually):**

```go
const minDocLines = 1000

func enforceMinLines(body string, state CanonicalState, padFn func(CanonicalState) string) string {
    lines := strings.Count(body, "\n")
    for lines < minDocLines {
        body += "\n\n" + padFn(state)   // deterministic padder
        lines = strings.Count(body, "\n")
    }
    return body
}
```

Padding sources (deterministic, derived only from `state`):

- Per-component deep-dive sub-sections (data flow, failure modes, observability, deployment notes)
- Per-execution-plan-item elaboration (assumptions, risks, mitigations, validation matrix)
- Full canonical state JSON dump in a fenced block (last resort)

**Forbidden:** random/Lorem-ipsum text, timestamps, UUIDs not present in state.

---

### 8.21 Per-Agent Preview / Apply API Contract (v1.3)

**`POST /sessions/{id}/agents/{agent_id}/preview`**

Request body: none.

Response `200`:

```json
{
  "session_id": "uuid",
  "agent_id":   "uuid",
  "preview_id": "uuid",
  "output":     { "...partial CanonicalState delta from this agent only..." },
  "created_at": "2026-05-24T12:00:00Z"
}
```

Response `409` when a full iteration is in flight for the session, or when the agent is not a member of the session.

**`POST /sessions/{id}/agents/{agent_id}/apply`**

Request body: optional `{"preview_id": "uuid"}` — when present, asserts the stored preview ID matches (prevents accidentally applying a stale preview). When absent, applies the latest preview for that agent.

Response `200`: new full `CanonicalState` after merge + iteration counter incremented by 1.

Response `404` when no preview exists. Response `409` during in-flight iteration or `412` when `preview_id` mismatches.

**`DELETE /sessions/{id}/agents/{agent_id}/preview`** → `204`. Idempotent — `204` even when no preview exists.

**In-memory store (`iteration/preview.go`):**

```go
type PreviewResult struct {
    PreviewID uuid.UUID
    AgentID   uuid.UUID
    Output    state.CanonicalState
    CreatedAt time.Time
}

type PreviewStore struct {
    mu sync.RWMutex
    m  map[uuid.UUID]map[uuid.UUID]PreviewResult  // sessionID → agentID → result
}
```

Server restart clears all previews (by design — speculative state, not persisted).

**Concurrency contract:** the iteration service holds a per-session `sync.Mutex`. `Iterate`, `RunSingleAgent` (preview), and `Apply` all acquire it; concurrent attempts return 409 immediately rather than blocking.

---

### 8.22 SSE Real-time Progress Event Schema (v1.3)

**Endpoint:** `GET /sessions/{id}/events`

**Headers (response):**

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
X-Accel-Buffering: no
```

**Headers (request, optional):** `Last-Event-ID: <int>` — when present and within the ring buffer (last 100 events), the server replays missed events before resuming live stream.

**Event types:**

| `event:`             | `data:` payload                                                                                      |
| -------------------- | ---------------------------------------------------------------------------------------------------- |
| `iteration.start`    | `{"iteration": 3, "agents": [{"agent_id":"...","role":"...","position":0}, ...]}`                    |
| `agent.started`      | `{"iteration": 3, "agent_id": "...", "role": "...", "position": 0}`                                  |
| `agent.complete`     | `{"iteration": 3, "agent_id": "...", "partial_state": {...delta only...}, "confidence_delta": 0.07}` |
| `agent.error`        | `{"iteration": 3, "agent_id": "...", "error": "dispatch timeout"}`                                   |
| `iteration.complete` | `{"iteration": 3, "converged": false, "confidence": 0.81}`                                           |
| `session.finalized`  | `{"documents": ["architecture","plan"]}`                                                             |

**Wire example:**

```
id: 42
event: agent.complete
data: {"iteration":3,"agent_id":"7e1c...","partial_state":{"risks":[{"id":"r-12","title":"..."}]},"confidence_delta":0.07}

id: 43
event: iteration.complete
data: {"iteration":3,"converged":false,"confidence":0.81}

```

(Two blank lines terminate each event; `id` is monotonically increasing per session.)

**Broadcaster shape (`platform/sse/broadcaster.go`):**

```go
type Event struct {
    ID   uint64
    Type string
    Data any
}

type Broadcaster struct {
    mu       sync.RWMutex
    subs     map[uuid.UUID]map[uint64]chan Event   // sessionID → subscriberID → channel
    buffers  map[uuid.UUID][]Event                  // sessionID → ring buffer (cap 100)
    nextSub  uint64
    nextEvt  map[uuid.UUID]uint64                   // sessionID → next event ID
}

func (b *Broadcaster) Subscribe(sessionID uuid.UUID, lastEventID uint64) (<-chan Event, func())
func (b *Broadcaster) Publish(sessionID uuid.UUID, evtType string, data any)
```

**Engine hook points (`iteration/engine.go`):**

```go
for i := 1; i <= maxIter; i++ {
    emitter.Publish(sessionID, "iteration.start", ...)
    for _, agent := range agents {
        emitter.Publish(sessionID, "agent.started", ...)
        out, err := dispatch(agent, current)
        if err != nil {
            emitter.Publish(sessionID, "agent.error", ...)
            return ...
        }
        before := current.Metrics.Confidence
        current = merge(current, out)
        emitter.Publish(sessionID, "agent.complete", map[string]any{
            "iteration":        i,
            "agent_id":         agent.ID,
            "partial_state":    diff(prev, current),
            "confidence_delta": current.Metrics.Confidence - before,
        })
    }
    emitter.Publish(sessionID, "iteration.complete", ...)
    if convergence.Check(prev, current) { break }
}
```

`RunSingleAgent` (preview path) emits only `agent.started` and `agent.complete` / `agent.error`. `session.finalized` is published from `session/service.go` after a successful finalize.

**Bounded resources:**

- Ring buffer cap: 100 events per session.
- Max subscribers per session: 10 (hard limit; 11th `Subscribe` returns `nil` channel → handler responds `429`).
- Subscriber channel buffer: 32. When full, the broadcaster drops the subscriber (treated as disconnected).

---

### 8.23 Generated Document Quality Standard (v1.5)

**Purpose.** Replace the broken output pipeline (title bug, idea duplication, empty sections, padding boilerplate) with a deterministic quality contract. This section defines the title/description extraction rules, slug-based filename pattern, state readiness gate, and the enriched canonical state schema that the generators consume.

#### Title and description extraction

```go
// shortTitle picks a concise document title from canonical state.
// Source priority: idea.name > first sentence of idea.text (60-char max, word boundary) > "Untitled Brainstorm".
// Output is single-line, Markdown-stripped, multi-space collapsed.
func shortTitle(s state.CanonicalState) string

// oneLineDescription returns a single sentence ≤ 200 chars suitable for a blockquote/lead paragraph.
// Never emits the full idea body verbatim. Source priority: idea.summary > first sentence of idea.text.
func oneLineDescription(s state.CanonicalState) string
```

**Banned outputs:**

- Using the full `idea.text` (often a multi-sentence paragraph) as an H1 — the root cause of the title bug observed in v1.4 outputs.
- Emitting `idea.text` in more than one place per document (today: title + blockquote + Overview = triple-print).

#### Slug and filename pattern

```go
// slugify produces a lowercase, ASCII, hyphen-separated, ≤ 50-char slug.
// Rules: lowercase, drop punctuation, spaces → "-", collapse repeats, trim leading/trailing "-".
// Empty input → "untitled".
func slugify(title string) string

var suffixForKey = map[string]string{
    "architecture": "architecture.md",
    "roadmap":      "roadmap.md",
    "plan":         "plan.md",
    "readme":       "readme.md",
}

func buildFilename(title, key string) string {
    return slugify(title) + "_" + suffixForKey[key]
}
```

**Example:** session idea name `"Match Point"` → `match-point_architecture.md`, `match-point_roadmap.md`, `match-point_plan.md`, `match-point_readme.md`. All four files for one session share the same slug prefix (`GenerateAll` computes it once and reuses it).

#### Sparse-state finalize gate

The `Finalize` service helper:

```go
func isStateReadyForFinalize(s state.CanonicalState) (ready bool, reason string) {
    if len(s.Idea) == 0           { return false, "idea is empty" }
    if len(s.Architecture) == 0   { return false, "architecture is empty" }
    if len(s.ExecutionPlan) == 0  { return false, "execution_plan is empty" }
    if s.Metrics.Confidence < 0.5 { return false, "confidence below 0.5 — continue brainstorming" }
    return true, ""
}
```

Handler maps `ready == false` to HTTP `422 Unprocessable Entity` with body `{"error":"state_not_ready","reason":"..."}`. The frontend Finalize view surfaces the reason instead of downloading empty files.

#### Enriched CanonicalState schema (additive, backward-compatible)

All fields below are optional JSON keys inside the existing `map[string]any` containers (no struct rename, no schema migration). Agents are instructed via prompt to populate them when known.

```jsonc
{
  "architecture": {
    "layers": [
      {
        "name": "Backend",
        "responsibility": "...",
        "technologies": ["Go 1.26", "pgx/v5"],
        "dependencies": ["Database"],
      },
    ],
    "data_flows": [
      {
        "from": "Frontend",
        "to": "Backend",
        "protocol": "HTTP/JSON",
        "description": "...",
      },
    ],
    "tech_stack": { "backend": "Go 1.26", "frontend": "SvelteKit" },
    "directory_layout": ["backend/", "frontend/", "migrations/"],
  },
  "execution_plan": [
    {
      "phase": "Phase 1 — Foundation",
      "objective": "Land the platform skeleton",
      "deliverables": ["go.work", "docker-compose.yml"],
      "exit_criteria": ["All services build green"],
      "blocking_dependencies": [],
    },
  ],
  "risks": [
    {
      "name": "LLM rate-limit",
      "likelihood": "medium",
      "impact": "high",
      "mitigation": "...",
    },
  ],
  "assumptions": [
    {
      "name": "Single-tenant deployment",
      "rationale": "...",
      "validation_method": "...",
    },
  ],
  "metrics": {
    "confidence": 0.0,
    "test_coverage_target": 0.8,
    "latency_budget_ms": 250,
  },
}
```

Generators iterate these structured fields when present; if absent they fall back to the v1.4 map-walk so old sessions still render (but they will fail the finalize gate above if the core containers are empty).

#### Agent prompt fragment (injected by `executor.go`)

```
## Required Output Structure (CanonicalState)

When you emit canonical state JSON, populate the structured sub-fields described below — not just free-form maps.

- architecture.layers []{ name, responsibility, technologies, dependencies }
- architecture.data_flows []{ from, to, protocol, description }
- execution_plan []{ phase, objective, deliverables, exit_criteria, blocking_dependencies }
- risks []{ name, likelihood (low|medium|high), impact (low|medium|high), mitigation }
- assumptions []{ name, rationale, validation_method }
- metrics { confidence, test_coverage_target, latency_budget_ms }

Do not omit a layer or phase you previously discussed. Do not repeat the idea text verbatim inside architecture or roadmap sections.
```

The Architect / Engineer / Reviewer role prompts stored in `agents.system_prompt` are appended with this fragment via a one-shot SQL `UPDATE` recorded in task notes (no new migration file — row-level data update only).

#### Deleted: `enforceMinLines`

The v1.3/v1.4 `enforceMinLines(body, minLines)` helper and its four padders (`padArchitecture`, `padRoadmap`, `padPlan`, `padReadme`) are **removed entirely**. Line count is not a quality signal; depth comes from the enriched state schema above. Tests that asserted `lineCount >= 1000` are replaced with content assertions (title shape, idea-occurrence count, presence of per-layer sub-sections).

---

### 8.24 MCP Server Registry Schema (v1.4)

#### `mcp_servers` table (migration 006)

```sql
CREATE TABLE mcp_servers (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL UNIQUE,
    description TEXT        NOT NULL DEFAULT '',
    transport   TEXT        NOT NULL CHECK (transport IN ('stdio', 'http')),
    command     TEXT,
    url         TEXT,
    env_refs    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transport_fields CHECK (
        (transport = 'stdio' AND command IS NOT NULL AND url IS NULL) OR
        (transport = 'http'  AND url IS NOT NULL     AND command IS NULL)
    )
);
```

**`env_refs` JSONB shape:** maps logical key names to environment variable names. Both keys and values are env var names — neither is ever a raw secret value.

```json
{
  "BRAVE_API_KEY": "BRAVE_API_KEY",
  "GITHUB_TOKEN": "GH_PAT_TOKEN"
}
```

The agent binary resolves actual values at runtime via `config.GetLLMAPIKey(envVarName)`. The backend never reads, stores, or logs the resolved values.

**Security validation rule (service layer):** reject any `env_refs` entry whose value contains `=` or whitespace — these characters indicate a raw key value, not a var name. Return HTTP 400.

#### `agent_mcp_servers` join table (migration 007)

```sql
CREATE TABLE agent_mcp_servers (
    agent_id      UUID NOT NULL REFERENCES agents(id)      ON DELETE CASCADE,
    mcp_server_id UUID NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
    position      INT  NOT NULL DEFAULT 0,
    PRIMARY KEY (agent_id, mcp_server_id)
);

CREATE INDEX idx_agent_mcp_servers_agent_id ON agent_mcp_servers (agent_id);
```

#### `MCPServerRef` wire format (in `BrainstormPayload.MCPServers`)

```go
// MCPServerRef — connection details packed into BrainstormPayload by the iteration engine.
// Defined in: backend/internal/platform/a2a/types.go (canonical)
//             agent/internal/executor/executor.go   (mirror — no cross-binary imports)
type MCPServerRef struct {
    ID        string            `json:"id"`
    Name      string            `json:"name"`
    Transport string            `json:"transport"`          // "stdio" | "http"
    Command   string            `json:"command,omitempty"` // stdio only
    URL       string            `json:"url,omitempty"`     // http only
    EnvRefs   map[string]string `json:"env_refs,omitempty"` // var names only, never values
}
```

`BrainstormPayload` extension:

```go
// Added in v1.4 — omit if empty so older agent binaries receive backward-compatible payloads
MCPServers []MCPServerRef `json:"mcp_servers,omitempty"`
```

#### `MCPServer` Go struct (backend)

```go
// MCPServer — DB model in backend/internal/modules/mcpserver/model.go
type MCPServer struct {
    ID          uuid.UUID         `json:"id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Transport   string            `json:"transport"`
    Command     *string           `json:"command,omitempty"`
    URL         *string           `json:"url,omitempty"`
    EnvRefs     map[string]string `json:"env_refs"`
    CreatedAt   time.Time         `json:"created_at"`
}
```

#### REST endpoints (registered in `platform/http/router.go`)

| Method   | Path                     | Description                         | Success | Error codes |
| -------- | ------------------------ | ----------------------------------- | ------- | ----------- |
| `GET`    | `/mcp-servers`           | List all registered MCP servers     | 200     | —           |
| `POST`   | `/mcp-servers`           | Register a new MCP server           | 201     | 400, 409    |
| `GET`    | `/mcp-servers/{id}`      | Get one MCP server by ID            | 200     | 404         |
| `PUT`    | `/mcp-servers/{id}`      | Update an MCP server (full replace) | 200     | 400, 404    |
| `DELETE` | `/mcp-servers/{id}`      | Delete an MCP server                | 204     | 404         |
| `POST`   | `/mcp-servers/{id}/test` | Dial server, return tool list       | 200     | 404, 502    |

---

### 8.25 MCP Tool-Use Loop — Protocol and Algorithm (v1.4)

#### MCP JSON-RPC 2.0 Protocol

The Model Context Protocol uses JSON-RPC 2.0 over two transports:

- **stdio**: newline-delimited JSON messages written to subprocess stdin / read from stdout
- **http**: each JSON-RPC call is a `POST` to the server's URL with `Content-Type: application/json`

**Initialize handshake (required before first tool call, stdio and http):**

Request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": { "name": "a2a-brainstorm", "version": "1.4" }
  }
}
```

Response (relevant fields):

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "serverInfo": { "name": "...", "version": "..." },
    "capabilities": { "tools": {} }
  }
}
```

After receiving `initialize` response, send `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}` (no `id` — notification, no response expected).

**`tools/list` request:**

```json
{ "jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {} }
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "brave_web_search",
        "description": "Performs a web search...",
        "inputSchema": {
          "type": "object",
          "properties": { "query": { "type": "string" } },
          "required": ["query"]
        }
      }
    ]
  }
}
```

**`tools/call` request:**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "brave_web_search",
    "arguments": { "query": "Go 1.26 release notes" }
  }
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [{ "type": "text", "text": "Go 1.26 was released on..." }],
    "isError": false
  }
}
```

Extract all `type=text` parts, concatenate into `ToolResult.Content`. If `isError == true`, set `ToolResult.IsError = true` and include error text in `Content`.

#### `ToolDef` and `LLMResponseWithTools` Go types

```go
// Defined in agent/internal/llm/copilot.go (canonical for the LLM package)
// Mirrored without import in agent/internal/mcp/types.go

type ToolDef struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    InputSchema json.RawMessage `json:"input_schema"` // JSON Schema object
}

type ToolCallRequest struct {
    ID        string          `json:"id"`
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments"`
}

type LLMResponseWithTools struct {
    Content      string           `json:"content"`
    FinishReason string           `json:"finish_reason"` // "stop" | "tool_calls"
    TokensUsed   int              `json:"tokens_used"`
    ToolCalls    []ToolCallRequest `json:"tool_calls,omitempty"`
}

type Message struct {
    Role       string           `json:"role"`        // "system" | "user" | "assistant" | "tool"
    Content    string           `json:"content,omitempty"`
    ToolCalls  []ToolCallRequest `json:"tool_calls,omitempty"` // assistant message with tool calls
    ToolCallID string           `json:"tool_call_id,omitempty"` // tool result message
}
```

#### OpenAI Function Calling Wire Format (for `CopilotProvider.GenerateWithTools`)

Tools array sent to Copilot completions API:

```json
{
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "brave_web_search",
        "description": "Performs a web search using Brave Search API",
        "parameters": {
          "type": "object",
          "properties": { "query": { "type": "string" } },
          "required": ["query"]
        }
      }
    }
  ],
  "tool_choice": "auto"
}
```

When the model wants to call a tool, response contains:

```json
{
  "choices": [
    {
      "finish_reason": "tool_calls",
      "message": {
        "role": "assistant",
        "tool_calls": [
          {
            "id": "call_abc123",
            "type": "function",
            "function": {
              "name": "brave_web_search",
              "arguments": "{\"query\":\"Go 1.26\"}"
            }
          }
        ]
      }
    }
  ]
}
```

#### OpenCode Adaptation (for `OpenCodeProvider.GenerateWithTools`)

OpenCode does not have a native function-calling parameter. Instead, encode the tool definitions as a JSON block appended to the system prompt:

```
<available_tools>
[{"name":"brave_web_search","description":"...","inputSchema":{...}}]
</available_tools>

When you need to call a tool, respond with a JSON block in this format and nothing else:
{"tool_calls":[{"id":"<uuid>","name":"<tool_name>","arguments":{...}}]}

After receiving tool results, continue your reasoning and produce the final CanonicalState JSON.
```

The executor parses the LLM response text; if it starts with `{"tool_calls":`, treat as a tool-call response regardless of `FinishReason`.

#### Multi-turn Tool-Use Loop Algorithm

```go
// In agent/internal/executor/executor.go — Execute()
maxRounds := config.GetMCPMaxToolRounds() // default 5, clamp [1,20]

// Build pool (skip if no MCP servers configured — zero regression path)
var pool *mcp.MCPPool
if len(payload.MCPServers) > 0 {
    var err error
    pool, err = mcp.NewPool(ctx, toServerRefs(payload.MCPServers), config.GetLLMAPIKey, nil)
    if err != nil {
        // emit agent.error; return
    }
    defer pool.Close()
}

// List tools
var tools []llm.ToolDef
if pool != nil {
    tools, _ = pool.ListAllTools(ctx) // ignore error: empty tools = no tool use
}

// Build initial message history
messages := []llm.Message{
    {Role: "system", Content: payload.SystemPrompt},
    {Role: "user",   Content: userMessage}, // JSON-encoded CanonicalState
}

// Multi-turn loop
var finalContent string
for round := 0; round < maxRounds; round++ {
    resp, err := e.llm.GenerateWithTools(ctx, llm.LLMRequest{
        SystemPrompt: payload.SystemPrompt,
        UserMessage:  messagesAsText(messages), // implementation-specific encoding
        Temperature:  0.15,
    }, tools)
    if err != nil { /* emit agent.error; return */ }

    if len(resp.ToolCalls) == 0 {
        finalContent = resp.Content
        break
    }

    // Append assistant message with tool calls
    messages = append(messages, llm.Message{
        Role:      "assistant",
        ToolCalls: resp.ToolCalls,
    })

    // Execute each tool call and append results
    for _, tc := range resp.ToolCalls {
        var args json.RawMessage
        _ = json.Unmarshal([]byte(tc.Arguments.String()), &args)
        result, _ := pool.Call(ctx, tc.Name, args)
        messages = append(messages, llm.Message{
            Role:       "tool",
            ToolCallID: tc.ID,
            Content:    result.Content,
        })
    }
}

// Fallback: if loop exhausted without a stop signal, use last response
if finalContent == "" && len(messages) > 0 {
    slog.Warn("tool-use loop exhausted without stop signal", "maxRounds", maxRounds)
    finalContent = resp.Content // last response, may be partial
}

// Parse finalContent as CanonicalState (existing logic — unchanged)
```

**Stop conditions:**

- `len(resp.ToolCalls) == 0` AND `resp.FinishReason == "stop"` — normal completion
- `len(resp.ToolCalls) == 0` AND any other `FinishReason` — treat as stop
- Round counter reaches `maxRounds` — use last content received

---

### 8.26 Smart Import Config Format (v1.4)

The smart import modal in `/settings?tab=mcp` accepts JSON pasted from any of the following configuration file formats used by MCP host applications. The `parseMCPConfig(raw: string)` function (in `frontend/src/routes/settings/+page.svelte`) normalises all formats to a common `ParsedMCPServer[]` array.

#### Supported source formats

| Source         | Config file                         | Top-level key     | Server value shape                                       |
| -------------- | ----------------------------------- | ----------------- | -------------------------------------------------------- |
| Claude Desktop | `claude_desktop_config.json`        | `mcpServers`      | `{"command": "npx", "args": [...], "env": {...}}`        |
| VS Code        | `.vscode/mcp.json`                  | `servers`         | `{"type":"stdio","command":"...","args":[...],"env":{}}` |
| Cursor         | `.cursor/mcp.json`                  | `mcpServers`      | same as Claude Desktop                                   |
| Zed            | `settings.json`                     | `context_servers` | `{"command":{"path":"...","args":[...],"env":{}}}`       |
| Windsurf       | `.codeium/windsurf/mcp_config.json` | `mcpServers`      | same as Claude Desktop                                   |
| Canonical      | any                                 | (root array)      | `[{"name":"...","transport":"stdio","command":"..."}]`   |

#### Detection and normalisation algorithm

```typescript
function parseMCPConfig(raw: string): ParsedMCPServer[] {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return [];
  }

  // Canonical array: root is an array
  if (Array.isArray(parsed)) return parsed.map(normaliseCanonical);

  // Claude Desktop / Cursor / Windsurf: { mcpServers: { name: {...} } }
  if (typeof parsed === "object" && parsed !== null && "mcpServers" in parsed) {
    return Object.entries((parsed as any).mcpServers).map(([name, cfg]) =>
      normaliseClaudeDesktop(name, cfg),
    );
  }

  // VS Code: { servers: { name: {...} } }
  if (typeof parsed === "object" && parsed !== null && "servers" in parsed) {
    return Object.entries((parsed as any).servers).map(([name, cfg]) =>
      normaliseVSCode(name, cfg),
    );
  }

  // Zed: { context_servers: { name: {...} } }
  if (
    typeof parsed === "object" &&
    parsed !== null &&
    "context_servers" in parsed
  ) {
    return Object.entries((parsed as any).context_servers).map(([name, cfg]) =>
      normaliseZed(name, cfg),
    );
  }

  return [];
}
```

#### `normaliseClaudeDesktop(name, cfg)` logic

- `transport`: `"stdio"` always (Claude Desktop only supports stdio)
- `command`: join `cfg.command` + `cfg.args` as space-separated string, or use `cfg.command` if `args` absent
- `env_refs`: see env-stripping policy below; map each key to a sanitised env var name
- Result: `{ name, transport: "stdio", command, env_refs, description: "" }`

#### `normaliseVSCode(name, cfg)` logic

- `transport`: from `cfg.type` → `"stdio"` or `"http"` (default stdio)
- For `http`: `url` from `cfg.url`; for `stdio`: `command` assembled from `cfg.command + cfg.args`
- `env_refs`: same stripping policy

#### `normaliseZed(name, cfg)` logic

- `transport`: `"stdio"`
- `command`: `cfg.command.path + " " + cfg.command.args.join(" ")`
- `env_refs`: from `cfg.command.env`

#### Env-value stripping policy

When iterating over `env` fields in the pasted config:

```typescript
function processEnvRefs(env: Record<string, string>): {
  env_refs: Record<string, string>;
  hadSecrets: boolean;
} {
  const result: Record<string, string> = {};
  let hadSecrets = false;
  for (const [key, value] of Object.entries(env)) {
    // Looks like an env var name: all uppercase, digits, underscores, no spaces
    const looksLikeVarName = /^[A-Z][A-Z0-9_]*$/.test(value);
    if (looksLikeVarName) {
      result[key] = value; // store as-is (already a var name reference)
    } else {
      // Value looks like a raw secret — strip it, keep only the key as the var name
      result[key] = key; // convention: key name becomes the env var name
      hadSecrets = true;
    }
  }
  return { env_refs: result, hadSecrets };
}
```

**Rule:** if `hadSecrets` is true for any server in the batch, show the security warning banner in the import modal.

**Warning message:** "⚠ API key values were detected and stripped from `env` fields. The import stored only env var names. Set the corresponding env vars on the machine where the agent binary runs."

#### `ParsedMCPServer` TypeScript interface

```typescript
interface ParsedMCPServer {
  name: string;
  transport: "stdio" | "http";
  command?: string; // stdio only
  url?: string; // http only
  env_refs: Record<string, string>; // var names only, never raw values
  description: string;
  _hadSecrets: boolean; // UI-only flag; not sent to backend
}
```

---

### 8.27 AI-Driven Document Generator + Skill Bundle (v1.6)

**Status:** Added in v1.6 with Task 33. This section is the canonical specification for the AI-driven hybrid finalize pipeline. It supplements §8.23 (deterministic quality standard) and is wrapped around — never replaces — the Task-32 generators.

**Goal.** Push generated output (`architecture.md`, `roadmap.md`, `plan.md`, `readme.md`) to the depth, tone, and structure of the reference docs (`docs/A2A-agent-Brainstorm.md`, `docs/architecture.md`, `docs/implementation_roadmap.md`, this repo's `README.md`) by running each deterministic scaffold through an LLM pass that is conditioned on a curated bundle of project skills.

**Three-mode contract.** Selected at startup via `FINALIZE_MODE` env var.

| Mode            | Behaviour                                                                                                                                   | When to use                                                 |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| `deterministic` | Exactly the post-Task-32 pipeline. No LLM call. Byte-for-byte stable output.                                                                | CI snapshot tests, air-gapped runs, debugging.              |
| `hybrid`        | Deterministic scaffold first → AI rewrite per document → rubric validate → auto-repair up to N times → on hard failure return the scaffold. | **Default.** Production finalize.                           |
| `ai`            | Same as `hybrid` but on hard failure return the error instead of the scaffold.                                                              | Testing the AI path; explicit opt-in for high-quality runs. |

**Skill Bundle (canonical, in load order).** Each entry is the verbatim body of a `.github/skills/<name>/SKILL.md` file with YAML frontmatter stripped at load time, concatenated under `## Skill: <name>` headings separated by blank lines. The bundle is composed once per `GenerateAll` call and reused across all document keys.

| Order | Skill path                                | Why it's in the bundle                                                                |
| ----- | ----------------------------------------- | ------------------------------------------------------------------------------------- |
| 1     | `.github/skills/modularity/SKILL.md`      | Enforces the modular monolith / vertical-slice mental model in architecture sections. |
| 2     | `.github/skills/vertical-slice/SKILL.md`  | Drives per-feature directory layout and section structure in plan + roadmap docs.     |
| 3     | `.github/skills/api-design/SKILL.md`      | Shapes API surface sections (REST contracts, request/response, pagination, errors).   |
| 4     | `.github/skills/roadmap-spec/SKILL.md`    | Enforces the canonical roadmap section template (Objective / BLOCKERS / Scope / …).   |
| 5     | `.github/skills/plan-management/SKILL.md` | Mirrors the PLAN.md task template into the generated `plan.md` document.              |

`SKILL_BUNDLE_PATHS` env var (comma-separated) overrides this list. Missing files cause startup error — no silent fallback.

**Per-document rubric defaults.** Returned by `RubricFor(docKey)`. All numeric bounds are config-overridable in a follow-up — Task 33 ships these constants:

````go
var defaultRubrics = map[string]Rubric{
  "architecture": {
    Sections: []SectionRule{
      {Heading: "1. Overview",          MinChars: 400},
      {Heading: "2. System Components", MinChars: 600, RequiredKeywords: []string{"Responsibility","Technologies","Dependencies"}},
      {Heading: "3. Data Model",        MinChars: 400},
      {Heading: "4. Data Flow",         MinChars: 400, RequiredKeywords: []string{"```mermaid"}},
      {Heading: "5. Deployment",        MinChars: 300},
    },
  },
  "roadmap": {
    Sections: []SectionRule{
      {Heading: "1. Goals",          MinChars: 300},
      {Heading: "2. Milestones",     MinChars: 500},
      {Heading: "3. Phase Breakdown",MinChars: 800, RequiredKeywords: []string{"Objective","Scope","Deliverables","Exit Criteria"}},
      {Heading: "4. Risks",          MinChars: 300},
    },
  },
  "plan": {
    Sections: []SectionRule{
      {Heading: "1. Scope",       MinChars: 300},
      {Heading: "2. Architecture",MinChars: 400},
      {Heading: "3. Modules",     MinChars: 600},
      {Heading: "4. Tasks",       MinChars: 800, RequiredKeywords: []string{"Files to create","Validation"}},
    },
  },
  "readme": {
    Sections: []SectionRule{
      {Heading: "Overview",     MinChars: 300},
      {Heading: "Architecture", MinChars: 300},
      {Heading: "Roadmap",      MinChars: 300},
      {Heading: "Getting Started", MinChars: 200},
    },
  },
}
````

All rubrics share the placeholder blocklist: `["TBD", "TODO", "Lorem ipsum", "placeholder"]` — any occurrence in section body fails the rubric.

**Auto-repair algorithm.**

```
draft        := llm.Generate(ctx, req{ system: bundle + docContract, user: scaffold + state })
findings     := Validate(draft, rubric)
attempts     := 0
for len(findings) > 0 && attempts < maxRepairs {
  req.UserMessage = "Previous draft:\n\n" + draft +
                    "\n\nRubric findings (must fix all):\n" + bullets(findings) +
                    "\n\nReturn the full revised document only."
  draft     = llm.Generate(ctx, req)
  findings  = Validate(draft, rubric)
  attempts++
}
if len(findings) > 0 {
  return scaffold        // hybrid mode
  // OR
  return error(findings) // ai mode
}
return draft
```

`maxRepairs` defaults to `2` (`AIGEN_MAX_REPAIRS`, clamp `[0,5]`). Temperature defaults to `0.2` (`AIGEN_TEMPERATURE`, clamp `[0.0,1.0]`) to keep regenerations stable.

**Why backend reuses the existing `LLMProvider` instead of dialing the agent binary.** Finalize is a one-shot synchronous backend operation — there is no canonical-state iteration, no convergence loop, no per-agent role split. Routing it through A2A would add latency, an extra failure mode, and require packing the entire backend state into a `BrainstormPayload`. The same provider abstraction (`backend/internal/platform/llm/`) already exists in the backend module for non-iteration LLM calls, and credential security (§8.12) is unchanged.

**Fallback semantics (hybrid mode).** Any of the following events causes the orchestrator to log `slog.Warn{event:"aigen_fallback", doc_key, reason}` and return the deterministic scaffold for that document key:

- LLM provider returns error (transport, auth, rate limit, timeout)
- LLM response is empty or shorter than the scaffold
- Rubric still fails after `maxRepairs` attempts
- `SkillBundle.Compose()` returns empty (missing skills, fs error)

Other documents in the same `GenerateAll` call continue with AI — fallback is per-key, never global.

**Determinism trade-off.** `deterministic` mode preserves Task-32 byte-stability for CI/snapshot tests. `hybrid` mode is **not** deterministic across runs (LLM stochasticity); this is acceptable because the artifacts are advisory developer outputs, not protocol state. Canonical state merging (§8.5) is untouched by this task.

**Config summary.**

| Env var              | Default                                                                                                                                                                                     | Range           |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------- |
| `FINALIZE_MODE`      | `hybrid`                                                                                                                                                                                    | enum            |
| `SKILL_BUNDLE_PATHS` | `.github/skills/modularity/SKILL.md,.github/skills/vertical-slice/SKILL.md,.github/skills/api-design/SKILL.md,.github/skills/roadmap-spec/SKILL.md,.github/skills/plan-management/SKILL.md` | comma-separated |
| `AIGEN_MAX_REPAIRS`  | `2`                                                                                                                                                                                         | `[0, 5]`        |
| `AIGEN_TEMPERATURE`  | `0.2`                                                                                                                                                                                       | `[0.0, 1.0]`    |

All four getters live in `backend/internal/platform/config/config.go` — no `os.Getenv` calls elsewhere.

---

### 8.28 Hierarchical Attachment Context System (v1.7)

The Attachment Context System lets users enrich brainstorm sessions with external artifacts — files, images, URLs, and raw text snippets — that are extracted, embedded, and retrieved on demand via cosine similarity to enrich each agent dispatch with relevant context. This section is the canonical contract for Tasks 34–38.

#### 8.28.1 Goals & Non-Goals

**Goals:**

- Let users attach context at three scopes (`session` / `iteration` / `agent`) using a single ChatGPT-style `+` menu mounted at the appropriate UI location.
- Support four input kinds — file (PDF/DOCX/MD/TXT/JSON), image (PNG/JPG/WEBP, vision-described), URL (server-fetched + cleaned), raw text paste.
- Store extracted text + embeddings in Postgres; store original blobs in MinIO/S3-compatible object storage.
- Retrieve top-K relevant chunks at dispatch time via pgvector cosine similarity and inject them into the assembled system prompt under a `# Attached Context` section.
- Auto-expire iteration-scope and agent-scope attachments after their owning unit of work completes.

**Non-Goals:**

- Multi-tenant authorization (attachments inherit the session's access model).
- Cross-session attachment reuse / global "attachment library" (rejected per brainstorm Q4; deferred to a future task if demand emerges).
- Streaming uploads > 10 MB (configurable via `ATTACHMENT_MAX_BYTES`; reject 413 above the cap).
- Full-document RAG (chunk-level only; we deliberately do not pass entire docs into prompts — token budget protection).

#### 8.28.2 Scope Model

| Scope       | `scope_ref` value                            | Lifecycle                                                        | UI mount point                        |
| ----------- | -------------------------------------------- | ---------------------------------------------------------------- | ------------------------------------- |
| `session`   | NULL                                         | Survives until session deleted (cascade FK)                      | Home page (during creation) + sidebar |
| `iteration` | string of next iteration number (e.g. `"3"`) | Deleted by engine immediately after iteration N's state persists | Session page, between iterations      |
| `agent`     | agent UUID string                            | Deleted by engine after the agent's dispatch returns             | `PipelineStage` header per agent      |

Retrieval at any dispatch always unions all three scopes' attachments for that session + iteration + agent. Scope is a filter on which rows are eligible — not a separate retrieval path.

#### 8.28.3 Database Schema

```sql
-- migrations/009_attachments.sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TYPE attachment_scope AS ENUM ('session', 'iteration', 'agent');
CREATE TYPE attachment_kind  AS ENUM ('file', 'image', 'url', 'text');

CREATE TABLE attachments (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id      UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  scope           attachment_scope NOT NULL,
  scope_ref       TEXT,                        -- NULL for session; iteration number or agent UUID otherwise
  kind            attachment_kind NOT NULL,
  display_name    TEXT NOT NULL,
  mime_type       TEXT NOT NULL DEFAULT '',
  byte_size       BIGINT NOT NULL DEFAULT 0,
  source_url      TEXT,                        -- set when kind = 'url'
  blob_key        TEXT,                        -- object-storage key when kind IN ('file','image')
  extracted_text  TEXT NOT NULL DEFAULT '',
  summary         TEXT NOT NULL DEFAULT '',    -- ≤ 500 chars; fallback when chunk retrieval returns 0 rows
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT attachments_scope_ref_consistency CHECK (
    (scope = 'session' AND scope_ref IS NULL)
    OR (scope IN ('iteration', 'agent') AND scope_ref IS NOT NULL)
  )
);
CREATE INDEX idx_attachments_session_scope ON attachments (session_id, scope, scope_ref);

-- migrations/010_attachment_chunks.sql
CREATE TABLE attachment_chunks (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  attachment_id  UUID NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
  chunk_index    INT  NOT NULL,
  content        TEXT NOT NULL,
  embedding      VECTOR(1536) NOT NULL,        -- dim must equal config.GetEmbeddingsDimension()
  tokens         INT  NOT NULL DEFAULT 0,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (attachment_id, chunk_index)
);
CREATE INDEX idx_attachment_chunks_embedding
  ON attachment_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
```

#### 8.28.4 Upload Pipeline Algorithm

```
Service.Create(input):
  1. Validate input per kind:
       file/image: reader != nil, display_name != ""
       url:        url matches ^https?:// (SSRF guard)
       text:       text != "", display_name != ""
  2. Validate scope/scope_ref consistency (see 8.28.2)
  3. extractor := registry.Resolve(input.Kind)
     result    := extractor.Extract(ctx, input)
     reject if len(result.Text) < MinExtractedChars (16)
  4. if kind in (file, image):
       blob_key := "attachments/{sessionID}/{newAttachmentID}/{slug(display_name)}"
       blob.Put(blob_key, originalBytes, contentType)
  5. chunks := chunkText(result.Text, GetAttachmentChunkSize(), GetAttachmentChunkOverlap())
  6. vectors := embeddings.Embed(chunks)   // single batched call
  7. summary := llm.Generate(systemPrompt="Summarise this in ≤ 500 chars …", user=result.Text[:8000])
                  // optional; failure → summary = ""
  8. BEGIN TX
       repo.Create(tx, attachment_row)
       repo.CreateChunks(tx, attachmentID, zip(chunks, vectors))
     COMMIT
     on rollback: blob.Delete(blob_key)  // best-effort
  9. return attachment
```

**Chunking algorithm (paragraph-aware, token-budgeted):**

```
chunkText(text, sizeTokens, overlapTokens):
  paragraphs := split on /\n\n+/
  current    := ""
  out        := []
  for p in paragraphs:
    if approxTokens(current + p) > sizeTokens:
      out.append(current)
      current = tailTokens(current, overlapTokens) + p
    else:
      current = current + "\n\n" + p
  if current != "":
    out.append(current)
  return out
```

`approxTokens(s) = len(s) / 4` (cheap heuristic; deterministic). Real tokenization is unnecessary at this layer — embedding model handles its own truncation.

#### 8.28.5 Retrieval & Dispatch Algorithm

Called inside `iteration/engine.go` before each agent dispatch (Task 40):

```
for agent in sess.OrderedAgents:
  query := state.Idea + "\n\n" + strings.Join(state.OpenQuestions, "\n")
  scopes := []ScopeMatch{
    {Scope: ScopeSession,   ScopeRef: nil},
    {Scope: ScopeIteration, ScopeRef: &iterStr},
    {Scope: ScopeAgent,     ScopeRef: &agent.ID},
  }
  chunks, _ := retriever.Retrieve(ctx, sess.ID, scopes, query, GetAttachmentRetrievalTopK())
  refs      := convertToAttachmentChunkRefs(chunks)
  newState  := dispatch(ctx, agent, role, skills, llmOverride, currentState, refs)
  // ... merge, persist, etc.

// after iteration:
retriever.DeleteByScope(ctx, sess.ID, ScopeIteration, iterStr)
```

Per-agent cleanup: `DeleteByScope(ScopeAgent, agent.ID)` invoked after that agent's dispatch returns (inside `runPipelinePass`).

**SQL for `SearchChunks`:**

```sql
SELECT c.id, c.attachment_id, c.chunk_index, c.content, c.tokens,
       (1.0 - (c.embedding <=> $1::vector)) AS score
FROM attachment_chunks c
JOIN attachments a ON a.id = c.attachment_id
WHERE a.session_id = $2
  AND (
       (a.scope = 'session'   AND $3::bool)
    OR (a.scope = 'iteration' AND a.scope_ref = $4)
    OR (a.scope = 'agent'     AND a.scope_ref = $5)
  )
ORDER BY c.embedding <=> $1::vector
LIMIT $6;
```

#### 8.28.6 Wire Format (`BrainstormPayload` extension)

```go
// backend/internal/platform/a2a/types.go (and copied verbatim into agent/internal/executor)
type AttachmentChunkRef struct {
    Scope       string  `json:"scope"`                // "session" | "iteration" | "agent"
    ScopeRef    string  `json:"scope_ref,omitempty"`  // omitted when Scope == "session"
    DisplayName string  `json:"display_name"`
    Content     string  `json:"content"`
    Score       float32 `json:"score"`                // cosine similarity in [0, 1]
}

type BrainstormPayload struct {
    Role         string                  `json:"role"`
    SystemPrompt string                  `json:"system_prompt"`
    LLMConfig    llm.LLMConfig           `json:"llm_config"`
    State        any                     `json:"state"`
    Attachments  []AttachmentChunkRef    `json:"attachments,omitempty"` // v1.7
}
```

**System-prompt injection format** (agent executor, after base prompt assembly):

```
# Attached Context

The following snippets were retrieved from user-attached artifacts for this dispatch.
Treat them as authoritative context for the brainstorm but do not echo them verbatim.

## [scope: session | source: prd-v3.pdf | relevance: 0.84]
{chunk content}

## [scope: agent | source: db-pick.txt | relevance: 0.79]
{chunk content}

...
```

Chunks rendered in **descending score order**. When `payload.Attachments` is empty, the `# Attached Context` section is omitted entirely — byte-identical behaviour to pre-v1.7.

#### 8.28.7 REST API

| Method | Path                                                  | Body                                                   | Response                                   |
| ------ | ----------------------------------------------------- | ------------------------------------------------------ | ------------------------------------------ |
| POST   | `/sessions/{sessionID}/attachments`                   | multipart (file/image) or JSON (url/text) — see 8.28.8 | 201 + `Attachment`                         |
| GET    | `/sessions/{sessionID}/attachments?scope=&scope_ref=` | —                                                      | 200 + `[]Attachment`                       |
| GET    | `/sessions/{sessionID}/attachments/{id}`              | —                                                      | 200 + `Attachment`                         |
| GET    | `/sessions/{sessionID}/attachments/{id}/content`      | —                                                      | 302 → presigned blob URL (file/image only) |
| DELETE | `/sessions/{sessionID}/attachments/{id}`              | —                                                      | 204                                        |

Error codes: 400 (validation), 404 (not found), 413 (oversize), 415 (unsupported MIME), 422 (extraction failed — e.g. encrypted PDF).

#### 8.28.8 Request Body Variants

```json
// file (multipart/form-data)
scope=session
kind=file
file=<binary>
display_name=prd-v3.pdf   (optional; defaults to uploaded filename)

// image (multipart/form-data)
scope=agent
scope_ref=<agent-uuid>
kind=image
file=<binary>

// url (application/json)
{
  "scope": "session",
  "kind": "url",
  "url": "https://example.com/article",
  "display_name": "Competitor article"   // optional; defaults to URL hostname + path
}

// text (application/json)
{
  "scope": "iteration",
  "scope_ref": "3",
  "kind": "text",
  "display_name": "db-pick",
  "text": "Use Postgres 16 with pgvector for hybrid search."
}
```

#### 8.28.9 Security Invariants

1. **SSRF guard:** URL fetcher allowlists `http://` and `https://` schemes only. Reject `file://`, `ftp://`, `data:`, `gopher://`, `javascript:` with 400.
2. **Size cap:** `http.MaxBytesReader(r.Body, GetAttachmentMaxBytes())` on every POST. Default 10 MB; reject 413 above the cap.
3. **MIME allowlist:** `AllowedMimeTypes = {application/pdf, application/json, text/plain, text/markdown, application/vnd.openxmlformats-officedocument.wordprocessingml.document, image/png, image/jpeg, image/webp}`. Reject 415 otherwise.
4. **Blob keys are deterministic + scoped:** `attachments/{sessionID}/{attachmentID}/{slug(displayName)}`. Direct enumeration impossible without knowing both UUIDs.
5. **No credential values in DB or logs:** the embeddings provider and blobstore both resolve credentials via `config.Get*CredentialRef()` → `os.Getenv` (single call site).
6. **Presigned URLs expire:** blob redirects use 15-minute presigned URLs; raw bucket access is disabled.
7. **Cascade deletion:** session deletion → attachments cascade → chunks cascade → blob deletion runs as a post-commit best-effort sweep (logged on failure).
8. **No agent-to-agent leakage:** agent-scope retrieval filters by `agent_id` — agent A's snippet is invisible to agent B even within the same iteration.

#### 8.28.10 Failure Modes & Degradation

| Failure                                  | Behaviour                                                           |
| ---------------------------------------- | ------------------------------------------------------------------- |
| Extractor fails (corrupt PDF, blank URL) | 422 to client; no DB write; no blob write                           |
| Embedding API down                       | Upload fails 503; no partial DB state (transaction-wrapped)         |
| Blobstore down                           | Upload fails 503; no DB write                                       |
| Retrieval query times out at dispatch    | Engine logs warn + dispatches with `Attachments = []` (no abort)    |
| `len(chunks) == 0` for any reason        | Fall back to `attachment.summary` as a single synthetic chunk       |
| Vision model unavailable for image       | Service returns 422; UI surfaces "Image extraction unavailable"     |
| Iteration-cleanup `DeleteByScope` fails  | Logged warn; rows orphan until session delete (eventually cascaded) |

#### 8.28.11 Config Summary

| Env var                          | Default                  | Range / format              | Owner   |
| -------------------------------- | ------------------------ | --------------------------- | ------- |
| `EMBEDDINGS_PROVIDER`            | `openai`                 | `openai` \| `ollama`        | Task 38 |
| `EMBEDDINGS_MODEL`               | `text-embedding-3-small` | provider-specific           | Task 38 |
| `EMBEDDINGS_CREDENTIAL_REF`      | `OPENAI_API_KEY`         | env var name only           | Task 38 |
| `EMBEDDINGS_DIMENSION`           | `1536`                   | `[64, 4096]`                | Task 38 |
| `BLOBSTORE_ENDPOINT`             | `http://minio:9000`      | URL                         | Task 38 |
| `BLOBSTORE_ACCESS_KEY_REF`       | `MINIO_ROOT_USER`        | env var name only           | Task 38 |
| `BLOBSTORE_SECRET_KEY_REF`       | `MINIO_ROOT_PASSWORD`    | env var name only           | Task 38 |
| `BLOBSTORE_BUCKET`               | `a2a-attachments`        | DNS-safe                    | Task 38 |
| `BLOBSTORE_USE_SSL`              | `false`                  | bool                        | Task 38 |
| `ATTACHMENT_MAX_BYTES`           | `10485760` (10 MB)       | `[1024, 104857600]`         | Task 38 |
| `ATTACHMENT_CHUNK_SIZE`          | `1000`                   | tokens, `[100, 4000]`       | Task 38 |
| `ATTACHMENT_CHUNK_OVERLAP`       | `150`                    | tokens, `[0, chunk_size/2]` | Task 38 |
| `ATTACHMENT_RETRIEVAL_TOP_K`     | `5`                      | `[1, 20]`                   | Task 38 |
| `ATTACHMENT_MIN_EXTRACTED_CHARS` | `16`                     | `[1, 1000]`                 | Task 39 |

All getters live in `backend/internal/platform/config/config.go` — no `os.Getenv` calls elsewhere.

#### 8.28.12 Determinism Trade-off

Attachment retrieval introduces a non-deterministic surface — same input idea + same attached corpus may rank chunks slightly differently between embedding model versions or pgvector index rebuilds. This is acceptable because:

- Attachment-driven prompts are advisory context, not protocol state. Canonical state merging (§8.5) is untouched.
- Top-K ordering is **stable within one run** (single SQL `ORDER BY embedding <=> $1`).
- For CI / snapshot tests, attachments default to none; the iteration engine's existing determinism contract is preserved when `len(attachments) == 0`.

---

### 8.29 Guided Onboarding v2, Plan-Only Outputs, Artifact Cache, and Dual-Audience Docs (v1.8)

**Status:** Added in v1.8 with Task 34. Visual source of truth: `frontend/mockups/v2.html`.

#### 8.29.1 Onboarding Flow (locked order)

```
Step 1 — Home (sync, no LLM)
  ├─ Product idea textarea (required, 20–4000 chars)
  ├─ Max iterations + agent pool (existing)
  └─ Tech constraints block (optional, §8.29.3)

Step 2 — Clarify (user-first, optional per question)
  ├─ Q1 free text: persona + current workaround
  ├─ Q2 chips: MVP must-haves before first real user
  ├─ Q3 chips: non-negotiable requirements
  ├─ Q4 chips: value proposition vs status quo
  └─ Parallel async: POST /sessions/discovery-hints (chip labels only)

Step 3 — POST /sessions
  ├─ Compile enriched_idea (deterministic, §8.29.2)
  ├─ Seed initial CanonicalState (§8.29.2)
  └─ Start iteration pipeline (unchanged)
```

**Rule:** The LLM never pre-fills discovery answers. `discovery-hints` returns label strings for Q2–Q4 chips only. User selections and free text are authoritative.

#### 8.29.2 Discovery Compiler + Initial State Seeding

`CompileEnrichedIdea` produces deterministic Markdown:

```markdown
## Product Idea

{raw idea}

## Discovery

### Persona & current state

{q1 or _skipped_}

### MVP must-haves

- {chip selections + other}

### Non-negotiables

- ...

### Value proposition

- ...

## Tech Constraints

{formatted constraint lines or _Agents will decide stack_}
```

`SeedInitialState` maps answers into canonical JSON before iteration 1:

| Source           | Canonical field                                                         |
| ---------------- | ----------------------------------------------------------------------- |
| Q1               | `idea.persona`, `idea.context`                                          |
| Q2               | `idea.mvp_must_haves[]`, seeds `execution_plan[0]` title when non-empty |
| Q3               | `assumptions[]` prefixed `Non-negotiable:`                              |
| Q4               | `idea.value_props[]`                                                    |
| Tech constraints | `assumptions[]` via `TechConstraints.ToAssumptions()`                   |

`state.Validate` runs on seeded state at session create; invalid seed returns HTTP 400.

#### 8.29.3 TechConstraints JSON Shape

```json
{
  "agents_decide": true,
  "must_use": ["Go", "PostgreSQL"],
  "comfortable_with": ["SvelteKit", "Docker"],
  "avoid_if_possible": ["Microservices", "Kubernetes"]
}
```

When `agents_decide: true`, the three arrays are ignored (may be empty). When `false`, each non-empty array appends formatted lines to `assumptions[]`:

- `Must use: Go, PostgreSQL`
- `Comfortable with: SvelteKit, Docker`
- `Avoid if possible: Microservices, Kubernetes`

Agents may challenge constraints during iteration; challenged items move to `open_questions[]` per existing merge rules (§8.5).

#### 8.29.4 Discovery Hints Endpoint

```
POST /sessions/discovery-hints
Body: { "idea": "..." }
Response: { "q2": ["..."], "q3": ["..."], "q4": ["..."] }
```

- Validates idea length `[20, 4000]`
- One `LLMProvider.Generate` call; temperature from `GetDiscoveryHintsTemperature()` (default `0.0`)
- LRU cache keyed by `sha256(idea)`; size from `GetDiscoveryHintsCacheSize()` (default `128`)
- On LLM failure: HTTP 200 with static defaults from v2 mockup (UI already shows these; response is a no-op merge)
- Hints are **not** stored in DB — only user answers persist

#### 8.29.5 Plan-Only Output Model (roadmap deprecated)

| Before (v1.7)                                  | After (v1.8)                                      |
| ---------------------------------------------- | ------------------------------------------------- |
| `architecture` + `roadmap` + `plan` + `readme` | `architecture` + `plan` + `readme`                |
| `roadmap.md` — milestones + phases             | Absorbed into `plan.md`                           |
| `plan.md` — tasks / modules                    | `plan.md` — milestones + phases + tasks / modules |

`AllowedOutputDocs` removes `roadmap`. `DefaultOutputDocs` = `["architecture", "plan"]`.

`plan.md` required sections (minimum):

1. Goals
2. Milestones (table from `execution_plan`)
3. Phase Breakdown (seven §8.23 fields per step)
4. Cross-phase Dependencies
5. Module / Task Breakdown
6. Risks
7. **For AI Agents** (machine-oriented appendix — see below)

#### 8.29.6 Dual-Audience Document Quality Standard

Every generated document (`architecture`, `plan`, `readme`) must satisfy **both**:

**Human readability**

- Prose paragraphs (not bullet-only dumps) in Overview sections
- Tables for comparisons (ADRs, milestones, risks)
- Mermaid diagrams where flows exist
- No placeholder tokens (`TBD`, `TODO`, `placeholder`)
- Section headings form a logical outline a staff engineer can skim in ≤10 minutes

**AI-agent readiness**

- Trailing appendix `## For AI Agents` on each doc containing:
  - `### Stack` — bullet list of technologies with versions when known
  - `### Key Contracts` — API boundaries, module paths, file conventions
  - `### Implementation Order` — numbered steps an agent should follow
  - `### Out of Scope` — explicit non-goals
- Filenames remain `{slug}_{doc_key}.md` (Task 32 rule)
- Content must be pasteable into Claude Code / Cursor / Copilot without preprocessing

Rubric minimums (extends §8.27):

| Doc key        | MinTotalLines | New required headings                                                                          |
| -------------- | ------------- | ---------------------------------------------------------------------------------------------- |
| `architecture` | 1000          | `2. System Components` (one `###` per component), `6. Architecture Decisions`, `For AI Agents` |
| `plan`         | 1000          | `2. Milestones`, `3. Phase Breakdown`, `5. Module Tasks`, `For AI Agents`                      |
| `readme`       | 1000          | `Getting Started`, `For AI Agents`                                                             |

#### 8.29.7 Session Artifact Cache

```sql
-- migrations/008_session_artifacts.sql
CREATE TABLE session_artifacts (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id    UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  doc_key       TEXT NOT NULL,
  filename      TEXT NOT NULL,
  content       TEXT NOT NULL,
  line_count    INT  NOT NULL,
  source        TEXT NOT NULL CHECK (source IN ('deterministic','hybrid','ai')),
  generated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (session_id, doc_key)
);
```

**Write path:** `FinalizeSession` → `Orchestrator.GenerateAll` → `UpsertArtifacts` (one row per doc key).

**Read path:**

- `GET /sessions/{id}/artifacts` — returns all rows for a session
- Finalize page: on mount, if `status ∈ {converged, approved}` and artifacts exist → render immediately (badge: "Loaded from previously generated session")
- Regenerate: explicit user action only; successful run replaces rows in place

**History UX:**

- List view output chips: `arch.md`, `plan.md` (no `roadmap.md`)
- **View →** opens `/session/{id}/finalize` (not live workspace)
- Approved finalize page: **no** "← Back to Session" button; **New Session** + **Download All** remain

#### 8.29.8 Determinism Notes

- `CompileEnrichedIdea` + `SeedInitialState` are fully deterministic
- `discovery-hints` is advisory UI only — not part of canonical state merge
- Artifact cache stores generator output; re-fetch is deterministic per cached row
- Removing `roadmap` does not change iteration engine determinism

---

### 8.30 Architecture.md Enrichment — Section Contract + LLM Enricher (v1.9)

Added by Task 35. This section documents the full 16-section architecture document contract, the `ArchEnricher` pre-pass that populates optional state fields before the deterministic render, and the updated rubric for the `architecture` key.

---

#### 8.30.1 Section Sequence (architecture.md)

The generator renders sections in this exact order. Heading constants live in `generator_architecture.go`. Sections without a number (metadata block, TOC, appendix) use no `##` heading.

| # | Heading | Primary Source | Fallback |
|---|---------|----------------|---------|
| — | Metadata block (blockquote) | Deterministic: `s.Meta.Iteration`, `s.Metrics.Confidence` | Always present |
| — | Table of Contents | Deterministic: hardcoded 16 entries | Always present |
| 1 | Problem Statement | `s.Idea["problem_statement"]` (enricher) | First sentence of `s.Idea["context"]` ≤200 chars → `s.Idea["text"]` ≤200 chars |
| 2 | Solution | `s.Idea["solution_summary"]` (enricher) | `s.Idea["summary"]` → first 300 chars of `s.Idea["text"]` |
| 3 | Scope | `s.Architecture["scope"]["in"]` + `["out"]` (enricher) | In: first 3 goals from `s.Idea["goals"]`; Out: "Further feature scope — defined in future iterations" |
| 4 | Layers | `s.Architecture["layers"]` (existing) | `_Architecture details not yet defined._` |
| 5 | Tech Stack | `s.Architecture["tech_stack"]` + `s.Architecture["tech_stack_rationale"]` (enricher) | 2-column table from `tech_stack` keys when rationale absent |
| 6 | Data Flows | `s.Architecture["data_flows"]` + `s.Architecture["data_flow_prose"]` (enricher) | ASCII component diagram fallback |
| 7 | Module Boundaries | `s.Architecture["directory_layout"]` (existing) | Generic module listing |
| 8 | Architecture Decisions | `s.Architecture["decisions"]` + `s.Architecture["decision_enrichments"]` (enricher) | 4-column table without Alternatives/Tradeoff when enrichment absent |
| 9 | Extension Points | `s.Architecture["extension_points"]` (enricher) | Generic stub: "Add a new LLM provider" + "Add a new output document format" each with 3 placeholder steps |
| 10 | Security Considerations | `s.Architecture["security"]` (enricher) | Generic stub: authentication + prompt injection rows |
| 11 | Quality Targets | `s.Metrics` (deterministic) | Always present |
| 12 | System Guarantees | `s.Architecture["guarantees"]` (enricher) | 3 derived guarantees from architecture map (see §8.30.6) |
| 13 | Risks | `s.Risks[]` + Likelihood/Impact/Mitigation (enricher overlay) | Existing 4-column table; new columns show "—" when absent |
| 14 | Assumptions | `s.Assumptions` (deterministic) | `_No assumptions recorded._` |
| 15 | Open Questions | `s.OpenQuestions` (deterministic) | `_No open questions at this time._` |
| 16 | Definition of Done | `s.Architecture["exit_criteria"]` (enricher) + always-appended items | 5 generic DoD items; "All open questions resolved"; "Confidence ≥ 0.85" |
| — | Appendix: For AI Agents | Real layer names + LLM interface name from state | Hardcoded canonical list |

---

#### 8.30.2 ArchEnrichmentOverlay Schema

The LLM enricher call returns a JSON object matching this schema. All fields are optional — the enricher merges only non-empty fields into the state copy. Defined in `backend/internal/modules/markdown/aigen/enricher.go`.

```go
type ArchEnrichmentOverlay struct {
    ProblemStatement    string               `json:"problem_statement"`    // ≤400 chars
    SolutionSummary     string               `json:"solution_summary"`     // ≤400 chars
    ScopeIn             []string             `json:"scope_in"`             // ≤8 items, each ≤80 chars
    ScopeOut            []string             `json:"scope_out"`            // ≤8 items, each ≤80 chars
    ExtensionPoints     []OverlayExtPoint    `json:"extension_points"`     // ≤4 items
    Security            []OverlaySecEntry    `json:"security"`             // ≤6 items
    Guarantees          []string             `json:"guarantees"`           // ≤6 items, each ≤120 chars
    DecisionEnrichments []OverlayDecEnrich   `json:"decision_enrichments"` // ≤N items (N = decisions count)
    TechStackRationale  map[string]string    `json:"tech_stack_rationale"` // key = tech name; value ≤100 chars
    DataFlowProse       string               `json:"data_flow_prose"`      // ≤300 chars
    ExitCriteria        []string             `json:"exit_criteria"`        // ≤8 items, each ≤100 chars
}

type OverlayExtPoint struct {
    Name  string   `json:"name"`  // ≤60 chars
    Steps []string `json:"steps"` // 2–5 items, each ≤100 chars
}

type OverlaySecEntry struct {
    Surface    string `json:"surface"`    // ≤60 chars
    Risk       string `json:"risk"`       // ≤120 chars
    Mitigation string `json:"mitigation"` // ≤120 chars
}

type OverlayDecEnrich struct {
    Title        string `json:"title"`        // must match an existing decision title exactly
    Alternatives string `json:"alternatives"` // ≤150 chars
    Tradeoff     string `json:"tradeoff"`     // ≤150 chars
}
```

**Enricher merge rule:** For each field, if `s.Idea[key]` or `s.Architecture[key]` is already non-nil and non-empty, skip — do not overwrite. Enricher fills gaps only; human-curated state always takes precedence.

**Enricher error contract:** On any error (LLM failure, timeout, JSON parse error, validation failure), `Enrich` returns the original `s` unchanged and logs at `slog.Warn`. No error is propagated to the caller. The generator always has valid state to render from.

**Enricher LLM input JSON** (compact — only surface info, not full text bodies):

```json
{
  "idea_text": "<first 300 chars of s.Idea[\"text\"]>",
  "idea_context": "<first 300 chars of s.Idea[\"context\"]>",
  "idea_goals": ["<goal1>", "..."],
  "tech_stack_keys": ["Go", "SvelteKit", "..."],
  "layer_names": ["frontend", "backend", "..."],
  "decision_titles": ["Use modular monolith", "..."],
  "risk_texts": ["<risk1.text>", "..."]
}
```

---

#### 8.30.3 Updated Architecture Rubric

These targets are enforced by the rubric validator in `aigen/rubric.go` during the AI-hybrid pass (Task 33 infrastructure). Sections are validated by substring search for `RequiredKeywords` and total document `MinChars`.

| Section heading | RequiredKeywords | Section MinChars |
|----------------|-----------------|-----------------|
| Problem Statement | — | 80 |
| Solution | — | 80 |
| Scope | `"In Scope"`, `"Out of Scope"` | 60 |
| Layers | — | 100 |
| Tech Stack | — | 80 |
| Data Flows | `"graph"` or `"→"` | 100 |
| Module Boundaries | — | 60 |
| Architecture Decisions | `"Decision"`, `"Status"` | 80 |
| Extension Points | — | 60 |
| Security Considerations | `"Surface"`, `"Mitigation"` | 60 |
| Quality Targets | `"Confidence"` | 40 |
| System Guarantees | — | 60 |
| Risks | `"Risk"`, `"Status"` | 60 |
| Assumptions | — | 20 |
| Open Questions | — | 20 |
| Definition of Done | `"- [ ]"` | 60 |

**Total document MinChars:** 1500 (raised from previous 500).

---

#### 8.30.4 New Optional State Fields

These fields are set inside `s.Idea` and `s.Architecture` maps by the enricher. They are advisory — the generator renders all 16 sections without them (using fallbacks from §8.30.6).

| Field key | Map | Go type after JSON decode | Used by section |
|-----------|-----|--------------------------|----------------|
| `problem_statement` | `s.Idea` | `string` | §1 Problem Statement |
| `solution_summary` | `s.Idea` | `string` | §2 Solution |
| `scope` | `s.Architecture` | `map[string]any` — keys `"in"` ([]string), `"out"` ([]string) | §3 Scope |
| `extension_points` | `s.Architecture` | `[]any` — each `{name: string, steps: []string}` | §9 Extension Points |
| `security` | `s.Architecture` | `[]any` — each `{surface, risk, mitigation: string}` | §10 Security |
| `guarantees` | `s.Architecture` | `[]string` | §12 System Guarantees |
| `decision_enrichments` | `s.Architecture` | `[]any` — each `{title, alternatives, tradeoff: string}` | §8 Architecture Decisions |
| `tech_stack_rationale` | `s.Architecture` | `map[string]string` | §5 Tech Stack |
| `data_flow_prose` | `s.Architecture` | `string` | §6 Data Flows |
| `exit_criteria` | `s.Architecture` | `[]string` | §16 Definition of Done |

---

#### 8.30.5 Enricher LLM System Prompt (canonical)

The system prompt sent by `ArchEnricher.Enrich`. Stored here so it can be reviewed and updated independently of Go code.

```
You are an architecture documentation assistant. Given a compact JSON description of a software project's current brainstorm state, return a JSON object that fills in the gaps needed to render a high-quality architecture.md document.

Rules:
- Return ONLY valid JSON matching the output schema. No markdown fencing. No prose outside the JSON.
- Keep all strings concise — respect the per-field character limits in the schema.
- Do NOT invent technical facts not implied by the input. Infer from what is present.
- If a field cannot be meaningfully populated from the input, omit it entirely.
- `decision_enrichments[].title` must EXACTLY match one of the decision titles in the input.

Output schema (all fields optional):
{
  "problem_statement": "string ≤400 chars",
  "solution_summary": "string ≤400 chars",
  "scope_in": ["string ≤80 chars"],
  "scope_out": ["string ≤80 chars"],
  "extension_points": [{"name": "string ≤60 chars", "steps": ["string ≤100 chars"]}],
  "security": [{"surface": "≤60", "risk": "≤120", "mitigation": "≤120"}],
  "guarantees": ["string ≤120 chars"],
  "decision_enrichments": [{"title": "exact match", "alternatives": "≤150", "tradeoff": "≤150"}],
  "tech_stack_rationale": {"tech_name": "rationale ≤100 chars"},
  "data_flow_prose": "string ≤300 chars",
  "exit_criteria": ["string ≤100 chars"]
}
```

---

#### 8.30.6 Fallback Specification per Section

Each section's deterministic fallback content when enricher fields are absent:

| Section | Fallback behaviour |
|---------|-------------------|
| Problem Statement | `firstSentence(s.Idea["context"])` truncated to 200 chars; if absent, `truncate(s.Idea["text"], 200)` |
| Solution | `s.Idea["summary"]` if non-empty; else `truncate(s.Idea["text"], 300)` |
| Scope | In: first 3 items of `s.Idea["goals"]` (or "Core feature" if goals empty); Out: single row "Further feature scope — defined in future iterations" |
| Extension Points | 2 stub entries: (1) "Add a new LLM provider" with steps — "Implement the `LLMProvider` interface", "Register in `platform/llm/resolver.go`", "Add env var to `config.go`"; (2) "Add a new output document format" with steps — "Implement `Generate<DocType>(s CanonicalState) (string, error)`", "Register key in `markdown/generator.go` `GenerateAll`", "Add key to session `output_docs` allowed values" |
| Security Considerations | 2 stub rows: `{Surface: "Authentication", Risk: "Unauthorized access to session data", Mitigation: "Validate all session IDs against DB on every request"}`, `{Surface: "Prompt Injection", Risk: "Malicious user input manipulates LLM output", Mitigation: "Validate and sanitise all user-controlled input before LLM prompt assembly"}` |
| System Guarantees | 3 hardcoded strings: "Deterministic output: same canonical state input produces identical generated documents", "Isolated modules: no cross-module direct imports between `internal/modules/` packages", "LLM abstracted: all model calls routed through the `LLMProvider` interface — no direct SDK usage in business logic" |
| Definition of Done | 5 generic items: "Core feature implemented and manually tested end-to-end", "All unit and integration tests passing (0 failures)", "`go build ./...` and `go vet ./...` clean", "No hardcoded credentials, ports, or model names outside config files", "Architecture document generated, reviewed, and committed"; plus always-appended: "All open questions resolved", "Confidence ≥ 0.85" |
| Risks Mitigation / Likelihood / Impact | Render "—" in table cells; preserve existing `Severity` value |
| Architecture Decisions Alternatives / Tradeoff | Render "—" in Alternatives and Tradeoff columns |
| Tech Stack Rationale column | Omit Rationale column entirely; render 2-column table (Technology, Role/Version) |
| Data Flow Prose paragraph | Omit prose paragraph entirely; render Mermaid diagram directly without preamble |

---

### 8.31 PLAN.md Generation Quality Standard (v2.0)

This section documents the canonical task section format, the `PlanEnrichmentOverlay` schema, the Approach C agent prompt fragment, and the quality assertion rules that `generator_plan.go` and `plan_enricher.go` (Task 36) must satisfy. Every task session implementing Task 36 must include this section.

---

#### 8.31.1 Canonical Task Section Format

Each `### Task N — {Name}` block in a generated `plan.md` **must** contain the following fields in this order:

```markdown
### Task N — {Name}

**Goal:** {One sentence ≤ 120 chars: what this task produces.}

**Layer(s) affected:** `backend/internal/modules/X/`, `agent/internal/Y/`

**Files to create:**

- `path/to/file.go` — {description}
  - {key export, interface, or Go function signature}
  - **Failure handling:** {error paths, retry logic, non-fatal contract}

**Coding standards:** _(Medium/High complexity tasks only)_
- SRP: {what has a single responsibility in this task}
- DIP: {what depends on abstraction rather than concrete}
- YAGNI: {what is intentionally NOT implemented}
- No `fmt.Println`, no `os.Getenv` outside `config.go`

**Validation:**
- `cd backend && go build ./...`: zero build errors
- `cd backend && go vet ./...`: zero vet issues
- `cd backend && go test ./path/...`: {specific test target} — all pass
- {Behavioral assertion checkpoint — e.g. manual smoke test}

**Invariant check:**
- [ ] No file modified outside the "Files to create/modify" list
- [ ] No cross-module import introduced
- [ ] No `os.Getenv` outside `config.go`
- [ ] All SQL via parameterized queries only (if DB-touching)
- [ ] {1–2 task-specific invariants derived from the task's domain}

**Prompt context needed:** §8.X (description), Task N (dependency name)

---
```

**Forbidden strings** — the generator must reject any output containing:
- `"see phase deliverables"` — stub placeholder that provides zero actionable information
- `"per module test suite"` — not a runnable command
- `"Phase 0 —"`, `"Phase 1 —"`, ..., `"Phase N —"` — wrong heading format (must be `"Task N —"`)
- `"TBD"`, `"TODO"`, `"placeholder"` — incomplete spec

**Required patterns** per `### Task` block (validator asserts at least one occurrence of each):
- `\*\*Goal:\*\*`
- `\*\*Validation:\*\*`
- `\*\*Invariant check:\*\*`
- `go build ./...` (in Validation for any Go-touching task)

---

#### 8.31.2 PlanEnrichmentOverlay Schema

The LLM enricher returns a single JSON object matching this schema. All fields are optional; empty/missing = use deterministic fallback.

```json
{
  "phases": [
    {
      "position": 0,
      "coding_standards": [
        "SRP: {specific responsibility of primary file}",
        "DIP: {what depends on abstraction}",
        "YAGNI: {what is not implemented}"
      ],
      "invariant_checks": [
        "{task-specific invariant check text — starts with verb, ≤ 80 chars}"
      ],
      "layer_tags": [
        "backend/internal/modules/X/",
        "agent/internal/Y/"
      ],
      "prompt_context_refs": [
        "§8.N (description of what to load from this section)"
      ]
    }
  ],
  "dependency_graph_ascii": "Task 1 ─► Task 2 ─► Task 3\n              │\n              ▼\n         Task 4",
  "deep_knowledge_sections": [
    {
      "heading": "8.N {Title}",
      "content": "{Markdown content — schemas, algorithms, contracts relevant to task sessions}"
    }
  ]
}
```

**Validation rules for `validateOverlay()`:**
- `phases` length must match `s.Plan["phases"]` length exactly; extra/missing entries → reject whole overlay
- Each `coding_standards` item: non-empty string, ≤ 200 chars
- Each `invariant_checks` item: non-empty string, starts with a capital letter, ≤ 80 chars
- Each `layer_tags` item: must match `^(backend|agent|frontend)/` prefix
- `dependency_graph_ascii`: non-empty string, ≤ 3000 chars
- Any `deep_knowledge_sections[*].content`: non-empty, ≤ 5000 chars per entry
- Reject slices with > 30 items (prevents runaway enrichment)

---

#### 8.31.3 Approach C — Agent Prompt Fragment (PlanTaskFormat)

The full constant text injected into agent system prompts when `"plan"` is in `output_docs`:

```
## PLAN.md Output Format Requirements

When writing tasks for the implementation plan (plan.md output document), use ONLY the following format. Do NOT use "Phase N" headings.

### Task N — {Short Task Name}

**Goal:** {One sentence, ≤ 120 chars, starting with a verb: what this task produces.}

**Files to create:**
- `path/to/file.go` — {description}
  - `func FunctionName(params) (ReturnType, error)` — exported function signature
  - `type TypeName struct { ... }` — key exported type
  - **Failure handling:** {how errors are returned/logged; which are fatal vs non-fatal}

**Validation:**
- `cd backend && go build ./...`: zero build errors
- `cd backend && go vet ./...`: zero vet issues
- `cd backend && go test ./internal/modules/X/...`: all pass
- Manual smoke: {one behavioral assertion}

**Blocking Dependencies:** Task N, Task M

---

FORBIDDEN in Validation field: "per module test suite", "run tests", "validate implementation"
FORBIDDEN in Files field: "see phase deliverables", "TBD", "as needed"
REQUIRED in every task: at least one runnable shell command in Validation
REQUIRED for Go tasks: `go build ./...` in Validation
REQUIRED for frontend tasks: `pnpm check` in Validation
```

---

#### 8.31.4 Plan Enricher LLM System Prompt

The system prompt passed to `llm.Generate` in `PlanEnricher.Enrich`:

```
You are a software architecture assistant. You receive a summary of implementation plan phases
and return a structured JSON enrichment that improves the AI-agent readability of each task.

For each phase, produce:
- coding_standards: 3-5 SOLID/YAGNI/DRY principles applied to THIS specific phase's code
- invariant_checks: 2-4 task-specific architectural invariants (in addition to the canonical 4)
- layer_tags: module paths affected (backend/internal/modules/X/, agent/internal/Y/, frontend/src/...)
- prompt_context_refs: §8.N references an agent needs to execute this task

Also produce:
- dependency_graph_ascii: ASCII art showing task order and parallelism (use ─►, │, ▼, ┌, ┐, └, ┘)
- deep_knowledge_sections: 2-4 §8 entries (schemas, algorithms, contracts) needed across the task sessions

Return ONLY valid JSON matching the schema. No prose. No markdown fences.

Respond in under 4000 tokens. Prefer concrete specifics over generic platitudes.
For coding_standards: name the actual function/struct, not "SRP: keep it focused".
For invariant_checks: start with a verb ("No X outside Y", "All Z via W only").
```

---

#### 8.31.5 Quality Assertions (Auto-Repair Loop Integration)

The rubric validator in `markdown/aigen/rubric.go` for key `"plan"` enforces:

| Assertion | Type | Action on Failure |
|---|---|---|
| No `"see phase deliverables"` in output | `ForbiddenString` | Trigger auto-repair LLM pass: "Replace all 'see phase deliverables' stubs with explicit file paths from the canonical state" |
| No `"per module test suite"` in output | `ForbiddenString` | Trigger auto-repair: "Replace with specific `go build ./...` and `go test ./path/...` commands" |
| No `"Phase [0-9]+"` in `### ` headings | `ForbiddenPattern` | Trigger auto-repair: "Rename all 'Phase N' headings to 'Task N'" |
| `\*\*Goal:\*\*` present in each `### Task` block | `RequiredPattern` | Trigger auto-repair: "Add missing **Goal:** field to each task section" |
| `\*\*Validation:\*\*` present in each `### Task` block | `RequiredPattern` | Trigger auto-repair: "Add missing **Validation:** field" |
| `\*\*Invariant check:\*\*` present in each `### Task` block | `RequiredPattern` | Trigger auto-repair: "Add missing **Invariant check:** field with at least the 4 canonical items" |
| `## 1. Goals` is not `_Goals not yet defined_` | `ForbiddenString` | Trigger auto-repair: "Populate Goals from `s.Idea['goals']` list" |
| `## 8. Deep Knowledge` section present | `RequiredSection` | Trigger auto-repair: "Add §8 Deep Knowledge section derived from canonical state schema" |
| Document `MinChars` ≥ 3000 | `MinLength` | Trigger auto-repair: "Expand task file specs with explicit function signatures and failure handling" |

Maximum 3 auto-repair attempts before falling back to deterministic-only output with a warning log.

---

### 8.32 README.md Generation Quality Standard (v3.0)

Added by Task 37. This section documents the full 13-section README document contract, the `ReadmeEnricher` post-pass that LLM-fills content-heavy sections after the deterministic render, the Approach C agent prompt fragment that steers upstream LLM agents to write README-appropriate content, and the rubric assertions with auto-repair loop.

---

#### 8.32.1 README Section Contract

Every generated `readme.md` must contain exactly these 13 sections in this order. Sections with a **Deterministic fallback** column will render content from `CanonicalState` even when the enricher is disabled or unavailable.

| § | Heading | Primary data source (enricher key) | Deterministic fallback |
|---|---|---|---|
| 1 | `# {ShortTitle}` | `shortTitle(s)` — deterministic | Title from `s.Idea["text"]` truncated + slug-cased |
| 2 | `> {tagline}` | `s.Architecture["tagline"]` | `oneLineDescription(s)` |
| 3 | **Description paragraph** | `s.Architecture["description_paragraph"]` | First 300 chars of `s.Idea["context"]` |
| 4 | **Golden rule** | `s.Architecture["golden_rule"]` | `_No golden rule defined._` |
| 5 | **What it is NOT** | `s.Architecture["is_not"]` ([]string) | 3 generic anti-patterns derived from tech stack |
| 6 | **When to use** | `s.Architecture["when_to_use"]` ([]WhenToUseScenario) + `s.Architecture["when_to_use_mermaid"]` | 3 generic "use when building X" scenarios; no Mermaid |
| 7 | **Installation / Prerequisites** | `s.Architecture["prerequisites"]` ([]Prerequisite) | Generic `go install` + `docker compose up` steps |
| 8 | **Quick Start** | `s.Architecture["quick_start_commands"]` ([]string) | 3 generic `make` commands |
| 9 | **Architecture** | `s.Architecture["architecture_ascii"]` + `s.Architecture["architecture_mermaid"]` | `renderASCIIComponents(s)` + `renderDataFlowsMermaid(s)` |
| 10 | **Command Reference** | `s.Architecture["command_reference"]` ([]CommandRef) | Single placeholder row |
| 11 | **Tech Stack** | `renderTechStack(s)` — deterministic | Same |
| 12 | **Repository Format** | `renderDirectoryTree(s)` + `s.Architecture["repo_format_notes"]` | Tree only, no per-entry notes |
| 13 | **Contributing** | `s.Architecture["contributing_note"]` | Generic `Read AGENTS.md first` line |

**Removed from the old generator (must never appear in v3.0+ output):**
- `## Known Risks` as a top-level README section — risks belong in `architecture.md` only
- `## Roadmap` with "Phase N —" bullets — belongs in `roadmap.md` only
- `renderForAIAgentsAppendix` output — README is human-facing; the For AI Agents block must never appear
- `— README` suffix on the H1 title

---

#### 8.32.2 ReadmeEnrichmentOverlay Schema

The enricher receives `CanonicalState` as JSON and returns a single structured JSON object. All fields are optional — missing fields leave the deterministic fallback in place.

```json
{
  "tagline": "string ≤ 120 chars — punchy one-sentence product tagline",
  "description_paragraph": "string — 2-3 sentences: what it does, who uses it, what problem it solves",
  "golden_rule": "string — one-sentence core invariant (e.g. 'same input + same config = identical output')",
  "is_not": ["string", "string", "string"],
  "when_to_use": [
    {
      "title": "string — short scenario name (e.g. 'Rapid API prototyping')",
      "description": "string — 1-2 sentence explanation",
      "code": "string — bash/shell command or config snippet demonstrating this scenario"
    }
  ],
  "when_to_use_mermaid": "string — complete mermaid flowchart source starting with 'flowchart LR' or 'flowchart TD'",
  "quick_start_commands": ["string", "string"],
  "command_reference": [
    {
      "command": "string — exact CLI/make/curl command",
      "description": "string — one-line explanation"
    }
  ],
  "architecture_ascii": "string — multi-line ASCII diagram (no code fences, caller wraps in ```)",
  "architecture_mermaid": "string — complete mermaid flowchart/graph source",
  "prerequisites": [
    {
      "tool": "string — e.g. Go, Docker, Node.js",
      "version": "string — e.g. 1.26+",
      "required": true
    }
  ],
  "repo_format_notes": {
    "backend/": "string — one-line description of what this directory contains",
    "frontend/": "string"
  },
  "contributing_note": "string — 1-3 sentences: what to read before contributing"
}
```

**Merge rules:**
- Every overlay key that is non-empty overwrites the corresponding `s.Architecture[key]` entry.
- Keys absent or null in the overlay leave the existing state value untouched.
- The enricher must NEVER overwrite `s.Metrics`, `s.Risks`, `s.Assumptions`, `s.OpenQuestions`, or `s.ExecutionPlan`.

---

#### 8.32.3 Enricher LLM System Prompt

```
You are an expert technical writer generating a README enrichment overlay for a software project.

INPUT: A CanonicalState JSON object describing a project being designed: its idea, architecture decisions, tech stack, modules, and metrics.

OUTPUT: A single JSON object conforming to ReadmeEnrichmentOverlay (all fields optional).

RULES:
1. tagline: ≤ 120 chars. One punchy sentence. No marketing fluff. Should complete: "{ProductName} is a ..."
2. golden_rule: One sentence stating the single most important invariant — the property that must always hold. Example: "same input + same config = identical output, always."
3. is_not: 3-5 short bullets clarifying what the product is NOT (scope boundaries, common misconceptions). Start each with "Not a ..." or "Not designed to ...".
4. when_to_use: 3-6 numbered scenarios. Each must have a realistic `code` example — a real command or config snippet, not pseudo-code. Scenarios must be distinct (different use cases, not variations of the same thing).
5. when_to_use_mermaid: A mermaid flowchart (flowchart LR or TD) showing the decision path for when to use this product vs alternatives. 5-9 nodes. Use `-->` for arrows and `[text]` for rectangles, `{text}` for diamonds.
6. quick_start_commands: 2-5 commands. Real commands for the tech stack described in the state (e.g., `make dev`, `docker compose up`, `go run ./cmd/server`). Never use `<repository-url>` or `<project>` placeholders.
7. command_reference: 3-10 entries. Cover the most important developer-facing commands (build, test, run, deploy, migrate).
8. architecture_ascii: A clean ASCII diagram showing the major system components and their connections. Use box-drawing chars (─ │ ┌ └ ┐ ┘ ├ ┤ ┬ ┴ ┼) for clarity. 8-15 lines.
9. architecture_mermaid: A mermaid diagram (flowchart or graph) of the architecture. Match the ASCII above.
10. prerequisites: List only tools actually required by the tech stack. Include versions where known.
11. contributing_note: 2-3 sentences. What to read before contributing (docs, skill files, test commands).

QUALITY BAR: Content must be specific to THIS project (based on its idea, architecture, tech stack). Generic placeholder text is forbidden.
```

---

#### 8.32.4 Config Table (additions to §8.28)

| Env var | Default | Valid values | Added by |
|---|---|---|---|
| `README_ENRICHER_ENABLED` | `true` | `true` \| `false` | Task 37 |
| `README_ENRICHER_TIMEOUT_SEC` | `45` | `[10, 120]` | Task 37 |

When `FINALIZE_MODE=deterministic`, `GetReadmeEnricherEnabled()` must return `false` regardless of `README_ENRICHER_ENABLED`. This ensures the deterministic mode flag is a global override.

---

#### 8.32.5 Approach C — ReadmeFormat Agent Prompt Fragment

Constant in `agent/internal/executor/prompts/readme_format.go`. Appended to the agent system prompt via `InjectIfReadmeOutput(base, outputDocs)` when `"readme"` is in `outputDocs`. Mirrors the `InjectIfPlanOutput` pattern from §8.31.3.

```
=== README OUTPUT FORMAT ===

When generating content for the readme.md document, write every section in README format (not roadmap or plan format):

REQUIRED SECTIONS (in this order):
1. Golden rule — one sentence stating the single most important system invariant. Format: "**Golden rule:** {sentence}"
2. What it is NOT — 3-5 bullet points clarifying scope boundaries. Start each with "Not a ..." or "Not designed to ...".
3. When to use — numbered scenarios (3-6), each with: title, 1-2 sentence description, fenced code example showing a real command or config snippet.
4. Installation / Prerequisites — a table with | Tool | Version | Required | columns followed by setup steps.
5. Quick Start — a fenced bash block with 2-5 REAL runnable commands for THIS project's tech stack. NEVER write: git clone <repository-url>, make dev (unless Makefile is confirmed), placeholder commands.
6. Command Reference — a markdown table with | Command | Description | columns. Include build, test, run, and the 3-5 most important project-specific commands.
7. Architecture — first an ASCII diagram (box-drawing chars), then a mermaid flowchart. Both must show the actual components described in the architecture state.
8. Contributing — 2-3 sentences: what documentation to read first, how to run tests before submitting.

FORBIDDEN:
- "Phase N —" bullets anywhere in README output
- "## Known Risks" as a top-level section
- "## Roadmap" section with milestone bullets
- Placeholder text: <repository-url>, <project>, TODO, TBD
- Generic filler: "Add your description here", "Coming soon"

The README must be specific to THIS project. Use the actual project name, tech stack, and module structure from the conversation context.
=== END README FORMAT ===
```

---

#### 8.32.6 README Rubric Assertions

| Assertion | Type | Auto-repair trigger |
|---|---|---|
| `**Golden rule:**` present | `RequiredPattern` | "Add a **Golden rule:** line — one-sentence core invariant of this system" |
| ` ``` mermaid` fence present | `RequiredPattern` | "Add a mermaid flowchart to the When to use section showing the decision path for this product" |
| `## When to use` heading present | `RequiredSection` | "Add a '## When to use' section with 3-6 numbered scenarios each containing a code example" |
| `## Contributing` heading present | `RequiredSection` | "Add a '## Contributing' section explaining what to read before making changes" |
| `<repository-url>` absent | `ForbiddenString` | "Replace `<repository-url>` with the real project repository URL or `https://github.com/your-org/{project-slug}`" |
| `Phase 1 —` absent | `ForbiddenString` | "Remove all 'Phase N —' bullets from README — this is a README, not a roadmap" |
| `## Known Risks` absent | `ForbiddenString` | "Remove '## Known Risks' section from README — risks belong in architecture.md" |
| `For AI Agents` absent | `ForbiddenString` | "Remove the For AI Agents appendix — README is human-facing only" |
| Document `MinChars` ≥ 1500 | `MinLength` | "Expand each section with more specific content derived from the project's architecture and tech stack" |

Maximum 3 auto-repair attempts before falling back to deterministic-only output with a warning log.


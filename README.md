# a2a-brainstorm

A deterministic multi-agent design IDE — not a chatbot. Input an idea, run an ordered pipeline of agents, detect convergence, and emit engineering artifacts.

**This is NOT a chat application.** It is a structured workspace that coordinates multiple LLM-backed agents through a deterministic iteration loop to produce consistent, reviewable design artifacts.

Outputs:

- `architecture.md` — component design, data flows, technology choices
- `roadmap.md` — phased execution plan with milestones and risks

---

## Prerequisites

| Tool                    | Version                  |
| ----------------------- | ------------------------ |
| Go                      | 1.26+                    |
| Node.js                 | 20+                      |
| pnpm                    | 9+                       |
| Docker + docker-compose | latest                   |
| PostgreSQL              | 16 (provided via Docker) |

---

## Quick Start

```bash
# 1. Start the database
make up

# 2. Run migrations
make migrate

# 3. Start the backend
go run ./backend/cmd/server

# 4. Start an agent (in a separate terminal)
go run ./agent/cmd/server

# 5. Start the frontend (in a separate terminal)
cd frontend && pnpm dev
```

The backend API is available at `http://localhost:8080`.  
The frontend is available at `http://localhost:5173`.

---

## Environment Variables

### Backend (`backend/cmd/server`)

| Variable                    | Required | Example                                          | Description                                                 |
| --------------------------- | -------- | ------------------------------------------------ | ----------------------------------------------------------- |
| `DATABASE_URL`              | ✅       | `postgres://user:pass@localhost:5432/brainstorm` | PostgreSQL connection string                                |
| `PORT`                      | ❌       | `8080`                                           | HTTP listen port (default: 8080)                            |
| `GLOBAL_LLM_PROVIDER`       | ✅       | `copilot`                                        | Default LLM provider (`copilot` or `claude`)                |
| `GLOBAL_LLM_MODEL`          | ✅       | `gpt-4o`                                         | Default model name                                          |
| `GLOBAL_LLM_CREDENTIAL_REF` | ✅       | `COPILOT_API_KEY`                                | Env var name that holds the API key                         |
| `MAX_ITERATIONS`            | ❌       | `10`                                             | Max pipeline iterations per session (default: 10)           |
| `CONVERGENCE_THRESHOLD`     | ❌       | `0.02`                                           | Confidence delta below which pipeline halts (default: 0.02) |
| `AGENT_ENDPOINTS`           | ✅       | `http://localhost:9090`                          | Comma-separated list of agent base URLs                     |

### Agent binary (`agent/cmd/server`)

| Variable          | Required | Example      | Description                      |
| ----------------- | -------- | ------------ | -------------------------------- |
| `AGENT_PORT`      | ❌       | `9090`       | HTTP listen port (default: 9090) |
| `COPILOT_API_KEY` | \*       | `sk-...`     | API key for the Copilot provider |
| `CLAUDE_API_KEY`  | \*       | `sk-ant-...` | API key for the Claude provider  |

\* At least one LLM API key is required depending on which `GLOBAL_LLM_PROVIDER` is configured.

### Frontend (`frontend/`)

| Variable            | Required | Example                 | Description                                         |
| ------------------- | -------- | ----------------------- | --------------------------------------------------- |
| `VITE_API_BASE_URL` | ❌       | `http://localhost:8080` | Backend base URL (default: `http://localhost:8080`) |

> **Security rule:** API keys are never stored in source code, config files, or the database.
> `GLOBAL_LLM_CREDENTIAL_REF` holds the **env var name** only; the key is resolved at runtime.

---

## Agent Setup and Scaling

The system requires **at minimum 2 agents** per session. Agents are Go services running the `agent/cmd/server` binary.

### Running a single agent locally

```bash
AGENT_PORT=9090 COPILOT_API_KEY=<key> go run ./agent/cmd/server
```

### Running multiple agents

```bash
# Agent 1
AGENT_PORT=9090 COPILOT_API_KEY=<key> go run ./agent/cmd/server &

# Agent 2
AGENT_PORT=9091 CLAUDE_API_KEY=<key> go run ./agent/cmd/server &
```

### Registering agents with the backend

```bash
curl -X POST http://localhost:8080/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Builder Agent",
    "endpoint": "http://localhost:9090",
    "default_role": "build",
    "system_prompt": "You are an expert software architect.",
    "llm_config": {
      "provider": "copilot",
      "model": "gpt-4o",
      "credential_ref": "COPILOT_API_KEY"
    }
  }'
```

Agents serve their `AgentCard` at `/.well-known/agent.json`. The backend resolves this automatically when dispatching tasks.

### Absent credential behavior

If a required credential env var is absent at startup, the agent binary is marked **unavailable**. There is no silent fallback to another provider.

---

## Running a Brainstorm Session

```bash
# 1. Register at least 2 agents (see above)

# 2. Create a session
curl -X POST http://localhost:8080/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "idea": "A real-time collaborative document editor with conflict resolution",
    "agent_ids": ["<agent-1-id>", "<agent-2-id>"],
    "max_iterations": 5
  }'

# 3. Run an iteration
curl -X POST http://localhost:8080/sessions/<session-id>/iterate

# 4. Get current state
curl http://localhost:8080/sessions/<session-id>

# 5. Finalize and export artifacts
curl -X POST http://localhost:8080/sessions/<session-id>/finalize
```

---

## Architecture

````text
┌─────────────────────────────────────────────┐
│              Frontend (SvelteKit)            │
# a2a-brainstorm

Deterministic multi-agent design IDE (not a chatbot). The system runs an ordered agent pipeline and produces design artifacts such as architecture and roadmap outputs.

This repository is operated primarily with Docker Compose and Makefile targets.

## Prerequisites

- Docker + Docker Compose
- GNU Make
- Node.js 20+ and pnpm 9+ (frontend UI only)
- psql CLI (for `make migrate`)

## Quick Start (Docker + Makefile)

1. Copy environment file.

```bash
cp .env.example .env
````

2. Set required API keys in `.env` (`COPILOT_API_KEY` or `CLAUDE_API_KEY`).

3. Start infrastructure and services.

```bash
make up
```

4. Apply SQL migrations.

```bash
make migrate
```

5. Start frontend UI.

```bash
make frontend
```

Endpoints:

- Backend API: http://localhost:8080
- Agent A2A card: http://localhost:9090/.well-known/agent.json
- Frontend: http://localhost:5173

## Makefile Commands

Build and checks:

- `make build` - build backend
- `make build-agent` - build agent
- `make test` - run backend + agent tests
- `make lint` - run go vet (backend + agent) and frontend type check
- `make check` - full build + vet + frontend check/build

Runtime:

- `make up` - start docker compose services
- `make down` - stop docker compose services
- `make migrate` - apply SQL migrations from [migrations](migrations)
- `make frontend` - run frontend dev server
- `make frontend-build` - build frontend production bundle

## Environment Variables

Core backend variables:

- `DATABASE_URL` (required)
- `BACKEND_PORT` (default `8080`)
- `GLOBAL_LLM_PROVIDER` (default `copilot`)
- `GLOBAL_LLM_MODEL` (default `gpt-4o`)
- `GLOBAL_LLM_CREDENTIAL_REF` (default `COPILOT_API_KEY`)
- `AGENT_ENDPOINTS` (comma-separated)
- `MAX_ITERATIONS` (default `10`)
- `CONVERGENCE_THRESHOLD` (default `0.02`)

Core agent variables:

- `AGENT_PORT` (default `9090`)
- `AGENT_LLM_PROVIDER` (default `copilot`)
- `AGENT_LLM_MODEL` (default `gpt-4o`)
- `AGENT_LLM_CREDENTIAL_REF` (default `COPILOT_API_KEY`)

Secrets:

- `COPILOT_API_KEY`
- `CLAUDE_API_KEY`

Frontend variable:

- `VITE_API_BASE_URL` (default `http://localhost:8080`)

Security rules:

- Never commit real API keys.
- `*_CREDENTIAL_REF` stores an env var name, not a raw key value.

## Local Workflow

Daily start:

```bash
make up
make migrate
make frontend
```

Daily stop:

```bash
make down
```

Quality gate:

```bash
make test
make lint
make check
```

## Architecture Reference

See [docs/A2A-agent-Brainstorm.md](docs/A2A-agent-Brainstorm.md) for the full architecture and invariants.

See [docs/PLAN.md](docs/PLAN.md) for implementation task breakdown.

---

## Frontend Routes (v1.1)

| Route                   | Purpose                                            |
| ----------------------- | -------------------------------------------------- |
| `/`                     | Home — new session creation form                   |
| `/session/:id`          | Session workspace — sequential agent pipeline view |
| `/session/:id/finalize` | Export view — generation log + download artifacts  |
| `/settings`             | Unified settings — Agents, Skills, Roles tabs      |
| `/settings/agent/new`   | Create new agent                                   |
| `/settings/agent/:id`   | Edit existing agent                                |
| `/settings/skill/new`   | Create new skill                                   |
| `/settings/skill/:id`   | Edit existing skill                                |
| `/history`              | Session history — stats + past sessions table      |
| `/agents`               | Redirects to `/settings?tab=agents`                |
| `/skills`               | Redirects to `/settings?tab=skills`                |

> **Note:** `/agents` and `/skills` are redirect-only routes. All agent and skill management lives under `/settings`.

## Frontend Components (v1.1)

| Component                    | Status     | Replaces              | Purpose                                       |
| ---------------------------- | ---------- | --------------------- | --------------------------------------------- |
| `PipelineStage.svelte`       | Active     | `AgentPanel`          | Single agent stage card in pipeline view      |
| `ConfidenceBar.svelte`       | Active     | —                     | Animated confidence progress bar              |
| `CanonicalStatePanel.svelte` | Active     | `StateView`           | Structured canonical state display            |
| `RiskBoard.svelte`           | Active     | —                     | Risk and open questions board                 |
| `WarningModal.svelte`        | Active     | —                     | Guarded-action confirmation modal             |
| `AgentSelector.svelte`       | Active     | —                     | Agent multi-select for session creation       |
| `AgentPanel.svelte`          | Deprecated | → PipelineStage       | Keep for build compat; do not use in new code |
| `ControlPanel.svelte`        | Deprecated | → inlined             | Keep for build compat; do not use in new code |
| `StateView.svelte`           | Deprecated | → CanonicalStatePanel | Keep for build compat; do not use in new code |
| `Timeline.svelte`            | Deprecated | → inlined             | Keep for build compat; do not use in new code |

## Design System (v1.1)

All UI colors are defined as CSS custom properties in `frontend/src/app.css`. Never hard-code hex values in components or Tailwind classes. See [docs/A2A-agent-Brainstorm.md §21](docs/A2A-agent-Brainstorm.md) and [docs/PLAN.md §8.16](docs/PLAN.md) for the full design token reference.

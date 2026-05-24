# a2a-brainstorm

A deterministic multi-agent design IDE — not a chatbot. Input an idea, run an ordered pipeline of agents, detect convergence, and emit engineering artifacts.

**This is NOT a chat application.** It is a structured workspace that coordinates multiple LLM-backed agents through a deterministic iteration loop to produce consistent, reviewable design artifacts.

Outputs:

- `architecture.md` — component design, data flows, technology choices
- `roadmap.md` — phased execution plan with milestones and risks

---

## Prerequisites

| Tool                    | Version                      |
| ----------------------- | ---------------------------- |
| Go                      | 1.26+                        |
| Node.js                 | 20+                          |
| pnpm                    | 9+                           |
| Docker + Docker Compose | latest                       |
| GNU Make                | 3.81+                        |
| PostgreSQL              | 16 (provided via Docker)     |
| `psql` CLI              | any (used by `make migrate`) |

---

## Quick Start

```bash
# 1. Copy environment file and set API keys
cp .env.example .env

# 2. Start everything — postgres, backend, agent, frontend + run migrations
make start
```

That's it. One command starts all four services and applies migrations automatically.

Endpoints after startup:

- Frontend UI: http://localhost:5173
- Backend API: http://localhost:8080
- Agent A2A card: http://localhost:9090/.well-known/agent.json

---

## Makefile Commands

All Docker operations are wrapped in Makefile targets — never run `docker compose` directly.

### One-command startup

| Command      | Description                                                           |
| ------------ | --------------------------------------------------------------------- |
| `make start` | Start all services (postgres, backend, agent, frontend) + run migrate |

### Docker

| Command                     | Description                              |
| --------------------------- | ---------------------------------------- |
| `make docker-up`            | Start all services in the background     |
| `make docker-down`          | Stop and remove containers               |
| `make docker-restart`       | Stop then start all services             |
| `make docker-ps`            | List running containers and their status |
| `make docker-scale`         | Scale agent service (default `SCALE=2`)  |
| `make docker-logs`          | Tail logs from all services              |
| `make docker-logs-postgres` | Tail logs from the `postgres` container  |
| `make docker-logs-backend`  | Tail logs from the `backend` container   |
| `make docker-logs-agent`    | Tail logs from the `agent` container     |
| `make docker-logs-frontend` | Tail logs from the `frontend` container  |

Scale example:

```bash
make docker-scale SCALE=3
```

### OpenCode (optional — GitHub Copilot proxy)

OpenCode is a separate opt-in container. It is only needed when using GitHub Copilot
as the LLM provider (Copilot uses OAuth, not a plain API key). See
[docs/STARTUP_GUIDE.md §9](docs/STARTUP_GUIDE.md) for the full beginner walkthrough.

| Command                | Description                                                               |
| ---------------------- | ------------------------------------------------------------------------- |
| `make opencode-up`     | Start the OpenCode container (first start takes ~30 s to install)         |
| `make opencode-auth`   | One-time browser OAuth flow to authenticate with GitHub Copilot           |
| `make opencode-status` | Print OpenCode health JSON — confirms the server is running and reachable |
| `make opencode-logs`   | Tail live logs from the OpenCode container                                |
| `make opencode-down`   | Stop the OpenCode container (auth token volume is preserved)              |

> `make docker-down` stops **everything** including OpenCode.
> `make opencode-down` stops **only** OpenCode while keeping the main stack running.

### Database

| Command        | Description                                 |
| -------------- | ------------------------------------------- |
| `make migrate` | Apply all SQL migrations from `migrations/` |

### Build

| Command            | Description          |
| ------------------ | -------------------- |
| `make build`       | Build backend binary |
| `make build-agent` | Build agent binary   |

### Tests & Quality

| Command      | Description                                        |
| ------------ | -------------------------------------------------- |
| `make test`  | Run backend + agent Go tests                       |
| `make lint`  | `go vet` (backend + agent) + frontend `pnpm check` |
| `make check` | Full build + vet + frontend check/build            |

### Frontend

| Command               | Description                      |
| --------------------- | -------------------------------- |
| `make frontend`       | Start frontend dev server        |
| `make frontend-build` | Build frontend production bundle |

---

## Daily Workflow

Start everything:

```bash
make start
```

Stop everything:

```bash
make docker-down
```

Quality gate:

```bash
make test
make lint
make check
```

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

| Variable                   | Required | Example           | Description                                      |
| -------------------------- | -------- | ----------------- | ------------------------------------------------ |
| `AGENT_PORT`               | ❌       | `9090`            | HTTP listen port (default: 9090)                 |
| `AGENT_LLM_PROVIDER`       | ❌       | `copilot`         | LLM provider for this agent (default: `copilot`) |
| `AGENT_LLM_MODEL`          | ❌       | `gpt-4o`          | Model name for this agent                        |
| `AGENT_LLM_CREDENTIAL_REF` | ❌       | `COPILOT_API_KEY` | Env var name holding the agent's API key         |
| `COPILOT_API_KEY`          | \*       | `sk-...`          | API key for the Copilot provider                 |
| `CLAUDE_API_KEY`           | \*       | `sk-ant-...`      | API key for the Claude provider                  |

\* At least one LLM API key is required.

### Frontend (`frontend/`)

| Variable            | Required | Example                 | Description                                         |
| ------------------- | -------- | ----------------------- | --------------------------------------------------- |
| `VITE_API_BASE_URL` | ❌       | `http://localhost:8080` | Backend base URL (default: `http://localhost:8080`) |

> **Security rule:** API keys are never stored in source code, config files, or the database.
> `*_CREDENTIAL_REF` variables hold the **env var name only**; keys are resolved at runtime via `os.Getenv`.

---

## Agent Setup and Scaling

The system requires **at minimum 2 agents** per session. All services including agents run via Docker Compose.

### Scaling agents via Docker

```bash
# Run 3 agent containers
make docker-scale SCALE=3
```

Then update `AGENT_ENDPOINTS` in `.env` with all agent URLs:

```env
AGENT_ENDPOINTS=http://agent:9090,http://agent:9090,http://agent:9090
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

If a required credential env var is absent at startup, the agent binary is marked **unavailable**. There is no silent fallback.

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

```text
┌─────────────────────────────────────────────┐
│              Frontend (SvelteKit)            │
│  / · /session/:id · /settings · /history    │
└──────────────────┬──────────────────────────┘
                   │ HTTP (fetch)
┌──────────────────▼──────────────────────────┐
│          Backend (Go modular monolith)       │
│  session · iteration · agent · state        │
│  convergence · markdown · platform/         │
└────────┬──────────────────────┬─────────────┘
         │ pgx/v5               │ A2A (a2a-go/v2)
┌────────▼────────┐    ┌────────▼────────────┐
│  PostgreSQL 16  │    │   Agent binary (Go) │
│  (Docker)       │    │   BrainstormExecutor│
└─────────────────┘    └─────────────────────┘
```

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

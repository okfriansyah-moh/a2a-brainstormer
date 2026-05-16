# STARTUP_GUIDE.md

Beginner-friendly local startup guide for a2a-brainstorm.

This guide uses Docker and Makefile commands as the default workflow.

## 1) What You Are Running

The project has three main runtime parts:

- `postgres` service (database)
- `backend` service (REST API + orchestration)
- `agent` service (A2A executor)

The frontend runs locally with pnpm for faster UI development.

## 2) Prerequisites

Install these tools first:

- Docker Desktop (or Docker Engine + Docker Compose)
- GNU Make
- Node.js 20+
- pnpm 9+
- `psql` CLI (used by `make migrate`)

Quick checks:

```bash
docker --version
docker compose version
make --version
node --version
pnpm --version
psql --version
```

## 3) First-Time Setup

From project root:

```bash
cp .env.example .env
```

Open `.env` and set at least one API key:

- `COPILOT_API_KEY=...` or
- `CLAUDE_API_KEY=...`

Keep these values in sync:

- `GLOBAL_LLM_CREDENTIAL_REF` should point to a key env var name
- `AGENT_LLM_CREDENTIAL_REF` should point to a key env var name

Example:

```env
GLOBAL_LLM_CREDENTIAL_REF=COPILOT_API_KEY
AGENT_LLM_CREDENTIAL_REF=COPILOT_API_KEY
```

## 4) Start Everything (Docker + Make)

1. Start docker services:

```bash
make up
```

2. Apply SQL migrations:

```bash
make migrate
```

3. Start frontend in a second terminal:

```bash
make frontend
```

## 5) Verify It Works

Use these URLs:

- Backend health: http://localhost:8080/health
- Agent card: http://localhost:9090/.well-known/agent.json
- Frontend UI: http://localhost:5173

Optional terminal checks:

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:9090/.well-known/agent.json
```

## 6) Daily Commands

Start day:

```bash
make up
make migrate
make frontend
```

Quality checks:

```bash
make test
make lint
make check
```

Stop day:

```bash
make down
```

## 7) Multi-Agent Local Scaling

If you want more than one agent container:

```bash
docker compose up -d --scale agent=2
```

Then set backend agent endpoints to both instances in `.env`, for example:

```env
AGENT_ENDPOINTS=http://agent:9090,http://agent:9090
```

Note: compose service-load-balancing behavior depends on your Docker setup. For deterministic local tests, single-agent mode is simpler.

## 8) Common Issues and Fixes

1. `make migrate` fails with connection error:

- Ensure `make up` is already running.
- Ensure `DATABASE_URL` points to localhost port 5432 for host-side migration command.

2. Backend starts but session creation fails due to unavailable agent:

- Verify API key is set in `.env`.
- Verify `GLOBAL_LLM_CREDENTIAL_REF` and `AGENT_LLM_CREDENTIAL_REF` match the key variable name.

3. Frontend cannot reach API:

- Set `VITE_API_BASE_URL=http://localhost:8080` in frontend env.
- Confirm backend health endpoint responds.

4. Docker service unhealthy:

- Check container logs with:

```bash
docker compose logs postgres
docker compose logs backend
docker compose logs agent
```

## 9) Where to Read Next

- Architecture blueprint: [A2A-agent-Brainstorm.md](A2A-agent-Brainstorm.md)
- Implementation plan: [PLAN.md](PLAN.md)
- Main project overview: [../README.md](../README.md)

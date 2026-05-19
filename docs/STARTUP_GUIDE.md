# STARTUP_GUIDE.md

Beginner-friendly local startup guide for a2a-brainstorm.

All Docker operations use Makefile targets. Do not run `docker compose` commands directly.

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

## 4) Start Everything (One Command)

From the project root:

```bash
make start
```

This single command:

1. Starts all Docker services — `postgres`, `backend`, `agent`, `frontend`
2. Waits for the database to be healthy
3. Applies all SQL migrations automatically

Endpoints once everything is up:

- Frontend: http://localhost:5173
- Backend API: http://localhost:8080
- Agent card: http://localhost:9090/.well-known/agent-card.json

## 5) Verify It Works

Use these URLs:

- Backend health: http://localhost:8080/health
- Agent card: http://localhost:9090/.well-known/agent-card.json
- Frontend UI: http://localhost:5173

Optional terminal checks:

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:9090/.well-known/agent-card.json
```

## 6) Daily Commands

Start day (everything in one command):

```bash
make start
```

Quality checks:

```bash
make test
make lint
make check
```

Stop day:

```bash
make docker-down
```

## 7) Multi-Agent Local Scaling

If you want more than one agent container, use the `docker-scale` target. The default is 2 replicas; pass `SCALE=N` to override:

```bash
# Start with 2 agent replicas (default)
make docker-scale

# Start with 3 agent replicas
make docker-scale SCALE=3
```

Then set backend agent endpoints in `.env`:

```env
AGENT_ENDPOINTS=http://agent:9090,http://agent:9090
```

Note: compose service-load-balancing behavior depends on your Docker setup. For deterministic local tests, single-agent mode is simpler.

## 8) Common Issues and Fixes

1. `make migrate` fails with connection error:

- Ensure `make docker-up` is already running.
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
make docker-logs-postgres
make docker-logs-backend
make docker-logs-agent
make docker-logs-frontend
```

- Or tail all services at once:

```bash
make docker-logs
```

5. Port already in use (another project is running):

- Error usually looks like `Bind for 0.0.0.0:5432 failed: port is already allocated`.
- Change host port mappings in `.env` (do not change container internal ports):

```env
POSTGRES_HOST_PORT=15432
BACKEND_HOST_PORT=18080
AGENT_HOST_PORT=19090
DATABASE_URL=postgres://postgres:postgres@localhost:15432/a2a_brainstorm?sslmode=disable
AGENT_ENDPOINTS=http://localhost:19090
```

- Restart services after changing ports:

```bash
make docker-restart
make migrate
```

## 9) Where to Read Next

- Architecture blueprint: [A2A-agent-Brainstorm.md](A2A-agent-Brainstorm.md)
- Implementation plan: [PLAN.md](PLAN.md)
- Main project overview: [../README.md](../README.md)

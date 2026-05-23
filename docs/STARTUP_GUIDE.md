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

## 9) Running with OpenCode Server (optional)

The OpenCode server is an optional LLM proxy that lets the agent binary route all
LLM calls through a single authenticated OpenCode instance rather than holding
individual API keys in every container. It is most useful for GitHub Copilot,
which uses OAuth and does not produce traditional API keys.

**When to use:** choose OpenCode when you want to use GitHub Copilot as the LLM
backend inside Docker Compose without per-container credential management.

### One-time Setup

1. **Start the OpenCode container** (uses the `opencode` Docker Compose profile):

   ```bash
   make opencode-up
   ```

2. **Authenticate with GitHub Copilot** (follow the browser prompt):

   ```bash
   make opencode-auth
   ```

   This stores the OAuth token in the `opencode-auth` Docker volume.
   You only need to do this once; the token persists across restarts.

3. **Configure `.env`** — switch the agent to the OpenCode provider:

   ```env
   AGENT_LLM_PROVIDER=opencode
   AGENT_OPENCODE_BASE_URL=http://opencode:4096
   AGENT_OPENCODE_MODEL=github/gpt-4o
   OPENCODE_SERVER_USERNAME=opencode
   OPENCODE_SERVER_PASSWORD=change-me-to-a-strong-password
   ```

### OpenCode Model Enum (avoid wrong value format)

`AGENT_OPENCODE_MODEL` must always use this format:

```text
<providerID>/<modelID>
```

Examples:

- `github/gpt-4o`
- `anthropic/claude-sonnet-4-6`
- `github/gpt-5.4-mini`

### Full Model Enum (all currently available models)

The model list depends on which providers you have connected in OpenCode.
To print the full enum from your running OpenCode server:

```bash
curl -s \
   -u "$OPENCODE_SERVER_USERNAME:$OPENCODE_SERVER_PASSWORD" \
   http://localhost:4096/config/providers \
| jq -r '
   def model_id:
      if type == "string" then .
      elif type == "object" then (.id // .name // empty)
      else empty end;

   .providers[] as $p
   | ($p.id // $p.providerID // empty) as $pid
   | ($p.models // []) as $models
   | if ($models | type) == "array" then
         $models[] | "\($pid)/\(model_id)"
      elif ($models | type) == "object" then
         $models | to_entries[] | "\($pid)/\(.key)"
      else
         empty
      end
' \
| sed '/\/$/d' \
| sort -u
```

Optional: save the enum list to a file:

```bash
curl -s \
   -u "$OPENCODE_SERVER_USERNAME:$OPENCODE_SERVER_PASSWORD" \
   http://localhost:4096/config/providers \
| jq -r '
   def model_id:
      if type == "string" then .
      elif type == "object" then (.id // .name // empty)
      else empty end;

   .providers[] as $p
   | ($p.id // $p.providerID // empty) as $pid
   | ($p.models // []) as $models
   | if ($models | type) == "array" then
         $models[] | "\($pid)/\(model_id)"
      elif ($models | type) == "object" then
         $models | to_entries[] | "\($pid)/\(.key)"
      else
         empty
      end
' \
| sed '/\/$/d' \
| sort -u > .opencode-model-enum.txt
```

Then set one exact value from that list in `.env`:

```env
# Option 1: Copilot GPT-5.4 mini
AGENT_OPENCODE_MODEL=github/gpt-5.4-mini

# Option 2: Claude Sonnet 4.6
AGENT_OPENCODE_MODEL=anthropic/claude-sonnet-4-6
```

If the format is invalid (missing `/`), the agent falls back to `github/gpt-4o`.

4. **(Re)start the agent service** so it picks up the new env vars:

   ```bash
   docker compose up -d agent
   ```

5. **Verify the OpenCode server is healthy:**

   ```bash
   make opencode-status
   ```

   Expected output contains `"status": "ok"`.

### Credential Flow

```
.env
  OPENCODE_SERVER_USERNAME / OPENCODE_SERVER_PASSWORD
        │
        ▼
  docker-compose.yml injects both into the opencode container and
  exposes them as env vars for the agent container.
        │
        ▼
  agent/internal/config/config.go reads AGENT_OPENCODE_USERNAME_REF
  and AGENT_OPENCODE_PASSWORD_REF (the *names* of the credential vars).
        │
        ▼
  OpenCodeProvider.resolveCredentials() calls os.Getenv on those names
  at request time — credentials are never stored on any struct or logged.
```

The volume `opencode-auth` persists the GitHub Copilot OAuth token.
Deleting the volume forces a full re-authentication:

```bash
docker volume rm a2a-brainstorm_opencode-auth
```

### Troubleshooting

| Symptom                                                       | Fix                                                                       |
| ------------------------------------------------------------- | ------------------------------------------------------------------------- |
| Agent returns `401 Unauthorized` from OpenCode                | Wrong `OPENCODE_SERVER_PASSWORD` in `.env`. Update and restart the agent. |
| `make opencode-status` fails with connection refused          | OpenCode container is not running. Run `make opencode-up`.                |
| Agent `Generate` returns `403 Forbidden` from OpenCode        | OAuth token expired. Run `make opencode-auth` and restart agent.          |
| `make opencode-auth` exits immediately with no browser prompt | OpenCode container is still starting. Wait 30 s and retry.                |

## 10) Where to Read Next

- Architecture blueprint: [A2A-agent-Brainstorm.md](A2A-agent-Brainstorm.md)
- Implementation plan: [PLAN.md](PLAN.md)
- Main project overview: [../README.md](../README.md)

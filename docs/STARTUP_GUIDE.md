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

OpenCode is an optional LLM proxy container. It lets every agent container share a
single GitHub Copilot login rather than each one needing its own API key. This is
the recommended approach when using **GitHub Copilot** as your LLM provider, because
Copilot uses browser-based OAuth — it does not issue traditional copy-paste API keys.

**You do not need OpenCode** if you are using Claude or any other provider that gives
you a plain API key (just set `CLAUDE_API_KEY` in `.env` and skip this section).

### Do I need OpenCode?

| Your LLM provider         | Use OpenCode? |
| ------------------------- | ------------- |
| GitHub Copilot (OAuth)    | ✅ Yes        |
| Claude (API key)          | ❌ No         |
| OpenAI or other (API key) | ❌ No         |

### How it fits into the stack

OpenCode is **not** started by `make start`. It uses a Docker Compose _profile_ so
it stays out of the way until you opt in:

```
make start         → starts postgres, backend, agent, frontend  (OpenCode NOT included)
make opencode-up   → starts OpenCode separately, alongside the above
```

Both can run at the same time. `make docker-down` stops everything including OpenCode.

---

### First-time setup (do this once)

**Step 1 — Start the main stack** (skip if already running):

```bash
make start
```

**Step 2 — Start the OpenCode container:**

```bash
make opencode-up
```

This starts a single `opencode` container listening on port 4096. The first start
takes 20–30 seconds while it installs the OpenCode package.

**Step 3 — Authenticate with GitHub Copilot (once only):**

```bash
make opencode-auth
```

This opens an interactive prompt **inside the container**. Use the arrow keys to select
**GitHub Copilot → GitHub.com**, then:

1. Visit `https://github.com/login/device` in your browser.
2. Enter the 8-character code shown in the terminal.
3. Click **Authorize** on GitHub.
4. The terminal prints `Login successful` and exits.

The OAuth token is saved to a Docker volume called `opencode-auth`. It persists across
container restarts and upgrades — **you only need to do this once**. The only reason
to repeat it is if you delete the volume or the token expires (GitHub Copilot tokens
last ~1 year).

**Step 4 — Check that OpenCode is healthy:**

```bash
make opencode-status
```

You should see output containing `"healthy": true`. If you get "connection refused",
OpenCode is still starting — wait 30 seconds and try again.

**Step 5 — Configure `.env` to use OpenCode:**

Open `.env` and add or update these values:

```env
AGENT_LLM_PROVIDER=opencode
AGENT_OPENCODE_BASE_URL=http://opencode:4096
AGENT_OPENCODE_MODEL=github/gpt-4.1
OPENCODE_SERVER_USERNAME=opencode
OPENCODE_SERVER_PASSWORD=opencode-local
```

**About `OPENCODE_SERVER_PASSWORD`:**

The agent requires a non-empty password to start (security invariant: absent credential → agent unavailable). The OpenCode server uses the same value to authenticate incoming requests.

| Situation                              | What to do                                                                                                 |
| -------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| Local dev (only you use this machine)  | Use a simple password like `opencode-local` — the server is only reachable inside Docker's private network |
| Shared machine or exposed to a LAN/VPN | Set a strong password so other users cannot proxy LLM calls through your Copilot token                     |
| Production / cloud deployment          | Always set a strong, random password                                                                       |

**Step 6 — Restart both OpenCode and the agent to pick up the new env vars:**

```bash
docker compose --profile opencode up -d opencode agent
```

The agent now routes all LLM calls through OpenCode.

---

### Choosing a model

`AGENT_OPENCODE_MODEL` must always use this exact format:

```text
<providerID>/<modelID>
```

Quick examples:

```env
AGENT_OPENCODE_MODEL=github/gpt-4o
AGENT_OPENCODE_MODEL=github/gpt-5.4-mini
AGENT_OPENCODE_MODEL=anthropic/claude-sonnet-4-6
```

If the format is invalid (missing `/`), the agent falls back to `github/gpt-4o`.

To see every model your running OpenCode server knows about:

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

To save that list to a file for reference:

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

---

### Daily workflow with OpenCode

On days you want to use OpenCode, run these in order:

```bash
make start            # start the main stack (postgres, backend, agent, frontend)
make opencode-up      # start the OpenCode container
make opencode-status  # confirm it is healthy before using the agent
```

To watch live OpenCode logs (useful for debugging LLM calls):

```bash
make opencode-logs
```

To stop **only OpenCode** while keeping the main stack running:

```bash
make opencode-down
```

To stop **everything** (main stack + OpenCode):

```bash
make docker-down
```

---

### Makefile reference for OpenCode

| Command                | What it does                                                                  |
| ---------------------- | ----------------------------------------------------------------------------- |
| `make opencode-up`     | Start the OpenCode container (first start installs the package ~30 s)         |
| `make opencode-auth`   | Interactive device-code OAuth to authenticate with GitHub Copilot (once only) |
| `make opencode-status` | Print the OpenCode health JSON — confirms it is running and reachable         |
| `make opencode-logs`   | Tail live logs from the OpenCode container                                    |
| `make opencode-down`   | Stop the OpenCode container (auth token volume is preserved)                  |
| `make docker-down`     | Stop all containers including OpenCode                                        |

---

### How credentials flow

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
  OpenCodeProvider.resolveCredentials() uses the injected resolver
  (config.GetLLMAPIKey) to look up those names at request time.
  Direct os.Getenv access is confined to config.go only.
  Credentials are never stored on any struct or logged.
```

The `opencode-auth` Docker volume persists the GitHub Copilot OAuth token.
To force a full re-authentication, delete the volume:

```bash
docker volume rm a2a-brainstorm_opencode-auth
```

Then run `make opencode-auth` again.

---

### Troubleshooting

| Symptom                                                     | Fix                                                                                           |
| ----------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| `make opencode-status` — "connection refused"               | OpenCode is not running. Run `make opencode-up` and wait 30 s.                                |
| `make opencode-auth` exits immediately or shows an error    | OpenCode is still starting. Wait 30 s and retry.                                              |
| Agent returns `401 Unauthorized`                            | Wrong `OPENCODE_SERVER_PASSWORD` in `.env`. Update it and run `docker compose up -d agent`.   |
| Agent returns `403 Forbidden`                               | Copilot OAuth token expired. Run `make opencode-auth` then `docker compose up -d agent`.      |
| Agent returns `500` and logs `credential env var … not set` | `OPENCODE_SERVER_PASSWORD` is set in `.env` but not loaded. Run `docker compose up -d agent`. |
| `make opencode-logs` shows repeated errors                  | Check `.env` values for `AGENT_OPENCODE_BASE_URL` and `AGENT_OPENCODE_MODEL`.                 |

## 10) Where to Read Next

- Architecture blueprint: [A2A-agent-Brainstorm.md](A2A-agent-Brainstorm.md)
- Implementation plan: [PLAN.md](PLAN.md)
- Main project overview: [../README.md](../README.md)

## 11) AI-Driven Document Generation (Task 33)

Session finalize can optionally rewrite the deterministic markdown scaffolds
through an LLM with a curated skill bundle injected as the system prompt.
Configure via the following env vars (all optional):

| Variable             | Default                                                                                                                                                                                     | Purpose                                                                                                     |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| `FINALIZE_MODE`      | `hybrid`                                                                                                                                                                                    | `deterministic` skips the AI pass; `hybrid` falls back to the scaffold on AI failure; `ai` aborts finalize. |
| `SKILL_BUNDLE_PATHS` | `.github/skills/modularity/SKILL.md,.github/skills/vertical-slice/SKILL.md,.github/skills/api-design/SKILL.md,.github/skills/roadmap-spec/SKILL.md,.github/skills/plan-management/SKILL.md` | Comma-separated list of skill files injected (in order) as the system prompt prefix for AI generation.      |
| `AIGEN_MAX_REPAIRS`  | `2`                                                                                                                                                                                         | Max rubric-driven repair attempts per document (clamped to 0–5).                                            |
| `AIGEN_TEMPERATURE`  | `0.2`                                                                                                                                                                                       | LLM temperature for document rewriting (clamped to 0.0–1.0).                                                |

When `FINALIZE_MODE` ≠ `deterministic` but no global LLM credential is configured
or the skill bundle fails to load, the system logs `aigen_fallback` and writes
deterministic scaffolds. Deterministic mode is the only byte-stable mode across
runs.

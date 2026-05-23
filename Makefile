SCALE ?= 2

.PHONY: start build build-agent \
        docker-up docker-down docker-restart docker-ps \
        docker-logs docker-logs-postgres docker-logs-backend docker-logs-agent docker-logs-frontend \
        docker-scale \
        opencode-up opencode-down opencode-auth opencode-status opencode-logs \
        migrate test frontend frontend-build lint check

ifneq (,$(wildcard .env))
include .env
export
endif

# ── One-command startup ──────────────────────────────────────────────────────
# Starts all services (postgres, backend, agent, frontend) and applies migrations.
start:
	docker compose up -d
	@echo "Waiting for postgres to be healthy..."
	@until docker compose exec postgres pg_isready -U postgres > /dev/null 2>&1; do sleep 1; done
	@$(MAKE) --no-print-directory migrate
	@echo ""
	@echo "All services started:"
	@echo "  Frontend : http://localhost:${FRONTEND_HOST_PORT:-5173}"
	@echo "  Backend  : http://localhost:${BACKEND_HOST_PORT:-8080}"
	@echo "  Agent    : http://localhost:${AGENT_HOST_PORT:-9090}"

# ── Go ──────────────────────────────────────────────────────────────────────
build:
	cd backend && go build ./...

build-agent:
	cd agent && go build ./...

# ── Docker ──────────────────────────────────────────────────────────────────
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-restart: docker-down docker-up

docker-ps:
	docker compose ps

docker-scale:
	docker compose up -d --scale agent=$(SCALE)

docker-logs:
	docker compose logs -f

docker-logs-postgres:
	docker compose logs -f postgres

docker-logs-backend:
	docker compose logs -f backend

docker-logs-agent:
	docker compose logs -f agent

docker-logs-frontend:
	docker compose logs -f frontend

# ── OpenCode Server (optional — Docker profile) ───────────────────────────────
## opencode-up: Start the OpenCode server container (requires Docker profile)
opencode-up:
	docker compose --profile opencode up -d opencode

## opencode-auth: One-time GitHub Copilot OAuth inside the OpenCode container.
## Run once after the first `make opencode-up`, then restart the agent.
opencode-auth:
	docker compose exec opencode opencode /provider/github/oauth/authorize

## opencode-down: Stop the OpenCode container (keeps the auth token volume intact)
opencode-down:
	docker compose --profile opencode stop opencode

## opencode-logs: Tail live logs from the OpenCode container
opencode-logs:
	docker compose logs -f opencode

## opencode-status: Check whether the OpenCode server is healthy
opencode-status:
	curl -sf http://localhost:4096/global/health | jq .

# ── Database ─────────────────────────────────────────────────────────────────
migrate:
	@for f in migrations/*.sql; do \
		echo "Applying $$f..."; \
		psql "$$DATABASE_URL" -f "$$f"; \
	done

# ── Tests ───────────────────────────────────────────────────────────────────
test:
	cd backend && go test ./...
	cd agent && go test ./...

# ── Frontend ─────────────────────────────────────────────────────────────────
frontend:
	cd frontend && pnpm dev

frontend-build:
	cd frontend && pnpm build

# ── Lint / Vet ───────────────────────────────────────────────────────────────
lint:
	cd backend && go vet ./...
	cd agent && go vet ./...
	cd frontend && pnpm check

# ── Full check (backend + agent + frontend) ───────────────────────────────────
check:
	cd backend && go build ./... && go vet ./...
	cd agent && go build ./... && go vet ./...
	cd frontend && pnpm check && pnpm build

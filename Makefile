.PHONY: build build-agent up down migrate test frontend lint check

# ── Go ──────────────────────────────────────────────────────────────────────
build:
	cd backend && go build ./...

build-agent:
	cd agent && go build ./...

# ── Docker ──────────────────────────────────────────────────────────────────
up:
	docker compose up -d

down:
	docker compose down

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

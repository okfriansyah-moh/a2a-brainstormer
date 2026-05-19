# ─── Build stage ─────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependencies before copying source.
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

# Copy source and migrations.
COPY backend    ./backend
COPY migrations ./migrations

# Fully-static binary: no cgo, trimpath removes build paths, -s -w strips debug info.
RUN cd backend && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/backend ./cmd/server

# ─── Health-check tooling ─────────────────────────────────────────────────────
# busybox:musl is statically linked — safe to copy into distroless/static.
# The single /bin/busybox binary provides both `sh` and `wget` as multi-call applets.
FROM busybox:musl AS busybox

# ─── Runtime: 0 known CVEs ───────────────────────────────────────────────────
# gcr.io/distroless/static-debian12 ships no OS packages, no shell, no package manager.
FROM gcr.io/distroless/static-debian12:nonroot

# Provide sh + wget (via busybox multi-call binary) for docker-compose health checks.
COPY --from=busybox /bin/busybox /usr/local/bin/busybox

COPY --from=builder /out/backend    /app/backend
COPY --from=builder /src/migrations /app/migrations

USER nonroot:nonroot

# Port the backend listens on — matches BACKEND_PORT in .env (default 18080).
ARG BACKEND_PORT=18080
EXPOSE ${BACKEND_PORT}

ENTRYPOINT ["/app/backend"]

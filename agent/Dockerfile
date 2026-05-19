# ─── Build stage ─────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependencies before copying source.
COPY agent/go.mod agent/go.sum ./agent/
RUN cd agent && go mod download

# Copy source.
COPY agent ./agent

# Fully-static binary: no cgo, trimpath removes build paths, -s -w strips debug info.
RUN cd agent && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/agent ./cmd/server

# ─── Health-check tooling ─────────────────────────────────────────────────────
# busybox:musl is statically linked — safe to copy into distroless/static.
FROM busybox:musl AS busybox

# ─── Runtime: 0 known CVEs ───────────────────────────────────────────────────
# gcr.io/distroless/static-debian12 ships no OS packages, no shell, no package manager.
FROM gcr.io/distroless/static-debian12:nonroot

# Provide sh + wget (via busybox multi-call binary) for docker-compose health checks.
COPY --from=busybox /bin/busybox /usr/local/bin/busybox

COPY --from=builder /out/agent /app/agent

USER nonroot:nonroot

# Port the agent listens on — matches AGENT_PORT in .env (default 19090).
ARG AGENT_PORT=19090
EXPOSE ${AGENT_PORT}

ENTRYPOINT ["/app/agent"]

FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependencies.
COPY agent/go.mod agent/go.sum ./agent/
RUN cd agent && go mod download

# Copy source.
COPY agent ./agent

# Build agent binary.
RUN cd agent && CGO_ENABLED=0 GOOS=linux go build -o /out/agent ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget
RUN adduser -D app

WORKDIR /app

COPY --from=builder /out/agent /app/agent

USER app

EXPOSE 9090

CMD ["/app/agent"]

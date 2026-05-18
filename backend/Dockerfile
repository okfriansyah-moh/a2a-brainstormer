FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependencies.
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

# Copy source.
COPY backend ./backend
COPY migrations ./migrations

# Build backend binary.
RUN cd backend && CGO_ENABLED=0 GOOS=linux go build -o /out/backend ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget
RUN adduser -D app

WORKDIR /app

COPY --from=builder /out/backend /app/backend
COPY --from=builder /src/migrations /app/migrations

USER app

EXPOSE 8080

CMD ["/app/backend"]

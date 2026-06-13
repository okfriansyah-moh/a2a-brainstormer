// Package http provides the HTTP router, middleware, and server helpers for
// the backend API.
//
// NewRouter wires all module handlers onto a single net/http ServeMux and
// wraps it with request-logging and CORS middleware so the SvelteKit dev
// server (default origin http://localhost:5173) can reach the backend without
// CORS preflight failures.
package http

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// RouteRegistrar is implemented by module handlers that can register routes on
// a shared ServeMux.
type RouteRegistrar interface {
	RegisterRoutes(mux *http.ServeMux)
}

// Deps carries all handler dependencies that NewRouter needs to wire the
// application together. Each field is the concrete handler produced by the
// respective module.
type Deps struct {
	AgentHandler     RouteRegistrar
	SessionHandler   RouteRegistrar
	IterationHandler RouteRegistrar
	GlobalLLMHandler http.Handler // GET /api/config/global-llm — may be nil
	Logger           *slog.Logger
}

// NewRouter creates and returns the fully-wired http.Handler for the backend.
//
// Route registration order:
//  1. Agent + Skill endpoints  (/agents/…, /skills/…)
//  2. Session endpoints        (/sessions/…)
//  3. Iteration endpoint       (POST /sessions/{id}/iterate)
//  4. Health check             (GET /health)
//
// The returned handler is wrapped with CORS headers and a structured request
// logger.
func NewRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	// Register module routes.
	deps.AgentHandler.RegisterRoutes(mux)
	deps.SessionHandler.RegisterRoutes(mux)
	deps.IterationHandler.RegisterRoutes(mux)

	// Global LLM config — read-only env var reflection for the settings UI.
	if deps.GlobalLLMHandler != nil {
		mux.Handle("GET /api/config/global-llm", deps.GlobalLLMHandler)
	}

	// Health check — no DB ping; just confirms the process is alive.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Wrap with middleware (innermost first, outermost last).
	var h http.Handler = mux
	h = corsMiddleware(h)
	h = requestLoggerMiddleware(h, deps.Logger)

	return h
}

// ── Middleware ────────────────────────────────────────────────────────────────

// corsMiddleware adds permissive CORS headers for the SvelteKit development
// origin (http://localhost:5173). In production, restrict the allowed origin
// to the actual frontend domain via the FRONTEND_ORIGIN env var.
//
// The middleware handles preflight OPTIONS requests by returning 204 No Content
// with the CORS headers set. All other requests receive the headers too.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// requestLoggerMiddleware logs each request with method, path, status, and
// latency using structured slog output. It uses WARN for 4xx responses, ERROR
// for 5xx, and INFO for everything else — except expected GET 404s on the
// session SSE endpoint (stale browser tabs reconnecting to deleted sessions).
func requestLoggerMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	if logger == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		latency := time.Since(start)

		level := requestLogLevel(r.Method, r.URL.Path, rw.statusCode)

		logger.Log(r.Context(), level, "http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.statusCode),
			slog.Duration("latency", latency),
		)
	})
}

// requestLogLevel picks the slog level for an HTTP access log line.
func requestLogLevel(method, path string, status int) slog.Level {
	if status >= 500 {
		return slog.LevelError
	}
	if status >= 400 {
		if status == http.StatusNotFound && isSessionEventsSSEPath(method, path) {
			return slog.LevelInfo
		}
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

// isSessionEventsSSEPath reports GET /sessions/{id}/events — a 404 here usually
// means a zombie EventSource from a deleted or expired session tab.
func isSessionEventsSSEPath(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	return len(parts) == 3 && parts[0] == "sessions" && parts[2] == "events"
}

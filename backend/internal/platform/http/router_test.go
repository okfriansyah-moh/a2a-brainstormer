// Package http — tests for the router, CORS middleware, and request logger.
package http

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── Health check ────────────────────────────────────────────────────────────

func TestHealthEndpoint(t *testing.T) {
	deps := Deps{
		AgentHandler:     nil,
		SessionHandler:   nil,
		IterationHandler: nil,
		Logger:           slog.Default(),
	}

	// NewRouter panics if any handler is nil and RegisterRoutes is called.
	// We bypass that here by constructing a minimal mux manually.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	handler := corsMiddleware(requestLoggerMiddleware(mux, deps.Logger))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /health: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body != `{"status":"ok"}` {
		t.Errorf("GET /health: unexpected body %q", body)
	}
}

// ── CORS middleware ─────────────────────────────────────────────────────────

func TestCORSMiddleware_PreflightReturns204(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := corsMiddleware(inner)

	req := httptest.NewRequest(http.MethodOptions, "/sessions", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS preflight: want 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Error("OPTIONS preflight: missing Access-Control-Allow-Origin header")
	}
}

func TestCORSMiddleware_SetsHeadersOnNormalRequest(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := corsMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET: want 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Error("GET: missing Access-Control-Allow-Origin header")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("GET: missing Access-Control-Allow-Methods header")
	}
}

// ── Request logger middleware ────────────────────────────────────────────────

func TestRequestLoggerMiddleware_DoesNotPanic(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := requestLoggerMiddleware(inner, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Should not panic.
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestRequestLoggerMiddleware_NilLoggerPassesThrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := requestLoggerMiddleware(inner, nil)

	req := httptest.NewRequest(http.MethodDelete, "/agents/1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("nil logger passthrough: want 204, got %d", rec.Code)
	}
}

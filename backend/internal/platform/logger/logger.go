// Package logger provides a structured, context-aware logger that wraps
// the standard library log/slog package. It is the ONLY logging mechanism
// allowed in the backend. fmt.Println, fmt.Printf, and log.Println are
// forbidden in production code.
//
// Security invariant: this logger must never emit resolved LLM API key values.
// CredentialRef env var names (e.g. "COPILOT_API_KEY") are safe to log;
// the actual key value obtained by resolving that reference must never appear
// in any log line.
package logger

import (
	"context"
	"log/slog"
	"os"
)

// Logger is a thin context-aware wrapper around *slog.Logger.
// Use New to construct it; do not construct it directly.
type Logger struct {
	inner *slog.Logger
}

// New creates a Logger that writes JSON-structured output to stderr.
// level controls the minimum severity that will be emitted.
func New(level slog.Level) *Logger {
	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	return &Logger{inner: slog.New(h)}
}

// ── Core methods ─────────────────────────────────────────────────────────────

// Info logs a message at INFO level. args are key-value pairs or slog.Attr
// values, identical to slog.Logger.Info.
func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	l.inner.InfoContext(ctx, msg, args...)
}

// Warn logs a message at WARN level.
func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	l.inner.WarnContext(ctx, msg, args...)
}

// Error logs a message at ERROR level.
func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	l.inner.ErrorContext(ctx, msg, args...)
}

// Debug logs a message at DEBUG level.
func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	l.inner.DebugContext(ctx, msg, args...)
}

// ── Derived loggers ──────────────────────────────────────────────────────────

// With returns a new Logger that includes the given attributes on every
// subsequent log line. Use for request-scoped or component-scoped metadata
// (e.g. logger.With("module", "session", "session_id", id)).
func (l *Logger) With(args ...any) *Logger {
	return &Logger{inner: l.inner.With(args...)}
}

// Slog returns the underlying *slog.Logger.
// Use this when code outside the platform layer (e.g. handlers, repositories)
// requires a *slog.Logger directly.
func (l *Logger) Slog() *slog.Logger {
	return l.inner
}

// ── Sentinel ─────────────────────────────────────────────────────────────────

// Nop returns a Logger that discards all output. Useful in tests.
func Nop() *Logger {
	return &Logger{inner: slog.New(discardHandler{})}
}

// discardHandler is an slog.Handler that drops every record.
type discardHandler struct{}

func (discardHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (discardHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (d discardHandler) WithAttrs(_ []slog.Attr) slog.Handler        { return d }
func (d discardHandler) WithGroup(_ string) slog.Handler             { return d }

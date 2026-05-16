// Package main is the entry point for the brainstorm agent binary.
//
// Start-up sequence:
//  1. Read all configuration from env vars via agent/internal/config.
//  2. Warn if LLM credential is unavailable (agent still starts; calls fail fast).
//  3. Build AgentCard, LLMProvider, and BrainstormExecutor.
//  4. Wire HTTP routes: AgentCard handler + A2A REST handler.
//  5. Serve until SIGTERM/SIGINT, then graceful shutdown.
//
// Security invariant: os.Getenv is NEVER called here. All configuration is
// obtained through agent/internal/config, which is the sole allowed caller of
// os.Getenv in this binary.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2asrv"

	agentpkg "a2a-brainstorm/agent"
	"a2a-brainstorm/agent/internal/config"
	"a2a-brainstorm/agent/internal/executor"
	"a2a-brainstorm/agent/internal/llm"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.ErrorContext(context.Background(), "agent exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	port := config.GetPort()
	credentialRef := config.GetLLMCredentialRef()
	model := config.GetLLMModel()

	// Validate credential availability at startup — warn but do not abort so the
	// AgentCard endpoint remains servable even when the LLM key is temporarily
	// absent. Individual Execute calls will fail fast if the key is still missing.
	if _, err := config.GetLLMAPIKey(credentialRef); err != nil {
		logger.WarnContext(ctx, "LLM credential unavailable at startup",
			slog.String("credential_ref", credentialRef),
		)
	}

	// Build AgentCard. parsePort falls back to 9090 on parse failure.
	portInt := parsePort(port)
	card := agentpkg.NewAgentCard(portInt)

	// Build LLM provider.
	// config.GetLLMAPIKey is passed as resolveKey so that all os.Getenv calls
	// remain confined to agent/internal/config/config.go.
	llmProvider := llm.NewCopilotProvider(model, credentialRef, "", nil, config.GetLLMAPIKey)

	// Build executor.
	exec := executor.New(llmProvider, logger)

	// Build A2A request handler and REST transport adapter.
	handler := a2asrv.NewHandler(exec)
	restHandler := a2asrv.NewRESTHandler(handler)
	cardHandler := a2asrv.NewStaticAgentCardHandler(card)

	// Wire HTTP mux.
	mux := http.NewServeMux()
	mux.Handle(a2asrv.WellKnownAgentCardPath, cardHandler)
	mux.Handle("/", restHandler)

	// HTTP server with conservative timeouts to match LLM call duration.
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.InfoContext(ctx, "agent starting", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		logger.InfoContext(context.Background(), "shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return <-errCh
}

// parsePort converts a decimal port string to int, returning 9090 on failure.
func parsePort(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 9090
		}
		n = n*10 + int(c-'0')
	}
	if n == 0 {
		return 9090
	}
	return n
}

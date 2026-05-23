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
	"strings"
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

	// Build LLM provider (copilot by default; opencode when AGENT_LLM_PROVIDER=opencode).
	llmProvider := buildLLMProvider(logger)

	// Build AgentCard. The public URL is read from AGENT_PUBLIC_URL env var
	// (set to http://agent:{port} in Docker Compose so backend→agent calls work
	// via the Docker service name; defaults to http://localhost:{port}).
	card := agentpkg.NewAgentCard()

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

// buildLLMProvider constructs the LLMProvider selected by AGENT_LLM_PROVIDER.
//
// Supported values:
//   - "opencode" — proxies through a running OpenCode HTTP server instance.
//   - "copilot" (default) — calls the GitHub Copilot chat completions API directly.
//
// Security invariant: os.Getenv is never called here. All configuration is
// obtained through agent/internal/config; that package is the sole allowed
// caller of os.Getenv in this binary.
func buildLLMProvider(logger *slog.Logger) llm.LLMProvider {
	provider := config.GetLLMProvider()
	logger.Info("LLM provider", slog.String("provider", provider))

	switch provider {
	case "opencode":
		model := config.GetOpenCodeModel()
		parts := strings.SplitN(model, "/", 2)
		providerID, modelID := "github", "gpt-4o" // safe default
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			providerID, modelID = parts[0], parts[1]
		} else {
			logger.Warn("AGENT_OPENCODE_MODEL must be 'providerID/modelID'; using default github/gpt-4o",
				slog.String("value", model),
			)
		}
		return llm.NewOpenCodeProvider(llm.OpenCodeConfig{
			BaseURL:     config.GetOpenCodeBaseURL(),
			ProviderID:  providerID,
			ModelID:     modelID,
			UsernameRef: config.GetOpenCodeUsernameRef(),
			PasswordRef: config.GetOpenCodePasswordRef(),
		}, nil, config.GetLLMAPIKey)

	default: // "copilot" and any unrecognised value
		credentialRef := config.GetLLMCredentialRef()
		model := config.GetLLMModel()

		// Warn at startup if the credential env var is missing; the agent continues
		// to serve the AgentCard endpoint. Individual Execute calls will fail fast.
		if _, err := config.GetLLMAPIKey(credentialRef); err != nil {
			logger.Warn("LLM credential unavailable at startup",
				slog.String("credential_ref", credentialRef),
			)
		}
		return llm.NewCopilotProvider(model, credentialRef, "", nil, config.GetLLMAPIKey)
	}
}

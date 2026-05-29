// Package main is the entry point for the brainstorm agent binary.
//
// Start-up sequence:
//  1. Read all configuration from env vars via agent/internal/config.
//  2. Fail fast if LLM credential is unavailable (agent must not start without resolvable credentials).
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
	"fmt"
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

	// Build all available LLM providers. copilot is always the fallback.
	// opencode is added to the map when its credentials are available, but its
	// absence must NOT prevent the agent from starting — it is optional.
	providers, fallback, err := buildAllProviders(logger)
	if err != nil {
		return fmt.Errorf("startup: %w", err)
	}

	// Build AgentCard. The public URL is read from AGENT_PUBLIC_URL env var
	// (set to http://agent:{port} in Docker Compose so backend→agent calls work
	// via the Docker service name; defaults to http://localhost:{port}).
	card := agentpkg.NewAgentCard()

	// Build executor.
	exec := executor.New(providers, fallback, logger)

	// Build A2A request handler and REST transport adapter.
	handler := a2asrv.NewHandler(exec)
	restHandler := a2asrv.NewRESTHandler(handler)
	cardHandler := a2asrv.NewStaticAgentCardHandler(card)

	// Wire HTTP mux.
	mux := http.NewServeMux()
	mux.Handle(a2asrv.WellKnownAgentCardPath, cardHandler)
	mux.Handle("/", restHandler)

	// HTTP server — WriteTimeout must exceed the longest expected LLM call.
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      config.GetHTTPWriteTimeout(),
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

// buildAllProviders constructs all available LLMProviders and returns:
//   - providers: map of provider name → LLMProvider (always includes "copilot")
//   - fallback: the default LLMProvider to use when the requested name is absent
//
// copilot is always the fallback. opencode is added to the map only when its
// credentials are present; a missing opencode credential is a warning, not a
// fatal error — the agent falls back to copilot transparently.
//
// Security invariant: os.Getenv is never called here; all configuration is
// obtained through agent/internal/config.
func buildAllProviders(logger *slog.Logger) (providers map[string]llm.LLMProvider, fallback llm.LLMProvider, err error) {
	providers = make(map[string]llm.LLMProvider)

	// ── copilot (required fallback) ───────────────────────────────────────────
	copilotProvider, err := buildCopilotProvider()
	if err != nil {
		return nil, nil, fmt.Errorf("copilot provider: %w", err)
	}
	providers["copilot"] = copilotProvider
	fallback = copilotProvider
	logger.Info("LLM provider ready", slog.String("provider", "copilot"))

	// ── opencode (optional) ───────────────────────────────────────────────────
	opencodeProvider, opencodeErr := buildOpenCodeProvider()
	if opencodeErr == nil {
		providers["opencode"] = opencodeProvider
		logger.Info("LLM provider ready", slog.String("provider", "opencode"))
	} else {
		logger.Warn("opencode provider unavailable; requests for provider=opencode will fallback to copilot",
			slog.String("reason", opencodeErr.Error()),
		)
	}

	return providers, fallback, nil
}

// buildCopilotProvider constructs the GitHub Copilot LLMProvider.
// Returns an error if the credential env var is absent (security invariant).
func buildCopilotProvider() (llm.LLMProvider, error) {
	credentialRef := config.GetLLMCredentialRef()
	model := config.GetLLMModel()
	if _, err := config.GetLLMAPIKey(credentialRef); err != nil {
		return nil, fmt.Errorf("LLM credential %q is not set: set the env var before starting the agent", credentialRef)
	}
	return llm.NewCopilotProvider(model, credentialRef, "", nil, config.GetLLMAPIKey), nil
}

// buildOpenCodeProvider constructs the OpenCode LLMProvider.
// Returns an error (non-fatal) if credentials are absent or configuration is invalid.
func buildOpenCodeProvider() (llm.LLMProvider, error) {
	model := config.GetOpenCodeModel()
	parts := strings.SplitN(model, "/", 2)
	providerID, modelID := "github", "gpt-4o" // safe default
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		providerID, modelID = parts[0], parts[1]
	}
	usernameRef := config.GetOpenCodeUsernameRef()
	passwordRef := config.GetOpenCodePasswordRef()
	if _, err := config.GetLLMAPIKey(usernameRef); err != nil {
		return nil, fmt.Errorf("OpenCode username credential %q not set", usernameRef)
	}
	if _, err := config.GetLLMAPIKey(passwordRef); err != nil {
		return nil, fmt.Errorf("OpenCode password credential %q not set", passwordRef)
	}
	return llm.NewOpenCodeProvider(llm.OpenCodeConfig{
		BaseURL:     config.GetOpenCodeBaseURL(),
		ProviderID:  providerID,
		ModelID:     modelID,
		UsernameRef: usernameRef,
		PasswordRef: passwordRef,
	}, &http.Client{Timeout: config.GetOpenCodeHTTPTimeout()}, config.GetLLMAPIKey), nil
}

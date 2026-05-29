// Package main is the entry point for the brainstorm backend HTTP server.
//
// Start order:
//  1. Read configuration from environment variables (platform/config).
//  2. Open a PostgreSQL connection pool (platform/db).
//  3. Run pending SQL migrations.
//  4. Initialise all repositories, services, and handlers.
//  5. Build the HTTP router and start listening.
//  6. Block until SIGTERM or SIGINT, then gracefully shut down.
//
// Security invariant: os.Getenv is NEVER called here; all config comes from
// the platform/config package.
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

	agentmod "a2a-brainstorm/backend/internal/modules/agent"
	itermod "a2a-brainstorm/backend/internal/modules/iteration"
	"a2a-brainstorm/backend/internal/modules/markdown"
	"a2a-brainstorm/backend/internal/modules/markdown/aigen"
	sessmod "a2a-brainstorm/backend/internal/modules/session"
	"a2a-brainstorm/backend/internal/platform/config"
	"a2a-brainstorm/backend/internal/platform/db"
	platformHTTP "a2a-brainstorm/backend/internal/platform/http"
	"a2a-brainstorm/backend/internal/platform/llm"
	"a2a-brainstorm/backend/internal/platform/logger"
	"a2a-brainstorm/backend/internal/platform/sse"
)

func main() {
	log := logger.New(slog.LevelInfo)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, log); err != nil && !errors.Is(err, context.Canceled) {
		log.Error(context.Background(), "server exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {
	// ── Database ────────────────────────────────────────────────────────────
	pool, err := db.NewPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool, "migrations"); err != nil {
		return err
	}

	// ── Repositories ────────────────────────────────────────────────────────
	agentRepo := agentmod.NewRepository(pool)
	sessRepo := sessmod.NewRepository(pool)

	// ── Services ────────────────────────────────────────────────────────────
	agentSvc := agentmod.NewService(agentRepo, log.Slog())
	sessSvc := sessmod.NewService(sessRepo, agentSvc, log.Slog())

	// ── SSE broadcaster ─────────────────────────────────────────────────────
	broadcaster := sse.NewBroadcaster()

	// Inject the broadcaster into session service so finalize emits events.
	sessSvc.SetEmitter(broadcaster)

	iterEngine := itermod.NewEngine(agentmod.Dispatch, agentSvc, sessRepo, broadcaster, log.Slog())
	iterSvc := itermod.NewService(iterEngine, sessSvc, sessRepo, log.Slog())

	// ── Markdown writer ─────────────────────────────────────────────────────
	outputDir := config.GetOutputDir()
	mdWriter := buildMarkdownWriter(ctx, log.Slog())

	// ── Handlers ────────────────────────────────────────────────────────────
	agentHandler := agentmod.NewHandler(agentSvc, log.Slog())
	sessHandler := sessmod.NewHandler(sessSvc, mdWriter, outputDir, log.Slog())
	iterHandler := itermod.NewHandler(iterSvc, broadcaster, log.Slog())

	// ── Router ──────────────────────────────────────────────────────────────
	router := platformHTTP.NewRouter(platformHTTP.Deps{
		AgentHandler:     agentHandler,
		SessionHandler:   sessHandler,
		IterationHandler: iterHandler,
		Logger:           log.Slog(),
	})

	// ── HTTP server ─────────────────────────────────────────────────────────
	port := config.GetBackendPort()
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      300 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info(ctx, "server starting", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// Block until shutdown signal or ListenAndServe error.
	select {
	case <-ctx.Done():
		log.Info(context.Background(), "shutdown signal received")
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}

	// Graceful shutdown with a 15-second timeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return <-errCh
}

// buildMarkdownWriter constructs the session-handler markdownWriter dependency.
// In deterministic mode (or when prerequisites for the AI path are missing) it
// returns the historical *markdown.Writer. Otherwise it returns an Orchestrator
// wired to an AI generator backed by the global LLMProvider and the configured
// skill bundle.
func buildMarkdownWriter(ctx context.Context, log *slog.Logger) sessmod.MarkdownWriter {
	mode := markdown.ParseFinalizeMode(config.GetFinalizeMode())
	if mode == markdown.FinalizeModeDeterministic {
		return &markdown.Writer{}
	}

	credRef := config.GetGlobalLLMCredentialRef()
	if credRef == "" {
		log.Warn("aigen_fallback",
			slog.String("reason", "no global LLM credential configured"),
			slog.String("mode", string(mode)),
		)
		return &markdown.Writer{}
	}

	bundle, err := aigen.LoadBundle(os.DirFS("."), config.GetSkillBundlePaths())
	if err != nil {
		if mode == markdown.FinalizeModeAI {
			// In strict AI mode do NOT fall back to the deterministic writer —
			// the operator explicitly opted in to AI-only generation. Proceed
			// with an empty skill bundle so the generator still runs; the AI
			// will produce output based on canonical-state context alone.
			log.Warn("aigen_skill_load_partial",
				slog.String("reason", "skill files unavailable — AI generation continues with empty bundle"),
				slog.Any("error", err),
			)
			bundle = aigen.SkillBundle{}
		} else {
			log.Warn("aigen_fallback",
				slog.String("reason", "failed to load skill bundle"),
				slog.Any("error", err),
			)
			return &markdown.Writer{}
		}
	}

	llmCfg := llm.LLMConfig{
		Provider:      config.GetGlobalLLMProvider(),
		Model:         config.GetGlobalLLMModel(),
		CredentialRef: credRef,
	}
	provider := llm.NewCopilotProvider(llmCfg, "", nil)

	aiMode := aigen.ModeHybrid
	if mode == markdown.FinalizeModeAI {
		aiMode = aigen.ModeAI
	}

	gen := aigen.New(
		provider,
		bundle,
		config.GetAIDocMaxRepairs(),
		config.GetAIDocTemperature(),
		aiMode,
		log,
	)
	_ = ctx // reserved for future cancellation wiring
	return markdown.NewOrchestrator(mode, gen)
}

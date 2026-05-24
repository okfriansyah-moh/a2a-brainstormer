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
	sessmod "a2a-brainstorm/backend/internal/modules/session"
	"a2a-brainstorm/backend/internal/platform/config"
	"a2a-brainstorm/backend/internal/platform/db"
	platformHTTP "a2a-brainstorm/backend/internal/platform/http"
	"a2a-brainstorm/backend/internal/platform/logger"
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

	iterEngine := itermod.NewEngine(agentmod.Dispatch, agentSvc, sessRepo, log.Slog())
	iterSvc := itermod.NewService(iterEngine, sessSvc, log.Slog())

	// ── Markdown writer ─────────────────────────────────────────────────────
	outputDir := config.GetOutputDir()
	mdWriter := &markdown.Writer{}

	// ── Handlers ────────────────────────────────────────────────────────────
	agentHandler := agentmod.NewHandler(agentSvc, log.Slog())
	sessHandler := sessmod.NewHandler(sessSvc, mdWriter, outputDir, log.Slog())
	iterHandler := itermod.NewHandler(iterSvc, log.Slog())

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

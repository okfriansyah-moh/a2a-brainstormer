// Package db provides the PostgreSQL connection pool and migration runner.
// All database access in the backend must go through the pgxpool.Pool
// returned by NewPool. Modules must never construct their own connection.
package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"a2a-brainstorm/backend/internal/platform/config"
)

// NewPool opens a pgx connection pool using the DATABASE_URL from config.
// The caller is responsible for calling pool.Close() on shutdown.
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn, err := config.GetDatabaseURL()
	if err != nil {
		return nil, fmt.Errorf("db.NewPool: %w", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("db.NewPool: open pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db.NewPool: ping database: %w", err)
	}

	return pool, nil
}

// RunMigrations applies all *.sql files in migrationsDir in lexicographic
// (filename) order. Each file is executed in its own transaction; if any
// statement fails the transaction is rolled back and the error is returned
// immediately — subsequent files are not applied.
//
// This is a simple sequential runner. For production, consider a proper
// migration tool (golang-migrate, goose) after the MVP.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("db.RunMigrations: read dir %q: %w", migrationsDir, err)
	}

	// Collect .sql files and sort lexicographically (001_, 002_, …).
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(migrationsDir, e.Name()))
		}
	}
	sort.Strings(files)

	for _, path := range files {
		if err := applyMigration(ctx, pool, path); err != nil {
			return err
		}
	}
	return nil
}

// applyMigration reads a single SQL file and executes it inside a transaction.
func applyMigration(ctx context.Context, pool *pgxpool.Pool, path string) error {
	sql, err := os.ReadFile(path) // #nosec G304 — path is constructed from a controlled directory
	if err != nil {
		return fmt.Errorf("db.RunMigrations: read %q: %w", path, err)
	}

	content := strings.TrimSpace(string(sql))
	if content == "" || strings.HasPrefix(content, "--") && !strings.Contains(content, "\n") {
		// Skip comment-only stub files (Task 1 placeholders).
		return nil
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("db.RunMigrations: acquire conn for %q: %w", path, err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db.RunMigrations: begin tx for %q: %w", path, err)
	}

	if _, err := tx.Exec(ctx, content); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("db.RunMigrations: exec %q: %w", path, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("db.RunMigrations: commit %q: %w", path, err)
	}

	return nil
}

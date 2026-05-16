// Package session provides the repository layer for the sessions and
// session_agents tables.
//
// All DB access for this module must go through this file.
//
// Security invariants:
//   - All SQL uses positional parameters ($1, $2, …) — never string interpolation.
//   - JSONB columns are marshalled via encoding/json.
//   - os.Getenv is NOT called here.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/llm"
)

// ErrNotFound is returned when a requested session or session_agent does not exist.
var ErrNotFound = errors.New("not found")

// Repository provides all DB operations for the session domain.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository backed by the given pgx pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ── Session CRUD ──────────────────────────────────────────────────────────────

// CreateSession inserts a new sessions row and returns the persisted record.
// ID and timestamps are populated by the database.
func (r *Repository) CreateSession(ctx context.Context, s Session) (Session, error) {
	const q = `
		INSERT INTO sessions (idea, status, max_iterations, current_state)
		VALUES ($1, $2, $3, $4)
		RETURNING id, idea, status, max_iterations, current_state, created_at, updated_at`

	stateJSON, err := marshalState(s.CurrentState)
	if err != nil {
		return Session{}, fmt.Errorf("create session: marshal state: %w", err)
	}

	row := r.pool.QueryRow(ctx, q,
		s.Idea,
		s.Status,
		s.MaxIterations,
		stateJSON,
	)
	return scanSession(row)
}

// GetSession returns the session with the given ID, including its ordered
// SessionAgent list (populated via GetOrderedAgents internally).
func (r *Repository) GetSession(ctx context.Context, id string) (Session, error) {
	const q = `
		SELECT id, idea, status, max_iterations, current_state, created_at, updated_at
		FROM sessions
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id)
	s, err := scanSession(row)
	if err != nil {
		return Session{}, err
	}

	agents, err := r.GetOrderedAgents(ctx, id)
	if err != nil {
		return Session{}, fmt.Errorf("get session: load agents: %w", err)
	}
	s.Agents = agents
	return s, nil
}

// ListSessions returns all sessions ordered newest-first. Agents are not loaded.
func (r *Repository) ListSessions(ctx context.Context) ([]Session, error) {
	const q = `
		SELECT id, idea, status, max_iterations, current_state, created_at, updated_at
		FROM sessions
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("list sessions: scan: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// UpdateStatus sets the session status and bumps updated_at.
func (r *Repository) UpdateStatus(ctx context.Context, id, status string) error {
	const q = `
		UPDATE sessions SET status = $2, updated_at = now()
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, id, status)
	if err != nil {
		return fmt.Errorf("update session status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateState persists a new CanonicalState snapshot for the session and bumps
// updated_at. Called after each full iteration pipeline pass.
func (r *Repository) UpdateState(ctx context.Context, id string, cs *state.CanonicalState) error {
	const q = `
		UPDATE sessions SET current_state = $2, updated_at = now()
		WHERE id = $1`

	stateJSON, err := marshalState(cs)
	if err != nil {
		return fmt.Errorf("update state: marshal: %w", err)
	}

	tag, err := r.pool.Exec(ctx, q, id, stateJSON)
	if err != nil {
		return fmt.Errorf("update state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── SessionAgent operations ───────────────────────────────────────────────────

// CreateSessionAgents inserts all agent bindings for a session in a single
// batch. Uses ON CONFLICT DO NOTHING for idempotent re-runs.
func (r *Repository) CreateSessionAgents(ctx context.Context, agents []SessionAgent) error {
	const q = `
		INSERT INTO session_agents
		    (session_id, agent_id, position, role, llm_override, skill_overrides)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT DO NOTHING`

	for _, a := range agents {
		llmJSON, err := marshalLLMOverride(a.LLMOverride)
		if err != nil {
			return fmt.Errorf("create session agents: marshal llm override: %w", err)
		}
		skillJSON, err := marshalSkillOverrides(a.SkillOverrides)
		if err != nil {
			return fmt.Errorf("create session agents: marshal skill overrides: %w", err)
		}

		_, err = r.pool.Exec(ctx, q,
			a.SessionID,
			a.AgentID,
			a.Position,
			a.Role,
			llmJSON,
			skillJSON,
		)
		if err != nil {
			return fmt.Errorf("create session agents: insert position %d: %w", a.Position, err)
		}
	}
	return nil
}

// GetOrderedAgents returns all agent bindings for a session ordered by
// position ASC. This is the canonical pipeline order used by the iteration engine.
func (r *Repository) GetOrderedAgents(ctx context.Context, sessionID string) ([]SessionAgent, error) {
	const q = `
		SELECT session_id, agent_id, position, role, llm_override, skill_overrides
		FROM session_agents
		WHERE session_id = $1
		ORDER BY position ASC`

	rows, err := r.pool.Query(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get ordered agents: %w", err)
	}
	defer rows.Close()

	var agents []SessionAgent
	for rows.Next() {
		a, err := scanSessionAgent(rows)
		if err != nil {
			return nil, fmt.Errorf("get ordered agents: scan: %w", err)
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// ── Scan helpers ──────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSession(row rowScanner) (Session, error) {
	var (
		s         Session
		stateJSON []byte
	)
	err := row.Scan(
		&s.ID,
		&s.Idea,
		&s.Status,
		&s.MaxIterations,
		&stateJSON,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrNotFound
		}
		return Session{}, fmt.Errorf("scan session: %w", err)
	}
	if len(stateJSON) > 0 && string(stateJSON) != "null" {
		var cs state.CanonicalState
		if err := json.Unmarshal(stateJSON, &cs); err != nil {
			return Session{}, fmt.Errorf("scan session: unmarshal state: %w", err)
		}
		s.CurrentState = &cs
	}
	return s, nil
}

func scanSessionAgent(row rowScanner) (SessionAgent, error) {
	var (
		a         SessionAgent
		llmJSON   []byte
		skillJSON []byte
		updatedAt time.Time // not stored but needed if scanning extra cols in future
	)
	_ = updatedAt
	err := row.Scan(
		&a.SessionID,
		&a.AgentID,
		&a.Position,
		&a.Role,
		&llmJSON,
		&skillJSON,
	)
	if err != nil {
		return SessionAgent{}, fmt.Errorf("scan session agent: %w", err)
	}
	if len(llmJSON) > 0 && string(llmJSON) != "null" {
		var cfg llm.LLMConfig
		if err := json.Unmarshal(llmJSON, &cfg); err != nil {
			return SessionAgent{}, fmt.Errorf("scan session agent: unmarshal llm override: %w", err)
		}
		a.LLMOverride = &cfg
	}
	if len(skillJSON) > 0 && string(skillJSON) != "null" {
		var skills []string
		if err := json.Unmarshal(skillJSON, &skills); err != nil {
			return SessionAgent{}, fmt.Errorf("scan session agent: unmarshal skill overrides: %w", err)
		}
		a.SkillOverrides = &skills
	}
	return a, nil
}

// ── Marshal helpers ───────────────────────────────────────────────────────────

func marshalState(cs *state.CanonicalState) ([]byte, error) {
	if cs == nil {
		return nil, nil
	}
	return json.Marshal(cs)
}

func marshalLLMOverride(cfg *llm.LLMConfig) ([]byte, error) {
	if cfg == nil {
		return nil, nil
	}
	return json.Marshal(cfg)
}

// marshalSkillOverrides preserves the three-state distinction:
//   - nil pointer  → SQL NULL (use agent defaults)
//   - &[]          → JSON "[]" (disable all skills)
//   - &[id1,...]   → JSON "[id1,...]"
func marshalSkillOverrides(overrides *[]string) ([]byte, error) {
	if overrides == nil {
		return nil, nil
	}
	return json.Marshal(*overrides)
}

// Package agent provides the repository layer for the agents, skills, and
// agent_skills tables.
//
// All DB access for this module must go through this file.
//
// Security invariants enforced here:
//   - All SQL uses positional parameters ($1, $2, …) — never string interpolation.
//   - JSONB columns are marshalled via encoding/json; raw credential values are
//     never stored (CredentialRef holds the env-var name only).
//   - os.Getenv is NOT called here; credential resolution lives in config.go.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"a2a-brainstorm/backend/internal/platform/llm"
)

// ErrNotFound is returned when a requested entity does not exist in the DB.
var ErrNotFound = errors.New("not found")

// rowScanner is implemented by both pgx.Row and pgx.Rows, allowing the same
// scan helpers to handle single-row and multi-row results.
type rowScanner interface {
	Scan(dest ...any) error
}

// Repository holds a reference to the pgx connection pool and provides all DB
// operations for the agent and skill domain.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository backed by the given pgx pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ── Agent CRUD ────────────────────────────────────────────────────────────────

// CreateAgent inserts a new agent row and returns the persisted record.
// The id and created_at fields are populated by the database.
func (r *Repository) CreateAgent(ctx context.Context, a Agent) (Agent, error) {
	llmJSON, err := marshalLLMConfig(a.LLMConfig)
	if err != nil {
		return Agent{}, fmt.Errorf("create agent: marshal llm_config: %w", err)
	}
	const q = `
		INSERT INTO agents (name, description, default_role, system_prompt, llm_config, endpoint)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, description, default_role, system_prompt, llm_config, endpoint, created_at`
	row := r.pool.QueryRow(ctx, q,
		a.Name, nilIfEmpty(a.Description), string(a.DefaultRole),
		nilIfEmpty(a.SystemPrompt), llmJSON, a.Endpoint,
	)
	return scanAgent(row)
}

// GetAgent fetches a single agent by UUID string.
// Returns ErrNotFound when the agent does not exist.
func (r *Repository) GetAgent(ctx context.Context, id string) (Agent, error) {
	const q = `
		SELECT id, name, description, default_role, system_prompt, llm_config, endpoint, created_at
		FROM agents WHERE id = $1`
	a, err := scanAgent(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Agent{}, fmt.Errorf("get agent %s: %w", id, ErrNotFound)
		}
		return Agent{}, fmt.Errorf("get agent %s: %w", id, err)
	}
	return a, nil
}

// ListAgents returns all agents ordered by created_at ascending.
func (r *Repository) ListAgents(ctx context.Context) ([]Agent, error) {
	const q = `
		SELECT id, name, description, default_role, system_prompt, llm_config, endpoint, created_at
		FROM agents ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	agents := make([]Agent, 0)
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, fmt.Errorf("list agents: scan: %w", err)
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// UpdateAgent replaces the mutable fields of an existing agent and returns the
// updated record.  Returns ErrNotFound if no agent with the given ID exists.
func (r *Repository) UpdateAgent(ctx context.Context, a Agent) (Agent, error) {
	llmJSON, err := marshalLLMConfig(a.LLMConfig)
	if err != nil {
		return Agent{}, fmt.Errorf("update agent: marshal llm_config: %w", err)
	}
	const q = `
		UPDATE agents
		SET name = $2, description = $3, default_role = $4,
		    system_prompt = $5, llm_config = $6, endpoint = $7
		WHERE id = $1
		RETURNING id, name, description, default_role, system_prompt, llm_config, endpoint, created_at`
	updated, err := scanAgent(r.pool.QueryRow(ctx, q,
		a.ID, a.Name, nilIfEmpty(a.Description), string(a.DefaultRole),
		nilIfEmpty(a.SystemPrompt), llmJSON, a.Endpoint,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Agent{}, fmt.Errorf("update agent %s: %w", a.ID, ErrNotFound)
		}
		return Agent{}, fmt.Errorf("update agent %s: %w", a.ID, err)
	}
	return updated, nil
}

// DeleteAgent removes an agent by UUID.
// Returns ErrNotFound if no row was deleted.
func (r *Repository) DeleteAgent(ctx context.Context, id string) error {
	const q = `DELETE FROM agents WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete agent %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete agent %s: %w", id, ErrNotFound)
	}
	return nil
}

// ── Skill CRUD ────────────────────────────────────────────────────────────────

// CreateSkill inserts a new skill row and returns the persisted record.
func (r *Repository) CreateSkill(ctx context.Context, s Skill) (Skill, error) {
	const q = `
		INSERT INTO skills (name, description, prompt)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, prompt, created_at`
	row := r.pool.QueryRow(ctx, q, s.Name, nilIfEmpty(s.Description), s.Prompt)
	return scanSkill(row)
}

// GetSkill fetches a single skill by UUID.  Returns ErrNotFound if absent.
func (r *Repository) GetSkill(ctx context.Context, id string) (Skill, error) {
	const q = `SELECT id, name, description, prompt, created_at FROM skills WHERE id = $1`
	sk, err := scanSkill(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Skill{}, fmt.Errorf("get skill %s: %w", id, ErrNotFound)
		}
		return Skill{}, fmt.Errorf("get skill %s: %w", id, err)
	}
	return sk, nil
}

// ListSkills returns all skills ordered by created_at ascending.
func (r *Repository) ListSkills(ctx context.Context) ([]Skill, error) {
	const q = `SELECT id, name, description, prompt, created_at FROM skills ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()

	skills := make([]Skill, 0)
	for rows.Next() {
		sk, err := scanSkill(rows)
		if err != nil {
			return nil, fmt.Errorf("list skills: scan: %w", err)
		}
		skills = append(skills, sk)
	}
	return skills, rows.Err()
}

// UpdateSkill replaces the mutable fields of a skill and returns the updated record.
// Returns ErrNotFound if no skill with the given ID exists.
func (r *Repository) UpdateSkill(ctx context.Context, s Skill) (Skill, error) {
	const q = `
		UPDATE skills
		SET name = $2, description = $3, prompt = $4
		WHERE id = $1
		RETURNING id, name, description, prompt, created_at`
	updated, err := scanSkill(r.pool.QueryRow(ctx, q,
		s.ID, s.Name, nilIfEmpty(s.Description), s.Prompt,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Skill{}, fmt.Errorf("update skill %s: %w", s.ID, ErrNotFound)
		}
		return Skill{}, fmt.Errorf("update skill %s: %w", s.ID, err)
	}
	return updated, nil
}

// DeleteSkill removes a skill by UUID.  Returns ErrNotFound if no row was deleted.
func (r *Repository) DeleteSkill(ctx context.Context, id string) error {
	const q = `DELETE FROM skills WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete skill %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete skill %s: %w", id, ErrNotFound)
	}
	return nil
}

// ── Agent-Skill bindings ──────────────────────────────────────────────────────

// AttachSkill creates an agent_skills row.
// Uses ON CONFLICT DO NOTHING so re-attaching an already-attached skill is a
// safe no-op (idempotency per §idempotency skill).
func (r *Repository) AttachSkill(ctx context.Context, agentID, skillID string) error {
	const q = `
		INSERT INTO agent_skills (agent_id, skill_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`
	_, err := r.pool.Exec(ctx, q, agentID, skillID)
	if err != nil {
		return fmt.Errorf("attach skill %s to agent %s: %w", skillID, agentID, err)
	}
	return nil
}

// DetachSkill removes an agent_skills row.
// Returns ErrNotFound if the skill was not attached to the agent.
func (r *Repository) DetachSkill(ctx context.Context, agentID, skillID string) error {
	const q = `DELETE FROM agent_skills WHERE agent_id = $1 AND skill_id = $2`
	tag, err := r.pool.Exec(ctx, q, agentID, skillID)
	if err != nil {
		return fmt.Errorf("detach skill %s from agent %s: %w", skillID, agentID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("detach skill %s from agent %s: %w", skillID, agentID, ErrNotFound)
	}
	return nil
}

// GetAgentSkills returns all skills attached to agentID, ordered by skill
// creation time ascending.
func (r *Repository) GetAgentSkills(ctx context.Context, agentID string) ([]Skill, error) {
	const q = `
		SELECT s.id, s.name, s.description, s.prompt, s.created_at
		FROM skills s
		INNER JOIN agent_skills ags ON ags.skill_id = s.id
		WHERE ags.agent_id = $1
		ORDER BY s.created_at ASC`
	rows, err := r.pool.Query(ctx, q, agentID)
	if err != nil {
		return nil, fmt.Errorf("get agent skills for %s: %w", agentID, err)
	}
	defer rows.Close()

	skills := make([]Skill, 0)
	for rows.Next() {
		sk, err := scanSkill(rows)
		if err != nil {
			return nil, fmt.Errorf("get agent skills for %s: scan: %w", agentID, err)
		}
		skills = append(skills, sk)
	}
	return skills, rows.Err()
}

// ── scan helpers ──────────────────────────────────────────────────────────────

func scanAgent(row rowScanner) (Agent, error) {
	var a Agent
	var llmJSON []byte
	var defaultRole string
	var description, systemPrompt *string

	err := row.Scan(
		&a.ID, &a.Name, &description,
		&defaultRole, &systemPrompt, &llmJSON,
		&a.Endpoint, &a.CreatedAt,
	)
	if err != nil {
		return Agent{}, err
	}
	if description != nil {
		a.Description = *description
	}
	if systemPrompt != nil {
		a.SystemPrompt = *systemPrompt
	}
	a.DefaultRole = Role(defaultRole)
	a.Skills = []Skill{}
	if len(llmJSON) > 0 {
		var cfg llm.LLMConfig
		if err := json.Unmarshal(llmJSON, &cfg); err != nil {
			return Agent{}, fmt.Errorf("unmarshal llm_config: %w", err)
		}
		a.LLMConfig = &cfg
	}
	return a, nil
}

func scanSkill(row rowScanner) (Skill, error) {
	var s Skill
	var description *string
	err := row.Scan(&s.ID, &s.Name, &description, &s.Prompt, &s.CreatedAt)
	if err != nil {
		return Skill{}, err
	}
	if description != nil {
		s.Description = *description
	}
	return s, nil
}

// ── marshal helpers ───────────────────────────────────────────────────────────

// marshalLLMConfig serialises cfg to JSON bytes for JSONB storage.
// Returns nil for a nil config, which pgx maps to SQL NULL.
func marshalLLMConfig(cfg *llm.LLMConfig) ([]byte, error) {
	if cfg == nil {
		return nil, nil
	}
	return json.Marshal(cfg)
}

// nilIfEmpty converts an empty string to nil so the nullable TEXT column stores
// SQL NULL rather than an empty string.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

-- Migration: 003_sessions
-- Tables: sessions, session_agents

-- sessions stores the brainstorm session lifecycle and its canonical state
-- snapshot after each iteration pipeline pass.
CREATE TABLE IF NOT EXISTS sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idea            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active',  -- active | converged | approved | failed
    max_iterations  INT  NOT NULL DEFAULT 10,
    current_state   JSONB,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- session_agents binds agents to a session with pipeline order, assigned role,
-- and optional per-session LLM / skill overrides.
-- skill_overrides: NULL = use agent defaults; [] = disable all; [...] = use listed IDs.
CREATE TABLE IF NOT EXISTS session_agents (
    session_id      UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    agent_id        UUID NOT NULL REFERENCES agents(id)   ON DELETE RESTRICT,
    position        INT  NOT NULL,        -- 0-indexed pipeline order
    role            TEXT NOT NULL,        -- role assigned for this session
    llm_override    JSONB,                -- optional {provider,model,credential_ref}
    skill_overrides JSONB,                -- null | [] | ["uuid",...]
    PRIMARY KEY (session_id, agent_id)
);

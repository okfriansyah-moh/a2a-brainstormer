-- Migration 001: Agent registry, skill library, and agent-skill bindings.
-- Append-only — never modify this file after deployment.
-- See docs/PLAN.md §8.11 for the canonical schema definition.

CREATE TABLE IF NOT EXISTS agents (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT        NOT NULL,
    description    TEXT,
    default_role   TEXT        NOT NULL,
    system_prompt  TEXT,
    llm_config     JSONB,
    endpoint       TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT agents_name_unique UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS skills (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    description TEXT,
    prompt      TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT skills_name_unique UNIQUE (name)
);

-- agent_skills joins an agent to zero or more skills.
-- ON DELETE CASCADE ensures orphan rows are cleaned up automatically.
CREATE TABLE IF NOT EXISTS agent_skills (
    agent_id   UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    skill_id   UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (agent_id, skill_id)
);

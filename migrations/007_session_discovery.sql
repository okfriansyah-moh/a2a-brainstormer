-- Migration: 007_session_discovery
-- Guided onboarding v2: discovery answers, tech constraints, enriched idea.

ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS discovery_answers JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS tech_constraints JSONB NOT NULL DEFAULT '{"agents_decide":true}',
    ADD COLUMN IF NOT EXISTS enriched_idea TEXT NOT NULL DEFAULT '';

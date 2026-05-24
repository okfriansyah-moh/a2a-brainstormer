-- Migration 005: Add output_docs column to sessions
-- Adds selectable output document keys per session.
-- Valid keys: architecture, roadmap, plan, readme
-- Default: ['architecture','roadmap'] (backwards-compatible with existing sessions)

ALTER TABLE sessions
    ADD COLUMN output_docs TEXT[] NOT NULL DEFAULT ARRAY['architecture','roadmap'];

UPDATE sessions
    SET output_docs = ARRAY['architecture','roadmap']
    WHERE output_docs IS NULL;

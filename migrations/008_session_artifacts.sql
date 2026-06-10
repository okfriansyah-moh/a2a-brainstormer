-- Migration: 008_session_artifacts
-- Persist finalized document bodies for instant history/finalize loads.

CREATE TABLE IF NOT EXISTS session_artifacts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    doc_key       TEXT NOT NULL,
    filename      TEXT NOT NULL,
    content       TEXT NOT NULL,
    line_count    INT  NOT NULL,
    source        TEXT NOT NULL CHECK (source IN ('deterministic', 'hybrid', 'ai')),
    generated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (session_id, doc_key)
);

CREATE INDEX IF NOT EXISTS idx_session_artifacts_session ON session_artifacts (session_id);

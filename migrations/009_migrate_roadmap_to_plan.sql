-- Migration: 009_migrate_roadmap_to_plan
-- Deprecate roadmap output key: merge into plan and dedupe.

UPDATE sessions s
SET output_docs = sub.new_docs
FROM (
    SELECT
        id,
        (
            SELECT array_agg(DISTINCT elem ORDER BY elem)
            FROM unnest(
                CASE
                    WHEN 'plan' = ANY(output_docs)
                    THEN array_remove(output_docs, 'roadmap')
                    ELSE array_replace(output_docs, 'roadmap', 'plan')
                END
            ) AS elem
        ) AS new_docs
    FROM sessions
    WHERE 'roadmap' = ANY(output_docs)
) sub
WHERE s.id = sub.id;

-- 0003: transcript_versions and transcript_segments (PRD 13.3).
-- confidence is nullable: caption-derived segments carry no confidence
-- (PRD 14.5 null-confidence rule).

CREATE TABLE IF NOT EXISTS transcript_versions (
    transcript_version_id UUID PRIMARY KEY,
    job_id                UUID        NOT NULL REFERENCES jobs (job_id),
    version_type          TEXT        NOT NULL CHECK (version_type IN
                              ('raw', 'clean', 'reviewed', 'approved')),
    source_version_id     UUID        NULL REFERENCES transcript_versions (transcript_version_id),
    created_by            TEXT        NOT NULL,
    is_immutable          BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_transcript_versions_job ON transcript_versions (job_id, created_at);

CREATE TABLE IF NOT EXISTS transcript_segments (
    segment_id            UUID PRIMARY KEY,
    transcript_version_id UUID             NOT NULL REFERENCES transcript_versions (transcript_version_id),
    start_ms              INTEGER          NOT NULL,
    end_ms                INTEGER          NOT NULL,
    speaker_label         TEXT             NOT NULL,
    text                  TEXT             NOT NULL,
    confidence            DOUBLE PRECISION NULL,
    flags                 JSONB            NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_transcript_segments_version
    ON transcript_segments (transcript_version_id, start_ms);

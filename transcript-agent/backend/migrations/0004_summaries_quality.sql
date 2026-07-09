-- 0004: summaries and quality_reports (PRD 13.3).
-- confidence_unavailable marks caption-derived transcripts (PRD R5: segments
-- without provider confidence are exempt from threshold flagging).

CREATE TABLE IF NOT EXISTS summaries (
    summary_id                   UUID PRIMARY KEY,
    job_id                       UUID        NOT NULL REFERENCES jobs (job_id),
    source_transcript_version_id UUID        NOT NULL REFERENCES transcript_versions (transcript_version_id),
    text                         TEXT        NOT NULL,
    validation_status            TEXT        NOT NULL CHECK (validation_status IN
                                     ('passed', 'needs_review', 'failed')),
    validation_notes             TEXT        NULL,
    created_by                   TEXT        NOT NULL,
    created_at                   TIMESTAMPTZ NOT NULL,
    updated_at                   TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_summaries_job ON summaries (job_id, created_at);

CREATE TABLE IF NOT EXISTS quality_reports (
    quality_report_id            UUID PRIMARY KEY,
    job_id                       UUID             NOT NULL REFERENCES jobs (job_id),
    transcript_version_id        UUID             NOT NULL REFERENCES transcript_versions (transcript_version_id),
    job_config_id                UUID             NOT NULL REFERENCES job_config (job_config_id),
    confidence_threshold         DOUBLE PRECISION NOT NULL,
    quality_score                DOUBLE PRECISION NULL,
    average_confidence           DOUBLE PRECISION NULL,
    low_confidence_segment_count INTEGER          NOT NULL DEFAULT 0,
    coverage_gap_seconds         INTEGER          NOT NULL DEFAULT 0,
    timestamp_gap_count          INTEGER          NOT NULL DEFAULT 0,
    diarization_warning_count    INTEGER          NOT NULL DEFAULT 0,
    confidence_unavailable       BOOLEAN          NOT NULL DEFAULT FALSE,
    issue_summary_json           JSONB            NOT NULL DEFAULT '[]'::jsonb,
    created_at                   TIMESTAMPTZ      NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_quality_reports_job ON quality_reports (job_id, created_at);

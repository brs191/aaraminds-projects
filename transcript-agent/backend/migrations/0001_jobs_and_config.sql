-- 0001: jobs and job_config (PRD 13.3).
-- The canonical status enum is enforced with a CHECK constraint so the
-- lifecycle vocabulary in the database matches internal/domain/status.go.

CREATE TABLE IF NOT EXISTS jobs (
    job_id             UUID PRIMARY KEY,
    source_type        TEXT        NOT NULL CHECK (source_type IN ('youtube', 'upload')),
    source_uri         TEXT        NOT NULL,
    status             TEXT        NOT NULL CHECK (status IN (
                           'submitted', 'queued', 'validating', 'metadata_extracted',
                           'caption_checked', 'needs_user_action', 'extracting_audio',
                           'transcribing', 'normalizing', 'quality_checking', 'drafted',
                           'in_review', 'approved', 'exported', 'failed', 'cancelled')),
    submitted_by       TEXT        NOT NULL,
    ownership_attested BOOLEAN     NOT NULL DEFAULT FALSE,
    language           TEXT        NOT NULL DEFAULT 'en',
    job_config_id      UUID        NULL,
    duration_seconds   INTEGER     NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL,
    updated_at         TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs (status);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs (created_at DESC);

CREATE TABLE IF NOT EXISTS job_config (
    job_config_id          UUID PRIMARY KEY,
    job_id                 UUID             NOT NULL REFERENCES jobs (job_id),
    language               TEXT             NOT NULL DEFAULT 'en',
    confidence_threshold   DOUBLE PRECISION NOT NULL DEFAULT 0.80,
    enable_diarization     BOOLEAN          NOT NULL DEFAULT TRUE,
    expected_speaker_count INTEGER          NULL,
    style_policy_id        TEXT             NOT NULL DEFAULT 'default-clean-v1',
    summary_max_words      INTEGER          NOT NULL DEFAULT 150,
    summary_style          TEXT             NOT NULL DEFAULT 'neutral-professional',
    stt_provider           TEXT             NOT NULL,
    stt_model              TEXT             NULL,
    max_duration_seconds   INTEGER          NULL,
    created_by             TEXT             NOT NULL,
    created_at             TIMESTAMPTZ      NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_job_config_job ON job_config (job_id);

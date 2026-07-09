-- 0006: exports table plus job runtime columns required by the frozen REST
-- contract. These extend the PRD 13.3 suggested schema:
--   * exports: R8 requires export artifacts versioned and linked to the
--     approved transcript version, and the API lists them individually.
--   * jobs.action_required / last_error_*: surfaced on the Job JSON.
--   * jobs.captions_available / caption_track_id / caption_reuse: persist the
--     caption pre-check result and the producer's reuse decision (11.1 step 6)
--     across restarts.
--   * jobs.cancel_reason: cancel_job audit support (14.14).

CREATE TABLE IF NOT EXISTS exports (
    export_id                      UUID PRIMARY KEY,
    job_id                         UUID        NOT NULL REFERENCES jobs (job_id),
    approved_transcript_version_id UUID        NOT NULL REFERENCES transcript_versions (transcript_version_id),
    format                         TEXT        NOT NULL CHECK (format IN ('txt', 'md', 'srt', 'vtt')),
    artifact_uri                   TEXT        NOT NULL,
    validation_status              TEXT        NOT NULL CHECK (validation_status IN ('passed', 'failed')),
    created_by                     TEXT        NOT NULL,
    created_at                     TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_exports_job ON exports (job_id, created_at);

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS action_required    TEXT    NOT NULL DEFAULT '';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS last_error_code    TEXT    NOT NULL DEFAULT '';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS last_error_message TEXT    NOT NULL DEFAULT '';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS captions_available BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS caption_track_id   TEXT    NOT NULL DEFAULT '';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS caption_reuse      BOOLEAN NULL;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS cancel_reason      TEXT    NOT NULL DEFAULT '';

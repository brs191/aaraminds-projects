-- 0002: media_artifacts (PRD 13.3). `superseded` supports replace_job_media
-- (PRD 14.13: prior artifacts stay under retention, marked superseded).

CREATE TABLE IF NOT EXISTS media_artifacts (
    artifact_id     UUID PRIMARY KEY,
    job_id          UUID        NOT NULL REFERENCES jobs (job_id),
    artifact_type   TEXT        NOT NULL CHECK (artifact_type IN
                        ('source_media', 'audio_extract', 'caption_source', 'export')),
    uri             TEXT        NOT NULL,
    mime_type       TEXT        NOT NULL,
    size_bytes      BIGINT      NOT NULL DEFAULT 0,
    superseded      BOOLEAN     NOT NULL DEFAULT FALSE,
    retention_until TIMESTAMPTZ NULL,
    created_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_media_artifacts_job ON media_artifacts (job_id, artifact_type);

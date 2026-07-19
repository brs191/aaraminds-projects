-- 0009: library mode — podcast feeds, episodes, and library job flags.
-- Library jobs are normal jobs with library_mode = TRUE: the pipeline stops at
-- drafted (no review gate) and the ownership basis is recorded programmatically
-- (source_basis = 'open_rss_personal_use') in lieu of the manual attestation.

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS library_mode BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS source_basis TEXT    NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS feeds (
    feed_id         UUID PRIMARY KEY,
    feed_url        TEXT        NOT NULL UNIQUE,
    title           TEXT        NOT NULL DEFAULT '',
    description     TEXT        NOT NULL DEFAULT '',
    image_url       TEXT        NULL,
    auto_transcribe BOOLEAN     NOT NULL DEFAULT FALSE,
    last_polled_at  TIMESTAMPTZ NULL,
    poll_error      TEXT        NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    deleted_at      TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS episodes (
    episode_id        UUID PRIMARY KEY,
    feed_id           UUID        NOT NULL REFERENCES feeds (feed_id),
    guid              TEXT        NOT NULL,
    title             TEXT        NOT NULL DEFAULT '',
    description       TEXT        NOT NULL DEFAULT '',
    audio_url         TEXT        NOT NULL,
    published_at      TIMESTAMPTZ NULL,
    duration_seconds  INTEGER     NULL,
    media_artifact_id UUID        NULL,
    job_id            UUID        NULL REFERENCES jobs (job_id),
    created_at        TIMESTAMPTZ NOT NULL,
    UNIQUE (feed_id, guid)
);

CREATE INDEX IF NOT EXISTS idx_episodes_feed ON episodes (feed_id, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_episodes_job ON episodes (job_id) WHERE job_id IS NOT NULL;

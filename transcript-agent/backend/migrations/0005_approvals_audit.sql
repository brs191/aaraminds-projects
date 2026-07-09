-- 0005: approvals and audit_events (PRD 13.3).
-- approvals.superseded_by_approval_id records the post-approval correction
-- chain (PRD 11.4): approvals are never deleted or edited in place.

CREATE TABLE IF NOT EXISTS approvals (
    approval_id                    UUID PRIMARY KEY,
    job_id                         UUID        NOT NULL REFERENCES jobs (job_id),
    approved_transcript_version_id UUID        NOT NULL REFERENCES transcript_versions (transcript_version_id),
    approved_by                    TEXT        NOT NULL,
    approved_at                    TIMESTAMPTZ NOT NULL,
    approval_note                  TEXT        NULL,
    superseded_by_approval_id      UUID        NULL REFERENCES approvals (approval_id)
);

CREATE INDEX IF NOT EXISTS idx_approvals_job ON approvals (job_id, approved_at);

-- Append-only control record (PRD 13.1). No UPDATE/DELETE is ever issued by
-- the application; revoke as defense-in-depth where role setup allows.
CREATE TABLE IF NOT EXISTS audit_events (
    audit_event_id UUID PRIMARY KEY,
    job_id         UUID        NULL REFERENCES jobs (job_id),
    actor_type     TEXT        NOT NULL CHECK (actor_type IN ('user', 'system', 'tool')),
    actor_id       TEXT        NOT NULL,
    event_type     TEXT        NOT NULL,
    event_payload  JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_events_job ON audit_events (job_id, created_at);

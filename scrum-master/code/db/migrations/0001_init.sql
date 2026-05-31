-- 0001_init.sql — Scrum Master Agent domain schema (P0)
--
-- Auto-applied by the postgres container on first init (this dir is mounted into
-- /docker-entrypoint-initdb.d). LangGraph's checkpointer manages its OWN tables via
-- checkpointer.setup() at runtime — those are separate from this domain schema.
--
-- Estimation is TIME-BASED (locked decision): time_* columns are seconds; there is
-- deliberately no story-points column.

CREATE TABLE IF NOT EXISTS team_config (
    id                  BIGSERIAL PRIMARY KEY,
    board_id            TEXT NOT NULL UNIQUE,
    team_name           TEXT NOT NULL,
    channel             TEXT NOT NULL DEFAULT 'teams',
    sprint_cadence      TEXT,
    definition_of_ready JSONB NOT NULL DEFAULT '{}'::jsonb,
    thresholds          JSONB NOT NULL DEFAULT '{"stale_days": 3}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sprint_snapshot (
    id          BIGSERIAL PRIMARY KEY,
    board_id    TEXT NOT NULL,
    sprint_id   TEXT NOT NULL,
    name        TEXT,
    state       TEXT,
    goal        TEXT,
    start_date  TIMESTAMPTZ,
    end_date    TIMESTAMPTZ,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    raw         JSONB
);
CREATE INDEX IF NOT EXISTS idx_sprint_snapshot_sprint ON sprint_snapshot (sprint_id, captured_at DESC);

CREATE TABLE IF NOT EXISTS issue_snapshot (
    id                     BIGSERIAL PRIMARY KEY,
    sprint_id              TEXT NOT NULL,
    issue_key              TEXT NOT NULL,
    summary                TEXT,
    status                 TEXT,
    assignee               TEXT,
    blocked                BOOLEAN NOT NULL DEFAULT false,
    block_reason           TEXT,
    time_original_estimate INTEGER,   -- seconds (time-based estimation)
    time_estimate          INTEGER,   -- seconds remaining
    time_spent             INTEGER,   -- seconds logged
    days_in_status         INTEGER,
    updated                TIMESTAMPTZ,
    captured_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    raw                    JSONB
);
CREATE INDEX IF NOT EXISTS idx_issue_snapshot_key ON issue_snapshot (issue_key, captured_at DESC);
CREATE INDEX IF NOT EXISTS idx_issue_snapshot_sprint ON issue_snapshot (sprint_id, captured_at DESC);

-- --- Trust chain: recommendation -> approval -> action_audit -----------------

CREATE TABLE IF NOT EXISTS recommendation (
    id         BIGSERIAL PRIMARY KEY,
    kind       TEXT NOT NULL,            -- daily_brief | blocker | story_quality | retro ...
    payload    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS approval (
    id                BIGSERIAL PRIMARY KEY,
    recommendation_id BIGINT NOT NULL REFERENCES recommendation(id) ON DELETE CASCADE,
    decision          TEXT NOT NULL,     -- approved | rejected | pending
    decided_by        TEXT NOT NULL,
    decided_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_approval_rec ON approval (recommendation_id);

CREATE TABLE IF NOT EXISTS action_audit (
    id                BIGSERIAL PRIMARY KEY,
    recommendation_id BIGINT NOT NULL REFERENCES recommendation(id) ON DELETE CASCADE,
    action            TEXT NOT NULL,     -- post_to_teams | add_comment | add_label | create_subtask | generate_report
    result            TEXT NOT NULL,     -- delivered | logged | failed ...
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_action_audit_rec ON action_audit (recommendation_id);

CREATE TABLE IF NOT EXISTS metric_event (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    value      DOUBLE PRECISION,
    tags       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_metric_event_name ON metric_event (name, created_at DESC);

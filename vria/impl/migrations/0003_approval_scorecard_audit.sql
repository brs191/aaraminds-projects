-- Generated from contracts/19_VRIA_Physical_Data_Model.md (v1.3). Do not hand-edit; regenerate.

create table approval_requests (
  approval_id uuid primary key,
  action_type text not null,
  target_id text not null,
  target_type text not null,
  requested_by text not null,
  approver_ids jsonb not null,
  approval_state text not null
    check (approval_state in ('Draft','Submitted','ChangesRequested','Approved','Rejected','Withdrawn')),
  risk_tier text,
  rationale text,
  submitted_at timestamptz not null default now(),
  decided_at timestamptz,
  decided_by text,
  decision_comments text
);

create table scorecards (
  scorecard_id uuid primary key,
  title text not null,
  summary text,
  period_start date not null,
  period_end date not null,
  evidence_coverage_summary jsonb not null default '{}'::jsonb,
  artifact_state text not null default 'Draft'
    check (artifact_state in ('Draft','Approved','Published','Superseded','Invalidated')),
  decision_log_pointer uuid,
  published_at timestamptz,
  supersedes_scorecard_id uuid references scorecards(scorecard_id),
  created_by text,
  created_at timestamptz not null default now(),
  constraint published_requires_timestamp
    check (artifact_state <> 'Published' or published_at is not null)
);

create table scorecard_items (
  scorecard_id uuid references scorecards(scorecard_id),
  assessment_id uuid references value_assessments(assessment_id),
  use_case_id text references use_cases(use_case_id),
  primary key (scorecard_id, assessment_id)
);

create table decision_log (
  decision_record_id uuid primary key,
  decision_type text not null,
  target_id text not null,
  decision text not null,
  rationale text,
  decided_by text not null,
  approval_id uuid references approval_requests(approval_id),
  created_at timestamptz not null default now()
);

create table audit_events (
  audit_id uuid primary key,
  actor_id text,
  action text not null,
  target_type text,
  target_id text,
  input_hash text,
  output_hash text,
  trace_id text,
  created_at timestamptz not null default now()
);

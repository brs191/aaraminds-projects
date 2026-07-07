# VRIA Physical Data Model

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This document defines the production-oriented PostgreSQL data model for VRIA.

## 2. Design Principles

- Use UUID primary keys for generated records.
- Keep source system IDs as external IDs.
- Store assessments and scorecards as immutable snapshots.
- Append decisions and audit events; do not update them in place.
- Version prompts, models, schemas, tools, and scoring rules.
- Enforce RBAC and row-level security where tenant/team separation is required.

## 3. Core Tables

```sql
create table use_cases (
  use_case_id text primary key,
  name text not null,
  tier text not null check (tier in ('Tool','Agent','Layer','Unclassified')),
  domain text,
  value_owner text,
  delivery_owner text,
  sponsor text,
  delivery_status text not null default 'Unknown',
  primary_metric_id text,
  approval_state text not null default 'Draft'
    check (approval_state in ('Draft','Approved','Published','Superseded','Invalidated')),
  record_version integer not null default 1,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table value_hypotheses (
  value_hypothesis_id uuid primary key,
  use_case_id text not null references use_cases(use_case_id),
  business_objective text,
  expected_benefit text,
  benefit_type text,
  primary_metric_id text,
  baseline_value text,
  baseline_period_start date,
  baseline_period_end date,
  target_value text,
  target_period_start date,
  target_period_end date,
  initiative_cost numeric,
  initiative_cost_currency text,
  initiative_cost_period_start date,
  initiative_cost_period_end date,
  attribution_method text not null default 'Unknown',
  known_confounders jsonb not null default '[]'::jsonb,
  net_value_check text not null default 'Unknown',
  approval_state text not null default 'Draft',
  record_version integer not null default 1,
  created_at timestamptz not null default now()
);

create table evidence_sources (
  evidence_source_id uuid primary key,
  use_case_id text references use_cases(use_case_id),
  source_type text not null,
  source_system text not null,
  source_owner text,
  citation_pointer text,
  authority text not null default 'Unknown',
  freshness text not null default 'Unknown',
  access_classification text not null default 'Internal',
  retrieved_at timestamptz,
  content_hash text
);

create table metric_snapshots (
  metric_snapshot_id uuid primary key,
  metric_id text not null,
  use_case_id text references use_cases(use_case_id),
  period_start date,
  period_end date,
  baseline_value text,
  current_value text,
  target_value text,
  metric_unit text,
  source_system text,
  source_owner text,
  authority text not null default 'Unknown',
  freshness text not null default 'Unknown',
  initiative_cost numeric,
  initiative_cost_currency text,
  created_at timestamptz not null default now()
);

create table value_assessments (
  assessment_id uuid primary key,
  use_case_id text references use_cases(use_case_id),
  value_state text not null
    check (value_state in ('NotReady','HypothesisOnly','BaselineReady','OnTrack','AtRisk','Realized','NotRealized','Regressed','Unproven')),
  realization_score integer not null check (realization_score between 0 and 100),
  pre_cap_score integer not null check (pre_cap_score between 0 and 100),
  score_breakdown jsonb not null,
  applied_caps jsonb not null default '[]'::jsonb,
  recommendation text not null,
  confidence text not null,
  attribution_method text,
  known_confounders jsonb not null default '[]'::jsonb,
  net_value_check text,
  initiative_cost numeric,
  initiative_cost_currency text,
  initiative_cost_period_start date,
  initiative_cost_period_end date,
  sustainment_threshold numeric,
  sustainment_status text not null default 'NotStarted'
    check (sustainment_status in ('NotStarted','Ok','AtRisk','Regressed')),
  missing_evidence jsonb not null default '[]'::jsonb,
  rationale text,
  approval_state text not null default 'Draft'
    check (approval_state in ('Draft','Approved','Published','Superseded','Invalidated')),
  scoring_rule_version text not null,
  model_version text,
  prompt_version text,
  created_at timestamptz not null default now()
);
```

```sql
create table assessment_evidence (
  assessment_id uuid not null references value_assessments(assessment_id),
  evidence_source_id uuid not null references evidence_sources(evidence_source_id),
  citation_pointer text,
  primary key (assessment_id, evidence_source_id)
);
```

## 4. Approval, Scorecard, and Audit Tables

```sql
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
```

## 5. Indexes

```sql
create index idx_use_cases_tier on use_cases(tier);
create index idx_use_cases_status on use_cases(delivery_status);
create index idx_value_assessments_use_case_created on value_assessments(use_case_id, created_at desc);
create index idx_value_assessments_state on value_assessments(value_state);
create index idx_approval_state on approval_requests(approval_state);
create index idx_evidence_use_case on evidence_sources(use_case_id);
create index idx_metric_snapshots_metric on metric_snapshots(metric_id, period_end desc);
create index idx_metric_snapshots_use_case on metric_snapshots(use_case_id);
create index idx_audit_events_target on audit_events(target_type, target_id, created_at desc);
create index idx_audit_events_trace on audit_events(trace_id);
create index idx_decision_log_target on decision_log(target_id, created_at desc);
create index idx_assessment_evidence_source on assessment_evidence(evidence_source_id);
```

## 5a. Immutability Enforcement

`decision_log`, `audit_events`, and published `scorecards`/`value_assessments` are append-only. Enforce at the database, not the service:

```sql
revoke update, delete on decision_log, audit_events from vria_app;
create or replace function block_published_mutation() returns trigger as $$
begin
  if old.artifact_state = 'Published' and tg_op = 'UPDATE'
     and new.artifact_state = old.artifact_state then
    raise exception 'published scorecards are superseded, not edited';
  end if;
  return new;
end $$ language plpgsql;
create trigger trg_scorecards_immutable before update on scorecards
  for each row execute function block_published_mutation();
```

## 6. Migration and Versioning

All schema changes require a migration file, data backfill plan, rollback plan, and golden eval run. Assessment snapshots must retain scoring rule, prompt, model, and schema version.

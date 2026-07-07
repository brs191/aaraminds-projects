-- Generated from contracts/19_VRIA_Physical_Data_Model.md (v1.3). Do not hand-edit; regenerate.

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

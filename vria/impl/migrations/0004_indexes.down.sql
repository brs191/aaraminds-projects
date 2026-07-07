-- Rollback for 0004_indexes.sql
drop index if exists idx_use_cases_tier; drop index if exists idx_use_cases_status;
drop index if exists idx_value_assessments_use_case_created; drop index if exists idx_value_assessments_state;
drop index if exists idx_approval_state; drop index if exists idx_evidence_use_case;
drop index if exists idx_metric_snapshots_metric; drop index if exists idx_metric_snapshots_use_case;
drop index if exists idx_audit_events_target; drop index if exists idx_audit_events_trace;
drop index if exists idx_decision_log_target; drop index if exists idx_assessment_evidence_source;

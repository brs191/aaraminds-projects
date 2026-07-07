-- Generated from contracts/19_VRIA_Physical_Data_Model.md (v1.3). Do not hand-edit; regenerate.

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

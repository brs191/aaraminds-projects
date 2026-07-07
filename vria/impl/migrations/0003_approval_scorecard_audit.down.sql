-- Rollback for 0003_approval_scorecard_audit.sql
drop table if exists audit_events;
drop table if exists decision_log;
drop table if exists scorecard_items;
drop table if exists scorecards;
drop table if exists approval_requests;

-- Rollback for 0005_immutability.sql
drop trigger if exists trg_scorecards_immutable on scorecards;
drop function if exists block_published_mutation();
grant update, delete on decision_log, audit_events to vria_app;

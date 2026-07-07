-- Generated from contracts/19_VRIA_Physical_Data_Model.md (v1.3). Do not hand-edit; regenerate.

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

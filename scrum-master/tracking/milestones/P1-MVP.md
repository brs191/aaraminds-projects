# P1 — MVP

**Phase gate:** all 5 features run on a live sprint; every write passes the approval gate; the pilot SM confirms Daily Brief + Blocker detection are accurate.
**Status:** blocked by P0 · **Blocks:** P2

## Deliverables
- [ ] Daily Scrum Brief
- [ ] Sprint Health Summary
- [ ] Blocker & Stale Detection
- [ ] Story Quality Review
- [ ] Sprint Closing / Retro Insights
- [ ] Approval queue + durable HITL gate (LangGraph checkpointer)
- [ ] Gated writes: comment, label, follow-up sub-task, generate `Report.md` (with TOC)
- [ ] Sprint Health + Story Quality use Jira time-tracking fields (time-based estimation)
- [ ] Scheduler (daily brief) + webhook listener (sprint / issue events)
- [ ] `recommendation → approval → action_audit`
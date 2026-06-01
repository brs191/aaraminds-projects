# Evaluation Rubric — Agent Quality Bar

**Owner:** Raja · **Stage:** quality gate

The Definition of Done for agent output. A feature ships only when it clears this bar on a live sprint.

## Accuracy
- Recommendations reference real, current issue state — no hallucinated keys or fields
- Blocker/stale false-positive rate within target; **zero** false positives on Done items
- Sprint-health signals reconcile with Jira's own burndown within tolerance

## Trust & transparency
- Every recommendation cites issue key(s) and the triggering signal
- No silent writes — 100% of writes traceable to an approval record
- Failure modes are explicit (the agent says "insufficient data," never guesses)

## Safety
- Write surface limited to the MVP set (comment, label, sub-task, generated `Report.md`)
- No status transitions, description edits, or deletions in MVP
- Sensitive fields redacted before being sent to the LLM; tenant isolation holds

## Usefulness
- Pilot SM agrees the Daily Brief is standup-ready with minimal edits
- Recommendations are specific and actionable, not generic advice

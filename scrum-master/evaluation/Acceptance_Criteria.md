# Acceptance Criteria — MVP Features

**Owner:** Raja · **Stage:** test target · **Source:** `../Scrum_Master_Agent_PRD.md` §6

Each feature is "done" only when its criteria pass on a live sprint.

## 1. Daily Scrum Brief
- [ ] Generates a brief for the active sprint, grouped by assignee
- [ ] Flags blocked and stalled items with issue keys
- [ ] Runs on schedule before standup; posts to Teams
- [ ] (Gated) can post the brief as a sprint comment on approval

## 2. Sprint Health Summary
- [ ] Computes completed vs. remaining time (original vs. remaining estimate)
- [ ] Detects scope added after sprint start
- [ ] Flags spillover risk with rationale (not just a RAG color)

## 3. Blocker & Stale Ticket Detection
- [ ] Detects dependency-blocked and time-in-status thresholds
- [ ] Surfaces age; no false positives on Done items
- [ ] Thresholds configurable per team
- [ ] (Gated) comment / label / follow-up sub-task on approval

## 4. Story Quality Review
- [ ] Flags missing acceptance criteria / estimate / owner
- [ ] Gives a concrete rewrite suggestion
- [ ] Runs on backlog + next-sprint candidates
- [ ] Never auto-edits the description (comment only)

## 5. Sprint Closing / Retro Insights
- [ ] Summarizes completion %, spillover, cycle-time trend
- [ ] Surfaces recurring blockers across the last K sprints with evidence
- [ ] (Gated) generates a `Report.md` with a navigable table of contents on approval

## Cross-cutting
- [ ] Every recommendation cites issue key(s) and the triggering signal
- [ ] No writ
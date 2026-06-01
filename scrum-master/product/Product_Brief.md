# Product Brief — Scrum Master Agent

**Owner:** Raja · **Stage:** stable reference · **Source:** `../Scrum_Master_Agent_PRD.md` §1–5

## The "why"

Scrum Masters and tech leads spend hours per sprint on mechanical work — chasing status, assembling standup notes, spotting stalled tickets, checking story readiness, writing retro summaries. The signal already lives in Jira but is scattered across boards, changelogs, and comments. SM time goes to data-gathering instead of facilitation and impediment removal, and quality issues surface late.

## Who it's for

- **Scrum Master / Agile lead** — standup prep, blocker triage, retro synthesis, sprint health
- **Engineering manager / tech lead** — sprint health, spillover risk, delivery visibility
- **Product owner** — story quality, backlog readiness
- **Team members** — daily brief, "what's blocked / waiting on me"

## The wedge

Existing tools cover slices — Spinach (standups/summaries), Rovo/Jira AI (Jira-native agents), ScrumGenius (async standups), Parabol (retros), LinearB (delivery analytics). None own the full Scrum Master loop anchored on Jira with a disciplined human-in-the-loop write model. That gap is the wedge.

## Product principles

1. **Jira is the system of record** — the agent never holds authoritative state.
2. **Advisory by default** — Read → Analyze → Recommend → Approve → Write. No silent writes in MVP.
3. **Show your sources** — every recommendation cites issue key(s) and the signal.
4. **Transparency over magic** — explained-but-wrong is recoverable; a confident black box kills trust.
5. **Minimum viable write surface** — MVP writes limited to comment, label, follow-up sub-task, and a generated `Report.md` (with table of contents).

## MVP in one line

Five features — Daily Scrum Brief, Sprint Health Summary, Blocker & Stale Detection, Story Quality Review, Sprint Closing/Retro Insights — all advisory or gated-write, on one team, one channel.

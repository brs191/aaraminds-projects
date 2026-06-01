# Roadmap — Scrum Master Agent

**Owner:** Raja · **Stage:** durable plan · **Source:** `../Scrum_Master_Agent_PRD.md` §12

Four phases, each with a **gate**. A phase is not done until its gate passes — gates, not checkbox counts, govern progress. Live state in `../tracking/Status.md`.

## P0 — Foundations

Jira MCP integration (read), OAuth 3LO auth, board & sprint config, Postgres, snapshot + changelog ingestion, one channel (Microsoft Teams).

**Gate:** the agent can authenticate (OAuth 3LO), ingest the active sprint of one real board, and post a hello-world message to Teams.

## P1 — MVP

The 5 features (PRD §6), advisory + gated comment/label/report writes, scheduler + webhook listener, approval queue. Pilot on one team.

**Gate:** all 5 features run on a live sprint; every write passes the approval gate; the pilot SM confirms the Daily Brief and Blocker detection are accurate (cite-checked).

## P2 — Expand

Backlog grooming assistant, sprint planning (capacity), Jira hygiene sweeps, Slack + Web UI, optional Confluence publishing (reports already generated as `Report.md` in MVP).

**Gate:** a second team is onboarded with config only (no code changes); Slack reaches parity with Teams.

## P3 — Controlled autonomy

Policy-bounded auto-actions, multi-team rollups, in-Atlassian surface via Rovo/Forge agent, analytics dashboard.

**Gate:** at least one auto-action runs under policy with a full audit trail and a tested rollback / undo.

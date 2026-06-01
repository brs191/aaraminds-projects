# Architecture — Scrum Master Agent

**Owner:** Raja · **Stage:** design · **Source:** `../Scrum_Master_Agent_PRD.md` §7–9 · **Key decision:** [ADR-0001](adr/0001-langgraph-orchestration.md)

## Component view

```
Teams (P0) · Slack (P2) · Web UI    ← channel adapters (Go)
        │
   LangGraph orchestrator ────────→ LLM (Claude / GPT)
   (reasoning + durable HITL gate via Postgres checkpointer)
        │
   Jira Integration (Go MCP server) ── read + gated write
        │
   Approval queue · Scheduler · Webhook listener (Go)
        │
   Postgres (Azure): config · snapshots · checkpoints · audit
        │
   Jira Cloud → Boards · Sprints · Issues · Backlog · Changelog
```

**Defining Operational Constraint:** *human-approved writes by construction* — no Jira/channel mutation without a persisted approval. Enforced by the `interrupt()` gate + the `recommendation → approval → action_audit` chain. Trust boundaries and failure modes: [Agent_Blueprint.md](Agent_Blueprint.md) §6–8.

## Stack (decided)

- **Orchestration:** LangGraph (Python) — durable interrupt/checkpoint primitives *are* the approval gate. See ADR-0001.
- **Jira integration:** Go MCP server (reuses the skills-pack pattern), consumed via `langchain-mcp-adapters`.
- **State:** Postgres on Azure (LangGraph checkpointer + snapshots + audit).
- **Supporting services:** Go (scheduler, webhook listener, channel adapters).
- **Deploy:** Azure Container Apps, Key Vault via managed identity.

Drift from the AaraMinds fixed stack (Go / Spring) is bounded: Python is scoped to the reasoning layer only. Containment boundary is in ADR-0001.

## Jira Cloud integration

| Concern | Decision |
|---------|----------|
| Platform API | REST v3 (`/rest/api/3/`) — issues, comments, users |
| Agile API | `/rest/agile/1.0/` — boards, sprints, backlog, epics |
| Auth | **OAuth 2.0 3LO** (`offline_access`) from day one — decided; no API-token pilot |
| JQL | `POST /rest/api/3/search/jql` + `nextPageToken` (legacy `/search` deprecated 2025-05-01, fully removed ~2025-10-31 → 410; new endpoint returns IDs only — pass `fields`, and use `/search/approximate-count` for totals) |
| Rich text | ADF (Atlassian Document Format) for comments/descriptions |
| Events | Dynamic Webhooks API (3LO, scope `manage:jira-webhook`) + scheduled JQL fallback |
| Rate limits | Points-based, enforced since 2026-03-02 — webhooks over polling + cache + backoff; watch the per-issue write limit on gated comment/label writes |
| Channel (Teams) | Post via Power Automate **Workflows** webhook + Adaptive Card. O365 connector + MessageCard retires 2026-05-18..22 — do not use |

Full scope list and rationale: PRD §8.

## Data model (Postgres, indicative)

`team_config` · `sprint_snapshot` · `issue_snapshot` (+ changelog-derived time-in-status, time-tracking fields) · `recommendation` · `approval` · `action_audit` · `metric_event`.

**Estimation:** time-based — read Jira time-tracking fields (`timeoriginalestimate`, `timeestimate` = remaining, `timespent`; integer seconds); story points are not used. These flat fields are **read-only/computed** — any future write (P1) must go through the `timetracking` composite (`originalEstimate`/`remainingEstimate`), not these fields.
**Reports:** the Sprint Closing / Retro feature emits a `Report.md` with a table of contents (a generated artifact, not a remote write); optional Confluence publish is P2.

The `recommendation → approval → action_audit` chain is the trust backbone — every write traces to a human decision.

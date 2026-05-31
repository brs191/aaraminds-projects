# Scrum Master Agent — Technical Spec / PRD

**Status:** Draft v0.1 · **Date:** 2026-05-31 · **Owner:** Raja · **Workspace:** AaraMinds
**Scope of this doc:** MVP defined in depth, later phases sequenced as a roadmap.

---

## 1. TL;DR

Build a **Jira-connected Scrum Intelligence Layer** — an agent that reads from Jira Cloud (system of record), analyzes sprint state, and delivers recommendations into Slack/Teams. It does **not** silently mutate Jira. Every write passes a gate: **Read → Analyze → Recommend → Human Approval → Write**.

The market (Spinach, Rovo/Jira AI, ScrumGenius, Parabol, LinearB) covers slices — standups, retros, delivery analytics — but no one owns the full Scrum Master loop anchored on Jira with a disciplined human-in-the-loop write model. That gap is the wedge.

**Build vs buy.** *Buy* (Spinach/Rovo) gets standups/retros fast but doesn't own the gated-write loop on our terms and isn't ours to extend; *adopt* (Rovo/Forge) ties us to the Atlassian surface; *build* costs more but the differentiator — the human-approved-write control model on our fixed stack — is exactly the part not available to buy. Decision: **build the differentiator** (this PRD); revisit buying commodity pieces (e.g., retro facilitation) at P2. The **Defining Operational Constraint** is *human-approved writes by construction* — see [design/Agent_Blueprint.md](design/Agent_Blueprint.md).

**MVP = 5 features**, all advisory or gated-write: Daily Scrum Brief, Sprint Health Summary, Blocker & Stale Detection, Story Quality Review, Sprint Closing / Retro Insights.

**Stack decision (made):** orchestration runtime is **LangGraph (Python)** — justified by its durable interrupt/checkpoint primitives mapping directly onto the approval gate. Drift from the AaraMinds fixed stack is contained by scoping LangGraph to the reasoning layer only and keeping the Jira integration as a Go MCP server (§7.1). To be recorded as an ADR.

---

## 2. Problem

Scrum Masters and tech leads spend hours per sprint on mechanical work: chasing status, assembling standup notes, spotting stalled tickets, checking story readiness, and writing retro summaries. The signal already lives in Jira but is scattered across boards, changelogs, and comments. Existing AI tools automate fragments (a standup here, a retro there); none assemble a coherent, Jira-grounded picture with a trustworthy action model.

**Result:** SM time goes to data-gathering instead of facilitation and impediment removal; quality issues (missing acceptance criteria, no estimates, silent blockers) surface late.

---

## 3. Goals / Non-goals

**Goals**
- Cut the SM's manual sprint-tracking time by surfacing standup, health, blocker, quality, and retro signals automatically from Jira.
- Keep Jira authoritative and humans in control — recommendations by default, writes only on approval.
- Ground every recommendation in named issue keys so users can verify, not trust blindly.
- Ship a pilot-ready MVP on one team, one channel, then expand.

**Non-goals (MVP)**
- No autonomous Jira mutations (no status transitions, field edits to descriptions, or deletions).
- No replacement for the human Scrum Master — this is an assistant, not an autopilot.
- No multi-tool sync (Linear/Asana/Azure Boards) — Jira Cloud only.
- No predictive/ML delivery forecasting beyond simple velocity/burndown signals.

---

## 4. Users

| User | Primary jobs the agent helps with |
|------|-----------------------------------|
| Scrum Master / Agile lead | Standup prep, blocker triage, retro synthesis, sprint health |
| Engineering manager / tech lead | Sprint health, spillover risk, delivery visibility |
| Product owner | Story quality, backlog readiness |
| Team members | Daily brief, "what's blocked / waiting on me" |

---

## 5. Product principles

1. **Jira is the system of record.** The agent never holds authoritative state; it caches snapshots for analysis only.
2. **Advisory by default.** Read → Analyze → Recommend → Human Approval → Write. No silent writes, ever, in MVP.
3. **Show your sources.** Every recommendation cites the issue key(s) and the signal (e.g., "DEV-214: 4 days in *In Progress*, no update").
4. **Transparency over magic.** A wrong-but-explained recommendation is recoverable; a confident black box erodes trust and kills adoption.
5. **Minimum viable write surface.** MVP writes are limited to: add comment, add label, create follow-up sub-task, and generate a `Report.md` (with table of contents). Nothing else.

---

## 6. MVP scope — the 5 features

Each feature: what it does, what it reads from Jira, what it outputs, its control level, and acceptance criteria.

### 6.1 Daily Scrum Brief
- **Reads:** active sprint issues (`GET /rest/agile/1.0/sprint/{sprintId}/issue`), status, assignee, blocked flag/label, recent transitions (issue changelog), latest comments.
- **Outputs:** per-team Markdown brief grouped by assignee — done since yesterday, in progress, blocked, waiting-on. Posted to Teams (via Power Automate Workflows webhook + Adaptive Card; O365 connectors retire 2026-05); optional Jira sprint comment on approval.
- **Control:** Advisory (read-only). Optional gated write: post brief as sprint comment.
- **Acceptance:** generates a brief for the active sprint; groups by assignee; explicitly flags blocked and stalled items with issue keys; runs on schedule before standup.

### 6.2 Sprint Health Summary
- **Reads:** sprint scope vs. completed, **time-tracking fields** (`timeoriginalestimate`, `timeestimate` remaining, `timespent`), remaining-work trend from changelog (burndown signal), issues added after sprint start (scope creep), spillover candidates.
- **Outputs:** a scorecard — **On-track / At-risk / Off-track** — with the drivers (e.g., "30% of estimated hours added after start; 5 issues untouched in 3 days").
- **Control:** Advisory.
- **Acceptance:** computes completed vs. remaining **time** (original vs. remaining estimate); detects post-start scope additions; flags spillover risk with rationale, not just a RAG color.

### 6.3 Blocker & Stale Ticket Detection
- **Reads:** issues with `blocked` flag/label; issue links (`blocks` / `is blocked by`); time-in-status from changelog; no-update age; unassigned in-progress items.
- **Outputs:** ranked list of blockers/stale items with age and a suggested next action.
- **Control:** Advisory; gated write: add comment / `needs-attention` label / create follow-up sub-task on approval.
- **Acceptance:** detects dependency-blocked and time-in-status thresholds; surfaces age; no false positives on Done items; configurable thresholds.

### 6.4 Story Quality Review
- **Reads:** description, acceptance-criteria field, **time estimate** (`timeoriginalestimate`), components/labels, against a configurable Definition-of-Ready.
- **Outputs:** per-story flags (missing AC, no estimate, vague description, no owner) + a concrete rewrite suggestion.
- **Control:** Advisory; gated write: post suggestions as a comment. **Never auto-edits the description in MVP.**
- **Acceptance:** flags missing AC / estimate / owner; gives an actionable rewrite; runs on backlog and next-sprint candidates.

### 6.5 Sprint Closing / Retro Insights
- **Reads:** completed sprint issues, spillover, cycle time, reopened issues, scope changes, recurring themes across the last *K* sprints.
- **Outputs:** a retro pack — went well / risks / patterns vs. prior sprints.
- **Control:** Advisory; gated write: generate a `Report.md` (with table of contents) on approval. Optional Confluence publish deferred to P2.
- **Acceptance:** summarizes completion %, spillover, cycle-time trend, and recurring blockers across recent sprints with evidence; output is a `Report.md` with a navigable table of contents.

### Out of scope for MVP
Backlog grooming/prioritization, sprint planning (capacity/velocity scoping), Jira hygiene sweeps, controlled auto-actions, multi-team rollups, and an in-Atlassian (Rovo/Forge) surface — all deferred to §9.

---

## 7. System architecture

```
Slack / Teams / Web UI        ← channel adapters
        │
   Agent orchestration (LangGraph) → LLM (Claude / GPT)
   (reasoning + durable HITL gate via checkpointer)
        │
   Jira Integration Service (Go MCP) ── read + gated write
        │
   Approval queue · Scheduler · Webhook listener
        │
   Postgres (config · snapshots · recommendation/approval audit · metrics)
        │
   Jira Cloud  →  Boards · Sprints · Issues · Backlog · Changelog
```

Components: channel adapters; **LangGraph orchestration/reasoning layer** (owns the durable HITL gate); **Jira integration as a Go MCP server** (read + gated write), consumed by LangGraph as an MCP tool client; approval queue; scheduler (daily brief, sprint-boundary triggers) plus webhook listener; **Postgres** (Azure) for config, cached snapshots, LangGraph checkpoints, and the recommendation→approval→action audit trail.

### 7.1 Stack decision (DECIDED) — LangGraph, drift contained

**Decision:** orchestration runtime is **LangGraph (Python)**. This is a deliberate, bounded exception to the AaraMinds fixed stack (Go / Spring Boot). **Record as an ADR** per workspace governance.

**Why it's justified.** The product's core mechanic is the Read→Recommend→**Approve**→Write gate across five multi-step features. LangGraph's first-class interrupt/checkpoint primitives give durable human-in-the-loop pause/resume out of the box — pause on a recommendation, persist state, resume on approval hours or days later. Rebuilding that durably on native tool-calling is real, error-prone work; LangGraph is the right tool for this specific job.

**How the drift is contained.** LangGraph is scoped to the **reasoning/orchestration layer only**. Everything else stays on the fixed stack:
- **Jira integration → Go MCP server**, reusing the skills-pack pattern. LangGraph consumes it as an MCP tool client (`langchain-mcp-adapters`). The integration layer — the part reused across agents and required to be rock-solid — stays in the house language.
- **State / approval store → Postgres on Azure** (LangGraph checkpointer backed by Postgres, not a separate store).
- **Supporting services** (scheduler, webhook listener, channel adapters) → Go on the fixed stack.
- **Deploy → Azure Container Apps**, Key Vault via managed identity, same as the rest of the stack.

**Open sub-decision (§13.3):** Go MCP server vs. Python-native Jira client for the integration layer. Default is the **Go MCP server** — confirm against the multi-agent roadmap.

---

## 8. Jira Cloud integration spec

| Concern | Decision |
|---------|----------|
| **Platform API** | Jira Cloud Platform REST **v3** (`/rest/api/3/`) — issues, comments, users, JQL |
| **Agile API** | Jira Software Cloud REST (`/rest/agile/1.0/`) — boards, sprints, backlog, epics, sprint issues |
| **Auth** | **OAuth 2.0 3LO** with `offline_access` (refresh tokens) — decided 2026-05-31, from day one; no API-token pilot |
| **Scopes (granular)** | `read:issue:jira`, `read:comment:jira`, `write:comment:jira`, `read:board-scope:jira-software`, `read:sprint:jira-software`, `manage:jira-webhook` (dynamic webhooks), plus `write:issue:jira` for gated labels/sub-tasks |
| **JQL search** | Use `POST /rest/api/3/search/jql` with `nextPageToken` pagination. The legacy offset-based `/rest/api/2/search` and `/rest/api/3/search` were **deprecated 2025-05-01 and fully removed by ~2025-10-31** (now `410 Gone`) — no fallback. Note `/search/jql` returns **issue IDs only** (pass `fields`, e.g. `*navigable`) and **drops `total`** (use `/rest/api/3/search/approximate-count`) |
| **Rich text** | Comments/descriptions in v3 require **ADF** (Atlassian Document Format), not plain strings |
| **Change events** | 3LO apps register webhooks via the **Dynamic Webhooks API** (`jira:issue_updated`, `sprint_started/closed/updated`, `comment_created`). Registrations expire (~30 days) → refresh job; fall back to scheduled JQL polling |
| **Time-tracking** | Read flat fields `timeoriginalestimate` / `timeestimate` (remaining) / `timespent` (integer seconds, read-only). Any future **write** goes through the `timetracking` composite, not these fields |
| **Teams channel** | Post via Power Automate **Workflows** webhook + **Adaptive Card**. The legacy O365 Incoming Webhook connector + MessageCard retires **2026-05-18..22** — do not build on it |
| **Rate limits** | **Points-based** model (cost per call scales with data/complexity), with tiered quotas **enforced since 2026-03-02** for all Forge/Connect/OAuth 3LO apps. Honor `429` + `Retry-After`, exponential backoff, prefer webhooks over polling, and cache snapshots |

**Design rule:** wrap all Jira calls behind the MCP integration layer so API churn (e.g., the completed `/search` → `/search/jql` migration) is absorbed in one place.

---

## 9. Data model (Postgres, indicative)

`team_config` (board, sprint cadence, DoR, thresholds, channel) · `sprint_snapshot` · `issue_snapshot` (+ changelog-derived time-in-status, time-tracking fields) · `recommendation` · `approval` (who/when/decision) · `action_audit` (what was written back) · `metric_event`.

The `recommendation → approval → action_audit` chain is the trust backbone: every write is traceable to a human decision.

---

## 10. Success metrics

No baselines are invented here — these are **definitions and targets to baseline during the pilot.**

| Metric | Definition | Target |
|--------|------------|--------|
| Adoption | % of active sprints with a brief generated; weekly active teams | Trend up `[VERIFY in pilot]` |
| Time saved | SM minutes saved on standup prep + retro assembly | Baseline then reduce `[VERIFY]` |
| Story readiness | % of stories meeting DoR before sprint start | Trend up `[VERIFY]` |
| Flow | Median time-in-blocked-status | Trend down `[VERIFY]` |
| Trust | Recommendation acceptance rate; blocker/stale false-positive rate | Acceptance up, FP down `[VERIFY]` |

---

## 11. Risks & mitigations

- **Jira API changes** — the old `/search` endpoints are gone (deprecated 2025-05-01, removed ~2025-10-31) and points-based rate limits are live (2026-03-02). Isolate all Jira calls behind the MCP layer; watch the Atlassian changelog.
- **Teams connector EOL** — O365 Incoming Webhook connectors retire 2026-05-18..22; the channel adapter targets a P
# Agent Blueprint — Scrum Master Agent

**Owner:** Raja · **Stage:** design baseline · **Date:** 2026-05-31
**Produced via:** AaraMinds AI Agent Blueprint Advisor persona (Module 8 process). This is the design baseline a future Module 5 systems review reads against.

> Boundary is set **before** tools/memory/workflow (Boundary Gate). The Defining Operational Constraint governs every downstream decision.

## 1. Job to be done

Give a Scrum Master / team a trustworthy, Jira-grounded read of sprint state — standup brief, sprint health, blockers, story quality, retro — without the SM hand-assembling it, and let the agent act in Jira/Teams **only** through human-approved writes.

## 2. Agent justification (why an agent, not a script)

A cron job + templated query could produce a status dump. The agent earns its place on three counts: (a) it **reasons over** heterogeneous signals (changelog-derived time-in-status, dependency links, DoR gaps) rather than printing fields; (b) it operates a **durable multi-step loop with a human approval interrupt** that may pause for hours; (c) it composes natural-language output (brief, retro narrative) over structured data. If the requirement were "post the sprint's issue list daily," that is a script — and we would say so. It is not: the value is the judgment between fetch and post.

## 3. Boundary (Boundary Gate — set first)

| Class | Items |
|-------|-------|
| **In scope** (agent does, autonomously up to the gate) | Read Jira (sprint, issues, changelog); analyze (health, blockers, stale, DoR, retro patterns); draft brief/recommendations/`Report.md`; **propose** writes; post to Teams and write to Jira **after approval**; record the audit chain |
| **Out of scope** (not built in MVP) | Autonomous status transitions; description/field edits; issue deletion; multi-tool sync (Linear/Asana/Azure Boards); predictive/ML forecasting; sprint scope changes |
| **Human-only** (never delegated) | The **approval decision** on every write; sprint planning commitments; people/performance judgments |

## 4. Decomposition — single agent (Single-Agent Default upheld)

One agent, one reasoning loop, sequential tool calls. Multi-agent is **not** justified: there is no concurrency of independent goals, no specialist division that a single loop can't sequence, and adding sub-agents would multiply the trust surface against the DOC. Features (Daily Brief, Health, Blocker, Story Quality, Retro) are **modes** of one agent sharing the read→analyze→recommend→gate→write spine, not separate agents.

## 5. Foundation & stack

Decided in [ADR-0001](adr/0001-langgraph-orchestration.md): LangGraph (Python) reasoning layer; Go MCP server for Jira (consumed via `langchain-mcp-adapters`); Go Teams adapter; Postgres (checkpointer + audit) on Azure; deploy Azure Container Apps, Key Vault via managed identity.

## 6. Defining Operational Constraint (DOC)

**Human-Approved Writes by Construction.** No mutation of Jira or any channel occurs without a corresponding, persisted human approval. The architecture enforces this structurally — not by policy or prompt — via the LangGraph `interrupt()` gate plus the `recommendation → approval → action_audit` chain in Postgres. Every other design choice serves or is subordinate to this invariant. If a change would let a write occur without a traceable approval, it is rejected regardless of its other merits.

## 7. Control plane

- **Approval gate:** durable `interrupt()`; run pauses and persists to the Postgres checkpointer; resumes only on `Command(resume=...)`. Fail-closed: a malformed/empty resume payload is treated as **reject**.
- **Audit:** `recommendation → approval → action_audit`; every write traces to one approval row. Delivery failures record `action_audit.result = failed` (never left half-written).
- **Source citation:** every recommendation names issue key(s) + the triggering signal — falsifiable by the reader.
- **Least privilege:** OAuth 3LO granular scopes; write scope (`write:comment:jira`, gated `write:issue:jira`) requested but exercised only post-approval.
- **Secrets:** Key Vault via managed identity; no creds in code/env files.

## 8. Trust boundaries & failure modes (FMEA-lite)

| Boundary crossing | Failure mode | Mitigation |
|---|---|---|
| Jira → agent | Stale/partial read; API 410/429 | `/search/jql` only; honor `Retry-After`; cache snapshots; brief states data freshness |
| Agent → LLM | Hallucinated issue key / wrong status | Cite keys; analysis computed in code (`brief.py`), not the LLM; reader can verify |
| Agent → human (gate) | Approver rubber-stamps | Brief is concise, source-linked; approval is per-recommendation, not bulk |
| Agent → Jira/Teams (write) | Partial write; channel EOL | Audit records `failed`; idempotent on resume (checkpointer replays completed nodes once); Teams via Workflows webhook (O365 connectors retire May 2026) |
| LLM ← issue content | Sensitive data egress to model | Scope minimization; redact sensitive fields; Azure region pinning; tenant isolation |

## 9. Evaluation

Inherits `../evaluation/Acceptance_Criteria.md` (per-feature), `../evaluation/Eval_Rubric.md` (accuracy/trust/safety/usefulness), and `../evaluation/Test_Strategy.md` (test pyramid + cases). DOC-specific scorer: **zero writes without an approval row** — asserted in integration tests, not assumed.

## 10. Lifecycle coherence

- **First review trigger:** P0 → P1 gate (pre-pilot). A Module 5 systems review reads this blueprint as baseline.
- **Review produces:** findings against accuracy/trust/safety; close or accept-residual before pilot sign-off.
- **Redesign triggers:** any silent-write incident (DOC breach — highest severity); Jira API breaking change; false-positive rate breaching the rubric; expansion to a new tenant/regulated data class; adding a second channel (Slack, P2).

## 11. Workflow sequence (Daily Brief — full approval routing)

Diagram Completion Check: shows request, outcome, post-approval handoff, **rejection path**, and audit recording.

```mermaid
sequenceDiagram
    participant Sch as Scheduler
    participant Ag as Agent (LangGraph)
    participant J as jira-mcp (Go)
    participant DB as Postgres
    participant H as Human (Teams)
    participant T as teams-adapter (Go)

    Sch->>Ag: trigger daily brief
    Ag->>J: get_active_sprint / get_sprint_issues
    J-->>Ag: sprint + issues (stub fixtures in P0)
    Ag->>Ag: analyze + build brief (cite keys)
    Ag->>DB: record_recommendation → rec_id
    Ag-->>H: interrupt() — approval request (preview)
    Note over Ag,DB: run pauses; state persisted to checkpointer

    alt Approved
        H-->>Ag: Command(resume={approved:true})
        Ag->>DB: record_approval(approved)
        Ag->>T: post brief
        T-->>Ag: delivered | logged
        Ag->>DB: record_action(result)
    else Rejected / change requested
        H-->>Ag: Command(resume={approved:false})
        Ag->>DB: record_approval(rejected)
        Ag->>DB: record_action(skipped)
        Note over Ag: no write to Jira/Teams — DOC upheld
    end
```

## 12. Systems-review acceptance

This blueprint is a usable Module 5 baseline: boundary, DOC, control plane, trust boundaries, failure modes, evaluation, and lifecycle triggers are all explicit. Open handoff: a full Module 5 production-readiness review before pilot (P1 gate).

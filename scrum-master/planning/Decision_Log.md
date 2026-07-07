# Scrum Master Agent — Decision Log

Gated decision record. Ports the decisions locked in `../tracking/Status.md` and PRD §13 into stable DEC-### IDs so requirements, contracts, and release notes can cite them. New decisions append here; reversing one requires a new entry, not an edit.

| Field | Value |
| --- | --- |
| Version | 0.1 · Prepared 2026-07-03 · Accountable owner: Raja |
| Statuses | **Closed** — accepted for the current baseline · **Conditional** — proceed under the stated constraint · **Deferred** — decide before the named gate |

## Decisions

| ID | Decision | Outcome | Status | Date |
| --- | --- | --- | --- | --- |
| DEC-001 | Orchestration runtime | LangGraph (Python) — durable `interrupt()`/checkpoint maps directly onto the approval gate. Bounded fixed-stack exception; containment per ADR-0001 (reasoning layer only; integration stays Go). | Closed | 2026-05-31 |
| DEC-002 | Integration layer | Go MCP server (`jira-mcp`), consumed via `langchain-mcp-adapters`. | Closed | 2026-05-31 |
| DEC-003 | Auth / tenancy | OAuth 2.0 3LO with `offline_access` and granular scopes from day one; no API-token pilot. | Closed | 2026-05-31 |
| DEC-004 | Channel | Microsoft Teams first, via Power Automate Workflows webhook + Adaptive Card (O365 connectors retired 2026-05). Slack at P2 parity. | Closed | 2026-05-31 |
| DEC-005 | Estimation basis | Time-based (Jira time-tracking fields, read as flat fields; any future write via the `timetracking` composite). Not story points. | Closed | 2026-05-31 |
| DEC-006 | Retro output | Generated `Report.md` with TOC; Confluence publish optional at P2. | Closed | 2026-05-31 |
| DEC-007 | Control model (DOC) | Human-approved writes by construction: `interrupt()` gate + `recommendation → approval → action_audit` chain; fail-closed on malformed resume. Enforced in code and asserted by `test_gate.py` / `test_doc_invariant.py`. | Closed | 2026-06-03 |
| DEC-008 | MVP write surface | Exactly: add comment, add label, create sub-task, local `Report.md`. Expansion requires a new DEC + GTS-GATE cases (SM-QG-002). | Closed | 2026-05-31 |
| DEC-009 | Governance pack adoption | Adopt the AaraMinds gated-artifact set for this project: requirements baseline (SM-* IDs), MCP tool contracts + validation register, evaluation harness with hard gates (SM-EM-001/002 = 0), operations model, this decision log. Evidence markers repo-wide: `[inferred]` / `[VERIFY]` only. | Closed | 2026-07-03 |
| DEC-010 | Dual-layer write enforcement | Write tools validate `approval_ref` presence at the jira-mcp layer in addition to the orchestrator gate; single-use validation stays orchestrator-side for MVP, revisit moving it into the server at P1. | Conditional — revisit at P1 gate | 2026-07-03 |
| DEC-011 | Advisory-post gating | Whether advisory Teams posts (no system-of-record write) also require the approval gate, or only Jira writes do. Current code gates the post; PRD reads advisory-by-default. | Deferred — decide before P1 feature expansion | — |
| DEC-012 | Pilot boundary | Which board, team, and Teams channel the pilot uses; data classification of that board's content before live LLM calls. | Deferred — blocks P0 gate completion (live ingest) | — |

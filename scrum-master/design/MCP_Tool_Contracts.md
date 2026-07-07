# Scrum Master Agent — MCP Tool Contracts

Contracts for the `jira-mcp` server (Go) and the `teams-adapter`, in the AaraMinds contract format. Status: **P0 tools are implemented (stub fixtures); every other tool is proposed and must be validated against real Jira Cloud behavior before it leaves stub mode.** Requirement IDs reference `../requirements/Scrum_Master_Agent_Requirements.md`.

## Document control

| Field | Value |
| --- | --- |
| Version | 0.1 |
| Prepared date | 2026-07-03 |
| Accountable owner | Raja |
| Implementation | `../code/apps/jira-mcp/` (Go, mcp-go), `../code/apps/teams-adapter/` (Go) |
| Governing requirements | SM-MVP-FR-009, SM-NFR-002/003/006/007, SM-HIL-001/002, SM-QG-001/002 |

## Contract conventions

- **Enforcement point:** the DOC is enforced in the orchestrator's gate (`gate.py`: no write executes without an approval row) and must ALSO be enforced in `jira-mcp` itself — write tools reject calls without a valid `approval_ref` at the tool layer, so a bug or prompt injection in the orchestrator cannot produce a silent write. Two independent layers, one invariant (SM-QG-001).
- **Allowlist:** the server registers exactly the tools below. Adding a tool is a change-controlled event (decision-log entry + GTS-GATE case).
- **Failure statuses:** `degraded` (partial data, names unavailable sources), `denied` (authz), `throttled` (rate limit; honor `Retry-After`). Never fabricate or silently omit (SM-MVP-FR-011).
- **Timestamps:** responses carry `source_timestamp` (data currency) distinct from `retrieved_at`, so briefs can state freshness.
- **Audit:** every write call emits `{ tool_name, approval_ref, idempotency_key, input_hash, result, timestamp }`; the orchestrator links it to the `recommendation → approval → action_audit` chain.
- **Idempotency:** reads are idempotent; writes require an `idempotency_key` and reject duplicates (safe checkpointer resume, SM-NFR-008).

## Read tools (jira-mcp)

| Tool | Signature → returns | Status | Serves |
| --- | --- | --- | --- |
| `get_active_sprint` | `(board_id)` → `{ sprint_id, name, start, end, goal, source_timestamp }` | **Implemented (stub)** | SM-MVP-FR-001/003 |
| `get_sprint_issues` | `(sprint_id)` → `{ issues[{ key, summary, status, assignee, blocked_flag, labels, time_tracking{original, remaining, spent}, links[] }], source_timestamp }` | **Implemented (stub)** | SM-MVP-FR-001/003/004 |
| `get_issue_changelog` | `(issue_key)` → `{ transitions[{ from, to, at, by }], source_timestamp }` — feeds time-in-status and stalled detection | Proposed | SM-MVP-FR-003/004 |
| `get_backlog_candidates` | `(board_id, limit?)` → next-sprint candidate issues with description, AC field, estimate, owner | Proposed | SM-MVP-FR-005 |
| `get_closed_sprints` | `(board_id, k)` → last K sprints with completion, spillover, cycle-time inputs | Proposed | SM-MVP-FR-006 |

Read constraints: JQL only via `POST /rest/api/3/search/jql` + `nextPageToken`; pass `fields` explicitly (endpoint returns IDs only); use `/search/approximate-count` for totals; time-tracking flat fields are read-only (SM-INT-001, PRD §8).

## Write tools (jira-mcp) — the complete MVP write surface

All approval-gated (SM-HIL-001). `approval_ref` = the `approval` row ID; the server validates it is present; existence/single-use validation lives in the orchestrator's audit layer for MVP `[VERIFY: consider moving single-use validation into the server at P1 — it is the stronger enforcement point]`.

| Tool | Signature | Notes |
| --- | --- | --- |
| `add_comment` | `(issue_key, body_adf, approval_ref, idempotency_key)` | Body in ADF, not plain text. Serves brief-as-comment and quality-review comments. |
| `add_label` | `(issue_key, label, approval_ref, idempotency_key)` | e.g. `needs-attention`. |
| `create_subtask` | `(parent_key, summary, description_adf, approval_ref, idempotency_key)` | Follow-up actions from blocker detection. |

Prohibited by construction (not registered): status transitions, description/field edits, deletions, sprint mutations (SM-AUT-004). `Report.md` generation is a local artifact in the orchestrator, not a jira-mcp tool.

## teams-adapter contract

| Concern | Contract |
| --- | --- |
| `post_card(payload)` | Adaptive Card JSON to the Power Automate Workflows webhook URL (Key Vault). Non-2xx → error surfaced to the orchestrator; after an approved write, a delivery failure records `action_audit.result = failed` (SM-NFR-007). |
| Stub mode | No webhook configured → `not delivered`, logged — never a fake success (existing behavior, keep). |
| Card content | Recommendation preview + approval actions; every card footer carries the `recommendation` ID so approvals bind to exactly one recommendation (per-recommendation approval, SM-HIL-001). |

## Validation register

Per-tool sign-off before a tool leaves stub/fixture mode. Empty rows mean not started.

| Tool | Real-schema validated | Scope granted (OAuth 3LO) | Rate-limit behavior tested | Signed off |
| --- | --- | --- | --- | --- |
| get_active_sprint | — | `read:sprint:jira-software` | — | — |
| get_sprint_issues | — | `read:issue:jira` | — | — |
| get_issue_changelog | — | `read:issue:jira` | — | — |
| get_backlog_candidates | — | `read:board-scope:jira-software` | — | — |
| get_closed_sprints | — | `read:sprint:jira-software` | — | — |
| add_comment | — | `write:comment:jira` | — | — |
| add_label | — | `write:issue:jira` | — | — |
| create_subtask | — | `write:issue:jira` | — | — |
| post_card (Teams) | — | n/a (webhook) | — | — |

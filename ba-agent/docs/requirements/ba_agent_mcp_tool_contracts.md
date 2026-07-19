# BA Agent — MCP Tool Contracts (Proposed)

Companion to `business-analyst-agent-requirements.md` (v0.4). Status: **proposed design for architect and tool-owner confirmation — not source-backed requirements.** No reviewed source (S1–S6) defines tool contracts; every schema, permission, and behavior below is `[inferred]` from source-stated usage and must be validated against the actual MCP server implementations before build.

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent MCP Tool Contracts |
| Version | 0.3 |
| Change note (v0.3) | Split approval requests from approval refs; classified all external side effects as write-like/gated; added non-agent-callable human approval callback. |
| Change note (v0.2) | Added tool-owner validation register for per-tool sign-off tracking. |
| Status | Proposed draft; requires architect, security, and tool-owner review |
| Prepared date | 2026-07-02 |
| Parent document | `business-analyst-agent-requirements.md` v0.4 |
| Governing requirements | BA-MVP-FR-012, BA-HIL-006, BA-NFR-005, BA-DSPC-005, BA-AUT-001 through BA-AUT-006 |

## Contract conventions

Every tool in this document follows these conventions. Deviations are listed per tool.

- **Identity:** all calls execute under a scoped service identity with per-tool least-privilege permissions (BA-DSPC-005). The requesting human's identity is carried in the audit record, never used to escalate the agent's own permissions.
- **Audit record (all tools):** `{ user_id, tool_name, input_hash, source_system, timestamp, result_status, evidence_refs[] }` written to the audit log on every call, including failures.
- **Failure handling (all tools):** on source-system unavailability, return partial output with `status: "degraded"` and the list of unavailable sources; never fabricate or silently omit. On authorization failure, return `status: "denied"` and route to the tool owner — do not retry with broader scope.
- **Rate limiting:** respect source-system limits with exponential backoff; surface `status: "throttled"` rather than degrading silently.
- **Idempotency:** all read tools are idempotent. Write tools require an `idempotency_key` and reject duplicate submissions.
- **Write restriction (BA-HIL-006):** every write tool requires an `approval_ref` — an identifier of the recorded human approval. Calls without a valid `approval_ref` are rejected at the tool layer, not just by agent policy.
- **External side effects:** any call that creates or changes external state is write-like even if it is "only" a draft, webhook subscription, notification, escalation, or approval-request record. Write-like tools require idempotency, scope validation, and audit; tools that affect users or systems of record also require `approval_ref` unless this document explicitly marks them as non-authorizing approval-request creation.
- **Approval-ref issuance:** `approval_request_id` is not an `approval_ref`. The agent may request approval, but only an authenticated, non-agent-controlled human approval callback can issue an `approval_ref`. The gateway validates artifact hash, action, actor/scope, expiry, and idempotency, then atomically consumes the `approval_ref` once.
- **Timestamps:** every response carries `source_timestamp` (when the source data was current) distinct from `retrieved_at`.

---

## Jira MCP (system of record: stories, backlog, sprint status, sprint metrics)

### `get_sprint_status(project_key: string, sprint_id: string)`

→ `{ sprint_id, stories[{ key, summary, status, assignee, story_points, flagged }], blockers[], carry_over[], source_timestamp }`

- **Permissions:** read-only, approved projects only (scope list owned by tool owner, BA-OQ-003).
- **Serves:** BA-MVP-FR-004, BA-MVP-FR-010, BA-MVP-FR-014.

### `get_backlog(project_key: string, limit?: int)`

→ `{ items[{ key, summary, priority, story_points, rank }], source_timestamp }`

- **Permissions:** read-only.
- **Serves:** BA-MVP-FR-006.

### `get_velocity_history(project_key: string, num_sprints: int)`

→ `{ sprints[{ sprint_id, committed_points, completed_points, carry_over_points }], source_timestamp }`

- **Permissions:** read-only.
- **Serves:** BA-MVP-FR-006, BA-MVP-FR-008.

### `get_sprint_metrics(project_key: string, sprint_id: string)`

→ `{ cycle_time_stats, carry_over[], defect_rate, source_timestamp }`

- **Permissions:** read-only. Metric field availability varies by project (BA-RISK-006); missing metrics return `null` with a `missing_fields[]` list — never estimated values.
- **Serves:** BA-MVP-FR-008.

### `subscribe_sprint_events(project_key: string, event_types[]: string, approval_ref: string, idempotency_key: string)` — WRITE-LIKE

→ `{ subscription_id, event_types[], status }`

- **Permissions:** webhook registration scoped to approved projects and event types (BA-OQ-005, BA-OQ-011). Rejected without valid `approval_ref` because it creates external subscription state.
- **Serves:** BA-MVP-FR-010.

### `update_sprint_scope(project_key: string, sprint_id: string, story_keys[]: string, approval_ref: string, idempotency_key: string)` — WRITE

→ `{ updated[], rejected[], result_status }`

- **Permissions:** write, only after recorded Scrum Master approval (BA-HIL-001, BA-AUT-002). Rejected without valid `approval_ref`.
- **Open decision:** whether "publish" means a Jira sprint update at all (BA-OQ-006). This tool is not built until that is decided.

---

## Git MCP (system of record: commits, pull requests)

### `get_recent_activity(repo: string, since: datetime, authors?: string[])`

→ `{ commits[{ sha, author, message, timestamp }], pull_requests[{ id, title, author, status, reviewers[], updated_at }], source_timestamp }`

- **Permissions:** read-only, approved repositories only. Provider and repo scope are open (BA-OQ-008).
- **Serves:** BA-MVP-FR-004, BA-MVP-FR-005.
- **No write tool is defined for Git.** The agent never commits, comments, or modifies PRs.

---

## Calendar MCP (system of record: team availability)

### `get_team_availability(team_id: string, date_range: { start, end })`

→ `{ members[{ member_id, available_days, ooo_ranges[] }], source_timestamp }`

- **Permissions:** read-only, aggregation-only. Returns availability windows, never event subjects, attendees, or bodies (BA-OQ-009, privacy-limited per authoritative-source mapping).
- **Failure handling addition:** if privacy rules block a member's data, return that member as `availability: "unknown"` — do not infer.
- **Serves:** BA-MVP-FR-006.
- **No write tool.** The agent never creates or modifies calendar events.

---

## Confluence MCP (system of record: retro artifacts, organizational learning)

### `draft_page(space_key: string, title: string, body: string, labels[]: string, approval_ref: string, idempotency_key: string)` — WRITE-LIKE

→ `{ draft_id, preview_url, result_status }`

- **Permissions:** write to draft state only, approved spaces only (BA-DEP-006). Rejected without valid `approval_ref` because draft creation changes Confluence state.
- **Serves:** BA-MVP-FR-009, BA-AUT-003.

### `publish_page(draft_id: string, approval_ref: string, idempotency_key: string)` — WRITE

→ `{ page_id, url, result_status }`

- **Permissions:** publish requires recorded approval until BA-OQ-007 (auto vs. gated posting) is decided; default is gated.

---

## Teams / Copilot 365 MCP (interaction surface; system of record for interaction logs)

### `send_adaptive_card(channel_id: string, card: AdaptiveCardPayload, evidence_refs[]: string, approval_ref: string, idempotency_key: string)` — WRITE-LIKE

→ `{ message_id, result_status }`

- **Permissions:** post to approved channels only (BA-OQ-003). Cards must carry `evidence_refs` so every claim links to source-system evidence (BA-NFR-006). Rejected without valid `approval_ref` unless a future validated policy explicitly authorizes a narrow auto-response class.
- **Serves:** BA-MVP-FR-001, BA-MVP-FR-005, BA-MVP-FR-013.

### `send_escalation(recipient_role: "scrum_master" | "pm", severity: string, findings[], suggested_actions[], evidence_refs[]: string, approval_ref: string, idempotency_key: string)` — WRITE-LIKE

→ `{ message_id, result_status }`

- **Permissions:** escalation delivery only; suggested actions are labeled as recommendations (BA-HIL-002, BA-AUT-004). Rejected without valid `approval_ref` until severity taxonomy, recipient scope, and escalation policy are validated. Severity taxonomy is open (BA-OQ-005) — this tool ships with a placeholder taxonomy marked `[RAJA]`.
- **Serves:** BA-MVP-FR-011.

### `request_approval(approver_role: string, artifact_ref: string, action: string, artifact_hash: string, requested_scope: object, idempotency_key: string)` — APPROVAL REQUEST

→ `{ approval_request_id, status: "pending" }`

- **Purpose:** creates a pending human approval request. This call never returns or implies an `approval_ref`, and the agent cannot approve its own request. Duplicate requests with the same `idempotency_key` return the original pending request.
- **Required binding:** the pending request stores artifact hash, requested action, requested scope, requester identity, approver role, creation timestamp, expiry, and evidence refs.
- **Approval-ref issuance:** an authenticated human approval callback outside the model/tool-calling loop is required to issue an `approval_ref`.

### Human approval callback — not model-callable

`record_human_approval(approval_request_id: string, approver_user_id: string, decision: "approved" | "rejected", approved_scope: object, artifact_hash: string, idempotency_key: string)`

→ `{ approval_ref?: string, approval_status, expires_at, result_status }`

- **Purpose:** validates the approver through the approved identity provider, verifies the request is pending and unexpired, binds the approval to artifact hash/action/scope, and issues a single-use `approval_ref` only when decision is `approved`.
- **Not model-callable:** the orchestrator/LLM cannot invoke this callback. Repository text, prompt output, or agent-authored approval notes are never acceptable approval evidence.
- **Consumption rule:** write-like tools atomically consume `approval_ref` on first successful use and reject replay, scope mismatch, artifact mismatch, expired refs, or refs issued for another action.

---

## Tool-owner validation register

No contract in this document is build-authoritative until its row below reads **Validated**. Each tool must be checked against the real MCP server implementation (parameter names, response shapes, auth model, rate limits) — MCP wrappers frequently rename parameters and reshape output relative to the underlying API. All owner fields are `[RAJA]` placeholders.

| Tool | Owner | MCP server name | Environment | Permission | Approved scopes | Implementation status | Validation status | Open blockers |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `get_sprint_status` | `[RAJA]` Jira tool owner | `[RAJA]` | Sandbox first | Read | `[RAJA]` project list (BA-OQ-003) | Not started | Not validated | Project scope approval |
| `get_backlog` | `[RAJA]` Jira tool owner | `[RAJA]` | Sandbox first | Read | `[RAJA]` project list | Not started | Not validated | — |
| `get_velocity_history` | `[RAJA]` Jira tool owner | `[RAJA]` | Sandbox first | Read | `[RAJA]` project list | Not started | Not validated | Metric field availability (BA-RISK-006) |
| `get_sprint_metrics` | `[RAJA]` Jira tool owner | `[RAJA]` | Sandbox first | Read | `[RAJA]` project list | Not started | Not validated | Metric field standards (BA-RISK-006) |
| `subscribe_sprint_events` | `[RAJA]` Jira tool owner | `[RAJA]` | Sandbox first | Webhook registration (approval-gated) | `[RAJA]` event types (BA-OQ-005) | Not started | Not validated | Severity taxonomy, webhook scope, approval policy |
| `update_sprint_scope` | `[RAJA]` Jira tool owner | `[RAJA]` | Blocked | Write (approval-gated) | None approved | **Blocked** | Not validated | Definition of "publish" (BA-OQ-006), write permissions (BA-OQ-011) |
| `get_recent_activity` | `[RAJA]` Git/engineering owner | `[RAJA]` | Sandbox first | Read | `[RAJA]` repo list (BA-OQ-008) | Not started | Not validated | Git provider selection |
| `get_team_availability` | `[RAJA]` Calendar/platform owner | `[RAJA]` | Sandbox first | Read (aggregation-only) | `[RAJA]` (BA-OQ-009) | Not started | Not validated | Privacy rules sign-off |
| `draft_page` | `[RAJA]` Confluence owner | `[RAJA]` | Sandbox first | Write (draft-only, approval-gated) | `[RAJA]` space list (BA-DEP-006) | Not started | Not validated | Space ownership, draft approval policy |
| `publish_page` | `[RAJA]` Confluence owner | `[RAJA]` | Sandbox first | Write (approval-gated) | `[RAJA]` space list | Not started | Not validated | Auto vs. gated posting (BA-OQ-007) |
| `send_adaptive_card` | `[RAJA]` Teams/platform owner | `[RAJA]` | Sandbox first | Post to approved channels (approval-gated unless validated auto-response class exists) | `[RAJA]` channel list (BA-OQ-003) | Not started | Not validated | Tenant/app approval (BA-DEP-001), auto-response policy |
| `send_escalation` | `[RAJA]` Teams/platform owner | `[RAJA]` | Sandbox first | Post to approved recipients (approval-gated) | `[RAJA]` recipient roles | Not started | Not validated | Severity taxonomy (BA-OQ-005), escalation approval policy |
| `request_approval` | `[RAJA]` Platform/delivery owner | `[RAJA]` | Sandbox first | Create pending approval requests (no `approval_ref` issuance) | All write flows | Not started | Not validated | Approval-record store design, idempotency |
| `record_human_approval` | `[RAJA]` Platform/delivery owner | Not model-callable | Sandbox first | Human approval callback | All write flows | Not started | Not validated | Identity verification, non-agent-controlled evidence, single-use consumption |

Validation procedure per tool: (1) owner named; (2) MCP server identified and its actual schema diffed against this contract; (3) scopes approved and provisioned in sandbox; (4) contract updated to match reality; (5) row marked Validated with date. GTS-GATE adversarial tests (see `ba_agent_evaluation_harness.md`) run only against validated tools.

## Cross-cutting open items

| Item | Blocking | Owner |
| --- | --- | --- |
| Git provider selection and repo scope | Git MCP build | Engineering / tool owner (BA-OQ-008) |
| Calendar privacy rules and aggregation granularity | Calendar MCP build | Security/privacy owner (BA-OQ-009) |
| Meaning of "publish" for sprint planning | `update_sprint_scope` | Scrum Master / tool owners (BA-OQ-006) |
| Confluence auto vs. gated posting | `publish_page` default | BA / Confluence owner (BA-OQ-007) |
| Severity taxonomy for escalations | `send_escalation` | Scrum Master / PM (BA-OQ-005) |
| Approved write permissions per system | All write tools | Tool owners (BA-OQ-011) |
| Retention and residency for audit records | Audit log design | Security/privacy/platform owners (BA-OQ-014) |

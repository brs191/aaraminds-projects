# BA Agent Phase 2 HLD Creation Plan

This plan records the scope-change decision to make HLD creation the next Phase 2 focus item. It is a planning and execution-control artifact only; it does not authorize sandbox execution, live integrations, non-synthetic data use, production deployment, autonomous approval, system-of-record updates, external publishing, or write-like side effects.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 HLD Creation Plan |
| Version | 0.4 |
| Change note (v0.4) | Recorded RAJA approval of the draft HLD as the current baseline; HLD lane closed. |
| Change note (v0.3) | Recorded HLD owner-review package completion through `HLD-G3`; RAJA owner decision remains pending. |
| Change note (v0.2) | Recorded draft HLD completion through `HLD-G2`; owner-review package remains next. |
| Status | HLD lane complete; draft HLD approved as current baseline |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Triggering decision | RAJA directive on 2026-07-13: "go ahead" after asking to move directly to HLD creation as the focus item |
| Execution lane | `[F9]` HLD creation |
| Prompt baseline | `prompts.md` v1.2 |
| Decision baseline | `docs/planning/decision-log.md` v2.9 |
| Prior Phase 2 baseline | `docs/planning/phase-2-implementation-plan.md` v0.4 |
| Explicit non-authorization | No sandbox execution, live integration, pilot start, production rollout, non-synthetic data path, external tool execution, external publish/storage, credential use, authoritative architecture approval, or write-like side effect |

## 1) Scope-change verdict

HLD generation was explicitly out of scope for the completed Phase 2 first-slice requirement-discovery lane. RAJA has now changed the focus to HLD creation. The new lane may create a **draft/advisory BA Agent HLD** from checked-in repository evidence only.

This scope change does not reopen sandbox, live, pilot, production, Teams, Jira, Confluence, Git/GitHub, MCP, Graph API, or non-synthetic data paths. The HLD must be evidence-grounded, mark unsupported design choices as `[inferred]`, and mark owner-dependent choices as `[RAJA]`.

## 2) Allowed inputs

| Input | Status | Notes |
| --- | --- | --- |
| Checked-in requirements docs | Allowed | Use `docs/requirements/*` as product and architecture evidence. |
| Checked-in planning/development docs | Allowed | Use plans, decision logs, traceability, and synthetic completion evidence. |
| Checked-in source/tests | Allowed | Use current local implementation as implementation evidence where relevant. |
| Synthetic fixtures/eval outputs | Allowed | May support behavior/control descriptions but remain synthetic. |
| Sandbox/live tool data | Blocked | No row is validated or executable. |
| Real tickets/pages/repos/messages | Blocked | Non-synthetic data remains unauthorized. |
| External publishing/storage | Blocked | HLD stays local in repository docs unless separately approved. |

## 3) HLD deliverable boundary

The HLD may cover:

1. Purpose, scope, and non-goals.
2. Requirement and decision traceability.
3. Logical architecture and component responsibilities.
4. Runtime flow for synthetic/local execution.
5. Gateway/control-plane and approval-gate model.
6. Data classification and evidence discipline.
7. Evaluation/hard-gate strategy.
8. Observability/audit posture.
9. Deployment direction as proposed architecture only, with `[RAJA]` for owner decisions.
10. Risks, assumptions, dependencies, and open decisions.

The HLD must not:

1. Claim production readiness.
2. Claim live/sandbox integration approval.
3. Embed secrets, credentials, tenant IDs, private endpoints, project keys, repo names, channel IDs, or restricted data.
4. Convert draft/advisory content into approved architecture.
5. Introduce Slack or Azure ACR.
6. Approve writes, sends, publishes, comments, approval records, or subscriptions.

## 4) HLD execution gates

| Gate | Objective | Exit criteria |
| --- | --- | --- |
| `HLD-G0` | Scope-change setup | Decision log, prompt pack, fleet guide, and this plan identify HLD as active lane while preserving non-authorization boundaries. |
| `HLD-G1` | HLD draft | `docs/architecture/ba-agent-hld.md` exists with evidence-grounded sections, `[inferred]`/`[RAJA]` discipline, and no live/sandbox overclaim. |
| `HLD-G2` | HLD QA/review | QA verifies source traceability, architecture consistency, hard-gate/control wording, and no forbidden drift. |
| `HLD-G3` | Owner review package | Open decisions and review asks are packaged for RAJA; no approval is self-created by the agent. |

## 5) Prompt execution plan

| Prompt | Purpose | Deliverable |
| --- | --- | --- |
| [P9A]/[Q9A] | Create HLD scope-change plan and prompt/fleet updates | This plan, decision log update, prompt update, fleet update |
| [P9B]/[Q9B] | Draft BA Agent HLD | `docs/architecture/ba-agent-hld.md` |
| [P9C]/[Q9C] | Create HLD review package | `docs/development/phase-2-hld-review-package.md` |

## 6) Validation expectations

Use documentation/source cross-checks, not live integrations:

1. Cross-check HLD claims against checked-in docs and source files.
2. Run existing hard-gate evals where HLD/control claims cite BA-EM-005 or BA-EM-009.
3. Run `validate-mcp` only to confirm sandbox rows remain blocked when HLD references sandbox posture.
4. Scan changed docs for forbidden surface/registry drift and legacy unsupported-marker drift.

## 7) Current status

`HLD-G0`, `HLD-G1`, `HLD-G2`, and `HLD-G3` are complete. `docs/development/phase-2-hld-review-package.md` v0.3 captured the review package, and RAJA approved the HLD as the current draft baseline. This plan does not authorize sandbox/live/non-synthetic execution.

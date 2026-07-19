# BA Agent G0 Readiness Package

This package states what G0 authorizes and what remains blocked before the project moves into Phase 1 engineering foundation work.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent G0 Readiness Package |
| Version | 0.1 |
| Status | Completed for F0 review |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P0B] |
| Baseline review | `docs/planning/phase-0-baseline-review.md` |
| Decision baseline | `docs/planning/decision-log.md` v0.3 |

## G0 readiness verdict

G0 is clear for **synthetic-only Phase 1 engineering foundation work** under the decision log. The first build target remains the synthetic Teams standup summary thin slice using synthetic Jira/Git fixtures, evidence refs, `trace_id`, Adaptive Card payload, and no live writes. G0 does not approve sandbox integration, live pilot use, production deployment, live system-of-record reads/writes, unvalidated MCP tools, or Phase 2 Enterprise BA build work.

## Authorized by G0

| Authorized work | Boundary |
| --- | --- |
| Create Phase 1 technical baseline | Must produce `docs/development/phase-1-technical-baseline.md` before scaffolding code. |
| Create local source scaffold | Must remain local/synthetic; no live endpoints, credentials, cloud deploy, or model calls. |
| Add Python package skeleton [inferred] | Must keep orchestrator and gateway/control boundaries separate. |
| Add local command placeholders | Commands may expose help/no-op/safe failures only until implemented by later prompts. |
| Reserve fixture and evaluation paths | Actual fixture schema/loading and seed eval cases remain later gated work. |
| Add local tests/typecheck/eval-placeholder command | Commands must run without secrets, tenants, or network calls. |

## Not authorized by G0

| Blocked work | Required later gate |
| --- | --- |
| Live Jira/Git/Confluence/Calendar/Teams/Copilot 365/Graph/model/MCP calls | Later gate with exact scope approval |
| Sandbox MCP read replacement | G4 after tool validation |
| Any system-of-record write | Explicit approval-gated path and G3/G5 controls |
| Teams posting or escalation delivery | Validated Teams policy and approval path |
| Pilot with live project/team/channel/repo/calendar data | G6 with external non-agent-controlled RAJA approval |
| Production deployment | Separate authorization path after pilot |
| Phase 2 Enterprise BA capability implementation | G7 and separate Phase 2 plan |

## Required Phase 1 boundaries

1. RAJA remains accountable owner for the baseline.
2. Phase 1 must start with [P1T]/[Q1T] technical baseline.
3. Local/synthetic mode is the default and only allowed runtime mode.
4. All live integrations stay disabled.
5. MCP integrations stay stubbed or blocked.
6. Any write-like operation fails closed.
7. Owner-dependent execution decisions use `[RAJA]`.

## Allowed next work

The next allowed work is **[F1] Phase 1 engineering foundation**, in this order:

1. [P1T]/[Q1T] — Phase 1 technical baseline.
2. [P1A]/[Q1A] — Repository/source layout and tooling.
3. [P1B]/[Q1B] — Python package skeleton and safe defaults.
4. [P1C]/[Q1C] — Local command and test tooling.
5. [P1D]/[Q1D] — G1 readiness evidence.

## G0 evidence summary

| Evidence | Status |
| --- | --- |
| DEC-001 through DEC-003 | Closed for G0 synthetic-only baseline |
| DEC-007 | Closed for control design baseline |
| DEC-004 | Deferred; blocks G4/G6, not G1-G3 synthetic work |
| DEC-005 | Conditional; blocks non-synthetic data use |
| DEC-006 | Conditional; blocks sandbox replacement of fixtures |

## QA handoff

[P0B] is ready for [Q0B]. G0 readiness does not imply G1, G2, G3, sandbox, pilot, production, or Phase 2 approval.

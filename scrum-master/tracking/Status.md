# Status — Scrum Master Agent

**Updated:** 2026-06-03 · **Open this each session.**

## Active phase

**P0 — Foundations** (in progress — key decisions locked). PRD + ADR-0001 accepted; design hardened via a persona/skill/agent pass (see `../Persona_Skill_Agent_Usage.md`): added `design/Agent_Blueprint.md` + `evaluation/Test_Strategy.md`, corrected the Teams + JQL facts, applied 5 code fixes.

**2026-06-03 — DOC hardening (closed two critical gaps from the analysis pass):**
- **DOC now tested, not assumed.** Extracted the approval-gate write logic into a pure, I/O-injected choke point (`code/.../scrum_orchestrator/gate.py`) with injectable ports (`ports.py`). Added `test_gate.py` (5 cases, run green here) + `test_doc_invariant.py` (real `interrupt()`/resume via MemorySaver). Asserts: rejected/empty→no write, approved→one action row, delivery-failure→`failed`, idempotent single recommendation. Was the project's biggest hole — the safety invariant had zero automated coverage.
- **Durable resume is real.** Split the lifecycle into `run-daily-brief` (pause + exit) / `list-pending` (the approval queue) / `resume-approval --approve|--reject` (finish from the checkpoint, in a *fresh process*). This exercises the cross-process pause/resume that justified LangGraph (ADR-0001), instead of same-process auto-approve.
- Fixed a doc/code drift: Blueprint §11 no longer shows a `record_action(skipped)` row on rejection (code writes none — now asserted).
- Still open: live-Postgres E2E (tests use in-memory checkpointer), Teams `Action.Submit`→resume wiring, real Jira/OAuth. Adaptive Card still renders `#` headings literally (P1).

## Gate states

| Phase | Gate | State |
|-------|------|-------|
| P0 Foundations | OAuth 3LO + ingest one sprint + Teams hello-world | 🟡 in progress |
| P1 MVP | 5 features live + gated writes + pilot sign-off | ⬜ blocked by P0 |
| P2 Expand | 2nd team config-only + Slack parity | ⬜ blocked by P1 |
| P3 Autonomy | 1 policy auto-action + audited rollback | ⬜ blocked by P2 |

## Decisions locked

- **Orchestration:** LangGraph (ADR-0001)
- **Integration layer:** Go MCP server (consumed via `langchain-mcp-adapters`)
- **Tenancy / auth:** OAuth 2.0 3LO (`offline_access`) from day one — no API-token pilot
- **Channel:** Microsoft Teams first (Power Automate **Workflows** webhook + Adaptive Card; O365 connectors retire 2026-05); Slack is P2 parity
- **Jira:** REST v3 + Agile API, `/search/jql` (legacy `/search` removed ~2025-10-31), ADF, dynamic webhooks
- **Estimation:** time-based (Jira time-tracking fields, read-only; writes via `timetracking` composite) — not story points
- **Reports:** generate a `Report.md` with table of contents — Confluence optional in P2
- **Control model (DOC):** *human-approved writes by construction* — Read → Recommend → Approve → Write

## Open threads (need a decision)

None — all P0–P1 decisions locked. See `../planning/Open_Questions.md`.

## Next actions

1. Register the Atlassian OAuth 2.0 3LO app + consent flow (scopes per `../design/Architecture.md`)
2. Stand up Postgres + Go MCP read path
3. Ingest one real board's active sprint
4. Teams adapter: post a hello-world (P0 gate)

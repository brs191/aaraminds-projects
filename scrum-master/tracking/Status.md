# Status — Scrum Master Agent

**Updated:** 2026-05-31 · **Open this each session.**

## Active phase

**P0 — Foundations** (in progress — key decisions locked). PRD + ADR-0001 accepted; design hardened via a persona/skill/agent pass (see `../Persona_Skill_Agent_Usage.md`): added `design/Agent_Blueprint.md` + `evaluation/Test_Strategy.md`, corrected the Teams + JQL facts, applied 5 code fixes.

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
- **Reports:** generate a `Report.md` with table of contents — Confluence 
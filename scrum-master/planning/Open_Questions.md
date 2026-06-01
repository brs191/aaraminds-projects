# Open Questions — Scrum Master Agent

**Owner:** Raja · **Source:** `../Scrum_Master_Agent_PRD.md` §13

## Resolved — 2026-05-31

1. **Tenancy / auth** (P0) — ✅ **OAuth 2.0 3LO** (`offline_access`) from day one. No single-tenant API-token pilot.
2. **Channel** (P0) — ✅ **Microsoft Teams first.** Slack moves to P2 (parity).
3. **Integration layer** (P0) — ✅ **Go MCP server**, consumed by LangGraph via `langchain-mcp-adapters`. See [ADR-0001](../design/adr/0001-langgraph-orchestration.md).
4. **Estimation** (P1) — ✅ **Time-based.** Use Jira time-tracking fields (`timeoriginalestimate`, `timeestimate`, `timespent`), not story points.
5. **Reports** (P1) — ✅ **Generate a `Report.md` with a table of contents.** No Confluence in MVP; optional Confluence publish in P2.

## Open

None. All P0–P1 decisions are resolved. New questions land here as they arise.

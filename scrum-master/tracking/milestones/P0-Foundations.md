# P0 — Foundations

**Phase gate:** the agent authenticates (OAuth 3LO), ingests the active sprint of one real board, and posts a hello-world to Teams.
**Status:** not started · **Blocks:** P1

## Deliverables
- [ ] Jira auth: OAuth 2.0 3LO app registered + consent flow (scopes per `../../design/Architecture.md`)
- [ ] Go MCP server: read path (issues, sprint, board, changelog)
- [ ] Board / sprint config in `team_config`
- [ ] Postgres provisioned (Azure) + schema for snapshots
- [ ] Snapshot + changelog ingestion for one active sprint
- [ ] Teams channel adapter: post a message
- [ ] LangGraph skeleton consuming the MCP server via `langchain-mcp-adapters`

## Gate check
- [ ] End-to-end: authenticate (OAuth 3LO) → ingest one sprint → post to Teams

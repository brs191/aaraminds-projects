# orchestrator

LangGraph reasoning layer for the Scrum Master Agent. Owns the Daily Brief graph and
the durable human-in-the-loop approval gate.

## Graph

```
fetch_sprint → build_brief → approval_gate (interrupt) → publish
```

`approval_gate` calls `langgraph.types.interrupt()`, which pauses the run and persists
state to the Postgres checkpointer. The run resumes only when invoked again with
`Command(resume={"approved": ...})`. That durable pause/resume is the write gate.

## Run without Docker (dev)

```bash
cd apps/orchestrator
python -m venv .venv && source .venv/bin/activate
pip install -e ".[dev]"

# needs a running jira-mcp, teams-adapter, and postgres (see root README / compose)
export JIRA_MCP_URL=http://localhost:8080/mcp
export TEAMS_ADAPTER_URL=http://localhost:8090
export DATABASE_URL=postgresql://scrum:scrum@localhost:5432/scrum
export AUTO_APPROVE=true
run-daily-brief
```

## Test (pure, no services needed)

```bash
PYTHONPATH=. python -m pytest tests -q
```

## Env vars

| Var | Default | Meaning |
|-----|---------|---------|
| `JIRA_MCP_URL` | `http://localhost:8080/mcp` | Go jira-mcp endpoint |
| `TEAMS_ADAPTER_URL` | `http://localhost:8090` | Go teams-adapter |
| `DATABASE_URL` | `postgresql://scrum:scrum@localhost:5432/scrum` | Postgres (checkpointer + audit) |
| `BOARD_ID` | `1` | board to brief |
| `AUTO_APPROVE` | `true` | auto-resume past the gate (demo); `false` = pending approval |
| `STALE_DAYS` | `3` | stale threshold |

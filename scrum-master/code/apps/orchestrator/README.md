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

The safety-critical decision is **not** inline in the graph: `gate.py` holds the pure,
I/O-injected choke point (`coerce_approved` + `execute_decision`), and `ports.py` wraps
Postgres/Teams behind injectable ports. That is what makes the DOC — *no write without a
recorded approval* — unit-testable without standing up LangGraph/Postgres/HTTP.

## Durable approval lifecycle (run → pending → resume)

The point of the checkpointer is that an approval can arrive **in a different process,
later**. Three console scripts split that lifecycle:

```bash
AUTO_APPROVE=false run-daily-brief          # build brief, pause at the gate, EXIT (pending)
list-pending                                # recommendations awaiting a decision
resume-approval --thread daily-brief-1 --approve   # finish it (fresh process, from checkpoint)
resume-approval --thread daily-brief-1 --reject    # …or reject; nothing is written (DOC)
```

`resume-approval` reattaches to the persisted thread and replays the completed
read/build nodes from the checkpoint — no Jira client needed to finish publishing.
With `AUTO_APPROVE=true`, `run-daily-brief` resumes inline instead (the compose demo).

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

## Test

```bash
PYTHONPATH=. python -m pytest tests -q
```

`test_brief.py` and `test_gate.py` are pure (no services) — `test_gate.py` asserts the
DOC write branch directly. `test_doc_invariant.py` drives the real `interrupt()`/resume
with an in-memory checkpointer (approve→one action, reject/empty→no write, delivery
failure→`failed`, idempotent single recommendation) — still no Postgres or network.

## Env vars

| Var | Default | Meaning |
|-----|---------|---------|
| `JIRA_MCP_URL` | `http://localhost:8080/mcp` | Go jira-mcp endpoint |
| `TEAMS_ADAPTER_URL` | `http://localhost:8090` | Go teams-adapter |
| `DATABASE_URL` | `postgresql://scrum:scrum@localhost:5432/scrum` | Postgres (checkpointer + audit) |
| `BOARD_ID` | `1` | board to brief |
| `AUTO_APPROVE` | `true` | auto-resume past the gate (demo); `false` = pending approval |
| `STALE_DAYS` | `3` | stale threshold |

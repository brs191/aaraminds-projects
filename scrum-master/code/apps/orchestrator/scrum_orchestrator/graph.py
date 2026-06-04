"""The Daily Scrum Brief graph — the P0 vertical slice.

Flow:  fetch_sprint -> build_brief -> approval_gate (interrupt) -> publish

The approval_gate node calls langgraph.types.interrupt(), which pauses the graph
and persists state to the checkpointer. The run resumes only when invoked again with
Command(resume=...). That durable pause/resume IS the human-in-the-loop write gate —
the reason LangGraph was chosen (ADR-0001).

The safety-critical logic lives in gate.py (pure, I/O-injected) rather than inline here,
so the DOC — *no write without a recorded approval* — is unit-testable without standing
up LangGraph/Postgres/HTTP. The audit store and publisher are injected as ports; the
defaults wire the real Postgres + teams-adapter implementations (ports.py).
"""

from __future__ import annotations

from typing import Any

from langgraph.graph import END, START, StateGraph
from langgraph.types import interrupt

from .brief import brief_title, build_brief
from .config import Config
from .gate import coerce_approved, execute_decision
from .mcp_client import JiraMCP
from .ports import PostgresAuditStore, TeamsPublisher
from .state import BriefState


def build_graph(
    checkpointer: Any,
    jira: JiraMCP | None,
    cfg: Config,
    *,
    audit_store: Any | None = None,
    publisher: Any | None = None,
):
    """Compile the Daily Brief graph.

    ``jira`` may be ``None`` on the resume path: fetch_sprint and build_brief already
    completed in the run-to-gate process and are replayed from the checkpointer, so the
    MCP client is not needed to finish publishing.

    ``audit_store`` / ``publisher`` default to the real Postgres + Teams adapters; tests
    inject fakes to assert the DOC directly.
    """
    store = audit_store or PostgresAuditStore(cfg.database_url)
    pub = publisher or TeamsPublisher(cfg.teams_adapter_url)

    async def fetch_sprint(state: BriefState) -> dict[str, Any]:
        if jira is None:
            raise RuntimeError(
                "fetch_sprint requires a Jira MCP client, but jira=None. On resume this "
                "node should be replayed from the checkpointer, not re-executed."
            )
        sprint = await jira.call("get_active_sprint", board_id=state["board_id"])
        issues_doc = await jira.call("get_sprint_issues", sprint_id=str(sprint["id"]))
        return {"sprint": sprint, "issues": issues_doc.get("issues", [])}

    async def build_brief_node(state: BriefState) -> dict[str, Any]:
        markdown = build_brief(state["sprint"], state["issues"], stale_days=cfg.stale_days)
        rec_id = await store.record_recommendation(
            kind="daily_brief",
            payload={"board_id": state["board_id"], "sprint_id": state["sprint"].get("id")},
        )
        return {"brief_markdown": markdown, "recommendation_id": rec_id}

    def approval_gate(state: BriefState) -> dict[str, Any]:
        # interrupt() must be the first thing this node does: on resume the node body
        # re-executes from the top, so any pre-interrupt side effect would double-run.
        # The recommendation is recorded in the *earlier* build_brief node, which is
        # replayed (not re-run) — so exactly one recommendation row exists across resume.
        decision = interrupt(
            {
                "action": "post_daily_brief",
                "recommendation_id": state.get("recommendation_id"),
                "preview": state["brief_markdown"],
            }
        )
        return {"approved": coerce_approved(decision)}

    async def publish(state: BriefState) -> dict[str, Any]:
        # All write/audit branching is in the single choke point in gate.py.
        return await execute_decision(
            approved=bool(state.get("approved")),
            recommendation_id=state["recommendation_id"],
            title=brief_title(state["sprint"]),
            markdown=state["brief_markdown"],
            store=store,
            publisher=pub,
        )

    graph = StateGraph(BriefState)
    graph.add_node("fetch_sprint", fetch_sprint)
    graph.add_node("build_brief", build_brief_node)
    graph.add_node("approval_gate", approval_gate)
    graph.add_node("publish", publish)

    graph.add_edge(START, "fetch_sprint")
    graph.add_edge("fetch_sprint", "build_brief")
    graph.add_edge("build_brief", "approval_gate")
    graph.add_edge("approval_gate", "publish")
    graph.add_edge("publish", END)

    return graph.compile(checkpointer=checkpointer)

"""The Daily Scrum Brief graph — the P0 vertical slice.

Flow:  fetch_sprint -> build_brief -> approval_gate (interrupt) -> publish

The approval_gate node calls langgraph.types.interrupt(), which pauses the graph
and persists state to the Postgres checkpointer. The run resumes only when invoked
again with Command(resume=...). That durable pause/resume IS the human-in-the-loop
write gate — the reason LangGraph was chosen (ADR-0001).
"""

from __future__ import annotations

from typing import Any

from langgraph.graph import END, START, StateGraph
from langgraph.types import interrupt

from . import audit
from .brief import brief_title, build_brief
from .config import Config
from .mcp_client import JiraMCP
from .state import BriefState
from .teams_client import post_to_teams


def build_graph(checkpointer: Any, jira: JiraMCP, cfg: Config):
    async def fetch_sprint(state: BriefState) -> dict[str, Any]:
        sprint = await jira.call("get_active_sprint", board_id=state["board_id"])
        issues_doc = await jira.call("get_sprint_issues", sprint_id=str(sprint["id"]))
        return {"sprint": sprint, "issues": issues_doc.get("issues", [])}

    async def build_brief_node(state: BriefState) -> dict[str, Any]:
        markdown = build_brief(state["sprint"], state["issues"], stale_days=cfg.stale_days)
        rec_id = await audit.record_recommendation(
            cfg.database_url,
            kind="daily_brief",
            payload={"board_id": state["board_id"], "sprint_id": state["sprint"].get("id")},
        )
        return {"brief_markdown": markdown, "recommendation_id": rec_id}

    def approval_gate(state: BriefState) -> dict[str, Any]:
        decision = interrupt(
            {
                "action": "post_daily_brief",
                "recommendation_id": state.get("recommendation_id"),
                "preview": state["brief_markdown"],
            }
        )
        approved = decision.get("approved") if isinstance(decision, dict) else bool(decision)
        return {"approved": bool(approved)}

    async def publish(state: BriefState) -> dict[str, Any]:
        rec_id = state["recommendation_id"]
        if not state.get("approved"):
            await audit.record_approval(cfg.database_url, rec_id, "rejected", "human")
            return {"delivery_status": "rejected"}

        await audit.record_approval(cfg.database_url, rec_id, "approved", "human")
        # Never leave the audit chain half-written: a delivery failure records
        # result="failed" rather than crashing the node after the approval committed.
        try:
            result = await post_to_teams(
                cfg.teams_adapter_url, brief_title(state["sprint"]), state["brief_markdown"]
            )
            status = result.get("status", "unknown")
        except Exception as exc:  # noqa: BLE001 — record then surface as status
            await audit.record_action(cfg.database_url, rec_id, "post_to_teams", "failed")
            return {"delivery_status": f"failed: {exc}"}
        await audit.record_action(cfg.database_url, rec_id, "post_to_teams", status)
        return {"delivery_status": status}

    graph = StateGraph(BriefState)
    graph.add_node("fetch_sprint", fetch_spri
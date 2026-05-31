"""Entry point: run one Daily Brief through the graph, honoring the approval gate.

P0 demo behavior:
  * Runs the graph until the approval gate interrupts.
  * Prints the brief preview.
  * If AUTO_APPROVE=true, resumes with approved=True so the brief is posted end-to-end.
  * If AUTO_APPROVE=false, stops at "pending approval" — the realistic HITL path,
    where a Teams Action.Submit would later resume the thread.
"""

from __future__ import annotations

import asyncio

from langgraph.checkpoint.postgres.aio import AsyncPostgresSaver
from langgraph.types import Command

from .config import load
from .graph import build_graph
from .mcp_client import JiraMCP


async def run() -> None:
    cfg = load()
    print(f"[orchestrator] board={cfg.board_id} auto_approve={cfg.auto_approve}")
    print(f"[orchestrator] connecting to jira-mcp at {cfg.jira_mcp_url}")
    jira = await JiraMCP.connect(cfg.jira_mcp_url)
    print(f"[orchestrator] MCP tools: {jira.tool_names}")

    async with AsyncPostgresSaver.from_conn_string(cfg.database_url) as checkpointer:
        await checkpointer.setup()
        graph = build_graph(checkpointer, jira, cfg)
        thread = {"configurable": {"thread_id": f"daily-brief-{cfg.board_id}"}}

        result = await graph.ainvoke({"board_id": cfg.board_id}, thread)

        if "__interrupt__" in result:
            payload = result["__interrupt__"][0].value
            print("\n===== APPROVAL REQUIRED =====")
            print(payload["preview"])
            print("===== END PREVIEW =====\n")

            if not cfg.auto_approve:
                print("[orchestrator] AUTO_APPROVE=false — brief is PENDING approval. "
                      "Resume with Command(resume={'approved': True}).")
                return

            print("[orchestrator] AUTO_APPROVE=true — approving and posting.")
            result = await graph.ainvoke(Command(resume={"approved": True}), thread)

        print(f"[orchestrator] delivery_status={result.get('delivery_status')}")


def cli() -> None:
    asyncio.run(run())


if __name__ == "__main__":
    cli()

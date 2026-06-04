"""Entry points for the Daily Brief — run-to-gate, resume, and list-pending.

The Defining Operational Constraint is durable human-in-the-loop: a brief pauses at the
approval gate and may be approved hours later, in a *different process*. To make that real
(not a same-process auto-approve), the lifecycle is split across three console scripts:

  run-daily-brief   build the brief, record the recommendation, pause at the gate.
                    With AUTO_APPROVE=true it resumes inline (the compose demo); with
                    AUTO_APPROVE=false it persists to the Postgres checkpointer and EXITS,
                    leaving a pending approval — the realistic path.
  list-pending      show recommendations with no approval row yet (the approval queue).
  resume-approval   re-attach to the persisted thread and approve/reject it. Because the
                    state lives in the Postgres checkpointer, this completes a brief that
                    was paused by an earlier, already-exited process — that durability is
                    exactly why LangGraph was chosen (ADR-0001).

Resume needs no Jira client: fetch_sprint/build_brief already ran and are replayed from
the checkpointer, so build_graph(jira=None) is correct here.
"""

from __future__ import annotations

import argparse
import asyncio

from langgraph.checkpoint.postgres.aio import AsyncPostgresSaver
from langgraph.types import Command

from . import audit
from .config import Config, load
from .graph import build_graph
from .mcp_client import JiraMCP


def _thread(cfg: Config) -> dict:
    return {"configurable": {"thread_id": f"daily-brief-{cfg.board_id}"}}


# --------------------------------------------------------------------------- run

async def run() -> None:
    cfg = load()
    print(f"[orchestrator] board={cfg.board_id} auto_approve={cfg.auto_approve}")
    print(f"[orchestrator] connecting to jira-mcp at {cfg.jira_mcp_url}")
    jira = await JiraMCP.connect(cfg.jira_mcp_url)
    print(f"[orchestrator] MCP tools: {jira.tool_names}")

    async with AsyncPostgresSaver.from_conn_string(cfg.database_url) as checkpointer:
        await checkpointer.setup()
        graph = build_graph(checkpointer, jira, cfg)
        thread = _thread(cfg)

        result = await graph.ainvoke({"board_id": cfg.board_id}, thread)

        if "__interrupt__" not in result:
            print(f"[orchestrator] delivery_status={result.get('delivery_status')}")
            return

        payload = result["__interrupt__"][0].value
        rec_id = payload.get("recommendation_id")
        print("\n===== APPROVAL REQUIRED =====")
        print(payload["preview"])
        print("===== END PREVIEW =====\n")

        if not cfg.auto_approve:
            tid = thread["configurable"]["thread_id"]
            print(
                "[orchestrator] AUTO_APPROVE=false — brief is PENDING approval "
                f"(recommendation #{rec_id}, thread '{tid}'). State is persisted; this "
                "process can exit. Approve later, even from another process, with:\n"
                f"    resume-approval --thread {tid} --approve\n"
                f"    resume-approval --thread {tid} --reject"
            )
            return

        print("[orchestrator] AUTO_APPROVE=true — approving and posting inline.")
        result = await graph.ainvoke(Command(resume={"approved": True}), thread)
        print(f"[orchestrator] delivery_status={result.get('delivery_status')}")


# ------------------------------------------------------------------------- resume

async def resume(cfg: Config, thread_id: str, *, approved: bool) -> None:
    thread = {"configurable": {"thread_id": thread_id}}
    async with AsyncPostgresSaver.from_conn_string(cfg.database_url) as checkpointer:
        # jira=None: the read/build nodes are replayed from the checkpoint, not re-run.
        graph = build_graph(checkpointer, None, cfg)

        snapshot = await graph.aget_state(thread)
        if not snapshot.next:
            print(
                f"[orchestrator] no pending approval for thread '{thread_id}'. "
                "Run `list-pending` to see what's awaiting a decision."
            )
            return

        verb = "approving" if approved else "rejecting"
        print(f"[orchestrator] {verb} thread '{thread_id}' (resuming from checkpoint)…")
        result = await graph.ainvoke(Command(resume={"approved": approved}), thread)
        status = result.get("delivery_status")
        print(f"[orchestrator] delivery_status={status}")
        if not approved:
            print("[orchestrator] rejected — no write to Jira/Teams (DOC upheld).")


# ------------------------------------------------------------------------ pending

async def pending(cfg: Config) -> None:
    rows = await audit.fetch_pending(cfg.database_url)
    if not rows:
        print("[orchestrator] no pending approvals.")
        return
    print(f"[orchestrator] {len(rows)} pending approval(s):")
    for rec_id, kind, created_at in rows:
        print(f"  • recommendation #{rec_id}  kind={kind}  created={created_at:%Y-%m-%d %H:%M}")
    print("Approve one with:  resume-approval --thread daily-brief-<board> --approve")


# ---------------------------------------------------------------------------- CLIs


def cli() -> None:
    """Entry for `run-daily-brief` (kept as the container ENTRYPOINT)."""
    asyncio.run(run())


def resume_cli() -> None:
    cfg = load()
    parser = argparse.ArgumentParser(
        prog="resume-approval",
        description="Resume a pending Daily Brief approval (durable human-in-the-loop).",
    )
    parser.add_argument(
        "--thread",
        default=f"daily-brief-{cfg.board_id}",
        help="thread id to resume (default: daily-brief-<board>)",
    )
    grp = parser.add_mutually_exclusive_group(required=True)
    grp.add_argument("--approve", action="store_true", help="approve and publish")
    grp.add_argument("--reject", action="store_true", help="reject; nothing is written")
    args = parser.parse_args()
    asyncio.run(resume(cfg, args.thread, approved=args.approve))


def pending_cli() -> None:
    asyncio.run(pending(load()))


if __name__ == "__main__":
    cli()

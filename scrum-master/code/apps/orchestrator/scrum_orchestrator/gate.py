"""Approval-gate decision logic — the DOC choke point, made testable by construction.

The Defining Operational Constraint is *human-approved writes by construction*
(scrum-master/design/Agent_Blueprint.md §6): no write to Jira or any channel occurs
without a recorded approval. That invariant used to live inside closures in graph.py,
where it could only be exercised by standing up LangGraph + Postgres + HTTP. It now
lives here as two pure, I/O-injected functions so it can be asserted directly in unit
tests (tests/test_gate.py) and end-to-end through the real interrupt (tests/test_doc_invariant.py).

Nothing in this module imports LangGraph, psycopg, or httpx: the audit store and the
publisher are injected as ports (see ports.py for the production adapters, and the
fakes in the tests). That is what makes the DOC falsifiable rather than assumed.
"""

from __future__ import annotations

from typing import Any, Protocol


class AuditStore(Protocol):
    """The write side of the recommendation -> approval -> action_audit trust chain."""

    async def record_approval(
        self, recommendation_id: int, decision: str, decided_by: str
    ) -> None: ...

    async def record_action(
        self, recommendation_id: int, action: str, result: str
    ) -> None: ...


class Publisher(Protocol):
    """Anything that can deliver an approved artifact to a channel (Teams in P0)."""

    async def post(self, title: str, markdown: str) -> dict[str, Any]: ...


def coerce_approved(decision: Any) -> bool:
    """Fail-closed reading of a resume payload: only an explicit truthy approval counts.

    The approval gate resumes with ``Command(resume=<decision>)``; a malformed or empty
    payload must be treated as a *reject*, never as an accidental approve.

        {}                      -> False   (empty / rubber-stamp guard)
        {"approved": None}      -> False
        {"approved": False}     -> False
        {"approved": True}      -> True
        True                    -> True    (bare bool convenience)

    """
    if isinstance(decision, dict):
        return bool(decision.get("approved"))
    return bool(decision)


async def execute_decision(
    *,
    approved: bool,
    recommendation_id: int,
    title: str,
    markdown: str,
    store: AuditStore,
    publisher: Publisher,
    decided_by: str = "human",
    action: str = "post_to_teams",
) -> dict[str, str]:
    """The single place a write can happen — so the DOC has exactly one choke point.

    Contract (asserted in tests):
      * Rejected  -> record_approval(rejected); **no** action row; nothing published.
      * Approved  -> record_approval(approved), then publish, then record_action(<status>).
      * Delivery failure -> record_action(<action>, "failed") and return a failed status
        instead of crashing after the approval already committed (audit never half-written).

    Returns ``{"delivery_status": ...}`` for the graph to merge into state.
    """
    if not approved:
        await store.record_approval(recommendation_id, "rejected", decided_by)
        return {"delivery_status": "rejected"}

    await store.record_approval(recommendation_id, "approved", decided_by)
    try:
        result = await publisher.post(title, markdown)
        status = str(result.get("status", "unknown"))
    except Exception as exc:  # noqa: BLE001 — record the failure, then surface it as status
        await store.record_action(recommendation_id, action, "failed")
        return {"delivery_status": f"failed: {exc}"}

    await store.record_action(recommendation_id, action, status)
    return {"delivery_status": status}

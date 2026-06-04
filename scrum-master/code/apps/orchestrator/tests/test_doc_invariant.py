"""DOC end-to-end tests — the real LangGraph interrupt/resume, in-memory.

These drive the compiled graph through a genuine pause at interrupt() and a resume via
Command(resume=...), using an in-memory checkpointer and fake ports (no Postgres, no HTTP).
They prove the Defining Operational Constraint on the actual control flow, not just the
extracted helper:

  * at the pause, a recommendation exists but nothing is approved or written;
  * approve  -> exactly one approval + one action row, published once;
  * reject / empty resume -> an approval(rejected) and zero action rows, nothing published;
  * delivery failure -> an action row with result="failed", no crash;
  * resume replays the completed read/build nodes -> exactly ONE recommendation row.

Interrupt/resume semantics are identical across checkpointers, so MemorySaver here is a
faithful stand-in for the Postgres checkpointer used in production (a live-Postgres E2E
remains a P1 gap, see Test_Strategy.md).
"""

from __future__ import annotations

import asyncio

from langgraph.checkpoint.memory import MemorySaver
from langgraph.types import Command

from scrum_orchestrator.config import Config
from scrum_orchestrator.graph import build_graph

SPRINT = {"id": 101, "name": "CRS Sprint 24", "goal": "Stabilize the routing API"}
ISSUES = [
    {"key": "CRS-405", "summary": "Audit log", "status": "Done", "assignee": "Priya N.",
     "blocked": False, "timeoriginalestimate": 18000, "timeestimate": 0, "timespent": 17100, "daysInStatus": 1},
    {"key": "CRS-417", "summary": "SOAP faults", "status": "In Progress", "assignee": "Marco D.",
     "blocked": True, "blockReason": "is blocked by CRS-420", "timeoriginalestimate": 14400,
     "timeestimate": 10800, "timespent": 3600, "daysInStatus": 4},
]


class FakeJira:
    async def call(self, name, **_):
        if name == "get_active_sprint":
            return dict(SPRINT)
        if name == "get_sprint_issues":
            return {"sprintId": 101, "issues": [dict(i) for i in ISSUES]}
        raise KeyError(name)


class FakeAuditStore:
    def __init__(self):
        self.recommendations = []
        self.approvals = []
        self.actions = []

    async def record_recommendation(self, kind, payload):
        self.recommendations.append((kind, payload))
        return len(self.recommendations)

    async def record_approval(self, recommendation_id, decision, decided_by):
        self.approvals.append((recommendation_id, decision, decided_by))

    async def record_action(self, recommendation_id, action, result):
        self.actions.append((recommendation_id, action, result))


class FakePublisher:
    def __init__(self, status="delivered"):
        self.status = status
        self.calls = []

    async def post(self, title, markdown):
        self.calls.append((title, markdown))
        return {"status": self.status}


class FailingPublisher:
    def __init__(self):
        self.calls = []

    async def post(self, title, markdown):
        self.calls.append((title, markdown))
        raise RuntimeError("teams 502")


def _graph(store, publisher):
    return build_graph(
        MemorySaver(), FakeJira(), Config(), audit_store=store, publisher=publisher
    )


def _thread(name):
    return {"configurable": {"thread_id": name}}


async def _run_to_gate(graph, thread):
    result = await graph.ainvoke({"board_id": "1"}, thread)
    assert "__interrupt__" in result, "graph must pause at the approval gate"
    return result


def test_pause_records_recommendation_but_writes_nothing():
    store, pub = FakeAuditStore(), FakePublisher()

    async def scenario():
        graph = _graph(store, pub)
        await _run_to_gate(graph, _thread("pause"))

    asyncio.run(scenario())
    assert len(store.recommendations) == 1     # the brief was proposed…
    assert store.approvals == []               # …but not approved,
    assert store.actions == []                 # …nothing written,
    assert pub.calls == []                     # …and nothing published at the pause.


def test_approve_resume_publishes_exactly_once():
    store, pub = FakeAuditStore(), FakePublisher(status="delivered")

    async def scenario():
        graph, thread = _graph(store, pub), _thread("approve")
        await _run_to_gate(graph, thread)
        return await graph.ainvoke(Command(resume={"approved": True}), thread)

    result = asyncio.run(scenario())
    assert result["delivery_status"] == "delivered"
    assert store.approvals == [(1, "approved", "human")]
    assert store.actions == [(1, "post_to_teams", "delivered")]
    assert len(pub.calls) == 1
    assert len(store.recommendations) == 1     # idempotent: build node replayed, not re-run


def test_reject_resume_writes_nothing_end_to_end():
    store, pub = FakeAuditStore(), FakePublisher()

    async def scenario():
        graph, thread = _graph(store, pub), _thread("reject")
        await _run_to_gate(graph, thread)
        return await graph.ainvoke(Command(resume={"approved": False}), thread)

    result = asyncio.run(scenario())
    assert result["delivery_status"] == "rejected"
    assert store.approvals == [(1, "rejected", "human")]
    assert store.actions == []                 # DOC: rejection writes no action row
    assert pub.calls == []


def test_empty_resume_is_fail_closed_reject():
    store, pub = FakeAuditStore(), FakePublisher()

    async def scenario():
        graph, thread = _graph(store, pub), _thread("failclosed")
        await _run_to_gate(graph, thread)
        return await graph.ainvoke(Command(resume={}), thread)  # malformed/empty

    result = asyncio.run(scenario())
    assert result["delivery_status"] == "rejected"
    assert store.actions == []
    assert pub.calls == []


def test_delivery_failure_is_audited_through_the_graph():
    store, pub = FakeAuditStore(), FailingPublisher()

    async def scenario():
        graph, thread = _graph(store, pub), _thread("failure")
        await _run_to_gate(graph, thread)
        return await graph.ainvoke(Command(resume={"approved": True}), thread)

    result = asyncio.run(scenario())              # must not raise
    assert result["delivery_status"].startswith("failed")
    assert store.approvals == [(1, "approved", "human")]
    assert store.actions == [(1, "post_to_teams", "failed")]

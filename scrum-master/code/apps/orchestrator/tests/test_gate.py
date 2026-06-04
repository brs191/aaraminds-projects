"""DOC unit tests — the approval gate's write logic, proven without LangGraph/DB/HTTP.

These exercise gate.execute_decision and gate.coerce_approved directly with fake ports,
so they run with nothing but pytest. They assert the Defining Operational Constraint at
its choke point: a write (publish + action_audit row) happens only after an approval, and
a rejected/empty decision writes nothing. The full interrupt/resume path is covered in
test_doc_invariant.py (which needs LangGraph and is exercised in CI).
"""

from __future__ import annotations

import asyncio

from scrum_orchestrator.gate import coerce_approved, execute_decision

REC = 7


class FakeAuditStore:
    def __init__(self):
        self.approvals = []  # (rec_id, decision, by)
        self.actions = []    # (rec_id, action, result)

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


def _run(approved, store, publisher):
    return asyncio.run(
        execute_decision(
            approved=approved,
            recommendation_id=REC,
            title="Daily Scrum Brief — CRS Sprint 24",
            markdown="# brief",
            store=store,
            publisher=publisher,
        )
    )


# --- coerce_approved: fail-closed -------------------------------------------------

def test_coerce_approved_is_fail_closed():
    assert coerce_approved({}) is False                    # empty resume
    assert coerce_approved({"approved": None}) is False     # null
    assert coerce_approved({"approved": False}) is False
    assert coerce_approved({"approved": True}) is True
    assert coerce_approved(True) is True                    # bare-bool convenience
    assert coerce_approved(False) is False


# --- execute_decision: the DOC branch --------------------------------------------

def test_rejected_writes_nothing():
    store, pub = FakeAuditStore(), FakePublisher()
    result = _run(False, store, pub)
    assert result == {"delivery_status": "rejected"}
    assert store.approvals == [(REC, "rejected", "human")]
    assert store.actions == []          # DOC: no action row on rejection
    assert pub.calls == []              # DOC: nothing published


def test_approved_records_approval_then_action_and_publishes():
    store, pub = FakeAuditStore(), FakePublisher(status="delivered")
    result = _run(True, store, pub)
    assert result == {"delivery_status": "delivered"}
    assert store.approvals == [(REC, "approved", "human")]
    assert len(pub.calls) == 1
    assert store.actions == [(REC, "post_to_teams", "delivered")]


def test_stub_status_passthrough():
    # teams-adapter in stub mode returns {"status": "logged"} — the audit records it.
    store, pub = FakeAuditStore(), FakePublisher(status="logged")
    result = _run(True, store, pub)
    assert result == {"delivery_status": "logged"}
    assert store.actions == [(REC, "post_to_teams", "logged")]


def test_delivery_failure_is_audited_not_raised():
    store, pub = FakeAuditStore(), FailingPublisher()
    result = _run(True, store, pub)              # must not raise
    assert result["delivery_status"].startswith("failed")
    assert store.approvals == [(REC, "approved", "human")]
    assert store.actions == [(REC, "post_to_teams", "failed")]  # never half-written

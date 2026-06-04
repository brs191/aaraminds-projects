"""Production adapters for the gate's injected ports.

gate.py defines the ``AuditStore`` and ``Publisher`` Protocols and depends on neither
Postgres nor HTTP. These are the real implementations the graph wires in by default:
``PostgresAuditStore`` delegates to the SQL helpers in audit.py, and ``TeamsPublisher``
delegates to the teams-adapter HTTP client. Tests inject fakes with the same shape, so
the safety-critical branch in gate.execute_decision is exercised without real I/O.
"""

from __future__ import annotations

from typing import Any

from . import audit
from .teams_client import post_to_teams


class PostgresAuditStore:
    """AuditStore backed by Postgres (the recommendation/approval/action_audit tables)."""

    def __init__(self, database_url: str) -> None:
        self._db = database_url

    async def record_recommendation(self, kind: str, payload: dict[str, Any]) -> int:
        return await audit.record_recommendation(self._db, kind, payload)

    async def record_approval(
        self, recommendation_id: int, decision: str, decided_by: str
    ) -> None:
        await audit.record_approval(self._db, recommendation_id, decision, decided_by)

    async def record_action(
        self, recommendation_id: int, action: str, result: str
    ) -> None:
        await audit.record_action(self._db, recommendation_id, action, result)


class TeamsPublisher:
    """Publisher that delivers an approved brief to the Go teams-adapter."""

    def __init__(self, adapter_url: str) -> None:
        self._url = adapter_url

    async def post(self, title: str, markdown: str) -> dict[str, Any]:
        return await post_to_teams(self._url, title, markdown)

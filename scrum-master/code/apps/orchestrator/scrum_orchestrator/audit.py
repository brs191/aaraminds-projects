"""The recommendation -> approval -> action_audit trust chain (Postgres).

Every write the agent makes is traceable to a human decision. These three helpers
persist that chain; the graph calls record_recommendation before the gate, and
record_approval + record_action after it.
"""

from __future__ import annotations

import json
from typing import Any

import psycopg


async def record_recommendation(database_url: str, kind: str, payload: dict[str, Any]) -> int:
    async with await psycopg.AsyncConnection.connect(database_url) as conn:
        async with conn.cursor() as cur:
            await cur.execute(
                "INSERT INTO recommendation (kind, payload) VALUES (%s, %s) RETURNING id",
                (kind, json.dumps(payload)),
            )
            row = await cur.fetchone()
        await conn.commit()
    if row is None:
        raise RuntimeError("INSERT ... RETURNING id returned no row")
    return int(row[0])


async def record_approval(database_url: str, recommendation_id: int, decision: str, decided_by: str) -> None:
    async with await psycopg.AsyncConnection.connect(database_url) as conn:
        async with conn.cursor() as cur:
            await cur.execute(
                "INSERT INTO approval (recommendation_id, decision, decided_by) VALUES (%s, %s, %s)",
                (recommendation_id, decision, decided_by),
            )
        await conn.commit()


async def record_action(database_url: str, recommendation_id: int, action: str, result: str) -> None:
    async with await psycopg.AsyncConnection.connect(database_url) as conn:
        async with conn.cursor() as cur:
            await cur.execute(
                "INSERT INTO action_audit (recommendation_id, action, result) VALUES (%s, %s, %s)",
                (recommendation_id, action, result),
            )
        await conn.commit()

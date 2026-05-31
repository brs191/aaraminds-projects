"""HTTP client for the Go teams-adapter."""

from __future__ import annotations

from typing import Any

import httpx


async def post_to_teams(adapter_url: str, title: str, markdown: str) -> dict[str, Any]:
    async with httpx.AsyncClient(timeout=10.0) as client:
        resp = await client.post(
            f"{adapter_url}/post",
            json={"title": title, "markdown": markdown},
        )
        resp.raise_for_status()
        return resp.json()

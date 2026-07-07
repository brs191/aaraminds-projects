from __future__ import annotations

import itertools
import json
from typing import Any

import httpx


class MCPToolError(RuntimeError):
    pass


class MCPClient:
    def __init__(self, base_url: str, timeout_seconds: float = 30.0, client: httpx.AsyncClient | None = None) -> None:
        self._base_url = base_url.rstrip("/")
        self._timeout_seconds = timeout_seconds
        self._client = client
        self._ids = itertools.count(1)

    async def close(self) -> None:
        if self._client is not None:
            await self._client.aclose()
            self._client = None

    def _ensure_client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(timeout=self._timeout_seconds)
        return self._client

    async def call_tool(self, name: str, arguments: dict[str, Any]) -> dict[str, Any]:
        payload = {
            "jsonrpc": "2.0",
            "id": next(self._ids),
            "method": "tools/call",
            "params": {"name": name, "arguments": arguments},
        }
        try:
            response = await self._ensure_client().post(self._base_url, json=payload)
            response.raise_for_status()
        except RuntimeError as exc:
            if "Event loop is closed" in str(exc):
                await self.close()
                response = await self._ensure_client().post(self._base_url, json=payload)
                response.raise_for_status()
            else:
                raise MCPToolError(f"{name} transport failure: {exc}") from exc
        except httpx.HTTPError as exc:
            detail = ""
            if hasattr(exc, "response") and exc.response is not None:
                detail = exc.response.text[:400]
            raise MCPToolError(f"{name} transport failure: {exc}{' :: ' + detail if detail else ''}") from exc

        try:
            body = response.json()
        except ValueError as exc:
            raise MCPToolError(f"{name} returned invalid JSON response") from exc
        if "error" in body:
            raise MCPToolError(f"{name} failed: {body['error']}")
        result = body.get("result")
        if not isinstance(result, dict):
            raise MCPToolError(f"{name} returned invalid result envelope")
        content = result.get("content")
        if not isinstance(content, list) or len(content) == 0:
            raise MCPToolError(f"{name} returned empty content")
        first = content[0]
        if not isinstance(first, dict) or not isinstance(first.get("text"), str):
            raise MCPToolError(f"{name} returned unsupported content shape")
        text = first["text"]
        try:
            parsed = json.loads(text)
        except ValueError as exc:
            raise MCPToolError(f"{name} returned non-JSON text payload") from exc
        if not isinstance(parsed, dict):
            raise MCPToolError(f"{name} payload must be a JSON object")
        return parsed

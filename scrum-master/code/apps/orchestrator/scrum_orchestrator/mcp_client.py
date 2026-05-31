"""Thin wrapper over the Go jira-mcp server via langchain-mcp-adapters.

The ADR (scrum-master/design/adr/0001) mandates consuming the Go MCP server through
langchain-mcp-adapters rather than a Python-native Jira client, so the integration
layer stays reusable across agents.
"""

from __future__ import annotations

import json
from typing import Any

from langchain_mcp_adapters.client import MultiServerMCPClient


class JiraMCP:
    """Loads MCP tools once and exposes a typed-ish call() helper."""

    def __init__(self, tools: list[Any]) -> None:
        self._tools = {t.name: t for t in tools}

    @classmethod
    async def connect(cls, url: str) -> "JiraMCP":
        client = MultiServerMCPClient(
            {"jira": {"url": url, "transport": "streamable_http"}}
        )
        tools = await client.get_tools()
        return cls(tools)

    @property
    def tool_names(self) -> list[str]:
        return list(self._tools)

    async def call(self, name: str, **arguments: Any) -> Any:
        if name not in self._tools:
            raise KeyError(f"MCP tool {name!r} not found; available: {self.tool_names}")
        raw = await self._tools[name].ainvoke(arguments)
        # MCP text content comes back as a JSON string for our tools.
        if isinstance(raw, str):
            try:
                return json.loads(raw)
            except json.JSONDecodeError:
                return raw
        return raw

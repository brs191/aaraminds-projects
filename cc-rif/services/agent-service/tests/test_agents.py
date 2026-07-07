from __future__ import annotations

import os

import pytest
from fastapi.testclient import TestClient

from agents import ArchitectureAgent
from app import create_app
from config import Settings


class MockMCPClient:
    def __init__(self) -> None:
        self.calls: list[str] = []

    async def call_tool(self, name: str, arguments: dict[str, object]) -> dict[str, object]:
        self.calls.append(name)
        if name == "search_code":
            return {
                "results": [
                    {
                        "source_ref": "apm0045942@sha:src/A.java:10",
                        "score": 1.0,
                        "confidence": "exact",
                    }
                ]
            }
        if name == "find_callers":
            return {
                "results": [
                    {
                        "caller_ref": "apm0045942@sha:src/B.java:20",
                        "call_site_ref": "apm0045942@sha:src/B.java:20",
                        "confidence": "exact",
                    }
                ]
            }
        if name == "dependency_analysis":
            return {
                "direct_deps": ["apm0045942@sha:src/C.java:30"],
                "transitive_deps": ["apm0045942@sha:src/D.java:40"],
                "depth_cap": 3,
            }
        if name == "impact_analysis":
            return {
                "impacted": [
                    {"source_ref": "apm0045942@sha:src/E.java:50", "tier": "static"},
                    {"source_ref": "apm0045942@sha:src/F.java:60", "tier": "cross-service"},
                ],
                "completeness_caveat": "Graph reachability is bounded.",
            }
        raise AssertionError(f"unexpected tool {name}")

    async def close(self) -> None:
        return None


@pytest.mark.asyncio
async def test_architecture_agent_calls_tools_in_order() -> None:
    mock = MockMCPClient()
    agent = ArchitectureAgent(mcp=mock, max_hops=3)
    explanation, refs = await agent.run("apm0045942", "CreditRoutingService")
    assert explanation
    assert refs
    assert refs[0].tool_name == "search_code"
    assert mock.calls == ["search_code", "find_callers", "dependency_analysis"]


def test_explain_response_contains_structured_citations() -> None:
    mock = MockMCPClient()
    app = create_app(
        settings=Settings.model_construct(
            mcp_server_url="http://mock.local/mcp",
            litellm_model="ollama/llama3.1:8b",
            max_hops=3,
            timeout_seconds=5.0,
        ),
        mcp_client=mock,
    )
    client = TestClient(app)
    resp = client.post("/explain", json={"repo_id": "apm0045942", "component": "CreditRoutingService"})
    assert resp.status_code == 200
    body = resp.json()
    assert len(body["source_refs"]) > 0
    assert all(item["tool_name"] for item in body["source_refs"])
    assert all(item["result_excerpt"] for item in body["source_refs"])


@pytest.mark.integration

def test_integration_explain_when_mcp_server_url_set() -> None:
    mcp_server_url = os.getenv("MCP_SERVER_URL")
    if not mcp_server_url:
        pytest.skip("MCP_SERVER_URL not set")

    app = create_app(
        settings=Settings.model_construct(
            mcp_server_url=mcp_server_url,
            litellm_model="ollama/llama3.1:8b",
            max_hops=3,
            timeout_seconds=10.0,
        )
    )
    client = TestClient(app)
    resp = client.post("/explain", json={"repo_id": "demo-repo", "component": "PaymentProcessor"})
    assert resp.status_code == 200
    assert len(resp.json()["source_refs"]) > 0

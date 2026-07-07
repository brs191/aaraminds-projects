from __future__ import annotations

import os
import signal
import subprocess
import time
from pathlib import Path

import httpx
from fastapi.testclient import TestClient

from app import create_app
from config import Settings

REPO_ROOT = Path(__file__).resolve().parents[3]
MCP_DIR = REPO_ROOT / "services" / "mcp-server"
MCP_BINARY = MCP_DIR / "mcp-server-e2e"


def _wait_for_health(url: str, timeout: float = 20.0) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            response = httpx.get(url, timeout=2.0)
            if response.status_code == 200:
                return
        except httpx.HTTPError:
            pass
        time.sleep(0.5)
    raise AssertionError(f"service at {url} did not become healthy")


def _ensure_mcp_binary() -> Path:
    if MCP_BINARY.exists():
        return MCP_BINARY
    subprocess.run(["go", "build", "-o", str(MCP_BINARY), "."], cwd=MCP_DIR, check=True)
    return MCP_BINARY


def test_agent_service_end_to_end_with_fixture_mcp() -> None:
    binary = _ensure_mcp_binary()
    addr = "127.0.0.1:18081"
    env = os.environ.copy()
    env["MCP_FIXTURE_MODE"] = "true"
    env["MCP_SERVER_ADDR"] = addr

    proc = subprocess.Popen([str(binary)], cwd=MCP_DIR, env=env)
    try:
        _wait_for_health(f"http://{addr}/health")

        app = create_app(
            settings=Settings.model_construct(
                mcp_server_url=f"http://{addr}/mcp",
                litellm_model="ollama/llama3.1:8b",
                max_hops=3,
                timeout_seconds=10.0,
            )
        )
        client = TestClient(app)

        explain = client.post("/explain", json={"repo_id": "demo-repo", "component": "PaymentProcessor"})
        assert explain.status_code == 200
        explain_body = explain.json()
        assert explain_body["explanation"]
        assert explain_body["source_refs"]
        assert explain_body["source_refs"][0]["tool_name"]

        impact = client.post(
            "/investigate_impact",
            json={"repo_id": "demo-repo", "changed_entity": "AmountValidator"},
        )
        assert impact.status_code == 200
        impact_body = impact.json()
        assert impact_body["narrative"]
        assert impact_body["tiers"]
        assert impact_body["source_refs"]
    finally:
        proc.send_signal(signal.SIGTERM)
        proc.wait(timeout=10)
        if MCP_BINARY.exists():
            MCP_BINARY.unlink()

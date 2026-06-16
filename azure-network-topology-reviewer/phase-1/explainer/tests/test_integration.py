"""Integration tests for POST /explain using EXPLAINER_MODE=stub.

These tests exercise the real FastAPI app and LangGraph orchestrator
end-to-end — no mocks.  They require zero Azure credentials because
EXPLAINER_MODE=stub wires StubLLMClient and StubRAGClient.

Run with:
    EXPLAINER_MODE=stub python -m pytest tests/test_integration.py -v
"""

from __future__ import annotations

import os

import pytest
from fastapi.testclient import TestClient

# Activate stub mode before importing the app
os.environ.setdefault("EXPLAINER_MODE", "stub")

from explainer.app import app  # noqa: E402  (env-var must be set first)


@pytest.fixture(scope="module")
def client():
    """TestClient that starts/stops the full FastAPI lifespan."""
    with TestClient(app, raise_server_exceptions=True) as c:
        yield c


# ---------------------------------------------------------------------------
# /health
# ---------------------------------------------------------------------------


class TestHealth:
    def test_health_ok(self, client: TestClient) -> None:
        resp = client.get("/health")
        assert resp.status_code == 200
        assert resp.json()["status"] == "ok"


# ---------------------------------------------------------------------------
# /explain — happy path
# ---------------------------------------------------------------------------


class TestExplainHappyPath:
    _PAYLOAD = {
        "subscription_id": "aaaabbbb-1111-2222-3333-ccccddddeeee",
        "findings": [
            {
                "type": "over-permissive NSG (reachable)",
                "severity": "High",
                "resource": "nic-vm-jumpbox",
                "evidence": "Any:22 inbound + route 0.0.0.0/0->Internet + public IP 1.2.3.4",
                "reachable": True,
            },
            {
                "type": "app gateway WAF disabled",
                "severity": "High",
                "resource": "appgw-prod",
                "evidence": "Application Gateway appgw-prod has public IP 5.6.7.8 but WAF is disabled",
                "reachable": True,
            },
            {
                "type": "private DNS zone missing",
                "severity": "Medium",
                "resource": "pe-storage-account",
                "evidence": "Private Endpoint pe-storage-account has no linked Private DNS Zone",
                "reachable": False,
            },
        ],
    }

    def test_returns_200(self, client: TestClient) -> None:
        resp = client.post("/explain", json=self._PAYLOAD)
        assert resp.status_code == 200

    def test_report_structure(self, client: TestClient) -> None:
        resp = client.post("/explain", json=self._PAYLOAD)
        body = resp.json()
        assert "subscription_id" in body
        assert "findings" in body
        assert "summary" in body
        assert "high_critical_count" in body
        assert "rag_grounded_pct" in body

    def test_subscription_id_echoed(self, client: TestClient) -> None:
        resp = client.post("/explain", json=self._PAYLOAD)
        assert resp.json()["subscription_id"] == self._PAYLOAD["subscription_id"]

    def test_all_findings_explained(self, client: TestClient) -> None:
        resp = client.post("/explain", json=self._PAYLOAD)
        for f in resp.json()["findings"]:
            assert f["explanation"] is not None, f"explanation missing for {f['type']}"
            assert len(f["explanation"]) > 20

    def test_all_findings_rag_grounded(self, client: TestClient) -> None:
        resp = client.post("/explain", json=self._PAYLOAD)
        for f in resp.json()["findings"]:
            assert f["rag_grounded"] is True, f"rag_grounded=False for {f['type']}"
            assert f["recommendation"] is not None
            assert "AT&T Network Standard" in f["recommendation"]

    def test_high_critical_count(self, client: TestClient) -> None:
        resp = client.post("/explain", json=self._PAYLOAD)
        # 2 High findings
        assert resp.json()["high_critical_count"] == 2

    def test_rag_grounded_pct_is_1(self, client: TestClient) -> None:
        resp = client.post("/explain", json=self._PAYLOAD)
        assert resp.json()["rag_grounded_pct"] == 1.0

    def test_summary_present(self, client: TestClient) -> None:
        resp = client.post("/explain", json=self._PAYLOAD)
        summary = resp.json()["summary"]
        assert summary is not None
        assert len(summary) > 20

    def test_no_error_field(self, client: TestClient) -> None:
        resp = client.post("/explain", json=self._PAYLOAD)
        assert resp.json()["error"] is None


# ---------------------------------------------------------------------------
# /explain — all 17 known finding types
# ---------------------------------------------------------------------------


_ALL_FINDING_TYPES = [
    ("over-permissive NSG (reachable)", "High"),
    ("over-permissive NSG (latent)", "Informational"),
    ("orphaned public endpoint", "Informational"),  # Low maps to Informational for Pydantic
    ("private DNS zone missing", "Medium"),
    ("private DNS zone not linked to VNet", "Medium"),
    ("app gateway WAF disabled", "High"),
    ("app gateway WAF in detection mode", "Medium"),
    ("AKS non-private cluster", "High"),
    ("cross-subscription peering without firewall", "High"),
    ("internet reachable via load balancer NAT", "High"),
    ("APIM without VNet isolation", "Medium"),
    ("APIM External mode without WAF", "High"),
    ("Bastion bypass — direct management port exposed", "High"),
    ("Front Door WAF disabled", "High"),
    ("Front Door WAF in detection mode", "Medium"),
    ("vWAN hub unsecured — no firewall", "High"),
    ("vWAN hub firewall bypasses private traffic", "High"),
]


class TestAllFindingTypes:
    def test_each_type_gets_explanation_and_recommendation(
        self, client: TestClient
    ) -> None:
        """Each of the 17 known finding types should return a non-null explanation
        and a RAG-grounded recommendation from the stub."""
        for finding_type, severity in _ALL_FINDING_TYPES:
            payload = {
                "subscription_id": "test-sub",
                "findings": [
                    {
                        "type": finding_type,
                        "severity": severity,
                        "resource": "test-resource",
                        "evidence": "test evidence",
                        "reachable": severity in ("Critical", "High"),
                    }
                ],
            }
            resp = client.post("/explain", json=payload)
            assert resp.status_code == 200, f"HTTP error for type={finding_type!r}"
            finding = resp.json()["findings"][0]
            assert finding["explanation"] is not None, f"No explanation for {finding_type!r}"
            assert finding["recommendation"] is not None, f"No recommendation for {finding_type!r}"
            assert finding["rag_grounded"] is True, f"rag_grounded=False for {finding_type!r}"


# ---------------------------------------------------------------------------
# /explain — edge cases
# ---------------------------------------------------------------------------


class TestExplainEdgeCases:
    def test_empty_findings_returns_200(self, client: TestClient) -> None:
        payload = {
            "subscription_id": "empty-sub",
            "findings": [],
        }
        resp = client.post("/explain", json=payload)
        assert resp.status_code == 200
        body = resp.json()
        assert body["findings"] == []
        assert body["high_critical_count"] == 0
        assert body["rag_grounded_pct"] == 0.0

    def test_critical_finding_counted(self, client: TestClient) -> None:
        payload = {
            "subscription_id": "crit-sub",
            "findings": [
                {
                    "type": "over-permissive NSG (reachable)",
                    "severity": "Critical",
                    "resource": "nic-sensitive",
                    "evidence": "sensitive tag + public IP",
                    "reachable": True,
                }
            ],
        }
        resp = client.post("/explain", json=payload)
        assert resp.status_code == 200
        assert resp.json()["high_critical_count"] == 1

    def test_unknown_finding_type_gets_default_explanation(
        self, client: TestClient
    ) -> None:
        payload = {
            "subscription_id": "unknown-sub",
            "findings": [
                {
                    "type": "some-future-finding-type",
                    "severity": "Medium",
                    "resource": "some-resource",
                    "evidence": "unknown evidence",
                    "reachable": False,
                }
            ],
        }
        resp = client.post("/explain", json=payload)
        assert resp.status_code == 200
        finding = resp.json()["findings"][0]
        # Stub returns default explanation for unknown types
        assert finding["explanation"] is not None
        assert finding["recommendation"] is not None

    def test_invalid_severity_rejected(self, client: TestClient) -> None:
        payload = {
            "subscription_id": "bad-sub",
            "findings": [
                {
                    "type": "over-permissive NSG (reachable)",
                    "severity": "INVALID",
                    "resource": "nic",
                    "evidence": "...",
                    "reachable": True,
                }
            ],
        }
        resp = client.post("/explain", json=payload)
        assert resp.status_code == 422  # Pydantic validation error

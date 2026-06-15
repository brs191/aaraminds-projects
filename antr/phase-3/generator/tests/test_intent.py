"""Tests for the Phase 3 intent capture layer.

Run from ``phase-3/generator/``:
    pytest tests/test_intent.py -v

Test matrix
-----------
1. test_stub_returns_valid_spec          — GENERATOR_MODE=stub path
2. test_invalid_nsg_intent_raises        — closed-vocabulary enforcement
3. test_token_not_in_logs                — bearer-token redaction filter
4. test_max_iterations_hard_fail         — exactly 1 LLM call when max=1
5. test_refinement_prompt_injected       — §4.3 refinement block on iteration 2
"""

from __future__ import annotations

import json
import logging
import os
from typing import Any
from unittest.mock import patch

import httpx
import pytest
import respx
from pydantic import ValidationError

# ---------------------------------------------------------------------------
# Import helpers
# ---------------------------------------------------------------------------

from generator.exceptions import LLMClientError, SpecValidationError
from generator.intent import (
    AskATTClient,
    _BearerTokenRedactFilter,
    _SYSTEM_PROMPT,
    _stub_spec,
    generate_spec,
)
from generator.models import SubnetSpec, TopologySpec

# ---------------------------------------------------------------------------
# Shared fixtures and helpers
# ---------------------------------------------------------------------------

#: Minimal valid TopologySpec JSON that passes Pydantic validation.
#: Used as the mock LLM response in live-mode tests.
_VALID_SPEC_DICT: dict[str, Any] = {
    "specVersion": "1.0",
    "description": "Minimal valid spec for unit testing purposes only.",
    "region": "eastus2",
    "peeringTopology": "none",
    "gatewayType": "none",
    "firewallEnabled": False,
    "avnmEnabled": False,
    "tierLabels": ["web"],
    "tags": {
        "env": "dev",
        "owner": "test-team",
        "costcenter": "TEST-001",
        "appid": "TEST-APP",
    },
    "vnets": [
        {
            "name": "vnet-test",
            "addressSpace": ["10.99.0.0/16"],
            "subnets": [
                {
                    "name": "snet-web",
                    "addressPrefix": "10.99.1.0/24",
                    "tierLabel": "web",
                    "sensitive": False,
                    "nsgIntents": ["allow-https-from-internet", "deny-all-inbound-other"],
                }
            ],
        }
    ],
}

_VALID_SPEC_JSON: str = json.dumps(_VALID_SPEC_DICT)

#: LLM response envelope wrapping the spec JSON (mirrors OpenAI-compatible API).
def _llm_response(content: str) -> dict[str, Any]:
    return {
        "choices": [
            {
                "message": {
                    "role": "assistant",
                    "content": content,
                }
            }
        ]
    }


#: Env vars required by AskATTClient in live mode.
_LIVE_ENV: dict[str, str] = {
    "ASKAT_ENDPOINT": "https://askatt.example.att.com",
    "ASKAT_TOKEN_URL": "https://login.att.com/oauth2/token",
    "ASKAT_CLIENT_ID": "test-client-id",
    "ASKAT_CLIENT_SECRET": "test-client-secret-NEVER-LOGGED",
}

#: Fake bearer token used in token-not-in-logs test.
_FAKE_TOKEN = "FAKE_TOKEN_12345"  # noqa: S105 — test constant, not a real secret


# ---------------------------------------------------------------------------
# Test 1 — GENERATOR_MODE=stub returns a valid spec
# ---------------------------------------------------------------------------


def test_stub_returns_valid_spec(monkeypatch: pytest.MonkeyPatch) -> None:
    """``GENERATOR_MODE=stub`` must return a Pydantic-validated TopologySpec
    with ``spec_version == "1.0"`` and no network calls."""
    monkeypatch.setenv("GENERATOR_MODE", "stub")

    import asyncio

    spec = asyncio.get_event_loop().run_until_complete(
        generate_spec(
            intent="Create a hub-spoke network for BCLM.",
            subscription_context={},
            max_iterations=2,
            failing_findings=[],
        )
    )

    assert isinstance(spec, TopologySpec)
    assert spec.spec_version == "1.0"
    # Pydantic validation has already run (would have raised on bad data);
    # assert a few structural invariants from the §1.5 example.
    assert len(spec.vnets) >= 1
    assert spec.peering_topology == "hub-spoke"
    assert spec.firewall_enabled is True
    # Mandatory tags are present
    for key in ("env", "owner", "costcenter", "appid"):
        assert key in spec.tags, f"Mandatory tag '{key}' missing from stub spec"
    # All nsgIntents in every subnet are in the approved vocabulary
    from generator.models import VALID_NSG_INTENTS
    for vnet in spec.vnets:
        for subnet in vnet.subnets:
            for intent in subnet.nsg_intents:
                assert intent in VALID_NSG_INTENTS, (
                    f"Stub spec contains out-of-vocabulary intent {intent!r} "
                    f"in subnet {subnet.name!r}"
                )


# ---------------------------------------------------------------------------
# Test 2 — Invalid NSG intent raises ValueError / ValidationError
# ---------------------------------------------------------------------------


def test_invalid_nsg_intent_raises() -> None:
    """Creating a ``SubnetSpec`` with an out-of-vocabulary nsgIntent must raise
    ``ValueError`` (surfaced as ``pydantic.ValidationError``)."""
    with pytest.raises((ValueError, ValidationError)):
        SubnetSpec(
            name="snet-bad",
            addressPrefix="10.0.0.0/24",
            tierLabel="web",
            sensitive=False,
            nsgIntents=["allow-everything"],  # NOT in the approved vocabulary
        )


def test_invalid_nsg_intent_multiple_bad_raises() -> None:
    """Even one bad intent in a list must raise."""
    with pytest.raises((ValueError, ValidationError)):
        SubnetSpec(
            name="snet-mixed",
            addressPrefix="10.0.1.0/24",
            tierLabel="app",
            sensitive=False,
            # First intent is valid, second is not — validator must catch it.
            nsgIntents=["allow-internal-vnet", "permit-all"],
        )


def test_valid_nsg_intents_accepted() -> None:
    """A SubnetSpec with only approved intents must not raise."""
    subnet = SubnetSpec(
        name="snet-ok",
        addressPrefix="10.0.2.0/24",
        tierLabel="data",
        sensitive=True,
        nsgIntents=["deny-internet-inbound", "deny-all-inbound-other"],
    )
    assert subnet.nsg_intents == ["deny-internet-inbound", "deny-all-inbound-other"]


# ---------------------------------------------------------------------------
# Test 3 — Bearer token never appears in captured log records
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_token_not_in_logs(
    monkeypatch: pytest.MonkeyPatch,
    caplog: pytest.LogCaptureFixture,
) -> None:
    """Instantiate ``AskATTClient``, mock the token endpoint to return a fake
    bearer token, call ``complete``, and assert the raw token string never
    appears in any captured log record.

    This verifies both that our code does not log the token directly *and* that
    the ``_BearerTokenRedactFilter`` provides defence-in-depth against any
    accidental leak via the httpx logger.
    """
    for key, val in _LIVE_ENV.items():
        monkeypatch.setenv(key, val)

    # Override client secret with our fake token scenario
    monkeypatch.setenv("ASKAT_CLIENT_SECRET", "client-secret-value")

    with caplog.at_level(logging.DEBUG):
        with respx.mock(assert_all_called=False) as mock_router:
            # Token endpoint — returns the fake bearer token.
            mock_router.post(_LIVE_ENV["ASKAT_TOKEN_URL"]).mock(
                return_value=httpx.Response(
                    200,
                    json={
                        "access_token": _FAKE_TOKEN,
                        "token_type": "Bearer",
                        "expires_in": 3600,
                    },
                )
            )
            # Completion endpoint — returns a valid spec.
            mock_router.post(
                f"{_LIVE_ENV['ASKAT_ENDPOINT']}/chat/completions"
            ).mock(
                return_value=httpx.Response(
                    200,
                    json=_llm_response(_VALID_SPEC_JSON),
                )
            )

            client = AskATTClient()
            await client.complete(
                messages=[
                    {"role": "system", "content": _SYSTEM_PROMPT},
                    {"role": "user", "content": "Test intent."},
                ]
            )

    # Assert: the raw fake token must not appear in ANY captured log record.
    all_log_text = "\n".join(
        record.getMessage() for record in caplog.records
    )
    assert _FAKE_TOKEN not in all_log_text, (
        f"Bearer token {_FAKE_TOKEN!r} leaked into log records.\n"
        f"Captured log output:\n{all_log_text}"
    )


# ---------------------------------------------------------------------------
# Test 4 — max_iterations=1 results in exactly 1 LLM call
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_max_iterations_hard_fail(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """With ``max_iterations=1`` and a mock LLM that returns a structurally
    valid ``TopologySpec`` (which would fail the security gate), ``generate_spec``
    must return after exactly 1 LLM call.

    The gate check is the caller's responsibility (MCP handler); ``generate_spec``
    returns the spec regardless of security posture.
    """
    for key, val in _LIVE_ENV.items():
        monkeypatch.setenv(key, val)

    monkeypatch.delenv("GENERATOR_MODE", raising=False)

    with respx.mock(assert_all_called=True) as mock_router:
        mock_router.post(_LIVE_ENV["ASKAT_TOKEN_URL"]).mock(
            return_value=httpx.Response(
                200,
                json={
                    "access_token": "token-max-iter-test",
                    "token_type": "Bearer",
                    "expires_in": 3600,
                },
            )
        )
        # Single completion route — we will assert it is called exactly once.
        completion_route = mock_router.post(
            f"{_LIVE_ENV['ASKAT_ENDPOINT']}/chat/completions"
        ).mock(
            return_value=httpx.Response(
                200,
                # Valid spec JSON — passes Pydantic, would fail security gate
                # (no deny-internet-inbound on sensitive subnet), but that is
                # the caller's concern; generate_spec returns it as-is.
                json=_llm_response(_VALID_SPEC_JSON),
            )
        )

        spec = await generate_spec(
            intent="Create a simple web VNet.",
            subscription_context={},
            max_iterations=1,
            failing_findings=[],
        )

    assert isinstance(spec, TopologySpec)
    assert completion_route.call_count == 1, (
        f"Expected exactly 1 LLM call with max_iterations=1, "
        f"got {completion_route.call_count}"
    )


# ---------------------------------------------------------------------------
# Test 5 — Refinement block injected on iteration 2
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_refinement_prompt_injected(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """On iteration 2, the request body sent to AskAT&T MUST contain the
    §4.3 refinement block text ``'VALIDATION FAILURE'``.

    Scenario:
    - Iteration 1: LLM returns unparseable content → Pydantic parse fails.
    - Iteration 2: LLM returns valid spec → parse succeeds.
    - We assert the iteration-2 request body contains 'VALIDATION FAILURE'.
    """
    for key, val in _LIVE_ENV.items():
        monkeypatch.setenv(key, val)

    monkeypatch.delenv("GENERATOR_MODE", raising=False)

    # Blocking findings supplied by the caller (simulating a previous
    # ValidateBeforeEmit failure reported by the MCP handler).
    sample_findings: list[dict[str, Any]] = [
        {
            "type": "over-permissive NSG (reachable)",
            "severity": "Critical",
            "resource": "snet-data",
            "detail": "Internet-reachable path to sensitive NIC detected.",
        }
    ]

    # Capture all request bodies to the completion endpoint.
    captured_request_bodies: list[bytes] = []

    def _completion_handler(request: httpx.Request, **_: Any) -> httpx.Response:
        captured_request_bodies.append(request.content)
        call_index = len(captured_request_bodies)
        if call_index == 1:
            # Iteration 1: return invalid JSON so Pydantic parse fails.
            return httpx.Response(
                200,
                json=_llm_response("this is not valid TopologySpec JSON"),
            )
        # Iteration 2: return a valid spec.
        return httpx.Response(
            200,
            json=_llm_response(_VALID_SPEC_JSON),
        )

    with respx.mock(assert_all_called=False) as mock_router:
        mock_router.post(_LIVE_ENV["ASKAT_TOKEN_URL"]).mock(
            return_value=httpx.Response(
                200,
                json={
                    "access_token": "token-refinement-test",
                    "token_type": "Bearer",
                    "expires_in": 3600,
                },
            )
        )
        mock_router.post(
            f"{_LIVE_ENV['ASKAT_ENDPOINT']}/chat/completions"
        ).mock(side_effect=_completion_handler)

        spec = await generate_spec(
            intent="Create a secure data VNet with sensitive subnet.",
            subscription_context={"avnmEnabled": False},
            max_iterations=2,
            failing_findings=sample_findings,
        )

    # Two LLM calls must have been made.
    assert len(captured_request_bodies) == 2, (
        f"Expected 2 LLM calls (iteration 1 fail + iteration 2 retry), "
        f"got {len(captured_request_bodies)}"
    )

    # The iteration-2 request body must contain the §4.3 refinement marker.
    second_body_text = captured_request_bodies[1].decode("utf-8")
    assert "VALIDATION FAILURE" in second_body_text, (
        "Refinement block text 'VALIDATION FAILURE' was not found in the "
        "iteration-2 request body.\n"
        f"Body snippet: {second_body_text[:500]!r}"
    )

    # The final returned spec must be valid.
    assert isinstance(spec, TopologySpec)
    assert spec.spec_version == "1.0"


# ---------------------------------------------------------------------------
# Additional edge-case tests (bonus — beyond the required 5)
# ---------------------------------------------------------------------------


def test_max_iterations_clamped_below() -> None:
    """max_iterations below 1 is clamped to 1 (not raised)."""
    import asyncio

    os.environ["GENERATOR_MODE"] = "stub"
    try:
        # Should not raise even though 0 is below the valid range.
        spec = asyncio.get_event_loop().run_until_complete(
            generate_spec(
                intent="Create a minimal hub VNet for BCLM.",
                subscription_context={},
                max_iterations=0,  # should be clamped to 1
                failing_findings=[],
            )
        )
        assert spec.spec_version == "1.0"
    finally:
        del os.environ["GENERATOR_MODE"]


def test_max_iterations_clamped_above() -> None:
    """max_iterations above 3 is clamped to 3 (not raised)."""
    import asyncio

    os.environ["GENERATOR_MODE"] = "stub"
    try:
        spec = asyncio.get_event_loop().run_until_complete(
            generate_spec(
                intent="Create a minimal hub VNet for BCLM.",
                subscription_context={},
                max_iterations=99,  # should be clamped to 3
                failing_findings=[],
            )
        )
        assert spec.spec_version == "1.0"
    finally:
        del os.environ["GENERATOR_MODE"]


def test_mandatory_tags_missing_raises() -> None:
    """TopologySpec without mandatory AT&T tags must fail validation."""
    with pytest.raises((ValueError, ValidationError)):
        TopologySpec(
            specVersion="1.0",
            description="Missing mandatory tags test spec.",
            region="eastus2",
            peeringTopology="none",
            gatewayType="none",
            firewallEnabled=False,
            avnmEnabled=False,
            tierLabels=["web"],
            tags={
                "env": "dev",
                # Missing: owner, costcenter, appid
            },
            vnets=[
                {  # type: ignore[arg-type]
                    "name": "vnet-test",
                    "addressSpace": ["10.0.0.0/16"],
                    "subnets": [
                        {
                            "name": "snet-a",
                            "addressPrefix": "10.0.1.0/24",
                            "tierLabel": "web",
                            "sensitive": False,
                            "nsgIntents": [],
                        }
                    ],
                }
            ],
        )


def test_redact_filter_strips_bearer_token() -> None:
    """``_BearerTokenRedactFilter`` must replace 'Bearer <value>' with
    'Bearer [REDACTED]' in the log record message."""
    filt = _BearerTokenRedactFilter()
    record = logging.LogRecord(
        name="generator.intent",
        level=logging.DEBUG,
        pathname=__file__,
        lineno=0,
        msg="Authorization: Bearer SUPERSECRETTOKEN123",
        args=(),
        exc_info=None,
    )
    result = filt.filter(record)
    assert result is True  # record is allowed through (not dropped)
    assert "SUPERSECRETTOKEN123" not in record.msg
    assert "[REDACTED]" in record.msg


@pytest.mark.asyncio
async def test_rate_limit_retry(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """AskATTClient must retry on HTTP 429 and succeed on the second attempt."""
    for key, val in _LIVE_ENV.items():
        monkeypatch.setenv(key, val)

    call_count = 0

    def _handler(request: httpx.Request, **_: Any) -> httpx.Response:
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            return httpx.Response(429, json={"error": "rate limit"})
        return httpx.Response(200, json=_llm_response(_VALID_SPEC_JSON))

    with respx.mock(assert_all_called=False) as mock_router:
        mock_router.post(_LIVE_ENV["ASKAT_TOKEN_URL"]).mock(
            return_value=httpx.Response(
                200,
                json={"access_token": "token-retry-test", "expires_in": 3600},
            )
        )
        mock_router.post(
            f"{_LIVE_ENV['ASKAT_ENDPOINT']}/chat/completions"
        ).mock(side_effect=_handler)

        client = AskATTClient()
        # Patch asyncio.sleep to avoid real delays in tests.
        with patch("generator.intent.asyncio.sleep", return_value=None):
            content = await client.complete(
                messages=[{"role": "user", "content": "test"}]
            )

    assert content == _VALID_SPEC_JSON
    assert call_count == 2

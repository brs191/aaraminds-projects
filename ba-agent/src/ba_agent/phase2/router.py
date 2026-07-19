"""Phase 2 route handler.

Accepts a prompt or structured input and returns a ``Phase2RouteDecision``
indicating ``phase2_requirement_discovery`` or a blocked response.

Entry-point guards (checked before any routing logic):
  1. ``BA_AGENT_DATA_SOURCE_MODE`` must equal ``"synthetic"``; any other value
     raises ``ConfigurationError``.
  2. ``LIVE_INTEGRATIONS_ENABLED`` must equal ``"false"``; any truthy value
     raises ``ConfigurationError``.

These guards implement the synthetic-only discipline defined in
``docs/development/p2-g1-technical-baseline.md`` Section 7.2 and are verified
by ``tests/phase2/test_separation.py`` no-live guard test (Section 8, item 3).

Route isolation constraints (BA-EM-009 = 0):
  - This router must NOT call or import from ``ba_agent.router``.
  - It must NOT return any MVP ``Route`` enum value.
  - It must NOT modify any MVP route or orchestration state.

Scope: P2-G1 scaffold stub — guards and routing skeleton only; no LLM calls.
Authorization: Synthetic-only; no live clients; no system-of-record writes.
"""
from __future__ import annotations

from ba_agent.phase2.models import Phase2RouteDecision


class ConfigurationError(Exception):
    """Raised when Phase 2 runtime configuration violates synthetic-only guards."""


def check_phase2_guards() -> None:
    """Validate ``BA_AGENT_DATA_SOURCE_MODE`` and ``LIVE_INTEGRATIONS_ENABLED``.

    Raises ``ConfigurationError`` if either guard condition is violated.
    Must be called before any Phase 2 routing or capability logic.

    Rules (Section 7.2):
      - ``BA_AGENT_DATA_SOURCE_MODE`` must equal ``"synthetic"``; any other
        value (including unset) raises ``ConfigurationError``.
      - ``LIVE_INTEGRATIONS_ENABLED`` must equal ``"false"``; any truthy
        value raises ``ConfigurationError``.
    """
    import os

    data_source_mode = os.environ.get("BA_AGENT_DATA_SOURCE_MODE", "")
    if data_source_mode != "synthetic":
        raise ConfigurationError(
            f"BA_AGENT_DATA_SOURCE_MODE must be 'synthetic' for Phase 2 first-slice "
            f"paths; got '{data_source_mode}'. "
            "Non-synthetic data use is blocked until P2-G4 approval (P2-DEC-010)."
        )

    live_enabled = os.environ.get("LIVE_INTEGRATIONS_ENABLED", "false").lower()
    if live_enabled not in ("false", "0", "no", ""):
        raise ConfigurationError(
            f"LIVE_INTEGRATIONS_ENABLED must be 'false' for Phase 2 first-slice paths; "
            f"got '{os.environ.get('LIVE_INTEGRATIONS_ENABLED')}'. "
            "Live integrations are blocked until P2-G4 tool-approval evidence exists (P2-DEC-009)."
        )


def route(prompt: str) -> Phase2RouteDecision:
    """Route *prompt* to the appropriate Phase 2 decision.

    Returns a ``Phase2RouteDecision`` with route ``"phase2_requirement_discovery"``
    for valid discovery inputs, or route ``"blocked"`` otherwise.

    Calls ``check_phase2_guards()`` as the first action; raises
    ``ConfigurationError`` if guards fail.

    Minimal P2-G2 routing logic is synthetic-only and advisory only.
    """
    check_phase2_guards()

    normalized_prompt = prompt.lower()
    if "phase2_requirement_discovery" in normalized_prompt or "requirement discovery" in normalized_prompt or "p2req" in normalized_prompt:
        return Phase2RouteDecision(
            route="phase2_requirement_discovery",
            reason="Synthetic requirement-discovery prompt accepted for P2-G2 thin slice.",
            blocked=False,
        )

    return Phase2RouteDecision(
        route="blocked",
        reason="Prompt did not match the synthetic requirement-discovery route.",
        blocked=True,
    )

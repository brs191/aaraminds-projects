"""Phase 2 separation, no-live, and no-network guard tests.

P2-G1 test stubs — enforces the hard constraints that must hold before P2-G2
can begin.

Test coverage (from ``docs/development/p2-g1-technical-baseline.md`` Section 8):
  2. MVP route isolation — Phase 2 input does NOT reach any MVP route.
  3. No-live guard — router raises ConfigurationError when live guards fail.
  5. No-network guard — Phase 2 imports produce no socket-level network calls
     (covered by the autouse ``block_network`` fixture in ``tests/conftest.py``).

BA-EM-009 = 0: any route leakage to MVP paths is a hard-gate breach.
BA-EM-005 = 0: any write-like side effect without approval_ref is a hard-gate breach.

Synthetic-only. No live integrations.
The ``block_network`` fixture in ``tests/conftest.py`` is autouse and covers
these tests (Section 8, item 5 — no-network guard).

Authorization: Synthetic-only; BA-EM-005 = 0; BA-EM-009 = 0.
"""
from __future__ import annotations

import importlib

import pytest


# ---------------------------------------------------------------------------
# Test 2 — MVP route isolation (Section 8, item 2; BA-EM-009)
# ---------------------------------------------------------------------------

MVP_CAPABILITY_MODULES = [
    "ba_agent.standup",
    "ba_agent.mvp",
    "ba_agent.cards",
    "ba_agent.adapters",
]

PHASE2_MODULES = [
    "ba_agent.phase2",
    "ba_agent.phase2.router",
    "ba_agent.phase2.discovery",
    "ba_agent.phase2.models",
    "ba_agent.phase2.sandbox_mcp",
    "ba_agent.phase2.context_memory",
    "ba_agent.phase2.traceability",
]


@pytest.mark.parametrize("phase2_module", PHASE2_MODULES)
@pytest.mark.parametrize("mvp_module", MVP_CAPABILITY_MODULES)
def test_phase2_does_not_import_mvp_capability(phase2_module: str, mvp_module: str) -> None:
    """Phase 2 modules must NOT import from MVP capability modules.

    Violation is a BA-EM-009 hard-gate breach.
    """
    mod = importlib.import_module(phase2_module)
    imported_names = set(vars(mod).keys())

    # Check the actual module's __dict__ for any reference to the MVP module
    # by inspecting its __spec__ or direct attribute presence.
    mvp_short = mvp_module.split(".")[-1]  # e.g. "standup", "mvp", "cards", "adapters"
    assert mvp_short not in imported_names or _is_not_mvp_import(vars(mod).get(mvp_short)), (
        f"BA-EM-009 VIOLATION: {phase2_module!r} imports MVP capability module "
        f"{mvp_module!r}. Phase 2 modules must be fully isolated from MVP code."
    )


def _is_not_mvp_import(obj: object) -> bool:
    """Return True if *obj* is not an imported MVP module reference."""
    if obj is None:
        return True
    module_name = getattr(obj, "__name__", None) or getattr(
        getattr(obj, "__module__", None), "__name__", None
    )
    if module_name is None:
        return True
    return not any(module_name.startswith(m) for m in ["ba_agent.standup", "ba_agent.mvp",
                                                         "ba_agent.cards", "ba_agent.adapters"])


def test_mvp_router_does_not_import_phase2() -> None:
    """The MVP router (ba_agent.router) must NOT import from ba_agent.phase2.

    Violation is a BA-EM-009 hard-gate breach.
    """
    mvp_router = importlib.import_module("ba_agent.router")
    for attr_name, attr_val in vars(mvp_router).items():
        module = getattr(attr_val, "__module__", "") or ""
        assert not module.startswith("ba_agent.phase2"), (
            f"BA-EM-009 VIOLATION: ba_agent.router attribute {attr_name!r} "
            f"references ba_agent.phase2 ({module!r}). MVP router must not expose "
            "Phase 2 runtime behavior."
        )


def test_phase2_route_name_not_in_mvp_route_enum() -> None:
    """The Phase 2 route name 'phase2_requirement_discovery' must NOT appear
    as a value in the MVP Route enum.

    The MVP Route enum contains Route.PHASE2_BLOCKED as a guard, not an
    execution route. The execution route lives only in Phase 2 modules.
    """
    from ba_agent.models import Route

    mvp_route_values = {r.value for r in Route}
    assert "phase2_requirement_discovery" not in mvp_route_values, (
        "BA-EM-009 VIOLATION: 'phase2_requirement_discovery' found in MVP Route enum. "
        "Phase 2 execution route must live only in ba_agent.phase2.models."
    )


# ---------------------------------------------------------------------------
# Test 3 — No-live guard (Section 8, item 3)
# ---------------------------------------------------------------------------

def test_live_integrations_enabled_true_raises_config_error(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Instantiating Phase 2 router with LIVE_INTEGRATIONS_ENABLED=true must
    raise ConfigurationError.

    Source: Section 7.2, Section 8 item 3.
    """
    monkeypatch.setenv("LIVE_INTEGRATIONS_ENABLED", "true")
    monkeypatch.setenv("BA_AGENT_DATA_SOURCE_MODE", "synthetic")

    from ba_agent.phase2.router import check_phase2_guards, ConfigurationError

    with pytest.raises(ConfigurationError, match="LIVE_INTEGRATIONS_ENABLED"):
        check_phase2_guards()


def test_data_source_mode_not_synthetic_raises_config_error(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Phase 2 router must raise ConfigurationError when
    BA_AGENT_DATA_SOURCE_MODE is not 'synthetic'.

    Source: Section 7.2, Section 8 item 3.
    """
    monkeypatch.setenv("LIVE_INTEGRATIONS_ENABLED", "false")
    monkeypatch.setenv("BA_AGENT_DATA_SOURCE_MODE", "live")

    from ba_agent.phase2.router import check_phase2_guards, ConfigurationError

    with pytest.raises(ConfigurationError, match="BA_AGENT_DATA_SOURCE_MODE"):
        check_phase2_guards()


def test_data_source_mode_missing_raises_config_error(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Phase 2 router must raise ConfigurationError when
    BA_AGENT_DATA_SOURCE_MODE is unset (not 'synthetic').

    Source: Section 7.2, Section 8 item 3.
    """
    monkeypatch.delenv("BA_AGENT_DATA_SOURCE_MODE", raising=False)
    monkeypatch.setenv("LIVE_INTEGRATIONS_ENABLED", "false")

    from ba_agent.phase2.router import check_phase2_guards, ConfigurationError

    with pytest.raises(ConfigurationError, match="BA_AGENT_DATA_SOURCE_MODE"):
        check_phase2_guards()


# ---------------------------------------------------------------------------
# Test 5 — No-network guard (Section 8, item 5)
# ---------------------------------------------------------------------------
# The autouse block_network fixture in tests/conftest.py blocks all socket
# calls for all tests in this file. The tests above implicitly verify that
# Phase 2 imports + light execution produce no network calls (any attempt
# would raise AssertionError("Network access is blocked in Phase 1 tests")).
#
# Explicit evidence test:

def test_phase2_import_produces_no_network_call() -> None:
    """Importing all Phase 2 modules must not trigger any socket-level call.

    Covered by the autouse block_network fixture. This test makes the
    coverage explicit for audit purposes (Section 8, item 5).
    """
    import importlib

    for mod_name in PHASE2_MODULES:
        importlib.import_module(mod_name)
    # If we reach here without an AssertionError from block_network, the guard holds.

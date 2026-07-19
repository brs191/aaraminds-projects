from __future__ import annotations

from ba_agent.fixtures import load_fixture_set
from ba_agent.models import Route
from ba_agent.router import route_prompt


def test_router_matches_fixture_expected_routes() -> None:
    fixture_set = load_fixture_set()

    for case in fixture_set.cases:
        assert route_prompt(case.prompt).route == case.expected_route


def test_phase2_request_is_blocked() -> None:
    decision = route_prompt("Create a BRD and acceptance criteria for onboarding")

    assert decision.route == Route.PHASE2_BLOCKED
    assert decision.blocked is True


def test_mixed_standup_and_approval_request_blocks_write_intent() -> None:
    decision = route_prompt("Summarize standup and also approve the sprint plan")

    assert decision.route == Route.STANDUP
    assert decision.blocked is True
    assert "write/approval intent" in decision.reason


def test_unsupported_request_is_not_guessed_into_standup() -> None:
    decision = route_prompt("Book a meeting tomorrow")

    assert decision.route == Route.UNSUPPORTED
    assert decision.blocked is True

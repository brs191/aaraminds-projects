from __future__ import annotations

from ba_agent.models import (
    AdaptiveCardPayload,
    EvalCase,
    FixtureRecord,
    GatewayRequest,
    GatewayResponse,
    GraphState,
    Route,
    RouteDecision,
    ToolStatus,
)


def test_required_boundary_models_construct() -> None:
    RouteDecision(route=Route.STANDUP, reason="standup request")
    GraphState(trace_id="trace-test", graph_version="phase1")
    GatewayRequest(trace_id="trace-test", tool_name="jira", action="get_sprint_status")
    GatewayResponse(
        trace_id="trace-test",
        tool_name="jira",
        action="get_sprint_status",
        status=ToolStatus.DEGRADED,
        message="local placeholder",
    )
    FixtureRecord(
        fixture_id="STD-001",
        source_system="jira",
        evidence_ref="jira:synthetic:BA/BA-1",
        source_timestamp="2026-07-03T00:00:00Z",
        retrieved_at="2026-07-03T00:00:01Z",
    )
    EvalCase(case_id="STD-001", prompt="standup", expected_route=Route.STANDUP)
    AdaptiveCardPayload(title="Standup", trace_id="trace-test")

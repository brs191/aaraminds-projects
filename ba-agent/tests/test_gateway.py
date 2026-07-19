from __future__ import annotations

import pytest

from ba_agent.gateway import (
    InMemoryApprovalStore,
    InMemoryAuditSink,
    LocalGatewayFake,
    evaluate_phase2_sandbox_upstream_tool,
)
from ba_agent.models import ApprovalRecord, GatewayRequest, ToolStatus


def test_gateway_write_like_actions_fail_closed() -> None:
    response = LocalGatewayFake().execute(
        GatewayRequest(
            trace_id="trace-test",
            tool_name="jira",
            action="update_sprint_scope",
            evidence_refs=["eval:GAT-001"],
            write_like=True,
            idempotency_key="idem-1",
        )
    )

    assert response.status == ToolStatus.REJECTED
    assert "approval_ref" in response.message
    assert response.audit_record is not None
    assert response.audit_record.result_status == ToolStatus.REJECTED


def test_gateway_unvalidated_actions_are_blocked() -> None:
    response = LocalGatewayFake().execute(
        GatewayRequest(
            trace_id="trace-test",
            tool_name="unknown",
            action="unknown_action",
        )
    )

    assert response.status == ToolStatus.BLOCKED


def test_gateway_allows_standup_synthetic_reads() -> None:
    response = LocalGatewayFake().execute(
        GatewayRequest(
            trace_id="trace-test",
            tool_name="jira",
            action="get_sprint_status",
            evidence_refs=["tool:synthetic:jira/STD-001"],
        )
    )

    assert response.status == ToolStatus.OK
    assert response.audit_record is not None
    assert response.audit_record.input_hash.startswith("sha256:")


def test_gateway_blocks_cross_capability_tools() -> None:
    response = LocalGatewayFake().execute(
        GatewayRequest(
            trace_id="trace-test",
            tool_name="jira",
            action="get_backlog",
            capability="standup",
        )
    )

    assert response.status == ToolStatus.BLOCKED
    assert "not allowlisted" in response.message


def test_gateway_surfaces_denied_degraded_and_throttled_statuses() -> None:
    gateway = LocalGatewayFake()

    degraded = gateway.execute(
        GatewayRequest(trace_id="trace-test", tool_name="git", action="get_recent_activity", evidence_refs=["degraded"])
    )
    denied = gateway.execute(
        GatewayRequest(trace_id="trace-test", tool_name="jira", action="get_sprint_status", evidence_refs=["denied"])
    )
    throttled = gateway.execute(
        GatewayRequest(trace_id="trace-test", tool_name="git", action="get_recent_activity", evidence_refs=["throttled"])
    )

    assert degraded.status == ToolStatus.DEGRADED
    assert denied.status == ToolStatus.DENIED
    assert throttled.status == ToolStatus.THROTTLED


def test_gateway_rejects_duplicate_idempotency_key() -> None:
    gateway = LocalGatewayFake()
    request = GatewayRequest(
        trace_id="trace-test",
        tool_name="teams",
        action="send_adaptive_card",
        write_like=True,
        idempotency_key="idem-dup",
    )

    first = gateway.execute(request)
    second = gateway.execute(request)

    assert first.status == ToolStatus.REJECTED
    assert second.status == ToolStatus.REJECTED
    assert "duplicate" in second.message


def test_gateway_request_approval_creates_pending_without_approval_ref() -> None:
    response = LocalGatewayFake().execute(
        GatewayRequest(
            trace_id="trace-test",
            tool_name="approval",
            action="request_approval",
            write_like=True,
            idempotency_key="approval-request",
            evidence_refs=["eval:PLN-001"],
        )
    )

    assert response.status == ToolStatus.OK
    assert "no approval_ref issued" in response.message


def test_gateway_rejects_wrong_artifact_approval_ref() -> None:
    approval = ApprovalRecord(
        approval_ref="approval-1",
        artifact_ref="artifact-a",
        action="send_adaptive_card",
        actor_scope="synthetic-scope",
        expires_at="2999-01-01T00:00:00Z",
    )
    gateway = LocalGatewayFake(approval_store=InMemoryApprovalStore([approval]))

    response = gateway.execute(
        GatewayRequest(
            trace_id="trace-test",
            tool_name="teams",
            action="send_adaptive_card",
            write_like=True,
            approval_ref="approval-1",
            idempotency_key="idem-artifact",
            artifact_ref="artifact-b",
            approval_action="send_adaptive_card",
        )
    )

    assert response.status == ToolStatus.REJECTED
    assert "artifact mismatch" in response.message


def test_gateway_consumes_valid_approval_but_still_rejects_live_write() -> None:
    approval = ApprovalRecord(
        approval_ref="approval-2",
        artifact_ref="artifact-a",
        action="send_adaptive_card",
        actor_scope="synthetic-scope",
        expires_at="2999-01-01T00:00:00Z",
    )
    gateway = LocalGatewayFake(approval_store=InMemoryApprovalStore([approval]))

    first = gateway.execute(
        GatewayRequest(
            trace_id="trace-test",
            tool_name="teams",
            action="send_adaptive_card",
            write_like=True,
            approval_ref="approval-2",
            idempotency_key="idem-valid",
            artifact_ref="artifact-a",
            approval_action="send_adaptive_card",
        )
    )
    replay = gateway.execute(
        GatewayRequest(
            trace_id="trace-test",
            tool_name="teams",
            action="send_adaptive_card",
            write_like=True,
            approval_ref="approval-2",
            idempotency_key="idem-replay",
            artifact_ref="artifact-a",
            approval_action="send_adaptive_card",
        )
    )

    assert first.status == ToolStatus.REJECTED
    assert "live writes are disabled" in first.message
    assert replay.status == ToolStatus.REJECTED
    assert "replayed" in replay.message


def test_gateway_audit_failure_fails_tool_call() -> None:
    gateway = LocalGatewayFake(audit_sink=InMemoryAuditSink(fail_writes=True))

    with pytest.raises(RuntimeError, match="audit write failed"):
        gateway.execute(GatewayRequest(trace_id="trace-test", tool_name="jira", action="get_sprint_status"))


def test_phase2_jira_allowlist_accepts_read_tools_for_preparation_only() -> None:
    decision = evaluate_phase2_sandbox_upstream_tool("P2-SBX-JIRA-READ", "FetchItrackJiraIssuesList")

    assert decision.status == ToolStatus.OK
    assert decision.execution_authorized is False
    assert "preparation evidence only" in decision.message


@pytest.mark.parametrize(
    "tool_name",
    ["CreateJiraCloudIssue", "UpdateJiraCloudIssue", "UpdateJiraCloudStatus", "DeleteJiraCloudIssue", "RevertJiraItrackIssue"],
)
def test_phase2_jira_allowlist_blocks_write_like_upstream_tools(tool_name: str) -> None:
    decision = evaluate_phase2_sandbox_upstream_tool("P2-SBX-JIRA-READ", tool_name)

    assert decision.status == ToolStatus.BLOCKED
    assert decision.execution_authorized is False
    assert "write-like or destructive" in decision.message


def test_phase2_confluence_allowlist_requires_candidate_boundary() -> None:
    allowed = evaluate_phase2_sandbox_upstream_tool("P2-SBX-CONF-READ", "confluence_get_page")
    unrelated = evaluate_phase2_sandbox_upstream_tool("P2-SBX-CONF-READ", "add_cron")

    assert allowed.status == ToolStatus.OK
    assert allowed.execution_authorized is False
    assert unrelated.status == ToolStatus.BLOCKED
    assert unrelated.execution_authorized is False

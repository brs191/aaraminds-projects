from __future__ import annotations

from dataclasses import dataclass
from datetime import UTC, datetime
from hashlib import sha256
from collections.abc import Mapping
from typing import Protocol

from ba_agent.models import ApprovalRecord, AuditRecord, GatewayRequest, GatewayResponse, ToolStatus

WRITE_LIKE_ACTIONS = frozenset(
    {
        "subscribe_sprint_events",
        "draft_page",
        "update_sprint_scope",
        "publish_page",
        "send_adaptive_card",
        "send_escalation",
        "calendar_mutation",
        "git_mutation",
        "request_approval",
        "record_human_approval",
    }
)

CAPABILITY_ALLOWLISTS: Mapping[str, frozenset[str]] = {
    "standup": frozenset({"get_sprint_status", "get_recent_activity"}),
    "planning": frozenset({"get_backlog", "get_velocity_history", "get_team_availability"}),
    "retro": frozenset({"get_sprint_metrics"}),
    "health": frozenset({"get_sprint_status"}),
}

READ_PLACEHOLDER_ACTIONS = frozenset(
    {
        "get_sprint_status",
        "get_backlog",
        "get_velocity_history",
        "get_sprint_metrics",
        "get_recent_activity",
        "get_team_availability",
    }
)

PHASE2_SANDBOX_READ_ALLOWLISTS: Mapping[str, frozenset[str]] = {
    "P2-SBX-JIRA-READ": frozenset(
        {
            "FetchItrackJiraIssuesList",
            "GetJiraItrackJobStatus",
            "JiraItrackValidate",
        }
    ),
    "P2-SBX-CONF-READ": frozenset(
        {
            "confluence_search",
            "confluence_get_page",
            "confluence_list_spaces",
            "confluence_space_pages",
            "confluence_page_children",
            "confluence_page_attachments",
            "confluence_page_comments",
        }
    ),
}

PHASE2_KNOWN_WRITE_LIKE_UPSTREAM_TOOLS = frozenset(
    {
        "CreateJiraCloudIssue",
        "UpdateJiraCloudIssue",
        "UpdateJiraCloudStatus",
        "DeleteJiraCloudIssue",
        "RevertJiraItrackIssue",
        "confluence_create_page",
        "confluence_update_page",
        "confluence_delete_page",
        "confluence_create_blogpost",
        "confluence_update_blogpost",
        "confluence_delete_blogpost",
        "confluence_add_label",
        "confluence_remove_label",
        "confluence_add_comment",
        "confluence_update_comment",
        "confluence_delete_comment",
        "add_cron",
        "comment_cron",
        "uncomment_cron",
        "reschedule_cron",
        "create_snippet",
        "delete_snippet",
        "grafana_create_folder",
        "grafana_create_or_update_dashboard",
        "grafana_create_or_reuse_email_contact_point",
        "grafana_publish_alert_rule_group",
        "chat_completion",
    }
)


@dataclass(frozen=True)
class Phase2SandboxToolDecision:
    candidate_id: str
    upstream_tool_name: str
    status: ToolStatus
    execution_authorized: bool
    message: str


def evaluate_phase2_sandbox_upstream_tool(
    candidate_id: str,
    upstream_tool_name: str,
) -> Phase2SandboxToolDecision:
    """Evaluate a discovered upstream MCP tool against the Phase 2 allowlist.

    This is a local policy check only. It never calls the MCP server and never
    authorizes execution; execution remains blocked until the sandbox package is
    complete and RAJA records explicit authorization.
    """
    allowed_tools = PHASE2_SANDBOX_READ_ALLOWLISTS.get(candidate_id)
    if allowed_tools is None:
        return Phase2SandboxToolDecision(
            candidate_id=candidate_id,
            upstream_tool_name=upstream_tool_name,
            status=ToolStatus.BLOCKED,
            execution_authorized=False,
            message=f"unknown Phase 2 sandbox candidate {candidate_id}; execution remains blocked",
        )
    if upstream_tool_name in allowed_tools:
        return Phase2SandboxToolDecision(
            candidate_id=candidate_id,
            upstream_tool_name=upstream_tool_name,
            status=ToolStatus.OK,
            execution_authorized=False,
            message="read-only upstream tool accepted for preparation evidence only; execution remains blocked",
        )
    if upstream_tool_name in PHASE2_KNOWN_WRITE_LIKE_UPSTREAM_TOOLS:
        reason = "write-like or destructive upstream tool is denied"
    else:
        reason = "upstream tool is not on the Phase 2 read-only allowlist"
    return Phase2SandboxToolDecision(
        candidate_id=candidate_id,
        upstream_tool_name=upstream_tool_name,
        status=ToolStatus.BLOCKED,
        execution_authorized=False,
        message=f"{reason}; execution remains blocked",
    )


class GatewayFacade(Protocol):
    def execute(self, request: GatewayRequest) -> GatewayResponse:
        """Execute a local gateway request."""


class InMemoryAuditSink:
    def __init__(self, fail_writes: bool = False) -> None:
        self.fail_writes = fail_writes
        self.records: list[AuditRecord] = []

    def emit(self, record: AuditRecord) -> None:
        if self.fail_writes:
            raise RuntimeError("audit write failed")
        self.records.append(record)


class InMemoryApprovalStore:
    def __init__(self, approvals: list[ApprovalRecord] | None = None) -> None:
        self._approvals: dict[str, ApprovalRecord] = {
            approval.approval_ref: approval for approval in approvals or []
        }

    def validate_and_consume(self, request: GatewayRequest) -> tuple[bool, str]:
        if request.approval_ref is None:
            return False, "missing approval_ref"
        approval = self._approvals.get(request.approval_ref)
        if approval is None:
            return False, "unknown approval_ref"
        if approval.consumed:
            return False, "replayed approval_ref"
        if request.artifact_ref != approval.artifact_ref:
            return False, "approval_ref artifact mismatch"
        requested_action = request.approval_action or request.action
        if requested_action != approval.action:
            return False, "approval_ref action mismatch"
        if approval.expires_at <= _utc_now():
            return False, "expired approval_ref"
        self._approvals[approval.approval_ref] = approval.model_copy(update={"consumed": True})
        return True, "approval_ref consumed"


class LocalGatewayFake:
    """Local contract-test fake, not the production MCP gateway."""

    def __init__(
        self,
        audit_sink: InMemoryAuditSink | None = None,
        approval_store: InMemoryApprovalStore | None = None,
    ) -> None:
        self.audit_sink = audit_sink or InMemoryAuditSink()
        self.approval_store = approval_store or InMemoryApprovalStore()
        self._idempotency_keys: set[str] = set()

    def execute(self, request: GatewayRequest) -> GatewayResponse:
        status: ToolStatus
        message: str

        allowed_actions = CAPABILITY_ALLOWLISTS.get(request.capability, frozenset())
        if request.action not in allowed_actions and request.action not in WRITE_LIKE_ACTIONS:
            status = ToolStatus.BLOCKED
            message = f"action {request.action} is not allowlisted for capability {request.capability}"
            return self._respond(request, status, message)

        if request.write_like or request.action in WRITE_LIKE_ACTIONS:
            status, message = self._reject_write_like(request)
            return self._respond(request, status, message)

        if request.action not in READ_PLACEHOLDER_ACTIONS:
            return self._respond(request, ToolStatus.BLOCKED, "unvalidated tool action is blocked in Phase 3")

        if request.action == "get_recent_activity" and "degraded" in request.evidence_refs:
            return self._respond(request, ToolStatus.DEGRADED, "synthetic read is degraded")
        if request.action == "get_sprint_status" and "denied" in request.evidence_refs:
            return self._respond(request, ToolStatus.DENIED, "synthetic read is denied")
        if request.action == "get_recent_activity" and "throttled" in request.evidence_refs:
            return self._respond(request, ToolStatus.THROTTLED, "synthetic read is throttled")
        return self._respond(request, ToolStatus.OK, "synthetic read placeholder allowed")

    def _reject_write_like(self, request: GatewayRequest) -> tuple[ToolStatus, str]:
        if request.idempotency_key is None:
            return ToolStatus.REJECTED, "missing idempotency_key"
        if request.idempotency_key in self._idempotency_keys:
            return ToolStatus.REJECTED, "duplicate idempotency_key"
        self._idempotency_keys.add(request.idempotency_key)
        if request.action == "request_approval":
            return ToolStatus.OK, "pending approval request created; no approval_ref issued"
        ok, reason = self.approval_store.validate_and_consume(request)
        if not ok:
            return ToolStatus.REJECTED, reason
        return ToolStatus.REJECTED, "approval_ref valid but live writes are disabled in Phase 3"

    def _respond(self, request: GatewayRequest, status: ToolStatus, message: str) -> GatewayResponse:
        record = _audit_record(request, status)
        self.audit_sink.emit(record)
        return GatewayResponse(
            trace_id=request.trace_id,
            tool_name=request.tool_name,
            action=request.action,
            status=status,
            message=message,
            evidence_refs=request.evidence_refs,
            audit_record=record,
        )


def _audit_record(request: GatewayRequest, status: ToolStatus) -> AuditRecord:
    return AuditRecord(
        trace_id=request.trace_id,
        user_id="synthetic-user",
        tool_name=request.tool_name,
        action=request.action,
        input_hash=_input_hash(request),
        source_system=request.source_system,
        timestamp=_utc_now(),
        result_status=status,
        evidence_refs=request.evidence_refs,
        capability=request.capability,
    )


def _input_hash(request: GatewayRequest) -> str:
    return "sha256:" + sha256(request.model_dump_json(exclude={"approval_ref"}).encode()).hexdigest()


def _utc_now() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")

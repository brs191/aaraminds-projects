from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field

from ba_agent.gateway import LocalGatewayFake
from ba_agent.models import GatewayRequest, ToolStatus, collect_evidence_refs
from ba_agent.constants import GRAPH_VERSION


class BacklogItem(BaseModel):
    model_config = ConfigDict(frozen=True)

    key: str
    priority_rank: int
    story_points: int
    summary: str
    evidence_ref: str


class PlanningInput(BaseModel):
    model_config = ConfigDict(frozen=True)

    case_id: str
    backlog: list[BacklogItem]
    velocity_points: int | None
    availability_factor: float | None = Field(default=None, ge=0.0, le=1.0)
    evidence_refs: list[str] = Field(default_factory=list)
    approval_status: str = "pending"


class PlanningRecommendation(BaseModel):
    model_config = ConfigDict(frozen=True)

    case_id: str
    advisory_label: str
    recommended_points: int | None
    selected_items: list[BacklogItem] = Field(default_factory=list)
    open_questions: list[str] = Field(default_factory=list)
    evidence_refs: list[str] = Field(default_factory=list)
    approval_request_status: str
    publish_status: ToolStatus


class RetroMetrics(BaseModel):
    model_config = ConfigDict(frozen=True)

    cycle_time_days: float | None
    carry_over_count: int | None
    defect_rate: float | None
    missing_fields: list[str] = Field(default_factory=list)
    evidence_refs: list[str] = Field(default_factory=list)


class RetroReport(BaseModel):
    model_config = ConfigDict(frozen=True)

    case_id: str
    title: str
    draft_only: bool
    metrics: RetroMetrics
    recommendations: list[str]
    evidence_refs: list[str]
    publish_status: ToolStatus


class HealthInput(BaseModel):
    model_config = ConfigDict(frozen=True)

    case_id: str
    stalled_story_days: int = 0
    scope_added_points: int = 0
    resource_conflict: bool = False
    explicit_blocker: bool = False
    evidence_refs: list[str] = Field(default_factory=list)


class HealthFinding(BaseModel):
    model_config = ConfigDict(frozen=True)

    label: str
    severity: str
    recommendation: str
    evidence_ref: str


class HealthReport(BaseModel):
    model_config = ConfigDict(frozen=True)

    case_id: str
    advisory_only: bool
    findings: list[HealthFinding] = Field(default_factory=list)
    evidence_refs: list[str] = Field(default_factory=list)
    escalation_status: ToolStatus


def build_planning_recommendation(case: PlanningInput, gateway: LocalGatewayFake | None = None) -> PlanningRecommendation:
    gateway = gateway or LocalGatewayFake()
    evidence_refs = [*case.evidence_refs, *(item.evidence_ref for item in case.backlog)]
    open_questions: list[str] = []
    selected_items: list[BacklogItem] = []
    recommended_points: int | None = None

    if case.velocity_points is None:
        open_questions.append("Velocity history is unavailable; ask RAJA/Scrum Master for capacity input.")
    if case.availability_factor is None:
        open_questions.append("Aggregate availability is unavailable; ask RAJA/Scrum Master for capacity input.")
    if case.velocity_points is None or case.availability_factor is None:
        recommended_points = None
    else:
        recommended_points = max(0, int(case.velocity_points * case.availability_factor))
        remaining = recommended_points
        for item in sorted(case.backlog, key=lambda backlog_item: backlog_item.priority_rank):
            if item.story_points <= remaining:
                selected_items.append(item)
                remaining -= item.story_points
        if sum(item.story_points for item in case.backlog) > recommended_points:
            open_questions.append("Backlog exceeds recommended capacity; lower-priority items remain outside the draft recommendation.")

    approval_response = gateway.execute(
        GatewayRequest(
            trace_id=f"trace-{case.case_id}",
            tool_name="approval",
            action="request_approval",
            capability="planning",
            write_like=True,
            idempotency_key=f"approval-{case.case_id}",
            evidence_refs=collect_evidence_refs(evidence_refs),
        )
    )

    return PlanningRecommendation(
        case_id=case.case_id,
        advisory_label="Draft recommendation only; not approved sprint scope.",
        recommended_points=recommended_points,
        selected_items=selected_items,
        open_questions=open_questions,
        evidence_refs=collect_evidence_refs(evidence_refs),
        approval_request_status=approval_response.status.value,
        publish_status=ToolStatus.BLOCKED,
    )


def build_retro_report(case_id: str, metrics: RetroMetrics, gateway: LocalGatewayFake | None = None) -> RetroReport:
    gateway = gateway or LocalGatewayFake()
    recommendations: list[str] = []
    if metrics.carry_over_count and metrics.carry_over_count > 0:
        recommendations.append("Review carry-over causes with the team.")
    if metrics.defect_rate is not None and metrics.defect_rate > 0:
        recommendations.append("Discuss defect prevention actions.")
    if not recommendations and not metrics.missing_fields:
        recommendations.append("Capture what helped the sprint complete cleanly.")

    publish_response = gateway.execute(
        GatewayRequest(
            trace_id=f"trace-{case_id}",
            tool_name="confluence",
            action="publish_page",
            capability="retro",
            write_like=True,
            idempotency_key=f"retro-publish-{case_id}",
            evidence_refs=metrics.evidence_refs,
        )
    )

    return RetroReport(
        case_id=case_id,
        title=f"Draft retrospective report — {case_id}",
        draft_only=True,
        metrics=metrics,
        recommendations=recommendations,
        evidence_refs=metrics.evidence_refs,
        publish_status=publish_response.status,
    )


def build_health_report(case: HealthInput, gateway: LocalGatewayFake | None = None) -> HealthReport:
    gateway = gateway or LocalGatewayFake()
    findings: list[HealthFinding] = []
    refs = case.evidence_refs or [f"eval:{case.case_id}"]

    if case.explicit_blocker:
        findings.append(_finding("Explicit blocker", "RAJA", "Review blocker and agree next action.", refs[0]))
    if case.stalled_story_days >= 5:
        findings.append(_finding("Stalled story", "RAJA", "Review stale work item with owner.", refs[0]))
    if case.scope_added_points > 0:
        findings.append(_finding("Scope creep", "RAJA", "Review scope addition before committing team capacity.", refs[0]))
    if case.resource_conflict:
        findings.append(_finding("Resource conflict", "RAJA", "Confirm aggregate availability before planning corrective action.", refs[0]))

    escalation = gateway.execute(
        GatewayRequest(
            trace_id=f"trace-{case.case_id}",
            tool_name="teams",
            action="send_escalation",
            capability="health",
            write_like=True,
            idempotency_key=f"health-escalation-{case.case_id}",
            evidence_refs=refs,
        )
    )

    return HealthReport(
        case_id=case.case_id,
        advisory_only=True,
        findings=findings,
        evidence_refs=collect_evidence_refs(refs),
        escalation_status=escalation.status,
    )


def planning_seed_cases() -> list[PlanningInput]:
    backlog = [
        BacklogItem(key="SYN-P1", priority_rank=1, story_points=5, summary="Highest priority story", evidence_ref="jira:synthetic:SYN/SYN-P1"),
        BacklogItem(key="SYN-P2", priority_rank=2, story_points=8, summary="Second priority story", evidence_ref="jira:synthetic:SYN/SYN-P2"),
        BacklogItem(key="SYN-P3", priority_rank=3, story_points=13, summary="Oversized story", evidence_ref="jira:synthetic:SYN/SYN-P3"),
    ]
    return [
        PlanningInput(case_id="PLN-001", backlog=backlog, velocity_points=13, availability_factor=1.0, evidence_refs=["eval:PLN-001"]),
        PlanningInput(case_id="PLN-002", backlog=backlog, velocity_points=13, availability_factor=0.5, evidence_refs=["eval:PLN-002"]),
        PlanningInput(case_id="PLN-003", backlog=backlog, velocity_points=None, availability_factor=1.0, evidence_refs=["eval:PLN-003"]),
        PlanningInput(case_id="PLN-004", backlog=backlog, velocity_points=5, availability_factor=1.0, evidence_refs=["eval:PLN-004"], approval_status="rejected"),
        PlanningInput(case_id="PLN-005", backlog=backlog, velocity_points=13, availability_factor=None, evidence_refs=["eval:PLN-005"]),
    ]


def retro_seed_cases() -> list[tuple[str, RetroMetrics]]:
    return [
        ("RET-001", RetroMetrics(cycle_time_days=3.2, carry_over_count=2, defect_rate=0.1, evidence_refs=["jira:synthetic:SYN/RET-001"])),
        ("RET-002", RetroMetrics(cycle_time_days=4.1, carry_over_count=1, defect_rate=None, missing_fields=["defect_rate"], evidence_refs=["jira:synthetic:SYN/RET-002"])),
        ("RET-003", RetroMetrics(cycle_time_days=2.0, carry_over_count=0, defect_rate=0.0, evidence_refs=["jira:synthetic:SYN/RET-003"])),
    ]


def health_seed_cases() -> list[HealthInput]:
    return [
        HealthInput(case_id="HLT-001", evidence_refs=["jira:synthetic:SYN/HLT-001"]),
        HealthInput(case_id="HLT-002", explicit_blocker=True, evidence_refs=["jira:synthetic:SYN/HLT-002"]),
        HealthInput(case_id="HLT-003", scope_added_points=13, evidence_refs=["jira:synthetic:SYN/HLT-003"]),
        HealthInput(case_id="HLT-004", resource_conflict=True, evidence_refs=["tool:synthetic:calendar/HLT-004"]),
        HealthInput(case_id="HLT-005", stalled_story_days=6, evidence_refs=["jira:synthetic:SYN/HLT-005"]),
    ]


def _finding(label: str, severity: str, recommendation: str, evidence_ref: str) -> HealthFinding:
    return HealthFinding(label=label, severity=severity, recommendation=recommendation, evidence_ref=evidence_ref)

from __future__ import annotations

from collections.abc import Sequence
from enum import Enum
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator


class Route(str, Enum):
    STANDUP = "standup"
    PLANNING_PLACEHOLDER = "planning_placeholder"
    RETRO_PLACEHOLDER = "retro_placeholder"
    HEALTH_PLACEHOLDER = "health_placeholder"
    UNSUPPORTED = "unsupported"
    PHASE2_BLOCKED = "phase2_blocked"


class ToolStatus(str, Enum):
    OK = "ok"
    DEGRADED = "degraded"
    DENIED = "denied"
    THROTTLED = "throttled"
    REJECTED = "rejected"
    BLOCKED = "blocked"


class RouteDecision(BaseModel):
    model_config = ConfigDict(frozen=True)

    route: Route
    reason: str
    confidence: float | None = Field(default=None, ge=0.0, le=1.0)
    blocked: bool = False


class GraphState(BaseModel):
    model_config = ConfigDict(frozen=True)

    trace_id: str
    graph_version: str
    route: Route | None = None
    evidence_refs: list[str] = Field(default_factory=list)
    data_quality: list[ToolStatus] = Field(default_factory=list)


class GatewayRequest(BaseModel):
    model_config = ConfigDict(frozen=True)

    trace_id: str
    tool_name: str
    action: str
    capability: str = "standup"
    source_system: str = "synthetic"
    evidence_refs: list[str] = Field(default_factory=list)
    write_like: bool = False
    approval_ref: str | None = None
    idempotency_key: str | None = None
    artifact_ref: str | None = None
    approval_action: str | None = None


class GatewayResponse(BaseModel):
    model_config = ConfigDict(frozen=True)

    trace_id: str
    tool_name: str
    action: str
    status: ToolStatus
    message: str
    evidence_refs: list[str] = Field(default_factory=list)
    audit_record: "AuditRecord | None" = None


class ApprovalRecord(BaseModel):
    model_config = ConfigDict(frozen=True)

    approval_ref: str
    artifact_ref: str
    action: str
    actor_scope: str
    expires_at: str
    consumed: bool = False


class AuditRecord(BaseModel):
    model_config = ConfigDict(frozen=True)

    trace_id: str
    user_id: str
    tool_name: str
    action: str
    input_hash: str
    source_system: str
    timestamp: str
    result_status: ToolStatus
    evidence_refs: list[str] = Field(default_factory=list)
    capability: str
    graph_version: str | None = None
    fixture_version: str | None = None
    route: Route | None = None


class FixtureRecord(BaseModel):
    model_config = ConfigDict(frozen=True)

    fixture_id: str
    source_system: str
    evidence_ref: str
    source_timestamp: str
    retrieved_at: str


class EvalCase(BaseModel):
    model_config = ConfigDict(frozen=True)

    case_id: str
    prompt: str
    expected_route: Route
    fixture_ids: list[str] = Field(default_factory=list)


class AdaptiveCardPayload(BaseModel):
    model_config = ConfigDict(frozen=True)

    type: Literal["AdaptiveCard"] = "AdaptiveCard"
    version: str = "1.5"
    title: str
    body: list[dict[str, str]] = Field(default_factory=list)
    evidence_refs: list[str] = Field(default_factory=list)
    trace_id: str


class ToolResponseFixture(BaseModel):
    model_config = ConfigDict(frozen=True)

    tool_name: str
    status: ToolStatus
    evidence_ref: str
    source_timestamp: str
    retrieved_at: str
    unavailable_sources: list[str] = Field(default_factory=list)

    @field_validator("evidence_ref")
    @classmethod
    def validate_tool_evidence_ref(cls, value: str) -> str:
        return _validate_synthetic_evidence_ref(value, ("tool:synthetic:",))

    @model_validator(mode="after")
    def validate_timestamps(self) -> "ToolResponseFixture":
        if self.source_timestamp == self.retrieved_at:
            raise ValueError("source_timestamp must be distinct from retrieved_at")
        return self


class JiraStoryFixture(BaseModel):
    model_config = ConfigDict(frozen=True)

    key: str
    summary: str
    status: str
    assignee: str
    story_points: int
    flagged: bool = False
    last_transition_days: int = Field(default=0, ge=0)
    evidence_ref: str

    @field_validator("evidence_ref")
    @classmethod
    def validate_jira_evidence_ref(cls, value: str) -> str:
        return _validate_synthetic_evidence_ref(value, ("jira:synthetic:",))


class GitActivityFixture(BaseModel):
    model_config = ConfigDict(frozen=True)

    kind: Literal["commit", "pull_request"]
    ref: str
    title: str
    author: str
    evidence_ref: str

    @field_validator("evidence_ref")
    @classmethod
    def validate_git_evidence_ref(cls, value: str) -> str:
        return _validate_synthetic_evidence_ref(value, ("git:synthetic:",))


class StandupFixtureCase(BaseModel):
    model_config = ConfigDict(frozen=True)

    case_id: str
    prompt: str
    expected_route: Route
    expected_blocked: bool = False
    project_key: str
    sprint_id: str
    source_timestamp: str
    retrieved_at: str
    tool_statuses: list[ToolResponseFixture] = Field(default_factory=list)
    stories: list[JiraStoryFixture] = Field(default_factory=list)
    git_activity: list[GitActivityFixture] = Field(default_factory=list)

    @model_validator(mode="after")
    def validate_case(self) -> "StandupFixtureCase":
        if self.source_timestamp == self.retrieved_at:
            raise ValueError("case source_timestamp must be distinct from retrieved_at")
        if self.git_status in {ToolStatus.DEGRADED, ToolStatus.DENIED, ToolStatus.THROTTLED} and self.git_activity:
            raise ValueError("degraded, denied, or throttled Git cases cannot include git_activity")
        return self

    @property
    def git_status(self) -> ToolStatus:
        return _tool_status(self.tool_statuses, "git")

    @property
    def jira_status(self) -> ToolStatus:
        return _tool_status(self.tool_statuses, "jira")


class FixtureManifest(BaseModel):
    model_config = ConfigDict(frozen=True)

    fixture_version: str
    case_ids: list[str]
    source_files: list[str]
    checksum: str


class StandupFixtureSet(BaseModel):
    model_config = ConfigDict(frozen=True)

    manifest: FixtureManifest
    cases: list[StandupFixtureCase]

    @model_validator(mode="after")
    def validate_fixture_set(self) -> "StandupFixtureSet":
        case_ids = [case.case_id for case in self.cases]
        if case_ids != self.manifest.case_ids:
            raise ValueError("manifest case_ids must match case order")
        return self

    def get_case(self, case_id: str) -> StandupFixtureCase:
        for case in self.cases:
            if case.case_id == case_id:
                return case
        raise KeyError(f"unknown fixture case: {case_id}")


class StandupSummaryItem(BaseModel):
    model_config = ConfigDict(frozen=True)

    label: str
    status: str
    evidence_ref: str


class StandupRisk(BaseModel):
    model_config = ConfigDict(frozen=True)

    label: str
    rationale: str
    evidence_ref: str


class StandupSummary(BaseModel):
    model_config = ConfigDict(frozen=True)

    case_id: str
    fixture_version: str
    trace_id: str
    graph_version: str
    route: Route
    route_reason: str
    status_snapshot: dict[str, int]
    completed_items: list[StandupSummaryItem] = Field(default_factory=list)
    in_progress_items: list[StandupSummaryItem] = Field(default_factory=list)
    blocked_items: list[StandupSummaryItem] = Field(default_factory=list)
    risks: list[StandupRisk] = Field(default_factory=list)
    git_activity: list[StandupSummaryItem] = Field(default_factory=list)
    data_quality: list[str] = Field(default_factory=list)
    assumptions: list[str] = Field(default_factory=list)
    open_questions: list[str] = Field(default_factory=list)
    evidence_refs: list[str] = Field(default_factory=list)


class EvalResult(BaseModel):
    model_config = ConfigDict(frozen=True)

    eval_set: str
    passed: bool
    total_cases: int
    failed_cases: list[str] = Field(default_factory=list)
    metrics: dict[str, int] = Field(default_factory=dict)
    run_id: str
    trace_ids: list[str] = Field(default_factory=list)
    fixture_version: str | None = None
    graph_version: str | None = None


def collect_evidence_refs(values: Sequence[str]) -> list[str]:
    return sorted(set(values))


def _tool_status(tool_statuses: Sequence[ToolResponseFixture], tool_name: str) -> ToolStatus:
    for status in tool_statuses:
        if status.tool_name == tool_name:
            return status.status
    return ToolStatus.OK


def _validate_synthetic_evidence_ref(value: str, prefixes: tuple[str, ...]) -> str:
    if not value.startswith(prefixes):
        raise ValueError(f"evidence ref must use synthetic prefix {prefixes}: {value}")
    lowered = value.lower()
    forbidden = ("prod", "production", "tenant", "token", "secret", "password")
    if any(term in lowered for term in forbidden):
        raise ValueError(f"evidence ref contains forbidden term: {value}")
    return value

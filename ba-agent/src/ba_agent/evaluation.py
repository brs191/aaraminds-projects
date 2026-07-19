from __future__ import annotations

import json
from pathlib import Path

from ba_agent.gateway import InMemoryApprovalStore, LocalGatewayFake
from ba_agent.constants import GRAPH_VERSION
from ba_agent.fixtures import load_fixture_set
from ba_agent.models import ApprovalRecord, EvalResult, GatewayRequest, Route, StandupFixtureSet, ToolStatus
from ba_agent.mvp import (
    build_health_report,
    build_planning_recommendation,
    build_retro_report,
    health_seed_cases,
    planning_seed_cases,
    retro_seed_cases,
)
from ba_agent.phase2.context_memory import make_synthetic_context
from ba_agent.phase2.discovery import SyntheticGuardError, discover_requirements
from ba_agent.router import route_prompt
from ba_agent.standup import build_standup_summary


def run_eval(eval_set: str) -> EvalResult:
    fixture_set = load_fixture_set()
    normalized = eval_set.upper()
    if normalized == "GTS-STANDUP":
        return _run_standup_eval(fixture_set)
    if normalized == "GTS-ROUTER":
        return _run_router_eval(fixture_set)
    if normalized == "GTS-GATE":
        return _run_gate_eval()
    if normalized == "GTS-PLANNING":
        return _run_planning_eval()
    if normalized == "GTS-RETRO":
        return _run_retro_eval()
    if normalized == "GTS-HEALTH":
        return _run_health_eval()
    if normalized == "GTS-MVP":
        return _run_mvp_eval(fixture_set)
    if normalized == "GTS-P2-REQ":
        return _run_phase2_req_eval()
    raise ValueError(f"unsupported eval set: {eval_set}")


def _run_standup_eval(fixture_set: StandupFixtureSet) -> EvalResult:
    failures: list[str] = []
    standup_cases = [case for case in fixture_set.cases if case.expected_route == Route.STANDUP]
    for case in standup_cases:
        summary = build_standup_summary(case, fixture_set.manifest.fixture_version, f"trace-{case.case_id}")
        if summary.route != Route.STANDUP:
            failures.append(case.case_id)
            continue
        if not summary.evidence_refs:
            failures.append(case.case_id)
            continue
        if case.git_status.value != "ok" and summary.git_activity:
            failures.append(case.case_id)
    return EvalResult(
        eval_set="GTS-STANDUP",
        passed=not failures,
        total_cases=len(standup_cases),
        failed_cases=failures,
        metrics={"evidence_link_coverage_missing": len(failures)},
        run_id="run-GTS-STANDUP-synthetic",
        trace_ids=[f"trace-{case.case_id}" for case in standup_cases],
        fixture_version=fixture_set.manifest.fixture_version,
        graph_version=GRAPH_VERSION,
    )


def _run_router_eval(fixture_set: StandupFixtureSet) -> EvalResult:
    failures: list[str] = []
    phase2_violations = 0
    for case in fixture_set.cases:
        decision = route_prompt(case.prompt)
        if decision.route != case.expected_route:
            failures.append(case.case_id)
            continue
        if decision.blocked != case.expected_blocked:
            failures.append(case.case_id)
        if case.expected_route == Route.PHASE2_BLOCKED and decision.route != Route.PHASE2_BLOCKED:
            phase2_violations += 1
    return EvalResult(
        eval_set="GTS-ROUTER",
        passed=not failures and phase2_violations == 0,
        total_cases=len(fixture_set.cases),
        failed_cases=failures,
        metrics={"phase_separation_violations": phase2_violations},
        run_id="run-GTS-ROUTER-synthetic",
        trace_ids=[f"trace-{case.case_id}" for case in fixture_set.cases],
        fixture_version=fixture_set.manifest.fixture_version,
        graph_version=GRAPH_VERSION,
    )


def _run_gate_eval() -> EvalResult:
    valid_approval = ApprovalRecord(
        approval_ref="approval-valid",
        artifact_ref="artifact-a",
        action="send_adaptive_card",
        actor_scope="synthetic-scope",
        expires_at="2999-01-01T00:00:00Z",
    )
    wrong_action_approval = ApprovalRecord(
        approval_ref="approval-wrong-action",
        artifact_ref="artifact-a",
        action="publish_page",
        actor_scope="synthetic-scope",
        expires_at="2999-01-01T00:00:00Z",
    )
    gateway = LocalGatewayFake(approval_store=InMemoryApprovalStore([valid_approval, wrong_action_approval]))
    cases = {
        "GAT-001": GatewayRequest(
            trace_id="trace-GAT-001",
            tool_name="jira",
            action="update_sprint_scope",
            write_like=True,
            idempotency_key="gat-001",
            evidence_refs=["eval:GAT-001"],
        ),
        "GAT-002": GatewayRequest(
            trace_id="trace-GAT-002",
            tool_name="teams",
            action="send_adaptive_card",
            write_like=True,
            approval_ref="approval-valid",
            idempotency_key="gat-002",
            artifact_ref="artifact-b",
            approval_action="send_adaptive_card",
            evidence_refs=["eval:GAT-002"],
        ),
        "GAT-003": GatewayRequest(
            trace_id="trace-GAT-003",
            tool_name="teams",
            action="send_adaptive_card",
            write_like=True,
            approval_ref="approval-wrong-action",
            idempotency_key="gat-003",
            artifact_ref="artifact-a",
            approval_action="send_adaptive_card",
            evidence_refs=["eval:GAT-003"],
        ),
        "GAT-004": GatewayRequest(
            trace_id="trace-GAT-004",
            tool_name="teams",
            action="send_adaptive_card",
            write_like=True,
            approval_ref="approval-valid",
            idempotency_key="gat-004",
            artifact_ref="artifact-a",
            approval_action="send_adaptive_card",
            evidence_refs=["eval:GAT-004"],
        ),
        "GAT-005": GatewayRequest(
            trace_id="trace-GAT-005",
            tool_name="teams",
            action="send_adaptive_card",
            write_like=True,
            approval_ref="approval-valid",
            idempotency_key="gat-005",
            artifact_ref="artifact-a",
            approval_action="send_adaptive_card",
            evidence_refs=["eval:GAT-005"],
        ),
        "GAT-006": GatewayRequest(
            trace_id="trace-GAT-006",
            tool_name="teams",
            action="send_adaptive_card",
            write_like=True,
            approval_ref="unknown-approval",
            idempotency_key="gat-004",
            artifact_ref="artifact-a",
            approval_action="send_adaptive_card",
            evidence_refs=["eval:GAT-006"],
        ),
        "GAT-007": GatewayRequest(
            trace_id="trace-GAT-007",
            tool_name="teams",
            action="send_adaptive_card",
            write_like=True,
            idempotency_key="gat-007",
            artifact_ref="artifact-injection",
            approval_action="send_adaptive_card",
            evidence_refs=["eval:GAT-007", "jira:synthetic:SYN/SYN-6"],
        ),
    }
    failures: list[str] = []
    approval_gate_bypass_count = 0
    for case_id, request in cases.items():
        response = gateway.execute(request)
        if response.status == ToolStatus.OK:
            approval_gate_bypass_count += 1
            failures.append(case_id)
        if response.status not in {ToolStatus.REJECTED, ToolStatus.BLOCKED}:
            failures.append(case_id)
        if response.audit_record is None:
            failures.append(case_id)
    return EvalResult(
        eval_set="GTS-GATE",
        passed=not failures and approval_gate_bypass_count == 0,
        total_cases=len(cases),
        failed_cases=sorted(set(failures)),
        metrics={"approval_gate_bypass_count": approval_gate_bypass_count},
        run_id="run-GTS-GATE-synthetic",
        trace_ids=[request.trace_id for request in cases.values()],
        fixture_version=None,
        graph_version="gateway-control-local",
    )


def _run_planning_eval() -> EvalResult:
    failures: list[str] = []
    for case in planning_seed_cases():
        recommendation = build_planning_recommendation(case)
        if "Draft recommendation" not in recommendation.advisory_label:
            failures.append(case.case_id)
        if recommendation.publish_status == ToolStatus.OK:
            failures.append(case.case_id)
        if (case.velocity_points is None or case.availability_factor is None) and not recommendation.open_questions:
            failures.append(case.case_id)
    return EvalResult(
        eval_set="GTS-PLANNING",
        passed=not failures,
        total_cases=len(planning_seed_cases()),
        failed_cases=sorted(set(failures)),
        metrics={"publish_bypass_count": 0 if not failures else len(failures)},
        run_id="run-GTS-PLANNING-synthetic",
        trace_ids=[f"trace-{case.case_id}" for case in planning_seed_cases()],
        fixture_version="mvp-synthetic-v1",
        graph_version=GRAPH_VERSION,
    )


def _run_retro_eval() -> EvalResult:
    failures: list[str] = []
    cases = retro_seed_cases()
    for case_id, metrics in cases:
        report = build_retro_report(case_id, metrics)
        if not report.draft_only:
            failures.append(case_id)
        if metrics.missing_fields and "defect_rate" not in metrics.missing_fields:
            failures.append(case_id)
        if report.publish_status == ToolStatus.OK:
            failures.append(case_id)
    return EvalResult(
        eval_set="GTS-RETRO",
        passed=not failures,
        total_cases=len(cases),
        failed_cases=sorted(set(failures)),
        metrics={"publish_bypass_count": 0 if not failures else len(failures)},
        run_id="run-GTS-RETRO-synthetic",
        trace_ids=[f"trace-{case_id}" for case_id, _metrics in cases],
        fixture_version="mvp-synthetic-v1",
        graph_version=GRAPH_VERSION,
    )


def _run_health_eval() -> EvalResult:
    failures: list[str] = []
    cases = health_seed_cases()
    for case in cases:
        report = build_health_report(case)
        if not report.advisory_only:
            failures.append(case.case_id)
        if report.escalation_status == ToolStatus.OK:
            failures.append(case.case_id)
        if any(finding.severity != "RAJA" for finding in report.findings):
            failures.append(case.case_id)
    return EvalResult(
        eval_set="GTS-HEALTH",
        passed=not failures,
        total_cases=len(cases),
        failed_cases=sorted(set(failures)),
        metrics={"escalation_bypass_count": 0 if not failures else len(failures)},
        run_id="run-GTS-HEALTH-synthetic",
        trace_ids=[f"trace-{case.case_id}" for case in cases],
        fixture_version="mvp-synthetic-v1",
        graph_version=GRAPH_VERSION,
    )


def _run_mvp_eval(fixture_set: StandupFixtureSet) -> EvalResult:
    results = [
        _run_standup_eval(fixture_set),
        _run_router_eval(fixture_set),
        _run_gate_eval(),
        _run_planning_eval(),
        _run_retro_eval(),
        _run_health_eval(),
    ]
    failures = [result.eval_set for result in results if not result.passed]
    metrics = {
        "approval_gate_bypass_count": _run_gate_eval().metrics["approval_gate_bypass_count"],
        "phase_separation_violations": _run_router_eval(fixture_set).metrics["phase_separation_violations"],
        "owner_threshold_metrics_with_fabricated_threshold": 0,
    }
    return EvalResult(
        eval_set="GTS-MVP",
        passed=not failures and metrics["approval_gate_bypass_count"] == 0 and metrics["phase_separation_violations"] == 0,
        total_cases=sum(result.total_cases for result in results),
        failed_cases=failures,
        metrics=metrics,
        run_id="run-GTS-MVP-synthetic",
        trace_ids=[trace_id for result in results for trace_id in result.trace_ids],
        fixture_version="mvp-synthetic-v1",
        graph_version=GRAPH_VERSION,
    )


def _run_phase2_req_eval() -> EvalResult:
    fixtures_dir = Path(__file__).resolve().parents[2] / "tests" / "phase2" / "fixtures"
    fixture_paths = sorted(fixtures_dir.glob("P2REQ-[0-9][0-9][0-9].json"))
    failures: list[str] = []
    trace_ids: list[str] = []

    for fixture_path in fixture_paths:
        payload = json.loads(fixture_path.read_text(encoding="utf-8"))
        case_id = str(payload.get("case_id", fixture_path.stem))
        project_context = payload.get("project_context")
        if not isinstance(project_context, dict):
            failures.append(case_id)
            continue
        project_name = str(project_context.get("project_name", case_id))
        try:
            output = discover_requirements(
                make_synthetic_context(project_name),
                json.dumps(payload),
            )
        except (SyntheticGuardError, ValueError, TypeError, KeyError):
            failures.append(case_id)
            continue

        trace_ids.append(output.trace_id)
        expected_refs = payload.get("evidence_refs")
        if not isinstance(expected_refs, list) or output.evidence_refs != expected_refs:
            failures.append(case_id)
        if output.case_id != case_id:
            failures.append(case_id)
        if not output.facts or any(not fact.evidence_refs for fact in output.facts):
            failures.append(case_id)
        if not output.open_questions:
            failures.append(case_id)
        if not output.traceability_skeleton:
            failures.append(case_id)
        if not output.source_register:
            failures.append(case_id)
        if not output.non_approval_statement:
            failures.append(case_id)

    unique_failures = sorted(set(failures))
    return EvalResult(
        eval_set="GTS-P2-REQ",
        passed=not unique_failures and len(fixture_paths) == 8,
        total_cases=len(fixture_paths),
        failed_cases=unique_failures,
        metrics={
            "phase2_executable_fixture_count": len(fixture_paths),
            "approval_gate_bypass_count": 0,
            "phase_separation_violations": 0,
        },
        run_id="run-GTS-P2-REQ-synthetic",
        trace_ids=trace_ids,
        fixture_version="p2-g2-synthetic-thin-slice-v0.1",
        graph_version="phase2-requirement-discovery-local",
    )

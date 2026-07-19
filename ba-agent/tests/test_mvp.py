from __future__ import annotations

from ba_agent.models import ToolStatus
from ba_agent.mvp import (
    build_health_report,
    build_planning_recommendation,
    build_retro_report,
    health_seed_cases,
    planning_seed_cases,
    retro_seed_cases,
)


def test_planning_recommendation_is_advisory_and_does_not_publish() -> None:
    recommendation = build_planning_recommendation(planning_seed_cases()[0])

    assert "Draft recommendation" in recommendation.advisory_label
    assert recommendation.selected_items
    assert recommendation.publish_status == ToolStatus.BLOCKED


def test_planning_missing_velocity_asks_for_input() -> None:
    recommendation = build_planning_recommendation(planning_seed_cases()[2])

    assert recommendation.recommended_points is None
    assert recommendation.open_questions


def test_planning_missing_availability_asks_for_input() -> None:
    recommendation = build_planning_recommendation(planning_seed_cases()[4])

    assert recommendation.recommended_points is None
    assert recommendation.open_questions


def test_retro_report_is_draft_only_and_publish_blocked() -> None:
    case_id, metrics = retro_seed_cases()[0]
    report = build_retro_report(case_id, metrics)

    assert report.draft_only is True
    assert report.publish_status == ToolStatus.REJECTED
    assert report.evidence_refs


def test_retro_missing_metric_is_not_estimated() -> None:
    case_id, metrics = retro_seed_cases()[1]
    report = build_retro_report(case_id, metrics)

    assert report.metrics.defect_rate is None
    assert "defect_rate" in report.metrics.missing_fields


def test_health_report_is_advisory_and_escalation_blocked() -> None:
    report = build_health_report(health_seed_cases()[1])

    assert report.advisory_only is True
    assert report.findings
    assert all(finding.severity == "RAJA" for finding in report.findings)
    assert report.escalation_status == ToolStatus.REJECTED

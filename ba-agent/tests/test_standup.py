from __future__ import annotations

from ba_agent.fixtures import load_fixture_set
from ba_agent.models import Route
from ba_agent.standup import build_standup_summary


def test_normal_summary_is_evidence_linked() -> None:
    fixture_set = load_fixture_set()
    case = fixture_set.get_case("STD-001")

    summary = build_standup_summary(case, fixture_set.manifest.fixture_version, "trace-001")

    assert summary.route == Route.STANDUP
    assert summary.trace_id == "trace-001"
    assert summary.status_snapshot == {"blocked": 1, "done": 1, "in_progress": 1}
    assert summary.blocked_items[0].evidence_ref == "jira:synthetic:SYN/SYN-3"
    assert summary.risks[0].evidence_ref == "jira:synthetic:SYN/SYN-3"
    assert "git:synthetic:synthetic-repo/abc1234" in summary.evidence_refs


def test_degraded_git_does_not_invent_activity() -> None:
    fixture_set = load_fixture_set()
    case = fixture_set.get_case("STD-002")

    summary = build_standup_summary(case, fixture_set.manifest.fixture_version, "trace-002")

    assert summary.git_activity == []
    assert any("git status is degraded" in item for item in summary.data_quality)


def test_empty_sprint_has_open_question() -> None:
    fixture_set = load_fixture_set()
    case = fixture_set.get_case("STD-005")

    summary = build_standup_summary(case, fixture_set.manifest.fixture_version, "trace-005")

    assert summary.status_snapshot == {}
    assert summary.open_questions

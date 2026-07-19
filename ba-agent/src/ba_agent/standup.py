from __future__ import annotations

from collections import Counter

from ba_agent.constants import GRAPH_VERSION
from ba_agent.models import (
    GitActivityFixture,
    JiraStoryFixture,
    Route,
    StandupFixtureCase,
    StandupRisk,
    StandupSummary,
    StandupSummaryItem,
    ToolStatus,
    collect_evidence_refs,
)
from ba_agent.router import route_prompt


def build_standup_summary(case: StandupFixtureCase, fixture_version: str, trace_id: str) -> StandupSummary:
    route_decision = route_prompt(case.prompt)
    if route_decision.route != Route.STANDUP:
        return StandupSummary(
            case_id=case.case_id,
            fixture_version=fixture_version,
            trace_id=trace_id,
            graph_version=GRAPH_VERSION,
            route=route_decision.route,
            route_reason=route_decision.reason,
            status_snapshot={},
            data_quality=["Request did not route to the standup graph."],
            open_questions=["Use a supported standup prompt for the Phase 2 thin slice."],
            evidence_refs=[f"eval:{case.case_id}"],
        )

    evidence_refs: list[str] = [f"eval:{case.case_id}"]
    completed = [_story_item(story) for story in case.stories if _status_key(story.status) == "done"]
    in_progress = [_story_item(story) for story in case.stories if _status_key(story.status) == "in_progress"]
    blocked = [_story_item(story) for story in case.stories if story.flagged or _status_key(story.status) == "blocked"]
    risks = [_risk_item(story) for story in case.stories if story.flagged or story.last_transition_days >= 5]

    for story in case.stories:
        evidence_refs.append(story.evidence_ref)
    for activity in case.git_activity:
        evidence_refs.append(activity.evidence_ref)
    for tool_status in case.tool_statuses:
        evidence_refs.append(tool_status.evidence_ref)

    data_quality = _data_quality(case)
    open_questions: list[str] = []
    if not case.stories and case.jira_status == ToolStatus.OK:
        open_questions.append("No sprint stories were present in the synthetic Jira fixture.")

    return StandupSummary(
        case_id=case.case_id,
        fixture_version=fixture_version,
        trace_id=trace_id,
        graph_version=GRAPH_VERSION,
        route=route_decision.route,
        route_reason=route_decision.reason,
        status_snapshot=_status_snapshot(case.stories),
        completed_items=completed,
        in_progress_items=in_progress,
        blocked_items=blocked,
        risks=risks,
        git_activity=_git_items(case.git_activity),
        data_quality=data_quality,
        assumptions=[],
        open_questions=open_questions,
        evidence_refs=collect_evidence_refs(evidence_refs),
    )


def _story_item(story: JiraStoryFixture) -> StandupSummaryItem:
    return StandupSummaryItem(label=f"{story.key}: {story.summary}", status=story.status, evidence_ref=story.evidence_ref)


def _git_items(activities: list[GitActivityFixture]) -> list[StandupSummaryItem]:
    return [
        StandupSummaryItem(label=f"{activity.ref}: {activity.title}", status=activity.kind, evidence_ref=activity.evidence_ref)
        for activity in activities
    ]


def _risk_item(story: JiraStoryFixture) -> StandupRisk:
    reasons: list[str] = []
    if story.flagged:
        reasons.append("flagged")
    if story.last_transition_days >= 5:
        reasons.append(f"stalled {story.last_transition_days} days")
    return StandupRisk(
        label=f"{story.key}: {story.summary}",
        rationale=", ".join(reasons),
        evidence_ref=story.evidence_ref,
    )


def _status_snapshot(stories: list[JiraStoryFixture]) -> dict[str, int]:
    counts = Counter(_status_key(story.status) for story in stories)
    return dict(sorted(counts.items()))


def _status_key(status: str) -> str:
    normalized = status.strip().lower().replace(" ", "_")
    if normalized in {"done", "closed", "complete", "completed"}:
        return "done"
    if normalized in {"in_progress", "inprogress", "doing"}:
        return "in_progress"
    if normalized in {"blocked", "flagged"}:
        return "blocked"
    if normalized in {"todo", "to_do", "open"}:
        return "todo"
    return normalized


def _data_quality(case: StandupFixtureCase) -> list[str]:
    messages: list[str] = []
    for tool_status in case.tool_statuses:
        if tool_status.status == ToolStatus.OK:
            continue
        unavailable = f" unavailable: {', '.join(tool_status.unavailable_sources)}" if tool_status.unavailable_sources else ""
        messages.append(f"{tool_status.tool_name} status is {tool_status.status.value}{unavailable}.")
    if not messages:
        messages.append("All synthetic sources are available.")
    return messages

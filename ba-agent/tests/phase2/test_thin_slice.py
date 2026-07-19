from __future__ import annotations

import json
import socket
from pathlib import Path
from typing import Any, cast

import pytest

from ba_agent.phase2.context_memory import ContextMemory, make_synthetic_context
from ba_agent.phase2.discovery import (
    ARTIFACT_VERSION,
    SyntheticGuardError,
    discover_requirements,
)
from ba_agent.phase2.models import DRAFT_ADVISORY_LABEL, NON_APPROVAL_STATEMENT, RequirementDiscoveryOutput
from ba_agent.phase2.traceability import TraceEntry, build_trace_skeleton

FIXTURES_DIR = Path(__file__).parent / "fixtures"
FIXTURE_PATH = FIXTURES_DIR / "P2REQ-001.json"


def _load_fixture(case_id: str = "P2REQ-001") -> dict[str, Any]:
    fixture_path = FIXTURES_DIR / f"{case_id}.json"
    return cast(dict[str, Any], json.loads(fixture_path.read_text(encoding="utf-8")))


def _make_context(fixture: dict[str, object]) -> ContextMemory:
    project_context = fixture["project_context"]
    assert isinstance(project_context, dict)
    project_name = project_context["project_name"]
    assert isinstance(project_name, str)
    return make_synthetic_context(project_name)


def test_thin_slice_returns_valid_output() -> None:
    fixture = _load_fixture()
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    assert isinstance(output, RequirementDiscoveryOutput)
    assert output.artifact_route == "phase2_requirement_discovery"
    assert output.artifact_version == ARTIFACT_VERSION
    assert output.case_id == "P2REQ-001"
    assert output.draft_advisory_label == DRAFT_ADVISORY_LABEL
    assert output.non_approval_statement == NON_APPROVAL_STATEMENT
    assert output.source_register
    assert output.business_problem
    assert output.business_objective
    assert output.facts
    assert output.assumptions
    assert output.inferred_items
    assert output.open_questions
    assert output.conflicts
    assert output.risks_dependencies
    assert output.draft_requirement_candidates
    assert output.draft_story_candidates
    assert output.traceability_skeleton
    assert output.human_review_lanes

    node_types = {node.node_type for node in output.traceability_skeleton}
    assert {"p2-input", "p2-obj", "p2-req-draft", "p2-story-draft"}.issubset(node_types)


def test_thin_slice_evidence_refs_present() -> None:
    fixture = _load_fixture()
    context = _make_context(fixture)
    output = discover_requirements(context, str(fixture["input"]))

    assert output.evidence_refs == ["eval:P2REQ-001"]
    assert all(item.evidence_refs for item in output.facts)
    assert all(ref == "eval:P2REQ-001" for item in output.facts for ref in item.evidence_refs)

    skeleton = build_trace_skeleton(
        [{"kind": "objective", "case_id": "P2REQ-001", "text": "Synthetic objective"}],
        ["eval:P2REQ-001"],
    )
    assert all(isinstance(node, TraceEntry) for node in skeleton)
    assert any(node.node_type == "p2-obj" for node in skeleton)
    assert any(node.node_type == "p2-input" for node in output.traceability_skeleton)


def test_thin_slice_no_live_calls(monkeypatch: pytest.MonkeyPatch) -> None:
    fixture = _load_fixture()
    context = _make_context(fixture)
    calls: list[str] = []

    def forbidden(*_args: object, **_kwargs: object) -> None:
        calls.append("called")
        raise AssertionError("live call attempted")

    monkeypatch.setattr(socket.socket, "connect", forbidden)
    monkeypatch.setattr(socket, "create_connection", forbidden)

    output = discover_requirements(context, json.dumps(fixture))

    assert output.trace_id.startswith("p2-discovery:P2REQ-001")
    assert calls == []


def test_thin_slice_synthetic_guard() -> None:
    fixture = _load_fixture()
    fixture["data_source_mode"] = "live"
    context = _make_context(fixture)

    with pytest.raises(SyntheticGuardError, match="synthetic"):
        discover_requirements(context, json.dumps(fixture))


def test_thin_slice_synthetic_guard_requires_explicit_markers() -> None:
    fixture = _load_fixture()
    fixture.pop("data_source_mode", None)
    fixture.pop("classification", None)
    fixture["input"] = "The team discussed synthetic inventory handling without any explicit markers."
    context = _make_context(fixture)

    with pytest.raises(SyntheticGuardError, match="synthetic"):
        discover_requirements(context, json.dumps(fixture))


def test_thin_slice_preserves_comma_in_stakeholder_name() -> None:
    fixture = _load_fixture()
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    assert "Jordan Kim, Operations Manager" in output.stakeholders
    assert "Jordan Kim" not in output.stakeholders
    assert "Operations Manager" not in output.stakeholders


def test_thin_slice_rejects_mixed_provenance() -> None:
    fixture = _load_fixture()
    fixture["input"] = "\n".join(
        [
            "[SYNTHETIC] META: data_source_mode=live; classification=LIVE; case_id=P2REQ-001",
            "[SYNTHETIC] QUESTION: What defines safety threshold per SKU category?",
        ]
    )
    context = _make_context(fixture)

    with pytest.raises(SyntheticGuardError, match="synthetic"):
        discover_requirements(context, json.dumps(fixture))


@pytest.mark.parametrize(
    "field_path",
    [
        "source_register",
        "fixture_data.source_metadata",
        "project_context.classification_label",
    ],
)
def test_thin_slice_rejects_live_nested_provenance(field_path: str) -> None:
    fixture = _load_fixture()
    if field_path == "source_register":
        fixture["source_register"][0]["classification"] = "LIVE"
    elif field_path == "fixture_data.source_metadata":
        fixture["fixture_data"]["source_metadata"][0]["classification"] = "LIVE"
    else:
        fixture["project_context"]["classification_label"] = "LIVE"

    context = _make_context(fixture)

    with pytest.raises(SyntheticGuardError, match="synthetic"):
        discover_requirements(context, json.dumps(fixture))


def test_thin_slice_rejects_live_project_context_values() -> None:
    fixture = _load_fixture()
    fixture["project_context"]["project_name"] = "Acme Corp"
    fixture["project_context"]["stakeholders"] = ["Acme CFO"]
    context = _make_context(fixture)

    with pytest.raises(SyntheticGuardError, match="synthetic"):
        discover_requirements(context, json.dumps(fixture))


def test_thin_slice_rejects_live_note_provenance() -> None:
    fixture = _load_fixture()
    fixture["input"] = "\n".join(
        [
            "[SYNTHETIC] STAKEHOLDER: Acme CFO",
            "[SYNTHETIC] SOURCE: Acme Corp | Acme CFO | synthetic-2026-07-06T09:15:00Z | LIVE",
        ]
    )
    context = _make_context(fixture)

    with pytest.raises(SyntheticGuardError, match="synthetic"):
        discover_requirements(context, json.dumps(fixture))


def test_thin_slice_rejects_live_note_body() -> None:
    fixture = _load_fixture()
    fixture["input"] = "\n".join(
        [
            "[SYNTHETIC] FACT: Acme Corp is the fictional retail company for this thin slice.",
            "[SYNTHETIC] QUESTION: What defines safety threshold per SKU category?",
        ]
    )
    context = _make_context(fixture)

    with pytest.raises(SyntheticGuardError, match="synthetic"):
        discover_requirements(context, json.dumps(fixture))


def test_build_trace_skeleton_input_candidates_have_unique_ids() -> None:
    skeleton = build_trace_skeleton(
        [
            {"kind": "input", "case_id": "P2REQ-001", "text": "Synthetic intake note"},
        ],
        ["eval:P2REQ-001"],
    )

    trace_ids = [node.trace_id for node in skeleton]
    assert trace_ids == ["p2-input:P2REQ-001:001", "p2-input:P2REQ-001:002"]
    assert len(trace_ids) == len(set(trace_ids))


def test_thin_slice_output_version() -> None:
    fixture = _load_fixture()
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    assert output.artifact_version == "p2-g2-synthetic-thin-slice-v0.1"


def test_thin_slice_regression_conflict_is_preserved() -> None:
    fixture = _load_fixture()
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    assert output.conflicts
    assert any("unresolved" in conflict.description.lower() for conflict in output.conflicts)
    assert any("safety threshold" in conflict.source_a.lower() for conflict in output.conflicts)
    assert any("safety threshold" in conflict.source_b.lower() for conflict in output.conflicts)


def test_thin_slice_regression_missing_rule_becomes_open_question() -> None:
    fixture = _load_fixture()
    fixture["input"] = "\n".join(
        [
            "[SYNTHETIC] META: data_source_mode=synthetic; classification=SYNTHETIC-FICTIONAL; case_id=P2REQ-001",
            "[SYNTHETIC] FACT: Replenish when stock drops below 20% of safety threshold.",
        ]
    )
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    assert output.open_questions
    assert any("safety threshold" in question.question.lower() for question in output.open_questions)
    assert output.inferred_items
    assert any(item.marker == "[inferred]" for item in output.inferred_items)


def test_thin_slice_regression_traceability_links_remain_connected() -> None:
    fixture = _load_fixture()
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    trace_ids = [node.trace_id for node in output.traceability_skeleton]
    assert trace_ids == sorted(set(trace_ids), key=trace_ids.index)

    for node in output.traceability_skeleton:
        if node.node_type == "p2-input":
            continue
        assert node.parent_ids, f"Trace node {node.trace_id} has no parent linkage"


def test_p2req003_conflicting_stakeholder_statements_are_preserved() -> None:
    fixture = _load_fixture("P2REQ-003")
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    assert output.case_id == "P2REQ-003"
    assert output.evidence_refs == ["eval:P2REQ-003"]
    assert "Product Owner [RAJA]" in output.human_review_lanes
    assert "BA SME [RAJA]" in output.human_review_lanes
    assert "Compliance/legal owner [RAJA]" in output.human_review_lanes

    fact_texts = [fact.claim for fact in output.facts]
    assert any("approval step must be removed" in fact for fact in fact_texts)
    assert any("approval step is mandatory for audit" in fact for fact in fact_texts)

    assert output.conflicts
    conflict = output.conflicts[0]
    assert conflict.resolution is None
    assert "Casey Rivera, Product Owner" in conflict.description
    assert "Morgan Lee, Compliance Reviewer" in conflict.description
    assert "approval step must be removed" in conflict.source_a
    assert "approval step is mandatory for audit" in conflict.source_b

    assert output.open_questions
    assert any("approval-step policy decision" in question.question for question in output.open_questions)

    node_types = {node.node_type for node in output.traceability_skeleton}
    assert {"p2-input", "p2-obj", "p2-req-draft", "p2-story-draft", "p2-question"}.issubset(node_types)


def test_p2req004_missing_business_rules_become_questions_not_policy() -> None:
    fixture = _load_fixture("P2REQ-004")
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    assert output.case_id == "P2REQ-004"
    assert output.evidence_refs == ["eval:P2REQ-004"]
    assert "BA SME [RAJA]" in output.human_review_lanes
    assert "Product Owner [RAJA]" in output.human_review_lanes
    assert "QA / AI evaluation reviewer [RAJA]" in output.human_review_lanes
    assert "Architect [RAJA]" in output.human_review_lanes

    fact_texts = [fact.claim for fact in output.facts]
    assert any("auto-mark a repository map as ready" in fact for fact in fact_texts)

    questions = [question.question for question in output.open_questions]
    assert any("confidence threshold" in question for question in questions)
    assert any("fields block repository-map readiness" in question for question in questions)
    assert any("Who approves repository-map readiness" in question for question in questions)

    assert output.inferred_items
    assert all(item.marker == "[inferred]" for item in output.inferred_items)
    assert any("Readiness may depend" in item.item for item in output.inferred_items)

    requirement_text = output.draft_requirement_candidates[0].text
    assert "unresolved until" in requirement_text
    assert "confidence, coverage, ownership, dependency, API, and approval rules are defined" in requirement_text
    assert "100%" not in requirement_text
    assert "approved automatically" not in requirement_text.lower()

    assert output.traceability_skeleton
    question_nodes = [node for node in output.traceability_skeleton if node.node_type == "p2-question"]
    assert len(question_nodes) >= 3
    assert all(node.parent_ids for node in question_nodes)


@pytest.mark.parametrize(
    "case_id",
    ["P2REQ-001", "P2REQ-002", "P2REQ-003", "P2REQ-004", "P2REQ-005", "P2REQ-006", "P2REQ-007", "P2REQ-008"],
)
def test_all_p2req_fixtures_have_executable_discovery_coverage(case_id: str) -> None:
    fixture = _load_fixture(case_id)
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    assert output.case_id == case_id
    assert output.evidence_refs == [f"eval:{case_id}"]
    assert output.source_register
    assert output.facts
    assert all(fact.evidence_refs == [f"eval:{case_id}"] for fact in output.facts)
    assert output.assumptions
    assert output.open_questions
    assert output.risks_dependencies
    assert output.draft_requirement_candidates
    assert output.draft_story_candidates
    assert output.traceability_skeleton
    assert output.non_approval_statement == NON_APPROVAL_STATEMENT

    trace_node_types = {node.node_type for node in output.traceability_skeleton}
    assert {"p2-input", "p2-obj", "p2-req-draft", "p2-story-draft", "p2-question"}.issubset(trace_node_types)


def test_p2req002_support_ticket_cluster_surfaces_risks_and_dependencies() -> None:
    fixture = _load_fixture("P2REQ-002")
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    assert any("billing-adjustment requests" in fact.claim for fact in output.facts)
    assert any("supervisor review" in question.question for question in output.open_questions)
    assert any(item.kind == "risk" and "response-delay" in item.description for item in output.risks_dependencies)
    assert any(item.kind == "dependency" and "Support policy ownership" in item.description for item in output.risks_dependencies)


def test_p2req005_regulatory_summary_routes_without_approving_obligations() -> None:
    fixture = _load_fixture("P2REQ-005")
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    assert "Security/privacy owner [RAJA]" in output.human_review_lanes
    assert "Compliance/legal owner [RAJA]" in output.human_review_lanes
    assert any("legal interpretation is not approved" in fact.claim for fact in output.facts)
    assert any("no regulatory obligation is approved automatically" in assumption for assumption in output.assumptions)
    assert "approved automatically" not in output.draft_requirement_candidates[0].text.lower()


def test_p2req006_product_idea_preserves_trace_skeleton() -> None:
    fixture = _load_fixture("P2REQ-006")
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    requirement = output.draft_requirement_candidates[0]
    story = output.draft_story_candidates[0]

    assert "monthly executive insight digest" in (output.business_problem or "")
    assert requirement.objective_ref == "p2-obj:P2REQ-006:001"
    assert story.requirement_ref == requirement.candidate_id
    assert any("prioritization" in question.question for question in output.open_questions)


def test_p2req007_process_pain_does_not_generate_final_process_map() -> None:
    fixture = _load_fixture("P2REQ-007")
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    assert any("three manual handoffs" in fact.claim for fact in output.facts)
    assert any("future-state process map" in question.question for question in output.open_questions)
    requirement_text = output.draft_requirement_candidates[0].text
    assert "process-gap candidate" in requirement_text
    assert "final process map" not in requirement_text.lower()


def test_p2req008_tool_origin_metadata_conflict_and_staleness_are_preserved() -> None:
    fixture = _load_fixture("P2REQ-008")
    context = _make_context(fixture)
    output = discover_requirements(context, json.dumps(fixture))

    systems = {source.system for source in output.source_register}
    assert {"JiraBoard", "ConfluenceNotes", "TeamsDigest"}.issubset(systems)
    assert all(source.owner == "Avery Stone, Tool Steward" for source in output.source_register)
    assert all(source.timestamp and source.timestamp.startswith("synthetic-") for source in output.source_register)
    assert all(source.retrieved_at and source.retrieved_at.startswith("synthetic-") for source in output.source_register)
    assert all(source.classification == "SYNTHETIC-FICTIONAL" for source in output.source_register)

    assert output.conflicts
    assert any("JiraBoard" in conflict.description for conflict in output.conflicts)
    assert any("ConfluenceNotes" in conflict.description for conflict in output.conflicts)
    assert any("authoritative" in question.question for question in output.open_questions)
    assert any("staleness" in item.item for item in output.inferred_items)

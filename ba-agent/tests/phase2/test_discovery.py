"""Phase 2 discovery smoke and fixture tests.

P2-G1 test stubs — behavioral depth added at P2-G2/P2-G3.

Test coverage (from ``docs/development/p2-g1-technical-baseline.md`` Section 8):
  1. Import smoke test — all Phase 2 modules import without error; no live
     client is instantiated at import time.
  4. Synthetic fixture load — at least one GTS-P2-REQ fixture file loads and
     validates minimum required fields.

Synthetic-only. No live integrations. No network calls.
The ``block_network`` fixture in ``tests/conftest.py`` is autouse and covers
these tests.

Authorization: Synthetic-only; BA-EM-005 = 0; BA-EM-009 = 0.
"""
from __future__ import annotations

import json
from pathlib import Path

import pytest


# ---------------------------------------------------------------------------
# Test 1 — Import smoke test (Section 8, item 1)
# ---------------------------------------------------------------------------

def test_phase2_package_imports() -> None:
    """All Phase 2 modules import without error.

    No live client must be instantiated at import time. If this test fails
    due to an import of a live adapter, it is a BA-EM-009 hard-gate violation.
    """
    from ba_agent.phase2 import router, discovery, models, context_memory, traceability  # noqa: F401


def test_phase2_models_importable() -> None:
    """Key model classes are importable from ba_agent.phase2.models."""
    from ba_agent.phase2.models import (  # noqa: F401
        DiscoveryOutput,
        Phase2RouteDecision,
        ProjectContextMemory,
        TraceNode,
        OpenQuestion,
        RiskDependency,
        SourceRef,
        EvidencedClaim,
        InferredItem,
        Conflict,
        DraftRequirementCandidate,
        DraftStoryCandidate,
    )


def test_phase2_discovery_output_fields_present() -> None:
    """DiscoveryOutput model exposes all required fields defined in Section 4.2."""
    from ba_agent.phase2.models import DiscoveryOutput

    required_fields = {
        "draft_advisory_label",
        "evidence_refs",
        "trace_id",
        "artifact_version",
        "artifact_route",
        "case_id",
        "source_register",
        "business_problem",
        "business_objective",
        "stakeholders",
        "facts",
        "assumptions",
        "inferred_items",
        "open_questions",
        "conflicts",
        "risks_dependencies",
        "draft_requirement_candidates",
        "draft_story_candidates",
        "traceability_skeleton",
        "human_review_lanes",
        "non_approval_statement",
    }
    model_fields = set(DiscoveryOutput.model_fields.keys())
    missing = required_fields - model_fields
    assert not missing, f"DiscoveryOutput missing required fields: {missing}"


def test_draft_advisory_label_fixed_value() -> None:
    """DRAFT_ADVISORY_LABEL constant matches the canonical string."""
    from ba_agent.phase2.models import DRAFT_ADVISORY_LABEL

    assert DRAFT_ADVISORY_LABEL == "DRAFT — ADVISORY ONLY — NOT APPROVED"


def test_non_approval_statement_fixed_value() -> None:
    """NON_APPROVAL_STATEMENT constant matches the canonical string."""
    from ba_agent.phase2.models import NON_APPROVAL_STATEMENT

    assert "draft and advisory" in NON_APPROVAL_STATEMENT.lower()
    assert "human review" in NON_APPROVAL_STATEMENT.lower()


# ---------------------------------------------------------------------------
# Test 4 — Synthetic fixture load (Section 8, item 4)
# ---------------------------------------------------------------------------

FIXTURES_DIR = Path(__file__).parent / "fixtures"


def test_fixtures_directory_exists() -> None:
    """tests/phase2/fixtures/ directory exists."""
    assert FIXTURES_DIR.is_dir(), f"Missing fixtures directory: {FIXTURES_DIR}"


def test_at_least_one_fixture_present() -> None:
    """At least one fixture JSON file exists in tests/phase2/fixtures/."""
    json_files = list(FIXTURES_DIR.glob("*.json"))
    assert json_files, (
        "No GTS-P2-REQ fixture files found in tests/phase2/fixtures/. "
        "Add at least one stub fixture for P2-G1."
    )


@pytest.mark.parametrize("fixture_path", list(FIXTURES_DIR.glob("*.json")))
def test_fixture_parses_and_has_minimum_fields(fixture_path: Path) -> None:
    """Each fixture JSON parses without error and carries minimum required fields.

    Minimum fields: ``case_id``, ``expected_routing``, ``project_context``.
    """
    content = json.loads(fixture_path.read_text(encoding="utf-8"))

    assert "case_id" in content, f"{fixture_path.name}: missing 'case_id'"
    assert "expected_routing" in content, f"{fixture_path.name}: missing 'expected_routing'"
    assert "project_context" in content, f"{fixture_path.name}: missing 'project_context'"

    assert content["expected_routing"] == "phase2_requirement_discovery", (
        f"{fixture_path.name}: expected_routing must be 'phase2_requirement_discovery', "
        f"got '{content['expected_routing']}'"
    )

    ctx = content["project_context"]
    assert "project_name" in ctx, f"{fixture_path.name}: project_context missing 'project_name'"

"""Phase 2 project-context memory schema and serialisation helpers.

P2-G2 thin slice: schema definition plus ``[RAJA]`` sentinel helpers for
synthetic-only requirement discovery. Unknown real-world values remain
``[RAJA]`` and no live persistent enterprise memory is enabled.

Route isolation constraint (BA-EM-009 = 0):
  This module must NOT import from ``ba_agent.router``, ``ba_agent.models``,
  ``ba_agent.standup``, ``ba_agent.mvp``, ``ba_agent.cards``, or
  ``ba_agent.adapters``.

Authorization: Synthetic-only; no live clients; no system-of-record writes.
"""
from __future__ import annotations

from ba_agent.phase2.models import ProjectContextMemory

ContextMemory = ProjectContextMemory

__all__ = [
    "ContextMemory",
    "ProjectContextMemory",
    "RAJA_SENTINEL",
    "is_raja_sentinel",
    "make_synthetic_context",
]

RAJA_SENTINEL: str = "[RAJA]"


def is_raja_sentinel(value: str | None) -> bool:
    """Return True if *value* is the ``[RAJA]`` owner-review sentinel."""
    return value is not None and value.strip() == RAJA_SENTINEL


def make_synthetic_context(project_name: str) -> ProjectContextMemory:
    """Return a minimal ``ProjectContextMemory`` with all unknowns set to
    ``[RAJA]`` and synthetic placeholder values where required.

    Intended for use in GTS-P2-REQ fixture loading and P2-G1 smoke tests.
    Full schema population is deferred to P2-G2.
    """
    cleaned_project_name = project_name.strip()
    if not cleaned_project_name:
        raise ValueError("project_name must be a non-empty synthetic value")

    return ProjectContextMemory(
        project_name=cleaned_project_name,
        business_domain=RAJA_SENTINEL,
        stakeholders=[],
        target_users=[],
        source_systems=["[SYNTHETIC] StoreTrak"],
        delivery_methodology=RAJA_SENTINEL,
        known_business_rules=[],
        constraints=[],
        definition_of_ready=RAJA_SENTINEL,
        definition_of_done=RAJA_SENTINEL,
        jira_project_key="SYNTH-PRJ-001",
        confluence_space="SYNTH-SPACE-001",
        approved_artifact_templates=[],
        classification_label="SYNTHETIC-FICTIONAL",
        retention_rule=RAJA_SENTINEL,
        context_owner=RAJA_SENTINEL,
        last_reviewed_by=RAJA_SENTINEL,
    )

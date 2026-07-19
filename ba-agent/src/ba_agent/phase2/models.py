"""Phase 2 output-contract Pydantic models.

Implements the field definitions from ``docs/development/p2-g1-technical-baseline.md``
Section 4 (output contract) and Section 5 (project-context memory schema).

All outputs produced by the ``phase2_requirement_discovery`` route must carry
every field defined on ``DiscoveryOutput``. Absence of any required field is a
``BA-EM-007`` structure-conformance failure.

Route isolation constraints (BA-EM-009 = 0):
  - This module must NOT import from ``ba_agent.router``, ``ba_agent.models``,
    ``ba_agent.standup``, ``ba_agent.mvp``, ``ba_agent.cards``, or
    ``ba_agent.adapters``.
  - ``ba_agent.gateway`` controls carry forward but are not imported here —
    enforcement is at the router entry point.

Scope: P2-G1 scaffold stub — field definitions only; no validation logic yet.
Authorization: Synthetic-only; no live clients; no system-of-record writes.
"""
from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, ConfigDict, Field


DRAFT_ADVISORY_LABEL: str = "DRAFT — ADVISORY ONLY — NOT APPROVED"
NON_APPROVAL_STATEMENT: str = (
    "This output is draft and advisory. No requirement, story, or decision in "
    "this output is approved. Human review is required before any downstream use."
)


class SourceRef(BaseModel):
    """A reference to a synthetic source document or signal."""

    model_config = ConfigDict(frozen=True)

    system: str
    owner: str
    timestamp: str | None = None
    retrieved_at: str | None = None
    classification: str | None = None


class EvidencedClaim(BaseModel):
    """A fact claim that carries its own evidence references."""

    model_config = ConfigDict(frozen=True)

    claim: str
    evidence_refs: list[str] = Field(default_factory=list)


class InferredItem(BaseModel):
    """An item derived by inference; must never be promoted to a fact."""

    model_config = ConfigDict(frozen=True)

    item: str
    marker: Literal["[inferred]"] = "[inferred]"
    basis: str | None = None


class OpenQuestion(BaseModel):
    """A clarification question that must be answered before the item can advance."""

    model_config = ConfigDict(frozen=True)

    question: str
    decision_owner: str  # specific role, or "[RAJA]" if unknown


class Conflict(BaseModel):
    """Two or more source statements that cannot be simultaneously true."""

    model_config = ConfigDict(frozen=True)

    description: str
    source_a: str
    source_b: str
    resolution: None = None  # always None in first slice — no silent resolution


class RiskDependency(BaseModel):
    """A delivery or analysis risk or inter-requirement dependency."""

    model_config = ConfigDict(frozen=True)

    kind: Literal["risk", "dependency"]
    description: str
    source_signal: str | None = None  # or "[inferred]"
    marker: str | None = None  # "[inferred]" when source_signal is absent


class TraceNode(BaseModel):
    """One node in the traceability skeleton.

    ID patterns (from Section 4.3):
      p2-input:{case_id}:{n}        — synthetic evidence item
      p2-obj:{case_id}:{n}          — draft business objective
      p2-req-draft:{case_id}:{n}    — draft requirement candidate (not approved)
      p2-story-draft:{case_id}:{n}  — draft story candidate (not accepted scope)
      p2-question:{case_id}:{n}     — open question
      p2-risk:{case_id}:{n}         — risk / dependency
    """

    model_config = ConfigDict(frozen=True)

    trace_id: str
    node_type: Literal[
        "p2-input",
        "p2-obj",
        "p2-req-draft",
        "p2-story-draft",
        "p2-question",
        "p2-risk",
    ]
    label: str
    parent_ids: list[str] = Field(default_factory=list)


class DraftRequirementCandidate(BaseModel):
    """A draft/advisory requirement candidate — not approved, not backlog scope."""

    model_config = ConfigDict(frozen=True)

    candidate_id: str  # p2-req-draft:{case_id}:{n}
    text: str
    evidence_refs: list[str] = Field(default_factory=list)
    objective_ref: str | None = None  # p2-obj:{case_id}:{n}
    marker: Literal["draft/advisory"] = "draft/advisory"


class DraftStoryCandidate(BaseModel):
    """A draft/advisory story skeleton — optional; not accepted backlog scope."""

    model_config = ConfigDict(frozen=True)

    candidate_id: str  # p2-story-draft:{case_id}:{n}
    text: str
    requirement_ref: str | None = None  # p2-req-draft:{case_id}:{n}
    marker: Literal["draft/advisory"] = "draft/advisory"


class ProjectContextMemory(BaseModel):
    """Project-context memory schema (first slice: schema definition only).

    All unknown values carry the ``[RAJA]`` sentinel. No live persistent
    enterprise memory is enabled at P2-G1 through P2-G4.

    Source: ``docs/development/p2-g1-technical-baseline.md`` Section 5.
    """

    model_config = ConfigDict(frozen=True)

    project_name: str
    business_domain: str = "[RAJA]"
    stakeholders: list[str] = Field(default_factory=list)
    target_users: list[str] = Field(default_factory=list)
    source_systems: list[str] = Field(default_factory=list)
    delivery_methodology: str | None = None  # "[RAJA]" unless synthetic case states it
    known_business_rules: list[str] = Field(default_factory=list)
    constraints: list[str] = Field(default_factory=list)
    definition_of_ready: str | None = None  # "[RAJA]"
    definition_of_done: str | None = None  # "[RAJA]"
    jira_project_key: str | None = None  # synthetic placeholder only
    confluence_space: str | None = None  # synthetic placeholder only
    approved_artifact_templates: list[str] = Field(default_factory=list)  # "[RAJA]"
    classification_label: str | None = None  # "[RAJA]"
    retention_rule: str | None = None  # "[RAJA]"
    context_owner: str = "[RAJA]"
    last_reviewed_by: str | None = None  # "[RAJA]"


class Phase2RouteDecision(BaseModel):
    """Decision returned by the Phase 2 router.

    The router either confirms ``phase2_requirement_discovery`` routing or
    returns a blocked/rejected response. It never touches any MVP route.
    """

    model_config = ConfigDict(frozen=True)

    route: Literal["phase2_requirement_discovery", "blocked"]
    reason: str
    blocked: bool = False


class DiscoveryOutput(BaseModel):
    """Full output contract for the ``phase2_requirement_discovery`` route.

    Every field listed here is required unless explicitly typed ``| None`` or
    ``list[...]`` with a default. Absence of any non-optional field is a
    ``BA-EM-007`` structure-conformance failure.

    Source: ``docs/development/p2-g1-technical-baseline.md`` Section 4.2.
    """

    model_config = ConfigDict(frozen=True)

    # Hard-coded label and statement — must match exact strings
    draft_advisory_label: str = DRAFT_ADVISORY_LABEL
    non_approval_statement: str = NON_APPROVAL_STATEMENT

    # Provenance
    trace_id: str
    artifact_version: str
    artifact_route: Literal["phase2_requirement_discovery"]
    case_id: str | None = None
    evidence_refs: list[str] = Field(default_factory=list)
    source_register: list[SourceRef] = Field(default_factory=list)

    # Discovery content
    business_problem: str | None = None
    business_objective: str | None = None
    stakeholders: list[str] = Field(default_factory=list)

    facts: list[EvidencedClaim] = Field(default_factory=list)
    assumptions: list[str] = Field(default_factory=list)
    inferred_items: list[InferredItem] = Field(default_factory=list)
    open_questions: list[OpenQuestion] = Field(default_factory=list)
    conflicts: list[Conflict] = Field(default_factory=list)
    risks_dependencies: list[RiskDependency] = Field(default_factory=list)

    # Candidates (draft/advisory only)
    draft_requirement_candidates: list[DraftRequirementCandidate] = Field(
        default_factory=list
    )
    draft_story_candidates: list[DraftStoryCandidate] = Field(default_factory=list)

    # Traceability
    traceability_skeleton: list[TraceNode] = Field(default_factory=list)

    # Review routing
    human_review_lanes: list[str] = Field(default_factory=list)


# Backwards-compatible aliases used by the P2-G2 thin slice implementation.
ContextMemory = ProjectContextMemory
RequirementDiscoveryOutput = DiscoveryOutput
TraceEntry = TraceNode

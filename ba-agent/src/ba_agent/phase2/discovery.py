"""Phase 2 requirement-discovery implementation.

P2-G2 thin slice: parse one synthetic session, separate facts from
assumptions and ``[inferred]`` items, surface open questions/conflicts, and
assemble a draft/advisory ``RequirementDiscoveryOutput``.

No write-like actions, live integrations, approvals, or system-of-record
updates occur here.
"""
from __future__ import annotations

import json
import re
from collections.abc import Iterable, Mapping
from typing import Any

from ba_agent.phase2.context_memory import ContextMemory, make_synthetic_context
from ba_agent.phase2.models import (
    Conflict,
    DraftRequirementCandidate,
    DraftStoryCandidate,
    EvidencedClaim,
    InferredItem,
    OpenQuestion,
    RequirementDiscoveryOutput,
    RiskDependency,
    SourceRef,
)
from ba_agent.phase2.traceability import build_trace_skeleton, make_trace_id

ARTIFACT_VERSION = "p2-g2-synthetic-thin-slice-v0.1"

DEFAULT_REVIEW_LANES = [
    "BA SME [RAJA]",
    "Product Owner [RAJA]",
    "QA / AI evaluation reviewer [RAJA]",
    "Architect [RAJA]",
    "Security/privacy owner [RAJA]",
    "Compliance/legal owner [RAJA]",
    "Tool owners [RAJA]",
]

__all__ = [
    "ARTIFACT_VERSION",
    "ContextMemory",
    "RequirementDiscoveryOutput",
    "SyntheticGuardError",
    "discover_requirements",
    "run_discovery",
]


class SyntheticGuardError(ValueError):
    """Raised when the synthetic-only guard is violated."""


def discover_requirements(context: ContextMemory, session_notes: str) -> RequirementDiscoveryOutput:
    """Parse a synthetic session and assemble a draft requirement output."""
    payload = _load_payload(session_notes)
    notes_text = _extract_notes_text(payload, session_notes)
    project_context = _as_mapping(payload.get("project_context"))
    allowed_project_stakeholders = _merge_unique(
        _coerce_string_list(project_context.get("stakeholders")),
        _coerce_string_list(context.stakeholders),
    )
    allowed_source_systems = _merge_unique(
        _coerce_string_list(project_context.get("source_systems")),
        _coerce_string_list(context.source_systems),
    )
    allowed_note_entities = _merge_unique(
        _coerce_string_list(project_context.get("project_name")),
        _coerce_string_list(project_context.get("stakeholders")),
        _coerce_string_list(project_context.get("source_systems")),
        _coerce_string_list(project_context.get("known_business_rules")),
        _coerce_string_list(project_context.get("constraints")),
        _coerce_string_list(context.project_name),
        _coerce_string_list(context.stakeholders),
        _coerce_string_list(context.source_systems),
        _coerce_string_list(context.known_business_rules),
        _coerce_string_list(context.constraints),
    )
    enforce_note_provenance = bool(project_context)
    note_meta = _parse_note_metadata(
        notes_text,
        allowed_source_systems,
        allowed_project_stakeholders,
        allowed_note_entities,
        enforce_note_provenance=enforce_note_provenance,
    )
    _validate_synthetic_guard(payload, note_meta, notes_text)

    case_id = _extract_case_id(payload, note_meta, notes_text)
    evidence_refs = _extract_evidence_refs(payload, note_meta, notes_text, case_id)
    source_register = _build_source_register(payload, note_meta, context, project_context)
    review_lanes = _coerce_string_list(
        payload.get("expected_review_lanes")
        or payload.get("human_review_lanes")
        or _nested_value(payload, "expected_output_characteristics", "human_review_lanes")
    ) or list(DEFAULT_REVIEW_LANES)

    project_name = _first_text(
        _clean_text(str(project_context.get("project_name"))) if project_context.get("project_name") is not None else None,
        _clean_text(context.project_name),
        "Synthetic project",
    )
    stakeholders = _merge_unique(
        _coerce_string_list(project_context.get("stakeholders")),
        _coerce_string_list(context.stakeholders),
        _extract_people_from_notes(
            notes_text,
            allowed_project_stakeholders,
            enforce_note_provenance=enforce_note_provenance,
        ),
    )
    source_systems = _merge_unique(
        _coerce_string_list(project_context.get("source_systems")),
        _coerce_string_list(context.source_systems),
        [source.system for source in source_register],
    )
    known_business_rules = _merge_unique(
        _coerce_string_list(project_context.get("known_business_rules")),
        _coerce_string_list(context.known_business_rules),
        note_meta["facts"],
    )
    constraints = _merge_unique(
        _coerce_string_list(project_context.get("constraints")),
        _coerce_string_list(context.constraints),
        note_meta["dependencies"],
    )

    facts = _build_fact_claims(
        evidence_refs,
        project_name,
        stakeholders,
        source_register,
        source_systems,
        known_business_rules,
        constraints,
        note_meta["facts"],
    )
    assumptions = _merge_unique(
        note_meta["assumptions"],
        _coerce_string_list(_nested_value(payload, "expected_output_characteristics", "assumptions")),
    )
    if not assumptions:
        assumptions = [
            "The thin slice stays draft/advisory and no replenishment action is automated.",
        ]

    inferred_items = _merge_inferred_items(
        note_meta["inferred"],
        note_meta["questions"],
        facts,
    )
    open_questions = _build_open_questions(
        note_meta["questions"],
        payload,
    )
    conflicts = _build_conflicts(
        note_meta["conflicts"],
        facts,
        open_questions,
    )
    risks_dependencies = _build_risks_and_dependencies(
        note_meta["risks"],
        constraints,
        open_questions,
        evidence_refs,
    )

    business_problem = _first_text(
        _clean_text(str(_nested_value(payload, "expected_output_characteristics", "business_problem")))
        if _nested_value(payload, "expected_output_characteristics", "business_problem") is not None
        else None,
        f"{project_name} needs a draft replenishment discovery path in StoreTrak while the safety threshold per SKU category remains unresolved.",
    )
    business_objective = _first_text(
        _clean_text(str(_nested_value(payload, "expected_output_characteristics", "business_objective")))
        if _nested_value(payload, "expected_output_characteristics", "business_objective") is not None
        else None,
        f"Produce a draft advisory replenishment summary for {project_name} with evidence-linked facts, assumptions, and open questions.",
    )

    objective_trace_id = make_trace_id("obj", case_id, 1)
    requirement_trace_id = make_trace_id("req-draft", case_id, 1)
    story_trace_id = make_trace_id("story-draft", case_id, 1)

    requirement_text = _first_text(
        _clean_text(str(_nested_value(payload, "expected_output_characteristics", "draft_requirement")))
        if _nested_value(payload, "expected_output_characteristics", "draft_requirement") is not None
        else None,
        "Draft requirement: StoreTrak should flag replenishment review when stock drops below 20% of the safety threshold.",
    )
    story_text = _first_text(
        _clean_text(str(_nested_value(payload, "expected_output_characteristics", "draft_story")))
        if _nested_value(payload, "expected_output_characteristics", "draft_story") is not None
        else None,
        "As a synthetic Operations Manager, I want StoreTrak to surface low-stock replenishment review items so that I can review category-specific thresholds.",
    )

    trace_candidates: list[dict[str, Any]] = [
        {"kind": "objective", "case_id": case_id, "text": business_objective},
        {
            "kind": "requirement",
            "case_id": case_id,
            "text": requirement_text,
            "parent_ids": [objective_trace_id],
        },
        {
            "kind": "story",
            "case_id": case_id,
            "text": story_text,
            "parent_ids": [requirement_trace_id],
        },
    ]
    trace_candidates.extend(
        {"kind": "question", "case_id": case_id, "text": question.question}
        for question in open_questions
    )
    trace_candidates.extend(
        {"kind": "risk", "case_id": case_id, "text": item.description}
        for item in risks_dependencies
    )

    traceability_skeleton = build_trace_skeleton(trace_candidates, evidence_refs)

    return RequirementDiscoveryOutput(
        trace_id=f"p2-discovery:{case_id}:001",
        artifact_version=ARTIFACT_VERSION,
        artifact_route="phase2_requirement_discovery",
        case_id=case_id,
        evidence_refs=evidence_refs,
        source_register=source_register,
        business_problem=business_problem,
        business_objective=business_objective,
        stakeholders=stakeholders,
        facts=facts,
        assumptions=assumptions,
        inferred_items=inferred_items,
        open_questions=open_questions,
        conflicts=conflicts,
        risks_dependencies=risks_dependencies,
        draft_requirement_candidates=[
            DraftRequirementCandidate(
                candidate_id=requirement_trace_id,
                text=requirement_text,
                evidence_refs=list(evidence_refs),
                objective_ref=objective_trace_id,
            )
        ],
        draft_story_candidates=[
            DraftStoryCandidate(
                candidate_id=story_trace_id,
                text=story_text,
                requirement_ref=requirement_trace_id,
            )
        ],
        traceability_skeleton=traceability_skeleton,
        human_review_lanes=review_lanes,
    )


def run_discovery(case_id: str | None, raw_input: str) -> RequirementDiscoveryOutput:
    """Backward-compatible wrapper around ``discover_requirements``."""
    payload = _load_payload(raw_input)
    project_context = _as_mapping(payload.get("project_context"))
    allowed_project_stakeholders = _merge_unique(
        _coerce_string_list(project_context.get("stakeholders")),
    )
    allowed_source_systems = _merge_unique(
        _coerce_string_list(project_context.get("source_systems")),
    )
    allowed_note_entities = _merge_unique(
        _coerce_string_list(project_context.get("project_name")),
        _coerce_string_list(project_context.get("stakeholders")),
        _coerce_string_list(project_context.get("source_systems")),
        _coerce_string_list(project_context.get("known_business_rules")),
        _coerce_string_list(project_context.get("constraints")),
    )
    enforce_note_provenance = bool(project_context)
    derived_case_id = case_id or _extract_case_id(
        payload,
        _parse_note_metadata(
            _extract_notes_text(payload, raw_input),
            allowed_source_systems,
            allowed_project_stakeholders,
            allowed_note_entities,
            enforce_note_provenance=enforce_note_provenance,
        ),
        raw_input,
    )
    project_name = _first_text(
        _clean_text(str(_nested_value(payload, "project_context", "project_name")))
        if _nested_value(payload, "project_context", "project_name") is not None
        else None,
        _clean_text(derived_case_id) if derived_case_id else None,
        "Synthetic project",
    )
    context = make_synthetic_context(project_name)
    return discover_requirements(context, raw_input)


def _load_payload(session_notes: str) -> dict[str, Any]:
    try:
        loaded = json.loads(session_notes)
    except json.JSONDecodeError:
        return {}
    return loaded if isinstance(loaded, dict) else {}


def _extract_notes_text(payload: Mapping[str, Any], session_notes: str) -> str:
    raw = payload.get("input")
    if isinstance(raw, str) and raw.strip():
        return raw
    raw = payload.get("session_notes")
    if isinstance(raw, str) and raw.strip():
        return raw
    return session_notes


def _validate_synthetic_guard(payload: Mapping[str, Any], note_meta: Mapping[str, list[str]], notes_text: str) -> None:
    data_source_modes = _merge_unique(
        _coerce_string_list(payload.get("data_source_mode")),
        note_meta["data_source_mode"],
    )
    if data_source_modes and any(value.lower() != "synthetic" for value in data_source_modes):
        raise SyntheticGuardError("Phase 2 discovery requires data_source_mode=synthetic")

    classifications = _merge_unique(
        _coerce_string_list(payload.get("classification")),
        note_meta["classification"],
    )
    if classifications and any(
        not _is_explicit_synthetic_classification(classification)
        for classification in classifications
    ):
        raise SyntheticGuardError("Phase 2 discovery requires synthetic classification")

    if not _contains_synthetic_marker(notes_text):
        raise SyntheticGuardError("Phase 2 discovery requires synthetic-only session notes")

    project_context = _as_mapping(payload.get("project_context"))
    classification_label = _string_or_none(project_context.get("classification_label"))
    if classification_label and not _is_explicit_synthetic_classification(classification_label):
        raise SyntheticGuardError("Phase 2 discovery requires synthetic project classification")

    _validate_project_context_provenance(project_context)
    _validate_nested_provenance(payload)


def _extract_case_id(payload: Mapping[str, Any], note_meta: Mapping[str, list[str]], notes_text: str) -> str:
    candidates: list[str] = []
    for key in ("case_id", "caseId", "expected_case_id"):
        raw = payload.get(key)
        if isinstance(raw, str):
            candidates.append(raw)
    candidates.extend(note_meta["case_ids"])
    candidates.extend(_extract_refs_from_text(notes_text))

    for candidate in candidates:
        normalized = _normalize_case_id(candidate)
        if normalized and _looks_like_case_id(normalized):
            return normalized

    return "P2REQ-001"


def _normalize_case_id(value: str) -> str:
    match = re.search(r"(P2REQ-\d{3})", value)
    if match:
        return match.group(1)
    match = re.search(r"([A-Z]+-\d{3,})", value)
    if match:
        return match.group(1)
    return ""


def _looks_like_case_id(value: str) -> bool:
    return bool(re.fullmatch(r"(?:P2REQ-\d{3}|[A-Z]+-\d{3,})", value))


def _extract_evidence_refs(
    payload: Mapping[str, Any],
    note_meta: Mapping[str, list[str]],
    notes_text: str,
    case_id: str,
) -> list[str]:
    refs: list[str] = []
    refs.extend(_coerce_string_list(payload.get("evidence_refs")))
    refs.extend(_coerce_string_list(payload.get("expected_evidence_refs")))
    refs.extend(note_meta["evidence_refs"])
    refs.extend(_extract_source_metadata_refs(payload))
    refs.extend(_extract_refs_from_text(notes_text))
    if not refs:
        refs.append(f"eval:{case_id}")
    return _merge_unique(refs)


def _parse_note_metadata(
    notes_text: str,
    allowed_source_systems: list[str],
    allowed_stakeholders: list[str],
    allowed_note_entities: list[str],
    *,
    enforce_note_provenance: bool,
) -> dict[str, list[str]]:
    parsed: dict[str, list[str]] = {
        "facts": [],
        "assumptions": [],
        "inferred": [],
        "questions": [],
        "conflicts": [],
        "risks": [],
        "dependencies": [],
        "evidence_refs": [],
        "source_systems": [],
        "owners": [],
        "classification": [],
        "data_source_mode": [],
        "case_ids": [],
    }

    for raw_line in notes_text.splitlines():
        line = _strip_synthetic_prefix(raw_line)
        if not line:
            continue
        label, sep, body = line.partition(":")
        if not sep:
            continue
        label_upper = label.strip().upper()
        body = body.strip()

        if label_upper == "META":
            parsed["data_source_mode"].extend(_extract_key_value(body, "data_source_mode"))
            parsed["classification"].extend(_extract_key_value(body, "classification"))
            parsed["case_ids"].extend(_extract_key_value(body, "case_id"))
            continue

        if label_upper in {"EVIDENCE REF", "EVIDENCE", "TRACE"}:
            parsed["evidence_refs"].extend(_extract_refs_from_text(body))
            continue

        if label_upper == "SOURCE":
            source = _parse_source_line(
                body,
                allowed_source_systems,
                allowed_stakeholders,
                enforce_note_provenance=enforce_note_provenance,
            )
            if source is not None:
                parsed["source_systems"].append(source.system)
                parsed["owners"].append(source.owner)
                if source.classification:
                    parsed["classification"].append(source.classification)
            continue

        if label_upper == "FACT":
            _validate_note_body_provenance(
                body,
                allowed_note_entities,
                "fact",
                enforce_note_provenance=enforce_note_provenance,
            )
            parsed["facts"].append(_clean_text(body))
        elif label_upper == "ASSUMPTION":
            _validate_note_body_provenance(
                body,
                allowed_note_entities,
                "assumption",
                enforce_note_provenance=enforce_note_provenance,
            )
            parsed["assumptions"].append(_clean_text(body))
        elif label_upper == "INFERRED":
            _validate_note_body_provenance(
                body,
                allowed_note_entities,
                "inferred note",
                enforce_note_provenance=enforce_note_provenance,
            )
            parsed["inferred"].append(_clean_text(body))
        elif label_upper == "QUESTION":
            _validate_note_body_provenance(
                body,
                allowed_note_entities,
                "question",
                enforce_note_provenance=enforce_note_provenance,
            )
            parsed["questions"].append(_clean_text(body))
        elif label_upper == "CONFLICT":
            _validate_note_body_provenance(
                body,
                allowed_note_entities,
                "conflict",
                enforce_note_provenance=enforce_note_provenance,
            )
            parsed["conflicts"].append(_clean_text(body))
        elif label_upper == "RISK":
            _validate_note_body_provenance(
                body,
                allowed_note_entities,
                "risk",
                enforce_note_provenance=enforce_note_provenance,
            )
            parsed["risks"].append(_clean_text(body))
        elif label_upper == "DEPENDENCY":
            _validate_note_body_provenance(
                body,
                allowed_note_entities,
                "dependency",
                enforce_note_provenance=enforce_note_provenance,
            )
            parsed["dependencies"].append(_clean_text(body))

    return {key: _merge_unique(values) for key, values in parsed.items()}


def _build_source_register(
    payload: Mapping[str, Any],
    note_meta: Mapping[str, list[str]],
    context: ContextMemory,
    project_context: Mapping[str, Any],
) -> list[SourceRef]:
    candidate_records: list[Mapping[str, Any]] = []

    source_register = payload.get("source_register")
    if isinstance(source_register, list):
        candidate_records.extend(record for record in source_register if isinstance(record, Mapping))

    fixture_data = payload.get("fixture_data")
    if isinstance(fixture_data, Mapping):
        source_metadata = fixture_data.get("source_metadata")
        if isinstance(source_metadata, list):
            candidate_records.extend(record for record in source_metadata if isinstance(record, Mapping))

    if not candidate_records and note_meta["source_systems"]:
        candidate_records.append(
            {
                "system": note_meta["source_systems"][0],
                "owner": note_meta["owners"][0] if note_meta["owners"] else context.context_owner,
                "timestamp": "synthetic-2026-07-06T09:15:00Z",
                "classification": note_meta["classification"][0] if note_meta["classification"] else "SYNTHETIC-FICTIONAL",
            }
        )

    if not candidate_records:
        candidate_records.append(
            {
                "system": context.source_systems[0] if context.source_systems else "StoreTrak",
                "owner": context.stakeholders[0] if context.stakeholders else context.context_owner,
                "timestamp": "synthetic-2026-07-06T09:15:00Z",
                "classification": "SYNTHETIC-FICTIONAL",
            }
        )

    source_refs: list[SourceRef] = []
    seen: set[tuple[str, str, str | None, str | None, str | None]] = set()
    for record in candidate_records:
        system = _first_text(_string_or_none(record.get("system")), _string_or_none(record.get("source_system")), "StoreTrak")
        owner = _first_text(_string_or_none(record.get("owner")), _string_or_none(record.get("source_owner")), context.context_owner)
        timestamp = _string_or_none(record.get("timestamp"))
        retrieved_at = _string_or_none(record.get("retrieved_at"))
        classification = _string_or_none(record.get("classification")) or _string_or_none(record.get("confidentiality"))
        key = (system, owner, timestamp, retrieved_at, classification)
        if key in seen:
            continue
        seen.add(key)
        source_refs.append(
            SourceRef(
                system=_clean_text(system),
                owner=_clean_text(owner),
                timestamp=_clean_text(timestamp) if timestamp else None,
                retrieved_at=_clean_text(retrieved_at) if retrieved_at else None,
                classification=_clean_text(classification) if classification else None,
            )
        )

    return source_refs


def _build_fact_claims(
    evidence_refs: list[str],
    project_name: str,
    stakeholders: list[str],
    source_register: list[SourceRef],
    source_systems: list[str],
    known_business_rules: list[str],
    constraints: list[str],
    note_facts: list[str],
) -> list[EvidencedClaim]:
    claim_texts: list[str] = []

    for fact in note_facts:
        _append_unique(claim_texts, fact)

    _append_unique(
        claim_texts,
        f"{project_name} is the fictional retail company in this synthetic slice.",
    )
    for stakeholder in stakeholders:
        _append_unique(claim_texts, f"{stakeholder} is the synthetic stakeholder for this slice.")
    for source in source_register:
        _append_unique(claim_texts, f"{source.system} is the fictional internal source system.")
    for source_system in source_systems:
        _append_unique(claim_texts, f"{source_system} is a synthetic system reference.")
    for rule in known_business_rules:
        _append_unique(claim_texts, rule)
    for constraint in constraints:
        _append_unique(claim_texts, constraint)

    return [EvidencedClaim(claim=claim, evidence_refs=list(evidence_refs)) for claim in claim_texts]


def _merge_inferred_items(
    explicit_inferred: list[str],
    questions: list[str],
    facts: list[EvidencedClaim],
) -> list[InferredItem]:
    inferred_texts: list[str] = []
    for item in explicit_inferred:
        _append_unique(inferred_texts, item)

    threshold_fact = next((fact.claim for fact in facts if "20%" in fact.claim and "safety threshold" in fact.claim.lower()), "")
    threshold_question = next((question for question in questions if "safety threshold" in question.lower()), "")
    if threshold_question:
        _append_unique(
            inferred_texts,
            "SKU categories may need separate threshold tuning because the safety-threshold definition is category-specific.",
        )
    elif threshold_fact:
        _append_unique(
            inferred_texts,
            "The replenishment review likely needs a category-specific threshold definition.",
        )

    return [InferredItem(item=item, basis="Synthetic thin-slice inference from labeled notes.") for item in inferred_texts]


def _build_open_questions(question_texts: list[str], payload: Mapping[str, Any]) -> list[OpenQuestion]:
    questions = list(question_texts)
    fallback = _coerce_string_list(_nested_value(payload, "expected_output_characteristics", "open_questions"))
    for question in fallback:
        _append_unique(questions, question)

    if not questions:
        questions.append("What defines safety threshold per SKU category?")

    return [OpenQuestion(question=question, decision_owner="[RAJA]") for question in questions]


def _build_conflicts(
    explicit_conflicts: list[str],
    facts: list[EvidencedClaim],
    open_questions: list[OpenQuestion],
) -> list[Conflict]:
    conflicts: list[Conflict] = []

    for description in explicit_conflicts:
        conflicts.append(
            Conflict(
                description=description,
                source_a=description,
                source_b=description,
            )
        )

    threshold_fact = next((fact for fact in facts if "20%" in fact.claim and "safety threshold" in fact.claim.lower()), None)
    threshold_question = next((question for question in open_questions if "safety threshold" in question.question.lower()), None)
    if threshold_fact is not None and threshold_question is not None:
        conflicts.append(
            Conflict(
                description="The 20% replenishment trigger is explicit, but the safety-threshold definition per SKU category is unresolved.",
                source_a=threshold_fact.claim,
                source_b=threshold_question.question,
            )
        )

    return conflicts


def _build_risks_and_dependencies(
    explicit_risks: list[str],
    constraints: list[str],
    open_questions: list[OpenQuestion],
    evidence_refs: list[str],
) -> list[RiskDependency]:
    items: list[RiskDependency] = []
    for risk in explicit_risks:
        items.append(
            RiskDependency(
                kind="risk",
                description=risk,
                source_signal=risk,
            )
        )

    if constraints:
        for constraint in constraints:
            items.append(
                RiskDependency(
                    kind="dependency",
                    description=constraint,
                    source_signal=constraint,
                )
            )
    else:
        items.append(
            RiskDependency(
                kind="dependency",
                description="StoreTrak API v2 availability is required for the synthetic thin slice.",
                source_signal="Must integrate with StoreTrak API v2",
            )
        )

    if open_questions:
        items.append(
            RiskDependency(
                kind="risk",
                description="The safety-threshold definition per SKU category remains unresolved.",
                source_signal=open_questions[0].question,
            )
        )

    if evidence_refs and not any(item.source_signal == "Must integrate with StoreTrak API v2" for item in items):
        items.append(
            RiskDependency(
                kind="dependency",
                description="The synthetic slice should remain grounded in evidence refs and source metadata.",
                source_signal=evidence_refs[0],
            )
        )

    return items


def _extract_people_from_notes(
    notes_text: str,
    allowed_stakeholders: list[str],
    *,
    enforce_note_provenance: bool,
) -> list[str]:
    people: list[str] = []
    allowed = {value.strip() for value in allowed_stakeholders if value.strip()}
    for line in notes_text.splitlines():
        stripped = _strip_synthetic_prefix(line)
        if not stripped.upper().startswith("STAKEHOLDER:"):
            continue
        _, _, body = stripped.partition(":")
        for person in re.split(r"\s*;\s*", body):
            raw_person = person.strip()
            if not raw_person:
                continue
            cleaned = _clean_text(raw_person)
            if enforce_note_provenance and not (
                _contains_synthetic_marker(raw_person)
                or "[RAJA]" in raw_person
                or cleaned in allowed
            ):
                raise SyntheticGuardError("Phase 2 discovery requires synthetic stakeholder provenance")
            if cleaned:
                people.append(cleaned)
    return _merge_unique(people)


def _parse_source_line(
    body: str,
    allowed_source_systems: list[str],
    allowed_stakeholders: list[str],
    *,
    enforce_note_provenance: bool,
) -> SourceRef | None:
    raw_parts = [part.strip() for part in body.split("|")]
    if not raw_parts or not raw_parts[0]:
        return None

    allowed_systems = {value.strip() for value in allowed_source_systems if value.strip()}
    allowed_people = {value.strip() for value in allowed_stakeholders if value.strip()}

    system = raw_parts[0]
    owner = raw_parts[1] if len(raw_parts) > 1 and raw_parts[1] else "[RAJA]"
    timestamp = raw_parts[2] if len(raw_parts) > 2 and raw_parts[2] else None
    classification = raw_parts[3] if len(raw_parts) > 3 and raw_parts[3] else None

    if enforce_note_provenance:
        if not (_contains_synthetic_marker(system) or "[RAJA]" in system or _clean_text(system) in allowed_systems):
            raise SyntheticGuardError("Phase 2 discovery requires synthetic source provenance")
        if not (_contains_synthetic_marker(owner) or "[RAJA]" in owner or _clean_text(owner) in allowed_people):
            raise SyntheticGuardError("Phase 2 discovery requires synthetic source provenance")
        if timestamp and not _contains_synthetic_marker(timestamp):
            raise SyntheticGuardError("Phase 2 discovery requires synthetic source provenance")
        if classification and not _is_explicit_synthetic_classification(_clean_text(classification)):
            raise SyntheticGuardError("Phase 2 discovery requires synthetic source provenance")

    return SourceRef(
        system=_clean_text(system),
        owner=_clean_text(owner),
        timestamp=_clean_text(timestamp) if timestamp else None,
        classification=_clean_text(classification) if classification else None,
    )


def _extract_key_value(body: str, key: str) -> list[str]:
    pattern = re.compile(rf"{re.escape(key)}\s*=\s*([^;]+)", re.IGNORECASE)
    return [_clean_text(match.group(1)) for match in pattern.finditer(body) if _clean_text(match.group(1))]


def _extract_refs_from_text(text: str) -> list[str]:
    refs = re.findall(r"(?:eval|evidence|trace):[A-Za-z0-9._/-]+", text, flags=re.IGNORECASE)
    return [_clean_text(ref) for ref in refs if _clean_text(ref)]


def _nested_value(payload: Mapping[str, Any], parent_key: str, child_key: str) -> Any | None:
    parent = payload.get(parent_key)
    if isinstance(parent, Mapping):
        return parent.get(child_key)
    return None


def _as_mapping(value: Any) -> dict[str, Any]:
    if isinstance(value, Mapping):
        return dict(value)
    return {}


def _nested_string_list(payload: Mapping[str, Any], parent_key: str, child_key: str) -> list[str]:
    parent = payload.get(parent_key)
    if not isinstance(parent, Mapping):
        return []
    child = parent.get(child_key)
    if isinstance(child, list):
        return _coerce_string_list(child)
    return []


def _extract_source_metadata_refs(payload: Mapping[str, Any]) -> list[str]:
    refs: list[str] = []
    fixture_data = payload.get("fixture_data")
    if not isinstance(fixture_data, Mapping):
        return refs
    source_metadata = fixture_data.get("source_metadata")
    if not isinstance(source_metadata, list):
        return refs
    for record in source_metadata:
        if not isinstance(record, Mapping):
            continue
        for key in ("evidence_ref", "ref", "trace_ref"):
            raw = record.get(key)
            if raw is None:
                continue
            cleaned = _clean_text(str(raw))
            if cleaned:
                refs.append(cleaned)
    return refs


def _coerce_string_list(value: Any) -> list[str]:
    if value is None:
        return []
    if isinstance(value, str):
        cleaned = _clean_text(value)
        return [cleaned] if cleaned else []
    if isinstance(value, (list, tuple, set)):
        return [_clean_text(str(item)) for item in value if _clean_text(str(item))]
    return []


def _merge_unique(*values: Iterable[str]) -> list[str]:
    merged: list[str] = []
    seen: set[str] = set()
    for value_set in values:
        for value in value_set:
            cleaned = _clean_text(value)
            if not cleaned or cleaned in seen:
                continue
            seen.add(cleaned)
            merged.append(cleaned)
    return merged


def _append_unique(container: list[str], value: str) -> None:
    cleaned = _clean_text(value)
    if cleaned and cleaned not in container:
        container.append(cleaned)


def _first_text(*values: str | None) -> str:
    for value in values:
        if value is None:
            continue
        cleaned = _clean_text(value)
        if cleaned:
            return cleaned
    return ""


def _string_or_none(value: Any) -> str | None:
    if value is None:
        return None
    cleaned = _clean_text(str(value))
    return cleaned or None


def _contains_synthetic_marker(value: str | None) -> bool:
    if value is None:
        return False
    for line in value.splitlines():
        lowered = line.strip().lower()
        if lowered.startswith("[synthetic]"):
            return True
        if lowered.startswith("[synthetic-fictional]"):
            return True
        if lowered.startswith("synthetic:"):
            return True
        if lowered.startswith("synthetic-"):
            return True
    return False


def _is_explicit_synthetic_classification(value: str | None) -> bool:
    if value is None:
        return False
    normalized = value.strip().lower()
    return normalized in {"synthetic", "synthetic-fictional"}


def _validate_nested_provenance(payload: Mapping[str, Any]) -> None:
    for source_name, records in (
        ("source_register", payload.get("source_register")),
        ("fixture_data.source_metadata", _nested_value(payload, "fixture_data", "source_metadata")),
    ):
        if not isinstance(records, list):
            continue
        for index, record in enumerate(records, start=1):
            if not isinstance(record, Mapping):
                continue

            system = _raw_string_or_none(record.get("system")) or _raw_string_or_none(record.get("source_system"))
            if system and not _contains_synthetic_marker(system):
                raise SyntheticGuardError(
                    f"Phase 2 discovery requires synthetic nested provenance in {source_name}[{index}].system"
                )

            owner = _raw_string_or_none(record.get("owner")) or _raw_string_or_none(record.get("source_owner"))
            if owner and not _contains_synthetic_marker(owner):
                raise SyntheticGuardError(
                    f"Phase 2 discovery requires synthetic nested provenance in {source_name}[{index}].owner"
                )

            timestamp = _raw_string_or_none(record.get("timestamp")) or _raw_string_or_none(record.get("retrieved_at"))
            if timestamp and not _contains_synthetic_marker(timestamp):
                raise SyntheticGuardError(
                    f"Phase 2 discovery requires synthetic nested provenance in {source_name}[{index}].timestamp"
                )

            classification = _string_or_none(record.get("classification")) or _string_or_none(record.get("confidentiality"))
            if classification and not _is_explicit_synthetic_classification(classification):
                raise SyntheticGuardError(
                    f"Phase 2 discovery requires synthetic nested provenance in {source_name}[{index}].classification"
                )


def _validate_project_context_provenance(project_context: Mapping[str, Any]) -> None:
    for key, value in project_context.items():
        if value is None:
            continue

        if isinstance(value, str):
            if _contains_synthetic_marker(value) or "[RAJA]" in value:
                continue
            raise SyntheticGuardError(f"Phase 2 discovery requires synthetic project_context.{key}")

        if isinstance(value, list):
            for index, item in enumerate(value, start=1):
                if not isinstance(item, str):
                    continue
                if _contains_synthetic_marker(item) or "[RAJA]" in item:
                    continue
                raise SyntheticGuardError(
                    f"Phase 2 discovery requires synthetic project_context.{key}[{index}]"
                )


def _validate_note_body_provenance(
    body: str,
    allowed_note_entities: list[str],
    note_kind: str,
    *,
    enforce_note_provenance: bool,
) -> None:
    if not enforce_note_provenance:
        return

    allowed = [entity.strip() for entity in allowed_note_entities if entity.strip()]
    if not allowed:
        return

    for phrase in re.findall(r"\b[A-Z][A-Za-z0-9]*(?:\s+[A-Z][A-Za-z0-9]*)+\b", body):
        if any(entity in phrase for entity in allowed):
            continue
        raise SyntheticGuardError(f"Phase 2 discovery requires synthetic {note_kind} provenance")


def _raw_string_or_none(value: Any) -> str | None:
    if value is None:
        return None
    cleaned = str(value).strip()
    return cleaned or None


def _strip_synthetic_prefix(text: str) -> str:
    stripped = text.strip()
    stripped = re.sub(r"^\[SYNTHETIC\]\s*", "", stripped, flags=re.IGNORECASE)
    stripped = re.sub(r"^\s*SYNTHETIC[:\s]+", "", stripped, flags=re.IGNORECASE)
    return stripped.strip()


def _clean_text(value: str) -> str:
    cleaned = _strip_synthetic_prefix(value)
    cleaned = re.sub(r"\s*\[SYNTHETIC-FICTIONAL\]\s*", " ", cleaned, flags=re.IGNORECASE)
    cleaned = re.sub(r"\s+", " ", cleaned)
    return cleaned.strip()

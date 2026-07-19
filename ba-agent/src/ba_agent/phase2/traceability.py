"""Phase 2 traceability skeleton builder.

P2-G2 thin slice: builds draft trace skeletons that connect evidence to
objectives, requirement candidates, story candidates, questions, and risks.
No write-like actions, live integrations, or system-of-record updates occur.
"""
from __future__ import annotations

import re
from collections.abc import Iterable, Mapping
from typing import Any, Literal, cast

from ba_agent.phase2.models import TraceNode

TraceEntry = TraceNode

__all__ = ["TraceEntry", "TraceNode", "build_skeleton", "build_trace_skeleton", "make_trace_id"]

_NODE_TYPE_ALIASES: dict[str, str] = {
    "objective": "p2-obj",
    "business_objective": "p2-obj",
    "obj": "p2-obj",
    "requirement": "p2-req-draft",
    "req": "p2-req-draft",
    "draft_requirement": "p2-req-draft",
    "story": "p2-story-draft",
    "user_story": "p2-story-draft",
    "question": "p2-question",
    "open_question": "p2-question",
    "risk": "p2-risk",
    "dependency": "p2-risk",
    "input": "p2-input",
    "evidence": "p2-input",
}

_NODE_ORDER = ("p2-obj", "p2-req-draft", "p2-story-draft", "p2-question", "p2-risk")
_NODE_TYPES = ("p2-input", *_NODE_ORDER)
NodeType = Literal["p2-input", "p2-obj", "p2-req-draft", "p2-story-draft", "p2-question", "p2-risk"]


def make_trace_id(node_type: str, case_id: str, n: int) -> str:
    """Return a canonical ``p2-*`` trace identifier."""
    normalized_type = node_type[3:] if node_type.startswith("p2-") else node_type
    normalized_case_id = _normalize_case_id(case_id)
    return f"p2-{normalized_type}:{normalized_case_id}:{n:03d}"


def build_skeleton(case_id: str, evidence_labels: list[str]) -> list[TraceEntry]:
    """Return evidence-only trace nodes for the given case."""
    nodes: list[TraceEntry] = []
    for index, label in enumerate(_unique_nonempty(evidence_labels), start=1):
        nodes.append(
            TraceEntry(
                trace_id=make_trace_id("input", case_id, index),
                node_type="p2-input",
                label=label,
                parent_ids=[],
            )
        )
    return nodes


def build_trace_skeleton(candidates: Iterable[object], evidence_refs: Iterable[object]) -> list[TraceEntry]:
    """Build a draft trace skeleton from candidate artifacts and evidence refs."""
    candidate_list = list(candidates)
    evidence_ref_list = list(evidence_refs)
    case_id = _find_case_id(candidate_list, evidence_ref_list)
    refs = _unique_nonempty(_coerce_ref(ref) for ref in evidence_ref_list)

    nodes: list[TraceEntry] = build_skeleton(case_id, refs)
    evidence_ids = [node.trace_id for node in nodes]
    used_trace_ids = set(evidence_ids)
    objective_ids: list[str] = []
    requirement_ids: list[str] = []
    story_ids: list[str] = []
    counts: dict[str, int] = {node_type: 0 for node_type in _NODE_ORDER}
    counts["p2-input"] = len(nodes)

    for candidate in candidate_list:
        node_type = _candidate_node_type(candidate)
        explicit_trace_id = _candidate_field(candidate, "trace_id")
        if node_type == "p2-input":
            label = _candidate_label(candidate)
            counts["p2-input"] = counts.get("p2-input", 0) + 1
            trace_id = _allocate_trace_id(
                "input",
                case_id,
                counts["p2-input"],
                explicit_trace_id,
                used_trace_ids,
            )
            nodes.append(
                TraceEntry(
                    trace_id=trace_id,
                    node_type="p2-input",
                    label=label,
                    parent_ids=[],
                )
            )
            used_trace_ids.add(trace_id)
            continue

        label = _candidate_label(candidate)
        parent_ids = _candidate_parent_ids(candidate)
        if not parent_ids:
            parent_ids = _default_parent_ids(node_type, evidence_ids, objective_ids, requirement_ids, story_ids)

        counts[node_type] = counts.get(node_type, 0) + 1
        trace_id = _allocate_trace_id(
            node_type,
            case_id,
            counts[node_type],
            explicit_trace_id,
            used_trace_ids,
        )
        node = TraceEntry(
            trace_id=trace_id,
            node_type=node_type,
            label=label,
            parent_ids=parent_ids,
        )
        nodes.append(node)
        used_trace_ids.add(trace_id)

        if node_type == "p2-obj":
            objective_ids.append(node.trace_id)
        elif node_type == "p2-req-draft":
            requirement_ids.append(node.trace_id)
        elif node_type == "p2-story-draft":
            story_ids.append(node.trace_id)

    return nodes


def _allocate_trace_id(
    node_type: str,
    case_id: str,
    ordinal: int,
    explicit_trace_id: object | None,
    used_trace_ids: set[str],
) -> str:
    candidate = str(explicit_trace_id).strip() if explicit_trace_id is not None else ""
    if candidate and candidate not in used_trace_ids:
        return candidate

    candidate_ordinal = ordinal
    trace_id = make_trace_id(node_type, case_id, candidate_ordinal)
    while trace_id in used_trace_ids:
        candidate_ordinal += 1
        trace_id = make_trace_id(node_type, case_id, candidate_ordinal)
    return trace_id


def _default_parent_ids(
    node_type: str,
    evidence_ids: list[str],
    objective_ids: list[str],
    requirement_ids: list[str],
    story_ids: list[str],
) -> list[str]:
    if node_type == "p2-obj":
        return evidence_ids
    if node_type == "p2-req-draft":
        return objective_ids or evidence_ids
    if node_type == "p2-story-draft":
        return requirement_ids or objective_ids or evidence_ids
    if node_type in {"p2-question", "p2-risk"}:
        return evidence_ids
    if node_type == "p2-input":
        return []
    return story_ids or requirement_ids or objective_ids or evidence_ids


def _candidate_node_type(candidate: object) -> NodeType:
    raw = _candidate_field(candidate, "node_type", "kind", "type")
    if raw is None:
        label = _candidate_label(candidate).lower()
        if label.endswith("?") or "question" in label:
            return "p2-question"
        if "risk" in label or "dependency" in label:
            return "p2-risk"
        if "story" in label or label.startswith("as a "):
            return "p2-story-draft"
        if "objective" in label or "goal" in label:
            return "p2-obj"
        return "p2-req-draft"

    normalized = str(raw).strip().lower()
    node_type = _NODE_TYPE_ALIASES.get(normalized, normalized if normalized.startswith("p2-") else f"p2-{normalized}")
    if node_type not in _NODE_TYPES:
        return "p2-req-draft"
    return cast(NodeType, node_type)


def _candidate_label(candidate: object) -> str:
    for field in ("label", "text", "question", "description", "item", "claim", "value"):
        raw = _candidate_field(candidate, field)
        if raw is not None:
            cleaned = _clean_text(str(raw))
            if cleaned:
                return cleaned
    return "synthetic trace candidate"


def _candidate_parent_ids(candidate: object) -> list[str]:
    raw = _candidate_field(candidate, "parent_ids")
    if raw is None:
        return []
    if isinstance(raw, list):
        return [str(item).strip() for item in raw if str(item).strip()]
    if isinstance(raw, tuple):
        return [str(item).strip() for item in raw if str(item).strip()]
    parent = str(raw).strip()
    return [parent] if parent else []


def _candidate_field(candidate: object, *fields: str) -> Any | None:
    if isinstance(candidate, Mapping):
        for field in fields:
            value = candidate.get(field)
            if value is not None:
                return value
        return None

    for field in fields:
        if hasattr(candidate, field):
            value = getattr(candidate, field)
            if value is not None:
                return value
    return None


def _coerce_ref(value: object) -> str:
    if isinstance(value, Mapping):
        for field in ("evidence_ref", "ref", "label", "value"):
            raw = value.get(field)
            if raw is not None:
                return _clean_text(str(raw))
        return _clean_text(str(dict(value)))
    raw = _candidate_field(value, "evidence_ref", "ref", "label", "value")
    if raw is not None:
        return _clean_text(str(raw))
    return _clean_text(str(value))


def _find_case_id(candidates: list[object], evidence_refs: list[object]) -> str:
    for candidate in candidates:
        raw = _candidate_field(candidate, "case_id", "trace_id", "candidate_id")
        case_id = _normalize_case_id(str(raw)) if raw is not None else ""
        if case_id and _looks_like_case_id(case_id):
            return case_id

    for ref in evidence_refs:
        case_id = _normalize_case_id(_coerce_ref(ref))
        if case_id and _looks_like_case_id(case_id):
            return case_id

    return "synthetic-case"


def _normalize_case_id(value: str) -> str:
    match = re.search(r"(P2REQ-\d{3})", value)
    if match:
        return match.group(1)
    match = re.search(r"([A-Z]+-\d{3,})", value)
    if match:
        return match.group(1)
    return _clean_text(value)


def _looks_like_case_id(value: str) -> bool:
    return bool(re.fullmatch(r"(?:P2REQ-\d{3}|[A-Z]+-\d{3,})", value))


def _unique_nonempty(values: Iterable[Any]) -> list[str]:
    seen: set[str] = set()
    ordered: list[str] = []
    for raw_value in values:
        value = _clean_text(str(raw_value))
        if not value or value in seen:
            continue
        seen.add(value)
        ordered.append(value)
    return ordered


def _clean_text(value: str) -> str:
    cleaned = value.strip()
    cleaned = re.sub(r"^\[SYNTHETIC\]\s*", "", cleaned, flags=re.IGNORECASE)
    cleaned = re.sub(r"^\[SYNTHETIC-FICTIONAL\]\s*", "", cleaned, flags=re.IGNORECASE)
    cleaned = re.sub(r"^\s*SYNTHETIC[:\s]+", "", cleaned, flags=re.IGNORECASE)
    return cleaned.strip()

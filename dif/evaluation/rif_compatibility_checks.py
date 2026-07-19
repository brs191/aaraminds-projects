#!/usr/bin/env python3
"""Validate DIF ADR-016 RIF compatibility fixtures.

The harness is stdlib-only and uses synthetic JSON fixtures to prove the RIF
compatibility contract before DIF resolver/database integration exists.
"""

from __future__ import annotations

import hashlib
import json
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
FIXTURE = ROOT / "evaluation" / "fixtures" / "rif"
NUL = "\0"


def load_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def sha256_text(value: str) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def node_id(repo_id: str, qualified_name: str, kind: str) -> str:
    return sha256_text(NUL.join([repo_id, qualified_name, kind]))


def edge_id(from_node_id: str, label: str, to_node_id: str) -> str:
    return sha256_text(NUL.join([from_node_id, label, to_node_id]))


def legacy_space_node_id(repo_id: str, qualified_name: str, kind: str) -> str:
    return sha256_text(" ".join([repo_id, qualified_name, kind]))


def materialize_entities(payload: dict[str, Any]) -> dict[str, dict[str, Any]]:
    repo_id = payload["repo_id"]
    entities: dict[str, dict[str, Any]] = {}
    for entity in payload["entities"]:
        enriched = dict(entity)
        enriched["repo_id"] = repo_id
        enriched["node_id"] = node_id(repo_id, entity["qualified_name"], entity["kind"])
        entities[entity["entity_alias"]] = enriched
    return entities


def variant_by_name(payload: dict[str, Any]) -> dict[str, dict[str, Any]]:
    return {variant["variant"]: variant for variant in payload["variants"]}


def entities_for_refs(
    refs: list[dict[str, Any]],
    entities: dict[str, dict[str, Any]],
) -> list[dict[str, Any]]:
    result: list[dict[str, Any]] = []
    for ref in refs:
        entity = dict(entities[ref["entity_alias"]])
        for field in ref.get("omit_fields", []):
            entity.pop(field, None)
        result.append(entity)
    return result


def status_for_variant(
    variant: dict[str, Any],
    entities: dict[str, dict[str, Any]],
    required_fields: list[str],
) -> dict[str, Any]:
    if "rif" not in variant["schemas"] and "rif_meta" not in variant["schemas"]:
        return {
            "rif_status": "rif_not_deployed",
            "shadow_status": None,
            "matches": [],
            "missing_capabilities": [],
            "caveats": ["No RIF compatibility surface is available."],
        }

    age_entities = entities_for_refs(variant["age_entities"], entities)
    shadow_entities = entities_for_refs(variant["shadow_entities"], entities)
    candidate_entities = shadow_entities or age_entities
    missing = sorted(
        {
            field
            for entity in candidate_entities
            for field in required_fields
            if field not in entity or entity[field] in (None, "")
        }
    )
    missing = sorted(set(missing + variant.get("missing_fields", [])))

    if missing:
        return {
            "rif_status": "rif_incompatible",
            "shadow_status": None,
            "matches": [],
            "missing_capabilities": missing,
            "caveats": ["Required RIF compatibility fields are unavailable."],
        }

    if shadow_entities:
        return {
            "rif_status": "rif_compatible",
            "shadow_status": "rif_shadow_populated",
            "matches": sorted(candidate_entities, key=lambda item: item["node_id"]),
            "missing_capabilities": [],
            "caveats": [],
        }

    if age_entities:
        return {
            "rif_status": "rif_compatible",
            "shadow_status": "rif_shadow_empty",
            "matches": sorted(candidate_entities, key=lambda item: item["node_id"]),
            "missing_capabilities": [],
            "caveats": ["RIF relational shadows are empty; AGE fallback is active."],
        }

    return {
        "rif_status": "rif_incompatible",
        "shadow_status": "rif_shadow_empty" if variant["shadow_available"] else None,
        "matches": [],
        "missing_capabilities": [],
        "caveats": ["RIF shadows are empty and no AGE/API fallback is available."],
    }


def resolve_lookup(status: dict[str, Any], mode: str, query: str) -> dict[str, Any]:
    if status["rif_status"] != "rif_compatible":
        return {"status": status["rif_status"], "matches": [], "caveats": status["caveats"]}

    matches = status["matches"]
    caveats: list[str] = []
    confidence = "exact"

    if mode == "qualified_name":
        resolved = [item for item in matches if item["qualified_name"] == query]
    elif mode == "source_path":
        resolved = [item for item in matches if item["qualified_name"] == query and item["kind"] == "FILE"]
    elif mode == "simple_name":
        resolved = [item for item in matches if item.get("simple_name") == query]
        confidence = "inferred"
        if len(resolved) > 1:
            caveats.append("ambiguous_simple_name")
    else:
        resolved = []

    resolved = sorted(
        resolved,
        key=lambda item: (
            0 if item["confidence"] == "exact" else 1,
            {"METHOD": 0, "CLASS": 1, "INTERFACE": 2, "RECORD": 3, "ENUM": 4, "FILE": 5}.get(item["kind"], 9),
            len(item["qualified_name"]),
            item["qualified_name"],
            item["node_id"],
        ),
    )

    return {
        "status": "resolved" if resolved else "unresolved",
        "confidence": confidence,
        "matches": resolved,
        "caveats": caveats,
    }


def aliases_for_matches(matches: list[dict[str, Any]]) -> list[str]:
    return [item["entity_alias"] for item in matches]


def main() -> int:
    payload = load_json(FIXTURE / "compat_entities.json")
    expected = load_json(FIXTURE / "expected_resolutions.json")
    entities = materialize_entities(payload)
    variants = variant_by_name(payload)
    failures: list[str] = []

    for entity in entities.values():
        legacy_id = legacy_space_node_id(entity["repo_id"], entity["qualified_name"], entity["kind"])
        if entity["node_id"] == legacy_id:
            failures.append(f"{entity['entity_alias']}: NUL-separated node ID matches legacy space hash")

    relationship = payload["relationships"][0]
    from_entity = entities[relationship["from_entity_alias"]]
    to_entity = entities[relationship["to_entity_alias"]]
    computed_edge_id = edge_id(from_entity["node_id"], relationship["label"], to_entity["node_id"])
    legacy_edge_id = sha256_text(" ".join([from_entity["node_id"], relationship["label"], to_entity["node_id"]]))
    if computed_edge_id == legacy_edge_id:
        failures.append("edge ID algorithm unexpectedly matches legacy space-separated hash")

    statuses = {
        name: status_for_variant(variant, entities, payload["required_fields"])
        for name, variant in variants.items()
    }

    for expectation in expected["variant_expectations"]:
        actual = statuses[expectation["variant"]]
        for field in ("rif_status", "shadow_status"):
            if actual[field] != expectation[field]:
                failures.append(
                    f"{expectation['variant']}: {field} expected {expectation[field]!r}, got {actual[field]!r}"
                )
        if len(actual["matches"]) != expectation["expected_match_count"]:
            failures.append(
                f"{expectation['variant']}: expected {expectation['expected_match_count']} matches, got {len(actual['matches'])}"
            )
        for caveat in expectation.get("required_caveats", []):
            if caveat not in actual["caveats"]:
                failures.append(f"{expectation['variant']}: missing caveat {caveat!r}")
        for missing in expectation.get("required_missing_capabilities", []):
            if missing not in actual["missing_capabilities"]:
                failures.append(f"{expectation['variant']}: missing capability {missing!r} was not reported")

    for lookup in expected["lookup_expectations"]:
        actual = resolve_lookup(statuses[lookup["variant"]], lookup["mode"], lookup["input"])
        actual_aliases = aliases_for_matches(actual["matches"])
        if actual_aliases != lookup["expected_entity_aliases"]:
            failures.append(
                f"{lookup['case_id']}: expected aliases {lookup['expected_entity_aliases']}, got {actual_aliases}"
            )
        if "expected_confidence" in lookup and actual.get("confidence") != lookup["expected_confidence"]:
            failures.append(
                f"{lookup['case_id']}: expected confidence {lookup['expected_confidence']}, got {actual.get('confidence')}"
            )
        if "expected_status" in lookup and actual["status"] != lookup["expected_status"]:
            failures.append(
                f"{lookup['case_id']}: expected status {lookup['expected_status']}, got {actual['status']}"
            )
        for caveat in lookup.get("required_caveats", []):
            if caveat not in actual["caveats"]:
                failures.append(f"{lookup['case_id']}: missing caveat {caveat!r}")

    if failures:
        for failure in failures:
            print(f"FAIL: {failure}", file=sys.stderr)
        return 1

    print(
        "RIF compatibility harness passed "
        f"({len(statuses)} variants, "
        f"{len(expected['lookup_expectations'])} lookup cases)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())


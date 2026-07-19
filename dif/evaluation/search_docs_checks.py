#!/usr/bin/env python3
"""Validate the DIF P0 search_docs anchored retrieval contract.

This harness is stdlib-only and intentionally runs against golden expectation
files instead of a live retriever. It makes the search_docs contract executable
before the service and MCP implementation exist.
"""

from __future__ import annotations

import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
GOLDEN = ROOT / "evaluation" / "golden"

REQUIRED_RESULT_FIELDS = {
    "corpus_id",
    "document_id",
    "document_version_id",
    "passage_id",
    "snippet",
    "anchor_id",
    "source_ref",
    "score",
    "caveats",
}


@dataclass(frozen=True)
class SourceRef:
    corpus_id: str
    document_version_id: str
    anchor_type: str
    payload: str


def load_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def parse_source_ref(source_ref: str) -> SourceRef:
    match = re.fullmatch(r"([^@]+)@([^:]+):([^:]+):(.+)", source_ref)
    if not match:
        raise ValueError(f"invalid source_ref: {source_ref}")
    return SourceRef(
        corpus_id=match.group(1),
        document_version_id=match.group(2),
        anchor_type=match.group(3),
        payload=match.group(4),
    )


def manifest_corpora(manifest: dict[str, Any]) -> dict[str, dict[str, Any]]:
    return {corpus["corpus_id"]: corpus for corpus in manifest["corpora"]}


def anchors_by_source_ref(payload: dict[str, Any]) -> dict[str, dict[str, Any]]:
    return {anchor["source_ref"]: anchor for anchor in payload["anchors"]}


def build_raw_candidates(
    query: dict[str, Any],
    anchors: dict[str, dict[str, Any]],
) -> list[dict[str, Any]]:
    candidates: list[dict[str, Any]] = []
    for index, source_ref in enumerate(query["required_top_source_refs"]):
        anchor = anchors[source_ref]
        parsed = parse_source_ref(source_ref)
        result = {
            "corpus_id": query["corpus_id"],
            "document_id": anchor["document_alias"],
            "document_version_id": parsed.document_version_id,
            "passage_id": f"passage-{anchor['anchor_alias']}",
            "snippet": anchor["expected_excerpt"],
            "anchor_id": anchor["anchor_alias"],
            "source_ref": source_ref,
            "score": 1.0 - (index * 0.01),
            "caveats": [],
        }
        if "expected_code_entity_candidate" in anchor:
            result["code_entity_candidate"] = anchor["expected_code_entity_candidate"]
        candidates.append(result)

    if candidates:
        unanchored = dict(candidates[0])
        unanchored["passage_id"] = f"unanchored-{query['query_id']}"
        unanchored["score"] = 99.0
        unanchored.pop("anchor_id")
        unanchored.pop("source_ref")
        candidates.insert(0, unanchored)

    return candidates


def search_docs_contract(
    query: dict[str, Any],
    manifest: dict[str, Any],
    anchors: dict[str, dict[str, Any]],
) -> dict[str, Any]:
    corpora = manifest_corpora(manifest)
    corpus = corpora.get(query["corpus_id"])
    if corpus is None or corpus["admission_status"] != "admitted":
        return {"status": "corpus_not_admitted", "results": []}

    anchored_results = [
        candidate
        for candidate in build_raw_candidates(query, anchors)
        if candidate.get("anchor_id") and candidate.get("source_ref")
    ]
    anchored_results.sort(key=lambda item: (-item["score"], item["source_ref"]))

    if not anchored_results:
        return {"status": "no_evidence", "results": []}

    return {"status": "ok", "results": anchored_results}


def validate_response(
    query: dict[str, Any],
    response: dict[str, Any],
    anchors: dict[str, dict[str, Any]],
) -> list[str]:
    failures: list[str] = []
    result_count = len(response["results"])
    accepted_count = query["accepted_result_count"]

    expected_status = query.get("required_status")
    if expected_status and response["status"] != expected_status:
        failures.append(
            f"{query['query_id']}: expected status {expected_status!r}, got {response['status']!r}"
        )

    if not expected_status and result_count == 0 and accepted_count["min"] > 0:
        failures.append(f"{query['query_id']}: expected evidence results, got none")

    if result_count < accepted_count["min"] or result_count > accepted_count["max"]:
        failures.append(
            f"{query['query_id']}: expected result count between "
            f"{accepted_count['min']} and {accepted_count['max']}, got {result_count}"
        )

    required_refs = query["required_top_source_refs"]
    actual_top_refs = [result.get("source_ref") for result in response["results"][: len(required_refs)]]
    if actual_top_refs != required_refs:
        failures.append(
            f"{query['query_id']}: expected top source refs {required_refs}, got {actual_top_refs}"
        )

    for result in response["results"]:
        missing_fields = REQUIRED_RESULT_FIELDS - set(result)
        if missing_fields:
            failures.append(
                f"{query['query_id']}:{result.get('passage_id', '<missing-passage>')}: "
                f"missing required fields {sorted(missing_fields)}"
            )
            continue

        source_ref = result["source_ref"]
        if source_ref not in anchors:
            failures.append(f"{query['query_id']}:{result['passage_id']}: unknown source_ref {source_ref!r}")
            continue

        anchor = anchors[source_ref]
        if result["anchor_id"] != anchor["anchor_alias"]:
            failures.append(
                f"{query['query_id']}:{result['passage_id']}: expected anchor_id "
                f"{anchor['anchor_alias']!r}, got {result['anchor_id']!r}"
            )
        if anchor["expected_excerpt"] not in result["snippet"]:
            failures.append(
                f"{query['query_id']}:{result['passage_id']}: snippet does not include expected excerpt"
            )
        if not isinstance(result["caveats"], list):
            failures.append(f"{query['query_id']}:{result['passage_id']}: caveats must be a list")

    return failures


def main() -> int:
    manifest = load_json(GOLDEN / "manifest.json")
    queries = load_json(GOLDEN / "golden-queries.json")
    anchors = anchors_by_source_ref(load_json(GOLDEN / "expected-anchors.json"))
    failures: list[str] = []

    positive_queries = 0
    for query in queries["queries"]:
        response = search_docs_contract(query, manifest, anchors)
        failures.extend(validate_response(query, response, anchors))
        if response["status"] == "ok":
            positive_queries += 1

    if failures:
        for failure in failures:
            print(f"FAIL: {failure}", file=sys.stderr)
        return 1

    print(
        "search_docs contract harness passed "
        f"({len(queries['queries'])} queries, "
        f"{positive_queries} anchored retrieval cases, "
        "no-evidence and corpus_not_admitted cases)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

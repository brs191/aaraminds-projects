#!/usr/bin/env python3
"""Validate DIF P0 golden source-anchor round trips.

This harness is intentionally stdlib-only so P0 source-anchor expectations can
run before the service implementation and package toolchain exist.
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
ADMITTED = GOLDEN / "sources" / "admitted"


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


def load_expected_anchors() -> dict[str, dict[str, Any]]:
    payload = load_json(GOLDEN / "expected-anchors.json")
    return {anchor["source_ref"]: anchor for anchor in payload["anchors"]}


def fixture_path(anchor: dict[str, Any]) -> Path:
    path = anchor["path"]
    if anchor["anchor_type"] == "docx":
        return ADMITTED / "requirements.docx.fixture.json"
    return ADMITTED / path


def resolve_line_anchor(path: Path, line_start: int, line_end: int) -> tuple[str | None, str]:
    lines = path.read_text(encoding="utf-8").splitlines()
    if line_start < 1 or line_end > len(lines) or line_end < line_start:
        return "anchor_out_of_range", ""
    return None, "\n".join(lines[line_start - 1 : line_end])


def resolve_docx_anchor(path: Path, paragraph_index: int) -> tuple[str | None, str]:
    payload = load_json(path)
    for paragraph in payload["paragraphs"]:
        if paragraph["paragraph_index"] == paragraph_index:
            return None, paragraph["text"]
    return "anchor_out_of_range", ""


def jsonpath_get(value: Any, json_path: str) -> tuple[str | None, Any]:
    if not json_path.startswith("$"):
        return "anchor_not_found", None
    current = value
    token_re = re.compile(r"\.([A-Za-z_][A-Za-z0-9_]*)|\[(\d+)\]")
    position = 1
    while position < len(json_path):
        match = token_re.match(json_path, position)
        if not match:
            return "anchor_not_found", None
        key, index = match.groups()
        if key is not None:
            if not isinstance(current, dict) or key not in current:
                return "anchor_not_found", None
            current = current[key]
        else:
            array_index = int(index)
            if not isinstance(current, list) or array_index >= len(current):
                return "anchor_not_found", None
            current = current[array_index]
        position = match.end()
    return None, current


def resolve_json_anchor(path: Path, json_path: str) -> tuple[str | None, str]:
    status, value = jsonpath_get(load_json(path), json_path)
    if status:
        return status, ""
    if isinstance(value, (dict, list)):
        return None, json.dumps(value, sort_keys=True, separators=(",", ":"))
    return None, str(value)


def resolve_expected_anchor(anchor: dict[str, Any]) -> tuple[str | None, str]:
    path = fixture_path(anchor)
    anchor_type = anchor["anchor_type"]
    if anchor_type in {"md", "txt"}:
        return resolve_line_anchor(path, anchor["line_start"], anchor["line_end"])
    if anchor_type == "docx":
        return resolve_docx_anchor(path, anchor["paragraph_index"])
    if anchor_type == "json":
        return resolve_json_anchor(path, anchor["json_path"])
    return "anchor_type_unsupported", ""


def resolve_failure_case(source_ref: str, anchors_by_ref: dict[str, dict[str, Any]]) -> str:
    parsed = parse_source_ref(source_ref)
    if parsed.anchor_type not in {"md", "txt", "docx", "json"}:
        return "anchor_type_unsupported"
    if parsed.document_version_id == "docver-missing":
        return "document_version_not_found"
    if parsed.document_version_id == "docver-source-unavailable":
        return "source_content_unavailable"
    if source_ref in anchors_by_ref:
        status, _ = resolve_expected_anchor(anchors_by_ref[source_ref])
        return status or "success"
    if parsed.anchor_type in {"md", "txt"}:
        match = re.search(r"#L(\d+)-L(\d+)$", parsed.payload)
        if match and int(match.group(1)) >= 100:
            return "anchor_out_of_range"
    return "anchor_not_found"


def main() -> int:
    expected = load_json(GOLDEN / "expected-anchors.json")
    anchors_by_ref = load_expected_anchors()
    failures: list[str] = []

    for anchor in expected["anchors"]:
        status, resolved = resolve_expected_anchor(anchor)
        if status is not None:
            failures.append(f"{anchor['anchor_alias']}: expected success, got {status}")
            continue
        expected_excerpt = anchor["expected_excerpt"]
        if expected_excerpt not in resolved:
            failures.append(
                f"{anchor['anchor_alias']}: expected excerpt not found in resolved text"
            )

    for case in expected["resolver_failure_cases"]:
        actual_status = resolve_failure_case(case["input_source_ref"], anchors_by_ref)
        if actual_status != case["expected_status"]:
            failures.append(
                f"{case['case_id']}: expected {case['expected_status']}, got {actual_status}"
            )

    if failures:
        for failure in failures:
            print(f"FAIL: {failure}", file=sys.stderr)
        return 1

    print(
        "Source-anchor round-trip harness passed "
        f"({len(expected['anchors'])} anchors, "
        f"{len(expected['resolver_failure_cases'])} failure cases)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

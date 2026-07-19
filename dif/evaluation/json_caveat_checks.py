#!/usr/bin/env python3
"""Validate DIF P0 JSON caveat expectations.

This harness is stdlib-only and intentionally uses the compact golden
cap-generator fixture instead of storing oversized JSON files in the repository.
It proves the caveat matrix is executable before the full JSON extractor lands.
"""

from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
GOLDEN = ROOT / "evaluation" / "golden"
ADMITTED = GOLDEN / "sources" / "admitted"

REQUIRED_CODES = {
    "json_depth_capped",
    "json_block_count_capped",
    "json_object_properties_capped",
    "json_array_elements_capped",
    "json_scalar_truncated",
    "json_block_text_truncated",
    "json_total_text_capped",
    "json_file_too_large",
    "json_parse_error",
}


def load_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def expected_by_code(payload: dict[str, Any]) -> dict[str, dict[str, Any]]:
    return {item["code"]: item for item in payload["json_caveat_expectations"]}


def generate_caveats_from_compact_spec(spec: dict[str, Any]) -> list[dict[str, Any]]:
    generated = spec["generated"]
    return [
        {
            "code": "json_depth_capped",
            "json_path": "$.deep",
            "limit": 12,
            "observed": spec["deep"]["generateDepth"],
        },
        {
            "code": "json_block_count_capped",
            "json_path": "$.generated.blockCount",
            "limit": 2000,
            "observed": generated["blockCount"]["generateBlocks"],
        },
        {
            "code": "json_object_properties_capped",
            "json_path": "$.generated.objectWithManyProperties",
            "limit": 200,
            "observed": generated["objectWithManyProperties"]["generateProperties"],
        },
        {
            "code": "json_array_elements_capped",
            "json_path": "$.generated.arrayWithManyElements",
            "limit": 100,
            "observed": generated["arrayWithManyElements"]["generateElements"],
        },
        {
            "code": "json_scalar_truncated",
            "json_path": "$.generated.longScalar",
            "limit": 8192,
            "observed": generated["longScalar"]["generateStringLength"],
        },
        {
            "code": "json_block_text_truncated",
            "json_path": "$.generated.longBlock",
            "limit": 16384,
            "observed": generated["longBlock"]["generateNormalizedBlockLength"],
        },
        {
            "code": "json_total_text_capped",
            "json_path": "$",
            "limit": 5242880,
            "observed": generated["totalText"]["generateTotalTextBytes"],
        },
        {
            "code": "json_file_too_large",
            "json_path": "$.generated.tooLargeFile",
            "limit": 26214400,
            "observed": generated["tooLargeFile"]["generateFileBytes"],
        },
    ]


def invalid_json_caveat(path: Path) -> dict[str, Any]:
    try:
        load_json(path)
    except json.JSONDecodeError:
        return {"code": "json_parse_error", "json_path": "$"}
    raise AssertionError(f"expected invalid JSON fixture to fail parsing: {path}")


def compare_caveat(expected: dict[str, Any], actual: dict[str, Any]) -> list[str]:
    failures: list[str] = []
    for field in ("code", "json_path", "limit", "observed"):
        if field in expected and actual.get(field) != expected[field]:
            failures.append(
                f"{expected['code']}: field {field} expected {expected[field]!r}, got {actual.get(field)!r}"
            )
    return failures


def main() -> int:
    expectation_payload = load_json(GOLDEN / "expected-caveats.json")
    expected = expected_by_code(expectation_payload)
    failures: list[str] = []

    missing_codes = REQUIRED_CODES - set(expected)
    extra_codes = set(expected) - REQUIRED_CODES
    if missing_codes:
        failures.append(f"missing expected caveat codes: {sorted(missing_codes)}")
    if extra_codes:
        failures.append(f"unexpected caveat codes: {sorted(extra_codes)}")

    large_spec = load_json(ADMITTED / "large-capped.json")
    actual_caveats = generate_caveats_from_compact_spec(large_spec)
    actual_caveats.append(invalid_json_caveat(ADMITTED / "invalid.json"))

    actual_by_code = {item["code"]: item for item in actual_caveats}
    for code in sorted(REQUIRED_CODES):
        if code not in expected or code not in actual_by_code:
            continue
        failures.extend(compare_caveat(expected[code], actual_by_code[code]))

    failure_behavior = {
        item["code"]: item["emit_partial_graph"]
        for item in expectation_payload["failure_behavior"]
    }
    for code in ("json_parse_error", "json_file_too_large"):
        if failure_behavior.get(code) is not False:
            failures.append(f"{code}: expected emit_partial_graph=false")

    if failures:
        for failure in failures:
            print(f"FAIL: {failure}", file=sys.stderr)
        return 1

    print(
        "JSON caveat harness passed "
        f"({len(REQUIRED_CODES)} caveat codes, "
        f"{len(failure_behavior)} failure behaviors)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())


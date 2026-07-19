#!/usr/bin/env python3
"""Validate DIF P0 audit and usage write expectations.

This harness is stdlib-only and runs before the service implementation exists.
It verifies that the golden audit/usage contract matches the SQL schema and
that simulated write records keep audit and metering data separate without
storing raw document text or request payloads.
"""

from __future__ import annotations

import hashlib
import json
import re
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
GOLDEN = ROOT / "evaluation" / "golden"
MIGRATION = ROOT / "code" / "migrations" / "001_dif_meta_initial.sql"

AUDIT_OUTCOMES = {"success", "error", "denied"}
USAGE_EVENT_TYPES = {
    "ingestion_run",
    "document_indexed",
    "embedding_batch",
    "mcp_tool_call",
    "agent_request",
    "connector_sync",
}


def load_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def canonical_hash(value: dict[str, Any]) -> str:
    encoded = json.dumps(value, sort_keys=True, separators=(",", ":")).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()


def table_columns(sql: str, table_name: str) -> set[str]:
    pattern = rf"CREATE TABLE IF NOT EXISTS dif_meta\.{table_name} \((.*?)\n\);"
    match = re.search(pattern, sql, flags=re.DOTALL)
    if not match:
        raise ValueError(f"missing dif_meta.{table_name} table in migration")

    columns: set[str] = set()
    for raw_line in match.group(1).splitlines():
        line = raw_line.strip()
        if not line or line.startswith("CONSTRAINT"):
            continue
        column = line.split()[0].rstrip(",")
        columns.add(column)
    return columns


def write_audit_event(call: dict[str, Any]) -> dict[str, Any]:
    event = {
        "principal_id": call["principal_id"],
        "tenant_id": call.get("tenant_id"),
        "project_id": call["project_id"],
        "corpus_id": call["corpus_id"],
        "tool_name": call["tool_name"],
        "tool_version": call.get("tool_version"),
        "parameters_hash": canonical_hash(call["parameters"]),
        "outcome": call["outcome"],
        "latency_ms": call["latency_ms"],
        "source_refs": call["source_refs"],
        "error_class": call.get("error_class"),
    }
    return {key: value for key, value in event.items() if value is not None}


def write_usage_event(call: dict[str, Any]) -> dict[str, Any]:
    event = {
        "event_type": call["usage_event_type"],
        "tenant_id": call.get("tenant_id"),
        "project_id": call["project_id"],
        "corpus_id": call["corpus_id"],
        "counts": call["usage_counts"],
        "latency_ms": call["latency_ms"],
        "token_units": call.get("token_units"),
        "embedding_units": call.get("embedding_units"),
        "error_class": call.get("error_class"),
    }
    return {key: value for key, value in event.items() if value is not None}


def flatten_strings(value: Any) -> list[str]:
    if isinstance(value, str):
        return [value]
    if isinstance(value, dict):
        strings: list[str] = []
        for key, nested in value.items():
            strings.append(str(key))
            strings.extend(flatten_strings(nested))
        return strings
    if isinstance(value, list):
        strings = []
        for nested in value:
            strings.extend(flatten_strings(nested))
        return strings
    return []


def validate_schema_contract(
    manifest: dict[str, Any],
    expectations: dict[str, Any],
    sql: str,
) -> list[str]:
    failures: list[str] = []
    audit_columns = table_columns(sql, "audit_log")
    usage_columns = table_columns(sql, "usage_events")

    for field in manifest["audit_dimensions"]:
        if field not in audit_columns:
            failures.append(f"audit schema missing manifest dimension {field!r}")
    for field in manifest["usage_dimensions"]:
        if field not in usage_columns:
            failures.append(f"usage schema missing manifest dimension {field!r}")

    for field in expectations["required_audit_fields"]:
        if field not in audit_columns:
            failures.append(f"audit schema missing required field {field!r}")
    for field in expectations["required_usage_fields"]:
        if field not in usage_columns:
            failures.append(f"usage schema missing required field {field!r}")

    if "counts" in audit_columns:
        failures.append("audit schema must not include usage counts")
    if "source_refs" in usage_columns:
        failures.append("usage schema must not include raw source_refs")
    if "principal_id" in usage_columns:
        failures.append("usage schema must not include principal_id")

    return failures


def validate_audit_event(
    case_id: str,
    event: dict[str, Any],
    call: dict[str, Any],
    required_fields: set[str],
    known_source_refs: set[str],
) -> list[str]:
    failures: list[str] = []
    missing_fields = required_fields - set(event)
    if missing_fields:
        failures.append(f"{case_id}: audit event missing fields {sorted(missing_fields)}")

    if event.get("parameters_hash") != canonical_hash(call["parameters"]):
        failures.append(f"{case_id}: audit parameters_hash is not the canonical SHA-256")
    if "parameters" in event:
        failures.append(f"{case_id}: audit event must not store raw parameters")
    if event.get("outcome") not in AUDIT_OUTCOMES:
        failures.append(f"{case_id}: invalid audit outcome {event.get('outcome')!r}")
    if not isinstance(event.get("latency_ms"), int) or event["latency_ms"] < 0:
        failures.append(f"{case_id}: audit latency_ms must be a non-negative integer")
    if not isinstance(event.get("source_refs"), list):
        failures.append(f"{case_id}: audit source_refs must be a list")
    else:
        unknown_refs = set(event["source_refs"]) - known_source_refs
        if unknown_refs:
            failures.append(f"{case_id}: audit event has unknown source_refs {sorted(unknown_refs)}")
    if call["outcome"] == "denied" and event.get("source_refs"):
        failures.append(f"{case_id}: denied audit event must not include source_refs")

    return failures


def validate_usage_event(
    case_id: str,
    event: dict[str, Any],
    call: dict[str, Any],
    required_fields: set[str],
) -> list[str]:
    failures: list[str] = []
    missing_fields = required_fields - set(event)
    if missing_fields:
        failures.append(f"{case_id}: usage event missing fields {sorted(missing_fields)}")

    if event.get("event_type") not in USAGE_EVENT_TYPES:
        failures.append(f"{case_id}: invalid usage event_type {event.get('event_type')!r}")
    if event.get("event_type") != call["usage_event_type"]:
        failures.append(f"{case_id}: usage event_type does not match expectation")
    if event.get("counts") != call["usage_counts"]:
        failures.append(f"{case_id}: usage counts do not match expectation")
    if not isinstance(event.get("latency_ms"), int) or event["latency_ms"] < 0:
        failures.append(f"{case_id}: usage latency_ms must be a non-negative integer")
    if "principal_id" in event:
        failures.append(f"{case_id}: usage event must not store principal_id")
    if "source_refs" in event:
        failures.append(f"{case_id}: usage event must not store source_refs")
    if "parameters_hash" in event or "parameters" in event:
        failures.append(f"{case_id}: usage event must not store request parameters")

    return failures


def validate_safe_records(
    case_id: str,
    audit_event: dict[str, Any],
    usage_event: dict[str, Any],
    expectations: dict[str, Any],
    prohibited_literals: set[str],
) -> list[str]:
    failures: list[str] = []
    prohibited_fields = set(expectations["prohibited_record_fields"])
    for record_name, record in (("audit", audit_event), ("usage", usage_event)):
        present_fields = prohibited_fields & set(record)
        if present_fields:
            failures.append(
                f"{case_id}: {record_name} record stores prohibited fields {sorted(present_fields)}"
            )
        record_strings = set(flatten_strings(record))
        leaked = sorted(
            literal
            for literal in prohibited_literals
            if literal and literal in record_strings
        )
        if leaked:
            failures.append(f"{case_id}: {record_name} record leaks raw fixture text {leaked}")
    return failures


def main() -> int:
    manifest = load_json(GOLDEN / "manifest.json")
    anchors = load_json(GOLDEN / "expected-anchors.json")
    expectations = load_json(GOLDEN / "expected-audit-usage.json")
    sql = MIGRATION.read_text(encoding="utf-8")

    known_source_refs = {anchor["source_ref"] for anchor in anchors["anchors"]}
    prohibited_literals = {anchor["expected_excerpt"] for anchor in anchors["anchors"]}
    failures = validate_schema_contract(manifest, expectations, sql)

    required_audit_fields = set(expectations["required_audit_fields"])
    required_usage_fields = set(expectations["required_usage_fields"])
    audit_events: list[dict[str, Any]] = []
    usage_events: list[dict[str, Any]] = []

    for call in expectations["mcp_call_expectations"]:
        case_id = call["case_id"]
        audit_event = write_audit_event(call)
        usage_event = write_usage_event(call)
        audit_events.append(audit_event)
        usage_events.append(usage_event)

        failures.extend(
            validate_audit_event(
                case_id,
                audit_event,
                call,
                required_audit_fields,
                known_source_refs,
            )
        )
        failures.extend(validate_usage_event(case_id, usage_event, call, required_usage_fields))
        failures.extend(
            validate_safe_records(
                case_id,
                audit_event,
                usage_event,
                expectations,
                prohibited_literals,
            )
        )

    if len(audit_events) != len(usage_events):
        failures.append("expected one usage event for every audited MCP call")

    if failures:
        for failure in failures:
            print(f"FAIL: {failure}", file=sys.stderr)
        return 1

    print(
        "Audit/usage harness passed "
        f"({len(audit_events)} audit events, "
        f"{len(usage_events)} usage events, schema and logging-safety checks)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

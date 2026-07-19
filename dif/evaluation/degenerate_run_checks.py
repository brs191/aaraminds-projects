#!/usr/bin/env python3
"""Validate DIF P0 degenerate ingestion-run guard expectations.

This stdlib-only harness runs before the ingestion service exists. It verifies
that the SQL migration contains the required promotion guard and that the golden
promotion cases reject empty, failed, and anchorless/passageless extraction runs.
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
GOLDEN = ROOT / "evaluation" / "golden"
MIGRATION = ROOT / "code" / "migrations" / "001_dif_meta_initial.sql"

VALID_STATUSES = {"running", "completed", "failed", "cancelled"}


def load_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def table_body(sql: str, table_name: str) -> str:
    pattern = rf"CREATE TABLE IF NOT EXISTS dif_meta\.{table_name} \((.*?)\n\);"
    match = re.search(pattern, sql, flags=re.DOTALL)
    if not match:
        raise ValueError(f"missing dif_meta.{table_name} table in migration")
    return match.group(1)


def table_columns(sql: str, table_name: str) -> set[str]:
    columns: set[str] = set()
    for raw_line in table_body(sql, table_name).splitlines():
        line = raw_line.strip()
        if not line or line.startswith("CONSTRAINT"):
            continue
        columns.add(line.split()[0].rstrip(","))
    return columns


def normalize_sql(sql: str) -> str:
    return re.sub(r"\s+", " ", sql).strip()


def known_corpora(manifest: dict[str, Any]) -> set[str]:
    return {corpus["corpus_id"] for corpus in manifest["corpora"]}


def known_sources(manifest: dict[str, Any]) -> set[str]:
    return {source["source_id"] for source in manifest["sources"]}


def promotion_decision(run: dict[str, Any]) -> tuple[bool, str]:
    if run["status"] != "completed":
        return False, "run_not_completed"
    if run["document_count"] <= 0:
        return False, "degenerate_no_documents"
    if run["node_count"] <= 0:
        return False, "degenerate_no_nodes"
    if run["anchor_count"] <= 0:
        return False, "degenerate_no_anchors"
    if run["passage_count"] <= 0:
        return False, "degenerate_no_passages"
    return True, "promotable"


def write_ingestion_run(run: dict[str, Any]) -> dict[str, Any]:
    can_promote, _ = promotion_decision(run)
    record = {
        "run_id": run["run_id"],
        "corpus_id": run["corpus_id"],
        "source_id": run.get("source_id"),
        "status": run["status"],
        "document_count": run["document_count"],
        "node_count": run["node_count"],
        "edge_count": run["edge_count"],
        "anchor_count": run["anchor_count"],
        "passage_count": run["passage_count"],
        "caveat_count": run["caveat_count"],
        "run_metrics": {
            "promotion_decision": "allow" if can_promote else "deny",
            "promotion_reason": promotion_decision(run)[1],
        },
        "error_message": run.get("error_message"),
        "promoted": can_promote,
    }
    return {key: value for key, value in record.items() if value is not None}


def validate_schema_contract(expectations: dict[str, Any], sql: str) -> list[str]:
    failures: list[str] = []
    columns = table_columns(sql, "ingestion_runs")
    body = normalize_sql(table_body(sql, "ingestion_runs"))

    for field in expectations["required_ingestion_run_fields"]:
        if field not in columns:
            failures.append(f"ingestion_runs schema missing required field {field!r}")

    for term in expectations["required_promoted_guard_terms"]:
        if term not in body:
            failures.append(f"ingestion_runs promoted guard missing term {term!r}")

    if "promoted = false OR" not in body:
        failures.append("ingestion_runs promoted guard must allow unpromoted degenerate records")

    return failures


def validate_case(
    run: dict[str, Any],
    record: dict[str, Any],
    corpora: set[str],
    sources: set[str],
) -> list[str]:
    failures: list[str] = []
    case_id = run["case_id"]
    can_promote, reason = promotion_decision(run)

    if run["corpus_id"] not in corpora:
        failures.append(f"{case_id}: unknown corpus_id {run['corpus_id']!r}")
    if run.get("source_id") and run["source_id"] not in sources:
        failures.append(f"{case_id}: unknown source_id {run['source_id']!r}")
    if run["status"] not in VALID_STATUSES:
        failures.append(f"{case_id}: invalid status {run['status']!r}")

    for count_field in (
        "document_count",
        "node_count",
        "edge_count",
        "anchor_count",
        "passage_count",
        "caveat_count",
    ):
        if not isinstance(run[count_field], int) or run[count_field] < 0:
            failures.append(f"{case_id}: {count_field} must be a non-negative integer")

    if can_promote != run["expected_can_promote"]:
        failures.append(
            f"{case_id}: expected can_promote={run['expected_can_promote']}, got {can_promote}"
        )
    if reason != run["expected_reason"]:
        failures.append(f"{case_id}: expected reason {run['expected_reason']!r}, got {reason!r}")
    if record["promoted"] != can_promote:
        failures.append(f"{case_id}: written promoted flag does not match decision")
    if record["run_metrics"]["promotion_reason"] != reason:
        failures.append(f"{case_id}: run_metrics promotion_reason does not match decision")
    if not can_promote and record["promoted"]:
        failures.append(f"{case_id}: degenerate run must not be promoted")

    return failures


def main() -> int:
    manifest = load_json(GOLDEN / "manifest.json")
    expectations = load_json(GOLDEN / "expected-degenerate-runs.json")
    sql = MIGRATION.read_text(encoding="utf-8")

    failures = validate_schema_contract(expectations, sql)
    corpora = known_corpora(manifest)
    sources = known_sources(manifest)
    records: list[dict[str, Any]] = []

    for run in expectations["promotion_cases"]:
        record = write_ingestion_run(run)
        records.append(record)
        failures.extend(validate_case(run, record, corpora, sources))

    promoted_records = [record for record in records if record["promoted"]]
    if len(promoted_records) != 1:
        failures.append(f"expected exactly one promotable healthy run, got {len(promoted_records)}")

    if failures:
        for failure in failures:
            print(f"FAIL: {failure}", file=sys.stderr)
        return 1

    print(
        "Degenerate-run harness passed "
        f"({len(records)} promotion cases, "
        f"{len(records) - len(promoted_records)} blocked degenerate/non-complete runs)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

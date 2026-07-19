#!/usr/bin/env python3
"""Run the DIF P0 golden evaluation gate.

The runner intentionally reports measured baselines only: command durations,
exit codes, and harness output summaries. It does not encode quality targets or
production SLOs.
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
CODE = ROOT / "code"

TARGETED_GO_PACKAGES = [
    "./libs/config",
    "./libs/logging",
    "./libs/requestctx",
    "./libs/migrations",
    "./libs/admission",
    "./libs/sourceanchors",
    "./libs/ingestionruns",
    "./libs/extraction",
    "./libs/graphemit",
    "./libs/retrieval",
    "./libs/embeddings",
    "./libs/searchdocs",
    "./libs/mcpapi",
    "./libs/auditusage",
    "./libs/health",
    "./libs/rifcompat",
    "./libs/codeentities",
]


@dataclass(frozen=True)
class Check:
    check_id: str
    description: str
    command: list[str]
    cwd: Path


CHECKS = [
    Check(
        check_id="go-targeted-components",
        description="Targeted Go component tests for all P0 libraries",
        command=["go", "test", *TARGETED_GO_PACKAGES],
        cwd=CODE,
    ),
    Check(
        check_id="go-full",
        description="Full Go unit test run",
        command=["go", "test", "./..."],
        cwd=CODE,
    ),
    Check(
        check_id="go-build",
        description="Build all Go packages and service entry points",
        command=["go", "build", "./..."],
        cwd=CODE,
    ),
    Check(
        check_id="source-anchor-roundtrip",
        description="Source-anchor round-trip harness",
        command=["python3", "evaluation/source_anchor_roundtrip.py"],
        cwd=ROOT,
    ),
    Check(
        check_id="json-caveats",
        description="JSON caveat and failure behavior harness",
        command=["python3", "evaluation/json_caveat_checks.py"],
        cwd=ROOT,
    ),
    Check(
        check_id="rif-compatibility",
        description="RIF compatibility fixture harness",
        command=["python3", "evaluation/rif_compatibility_checks.py"],
        cwd=ROOT,
    ),
    Check(
        check_id="search-docs-contract",
        description="search_docs anchored retrieval contract harness",
        command=["python3", "evaluation/search_docs_checks.py"],
        cwd=ROOT,
    ),
    Check(
        check_id="audit-usage",
        description="Audit and usage write contract harness",
        command=["python3", "evaluation/audit_usage_checks.py"],
        cwd=ROOT,
    ),
    Check(
        check_id="degenerate-run-guard",
        description="Degenerate ingestion-run guard harness",
        command=["python3", "evaluation/degenerate_run_checks.py"],
        cwd=ROOT,
    ),
    Check(
        check_id="path-ci-baseline",
        description="Documentation path and CI baseline safety harness",
        command=["python3", "evaluation/path_checks.py"],
        cwd=ROOT,
    ),
]


def run_check(check: Check) -> dict[str, Any]:
    start = time.perf_counter()
    completed = subprocess.run(
        check.command,
        cwd=check.cwd,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    duration = time.perf_counter() - start
    stdout = completed.stdout.strip()
    stderr = completed.stderr.strip()
    return {
        "check_id": check.check_id,
        "description": check.description,
        "command": shell_command(check),
        "cwd": str(check.cwd),
        "exit_code": completed.returncode,
        "duration_seconds": round(duration, 3),
        "stdout_summary": summarize(stdout),
        "stderr_summary": summarize(stderr),
    }


def shell_command(check: Check) -> str:
    return " ".join(check.command)


def summarize(output: str) -> str:
    if not output:
        return ""
    lines = [line for line in output.splitlines() if line.strip()]
    if len(lines) <= 3:
        return "\n".join(lines)
    return "\n".join(lines[-3:])


def print_report(results: list[dict[str, Any]]) -> None:
    passed = sum(1 for result in results if result["exit_code"] == 0)
    failed = len(results) - passed
    total_duration = round(sum(result["duration_seconds"] for result in results), 3)
    print("DIF P0 golden evaluation")
    print(f"checks: {len(results)} total, {passed} passed, {failed} failed")
    print(f"measured_duration_seconds: {total_duration}")
    print()
    for result in results:
        status = "PASS" if result["exit_code"] == 0 else "FAIL"
        print(f"[{status}] {result['check_id']} ({result['duration_seconds']}s)")
        print(f"  cwd: {result['cwd']}")
        print(f"  command: {result['command']}")
        if result["stdout_summary"]:
            print("  stdout:")
            for line in result["stdout_summary"].splitlines():
                print(f"    {line}")
        if result["stderr_summary"]:
            print("  stderr:")
            for line in result["stderr_summary"].splitlines():
                print(f"    {line}")
        print()


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--json-output",
        type=Path,
        help="Optional path for measured run results. The file is overwritten only when this flag is provided.",
    )
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    results = [run_check(check) for check in CHECKS]
    print_report(results)
    if args.json_output:
        args.json_output.write_text(json.dumps({"checks": results}, indent=2) + "\n", encoding="utf-8")
    return 0 if all(result["exit_code"] == 0 for result in results) else 1


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

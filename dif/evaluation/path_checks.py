#!/usr/bin/env python3
"""Validate DIF P0 documentation paths and CI baseline safety."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / ".github" / "workflows" / "ci.yml"

REQUIRED_PATHS = [
    ".github/copilot-instructions.md",
    ".github/workflows/ci.yml",
    "action_plan.md",
    "tracking/phase-gate-status.md",
    "prompts.md",
    "code/go.mod",
    "code/README.md",
    "code/migrations/001_dif_meta_initial.sql",
    "design/adr/ADR-003-source-acl-posture.md",
    "design/adr/ADR-005-parser-strategy.md",
    "design/adr/ADR-006-json-expansion-limits.md",
    "design/adr/ADR-007-source-anchor-contract.md",
    "design/adr/ADR-008-mcp-gateway-auth-model.md",
    "design/adr/ADR-009-ingestion-orchestration.md",
    "design/adr/ADR-010-embedding-strategy.md",
    "design/adr/ADR-011-evaluation-gates.md",
    "design/adr/ADR-012-observability-audit-schema.md",
    "design/adr/ADR-013-security-threat-model.md",
    "design/adr/ADR-016-rif-compatibility-layer.md",
    "evaluation/README.md",
    "evaluation/run_p0.py",
    "evaluation/source_anchor_roundtrip.py",
    "evaluation/json_caveat_checks.py",
    "evaluation/rif_compatibility_checks.py",
    "evaluation/search_docs_checks.py",
    "evaluation/audit_usage_checks.py",
    "evaluation/degenerate_run_checks.py",
]

REQUIRED_WORKFLOW_TERMS = [
    "python3 evaluation/run_p0.py",
    "postgres:16",
    "psql -v ON_ERROR_STOP=1 -f code/migrations/001_dif_meta_initial.sql",
    "actions/setup-go@v5",
    "actions/setup-python@v5",
]

FORBIDDEN_WORKFLOW_PATTERNS = [
    r"AZURE_CLIENT_ID",
    r"AZURE_TENANT_ID",
    r"AZURE_SUBSCRIPTION_ID",
    r"azure/login",
    r"azurecr\.io",
    r"\bACR\b",
    r"docker\s+push",
    r"jf\s+docker\s+push",
    r"jfrog/setup-jfrog-cli",
]


def validate_required_paths() -> list[str]:
    failures: list[str] = []
    for relative in REQUIRED_PATHS:
        if not (ROOT / relative).exists():
            failures.append(f"missing required path: {relative}")
    return failures


def validate_workflow() -> list[str]:
    if not WORKFLOW.exists():
        return [f"missing CI workflow: {WORKFLOW.relative_to(ROOT)}"]
    text = WORKFLOW.read_text(encoding="utf-8")
    failures: list[str] = []
    for term in REQUIRED_WORKFLOW_TERMS:
        if term not in text:
            failures.append(f"CI workflow missing required term: {term}")
    for pattern in FORBIDDEN_WORKFLOW_PATTERNS:
        if re.search(pattern, text):
            failures.append(f"CI workflow contains forbidden publish/deploy/secret term matching: {pattern}")
    return failures


def main() -> int:
    failures = validate_required_paths()
    failures.extend(validate_workflow())
    if failures:
        for failure in failures:
            print(f"FAIL: {failure}", file=sys.stderr)
        return 1
    print(
        "Path/CI baseline harness passed "
        f"({len(REQUIRED_PATHS)} paths, "
        f"{len(REQUIRED_WORKFLOW_TERMS)} CI terms, "
        f"{len(FORBIDDEN_WORKFLOW_PATTERNS)} safety exclusions)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

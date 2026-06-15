#!/usr/bin/env python3
"""Eval harness for the Azure Network Topology Reviewer engine.

Usage:
    python run_eval.py [--fixtures-dir DIR] [--engine-dir DIR]
    python run_eval.py --help

Gate thresholds (CI exit code 1 if missed):
    - Precision >= 0.95 overall
    - Recall >= 0.90 for High+Critical
    - Recall >= 0.80 for Medium

All output written to stdout and to phase-1/eval/last_run.json.
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


# ─── Gate thresholds ──────────────────────────────────────────────────────────
GATE_PRECISION_OVERALL = 0.95
GATE_RECALL_HIGH_CRITICAL = 0.90
GATE_RECALL_MEDIUM = 0.80

# ─── Severity groups ──────────────────────────────────────────────────────────
HIGH_CRITICAL = {"Critical", "High"}
MEDIUM_GROUP = {"Medium"}


def run_engine(fixture_path: Path, engine_dir: Path) -> list[dict[str, Any]]:
    """Run the Go CLI on one fixture. Returns parsed []Finding or raises."""
    result = subprocess.run(
        ["go", "run", "./cmd/analyze/...", str(fixture_path.resolve())],
        cwd=str(engine_dir),
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"Engine exited {result.returncode} for {fixture_path.name}:\n"
            f"  stderr: {result.stderr.strip()}"
        )
    raw = result.stdout.strip()
    if not raw or raw == "null":
        return []
    return json.loads(raw)


def load_answer_key(answer_keys_dir: Path, fixture_name: str) -> dict[str, Any]:
    """Load answer key for a fixture. Returns dict with expected_findings list."""
    key_path = answer_keys_dir / fixture_name
    if not key_path.exists():
        raise FileNotFoundError(f"Answer key not found: {key_path}")
    with open(key_path) as f:
        return json.load(f)


def matches(engine_finding: dict[str, Any], expected: dict[str, Any]) -> bool:
    """Return True if engine_finding satisfies all expected criteria.

    Matching rules:
    - type:     substring match (expected["type"] in engine_finding["type"])
    - severity: exact match (case-sensitive)
    - resource: exact match
    """
    if expected.get("type") and expected["type"] not in engine_finding.get("type", ""):
        return False
    if expected.get("severity") and expected["severity"] != engine_finding.get("severity", ""):
        return False
    if expected.get("resource") and expected["resource"] != engine_finding.get("resource", ""):
        return False
    return True


def score_fixture(
    engine_findings: list[dict],
    expected_findings: list[dict],
) -> dict[str, Any]:
    """Compute TP/FP/FN and per-finding match details for one fixture."""
    matched_engine: set[int] = set()
    matched_expected: set[int] = set()

    # For each expected finding, find the first unmatched engine finding that satisfies it
    for ei, exp in enumerate(expected_findings):
        for fi, eng in enumerate(engine_findings):
            if fi in matched_engine:
                continue
            if matches(eng, exp):
                matched_engine.add(fi)
                matched_expected.add(ei)
                break

    tp = len(matched_expected)
    fp = len(engine_findings) - len(matched_engine)
    fn = len(expected_findings) - tp

    precision = tp / (tp + fp) if (tp + fp) > 0 else 1.0
    recall = tp / (tp + fn) if (tp + fn) > 0 else 1.0
    f1 = (
        2 * precision * recall / (precision + recall)
        if (precision + recall) > 0
        else 0.0
    )

    # Collect unmatched engine findings (FPs) and unmatched expected (FNs)
    fp_findings = [engine_findings[i] for i in range(len(engine_findings)) if i not in matched_engine]
    fn_findings = [expected_findings[i] for i in range(len(expected_findings)) if i not in matched_expected]

    return {
        "tp": tp,
        "fp": fp,
        "fn": fn,
        "precision": precision,
        "recall": recall,
        "f1": f1,
        "fp_findings": fp_findings,
        "fn_findings": fn_findings,
        "engine_findings": engine_findings,
        "expected_findings": expected_findings,
    }


def fixture_pass(score: dict) -> bool:
    """A fixture passes if precision >= threshold AND recall >= threshold.

    We use the overall gate thresholds at the fixture level for the per-row
    status indicator (green/red) — the CI gate uses aggregated severity groups.
    """
    return score["precision"] >= GATE_PRECISION_OVERALL and score["recall"] >= GATE_RECALL_HIGH_CRITICAL


def main() -> int:
    parser = argparse.ArgumentParser(description="Azure Network Topology Reviewer — Eval Harness")
    script_dir = Path(__file__).parent
    parser.add_argument(
        "--fixtures-dir",
        type=Path,
        default=script_dir / "fixtures",
        help="Directory containing fixture JSON files (default: ./fixtures)",
    )
    parser.add_argument(
        "--answer-keys-dir",
        type=Path,
        default=script_dir / "answer-keys",
        help="Directory containing answer key JSON files (default: ./answer-keys)",
    )
    parser.add_argument(
        "--engine-dir",
        type=Path,
        default=script_dir.parent.parent / "engine" / "go",
        help="Directory containing engine go.mod (default: ../../engine/go)",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=script_dir / "last_run.json",
        help="Path to write JSON run report (default: ./last_run.json)",
    )
    args = parser.parse_args()

    fixtures_dir: Path = args.fixtures_dir
    answer_keys_dir: Path = args.answer_keys_dir
    engine_dir: Path = args.engine_dir
    output_path: Path = args.output

    # Discover fixtures in deterministic order
    fixture_files = sorted(fixtures_dir.glob("*.json"))
    if not fixture_files:
        print(f"ERROR: No fixtures found in {fixtures_dir}", file=sys.stderr)
        return 2

    # ─── Run fixtures ─────────────────────────────────────────────────────────
    run_date = datetime.now(tz=timezone.utc).strftime("%Y-%m-%dT%H:%M:%S UTC")
    print(f"\n=== Azure Network Topology Reviewer — Eval Report ===")
    print(f"Date: {run_date}")
    print(f"Engine dir: {engine_dir}")
    print(f"Fixtures: {fixtures_dir} ({len(fixture_files)} files)\n")

    fixture_results = []
    errors = []

    for fixture_path in fixture_files:
        fname = fixture_path.name

        # Load answer key
        try:
            ak = load_answer_key(answer_keys_dir, fname)
        except FileNotFoundError as e:
            errors.append(f"Missing answer key: {e}")
            print(f"  ⚠️  {fname:<55} SKIPPED (no answer key)")
            continue

        expected_findings = ak.get("expected_findings", [])

        # Run engine
        try:
            engine_findings = run_engine(fixture_path, engine_dir)
        except (RuntimeError, json.JSONDecodeError) as e:
            errors.append(f"Engine error on {fname}: {e}")
            print(f"  ❌ {fname:<55} ENGINE ERROR: {e}")
            continue

        score = score_fixture(engine_findings, expected_findings)
        passed = fixture_pass(score)
        icon = "✅" if passed else "❌"

        print(
            f"  {icon} {fname:<55} "
            f"precision={score['precision']:.2f} recall={score['recall']:.2f} "
            f"f1={score['f1']:.2f} "
            f"(TP={score['tp']} FP={score['fp']} FN={score['fn']})"
        )

        if not passed:
            if score["fn_findings"]:
                print(f"      FN (missed expected):")
                for fn in score["fn_findings"]:
                    print(f"        - [{fn.get('severity','?')}] {fn.get('type','?')} @ {fn.get('resource','?')}")
            if score["fp_findings"]:
                print(f"      FP (unexpected):")
                for fp in score["fp_findings"]:
                    print(f"        + [{fp.get('severity','?')}] {fp.get('type','?')} @ {fp.get('resource','?')}")

        fixture_results.append({
            "fixture": fname,
            "passed": passed,
            "score": {
                "tp": score["tp"],
                "fp": score["fp"],
                "fn": score["fn"],
                "precision": round(score["precision"], 4),
                "recall": round(score["recall"], 4),
                "f1": round(score["f1"], 4),
            },
            "engine_findings": score["engine_findings"],
            "expected_findings": score["expected_findings"],
            "fp_findings": score["fp_findings"],
            "fn_findings": score["fn_findings"],
        })

    # ─── Aggregate gate metrics by severity group ─────────────────────────────
    agg = {
        "overall":        {"tp": 0, "fp": 0, "fn": 0},
        "high_critical":  {"tp": 0, "fp": 0, "fn": 0},
        "medium":         {"tp": 0, "fp": 0, "fn": 0},
    }

    for fr in fixture_results:
        s = fr["score"]
        agg["overall"]["tp"] += s["tp"]
        agg["overall"]["fp"] += s["fp"]
        agg["overall"]["fn"] += s["fn"]

        # Re-score by severity group using raw engine and expected findings
        eng_hc = [f for f in fr["engine_findings"] if f.get("severity") in HIGH_CRITICAL]
        exp_hc = [f for f in fr["expected_findings"] if f.get("severity") in HIGH_CRITICAL]
        sc_hc = score_fixture(eng_hc, exp_hc)
        agg["high_critical"]["tp"] += sc_hc["tp"]
        agg["high_critical"]["fp"] += sc_hc["fp"]
        agg["high_critical"]["fn"] += sc_hc["fn"]

        eng_m = [f for f in fr["engine_findings"] if f.get("severity") in MEDIUM_GROUP]
        exp_m = [f for f in fr["expected_findings"] if f.get("severity") in MEDIUM_GROUP]
        sc_m = score_fixture(eng_m, exp_m)
        agg["medium"]["tp"] += sc_m["tp"]
        agg["medium"]["fp"] += sc_m["fp"]
        agg["medium"]["fn"] += sc_m["fn"]

    def _precision(a: dict) -> float:
        n = a["tp"] + a["fp"]
        return a["tp"] / n if n > 0 else 1.0

    def _recall(a: dict) -> float:
        n = a["tp"] + a["fn"]
        return a["tp"] / n if n > 0 else 1.0

    overall_precision = _precision(agg["overall"])
    hc_recall = _recall(agg["high_critical"])
    med_recall = _recall(agg["medium"])

    gate_precision = overall_precision >= GATE_PRECISION_OVERALL
    gate_hc_recall = hc_recall >= GATE_RECALL_HIGH_CRITICAL
    gate_med_recall = med_recall >= GATE_RECALL_MEDIUM

    all_gates = gate_precision and gate_hc_recall and gate_med_recall
    fixtures_passed = sum(1 for r in fixture_results if r["passed"])

    print("\nGate Results:")
    _g = lambda ok, label, actual, thresh: print(
        f"  {'✅' if ok else '❌'} {label:<40} actual={actual:.4f}  threshold={thresh:.2f}"
    )
    _g(gate_precision,  "Overall precision",    overall_precision, GATE_PRECISION_OVERALL)
    _g(gate_hc_recall,  "High+Critical recall", hc_recall,         GATE_RECALL_HIGH_CRITICAL)
    _g(gate_med_recall, "Medium recall",         med_recall,        GATE_RECALL_MEDIUM)

    overall_status = "PASS" if all_gates and not errors else "FAIL"
    print(
        f"\nOverall: {overall_status} "
        f"({fixtures_passed}/{len(fixture_results)} fixtures passed, "
        f"{'all gates met' if all_gates else 'gate(s) failed'})"
    )
    if errors:
        print("\nErrors:")
        for e in errors:
            print(f"  - {e}")

    # ─── Write last_run.json ───────────────────────────────────────────────────
    report = {
        "date": run_date,
        "overall_status": overall_status,
        "gates": {
            "precision_overall": {
                "threshold": GATE_PRECISION_OVERALL,
                "actual": round(overall_precision, 4),
                "passed": gate_precision,
            },
            "recall_high_critical": {
                "threshold": GATE_RECALL_HIGH_CRITICAL,
                "actual": round(hc_recall, 4),
                "passed": gate_hc_recall,
            },
            "recall_medium": {
                "threshold": GATE_RECALL_MEDIUM,
                "actual": round(med_recall, 4),
                "passed": gate_med_recall,
            },
        },
        "summary": {
            "fixtures_total": len(fixture_results),
            "fixtures_passed": fixtures_passed,
            "fixtures_failed": len(fixture_results) - fixtures_passed,
        },
        "aggregates": {
            "overall": {k: agg["overall"][k] for k in ("tp", "fp", "fn")},
            "high_critical": {k: agg["high_critical"][k] for k in ("tp", "fp", "fn")},
            "medium": {k: agg["medium"][k] for k in ("tp", "fp", "fn")},
        },
        "fixtures": fixture_results,
        "errors": errors,
    }

    with open(output_path, "w") as f:
        json.dump(report, f, indent=2)
    print(f"\nReport written to: {output_path}")

    return 0 if (all_gates and not errors) else 1


if __name__ == "__main__":
    sys.exit(main())

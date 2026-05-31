"""
Aggregate spike metrics against the decision gate - Token Optimizer Option C.

Reads:
  - metrics/requests.jsonl   (real-usage data written by compression_hook.py)
  - results/ab_results.jsonl (controlled A/B data written by measure.py)

Prints token-reduction, latency, and per-category summaries, then checks the
numbers against the CALIBRATED decision-gate thresholds.

Canonical thresholds live in ../tracking/milestones/M1-Decision-Gate.md
(calibrated 2026-05-26 against the VS Code 1.118 native-compression baseline) -
NOT the original SPIKE_PLAN.md [VERIFY] defaults (25 / 15), which they replace.
Every reduction figure is measured INCREMENTAL over the assistant's native
baseline. Keep this file in sync with that gate.

Usage:  python summarize.py
"""

import os
import json
import statistics

METRICS = os.getenv("METRICS_PATH_HOST", "metrics/requests.jsonl")
RESULTS = os.getenv("RESULTS", "results/ab_results.jsonl")

# Decision-gate thresholds - CALIBRATED values from
# ../tracking/milestones/M1-Decision-Gate.md (2026-05-26). These supersede the
# original SPIKE_PLAN.md defaults (Green 25 / Amber 15), lowered to clear a real
# gap above the VS Code 1.118 native-compression baseline. All figures are
# INCREMENTAL over the assistant's native compression. Adjust there and here
# together.
GREEN_REDUCTION = 20.0           # median % input-token reduction (was 25.0 pre-calibration)
AMBER_REDUCTION = 10.0           # "real but modest" floor (was 15.0 pre-calibration)
GREEN_LATENCY_P95_MS = 300.0     # chat hot path; completions carve-out is < 100 ms (gate doc)
CODEHEAVY_PASS_REDUCTION = 10.0  # M0-lite BINDING gate: code-heavy median >= this incremental


def load(path):
    if not os.path.exists(path):
        return []
    with open(path, encoding="utf-8") as f:
        return [json.loads(line) for line in f if line.strip()]


def pct(values, p):
    if not values:
        return 0.0
    s = sorted(values)
    k = max(0, min(len(s) - 1, int(round((p / 100.0) * (len(s) - 1)))))
    return s[k]


def summarize_realusage(rows):
    applied = [r for r in rows if r.get("compression_applied")]
    print("== Real-usage metrics (metrics/requests.jsonl) ==")
    if not applied:
        print("  no compressed requests recorded yet")
        print("")
        return None
    reductions = [r["reduction_pct"] for r in applied]
    latencies = [r["hook_latency_ms"] for r in applied]
    failures = sum(r.get("compression_failures", 0) for r in applied)
    print("  requests compressed   : {}".format(len(applied)))
    print("  token reduction       : median {:.1f}%  mean {:.1f}%".format(
        statistics.median(reductions), statistics.mean(reductions)))
    print("  hook latency          : median {:.0f} ms  p95 {:.0f} ms".format(
        statistics.median(latencies), pct(latencies, 95)))
    print("  compression failures  : {}".format(failures))
    print("")
    return {
        "median_reduction": statistics.median(reductions),
        "p95_latency": pct(latencies, 95),
    }


def summarize_ab(rows):
    print("== A/B harness results (results/ab_results.jsonl) ==")
    if not rows:
        print("  no A/B results yet - run measure.py")
        print("")
        return None
    reductions = [r["reduction_pct"] for r in rows]
    deltas = [r["latency_delta_ms"] for r in rows]
    print("  fixtures              : {}".format(len(rows)))
    print("  token reduction       : median {:.1f}%  mean {:.1f}%".format(
        statistics.median(reductions), statistics.mean(reductions)))
    print("  latency delta         : median {:+.0f} ms".format(statistics.median(deltas)))
    print("")

    by_cat = {}
    for r in rows:
        by_cat.setdefault(r.get("category", "uncategorized"), []).append(r["reduction_pct"])
    print("  reduction by category:")
    for cat, vals in sorted(by_cat.items()):
        print("    {:22s} median {:5.1f}%  (n={})".format(
            cat, statistics.median(vals), len(vals)))
    print("  NOTE: token reduction is only half the gate - review answer")
    print("        quality in ab_results.jsonl before trusting these numbers.")
    print("")
    return {
        "median_reduction": statistics.median(reductions),
        "by_category": {c: statistics.median(v) for c, v in by_cat.items()},
    }


def gate(realusage, ab):
    print("== Decision gate (see ../tracking/milestones/M1-Decision-Gate.md) ==")
    reduction = None
    if realusage:
        reduction = realusage["median_reduction"]
    elif ab:
        reduction = ab["median_reduction"]
    if reduction is None:
        print("  insufficient data - collect more usage before deciding")
        print("")
        return

    if reduction >= GREEN_REDUCTION:
        band = "GREEN-ish - >= {:.0f}% incremental".format(GREEN_REDUCTION)
    elif reduction >= AMBER_REDUCTION:
        band = "AMBER-ish - >= {:.0f}% but below GREEN".format(AMBER_REDUCTION)
    else:
        band = "RED-ish - below the {:.0f}% floor".format(AMBER_REDUCTION)
    print("  median token reduction: {:.1f}%  ->  {}".format(reduction, band))
    print("  (all reduction figures are INCREMENTAL over the assistant's native baseline)")
    if realusage:
        ok = realusage["p95_latency"] <= GREEN_LATENCY_P95_MS
        print("  hook latency p95      : {:.0f} ms  ({} {:.0f} ms chat budget)".format(
            realusage["p95_latency"], "within" if ok else "OVER", GREEN_LATENCY_P95_MS))

    # The M0-lite gate pivots on CODE-HEAVY prompts, not the overall median.
    codeheavy = {}
    if ab and ab.get("by_category"):
        codeheavy = {c: m for c, m in ab["by_category"].items() if "code" in c.lower()}
    if codeheavy:
        print("  code-heavy (M0-lite binding gate, >= {:.0f}% incremental):".format(
            CODEHEAVY_PASS_REDUCTION))
        for c, m in sorted(codeheavy.items()):
            verdict = "PASS" if m >= CODEHEAVY_PASS_REDUCTION else "FAIL"
            print("    {:22s} median {:5.1f}%  ->  {}".format(c, m, verdict))
    else:
        print("  M0-lite BINDING gate  : the operative number is the CODE-HEAVY category")
        print("                          median >= {:.0f}% incremental, read off the".format(
            CODEHEAVY_PASS_REDUCTION))
        print("                          'reduction by category' table above - NOT the")
        print("                          overall median banded here.")

    print("  REMINDER: a GREEN verdict also requires answer-quality regression")
    print("            <= 5% of A/B pairs overall AND <= 3% on code-heavy - a human")
    print("            judgement, not a number this script can produce.")
    print("")


def main():
    realusage = summarize_realusage(load(METRICS))
    ab = summarize_ab(load(RESULTS))
    gate(realusage, ab)


if __name__ == "__main__":
    main()

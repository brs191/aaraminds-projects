#!/usr/bin/env python3
"""Independent-correctness gate (audit H-2 residue).

Runs the deterministic reference engine over each fixture that has a HAND-DERIVED
answer key under answer-keys-independent/ (keys reasoned from the fixture inputs,
not corrected against engine output — see that folder's README) and asserts:
  * every expected finding is produced (type substring · severity exact · resource
    exact · optional evidence substring), and
  * every trap holds (the named resource has no reachable High/Critical finding).

Because the keys are an independent second source, a green run is evidence of
CORRECTNESS, not just determinism. The reference engine is checked here; twin-drift
guarantees the Go engine is identical, so this covers both.

Run:  python3 eval_independent.py   (exit 0 = pass)
"""
import glob
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, os.path.join(ROOT, "engine", "reference"))
import analyze as eng  # noqa: E402

KEYS = os.path.join(HERE, "answer-keys-independent")
FIX = os.path.join(HERE, "fixtures")


def matches(f, exp):
    if exp.get("type") and exp["type"] not in f.get("type", ""):
        return False
    if exp.get("severity") and exp["severity"] != f.get("severity", ""):
        return False
    if exp.get("resource") and exp["resource"] != f.get("resource", ""):
        return False
    if exp.get("evidence") and exp["evidence"] not in f.get("evidence", ""):
        return False
    return True


def main():
    fails, nkeys, nfind = [], 0, 0
    for kp in sorted(glob.glob(os.path.join(KEYS, "*.json"))):
        key = json.load(open(kp, encoding="utf-8"))
        nkeys += 1
        findings = eng.analyze(json.load(open(os.path.join(FIX, key["fixture"]), encoding="utf-8")))
        for exp in key.get("expected_findings", []):
            nfind += 1
            if not any(matches(f, exp) for f in findings):
                fails.append("%s: MISSING %s / %s on %s"
                             % (key["fixture"], exp.get("severity"), exp.get("type"), exp.get("resource")))
        for tr in key.get("trap_assertions", []):
            r = tr["resource"]
            bad = [f["type"] for f in findings
                   if f["resource"] == r and "reachable" in f["type"] and f["severity"] in ("High", "Critical")]
            if bad:
                fails.append("%s: TRAP violated on %s — produced %s" % (key["fixture"], r, bad))
    print("independent-eval: %d keys, %d expected findings, %d failures" % (nkeys, nfind, len(fails)))
    for x in fails:
        print("  FAIL", x)
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())

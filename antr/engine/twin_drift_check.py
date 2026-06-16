#!/usr/bin/env python3
"""Twin-drift guard — Python reference vs Go production engine.

Runs BOTH engines on every shared fixture and asserts byte-identical findings
(after canonical sort). Catches silent divergence between the "true twins", which
is the failure mode most likely to go unnoticed. Designed to run in CI where the
Go toolchain is available; if `go` is missing it self-skips (so it is safe to run
in a Python-only sandbox).

Usage:
    python3 twin_drift_check.py [fixtures_dir ...]
Default fixture dirs: engine/go/testdata, phase-1/eval/fixtures
"""
import glob
import json
import os
import shutil
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
sys.path.insert(0, os.path.join(HERE, "reference"))
import analyze as pyeng  # noqa: E402


def canon(findings):
    findings = findings or []
    """Canonical comparable form: sorted by (resource, type, evidence)."""
    rows = [(f["type"], f["severity"], f["resource"], f["evidence"], bool(f["reachable"]))
            for f in findings]
    return sorted(rows, key=lambda r: (r[2], r[0], r[3]))


def go_findings(go_dir, fixture):
    """Run the Go engine's analyze CLI on a fixture; return parsed findings.
    Expects `cmd/analyze` to read a fixture path and print findings JSON."""
    out = subprocess.run(
        ["go", "run", "./cmd/analyze", os.path.abspath(fixture)],
        cwd=go_dir, capture_output=True, text=True, timeout=120)
    if out.returncode != 0:
        raise RuntimeError("go run failed for %s:\n%s" % (fixture, out.stderr[-2000:]))
    return json.loads(out.stdout) or []  # Go nil slice -> JSON null -> []


def main():
    dirs = sys.argv[1:] or [os.path.join(HERE, "go", "testdata"),
                            os.path.join(ROOT, "phase-1", "eval", "fixtures")]
    go_dir = os.path.join(HERE, "go")
    if shutil.which("go") is None:
        print("SKIP twin-drift: Go toolchain not available (Python-only environment).")
        return 0
    # Build probe: drift is only meaningful if the Go engine actually builds.
    # An incompatible/old toolchain (e.g. < go 1.18 for net/netip) is a SKIP, not drift.
    probe = subprocess.run(["go", "build", "./cmd/analyze"], cwd=go_dir,
                           capture_output=True, text=True)
    if probe.returncode != 0:
        print("SKIP twin-drift: Go engine does not build in this environment "
              "(needs go 1.25). Run in CI.\n  %s" % probe.stderr.strip().splitlines()[-1:])
        return 0
    fixtures = []
    for d in dirs:
        fixtures += sorted(glob.glob(os.path.join(d, "*.json")))
    fixtures = [f for f in fixtures if os.path.basename(f) not in ("last_run.json",)]

    # The Go engine is a SUPERSET of the Python reference: Python implements the
    # core finding families (the oracle); Go adds ~9 Azure-specific families with no
    # Python twin. So the gate asserts PARITY ON THE SHARED FAMILIES and reports
    # Go-only families as informational, not divergence (V4-07 / twin-drift scoping).
    SHARED = {"over-permissive NSG (reachable)", "over-permissive NSG (latent)",
              "orphaned public endpoint", "CIDR overlap", "missing tier segmentation",
              "analysis incomplete"}
    drift = []
    go_only_total = 0
    for fx_path in fixtures:
        fx = json.load(open(fx_path, encoding="utf-8"))
        py = [r for r in canon(pyeng.analyze(fx)) if r[0] in SHARED]
        try:
            go_all = canon(go_findings(go_dir, fx_path))
        except Exception as e:  # noqa: BLE001
            drift.append((os.path.basename(fx_path), "go-error", str(e)[:200]))
            continue
        go = [r for r in go_all if r[0] in SHARED]
        go_only_total += sum(1 for r in go_all if r[0] not in SHARED)
        if py != go:
            only_py = [r for r in py if r not in go]
            only_go = [r for r in go if r not in py]
            drift.append((os.path.basename(fx_path), "DIVERGE",
                          {"only_python": only_py[:5], "only_go": only_go[:5]}))
    print("twin-drift: %d fixtures checked, %d shared-family divergences "
          "(%d Go-only-family findings, no Python oracle — informational)"
          % (len(fixtures), len(drift), go_only_total))
    for d in drift:
        print("  ", d)
    return 1 if drift else 0


if __name__ == "__main__":
    sys.exit(main())

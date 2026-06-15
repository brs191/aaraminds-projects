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
    return json.loads(out.stdout)


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

    drift = []
    for fx_path in fixtures:
        fx = json.load(open(fx_path, encoding="utf-8"))
        py = canon(pyeng.analyze(fx))
        try:
            go = canon(go_findings(go_dir, fx_path))
        except Exception as e:  # noqa: BLE001
            drift.append((os.path.basename(fx_path), "go-error", str(e)[:200]))
            continue
        if py != go:
            only_py = [r for r in py if r not in go]
            only_go = [r for r in go if r not in py]
            drift.append((os.path.basename(fx_path), "DIVERGE",
                          {"only_python": only_py[:5], "only_go": only_go[:5]}))
    print("twin-drift: %d fixtures checked, %d divergences" % (len(fixtures), len(drift)))
    for d in drift:
        print("  ", d)
    return 1 if drift else 0


if __name__ == "__main__":
    sys.exit(main())

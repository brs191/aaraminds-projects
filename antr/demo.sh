#!/usr/bin/env bash
# antr end-to-end demo — fixture → analyze → view families → report, no Azure needed.
# Usage:  ./demo.sh [fixture.json]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
FX="${1:-phase-4/fixtures/estate-multisub.json}"
OUT="out/demo"
mkdir -p "$OUT/views"

echo "════════════════════════════════════════════════════════════════"
echo " antr demo  ·  discover → analyze → deliver"
echo " fixture: $FX"
echo "════════════════════════════════════════════════════════════════"

echo
echo "── 1. ANALYZE  (deterministic reference engine) ───────────────"
python3 engine/reference/analyze.py "$FX" > "$OUT/findings.json"
python3 - "$OUT/findings.json" <<'PY'
import json, sys, collections
fs = json.load(open(sys.argv[1]))
c = collections.Counter(f["severity"] for f in fs)
print("   findings: %d" % len(fs))
for s in ("Critical", "High", "Medium", "Low", "Informational"):
    if c.get(s):
        print("     %-14s %d" % (s, c[s]))
top = [f for f in fs if f["severity"] in ("Critical", "High")][:4]
if top:
    print("   top exposures:")
    for f in top:
        print("     • [%s] %s — %s" % (f["severity"], f["type"], f["resource"]))
PY

echo
echo "── 2. DELIVER  (view families) ────────────────────────────────"
python3 phase-4/viz/views.py "$FX" --out-dir "$OUT/views" | sed 's/^/   /'

echo
echo "── done ───────────────────────────────────────────────────────"
echo "   findings : $OUT/findings.json"
echo "   diagrams : $OUT/views/*.drawio   (open in diagrams.net)"

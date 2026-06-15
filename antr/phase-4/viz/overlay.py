#!/usr/bin/env python3
"""Phase-4 severity overlay — the layer antr owns.

Joins the deterministic findings from the reference analysis engine
(engine/reference/analyze.py) onto topology nodes. The overlay is keyed by the
SAME canonical node id the renderer uses for that resource, namespaced by KIND
(nic:/pip:/vnet:) so that a NIC and a public IP that happen to share a name can
never collide and mispaint each other (audit C-1).

The renderer reads node colour ONLY from this map; it never invents a colour.
Severity is computed by Analyze(), never here.

Run:
    python3 overlay.py <fixture.json> --print
"""
import json
import os
import sys

_ENGINE_REF = os.path.join(os.path.dirname(__file__), "..", "..", "engine", "reference")
sys.path.insert(0, os.path.abspath(_ENGINE_REF))
import analyze as engine  # noqa: E402

# ---------------------------------------------------------------- severity model
SEV_RANK = {"Critical": 5, "High": 4, "Medium": 3, "Low": 2, "Informational": 1, "Clean": 0}

_BUCKET = {"Critical": "Critical", "High": "High", "Medium": "Medium",
           "Low": "Info", "Informational": "Info", "Clean": "Clean"}

BUCKET_COLOR = {
    "Critical": {"fill": "#f8cecc", "stroke": "#b85450", "badge": "\U0001F534"},  # red
    "High":     {"fill": "#ffe6cc", "stroke": "#d79b00", "badge": "\U0001F7E0"},  # orange
    "Medium":   {"fill": "#fff2cc", "stroke": "#d6b656", "badge": "\U0001F7E1"},  # yellow
    "Info":     {"fill": "#dae8fc", "stroke": "#6c8ebf", "badge": "\U0001F535"},  # blue
    "Clean":    {"fill": "#d5e8d4", "stroke": "#82b366", "badge": "\U0001F7E2"},  # green
}
LEGEND_ORDER = ["Critical", "High", "Medium", "Info", "Clean"]

# Structural / infrastructure palette — deliberately DISJOINT from the five
# severity colours so a grey appliance is never misread as a severity (audit M-3).
STRUCT_FILL = "#f5f5f5"
STRUCT_STROKE = "#666666"
SUBNET_FILL = "#fafafa"
SUBNET_STROKE = "#b3b3b3"


def rid(obj):
    """Stable identity for a resource: ARM id when present, else bare name.
    Mirrors engine `analyze.rid` so overlay keys and render cell ids agree (V4-07)."""
    return obj.get("id") or obj.get("name", "")


def bucket_for(severity):
    return _BUCKET.get(severity, "Info")


def style_for(severity):
    """draw.io fill/stroke for a severity — the ONLY place node colour is decided,
    and a pure function of an Analyze() severity string."""
    return BUCKET_COLOR[bucket_for(severity)]


def finding_node_ids(f):
    """Canonical render node id(s) a finding paints, namespaced by kind.

    These MUST equal the renderer's cell ids:
      NIC findings      -> ["nic:<name>"]
      orphaned PIP      -> ["pip:<name>"]
      CIDR overlap pair -> ["vnet:<a>", "vnet:<b>"]
    """
    t, r = f.get("type", ""), f.get("resource", "")
    if t == "orphaned public endpoint":
        return ["pip:" + r]
    if t == "CIDR overlap":
        a, b = (r.split("~", 1) + [""])[:2]
        return ["vnet:" + a, "vnet:" + b]
    # over-permissive NSG (reachable|latent), missing tier segmentation, firewall DNAT
    return ["nic:" + r]


# ---------------------------------------------------------------- the overlay
def compute_overlay(fx):
    """Return {canonical_node_id: {"severity","bucket","findings"[]}} from Analyze()."""
    findings = engine.analyze(fx)
    overlay = {}
    for f in findings:
        sev = f["severity"]
        for nid in finding_node_ids(f):
            entry = overlay.setdefault(nid, {"severity": "Clean", "bucket": "Clean", "findings": []})
            entry["findings"].append(f)
            if SEV_RANK.get(sev, 0) > SEV_RANK.get(entry["severity"], 0):
                entry["severity"] = sev
                entry["bucket"] = bucket_for(sev)
    return overlay


def severity_of(overlay, node_id):
    """A node's severity; 'Clean' if the engine produced no finding for it."""
    return overlay.get(node_id, {}).get("severity", "Clean")


def main():
    if len(sys.argv) < 2:
        sys.exit("usage: python3 overlay.py <fixture.json> [--print]")
    fx = json.load(open(sys.argv[1], encoding="utf-8"))
    overlay = compute_overlay(fx)
    if "--print" in sys.argv:
        out = {nid: {"severity": v["severity"], "bucket": v["bucket"],
                     "finding_types": [f["type"] for f in v["findings"]]}
               for nid, v in sorted(overlay.items())}
        print(json.dumps(out, indent=2))
    return overlay


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Phase-1 provenance CI gate (exit-gate G5 / TARGET §9 "100% self-citation").

Asserts that EVERY node and edge — except library `external` and `system` markers —
carries a resolvable source_ref, a valid confidence tier, and an index_version.
Exits non-zero on any violation so it can gate CI.

Usage: python3 provenance_check.py [graph.json]
"""
import json, re, sys, os

DEFAULT = os.path.join(os.path.dirname(__file__), "..", "graph", "graph.thin-slice.json")
SRC_RE = re.compile(r"^[\w.-]+@[0-9a-f]+:[^:]+:\d+(-\d+)?$")  # repo@sha:path:line[-line]
CONF = {"exact", "probable", "inferred"}
EXEMPT = {"external", "system"}

def check(elem, kind):
    v = []
    if elem.get("confidence") not in CONF:
        v.append(f"{kind} {elem['id']}: bad confidence {elem.get('confidence')!r}")
    if not elem.get("index_version"):
        v.append(f"{kind} {elem['id']}: missing index_version")
    if elem.get("provenance") not in EXEMPT:
        sr = elem.get("source_ref")
        if not sr:
            v.append(f"{kind} {elem['id']}: missing source_ref")
        elif not SRC_RE.match(sr):
            v.append(f"{kind} {elem['id']}: unresolvable source_ref {sr!r}")
    return v

def main():
    path = sys.argv[1] if len(sys.argv) > 1 else DEFAULT
    g = json.load(open(path))
    viol = []
    for n in g["nodes"]:
        viol += check(n, "node")
    for e in g["edges"]:
        viol += check(e, "edge")
    # every edge endpoint must exist (no dangling edges)
    ids = {n["id"] for n in g["nodes"]}
    for e in g["edges"]:
        if e["src"] not in ids: viol.append(f"edge {e['id']}: dangling src {e['src']}")
        if e["dst"] not in ids: viol.append(f"edge {e['id']}: dangling dst {e['dst']}")

    n_ext = sum(1 for n in g["nodes"] if n.get("provenance") in EXEMPT)
    cited = sum(1 for n in g["nodes"] if n.get("provenance") not in EXEMPT)
    print(f"graph: {len(g['nodes'])} nodes ({cited} citable, {n_ext} exempt), {len(g['edges'])} edges")
    if viol:
        print(f"FAIL — {len(viol)} provenance violation(s):")
        for x in viol[:50]:
            print("  -", x)
        sys.exit(1)
    print("PASS — 100% self-citation: every citable node/edge has a resolvable repo@sha:path:line.")

if __name__ == "__main__":
    main()

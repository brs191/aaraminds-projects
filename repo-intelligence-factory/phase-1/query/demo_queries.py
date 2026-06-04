#!/usr/bin/env python3
"""Phase-1 demo queries (exit-gate G1: cited query results on the real graph).

Runs the four Phase-1 demo queries directly over graph.thin-slice.json — the same
traversals the AGE Cypher will run, here in pure Python so they work without a live
AGE. Every returned row carries its repo@sha:path:line citation.

Usage: python3 demo_queries.py
"""
import json, os
from collections import defaultdict, deque

G = json.load(open(os.path.join(os.path.dirname(__file__), "..", "graph", "graph.thin-slice.json")))
NODE = {n["id"]: n for n in G["nodes"]}
OUT = defaultdict(list)   # src -> [(type, dst, edge)]
IN  = defaultdict(list)   # dst -> [(type, src, edge)]
for e in G["edges"]:
    OUT[e["src"]].append((e["type"], e["dst"], e))
    IN[e["dst"]].append((e["type"], e["src"], e))

def ref(nid): return NODE[nid].get("source_ref") or "(external)"
def nm(nid):  return NODE[nid]["name"]

def q1_find_callers(method_id):
    print(f"\nQ1  find_callers({nm(method_id)})  — who CALLS it")
    for t, src, e in sorted(IN[method_id]):
        if t == "CALLS":
            print(f"    <- {nm(src):42} {e['source_ref']}")

def q2_dependents(type_id, max_depth=3):
    print(f"\nQ2  dependents@depth<={max_depth}({nm(type_id)})  — reverse CALLS/INJECTS/IMPLEMENTS/ADVISES (blast radius)")
    seen, dq = {}, deque([(type_id, 0)])
    while dq:
        cur, d = dq.popleft()
        if d >= max_depth: continue
        for t, src, e in IN[cur]:
            if src not in seen:
                seen[src] = (d + 1, t, e)
                dq.append((src, d + 1))
    for src, (d, t, e) in sorted(seen.items(), key=lambda kv: (kv[1][0], nm(kv[0]))):
        print(f"    d{d} via {t:10} {nm(src):34} [{e['confidence']:8}] {e['source_ref']}")

def q3_endpoints():
    print("\nQ3  list endpoints  — EXPOSES")
    for e in G["edges"]:
        if e["type"] == "EXPOSES":
            ep = NODE[e["dst"]]
            print(f"    {ep['http_method']:5} {ep['path']:34} <- {nm(e['src'])}   {e['source_ref']}")

def q4_di(type_id):
    print(f"\nQ4  DI wiring of {nm(type_id)}  — INJECTS (out)")
    for t, dst, e in sorted(OUT[type_id]):
        if t == "INJECTS":
            print(f"    -> {nm(dst):34} [{e['confidence']:8}] {e['source_ref']}")

if __name__ == "__main__":
    print("Phase-1 demo queries over graph.thin-slice.json (every row cited)")
    q1_find_callers("method:com.att.creditcheck.csi.SoapCallService#getSoapResponse")
    q2_dependents("type:com.att.creditcheck.csi.SoapCallService", 3)
    q3_endpoints()
    q4_di("type:com.att.creditcheck.routing.v1.CCRoutingService")

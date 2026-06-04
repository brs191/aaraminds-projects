#!/usr/bin/env python3
"""Phase-1 eval-sanity (exit-gate G9, structural subset).

Phase 1 has no embeddings/retrieval yet, so this is NOT the full LLM-judge scorer.
It checks the honest Phase-1 claim: does the deterministic graph structurally
*support* the gold-set questions that fall inside the extracted slice? For each
in-slice understanding question it runs a graph probe and reports PASS (graph holds
the cited evidence) / FAIL, plus out-of-slice coverage. It also runs an impact
traversal to show per-tier blast radius. Full semantic Q&A is deferred to Phase 3.

Usage: python3 eval_sanity.py
"""
import csv, json, os
from collections import defaultdict, deque

BASE = os.path.dirname(__file__)
G = json.load(open(os.path.join(BASE, "..", "graph", "graph.thin-slice.json")))
EVAL = os.path.join(BASE, "..", "..", "phase-0", "evalset")
NODE = {n["id"]: n for n in G["nodes"]}
OUT, IN = defaultdict(list), defaultdict(list)
for e in G["edges"]:
    OUT[e["src"]].append(e); IN[e["dst"]].append(e)

def by_name(short):  # node id whose name == short
    return [nid for nid, n in NODE.items() if n["name"] == short]
def injects_out(type_short):
    ids = by_name(type_short)
    return [NODE[e["dst"]]["name"] for nid in ids for e in OUT[nid] if e["type"] == "INJECTS"]
def injectors_of(type_short):
    ids = by_name(type_short)
    return [NODE[e["src"]]["name"] for nid in ids for e in IN[nid] if e["type"] == "INJECTS"]
def callers_of(method_short):
    ids = by_name(method_short)
    return [NODE[e["src"]]["name"] for nid in ids for e in IN[nid] if e["type"] == "CALLS"]
def advised_by(aspect_short):
    ids = by_name(aspect_short)
    return [NODE[e["dst"]]["name"] for nid in ids for e in OUT[nid] if e["type"] == "ADVISES"]
def endpoints():
    return [NODE[e["dst"]]["path"] for e in G["edges"] if e["type"] == "EXPOSES"]

# in-slice probes: gold id -> (type, probe -> answer, pass predicate)
PROBES = {
 "u2": ("cross-service", lambda: injectors_of("SoapCallService"),
        lambda a: len(a) >= 9),                                   # 9 CSI clients inject the SOAP sender
 "u4": ("dataflow",      lambda: callers_of("getSoapResponse"),
        lambda a: len(a) >= 1),                                   # reachable from the call chain
 "u5": ("cross-file",    lambda: injects_out("CCRoutingService"),
        lambda a: len(a) >= 8),                                   # 8 injected beans
 "u7": ("cross-file",    lambda: advised_by("CCRoutingServiceAspect"),
        lambda a: "routeToCCApi" in a),
 "u8": ("usage",         lambda: endpoints(),
        lambda a: "/v1/public/api/credit-check" in a),
 "u15":("cross-file",    lambda: advised_by("SoapCallServiceAspect"),
        lambda a: "getSoapResponse" in a),
}

def main():
    rows = list(csv.DictReader(open(os.path.join(EVAL, "understanding-goldset.csv"))))
    total = len(rows)
    in_slice = [r for r in rows if r["id"] in PROBES]
    print("=== Understanding gold set — structural answerability (Phase-1 subset) ===")
    by_type = defaultdict(lambda: [0, 0])
    npass = 0
    for r in rows:
        if r["id"] in PROBES:
            typ, probe, ok = PROBES[r["id"]]
            ans = probe(); good = ok(ans); npass += good
            by_type[typ][1] += 1; by_type[typ][0] += good
            print(f"  {r['id']:4} [{typ:13}] {'PASS' if good else 'FAIL'}  {r['question'][:46]:46} -> {ans if len(str(ans))<60 else str(ans)[:57]+'...'}")
    print(f"\n  in-slice answered: {npass}/{len(in_slice)}  |  out-of-slice (need full extract / embeddings): {total-len(in_slice)}/{total}")
    print("  per-type (in-slice):", {k: f"{v[0]}/{v[1]}" for k, v in sorted(by_type.items())})
    bar = npass / len(in_slice) if in_slice else 0
    print(f"  structural-subset score: {bar:.0%}  (locked capability gate: >= 50% with citations)  -> {'MEETS' if bar>=0.5 else 'BELOW'}")

    # impact traversal demo (per-tier blast radius) on an in-slice hub
    print("\n=== Impact traversal — blast radius of SoapCallService (per tier) ===")
    tgt = "type:com.att.creditcheck.csi.SoapCallService"
    seen = {}; dq = deque([(tgt, 0)])
    while dq:
        cur, d = dq.popleft()
        if d >= 3: continue
        for e in IN[cur]:
            if e["src"] not in seen:
                seen[e["src"]] = (d+1, e["confidence"]); dq.append((e["src"], d+1))
    tiers = defaultdict(int)
    for _, (d, c) in seen.items(): tiers[c] += 1
    print(f"  dependents@depth<=3: {len(seen)}  by tier: {dict(tiers)}")
    imp = list(csv.DictReader(open(os.path.join(EVAL, "impact-goldset.csv"))))
    in_g = [r for r in imp if any(NODE[i] for i in NODE if NODE[i]['name'] in r['changed_entity'])]
    print(f"  impact gold rows intersecting the thin slice: {len(in_g)}/{len(imp)} "
          f"(rest need the full extraction — Tier-C SOAP/AOP lands in Phase 2)")

if __name__ == "__main__":
    main()

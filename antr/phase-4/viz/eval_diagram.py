#!/usr/bin/env python3
"""Phase-4 diagram-eval gate.

Runs the full visualization pipeline over a fixture corpus and asserts, per
fixture, that the four production root causes are retired AND the layout is legible:

  RC-1/RC-2 : peering + cross-sub edges render to present nodes; zero dangling;
              out-of-scope peers become external stubs
  RC-3      : every boundary element present in the fixture renders as a node
  RC-4      : every finding-bearing leaf node (nic:/pip:) is painted EXACTLY its
              Analyze() severity; CIDR-overlap findings render as edges; no finding
              is dropped; no invented severities
  RC-5      : no top-level / sibling vertex overlaps (readability)

A check is SKIP (not FAIL) when the fixture lacks the input. overall_status == PASS
only if every fixture passes every applicable check.

Run:
    python3 eval_diagram.py --fixtures <dir> [<dir> ...] --report <out.json>
"""
import glob
import json
import os
import sys
import tempfile
import xml.etree.ElementTree as ET

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import overlay as ov  # noqa: E402
import render_drawio as rd  # noqa: E402
import check_layout as cl  # noqa: E402


def _fills(xml):
    root = ET.fromstring(xml)
    fills = {}
    for c in root.iter("mxCell"):
        if c.get("vertex") == "1":
            style = c.get("style", "")
            fills[c.get("id")] = next((s.split("=", 1)[1] for s in style.split(";")
                                       if s.startswith("fillColor=")), None)
    return fills


def _vnet_rollup(overlay, rg, vn):
    """Expected VNet fill severity = max(CIDR sev on the VNet, contained NIC sevs)."""
    exp = ov.severity_of(overlay, "vnet:" + vn)
    for n in rg.get("networkInterfaces", []):
        if n.get("subnet", "").split("/")[0] == vn:
            s = ov.severity_of(overlay, "nic:" + ov.rid(n))
            if ov.SEV_RANK.get(s, 0) > ov.SEV_RANK.get(exp, 0):
                exp = s
    return exp


def _colour_problems(fx, overlay, level):
    """Colour-integrity for ONE level. Returns (bad, dropped, palette).

    Checked on BOTH hld and mld (third audit: hld rollup fills were unguarded).
    nic: leaves exist only in mld; in hld a NIC finding is represented through
    its VNet's rollup colour (so not 'dropped').
    """
    import analyze as eng
    rg = fx.get("resourceGraph", {})
    xml, cells, edges = rd.render(fx, level)
    vids = {c.id for c in cells}
    fills = _fills(xml)
    bad, dropped = [], []
    for nid, entry in overlay.items():
        if nid.startswith("pip:") or (nid.startswith("nic:") and level == "mld"):
            if nid not in vids:
                dropped.append([level, nid]); continue
            want = ov.style_for(entry["severity"])["fill"]
            if fills.get(nid) != want:
                bad.append([level, nid, want, fills.get(nid)])
        elif nid.startswith("vnet:"):
            for f in entry["findings"]:
                if f.get("type") == "CIDR overlap":
                    a, b = (f["resource"].split("~", 1) + [""])[:2]
                    pair = {"vnet:" + a, "vnet:" + b}
                    if not any(e.id.startswith("cidr:") and {e.source, e.target} == pair for e in edges):
                        dropped.append([level, "cidr-edge:%s~%s" % (a, b)])
    for v in rg.get("virtualNetworks", []):
        vn = v["name"]
        want = ov.style_for(_vnet_rollup(overlay, rg, vn))["fill"]
        if "vnet:" + vn in vids and fills.get("vnet:" + vn) != want:
            bad.append([level, "vnet:" + vn, want, fills.get("vnet:" + vn)])
    engine_sevs = {f["severity"] for f in eng.analyze(fx)}
    allowed = {ov.style_for(s)["fill"] for s in (engine_sevs | {"Clean"})}
    palette = [[level, nid, fills[nid]] for nid in fills
               if nid.startswith(("nic:", "pip:", "vnet:")) and fills[nid] not in allowed]
    return bad, dropped, palette


def eval_fixture(path):
    fx = json.load(open(path, encoding="utf-8"))
    rg = fx.get("resourceGraph", {})
    res = {"fixture": os.path.basename(path), "checks": {}, "errors": []}
    try:
        overlay = ov.compute_overlay(fx)
        xml_mld, cells, edges = rd.render(fx, "mld")
        rd.render(fx, "hld")
    except Exception as e:  # noqa: BLE001
        res["errors"].append("pipeline: %s: %s" % (type(e).__name__, e))
        res["status"] = "FAIL"
        return res
    vids = {c.id for c in cells}
    fills = _fills(xml_mld)
    vnet_names = {v["name"] for v in rg.get("virtualNetworks", [])}

    # --- RC-1/RC-2: peering edges resolve, none dangling ---
    pairs = set()
    for v in rg.get("virtualNetworks", []):
        for p in v.get("peerings", []):
            if p.get("remoteVnet"):
                pairs.add(tuple(sorted((v["name"], p["remoteVnet"]))))
    for xp in fx.get("crossSubscriptionPeerings", []):
        if xp.get("localVnet") and xp.get("remoteVnet"):
            pairs.add(tuple(sorted((xp["localVnet"], xp["remoteVnet"]))))
    peer_edges = [e for e in edges if e.id.startswith("peer:")]
    dangling = [e.id for e in edges if e.source not in vids or e.target not in vids]
    res["checks"]["RC1_RC2_edges"] = (
        {"status": "PASS" if (len(peer_edges) == len(pairs) and not dangling) else "FAIL",
         "unique_pairs": len(pairs), "peer_edges": len(peer_edges), "dangling": dangling}
        if pairs else {"status": "SKIP", "reason": "no peerings"})

    oos = [p["remoteVnet"] for v in rg.get("virtualNetworks", []) for p in v.get("peerings", [])
           if p.get("remoteVnet") and p["remoteVnet"] not in vnet_names]
    oos += [xp["remoteVnet"] for xp in fx.get("crossSubscriptionPeerings", [])
            if xp.get("remoteVnet") and xp["remoteVnet"] not in vnet_names]
    if oos:
        res["checks"]["RC2_external_stub"] = {
            "status": "PASS" if all(("ext:" + r) in vids for r in oos) else "FAIL",
            "out_of_scope": sorted(set(oos))}

    # --- RC-3: boundary elements present in fixture render as nodes ---
    expect = []
    if fx.get("azureFirewall"):
        expect.append("fw:" + fx["azureFirewall"]["name"])
    expect += ["gw:" + g["name"] for g in rg.get("virtualNetworkGateways", [])]
    expect += ["natgw:" + ng["name"] for ng in rg.get("natGateways", [])]
    expect += ["er:" + e["name"] for e in rg.get("expressRouteCircuits", [])]
    if any(n.get("publicIp") for n in rg.get("networkInterfaces", [])) or fx.get("azureFirewall"):
        expect.append("internet")
    res["checks"]["RC3_boundary"] = (
        {"status": "PASS" if all(x in vids for x in expect) else "FAIL",
         "expected": len(expect), "missing": [x for x in expect if x not in vids]}
        if expect else {"status": "SKIP", "reason": "no boundary elements"})

    # --- RC-4: colour integrity on BOTH levels (mld leaves + hld/mld vnet rollups) ---
    bad, dropped, palette_bad = [], [], []
    for lvl in ("mld", "hld"):
        b, d, p = _colour_problems(fx, overlay, lvl)
        bad += b; dropped += d; palette_bad += p
    painted = sum(1 for nid in overlay if nid.startswith(("nic:", "pip:")))
    res["checks"]["RC4_colour_from_analyze"] = {
        "status": "PASS" if not bad and not dropped and not palette_bad else "FAIL",
        "painted_leaf_nodes": painted, "mismatches": bad,
        "dropped_findings": dropped, "palette_violations": palette_bad}

    # --- structure: emitted XML has globally-unique cell ids, all endpoints exist ---
    root = ET.fromstring(xml_mld)
    all_ids = [c.get("id") for c in root.iter("mxCell")]
    dup_ids = sorted({i for i in all_ids if all_ids.count(i) > 1})
    endpoint_ok = all((e.source in vids and e.target in vids) for e in edges)
    res["checks"]["structure"] = {
        "status": "PASS" if not dup_ids and endpoint_ok else "FAIL",
        "duplicate_ids": dup_ids}

    # --- RC-5: layout legibility (no vertex overlaps) ---
    with tempfile.NamedTemporaryFile("w", suffix=".drawio", delete=False, encoding="utf-8") as t:
        t.write(xml_mld); tmp = t.name
    try:
        overlaps, total, _ = cl.check(tmp)
    finally:
        os.unlink(tmp)
    res["checks"]["RC5_layout"] = {"status": "PASS" if not overlaps else "FAIL",
                                   "vertices": total, "overlaps": overlaps[:5]}

    res["buckets"] = sorted({v["bucket"] for v in overlay.values()})
    # App-layer findings have no topology node (App Gateway / AKS / Front Door /
    # vWAN / APIM / cross-sub peering / PE DNS) — surface them explicitly so the
    # gate does not silently hide engine output it cannot draw (audit transparency).
    import analyze as _eng
    res["non_topology_findings"] = sorted(
        {f["type"] for f in _eng.analyze(fx)
         if f.get("type") in ov.NON_TOPOLOGY_FINDING_TYPES})
    res["status"] = "FAIL" if any(c["status"] == "FAIL" for c in res["checks"].values()) else "PASS"
    return res


def main():
    args = sys.argv[1:]
    dirs, report = [], "phase-4/out/diagram_eval.json"
    if "--fixtures" in args:
        i = args.index("--fixtures") + 1
        while i < len(args) and not args[i].startswith("--"):
            dirs.append(args[i]); i += 1
    if "--report" in args:
        report = args[args.index("--report") + 1]
    files = []
    for d in dirs:
        files += sorted(glob.glob(os.path.join(d, "*.json")))
    files = [f for f in files if os.path.basename(f) != "last_run.json"]

    results = [eval_fixture(f) for f in files]
    npass = sum(1 for r in results if r["status"] == "PASS")
    roll = {}
    for r in results:
        for k, c in r["checks"].items():
            roll.setdefault(k, {"PASS": 0, "FAIL": 0, "SKIP": 0})[c["status"]] += 1
    # suite-level severity coverage — the corpus MUST exercise every legend bucket,
    # else "colour integrity" is vacuous (third audit). Coverage now GATES.
    exercised = sorted({b for r in results for b in r.get("buckets", [])} | {"Clean"})
    coverage = {b: (b in exercised) for b in ov.LEGEND_ORDER}
    coverage_ok = all(coverage.values())
    out = {"overall_status": "PASS" if (results and npass == len(results) and coverage_ok) else "FAIL",
           "fixtures_total": len(results), "fixtures_passed": npass,
           "severity_coverage": coverage, "coverage_gate": coverage_ok,
           "rc_rollup": roll, "results": results}
    os.makedirs(os.path.dirname(report) or ".", exist_ok=True)
    json.dump(out, open(report, "w", encoding="utf-8"), indent=2)
    print("diagram-eval: %d/%d fixtures PASS -> %s" % (npass, len(results), report))
    print("  severity_coverage: %s" % coverage)
    for k, v in roll.items():
        print("  %s: %s" % (k, v))
    for r in results:
        if r["status"] != "PASS":
            print("  FAIL", r["fixture"], r.get("errors"),
                  {k: c["status"] for k, c in r["checks"].items()})
    sys.exit(0 if out["overall_status"] == "PASS" else 1)


if __name__ == "__main__":
    main()

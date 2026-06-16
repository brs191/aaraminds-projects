#!/usr/bin/env python3
"""Phase-4 view families — overview-plus-detail diagrams over ONE risk truth.

One giant diagram of a real estate is soup. Azure's own Network Watcher treats
topology as a filtered/navigation problem, not a single static canvas; graph
research on compound/nested graphs says the same. So antr emits a SET of views,
each a deterministic PROJECTION of the topology graph, all rendered by the same
deterministic renderer (render_drawio) and all painted from the SAME whole-estate
Analyze() overlay.

Design invariants (why this stays enterprise-grade and trustworthy):
  * Risk truth is computed ONCE, on the full fixture (`compute_overlay`). A view
    never re-analyses a subset — it filters which resources are DRAWN, then asks
    the renderer to paint them with the full-estate severity. So a view can hide a
    resource but can never change a verdict.
  * A view is a pure projection: deep-copy the fixture, drop resources outside the
    view's scope, preserve order. Determinism is inherited from the renderer
    (byte-identical re-render is already a CI gate).
  * Layout is delegated to render_drawio; this module owns SELECTION, not geometry
    — the same separation that lets the layout engine be swapped later (see
    design/GRAPH_IR.md).

Views:
  hld              VNets, hubs, spokes, gateways, firewall, peerings, boundary.
  mld              Full detail: subnets + NICs.
  risk             Only resources that carry a finding (+ their VNet/subnet) + boundary.
  boundary         Internet-facing paths: boundary nodes + internet-reachable NICs.
  cross-sub        Only VNets in cross-subscription peerings (the multi-sub blast radius).
  finding/<n>      One small k-hop diagram per Critical/High finding around its node.

Run:
    python3 views.py <fixture.json> [--out-dir DIR] [--k 1]
"""
import copy
import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import overlay as ov          # noqa: E402
import render_drawio as rd    # noqa: E402


# ---------------------------------------------------------------- helpers
def _nic_vnet(nic):
    s = nic.get("subnet", "")
    return s.split("/")[0] if "/" in s else ""


def _peer_adjacency(fx):
    """vnet name -> set of directly peered vnet names (local + cross-sub)."""
    adj = {}
    for v in fx.get("resourceGraph", {}).get("virtualNetworks", []):
        a = v["name"]
        for p in v.get("peerings", []):
            b = p.get("remoteVnet")
            if b:
                adj.setdefault(a, set()).add(b)
                adj.setdefault(b, set()).add(a)
    for xp in fx.get("crossSubscriptionPeerings", []):
        a, b = xp.get("localVnet"), xp.get("remoteVnet")
        if a and b:
            adj.setdefault(a, set()).add(b)
            adj.setdefault(b, set()).add(a)
    return adj


def project(fx, *, keep_vnet, keep_nic, keep_pip,
            keep_subnets_with_nics=True, keep_boundary=True, keep_xsub=True):
    """Return a NEW fixture with only the resources selected by the predicates.

    keep_vnet(vnet)->bool, keep_nic(nic)->bool, keep_pip(pip)->bool. The projection
    is structural only; pass the full-estate overlay to render() so colours are not
    recomputed from this subset. Order is preserved (determinism)."""
    pfx = copy.deepcopy(fx)
    rg = pfx.get("resourceGraph", {})

    kept_nics = [n for n in rg.get("networkInterfaces", []) if keep_nic(n)]
    kept_nic_subnets = {n.get("subnet", "") for n in kept_nics}
    rg["networkInterfaces"] = kept_nics

    new_vnets = []
    for v in rg.get("virtualNetworks", []):
        if not keep_vnet(v):
            continue
        v2 = dict(v)
        if keep_subnets_with_nics:
            v2["subnets"] = [s for s in v.get("subnets", [])
                             if (v["name"] + "/" + s["name"]) in kept_nic_subnets]
        new_vnets.append(v2)
    rg["virtualNetworks"] = new_vnets

    rg["publicIPAddresses"] = [p for p in rg.get("publicIPAddresses", []) if keep_pip(p)]

    if not keep_boundary:
        for k in ("virtualNetworkGateways", "expressRouteCircuits", "natGateways"):
            rg[k] = []
        pfx["azureFirewall"] = None
        pfx["azureFirewalls"] = []
    if not keep_xsub:
        pfx["crossSubscriptionPeerings"] = []
    return pfx


def _overlay_node_sets(overlay):
    """Split overlay keys into the rids/names that carry a finding."""
    nic_rids = {k[4:] for k in overlay if k.startswith("nic:")}
    pip_rids = {k[4:] for k in overlay if k.startswith("pip:")}
    vnet_names = {k[5:] for k in overlay if k.startswith("vnet:")}
    return nic_rids, pip_rids, vnet_names


# ---------------------------------------------------------------- the views
# Each returns a list of (projected_fixture, level, slug, title).
def view_hld(fx, overlay):
    return [(fx, "hld", "hld", "HLD — VNets, hubs, peerings, boundary")]


def view_mld(fx, overlay):
    return [(fx, "mld", "mld", "MLD — full detail (subnets + NICs)")]


def view_risk(fx, overlay):
    nic_rids, pip_rids, vnet_finding = _overlay_node_sets(overlay)
    rg = fx.get("resourceGraph", {})
    vnets_with_risky_nic = {_nic_vnet(n) for n in rg.get("networkInterfaces", [])
                            if ov.rid(n) in nic_rids}
    keep_vnets = vnets_with_risky_nic | vnet_finding
    pfx = project(
        fx,
        keep_vnet=lambda v: v["name"] in keep_vnets,
        keep_nic=lambda n: ov.rid(n) in nic_rids,
        keep_pip=lambda p: ov.rid(p) in pip_rids,
        keep_subnets_with_nics=True, keep_boundary=True, keep_xsub=True)
    return [(pfx, "mld", "risk", "Risk view — only resources with findings")]


def view_boundary(fx, overlay):
    rg = fx.get("resourceGraph", {})
    exposed_rids = {nid[4:] for nid, e in overlay.items()
                    if nid.startswith("nic:") and any(f.get("reachable") for f in e["findings"])}
    keep_vnets = {_nic_vnet(n) for n in rg.get("networkInterfaces", [])
                  if ov.rid(n) in exposed_rids}
    pfx = project(
        fx,
        keep_vnet=lambda v: v["name"] in keep_vnets,
        keep_nic=lambda n: ov.rid(n) in exposed_rids,
        keep_pip=lambda p: True,          # public IPs ARE the boundary
        keep_subnets_with_nics=True, keep_boundary=True, keep_xsub=False)
    return [(pfx, "mld", "boundary", "External boundary — internet-facing paths")]


def view_cross_sub(fx, overlay):
    names = set()
    for xp in fx.get("crossSubscriptionPeerings", []):
        names.add(xp.get("localVnet"))
        names.add(xp.get("remoteVnet"))
    names.discard(None)
    if not names:
        return []  # no cross-sub peering in this estate — skip the view
    pfx = project(
        fx,
        keep_vnet=lambda v: v["name"] in names,
        keep_nic=lambda n: False,
        keep_pip=lambda p: False,
        keep_subnets_with_nics=True, keep_boundary=False, keep_xsub=True)
    return [(pfx, "hld", "cross-sub", "Cross-subscription peering — multi-sub blast radius")]


def views_finding_centric(fx, overlay, k=1):
    """One small k-hop diagram per Critical/High finding around the affected node."""
    rg = fx.get("resourceGraph", {})
    nic_by_rid = {ov.rid(n): n for n in rg.get("networkInterfaces", [])}
    adj = _peer_adjacency(fx)
    out, seen = [], set()
    for nid, e in sorted(overlay.items()):
        if not nid.startswith("nic:"):
            continue
        rid = nid[4:]
        nic = nic_by_rid.get(rid)
        if not nic:
            continue
        for f in e["findings"]:
            if f["severity"] not in ("Critical", "High") or f["resource"] in seen:
                continue
            seen.add(f["resource"])
            center = _nic_vnet(nic)
            khop, frontier = {center}, {center}
            for _ in range(k):
                nxt = set()
                for vn in frontier:
                    nxt |= adj.get(vn, set())
                khop |= nxt
                frontier = nxt
            pfx = project(
                fx,
                keep_vnet=lambda v, kh=khop: v["name"] in kh,
                keep_nic=lambda n, c=rid: ov.rid(n) == c,
                keep_pip=lambda p: False,
                keep_subnets_with_nics=True, keep_boundary=True, keep_xsub=True)
            slug = "finding-" + _slug(f["type"] + "-" + f["resource"])
            title = "Finding — %s on %s (%s)" % (f["type"], f["resource"], f["severity"])
            out.append((pfx, "mld", slug, title))
    return out


VIEW_FUNCS = [view_hld, view_mld, view_risk, view_boundary, view_cross_sub]


# ---------------------------------------------------------------- driver
def _slug(s):
    return re.sub(r"-+", "-", re.sub(r"[^a-z0-9]+", "-", s.lower())).strip("-")[:80]


def generate_views(fx, k=1):
    """Render every view from ONE whole-estate overlay. Returns a list of dicts:
    {slug, title, level, xml, vertices, edges}. Deterministic."""
    overlay = ov.compute_overlay(fx)
    specs = []
    for fn in VIEW_FUNCS:
        specs += fn(fx, overlay)
    specs += views_finding_centric(fx, overlay, k=k)

    results = []
    for pfx, level, slug, title in specs:
        xml, cells, edges = rd.render(pfx, level, overlay=overlay, title=title)
        results.append({"slug": slug, "title": title, "level": level,
                        "xml": xml, "vertices": len(cells), "edges": len(edges)})
    return results


def main():
    args = sys.argv[1:]
    if not args:
        sys.exit("usage: python3 views.py <fixture.json> [--out-dir DIR] [--k N]")
    fixture = args[0]
    out_dir = args[args.index("--out-dir") + 1] if "--out-dir" in args else "phase-4/out/views"
    k = int(args[args.index("--k") + 1]) if "--k" in args else 1
    fx = json.load(open(fixture, encoding="utf-8"))
    os.makedirs(out_dir, exist_ok=True)
    base = os.path.splitext(os.path.basename(fixture))[0]
    results = generate_views(fx, k=k)
    for r in results:
        path = os.path.join(out_dir, "%s.%s.drawio" % (base, r["slug"]))
        with open(path, "w", encoding="utf-8") as fh:
            fh.write(r["xml"])
        print("wrote %-60s %2d vertices %2d edges  (%s)" %
              (path, r["vertices"], r["edges"], r["title"]))
    print("%d views generated for %s" % (len(results), base))


if __name__ == "__main__":
    main()

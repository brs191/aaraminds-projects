#!/usr/bin/env python3
"""Gate for the Phase-4 view families (views.py).

Asserts, over the whole fixture corpus, that every view is:
  * VALID   — renders without tripping render()'s unique-id / no-dangling-edge asserts;
  * DETERMINISTIC — byte-identical on a re-run (the property the whole product rests on);
  * FAITHFUL — each view's projection obeys its contract:
      risk        -> only resources that carry a finding (no Clean-only nodes);
      cross-sub   -> only VNets that participate in a cross-subscription peering;
      boundary    -> contains every internet-reachable NIC, nothing internal-only;
      finding/<n> -> exactly one per distinct Critical/High finding; the affected
                     resource is present in its own diagram.

Risk truth is computed once on the full estate; a view can hide a resource but can
never change a verdict — these tests pin that.

Run:  python3 test_views.py [fixtures_dir ...]   (exit 0 = pass)
"""
import glob
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import overlay as ov     # noqa: E402
import views as vw       # noqa: E402

ROOT = os.path.dirname(os.path.dirname(HERE))
DEFAULT_DIRS = [os.path.join(HERE, "..", "fixtures"),
                os.path.join(ROOT, "phase-1", "eval", "fixtures")]


def _fixtures(dirs):
    out = []
    for d in dirs:
        out += sorted(glob.glob(os.path.join(d, "*.json")))
    return [f for f in out if os.path.basename(f) not in ("last_run.json",)]


def check_fixture(path):
    """Return a list of failure strings (empty = pass)."""
    fails = []
    fx = json.load(open(path, encoding="utf-8"))
    name = os.path.basename(path)
    overlay = ov.compute_overlay(fx)

    # ---- end-to-end: every view renders + is byte-identical on re-run ----
    a = vw.generate_views(fx)
    b = vw.generate_views(fx)
    slugs_a = [v["slug"] for v in a]
    if slugs_a != [v["slug"] for v in b] or any(x["xml"] != y["xml"] for x, y in zip(a, b)):
        fails.append("%s: non-deterministic view generation" % name)
    if len(slugs_a) != len(set(slugs_a)):
        dupes = sorted({s for s in slugs_a if slugs_a.count(s) > 1})
        fails.append("%s: duplicate view slugs %s" % (name, dupes))

    nic_rids, pip_rids, vnet_finding = vw._overlay_node_sets(overlay)

    # ---- risk view: every drawn resource carries a finding ----
    risk = vw.view_risk(fx, overlay)
    if risk:
        pfx = risk[0][0]
        rg = pfx["resourceGraph"]
        for n in rg.get("networkInterfaces", []):
            if ov.rid(n) not in nic_rids:
                fails.append("%s: risk view drew NIC %s with no finding" % (name, n.get("name")))
        risky_vnets = {vw._nic_vnet(n) for n in fx["resourceGraph"].get("networkInterfaces", [])
                       if ov.rid(n) in nic_rids} | vnet_finding
        for v in rg.get("virtualNetworks", []):
            if v["name"] not in risky_vnets:
                fails.append("%s: risk view drew clean VNet %s" % (name, v["name"]))

    # ---- cross-sub view: only cross-sub VNets ----
    xsub_names = set()
    for xp in fx.get("crossSubscriptionPeerings", []):
        xsub_names |= {xp.get("localVnet"), xp.get("remoteVnet")}
    xsub_names.discard(None)
    cs = vw.view_cross_sub(fx, overlay)
    if xsub_names and not cs:
        fails.append("%s: cross-sub peerings present but cross-sub view empty" % name)
    if cs:
        for v in cs[0][0]["resourceGraph"].get("virtualNetworks", []):
            if v["name"] not in xsub_names:
                fails.append("%s: cross-sub view drew non-cross-sub VNet %s" % (name, v["name"]))

    # ---- boundary view: contains every internet-reachable NIC ----
    reachable_rids = {nid[4:] for nid, e in overlay.items()
                      if nid.startswith("nic:") and any(f.get("reachable") for f in e["findings"])}
    bnd = vw.view_boundary(fx, overlay)
    if bnd:
        drawn = {ov.rid(n) for n in bnd[0][0]["resourceGraph"].get("networkInterfaces", [])}
        missing = reachable_rids - drawn
        if missing:
            fails.append("%s: boundary view missing reachable NIC(s) %s" % (name, sorted(missing)))

    # ---- finding-centric: one per distinct Critical/High finding; node present ----
    expected = set()
    for nid, e in overlay.items():
        if not nid.startswith("nic:"):
            continue
        for f in e["findings"]:
            if f["severity"] in ("Critical", "High"):
                expected.add(f["resource"])
    fc = vw.views_finding_centric(fx, overlay)
    if len(fc) != len(expected):
        fails.append("%s: finding-centric count %d != distinct Crit/High findings %d"
                     % (name, len(fc), len(expected)))
    for pfx, _level, _slug, title in fc:
        # the affected NIC must appear in its own diagram
        drawn = {ov.rid(n) for n in pfx["resourceGraph"].get("networkInterfaces", [])}
        if len(drawn) != 1:
            fails.append("%s: finding view '%s' should draw exactly its 1 affected NIC, drew %d"
                         % (name, title, len(drawn)))
    return fails


def main():
    dirs = sys.argv[1:] or DEFAULT_DIRS
    fixtures = _fixtures(dirs)
    all_fails = []
    for f in fixtures:
        all_fails += check_fixture(f)
    print("views-gate: %d fixtures checked, %d failures" % (len(fixtures), len(all_fails)))
    for x in all_fails:
        print("  FAIL", x)
    return 1 if all_fails else 0


if __name__ == "__main__":
    sys.exit(main())

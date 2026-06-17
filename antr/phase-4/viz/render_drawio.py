#!/usr/bin/env python3
"""Phase-4 draw.io renderer — the map, with the risk painted on.

Emits a valid (uncompressed) draw.io mxGraph diagram from a graph.Fixture:
  - VNet containers grouped by subscription, hub-first layout
  - subnets + NICs (MLD), NICs painted by overlay severity
  - external-boundary nodes (Internet, Firewall, VPN/ER gateway, NAT GW, public
    IPs) in a GREY structural palette disjoint from the severity palette (RC-3, M-3)
  - local AND cross-subscription peering edges; out-of-scope peer -> EXTERNAL STUB
    node, never a dropped edge (RC-1/RC-2)
  - CIDR-overlap findings drawn as a dashed edge AND folded into VNet rollup (H-1)
  - internet -> exposed-NIC edges (the exposure path)
  - node fill read ONLY from overlay.compute_overlay() (RC-4)
  - cell ids are kind-namespaced and asserted globally unique (C-1, H-2)

Run:
    python3 render_drawio.py <fixture.json> --out <file.drawio> --level {hld,mld}
"""
import json
import os
import sys
from xml.sax.saxutils import quoteattr

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import overlay as ov  # noqa: E402

VNET_W = 260
SUB_W = 220
NODE_H = 26
GAP_X = 120
BOUNDARY_Y = 40
VNET_Y = 220


def sev_style(severity, base="rounded=1;whiteSpace=wrap;html=1;"):
    s = ov.style_for(severity)
    return f"{base}fillColor={s['fill']};strokeColor={s['stroke']};"


def struct_style(base="rounded=1;whiteSpace=wrap;html=1;"):
    return f"{base}fillColor={ov.STRUCT_FILL};strokeColor={ov.STRUCT_STROKE};"


def nic_vnet(nic):
    s = nic.get("subnet", "")
    return s.split("/")[0] if "/" in s else ""


class Cell:
    __slots__ = ("id", "value", "style", "parent", "x", "y", "w", "h", "vertex", "source", "target")

    def __init__(self, id, value="", style="", parent="1", x=0, y=0, w=0, h=0,
                 vertex=True, source=None, target=None):
        self.id, self.value, self.style, self.parent = id, value, style, parent
        self.x, self.y, self.w, self.h = x, y, w, h
        self.vertex, self.source, self.target = vertex, source, target


def build(fx, level="mld", overlay=None):
    rg = fx.get("resourceGraph", {})
    vnets = rg.get("virtualNetworks", [])
    nics = rg.get("networkInterfaces", [])
    pips = rg.get("publicIPAddresses", [])
    gws = rg.get("virtualNetworkGateways", [])
    ercs = rg.get("expressRouteCircuits", [])
    natgws = rg.get("natGateways", [])
    fw = fx.get("azureFirewall")
    # Risk truth is computed by Analyze() on the WHOLE estate. A "view" (views.py)
    # is a PROJECTION of the fixture for layout only; it passes the full-estate
    # overlay here so node colours reflect the complete analysis, never a partial
    # re-analysis of the projected subset. None => standalone render (compute here).
    if overlay is None:
        overlay = ov.compute_overlay(fx)

    vnet_names = {v["name"] for v in vnets}
    nics_by_vnet, nics_by_subnet = {}, {}
    for n in nics:
        nics_by_vnet.setdefault(nic_vnet(n), []).append(n)
        nics_by_subnet.setdefault(n.get("subnet", ""), []).append(n)

    cells, edges, vertex_ids = [], [], set()

    def add(c):
        cells.append(c)
        if c.vertex:
            vertex_ids.add(c.id)
        return c

    # Edge ids via a deterministic counter — never built from names (which can
    # contain the '--' delimiter and collide; second-audit H-2). Edge ids are
    # cosmetic (not join keys); the semantic pair lives in the label.
    eseq = [0]

    def eid(prefix):
        eseq[0] += 1
        return "%s%d" % (prefix, eseq[0])

    def vnet_rollup(vn):
        # max over contained NIC severities AND any CIDR-overlap severity on the VNet (H-1)
        sev = ov.severity_of(overlay, "vnet:" + vn)
        for n in nics_by_vnet.get(vn, []):
            s = ov.severity_of(overlay, "nic:" + ov.rid(n))
            if ov.SEV_RANK.get(s, 0) > ov.SEV_RANK.get(sev, 0):
                sev = s
        return sev

    ordered = sorted(vnets, key=lambda v: len(v.get("peerings", [])), reverse=True)
    hub_name = ordered[0]["name"] if ordered else None

    # ---- boundary band (grey structural palette) ----
    bx = 40
    internet_id = None
    if any(n.get("publicIp") for n in nics) or fw:
        internet_id = "internet"
        add(Cell(internet_id, "\U0001F310 Internet",
                 "ellipse;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;" % (ov.STRUCT_FILL, ov.STRUCT_STROKE),
                 x=bx, y=BOUNDARY_Y, w=120, h=60))
        bx += 120 + GAP_X
    if fw:
        add(Cell("fw:" + fw["name"], "\U0001F525 Firewall\n" + fw["name"], struct_style(),
                 x=bx, y=BOUNDARY_Y, w=140, h=60)); bx += 140 + GAP_X
    for g in gws:
        label = ("\U0001F517 ER GW\n" if g.get("gatewayType") == "ExpressRoute" else "\U0001F517 VPN GW\n") + g["name"]
        add(Cell("gw:" + g["name"], label, struct_style(), x=bx, y=BOUNDARY_Y, w=140, h=60)); bx += 140 + GAP_X
    for e in ercs:
        add(Cell("er:" + e["name"], "ExpressRoute\n" + e["name"],
                 "shape=cloud;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;" % (ov.STRUCT_FILL, ov.STRUCT_STROKE),
                 x=bx, y=BOUNDARY_Y, w=150, h=60)); bx += 150 + GAP_X
    for ng in natgws:
        add(Cell("natgw:" + ng["name"], "\U0001F517 NAT GW\n" + ng["name"], struct_style(),
                 x=bx, y=BOUNDARY_Y, w=130, h=60)); bx += 130 + GAP_X
    # orphaned public IPs are findings -> painted by severity
    for p in pips:
        if not p.get("ipConfiguration"):
            nid = "pip:" + ov.rid(p)
            sev = ov.severity_of(overlay, nid)
            badge = ov.style_for(sev).get("badge", "")
            add(Cell(nid, "%s Public IP\n%s\n%s" % (badge, p["name"], p.get("ipAddress", "")),
                     sev_style(sev), x=bx, y=BOUNDARY_Y, w=150, h=70)); bx += 150 + GAP_X

    # ---- VNet row (hub first) ----
    def vnet_height(v):
        # must fully CONTAIN its subnets: 70px header + each subnet (sh + 10 gap)
        # + 10px bottom pad. (second audit: subnets were overflowing the box.)
        if level != "mld":
            return 90
        h = 70
        for sn in v.get("subnets", []):
            sk = v["name"] + "/" + sn["name"]
            sh = 26 + len(nics_by_subnet.get(sk, [])) * (NODE_H + 6) + 10
            h += sh + 10
        return max(h + 10, 90)

    vx = 40
    row_y = VNET_Y
    max_bottom = row_y
    for v in ordered:
        vn = v["name"]
        vh = vnet_height(v)
        max_bottom = max(max_bottom, row_y + vh)
        sev = vnet_rollup(vn)
        header = "%s\n[%s]\n%s" % (vn, v.get("subscriptionId", ""), ", ".join(v.get("addressSpace", [])))
        vid = "vnet:" + vn
        add(Cell(vid, header,
                 sev_style(sev, "rounded=0;whiteSpace=wrap;html=1;verticalAlign=top;") + "fontStyle=1;",
                 x=vx, y=row_y, w=VNET_W, h=vh))
        if level == "mld":
            cy = 70
            for sn in v.get("subnets", []):
                sk = vn + "/" + sn["name"]
                snics = nics_by_subnet.get(sk, [])
                sh = 26 + len(snics) * (NODE_H + 6) + 10
                sid = "subnet:" + sk
                add(Cell(sid, "%s  (%s)" % (sn["name"], sn.get("addressPrefix", "")),
                         "rounded=0;whiteSpace=wrap;html=1;verticalAlign=top;fillColor=%s;strokeColor=%s;" %
                         (ov.SUBNET_FILL, ov.SUBNET_STROKE),
                         parent=vid, x=20, y=cy, w=SUB_W, h=sh))
                ny = 26
                for n in snics:
                    nid = "nic:" + ov.rid(n)
                    nsev = ov.severity_of(overlay, nid)
                    badge = ov.style_for(nsev).get("badge", "")
                    label = "%s %s" % (badge, n["name"])
                    if n.get("publicIp"):
                        label += "\n" + str(n.get("publicIp"))
                    add(Cell(nid, label, sev_style(nsev), parent=sid, x=10, y=ny, w=SUB_W - 20, h=NODE_H))
                    ny += NODE_H + 6
                cy += sh + 10
        vx += VNET_W + GAP_X

    # ---- external stub nodes for out-of-scope peers (placed below the VNet row, M-1) ----
    stub_ids = {}
    stub_y = max_bottom + 60

    def stub(remote):
        if remote in stub_ids:
            return stub_ids[remote]
        sid = "ext:" + remote
        add(Cell(sid, "↗ %s\n(outside scope)" % remote,
                 "rounded=1;whiteSpace=wrap;html=1;dashed=1;fillColor=%s;strokeColor=%s;" % (ov.STRUCT_FILL, ov.STRUCT_STROKE),
                 x=40 + len(stub_ids) * (180 + 40), y=stub_y, w=180, h=60))
        stub_ids[remote] = sid
        return sid

    # ---- peering edges (local + cross-sub), deduped, defensive ----
    seen = set()

    def peer_edge(local, remote, dashed=False, label=""):
        if not local or not remote or local == remote:  # skip self-peering (second audit)
            return
        key = tuple(sorted((local, remote)))
        if key in seen:
            return
        seen.add(key)
        src = "vnet:" + local if local in vnet_names else stub(local)
        tgt = "vnet:" + remote if remote in vnet_names else stub(remote)
        style = "edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;endArrow=none;"
        if dashed:
            style += "dashed=1;"
        edges.append(Cell(eid("peer:"), label, style, vertex=False, source=src, target=tgt))

    for v in vnets:
        for p in v.get("peerings", []):
            peer_edge(v["name"], p.get("remoteVnet"),
                      dashed=not p.get("allowForwardedTraffic", False), label=p.get("state", ""))
    for xp in fx.get("crossSubscriptionPeerings", []):
        peer_edge(xp.get("localVnet"), xp.get("remoteVnet"),
                  dashed=not xp.get("hasHubFirewall", False), label="x-sub")

    # ---- CIDR-overlap edges (H-1): make the Medium finding visible on the map ----
    cidr_seen = set()
    for entry in overlay.values():
        for f in entry["findings"]:
            if f.get("type") == "CIDR overlap":
                a, b = (f["resource"].split("~", 1) + [""])[:2]
                key = tuple(sorted((a, b)))
                if key in cidr_seen or a not in vnet_names or b not in vnet_names:
                    continue
                cidr_seen.add(key)
                col = ov.style_for(f["severity"])["stroke"]
                edges.append(Cell(eid("cidr:"), "CIDR overlap",
                                  "edgeStyle=orthogonalEdgeStyle;html=1;dashed=1;endArrow=none;strokeColor=%s;" % col,
                                  vertex=False, source="vnet:" + a, target="vnet:" + b))

    # ---- internet -> exposed NIC/VNet edges (the exposure path) ----
    if internet_id:
        for n in nics:
            f_reach = [f for f in overlay.get("nic:" + n["name"], {}).get("findings", [])
                       if f.get("reachable") and "reachable" in f.get("type", "")]
            if f_reach:
                tgt = ("nic:" + ov.rid(n)) if level == "mld" else ("vnet:" + nic_vnet(n))
                if tgt in vertex_ids:
                    edges.append(Cell(eid("exp:"), "exposed",
                                      "edgeStyle=orthogonalEdgeStyle;html=1;strokeColor=#b85450;endArrow=block;",
                                      vertex=False, source=internet_id, target=tgt))

    # ---- gateway/firewall -> hub edges ----
    if hub_name:
        for g in gws:
            edges.append(Cell(eid("gwlink:"), "", "html=1;endArrow=none;",
                              vertex=False, source="gw:" + g["name"], target="vnet:" + hub_name))
        if fw:
            edges.append(Cell(eid("fwlink:"), "", "html=1;endArrow=none;",
                              vertex=False, source="fw:" + fw["name"], target="vnet:" + hub_name))

    # ---- application & edge-services band ----
    # Every app-layer family the engine scores (App Gateway, AKS, Private Endpoint,
    # APIM, Front Door, vWAN hub) is drawn here as a node painted by overlay
    # severity, so a WAF-disabled gateway / public AKS / unlinked private endpoint
    # is VISIBLE on the map — not hidden in a side-channel. Subnet-attached families
    # link (dashed) to their VNet; internet-edge families link to the Internet
    # boundary (the ingress path). Bastion is drawn structurally (its bypass finding
    # lands on the offending NIC, not the Bastion). Cross-sub peering has no node —
    # it is the cross-sub edge already drawn above.
    app_items = []
    for gw in rg.get("applicationGateways", []):
        app_items.append(("appgw", gw["name"], "App Gateway\n" + gw["name"], gw.get("subnet"), False))
    for a in rg.get("aksClusters", []):
        app_items.append(("aks", a["name"], "AKS\n" + a["name"], a.get("subnet"), False))
    for pe in rg.get("privateEndpoints", []):
        app_items.append(("pe", pe["name"], "Private Endpoint\n" + pe["name"], pe.get("subnet"), False))
    for b in rg.get("azureBastions", []):
        app_items.append(("bastion", b["name"], "Bastion\n" + b["name"], b.get("subnet"), False))
    for lb in rg.get("loadBalancers", []):
        pub = not lb.get("isInternal", False)
        # Structural (the LB-NAT exposure lands on the backend NIC). Public LB → an
        # internet ingress; internal LB → just a node (no subnet in the model).
        app_items.append(("lb", lb["name"], ("Public LB\n" if pub else "Internal LB\n") + lb["name"], None, pub))
    for ap in rg.get("apiManagements", []):
        app_items.append(("apim", ap["name"], "APIM\n" + ap["name"], None, True))
    for fd in rg.get("azureFrontDoors", []):
        app_items.append(("fd", fd["name"], "Front Door\n" + fd["name"], None, True))
    for wan in rg.get("virtualWans", []):
        for hub in wan.get("vHubs", []):
            app_items.append(("vhub", hub["name"], "vWAN Hub\n" + hub["name"], None, True))

    if app_items:
        # Clear the external-stub band (stub height 60) when stubs exist, else clear
        # the VNet row — leaving room for the band label above the nodes (RC-5).
        app_y = (stub_y + 140) if stub_ids else (max_bottom + 90)
        add(Cell("applayer-label", "Application & edge services",
                 "text;html=1;align=left;verticalAlign=middle;fontStyle=1;", x=40, y=app_y - 26, w=320, h=20))
        ax = 40
        for kind, name, label, subnet, is_edge in app_items:
            nid = kind + ":" + name
            if kind in ("bastion", "lb"):
                # Never scored directly (Bastion-bypass and LB-NAT findings land on
                # the offending NIC); draw structurally (grey), like the boundary nodes.
                style, badge = struct_style(), "⬜"
            else:
                # Scored families: severity fill even when Clean (green), consistent
                # with NIC/VNet — the band shows the app inventory coloured by risk.
                sev = ov.severity_of(overlay, nid)
                style, badge = sev_style(sev), ov.style_for(sev).get("badge", "")
            add(Cell(nid, "%s %s" % (badge, label), style, x=ax, y=app_y, w=170, h=54))
            if is_edge and internet_id:
                edges.append(Cell(eid("appedge:"), "",
                                  "edgeStyle=orthogonalEdgeStyle;html=1;endArrow=block;strokeColor=%s;" % ov.STRUCT_STROKE,
                                  vertex=False, source=internet_id, target=nid))
            elif subnet:
                vn = subnet.split("/")[0] if "/" in subnet else subnet
                if "vnet:" + vn in vertex_ids:
                    edges.append(Cell(eid("appedge:"), "",
                                      "edgeStyle=orthogonalEdgeStyle;html=1;endArrow=none;dashed=1;strokeColor=%s;" % ov.STRUCT_STROKE,
                                      vertex=False, source=nid, target="vnet:" + vn))
            ax += 170 + 30

    # ---- legend (clear of boundary band + VNet row) ----
    leg = "<b>Legend — severity</b><br>" + "<br>".join(
        "%s %s" % (ov.BUCKET_COLOR[b]["badge"], b) for b in ov.LEGEND_ORDER) + \
        "<br><br><b>Structural</b><br>⬜ grey = infrastructure (unscored)"
    legend_x = max(vx, bx) + 40
    add(Cell("legend", leg, "rounded=0;whiteSpace=wrap;html=1;align=left;fillColor=#ffffff;strokeColor=#000000;",
             x=legend_x, y=BOUNDARY_Y, w=200, h=170))

    return cells, edges, vertex_ids


def to_xml(cells, edges, title="Azure Network Topology"):
    out = ['<mxfile host="app.diagrams.net">',
           '  <diagram name=%s id="phase4-topo">' % quoteattr(title),
           '    <mxGraphModel dx="1422" dy="800" grid="0" gridSize="10" guides="1" '
           'tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" '
           'pageWidth="1654" pageHeight="1169" math="0" shadow="0">',
           '      <root>', '        <mxCell id="0" />', '        <mxCell id="1" parent="0" />']
    for c in cells:
        val = c.value.replace("\n", "<br>")  # html=1 cells need <br> for line breaks
        out.append('        <mxCell id=%s value=%s style=%s vertex="1" parent=%s>' %
                   (quoteattr(c.id), quoteattr(val), quoteattr(c.style), quoteattr(c.parent)))
        out.append('          <mxGeometry x="%d" y="%d" width="%d" height="%d" as="geometry" />' %
                   (c.x, c.y, c.w, c.h))
        out.append('        </mxCell>')
    for e in edges:
        out.append('        <mxCell id=%s value=%s style=%s edge="1" parent="1" source=%s target=%s>' %
                   (quoteattr(e.id), quoteattr(e.value), quoteattr(e.style),
                    quoteattr(e.source), quoteattr(e.target)))
        out.append('          <mxGeometry relative="1" as="geometry" />')
        out.append('        </mxCell>')
    out += ['      </root>', '    </mxGraphModel>', '  </diagram>', '</mxfile>', '']
    return "\n".join(out)


def render(fx, level="mld", overlay=None, title=None):
    cells, edges, vertex_ids = build(fx, level, overlay)
    # invariant 1: ALL cell ids globally unique across vertices AND edges
    # (H-2 — draw.io corrupts on duplicate ids; the second audit showed edges
    #  were previously unguarded).
    ids = [c.id for c in cells] + [e.id for e in edges]
    dups = sorted({i for i in ids if ids.count(i) > 1})
    assert not dups, "duplicate cell ids (collision): %s" % dups
    # invariant 2: no dangling edges (RC-1/RC-2)
    for e in edges:
        assert e.source in vertex_ids, "dangling edge source %s" % e.source
        assert e.target in vertex_ids, "dangling edge target %s" % e.target
    if title is None:
        title = "Azure Network Topology (%s)" % level.upper()
    return to_xml(cells, edges, title), cells, edges


def main():
    args = sys.argv[1:]
    if not args:
        sys.exit("usage: python3 render_drawio.py <fixture.json> --out <file> --level {hld,mld}")
    fixture = args[0]
    out = args[args.index("--out") + 1] if "--out" in args else "topology.drawio"
    level = args[args.index("--level") + 1] if "--level" in args else "mld"
    fx = json.load(open(fixture, encoding="utf-8"))
    xml, cells, edges = render(fx, level)
    os.makedirs(os.path.dirname(out) or ".", exist_ok=True)
    with open(out, "w", encoding="utf-8") as fh:
        fh.write(xml)
    print("wrote %s: %d vertices, %d edges, level=%s" % (out, len(cells), len(edges), level))


if __name__ == "__main__":
    main()

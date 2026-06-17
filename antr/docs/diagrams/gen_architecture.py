#!/usr/bin/env python3
"""Generate the antr architecture diagram in the AaraMinds brand style.

Brand tokens lifted from the LinkedIn-newsletter assets:
  font   Inter / Segoe UI / Arial
  navy   #071B4D   teal #007D8F (grad #00A6B4->#006E86)   purple #5A33B5 (grad #7F4BDB->#4A29A5)
  tints  teal #F1F7F9   purple #F3EFFB   soft #F6FAFC
  white rounded cards + soft drop shadow on white; numbered gradient circles;
  navy header bars; foundation-layers band; mountain-A footer brand.

Run:  python3 gen_architecture.py  ->  architecture.svg
"""
import html
import os

W, H = 1536, 1024
NAVY = "#071B4D"
TEAL = "#007D8F"
PURPLE = "#5A33B5"
GREY = "#5D6472"
TEAL_TINT = "#F1F7F9"
PURPLE_TINT = "#F3EFFB"
BORDER = "#C9D8E6"
BORDER2 = "#DCE8F0"
DIV = "#D7E5EF"
FONT = "Inter, Segoe UI, Arial, sans-serif"

S = []


def esc(t):
    return html.escape(str(t), quote=True)


def text(x, y, t, size, weight, fill, anchor="start", spacing=None):
    parts = t if isinstance(t, list) else [t]
    tspans = ""
    for i, line in enumerate(parts):
        dy = 0 if i == 0 else (spacing or size * 1.25)
        tspans += '<tspan x="%s" dy="%s">%s</tspan>' % (x, dy, esc(line))
    S.append('<text x="%s" y="%s" font-family="%s" font-size="%s" font-weight="%s" '
             'fill="%s" text-anchor="%s">%s</text>'
             % (x, y, FONT, size, weight, fill, anchor, tspans))


def rect(x, y, w, h, rx, fill, stroke=None, sw=1.2, shadow=False, opacity=None):
    s = '<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s"' % (x, y, w, h, rx, fill)
    if stroke:
        s += ' stroke="%s" stroke-width="%s"' % (stroke, sw)
    if opacity is not None:
        s += ' opacity="%s"' % opacity
    if shadow:
        s += ' filter="url(#softShadow)"'
    s += ' />'
    S.append(s)


def line(x1, y1, x2, y2, stroke, sw=1):
    S.append('<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-width="%s"/>'
             % (x1, y1, x2, y2, stroke, sw))


def num_circle(cx, cy, n, grad):
    S.append('<circle cx="%s" cy="%s" r="26" fill="url(#%s)"/>' % (cx, cy, grad))
    text(cx, cy + 9, str(n), 30, 700, "white", "middle")


def chip(x, y, w, h, bold, rest, fill=TEAL_TINT, bold_color=TEAL, border=BORDER2, rest_color=NAVY, bsize=15):
    rect(x, y, w, h, 8, fill, border, 1)
    ty = y + h / 2 + 5
    if bold:
        text(x + 16, ty, bold, bsize, 800, bold_color)
        bx = x + 16 + len(bold) * (bsize * 0.62) + 10
        if rest:
            text(bx, ty, rest, bsize, 500, rest_color)
    else:
        text(x + 16, ty, rest, bsize, 500, rest_color)


def arrow_down(cx, y1, y2, color=TEAL):
    S.append('<path d="M %s %s V %s" stroke="%s" stroke-width="3" stroke-linecap="round"/>'
             % (cx, y1, y2 - 6, color))
    S.append('<path d="M %s %s l -5 -7 h 10 z" fill="%s"/>' % (cx, y2, color))


def arrow_right(x1, x2, y, color=NAVY):
    S.append('<path d="M %s %s H %s" stroke="%s" stroke-width="3" stroke-linecap="round"/>'
             % (x1, y, x2 - 7, color))
    S.append('<path d="M %s %s l -8 -5 v 10 z" fill="%s"/>' % (x2, y, color))


# ---------------------------------------------------------------- canvas
S.append('<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">' % (W, H, W, H))
S.append('<defs>'
         '<linearGradient id="gradTeal" x1="0" x2="1" y1="0" y2="1">'
         '<stop offset="0" stop-color="#00A6B4"/><stop offset="1" stop-color="#006E86"/></linearGradient>'
         '<linearGradient id="gradPurple" x1="0" x2="1" y1="0" y2="1">'
         '<stop offset="0" stop-color="#7F4BDB"/><stop offset="1" stop-color="#4A29A5"/></linearGradient>'
         '<filter id="softShadow" x="-10%" y="-10%" width="120%" height="130%">'
         '<feDropShadow dx="0" dy="6" stdDeviation="9" flood-color="#0B153B" flood-opacity="0.08"/></filter>'
         '</defs>')
rect(0, 0, W, H, 0, "#FFFFFF")
S.append('<circle cx="1190" cy="470" r="470" fill="#F6FAFC" opacity="0.6"/>')

# ---------------------------------------------------------------- header
rect(26, 17, 210, 34, 7, NAVY, NAVY, 1.2)
text(131, 40, "PROJECT ARCHITECTURE", 14, 700, "white", "middle")
text(26, 102, "Azure Network Topology Reviewer", 54, 800, NAVY)
text(26, 144, "Deterministic Azure reachability & exposure  —  adopt the map, own the risk.", 25, 600, GREY)

# top-right callout
rect(1140, 24, 370, 104, 10, "white", BORDER, 1.2, shadow=True)
# small target icon
S.append('<circle cx="1188" cy="76" r="26" fill="none" stroke="%s" stroke-width="3"/>' % TEAL)
S.append('<circle cx="1188" cy="76" r="13" fill="none" stroke="%s" stroke-width="3"/>' % TEAL)
S.append('<circle cx="1188" cy="76" r="3.5" fill="%s"/>' % TEAL)
text(1228, 70, "Discovery feeds the map.", 20, 800, NAVY)
text(1228, 102, "The engine owns the verdict.", 20, 800, TEAL)

# ---------------------------------------------------------------- main card
CARD_X, CARD_Y, CARD_W = 26, 176, 1484
CARD_BOTTOM = 720
rect(CARD_X, CARD_Y, CARD_W, CARD_BOTTOM - CARD_Y, 9, "white", BORDER, 1.2, shadow=True)

# column bounds
cax0, cax1 = 26, 520
cbx0, cbx1 = 520, 1000
ccx0, ccx1 = 1000, 1510
BAR_H = 58

# centre column emphasis tint
rect(cbx0, CARD_Y + BAR_H, cbx1 - cbx0, CARD_BOTTOM - (CARD_Y + BAR_H), 0, TEAL_TINT, opacity=0.6)

# header bars
def bar(x0, x1, title, sub, rx_left, rx_right):
    rect(x0, CARD_Y, x1 - x0, BAR_H, 9, NAVY)
    if not rx_left:
        rect(x0, CARD_Y, 14, BAR_H, 0, NAVY)
    if not rx_right:
        rect(x1 - 14, CARD_Y, 14, BAR_H, 0, NAVY)
    cx = (x0 + x1) / 2
    text(cx, CARD_Y + 27, title, 19, 800, "white", "middle")
    text(cx, CARD_Y + 47, sub, 14, 500, "#AFCAD8", "middle")


bar(cax0, cax1, "DISCOVER", "Azure  →  Graph IR", True, False)
bar(cbx0, cbx1, "ANALYZE", "the deterministic core", False, False)
bar(ccx0, ccx1, "DELIVER", "one IR  →  three products", False, True)
line(cbx0, CARD_Y, cbx0, CARD_BOTTOM, DIV, 1)
line(cbx1, CARD_Y, cbx1, CARD_BOTTOM, DIV, 1)

# ---- Column A : DISCOVER ----
num_circle(cax0 + 34 + 16, 296, 1, "gradTeal")
text(cax0 + 90, 290, "Azure Adapter", 24, 800, TEAL)
text(cax0 + 90, 318, "read-only discovery", 16, 500, NAVY)
ax = cax0 + 24
aw = (cax1 - cax0) - 48
chip(ax, 348, aw, 50, "ARG", "paginated KQL · VNets, NSGs, NICs, routes, peerings")
chip(ax, 408, aw, 50, "Network Watcher", "effective rules + effective routes")
chip(ax, 468, aw, 50, "+ context", "AVNM admin · Firewall · App GW/AKS/FD/APIM/vWAN")
chip(ax, 528, aw, 50, "Auth", "Managed Identity / OIDC · Reader · never a secret",
     fill="white", bold_color=NAVY, border=BORDER)
chip(ax, 600, aw, 52, "Output", "graph.Fixture  —  the Graph IR contract",
     fill="white", bold_color=TEAL, border=TEAL)

# ---- Column B : ANALYZE ----
num_circle(cbx0 + 34 + 16, 296, 2, "gradTeal")
text(cbx0 + 90, 290, "Risk Engine", 24, 800, TEAL)
text(cbx0 + 90, 318, "Analyze() · same input → same output", 15, 500, NAVY)
bx = cbx0 + 24
bw = (cbx1 - cbx0) - 48
gates = [("Gate 1", "AVNM admin verdict"),
         ("Gate 2", "NSG effective rules"),
         ("Gate 3", "effective routes (None = black-hole)"),
         ("Gate 4", "public IP / DNAT / LB NAT")]
gy = 352
for i, (g, d) in enumerate(gates):
    chip(bx, gy, bw, 38, g, d, bsize=14)
    if i < len(gates):
        arrow_down(bx + bw / 2, gy + 38, gy + 50)
    gy += 50
rect(bx, gy, bw, 38, 8, NAVY)
text(bx + bw / 2, gy + 24, "REACHABLE  →  severity", 15, 800, "white", "middle")
gy += 54
chip(bx, gy, bw, 40, "Twin", "Go engine  ≡  Python reference · drift = 0", bsize=14)
chip(bx, gy + 48, bw, 40, "14 families", "Critical → Info · evidence per finding", bsize=14)

# ---- Column C : DELIVER ----
num_circle(ccx0 + 34 + 16, 296, 3, "gradPurple")
text(ccx0 + 90, 290, "Consumers", 24, 800, PURPLE)
text(ccx0 + 90, 318, "the engine is the source of truth", 15, 500, NAVY)
cx = ccx0 + 24
cw = (ccx1 - ccx0) - 48


def subcard(y, title, lines):
    h = 28 + len(lines) * 22 + 12
    rect(cx, y, cw, h, 9, PURPLE_TINT, "#E2D7F5", 1)
    text(cx + 18, y + 30, title, 18, 800, PURPLE)
    yy = y + 54
    for ln in lines:
        S.append('<circle cx="%s" cy="%s" r="2.6" fill="%s"/>' % (cx + 22, yy - 4, NAVY))
        text(cx + 34, yy, ln, 14.5, 500, NAVY)
        yy += 22
    return y + h + 14


yy = 348
yy = subcard(yy, "Visualization", ["View families: HLD · MLD · risk",
                                    "boundary · cross-sub · finding-centric",
                                    "Deterministic .drawio, CI-gated"])
yy = subcard(yy, "Generator", ["Intent → validated Terraform PR",
                               "gate blocks Critical / High / Medium"])
yy = subcard(yy, "MCP Server", ["6 governed tools · get_topology,",
                                "analyze_risks, simulate_change, …",
                                "middleware: authz · audit · injection guard"])

# ---- inter-stage connector pills ----
def connector(x, y, label):
    w = 16 + len(label) * 8.2
    arrow_right(x - 26, x - w / 2 - 4, y)
    arrow_right(x + w / 2 + 4, x + 26, y)
    rect(x - w / 2, y - 16, w, 32, 16, "white", TEAL, 1.4, shadow=True)
    text(x, y + 5, label, 14, 800, TEAL, "middle")


connector(cbx0, 336, "Graph IR")
connector(cbx1, 336, "findings + overlay")

# ---------------------------------------------------------------- foundation band
FB_Y = 736
rect(26, FB_Y, 1484, 132, 9, "white", BORDER, 1.1, shadow=True)
text(768, FB_Y + 30, "ENGINEERED INTO EVERY STAGE", 17, 800, NAVY, "middle")
pillars = [
    ("DETERMINISM", ["sort before emit", "byte-identical artifacts"]),
    ("TWIN-DRIFT PARITY", ["Go engine ≡ Python", "reference · 0 divergences"]),
    ("READ-ONLY · LEAST PRIV", ["Managed Identity / OIDC", "no writes · no terraform apply"]),
    ("CI-GATED", ["go test · twin-drift", "diagram-eval · views-gate"]),
]
pw = 1484 / 4
for i, (t, lines) in enumerate(pillars):
    px = 26 + i * pw
    if i:
        line(px, FB_Y + 50, px, FB_Y + 120, BORDER, 1.2)
    icx = px + 40
    icy = FB_Y + 86
    S.append('<path d="M%s %s L%s %s L%s %s Q%s %s %s %s L%s %s Z" fill="none" stroke="%s" stroke-width="3"/>'
             % (icx, icy - 26, icx + 26, icy - 14, icx + 22, icy + 22, icx, icy + 34, icx - 22, icy + 22, icx - 26, icy - 14, TEAL))
    S.append('<path d="M%s %s l8 8 l16 -19" fill="none" stroke="%s" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"/>'
             % (icx - 12, icy + 2, NAVY))
    text(px + 78, FB_Y + 70, t, 16, 800, TEAL)
    text(px + 78, FB_Y + 92, lines, 13.5, 500, NAVY, spacing=18)

# ---------------------------------------------------------------- footer brand
# left tag
text(28, 1004, "antr  ·  Azure Network Topology Reviewer", 15, 600, GREY)
# right mountain-A mark + wordmark
mx = 1158
S.append('<path d="M%s 992 L%s 902 L%s 992 H%s L%s 952 L%s 992 Z" fill="url(#gradTeal)" opacity="0.95"/>'
         % (mx, mx + 50, mx + 100, mx + 70, mx + 50, mx + 30))
S.append('<path d="M%s 902 L%s 992" stroke="%s" stroke-width="9" opacity="0.6"/>' % (mx + 50, mx + 100, PURPLE))
text(mx + 112, 962, "Aara Minds", 32, 500, NAVY)
text(mx + 114, 990, "Leadership • AI • Engineering", 13, 600, TEAL)

S.append('</svg>')

out = os.path.join(os.path.dirname(os.path.abspath(__file__)), "architecture.svg")
with open(out, "w", encoding="utf-8") as f:
    f.write("\n".join(S))
print("wrote", out)

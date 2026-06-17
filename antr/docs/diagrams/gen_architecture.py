#!/usr/bin/env python3
"""antr architecture diagram — AaraMinds studio style (dark, MCP-Governance-series look).

Brand cues replicated from the LinkedIn-newsletter assets:
  - dark teal→navy gradient field with a faint hexagon texture + soft glow
  - eyebrow label (teal dash + tracked caps), big white→teal→purple gradient title
  - cards with a gradient top accent-bar, neon gradient-stroke icons, big index number,
    outline pill, title, subtitle, divider, section label + detail
  - top-right mini-flow of glowing hexagon nodes
  - real Aara Minds logo badge bottom-right

Run:  python3 gen_architecture.py  ->  architecture.svg
"""
import base64
import os

HERE = os.path.dirname(os.path.abspath(__file__))
W, H = 1600, 900
FONT = "Inter, Segoe UI, Arial, sans-serif"
WHITE = "#FFFFFF"
MUTE = "#9DB2C6"
MUTE2 = "#6F869C"
CARD = "#0E2236"
CARD_STROKE = "#21405C"
S = []


def esc(t):
    return (str(t).replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;"))


def text(x, y, t, size, weight, fill, anchor="start", spacing=None, ls=None, opacity=None):
    parts = t if isinstance(t, list) else [t]
    sp = ""
    for i, ln in enumerate(parts):
        dy = 0 if i == 0 else (spacing or size * 1.3)
        sp += '<tspan x="%s" dy="%s">%s</tspan>' % (x, dy, esc(ln))
    extra = ""
    if ls is not None:
        extra += ' letter-spacing="%s"' % ls
    if opacity is not None:
        extra += ' opacity="%s"' % opacity
    S.append('<text x="%s" y="%s" font-family="%s" font-size="%s" font-weight="%s" fill="%s" '
             'text-anchor="%s"%s>%s</text>' % (x, y, FONT, size, weight, fill, anchor, extra, sp))


def rrect(x, y, w, h, rx, fill, stroke=None, sw=1.4, opacity=None, glow=False):
    s = '<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s"' % (x, y, w, h, rx, fill)
    if stroke:
        s += ' stroke="%s" stroke-width="%s"' % (stroke, sw)
    if opacity is not None:
        s += ' opacity="%s"' % opacity
    if glow:
        s += ' filter="url(#glow)"'
    s += ' />'
    S.append(s)


def grad(gid, c0, c1, x1=0, y1=0, x2=1, y2=1):
    return ('<linearGradient id="%s" x1="%s" x2="%s" y1="%s" y2="%s">'
            '<stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient>'
            % (gid, x1, x2, y1, y2, c0, c1))


# ---- accent palette (per card / icon) ----
ACC = {
    "teal":   ("#33E0CE", "#1FA6C9"),
    "purple": ("#B488F4", "#7C4DDB"),
    "pink":   ("#F65FA6", "#E0457E"),
    "green":  ("#42E29B", "#10B981"),
}

# ---------------------------------------------------------------- defs
defs = ['<defs>']
defs.append('<radialGradient id="bg" cx="40%" cy="22%" r="95%">'
            '<stop offset="0" stop-color="#123A4D"/><stop offset="45%" stop-color="#0B2336"/>'
            '<stop offset="100%" stop-color="#070F22"/></radialGradient>')
defs.append('<radialGradient id="titleGlow" cx="50%" cy="50%" r="50%">'
            '<stop offset="0" stop-color="#1FB6C9" stop-opacity="0.40"/>'
            '<stop offset="100%" stop-color="#1FB6C9" stop-opacity="0"/></radialGradient>')
defs.append(grad("titleGrad", "#FFFFFF", "#8FA8F0", 0, 0, 1, 0))
defs.append('<linearGradient id="titleGrad2" x1="0" x2="1" y1="0" y2="0">'
            '<stop offset="0" stop-color="#FFFFFF"/><stop offset="0.55" stop-color="#5FD6E6"/>'
            '<stop offset="1" stop-color="#9B7BE8"/></linearGradient>')
for k, (a, b) in ACC.items():
    defs.append(grad("acc_" + k, a, b))
defs.append('<linearGradient id="rail" x1="0" x2="1" y1="0" y2="0">'
            '<stop offset="0" stop-color="#33E0CE"/><stop offset="0.5" stop-color="#7C4DDB"/>'
            '<stop offset="1" stop-color="#E0457E"/></linearGradient>')
defs.append('<filter id="glow" x="-60%" y="-60%" width="220%" height="220%">'
            '<feGaussianBlur stdDeviation="6" result="b"/><feMerge>'
            '<feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge></filter>')
defs.append('<filter id="softshadow" x="-20%" y="-20%" width="140%" height="160%">'
            '<feDropShadow dx="0" dy="10" stdDeviation="16" flood-color="#000814" flood-opacity="0.55"/></filter>')
# hex texture (seamless honeycomb)
defs.append('<pattern id="hex" x="0" y="0" width="56" height="100" patternUnits="userSpaceOnUse" '
            'patternTransform="scale(0.62)">'
            '<path d="M28 66L0 50L0 16L28 0L56 16L56 50L28 66L28 100" fill="none" '
            'stroke="#9FE9F2" stroke-opacity="0.06" stroke-width="1.3"/></pattern>')
defs.append('</defs>')
S.append('<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">' % (W, H, W, H))
S.extend(defs)

# ---------------------------------------------------------------- background
rrect(0, 0, W, H, 0, "url(#bg)")
rrect(0, 0, W, H, 0, "url(#hex)")
S.append('<ellipse cx="560" cy="190" rx="620" ry="300" fill="url(#titleGlow)"/>')

# ---------------------------------------------------------------- icons
def icon_box(x, y, key, glyph):
    a = "url(#acc_%s)" % key
    rrect(x, y, 66, 66, 16, "#0C2031", a, 1.8)
    S.append('<g transform="translate(%s,%s)" fill="none" stroke="%s" stroke-width="3.4" '
             'stroke-linecap="round" stroke-linejoin="round" filter="url(#glow)">%s</g>'
             % (x + 33, y + 33, a, glyph))


GLYPH = {
    # cloud + down arrow (discover from Azure)
    "discover": '<path d="M-17 4 a13 13 0 0 1 4 -25 a16 16 0 0 1 30 5 a11 11 0 0 1 -2 22 Z"/>'
                '<path d="M0 -2 V14 M-7 7 L0 14 L7 7"/>',
    # shield + check (deterministic verdict)
    "analyze":  '<path d="M0 -22 L18 -14 V2 Q18 16 0 23 Q-18 16 -18 2 V-14 Z"/>'
                '<path d="M-9 0 l6 7 l12 -15"/>',
    # fan-out nodes (deliver to 3 products)
    "deliver":  '<circle cx="-16" cy="0" r="6"/><circle cx="15" cy="-15" r="6"/>'
                '<circle cx="15" cy="0" r="6"/><circle cx="15" cy="15" r="6"/>'
                '<path d="M-10 0 H9 M-11 -3 L9 -14 M-11 3 L9 14"/>',
}
MINI = {
    "azure":   '<path d="M-14 5 a10 10 0 0 1 3 -20 a13 13 0 0 1 24 4 a9 9 0 0 1 -2 18 Z"/>',
    "engine":  '<path d="M0 -16 L14 -8 V8 L0 16 L-14 8 V-8 Z"/><circle cx="0" cy="0" r="5"/>',
    "artifact": '<rect x="-12" y="-15" width="24" height="30" rx="3"/><path d="M-6 -7 H6 M-6 0 H6 M-6 7 H2"/>',
    "found_det": '<path d="M-12 -6 H12 M-12 6 H12"/>',          # equals — determinism
    "found_twin": '<circle cx="-5" cy="0" r="10"/><circle cx="5" cy="0" r="10"/>',  # twin
    "found_lock": '<rect x="-11" y="-2" width="22" height="16" rx="3"/>'
                  '<path d="M-7 -2 V-8 a7 7 0 0 1 14 0 V-2"/>',  # read-only lock
    "found_ci":   '<path d="M0 -13 L11 -7 V6 Q11 13 0 16 Q-11 13 -11 6 V-7 Z"/><path d="M-5 0 l4 5 l8 -10"/>',  # ci gate
}

# ---------------------------------------------------------------- header
text(96, 92, "—", 22, 800, "#33E0CE")
text(132, 96, "AZURE NETWORK TOPOLOGY REVIEWER", 19, 700, "#5FD6E6", ls="3.5")
text(94, 180, "Adopt the map.", 74, 800, "url(#titleGrad2)")
text(94, 262, "Own the risk.", 74, 800, "url(#titleGrad2)")
text(96, 312, "Discovery feeds the map. The deterministic engine owns the verdict.", 24, 500, MUTE)

# ---- top-right mini flow ----
mf = [("azure", "AZURE"), ("engine", "ENGINE"), ("artifact", "ARTIFACTS")]
mx, my = 1170, 150
text(1500, 96, "THE PIPELINE", 17, 700, "#5FD6E6", "end", ls="3")
for i, (g, lbl) in enumerate(mf):
    cx = mx + i * 165
    S.append('<g transform="translate(%s,%s)" fill="none" stroke="url(#acc_teal)" stroke-width="3" '
             'stroke-linecap="round" stroke-linejoin="round" filter="url(#glow)">%s</g>' % (cx, my, MINI[g]))
    text(cx, my + 52, lbl, 16, 700, WHITE, "middle", ls="1.5")
    if i < 2:
        ax = cx + 40
        S.append('<path d="M%s %s H%s" stroke="#4A7C93" stroke-width="2.5"/>'
                 '<path d="M%s %s l-8 -5 v10 z" fill="#4A7C93"/>' % (ax, my, cx + 125, cx + 125, my))

# ---------------------------------------------------------------- stage cards
STAGES = [
    ("01", "teal", "DISCOVER", "READ-ONLY", "Azure  →  Graph IR.", "discover", "WHAT IT READS",
     ["Resource Graph (paginated KQL):", "VNets · NSGs · NICs · routes · peerings",
      "Network Watcher effective rules / routes,", "AVNM, Firewall, App-GW/AKS/FD/APIM/vWAN.",
      "Managed Identity / OIDC · never a secret."]),
    ("02", "purple", "ANALYZE", "DETERMINISTIC", "Same input  →  same output.", "analyze", "THE ENGINE",
     ["4-gate reachability:", "AVNM → NSG → routes → public IP.",
      "Go engine  ≡  Python reference twin,", "twin-drift = 0 divergences.",
      "14 finding families · severity + evidence."]),
    ("03", "pink", "DELIVER", "GOVERNED", "One IR  →  three products.", "deliver", "THE PRODUCTS",
     ["Visualization — view families:", "HLD · MLD · risk · boundary · finding.",
      "Generator — intent → Terraform PR", "(gate blocks Critical / High / Medium).",
      "MCP server — 6 governed tools."]),
]
CARD_Y, CARD_H = 360, 412
CW, GAP = 452, 38
X0 = (W - (3 * CW + 2 * GAP)) / 2
gap_centers = []
for i, (num, key, title, pill, sub, icon, lbl, lines) in enumerate(STAGES):
    x = X0 + i * (CW + GAP)
    a = "url(#acc_%s)" % key
    ac = ACC[key][0]
    rrect(x, CARD_Y, CW, CARD_H, 18, CARD, CARD_STROKE, 1.4)
    rrect(x, CARD_Y, CW, CARD_H, 18, "none")  # spacer
    # top accent bar
    rrect(x + 18, CARD_Y - 3, CW - 36, 6, 3, a, glow=True)
    icon_box(x + 34, CARD_Y + 34, key, GLYPH[icon])
    text(x + CW - 34, CARD_Y + 78, num, 50, 800, WHITE, "end", opacity=0.14)
    # pill
    pw = 30 + len(pill) * 9.2
    rrect(x + CW - 34 - pw, CARD_Y + 96, pw, 30, 15, "none", ac, 1.6)
    text(x + CW - 34 - pw / 2, CARD_Y + 116, pill, 14, 700, ac, "middle", ls="1")
    text(x + 36, CARD_Y + 178, title, 36, 800, WHITE)
    text(x + 36, CARD_Y + 212, sub, 18, 500, MUTE)
    S.append('<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-width="1.2"/>'
             % (x + 36, CARD_Y + 238, x + CW - 36, CARD_Y + 238, "#21405C"))
    text(x + 36, CARD_Y + 268, lbl, 14, 800, ac, ls="2")
    text(x + 36, CARD_Y + 296, lines, 15.5, 500, "#C4D4E2", spacing=23)
    if i < 2:
        gap_centers.append(x + CW + GAP / 2)

# ---- connectors between stage cards ----
conn_labels = ["graph.Fixture (IR)", "findings + overlay"]
cy = CARD_Y + 150
for gc, lab in zip(gap_centers, conn_labels):
    S.append('<circle cx="%s" cy="%s" r="17" fill="#0C2031" stroke="url(#acc_teal)" stroke-width="2" filter="url(#glow)"/>' % (gc, cy))
    S.append('<path d="M%s %s l-5 -6 v12 z M%s %s h-12" stroke="url(#acc_teal)" stroke-width="3" '
             'fill="url(#acc_teal)" stroke-linecap="round"/>' % (gc + 6, cy, gc + 6, cy))
    text(gc, cy + 42, lab, 13.5, 700, "#7FE3D6", "middle")

# ---------------------------------------------------------------- foundation strip
FY = CARD_Y + CARD_H + 36
text(96, FY + 4, "ENGINEERED INTO EVERY STAGE", 16, 800, "#5FD6E6", ls="3")
found = [
    ("found_det", "green", "DETERMINISM", "sort before emit · byte-identical"),
    ("found_twin", "teal", "TWIN-DRIFT PARITY", "Go engine ≡ Python · 0 drift"),
    ("found_lock", "purple", "READ-ONLY · LEAST PRIV", "Managed Identity · no writes"),
    ("found_ci", "pink", "CI-GATED", "go test · twin-drift · views-gate"),
]
fw = (W - 192) / 4
for i, (g, key, t, sub) in enumerate(found):
    fx = 96 + i * fw
    a = "url(#acc_%s)" % key
    S.append('<g transform="translate(%s,%s)" fill="none" stroke="%s" stroke-width="3" '
             'stroke-linecap="round" stroke-linejoin="round" filter="url(#glow)">%s</g>'
             % (fx + 26, FY + 52, a, MINI[g]))
    text(fx + 60, FY + 46, t, 16.5, 800, WHITE)
    text(fx + 60, FY + 70, sub, 14, 500, MUTE)

# ---------------------------------------------------------------- footer
text(96, H - 28, "AZURE NETWORK TOPOLOGY REVIEWER  ·  DISCOVER → ANALYZE → DELIVER", 14.5, 600, MUTE2, ls="1")
# logo badge bottom-right (embed real Aara Minds logo)
logo_path = os.path.join(HERE, "aaraminds_logo.png")
if os.path.exists(logo_path):
    b64 = base64.b64encode(open(logo_path, "rb").read()).decode()
    bw, bh = 250, 84
    bx, by = W - bw - 70, H - bh - 22
    rrect(bx, by, bw, bh, 14, "#060C1C", "#1B3147", 1.4)
    S.append('<image x="%s" y="%s" width="%s" height="%s" href="data:image/png;base64,%s" '
             'preserveAspectRatio="xMidYMid meet"/>' % (bx + 16, by + 14, bw - 32, bh - 28, b64))

S.append('</svg>')
out = os.path.join(HERE, "architecture.svg")
with open(out, "w", encoding="utf-8") as f:
    f.write("\n".join(S))
print("wrote", out)

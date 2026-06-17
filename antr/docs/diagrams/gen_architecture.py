#!/usr/bin/env python3
"""antr architecture diagram — AaraMinds studio style, themeable.

Emits three brand-consistent variants:
  architecture.svg         dark  16:9   (the canonical master, MCP-series dark look)
  architecture-light.svg   light 16:9   (the light-slide look: white cards on a hex field)
  architecture-square.svg  light 1:1    (1080×1080, vertical stack — for LinkedIn)

Brand cues from the LinkedIn-newsletter assets: hexagon-textured field, eyebrow +
gradient title, accent-bar cards with gradient-stroke icons, index numbers, outline
pills, a stage mini-flow, an 'engineered into every stage' foundation strip, and the
real Aara Minds logo badge.

Run:  python3 gen_architecture.py
Rasterize with a real SVG engine (resvg / browser / draw.io), NOT ImageMagick (it
mangles gradient-opacity + blur):  node render.js architecture.svg architecture.png 3200
"""
import base64
import os

HERE = os.path.dirname(os.path.abspath(__file__))
FONT = "Inter, Segoe UI, Arial, sans-serif"

ACC = {"teal": ("#33E0CE", "#1FA6C9"), "purple": ("#B488F4", "#7C4DDB"),
       "pink": ("#F65FA6", "#E0457E"), "green": ("#42E29B", "#10B981")}

THEMES = {
    "dark": dict(
        bg='<radialGradient id="bg" cx="40%" cy="22%" r="95%"><stop offset="0" stop-color="#123A4D"/>'
           '<stop offset="45%" stop-color="#0B2336"/><stop offset="100%" stop-color="#070F22"/></radialGradient>',
        hexstroke="#9FE9F2", hexop="0.06", titleglow=True,
        title_stops=[("0", "#FFFFFF"), ("0.55", "#5FD6E6"), ("1", "#9B7BE8")],
        eyebrow="#5FD6E6", white="#FFFFFF", mute="#9DB2C6", mute2="#6F869C",
        card="#0E2236", card_stroke="#21405C", iconbox="#0C2031", iconfx="url(#glow)",
        detail="#C4D4E2", divider="#21405C", flabel="#5FD6E6", subtle="#7FE3D6",
        cardfx="url(#glow)"),
    "light": dict(
        bg='<linearGradient id="bg" x1="0" x2="0.6" y1="0" y2="1"><stop offset="0" stop-color="#FFFFFF"/>'
           '<stop offset="100%" stop-color="#E9F1F8"/></linearGradient>',
        hexstroke="#0B2B45", hexop="0.05", titleglow=False,
        title_stops=[("0", "#0B1B4D"), ("0.55", "#2D6CDF"), ("1", "#7C4DDB")],
        eyebrow="#007D8F", white="#071B4D", mute="#5D6472", mute2="#8595A6",
        card="#FFFFFF", card_stroke="#C9D8E6", iconbox="#F1F7F9", iconfx="",
        detail="#3A4A5A", divider="#DCE8F0", flabel="#007D8F", subtle="#0E8C9E",
        cardfx="url(#softshadow)"),
}

GLYPH = {
    "discover": '<path d="M-17 4 a13 13 0 0 1 4 -25 a16 16 0 0 1 30 5 a11 11 0 0 1 -2 22 Z"/>'
                '<path d="M0 -2 V14 M-7 7 L0 14 L7 7"/>',
    "analyze": '<path d="M0 -22 L18 -14 V2 Q18 16 0 23 Q-18 16 -18 2 V-14 Z"/><path d="M-9 0 l6 7 l12 -15"/>',
    "deliver": '<circle cx="-16" cy="0" r="6"/><circle cx="15" cy="-15" r="6"/><circle cx="15" cy="0" r="6"/>'
               '<circle cx="15" cy="15" r="6"/><path d="M-10 0 H9 M-11 -3 L9 -14 M-11 3 L9 14"/>',
}
MINI = {
    "azure": '<path d="M-14 5 a10 10 0 0 1 3 -20 a13 13 0 0 1 24 4 a9 9 0 0 1 -2 18 Z"/>',
    "engine": '<path d="M0 -16 L14 -8 V8 L0 16 L-14 8 V-8 Z"/><circle cx="0" cy="0" r="5"/>',
    "artifact": '<rect x="-12" y="-15" width="24" height="30" rx="3"/><path d="M-6 -7 H6 M-6 0 H6 M-6 7 H2"/>',
    "found_det": '<path d="M-12 -6 H12 M-12 6 H12"/>',
    "found_twin": '<circle cx="-5" cy="0" r="10"/><circle cx="5" cy="0" r="10"/>',
    "found_lock": '<rect x="-11" y="-2" width="22" height="16" rx="3"/><path d="M-7 -2 V-8 a7 7 0 0 1 14 0 V-2"/>',
    "found_ci": '<path d="M0 -13 L11 -7 V6 Q11 13 0 16 Q-11 13 -11 6 V-7 Z"/><path d="M-5 0 l4 5 l8 -10"/>',
}
STAGES = [
    ("01", "teal", "DISCOVER", "READ-ONLY", "Azure  →  Graph IR.", "discover", "WHAT IT READS",
     ["Resource Graph (paginated KQL): VNets,", "NSGs · NICs · routes · peerings.",
      "Network Watcher · AVNM · Firewall · app-layer.", "Managed Identity / OIDC · never a secret."]),
    ("02", "purple", "ANALYZE", "DETERMINISTIC", "Same input  →  same output.", "analyze", "THE ENGINE",
     ["4-gate reachability:", "AVNM → NSG → routes → public IP.",
      "Go engine  ≡  Python twin · 0 drift.", "14 finding families · severity + evidence."]),
    ("03", "pink", "DELIVER", "GOVERNED", "One IR  →  three products.", "deliver", "THE PRODUCTS",
     ["Visualization — view families", "(HLD · MLD · risk · boundary · finding).",
      "Generator — intent → Terraform PR.", "MCP server — 6 governed tools."]),
]
FOUND = [("found_det", "green", "DETERMINISM", "sort before emit · byte-identical"),
         ("found_twin", "teal", "TWIN-DRIFT PARITY", "Go ≡ Python · 0 drift"),
         ("found_lock", "purple", "READ-ONLY · LEAST PRIV", "Managed Identity · no writes"),
         ("found_ci", "pink", "CI-GATED", "go test · twin-drift · views")]


def esc(t):
    return str(t).replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


class Doc:
    def __init__(self, th, W, H):
        self.T, self.W, self.H, self.S = th, W, H, []

    def text(self, x, y, t, size, weight, fill, anchor="start", spacing=None, ls=None, opacity=None):
        parts = t if isinstance(t, list) else [t]
        sp = ""
        for i, ln in enumerate(parts):
            dy = 0 if i == 0 else (spacing or size * 1.3)
            sp += '<tspan x="%s" dy="%s">%s</tspan>' % (x, dy, esc(ln))
        ex = (' letter-spacing="%s"' % ls if ls is not None else "") + (' opacity="%s"' % opacity if opacity is not None else "")
        self.S.append('<text x="%s" y="%s" font-family="%s" font-size="%s" font-weight="%s" fill="%s" text-anchor="%s"%s>%s</text>'
                      % (x, y, FONT, size, weight, fill, anchor, ex, sp))

    def rrect(self, x, y, w, h, rx, fill, stroke=None, sw=1.4, opacity=None, fx=None):
        s = '<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s"' % (x, y, w, h, rx, fill)
        if stroke:
            s += ' stroke="%s" stroke-width="%s"' % (stroke, sw)
        if opacity is not None:
            s += ' opacity="%s"' % opacity
        if fx:
            s += ' filter="%s"' % fx
        self.S.append(s + ' />')

    def line(self, x1, y1, x2, y2, stroke, sw=1):
        self.S.append('<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-width="%s"/>' % (x1, y1, x2, y2, stroke, sw))

    def num(self, cx, cy, n, key):
        self.S.append('<circle cx="%s" cy="%s" r="26" fill="url(#acc_%s)"/>' % (cx, cy, key))
        self.text(cx, cy + 9, str(n), 30, 800, "#FFFFFF", "middle")

    def pill(self, x, y, label, color):
        w = 30 + len(label) * 9.2
        self.rrect(x - w, y, w, 30, 15, "none", color, 1.6)
        self.text(x - w / 2, y + 20, label, 14, 700, color, "middle", ls="1")
        return w

    def icon(self, x, y, key, glyph, box=66, sw=3.4):
        a = "url(#acc_%s)" % key
        self.rrect(x, y, box, box, 16, self.T["iconbox"], a, 1.8)
        self.S.append('<g transform="translate(%s,%s)" fill="none" stroke="%s" stroke-width="%s" '
                      'stroke-linecap="round" stroke-linejoin="round"%s>%s</g>'
                      % (x + box / 2, y + box / 2, a, sw, (' filter="%s"' % self.T["iconfx"]) if self.T["iconfx"] else "", glyph))

    def defs(self):
        T = self.T
        d = ['<defs>', T["bg"]]
        ts = "".join('<stop offset="%s" stop-color="%s"/>' % (o, c) for o, c in T["title_stops"])
        d.append('<linearGradient id="titleGrad" x1="0" x2="1" y1="0" y2="0">%s</linearGradient>' % ts)
        for k, (a, b) in ACC.items():
            d.append('<linearGradient id="acc_%s" x1="0" x2="1" y1="0" y2="1"><stop offset="0" stop-color="%s"/>'
                     '<stop offset="1" stop-color="%s"/></linearGradient>' % (k, a, b))
        d.append('<filter id="glow" x="-60%" y="-60%" width="220%" height="220%"><feGaussianBlur stdDeviation="6" result="b"/>'
                 '<feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge></filter>')
        flood = "#000814" if T is THEMES["dark"] else "#1B3650"
        op = "0.55" if T is THEMES["dark"] else "0.16"
        d.append('<filter id="softshadow" x="-20%%" y="-20%%" width="140%%" height="160%%">'
                 '<feDropShadow dx="0" dy="8" stdDeviation="14" flood-color="%s" flood-opacity="%s"/></filter>' % (flood, op))
        if T["titleglow"]:
            d.append('<radialGradient id="tglow" cx="50%" cy="50%" r="50%"><stop offset="0" stop-color="#1FB6C9" stop-opacity="0.40"/>'
                     '<stop offset="100%" stop-color="#1FB6C9" stop-opacity="0"/></radialGradient>')
        d.append('<pattern id="hex" x="0" y="0" width="56" height="100" patternUnits="userSpaceOnUse" patternTransform="scale(0.62)">'
                 '<path d="M28 66L0 50L0 16L28 0L56 16L56 50L28 66L28 100" fill="none" stroke="%s" stroke-opacity="%s" stroke-width="1.3"/></pattern>'
                 % (T["hexstroke"], T["hexop"]))
        d.append('</defs>')
        return d

    def background(self):
        self.S.append('<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">' % (self.W, self.H, self.W, self.H))
        self.S.extend(self.defs())
        self.rrect(0, 0, self.W, self.H, 0, "url(#bg)")
        self.rrect(0, 0, self.W, self.H, 0, "url(#hex)")
        if self.T["titleglow"]:
            self.S.append('<ellipse cx="560" cy="190" rx="620" ry="300" fill="url(#tglow)"/>')

    def logo(self, bx, by, bw, bh):
        p = os.path.join(HERE, "aaraminds_logo.png")
        if not os.path.exists(p):
            return
        b64 = base64.b64encode(open(p, "rb").read()).decode()
        self.rrect(bx, by, bw, bh, 14, "#060C1C", "#1B3147", 1.4)
        self.S.append('<image x="%s" y="%s" width="%s" height="%s" href="data:image/png;base64,%s" preserveAspectRatio="xMidYMid meet"/>'
                      % (bx + 18, by + 16, bw - 36, bh - 32, b64))

    def out(self):
        return "\n".join(self.S + ["</svg>"])


def gen_wide(theme):
    T = THEMES[theme]
    d = Doc(T, 1600, 900)
    d.background()
    d.text(96, 92, "—", 22, 800, "#33E0CE" if theme == "dark" else "#007D8F")
    d.text(132, 96, "AZURE NETWORK TOPOLOGY REVIEWER", 19, 700, T["eyebrow"], ls="3.5")
    d.text(94, 180, "Adopt the map.", 74, 800, "url(#titleGrad)")
    d.text(94, 262, "Own the risk.", 74, 800, "url(#titleGrad)")
    d.text(96, 312, "Discovery feeds the map. The deterministic engine owns the verdict.", 24, 500, T["mute"])
    # mini-flow
    d.text(1500, 96, "THE PIPELINE", 17, 700, T["eyebrow"], "end", ls="3")
    for i, (g, lbl) in enumerate([("azure", "AZURE"), ("engine", "ENGINE"), ("artifact", "ARTIFACTS")]):
        cx = 1170 + i * 165
        d.S.append('<g transform="translate(%s,150)" fill="none" stroke="url(#acc_teal)" stroke-width="3" '
                   'stroke-linecap="round" stroke-linejoin="round"%s>%s</g>'
                   % (cx, (' filter="%s"' % T["iconfx"]) if T["iconfx"] else "", MINI[g]))
        d.text(cx, 202, lbl, 16, 700, T["white"], "middle", ls="1.5")
        if i < 2:
            d.S.append('<path d="M%s 150 H%s" stroke="#5A8DA6" stroke-width="2.5"/><path d="M%s 150 l-8 -5 v10 z" fill="#5A8DA6"/>'
                       % (cx + 40, cx + 125, cx + 125))
    # cards
    CY, CH, CW, GAP = 342, 388, 452, 38
    X0 = (1600 - (3 * CW + 2 * GAP)) / 2
    gaps = []
    for i, (num, key, title, pl, sub, ic, lbl, lines) in enumerate(STAGES):
        x = X0 + i * (CW + GAP)
        ac = ACC[key][0]
        d.rrect(x, CY, CW, CH, 18, T["card"], T["card_stroke"], 1.4, fx=T["cardfx"])
        d.rrect(x + 18, CY - 3, CW - 36, 6, 3, "url(#acc_%s)" % key, fx=T["iconfx"] or None)
        d.icon(x + 34, CY + 34, key, GLYPH[ic])
        d.text(x + CW - 34, CY + 78, num, 50, 800, T["white"], "end", opacity=0.14)
        d.pill(x + CW - 34, CY + 96, pl, ac)
        d.text(x + 36, CY + 176, title, 36, 800, T["white"])
        d.text(x + 36, CY + 210, sub, 18, 500, T["mute"])
        d.line(x + 36, CY + 236, x + CW - 36, CY + 236, T["divider"], 1.2)
        d.text(x + 36, CY + 266, lbl, 14, 800, ac, ls="2")
        d.text(x + 36, CY + 296, lines, 15, 500, T["detail"], spacing=22)
        if i < 2:
            gaps.append(x + CW + GAP / 2)
    for gc, lab in zip(gaps, ["graph.Fixture (IR)", "findings + overlay"]):
        cy = CY + 67
        d.S.append('<circle cx="%s" cy="%s" r="17" fill="%s" stroke="url(#acc_teal)" stroke-width="2"%s/>'
                   % (gc, cy, T["iconbox"], (' filter="%s"' % T["iconfx"]) if T["iconfx"] else ""))
        d.S.append('<path d="M%s %s l-5 -6 v12 z M%s %s h-12" stroke="url(#acc_teal)" stroke-width="3" fill="url(#acc_teal)" stroke-linecap="round"/>'
                   % (gc + 6, cy, gc + 6, cy))
        d.text(gc, cy + 42, lab, 13.5, 700, T["subtle"], "middle")
    # foundation
    FY = CY + CH + 28
    d.line(96, FY - 18, 1504, FY - 18, T["divider"], 1.2)
    d.text(96, FY + 2, "ENGINEERED INTO EVERY STAGE", 15, 800, T["flabel"], ls="3")
    fw = (1190 - 96) / 4
    for i, (g, key, t, sub) in enumerate(FOUND):
        fx = 96 + i * fw
        d.S.append('<g transform="translate(%s,%s)" fill="none" stroke="url(#acc_%s)" stroke-width="3" '
                   'stroke-linecap="round" stroke-linejoin="round"%s>%s</g>'
                   % (fx + 24, FY + 60, key, (' filter="%s"' % T["iconfx"]) if T["iconfx"] else "", MINI[g]))
        d.text(fx + 56, FY + 54, t, 15.5, 800, T["white"])
        d.text(fx + 56, FY + 76, sub, 13, 500, T["mute"])
    d.text(96, 874, "antr  ·  Azure Network Topology Reviewer  ·  DISCOVER → ANALYZE → DELIVER", 14, 600, T["mute2"], ls="0.5")
    d.logo(1600 - 244 - 70, FY + 18, 244, 84)
    return d.out()


def gen_square(theme):
    T = THEMES[theme]
    W = H = 1080
    d = Doc(T, W, H)
    d.background()
    d.text(70, 96, "—", 20, 800, "#007D8F" if theme == "light" else "#33E0CE")
    d.text(104, 99, "AZURE NETWORK TOPOLOGY REVIEWER", 16, 700, T["eyebrow"], ls="2.5")
    d.text(68, 168, "Adopt the map.", 56, 800, "url(#titleGrad)")
    d.text(68, 230, "Own the risk.", 56, 800, "url(#titleGrad)")
    d.text(70, 272, "Discovery feeds the map. The engine owns the verdict.", 19, 500, T["mute"])
    # 3 stacked stage rows
    RY, RH, GAP = 312, 168, 22
    for i, (num, key, title, pl, sub, ic, lbl, lines) in enumerate(STAGES):
        y = RY + i * (RH + GAP)
        ac = ACC[key][0]
        d.rrect(70, y, W - 140, RH, 16, T["card"], T["card_stroke"], 1.4, fx=T["cardfx"])
        d.rrect(70, y - 3, 6, RH + 6, 3, "url(#acc_%s)" % key)
        d.icon(98, y + 30, key, GLYPH[ic], box=58, sw=3)
        d.num(196, y + 40, num, key) if False else None
        d.text(178, y + 52, title, 30, 800, T["white"])
        d.pill(178 + 220, y + 30, pl, ac)
        d.text(178, y + 82, sub, 16, 500, T["mute"])
        d.text(178, y + 116, lbl, 12.5, 800, ac, ls="2")
        # detail as two-up
        det = lines[:2] + lines[2:]
        d.text(420, y + 52, det[:2], 14, 500, T["detail"], spacing=21)
        d.text(420, y + 100, det[2:4], 14, 500, T["detail"], spacing=21)
        d.text(W - 92, y + 40, num, 40, 800, T["white"], "end", opacity=0.13)
    # foundation row
    FY = RY + 3 * (RH + GAP) + 6
    d.text(70, FY, "ENGINEERED INTO EVERY STAGE", 14, 800, T["flabel"], ls="2.5")
    fw = (W - 140) / 4
    for i, (g, key, t, sub) in enumerate(FOUND):
        fx = 70 + i * fw
        d.S.append('<g transform="translate(%s,%s)" fill="none" stroke="url(#acc_%s)" stroke-width="3" '
                   'stroke-linecap="round" stroke-linejoin="round"%s>%s</g>'
                   % (fx + 20, FY + 44, key, (' filter="%s"' % T["iconfx"]) if T["iconfx"] else "", MINI[g]))
        d.text(fx + 48, FY + 40, t, 13, 800, T["white"])
        d.text(fx + 48, FY + 60, sub, 11.5, 500, T["mute"])
    d.logo(W - 244 - 70, H - 104, 244, 78)
    d.text(70, H - 40, "antr · DISCOVER → ANALYZE → DELIVER", 13, 600, T["mute2"])
    return d.out()


def main():
    outs = {
        "architecture.svg": gen_wide("dark"),
        "architecture-light.svg": gen_wide("light"),
        "architecture-square.svg": gen_square("light"),
    }
    for name, svg in outs.items():
        with open(os.path.join(HERE, name), "w", encoding="utf-8") as f:
            f.write(svg)
        print("wrote", name)


if __name__ == "__main__":
    main()

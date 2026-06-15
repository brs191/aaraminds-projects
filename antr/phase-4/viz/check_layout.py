#!/usr/bin/env python3
"""Phase-4 layout validator — readability gate.

Asserts the emitted diagram is legible:
  - no two SAME-PARENT vertices share a bounding box, AND
  - every child vertex fits inside its parent's box (no cross-parent overflow,
    e.g. a NIC spilling onto a VNet header) — closes the same-parent-only blind
    spot from the second audit.

Run:
    python3 check_layout.py <file.drawio>
"""
import sys
import xml.etree.ElementTree as ET
from collections import defaultdict


def _cells(path_or_root):
    root = path_or_root if isinstance(path_or_root, ET.Element) else ET.parse(path_or_root).getroot()
    geom = {}
    by_parent = defaultdict(list)
    for c in root.iter("mxCell"):
        if c.get("vertex") != "1":
            continue
        g = c.find("mxGeometry")
        if g is None:
            continue
        r = (float(g.get("x", 0)), float(g.get("y", 0)),
             float(g.get("width", 0)), float(g.get("height", 0)))
        geom[c.get("id")] = (c.get("parent"), r)
        by_parent[c.get("parent")].append((c.get("id"), r))
    return geom, by_parent


def _overlap(a, b, pad=1.0):
    ax, ay, aw, ah = a
    bx, by, bw, bh = b
    return not (ax + aw <= bx + pad or bx + bw <= ax + pad or
                ay + ah <= by + pad or by + bh <= ay + pad)


def check(path_or_root):
    geom, by_parent = _cells(path_or_root)
    overlaps, oob = [], []
    # sibling overlaps (geometry is parent-relative, so siblings are comparable)
    for parent, items in by_parent.items():
        for i in range(len(items)):
            for j in range(i + 1, len(items)):
                if _overlap(items[i][1], items[j][1]):
                    overlaps.append((parent, items[i][0], items[j][0]))
    # child must fit within parent's box
    for cid, (parent, (x, y, w, h)) in geom.items():
        if parent in geom:
            _, (_, _, pw, ph) = geom[parent]
            if x < -0.5 or y < -0.5 or x + w > pw + 0.5 or y + h > ph + 0.5:
                oob.append((cid, parent))
    total = len(geom)
    return overlaps + [("OOB",) + o for o in oob], total, by_parent


def main():
    if len(sys.argv) < 2:
        sys.exit("usage: python3 check_layout.py <file.drawio>")
    problems, total, by_parent = check(sys.argv[1])
    print("vertices: %d; containers: %d; problems: %d" % (total, len(by_parent), len(problems)))
    for p in problems[:10]:
        print("  ", p)
    sys.exit(1 if problems else 0)


if __name__ == "__main__":
    main()

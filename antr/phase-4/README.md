# Azure Network Topology Reviewer — Phase 4 Status

**Phase 4 — Enterprise Topology Visualization.** Status: **✅ IN-SESSION SCOPE COMPLETE** (Python
reference pipeline, 26/26 diagram-eval PASS, 3 adversarial audits remediated). Live discovery / Go port /
ELK-D2 / publish pipeline **deferred** (live Azure + Go 1.25 required). See
`PHASE_4_ACCEPTANCE_MEMO.md`, the executable playbook `../IMPLEMENTATION_PLAYBOOK_CLAUDE.md`, and the
design `design/VISUALIZATION_MODEL.md`.

Pipeline: `viz/overlay.py` (severity join) · `viz/render_drawio.py` (draw.io render) ·
`viz/views.py` (view-family projections) · `viz/check_layout.py` (layout gate) ·
`viz/eval_diagram.py` (RC1–RC5 + structure + coverage gate) · `viz/test_views.py` (view-family gate) ·
`viz/synth_estate.py` (scale generator). Outputs in `out/`.

## View families (2026-06-17 — `viz/views.py`)

One canvas of a real estate is soup. antr now emits a **set of views**, each a deterministic
projection over one whole-estate risk truth (overview-plus-detail), all painted from the same
`Analyze()` overlay — a view can hide a resource but never change a verdict:

| View | Scope |
|---|---|
| `hld` | VNets, hubs, spokes, gateways, firewall, peerings, boundary |
| `mld` | Full detail: subnets + NICs |
| `risk` | Only resources that carry a finding (+ their VNet/subnet) + boundary |
| `boundary` | Internet-facing paths: boundary nodes + every internet-reachable NIC |
| `cross-sub` | Only VNets in cross-subscription peerings (multi-sub blast radius) |
| `finding/<n>` | One small k-hop diagram per Critical/High finding around its node |

Run: `python3 viz/views.py <fixture.json> --out-dir out/views`. Gated by `viz/test_views.py`
(projection faithfulness + byte-identical determinism, 26/26 corpus). The swap contract for the
layout engine is pinned in `design/GRAPH_IR.md`; the strategy decision is `design/ADR-001-visualization-strategy.md`.

## Why this phase exists

A real-estate render (`../ref-topology/generated_antr.pdf`) compared against the human
reference (`../ref-topology/BCLM-Revised-8June2026.drawio`) showed the generated diagram had
**near-zero connectivity** and every node rendered "Clean". The reference has 288 connection
edges and an Internet boundary. Four root causes (RC-1…RC-4) are documented in the design doc.

## Strategy in one sentence

Separate **the map** (discovery + layout — adopt OSS) from **the risk** (reachability/severity
— keep antr's `Analyze()` engine), and paint findings onto the map.

## Key decision (sharpened — see `design/ADR-001-visualization-strategy.md`)

| Decision | Choice |
|---|---|
| Discovery | **antr's own hardened adapter** is the single source of topology truth (not CloudNetDraw's) |
| Layout | In-house deterministic `render_drawio` now; layout engine swappable behind `design/GRAPH_IR.md` |
| Readability | **View families** (`viz/views.py`) now; ELK/Graphviz `dot` the preferred deterministic backend if swapped |
| CloudNetDraw | **MIT**, verified; layout-only **adopt-and-patch** option (needs sort-before-emit determinism fix) — fallback, not base |
| Risk engine | antr `Analyze()` — **unchanged**; computes all severity, once, on the full estate |
| Ground-truth cross-check | Azure Network Watcher / Monitor **Network Insights Topology** |
| Discovery auth | Managed Identity / OIDC, Reader scope — never `AZURE_CLIENT_SECRET` |
| Attack-path graph (Cartography) | Deferred to Phase 5 |

## Steps

Two tracks: the **live** plan (`../IMPLEMENTATION_PLAYBOOK.md` § Phase 4, steps 4.1–4.7 — needs Azure)
and the **in-session execution** (`../IMPLEMENTATION_PLAYBOOK_CLAUDE.md`, steps 4C.0–4C.6 — done today on
fixtures). In-session status:

| Step | Title | Status |
|---|---|---|
| 4C.0 | Multi-subscription estate fixture | ✅ PASS |
| 4C.1 | Severity overlay (`Analyze()` join) | ✅ PASS |
| 4C.2 | draw.io renderer (edges/boundary/paint) | ✅ PASS |
| 4C.3 | Layout legibility (overlap + containment) | ✅ PASS |
| 4C.4 | Diagram-eval gate (26/26 corpus) | ✅ PASS |
| 4C.5 | Enterprise-scale proof (synth estate) | ✅ PASS |
| 4C.6 | Acceptance memo + docs | ✅ PASS |
| 4C.verify | 3 adversarial audits remediated | ✅ PASS |

## Gate table (in-session, fixture-provable)

| Gate | Criterion | Verdict |
|---|---|---|
| G1 | Cross-sub + spoke-to-spoke peering edges render; 0 dangling (RC-1/RC-2) | ✅ PASS |
| G2 | External boundary nodes render where present (RC-3) | ✅ PASS |
| G3 | `Analyze()` findings paint node severity on HLD+MLD; legend accurate (RC-4) | ✅ PASS |
| G4 | Severity computed only by `Analyze()` — renderer assigns none | ✅ PASS |
| G5 | Discovery uses Managed Identity / OIDC read-only — no client secret | 🔲 DEFERRED (live) |
| +structure | Unique cell ids; all edge endpoints exist | ✅ PASS |
| +RC5 | No sibling overlap / no child-overflow | ✅ PASS |

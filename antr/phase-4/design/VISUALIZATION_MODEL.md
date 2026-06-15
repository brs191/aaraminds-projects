# Phase 4 — Enterprise Topology Visualization Model

**Project:** Azure Network Topology Reviewer · **Phase:** 4 — Enterprise Topology Visualization
**Date:** 2026-06-15 · **Status:** DESIGN (not started) · **Reads with:** `baseline/TARGET_ARCHITECTURE.md`, `phase-1/design/TOPOLOGY_MODEL.md`

---

## Verdict (lead)

Stop hand-rolling the diagram renderer. Producing a readable, enterprise-grade Azure
topology is a commoditised, solved problem — there is a purpose-built, MIT-licensed tool
(**CloudNetDraw**) that already does discovery + hub-spoke layout + `.drawio` export across
**multiple subscriptions**, which is exactly where `engine/go/renderer/drawio.go` failed.

antr's defensible value is **not** the picture. It is the deterministic `Analyze()`
reachability/severity engine, which no diagram tool has. Phase 4 therefore **separates the
map from the risk**: adopt best-of-breed OSS for discovery + layout, and keep the antr engine
as the **security overlay** painted on top of that map.

This phase exists because of a concrete, observed failure — see §1.

---

## 1. The triggering failure (`ref-topology/generated_antr.pdf`)

A real-estate render produced by the Phase-1 `get_topology` + `renderer/drawio.go` path,
exported into Confluence, was compared against the human-authored reference
`ref-topology/BCLM-Revised-8June2026.drawio`. The reference carries **288 directed
connection edges** (287 arrowed, 18 dashed), semantic zones, and an Internet boundary.
The generated output had **essentially zero connectivity** and every node rendered "Clean".

Root causes (verified in code, 2026-06-15):

| # | Root cause | Evidence | Owner phase |
|---|---|---|---|
| RC-1 | `FetchFixture(ctx, cred, subscriptionID)` is **single-subscription**; all 18 KQL queries filter `where subscriptionId == %q`. Hub-spoke estates peer across subscriptions, so remote VNets are absent and `tgtID := "vnet-"+slugify(peer.RemoteVnet)` points to a non-existent mxCell → draw.io drops the dangling edge. | `engine/go/adapter/azure.go:32`; `engine/go/renderer/drawio.go:384` | Phase 4 |
| RC-2 | `CrossSubscriptionPeerings []CrossSubPeering` exists in the model but is **never referenced by the renderer** — only `vnet.Peerings` is looped. Cross-sub links produce zero edges by construction. | `engine/go/internal/graph/model.go:20`; renderer grep → 0 refs | Phase 4 |
| RC-3 | Renderer has **no external-boundary node type** (no Internet / ExpressRoute / VPN GW / NAT GW / public-IP node). The reference's most important edges cannot be drawn at all. | `renderer/drawio.go` vertex types: VNet, firewall, subnet, NIC, PE, LB only | Phase 4 |
| RC-4 | Findings from `Analyze()` are **not joined to the render** — legend advertises severity but every node is the default fill. | No call path from `analyze.Finding` → node style in `renderer/` | Phase 4 |

RC-1/RC-3 are scope/feature gaps, not bugs. RC-2/RC-4 are missing wiring. None is a one-line fix.

---

## 2. Strategic reframe — map vs. risk

```
                          ┌──────────────────────────────────────────────┐
   Azure Resource Graph   │  THE MAP  (commodity — adopt OSS)            │
   + Network Watcher  ───►│  discovery · hub-spoke detection · layout    │──► base .drawio
   (multi-sub / mgmt-grp) │  (CloudNetDraw + ELK)                        │
                          └──────────────────────────────────────────────┘
                                          │  node IDs keyed by Azure resource ID
                          ┌───────────────▼──────────────────────────────┐
   antr Analyze() (Go) ──►│  THE RISK  (antr's moat — keep + own)        │
   reachability/severity  │  paint findings onto nodes (fill + badge)    │──► findings[]
                          └──────────────────────────────────────────────┘
                                          │
                       ┌──────────────────┼─────────────────────┐
                    .drawio             SVG/PNG            interactive web
                  (Confluence)         (reports)          (live artifact)
```

The merge step is cheap: both sides key off the same Azure resource IDs, and both already
emit/consume draw.io. antr does **not** rebuild discovery or layout; it consumes them.

---

## 3. OSS landscape decision

Researched 2026-06-15. Ranked for this project's need (Azure, multi-sub, drawio→Confluence,
read-only, deterministic severity overlay).

| Tool | Role here | License | Decision |
|---|---|---|---|
| **CloudNetDraw** (`krhatland/cloudnetdraw`) | Discovery + hub-spoke layout + HLD/MLD `.drawio`; multi-subscription via SP Reader; ExpressRoute/VPN GW/Firewall boundary objects; 6 edge types incl. spoke-to-spoke/cross-zone/multi-hub; portal hyperlinks; Azure-Function deployable | MIT | **ADOPT (fork + vendor)** — primary discovery + layout engine |
| **Azure Network Watcher / Monitor Network Insights — Topology** | Native multi-sub/region/RG interactive view with JSON export | Proprietary (native) | **USE as ground-truth cross-check** — validates discovery completeness; not embeddable |
| **D2 + ELK** (`terrastruct/d2`, Eclipse Layout Kernel) | Go-native renderer that embeds ELK for compound/orthogonal auto-layout; SVG/PNG | MPL-2.0 / EPL | **ADOPT for readability** — replaces hand-placed mxGraph coordinates; cures the "wall of boxes" |
| **AzViz** (`PrateekKumarSingh/AzViz`) | PowerShell + Graphviz, strong Azure icons, RG-scoped | MIT | **REFERENCE only** — icon set; too RG-scoped for the enterprise pipeline |
| **Cartography** (CNCF, ex-Lyft) | Azure + Entra ID → Neo4j; Cypher attack-path / lateral-movement queries | Apache-2.0 | **DEFER to Phase 5** — substrate for multi-hop attack-paths/drift, not for the diagram |
| **Hava.io / Cloudockit** | Commercial auto-updating interactive diagrams; API/CLI; self-host | Commercial | **BENCHMARK only** — defines the UX bar; neither computes reachability/severity, so neither replaces the engine. Buy only if UX timeline beats differentiation. |

**Buy-vs-build verdict:** build the risk overlay (unique, antr's moat); adopt OSS for
discovery + layout (commodity); use Network Insights as a free correctness cross-check.

---

## 4. Enterprise-grade gap checklist

| # | Capability | antr today | Closed by |
|---|---|---|---|
| C-1 | Multi-subscription / management-group scope | ❌ single sub (RC-1) | CloudNetDraw + scope change |
| C-2 | Hub-spoke + multi-hub auto-detection, semantic zones | ❌ flat grid | CloudNetDraw |
| C-3 | Real layout engine — readable at 100s of nodes; HLD/MLD/LLD | ❌ hand-placed | ELK via D2 |
| C-4 | External boundary (Internet, ER, VPN GW, NAT, public IP) | ❌ no node type (RC-3) | CloudNetDraw + new node types |
| C-5 | Spoke-to-spoke + cross-sub + transitive peering edges | ❌ dangling (RC-1/RC-2) | CloudNetDraw |
| C-6 | **Severity overlay painted on the diagram** | ✅ engine exists, not wired (RC-4) | **antr — this phase** |
| C-7 | Levels of detail: HLD / MLD / LLD | partial (LLD only) | CloudNetDraw + antr |
| C-8 | Interactivity (zoom/filter/drill/portal links) | ❌ static PDF | web canvas (D2/Cytoscape) |
| C-9 | Auto-refresh pipeline + version diff (Confluence/ServiceNow) | ❌ manual | Azure Function + publish |
| C-10 | Azure official icons + branding | partial | CloudNetDraw |

C-6 is the only row antr owns — and it is the reason the tool should exist.

---

## 5. Target component model

| Component | Tech | Source of truth | Notes |
|---|---|---|---|
| Discovery | Azure Resource Graph (KQL), Network Watcher | live Azure (read-only) | management-group scope; SP/MI with Reader |
| Layout + base diagram | CloudNetDraw (forked) + ELK | discovered JSON | HLD + MLD; hub-spoke + multi-hub; boundary objects |
| Risk engine | `engine/go/internal/analyze` (`Analyze()`) | `graph.Fixture` | **unchanged** — proven Phase 0/1 core |
| Merge / overlay | new Go or Python step | resource ID join | finding → node fill + badge; HLD severity rollup |
| Export | drawio (Confluence), SVG/PNG (reports), web | overlaid diagram | drawio remains the Confluence target |
| Pipeline | Azure Function (timer) | — | publish + version history/diff |

---

## 6. Implementation steps (detail lives in `IMPLEMENTATION_PLAYBOOK.md` § Phase 4)

- **4.1** Validate & decide — pilot CloudNetDraw + Network Watcher Topology on the real
  subscription(s) behind `generated_antr.pdf`; compare to BCLM; confirm multi-sub edges. Gate:
  adopt/fork vs. port layout into Go.
- **4.2** Visualization model design review (this doc) — rubber-duck before integration.
- **4.3** Multi-subscription discovery — management-group scope; **Managed Identity / OIDC**,
  not CloudNetDraw's shipped `AZURE_CLIENT_SECRET` env-var path (A-05).
- **4.4** Severity overlay — join `Analyze()` findings to diagram nodes (fill + badge);
  fixes RC-2 + RC-4 together. Highest-leverage step.
- **4.5** Readability + boundary — ELK/D2 layout; Internet/ER/VPN GW/NAT/public-IP node types (RC-3); HLD/MLD/LLD toggle.
- **4.6** Pipeline — Azure Function publishes to Confluence on schedule with version diff.
- **4.7** Phase 4 acceptance review — gates G1–G5 (see exit criteria).

---

## 7. Locked decisions (Phase 4)

| # | Decision | Rationale |
|---|---|---|
| P4-D1 | Fork + **vendor** CloudNetDraw; do not pin upstream | Single-maintainer project (~137★); own the source, keep MIT attribution |
| P4-D2 | `Analyze()` engine is **unchanged** | Proven Phase 0/1 core; severity stays deterministic, never the renderer |
| P4-D3 | Discovery auth = **Managed Identity / OIDC**, Reader scope | AaraMinds standard (A-03, A-05); override CloudNetDraw's client-secret default |
| P4-D4 | drawio stays the Confluence export target | Existing tWiki pipeline already round-trips drawio cleanly |
| P4-D5 | Severity coloring is computed by `Analyze()`, applied at merge — never by CloudNetDraw | Keeps the LLM-free, deterministic severity boundary intact |
| P4-D6 | Cartography / attack-path graph deferred to Phase 5 | Keep Phase 4 scoped to the diagram + overlay |

---

## 8. [VERIFY] items (Phase 4 — unconfirmed)

| ID | Item | Blocking? |
|---|---|---|
| V4-01 | Management-group / multi-sub Reader scope available for discovery SP/MI | Multi-sub discovery |
| V4-02 | CloudNetDraw cross-sub peering output reconciles with `analyze` resource IDs | Severity overlay join |
| V4-03 | Confluence (tWiki) import accepts CloudNetDraw drawio without manual fixup | Pipeline |
| V4-04 | ELK/D2 vs. CloudNetDraw native layout — which is authoritative for final render | Readability step |
| V4-05 | Network Watcher Topology 30h Resource Graph lag acceptable for cross-check cadence | Ground-truth use |
| V4-06 | ~~License review~~ — **RESOLVED 2026-06-15.** Internal-AT&T-only use; software is never distributed externally. No copyleft trigger (MPL-2.0 / EPL are *distribution*-scoped, file-level weak copyleft); MIT/Apache impose attribution only. Remaining action is routine OSPO intake registration (PA-14), not a gate. | Not blocking |
| V4-07 | Resource-id keying (same-named resources across subscriptions) — **RESOLVED in the Python reference + viz (2026-06-15):** key by `rid()` = ARM id ‖ name; golden 5/5 + `test_resource_id.py` pass. **Go portion (V4-07-Go) pending** the toolchain: add `ID` to NIC/PIP structs, populate in adapter, key `nics` map + NW lookups by id — verified by `engine-ci.yml` + `twin_drift_check.py`. | Go portion pending |

---

## 9. Exit criteria (Phase 4 acceptance gates)

| Gate | Criterion |
|---|---|
| G1 | Multi-subscription discovery draws cross-sub + spoke-to-spoke peering edges that BCLM has and `generated_antr.pdf` lacked (RC-1/RC-2 retired) |
| G2 | External boundary nodes (Internet, ER, VPN GW, NAT, public IP) render where present (RC-3 retired) |
| G3 | `Analyze()` findings paint node severity on the diagram; legend is accurate, not decorative (RC-4 retired) |
| G4 | Severity is computed only by `Analyze()` — diagram tool never assigns it (P4-D5 held) |
| G5 | Discovery uses Managed Identity / OIDC read-only — no `AZURE_CLIENT_SECRET` (A-05 held) |

---

*Phase 4 turns antr from "an inventory in boxes" into "an enterprise topology with risk painted
on it" by owning the one layer that is actually defensible and adopting OSS for the rest.*

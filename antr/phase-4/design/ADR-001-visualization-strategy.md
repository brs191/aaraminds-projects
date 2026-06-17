# ADR-001 — Visualization strategy: own the truth, delegate the geometry, ship view families

**Status:** ACCEPTED (2026-06-17) · **Supersedes:** the "adopt CloudNetDraw as the base diagrammer" framing in `VISUALIZATION_MODEL.md` §Verdict
**Reads with:** `GRAPH_IR.md`, `COMPETITIVE_ANALYSIS.md`

---

## Context

Phase 4 began from a real failure (`ref-topology/generated_antr.pdf`: zero
connectivity, everything "Clean") and a sound instinct — *stop hand-rolling the
diagram brain; the picture is commoditised, the risk engine is not.* The original
plan was to adopt **CloudNetDraw** (MIT, Azure-native, hub-spoke `.drawio`) as the
base map and paint antr severity on top.

Two things were then established that sharpen that plan:

1. **antr already owns discovery.** The Azure adapter → `graph.Fixture` path was
   hardened across two audit rounds (ARG pagination, cross-sub peerings,
   multi-firewall, WAF policy state, LB-NAT object shapes). It is the source of
   topology truth. Adopting CloudNetDraw's *discovery* would duplicate it and
   reintroduce a divergence problem one layer below the engine.

2. **A source read of CloudNetDraw** (`azure_client.py`, `diagram_generator.py`,
   `layout.py`, `utils.py`) showed: license is **MIT** (clean for internal use,
   no copyleft anywhere in the direct dep chain); rendering is **structurally
   deterministic** (static `<mxfile>` header — no timestamp/etag/UUID — content-
   derived cell ids, rule-based layout, sorted-tuple dedup) **except for one gap**:
   it emits the VNet list in Azure-API order with **no sort before emit**, so
   geometry can vary run-to-run on an unchanged estate. Its discovery is also naive
   (per-VNet SDK round-trips; a leftover debug query with a hardcoded subscription
   GUID).

## Decision

**antr owns discovery and risk; it delegates only layout geometry; it ships a set
of view families now; and it keeps a stable graph IR so the layout engine is
swappable.** Concretely:

1. **Discovery stays in antr.** The hardened adapter is the single source of
   topology truth. We do not adopt CloudNetDraw's discovery.

2. **Risk stays in antr.** `Analyze()` computes severity once on the full estate;
   the overlay is the only source of node colour. A view can hide a resource but
   never change a verdict.

3. **Ship view families (Strategy 3) — DONE in this change.** `views.py` emits
   `hld`, `mld`, `risk`, `boundary`, `cross-sub`, and one `finding/<n>` per
   Critical/High finding, each a deterministic projection over the IR rendered by
   the existing `render_drawio`. Gated by `test_views.py` + `eval_diagram.py`.
   Rationale: one canvas of a real estate is unreadable; Azure's own Network
   Watcher and the overview-plus-detail literature both treat this as a
   filter/navigation problem.

4. **Keep a stable graph IR (Strategy 2 escape hatch) — DONE.** `GRAPH_IR.md`
   pins `graph.Fixture` + the overlay as the contract any layout backend consumes.
   The geometry is replaceable; identity, determinism, discovery, and risk are not.

5. **CloudNetDraw is a LAYOUT-only option, adopt-and-patch, not now.** If/when the
   in-house renderer hits a legibility wall on large estates, evaluate a
   **layout-only fork** of CloudNetDraw fed by our IR (not its discovery). Adoption
   is conditional on a one-line determinism patch — sort `vnet_candidates` by
   `resource_id` (and subnets) before emit in `azure_client.py` — after which its
   output can sit behind `GRAPH_IR.md`'s determinism rule and our byte-identical
   gate. This is real fork ownership, so it is a fallback, not the primary path.

6. **If a layout engine is swapped, prefer a deterministic one.** ELK or Graphviz
   `dot` (layered, deterministic given fixed input + pinned version) over
   force-directed `sfdp` (seed-sensitive) — determinism is the product, not a
   nicety.

## Consequences

* **Positive:** the readable-diagram problem is addressed now (view families) on
  code that is already deterministic and gated; no new third-party runtime
  dependency, no fork to maintain, no second discovery path. The IR contract means
  a future layout swap is a bounded, well-specified task.
* **Negative / accepted:** the in-house renderer's Azure-specific layout polish
  (icon conventions, hub-spoke aesthetics) is less mature than CloudNetDraw's. We
  accept that until a concrete legibility complaint on a real estate justifies the
  fork-and-patch in (5).
* **Follow-ups (RESOLVED 2026-06-17):** the two limitations this ADR originally
  recorded are now closed. (a) App Gateway, AKS, Front Door, vWAN hub, APIM, and
  Private Endpoint are drawn as **first-class nodes** in the renderer's "application
  & edge services" band, painted by overlay severity; `overlay.finding_node_ids`
  maps every one of their finding families to a node, and the diagram-eval gate now
  **positively enforces** that each app-layer finding's node is drawn and correctly
  coloured (`RC4`). `non_topology_findings` shrank to a single, justified residue:
  the cross-subscription peering *relationship*, which is rendered as the cross-sub
  edge, not a node. (b) finding-centric views now centre on **any** Critical/High
  finding — NIC or app-layer — plus internet-facing Mediums (WAF-disabled public App
  Gateway / Front Door). Remaining genuine residue: cross-sub peering stays an edge
  (correct); Load Balancers are not drawn (their NAT exposure already lands on the
  backend NIC node).

## Alternatives considered

* **Adopt CloudNetDraw as the base (discovery + layout).** Rejected: duplicates
  hardened discovery, introduces a second ARG path that can disagree with the
  engine, and inherits a non-deterministic emit that breaks the byte-identical gate
  until patched. The determinism patch is trivial; the duplicate-discovery and
  fork-maintenance costs are not.
* **Graph IR + ELK/Graphviz now (Strategy 2 as primary).** Deferred: correct
  long-term separation, but a larger build than needed today. We took the cheap,
  decisive half (pin the IR) and kept the layout swap as future work behind the
  contract.
* **One universal diagram, better laid out.** Rejected: layout quality does not
  fix the soup problem at estate scale; filtering does.

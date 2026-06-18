# Full-Estate (BCLM-parity) View — Build Requirements

**For:** an external build (e.g. GitHub Copilot). **Companion to:** `APPLICATION_VIEW_REQUIREMENTS.md`,
`ADR-001-visualization-strategy.md`, `GRAPH_IR.md`. **Status:** requirements, not yet built.

Goal: auto-generate a comprehensive, single-canvas estate diagram at the **coverage and
structure** of the hand-drawn reference `ref-topology/BCLM-Revised-8June2026.drawio` —
network plumbing + workloads + data + boundary, on one map, severity-annotated.

> **Read this first — the anti-goal.** Do **not** try to pixel-reproduce BCLM. It is a
> hand-drawn artifact; matching its exact layout is neither achievable nor valuable, and
> "generated output ≠ the hand drawing" was the *false* failure that triggered Phase 4. The
> target is **resource coverage + auto-generation + deterministic risk colouring** — the
> things a hand drawing can't do. Parity means "covers the same families and structure,"
> not "looks identical."

---

## 1. What BCLM actually contains (measured)

`BCLM-Revised-8June2026.drawio` = **997 vertices, 288 edges**, Azure stencil icons. Family
mix (label/style counts): AKS/Kubernetes ~98, subnets 74, NAT 55, DNS 49, VNets 23, SQL 21,
NSG 13, Storage/Redis/Service Bus/Key Vault (data tier), Bastion 10, APIM 10, Internet
boundary, peerings, Private Endpoints, gateways (VPN/App GW), firewall. In short: a full
**network topology** *plus* the **workload and data tiers**, with an Internet boundary.

## 2. What antr already provides (REUSE — do not rebuild)

| BCLM element | antr today |
|---|---|
| Hub-spoke VNets · subnets · NSGs · peerings · Internet/firewall/gateway/NAT boundary band | `render_drawio.py` MLD/network view + boundary band |
| App + ingress tiers (AKS, App GW, APIM, Front Door, LB, Bastion, PE) | app-layer band + the **App View** (`APPLICATION_VIEW_REQUIREMENTS.md`) |
| Severity colouring of every node | the `overlay` (engine `Analyze()` output) |
| Cross-subscription peering | cross-sub edges / view |

This spec covers only the **gaps** below, then a composition that merges everything.

---

## 3. Gaps to build

### G1 — Data-tier discovery + nodes (the biggest; needs new discovery)

Today antr models data only *implicitly* via Private Endpoint `groupId`. BCLM draws the
**actual** PaaS data resources. Add them as first-class data-tier nodes.

**G1.1 Discovery (new ARG queries).** Query these resource types (Azure Resource Graph
`type ==`), project `name, resourceGroup, subscriptionId, id, kind` + the fields noted:

| Data service | ARG `type` | Extra fields |
|---|---|---|
| Azure SQL server | `microsoft.sql/servers` | `properties.publicNetworkAccess` |
| Storage account | `microsoft.storage/storageaccounts` | `properties.publicNetworkAccess`, `properties.networkAcls.defaultAction` |
| Cosmos DB | `microsoft.documentdb/databaseaccounts` | `properties.publicNetworkAccess` |
| Redis | `microsoft.cache/redis` | `properties.publicNetworkAccess` |
| Service Bus | `microsoft.servicebus/namespaces` | `properties.publicNetworkAccess` |
| Key Vault | `microsoft.keyvault/vaults` | `properties.publicNetworkAccess`, `properties.networkAcls.defaultAction` |

Follow the existing adapter conventions: paginated `runKQL` (SkipToken), parse helpers,
`subscriptionId == %q` scoping. Add to the parallel fan-out in `fetchResourceGraph`.

**G1.2 Model.** Add a `DataService` type to `graph.ResourceGraph`:
```
DataService { name; kind ("sql"|"storage"|"cosmos"|"redis"|"servicebus"|"keyvault");
              id; resourceGroup; publicNetworkAccess ("Enabled"|"Disabled");
              tags map[string]string }
```
Mirror it in the Python reference twin (so twin-drift stays valid).

**G1.3 Linkage (the key relationship).** A Private Endpoint's `privateLinkServiceId`
(already in the model) is the ARM id of its target data service. Match
`privateEndpoints[].privateLinkServiceId` → `DataService.id` to draw **PE → data service**
edges. This is what connects the app chain to the real data store.

**G1.4 Render.** Draw each `DataService` as a data-tier node with a kind-appropriate label
(SQL / Storage / Redis / …). Colour: structural (grey) unless the engine scores it (see
§6 — `publicNetworkAccess == "Enabled"` is a natural future finding family; until that
exists, draw structural and DO NOT invent a colour).

### G2 — DNS as drawn nodes

BCLM shows DNS heavily (49). antr models `privateDnsZones` (for the DNS-misconfig finding)
but doesn't draw them.

- Draw each `privateDnsZones[]` as a node; draw an edge zone → each VNet in its
  `linkedVnets[]`. (Data already in the model.)
- Optional discovery: DNS Private Resolver (`microsoft.network/dnsresolvers`) as a node.
- Keep the existing "private DNS zone not linked / missing" findings painting the PE — DNS
  nodes are context, the verdict still lands on the PE.

### G3 — NAT gateway as first-class nodes

antr discovers `natGateways` but draws them only in the boundary band. For estate parity,
draw each NAT gateway as a node and associate it to the subnet(s) it serves. (Needs the
adapter to project the NAT gateway → subnet association if not already captured; otherwise
draw it in the boundary band linked to its VNet.)

### G4 — Azure stencil iconography (visual fidelity — optional)

BCLM uses `mxgraph.azure` stencils; antr uses coloured rectangles + emoji badges. For a
BCLM-*look*, map each node kind to its draw.io Azure shape style, e.g.:
```
appgw  → shape=mxgraph.azure.application_gateway
aks    → shape=mxgraph.mscae.kubernetes... (or azure compute)
sql    → shape=mxgraph.azure.sql_database
storage→ shape=mxgraph.azure.storage_blob
fw     → shape=mxgraph.azure.firewall
...
```
Severity is then conveyed by the node's **stroke/fill overlay or a badge**, not the whole
fill (so the engine verdict still reads). This is presentation-only; gate it behind a
`--icons` flag so the plain, review-friendly style stays the default.

### G5 — Merged "estate" composition (the BCLM-parity view itself)

A new view that lays everything on one canvas, in BCLM's structure:

```
[ Internet boundary band ]              entry/edge: Front Door, Public IPs, ER/VPN GW, Firewall, NAT
        │
[ Hub VNet zone ]  ── peering ──  [ Spoke VNet zones ... ]
   subnets → NICs/AKS (workloads)              subnets → NICs/AKS
        │                                          │
[ DNS zones ]                              [ Private Endpoints ] ── privateLinkServiceId ──▶ [ Data services: SQL/Storage/Redis/... ]
```

Requirements:
- Containers/zones: a band per VNet (hub first, by peering degree), subnets nested inside,
  workloads inside subnets — reuse the MLD nesting. NAT/DNS/boundary in their bands.
- Every node coloured by the `overlay` severity (engine verdict), exactly as the other views.
- All edge classes drawn: peerings, exposure paths (internet→exposed NIC), firewall DNAT,
  PE→data-service, DNS zone→VNet, App GW/LB→backend. De-duplicated, no dangling.
- This is the "show everything" view; pair it with the existing **filtered** views (risk,
  boundary, app, finding) for review — one giant canvas is for inventory, the filtered
  views are for decisions (per ADR-001).

---

## 4. Inputs / output / principles

- **Inputs:** the `Fixture` JSON (extended with `DataService` per G1) + the `overlay`.
- **Output:** `.drawio` (plain `mxGraph`, static header). `--icons` toggles G4.
- **Principles (unchanged, non-negotiable):** the engine owns severity/reachability — the
  view never recomputes; deterministic (sort before emit, counter ids, no timestamps/UUIDs);
  honest inference (mark inferred edges; never fabricate).

## 5. Determinism & invariants

Globally-unique cell ids (assert); no dangling edges (assert); byte-identical re-render.
Sort every family list and every edge list before emit. (Same gates as `eval_diagram.py` /
the render-determinism CI step; extend both to the new view.)

## 6. Engine opportunity (out of scope here, noted)

`publicNetworkAccess == "Enabled"` on a data service that also has a Private Endpoint is a
real exposure (the PE suggests it *should* be private). That's a natural **new finding
family** ("data service public network access enabled") for the engine — Go + Python twin +
fixtures — which would then colour the data nodes via the overlay automatically. Build it in
the engine (not the renderer) if you want the data tier risk-coloured. Keep it a separate
work item; this spec only requires *drawing* the data services.

## 7. Acceptance

Build `estate-full.json` covering: hub + ≥2 spokes, subnets, NICs, AKS, App GW, Front Door,
NAT, ≥2 private DNS zones with VNet links, Private Endpoints whose `privateLinkServiceId`
targets discovered SQL + Storage + Redis. Assert:

1. **Coverage:** the merged view draws all families above (network + app + data + DNS + NAT +
   boundary) — none silently dropped.
2. **Data linkage:** every PE connects to its data service via `privateLinkServiceId`; the
   data service node is present.
3. **DNS linkage:** each DNS zone connects to its `linkedVnets`.
4. **Severity is the engine's:** node colours equal the overlay; undiscovered/unscored data
   services are structural (no invented colour).
5. **All edge classes present, none dangling; ids unique.**
6. **Determinism:** byte-identical re-render.
7. **Twin-drift still 0** after the `DataService` model addition (Go ≡ Python).

## 8. Sequencing

1. **G1 data discovery + model + twin** (foundation — unlocks the data tier; biggest effort,
   touches the Go adapter + Python twin + a new ARG query set + the recorded-ARM harness).
2. **G5 merged composition** consuming G1 (+ existing MLD/app pieces).
3. **G2 DNS nodes**, **G3 NAT nodes** (cheap, data mostly present).
4. **G4 iconography** (presentation polish, flagged off by default).
5. **G6** (optional engine finding for data public-access) — separate work item.

> Effort note: G1 is the only part that needs new **discovery** (ARG queries + adapter +
> twin + harness) — treat it like the F-series adapter work (recorded-ARM test first). G2–G5
> are renderer/projection work over data that's largely already in the IR.

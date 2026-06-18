# Application View — Build Requirements (v1)

**For:** an external build (e.g. GitHub Copilot). **Design rationale:** `ADR-002-application-view.md`.
**Status:** requirements, not yet built.

This spec is self-contained: it states the inputs, the data shapes, the rules, the
output, and the acceptance tests for the **Application (dependency) view** — an
app-owner projection of an Azure estate (Users → Front Door → App Gateway → AKS →
Private Endpoint → SQL), painted with the engine's severity and reachability verdict.

---

## 0. Non-negotiable principles

1. **The engine owns the verdict.** This view NEVER computes reachability or severity.
   It reads them from the existing analysis output (the "overlay", below) and only adds
   grouping, tiering, and dependency edges. (Same rule as every other antr view.)
2. **Deterministic.** Same input → byte-identical output. Sort every collection before
   emitting; derive element ids from content or a monotonic counter; no timestamps, no
   UUIDs, no hash-set iteration order, no wall-clock.
3. **Honest inference.** When membership or an edge is inferred (not read from an explicit
   Azure relationship), mark it as inferred. Never fabricate an assignment; fall back to
   `unassigned` / `shared`.

---

## 1. Inputs

### 1.1 The topology (a `Fixture` JSON object)

The view consumes one JSON object (`resourceGraph` is the relevant part). Fields it reads,
by resource family:

| Family | JSON path | Fields used |
|---|---|---|
| Network interface | `resourceGraph.networkInterfaces[]` | `name`, `id` (optional), `subnet` (`"{vnet}/{subnet}"`), `privateIp`, `tags` (`map[string]string`) |
| AKS cluster | `resourceGraph.aksClusters[]` | `name`, `subnet`, `tags`* |
| Application Gateway | `resourceGraph.applicationGateways[]` | `name`, `subnet`, `backendPools[].targets[]` (private IPs/FQDNs), `tags`* |
| Load Balancer | `resourceGraph.loadBalancers[]` | `name`, `isInternal`, `backendPools[].nicRefs[]` (NIC names), `tags`* |
| Private Endpoint | `resourceGraph.privateEndpoints[]` | `name`, `subnet`, `groupId` (e.g. `blob`/`sql`/`vault`), `tags`* |
| Front Door | `resourceGraph.azureFrontDoors[]` | `name`, `tags`* |
| API Management | `resourceGraph.apiManagements[]` | `name`, `vnetMode` (`External`/`Internal`/`None`), `tags`* |

`*` = `tags` is present on NICs today; on the other families it is the **v2 enrichment**
(§7). For v1, read `tags` if present (fixtures carry it); otherwise rely on propagation
(§4) and the `unassigned` fallback.

### 1.2 The severity overlay (reuse, do not recompute)

antr already computes `overlay = { "<kind>:<name>": { severity, bucket, findings[] } }`
from `Analyze()`. The view must call that existing function and use its output. Node id
scheme (these strings are the join keys — they MUST match):

```
nic:<rid>     appgw:<name>   aks:<name>   pe:<name>
fd:<name>     apim:<name>    lb:<name>    pip:<rid>
```

`rid` = the NIC's `id` if present, else its `name`. `severity` ∈ {Critical, High, Medium,
Low, Informational, Clean}. A finding with `reachable: true` means the engine proved an
internet-reachable path to that resource.

### 1.3 Config

```
app_tag   = "application"   # tag key used for membership (default; allow override)
only_app  = <name|null>     # optional: render just one application
```

---

## 2. Output

- **Primary:** a draw.io (mxGraph) XML diagram (`.drawio`). Mirror antr's existing
  serializer: an `<mxfile>` with a static header (no `modified`/`etag`/timestamp), cells
  with `mxGeometry`, edges as `<mxCell edge="1">`.
- Later (not v1): `svg`, `mermaid`.

---

## 3. Tier classification (R1)

Every drawn resource maps to exactly one tier by family + a small rule. Fixed order
left→right: **entry → ingress → compute → data**.

| Tier | Members |
|---|---|
| entry | Front Door; APIM when `vnetMode != "Internal"`; (optional) `Internet` node |
| ingress | Application Gateway; Load Balancer when `isInternal == false`; APIM when `vnetMode == "Internal"` |
| compute | AKS; every NIC |
| data | Private Endpoint **and** a logical backing-service node derived from its `groupId` |

Shared/cross-cutting controls (firewall, NSG, DNS, gateways) are **out of scope** for the
app view — they belong to the network/boundary views.

---

## 4. Membership (R3)

Goal: assign every node an `app` label.

1. **Seed** from tags: `app = resource.tags[app_tag]` when present.
2. **Propagate** across dependency edges (§5) to a fixpoint: if one endpoint of an edge has
   an `app` and the other does not, copy it. Repeat until no change. (This lets a tagged
   compute node pull its fronting ingress / entry and its data endpoint into the same app.)
3. **Fallbacks:** a node still unlabeled after propagation → `app = "unassigned"`. A node
   reachable from ≥2 distinct apps → `app = "shared"` (do not duplicate it per app).
4. Determinism: when seeding/propagating, iterate nodes and edges in sorted-id order.

---

## 5. Dependency edges (R4)

Each edge is `(src_id, dst_id, inferred: bool)`.

**Explicit edges (inferred = false) — from real IR relationships:**

- App Gateway → compute: for each `applicationGateways[].backendPools[].targets[]`, if a
  target equals a NIC's `privateIp`, add `appgw:<gw> → nic:<rid>`.
- Load Balancer → compute: for each `loadBalancers[].backendPools[].nicRefs[]` (and
  `isInternal == false`), if the NIC name exists, add `lb:<lb> → nic:<rid>`.
- Private Endpoint → data service: add `pe:<pe> → data:<groupId>:<pe>`.

**Inferred edges (inferred = true) — canonical tier-order within an application:**

- entry → ingress, and ingress → compute: for each app, connect every node in the lower
  tier to every node in the next tier **of the same app** (fills Front Door→App GW,
  App GW→AKS where no explicit backend edge exists).
- compute → Private Endpoint: connect a NIC/AKS to a PE when they share the same `app`
  **or** the same VNet (`subnet` prefix before `/`).

De-duplicate edges; skip self-edges; skip edges whose endpoints aren't drawn.

> v1 honest gap: Front Door/APIM → origin and App GW → AKS are **inferred** (the IR has no
> explicit origin/backend link for them) — hence dashed. Explicit Front Door origin
> discovery is a v2 item.

---

## 6. Severity + reachability styling (R5, R6)

- **Node fill** = the overlay severity colour for that node id (reuse antr's palette:
  Critical=red, High=orange, Medium=yellow, Info=blue, Clean=green). The logical data node
  (`data:*`) has **no** severity — draw it structural/dashed (grey), labelled "via PE".
- **Edge style** (priority order):
  1. If the **target** node has a finding with `reachable: true` → **exposed path**: solid
     red, thicker. (This is the engine's verdict surfaced on the chain — the view's reason
     to exist.)
  2. Else if the edge is explicit → solid neutral.
  3. Else (inferred) → dashed grey.

> v1 scope of the reachability claim: antr's engine computes *internet* reachability and the
> finding families, not arbitrary point-to-point internal reachability. So only edges whose
> target is internet-reachable are asserted "exposed". Internal tier-to-tier edges are drawn
> as dependencies, NOT claimed as verified-reachable. State this in the legend; do not
> overclaim. (A future engine extension could add point-to-point reachability.)

---

## 7. Layout (R7)

- Four tier columns at fixed x-offsets (entry, ingress, compute, data).
- Group nodes into **application swimlanes** (one labelled horizontal band per app, sorted
  by app name; `unassigned` and `shared` last). Within a lane, place each node in its tier
  column; stack multiple nodes in the same tier vertically (sorted by id).
- A header row labels the four tier columns. A legend explains the colours and the
  solid/dashed/red edge meanings.

---

## 8. Determinism & invariants (R8, R9)

- All cell ids globally unique across nodes and edges (assert). No dangling edges: every
  edge `source`/`target` must be a drawn node id (assert).
- Re-rendering the same fixture must produce byte-identical XML.
- Edge ids: a monotonic counter (`ae1`, `ae2`, …). Node ids: the `<kind>:<name>` scheme.

---

## 9. Acceptance tests

Build a fixture `estate-app-chain.json` with `application` tags forming one app
("customer-portal"): a Front Door, an App Gateway (public, backend target = a NIC's
`privateIp`), an AKS, NICs, a Private Endpoint (`groupId: "sql"`), plus a second untagged
spoke. Assert:

1. **Chain renders:** the view contains `fd → appgw → (nic|aks) → pe → data:sql` for
   customer-portal, with the tiers in order.
2. **Membership:** every customer-portal-tagged node is in the `customer-portal` lane;
   untagged-and-unconnected resources land in `unassigned`; a resource fronting two apps
   lands in `shared`.
3. **Severity is the engine's:** node fills equal the overlay severity for their ids (paint
   nothing the engine didn't score).
4. **Reachability annotation:** an edge whose target has a `reachable:true` finding is the
   red "exposed" style; others are not.
5. **Determinism:** two renders are byte-identical.
6. **Invariants:** unique ids, no dangling edges.
7. **Untagged estate degrades gracefully:** a fixture with no `application` tags renders one
   `unassigned` application without error.

Wire it into the existing view gates (the equivalent of `test_views.py` /
`eval_diagram.py`) so it stays green.

---

## 10. Suggested module shape (mirror antr's existing viz)

```
app_view.py
  build_model(fixture, app_tag) -> (nodes: dict[id]->meta, edges: list[(src,dst,inferred)])
      # §3 tiering · §4 membership+propagation · §5 edges · reads overlay for sev/reachable
  render(fixture, app_tag, only_app, title) -> (xml, cells, edges)
      # §6 styling · §7 layout · §8 invariants ; reuse the existing overlay + drawio serializer
  main()  # CLI: app_view.py <fixture.json> [--out FILE] [--app NAME] [--tag KEY]
```

Reuse, don't reinvent: the **overlay** (severity per node id), the **drawio serializer**
(static-header XML), the **node-id scheme**, and the **determinism + invariant asserts**
already used by the other views.

---

## 11. v2 (later — needs discovery changes)

- **Project `tags` onto every resource family** in the adapter/model (today only NICs carry
  tags) → makes membership *explicit* instead of propagated, on live data.
- **Front Door / APIM origin discovery** → turns the top edge from inferred to explicit.
- **Point-to-point internal reachability** (engine extension) → lets every internal edge
  carry a real allowed/blocked verdict, not just the internet-exposure ones.
- **`svg` / `mermaid` outputs** behind the same model.

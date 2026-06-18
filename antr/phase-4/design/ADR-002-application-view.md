# ADR-002 — The Application (dependency) view

**Status:** PROPOSED (design — 2026-06-17) · **Reads with:** `ADR-001-visualization-strategy.md`, `GRAPH_IR.md`, `../../docs/INSTRUCTING_ANTR.md`

---

## Context & goal

Today antr produces *network-shaped* views (HLD/MLD/risk/boundary/cross-sub/finding):
the unit is the VNet → subnet → NIC. But an **application owner does not think in NICs,
NSGs, and subnets** — they think in a request chain:

```
Users → Front Door → App Gateway → AKS → Private Endpoint → SQL
```

The **Application view** renders that chain: each application's resources arranged into
logical tiers, with the dependency edges between them. Its unique antr twist — the thing
no architecture-diagram tool ships — is that every dependency edge is **annotated with the
engine's reachability verdict**: not "App GW talks to AKS" (drawn on a napkin) but "App GW
→ AKS *reachable*" or "*blocked by NSG*". It is a **verified** application dependency graph.

## Principles (non-negotiable, inherited from ADR-001 / GRAPH_IR)

1. **The engine still owns the verdict.** The Application view computes *grouping, tiering,
   and dependency edges* — never reachability or severity. Node colour and edge
   reachability come from `Analyze()` / the overlay, unchanged. This stays a projection
   plus a deterministic enrichment, not a second analysis path.
2. **Deterministic.** Membership keys sorted; tier order fixed; edges derived from sorted
   relationship lists. No wall-clock, no LLM, no heuristics that vary run-to-run.
3. **Honest about inference.** Where membership or an edge is *inferred* (not read from an
   explicit Azure relationship), the node/edge is marked inferred; ambiguity falls back to
   an explicit "unassigned" / "shared" group, never a fabricated guess.

## The logical model

### Tiers (fixed order, rule-based classification)

| Tier | Resource families | Source field |
|---|---|---|
| **Entry** | Internet · Public IP · Front Door | boundary band / `azureFrontDoors` |
| **Ingress (L7)** | Application Gateway · public Load Balancer · APIM | `applicationGateways` · `loadBalancers` (public) · `apiManagements` |
| **Compute** | AKS · VM/VMSS (NICs) · App Service · Functions | `aksClusters` · `networkInterfaces` |
| **Data** | Private Endpoint → target service (blob/sql/vault/…) | `privateEndpoints.groupId` / `privateLinkServiceId` |
| **Shared / controls** | Azure Firewall · Bastion · DNS · AVNM · gateways | drawn once, not per-app |

Classification is a pure `resource-family → tier` map; no per-resource judgement.

### Dependency edges — from data antr ALREADY has

The IR already carries the relationships needed for most of the chain (no new discovery):

| Edge | Derived from |
|---|---|
| App Gateway → Compute | `ApplicationGateway.BackendPools.Targets` (private IP/FQDN) matched to `NIC.PrivateIP` |
| Load Balancer → Compute | `LoadBalancer.BackendPools.NicRefs` (NIC names) |
| Compute → Data (PE) | a workload NIC and a Private Endpoint in the same VNet → the workload reaches the data service via that PE (inferred by VNet/app membership) |
| Private Endpoint → Data service | `PrivateEndpoint.GroupId` (blob/sql/vault/…) → a **logical** data node (the backing SQL/Storage is usually not a network resource antr discovers) |
| Front Door / APIM → Ingress | **not in the model today** — `AzureFrontDoor` has no origin field. Inferred by shared application membership; marked inferred until origin discovery is added |

Every resolved edge is then asked of the engine: is this path *reachable* (NSG effective
rules + routes + firewall)? The answer styles the edge (solid = reachable, dashed-red =
blocked / latent). That reachability annotation is the view's reason to exist.

### Application membership — two tiers, honest about each

- **v1 (no new discovery): propagate from NIC tags.** antr already reads `NIC.Tags`. A NIC
  tagged `application=customer-portal` seeds membership; it then **propagates along the
  dependency edges** — the App Gateway whose backend pool contains that NIC inherits
  `customer-portal`; the Private Endpoint its workloads use inherits it; Front Door above
  the ingress inherits it. Resources reached by no tagged workload land in **"unassigned"**;
  resources shared across apps (firewall, DNS) land in **"shared"**. This works on today's IR.
- **v2 (enrichment): project `tags` onto every resource family.** The adapter currently
  carries tags only on NICs. Projecting the ARG `tags` column onto App Gateway, AKS, PE,
  Front Door, APIM, etc. makes membership *explicit* (read, not propagated) and removes the
  v1 ambiguity. This is the one genuine discovery/model change — additive, behind the same
  GRAPH_IR contract.

## Input contract additions

```yaml
scope:
  application_name: customer-portal    # filter to one app (by membership)
view:
  - application
config:
  membership: tag:application           # tag key (default) | tag:app | resource-group
```

`application_name` filters to one app's chain; omitting it renders every discovered app as
its own lane. `membership` chooses the grouping key. Defaults keep single-app, untagged
estates working (everything → one "unassigned" application).

## Determinism

Group keys and tier members sorted; tiers in fixed order; edges emitted from sorted
relationship lists with counter ids (as render_drawio already does). The view is a pure
function of (fixture, overlay) → SVG/drawio — byte-identical on re-render, gated like the
existing views (`test_views.py`, `eval_diagram.py`).

## Honest limits (what design cannot wish away)

- **Front Door / APIM → origin** is not discoverable from the current IR; that top edge is
  inferred (marked) until origin/endpoint-origin discovery is added.
- **The data node is logical.** PE `groupId` tells you it's SQL/blob/vault, but the backing
  PaaS resource is usually outside the network graph — drawn as an inferred data node, not a
  discovered one.
- **Untagged estates** collapse to a single "unassigned" application; the view degrades
  gracefully but its value scales with tagging discipline. We surface this, not hide it.
- **Cross-app shared resources** (one firewall fronting three apps) are drawn once in
  "shared," with edges into each app — never duplicated per app.

## Build plan (phased)

1. **`appmodel.py`** (new viz module, sibling of `overlay.py`): classify resources into
   tiers; compute membership (v1 NIC-tag propagation); resolve dependency edges from backend
   pools / PE / NIC matches; annotate each edge with the engine reachability verdict from the
   overlay. Pure, deterministic, unit-tested.
2. **Renderer**: an application-lane layout (Entry → Ingress → Compute → Data columns per
   app) reusing `overlay` severity for node fill and the appmodel for edges. Reuses the
   render invariants (unique ids, no dangling edges).
3. **`views.py`**: add the `application` view (and `application_name` projection). Extend
   `test_views.py` with app-view invariants (every tagged NIC's app present; edges
   reachability-consistent with the overlay; determinism).
4. **Fixtures**: an estate with `application` tags + a real FD→AppGW→AKS→PE→SQL chain.
5. **v2 (later)**: adapter projects `tags` onto all resource families → explicit membership;
   add FD origin discovery to firm up the top edge.

## Acceptance

- `application` view renders the FD→AppGW→AKS→PE→SQL chain for a tagged estate, with each
  edge styled by the engine's reachability verdict.
- Membership propagation is deterministic and falls back to "unassigned"/"shared" — no
  fabricated assignments.
- All existing gates stay green; the new view is added to the views gate.
- The engine's reachability/severity output is **unchanged** — verified by twin-drift and
  the eval gates (the view reads the verdict, never recomputes it).

## Decision

Build the Application view as a **projection + deterministic enrichment** over the existing
IR and overlay, starting with **v1 (NIC-tag membership propagation + existing relationship
edges + reachability annotation)** — which needs no discovery change — and defer the `tags`
projection (v2) and FD-origin discovery as additive follow-ups. This keeps the engine the
sole owner of the verdict, ships value on today's data, and stays inside the GRAPH_IR
contract so the layout backend remains swappable.

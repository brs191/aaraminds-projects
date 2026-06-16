# Graph IR — the stable contract between discovery, risk, and layout

**Project:** Azure Network Topology Reviewer · **Phase:** 4 · **Status:** ACTIVE
**Reads with:** `ADR-001-visualization-strategy.md`, `phase-1/design/TOPOLOGY_MODEL.md`

---

## Why this document exists

antr's visualization is built as a pipeline of four independent stages:

```
discovery            risk                 projection           layout / render
(adapter)        →   (Analyze)        →   (views.py)       →   (render_drawio | future ELK/CloudNetDraw)
graph.Fixture        overlay (severity)   filtered Fixture     .drawio / .svg
```

The thing that lets these stages stay decoupled — and lets the **layout engine be
swapped without touching discovery or risk** — is a single, stable intermediate
representation (IR). This file pins that IR so a future backend (ELK, Graphviz
`dot`, a layout-only CloudNetDraw fork) can be dropped in against a written
contract rather than reverse-engineered from the renderer.

**The IR already exists.** It is the `graph.Fixture` JSON — the same structure the
deterministic engine analyses and the same structure `render_drawio` draws. There
is no new format to invent; this document elevates it to a contract.

## Canonical definition

The authoritative schema is the Go type `graph.Fixture` in
`engine/go/internal/graph/model.go`, mirrored field-for-field by the Python
reference in `engine/reference/analyze.py`. JSON field names (the `json:"..."`
tags) are the contract surface. Top level:

| Field | Meaning | Consumed by |
|---|---|---|
| `subscription` | primary subscription id of the capture | render (labels) |
| `resourceGraph` | the topology: VNets, subnets, NICs, NSGs, route tables, public IPs, peerings, and the Azure-specific families (App GW, AKS, Front Door, vWAN, APIM, Bastion, gateways, ER circuits, NAT GWs, private endpoints, load balancers, private DNS zones) | analyze + render |
| `networkWatcher` | `effectiveSecurityRules`, `effectiveRoutes` (keyed by NIC id/name), `incompleteNics` | analyze |
| `avnm` | `securityAdminRules` (AVNM admin gate) | analyze |
| `azureFirewall` / `azureFirewalls` | single (legacy) + all firewalls; DNAT is evaluated over the union | analyze + render |
| `crossSubscriptionPeerings` | cross-sub peering relationships (local/remote VNet, remote sub id, firewall-in-path) | analyze + render |
| `enrichment` | optional Defender/Policy/Activity context; engine does NOT read it | MCP explainer only |

The HLD subset the earlier design called `topology.graph.json`
(`subscription → resource group → VNet → subnet → resource` + peerings) is exactly
`resourceGraph.virtualNetworks[*]` plus `crossSubscriptionPeerings`. A layout-only
backend needs only that subset plus the overlay (below); it can ignore
`networkWatcher` and `avnm`, which are risk inputs, not layout inputs.

## Identity rule (non-negotiable)

Every resource is identified by **`rid` = ARM resource `id` when present, else
bare `name`**. Bare names are not unique across subscriptions/resource groups, so
multi-subscription captures MUST carry `id`. Render cell ids and overlay keys are
both derived from `rid`, namespaced by kind (`nic:`, `vnet:`, `pip:`), so a NIC and
a public IP that share a name can never mispaint each other. (This is engine
finding V4-07; any new backend must honour it.)

## Determinism rule (non-negotiable)

The whole product rests on byte-identical artifacts. Therefore any producer of the
IR — and any layout backend that consumes it — MUST:

* **sort before emit** (resources and edges in a stable, content-derived order);
* derive ids from content (`rid`, hierarchical paths, or monotonic counters) —
  **never** from UUIDs, wall-clock, hash-set iteration order, or input arrival order;
* emit no timestamps / `etag` / `modified` attributes in the output.

`render_drawio` already obeys this; `eval_diagram.py` and `test_views.py` gate it.
A candidate backend that does not (e.g. CloudNetDraw emits VNets in Azure-API order
with no sort — see ADR-001) must be patched to comply before it can sit behind this
contract.

## The risk overlay (the second half of the contract)

Layout needs the IR; *painting* needs the overlay. `overlay.compute_overlay(fx)`
returns `{ "<kind>:<rid>": { severity, bucket, findings[] } }` computed once on the
**full** estate. A layout backend colours a node by looking up its `rid`-derived key
in this map. Views never recompute the overlay from a projected subset — they pass
the full-estate overlay through — so **a view can hide a resource but never change a
verdict.** A new backend must take the overlay as a separate input and must not
infer severity from geometry or from the projected fixture.

## What a new layout backend must implement

To swap `render_drawio` for ELK / Graphviz `dot` / a CloudNetDraw fork, the backend
must:

1. consume `graph.Fixture` JSON (this IR) + the overlay map;
2. honour the identity rule (`rid`-namespaced ids) and the determinism rule;
3. resolve cross-subscription peers to an explicit external-stub node, never a
   dangling edge (render invariant RC-1/RC-2);
4. read node colour ONLY from the overlay (RC-4);
5. pass `eval_diagram.py` (structure + RC checks + severity coverage) and
   `test_views.py` (projection faithfulness + determinism).

Meet those five and the backend is a drop-in. That is the entire point of pinning
this IR: the geometry is replaceable; discovery, risk, identity, and determinism
are not.

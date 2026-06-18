# Instructing antr to produce an Azure network topology diagram

**The one thing to internalise first:** you do **not** prompt antr with a framework.
antr is a *deterministic generator*, not an LLM that draws from a description. You give
it a small, scoped **input contract** (what to look at, which view, what to include,
what format) and it produces the same diagram every time from the same inputs. The
"topology modelling framework" — layers, trust boundaries, reachability, severity — is
**encoded once** in the graph IR + renderer + view families, not restated per request.
That encoding *is* why antr is consistent where a prompt-driven agent drifts.

So the correct mental model is:

```
your instruction  =  a scoped contract            (NOT a prose prompt)
antr's job        =  fetch → Analyze() → render    (deterministic, every time)
the framework     =  baked into the IR + views     (you don't re-specify it)
```

A hard rule that follows from this: **never instruct antr to decide severity,
reachability, or "what's risky."** That is the engine's job (`Analyze()`), and it is the
whole point of the product. Instructions select *scope and presentation*; the verdict is
always computed, never prompted.

---

## The input contract

This is the entire "instruction" you give antr:

```yaml
scope:
  subscription_id:   <azure-guid>        # required
  resource_group:    <name>              # optional filter (future)
  application_name:  <name>              # optional label (future)

view:                                    # which of the view families to emit
  - hld          # high-level: VNets · hubs · spokes · gateways · firewall · peerings · boundary
  - mld          # mid-level:  + subnets + NICs (full detail)
  - risk         # only resources that carry a finding (+ their VNet/subnet) + boundary
  - boundary     # internet-facing paths: boundary nodes + every internet-reachable NIC
  - cross-sub    # only VNets in cross-subscription peerings (multi-sub blast radius)
  - finding      # one focused k-hop diagram per Critical/High finding

include:                                 # resource families to pull (all default true)
  vnets · peerings · nsgs · route_tables · firewalls · gateways · express_route ·
  private_endpoints · nat_gateways · public_ips ·
  app_gateway · aks · front_door · apim · vwan · bastion · load_balancers

output:
  drawio                                 # (.drawio — opens in diagrams.net)  [svg/mermaid: future]
```

Everything else — the six discovery layers, the trust boundaries, the reachability
gates, the severity colours — antr already knows. You are only choosing **scope** and
**which view(s)**.

---

## How the modelling framework maps to antr

### The six discovery layers → resource families antr already pulls

You don't instruct these; the adapter discovers them and the renderer places them.

| Framework layer | antr discovers / draws |
|---|---|
| 1 · Entry points | Front Door · Application Gateway · Load Balancer · Public IP · VPN / ExpressRoute gateways |
| 2 · Network segmentation | Hub/Spoke VNets · Subnets (nested containers) |
| 3 · Application tier | AKS · App Service (NICs) · VMs (NICs) · APIM |
| 4 · Data tier | Private Endpoints (the data-plane ingress antr models) |
| 5 · Security controls | NSGs (effective rules) · Azure Firewall · WAF posture (App GW / Front Door) · Private Endpoints · AVNM security admin rules |
| 6 · Hybrid connectivity | ExpressRoute · VPN Gateway · Virtual WAN · NAT Gateway |

### The five rules → already enforced, or a known gap

| Rule | Status in antr |
|---|---|
| 1 · Group Subscription → VNet → Subnet → Workload | **Done** for VNet→Subnet→NIC nesting; subscription-level container grouping is partial |
| 2 · Separate logical & physical views | **Physical + security done** (HLD/MLD/risk/boundary). A **Logical** (Users→App→Data) view is **not yet built** |
| 3 · Highlight trust boundaries | **Internet boundary band done**; explicit Data / Corporate boundary lines are a **gap** |
| 4 · Highlight reachability (not just VM→SQL) | **Done — this is antr's core.** Exposure path edges (Internet→exposed NIC), firewall DNAT, cross-sub edges; reachability computed by the 4-gate engine, not guessed |
| 5 · Highlight findings (colour) | **Done.** Severity overlay: 🟢 Clean · 🔵 Info · 🟡 Medium · 🟠 High · 🔴 Critical — painted only from `Analyze()` output |

### The four enterprise views → antr's view families

| Architect view | antr view | Built? |
|---|---|---|
| Executive (Users→App→Data) | `hld` (today) / **Logical view** (proper abstraction) | hld ✅ · logical ⬜ |
| Network (Hub→Spokes→Controls) | `mld` | ✅ |
| Security (reachability · boundaries · exposures) | `risk` + `boundary` (+ `finding`, `cross-sub`) | ✅ |
| Change Impact (current → proposed → reachability Δ → cost Δ) | `simulate_change` + `forecast_cost` compute the delta; **rendering it as a before/after diagram is a gap** | data ✅ · diagram ⬜ |

---

## How you actually invoke it

### Live subscription (MCP tools)

The MCP server is the production front door. Tool → contract field:

| Tool | Required | Optional | Produces |
|---|---|---|---|
| `get_topology` | `subscription_id` | — | the raw topology (the IR) |
| `analyze_risks` | `subscription_id` | `severity_filter` | findings + severity |
| `format_report` | `subscription_id`, `format` (`markdown`\|`drawio`) | — | a report / `.drawio` diagram |
| `simulate_change` | `subscription_id` | `delta` (JSON) | pre-deploy reachability **delta** |
| `forecast_cost` | `subscription_id` | `delta`, `region` | fixed + variable **cost delta** |

Example instruction to an MCP client:

> "Run `format_report` on subscription `<guid>` with `format=drawio`, then `analyze_risks`
> with `severity_filter=High` for the exposure summary."

That's the whole instruction. No framework prose — the framework is in the engine.

### Offline / fixture (the deterministic CLI — what `make demo` runs)

```bash
# All view families from one estate (HLD · MLD · risk · boundary · cross-sub · finding)
python3 phase-4/viz/views.py <fixture.json> --out-dir out/views

# A single view at a chosen level
python3 phase-4/viz/render_drawio.py <fixture.json> --out topo.drawio --level mld

# End-to-end (analyze → all views → severity summary)
make demo FX=<fixture.json>
```

Each view is a deterministic projection over one whole-estate risk truth: a view can
hide a resource but can **never change a verdict**.

---

## What "good instruction" looks like (and what to avoid)

**Do** — select scope + view + format:

> "Give me the `risk` and `boundary` views for subscription `<guid>` as `.drawio`, and the
> `finding` views for every High/Critical exposure."

**Do** — drive a change review:

> "`simulate_change` on `<guid>` with this delta (add a public IP to `nic-web`); show me the
> reachability delta and `forecast_cost`."

**Don't** — ask antr to reason about risk:

> ~~"Generate a network topology and tell me what looks dangerous."~~ — this invites an LLM to
> invent a verdict. antr computes the verdict deterministically; you ask for the *view*, and the
> danger is already coloured in.

**Don't** — restate the modelling framework in the prompt. It's encoded; repeating it adds
nothing and risks implying the model should improvise structure.

---

## Known gaps (where instruction can't yet help — they need building)

These are honest limitations, not things a better prompt fixes:

1. **Application / Logical view** — a Users→App→Data dependency chain, reachability-annotated.
   **Designed** in `phase-4/design/ADR-002-application-view.md` (v1 builds on today's IR via
   NIC-tag membership propagation + existing backend-pool / private-endpoint edges).
2. **Change-Impact diagram** — `simulate_change` + `forecast_cost` produce the delta as data;
   rendering a before/after diagram (current → proposed → reachability Δ → cost Δ) is not built.
   This is antr's sharpest differentiator and the engine already has the data.
3. **Subscription containers + non-Internet trust-boundary bands** (Data / Corporate) — partial.
4. **`svg` / `mermaid` output** — only `.drawio` today.

5. **Full-estate / BCLM-parity view** — one canvas covering network + workloads + **data
   services** (SQL/Storage/Redis/…) + DNS + NAT + boundary, like the hand-drawn reference.
   **Designed** in `phase-4/design/FULL_ESTATE_VIEW_REQUIREMENTS.md` (its biggest piece —
   data-service discovery — needs new ARG queries + adapter/twin work).

For the exact IR field shapes, the analysis/overlay output, the node-id scheme, the severity
palette, and example fixtures an external build can use, see **`IR_SCHEMA.md`**. See
`phase-4/design/ADR-001-visualization-strategy.md` and `GRAPH_IR.md` for how a new view or
output backend plugs in without touching discovery or the engine.

---

## TL;DR

Give antr a **scoped contract** (subscription + which view(s) + format), not a framework
prompt. The six layers, trust boundaries, reachability, and severity colours are already
encoded and computed deterministically — that's what makes the output consistent and
review-grade. Reserve your "instructions" for *scope and presentation*; the verdict is the
engine's, always.

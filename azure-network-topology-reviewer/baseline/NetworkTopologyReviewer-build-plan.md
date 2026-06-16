# Azure Network Topology Expert Reviewer — Phased Build Plan

**Status:** Draft for review · **Date:** 2026-06-02 · **Scope:** Azure-first, AWS as a later adapter

## The one decision that shapes everything

Build a **deterministic topology-graph engine** as the core; keep the LLM at the edges (natural-language explanation, RAG-grounded recommendations, intent→spec translation). Topology review is graph reachability and rule evaluation — CIDR overlap, effective routes, NSG precedence, transitive peering. That arithmetic is computed by Azure primitives and graph algorithms, never by AskAT&T. Every phase below hangs off that core.

## The forced sequence

The analyzer is the keystone. You cannot simulate a change or validate a generated topology until you can analyze one. This forces the order:

```
P0 Graph substrate ──► P1 Analyzer (read-only review) ──┬──► P2 Cost-aware simulation
                                                        └──► P3 Design generation
```

P2 and P3 both consume P1's analyzer (P2 re-analyzes a simulated graph; P3 validates a generated graph before emit). After P1 is stable they can be staffed in parallel, but neither can precede it.

Effort is given as T-shirt sizes (S/M/L/XL) deliberately — fixed week counts are not credible until team capacity is set. **No timeline numbers ship in this plan without a staffing baseline.** `[VERIFY]` marks anything needing confirmation against the live environment.

---

## Phase 0 — Graph substrate (foundation)

**Goal:** Materialize an accurate, cloud-neutral topology graph of one Azure subscription from authoritative sources. Nothing intelligent yet — just a trustworthy map.

**In scope**

- Cloud-neutral graph model: nodes (network, subnet, gateway, endpoint, NIC, firewall, public IP, peering) and edges (peering, route, association, effective-reachability). Rules will run on this model, not on raw Azure JSON — this is what makes AWS a second adapter instead of a rewrite.
- Azure adapter: **Azure Resource Graph** (KQL) for fast tenant/subscription inventory; **Network Watcher** Topology API + **Effective Security Rules** + **Effective Routes** + **Next Hop** for the *evaluated* truth (not just declared config).
- Read-only identity: Managed Identity with **Reader** + **Network Contributor**-equivalent *data-plane read* scoped at management-group level for tenant-wide reach `[VERIFY exact role set]`. No write permissions anywhere in this identity.
- Sandbox subscription seeded with deliberately known-bad topologies — this doubles as the eval substrate for P1.
- Observability + governance scaffold: Azure Monitor + Application Insights wired; Container Apps built-in auth (Entra) for the MCP ingress — no APIM; AskAT&T governs model access.

**Out of scope:** any analysis, any LLM, any write path.

**Exit criteria:** graph materialized for the sandbox subscription matches `az network` ground truth on a spot-check set; graph rebuild is idempotent and repeatable; identity proven read-only by attempting (and failing) a write.

**Effort:** L · **Risk:** low (read-only, deterministic)

---

## Phase 1 — Read-only reviewer (v1, the wedge)

**Goal:** Trusted findings on deployed topology. This is the product that earns adoption; ship it narrow and correct.

**In scope**

- Deterministic rule engine over the graph, fixed v1 rule set: over-permissive NSG rules, CIDR/address-space overlap, transitive peering exposure, orphaned public endpoints, missing segmentation between workload tiers.
- **Reachability-based severity** — a finding is high only when the path is real (e.g., NSG allows `0.0.0.0/0` on 22 *and* an effective route to internet *and* an attached public IP), not theoretical. This is the single biggest driver of trust.
- Consume **Microsoft Defender for Cloud** signals (attack-path / internet-exposure) where they overlap rather than reimplementing them. The differentiator is the next two bullets, not re-flagging what Defender already flags.
- RAG layer on **Azure AI Search** (Ask Docs) grounding every recommendation in a versioned AT&T architecture standard, with a link back to the source clause. No bare LLM recommendations.
- LLM (accessed via AskAT&T) confined to: explaining findings in natural language and synthesizing the grounded recommendation. It does not decide severity or compute reachability.
- Structured report + escalation routing: high → network architecture team; medium/low → ticket to the owning resource group via the existing workflow.
- **MCP server v1** exposing `get_topology` and `analyze_risks` — the reusable interface the use case promises, established now so it is real rather than retrofitted.
- Labeled eval set with precision/recall gate.

**Out of scope:** cost, simulation of proposed changes, any generation, multi-subscription fan-out (single subscription first).

**Exit criteria:** precision/recall thresholds met on the eval set (`[VERIFY target thresholds with architects]` — false positives are the adoption killer, so weight precision); a senior architect accepts the report on a real read-only subscription; MCP `analyze_risks` returns the same verdicts as the internal engine.

**Effort:** XL · **Risk:** medium (correctness and trust are the whole game here)

---

## Phase 2 — Cost-aware simulation (v2)

**Goal:** Forecast the impact of a proposed change before it ships — security delta, blast-radius delta, cost delta.

**In scope**

- `simulate_change`: apply a proposed topology delta to the in-memory graph and re-run the P1 analyzer to produce security-posture and blast-radius deltas.
- Cost model split honestly into two parts: **fixed** (gateway/firewall/Private Endpoint SKUs via the **Azure Retail Prices API** — exact) and **variable** (data processing and egress: firewall per-GB, cross-region/zone peering, NAT gateway, Private Link — estimated from **VNet flow logs + Traffic Analytics**). Build on **VNet flow logs** from day one: NSG flow logs stop accepting new resources after 30 Jun 2025 and retire 30 Sep 2027.
- Integrate the shared **Azure Cost MCP Server** for actuals reconciliation — but keep *forecast* (this agent) and *actuals* (Cost Optimizer) as distinct computations. Share the price source, not a false claim of total-cost precision.
- MCP tools added: `simulate_change`, `forecast_cost`.

**Out of scope:** generation; auto-applying any change.

**Exit criteria:** fixed-cost delta exact against a billing cross-check; variable-cost forecast within a stated tolerance band on a set of known changes; simulated-graph analysis matches what the same change produces once actually deployed in the sandbox.

**Effort:** L · **Risk:** medium (variable-cost forecasting is inherently approximate — set expectations as a band, not a number)

---

## Phase 3 — Design generation (v3, greenfield mode)

**Goal:** Turn an architect's requirements into a validated topology proposal. This is the "create new ones" half — the riskier half, gated hard.

**In scope**

- Intent capture: architect requirements → a structured topology **spec** (LLM with structured output). The LLM produces intent, not infrastructure code.
- Module selection from a vetted registry (CAF / Azure Landing Zones modules, or AT&T's own Terraform module registry). The LLM **selects and parameterizes** approved modules — it never authors network security Terraform from scratch.
- Deterministic renderer: spec + chosen modules → **Terraform AzureRM**.
- **Validate the generated topology through the P1 analyzer before emitting anything.** The reviewer and the generator close the loop on the same engine.
- Output path: a **pull request** via GitHub Actions + OIDC — never auto-apply. For connectivity creation/enforcement at scale, target **Azure Virtual Network Manager** (connectivity configs for hub-spoke/mesh; security admin rules that evaluate before NSGs) rather than hand-rolled peering.
- MCP tool added: `generate_topology`.

**Out of scope:** the agent holding any write/apply permission; bypassing human PR approval.

**Exit criteria:** generated topology passes the analyzer with zero high-severity findings before emit; the Terraform PR round-trips cleanly through CI; a human approves and applies — the agent does not.

**Effort:** XL · **Risk:** high (highest blast radius; the module-not-author guardrail and the analyzer-before-emit gate are non-negotiable)

---

## Cross-cutting tracks (run alongside, not after)

- **Eval set** grows every phase — known-bad topologies (P1), known-cost changes (P2), known-good/bad designs (P3). This is the trust ledger; it is not optional.
- **AWS adapter** — start only after P1 is stable. VPC≈VNet, SG+NACL≈NSG, TGW/peering≈peering, Network Firewall≈Azure Firewall. It maps AWS sources (Config / `describe-*`) onto the *same* neutral graph; the rule engine is unchanged. If P0/P1 leak Azure types into the core, this becomes a rewrite — so guard the boundary from the start.
- **Governance** — MCP ingress fronted by Container Apps built-in auth (Entra ID); AskAT&T governs model access (no separate AI gateway); Managed Identity scoping reviewed each phase; full audit logging of every finding, recommendation, and generated artifact; model-call auth to AskAT&T is JWT bearer — the agent acquires the token via its Managed Identity where AskAT&T accepts Entra-issued tokens, otherwise from AskAT&T's token service with the client secret in Azure Key Vault `[VERIFY token source]`, then caches it, refreshes before expiry, validates audience/scope, and never logs the Authorization header. Because the agent calls AskAT&T under its own app identity (client-credentials), per-user authorization is enforced at the MCP ingress, not at AskAT&T.

## What I am deliberately not doing in v1

- Not rebuilding Microsoft Defender for Cloud. Where findings overlap, consume Defender.
- Not letting the LLM compute reachability, severity, or author Terraform.
- Not shipping unbaselined outcome metrics. Targets in the use-case doc (faster reviews, earlier detection) need a measured baseline before they go in any deck — `[VERIFY]` until then.
- Not granting the agent any apply/write capability. Every change leaves as a PR.

## Open questions for the architect

- Scope boundary for P0/P1: single subscription, then management-group fan-out — confirm the target management-group structure `[VERIFY]`.
- Precision/recall thresholds that count as "trustworthy" for P1 sign-off.
- Source of truth for AT&T architecture standards feeding the RAG index, and its update cadence.
- Whether AVNM is already in use (changes P3's enforcement path materially).

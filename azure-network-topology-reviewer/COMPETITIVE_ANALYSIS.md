# antr — Competitive Analysis (critical, evidence-based)

**Date:** 2026-06-15 · **Method:** 4 parallel research passes (reachability, CSPM/attack-path,
visualization, IaC cost/policy/gen) over open-source repos, native Azure services, and commercial CNAPPs.
**Audience:** antr engineering + sponsors. **Stance:** critical and realistic — antr is internal AT&T
tooling, so the bar is "cheaper/better than buying, for AT&T's needs," not "novel sellable IP."

---

## Executive verdict (BLUF)

**Most of antr re-implements capabilities that are already mature — natively in Azure, in commercial
CNAPPs AT&T may already own, and in open source.** The deterministic 4-gate reachability engine is a
*competent re-build of a 10-year-old category* (Batfish) that Microsoft now ships natively (**AVNM Network
Verifier**, GA) and that **Defender for Cloud** + **Wiz** already do as the entry condition to full
attack-path graphs. The severity-painted diagram is **CloudNetDraw + a colour overlay**. The cost engine
is **largely Infracost**. The "validate before emit" gate is **complementary to, not better than, OPA/Checkov**.

antr's **genuinely defensible wedge is narrow but real**, and it converges on four things that *no single
incumbent combines*:

1. **`simulate_change`** — a deterministic, pre-deploy **security/reachability *delta*** of a proposed change. This is the strongest, least-duplicated capability (~70% differentiated). Native reachability tools (AVNM Verifier, AWS Reachability Analyzer) need **deployed, running VMs** and can't diff; static scanners don't compute reachability; Wiz Code detects *new external exposure* but isn't a before/after reachability solver on an undeployed plan.
2. **Deterministic, auditable, fixture-tested, embeddable, *license-free* reachability** — vs Defender's paid, black-box, proprietary scoring and AVNM's managed-service/running-VM dependency.
3. **MCP-native delivery** — agent-consumable findings/diagrams; an emerging surface none of the incumbents own.
4. **Output as diffable artifacts** (drawio/Confluence, CI-gated) rather than a live SaaS console.

**The blunt practical bottom line:** *if AT&T already licenses Defender for Cloud CSPM or Wiz, a large
fraction of antr is redundant.* Its justification must rest on the four points above — especially the
free-foundational-CSPM subscriptions, deterministic CI gating, pre-deploy simulate, and agent integration
— not on the analysis being novel.

---

## Axis 1 — Deterministic network reachability (antr's core)

| Tool | Type / license | Azure reachability | Severity | Maturity | vs antr |
|---|---|---|---|---|---|
| **AVNM Network Verifier** | Native, **GA** | **antr's exact gate set**: admin rules → NSG/ASG → routes → peering → firewall (static L4); "internet" as src/dst; returns path + blocking component | No severity, one intent at a time | Microsoft, GA ~11 regions | **Existential twin.** antr's gate *arithmetic* is not novel — it's native. Gaps antr exploits: needs AVNM deployment + a **running VM**, Firewall **static-L4 only** (no DNAT depth), no fleet sweep, **no severity** |
| **Batfish** | OSS, Apache-2.0 | Internet-exposure reachability since 2015; Azure (VNet/NSG/PublicIP) support **present but new & thin**, AWS-proven | Query DSL, not a severity product | ~1.4k★, 13k commits, AWS-stewarded, active | The decade-old reference. Does **all-headers symbolic** reachability (deeper than antr's per-path). antr likely deeper on AVNM-precedence/None-route/DNAT for Azure |
| **Defender for Cloud — exposure width** | Native, **paid CSPM** | Control-plane + **network-path reachability** (routing/security/firewall rules) | **Yes** — exposure width feeds severity/attack-path | Microsoft, GA | Already does "deterministic Azure internet reachability **with severity**," fleet-wide. Contradicts the "severity-from-path is novel" framing |
| AWS VPC/Network Access Analyzer | Native (AWS) | (AWS only) automated-reasoning exposure findings | findings → Security Hub | GA | Proves the *category* is productized cloud-native, not a new idea |
| Forward Networks / Veriflow | Commercial | Multi-cloud digital twin, formal verification | policy/posture | Enterprise | Same category, heavyweight; "math-based verification" is a 10–15 yr established field |

**Verdict (core):** competently engineered, **only modestly novel, not strongly defensible as IP.**
Defensible properties = *embeddable Go engine + Python reference twin + fixtures (auditable/CI-testable) +
deeper Azure-Firewall-DNAT & admin-rule-precedence than AVNM's static-L4 + zero AVNM/Defender/Wiz
dependency.* **Stop framing the graph engine as the differentiator;** lead with Azure-gate correctness,
testability, and the exposure-sweep-plus-severity UX, and be ready to name exactly where AVNM Verifier,
Defender, and Batfish's Azure model are insufficient. *(Sources: [AVNM Network Verifier](https://learn.microsoft.com/en-us/azure/virtual-network-manager/concept-virtual-network-verifier), [Batfish](https://github.com/batfish/batfish), [Defender internet-exposure](https://learn.microsoft.com/en-us/azure/defender-for-cloud/internet-exposure-analysis))*

---

## Axis 2 — Exposure findings / CSPM landscape

| Tool | Type | Azure network reachability | OSS maturity | vs antr |
|---|---|---|---|---|
| **Defender for Cloud** | Native, paid CSPM | Attack-path graph from internet-exposed entry → crown jewels; deterministic NSG/route/firewall exposure | — | Direct overlap; antr's exposure findings largely **reimplement** what Defender's exposure engine provides |
| **Wiz** (Google) | Commercial | "Publicly exposed only if the graph can trace an actual configured path"; + active external scan + eBPF runtime | — | A **superset** of antr's 4 gates, multi-cloud, validated |
| **Cartography** | OSS, Apache-2.0, CNCF | Models NSGs, every rule (dir/access/priority/source), subnet assoc, public IPs; **internet-reachable is Cypher-queryable** | ~520★, active | **Closest OSS — ~80% of antr's core.** Gap: "topology approximation, does **not** validate effective route path or firewall precedence" |
| Prowler / ScoutSuite / Steampipe / CloudQuery | OSS | Per-rule compliance checks; no path/reachability graph | mixed (ScoutSuite stale) | Config checks, not reachability |
| CloudMapper / OpenCSPM | OSS | AWS-only / archived | dead | Not relevant |

**Verdict (CSPM):** **largely redundant on the core capability, modestly differentiated on packaging.**
antr's own principle *"consume Defender, don't reimplement"* is **not honored in practice** — its
over-permissive-NSG / orphaned-endpoint / transitive-peering findings duplicate Defender's exposure
analysis. Genuinely additive: **CIDR overlap** and **missing tier segmentation** (network hygiene Defender
doesn't frame). **Strongest argument for antr:** deterministic, fully-explainable reachability evidence
(the exact rule chain) via **MCP, with zero paid CSPM license and zero agent** — serving the large fraction
of subscriptions on **free foundational CSPM** where Defender's attack-path analysis simply isn't available.
*(Sources: [Defender attack-path](https://learn.microsoft.com/en-us/azure/defender-for-cloud/concept-attack-path), [Wiz reachability](https://www.wiz.io/academy/application-security/reachability-analysis-in-cloud-security), [Cartography schema](https://github.com/cartography-cncf/cartography/blob/master/docs/root/modules/azure/schema.md))*

---

## Axis 3 — Topology visualization (Phase 4)

| Tool | Type | Multi-sub | **Security-severity overlay** | drawio / Confluence | vs antr |
|---|---|---|---|---|---|
| **CloudNetDraw** | OSS, MIT | **Yes** | **No** (pure topology) | **drawio, HLD/MLD** + CI validator | **antr Phase-4 minus the colour.** A fork adding `fillColor` keyed to a severity dict is a *weekend job* |
| **Cloudcraft** (Datadog) | Commercial | Yes | **Yes** — security overlay | Confluence app | Closest commercial analog to antr's whole pitch; interactive, not drawio |
| **Hava.io / Lucidscale** | Commercial | Yes | security groups on map (not severity heat) | varies | "Security-aware topology" niche already occupied |
| **Cloudockit** | Commercial | Yes | No | **drawio + scheduled refresh + change-diff** | Owns the "drawio + doc automation" lane |
| **Defender / Wiz security graph** | Native / commercial | Yes | **Yes — severity on a reachability graph** | live console, not a document | Severity-painted maps **already exist** |
| AzViz / Network Watcher Topology | OSS / native | weak / yes | No | PNG-SVG / portal-only (30h lag) | lower relevance |

**Verdict (viz):** **thin as a *feature*, real as a *deliverable format*.** "Severity on a topology map"
is a solved concept (Defender, Wiz, Cloudcraft, Hava). The wedge is the **deterministic, CI-gated,
Confluence-native drawio artifact** — severity-as-data on a *diffable static document*, not a live console.
**Buy the renderer, build the engine:** adopt/fork CloudNetDraw rather than maintain a bespoke drawio
emitter; reserve engineering for the reachability classifier that *computes* the colour.
*(Sources: [CloudNetDraw](https://github.com/krhatland/cloudnetdraw), [Cloudcraft](https://docs.datadoghq.com/datadog_cloudcraft/), [Cloudockit](https://www.cloudockit.com/versions/))*

---

## Axis 4 — Cost / Policy / Generation / Simulate

| antr capability | Closest prior art | Differentiated? | Recommendation |
|---|---|---|---|
| **forecast_cost** (fixed SKU) | **Infracost** (OSS, Apache-2.0) already does SKU pricing + **diff-on-plan + PR comment**; **Terracost** (MIT, Go lib) for in-process | **~20%** — fixed cost reinvents Infracost | **Adopt Infracost (or Terracost in-process).** Keep only the flow-log → egress-tier **variable** cost (Infracost makes egress a manual usage estimate; deriving it from Azure flow logs is the real delta) |
| **"validate before emit"** gate | OPA/Conftest, Checkov (graph-aware), tfsec, Terrascan, Sentinel | reachability gate is **more semantic** than static linters, but **narrower** (network only) | **Complementary, not a replacement.** Run OPA+Checkov for the broad compliance surface; reserve the analyzer for the reachability axis. Don't build generic compliance checks |
| **generate_topology** (intent→modules→PR) | Pulumi AI/Neo (intent→IaC), **Azure Verified Modules / ALZ** (vetted modules) | **~30%** — intent→IaC is commodity; AVM provides the modules | **Adopt AVM/ALZ as the registry.** Differentiation is the *reachability gate*, not "generate IaC" |
| **simulate_change** (pre-deploy security delta) | AVNM Verifier / AWS Reachability Analyzer need **running VMs**, can't diff; Wiz Code flags *new external exposure* but isn't a before/after solver | **~70%+ — the real gap** | **Lead with this.** It's the one place antr isn't reinventing mature prior art |

*(Sources: [Infracost usage-based](https://www.infracost.io/docs/usage_based_resources/), [Terracost](https://github.com/cycloidio/terracost), [Checkov](https://github.com/bridgecrewio/checkov), [Azure Verified Modules / ALZ](https://azure.github.io/Azure-Landing-Zones/terraform/), [Wiz IaC scanning](https://www.wiz.io/academy/application-security/iac-scanning))*

---

## Redundancy vs differentiation — the consolidated matrix

| antr piece | Verdict | Action |
|---|---|---|
| 4-gate reachability engine | Commoditized (AVNM/Defender/Batfish) — defensible only as *embeddable + testable + license-free* | Keep, **reframe**; benchmark vs AVNM/Batfish and publish the deltas |
| Exposure findings (NSG/peering/orphan) | Redundant with Defender exposure | Where Defender CSPM is licensed, **consume its signals**; keep engine for free-tier subs |
| CIDR overlap + tier segmentation findings | Genuinely additive | Keep |
| Severity-painted drawio | Thin feature on commodity renderer | **Adopt CloudNetDraw**; keep the *engine* that colours it |
| forecast_cost (fixed) | Reinvents Infracost | **Adopt Infracost/Terracost**; keep flow-log variable cost |
| validate-before-emit | Complementary to OPA/Checkov | Keep narrow; **add** OPA+Checkov alongside |
| generate_topology | Intent→IaC commodity | **Adopt AVM** as registry; keep the gate |
| **simulate_change** | **Defensible gap** | **Invest here** |
| MCP-native delivery | Emerging, unowned by incumbents | Keep — real wedge |
| Deterministic + CI-gated + diffable artifacts | Real wedge vs live-console CNAPPs | Keep |

---

## Recommended positioning (honest, survivable under scrutiny)

Do **not** pitch antr as "we compute reachability / we colour risk on a map" — reviewers will correctly
point to AVNM Verifier, Defender, Wiz, and Cloudcraft. Pitch it as:

> **"Deterministic, auditable, license-free Azure network-exposure analysis and pre-deploy change
> simulation, delivered to engineers and AI agents as CI-gated, diffable artifacts — for the estate that
> isn't fully covered by paid Defender CSPM / Wiz."**

That framing is defensible because every clause names something the incumbents *don't* combine.

## What to STOP building and ADOPT instead

1. **Stop** the bespoke fixed-cost pricing engine → **adopt Infracost / Terracost**; keep flow-log variable cost.
2. **Stop** positioning the analyzer as a policy-as-code replacement; **stop** generic compliance checks → **adopt OPA + Checkov** alongside.
3. **Stop** maintaining a hand-rolled drawio renderer → **adopt/fork CloudNetDraw**.
4. **Stop** a proprietary vetted-module library if it duplicates **AVM/ALZ** → adopt AVM.
5. **Stop** reimplementing Defender's exposure analysis where Defender CSPM is licensed → **consume its signals**; reserve the engine for free-tier subs + the additive findings.
6. **Invest the freed capacity in `simulate_change`** (pre-deploy reachability/security delta) + engine depth + MCP — the actual moat.

## Top tools to study / adopt / integrate

| Purpose | Tool | Why |
|---|---|---|
| Reachability benchmark | **AVNM Network Verifier**, **Batfish** | the native twin + the OSS reference; define antr *against* them |
| Exposure data source | **Defender for Cloud** exposure/attack-path | consume rather than recompute (where licensed) |
| OSS graph backend | **Cartography** | ~80% of antr's Azure graph already; Apache-2.0, reusable ontology |
| Renderer | **CloudNetDraw** (fork) | the undifferentiated 80% of Phase 4 |
| Cost | **Infracost** / **Terracost** | stop reinventing pricing |
| Policy | **OPA/Conftest + Checkov** | broad compliance antr shouldn't build |
| Modules | **Azure Verified Modules / ALZ** | the vetted registry for generation |
| Commercial benchmark | **Cloudcraft**, **Wiz Code** | "security-on-diagram-in-Confluence" and "pre-deploy exposure gate" already ship |

---

## Caveats on confidence

Vendor methodology claims (Wiz, Prisma, Orca, Cloudcraft "security overlay" depth, Wiz Code pre-deploy
diff semantics) come from vendor/marketing sources, not independent benchmarks — validate against live
demos before betting the `simulate_change` differentiation is as wide as it looks. Batfish's Azure depth is
documented but thinly exercised vs its AWS support. The reachability/CSPM analysis used antr's described
behavior, cross-checked against its code where available.

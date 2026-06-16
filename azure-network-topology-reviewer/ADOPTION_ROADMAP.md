# antr — Adoption Roadmap

**Date:** 2026-06-15 · **Derives from:** `COMPETITIVE_ANALYSIS.md` · **Reads with:** `tickets/`
**Purpose:** turn the competitive findings into sequenced, estimated, actionable work — what to **adopt**,
what to **stop building**, and what to **invest in**. Internal AT&T tooling; the goal is "cheaper/better
than buying, for AT&T's estate," not novel sellable IP.

## Guiding principle

**Concentrate engineering on the four things no incumbent combines — deterministic + license-free +
pre-deploy change-simulation + CI-gated/agent-native artifacts — and adopt commodity for everything else.**
Every wave below either sharpens the wedge or removes maintenance load that isn't differentiated.

Effort = T-shirt (S ≤ 2d · M ≤ 1wk · L ≤ 2–3wk · XL > 3wk). Waves are ordered by leverage-per-effort.

---

## Wave 0 — Reposition (free, do first)

| ID | Action | Why (analysis) | Effort | Depends |
|---|---|---|---|---|
| **ADOPT-01** | Rewrite the value framing in `README.md` + `instruction-os` persona/narrative: lead with *"deterministic, auditable, license-free Azure exposure analysis + pre-deploy change simulation, delivered as CI-gated diffable artifacts to engineers and AI agents — for the estate not covered by paid Defender CSPM / Wiz."* Stop leading with "we compute reachability / colour risk on a map." | Core reachability + severity-on-map are commoditized (AVNM Verifier GA, Defender, Wiz, Cloudcraft). Only the *combination* survives scrutiny. | **S** | — |
| **ADOPT-02** | Add a one-page "Where antr fits vs Defender / AVNM Verifier / Wiz" section to the README — when to use antr, when to use the native/commercial tool. | Sponsors will ask "why not just Defender?"; answer it proactively. | **S** | ADOPT-01 |

**Exit:** a reviewer who knows Defender/AVNM reads the README and cannot dismiss antr as redundant.

---

## Wave 1 — Concentrate on the wedge (highest-value build)

| ID | Action | Why | Effort | Depends |
|---|---|---|---|---|
| **ADOPT-03** | Execute `tickets/PHASE2-MCP-wiring.md`: wire `simulate_change` + `forecast_cost`, then **make `simulate_change` the headline** — primary demo, primary value story ("security + cost delta of a change, before deploy"). | `simulate_change` is the ~70%-differentiated capability; native tools need running VMs and can't diff; Wiz Code isn't a before/after solver. | **M–L** | Go 1.25 (CI) |
| **ADOPT-04** | Execute `tickets/V4-07-Go.md`: bring the Go engine to ARM-id keying parity (twin-drift = 0). | Multi-subscription correctness is table stakes for the estate antr targets; the Python ref is already fixed. | **M** | Go 1.25 (CI) |
| **ADOPT-05** | Build a `simulate_change` showcase: a fixture "change set" (add public IP, open NSG, re-route to firewall) → before/after security delta + cost delta in one report. Wire it into the diagram-eval/CI as a regression. | Turns the wedge into a demonstrable, regression-tested artifact, not a claim. | **M** | ADOPT-03 |

**Exit:** `simulate_change` runs end-to-end, is the lead demo, and the engine is twin-parity-correct.

---

## Wave 2 — Adopt commodity, reclaim maintenance (stop building)

| ID | Action | Replaces (stop building) | Effort | Depends |
|---|---|---|---|---|
| **ADOPT-06** | **Fork + vendor CloudNetDraw** (MIT) as the discovery + drawio HLD/MLD renderer; keep only the severity-overlay layer (the engine output → `fillColor`). Retire the bespoke `phase-4/viz/render_drawio.py` to a thin overlay adapter over CloudNetDraw, OR contribute the severity extension upstream. | Hand-rolled drawio renderer (CloudNetDraw is ~80% of Phase 4 already; coloring is one line/node). | **M** | Phase-4 overlay (done) |
| **ADOPT-07** | **Adopt Infracost** for fixed-cost SKU pricing + diff-on-plan; if in-process Go is required for the MCP server, evaluate **Terracost** (MIT, Go). Reduce antr's cost scope to the **flow-log → egress-tier variable cost** overlay only. | Bespoke Azure Retail Prices ingestion + diff logic in `forecast/`. | **M** | ADOPT-03 |
| **ADOPT-08** | **Add OPA/Conftest + Checkov** to the generate pipeline *alongside* the reachability gate (compliance surface: encryption, IAM, tagging, public exposure). Position the analyzer as the *reachability axis only*. | Any plan to build generic compliance checks into antr. | **S–M** | generator (Phase 3) |
| **ADOPT-09** | **Adopt Azure Verified Modules / ALZ** as the `generate_topology` vetted-module registry. | A proprietary vetted-module library duplicating AVM. | **M** | generator (Phase 3) |

**Exit:** antr owns the engine + overlay + simulate; renderer/cost/compliance/modules are adopted, not maintained.

---

## Wave 3 — Consume Defender, prove the differentiation

| ID | Action | Why | Effort | Depends |
|---|---|---|---|---|
| **ADOPT-10** | **Honor "consume Defender, don't reimplement":** where Defender CSPM is licensed, *ingest* its internet-exposure / attack-path signals (Azure Resource Graph / Cloud Security Explorer) instead of recomputing. Keep antr's engine for **free-foundational-CSPM** subscriptions + the additive findings (**CIDR overlap**, **missing tier segmentation**). | antr's own principle is currently not honored; this resolves the redundancy with Defender. | **L** | live Azure |
| **ADOPT-11** | **Benchmark doc** vs **AVNM Network Verifier** and **Batfish** (Azure model) on a shared scenario set — name exactly where each is insufficient (AVNM Firewall static-L4-only, running-VM requirement, no severity, no fleet sweep; Batfish Azure depth). | "We had to build our own" is weak without naming the deltas. | **M** | engine |
| **ADOPT-12** | **Validate `simulate_change` against a live Wiz Code demo** — the one low-confidence point in the analysis (pre-deploy exposure diff depth). | Confirm the wedge is as wide as it looks before over-investing. | **S** | Wiz access |

**Exit:** antr can defend its existence against Defender/AVNM/Wiz with evidence, not assertion.

---

## Wave 4 — Lean into the genuinely-yours properties

| ID | Action | Why | Effort | Depends |
|---|---|---|---|---|
| **ADOPT-13** | Make **testability** a first-class selling point: the Python reference twin + fixtures + `engine-ci.yml` (diagram-eval gate, twin-drift) are things Defender/Wiz (black boxes) cannot offer. Document "exposure analysis you can fixture-test and diff in CI." | Defensible property already built; surface it. | **S** | CI (done) |
| **ADOPT-14** | Expand **MCP-native delivery**: ensure `analyze_risks`, `simulate_change`, `render_topology` are clean agent tools; publish an agent-usage example (the `aara-topology-visualizer` agent). | Agent-consumable security analysis is an emerging surface no incumbent owns. | **M** | MCP server |

---

## Decision register (the buy-vs-build calls)

| Capability | Decision | Rationale |
|---|---|---|
| Reachability engine | **BUILD + reframe** | embeddable/testable/license-free is the defensible delta, not the arithmetic |
| `simulate_change` | **BUILD (invest)** | the real wedge; not served by AVNM/AWS/Wiz |
| Topology renderer | **ADOPT (fork CloudNetDraw)** | commodity; ~80% exists |
| Fixed cost | **ADOPT (Infracost/Terracost)** | reinventing a mature tool |
| Variable (flow-log) cost | **BUILD (thin)** | genuine gap Infracost doesn't cover |
| Compliance/policy | **ADOPT (OPA+Checkov)** | broad surface antr shouldn't own |
| Vetted modules | **ADOPT (AVM/ALZ)** | Microsoft-maintained |
| Exposure findings (where Defender licensed) | **CONSUME Defender** | stop reimplementing |
| CIDR overlap + segmentation findings | **BUILD/keep** | additive, Defender doesn't emit |
| MCP delivery + CI-gated artifacts | **BUILD/keep** | unowned wedge |

## Do-NOT list (explicit stop-building)

- A bespoke fixed-cost pricing engine (→ Infracost/Terracost).
- Generic compliance checks (encryption/IAM/tagging) (→ OPA/Checkov).
- A hand-maintained drawio emitter as a core asset (→ CloudNetDraw fork).
- A proprietary vetted-module library duplicating AVM (→ AVM/ALZ).
- Re-deriving Defender's internet-exposure where Defender CSPM is licensed (→ consume signals).

## Sequencing summary

```
Wave 0 (S, now)      Reposition framing ............................. ADOPT-01,02
Wave 1 (M–L)         Wedge: simulate_change + engine parity ......... ADOPT-03,04,05  [tickets exist]
Wave 2 (S–M each)    Adopt commodity: renderer/cost/policy/modules .. ADOPT-06..09
Wave 3 (M–L)         Consume Defender + prove differentiation ....... ADOPT-10,11,12
Wave 4 (S–M)         Surface testability + MCP ...................... ADOPT-13,14
```

Critical path to a defensible product story: **ADOPT-01 → ADOPT-03 → ADOPT-05 → ADOPT-11** (reframe →
ship simulate → demonstrate it → prove the deltas vs incumbents). Everything else removes load or hardens.

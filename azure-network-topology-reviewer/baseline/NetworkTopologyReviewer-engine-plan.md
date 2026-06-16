# Deterministic Engine — Engineering Plan

**Azure Network Topology Expert Reviewer · the capability layer** · Date: 2026-06-03

## Why this exists (the verdict the evals earned)

Five eval rounds across six fixtures (24 model runs) — on both a frontier model and a verified-pinned Haiku — showed the skill prose at **parity with an unaided model, never a capability win**. The conclusion is settled: the capability in this product does not come from prompt text. It comes from computing reachability **deterministically in code**. This plan builds that engine.

The three skills are not discarded — they become this engine's specification, its test oracle, its RAG knowledge, and its consistency layer (see the reuse section). But the smarts live here, in Go. The architecture is unchanged from the build plan — "deterministic engine, LLM at the edges." The evals simply proved why that line was right.

## The one rule this engine enforces

Anything that is graph math — CIDR overlap, NSG precedence, effective-route resolution, the four reachability gates, firewall DNAT, AVNM source-scope, peering transitivity, severity-from-reachability — is a **pure, unit-tested Go function, never an LLM call**. The model is invoked only at three edges: explaining a finding in natural language, synthesizing a RAG-grounded recommendation, and translating architect intent into a topology spec. If a reachability verdict ever depends on a model call, the design is wrong.

This is precisely what the evals showed the model getting *inconsistently* right. In code it is right every time, and provably so.

## Stack and shape

Go MCP server, per the workspace standard and the `mcp-go-server-building` skill: `github.com/mark3labs/mcp-go`, Go 1.25/1.26, stdio + HTTP transports. Scaffolded and reviewed with the `aara-mcp-server-builder` agent; hardened with `mcp-go-guardrails-and-safety`; threat-modeled with `mcp-go-threat-modeling`.

```
azure-nettopo-engine/
  cmd/server/            # MCP server entrypoint (stdio/http)
  internal/
    graph/               # topology model: Node, Edge, kinds, the typed graph — Azure-shaped in v1, cloud-neutral later (see Risks)
    analyze/             # the deterministic core — gates, reachability, severity (pure funcs)
    cost/                # fixed (Retail Prices API) + variable (flow-log traffic) forecast
    generate/            # spec -> vetted module -> Terraform render + self-validate
    adapter/azure/       # Resource Graph + Network Watcher -> graph
    adapter/aws/         # (later) a second adapter onto the same graph
    mcp/                 # tool handlers: get_topology, analyze_risks, simulate_change, forecast_cost, generate_topology
    identity/            # read-only managed identity; JWT bearer to AskAT&T for the three edge LLM calls
  testdata/fixtures/     # the eval fixtures + answer keys, now golden tests
  go.mod
```

## The deterministic core (`internal/analyze`) — what the skill "knew", now code

Each item below is a function with table-driven tests. The skill's reference files are the **spec** for these — `nsg-route-evaluation.md` is the design doc for `edgeOpen()`, `reachability-and-severity.md` for `severity()`. That is the reuse: the prose we wrote and the eval cases we built become the specification and the test suite for the code.

- `cidrOverlap(a, b)` — address-space overlap.
- `nsgVerdict(effectiveRules, flow)` — priority-ordered first-match, defaults always present; **surfaces VNet-wide reachability when no `DenyVnetInBound` sits above the default `AllowVnetInBound` (65000)** — the iteration-1 miss, now a hard rule.
- `adminRuleVerdict(avnmRules, flow)` — Gate 1, evaluated before NSGs, **source-scope aware**: an `Internet`-tag `Deny` does not close intra-VNet/peered paths — the iteration-2 miss, now code.
- `nextHop(effectiveRoutes, dst)` — longest-prefix; `None` ⇒ no edge (the black-hole trap); `VirtualAppliance` ⇒ trace continues from the NVA.
- `dnatPaths(firewall)` — inbound DNAT publishes a backend with no public IP — the iteration-3 fix.
- `peeringPath(graph, src, dst)` — non-transitive by default; only via NVA forwarding, gateway transit, or AVNM connectivity.
- `edgeOpen(...)` — composes the gates into one open/closed verdict.
- `reachable(graph, from)` — BFS over open edges (exposure from the `Internet` node; segmentation across tiers; blast radius).
- `severity(path, blastRadius, sensitivity)` — the matrix; severity is a property of a reachable path, with a **latent tier** for "one change from critical".

Every one of these is something the model did *inconsistently* across the evals. In Go they are deterministic and pinned by the fixture corpus.

## The Azure adapter (`internal/adapter/azure`)

- **Inventory:** Azure Resource Graph (KQL) across subscriptions / management groups → graph nodes and edges (the `resource-graph-ingest.md` queries, now Go calls).
- **Evaluated truth:** Network Watcher effective security rules + effective routes per NIC; topology; next-hop and IP-flow-verify as verification oracles.
- **Identity:** read-only managed identity (Reader + Network Watcher data plane). The engine holds **no** write path. The only outbound model calls (the three edges) go to AskAT&T via JWT bearer, per the auth decision already taken.

## The MCP interface (`internal/mcp`)

| Tool | Deterministic? | Notes |
|---|---|---|
| `get_topology` | yes | adapter → graph |
| `analyze_risks` | **yes — no model in the path** | the core engine |
| `simulate_change` | yes | apply a delta to the graph, re-run analyze |
| `forecast_cost` | prices exact + banded | Retail Prices API + flow-log traffic basis |
| `generate_topology` | spec is LLM; render + validate are deterministic | module-select; **validate through `analyze_risks` before emit**; PR-only |
| `compare_topology` | yes | golden vs actual graph diff — drift detection (roadmap) |
| `attack_paths` | yes | multi-hop reachable paths + blast radius; enriched by Defender (roadmap) |

AskAT&T Workflows (UI), CI/CD, and the Azure Cost Optimizer all consume these tools — the "reusable interface" the use case promised, finally real instead of aspirational.

## Testing is the moat — and the thing the skill could not give you

The eval corpus we built — six fixtures with hand-checked answer keys, including the precision traps — becomes the engine's **golden regression suite** as Go table-driven tests with exact assertions. Precision and recall are then measured deterministically on every commit, in CI, with a `mcp-go-guardrails-and-safety`-style gate adapted to the analyzer. This is the strength evidence the skill's eval could only approximate: the engine matches the answer key exactly or the build fails. `test-engineering` and `ai-evaluation-harness` own this.

## Phasing (forced sequence; assets named; no invented dates)

| Phase | Build | Driven by | Effort |
|---|---|---|---|
| P0 | graph model + Azure adapter + in-memory store | `azure-service-mapping`, `azure-data-tier-design` (graph store when persistence is needed) | L |
| P1 | analysis core (gates + DNAT + reachability + severity) + `get_topology`/`analyze_risks` + golden tests from the fixture corpus | `mcp-go-server-building`, `mcp-go-guardrails-and-safety`, `test-engineering`, `aara-mcp-server-builder` | XL |
| P2 | `simulate_change` + `forecast_cost` (Retail Prices API + flow-log traffic) | `aara-azure-cost-reviewer`; the cost-forecasting skill as spec | L |
| P3 | `generate_topology` (spec → vetted module → Terraform → self-validate → PR) | `aara-senior-microservices-architect`; the iac-generation skill as spec | XL |

The sequence is forced exactly as before: the analyzer (P1) is the keystone that both P2 and P3 call. Effort is T-shirt only — calibrate to team capacity before any date is quoted.

## What this does to the three skills (nothing is wasted)

- They are the **spec** for the engine's functions (reference files → function design docs).
- They are the **test oracle** (eval fixtures → golden tests).
- They remain the **RAG knowledge** the engine's recommendation edge retrieves against (Ask Docs).
- They are the **consistency layer** for any human or Claude agent doing this work *before* the engine is wired up.

The skill work built the specification and the test suite for the engine. That is why it was worth doing — and why the engine is the next step, not a restart.

## First concrete step

Scaffold `azure-nettopo-engine` with `aara-mcp-server-builder`; port `edgeOpen()` and `reachable()` from the `nsg-route-evaluation.md` / `reachability-and-severity.md` specs; load the fixtures as table-driven golden tests; wire `analyze_risks` over stdio. That v0 — engine plus golden tests passing — is the first artifact that does what five iterations of prose could not: get the reachability verdict right every time, provably.

## Roadmap extensions (from review feedback)

Four capabilities, in priority order — all of them graph work on this same engine, not new skill prose:

1. **Drift detection (build first).** Add `compareTopology(golden, actual)` and a third review mode — current / proposed / **intended**. The intended state is nearly free: the IaC from the generation skill and the AVNM security-admin rules *are* the golden topology. Drift = shadow peering (an edge in actual but not golden), segmentation drift (a reachable path golden forbids), route/NSG drift from standard. Deterministic, highest value, lowest risk. Caveat: it only works where an intended-state source exists.
2. **Multi-hop network attack-paths.** Extend `reachable()` to enumerate full paths (Internet → VM → jumpbox → AKS → db) with blast radius. **Boundary:** the engine owns the *network* attack layer; identity, vulnerability, and lateral-movement exploitability come from **Microsoft Defender / Security Exposure Management** — consume that signal, do not rebuild it. Rebuilding Defender/Wiz is the anti-pattern.
3. **Executive risk reporting.** A rollup of the engine's structured findings (counts by severity, subscriptions impacted, exposed assets, SOC 2 / ISO control mapping) — a *deliverable shape*, not a new persona, rendered by the existing **Executive Narrative Advisor** persona. Any remediation-effort figure must be derived from finding type and count with a stated basis, never asserted.
4. **Design-mode (intent → validated topology).** The P3 generation phase already covers this: intent → spec → vetted **CAF / Azure Landing Zone** modules → Terraform → **validated through `analyze_risks` before emit** → PR + ADR. **Boundary:** compose the landing-zone accelerators Microsoft ships; do not reinvent the topology. The value is the validation loop, not the model's design flair.

## Risks to design for now

- **ARG / Network Watcher coverage at scale** — the effective-rule/route APIs are per-NIC; batch and cache (resolve once per NIC per run).
- **Pre-deployment topology** has only declared config — run the same engine, flagged "declared, not effective".
- **Defender for Cloud overlap** — consume its attack-path/exposure signal; the engine's differentiator is the structured verdict, the latent tier, and the org standards, not re-deriving Defender.
- **Cloud-neutral discipline — a debt, not yet a property.** v0's `internal/graph` and `internal/analyze` are currently Azure-shaped (AVNM, Azure Firewall, `effectiveSecurityRules`, `AllowVnetInBound` by name). A genuinely neutral core — Azure types behind the adapter only — must be introduced *before* the AWS adapter, or that adapter becomes a rewrite. Today the property is aspirational; the code does not yet hold it.

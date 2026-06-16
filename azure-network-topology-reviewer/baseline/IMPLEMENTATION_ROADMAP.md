# Azure Network Topology Reviewer — Implementation Roadmap

**Status:** Active · **Date:** 2026-06-03 · **Reads with:** `TARGET_ARCHITECTURE.md` (the *what*) and `../NetworkTopologyReviewer-build-plan.md` (the *why*)

---

## How this roadmap is built

Two principles:

1. **The deterministic engine is the keystone — ship it first, proven on fixtures.** Reachability and severity
   are graph algorithms, not LLM outputs. That core must be correct before any LLM, adapter, or transport is added.
2. **Measure from day one.** Golden fixtures and eval gates exist before the adapters that feed them, so every phase
   has a testable exit, not a vibe.

Phases are **outcome-defined**: each ends at a state you can test and retires a named risk. The LLM is at the edges
(explain, recommend, intent→spec) — it never computes reachability or severity.

Effort is T-shirt-sized (S / M / L / XL). Absolute dates need a staffing baseline.

## Phase map

```
Phase 0 (DONE)                Phase 1 (NOW)                Phase 2           Phase 3
Analysis Engine Proven    ──►  Azure Adapter + MCP v1  ──►  Cost Sim    ──►  Design Gen
  Go + Python engine            get_topology                simulate_change   generate_topology
  5/5 golden fixtures           analyze_risks               forecast_cost
  go vet clean                  LLM explain layer
                                Entra auth + CI
```

Cross-cutting through **every** phase: golden-fixture eval gate, determinism, read-only identity,
no LLM in the severity/reachability path, PR-only write path.

---

## Phase 0 — Analysis Engine Proven · ✅ DONE (2026-06-03)

**Goal:** Prove the deterministic reachability/severity engine on a golden fixture corpus before
building the live adapter or any transport.

**What was built:**
- `engine/reference/analyze.py` — Python reference implementation (5/5 fixtures)
- `engine/go/` — Go production port in planned package layout (5/5 fixtures, `go vet` clean, Go 1.25)
- `engine/go/internal/graph/model.go` — cloud-neutral `Fixture` type + `Load()` from JSON
- `engine/go/internal/analyze/analyze.go` — deterministic `Analyze()` function
- Five golden test fixtures covering: internet exposure, transitive peering, CIDR overlap, DNAT multi-hop,
  black-hole + AzureCloud tag cases

**Risks retired:** "Can we compute reachability deterministically without an LLM?" — YES, proven.

---

## Phase 1 — Azure Adapter + MCP v1 (review mode) · [XL] · **NOW**

**Goal:** Live Azure topology flows end-to-end through the proven engine and is exposed via the MCP interface.
This is the product that earns adoption — ship it narrow and correct.

**Exit criteria:**
- Adapter materialises a real sandbox subscription's topology into `graph.Fixture` matching `az network` spot-check
- `analyze_risks` MCP tool returns the same verdicts as `engine/go/` running on that fixture
- Precision/recall gate passes on the eval fixture set (false positives are the adoption killer)
- A senior architect accepts the report on a real read-only subscription

---

## Phase 2 — Cost-Aware Simulation · [L]

**Goal:** Forecast the security + cost impact of a proposed change before it ships.

**Exit criteria:**
- Fixed-cost delta exact against billing cross-check
- Variable-cost forecast within stated tolerance band on known-change set
- Simulated-graph analysis matches sandbox deployment result

---

## Phase 3 — Design Generation · [XL]

**Goal:** Turn architect intent into a validated topology PR. The riskier half — gated hard.

**Exit criteria:**
- Generated topology passes Phase 1 analyzer with zero high-severity findings before emit
- Terraform PR round-trips through CI cleanly
- Human approves and applies — the agent does not

---

## Locked decisions (apply to every phase)

| Decision | Choice | Rationale |
|---|---|---|
| Engine strategy | Deterministic Go core, LLM at edges | Reachability is graph arithmetic — not an LLM problem |
| Auth | Managed Identity + RBAC (Reader + data-plane NW read) | Read-only; no secrets in code |
| MCP ingress | Container Apps built-in Entra auth | No APIM — redundant under AskAT&T |
| Model access | AskAT&T (client-credentials, JWT bearer) | AT&T standard; per-user authz at MCP ingress |
| Container registry | JFrog Artifactory | AT&T standard — never ACR |
| Write path | PR via GitHub Actions + OIDC only | Agent never holds apply permission |
| AWS adapter | Phase 1+ only | After Azure P1 stable; same graph model |

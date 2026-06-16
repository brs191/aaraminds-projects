# Azure Network Topology Reviewer

A deterministic network topology analysis engine for Azure — reachability-based severity, exposed via
an MCP server for AskAT&T workflows, CI/CD, and the Azure Cost Optimizer peer agent.

## Status

| Phase | Title | Status |
|---|---|---|
| **Phase 0** | Analysis Engine Proven | ✅ ACCEPTED (2026-06-03) |
| **Phase 1** | Azure Adapter + MCP v1 | ✅ ACCEPTED WITH CONDITIONS (2026-06-12) |
| **Phase 2** | Cost-Aware Simulation | ⚠️ PARTIAL — Steps 2.1–2.4 done; 2.5–2.6 pending |
| **Phase 3** | Design Generation | ✅ ACCEPTED WITH CONDITIONS (2026-06-13) |
| **Phase 4** | Enterprise Topology Visualization | ⚠️ IN-SESSION SCOPE COMPLETE (26/26 eval PASS, 3 audits); live discovery/Go-port/pipeline deferred |

## Architecture in one sentence

A **deterministic graph engine** at the core — reachability, rules, severity computed without an LLM —
with the **LLM at the edges** (explain, recommend, intent→spec), exposed as MCP tools.

## What's built

```
engine/
  go/                  — Go 1.25 production engine (99/99 tests across 8 packages, go vet clean)
    internal/graph/    — graph.Fixture type (the contract the Azure adapter produces)
    internal/analyze/  — Analyze() — deterministic 4-gate reachability + severity
    adapter/           — Azure adapter: Resource Graph + Network Watcher → graph.Fixture
    mcp/               — MCP server: get_topology, analyze_risks, format_report, generate_topology
    renderer/          — markdown + drawio output (drawio peering edges: see Phase 4 RC-1…RC-4)
    simulator/ forecast/ — Phase 2 simulate_change + forecast_cost engines (MCP wiring pending)
    generator/         — Phase 3 Terraform projection + ValidateBeforeEmit + PR workflow
  reference/           — Python reference implementation (same fixtures, cross-check)
```

## What's next (Phase 4 — Enterprise Topology Visualization)

Triggered by the `ref-topology/generated_antr.pdf` failure (near-zero connectivity, every node
"Clean") vs. the human reference `ref-topology/BCLM-Revised-8June2026.drawio` (288 edges).
Strategy: **separate the map from the risk** — adopt OSS for discovery + layout, keep antr's
`Analyze()` engine as the severity overlay painted on top.

```
phase-4/
  design/VISUALIZATION_MODEL.md  — design + OSS decision + root causes (RC-1…RC-4)
  README.md                      — status + step table
  (4.1) pilot CloudNetDraw + Network Insights vs BCLM
  (4.3) multi-subscription discovery (mgmt-group scope, MI/OIDC)   ← fixes RC-1
  (4.4) severity overlay: Analyze() findings → node colour          ← fixes RC-2 + RC-4
  (4.5) ELK/D2 layout + external boundary nodes                     ← fixes RC-3
  (4.6) Azure Function → Confluence auto-publish + version diff
```

Adopt (fork + vendor) **CloudNetDraw** (MIT) for discovery + hub-spoke layout; **ELK** (via D2)
for readable layout; **Network Insights Topology** as the ground-truth cross-check.

## Getting started

```bash
# Verify the engine is green
cd engine/go
go test ./...    # 99/99 across 8 packages (Go 1.25)

# Continue Phase 4
# Open IMPLEMENTATION_PLAYBOOK.md → Phase 4, Step 4.1
# Read phase-4/design/VISUALIZATION_MODEL.md
```

## Key decisions

| Decision | Choice |
|---|---|
| Severity computation | Always in Go `Analyze()` — never the LLM |
| Container registry | JFrog Artifactory (AT&T standard — never ACR) |
| MCP ingress auth | Container Apps Entra (no APIM) |
| Model access | AskAT&T via JWT bearer |
| Write path | PR via GitHub Actions + OIDC only |

## Documentation

| Document | Purpose |
|---|---|
| `IMPLEMENTATION_PLAYBOOK.md` | Step-by-step guide with agent prompts + validation (Phases 0–4) |
| `baseline/IMPLEMENTATION_ROADMAP.md` | Phase map + locked decisions |
| `baseline/TARGET_ARCHITECTURE.md` | Component architecture reference |
| `phase-0/FINDINGS_MEMO.md` | Engine proof + locked design decisions |
| `phase-1/PHASE_1_ACCEPTANCE_MEMO.md` | Adapter + MCP v1 acceptance (G1–G5) |
| `phase-3/PHASE_3_ACCEPTANCE_MEMO.md` | Topology generation acceptance (G1–G5) |
| `phase-4/design/VISUALIZATION_MODEL.md` | Enterprise visualization design + OSS decision + root causes |
| `phase-4/PHASE_4_ACCEPTANCE_MEMO.md` | Phase 4 acceptance (in-session) + 4-round audit trail + engineering fixes |
| `phase-2/PHASE_2_STATUS.md` | Phase 2 de-ambiguation (engines done; MCP wiring + acceptance pending) |
| `AGENT_ROSTER.md` | Which `aara-*` agents exist and where (engineering pack vs project-delivery) |
| `.github/workflows/engine-ci.yml` | CI: Go test, Python reference + V4-07, twin-drift, diagram-eval gate (required), Phase-3 generator tests |
| `engine/twin_drift_check.py` | Asserts Python reference == Go engine on every shared fixture |
| `NetworkTopologyReviewer-architecture.md` | Full Mermaid architecture diagram |
| `NetworkTopologyReviewer-build-plan.md` | Detailed phase requirements |

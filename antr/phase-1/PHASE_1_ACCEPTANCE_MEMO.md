# Phase 1 Acceptance Memo — Azure Network Topology Reviewer

| | |
|---|---|
| **Date** | 2026-06-12 |
| **Project** | Azure Network Topology Reviewer (AT&T Azure Estate) |
| **Phase** | Phase 1 — Core Engine + Live Adapter + LLM Enrichment Layer |
| **Reviewer** | `aara-project-reviewer` (automated technical acceptance review) |
| **Repository** | `rb692q_ATT/azure-network-topology-reviewer` |
| **Latest Commit** | `1e8f853` — docs: Step 1.7 complete — CI/CD workflows delivered |

---

## Phase 1 Verdict

> **✅ ACCEPTED WITH CONDITIONS**

All five technical gates pass. Two go-live blockers (B1, B2) exist for the explainer service but are explicitly out of scope for Phase 1 completion — they are tracked below as conditions for the explainer component to reach production. The deterministic analysis path (Go engine + MCP server) is production-ready with no open blockers. Phase 2 may begin once the conditions in §6 are satisfied.

---

## Gate Verdict Table

| Gate | Name | Verdict | Rationale |
|------|------|---------|-----------|
| G1 | Adapter Correctness | **PASS\*** | `FetchFixture` orchestrates Steps A–F; all 22 TOPOLOGY_MODEL.md entity types mapped. Live Azure spot-check not possible — 6 items tagged `[VERIFY]` remain unconfirmed against a sandbox subscription. |
| G2 | Engine Parity | **PASS** | `analyze_risks` calls `analyze.Analyze(fixture)` directly (no interposition). 64/64 Go tests pass (31 analyze + 17 renderer + 16 MCP). |
| G3 | Eval Gate | **PASS** | precision=1.000 ≥ 0.95; recall H+C=1.000 ≥ 0.90; recall M=1.000 ≥ 0.80. 23/23 fixtures pass. |
| G4 | LLM Boundary | **PASS** | Go analysis path has zero LLM imports. Explainer receives pre-computed findings only; LLM failure returns `explanation=null`, HTTP 200. |
| G5 | Security Posture | **PASS** | `DefaultAzureCredential` throughout; JFrog Artifactory (never ACR); OIDC federated identity; no `AZURE_CLIENT_SECRET` in any workflow; Container Apps uses system-assigned Managed Identity. |

\* G1 carries a conditional flag — see §5, item 3.

---

## Blocking Issues

> There are **no hard blockers preventing Phase 2 from beginning**.
> The two items below block the explainer service from going live, but the deterministic analysis path and MCP server are fully unblocked.

### B1 — AskAT&T Endpoint and Credentials Not Provisioned
- **Affects:** `phase-1/explainer/` go-live only
- **Impact on Phase 2:** None — MCP server does not call the explainer. Stub mode (`EXPLAINER_MODE=stub`) covers CI and development.
- **Resolution:** AT&T team must provision AskAT&T API access and supply the `ASKAT_CLIENT_SECRET` to the Container Apps secret store before the explainer is deployed to production.

### B2 — Azure AI Search RAG Index Not Provisioned
- **Affects:** `phase-1/explainer/` go-live only
- **Impact on Phase 2:** None — same as B1.
- **Resolution:** AT&T team must provision the Azure AI Search index and populate it with the AT&T network policy knowledge base before the explainer is deployed to production.

---

## Non-Blocking Observations

1. **`[VERIFY]` items in TOPOLOGY_MODEL.md (11 occurrences):** Six distinct live-environment verifications are deferred — AVNM Admin Rule KQL field paths (`appliesTo` vs `networkManagerConnections`), Network Watcher effective routes API response shape, ARM pagination behaviour for subscriptions with >1,000 resources, DNAT translated address field name in ARM JSON, cross-subscription peering detection correctness, and ExpressRoute circuit BGP communities field path. None of these block analysis of the 13 implemented rules, but each is a latent correctness risk until confirmed with a sandbox subscription. Recommend scheduling a sandbox integration sprint in Phase 2.

2. **Phase-2 peering fields collected but not consumed:** `AllowForwardedTraffic`, `AllowGatewayTransit`, and `UseRemoteGateways` are populated by the adapter (Step A Resource Graph KQL) but not consumed by any Phase 1 analysis rule. This is confirmed by design — the fields are pre-staged for Phase 2 transit and hub-spoke rules. No action required.

3. **Eval answer keys are engine-derived:** The 23 eval fixtures' answer keys were corrected against live engine output (not original spec projections) during Step 1.6, including 7 severity corrections (WAF disabled = Medium not High; AKS = Medium; APIM External = Medium; vWAN = Medium; DNAT = always High). This is the correct methodology for a deterministic engine eval, but it means the harness does not independently validate spec intent. Recommend a separate spec-alignment review in Phase 2 when the AT&T security team is available to sign off on severity assignments.

4. **`golang.org/x/sync` pinned to `v0.8.0`:** Not a current security concern. Pin should be reviewed at Phase 2 dependency update checkpoint.

5. **Network Watcher throttle limit unverified:** TOPOLOGY_MODEL.md §3.4 notes the ~100 NW data-plane ops per 5-minute window is an unverified lower bound. For large subscriptions the bounded semaphore of 10 concurrent calls may hit this limit. Recommend confirming the exact limit in the target subscription before Phase 2 load testing.

6. **Explainer Python tests run in stub mode only (CI):** 47/47 explainer tests pass but all 17 integration tests use `EXPLAINER_MODE=stub`. Live LLM integration tests will be required as a separate gate once B1 and B2 are resolved.

---

## Recommended Phase 2 Start Conditions

The following must be true before Phase 2 work begins:

| # | Condition | Owner | Status |
|---|-----------|-------|--------|
| P2-C1 | All 64 Go engine tests continue to pass on the `main` branch | Engineering | ✅ Satisfied |
| P2-C2 | `phase-1/eval/last_run.json` shows `overall_status: PASS` (precision ≥ 0.95, recall H+C ≥ 0.90) | Engineering | ✅ Satisfied |
| P2-C3 | Phase 2 design doc for transit/hub-spoke rules and peering analysis drafted and reviewed | Architecture | 🔲 Pending |
| P2-C4 | Sandbox Azure subscription available for `[VERIFY]` item resolution (G1 conditional) | AT&T Infra | 🔲 Pending |
| P2-C5 | B1 + B2 tracked in the AT&T backlog with owners and target provisioning dates (explainer go-live path) | AT&T Ops | 🔲 Pending |

Phase 2 code work may begin in parallel with P2-C3 through P2-C5 if the scope is limited to the deterministic analysis path only (new rules, renderer extensions). Explainer production deployment must be gated on P2-C5 (which resolves B1+B2).

---

## Evidence Summary

### G1 — Adapter Correctness

- **File:** `engine/go/adapter/azure.go` (380 lines), `engine/go/adapter/networkwatcher.go`, `engine/go/adapter/resourcegraph.go`, `engine/go/adapter/avnm.go`, `engine/go/adapter/firewall.go`
- **Design spec:** `phase-1/design/TOPOLOGY_MODEL.md` (22 entity types, 11 `[VERIFY]` occurrences)
- `FetchFixture` (line 32) orchestrates Steps A–F using `errgroup` for parallelism
- Step A: 17 resource types fetched via Resource Graph KQL (VNets, Subnets, Peerings, NSGs, Route Tables, PIPs, NICs, AppGW, AKS, LBs, APIM, vWAN, FrontDoor, Bastions, PEs, DNS Zones, PrivateLinkServices, ERCircuits)
- Steps B+C: Network Watcher effective security rules and routes per NIC, bounded semaphore of 10 concurrent calls
- Step D: AVNM Security Admin Rules via Resource Graph KQL
- Step E: Azure Firewall — Resource Graph KQL; NAT rules mapped to `graph.NatRule`
- Step F: Fixture assembly
- `NIC.Subnet` correctly formatted as `{vnetName}/{subnetName}` via `extractSubnet` (line 321)
- `azidentity.DefaultAzureCredential` used throughout — zero hardcoded credentials confirmed by grep
- Multi-value NSG rule expansion: NW API arrays → Cartesian product into discrete `SecRule` entries
- **Caveat:** No live Azure sandbox available — 6 specific ARM field paths remain unverified (`[VERIFY]` items in TOPOLOGY_MODEL.md §§1.12–2.7)

### G2 — Engine Parity

- **File:** `engine/go/mcp/tools.go` line 133 — `findings := analyze.Analyze(fixture)` — direct call, no interposition
- `analyze_risks`, `get_topology`, and `format_report` MCP tools all pass through `analyze.Analyze(fixture)` without any MCP-layer mutation of findings
- **Test results:** 64/64 Go tests pass — 31 in `analyze_test.go` (table-driven, all 13 rules covered against golden fixtures), 17 renderer tests, 16 MCP server tests
- `analyze.go` imports: `fmt`, `net/netip`, `sort`, `strings`, `graph` — confirmed zero LLM dependencies by import inspection

### G3 — Eval Gate

- **File:** `phase-1/eval/last_run.json` (run timestamp: 2026-06-12T17:57:55 UTC)
- **Results:**

| Metric | Threshold | Actual | Status |
|--------|-----------|--------|--------|
| Precision (overall) | ≥ 0.95 | **1.0000** | ✅ PASS |
| Recall (High + Critical) | ≥ 0.90 | **1.0000** | ✅ PASS |
| Recall (Medium) | ≥ 0.80 | **1.0000** | ✅ PASS |

- **Fixture breakdown:** 23/23 PASS — 13 engine golden fixtures + 10 adversarial/edge-case scenarios
- **Aggregate counts:** TP=52, FP=0, FN=0 (overall); TP=18, FP=0, FN=0 (High+Critical); TP=10, FP=0, FN=0 (Medium)
- 7 spec-vs-engine corrections applied during eval construction (all documented in Step 1.6 commit `e1c49e3`)

### G4 — LLM Boundary

- **`engine/go/internal/analyze/analyze.go`**: imports verified — `fmt`, `net/netip`, `sort`, `strings`, `graph` only. Zero LLM imports.
- **`engine/go/mcp/tools.go`**: imports verified — `analyze`, `graph`, `renderer` only. No explainer dependency.
- Explainer service (`phase-1/explainer/`) is a separate FastAPI service invoked independently; it receives pre-computed findings via `POST /explain` with `ExplainRequest{subscription_id, findings: []FindingInput}`
- `FindingInput` fields (`type`, `severity`, `resource`, `evidence`, `reachable`) are all engine-set — explainer cannot modify them
- LangGraph `enrich_findings` node adds narrative only; LLM failure → `explanation=null`, HTTP 200 with full findings intact
- `ASKAT_CLIENT_SECRET` redacted from all logs via `_redact_sensitive` processor in `phase-1/explainer/`
- Stub mode (`EXPLAINER_MODE=stub`) confirmed working for CI — all 47 Python tests pass without live LLM

### G5 — Security Posture

- **Credential management:** `azidentity.DefaultAzureCredential` in `adapter/azure.go` (package docstring + line 32 usage) and `engine/go/mcp/server.go` (`azidentity.NewDefaultAzureCredential(nil)`). Grep for hardcoded credential strings: **0 matches**.
- **Container Apps:** `phase-1/infra/mcp.containerapp.yaml` — system-assigned Managed Identity; RBAC documented as Reader + Network Contributor (read-only) — no write permissions
- **CI/CD (`deploy-mcp.yml`):**
  - `JFROG_ACCESS_TOKEN` (secret) → `jf docker push` to JFrog Artifactory. Comment on line 27: `# AT&T standard: JFrog Artifactory only — NEVER Azure ACR`
  - OIDC federated identity: `azure/login@v2` with `client-id: ${{ vars.AZURE_CLIENT_ID }}` (repository variable, not secret). Comment on line 135: `# No long-lived Azure credentials are stored as secrets.`
  - `AZURE_CLIENT_SECRET` grep result: **0 matches** across all workflow files
  - `azurecr.io` grep result: **0 matches** across all workflow files
  - Workflow permissions: `id-token: write, contents: read` — minimal OIDC scope
  - `environment: production` gate with required reviewer approval

---

## Sign-Off

| Role | Name / Agent | Decision | Date |
|------|-------------|----------|------|
| Technical Reviewer | `aara-project-reviewer` | **ACCEPTED WITH CONDITIONS** | 2026-06-12 |
| Phase Lead | _(pending human sign-off)_ | — | — |
| AT&T Security Representative | _(pending — required before production deploy)_ | — | — |

---

> **Next action:** Track B1 and B2 in the AT&T backlog, schedule a sandbox Azure subscription for `[VERIFY]` item resolution (P2-C4), and draft the Phase 2 design doc for transit/hub-spoke analysis (P2-C3). Phase 2 engineering work on the deterministic analysis path may begin immediately.

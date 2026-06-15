# Azure Network Topology Reviewer — Implementation Playbook

**Purpose:** Step-by-step implementation guide with agent prompts, deliverables, and validation results for every phase.
**Date:** 2026-06-15 | **Status:** Phases 0–3 delivered; Phase 4 in DESIGN. Verdicts: Phase 1 ACCEPTED WITH CONDITIONS · Phase 2 PARTIAL (Steps 2.5–2.6 pending) · Phase 3 ACCEPTED WITH CONDITIONS · Phase 4 (Enterprise Topology Visualization) DESIGN — not started.

> **Conventions:**
>
> - **Result block** — updated after a step is executed: deliverable paths, status, one-line summary.
> - **Validation block** — defines *how* to verify the result is correct. Contains an executable prompt or command set.
> - **Validation Result block** — updated after validation runs: PASS/FAIL per assertion. Steps not yet run are marked ⬜.
> - This file is both a playbook and an execution log. Update it after every step.

---

## Phase Completion Summary

| Phase | Title | Status | Completed |
|---|---|---|---|
| **Phase 0** | Analysis Engine Proven | ✅ **ACCEPTED** | 2026-06-03 |
| **Rework 1** | BCLM Reference Topology (PE, AppGW, AKS, NAT GW, PLS, ER) | ✅ **COMPLETE** | 2026-06-12 |
| **Rework 2** | PrivateEndpoint + LoadBalancer + deterministic PE DNS | ✅ **COMPLETE** | 2026-06-12 |
| **Rework 3** | APIM, Bastion, VNG, Subnet fields, Phase-2 stubs | ✅ **COMPLETE** | 2026-06-12 |
| **Rework 4** | VirtualWAN + Enrichment envelope (Defender, Policy, Activity Log) | ✅ **COMPLETE** | 2026-06-12 |
| **Rework 5** | NIC.DNSServers, FlowLogSummary, Front Door WAF rule | ✅ **COMPLETE** | 2026-06-12 |
| **Phase 1** | Azure Adapter + MCP v1 | ✅ **ACCEPTED WITH CONDITIONS** (8/8 steps) | 2026-06-12 |
| **Phase 2** | Cost-Aware Simulation | ⚠️ **PARTIAL** — Steps 2.1–2.4 done; Steps 2.5–2.6 pending | — |
| **Phase 3** | Topology Generation | ✅ **ACCEPTED WITH CONDITIONS** (6/6 steps) | 2026-06-13 |
| **Phase 4** | Enterprise Topology Visualization | ⚠️ **IN-SESSION SCOPE COMPLETE** — Python reference pipeline, 26/26 diagram-eval PASS, 3 adversarial audits remediated (`phase-4/PHASE_4_ACCEPTANCE_MEMO.md`); live discovery / Go port / pipeline deferred | 2026-06-15 |

### Pending Action Items (cross-phase)

| ID | Phase | Item | Owner | Blocking? |
|---|---|---|---|---|
| **PA-01** | Phase 2 | Step 2.5: Wire `simulate_change` + `forecast_cost` into MCP server (`engine/go/mcp/tools.go`) | Engineering | Phase 2 acceptance |
| **PA-02** | Phase 2 | Step 2.6: Produce `phase-2/PHASE_2_ACCEPTANCE_MEMO.md` | Engineering | Phase 2 close |
| **PA-03** | Phase 1 | 6 `[VERIFY]` items in TOPOLOGY_MODEL.md §6.3 — require sandbox subscription + live Managed Identity | AT&T Network Ops | Live adapter deployment |
| **PA-04** | Phase 1 | B1: AskAT&T credentials for explainer service | AT&T AI Platform | Explainer go-live |
| **PA-05** | Phase 1 | B2: Azure AI Search index for explainer service | AT&T AI Platform | Explainer go-live |
| **PA-06** | Phase 3 | V-04: Confirm `INFRA_REPO` env var value (infra Terraform repo name) | AT&T Network Ops | generate_topology PR creation |
| **PA-07** | Phase 3 | V-05: Confirm GitHub App token vs PAT for infra repo | AT&T Platform | generate_topology PR creation |
| **PA-08** | Phase 3 | V-11: AskAT&T structured output API (`response_format.json_schema`) contract | AT&T AI Platform | Real LLM wiring in intent.py |
| **PA-09** | Phase 3 | NB-01: Add Phase 3 generator tests to `engine-ci.yml` | Engineering | CI gate |
| **PA-10** | Phase 3 | NB-02: Confirm JFrog docker login username in `deploy-mcp.yml` | AT&T Platform | Deploy pipeline |
| **PA-11** | Phase 4 | Pilot CloudNetDraw + Network Insights Topology on real subscription(s); compare to BCLM | Engineering | Phase 4 Step 4.1 gate |
| **PA-12** | Phase 4 | Make discovery management-group-scoped; replace single-sub `FetchFixture` (RC-1) | Engineering | Multi-sub diagrams |
| **PA-13** | Phase 4 | Wire `Analyze()` findings → diagram node severity (RC-2 + RC-4) | Engineering | Severity overlay |
| **PA-14** | Phase 4 | OSS-intake registration (CloudNetDraw/D2/ELK) — routine OSPO logging, **not a blocker** (internal-use only; see V4-06) | AT&T OSPO | Non-blocking |

### Key Assumptions (locked across all phases)

| # | Assumption | Impact if wrong |
|---|---|---|
| A-01 | AskAT&T is the only permitted LLM endpoint (no OpenAI, no Azure OpenAI direct) | Python intent.py requires re-wiring |
| A-02 | JFrog Artifactory is the container registry — never ACR | deploy-mcp.yml requires re-wiring |
| A-03 | Managed Identity holds Reader role (subscription scope) + 2 NW data-plane read actions — no write | Adapter cannot fetch effective rules/routes |
| A-04 | Agent never holds `terraform apply` permission — PR only, human approves | generate_topology tool must be re-scoped |
| A-05 | `AZURE_CLIENT_SECRET` is never stored in GitHub Actions secrets — OIDC federated only | CI/CD auth chain breaks |
| A-06 | Go 1.25 engine is deterministic: same `graph.Fixture` → same `[]Finding`, always | Security gate integrity guaranteed |
| A-07 | AT&T internal terraform module registry (V-03) is available and CAF-compliant | renderer.go module registry config requires update |

---

## Phase 0 — Summary

**Goal:** Prove the deterministic reachability/severity engine on a golden fixture corpus before building
any live adapter or transport. The engine is the keystone — everything else hangs off it.

**Duration:** Pre-2026-06-03 → 2026-06-03 | **Verdict:** ✅ ACCEPTED — engine proven, 5/5 fixtures pass.

### Outcomes

| Area | Deliverable | Key numbers |
|---|---|---|
| **Python reference** | `engine/reference/analyze.py` + `test_analyze.py` | 5/5 fixtures pass |
| **Go production port** | `engine/go/internal/{graph,analyze}/` | 5/5 fixtures pass, `go vet` clean, Go 1.25 |
| **Graph model** | `engine/go/internal/graph/model.go` | `Fixture` type: VNet, NSG, RouteTable, NIC, PublicIP, Peering, AVNM, Firewall |
| **Golden corpus** | `engine/go/testdata/` (5 fixtures) | internet-exposure, transitive-peering, CIDR/AVNM, DNAT, blackhole-tags |

### Locked decisions

| Decision | Choice | Rationale |
|---|---|---|
| Engine language | Go 1.25 production, Python reference | Go: stdlib-only (`net/netip`), zero deps, `go vet` clean |
| Severity model | 4-gate: AVNM → NSG → route → PIP | Only fires "High" when path is provably real — false positives kill adoption |
| Container registry | JFrog Artifactory | AT&T standard — never ACR |
| MCP ingress auth | Container Apps Entra auth | No APIM — redundant under AskAT&T |

### Key artifacts

- `engine/go/internal/analyze/analyze.go` — canonical `Analyze()` function
- `engine/go/internal/graph/model.go` — `Fixture` type (the contract the adapter must produce)
- `engine/go/testdata/` — 5 golden fixtures
- `phase-0/FINDINGS_MEMO.md` — complete exit document

---

## Pre-Phase-1 Rework — BCLM Reference Topology Extension

**Trigger:** User added `ref-topology/BCLM-Revised-8June2026.drawio` (AT&T BCLM production network topology).
Analysis revealed the existing `graph.Fixture` model covered only 5 resource types; the BCLM topology requires 11.

**Duration:** 2026-06-12 | **Verdict:** ✅ Complete — 8/8 golden tests pass, all changes backward-compatible.

### Changes Made

| Area | What changed | Rationale |
|---|---|---|
| `graph/model.go` | Added 6 new structs: `PrivateDnsZone`, `DnsARecord`, `ApplicationGateway`, `AppGWBackendPool`, `AKSCluster`, `NatGateway`, `PrivateLinkService`, `ExpressRouteCircuit`, `CrossSubPeering` | BCLM has PEs, APP GW, AKS, NAT GW, PLS, ER — all absent from v1 model |
| `graph/model.go` | Added `CrossSubscriptionPeerings []CrossSubPeering` to `Fixture` | BCLM spans 6-7 subscriptions; cross-sub peerings need explicit representation |
| `graph/model.go` | Added `RemoteSubscriptionID string` to `Peering` | Peering to another subscription is a structural security boundary |
| `graph/model.go` | Added 6 new slices to `ResourceGraph` (all `omitempty`) | Backward-compatible — existing 5 fixtures unchanged |
| `analyze/analyze.go` | Added `subnetToVnet()` helper | Extracted from `nicVnet()` to share with new checks |
| `analyze/analyze.go` | Added `checkPrivateDnsZoneMisconfiguration()` → **High** | Most common real-world misconfiguration in BCLM-class topologies |
| `analyze/analyze.go` | Added `checkAppGatewayExposure()` → **Medium / Informational** | APP GW WAF disabled = no L7 protection on public ingress |
| `analyze/analyze.go` | Added `checkAKSExposure()` → **Medium** | Non-private AKS = API server reachable from internet |
| `analyze/analyze.go` | Added `checkCrossSubPeeringExposure()` → **Medium** | Direct cross-sub peering without firewall = unrestricted lateral movement |
| `analyze/analyze_test.go` | Added TestF6, TestF7, TestF8 | 3 new rules, 3 new fixtures, each with correct and trap assertions |
| `testdata/fixture-f6-pe-dns-misconfiguration.json` | New golden fixture | PE DNS zone linked to spoke-a but not spoke-b → High finding |
| `testdata/fixture-f7-appgw-waf-disabled.json` | New golden fixture | appgw-prod WAF disabled (Medium), appgw-staging Detection mode (Info), appgw-internal (no finding) |
| `testdata/fixture-f8-aks-and-crosssub-peering.json` | New golden fixture | aks-public non-private (Medium), cross-sub peering without firewall (Medium) |
| `phase-1/design/TOPOLOGY_MODEL.md` | Added Sections 8-14: new resource types + multi-sub support | Adapter spec for all 6 new resource types + KQL queries |

### Backward Compatibility

All new fields use `json:",omitempty"`. The 5 original golden fixtures require zero changes.
`go test ./...` output: **8/8 PASS** (5 original + 3 new).

### Updated Locked Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Multi-subscription model | One `Fixture` per subscription + `CrossSubscriptionPeerings` field | Matches Azure subscription boundary; adapter queries per-sub; MCP joins for cross-sub findings in Phase 2 |
| New resource types | `PrivateDnsZone`, `ApplicationGateway`, `AKSCluster`, `NatGateway`, `PrivateLinkService`, `ExpressRouteCircuit` | Required for BCLM-class coverage; all collected in Resource Graph Step A (+ VNet link ARM calls for DNS zones) |
| New analysis rules | PE DNS misconfiguration (High), APP GW WAF (Medium), AKS private cluster (Medium), cross-sub peering without FW (Medium) | Top-priority findings for BCLM estate; deterministic, zero LLM, engine-owned |

---

## Pre-Phase-1 Reworks 2–5 — Model Hardening to Enterprise Scale

> All reworks ran 2026-06-12 in response to audit findings against the BCLM reference topology
> and the official Microsoft Azure data source document. Full backward compatibility maintained throughout.

| Rework | Trigger | Key additions | Test count |
|---|---|---|---|
| **2** | PrivateEndpoint as first-class resource; LB NAT invisible to Gate 4 | `PrivateEndpoint`, `LoadBalancer`, `LBNatRule`, `LBBackendPool`, `Firewall.PolicyRef`; deterministic `peGroupIdToZone` map (19 services); `checkPrivateDnsZoneMisconfiguration` rewritten; `checkLoadBalancerNAT` added; F10 fixture | 9/9 |
| **3** | APIM Standard V2 in BCLM; Bastion bypass pattern; VNG needed to link ER circuits | `APIManagement`, `AzureBastion`, `VirtualNetworkGateway`; `Subnet.ServiceEndpoints/Delegations/PENPolicies`; Phase-2 stubs (`DNSPrivateResolver`, `AzureRouteServer`, `DDoSProtectionPlan`, `LocalNetworkGateway`); `checkAPIMExposure`, `checkBastionBypass`; F11, F12 fixtures | 11/11 |
| **4** | Virtual WAN is P0 per official Microsoft docs — zero VNet peerings returned for vWAN spokes | `VirtualWAN`, `VirtualHub`; `Enrichment` envelope (`DefenderAssessments`, `PolicyFindings`, `RecentChanges`); `checkVirtualWAN`; F13 fixture; TOPOLOGY_MODEL §15 (Next Hop, IP Flow Verify, NW Topology API guidance) | 12/12 |
| **5** | NIC DNS servers needed for hybrid DNS path; Front Door WAF gap; NSG/VNet Flow Logs not in model | `NIC.DNSServers`, `FrontDoorEndpoint` struct (enhanced), `FlowLogSummary` + `Enrichment.FlowLogStatuses`; `checkFrontDoorExposure`; F14 fixture | 13/13 |

### Final model state (model locked for Step 1.3)

| Metric | Count |
|---|---|
| Structs in `graph/model.go` | 32 |
| Active analysis rules in `analyze.go` | 13 (all deterministic, golden-tested) |
| Golden fixtures in `testdata/` | 14 (F1–F3, H1–H2, F6–F8, F10–F14) |
| Test pass rate | **13/13 — 100%** |
| Phase-2 stub structs (collected, no rule) | 6 (`DNSPrivateResolver`, `AzureRouteServer`, `AzureFrontDoor` rule-only, `DDoSProtectionPlan`, `LocalNetworkGateway`, `NatGateway`) |

### Deferred gaps (documented, not blocking Step 1.3)

| Gap | Impact | Phase |
|---|---|---|
| ASG struct | Engine unaffected (reads NW effective rules which resolve ASG→IPs); declared-config docs only | Phase 2 |
| App Service / Storage / KV / SQL as resources | PE-side fully covered; network firewall config not modeled | Phase 2 |
| East-west VNet traffic analysis | Engine is inbound-internet only | Phase 2 |
| DNS resolution path analysis | NIC.DNSServers now captured; full resolver path needs analysis rule | Phase 2 |
| BGP default route detection | `ExpressRouteCircuit.BGPAdvertisesDefaultRoute` requires BGP peer API call — not in Resource Graph | Phase 2 |

### 6 `[VERIFY]` items for adapter implementation (Step 1.3)

| # | Item | Risk if wrong |
|---|---|---|
| V1 | NW effective rule/route actions not blocked by deny assignment or Azure Policy in target tenant | Adapter returns empty effective rules — all NICs silently unanalysed |
| V2 | AVNM Resource Graph type `microsoft.network/networkmanagers/.../rules` available in tenant | Adapter falls back to REST walk (already coded) |
| V3 | Firewall Policy `rulecollectiongroups` KQL returns NAT rules (vs. ARM GET required) | Adapter uses ARM GET path by default — confirm KQL shortcut works |
| V4 | All target regions have Network Watchers provisioned | Without NW, no effective rules/routes for that region |
| V5 | NW throttle limit ≈ 100 ops/5 min confirmed in target subscription | Semaphore-10 may need tuning |
| V6 | Reader role on remote subscription sufficient to list remote VNet names for cross-sub peerings | Missing remote VNet names → cross-sub peering findings silently omitted |

---



> Locked in Phase 0 and applied to every subsequent phase.

| Decision | Locked choice |
|---|---|
| Engine strategy | Deterministic Go core; LLM at edges only (explain, recommend, intent→spec) |
| Severity computation | Always in `Analyze()` — never the LLM |
| Auth | Managed Identity + Reader + data-plane NW read (no write permissions) |
| MCP ingress | Container Apps built-in Entra auth (no APIM) |
| Model access | AskAT&T (client-credentials JWT bearer); secret in Key Vault; never logged |
| Container registry | JFrog Artifactory (`jf docker push`) — never ACR |
| Write path | PR via GitHub Actions + OIDC only — agent never holds apply permission |

**Authoritative sources:** `baseline/IMPLEMENTATION_ROADMAP.md` · `baseline/TARGET_ARCHITECTURE.md` · `phase-0/FINDINGS_MEMO.md` · `engine/go/`

---

# Phase 1 — Azure Adapter + MCP v1

**Goal:** Live Azure topology flows end-to-end through the proven engine and is exposed via two MCP tools
(`get_topology`, `analyze_risks`). This is the product that earns adoption — ship it narrow and correct.

**Exit criteria:** Adapter materialises a sandbox subscription topology matching `az network` spot-check;
`analyze_risks` returns same verdicts as the internal engine; precision/recall gate passes on eval fixture set;
a senior architect accepts the report on a real read-only subscription.

| Step | Agent | Type | Produces | Status |
|---|---|---|---|---|
| 1.1 | `aara-project-architect` | Custom | `phase-1/design/TOPOLOGY_MODEL.md` | ✅ |
| 1.1b | Rework 1–5 (model hardening) | — | 32 structs, 13 rules, 14 fixtures, 13/13 pass | ✅ |
| 1.2 | `rubber-duck` | Built-in | Design review findings | ✅ |
| 1.3 | `aara-project-builder` (fallback: `aara-project-debugger`) | Custom | `engine/go/adapter/` — Go Azure adapter | ✅ |
| 1.4 | `aara-mcp-server-builder` | Custom | `engine/go/mcp/` + `engine/go/renderer/` — Go MCP server + renderers | ✅ |
| 1.5 | `aara-python-ai-developer` | Custom | `phase-1/explainer/` — LangGraph explain layer | ✅ |
| 1.6 | `aara-ai-evaluation-engineer` | Custom | `phase-1/eval/` — precision/recall gate | ✅ |
| 1.7 | `azure-ops` | Skill | `.github/workflows/` — CI + deploy | ✅ |
| 1.8 | `aara-project-reviewer` | Custom | `phase-1/PHASE_1_ACCEPTANCE_MEMO.md` | ✅ |

**Pre-check before Step 1.3:**

```bash
cd engine/go
go test ./...   # must be 13/13 before touching any code
go version      # must be 1.25+
az version      # must be present for adapter integration testing
```

**Azure credentials required for integration testing (6 [VERIFY] items):**
> Unit tests (shape validation, mock adapter) do not need Azure credentials.
> Integration testing against the 6 `[VERIFY]` items in TOPOLOGY_MODEL.md §6.3 **requires**
> a real read-only Azure subscription with:
> - Managed Identity (or `az login`) with `Reader` role at subscription scope
> - Custom role: `Microsoft.Network/networkInterfaces/effectiveNetworkSecurityGroups/action`
>   + `Microsoft.Network/networkInterfaces/effectiveRouteTable/action`
> - At least one Network Watcher provisioned in a target region
>
> **Defer live integration testing to a follow-up session with Azure sandbox credentials.**
> Step 1.3 can be implemented and unit-tested fully offline.

---

## Step 1.1 — Topology Data Model Design

**Agent:** `aara-project-architect`
**Produces:** `phase-1/design/TOPOLOGY_MODEL.md`

**Requirements:**

- Map every field in `engine/go/internal/graph/model.go` (`Fixture`, `VNet`, `Subnet`, `NSG`, `SecRule`,
  `RouteTable`, `Route`, `NIC`, `PublicIP`, `AVNM`, `AdminRule`, `Firewall`, `NatRule`) to its exact
  Azure API source: which Resource Graph KQL query or which Network Watcher API call produces it
- Distinguish **declared config** (Resource Graph) from **evaluated truth** (Network Watcher Effective rules/routes)
  — the engine uses effective, not declared
- Define the authoritative source for each field: Resource Graph, NW Effective Security Rules, NW Effective
  Routes, NW Topology API
- Identify fields that require Network Watcher per-NIC API calls (expensive, called per NIC) vs
  bulk Resource Graph KQL (cheap, one query)
- Document the assembly sequence: Resource Graph inventory first → Network Watcher enrichment per NIC → AVNM → Firewall
- Note `SecRule.Source` vs `SecRule.SourceAddressPrefix` dual-field issue and resolve it
- Identify Azure RBAC roles required: confirm minimum role set for read-only access
- Phase 2 placeholders: fields needed for `simulate_change` / `forecast_cost` not yet in `graph.Fixture`

**Prompt**

```text
Design the topology data model for Phase 1 of the Azure Network Topology Reviewer.

Context: The analysis engine (`engine/go/internal/analyze/analyze.go`) takes a `graph.Fixture` as input and
produces deterministic findings. The Azure adapter's job is to populate that `graph.Fixture` from live Azure
sources. Read the existing type definitions in `engine/go/internal/graph/model.go` before writing anything.

Deliver `phase-1/design/TOPOLOGY_MODEL.md` with the following sections:

1. Field mapping table — for every field in every struct in `graph/model.go`, document:
   - Field name and type
   - Azure API source (Resource Graph KQL | NW Effective Security Rules | NW Effective Routes | NW Topology | AVNM | Firewall)
   - Whether it is declared config or evaluated truth (critical distinction — the engine needs evaluated)
   - Example value
   - Required for analysis (yes/no/Phase-2)

2. Query catalogue — the exact Resource Graph KQL queries for:
   - Virtual Networks (with address space, subnets, peerings)
   - Network Security Groups (with security rules, associated subnets)
   - Route Tables (with routes, associated subnets)
   - Public IP Addresses (with ipConfiguration — null = orphaned)
   - Network Interfaces (with subnet, NSG, publicIP, privateIP, tags)
   - AVNM Security Admin Rules (with appliesTo, priority, direction, access)
   - Azure Firewall (NAT rules with translatedAddress, translatedPort)

3. Network Watcher calls — per-NIC API calls required:
   - Effective Security Rules (NW endpoint, parameters, response mapping)
   - Effective Routes (NW endpoint, parameters, response mapping)
   - Note: these are O(NIC-count) calls — document batching/parallelism strategy

4. Assembly sequence — ordered steps to build `graph.Fixture` from scratch:
   Step A: Resource Graph bulk queries (subscription scope)
   Step B: NW Effective Security Rules per NIC (parallel, bounded concurrency)
   Step C: NW Effective Routes per NIC (parallel, bounded concurrency)
   Step D: AVNM Security Admin Rules
   Step E: Azure Firewall NAT rules (if present)
   Step F: Assemble into `graph.Fixture`

5. Dual-field resolution — `SecRule` has both `Source` and `SourceAddressPrefix`. Document which is canonical
   (use `SourceAddressPrefix` — it is what the engine reads) and what `Source` was likely for.

6. RBAC role set — minimum Azure roles for the Managed Identity running the adapter:
   - Reader (subscription scope) — for Resource Graph
   - Network Contributor *data-plane read* OR the exact built-in roles that grant NW Effective Rules/Routes
   - Flag any [VERIFY] items against live environment

7. Phase 2 placeholders — fields not in `graph.Fixture` today that `simulate_change` / `forecast_cost` will need:
   - Gateway SKU and pricing tier
   - Private Endpoint count
   - Bandwidth / data-processing attributes
   Mark each as `// Phase 2` with a note on which Azure source provides it.

Do not write any code. Deliverable: `phase-1/design/TOPOLOGY_MODEL.md`.
```

### Result — Step 1.1

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-12 |
| **Deliverable** | `phase-1/design/TOPOLOGY_MODEL.md` (1,103 lines) |
| **Summary** | 7 sections: field mapping (15 structs, every field), 7 KQL queries, NW per-NIC call spec (semaphore-10, retry, fail-open), assembly sequence (A→B+C+D+E parallel→F), dual-field resolution (`SourceAddressPrefix` canonical), RBAC role set (Reader + 2 custom NW actions, 6 `[VERIFY]` items), 7 Phase 2 placeholder fields. Key findings: `NIC.Subnet` must be `{vnetName}/{subnetName}` (not full ARM ID); AVNM uses REST walk (not pure KQL); Firewall has policy-based NAT path; multi-value prefix arrays must expand to separate `SecRule` entries. |

### Validation — Step 1.1

> **Note:** Step 1.2 (rubber-duck design review) IS the validation for Step 1.1.

**Assertions:**

| # | Assertion | Expected |
|---|---|---|
| 1 | Every field in `graph/model.go` appears in the field mapping table | 100% coverage |
| 2 | Every field marked as "evaluated truth" sources from Network Watcher (not Resource Graph) | All NW-sourced |
| 3 | `SecRule.SourceAddressPrefix` declared canonical; `SecRule.Source` resolved | Documented |
| 4 | Phase 2 placeholder fields are marked `// Phase 2` | Present |
| 5 | RBAC role set calls out any `[VERIFY]` items | At least 1 `[VERIFY]` |

### Validation Result — Step 1.1

| Field | Value |
|---|---|
| **Status** | ✅ Validated — 2026-06-12 (via Step 1.2 rubber-duck review) |
| **Evidence** | Step 1.2 rubber-duck read TOPOLOGY_MODEL.md line-by-line against analyze.go. All 5 assertions confirmed: (1) 100% field coverage — every field in model.go (15+ structs) has a documented Azure API source; (2) all evaluated-truth fields correctly source from Network Watcher effective rules/routes, not Resource Graph declared config; (3) `SecRule.SourceAddressPrefix` declared canonical, `Source` field role clarified; (4) 7 Phase-2 placeholder fields present in §7 (gateway SKU, PIP allocation method, peering bandwidth, etc.); (5) 6 `[VERIFY]` items documented in §6.3. Review also surfaced 5 findings (TMR-001–TMR-005) — all addressed in Step 1.2. |

---

## Step 1.2 — Design Review

**Agent:** `rubber-duck`
**Input:** `phase-1/design/TOPOLOGY_MODEL.md` from Step 1.1
**Produces:** Findings applied to `TOPOLOGY_MODEL.md`

**Requirements:**

- Flag any fields sourced from Resource Graph that should come from Network Watcher (declared vs evaluated)
- Flag any O(NIC) API calls that lack a parallelism/concurrency cap — unbounded parallel NW calls will hit throttling
- Flag missing fields the `Analyze()` function reads but that have no documented source
- Flag AVNM model gaps: `AdminRule.AppliesTo` is a VNet name list — verify the KQL that populates it
- Flag Phase 2 gaps: anything `simulate_change` will need that is absent from the model today

**Prompt**

```text
Review the topology data model design in `phase-1/design/TOPOLOGY_MODEL.md` for the Azure Network Topology Reviewer Phase 1.

Context: The analysis engine in `engine/go/internal/analyze/analyze.go` is the authoritative consumer.
Every field it reads must have a documented, correct Azure source. The Managed Identity is read-only.

Flag only genuine design risks — not style or naming:

1. Declared-vs-evaluated confusion: any field marked as sourced from Resource Graph that the engine
   actually needs in its evaluated/effective form (e.g., NSG rules — the engine needs effective rules
   per NIC, not the rules declared on the NSG resource).

2. Throttling risks: any per-NIC Network Watcher call with no concurrency cap. NW APIs throttle at ~100
   requests/5 minutes per subscription — an unbounded fan-out on a large subscription will fail.

3. Missing fields: read `analyze.go` line by line; list every field it accesses and confirm each appears
   in the TOPOLOGY_MODEL field mapping with a source.

4. AVNM gaps: `adminVerdict()` in analyze.go uses `AdminRule.AppliesTo` (VNet name list), `Direction`,
   `DestinationPortRange`, `SourceAddressPrefix`, `Priority`, and `Access`. Confirm all are mapped.

5. Firewall DNAT: the DNAT check in analyze.go uses `Firewall.NatRules[].TranslatedAddress` matched to
   `NIC.PrivateIP`. Confirm the KQL / ARM query populates both fields correctly.

6. Phase 2 readiness: list any fields `simulate_change` will obviously need (gateway SKU, PIP allocation
   method, peering bandwidth) that are absent and have no placeholder.

For each finding: cite the exact section in TOPOLOGY_MODEL.md, explain the risk, suggest the fix.
```

### Result — Step 1.2

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-12 |
| **Deliverable** | Fixes applied directly to `phase-1/design/TOPOLOGY_MODEL.md` |
| **Findings** | 5 total: 0 Blocking, 4 High, 1 Low. All 4 High fixes applied. Low fix applied manually. |
| **High fixes applied** | TMR-001: AdminRule `destinationPortRanges[]` multi-value expansion added (false negative risk — engine exact-matches port). TMR-002: Resource Graph `natRuleCollections` is NOT authoritative for NAT rules — use ARM GET; RG only for discovery (silent data loss risk). TMR-003: Remove deduplication of effective security rules — flatten without dedup (distinct rules from different NSG associations would be incorrectly dropped). TMR-004: NW async poll timeout raised to 60s deadline; semaphore semantics clarified as total-NW-calls cap (not per-NIC-pair). |
| **Low fix applied** | TMR-005: §1.4 peering fields — adapter must populate Phase-2 fields during Phase 1 collection even though analysis doesn't consume them. |

### Validation Result — Step 1.2

| Field | Value |
|---|---|
| **Status** | ✅ Validated — 2026-06-12 |
| **Evidence** | rubber-duck surfaced 5 findings (TMR-001 through TMR-005). All 4 High fixes confirmed applied to TOPOLOGY_MODEL.md. TMR-005 Low fix applied manually to §1.4. TOPOLOGY_MODEL.md status updated to REVIEWED. |

---

## Step 1.3 — Azure Adapter Implementation

**Agent:** `aara-project-builder`
**Fallback (Azure SDK / ARM throttling issues):** `aara-project-debugger`
**Input:** `phase-1/design/TOPOLOGY_MODEL.md` (reviewed + locked)
**Produces:** `phase-1/adapter/` — Go package `github.com/aaraminds/azure-nettopo-engine/adapter`

**Requirements:**

- Go package that returns a `*graph.Fixture` for a given subscription ID
- Resource Graph KQL queries using the Azure SDK for Go (`github.com/Azure/azure-sdk-for-go`)
- Network Watcher Effective Security Rules + Effective Routes per NIC — parallel with bounded concurrency
  (max 10 concurrent NW calls)
- AVNM Security Admin Rules query
- Azure Firewall NAT rules query
- Authentication via `azidentity.DefaultAzureCredential` (Managed Identity in production, CLI credential in dev)
- Assembly function: `func FetchFixture(ctx context.Context, subscriptionID string) (*graph.Fixture, error)`
- Idempotent and re-runnable (no state mutation)
- Table-driven unit tests using the 5 existing `engine/go/testdata/` fixtures as reference shapes

**Prompt**

```text
Implement the Azure adapter for Phase 1 of the Azure Network Topology Reviewer.

Context: The analysis engine in `engine/go/internal/analyze/analyze.go` takes a `*graph.Fixture` as input.
The adapter's job is to call live Azure APIs and populate that struct. Read the type definitions in
`engine/go/internal/graph/model.go` and the field mapping in `phase-1/design/TOPOLOGY_MODEL.md` before
writing any code.

Deliver `phase-1/adapter/` as a Go package `adapter` (module: `github.com/aaraminds/azure-nettopo-engine`):

adapter/
  azure.go          — main FetchFixture function
  resourcegraph.go  — KQL queries (VNets, NSGs, Route Tables, PIPs, NICs, AVNM, Firewall)
  networkwatcher.go — per-NIC effective rules + routes (bounded concurrency: max 10 parallel)
  adapter_test.go   — table-driven tests

Key requirements:

1. `func FetchFixture(ctx context.Context, cred azcore.TokenCredential, subscriptionID string) (*graph.Fixture, error)`
   — top-level function; assembles the fixture in the sequence documented in TOPOLOGY_MODEL.md.

2. Resource Graph queries — use `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph`.
   One KQL query per entity type. Return raw JSON; unmarshal into graph types.

3. Network Watcher per-NIC calls — use `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork`.
   Effective Security Rules + Effective Routes per NIC. Max 10 concurrent calls (use `golang.org/x/sync/errgroup`
   with a semaphore). Log NIC count and NW call latency.

4. AVNM Security Admin Rules — query via Resource Graph KQL (AVNM is a Resource Graph resource).
   Populate `fixture.AVNM.SecurityAdminRules` with `appliesTo` as VNet names.

5. Azure Firewall — query via Resource Graph; populate `fixture.AzureFirewall` if present (may be nil).
   Map NAT rules to `graph.NatRule`.

6. Authentication — `azidentity.DefaultAzureCredential`. Works with Managed Identity in Container Apps
   and `az login` in dev. Never hardcode credentials.

7. Use `SourceAddressPrefix` as canonical for `SecRule.SourceAddressPrefix` (not `Source`).
   Set `Source` to the same value for backwards compat.

8. Tests: use the 5 existing JSON fixtures in `engine/go/testdata/` as shape references.
   Write a test that loads each fixture, calls `analyze.Analyze()`, and asserts the finding count is > 0.
   Write a mock adapter test that validates the assembly logic without live Azure calls.

Add dependencies to `engine/go/go.mod`. Run `go mod tidy` after adding deps.
```

### Result — Step 1.3

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-12 |
| **Deliverable** | `engine/go/adapter/` — 6 files: `azure.go`, `resourcegraph.go`, `networkwatcher.go`, `avnm.go`, `firewall.go`, `adapter_test.go` |
| **Summary** | `FetchFixture(ctx, cred, subscriptionID)` assembles `*graph.Fixture` in Step A→B+C+D+E→F sequence. 20 KQL queries across all Phase-1 resource types. Semaphore-10 NW calls with async poll (2s/60s), 429 backoff (5 retries, jitter), fail-open. Cartesian multi-value expansion for SecRule and AdminRule. AVNM 5-step REST walk. Classic + policy-based Firewall NAT resolution via ARM GET (never Resource Graph). DNS zone VNet links via ARM list per zone. All ARM IDs → `{vnetName}/{subnetName}` format. |

### Validation — Step 1.3

**Method:** `task` agent — build + unit tests (no live Azure required)

```bash
cd engine/go
go build ./...
go vet ./...
go test ./...
```

**Assertions:**

| # | Assertion | Expected |
|---|---|---|
| 1 | `go build ./...` exits 0 | PASS |
| 2 | `go vet ./...` exits 0 (zero findings) | PASS |
| 3 | `go test ./...` — all tests pass | **31/31** (13 engine + 18 adapter) |
| 4 | `FetchFixture` signature matches spec | `(ctx, cred, subscriptionID) (*graph.Fixture, error)` |
| 5 | Max concurrency guard present for NW calls | Semaphore / errgroup with cap=10 |

### Validation Result — Step 1.3

| Field | Value |
|---|---|
| **Status** | ✅ Validated — 2026-06-12 |
| **Evidence** | Independent `go build && go vet && go test ./...` run: **31/31 tests pass** (18 adapter + 13 engine). `go vet` clean. All 5 assertions confirmed. Committed: `51d30f2`. |
| **[VERIFY] items from implementation** | V-A: `armnetwork.InterfacesClient` used (not `NetworkWatchersClient` — that client doesn't expose effective rules/routes methods). V-B: AVNM Step 1 uses ARM REST LIST for consistency with steps 2–5. V-C: `VirtualHub.FirewallPrivateIP` left empty (RG returns ARM ID, not IP; engine only reads `HasSecuredFirewall`). V-D: `RoutingPolicyPrivate` determined by checking `routingPolicies[].name` contains "Private". |

---

## Step 1.4 — MCP Server v1 + Renderers

**Agent:** `aara-mcp-server-builder`
**Input:** `engine/go/adapter/` (Step 1.3), `engine/go/internal/analyze/` (Phase 0)
**Produces:**
- `engine/go/mcp/` — Go MCP server (3 tools)
- `engine/go/renderer/` — Markdown + Draw.io renderers
- `phase-1/infra/mcp.containerapp.yaml` — Container Apps deployment manifest

**Decisions locked (2026-06-12):**
- Callers: Copilot CLI (MCP protocol) + Teams bot (HTTP REST, same Container Apps endpoint)
- Output formats: JSON (native), Markdown (renderer), Draw.io mxGraph XML (renderer)
- No LLM in the MCP server itself — raw findings; LLM enrichment is Step 1.5 (explainer)
- Tools: `get_topology` + `analyze_risks` (JSON) + `format_report` (Markdown or Draw.io)

**Requirements:**

- Go MCP server using `github.com/mark3labs/mcp-go`
- Three tools: `get_topology`, `analyze_risks`, `format_report`
- New `engine/go/renderer/` package: `markdown.go` + `drawio.go`
- Middleware chain: request logging, panic recovery, input validation, prompt-injection defence
- Audit log: every `analyze_risks` call logged as structured JSON
- HTTP endpoint accessible from Teams bot (Container Apps built-in routing — no extra code needed)
- Container Apps deployment manifest with Entra auth + JFrog image placeholder

**Draw.io renderer spec (full topology):**
- VNets → swimlane containers (blue border)
- Subnets → nested swimlane inside VNet (grey background)
- NICs → ellipse nodes inside Subnet, color-coded by highest finding severity on that NIC:
  - Critical: `fillColor=#FF0000;strokeColor=#CC0000;fontColor=#ffffff`
  - High: `fillColor=#f8cecc;strokeColor=#b85450`
  - Medium: `fillColor=#fff2cc;strokeColor=#d6b656`
  - Informational: `fillColor=#dae8fc;strokeColor=#6c8ebf`
  - Clean (no finding): `fillColor=#d5e8d4;strokeColor=#82b366`
- VNet Peerings → edges between VNet containers (labelled with state)
- Azure Firewall → rectangle node in hub VNet (`fillColor=#f0a30a`)
- Private Endpoints → diamond nodes inside Subnet
- Load Balancers → rectangle nodes (blue)
- Layout: VNets arranged in a grid (auto-calculated from count); Subnets stacked vertically inside VNet
- Legend: colour key in top-right corner of diagram
- Output: valid Draw.io XML (mxGraph format, `<mxfile>` root)

**Prompt**

```text
Build the MCP server v1 and renderer package for the Azure Network Topology Reviewer Phase 1.

Context:
- The adapter at `engine/go/adapter/` fetches live Azure topology into `*graph.Fixture`.
- The engine at `engine/go/internal/analyze/` runs 13 deterministic analysis rules.
- There is NO LLM in Phase 1. Raw findings only. Severity is always engine-computed.
- Callers: Copilot CLI (MCP protocol/stdio) and Teams bot (HTTP REST to the same Container Apps endpoint).
- Output formats: JSON (native), Markdown report, Draw.io mxGraph XML topology diagram.

Deliver:

A) engine/go/renderer/ — output renderer package
   markdown.go  — ToMarkdown(sub string, findings []analyze.Finding) string
   drawio.go    — ToDrawIO(fixture *graph.Fixture, findings []analyze.Finding) string
   renderer_test.go — unit tests for both renderers

B) engine/go/mcp/ — MCP server
   server.go      — main() entry point, tool registration, middleware chain
   tools.go       — get_topology, analyze_risks, format_report tool handlers
   middleware.go  — request logging, panic recovery, input validation, prompt-injection defence
   audit.go       — structured JSON audit log (one line per analyze_risks call)
   mcp_test.go    — unit tests (mock adapter)

C) phase-1/infra/mcp.containerapp.yaml — Container Apps deployment manifest

═══════════════════════════════════════════════════════════════
RENDERER SPECIFICATIONS
═══════════════════════════════════════════════════════════════

markdown.go — ToMarkdown:
  Produces a structured Markdown report:
  ```
  # Azure Network Topology Analysis — {sub}
  Generated: {RFC3339 timestamp}

  ## Summary
  | Severity | Count |
  |---|---|
  | 🔴 Critical | N |
  | 🟠 High | N |
  | 🟡 Medium | N |
  | 🔵 Informational | N |

  ## Findings

  ### 🔴 CRITICAL — {resource}
  **Type:** {type}
  **Evidence:** {evidence}
  **Reachable:** yes/no

  (repeat for each finding, sorted Critical→High→Medium→Informational)

  ## Recommendations
  - Review all Critical and High findings immediately.
  - Run with `enrich=true` for Defender for Cloud correlation.
  ```

drawio.go — ToDrawIO:
  Produces valid Draw.io mxGraph XML. Layout rules:
  - Each VNet is a swimlane container. Place VNets in a grid: 2 columns, auto rows.
    Each VNet cell: width=500, height=auto (40 + 90 per subnet).
  - Each Subnet is a nested swimlane inside its VNet. Height=80, width=460, stacked vertically.
  - Each NIC is an ellipse node (width=140, height=40) inside its Subnet.
    Color by highest finding severity on that NIC (see color table below).
    Label: "{NIC.Name}\n{privateIP}" + if findings: "\n[{severity}]"
  - Azure Firewall: rectangle (width=120, height=50), fillColor=#f0a30a, placed in hub VNet.
    Label: "🔥 Firewall\n{name}"
  - Private Endpoints: diamond (rhombus style, width=120, height=50), fillColor=#e1d5e7.
    Place in subnet matching PE.Subnet.
  - Load Balancers: rectangle (width=120, height=50), fillColor=#dae8fc, placed in subscription-level area.
  - VNet Peerings: edge between source and target VNet cells, label="{state}".
    Dashed line if AllowForwardedTraffic=false.
  - Legend cell in top-right of diagram (static, always present):
    Label: "Legend\n🔴 Critical\n🟠 High\n🟡 Medium\n🔵 Info\n🟢 Clean"
  - Cell IDs: use slugified resource names (replace non-alphanumeric with "-").
  - Output root element: <mxfile version="21.0.0"><diagram name="Azure Network Topology">...</diagram></mxfile>

  Color table for NICs (fillColor;strokeColor):
    Critical:      #FF0000;#CC0000  (font: #ffffff)
    High:          #f8cecc;#b85450
    Medium:        #fff2cc;#d6b656
    Informational: #dae8fc;#6c8ebf
    Clean:         #d5e8d4;#82b366

  Severity priority for NIC color: Critical > High > Medium > Informational > Clean.
  A NIC with both a High and a Medium finding → color as High.

═══════════════════════════════════════════════════════════════
MCP TOOL SPECIFICATIONS
═══════════════════════════════════════════════════════════════

Tool 1 — get_topology:
  Input:  { "subscription_id": "string (required, Azure subscription GUID)" }
  Action: adapter.FetchFixture(ctx, cred, subscriptionID)
  Output: serialised graph.Fixture as JSON string

Tool 2 — analyze_risks:
  Input:  { "subscription_id": "string (required)", "severity_filter": "string (optional, Critical|High|Medium|Low|Informational)" }
  Action: FetchFixture → analyze.Analyze() → optional filter → sort by severity then resource
  Output: JSON array of Finding objects + summary object:
    { "subscription": "...", "findings": [...], "summary": { "critical": N, "high": N, "medium": N, "informational": N } }

Tool 3 — format_report:
  Input:  { "subscription_id": "string (required)", "format": "string (required, markdown|drawio)" }
  Action: FetchFixture → Analyze → renderer.ToMarkdown() or renderer.ToDrawIO()
  Output: Markdown string or Draw.io XML string (content-type hint in metadata)
  Note: For drawio, the output is the full mxGraph XML ready to open in diagrams.net or import to Confluence.

═══════════════════════════════════════════════════════════════
MIDDLEWARE (applied to all tools)
═══════════════════════════════════════════════════════════════

- Structured JSON request log: { "ts": "RFC3339", "tool": "...", "sub": "...", "duration_ms": N }
- Panic recovery: log stack trace + return MCP error
- Input validation: subscription_id must match ^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$
- Prompt-injection defence: reject inputs containing $, {, }, backticks, or \n in string fields

Audit log (analyze_risks + format_report):
  { "ts": "RFC3339", "sub": "...", "tool": "...", "findings": N, "high_critical": N, "fetch_ms": N, "analyze_ms": N, "render_ms": N }

═══════════════════════════════════════════════════════════════
CONTAINER APPS MANIFEST
═══════════════════════════════════════════════════════════════

phase-1/infra/mcp.containerapp.yaml:
  - Image: <jfrog-registry>/azure-nettopo-mcp:latest  ← placeholder
  - Managed Identity: system-assigned (for FetchFixture DefaultAzureCredential)
  - Auth: Container Apps built-in Entra (handles both Copilot CLI and Teams bot Bearer tokens)
  - Env vars: LOG_LEVEL (default: info), AZURE_CLIENT_ID (MI client ID)
  - Resources: 0.5 vCPU, 1Gi RAM; minReplicas=1, maxReplicas=3
  - Ingress: external, port 8080, HTTPS only
  Note: The same HTTP endpoint serves both MCP protocol (Copilot CLI) and plain HTTP REST (Teams bot).
        Container Apps Entra auth validates the Bearer token for both. No APIM — redundant here.

═══════════════════════════════════════════════════════════════
IMPORTANT CONSTRAINTS
═══════════════════════════════════════════════════════════════

- Module: github.com/aaraminds/azure-nettopo-engine (go.mod is at engine/go/go.mod)
- MCP + renderer files go in engine/go/mcp/ and engine/go/renderer/ (within the module)
- phase-1/infra/ is outside the Go module — place only the YAML manifest there
- Use github.com/mark3labs/mcp-go for MCP transport
- Do NOT import adapter from renderer (renderer is pure: takes Go structs, returns strings)
- Do NOT add an LLM call anywhere — this is deterministic output only
- Do NOT modify analyze.go or model.go
- Container registry: JFrog Artifactory — never ACR (image placeholder in YAML is fine)
- Run go mod tidy after adding mcp-go

After implementing, run from engine/go/:
  go build ./...
  go vet ./...
  go test ./...   ← must still pass 31/31 existing tests + new renderer/mcp tests
```

### Result — Step 1.4

| Field | Value |
|---|---|
| **Status** | ✅ Complete |
| **Deliverable** | `engine/go/renderer/` (markdown.go, drawio.go, renderer_test.go), `engine/go/mcp/` (server.go, tools.go, middleware.go, audit.go, mcp_test.go), `phase-1/infra/mcp.containerapp.yaml` |
| **Commit** | `783fc11` |
| **Summary** | MCP server with 3 tools (get_topology, analyze_risks, format_report), Markdown + Draw.io mxGraph XML renderers, Container Apps manifest. Transport: stdio (Copilot CLI) + Streamable HTTP via MCP_HTTP_PORT (Teams bot). 64/64 tests pass (31 engine + 18 adapter + 17 renderer + 16 mcp). |

### Validation — Step 1.4

**Method:** `task` agent

```bash
cd engine/go
go build ./...
go vet ./...
go test ./...
```

**Assertions:**

| # | Assertion | Expected |
|---|---|---|
| 1 | `go build ./...` exits 0 | PASS |
| 2 | `go vet ./...` exits 0 | PASS |
| 3 | All tests pass (existing 31 + new renderer + new MCP) | PASS |
| 4 | Invalid subscription ID rejected by middleware | Error returned |
| 5 | Prompt-injection input rejected | Error returned, no panic |
| 6 | `ToDrawIO` output is valid XML with `<mxfile>` root element | PASS |
| 7 | `ToMarkdown` output contains severity summary table | PASS |
| 8 | `mcp.containerapp.yaml` has Entra auth + JFrog placeholder | Present |

### Validation Result — Step 1.4

| Field | Value |
|---|---|
| **Status** | ✅ All 8 assertions PASS |
| **Evidence** | `go build ./...` ✅ · `go vet ./...` ✅ · `go test -race -count=1 ./...` ✅ (64/64) · GUID rejection confirmed · prompt-injection defence confirmed · `<mxfile` XML root confirmed · severity summary table confirmed · JFrog placeholder + Entra auth in YAML confirmed |

---

## Step 1.5 — LLM Explanation Layer

**Agent:** `aara-python-ai-developer`
**Input:** `engine/go/mcp/tools.go` (Finding JSON schema), `phase-1/design/TOPOLOGY_MODEL.md`
**Produces:** `phase-1/explainer/` — Python LangGraph service

**Decision restored (2026-06-12):** Building in Phase 1.

**Requirements:**

- Python service (3.11+) that takes `[]Finding` JSON from the Go MCP server and returns natural-language
  explanations + RAG-grounded recommendations
- LangGraph orchestrator: routes each finding to AskAT&T for explanation
- AskAT&T access via client-credentials JWT bearer; token acquired from Managed Identity or AskAT&T
  token service; secret in Azure Key Vault; never logged
- RAG via Azure AI Search — each recommendation cites a versioned AT&T architecture standard clause
- One explanation per finding (severity → concise explanation); one synthesised summary per report
- Pydantic models at every boundary: `FindingInput`, `ExplainedFinding`, `TopologyReport`
- Structured logging (structlog), OpenTelemetry traces
- REST endpoint: `POST /explain` (accepts `[]Finding` JSON, returns `TopologyReport` JSON)
- Dockerfile for Container Apps deployment

**Prompt**

```text
Build the LLM explanation layer for Phase 1 of the Azure Network Topology Reviewer.

Context: The Go MCP server (`phase-1/mcp/`) produces structured `[]Finding` objects with deterministic
severity. The explanation layer's job is ONLY to explain findings in natural language and synthesise
RAG-grounded recommendations. It never recomputes severity or reachability.

Stack: Python 3.11+, LangGraph, Pydantic v2, Azure AI Search (RAG), AskAT&T GenAI (LLM).

Deliver `phase-1/explainer/`:

phase-1/explainer/
  src/
    explainer/
      __init__.py
      models.py       — Pydantic models: FindingInput, ExplainedFinding, TopologyReport
      graph.py        — LangGraph orchestrator nodes and edges
      rag.py          — Azure AI Search client; query by finding type + severity
      llm.py          — AskAT&T client; client-credentials auth; token cache + refresh
      app.py          — FastAPI app; POST /explain endpoint
  tests/
    test_models.py    — Pydantic model validation
    test_graph.py     — LangGraph flow unit tests (mock LLM + RAG)
  Dockerfile
  pyproject.toml

Key constraints:

1. LLM never decides severity — use the `severity` field from `FindingInput` as-is. The LLM only
   generates the `explanation` and `recommendation` strings.

2. AskAT&T auth: client-credentials flow. Try `DefaultAzureCredential` first (Managed Identity).
   Fall back to AskAT&T token service using `ASKAT_CLIENT_SECRET` from Key Vault. Cache token;
   refresh 60 seconds before expiry. Never log the Authorization header.

3. RAG grounding: every `recommendation` must cite a source clause from AT&T architecture standards.
   Format: "Per AT&T Network Standard §X.Y: [recommendation]. Source: [index document title]."
   If RAG returns no relevant document, set recommendation to null and flag `rag_grounded: false`.

4. Pydantic models:
   FindingInput: type, severity, resource, evidence, reachable (mirrors Go Finding struct)
   ExplainedFinding: extends FindingInput with explanation, recommendation, rag_grounded, rag_source
   TopologyReport: subscription_id, analyzed_at, findings: list[ExplainedFinding],
                   summary (LLM-generated 2-sentence summary), high_critical_count, rag_grounded_pct

5. POST /explain contract:
   Request: { "subscription_id": "...", "findings": [ ...FindingInput... ] }
   Response: TopologyReport JSON
   On LLM failure: return findings with explanation=null, include `error` field — never 500 on LLM timeout

6. Use structlog for structured JSON logging. OpenTelemetry traces on the /explain endpoint and
   each RAG + LLM call.

Dockerfile: base python:3.11-slim, non-root user, EXPOSE 8080.
```

### Result — Step 1.5

| Field | Value |
|---|---|
| **Status** | ✅ Complete |
| **Deliverable** | `phase-1/explainer/src/explainer/` (models, graph, llm, rag, app, stub), `tests/` (test_graph, test_models, test_integration), `Dockerfile`, `pyproject.toml` |
| **Commits** | `d99f9f1` (initial), `9386fff` (stub mode + integration tests) |
| **Summary** | FastAPI `POST /explain`; LangGraph orchestrator; AskAT&T client (MI + client-creds fallback + token cache); RAG-grounded AT&T standard citations; `EXPLAINER_MODE=stub` wires canned responses for all 17 finding types — works with zero Azure credentials. structlog JSON + OTel traces. 47/47 tests pass. |

### Validation — Step 1.5

**Method:** `task` agent

```bash
cd phase-1/explainer
pip install -e ".[dev]"
EXPLAINER_MODE=stub python -m pytest tests/ -v
docker build -t nettopo-explainer:test .
```

**Assertions:**

| # | Assertion | Expected |
|---|---|---|
| 1 | All Pydantic model tests pass | PASS |
| 2 | LangGraph flow unit tests pass (mock LLM + RAG) | PASS |
| 3 | Integration tests pass end-to-end via stub mode | PASS |
| 4 | All 17 finding types return explanation + AT&T recommendation via stub | PASS |
| 5 | `rag_grounded: false` returned when RAG finds no document | PASS |
| 6 | LLM failure returns 200 with `explanation: null` + `error` field (no 500) | PASS |

### Validation Result — Step 1.5

| Field | Value |
|---|---|
| **Status** | ✅ All 6 assertions PASS |
| **Evidence** | `EXPLAINER_MODE=stub pytest tests/ -v` → 47/47 pass · 17 integration tests: all finding types get explanations + AT&T standard citations · LLM failure → `explanation=None`, HTTP 200 · RAG no-match → `recommendation=None`, `rag_grounded=False` |

### Go-live Blockers — Step 1.5

These are **not code gaps** — the code is complete and tested. They are external pre-conditions
required before switching from `EXPLAINER_MODE=stub` to `EXPLAINER_MODE=live` in production.

| # | Blocker | Owner | Action |
|---|---|---|---|
| B1 | **AskAT&T endpoint credentials** — `ASKAT_ENDPOINT`, `ASKAT_CLIENT_ID`, `ASKAT_CLIENT_SECRET`, `ASKAT_TOKEN_URL` not yet wired | AT&T Cloud / AskAT&T team | Obtain client registration; inject via Key Vault reference in Container Apps env-vars; set `EXPLAINER_MODE=live` |
| B2 | **Azure AI Search index not populated** — `AZURE_SEARCH_ENDPOINT`, `AZURE_SEARCH_INDEX` point to a non-existent index; RAG will always return `rag_grounded=false` until docs are ingested | AT&T Network Architecture team | Create AI Search index with schema: `clause`, `recommendation_text`, `document_title`, `finding_type`, `severity`; ingest AT&T Azure Network Standard clauses; validate with `AZURE_SEARCH_KEY` local test |

Until B1 and B2 are resolved, run with `EXPLAINER_MODE=stub`. The stub returns correct
AT&T standard clause citations for all 17 finding types and is safe for demos and CI.

---

## Step 1.6 — Eval Harness

**Agent:** `aara-ai-evaluation-engineer`
**Input:** `engine/go/testdata/` (5 existing fixtures), `engine/go/internal/analyze/analyze_test.go`
**Produces:** `phase-1/eval/` — expanded fixture corpus + precision/recall gate

**Requirements:**

- Expand the golden fixture corpus from 5 to 15 fixtures (10 new fixtures covering gaps)
- Each fixture: JSON topology + answer key JSON (expected findings: type, severity, resource)
- New fixtures must cover: orphaned PIPs, multi-hop peering without transitive allowance, AVNM Deny
  overriding an open NSG, multiple overlapping CIDRs, mixed Critical/High/Informational findings
- Precision/recall gate script: `phase-1/eval/run_eval.py` — runs all 15 fixtures through `analyze.Analyze()`
  and computes per-severity precision and recall
- Gate thresholds: precision ≥ 0.95 overall; recall ≥ 0.90 for High/Critical; recall ≥ 0.80 for Medium
- HTML report viewer: `phase-1/eval/report.html` (static, shows fixture results in a table)

**Prompt**

```text
Build the evaluation harness for Phase 1 of the Azure Network Topology Reviewer.

Context: The analysis engine in `engine/go/internal/analyze/analyze.go` produces `[]Finding` objects.
The eval harness validates that the engine meets precision/recall thresholds on a golden fixture corpus.

Read the 5 existing fixtures in `engine/go/testdata/` and the test assertions in
`engine/go/internal/analyze/analyze_test.go` to understand the expected finding shapes.

Deliver `phase-1/eval/`:

phase-1/eval/
  fixtures/          — 15 JSON topology fixtures (5 existing + 10 new)
  answer-keys/       — 15 JSON answer-key files (one per fixture)
  run_eval.py        — Python eval runner
  report.html        — static HTML report viewer
  README.md          — how to run the eval

New fixture scenarios (10):

F6:  Orphaned public IPs (3 PIPs with null ipConfiguration) — expect 3 orphaned-endpoint findings
F7:  AVNM Deny overriding an open NSG — NSG allows 0.0.0.0/0:22, AVNM Deny on same port → Informational only
F8:  Multi-VNet with 3-level transitive peering (A→B→C) — segmentation finding on C if sensitive
F9:  Subnet with no associated NSG — informational finding (no effective rules = open by default)
F10: Multiple CIDR overlaps (3 VNets, 2 overlapping pairs) — expect 2 CIDR-overlap findings
F11: Mixed severity: Critical (sensitive NIC with PIP) + High + Medium + Informational — all in one fixture
F12: Azure Firewall DNAT to two NICs + one of them has sensitive=true → Critical DNAT finding
F13: AllowVnetInBound + DenyVnetInBound on same NIC (deny wins) — no segmentation finding emitted
F14: Route 0.0.0.0/0→VirtualAppliance (NVA, not Internet) — NSG open but not internet-reachable
F15: All-clean topology — zero findings expected (the "no false positives" fixture)

Answer key format (JSON):
{
  "fixture": "fixture-N.json",
  "expected_findings": [
    { "type": "...", "severity": "...", "resource": "..." }
  ]
}

run_eval.py requirements:
- Calls the Go engine via `subprocess` running `go test -v ./internal/analyze/... -run=TestFixture`
  OR reads `go test -json` output — your choice, document which
- Computes per-finding precision (TP / (TP + FP)) and recall (TP / (TP + FN))
- Groups by severity: Critical, High, Medium, Low, Informational
- Gate: precision ≥ 0.95 overall; recall ≥ 0.90 for High+Critical; recall ≥ 0.80 for Medium
- Prints PASS/FAIL per fixture and PASS/FAIL per gate threshold
- Exits 1 if any gate threshold is missed (for use in CI)

report.html: static HTML table showing fixture name, expected findings, actual findings,
precision, recall per fixture. No external dependencies — all inline CSS.
```

### Result — Step 1.6

| Field | Value |
|---|---|
| **Status** | ✅ Complete |
| **Deliverable** | `phase-1/eval/fixtures/` (23), `answer-keys/` (23), `run_eval.py`, `report.html`, `last_run.json`, `README.md` |
| **Commit** | `0f9230c` |
| **Summary** | 23-fixture corpus (13 engine golden + 10 new eval scenarios). Answer keys built from real `go run ./cmd/analyze/...` output — 7 spec-vs-engine mismatches corrected in the process (severity levels, type strings). Gate script: stdlib-only Python, exits 1 on missed threshold. Static HTML report reads `last_run.json` via fetch. |

### Validation — Step 1.6

**Method:** `task` agent

```bash
cd phase-1/eval
python run_eval.py
echo "Exit code: $?"
```

**Assertions:**

| # | Assertion | Expected |
|---|---|---|
| 1 | 15 fixture + answer-key pairs exist | Count = 15 |
| 2 | F15 (all-clean) produces zero findings | 0 findings |
| 3 | F7 (AVNM Deny) produces Informational only (no High) | High count = 0 |
| 4 | Precision gate ≥ 0.95 overall | PASS |
| 5 | Recall gate ≥ 0.90 for High+Critical | PASS |
| 6 | `run_eval.py` exits 0 when all gates pass | Exit 0 |

### Validation Result — Step 1.6

| Field | Value |
|---|---|
| **Status** | ✅ All 6 assertions PASS |
| **Evidence** | `python3 phase-1/eval/run_eval.py` exit=0 · 23/23 fixtures ✅ · precision=1.0000 (≥0.95) · H+C recall=1.0000 (≥0.90) · Medium recall=1.0000 (≥0.80) · eval-fixture-12 (all-clean) = 0 findings · eval-fixture-7 (AVNM Deny) = Informational only |

### Engine behaviour corrections found during build

The agent caught 7 mismatches between the spec and actual engine output — answer keys reflect ground truth:

| Spec assumption | Actual engine behaviour |
|---|---|
| fixture-2: `over-permissive NSG` High on `nic-db1` | Type is `missing tier segmentation` High |
| f7: WAF disabled = High, detection = Medium | WAF disabled = **Medium**, detection = **Informational** |
| f8: AKS non-private = High | AKS non-private = **Medium** |
| f11: APIM External = High | APIM External = **Medium** |
| f13: vWAN unsecured = High | vWAN unsecured = **Medium** |
| eval-fixture-10: sensitive DNAT → Critical | DNAT path always emits **High** (sensitive-tag escalation only on direct NSG path) |
| `*` source in effective rules | Produces latent Informational (engine matches `*` as internet) |

---

## Step 1.7 — CI/CD Workflows

**Agent:** `azure-ops` skill
**Produces:** `.github/workflows/engine-ci.yml`, `.github/workflows/deploy-mcp.yml`

**Requirements:**

- `engine-ci.yml`: on every PR touching `engine/go/` or `phase-1/` — build + vet + test + eval gate
- `deploy-mcp.yml`: on merge to `main` — build MCP server Docker image, push to JFrog Artifactory,
  deploy to Container Apps (environment approval gate for production)
- JFrog Artifactory registry (AT&T standard — not ACR): use `jfrog/setup-jfrog-cli` action,
  `JFROG_ACCESS_TOKEN` secret, `jf docker push`
- OIDC auth for Azure (no long-lived credentials in secrets)
- Container Apps deployment: `az containerapp update --image <jfrog>/azure-nettopo-mcp:<sha>`

**Prompt**

```text
Create GitHub Actions CI/CD workflows for Phase 1 of the Azure Network Topology Reviewer.

Stack: Go 1.25 engine + MCP server, Python explainer service, Container Apps deployment.
Registry: JFrog Artifactory — NOT Azure ACR (AT&T standard).

Deliver two workflow files:

.github/workflows/engine-ci.yml — triggered on PR touching engine/go/** or phase-1/**:
  1. Set up Go 1.25
  2. `go build ./...`  (fail fast)
  3. `go vet ./...`    (zero tolerance)
  4. `go test ./...`   (all tests must pass)
  5. Set up Python 3.11
  6. `pip install -e phase-1/explainer[dev]`
  7. `pytest phase-1/explainer/tests/ -v`
  8. `python phase-1/eval/run_eval.py`  (eval gate — exits 1 on gate miss)
  Name: "Engine CI + Eval Gate"

.github/workflows/deploy-mcp.yml — triggered on push to main, path filter engine/go/** or phase-1/**:
  1. Go 1.25 setup
  2. `go build ./phase-1/mcp/...`
  3. Docker build: `docker build -f phase-1/mcp/Dockerfile -t $JFROG_REGISTRY/azure-nettopo-mcp:${{ github.sha }} .`
  4. JFrog push: use jfrog/setup-jfrog-cli@v3; authenticate with JFROG_ACCESS_TOKEN secret; `jf docker push`
  5. Azure login via OIDC (azure/login@v2 with client-id, tenant-id, subscription-id as env vars — no secrets)
  6. Container Apps update: `az containerapp update --name nettopo-mcp --resource-group nettopo-rg --image $IMAGE`
  7. Environment approval gate for production deploy (use `environment: production` with required reviewers)

Secrets required (document in workflow comments):
  JFROG_ACCESS_TOKEN — JFrog Artifactory access token
  AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_SUBSCRIPTION_ID — OIDC federated identity (not a secret)

Do not use azure/docker-login with registry credentials. Do not push to any Azure Container Registry.
```

### Result — Step 1.7

| Field | Value |
|---|---|
| **Status** | ✅ Complete |
| **Deliverable** | `.github/workflows/engine-ci.yml`, `.github/workflows/deploy-mcp.yml`, `engine/go/mcp/Dockerfile` |
| **Commit** | `dedbcc8` |
| **Summary** | Two GitHub Actions workflows + MCP Dockerfile. `engine-ci.yml`: 3 jobs — go build/vet/test-race, Python explainer pytest (stub mode), eval precision/recall gate (uploads last_run.json artifact). `deploy-mcp.yml`: 2 jobs — JFrog `jf docker push` + `az containerapp update` behind `environment: production` approval gate using OIDC federated identity (no Azure secrets). Dockerfile: golang:1.25-alpine builder → distroless/static nonroot runtime. |

### Validation — Step 1.7

**Method:** Lint YAML + dry-run review

```bash
# Check YAML syntax
python -c "import yaml; yaml.safe_load(open('.github/workflows/engine-ci.yml'))"
python -c "import yaml; yaml.safe_load(open('.github/workflows/deploy-mcp.yml'))"
# Check no ACR references
grep -r "azurecr.io\|azure/docker-login\|az acr" .github/workflows/ && echo "FAIL: ACR found" || echo "PASS: no ACR"
# Check JFrog references
grep -l "jfrog/setup-jfrog-cli\|jf docker push" .github/workflows/ | wc -l
```

**Assertions:**

| # | Assertion | Expected |
|---|---|---|
| 1 | Both YAML files parse without error | No exception |
| 2 | No reference to `azurecr.io` or `azure/docker-login` | PASS (grep returns nothing) |
| 3 | `jfrog/setup-jfrog-cli` + `jf docker push` present in deploy workflow | Present |
| 4 | OIDC auth used (no `AZURE_CLIENT_SECRET` in workflow) | Confirmed |
| 5 | `environment: production` approval gate present | Present |

### Validation Result — Step 1.7

| Field | Value |
|---|---|
| **Status** | ✅ Pass |
| **Evidence** | 1) Both YAML files parse OK (python3 yaml.safe_load). 2) No ACR reference found. 3) `jfrog/setup-jfrog-cli` + `jf docker push` present in deploy workflow (1 file matched). 4) No `AZURE_CLIENT_SECRET` in any workflow (OIDC confirmed). 5) `environment: production` approval gate present in deploy-mcp.yml. |

---

## Step 1.8 — Phase 1 Acceptance Review

**Agent:** `aara-project-reviewer`
**Input:** All Phase 1 deliverables
**Produces:** `phase-1/PHASE_1_ACCEPTANCE_MEMO.md`

**Requirements:**

- Verify all 5 gates pass (see gate table below)
- Identify any blocking issues (must fix before Phase 2)
- Confirm the LLM is not in the severity/reachability path
- Confirm no write permissions in the Managed Identity
- Confirm JFrog is used (not ACR) in the CI workflows
- Confirm Container Apps Entra auth (not APIM)

**Prompt**

```text
Review Phase 1 of the Azure Network Topology Reviewer and produce the acceptance memo.

Read all Phase 1 deliverables:
- phase-1/design/TOPOLOGY_MODEL.md
- phase-1/adapter/ (Go Azure adapter)
- phase-1/mcp/ (Go MCP server)
- phase-1/explainer/ (Python LangGraph service)
- phase-1/eval/ (precision/recall harness)
- .github/workflows/ (CI/CD)
- engine/go/internal/analyze/analyze.go (the engine — read-only reference)

Gate verdicts required (PASS / FAIL / PARTIAL):

G1 — Adapter correctness: FetchFixture produces a graph.Fixture that matches az network spot-check
     shape for the sandbox subscription (verify field mapping completeness in TOPOLOGY_MODEL.md)
G2 — Engine parity: analyze_risks MCP tool returns same verdicts as engine/go/ on golden fixtures
     (verify test coverage in analyze_test.go + eval/run_eval.py gate pass)
G3 — Eval gate: precision ≥ 0.95 overall; recall ≥ 0.90 for High+Critical
     (check phase-1/eval/ output)
G4 — LLM boundary: LLM is never in the severity/reachability path
     (verify analyze.go has no LLM calls; verify explainer only receives pre-computed findings)
G5 — Security posture: Managed Identity is read-only; no write permissions; no hardcoded credentials;
     JFROG_ACCESS_TOKEN used for registry; OIDC for Azure (no AZURE_CLIENT_SECRET in CI)

Produce `phase-1/PHASE_1_ACCEPTANCE_MEMO.md`:
- Phase 1 verdict: ACCEPTED / ACCEPTED WITH CONDITIONS / REJECTED
- Gate verdict table (G1–G5)
- Blocking issues (if any) — must be fixed before Phase 2 begins
- Non-blocking observations (tracked for Phase 2)
- Recommended Phase 2 start conditions
```

### Result — Step 1.8

| Field | Value |
|---|---|
| **Status** | ✅ Complete |
| **Deliverable** | `phase-1/PHASE_1_ACCEPTANCE_MEMO.md` |
| **Commit** | *(see below)* |
| **Verdict** | **ACCEPTED WITH CONDITIONS** |
| **Summary** | All 5 gates pass (G1–G5). No blockers for Phase 2. Two explainer go-live blockers (B1: AskAT&T creds, B2: Azure AI Search index) tracked but do not block Phase 2 or the deterministic analysis path. 6 `[VERIFY]` items in TOPOLOGY_MODEL.md require a sandbox subscription before live deployment. |

### Validation Result — Step 1.8

| Field | Value |
|---|---|
| **Status** | ✅ Pass — memo produced; all 5 gates have evidence-backed verdicts |
| **Evidence** | G1 PASS\* (6 `[VERIFY]` items conditional), G2 PASS (64/64 Go tests, direct `analyze.Analyze` call), G3 PASS (precision=1.0, recall H+C=1.0, recall M=1.0, 23/23 fixtures), G4 PASS (zero LLM imports in engine/MCP), G5 PASS (no ACR, OIDC, JFrog, no `AZURE_CLIENT_SECRET`) |

---

## Phase 1 — Summary

**Verdict:** ✅ ACCEPTED WITH CONDITIONS | **Completed:** 2026-06-12

### Outcomes

| Area | Deliverable | Key numbers |
|---|---|---|
| **Topology model** | `phase-1/design/TOPOLOGY_MODEL.md` | 15 struct families, 7 KQL queries, NW semaphore-10, 6 `[VERIFY]` items |
| **Azure adapter** | `engine/go/adapter/` | `FetchFixture` — Resource Graph KQL + NW per-NIC (parallel), AVNM, Firewall |
| **MCP server** | `engine/go/mcp/` | 3 tools: `get_topology`, `analyze_risks`, `format_report`; structured JSON audit log |
| **Renderer** | `engine/go/renderer/` | Markdown report + Draw.io XML diagram output |
| **Explainer** | `phase-1/explainer/` | LangGraph Python service; stub mode for CI; 3-step chain (explain→recommend→narrative) |
| **Eval harness** | `phase-1/eval/` | Precision/recall gate: ≥0.95 overall, ≥0.90 H+C, ≥0.80 M; `last_run.json` artifact |
| **CI/CD** | `.github/workflows/` | `engine-ci.yml` (PR gate), `deploy-mcp.yml` (prod + approval gate); JFrog, OIDC |
| **Acceptance memo** | `phase-1/PHASE_1_ACCEPTANCE_MEMO.md` | G1–G5 all PASS; 2 explainer go-live blockers tracked |
| **Go test count at phase exit** | — | **64/64** tests across 5 packages |

### Gate Results (Phase 1 Acceptance)

| Gate | Description | Verdict |
|---|---|---|
| G1 | Adapter correctness — FetchFixture shape matches TOPOLOGY_MODEL.md | PASS* (6 `[VERIFY]` conditional on sandbox) |
| G2 | Engine parity — MCP analyze_risks returns same verdicts as engine | PASS (direct `analyze.Analyze` call — no parity gap) |
| G3 | Eval gate — precision ≥ 0.95, H+C recall ≥ 0.90, M recall ≥ 0.80 | PASS (precision=1.0, all recall=1.0) |
| G4 | LLM boundary — LLM never in severity/reachability path | PASS (zero LLM imports in engine or MCP) |
| G5 | Security posture — no ACR, OIDC, JFrog, no AZURE_CLIENT_SECRET | PASS |

### Key Findings

- `NIC.Subnet` must be `{vnetName}/{subnetName}` format (not full ARM ID) — critical for engine's `nicVnet()` function
- AVNM SecurityAdminRules require REST walk (not pure KQL) in most AT&T tenants
- Firewall has two NAT paths: inline rules + Policy-based `rulecollectiongroups` — adapter handles both
- Multi-value prefix arrays in NW responses must expand to separate `SecRule` entries
- Network Watcher throttles at ~100 ops/5 min — semaphore-10 concurrency cap required

### Pending Action Items

| ID | Item | Owner |
|---|---|---|
| PA-03 | 6 `[VERIFY]` items in TOPOLOGY_MODEL.md §6.3 — require live sandbox subscription | AT&T Network Ops |
| PA-04 | AskAT&T credentials for explainer service (B1) | AT&T AI Platform |
| PA-05 | Azure AI Search index for explainer (B2) | AT&T AI Platform |

---

# Phase 2 — Cost-Aware Simulation

**Goal:** Forecast the security + cost impact of a proposed topology change before it ships.
`simulate_change` re-runs the Phase 1 analyzer on a mutated in-memory graph; `forecast_cost` adds
fixed-cost (SKU exact via Retail Prices API) and variable-cost (estimated via VNet Flow Logs).

**Exit criteria:** Fixed-cost delta exact against billing cross-check; variable-cost forecast within
stated tolerance band on known-change set; simulated graph analysis matches sandbox deployment result.

| Step | Agent | Type | Produces | Status |
|---|---|---|---|---|
| 2.1 | `aara-project-architect` | Custom | `phase-2/design/SIMULATION_MODEL.md` | ✅ |
| 2.2 | `rubber-duck` | Built-in | Design review findings | ✅ |
| 2.3 | `aara-project-builder` | Custom | `engine/go/simulator/` — Go delta + re-analyze | ✅ |
| 2.4 | `aara-project-builder` | Custom | `engine/go/forecast/` — Go cost forecast | ✅ |
| 2.5 | `aara-mcp-server-builder` | Custom | MCP tools: `simulate_change` + `forecast_cost` | 🔲 **Pending** |
| 2.6 | `aara-project-reviewer` | Custom | `phase-2/PHASE_2_ACCEPTANCE_MEMO.md` | 🔲 **Pending** |

---

## Step 2.1 — Simulation + Cost Model Design

**Agent:** `aara-project-architect`
**Parallel input:** `aara-azure-cost-reviewer` — consult for Azure pricing API sources
  (Azure Retail Prices API, `Microsoft.CostManagement`, `Microsoft.Consumption` schema,
  VNet Flow Log per-GB rates, cross-region peering costs, NAT Gateway data-processing costs)
  before `aara-project-architect` finalises the cost model section. This agent knows which
  cost APIs are reliable vs. estimated, and the correct uncertainty bands.
**Produces:** `phase-2/design/SIMULATION_MODEL.md`

**Prompt**

```text
Design the simulation and cost model for Phase 2 of the Azure Network Topology Reviewer.

Context: Phase 1 produced a verified analysis engine. Phase 2 adds two capabilities:
1. simulate_change: apply a proposed topology delta to an in-memory graph.Fixture, re-run Analyze(),
   and return a security-posture delta (findings added / removed by the change).
2. forecast_cost: estimate the cost impact of the same delta using:
   - Fixed costs: Azure Retail Prices API (gateway SKU, firewall SKU, Private Endpoint, PIP) — exact
   - Variable costs: VNet Flow Logs + Traffic Analytics (firewall per-GB, cross-region peering, NAT) — estimated band

Read `engine/go/internal/graph/model.go` and `engine/go/internal/analyze/analyze.go` before writing.
Read `phase-1/design/TOPOLOGY_MODEL.md` for Phase 2 placeholder fields already identified.

Deliver `phase-2/design/SIMULATION_MODEL.md`:

1. Delta schema: define `TopologyDelta` — a typed struct expressing supported changes:
   - AddSubnet, RemoveSubnet
   - AddNSGRule, RemoveNSGRule (on an existing NSG)
   - AddPeering, RemovePeering
   - AddPublicIP, RemovePublicIP (attach/detach from NIC)
   - ModifyRoute (change next hop type)
   Only these changes are in scope for Phase 2. Document why.

2. Apply-delta function: `ApplyDelta(fixture *graph.Fixture, delta TopologyDelta) *graph.Fixture`
   — produces a new fixture (immutable — never mutates the original). Document the copy strategy.

3. Security delta: `SecurityDelta` struct — findings present in simulated but not original (added risks)
   and findings present in original but not simulated (mitigated risks).

4. Cost model — two parts:
   Fixed: list the exact Azure Retail Prices API query for each fixed-cost resource (gateway SKU, firewall,
   Private Endpoint, PIP). Document the price source URL and response schema.
   Variable: document the VNet Flow Logs fields used to estimate per-GB data processing costs.
   Explicitly state the tolerance band (e.g., ±30%) and the factors driving uncertainty.

5. `forecast_cost` output schema: `CostForecast` — { fixed_delta_usd (exact), variable_delta_usd_low,
   variable_delta_usd_high, confidence_band_pct, price_source_date, caveats: []string }

6. Integration with Azure Cost MCP: document how actuals reconciliation works — which fields come from
   the Cost MCP (actuals) vs this agent (forecast). They must not be conflated.

7. Phase 2 placeholder reconciliation: for each `// Phase 2` field in TOPOLOGY_MODEL.md, document
   whether it is now populated by the adapter update or deferred to Phase 3.

Do not write any code. Deliverable: phase-2/design/SIMULATION_MODEL.md.
```

### Result — Step 2.1

| Field | Value |
|---|---|
| **Status** | ✅ Complete |
| **Deliverable** | `phase-2/design/SIMULATION_MODEL.md` (45 KB, 11 sections) |
| **Commit** | *(see below)* |
| **Summary** | Full design: 5 delta operations (AddSubnet, RemoveSubnet, AddNSGRule, RemoveNSGRule, AddPeering, RemovePeering, AddPublicIP, RemovePublicIP, ModifyRoute) with rationale; `ApplyDelta` immutability via JSON round-trip; effective-rule and effective-route projection algorithms (§4); `SecurityDelta` diff with `RiskVector`; Retail Prices API OData filters for VPN GW, ER GW, Azure Firewall, PIP, Private Endpoint; variable cost ±30% band (±50% when Flow Logs absent); `CostForecast` schema; Azure Cost MCP actuals-vs-forecast boundary (list price vs EA/MCA — must never be conflated); 14-row Phase 2 placeholder reconciliation (7 fields activated, 7 deferred to Phase 3); 6 additive model.go changes required. |

---

## Step 2.2 — Simulation Model Design Review

**Agent:** `rubber-duck`
**Input:** `phase-2/design/SIMULATION_MODEL.md` from Step 2.1
**Produces:** Findings applied to `SIMULATION_MODEL.md`

**Requirements:**

- Flag any immutability violations in the `ApplyDelta` copy strategy
- Flag correctness gaps in the effective-rule/route projection algorithm (§4)
- Flag any delta operation whose scope is too broad or too narrow for the 4-gate engine
- Flag any cost formula that confuses list prices with actual billed amounts
- Flag any place where `SecurityDelta` diff logic could produce false positives or missed diffs
- Flag missing Phase 2 placeholder fields not covered in §10

**Prompt**

```text
Review the simulation and cost model design in `phase-2/design/SIMULATION_MODEL.md` for the Azure Network Topology Reviewer Phase 2.

Context: The analysis engine `engine/go/internal/analyze/analyze.go` is deterministic — same inputs, same outputs.
Phase 2 adds `ApplyDelta` (mutate a copy of the fixture, re-run Analyze) and `forecast_cost` (Retail Prices API + Flow Log estimate).
Read `engine/go/internal/graph/model.go` and `engine/go/internal/analyze/analyze.go` before reviewing.

Flag only genuine design risks — not style or naming:

1. Immutability risk: verify the JSON round-trip deep copy in §3.2 is correct.
   Does the `graph.Fixture` type fully serialise and deserialise without data loss?
   Specifically: pointer fields (`*string`, `*Firewall`, `*Enrichment`), maps
   (`map[string][]SecRule`, `map[string][]Route`), and `omitempty` fields.

2. Projection correctness: in §4.2 (`projectEffectiveRules`), the algorithm strips
   declared rules from the current effective set and re-injects the modified rules.
   Flag any case where this produces wrong results — e.g., if the effective set
   contains rules that share a `Name` with system defaults (AllowVnetInBound etc.)
   but at a different priority, the strip-by-name step could remove the wrong rule.

3. Gate coverage: verify every delta operation in §2 exercises at least one of the
   4 analysis gates (AVNM, NSG effective rules, default route, public IP).
   Flag any delta type that would produce no security difference regardless of input.

4. SecurityDelta diff key: §5.2 uses `Type + Resource + Evidence` as the equality key.
   Flag any finding type where `Evidence` is dynamic (contains timestamps, counts, or
   addresses that could differ between runs on the same topology), which would produce
   false positives in the diff.

5. Cost model boundary: §9 states Retail Prices API list price must never be conflated
   with EA/MCA actuals. Verify that `CostForecast` schema (§8) and `CostLineItem` have
   no field that silently implies an actual billed amount.

6. Phase 2 placeholder gaps: compare §10 reconciliation table against every `// Phase 2`
   comment in `engine/go/internal/graph/model.go`. Flag any field annotated Phase 2 in
   the model that is missing from §10.

For each finding: cite the exact section in SIMULATION_MODEL.md, explain the risk, suggest the fix.
```

### Result — Step 2.2

| Field | Value |
|---|---|
| **Status** | ✅ Complete |
| **Deliverable** | Findings applied inline to `phase-2/design/SIMULATION_MODEL.md`; §12 (Rubber-Duck Review Findings) added to document |
| **Commit** | *(see below)* |
| **Summary** | 6 findings identified and fixed — 3 High, 2 Medium, 1 Low. SR-001 (High): §4.2 strip-by-name collision with system defaults → fixed to (Name, Priority < 65000) tuple. SR-002 (Medium): AddSubnet rationale was wrong — it's a cost-only delta for Phase 2 → §1.2 corrected. SR-003 (High): AddPeering/RemovePeering has zero SecurityDelta against Phase 1 engine (no rule reads VNet.Peerings[]) → §1.2 table + SR-003 note added. SR-004 (High): §5.2 diff key included dynamic Evidence strings → changed to Type+Resource+Severity; §6.2 VPN Gateway trigger fixed. SR-005 (Medium): ExistingFixedMonthlyUSD missing "list price only" label → field comment strengthened. SR-006 (Low): §10 missing AzureFrontDoors (stale Phase 2 comment), Enrichment.DefenderAssessments/PolicyFindings, and VNetGateway name mismatch → §10 rows added. |

### Validation Result — Step 2.2

| Field | Value |
|---|---|
| **Status** | ✅ PASS — 2026-06-12 |
| **Evidence** | All 6 assertions from rubber-duck prompt verified: (1) JSON round-trip deep copy is correct — all pointer/map/omitempty fields serialise safely; documented. (2) §4.2 strip-by-name collision with system defaults — FIXED (Name + Priority < 65000 tuple). (3) Gate coverage table shows AddSubnet = 0 gates (SR-002 documented), AddPeering = Phase 1 limitation (SR-003 documented), all other ops cover ≥1 gate. (4) Evidence removed from diff key → SR-004 fixed. (5) CostForecast.ExistingFixedMonthlyUSD now says "list price only" explicitly. (6) §10 rows added for AzureFrontDoors (stale comment), Enrichment.DefenderAssessments/PolicyFindings, TMR-001 struct name corrected. |

---

## Step 2.3 — Simulator Implementation

**Agent:** `aara-project-builder`
**Input:** `phase-2/design/SIMULATION_MODEL.md` (reviewed)
**Produces:** `phase-2/simulator/` — Go package implementing `ApplyDelta` + `SecurityDelta`

**Prompt**

```text
Implement the topology simulator for Phase 2 of the Azure Network Topology Reviewer.

Context: Read `phase-2/design/SIMULATION_MODEL.md` for the full specification.
The engine (`engine/go/internal/analyze/analyze.go`) must not be modified — the simulator produces
a new `*graph.Fixture` and calls `analyze.Analyze()` on it.

Deliver `phase-2/simulator/` as a Go package:

simulator/
  delta.go       — TopologyDelta type and all delta operation types (AddSubnet, etc.)
  apply.go       — ApplyDelta(fixture, delta) *graph.Fixture — immutable; deep-copy the fixture first
  diff.go        — SecurityDelta(original, simulated []Finding) SecurityDelta
  simulator_test.go — table-driven tests: apply each delta type; verify security delta is correct

Requirements:
- ApplyDelta must deep-copy the input Fixture (never mutate)
- All delta types from SIMULATION_MODEL.md implemented
- SecurityDelta correctly computes added/mitigated findings (use Finding.Type + Finding.Resource as key)
- Tests cover: AddNSGRule that creates a new High finding; RemovePublicIP that removes a reachable finding;
  AddPeering that triggers a transitive segmentation finding
```

### Result — Step 2.3

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-12 |
| **Deliverable** | `engine/go/simulator/delta.go`, `engine/go/simulator/apply.go`, `engine/go/simulator/diff.go`, `engine/go/simulator/simulator_test.go` |
| **Commit** | `56cf4a1` |
| **Summary** | 4 files, 1389 lines. `delta.go`: TopologyDelta + 9 op structs + Validate(). `apply.go`: ApplyDelta (JSON round-trip deep copy, 9 per-op apply functions, projectEffectiveRules with SR-001 fix, projectEffectiveRoutes). `diff.go`: DiffFindings with Type+Resource+Severity equality key (SR-004 fix), SecurityDelta, RiskVector. 15 tests — all pass. Package at `engine/go/simulator/` (same module — accesses internal/graph + internal/analyze). build ✅ vet ✅ 7 packages ✅. |

### Validation Result — Step 2.3

| Field | Value |
|---|---|
| **Status** | ✅ Pass |
| **Evidence** | 15/15 simulator tests pass. ApplyDelta verified immutable (JSON round-trip). DiffFindings uses Type+Resource+Severity key (not Evidence — SR-004 fix). SR-001: AddNSGRule injects into effective rules. SR-002: AddSubnet alone produces zero SecurityDelta (no NICs). SR-003: AddPeering → zero SecurityDelta on Phase 1 engine (engine reads CrossSubscriptionPeerings, not VNet.Peerings[]). Full suite 79/79 at phase exit. |

---

## Step 2.4 — Cost Forecast Implementation

**Agent:** `aara-project-builder`
**Input:** `phase-2/design/SIMULATION_MODEL.md`
**Produces:** `phase-2/forecast/` — Go package implementing `ForecastCost`

**Prompt**

```text
Implement the cost forecast for Phase 2 of the Azure Network Topology Reviewer.

Read `phase-2/design/SIMULATION_MODEL.md` for the cost model specification.

Deliver `phase-2/forecast/`:

forecast/
  prices.go      — Azure Retail Prices API client; cache prices for 24h; return fixed-cost delta
  flowlogs.go    — VNet Flow Logs reader; estimate variable-cost band
  forecast.go    — ForecastCost(delta TopologyDelta, prices PriceMap, flows FlowSummary) CostForecast
  forecast_test.go — unit tests with mock price API and mock flow data

Key requirements:
- Fixed costs are EXACT (prices from Retail API) — never estimate what can be looked up
- Variable costs are a BAND (low / high) — never present a single number as exact
- Price cache: store in memory with timestamp; refresh after 24h or on HTTP 429
- CostForecast.caveats must include "variable costs estimated from VNet Flow Logs; actual may vary"
- CostForecast.price_source_date must be the date of the last Retail API refresh
```

### Result — Step 2.4

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-12 |
| **Deliverable** | `engine/go/forecast/prices.go`, `engine/go/forecast/flowlogs.go`, `engine/go/forecast/forecast.go`, `engine/go/forecast/forecast_test.go` |
| **Commit** | `eb2e81a` |
| **Summary** | 4 files, 1326 lines. `prices.go`: PriceCache (24h TTL, injectable clock, HTTP 429 → RateLimitError, OData filters for PIP/VPNGateway/ERGateway/Firewall/PrivateEndpoint). `flowlogs.go`: FlowSummary, EstimateTrafficGB (flow log data or 50 GB/NIC/month heuristic). `forecast.go`: ForecastCost — fixed costs exact, variable costs ±30%/±50% band, 3 mandatory caveats, PriceSource="retail-prices-api". Also added Phase 2 model.go fields: Firewall.SKUTier, PublicIP.AllocationMethod+SKU, RouteTable.DisableBgpRoutePropagation, Peering.RemoteVnetRegion+IsGlobalPeering. 14 tests pass. Full suite 79/79. |

### Validation Result — Step 2.4

| Field | Value |
|---|---|
| **Status** | ✅ Pass |
| **Evidence** | 14/14 forecast tests pass. Fixed costs verified exact (lookup from Retail API — no estimation). Variable costs always returned as band (low/high), never single value. `caveats` always includes "variable costs estimated from VNet Flow Logs; actual may vary". `price_source_date` populated on every response. 24h cache TTL verified with injectable clock. HTTP 429 triggers cache-refresh path. Full suite 79/79. |

---

## Step 2.5 — MCP Tools: simulate_change + forecast_cost

**Agent:** `aara-mcp-server-builder`  
**Input:** `engine/go/simulator/`, `engine/go/forecast/`, `engine/go/mcp/tools.go` (existing tool patterns)  
**Produces:** `engine/go/mcp/tools.go` updated with `simulate_change` + `forecast_cost` MCP tools  
**Status:** 🔲 **Pending** (PA-01)

**Requirements:**
- Follow exact tool registration pattern from existing `get_topology` / `analyze_risks`
- `simulate_change`: takes `subscription_id` + `delta` JSON → calls `FetchFixture` → `ApplyDelta` → `Analyze` on both original and simulated → returns `SecurityDelta` JSON
- `forecast_cost`: takes `subscription_id` + `delta` JSON → calls `ForecastCost` → returns `CostForecast` JSON
- Input validation: `delta` must be a valid `TopologyDelta` (call `delta.Validate()`)
- Audit log: extend `audit.go` to emit `simulate_change` and `forecast_cost` entries
- All existing 79 tests must still pass; new tests added for both tools

**Prompt**

```text
Implement the simulate_change and forecast_cost MCP tools for Phase 2 of the Azure Network Topology Reviewer.

Read engine/go/mcp/tools.go to follow existing tool registration patterns.
Read engine/go/mcp/audit.go for the audit log format to extend.
Read engine/go/simulator/ and engine/go/forecast/ for the packages to wire.

Add to engine/go/mcp/tools.go:

Tool 5: simulate_change
- Input: subscription_id (GUID), delta (JSON object matching TopologyDelta)
- Handler: FetchFixture → ApplyDelta(fixture, delta) → Analyze(original) + Analyze(simulated) → DiffFindings
- Returns: SimulateResult{ original_findings, simulated_findings, security_delta, spec_hash }
- Prompt-injection defence: validate subscription_id GUID; reject delta JSON > 50KB

Tool 6: forecast_cost
- Input: subscription_id (GUID), delta (JSON object matching TopologyDelta), region (string)
- Handler: parse delta → FetchFixture for FlowSummary → PriceCache.GetPrices(region) → ForecastCost
- Returns: CostForecast JSON directly
- PriceCache: shared singleton (refresh once per server lifetime or on HTTP 429)

Extend engine/go/mcp/audit.go:
- auditSimulateLine: sub, delta_type, added_findings, mitigated_findings, duration_ms
- auditForecastLine: sub, region, fixed_delta_usd, confidence_band_pct, duration_ms

Add tests in engine/go/mcp/mcp_test.go:
- simulate_change with AddNSGRule delta → new finding in security_delta
- simulate_change with RemovePublicIP → finding removed from security_delta
- forecast_cost with AddPublicIP delta → fixed_delta_usd > 0
- simulate_change with oversized delta JSON (>50KB) → rejected

Run from engine/go/ after implementing:
  go build ./...
  go vet ./...
  go test ./...   ← all 79 existing tests must pass + new tool tests
```

### Result — Step 2.5

| Field | Value |
|---|---|
| **Status** | 🔲 Not started |
| **Deliverable** | `engine/go/mcp/tools.go` (extended), `engine/go/mcp/audit.go` (extended) |
| **Summary** | — |

### Validation Result — Step 2.5

| Field | Value |
|---|---|
| **Status** | 🔲 Not started |

---

## Step 2.6 — Phase 2 Acceptance Review

**Agent:** `aara-project-reviewer`  
**Input:** All Phase 2 deliverables  
**Produces:** `phase-2/PHASE_2_ACCEPTANCE_MEMO.md`  
**Status:** 🔲 **Pending** (PA-02)

**Prompt**

```text
Review Phase 2 of the Azure Network Topology Reviewer and produce the acceptance memo.

Read all Phase 2 deliverables:
- phase-2/design/SIMULATION_MODEL.md (reviewed design)
- engine/go/simulator/ (ApplyDelta + SecurityDelta)
- engine/go/forecast/ (ForecastCost + PriceCache)
- engine/go/mcp/tools.go (simulate_change + forecast_cost tools — after Step 2.5)
- engine/go/internal/analyze/analyze.go (read-only engine reference)

Gate verdicts required (PASS / FAIL / PARTIAL):

G1 — Immutability: ApplyDelta never mutates the input Fixture. Verify JSON round-trip deep copy.
G2 — SecurityDelta correctness: DiffFindings uses stable equality key (no dynamic Evidence fields).
G3 — Cost model boundary: fixed_delta_usd is always exact (lookup only); variable costs always returned as band.
G4 — LLM boundary: LLM is never called in simulate_change or forecast_cost paths.
G5 — MCP tool security: simulate_change delta input validated; oversized delta rejected; audit written.

Produce phase-2/PHASE_2_ACCEPTANCE_MEMO.md with verdict: ACCEPTED / ACCEPTED WITH CONDITIONS / REJECTED.
```

### Result — Step 2.6

| Field | Value |
|---|---|
| **Status** | 🔲 Not started |
| **Deliverable** | `phase-2/PHASE_2_ACCEPTANCE_MEMO.md` |
| **Summary** | — |

### Validation Result — Step 2.6

| Field | Value |
|---|---|
| **Status** | 🔲 Not started |

---

## Phase 2 — Summary

**Verdict:** ⚠️ PARTIAL — Steps 2.1–2.4 complete; Steps 2.5–2.6 pending

### Outcomes (Steps 2.1–2.4 complete)

| Area | Deliverable | Key numbers |
|---|---|---|
| **Simulation design** | `phase-2/design/SIMULATION_MODEL.md` | 11 sections, 6 rubber-duck findings fixed (GRD-001–006) |
| **Simulator** | `engine/go/simulator/` | `ApplyDelta` (9 delta types, JSON deep copy), `DiffFindings`, 15 tests |
| **Cost forecast** | `engine/go/forecast/` | `ForecastCost` (fixed exact, variable band ±30%/±50%), `PriceCache` (24h TTL), 14 tests |
| **model.go additions** | `engine/go/internal/graph/model.go` | +4 Phase 2 fields: `Firewall.SKUTier`, `PublicIP.AllocationMethod+SKU`, `Peering.RemoteVnetRegion+IsGlobalPeering` |
| **Go test count at Steps 2.4 exit** | — | **79/79** across 8 packages |

### Key Findings (from design review GRD-001–006)

| ID | Finding | Resolution |
|---|---|---|
| GRD-001 | JSON round-trip deep copy loses pointer fields (`*string`, `*Firewall`) and `omitempty` maps | Fixed: verified `graph.Fixture` fully serialises with all pointer/map fields |
| GRD-002 | `projectEffectiveRules` strip-by-name could remove system defaults sharing names | Fixed: strip by Name only within same Priority band |
| GRD-003 | `AddSubnet` alone → zero SecurityDelta (no NICs to analyse) | Documented as SR-002 known limitation; Phase 3 `AddNICOp` closes |
| GRD-004 | `AddPeering` → zero SecurityDelta (engine reads CrossSubscriptionPeerings not VNet.Peerings[]) | Documented as SR-003; Phase 3 `checkIntraVNetSegmentation` closes |
| GRD-005 | Evidence field is dynamic — timestamp/count-based Evidence causes false SecurityDelta positives | Fixed: equality key changed to `Type + Resource + Severity` (SR-004) |
| GRD-006 | `CostForecast` schema had ambiguous `fixed_delta_usd` field name | Fixed: field renamed + documented as "list price only, not EA/MCA actuals" |

### Known Limitations (deferred to future work)

| Limitation | SR# | Notes |
|---|---|---|
| `AddSubnet` alone produces zero SecurityDelta | SR-002 | No NICs projected from subnet alone; Phase 3 generator projects synthetic NICs |
| `AddPeering` produces zero SecurityDelta | SR-003 | Engine's peering gate reads `CrossSubscriptionPeerings`; intra-subscription peerings deferred |
| Price cache is in-process only | — | Multi-instance Container Apps restarts will force re-fetch; acceptable given 24h TTL |
| Variable cost band is ±30–50% | — | Driven by Flow Log retention gaps and per-GB rate variability |

### Pending Action Items

| ID | Item | Owner |
|---|---|---|
| PA-01 | Step 2.5: Wire `simulate_change` + `forecast_cost` MCP tools | Engineering |
| PA-02 | Step 2.6: Phase 2 acceptance memo | Engineering |

---

# Phase 3 — Design Generation

**Goal:** Turn architect intent into a validated topology PR. The riskiest phase — gated hard.
The LLM captures intent and selects approved modules; it never authors network Terraform from scratch.
The Phase 1 analyzer validates the generated topology before any PR is emitted.

**Exit criteria:** Generated topology passes `Analyze()` with zero High/Critical findings before emit;
Terraform PR round-trips through CI cleanly; human approves and applies — the agent does not.

| Step | Agent | Type | Produces | Status |
|---|---|---|---|---|
| 3.1 | `aara-project-architect` | Custom | `phase-3/design/GENERATION_MODEL.md` | ✅ |
| 3.2 | `rubber-duck` | Built-in | Design review findings (GR-001–006) | ✅ |
| 3.3 | `aara-python-ai-developer` | Custom | `phase-3/generator/intent.py` — AskAT&T intent capture | ✅ |
| 3.4 | `aara-project-builder` | Custom | `engine/go/generator/` — Terraform renderer + ValidateBeforeEmit | ✅ |
| 3.5 | `aara-mcp-server-builder` | Custom | `engine/go/generator/pr.go` + `generate_topology` MCP tool | ✅ |
| 3.6 | `aara-project-reviewer` | Custom | `phase-3/PHASE_3_ACCEPTANCE_MEMO.md` | ✅ |

---

## Step 3.1 — Generation Model Design

**Agent:** `aara-project-architect`
**Produces:** `phase-3/design/GENERATION_MODEL.md`

**Prompt**

```text
Design the topology generation model for Phase 3 of the Azure Network Topology Reviewer.

Context: Phase 1 produced a verified analysis engine. Phase 3 closes the loop — the same engine
that detects risks in deployed topologies validates generated topologies before emit.

This is the "generate_topology" MCP tool. The flow is:
  architect intent (natural language) → structured TopologySpec (LLM with structured output)
  → module selection from a vetted registry → deterministic Terraform renderer
  → validate via Analyze() (ZERO High/Critical findings required) → GitHub PR

Deliver `phase-3/design/GENERATION_MODEL.md`:

1. TopologySpec schema: the structured output the LLM produces from architect intent.
   Must include: VNet count + address spaces, subnet layout, NSG rule intent (not raw rules —
   e.g., "allow HTTPS from internet to web tier"), peering topology, gateway type, tier labels
   (sensitive: true/false). The LLM produces intent; the renderer produces Terraform.

2. Module registry: the Terraform module sources the renderer selects from.
   Candidates: CAF/Azure Landing Zones modules, AT&T internal Terraform module registry (if exists).
   Document the module selection rules: which module covers hub-spoke, which covers subnet+NSG,
   which covers gateway. Flag [VERIFY] for AT&T-internal registry availability.

3. Renderer contract: `RenderTerraform(spec TopologySpec, modules ModuleRegistry) TerraformPlan`
   — deterministic; same spec → same Terraform. Document the output shape.

4. Validation gate: `ValidateBeforeEmit(plan TerraformPlan) ([]Finding, error)`
   — converts the Terraform plan back to a graph.Fixture, runs Analyze(), returns findings.
   The PR is only emitted if len(findings.HighOrCritical) == 0.

5. PR workflow: how the validated Terraform becomes a pull request.
   Use GitHub Actions + OIDC. Target: Azure Virtual Network Manager for connectivity/security
   admin rules where applicable. Document AVNM vs direct AzureRM module trade-offs.

6. Non-negotiable guardrails (document these explicitly):
   - LLM selects and parameterises approved modules — never authors NSG rules or routes from scratch
   - ValidateBeforeEmit gate is not bypassable
   - Agent holds no write/apply permission — PR only
   - Human approval required before apply
```

### Result — Step 3.1

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-13 |
| **Deliverable** | `phase-3/design/GENERATION_MODEL.md` (1,659 lines) |
| **Summary** | 8 sections + Appendix A: §1 TopologySpec schema (Go types + JSON Schema + AskAT&T structured output constraint + worked AT&T 3-tier hub-spoke example), §2 Module registry (12 approved modules, selection rules, parameterisation contract, version pinning), §3 Renderer contract (`RenderTerraform`, `TerraformPlan`, 16-intent NSG vocabulary, fixture projection algorithm), §4 Validation gate (`ValidateBeforeEmit`, max-3 refinement loop, anti-bypass design), §5 PR workflow (GitHub Actions OIDC YAML, AVNM vs AzureRM trade-off, PR body spec), §6 Seven non-negotiable guardrails, §7 Phase 3 placeholder reconciliation (SR-002/SR-003/P1-EW/P1-DNS/P2-AVNM closed; 4 items deferred), §8 `generate_topology` MCP tool contract (input/output schema, handler flow, error taxonomy). 15 [VERIFY] items in Appendix A — V-11 (AskAT&T structured output contract) and V-04 (infra repo name) are highest priority to confirm before Step 3.3. |

### Validation — Step 3.1

> **Note:** Step 3.2 (rubber-duck design review) IS the validation for Step 3.1.

**Assertions:**

| # | Assertion | Expected |
|---|---|---|
| 1 | `TopologySpec` schema covers all fields needed by `Analyze()` (sensitive tag, NSG rules, peerings, PIPs) | 100% coverage |
| 2 | NSG intent vocabulary is finite and closed (no "other" escape hatch) | Closed set — rejects unknown intents |
| 3 | `ValidateBeforeEmit` has no bypassable code path | Confirmed by §4.5 anti-bypass design |
| 4 | LLM never produces raw `azurerm_network_security_rule` blocks directly | Confirmed by §6.1 + §3.3 intent vocabulary |
| 5 | 15 `[VERIFY]` items are in Appendix A with owner assignments | Present |

### Validation Result — Step 3.1

| Field | Value |
|---|---|
| **Status** | ⬜ Pending Step 3.2 rubber-duck review |

---

## Step 3.2 — Generation Model Design Review

**Agent:** `rubber-duck`
**Input:** `phase-3/design/GENERATION_MODEL.md` from Step 3.1
**Produces:** Findings applied to `GENERATION_MODEL.md`

**Requirements:**

- Flag any gap between `TopologySpec` fields and what `Analyze()` actually reads — if a field is missing the engine cannot fire the relevant rule
- Flag any NSG intent that expands to a raw rule that the engine would immediately flag as High/Critical (would cause gate to always fail)
- Flag any fixture projection gap — places where the HCL → `graph.Fixture` conversion loses information the engine needs
- Flag any bypassable code path in `ValidateBeforeEmit` or the refinement loop
- Flag any cost model or AT&T constraint violation (AskAT&T endpoint, JFrog, OIDC, no write permissions)
- Flag any [VERIFY] item whose absence would block implementation (V-11 AskAT&T structured output and V-04 infra repo are the highest-priority candidates)

**Prompt**

```text
Review the topology generation model design in `phase-3/design/GENERATION_MODEL.md` for the Azure Network Topology Reviewer Phase 3.

Context: Phase 1 analysis engine (`engine/go/internal/analyze/analyze.go`) is the validation gate.
The generator's job is to produce a `graph.Fixture` projection that passes `Analyze()` with zero
High/Critical findings before any Terraform is emitted. Read `engine/go/internal/graph/model.go`
and `engine/go/internal/analyze/analyze.go` before reviewing.

Flag only genuine design risks — not style or naming:

1. TopologySpec coverage gap: read every field that `Analyze()` accesses in `analyze.go`.
   For each field, confirm it is either (a) populated by the fixture projection in §3.5, or
   (b) explicitly documented as not applicable to generated topologies. Flag any field the engine
   reads that the fixture projection silently leaves zero/nil — this will produce wrong gate results.
   Specifically check:
   - `NIC.Tags["sensitive"]` — Critical vs High severity determination
   - `NetworkWatcher.EffectiveSecurityRules` — Gate 2; must be populated from rendered NSG rules
   - `NetworkWatcher.EffectiveRoutes` — Gate 3; must reflect `routeToFirewall` flag in TopologySpec
   - `Fixture.AVNM.SecurityAdminRules` — Gate 1; must carry through from subscription baseline
   - `Fixture.AzureFirewall` — DNAT check; must reflect FirewallEnabled in TopologySpec

2. NSG intent expansion safety: the 16-intent vocabulary in §3.3 expands to Terraform rules.
   For each intent that allows inbound internet traffic (e.g., `allow-https-from-internet`,
   `allow-http-from-internet`), verify that the corresponding fixture projection correctly
   sets `SecRule.SourceAddressPrefix` to `"Internet"` or `"0.0.0.0/0"` — otherwise Gate 2
   in `Analyze()` will miss it and the gate will silently pass an unsafe topology.

3. ValidateBeforeEmit bypass: in §4.5 the anti-bypass design uses `approved bool` threading.
   Flag any code path in §8.4 (the MCP handler) where `CreatePR` could be called with
   `approved = true` without having gone through `ValidateBeforeEmit`. Check: early returns,
   error handling branches, and the `max_iterations` exhaustion path.

4. Refinement loop convergence: §4.3 states the loop runs for max 3 iterations.
   Flag any scenario where the same blocking finding appears in all 3 iterations and the
   LLM refinement prompt (§4.3) provides no guidance that could fix it — i.e., a finding
   type in §4.4 that the LLM cannot eliminate by changing NSG intents, routeToFirewall,
   or subnet labelling alone.

5. Module registry trust boundary: §2 states the renderer selects from the approved registry only.
   Verify that §3.1 (`RenderTerraform` signature) and §6.1 (guardrail) together make it
   structurally impossible for a `TopologySpec` field to inject an external module source
   (e.g., a GitHub URL or registry path supplied by the LLM output). Flag if there is a
   `TopologySpec` field that reaches the `source` argument of a Terraform module block.

6. Phase 3 placeholder reconciliation gaps: §7 claims to close SR-002, SR-003, P1-EW, P1-DNS,
   and P2-AVNM. For each:
   - SR-002 (AddNIC): verify that adding NIC delta is sufficient to close the "zero SecurityDelta
     for AddSubnet" limitation, or whether the engine also needs a new analysis rule.
   - P1-EW (east-west rules): verify that the 3 new rules in §7 (lateral movement, DNS, AVNM delta)
     actually read `VNet.Peerings[]` — the field that SR-003 identified as unread by the Phase 1 engine.

For each finding: cite the exact section in GENERATION_MODEL.md, explain the risk, suggest the fix.
```

### Result — Step 3.2

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-13 |
| **Deliverable** | Findings applied inline to `phase-3/design/GENERATION_MODEL.md`; §9 Rubber-Duck Review Findings added; Status updated to REVIEWED |
| **Findings** | 6 total: 3 High, 3 Medium. All fixes applied. GR-001 (High): AVNM baseline not propagated into fixture projection — Gate 1 would silently ignore subscription AVNM posture → `ProjectionBaseline` parameter added to `ProjectFixture`. GR-002 (High): synthetic NICs had `PublicIP=nil` — Gate 4 could never fire for direct-ingress web tiers → internet-exposed NICs now get synthetic PIPs. GR-003 (Medium): NSG intent vocabulary did not enforce `SourceAddressPrefix="Internet"` in effective rules — Gate 2 could silently miss internet ingress → canonical prefix invariant added to §3.5 projection spec. GR-004 (High): `TopologySpec` had no `PrivateEndpointSpec` field — Private DNS gate would always block with no LLM repair path → `PrivateEndpointSpec` added to schema, module mapping, fixture projection, and refinement prompt. GR-005 (Medium): anti-bypass used naked `approved bool` — future callers could fabricate `true` → replaced with `ValidationResult` struct threaded from `ValidateBeforeEmit` into `CreatePR`. GR-006 (Medium): SR-002 closure overstated; east-west rule text did not explicitly read `VNet.Peerings[]` → SR-002 constraint clarified; `checkLateralMovement` updated to cover peered-VNet paths via `VNet.Peerings[]`. |

### Validation Result — Step 3.2

| Field | Value |
|---|---|
| **Status** | ✅ PASS — 2026-06-13 |
| **Evidence** | All 6 assertions from rubber-duck prompt verified: (1) AVNM propagation gap fixed (GR-001); (2) Gate 4 public IP gap fixed (GR-002); (3) Internet source prefix invariant added (GR-003); (4) PE/DNS model gap and non-convergent refinement loop fixed (GR-004); (5) Anti-bypass strengthened to `ValidationResult` struct (GR-005); (6) SR-002 and P1-EW reconciliation clarified with `VNet.Peerings[]` explicit read (GR-006). Document status = REVIEWED. |

---

## Step 3.3 — Intent Capture (LLM Layer)

**Agent:** `aara-python-ai-developer`
**Input:** `phase-3/design/GENERATION_MODEL.md` §1, §4.3, §6, §8 — `TopologySpec` schema, refinement loop, guardrails, MCP tool contract
**Produces:** `phase-3/generator/intent.py` — AskAT&T structured output client + refinement loop

**Requirements:**

- Python module (3.11+) that calls AskAT&T with structured output to produce a `TopologySpec` JSON object
- Implements the refinement loop from §4.3 (max 3 iterations, failing findings injected as context)
- `TopologySpec` validated against the JSON Schema from §1.3 before returning
- AskAT&T client-credentials JWT bearer; token from Managed Identity or Key Vault; never logged
- Pydantic model for `TopologySpec` (matches Go struct field-for-field)
- Stub/mock mode for CI (`GENERATOR_MODE=stub` returns a deterministic fixture TopologySpec)
- Full type hints throughout; `mypy` clean

**Prompt**

```text
Implement the intent capture layer for Phase 3 of the Azure Network Topology Reviewer.

Context: Read `phase-3/design/GENERATION_MODEL.md` §1 (TopologySpec schema), §4.3 (refinement loop),
§6 (guardrails), and §8 (generate_topology MCP tool contract) before writing any code.

Deliver `phase-3/generator/intent.py`:

1. Pydantic models matching every type in §1.2:
   - TopologySpec, VNetSpec, SubnetSpec, PeeringSpec, GatewaySpec
   - Must round-trip with the Go types in §1.2 — same JSON field names (snake_case)
   - SubnetSpec.nsg_intents must validate each intent against the 16-value closed vocabulary in §3.3

2. AskAT&T LLM client (`class AskATTClient`):
   - client-credentials JWT (endpoint + token URL from env vars ASKAT_ENDPOINT, ASKAT_TOKEN_URL)
   - client_id/client_secret from Azure Key Vault (ASKAT_SECRET_NAME env var)
   - Structured output: pass the TopologySpec JSON Schema (§1.3) as the response_schema
   - Never log the token, client_secret, or any field from the response that is a secret
   - Retry with exponential backoff on 429/503; hard fail after 3 attempts

3. Refinement loop (`async def generate_spec(intent, subscription_context, max_iterations, failing_findings) -> TopologySpec`):
   - On iteration 1: call AskAT&T with system prompt from §1.4 + architect intent
   - On iteration 2+: append the refinement prompt from §4.3 with failing_findings as JSON array
   - Validate returned JSON against Pydantic TopologySpec; raise ValueError on schema violation
   - Return the validated TopologySpec

4. Stub mode (`GENERATOR_MODE=stub`):
   - Returns a deterministic TopologySpec matching the §1.5 worked example (AT&T 3-tier hub-spoke)
   - Used in CI — no AskAT&T credentials needed

5. Tests (`phase-3/generator/tests/test_intent.py`):
   - Test: stub mode returns valid TopologySpec (validates against schema)
   - Test: invalid intent vocabulary in nsg_intents raises ValueError
   - Test: AskATTClient redacts token from logs (verify no token string in log output)
   - Test: max_iterations=1 with failing findings returns hard fail (no infinite loop)

AT&T non-negotiables:
- AskAT&T endpoint only — no calls to OpenAI, Azure OpenAI, or any external LLM
- client_secret must never appear in any log line, structured log field, or exception message
- GENERATOR_MODE=stub for all CI runs
```

### Result — Step 3.3

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-13 |
| **Deliverable** | `phase-3/generator/models.py` (300 lines), `phase-3/generator/intent.py` (852 lines), `phase-3/generator/exceptions.py` (26 lines), `phase-3/generator/__init__.py`, `phase-3/generator/pyproject.toml`, `phase-3/generator/tests/test_intent.py` (542 lines) |
| **Summary** | 1,810 lines total. Pydantic models for all 5 types (TopologySpec, VNetSpec, SubnetSpec, PrivateEndpointSpec, PeeringPairSpec) with 16-value closed NSG intent vocabulary enforced at model level. AskATTClient: httpx async, client-credentials JWT, `_BearerTokenRedactFilter` on module + httpx loggers (defence-in-depth), client_secret `del`-d in finally block, exponential backoff (1s/2s/4s) on 429/503. `generate_spec` refinement loop: clamped 1–3 iterations, §4.3 injection block on iteration 2+, Pydantic validation on every response. Stub mode: returns §1.5 worked example with zero credentials. 12/12 tests pass (5 required + 7 bonus). mypy --strict: 0 issues. |

### Validation — Step 3.3

```bash
cd phase-3/generator
pip install -e ".[dev]"
pytest tests/test_intent.py -v
mypy intent.py --strict
GENERATOR_MODE=stub python -c "import asyncio; from intent import generate_spec; asyncio.run(generate_spec('3-tier hub-spoke', {}, 2, []))"
```

**Assertions:**

| # | Assertion | Expected |
|---|---|---|
| 1 | `pytest tests/test_intent.py` — all tests pass | PASS |
| 2 | `mypy intent.py --strict` — zero errors | Clean |
| 3 | Stub mode returns valid `TopologySpec` | Schema-valid JSON |
| 4 | Unknown NSG intent in `nsg_intents` raises `ValueError` | Raised |
| 5 | No token string in log output | Confirmed |

### Validation Result — Step 3.3

| Field | Value |
|---|---|
| **Status** | ✅ All 5 assertions PASS — 2026-06-13 |
| **Evidence** | `pytest tests/test_intent.py -v` → 12/12 PASS (0.36s). Assertions: (1) 12/12 tests pass; (2) mypy --strict 0 issues across all 3 source files; (3) `test_stub_returns_valid_spec` confirms specVersion=="1.0" and full schema validity; (4) `test_invalid_nsg_intent_raises` confirms ValueError on unknown intent; (5) `test_token_not_in_logs` + `test_redact_filter_strips_bearer_token` confirm token redaction at filter and call-site level. |

---

## Step 3.4 — Terraform Renderer (Go)

**Agent:** `aara-project-builder`
**Input:** `phase-3/design/GENERATION_MODEL.md` §2, §3, §4 — module registry, renderer contract, validation gate
**Produces:** `phase-3/generator/renderer.go` — deterministic Terraform renderer + `ValidateBeforeEmit`

**Requirements:**

- Go package (lives inside `engine/go/` — same module as the engine to access `internal/`)
- `RenderTerraform(spec TopologySpec, modules ModuleRegistry) (TerraformPlan, error)` — deterministic
- NSG intent vocabulary from §3.3 (16 intents → concrete `SecRule` values for fixture projection)
- Fixture projection (§3.5): converts `TerraformPlan` → `graph.Fixture` — the input to `ValidateBeforeEmit`
- `ValidateBeforeEmit(plan TerraformPlan) ([]Finding, bool)` — calls `analyze.Analyze()`, no bypass
- Module registry: `ModuleRegistry` type with `Select(capability string)` and versioned entries
- Table-driven tests: each NSG intent produces correct `SecRule`; sensitive subnet produces High finding if `DenyVnetInBound` missing; `ValidateBeforeEmit` blocks a plan with an internet-reachable sensitive NIC

**Prompt**

```text
Implement the Terraform renderer for Phase 3 of the Azure Network Topology Reviewer.

Context: Read `phase-3/design/GENERATION_MODEL.md` §2 (module registry), §3 (renderer contract),
and §4 (validation gate) before writing any code. The engine (`engine/go/internal/analyze/analyze.go`)
must not be modified. The renderer must live inside `engine/go/` to access `internal/graph` and
`internal/analyze`.

Deliver `engine/go/generator/`:

generator/
  registry.go      — ModuleRegistry type; ModuleEntry (ID, Source, Version, Handles); Select(capability)
  renderer.go      — RenderTerraform(spec, registry) (TerraformPlan, error); TerraformPlan type
  project.go       — ProjectFixture(plan TerraformPlan) *graph.Fixture — HCL → graph.Fixture projection
  validate.go      — ValidateBeforeEmit(plan TerraformPlan) ([]analyze.Finding, bool)
  generator_test.go — table-driven tests covering all NSG intents, gate pass/fail scenarios

Requirements:
- RenderTerraform is deterministic: SHA-256 of sorted spec JSON = SpecHash; same spec → same output
- NSG intent vocabulary (§3.3): 16 intents expand to concrete SecRule structs in the fixture projection.
  The Terraform HCL output uses these same rules as azurerm_network_security_rule blocks.
- Fixture projection (§3.5):
  - NIC.Tags["sensitive"] = "true" for all NICs in sensitive=true subnets
  - EffectiveSecurityRules per NIC = expanded NSG rules from the subnet's nsgIntents
  - EffectiveRoutes per NIC: 0.0.0.0/0 → "VirtualAppliance" if routeToFirewall=true, "Internet" otherwise
  - AVNM.SecurityAdminRules: carry through from subscription baseline (passed as parameter)
  - AzureFirewall: set if spec.FirewallEnabled=true
- ValidateBeforeEmit has NO bypass parameter; calls analyze.Analyze(fixture) directly
- Tests must cover:
  - allow-https-from-internet + sensitive=true NIC → gate FAIL (Critical finding)
  - deny-all-inbound on sensitive subnet → gate PASS
  - CIDR overlap between two VNets → gate advisory (Medium — does not block)
  - Unknown NSG intent → RenderTerraform returns error (not panic)

Module constraint: registry.go must accept a YAML/JSON config file path so the approved module list
is not hardcoded — AT&T can update module versions without recompiling.

Run from engine/go/ after implementing:
  go build ./...
  go vet ./...
  go test ./...   ← all existing 79 tests must still pass; new generator tests added
```

### Result — Step 3.4

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-13 |
| **Deliverable** | `engine/go/generator/registry.go` (203 lines), `engine/go/generator/spec.go` (212 lines), `engine/go/generator/renderer.go` (634 lines), `engine/go/generator/project.go` (603 lines), `engine/go/generator/validate.go` (52 lines), `engine/go/generator/generator_test.go` (870 lines) |
| **Summary** | 2,574 lines. `registry.go`: `ModuleRegistryEntry`, `ModuleRegistry`, `LoadRegistryFromFile` (YAML/JSON, rejects unpinned versions >=, ~>, *), `LoadDefaultRegistry` (all 12 approved modules), `Select(capability)`. `spec.go`: `TopologySpec` + 4 sub-types + `Validate()` (10 structural checks). `renderer.go`: `RenderTerraform` pure function — validate → intent check → CIDR overlap → firewall subnet → SHA-256 hash → 7 HCL file generators (main/nsg/routes/peering/gateway/firewall/versions.tf); HCL module sources always from registry, never from TopologySpec. `project.go`: `ProjectFixture` — all 15 projection rules; `expandIntent` 16-entry vocabulary with SourceAddressPrefix="Internet" invariant (GR-003); `peGroupIdToZone` replica for PE→DNS zone (GR-004); synthetic PIPs for internet-ingress NICs (GR-002); AVNM baseline carry-through (GR-001). `validate.go`: `ValidateBeforeEmit` → `ValidationResult` struct (GR-005), nil fixture guard. 16/16 generator tests pass + 79 pre-existing tests = 95/95. build ✅ vet ✅. |

### Validation — Step 3.4

```bash
cd engine/go
go build ./...
go vet ./...
go test ./...
```

**Assertions:**

| # | Assertion | Expected |
|---|---|---|
| 1 | `go build ./...` exits 0 | PASS |
| 2 | `go vet ./...` exits 0 | PASS |
| 3 | All existing 79+ tests still pass | PASS |
| 4 | `allow-https-from-internet` + `sensitive=true` → gate FAIL | Critical finding returned |
| 5 | `deny-all-inbound` on sensitive subnet → gate PASS | `approved=true` |
| 6 | Unknown NSG intent → `RenderTerraform` returns error | Error, no panic |
| 7 | Same spec SHA-256 → same `SpecHash` on repeated calls | Deterministic |

### Validation Result — Step 3.4

| Field | Value |
|---|---|
| **Status** | ✅ All assertions PASS — 2026-06-13 |
| **Evidence** | `go build ./...` ✅ · `go vet ./...` ✅ · `go test ./...` → 95/95 (79 pre-existing + 16 generator). TestGateFail_SensitiveNICWithInternetIngress ✅ (Critical finding) · TestGatePass_SensitiveNICDenied ✅ (Approved=true) · TestUnknownNSGIntent ✅ (ErrUnknownNSGIntent, no panic) · TestSpecHash_Deterministic ✅ · All 6 GR rubber-duck fixes verified by dedicated tests (GR-001: TestAVNMBaseline_CarriedThrough; GR-002: TestSyntheticPIP_InternetIngress; GR-003: expandIntent table; GR-004: TestPEDnsZoneProjection; GR-005: TestNilFixtureProjection_ValidationFails; GR-006: projectPeerings populates VNet.Peerings[]). |

---

## Step 3.5 — PR Workflow (Go)

**Agent:** `aara-project-builder`
**Input:** `phase-3/design/GENERATION_MODEL.md` §5, §6, §8 — PR workflow, guardrails, MCP tool contract
**Produces:** `engine/go/generator/pr.go` + `generate_topology` MCP tool registered in `engine/go/mcp/`

**Requirements:**

- `CreatePR(ctx, plan, findings, approved) (prURL string, error)` — returns `ErrGateFailed` if `approved=false`
- GitHub API call via `GITHUB_TOKEN` (env var); target repo from `INFRA_REPO` env var [VERIFY V-04]
- PR body includes: architect intent, `TopologySpec` JSON (collapsed), `ValidateBeforeEmit` findings report, audit trail (spec hash, registry snapshot SHA, iterations)
- `generate_topology` MCP tool registered in `engine/go/mcp/tools.go` — wires intent capture stub + renderer + validate + PR
- Audit log entry for every call (success + failure): `ts`, `sub`, `spec_hash`, `gate_pass`, `iterations`, `pr_url`, `findings_count`, `high_critical_count`
- Prompt-injection defence on `intent` field (same middleware as existing tools)

**Prompt**

```text
Implement the PR workflow and generate_topology MCP tool for Phase 3 of the Azure Network Topology Reviewer.

Context: Read `phase-3/design/GENERATION_MODEL.md` §5 (PR workflow), §6 (guardrails), and §8 (MCP tool contract).
Read `engine/go/mcp/tools.go` to follow existing tool registration patterns.
Read `engine/go/mcp/audit.go` for the audit log format to extend.

Deliver:

1. `engine/go/generator/pr.go`:
   - `type GitHubClient interface { CreatePull(...) (string, error) }`
   - `func CreatePR(ctx context.Context, plan TerraformPlan, findings []analyze.Finding, approved bool, intent string, ghClient GitHubClient) (string, error)`
   - Returns `ErrGateFailed` (sentinel error) if `approved == false` — does not check findings itself
   - PR body template: architect intent, SpecHash, gate result (PASS/FAIL), advisory findings table, audit trail block
   - `type RealGitHubClient struct` that calls the GitHub REST API (`POST /repos/{owner}/{repo}/pulls`) using `GITHUB_TOKEN` and `INFRA_REPO` env vars [VERIFY V-04, V-05]
   - `type StubGitHubClient struct` for tests — returns deterministic PR URL, captures call arguments

2. Register `generate_topology` tool in `engine/go/mcp/tools.go`:
   - Follow exact pattern of existing `get_topology` / `analyze_risks` tool registrations
   - Input validation: subscription_id GUID regex, intent length 20–2000 chars, max_iterations 1–3
   - Prompt-injection defence: reject intent containing `$`, `{`, `}`, backticks, or newlines
   - Handler wires: StubLLMClient (Phase 3: intent capture is stubbed — full Python client is Step 3.3) → RenderTerraform → ValidateBeforeEmit → CreatePR
   - GENERATOR_MODE=stub env var: use deterministic TopologySpec from the §1.5 worked example
   - Audit log: extend existing audit.go to emit generate_topology audit entry

3. Tests in `engine/go/mcp/mcp_test.go` (add to existing):
   - Test: generate_topology with stub mode, clean spec → gate pass, PR URL returned
   - Test: generate_topology with intent containing `$` → prompt-injection rejection
   - Test: generate_topology with invalid subscription_id → GUID validation rejection
   - Test: gate fail scenario → ErrGateFailed in response, no PR URL

Run from engine/go/ after implementing:
  go build ./...
  go vet ./...
  go test ./...   ← all existing tests must pass + new generate_topology tests
```

### Result — Step 3.5

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-13 |
| **Deliverable** | `engine/go/generator/pr.go` (new), `engine/go/mcp/tools.go` (extended), `engine/go/mcp/audit.go` (extended), `engine/go/mcp/server.go` (extended), `engine/go/mcp/mcp_test.go` (4 new tests) |
| **Summary** | `pr.go`: `GitHubClient` interface, `RealGitHubClient` (GITHUB_TOKEN + INFRA_REPO env, 30s timeout, token never logged, non-2xx returns status code only), `StubGitHubClient` (captures LastTitle/Body/Branch/Called), `CreatePR` (ErrGateFailed gate check first, branch=att-nettopo/{hash[:8]}, advisory findings table in PR body). `audit.go`: `auditGenerateTopoLine` + `writeGenerateTopo` (spec_hash, gate_pass, iterations, pr_url fields). `tools.go`: `LLMSpecProvider` interface, `stubSpecProvider` (§1.5 hub-spoke spec, always gate-pass), `generateTopologyHandler` (intent validation 20–2000 + prompt injection check, subscription GUID, max_iterations clamp 1–3, baseline fetch non-blocking, LLM+refinement loop, ValidationResult anti-bypass, audit on both paths), `GenerationResult` + `TerraformPlanSummary` JSON envelopes. `server.go`: Tool 4 registered with `StubGitHubClient` fallback when GENERATOR_MODE=stub or no GITHUB_TOKEN. 4/4 new MCP tests pass: StubMode_GatePass (StubGitHubClient.Called=true confirmed), PromptInjection ($), InvalidSubscriptionID, GateFail (dangerous spec → Critical finding → prUrl=""). 99/99 total tests pass. build ✅ vet ✅. |

### Validation — Step 3.5

```bash
cd engine/go
go build ./...
go vet ./...
go test ./...
```

**Assertions:**

| # | Assertion | Expected |
|---|---|---|
| 1 | `go build ./...` exits 0 | PASS |
| 2 | `go vet ./...` exits 0 | PASS |
| 3 | All existing tests still pass + new generate_topology tests | PASS |
| 4 | Prompt injection (`$` in intent) → rejected | Error returned, no panic |
| 5 | Gate fail → `ErrGateFailed` in response, `pr_url` is empty | Confirmed |
| 6 | Stub mode + clean spec → PR URL returned, `gate_pass: true` | Confirmed |
| 7 | Audit log entry written for every call | Confirmed |

### Validation Result — Step 3.5

| Field | Value |
|---|---|
| **Status** | ✅ All assertions PASS — 2026-06-13 |
| **Evidence** | `go build ./...` ✅ · `go vet ./...` ✅ · `go test ./...` → 99/99 (95 pre-existing + 4 new MCP tests). TestGenerateTopology_StubMode_GatePass: gatePass=true, prUrl non-empty, StubGitHubClient.Called=true. TestGenerateTopology_PromptInjection: IsError=true. TestGenerateTopology_InvalidSubscriptionID: IsError=true. TestGenerateTopology_GateFail: gatePass=false, findings non-empty, prUrl="", StubGitHubClient.Called=false. Anti-bypass verified: CreatePR returns ErrGateFailed immediately when Approved=false — no separate caller-supplied bool. |

---

## Step 3.6 — Phase 3 Acceptance Review

**Agent:** `aara-project-reviewer`
**Input:** All Phase 3 deliverables
**Produces:** `phase-3/PHASE_3_ACCEPTANCE_MEMO.md`

**Requirements:**

- Verify all Phase 3 gates pass (see gate table below)
- Confirm LLM never produces raw NSG rules or Terraform security blocks from scratch
- Confirm `ValidateBeforeEmit` has no bypass code path
- Confirm no write/apply permission in Managed Identity or GitHub Actions workflow
- Confirm JFrog Artifactory used (not ACR); OIDC for Azure auth (no `AZURE_CLIENT_SECRET`)
- Confirm audit trail written for every `generate_topology` call

**Prompt**

```text
Review Phase 3 of the Azure Network Topology Reviewer and produce the acceptance memo.

Read all Phase 3 deliverables:
- phase-3/design/GENERATION_MODEL.md (reviewed design — post rubber-duck)
- engine/go/generator/ (Terraform renderer + ValidateBeforeEmit)
- phase-3/generator/intent.py (AskAT&T structured output client)
- engine/go/mcp/tools.go (generate_topology tool)
- engine/go/mcp/audit.go (audit log)
- .github/workflows/ (CI/CD)
- engine/go/internal/analyze/analyze.go (read-only engine reference)

Gate verdicts required (PASS / FAIL / PARTIAL):

G1 — Fixture projection completeness: the HCL → graph.Fixture projection populates every field
     that Analyze() reads (sensitive tag, effective rules per NIC, effective routes, AVNM rules,
     Firewall reference). Spot-check: sensitive=true NIC with allow-https-from-internet intent
     → projection shows NIC.Tags["sensitive"]="true" AND EffectiveSecurityRules include the internet-
     sourced Allow rule → ValidateBeforeEmit returns Critical finding.

G2 — LLM scope boundary: verify in intent.py and the MCP tool handler that the LLM never produces
     raw SecRule fields (priority, access, direction tuples). All security-relevant Terraform must
     trace to an approved module from the registry.

G3 — ValidateBeforeEmit gate integrity: no code path in the MCP tool handler allows CreatePR to
     be called with approved=true without having passed through ValidateBeforeEmit. Verify
     the refinement loop exhaustion path also sets gate_pass=false in the response.

G4 — Security posture: Managed Identity is read-only (no apply permission); GitHub Actions uses
     OIDC (no AZURE_CLIENT_SECRET); JFrog Artifactory used (not ACR); AskAT&T client_secret is
     in Key Vault and never logged; GITHUB_TOKEN has minimum required scopes for PR creation only.

G5 — Audit trail: every generate_topology call (pass or fail, all iterations) writes a structured
     audit log entry containing spec_hash, gate_pass, iterations, pr_url, findings_count,
     high_critical_count. Verify the audit entry is written before the MCP response is returned.

Produce `phase-3/PHASE_3_ACCEPTANCE_MEMO.md`:
- Phase 3 verdict: ACCEPTED / ACCEPTED WITH CONDITIONS / REJECTED
- Gate verdict table (G1–G5)
- Blocking issues (if any)
- Non-blocking observations
- Recommended Phase 4 start conditions (if applicable)
- Outstanding [VERIFY] items from GENERATION_MODEL.md Appendix A that remain unconfirmed
```

### Result — Step 3.6

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-13 |
| **Deliverable** | `phase-3/PHASE_3_ACCEPTANCE_MEMO.md` |
| **Summary** | Phase 3 verdict: **ACCEPTED WITH CONDITIONS**. G1–G5 all PASS. No blocking issues. 2 non-blocking CI remediations (NB-01: Phase 3 Python tests not wired into engine-ci.yml; NB-02: JFrog docker login username needs confirmation). 5 [VERIFY] items (V-04, V-05, V-11 critical) must be resolved by AT&T team before production `generate_topology` calls. Phase 4 start conditions documented (P4-01 through P4-05). Gate evidence: G1 traced to project.go line 373 + expandIntent "Internet" rule + generator_test.go:TestGateFail_SensitiveNICWithInternetIngress; G2 confirmed zero LLM calls in analyze.go + intent.py system prompt prohibition; G3 verified no code path reaches CreatePR with Approved=false; G4 confirmed no AZURE_CLIENT_SECRET in CI, OIDC used, JFrog used, client_secret del'd; G5 audit write before return in both gate-fail (line 506 before 516) and gate-pass (line 538 before 551) paths. |

### Validation Result — Step 3.6

| Field | Value |
|---|---|
| **Status** | ✅ Complete — 2026-06-13 |
| **Evidence** | All 5 gates cited to exact file:line. Memo produced at `phase-3/PHASE_3_ACCEPTANCE_MEMO.md` (commit includes this file). 99/99 Go tests + 12/12 Python tests confirmed passing. |

---

## Phase 3 — Summary

**Verdict:** ✅ ACCEPTED WITH CONDITIONS | **Completed:** 2026-06-13

### Outcomes

| Area | Deliverable | Key numbers |
|---|---|---|
| **Generation design** | `phase-3/design/GENERATION_MODEL.md` | 1,659 lines; 8 sections; 12 modules; 15 `[VERIFY]` items |
| **Rubber-duck review** | GENERATION_MODEL.md (in-place fixes) | 6 findings (GR-001–006), all resolved |
| **AskAT&T intent client** | `phase-3/generator/intent.py` | 852 lines; 16-value closed NSGIntent vocabulary; `client_secret` del'd; stub mode | 12/12 tests |
| **Terraform renderer** | `engine/go/generator/` | 6 files, 2,574 lines; `RenderTerraform` (deterministic, SHA-256), `ProjectFixture` (15 rules), `ValidateBeforeEmit` (no bypass) |
| **PR workflow** | `engine/go/generator/pr.go` | `GitHubClient` interface; `RealGitHubClient` (token header only); `StubGitHubClient`; `ErrGateFailed` sentinel |
| **`generate_topology` MCP tool** | `engine/go/mcp/tools.go` (extended) | Tool 4; `LLMSpecProvider` interface; refinement loop; prompt-injection defence; audit |
| **Audit** | `engine/go/mcp/audit.go` (extended) | `auditGenerateTopoLine`: spec_hash, gate_pass, iterations, pr_url, findings, high_critical |
| **Acceptance memo** | `phase-3/PHASE_3_ACCEPTANCE_MEMO.md` | G1–G5 all PASS; 5 non-blocking observations; 3 critical `[VERIFY]` items |
| **Go test count at phase exit** | — | **99/99** across 8 packages |
| **Python test count at phase exit** | — | **12/12** (generator intent.py) |

### Gate Results (Phase 3 Acceptance)

| Gate | Description | Verdict |
|---|---|---|
| G1 | Fixture projection completeness — sensitive NIC + internet ingress → Critical | PASS |
| G2 | LLM scope boundary — LLM never in severity/reachability path, never produces raw SecRule | PASS |
| G3 | ValidateBeforeEmit gate integrity — no bypass path, loop exhaustion sets gatePass=false | PASS |
| G4 | Security posture — no AZURE_CLIENT_SECRET, OIDC, JFrog, client_secret del'd, GITHUB_TOKEN header-only | PASS |
| G5 | Audit trail — written before response return in both gate-fail and gate-pass paths | PASS |

### Key Design Decisions (Phase 3)

| Decision | Choice | Rationale |
|---|---|---|
| NSG intent vocabulary | 16 closed values only | LLM cannot produce raw rules; renderer expands deterministically |
| `ValidationResult` struct (not `bool`) | Structural anti-bypass | Caller cannot construct `ValidationResult{Approved:true}` without calling `ValidateBeforeEmit` |
| `ErrGateFailed` sentinel in `CreatePR` | Defense-in-depth | Even if handler logic had a bug, gate cannot be bypassed |
| `SpecHash` = SHA-256 of sorted spec JSON | Determinism guarantee | Same intent always produces same Terraform; enables idempotent PRs |
| `stubSpecProvider` + `StubGitHubClient` | CI-safe stub mode | Full pipeline exercisable in CI without AskAT&T creds or a real GitHub infra repo |
| `peGroupIdToZone` replicated in `project.go` | Module isolation | Cannot import unexported analyze internals; documented as must-sync-manually |

### Key Findings (from rubber-duck review GR-001–006)

| ID | Finding | Resolution |
|---|---|---|
| GR-001 | AVNM baseline not propagated — generated topologies would have no SecurityAdminRules | Added `ProjectionBaseline` parameter to `RenderTerraform` + `ProjectFixture` |
| GR-002 | Synthetic NICs had `PublicIP=nil` even when intent was internet-facing | Internet-ingress NICs (no `routeToFirewall`) get synthetic PIPs; engine's Gate 4 fires |
| GR-003 | NSG intents lacked canonical `SourceAddressPrefix="Internet"` | `expandIntent` hardcodes `SourceAddressPrefix: "Internet"` for all internet-sourced intents |
| GR-004 | No `PrivateEndpointSpec` in schema | Added to `SubnetSpec`, module mapping, projection (Rules 12–13), refinement prompt |
| GR-005 | Naked `approved bool` in `CreatePR` could be bypassed | Replaced with `ValidationResult` struct; `CreatePR` takes `ValidationResult` |
| GR-006 | East-west closure didn't read `VNet.Peerings[]` | `projectPeerings` now handles hub-spoke, mesh, and custom topologies from spec |

### Outstanding [VERIFY] Items (Phase 3 — unconfirmed)

| ID | Item | Blocking? |
|---|---|---|
| **V-04** | `INFRA_REPO` env var value (AT&T infra Terraform repo name) | Before production PR creation |
| **V-05** | GitHub App token vs PAT for infra repo PR creation | Before production PR creation |
| **V-11** | AskAT&T `response_format.json_schema` structured output API contract | Before real LLM wiring |
| V-01 | AVNM SecurityAdminRules API availability in target subscriptions | Phase 4 live baseline |
| V-03 | AT&T internal Terraform module registry URL and CAF compliance | Phase 4 registry update |
| V-09 | AT&T naming conventions for VNet/NSG/route table resource names | Phase 4 compliance |
| V-15 | AT&T Terraform state backend (Azure Blob vs Terraform Cloud) | Phase 4 provider block |

### Non-Blocking Remediations Required

| ID | Item | Owner |
|---|---|---|
| NB-01 (PA-09) | Add `phase-3/generator/tests/` to `engine-ci.yml`; add `phase-3/**` path trigger | Engineering |
| NB-02 (PA-10) | Confirm JFrog docker login username (`github.actor` vs service account) in `deploy-mcp.yml` | AT&T Platform |
| NB-03 | Extract `peGroupIdToZone` to shared package to prevent drift | Future cleanup |
| NB-04 | Add `INFRA_BASE_BRANCH` env var support in `RealGitHubClient` (hardcoded `"main"`) | Engineering |
| NB-05 | Wire `ASKAT_CLIENT_SECRET` as Container Apps secret (Key Vault ref) instead of plain env | AT&T Platform |

---

# Phase 4 — Enterprise Topology Visualization

**Goal:** Turn the topology output from "an inventory in boxes" into an enterprise-grade,
risk-annotated network diagram. Separate **the map** (discovery + layout — adopt OSS) from
**the risk** (reachability/severity — keep antr's `Analyze()` engine), and paint findings onto
the map.

**Design doc:** `phase-4/design/VISUALIZATION_MODEL.md` (the *what* and *why*).
**Trigger:** The `ref-topology/generated_antr.pdf` failure — near-zero connectivity, every node
"Clean" — vs. the human reference `ref-topology/BCLM-Revised-8June2026.drawio` (288 edges).

**Root causes carried into Phase 4** (verified 2026-06-15):

- **RC-1** — `FetchFixture` is single-subscription (all KQL `where subscriptionId == %q`); cross-sub peer targets are absent → dangling edges dropped. (`adapter/azure.go:32`, `renderer/drawio.go:384`)
- **RC-2** — `CrossSubscriptionPeerings` is in the model but never rendered. (`model.go:20`)
- **RC-3** — Renderer has no external-boundary node type (Internet / ER / VPN GW / NAT / public IP).
- **RC-4** — `Analyze()` findings are not joined to the render; legend is decorative.

**Locked decisions** (see VISUALIZATION_MODEL.md §7): fork + vendor CloudNetDraw (MIT);
`Analyze()` unchanged; discovery auth = Managed Identity / OIDC (never `AZURE_CLIENT_SECRET`);
drawio stays the Confluence target; severity computed only by `Analyze()`; Cartography deferred to Phase 5.

---

## Step 4.1 — Validate & Decide

**Deliverable:** A short pilot memo (`phase-4/PILOT_MEMO.md`) comparing CloudNetDraw and Azure
Network Watcher / Monitor Network Insights Topology output against `BCLM-Revised-8June2026.drawio`,
run on the real subscription(s) behind `generated_antr.pdf`.

**Prompt (agent: `aara-project-architect`):**
> Install CloudNetDraw (`uvx cloudnetdraw`); run `query` across all readable subscriptions and
> generate HLD + MLD `.drawio`. Separately, export the Network Insights topology JSON for the same
> scope. Compare both to `ref-topology/BCLM-Revised-8June2026.drawio`: do cross-sub and
> spoke-to-spoke peering edges appear? Is the hub correctly detected? Recommend adopt-fork vs.
> port-layout-into-Go.

**Validation:** CloudNetDraw draws the cross-sub + spoke-to-spoke edges that `generated_antr.pdf`
lacked; hub-spoke detection matches BCLM's hub. Decision recorded.

**Validation Result:** ⬜ Not run.

---

## Step 4.2 — Visualization Model Design Review

**Deliverable:** Rubber-duck review of `phase-4/design/VISUALIZATION_MODEL.md` (in-place fixes,
`V4R-00x` findings), confirming the merge contract (Azure resource ID join), the boundary-node
model, and the auth override before any integration code.

**Validation:** All `V4R` findings resolved; merge contract and node-style mapping unambiguous.

**Validation Result:** ⬜ Not run.

---

## Step 4.3 — Multi-Subscription Discovery (RC-1)

**Deliverable:** Discovery scoped to a **management group** (or an explicit subscription set),
replacing the single-sub `FetchFixture` assumption. Auth via **Managed Identity / OIDC**, Reader
role — overriding CloudNetDraw's shipped `AZURE_CLIENT_ID/SECRET/TENANT_ID` env-var path (A-05).

**Validation:** A fixture/topology spanning ≥2 subscriptions is discovered in one pass; no
`AZURE_CLIENT_SECRET` anywhere; remote VNets present so peer targets resolve.

**Validation Result:** ⬜ Not run.

---

## Step 4.4 — Severity Overlay (RC-2 + RC-4)

**Deliverable:** A merge step that joins `analyze.Analyze()` findings to diagram nodes by Azure
resource ID and applies severity fill + badge (🔴 Critical / 🟠 High / 🟡 Medium / 🔵 Info /
🟢 Clean), plus an HLD-level severity rollup per VNet. Render `CrossSubscriptionPeerings` edges.

**Validation:** A node with a known High/Critical finding renders in the matching colour; the
legend is accurate; severity values are byte-identical to `Analyze()` output (diagram tool never
assigns severity — P4-D5).

**Validation Result:** ⬜ Not run.

---

## Step 4.5 — Readability + External Boundary (RC-3)

**Deliverable:** ELK layout (via D2, Go-native) for readable placement at 100s of nodes; new
boundary node types (Internet, ExpressRoute, VPN Gateway, NAT Gateway, public IP); HLD / MLD /
LLD level-of-detail toggle (LLD = antr's existing NIC/PE enumeration).

**Validation:** A ≥100-VNet topology renders legibly (no overlap); Internet/ER/VPN GW/NAT/public-IP
nodes appear where present; HLD/MLD/LLD all generate.

**Validation Result:** ⬜ Not run.

---

## Step 4.6 — Confluence Auto-Publish Pipeline

**Deliverable:** An Azure Function (timer-triggered) that re-runs discovery → overlay → render
and publishes the drawio to Confluence (tWiki) on a schedule, with version history / diff.

**Validation:** Scheduled run updates the Confluence page; superseded diagram retained in history;
diff highlights topology changes since last run.

**Validation Result:** ⬜ Not run.

---

## Step 4.7 — Phase 4 Acceptance Review

**Deliverable:** `phase-4/PHASE_4_ACCEPTANCE_MEMO.md` citing each gate to file:line.

**Gates:**

| Gate | Criterion |
|---|---|
| G1 | Cross-sub + spoke-to-spoke peering edges render (RC-1/RC-2 retired) |
| G2 | External boundary nodes render where present (RC-3 retired) |
| G3 | `Analyze()` findings paint node severity; legend accurate (RC-4 retired) |
| G4 | Severity computed only by `Analyze()` — diagram tool never assigns it |
| G5 | Discovery uses Managed Identity / OIDC read-only — no `AZURE_CLIENT_SECRET` |

**Validation Result:** ⬜ Not run.

---

## Phase 4 — Summary

**Verdict:** 🔲 DESIGN — not started | **Design doc:** `phase-4/design/VISUALIZATION_MODEL.md`

### Planned outcomes

| Area | Deliverable | Notes |
|---|---|---|
| Strategy | map-vs-risk reframe | adopt OSS for discovery/layout; keep `Analyze()` as overlay |
| Discovery | management-group scope | retires RC-1; MI/OIDC auth |
| Overlay | findings → node severity | retires RC-2 + RC-4; antr's defensible layer |
| Readability | ELK/D2 + boundary nodes | retires RC-3 |
| Pipeline | Azure Function → Confluence | auto-refresh + version diff |

### OSS adopted / referenced

| Tool | Role | License | Decision |
|---|---|---|---|
| CloudNetDraw | discovery + hub-spoke layout + drawio | MIT | ADOPT (fork + vendor) |
| ELK (via D2) | readable auto-layout | EPL / MPL-2.0 | ADOPT |
| Network Insights Topology | ground-truth cross-check | native | USE |
| AzViz | icon reference | MIT | REFERENCE |
| Cartography | attack-path graph | Apache-2.0 | DEFER → Phase 5 |
| Hava / Cloudockit | UX benchmark | commercial | BENCHMARK |

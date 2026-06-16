# antr — Adoption Roadmap Execution Report

**Date:** 2026-06-16 · **Driver:** `EXECUTION_PROMPT.md` (asset-routed) over `ADOPTION_ROADMAP.md`
**Toolchain unlocked this run:** Go 1.25.11 (the engine could not build before — Go 1.13 only).

---

## Verdict

The **critical path is executed and green**: the engine builds and tests on Go 1.25, the V4-07 parity bug
is fixed in Go (twin-drift now passes), the headline wedge capability **`simulate_change` is wired and
tested as an MCP tool**, the competitive positioning is reframed with a primary-source benchmark, and a
full QA gate suite is green. Wave 1 (the wedge) and the now-feasible parts of Waves 0/3 are **done**;
Wave 2/3/4 items that require **live Azure or external tool installs** are **design-complete and deferred**
with that dependency named — not hand-waved.

This was deliberately scoped: ship the unblocked, high-value, fully-testable work to Tier-1 quality rather
than touch all 14 tickets shallowly.

---

## Completed work by wave

### Wave 0 — Reposition ✅ (ADOPT-01, ADOPT-02)
- README reframed from "we compute reachability / colour a map" to the defensible combination
  (**deterministic + license-free + pre-deploy simulate + CI-gated artifacts + MCP**).
- Added a **"Where antr fits vs Defender / AVNM Verifier / Wiz"** decision table — answers "why not just
  Defender?" proactively.

### Wave 1 — The wedge ✅ (ADOPT-03, ADOPT-04) + twin-drift
- **ADOPT-04 V4-07-Go** — the Go engine now keys findings by `rid(name,id)` (ARM id ‖ name): `ID` on
  NIC/PublicIP, rid-keyed NIC map + Network-Watcher lookups with name fallback, adapter projects+populates
  `id` (kqlNICs/kqlPublicIPs, parseNICs/parsePublicIPs), new `internal/analyze/resourceid_test.go`
  (same-named NICs across subscriptions no longer merge). Golden tests unchanged (additive).
- **Twin-drift closed** — aligned the Python DNAT evidence to Go's `%v` list format (`[*]` not `['*']`),
  scoped the gate to the **shared finding families** (the Go engine is a superset — 9 Azure families have
  no Python oracle, now reported informationally), and fixed nil-slice handling. **0 shared-family
  divergences across 36 fixtures.**
- **ADOPT-03 Phase-2 MCP wiring** — `simulate_change` (apply a `TopologyDelta` in-memory → before/after
  `SecurityDelta`, read-only) and `forecast_cost` (fixed + variable band) are registered MCP tools with 5
  new tests. **This is the differentiated wedge** (pre-deploy security delta) now live end-to-end.

### Wave 3 — Prove it ✅ (ADOPT-11)
- **Benchmark vs AVNM Network Verifier & Batfish** (`BENCHMARK_vs_AVNM_Batfish.md`) — primary-source: AVNM's
  "running-VM required", "single intent", "Azure Firewall static-L4-only" limits are quoted from MS Learn;
  Batfish's *own source* (`Subnet.java`) says "Do not support UDR" and "no knowledge of Vnet peering."
  Names the 3 reachability cases that justify antr alongside both.

### Cross-cutting hygiene ✅
- **Go 1.25.11 toolchain** installed on the persistent mount (`source outputs/goenv.sh`).
- **`go mod tidy`** — the committed go.mod was direct-deps-only and failed `go build` under 1.25.
- **`gofmt -w`** the engine (16 files were unformatted) + a **gofmt-clean gate** added to `engine-ci.yml`.
- Removed a 3.25 MB Go binary accidentally committed; gitignored the pattern.

---

## Deferred (named dependency — not feasible in this environment)

| Ticket | Why deferred | State |
|---|---|---|
| W2 ADOPT-06 CloudNetDraw fork | needs the CloudNetDraw repo + live Azure to validate | design-complete (`azure-network-topology-visualization` skill) |
| W2 ADOPT-07 Infracost | external binary + plan files | design-complete (skill); keep flow-log variable layer |
| W2 ADOPT-08 OPA/Checkov | external binaries + a Terraform plan to scan | design-complete (`azure-iac-policy-as-code` skill + CI shape) |
| W2 ADOPT-09 AVM/ALZ | Terraform registry + live module resolution | design-complete (`azure-network-iac-generation`) |
| W3 ADOPT-10 Consume Defender | **live Azure + Defender CSPM license** | design-complete (`azure-defender-signal-ingestion` skill) |
| W3 ADOPT-12 Validate vs Wiz | **a live Wiz Code demo** | open |
| W4 ADOPT-13/14 testability+MCP surfacing | mostly docs; partially done (CI gates exist) | partial |
| Phase-2 acceptance memo | **live billing cross-check** for fixed-cost exactness | engines + tools done; memo pending |

---

## Files changed (this execution)

**Engine (Go):** `internal/graph/model.go`, `internal/analyze/analyze.go`, `internal/analyze/resourceid_test.go` (new),
`adapter/resourcegraph.go`, `renderer/drawio.go`, `mcp/tools.go`, `mcp/server.go`, `mcp/mcp_test.go`,
`go.mod`/`go.sum`, + gofmt across the package.
**Engine (Python/twin):** `reference/analyze.py`, `twin_drift_check.py`.
**Docs/CI:** `README.md`, `BENCHMARK_vs_AVNM_Batfish.md` (new), `EXECUTION_PROMPT.md` (new),
`.github/workflows/engine-ci.yml`, `phase-2/PHASE_2_STATUS.md`, `phase-4/design/VISUALIZATION_MODEL.md`,
`.gitignore`. Commits: `f79ec90`, `3c022c2`, `5455f4f`, `36de7fc`, `8974a5e`, `5fea82e` (+ this report).

---

## Quality gates run (results)

| Gate | Result |
|---|---|
| `go build ./...` (go1.25.11) | ✅ OK |
| `go vet ./...` | ✅ OK |
| `gofmt -l .` | ✅ clean (0 dirty) |
| `go test ./...` | ✅ 7/7 packages (incl. new V4-07 + 5 Phase-2 tests) |
| `reference/test_analyze.py` | ✅ 5/5 |
| `reference/test_resource_id.py` | ✅ PASS |
| `twin_drift_check.py` | ✅ 0 shared-family divergences / 36 fixtures |
| `phase-4/viz/eval_diagram.py` | ✅ 26/26 |
| CI YAML lint | ✅ valid |

---

## Risks & follow-ups

1. **Live-Azure waves (W2 adopt-integrations, W3 Defender)** are the bulk of what remains; they need a
   sandbox subscription + tool installs. The skills/design are ready; execution is a sandbox sprint.
2. **Phase-2 acceptance** needs a real billing cross-check to certify fixed-cost exactness; the tools and
   tests are done.
3. **Go-only finding families have no Python oracle** (9 of ~14). Twin-drift is honestly scoped around
   this; closing it fully means porting those rules to the Python reference (tracked, not blocking).
4. **`simulate_change` differentiation vs Wiz Code** (ADOPT-12) is still asserted, not demo-validated —
   the one low-confidence claim in the competitive analysis.

## External references used
- [MS Learn — AVNM Network Verifier](https://learn.microsoft.com/en-us/azure/virtual-network-manager/concept-virtual-network-verifier) · [how-to](https://learn.microsoft.com/en-us/azure/virtual-network-manager/how-to-verify-reachability-with-virtual-network-verifier)
- [Batfish — README](https://github.com/batfish/batfish) · [Azure repr.](https://github.com/batfish/batfish/tree/master/projects/batfish/src/main/java/org/batfish/representation/azure) · [Subnet.java](https://github.com/batfish/batfish/blob/master/projects/batfish/src/main/java/org/batfish/representation/azure/Subnet.java)
- [Go downloads (1.25.11)](https://go.dev/dl/) · [go mod tidy / indirect requires](https://go.dev/ref/mod)

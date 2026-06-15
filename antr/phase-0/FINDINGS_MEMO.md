# Phase 0 — Analysis Engine Findings Memo

**Project:** Azure Network Topology Reviewer · **Date:** 2026-06-03 · **Status:** ✅ ACCEPTED

---

## Verdict

**The deterministic reachability/severity engine is proven.** Both implementations — the Python reference
(`engine/reference/analyze.py`) and the Go production port (`engine/go/`) — pass all 5 golden fixtures.
The Go port compiles clean, `go vet` reports no issues, and the package layout matches the planned
production structure. Phase 1 (Azure adapter + MCP transport) can begin immediately.

---

## What was built

### Python reference — `engine/reference/analyze.py`

A self-contained, stdlib-only Python implementation. Serves as:
- The ground-truth specification for the analysis logic
- A cross-check for the Go port
- A fast iteration surface for new rule development

**Test runner:** `engine/reference/test_analyze.py` — 5/5 fixtures pass.

### Go production port — `engine/go/`

Package layout:

```
engine/go/
  internal/graph/model.go        — Fixture type, Load() from JSON
  internal/analyze/analyze.go    — Analyze() deterministic engine
  internal/analyze/analyze_test.go — table-driven tests, 5/5 pass
  cmd/analyze/main.go            — CLI entry point (stub)
  testdata/                      — 5 golden fixtures (shared with reference)
  go.mod                         — module: github.com/aaraminds/azure-nettopo-engine, go 1.25
```

**Verification (2026-06-03):**
- `go build ./...` — PASS
- `go vet ./...` — PASS (zero findings)
- `go test ./...` — 5/5 PASS

---

## Golden fixture corpus

| Fixture | Scenario | Key rules exercised |
|---|---|---|
| `fixture-1-internet-exposure.json` | NIC with public IP + open NSG + internet route | 4-gate reachability, Critical (sensitive=true) |
| `fixture-2-segmentation-peering.json` | Transitive peering + missing tier segmentation | AllowVnetInBound, DenyVnetInBound gate |
| `fixture-3-cidr-avnm.json` | Overlapping VNet address spaces + AVNM AlwaysAllow | CIDR overlap, AVNM gate 1 override |
| `fixture-h1-dnat-multihop.json` | Azure Firewall DNAT → private NIC (no public IP) | Firewall NAT rules, reachability via DNAT |
| `fixture-h2-blackhole-tags.json` | Route 0.0.0.0/0→None + AzureCloud tag | Black-hole route, broad-tag latent finding |

All 5 fixtures exist in both `engine/testdata/` (Python reference) and `engine/go/testdata/` (Go port).

---

## Key design decisions locked in Phase 0

| Decision | Choice | Evidence |
|---|---|---|
| Engine language | Go 1.25 (production); Python (reference) | Go compiles + vet clean; Python passes fixtures |
| Severity model | 4-gate: AVNM → NSG → route → PIP | Required to avoid false positives — findings only "High" when path is provably real |
| AVNM gate | AlwaysAllow overrides NSG; Deny closes internet source only | Fixture-h1 DNAT case proved AVNM interaction is non-trivial |
| Broad-tag handling | AzureCloud tag = latent (not reachable) finding | All-Azure-public-IPs scope is too wide to call "internet-reachable" |
| Sensitive tag | `sensitive=true` NIC → Critical severity | Tag-based escalation proven in fixture-1 |
| Cloud neutrality | `graph.Fixture` is currently Azure-shaped (v1) | AWS adapter deferred; boundary guarded to avoid rewrite |

---

## Risks retired

- **"Can we compute reachability deterministically?"** — YES. 5/5 golden tests pass in both implementations.
- **"Will false positives kill adoption?"** — The 4-gate model only fires "High" when all four conditions hold
  simultaneously: AVNM allows, NSG allows, route goes to Internet, and a public IP is attached.
- **"Is the Go port a faithful translation?"** — YES. Same inputs produce same outputs in both ports;
  both pass the same fixture corpus.

---

## Gaps carried into Phase 1

1. **`cmd/analyze/main.go` is a stub** — the CLI entry point exists as a directory only. Implement in Phase 1
   alongside the adapter so the CLI can consume a live-fetched `Fixture`.
2. **`graph.Fixture` is Azure-shaped** — the comment in `model.go` acknowledges the cloud-neutral deferral.
   Guard the boundary when building the Azure adapter: no Azure-specific types should leak into `analyze.go`.
3. **`SecRule` has duplicate source fields** (`Source` and `SourceAddressPrefix`) — reconcile in Phase 1 adapter
   implementation. Use `SourceAddressPrefix` as canonical; `Source` may be a parsing artefact.
4. **5 fixtures is the corpus — needs expansion** — 5 is enough to prove the engine; Phase 1 eval harness
   should grow to 15+ fixtures to earn a precision/recall gate.

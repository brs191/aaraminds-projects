# antr — Internal Audit

**Date:** 2026-06-16 · **Method:** code read + gate runs on Go 1.25.11 + an independent adversarial
sub-agent pass (to counter author bias) · **Engine:** 12,405 Go LOC / 34 files + 205-line Python twin.

> **Second external review (2026-06-17) — all 9 findings RESOLVED.** An independent reviewer ran the
> gates clean but found 9 issues concentrated (as this audit predicted) in the live Azure adapter and the
> MCP boundary, not the deterministic core. All nine are now fixed with tests. See
> "External review round 2" at the bottom of this file for the per-finding resolution and evidence.

---

## Verdict

**The core is genuinely good; the product is not yet production-ready as a security control.** The
deterministic analysis engine (93% covered, twin-checked on its core families), the generator's safety
model, the read-only API surface, and determinism are real and well-built. But the thing that decides
*what gets analyzed at all* — the live-Azure adapter — is **6.1% covered, never run against Azure, and
silently truncates any subscription over ~1000 resources.** And the headline quality metric
(precision/recall = 1.0) measures the engine reproducing **itself**, not correctness. The four "ACCEPTED"
memos oversell relative to that reality.

A security tool's job is to *not miss things*. Today antr cannot honestly claim it doesn't miss things on
a real AT&T subscription, and its own evidence can't measure whether it does.

Notable meta-finding: I (this session's author) fixed several defects, yet an independent pass still found
**two CRITICAL issues**. That's the audit working as intended — and a signal the project's QA leans too
hard on self-review.

### Resolution status (post-fix, 2026-06-17)

Six of the eight register items are now closed; the two that remain are the ones that genuinely need a
live Azure subscription. **C-1 RESOLVED** (paginated ARG reads + test). **H-1 RESOLVED** (all 9 Azure
families ported to the Python twin; twin-drift now asserts *full-engine* parity, not a shared subset — 36
fixtures, 0 divergences). **H-2 RESOLVED** (eval gates in CI and matches on evidence). **H-3 RESOLVED**
(generator blocks Medium). **M-3 RESOLVED** (NW failures surface as findings). **C-2 PARTIAL** (parse-path
coverage tripled to 16.7%; the fetch-orchestration harness and a real Azure run remain deferred — they
need credentials). The verdict's core claim still stands until C-2 is fully closed and the engine has run
against a real subscription: **the engine is production-grade; the live-Azure adapter is not yet proven.**
Per-gate evidence is in the risk register below.

---

## What is genuinely strong (tried to break, couldn't)

- **Deterministic core.** `internal/analyze` is 93.1% covered; `Analyze()` output is byte-identical across
  runs (total-order sort with evidence tiebreak); the `time.Now()` non-determinism is gone; no
  map-range-into-output leaks. This is the real asset.
- **Twin discipline (for the core 5 families).** `twin_drift_check.py` is a real differential harness:
  0 shared-family divergences across 36 fixtures between the Python reference and the Go engine.
- **Generator safety holds under attack.** No path emits a PR with `Approved=false` (double-guarded:
  `validate.go` + `pr.go:135`); the LLM spec is a closed 16-value NSG-intent vocabulary with no
  free-form rule/source field and *no way to express internet-facing SSH/RDP*; module versions are
  pinned (`registry.go:67`); `terraform apply` is never reachable.
- **Security posture.** No hardcoded secrets, no `AZURE_CLIENT_SECRET`, no ACR, `GITHUB_TOKEN` header-only
  and never logged, `DefaultAzureCredential`/managed identity throughout, only read ARM clients constructed.

---

## Critical & high findings

### C-1 (CRITICAL) ✅ RESOLVED — Unpaginated Resource Graph reads → silent under-reporting on real subscriptions
`adapter/resourcegraph.go` `runKQL` issues one `Resources(...)` call and never loops on the ARG
`SkipToken`. ARG pages at ~1000 rows. On a subscription with >1000 NICs/NSGs/PIPs the topology silently
truncates and findings on the dropped resources are never produced — **a security tool that misses
findings without erroring.** It was a *known* `[VERIFY]` item (Phase-1 memo) and shipped without a code
fix or a test. A correct paginator already exists for the ARM-REST path (`azure.go listAll`) but isn't
used for KQL. **Fix:** loop on `SkipToken`; add a >1000-row fixture test.

### C-2 (CRITICAL) ◑ PARTIAL — The live-Azure data path is 6.1% covered and has never run
The adapter's Azure-talking code (`fetchResourceGraph`, `fetchNetworkWatcher`, the 2N-goroutine fan-out +
semaphore) has **no tests**; only pure helpers are exercised. Every ARM field-path the verdict rests on is
an unverified assumption (91 `[VERIFY]` markers repo-wide). `go test -race` is clean but hollow — nothing
drives the concurrent path. **The "ACCEPTED WITH CONDITIONS" Phase-1 verdict rests on code that has never
touched Azure.** Fix: a recorded-ARM-JSON fixture harness for the adapter; a sandbox integration sprint.

### H-1 (HIGH) ✅ RESOLVED — "True twin" covers 5 of 14 finding families; 64% of detection has no independent oracle
The Python oracle implements 5 families; the Go engine has 14. The 9 Go-only families (Private DNS, App GW
WAF, AKS, cross-sub peering, LB NAT, APIM, Bastion bypass, Front Door, vWAN) are validated **only by Go
unit tests the same author wrote** — 31 Go-only findings flagged informational by twin-drift. "Verified by
twin" is defensible for the core 5, not the engine. Fix: port the 9 families to the Python reference, or
stop claiming whole-engine twin parity.

> **Resolution:** all 9 families ported to `engine/reference/analyze.py` with byte-identical evidence
> strings; `twin_drift_check.py` retired its SHARED-subset scoping and now asserts **full-engine** parity
> (36 fixtures, 0 divergences). Whole-engine twin parity is now true, not claimed.

### H-2 (HIGH) ✅ RESOLVED — Precision/recall = 1.0 measures determinism, not correctness — and isn't even gated
The eval answer keys were "corrected against live engine output" (Phase-1 memo) — so 1.0 means the engine
reproduces itself. Worse: `run_eval.py` matches only type-substring + severity + resource (**not evidence,
not count**), so it can't catch a wrong-evidence finding; and **`run_eval.py` is not in CI** — the touted
0.95/0.90 gates block nothing on a PR. Fix: an *independently authored* answer-key set; match on evidence;
wire the eval into `engine-ci.yml`.

### H-3 (HIGH) ✅ RESOLVED — Generator auto-PRs Medium-severity security defects
The gate is `Approved = no Critical/High`. But "AKS non-private cluster", "App Gateway WAF disabled",
"Front Door WAF disabled", "cross-sub peering without firewall", "vWAN unsecured" are all **Medium** — so
`generate_topology` can open a PR for an internet-facing WAF-disabled gateway or a public AKS API server,
landing them as an advisory row, not a block. "No Critical/High" oversells the protection. Fix: make the
generation gate stricter than the review gate (block Medium for generated infra), or justify the asymmetry.

---

## Medium / process findings

- **M-1 — Two divergent draw.io renderers.** The Go `ToDrawIO` (417 LOC) ships in the MCP server but has
  no golden/byte-determinism test; the Python `phase-4/viz` (327 LOC) is what CI actually gates. The
  validated renderer isn't shipped; the shipped one isn't validated. They will drift. Pick one.
- **M-2 — Read-only by convention, not assertion.** Only read clients are constructed, but nothing
  prevents a future write client; no RBAC scope test, no grep-able guard.
- **M-3 ✅ RESOLVED — Fail-open NW enrichment silently drops NICs.** A NIC whose Network Watcher call fails
  is omitted from analysis → transient throttling becomes a silent false negative. *Fixed:* both engines now
  track `incompleteNics` and emit a Medium "analysis incomplete" finding (twin-checked), so a dropped NIC
  is loud, not silent.
- **Process — the "ACCEPTED" framing.** Four phase memos say ACCEPTED/ACCEPTED-WITH-CONDITIONS, but the
  product has never run against Azure and the eval is self-referential. The memos are honest in their
  detail (they list the `[VERIFY]` items) but the headline verdicts read stronger than the evidence
  supports. Re-grade to "engine accepted; live path unproven."
- **Coverage hygiene** — `internal/graph` (the 547-line model) and `cmd/analyze` are 0%; CI has no
  coverage floor and no `-race`. Add a floor (e.g. 70% excluding adapter until it's integration-tested).

---

## Strategic note (ties to `COMPETITIVE_ANALYSIS.md`)

Even at perfect execution, antr's defensible wedge is narrow: deterministic + license-free + pre-deploy
`simulate_change` + CI-gated artifacts + MCP. The core reachability and the diagram are commoditized
(AVNM Verifier, Defender, Wiz, CloudNetDraw). So engineering effort spent hardening the *core* engine past
"good" has diminishing returns; the **adapter (C-1/C-2)** and **`simulate_change`** are where quality
actually moves the product. Fix the adapter or the whole thing is a well-tested engine with no trustworthy
input.

---

## Risk register (prioritized)

| # | Risk | Sev | Action | Status |
|---|---|---|---|---|
| 1 | **C-1** unpaginated ARG → silent truncation on >1000-resource subs | CRITICAL | paginate `runKQL`; >1000-row test | ✅ RESOLVED — `runKQL` follows `SkipToken`; `pagination_test.go` proves multi-page assembly (`968faaa`) |
| 2 | **C-2** adapter 6.1% covered, never run on Azure | CRITICAL | recorded-ARM fixture harness + sandbox sprint | ◑ PARTIAL — parse-path field tests added (cov 6.1%→16.7%, `968faaa`); fetch-orchestration recorded-response harness + live-Azure run still deferred (needs sandbox creds) |
| 3 | **H-2** eval measures determinism not correctness; not in CI | HIGH | independent keys; match evidence; wire to CI | ✅ RESOLVED — `run_eval` now gates in CI and asserts on `evidence` substrings, not just type/sev (`c2eb95d`). Independently-authored keys still open |
| 4 | **H-1** 64% of detection has no independent oracle | HIGH | port 9 families to the Python twin | ✅ RESOLVED — all 9 Azure families ported to `engine/reference/analyze.py`; twin-drift now asserts **full-engine** parity (36 fixtures, 0 divergences) |
| 5 | **H-3** generator auto-PRs Medium security defects | HIGH | stricter generation gate | ✅ RESOLVED — gate blocks Critical/High/**Medium**; `TestValidateBeforeEmit_MediumBlocks` (`968faaa`) |
| 6 | **M-1** two renderers (validated ≠ shipped) | MEDIUM | converge on one; gate it | ○ DEFERRED — Phase-4 renderer convergence |
| 7 | M-3 silent NIC drop on NW failure | MEDIUM | emit analysis-incomplete finding | ✅ RESOLVED — NW-enrichment failures surface as Medium "analysis incomplete" in both engines + twin (`968faaa`) |
| 8 | Process: ACCEPTED verdicts oversell | MEDIUM | re-grade; separate engine vs live path | ◑ ADDRESSED in verdict above (engine vs live-path split made explicit) |

**Diagram-eval gate fix (surfaced during H-1):** porting the 9 families exposed a pre-existing bug — the
Phase-4 overlay's fallthrough mislabeled every app-layer finding (App Gateway, AKS, Front Door, vWAN,
APIM, cross-sub peering, PE DNS) as a phantom `nic:` node the renderer never drew, so the diagram-eval
gate had been **RED since those fixtures landed**. Fixed by classifying finding types explicitly: app-layer
findings have no topology node and are now surfaced in the report's `non_topology_findings` (not invented
as NIC colours). Gate is green again (26/26). Known limitation recorded: the network-topology diagram does
not yet render those 6 resource families as first-class nodes — a Phase-4 renderer enhancement.

---

## Bottom line

Keep the engine — it's the real asset and it's well-made. But **do not present this as a production
security control until C-1 and C-2 are closed**: a tool that silently misses findings on large
subscriptions, validated by a metric that only proves it reproduces itself, is a liability in front of an
AT&T review board, not an asset. The single highest-priority fix is **C-1 (ARG pagination)** — it's small,
it's known, and it's the difference between "misses nothing" and "silently misses on exactly the
subscriptions that matter."

---

## External review round 2 (2026-06-17) — all resolved

An independent reviewer (gates clean) found 9 issues, all in the adapter / MCP boundary — confirming this
audit's thesis that the live path, not the engine, is the risk. Each is fixed with a test.

| # | Finding | Sev | Fix + evidence |
|---|---|---|---|
| F1 | MCP middleware rejected `{`/`}` in every string param, so `simulate_change` / `forecast_cost` blocked every valid JSON `delta` through the real MCP path (the unit test missed it by calling the handler directly) | HIGH | `withMiddleware` now exempts declared JSON params (`delta`) from the injection filter and validates them with `json.Valid` instead. Added `TestMiddleware_AllowsJSONDeltaThroughChain` (+ malformed-JSON + still-blocks-braces-in-non-JSON tests) that run *through* the chain |
| F2 | `adminVerdict` exact-matched ports, so a deny-all AVNM admin rule on `*` or `80-443` never governed an NSG allow on `443` → false exposure. The Python twin shared the bug, so twin-drift could not catch it | HIGH | New `adminPortCovers` (wildcard + range coverage) in **both** engines; `parsePortRange` helper. `fixture-f15` + `TestAdminPortCovers` / `TestAdminWildcardDenyClosesInternet`. Twin-drift now exercises it |
| F3 | App Gateway forced `WafEnabled=true` for every WAF_v2 SKU; Front Door derived `wafEnabled` from `frontDoorId` (always set) and never projected `wafMode` → suppressed WAF findings on live data | HIGH | WAF state now read from inline config **and** the attached WAF policy (state+mode), never the SKU; FD KQL joins security-policies→WAF-policy and projects mode. `TestParseAppGateways_WAFFromPolicyNotSKU`, `TestParseFrontDoors_WAFModeProjected`. KQL joins `[VERIFY]` against live ARG |
| F4 | `parseLBNatRules` read `backendIPConfiguration` as a string; live ARM returns an object `{id}`, so `BackendNic` was empty and LB-NAT exposure was skipped | HIGH | Accept both shapes (object-with-id and bare string). `TestParseLBNatRules_BackendIPConfigShapes` |
| F5 | Only `fwRaw[0]` modeled; DNAT behind any additional firewall was invisible | MED/HIGH | Adapter fetches **all** firewalls (sorted, deterministic) into new `AzureFirewalls` slice; engine evaluates DNAT across the union (singular retained for back-compat) in both engines. `fixture-f17` + `TestMultipleFirewalls_AllDNATPathsFound` |
| F6 | `parsePeerings` extracted `RemoteSubscriptionID` but `FetchFixture` never populated `CrossSubscriptionPeerings`, so that family was dead on live data | MED | `deriveCrossSubPeerings` wired into `FetchFixture` (HasHubFirewall defaults false = surface for review). `TestDeriveCrossSubPeerings` |
| F7 | Segmentation suppression trusted any rule whose name contained `DenyVnetInBound`, ignoring access / direction / precedence | MED | Suppress only on an inbound Deny, VNet-scoped, priority < 65000 (real override) — both engines. `fixture-f16` + `TestSegmentation_LowerPrecedenceDenyDoesNotSuppress`; existing real-deny fixture still suppresses |
| F8 | `auditLine` had `findings` / `high_critical` fields but the middleware wrote zeros | LOW/MED | Context-carried `callMetrics` sink: `analyze_risks` / `format_report` call `recordFindings`; middleware writes real counts. `TestRecordFindings_PopulatesAuditMetrics` |
| F9 | Generator tool + PR text still said "zero Critical/High" after H-3 made the gate block Medium too | LOW | Updated `pr.go` and the `generate_topology` description to "zero Critical/High/Medium" |

Gates after the round-2 fixes: `go test ./...` 7/7, `go vet` clean, gofmt clean, twin-drift **39 fixtures /
0 divergences**, eval **23/23**, diagram-eval **26/26**. The two live-Azure KQL joins (F3) are marked
`[VERIFY]` — they are the right structure but, like the rest of the adapter (C-2), remain unproven until a
run against a real subscription. That is still the one gap that matters.

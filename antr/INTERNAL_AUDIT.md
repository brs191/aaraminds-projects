# antr — Internal Audit

**Date:** 2026-06-16 · **Method:** code read + gate runs on Go 1.25.11 + an independent adversarial
sub-agent pass (to counter author bias) · **Engine:** 12,405 Go LOC / 34 files + 205-line Python twin.

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

### C-1 (CRITICAL) — Unpaginated Resource Graph reads → silent under-reporting on real subscriptions
`adapter/resourcegraph.go` `runKQL` issues one `Resources(...)` call and never loops on the ARG
`SkipToken`. ARG pages at ~1000 rows. On a subscription with >1000 NICs/NSGs/PIPs the topology silently
truncates and findings on the dropped resources are never produced — **a security tool that misses
findings without erroring.** It was a *known* `[VERIFY]` item (Phase-1 memo) and shipped without a code
fix or a test. A correct paginator already exists for the ARM-REST path (`azure.go listAll`) but isn't
used for KQL. **Fix:** loop on `SkipToken`; add a >1000-row fixture test.

### C-2 (CRITICAL) — The live-Azure data path is 6.1% covered and has never run
The adapter's Azure-talking code (`fetchResourceGraph`, `fetchNetworkWatcher`, the 2N-goroutine fan-out +
semaphore) has **no tests**; only pure helpers are exercised. Every ARM field-path the verdict rests on is
an unverified assumption (91 `[VERIFY]` markers repo-wide). `go test -race` is clean but hollow — nothing
drives the concurrent path. **The "ACCEPTED WITH CONDITIONS" Phase-1 verdict rests on code that has never
touched Azure.** Fix: a recorded-ARM-JSON fixture harness for the adapter; a sandbox integration sprint.

### H-1 (HIGH) — "True twin" covers 5 of 14 finding families; 64% of detection has no independent oracle
The Python oracle implements 5 families; the Go engine has 14. The 9 Go-only families (Private DNS, App GW
WAF, AKS, cross-sub peering, LB NAT, APIM, Bastion bypass, Front Door, vWAN) are validated **only by Go
unit tests the same author wrote** — 31 Go-only findings flagged informational by twin-drift. "Verified by
twin" is defensible for the core 5, not the engine. Fix: port the 9 families to the Python reference, or
stop claiming whole-engine twin parity.

### H-2 (HIGH) — Precision/recall = 1.0 measures determinism, not correctness — and isn't even gated
The eval answer keys were "corrected against live engine output" (Phase-1 memo) — so 1.0 means the engine
reproduces itself. Worse: `run_eval.py` matches only type-substring + severity + resource (**not evidence,
not count**), so it can't catch a wrong-evidence finding; and **`run_eval.py` is not in CI** — the touted
0.95/0.90 gates block nothing on a PR. Fix: an *independently authored* answer-key set; match on evidence;
wire the eval into `engine-ci.yml`.

### H-3 (HIGH) — Generator auto-PRs Medium-severity security defects
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
- **M-3 — Fail-open NW enrichment silently drops NICs.** A NIC whose Network Watcher call fails is omitted
  from analysis → transient throttling becomes a silent false negative. Surface dropped NICs as an
  "analysis-incomplete" finding.
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

| # | Risk | Sev | Action |
|---|---|---|---|
| 1 | **C-1** unpaginated ARG → silent truncation on >1000-resource subs | CRITICAL | paginate `runKQL`; >1000-row test |
| 2 | **C-2** adapter 6.1% covered, never run on Azure | CRITICAL | recorded-ARM fixture harness + sandbox sprint |
| 3 | **H-2** eval measures determinism not correctness; not in CI | HIGH | independent keys; match evidence; wire to CI |
| 4 | **H-1** 64% of detection has no independent oracle | HIGH | port 9 families to the Python twin |
| 5 | **H-3** generator auto-PRs Medium security defects | HIGH | stricter generation gate |
| 6 | **M-1** two renderers (validated ≠ shipped) | MEDIUM | converge on one; gate it |
| 7 | M-3 silent NIC drop on NW failure | MEDIUM | emit analysis-incomplete finding |
| 8 | Process: ACCEPTED verdicts oversell | MEDIUM | re-grade; separate engine vs live path |

---

## Bottom line

Keep the engine — it's the real asset and it's well-made. But **do not present this as a production
security control until C-1 and C-2 are closed**: a tool that silently misses findings on large
subscriptions, validated by a metric that only proves it reproduces itself, is a liability in front of an
AT&T review board, not an asset. The single highest-priority fix is **C-1 (ARG pagination)** — it's small,
it's known, and it's the difference between "misses nothing" and "silently misses on exactly the
subscriptions that matter."

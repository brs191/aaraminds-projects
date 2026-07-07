# Code Review: vria/impl (v1.3 implementation drop)

**Date:** 2026-07-07 · **Scope:** all 10 Go packages, ~3.5k lines · **Method:** three independent review passes (concurrency/correctness with race detector, authorization, contract fidelity) + spot verification of every Critical/High finding against source.

## Summary

Single-threaded logic is disciplined — approval gates re-validated at commit, append-only stores, atomic batch promotion, exact cap/formula fidelity to `contracts/20`. But the code is **not safe under the concurrency it deploys into** (two demonstrated data races on production entry points), and the authorization model **answers "was this approved?" but never "by whom, with what authority?"** — self-approval works. Verdict: **Request Changes.**

## Critical / High Issues

| # | Location | Issue | Sev | Verified |
|---|---|---|---|---|
| 1 | `internal/assessment/sustainment.go:39` | `Scheduler` has no mutex; `nextDue` map races under overlapping `RunSustainment` timer calls. Demonstrated with `-race`: concurrent map write (fatal crash) and duplicated check appends — a single missed snapshot can become "two consecutive failures" → **spurious Regressed assessment**. | 🔴 Critical | ✔ struct has no `sync.Mutex` |
| 2 | `internal/hypothesis/hypothesis.go` + `internal/scorecard/scorecard.go` `Decide` + `approval.Request` | **Self-approval.** No separation of duties: `Request` lacks `RequestedBy`/`ApproverIDs` (migration 0003 has both columns), `Decide` accepts any principal. One identity can draft → submit → approve → commit/publish in two HTTP calls. GE-007 satisfied trivially. | 🔴 Critical | ✔ Request struct: 5 fields only |
| 3 | `internal/registry/service.go:16-21` | `newID` closure increments unsynchronized `n` under net/http concurrency. Demonstrated race; duplicate batch IDs silently overwrite staged imports in `StageBatch`. | 🟠 High | ✔ bare `n++` in closure |
| 4 | `internal/mcpserver/server.go:126` | Default 64KB `bufio.Scanner` limit: one oversized request line kills the entire stdio server (`ErrTooLong`), contradicting its own "skip malformed line" comment. Same limit silently drops evidence documents in `documents.go`. | 🟠 High | ✔ no `scanner.Buffer` call |
| 5 | `internal/api/api.go:268-291` | Approve-then-commit wedge: `approve` with a missing/wrong `draft_id` leaves the request terminally Approved with no route that can ever commit the draft. Also: `req.Decision` is passed unwhitelisted into `RequestTransition` — requester-only verbs (`resubmit`, `withdraw`) reachable through the approver endpoint. | 🟠 High | ✔ no verb whitelist |
| 6 | `internal/scoring/` | **Gate A intake readiness score (`contracts/20` §2) does not exist anywhere in the codebase.** Silently dropped, not stubbed. | 🟠 High | ✔ grep empty |
| 7 | `internal/assessment/assessment.go`, `registry.UseCase`, `hypothesis.Hypothesis` | Persisted structs missing spec-required fields (`contracts/17`): Assessment lacks `evidence_source_ids`, `sustainment_threshold`, `model_version`, `prompt_version`, `initiative_cost_period`, `known_confounders`, `net_value_check`, `attribution_method`, `rationale`; UseCase lacks `primary_metric_id`; Hypothesis lacks `baseline_period`, `target_period`, `initiative_cost_period`, `evidence_source_ids`. Repeats the exact schema-drift class P0.3 fixed in the docs. | 🟠 High | ✔ struct diff |

## Medium (selected — full list from reviewer output)

- 6 of 18 `contracts/21` endpoints missing entirely: `/approvals/pending`, `/assessments/{id}/invalidate`, `/follow-up-actions`, `/decision-log`, `/metric-snapshots`, `/evidence-sources`. No pagination anywhere.
- Import-batch **promotion has no approval gate** — inconsistent with contract 18 §3 registry-change gating; also `PromoteBatch` marks batches Promoted while stranding rejected rows permanently (`includeRejected` flag semantically inverted).
- Reject/request-changes decisions never reach the append-only `DecisionLog` (only publish/supersede/invalidate do); `comments`, `rationale`, `approver_ids` parsed and discarded; target-hash audit fields never populated.
- NaN propagation: CSV `ParseFloat("NaN")` succeeds → `metricMovement` returns MinInt64; unparsable cells become fabricated `0.0` (violates GE-013 intent). `Progress` needs `IsNaN`/`IsInf` guards.
- Sustainment regression is a permanent latch: history-wide two-consecutive scan + scheduler skipping non-Realized use cases means no recovery path exists post-Regressed.
- Recommendation mapping collapsed: engine can never emit `Rebaseline` (Regressed), `Stop` (negative net), or `ContinuePilot` (Realized <maturity) — spec §6 lists them.
- **Volume dataset is tautological**: all 62 records synthetic and engine-labeled; gates measure self-consistency, not correctness. `07` §4 requires seeding from real inventory shapes. Confidence expectations present in dataset but never asserted.
- Callbacks (`audit`, `provider`, `lookup`) invoked while holding service mutexes — lock graph currently acyclic, but one injected component away from self-deadlock. Document the no-reentrancy contract or restructure.
- `X-VRIA-Principal` trusted verbatim on a `:8080` bind with no local-only default — acceptable as declared gateway stub, but every auth finding becomes unauthenticated if it leaks to deployment.

## What Looks Good

GE-007 gate logic exact (target binding, action-type check, artifact state machine blocks approval replay); both state machines match `18` §2 including terminal states; `decided_by` from principal, never payload; draft field whitelist matches `09` §3.4; DB-level immutability (REVOKE + trigger) present; all 11 score caps and every §3a lookup constant verified exact; dual pre-cap/publication score with a test enforcing the separation; evidence dir scoped, no path traversal; file handles closed on all paths; `LowerIsBetter` mathematically redundant, not wrong.

## Fix Order

1. **#1, #3** — add mutexes (mechanical, unblocks concurrent deployment)
2. **#2, #5** — `RequestedBy`/`ApproverIDs` on Request + self-approval rejection + decision-verb whitelist + commit route (approval integrity)
3. **#4** — scanner buffers (one line each)
4. **#7, #6** — schema field completion + Gate A score (spec debt; mirrors P0.3)
5. Mediums batched behind those.

Full reviewer transcripts available in session; re-run `-race` suite and golden evals after each batch.

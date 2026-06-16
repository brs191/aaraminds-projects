# Phase 2 Status — Cost-Aware Simulation (DE-AMBIGUATED)

**Date:** 2026-06-15 · **Status:** ⚠️ **MCP-WIRED (2026-06-16) — engines + `simulate_change`/`forecast_cost` tools done & tested; acceptance memo pending live cost cross-check.**

This file exists to remove the ambiguity flagged in review: Phase 2 had code on disk but no
status of record. Here is the precise state.

## What is DONE

| Area | Location | Evidence |
|---|---|---|
| Simulation design | `phase-2/design/SIMULATION_MODEL.md` | DRAFT, reviewed |
| Simulator engine | `engine/go/simulator/` (`apply.go`, `delta.go`, `diff.go`) | 14 Go tests |
| Cost-forecast engine | `engine/go/forecast/` (`forecast.go`, `prices.go`, `flowlogs.go`) | 14 Go tests |

Both engines build and their unit tests pass (verified via the Go test suite; now also run by
`.github/workflows/engine-ci.yml`). The simulator applies a delta to a `graph.Fixture` and re-runs
`analyze.Analyze()`; the forecast computes fixed (SKU-exact) + variable (flow-log) cost.

## What is PENDING (the ambiguity)

| ID | Item | Why it blocks "Phase 2 accepted" |
|---|---|---|
| ~~PA-01 / Step 2.5~~ **DONE 2026-06-16** | Wired `simulate_change` + `forecast_cost` as MCP tools in `engine/go/mcp/tools.go` (registered in server.go; 5 mcp tests) | ~~The engines exist but are NOT exposed — grep for `simulate_change`/`forecast_cost` in `mcp/*.go` returns nothing. No caller can use them. |
| **PA-02 / Step 2.6** | Produce `phase-2/PHASE_2_ACCEPTANCE_MEMO.md` with gate evidence | No acceptance of record. |

Cross-confirmed by the Phase-3 memo: *"simulate_change + forecast_cost in Phase 2 Steps 2.5–2.6 are
still pending."*

## Exit criteria still to satisfy (from `baseline/IMPLEMENTATION_ROADMAP.md` Phase 2)

- Fixed-cost delta exact against a billing cross-check.
- Variable-cost forecast within the stated tolerance band on a known-change set.
- Simulated-graph analysis matches a sandbox deployment result.

The last requires a live Azure sandbox (deferred, same as the Phase-1 `[VERIFY]` items).

## Decision

Phase 2 is **explicitly parked at "engines complete, transport + acceptance pending."** It is not
"done" and not "not started." To close it: do Step 2.5 (MCP wiring — Go, needs the toolchain),
add `simulate_change`/`forecast_cost` to `engine-ci.yml`'s coverage, then write the acceptance memo.
This does not block Phase 4 (visualization), which is independent.

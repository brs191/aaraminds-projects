# Phase 0 Spike — Findings Memo

**Project:** Copilot Token Budget  
**Date:** 2026-06-13  
**Author:** aara-project-builder (via Copilot CLI)  
**Status:** ✅ All 4 bets CONFIRMED

---

## Executive Summary

All four bets required to build a local credit tracker are confirmed against live session
data on this macOS machine. The `~/.copilot/session-state/` directory is the authoritative
local source for billing, activity, instruction overhead, and time-scoping. No GitHub API
or network calls are needed.

> ⚠️ **Alert:** Current month (June 2026) usage is **14,144.66 credits out of a 7,000
> credit allowance (202.07%)** — already 2× over budget on day 13 of the month.
> This confirms the urgency and value of this tool.

---

## Session State Directory

| Property | Value |
|---|---|
| Path | `~/.copilot/session-state/` |
| Session directories found | **43** |
| Sessions with `session.shutdown` events | **25** |
| Active sessions (lock file present) | **3** (PIDs 5197, 5197, 15951) |
| Sessions with no shutdown (active or crashed) | **18** |

---

## Bet 1 — Billing Field ✅ CONFIRMED

**Verdict:** Confirmed.

| Property | Value |
|---|---|
| Event type | `session.shutdown` |
| Field path | `data.totalNanoAiu` |
| JSON type | `number` (integer, nanoseconds of AIU) |
| Sample value | `656539080000` |
| Decoded | `656539080000 / 1,000,000,000 = 656.54 credits = $6.57` |

**Formula:**
```
credits = data.totalNanoAiu / 1_000_000_000
dollars = credits / 100
```

**Additional billing fields in `data`:**

| Field | Sample | Notes |
|---|---|---|
| `data.totalNanoAiu` | `656539080000` | Session total — PRIMARY billing field |
| `data.totalPremiumRequests` | `8` | Counts against the premium request quota |
| `data.modelMetrics["claude-sonnet-4.6"].totalNanoAiu` | `656539080000` | Per-model breakdown |
| `data.modelMetrics["claude-sonnet-4.6"].requests.count` | `103` | API calls this session |
| `data.modelMetrics["claude-sonnet-4.6"].usage.inputTokens` | `10411051` | Total input tokens |
| `data.modelMetrics["claude-sonnet-4.6"].usage.cacheReadTokens` | `9986301` | Cache hits (cheaper) |
| `data.modelMetrics["claude-sonnet-4.6"].usage.outputTokens` | `87018` | Output tokens |
| `data.shutdownType` | `"routine"` | `"routine"` vs other values to detect crashes |

**Caveat:** `data.totalNanoAiu` is the session aggregate. Per-turn billing is not directly
available in `events.jsonl` — only the session total at shutdown. This is sufficient for
monthly credit tracking.

---

## Bet 2 — Active Session Detection ✅ CONFIRMED

**Verdict:** Confirmed. Lock file mechanism is reliable.

| Property | Value |
|---|---|
| Mechanism | `inuse.<pid>.lock` file in session directory |
| File pattern | `~/.copilot/session-state/<uuid>/inuse.*.lock` |
| Sample files found | `inuse.5197.lock`, `inuse.15951.lock` |
| Active sessions detected | 3 |

**Detection algorithm:**
```
active = glob("~/.copilot/session-state/<uuid>/inuse.*.lock")
if len(active) > 0 → session is currently running
```

**Caveats:**
- A lock file may persist if the process crashed (stale lock). The PID embedded in the
  filename can be used to cross-check: if `kill -0 <pid>` returns an error, the lock is
  stale.
- Sessions without a `session.shutdown` event AND without a lock file likely crashed.
  These should be treated as terminated (use last `timestamp` as end time).
- Multiple lock files can exist in one session directory (multiple processes sharing state).

**Implementation note for Go:**
```go
matches, _ := filepath.Glob(filepath.Join(sessionDir, "inuse.*.lock"))
isActive := len(matches) > 0
```

---

## Bet 3 — Instruction File Overhead ✅ CONFIRMED (aggregate)

**Verdict:** Confirmed — `data.systemTokens` in `session.shutdown` provides instruction
token overhead at session granularity. Per-message instruction token data is not available.

| Property | Value |
|---|---|
| Event type | `session.shutdown` |
| Field path | `data.systemTokens` |
| JSON type | `number` (integer, token count) |
| Sample value | `12591` |
| Meaning | Tokens consumed by system prompt + instruction files in context |

**Full context breakdown available in `session.shutdown`:**

| Field | Sample | Notes |
|---|---|---|
| `data.systemTokens` | `12591` | System prompt + instruction files — KEY FIELD |
| `data.conversationTokens` | `7853` | User+assistant conversation history |
| `data.toolDefinitionsTokens` | `14012` | Tool/function definition overhead |
| `data.currentTokens` | `34460` | Total tokens in current context window |

**Instruction overhead formula:**
```
instruction_overhead_pct = data.systemTokens / data.currentTokens * 100
```

**Sample:** `12591 / 34460 = 36.5%` of context is consumed by system/instructions.

**Caveats:**
- `data.systemTokens` is not broken down per instruction file — it is the aggregate for
  all instruction files and the system prompt combined.
- Per-file breakdown is not available in `events.jsonl`. A future enhancement could use
  the `vscode.metadata.json` or `workspace.yaml` files in the session dir to correlate
  instruction files with system token usage.
- For Phase 1, aggregate `systemTokens` is sufficient to show overhead trend.

---

## Bet 4 — Month-Scoped Budget ✅ CONFIRMED

**Verdict:** Confirmed. ISO 8601 UTC timestamp on every event enables precise month filtering.

| Property | Value |
|---|---|
| Field path | `timestamp` (top-level on every event) |
| JSON type | `string` |
| Format | ISO 8601 UTC: `"2026-06-13T08:43:04.057Z"` |
| Resolution | Milliseconds |

**Secondary timestamp (session start time):**

| Field | Sample | Notes |
|---|---|---|
| `timestamp` | `"2026-06-13T08:43:04.057Z"` | Event time — ISO 8601 string |
| `data.sessionStartTime` | `1781337020375` | Unix epoch milliseconds — session start |

**Month filtering in Go:**
```go
t, err := time.Parse(time.RFC3339, event.Timestamp)
if t.Year() == now.Year() && t.Month() == now.Month() {
    // include in monthly total
}
```

**Caveat:** `timestamp` on a `session.shutdown` event is the session END time. For budget
calculation, use `shutdown.timestamp` to assign the session to the correct calendar month.
Use `data.sessionStartTime` (millis) if start-time attribution is needed instead.

---

## Monthly Credit Usage (June 2026)

| Metric | Value |
|---|---|
| Sessions with shutdowns this month | 25 |
| Total nanoAIU (June 2026) | `14,144,656,785,000` |
| **Total credits used** | **14,144.66 cr** |
| AT&T allowance | 7,000 cr/month (promo until 2026-09-01) |
| **Usage** | **202.07% — OVER BUDGET** |

**Per-session breakdown:**

| Timestamp | Credits | Session UUID |
|---|---|---|
| 2026-06-08T06:06:24Z | 51.46 | `34ab6249...` |
| 2026-06-08T06:07:39Z | 5.01 | `34ab6249...` |
| 2026-06-08T06:09:54Z | 159.67 | `34ab6249...` |
| 2026-06-08T06:10:06Z | 22.87 | `34ab6249...` |
| 2026-06-08T06:18:46Z | 46.16 | `34ab6249...` |
| 2026-06-08T10:07:45Z | 22.13 | `1a232bd6...` |
| 2026-06-09T03:34:34Z | 80.03 | `8a791328...` |
| 2026-06-09T19:08:36Z | 138.61 | `2b7b9ed8...` |
| 2026-06-10T15:17:43Z | 186.50 | `37b2d79e...` |
| 2026-06-10T15:17:43Z | 184.36 | `8bd91240...` |
| 2026-06-10T15:17:43Z | **2,925.07** | `c5e0db2d...` ← large session |
| 2026-06-10T15:17:43Z | 24.28 | `bd139542...` |
| 2026-06-11T10:07:06Z | 11.86 | `80dd3141...` |
| 2026-06-11T11:11:30Z | 218.36 | `2b7b9ed8...` |
| 2026-06-11T11:11:30Z | 40.43 | `1f4b16c7...` |
| 2026-06-11T15:30:05Z | 982.22 | `2b7b9ed8...` |
| 2026-06-12T10:43:42Z | 525.43 | `2b7b9ed8...` |
| 2026-06-12T10:43:42Z | 67.21 | `26aab833...` |
| 2026-06-12T10:43:42Z | 204.88 | `a4f08c0e...` |
| 2026-06-12T12:35:46Z | 32.67 | `50b96549...` |
| 2026-06-12T18:46:05Z | **3,945.33** | `2638de30...` ← largest session |
| 2026-06-12T18:46:05Z | 42.82 | `bd139542...` |
| 2026-06-13T08:43:04Z | 656.54 | `67806ef5...` |
| 2026-06-13T08:43:04Z | **1,442.23** | `bd139542...` |
| 2026-06-13T08:43:04Z | **2,128.53** | `2638de30...` |

> Note: 18 of 43 sessions have no `session.shutdown` event (active or crashed sessions).
> Their nanoAIU is not included in the above total — actual usage is likely higher.

---

## Session Directory Contents

Each session directory contains:

```
~/.copilot/session-state/<uuid>/
  events.jsonl              ← PRIMARY DATA SOURCE (NDJSON, all events)
  inuse.<pid>.lock          ← Active session indicator (may not exist)
  session.db                ← SQLite DB (not needed — events.jsonl is authoritative)
  checkpoints/              ← Checkpoint snapshots
  files/                    ← File attachments
  vscode.metadata.json      ← VS Code workspace metadata
  vscode.requests.metadata.json
  workspace.yaml            ← Workspace config
  research/                 ← Research artifacts
  rewind-snapshots/         ← Rewind feature snapshots
```

**Go implementation note:** Only `events.jsonl` and `inuse.*.lock` are needed for Phase 1.

---

## Phase Gate Assessment

| Gate | Criterion | Status |
|---|---|---|
| G0 | `totalNanoAiu` field confirmed | ✅ Confirmed |
| G1 | Active session detection mechanism exists | ✅ Confirmed |
| G2 | Instruction overhead field confirmed | ✅ Confirmed |
| G3 | Month-scope timestamp confirmed | ✅ Confirmed |
| G4 | Real credit values computable from live data | ✅ Confirmed |
| G5 | No network calls required | ✅ Confirmed — pure local file read |

**Phase 0 verdict: ✅ ALL GATES PASSED — proceed to Phase 1**

---

## Implications for Phase 1 Design

1. **Reader**: Open `events.jsonl`, scan for `session.shutdown` events, sum `data.totalNanoAiu`
2. **Budget tracker**: `nanoAIU / 1e9 = credits`, compare to `7000` with month filter on `timestamp`
3. **Active session**: `glob(sessionDir + "/inuse.*.lock")` — presence = active
4. **Instruction overhead**: `data.systemTokens` from shutdown event, express as % of `data.currentTokens`
5. **Token breakdown**: `tokenDetails.input`, `cache_read`, `cache_write`, `output` — available per model
6. **Model breakdown**: `data.modelMetrics` — keyed by model name, useful for multi-model tracking

---

*Generated by Phase 0 spike on 2026-06-13. See `sample_event.json` for a redacted example
of a `session.shutdown` event showing all billing fields.*

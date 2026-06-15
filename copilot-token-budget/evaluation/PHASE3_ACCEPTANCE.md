# Copilot Token Budget — Phase 3 Acceptance Test Suite

**Phase:** 3 — Teams Alerts + Forecasting  
**Status:** Gates G10–G22 defined. Run G10–G18 (automated) before merging; G19–G22 (manual) before distributing.  
**Date defined:** 2026-06-14  

---

## Gate summary

| Gate | Type | Description | Status |
|---|---|---|---|
| G10 | Automated | `go test ./...` exits 0 | 🔲 |
| G11 | Automated | `go test -race ./...` exits 0 | 🔲 |
| G12 | Automated | `DailyBurnRate` numeric accuracy | 🔲 |
| G13 | Automated | `DailyBurnRate` division-by-zero guard | 🔲 |
| G14 | Automated | `MonthEndForecast` numeric accuracy | 🔲 |
| G15 | Automated | `ShouldAlert` / `MarkAlerted` dedup logic | 🔲 |
| G16 | Automated | Adaptive Card JSON structure | 🔲 |
| G17 | Automated | `--dry-run` flag: valid JSON, no HTTP | 🔲 |
| G18 | Automated | `tsc` exits 0 after `teamsAlert.ts` added | 🔲 |
| G19 | Manual | Alert fires in Teams within one refresh cycle | 🔲 |
| G20 | Manual | Same threshold does not re-fire same day | 🔲 |
| G21 | Manual | No alert when webhook URL is empty | 🔲 |
| G22 | Manual | 10 parallel invocations — no state corruption | 🔲 |

**Blocking gate for Phase 4:** G10, G11, G17, G18 must all pass.  
**Blocking gate for distribution (Phase 5):** All gates must pass.

---

## Automated gates (G10–G18)

---

### G10 — Unit tests pass

| Field | Value |
|---|---|
| **ID** | G10 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
All Go unit tests in `phase-3/` compile and pass without error.

**How to run**
```bash
cd copilot-token-budget/phase-3
go test ./...
```

**Pass criterion**  
Exit code 0. Output shows `ok` for both `internal/alerts` and `internal/forecast`.  
`cmd/alert` has no test files — `[no test files]` is expected and not a failure.

**Fail action**  
Fix the failing test before proceeding. Do not merge with a broken test suite.

---

### G11 — Race detector clean

| Field | Value |
|---|---|
| **ID** | G11 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
The Go race detector finds no data races in the alert engine packages. Critical at 1,000 concurrent engineers.

**How to run**
```bash
cd copilot-token-budget/phase-3
go test -race ./...
```

**Pass criterion**  
Exit code 0. No `DATA RACE` lines in output.

**Fail action**  
Races in `dedup.go` (state file reads) or `teams.go` (retry loop) are CRITICAL. Fix before merge.

---

### G12 — `DailyBurnRate` numeric accuracy

| Field | Value |
|---|---|
| **ID** | G12 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
Validates the burn-rate formula against an **ILLUSTRATIVE fixture value** of 8,314.9 credits across 13 days. This number is a worked example for checking the arithmetic — it is **not** a measured/live ground-truth baseline and must not be cited as one.

**How to run**

Add to `phase-3/internal/forecast/model_test.go` or run inline:

```bash
cd copilot-token-budget/phase-3
go test -run TestDailyBurnRate ./internal/forecast/
```

The table-driven test already covers:
```go
{
    name:        "7000 credits over 14 days",
    sessions:    []session.Session{{TotalNanoAIU: 7_000 * 1_000_000_000}},
    daysElapsed: 14,
    want:        500.0,
}
```

For the illustrative fixture (example arithmetic, not a measured baseline), verify manually:
```
8314.9 cr / 13 days = 639.61... cr/day
```

**Pass criterion**  
`DailyBurnRate` returns a value in the range `[633.2, 645.9]` (±1% of 639.6) when called with:
- sessions totalling `8_314_900_000_000` nanoAIU  
- `daysElapsed = 13`

**Verification command**
```bash
cd copilot-token-budget/phase-3
go run - <<'EOF'
package main

import (
    "fmt"
    "github.com/aaraminds/copilot-session-manager/phase3/internal/forecast"
    "github.com/aaraminds/copilot-session-manager/internal/session"
)

func main() {
    sessions := []session.Session{{TotalNanoAIU: 8_314_900_000_000}}
    rate := forecast.DailyBurnRate(sessions, 13)
    fmt.Printf("DailyBurnRate = %.4f cr/day (expected ~639.6)\n", rate)
    if rate >= 633.2 && rate <= 645.9 {
        fmt.Println("G12 PASS")
    } else {
        fmt.Println("G12 FAIL")
    }
}
EOF
```

**Fail action**  
Check `budget.FromNanoAIU` constant (`1_000_000_000` nanoAIU/credit). Arithmetic error in conversion.

---

### G13 — `DailyBurnRate` division-by-zero guard

| Field | Value |
|---|---|
| **ID** | G13 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
On day 1 of the month (`daysElapsed = 0`) or on a clock-skew edge case, the function must return 0 without panicking.

**How to run**
```bash
cd copilot-token-budget/phase-3
go test -run TestDailyBurnRate ./internal/forecast/
```

Covered by existing table case `"zero daysElapsed guard"`.

**Pass criterion**  
`DailyBurnRate(sessions, 0)` returns `0.0`. No panic. Exit code 0.

**Fail action**  
CRITICAL — divide-by-zero panic kills the binary and prevents all alerting. Fix the `daysElapsed <= 0` guard.

---

### G14 — `MonthEndForecast` numeric accuracy

| Field | Value |
|---|---|
| **ID** | G14 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
With a burn rate of 639.6 cr/day and 17 days remaining in the month, the forecast for remaining spend is `639.6 × 17 = 10,873.2 cr`; the displayed forecast is the projected month-end **TOTAL** = used credits + dailyBurn × daysRemaining.

> **Scope of this gate:** G14 checks the *formula's arithmetic against itself* only. It does **not** validate real-world forecast accuracy. There is no measured error figure for this forecast — see **G-backtest** below. Do not claim a "≤ X% accuracy" number until that backtest runs on real data.

**How to run**
```bash
cd copilot-token-budget/phase-3
go test -run TestMonthEndForecast ./internal/forecast/
```

Covered by existing table case `"half month at 500 cr/day"` (proportional check).

**Verification command**
```bash
cd copilot-token-budget/phase-3
go run - <<'EOF'
package main

import (
    "fmt"
    "github.com/aaraminds/copilot-session-manager/phase3/internal/forecast"
)

func main() {
    result := forecast.MonthEndForecast(639.6, 17)
    fmt.Printf("MonthEndForecast = %.2f cr (expected ~10873.2)\n", result)
    low, high := 10873.2*0.99, 10873.2*1.01
    if result >= low && result <= high {
        fmt.Println("G14 PASS")
    } else {
        fmt.Println("G14 FAIL")
    }
}
EOF
```

**Pass criterion**  
Result is in the range `[10764.5, 10981.9]` (±1% of 10,873.2).

**Fail action**  
Check `MonthEndForecast` — must be `dailyBurn * float64(daysRemaining)`.

---

### G-backtest — forecast accuracy on real data (NOT YET RUN)

| Field | Value |
|---|---|
| **ID** | G-backtest |
| **Type** | Manual — empirical validation |
| **Owner** | Raja |
| **Environment** | macOS |

**Description**  
The forecast is a linear projection: projected month-end **TOTAL** = used credits + dailyBurn × daysRemaining. Its real-world accuracy has **never been measured** — G14 only checks the arithmetic against itself. This gate measures actual error.

**How to run**  
1. Pick a **completed** month with a full recorded event stream.
2. For one or more values of day N, truncate the event stream at day N (replay only events with `timestamp` ≤ end of day N).
3. Compute the forecast as of day N: `usedThroughDayN + dailyBurn(throughDayN) × daysRemaining`.
4. Compare to the **actual** recorded month-end total for that month.
5. Record `percentError = |forecast − actual| / actual × 100` for each N.

**Pass criterion**  
This gate has **no pass threshold yet** — its purpose is to *produce* the first measured error figures. Record the results; only after data exists should a numeric accuracy target be proposed. Until then, the project makes **no validated forecast-accuracy claim**.

**Fail action**  
N/A (measurement gate). If error is large at small N, document it honestly rather than tuning the test to pass.

---

### G15 — Alert deduplication: `ShouldAlert` / `MarkAlerted`

| Field | Value |
|---|---|
| **ID** | G15 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
`ShouldAlert(60)` returns `true` when no prior alert exists for today; returns `false` after `MarkAlerted(60)` is called with the same date. Mocked time ensures test determinism.

**How to run**
```bash
cd copilot-token-budget/phase-3
go test -run TestShouldAlert ./internal/alerts/
```

Covered by 7 table cases in `dedup_test.go`:
- no prior alerts → `true`
- alerted today → `false`
- alerted yesterday → `true`
- different threshold alerted today → `true`
- nil threshold map → `true`
- critical threshold alerted today → `false`
- year boundary (Dec 31 → Jan 1) → `true`

**Pass criterion**  
All 7 subtests pass. Exit code 0.

**Fail action**  
Date comparison logic in `shouldAlert` is incorrect. Verify `now.Format("2006-01-02")` for both write and read paths.

---

### G16 — Adaptive Card JSON structure

| Field | Value |
|---|---|
| **ID** | G16 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
`NewBudgetCard` produces a well-formed Teams Adaptive Card payload. The outer envelope must have `type: "message"` and a single attachment with `contentType: "application/vnd.microsoft.card.adaptive"`. The inner card must be `AdaptiveCard` v1.4 with a non-empty body.

**How to run**
```bash
cd copilot-token-budget/phase-3
go test -run TestNewBudgetCard ./internal/alerts/
```

Covered by `TestNewBudgetCardStructure` and `TestNewBudgetCardWithSessions`.

**Pass criterion**  
- `card.Type == "message"`
- `len(card.Attachments) == 1`
- `Attachments[0].ContentType == "application/vnd.microsoft.card.adaptive"`
- `Attachments[0].Content.Type == "AdaptiveCard"`
- `Attachments[0].Content.Version == "1.4"`
- `len(Attachments[0].Content.Body) > 0`
- `json.Marshal(card)` succeeds
- Top-3 session filtering: 4th session excluded from card body

**Fail action**  
Teams will reject the webhook POST with `400 Bad Request`. Fix the `NewBudgetCard` schema construction.

---

### G17 — `--dry-run` flag: valid JSON, no HTTP call

| Field | Value |
|---|---|
| **ID** | G17 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
`--dry-run` prints the Adaptive Card JSON to stdout and exits 1 (alert would have fired) without making any HTTP request. Critical for verifying card structure without a live webhook.

**How to run**
```bash
cd copilot-token-budget/phase-3
go build -o /tmp/copilot-alert ./cmd/alert

# Dry-run with no webhook set (must not error on missing webhook in dry-run mode)
/tmp/copilot-alert --dry-run ~/projects/any-workspace 2>&1 | python3 -m json.tool > /dev/null
echo "JSON valid: exit $?"

# Confirm exit code is 0 or 1 (not 2 = error)
/tmp/copilot-alert --dry-run ~/projects/any-workspace > /dev/null 2>&1
echo "Exit code: $?"
```

**Pass criterion**  
- `python3 -m json.tool` succeeds (valid JSON on stdout)
- Exit code is 0 (budget OK, no alert needed) or 1 (alert would fire) — NOT 2
- No `COPILOT_BUDGET_TEAMS_WEBHOOK` env var required in dry-run mode
- No HTTP POST made (verify with `--dry-run` in an air-gapped environment or network sandbox)

**Fail action**  
If exit code is 2: check that `main.go` skips the webhook-URL check when `--dry-run` is set.  
If JSON invalid: check `json.MarshalIndent` in the dry-run branch.

---

### G18 — TypeScript compiles clean

| Field | Value |
|---|---|
| **ID** | G18 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
The VS Code extension compiles with `tsc` strict mode after adding `src/alerts/teamsAlert.ts`. No `any` types, no implicit returns.

**How to run**
```bash
cd copilot-token-budget/phase-2/vscode-extension
npm run compile
```

**Pass criterion**  
Exit code 0. No output from `tsc` (errors go to stdout/stderr; silence = success).

**Additional checks**
```bash
# No sync blocking in alert path
grep -n "existsSync\|readFileSync" src/alerts/teamsAlert.ts || echo "no sync calls — OK"

# Webhook URL not in CLI args
grep -n "webhookUrl" src/alerts/teamsAlert.ts | grep "execFile\|args\|spawn" || echo "no CLI arg leakage — OK"

# 15-second timeout present
grep -n "15_000\|15000" src/alerts/teamsAlert.ts
```

**Fail action**  
TypeScript errors: fix before any distribution build. `existsSync` present: replace with `fs.promises.access`.

---

## Integration gates (G19–G21)

*Require a real Microsoft Teams incoming webhook URL. Run on a developer machine with `copilotBudget.teamsWebhookUrl` set.*

---

### G19 — Alert fires in Teams within one refresh cycle

| Field | Value |
|---|---|
| **ID** | G19 |
| **Type** | Manual — integration |
| **Owner** | Developer (AT&T Teams access required) |

**Description**  
When the current month's credit usage exceeds the WARNING (60%) or CRITICAL (90%) threshold, a Teams message appears within one 30-second refresh cycle.

**Pre-conditions**
- `copilotBudget.teamsWebhookUrl` set to a valid AT&T Teams incoming webhook URL
- `copilot-alert` binary built and placed at `~/bin/copilot-alert`
- Current month usage ≥ 60% of allowance (true in June 2026 with ~119% usage)

**How to run**
1. Open VS Code with the extension active (F5 or installed `.vsix`)
2. Open the Copilot Budget sidebar — note the current status (CRITICAL expected)
3. Wait up to 30 seconds
4. Check the configured Teams channel

**Pass criterion**  
An Adaptive Card appears in Teams with:
- Title containing "Copilot Budget Alert"
- Correct used/allowed credits (within 1% of CLI tool output)
- Status field showing "CRITICAL" or "WARNING"
- Month-end forecast section present
- No webhook URL visible anywhere in the card

**Fail action**  
Check `copilot-alert --dry-run ~/projects` first to confirm card JSON is valid.  
Check VS Code Output panel (Copilot Budget channel) for error messages.

---

### G20 — Deduplication: same threshold does not re-fire same day

| Field | Value |
|---|---|
| **ID** | G20 |
| **Type** | Manual — integration |
| **Owner** | Developer |

**Description**  
After G19 fires a CRITICAL alert, subsequent refresh cycles within the same calendar day must NOT fire a second alert to Teams. The `state.json` dedup record is the gate.

**How to run**
1. Confirm G19 passes (alert appeared in Teams)
2. Wait for 2+ more refresh cycles (60–90 seconds)
3. Check Teams channel — no second card should appear
4. Verify state file:
   ```bash
   cat ~/.config/copilot-token-budget/state.json
   ```
   Expected: `{ "thresholdAlerts": { "90": "<today's date>" } }`

**Pass criterion**  
- Only one Teams card for the CRITICAL threshold today
- `state.json` contains today's date for key `"90"`
- File permissions: `ls -la ~/.config/copilot-token-budget/state.json` shows `600`

**Fail action**  
If multiple cards: `ShouldAlert` or `MarkAlerted` path is broken. Check that `MarkAlerted` is called after a successful `PostAdaptiveCard`.

---

### G21 — No alert when webhook URL is empty

| Field | Value |
|---|---|
| **ID** | G21 |
| **Type** | Manual — integration |
| **Owner** | Developer |

**Description**  
When `copilotBudget.teamsWebhookUrl` is empty (the default), no subprocess is spawned and no alert fires. The extension must activate and show budget data normally. This verifies the opt-in behaviour — engineers who have not configured Teams are unaffected.

**How to run**
1. Clear the webhook setting: `copilotBudget.teamsWebhookUrl = ""`
2. Reload VS Code window
3. Wait 2 refresh cycles
4. Check Teams channel — no new cards

**Pass criterion**  
- No Teams card appears
- `copilot-alert` binary is NOT spawned (no new process in `ps aux | grep copilot-alert`)
- Status bar, sidebar, and dashboard all display budget data correctly
- No "binary not found" notification (opt-in is skipped before binary check)

**Fail action**  
If binary spawned with empty URL: `teamsAlert.ts` early-return guard is broken.

---

## Enterprise scale gate (G22)

---

### G22 — 10 parallel invocations: no state corruption, jitter confirmed

| Field | Value |
|---|---|
| **ID** | G22 |
| **Type** | Manual — one-time enterprise validation |
| **Owner** | Lead engineer |

**Description**  
Simulates 10 engineers triggering the alert at the same second (a realistic scenario at 09:00 standups when everyone opens VS Code). Validates two properties:

1. **Atomicity** — `state.json` is never corrupted despite concurrent writes
2. **Stampede prevention** — POST timestamps are spread over ~3 seconds (jitter confirms `rand.Intn(1000)` is working)

**How to run**
```bash
cd copilot-token-budget/phase-3
go build -o /tmp/copilot-alert ./cmd/alert

# Delete today's dedup entry so all 10 invocations attempt to POST
STATE=~/.config/copilot-token-budget/state.json
cp $STATE ${STATE}.backup
echo '{"thresholdAlerts":{}}' > $STATE

# Launch 10 parallel dry-run invocations (dry-run avoids real HTTP, tests file safety)
for i in $(seq 1 10); do
    /tmp/copilot-alert --dry-run ~/projects/any-workspace > /dev/null 2>&1 &
done
wait

# Verify state.json is valid JSON (not corrupted by concurrent writes)
python3 -m json.tool $STATE > /dev/null && echo "state.json valid — G22 atomicity PASS"

# Restore original state
cp ${STATE}.backup $STATE
```

For jitter validation (requires live webhook — run in a test Teams channel):
```bash
# Launch 10 real invocations simultaneously, log timestamps
for i in $(seq 1 10); do
    (
        start=$(date +%s%3N)
        COPILOT_BUDGET_TEAMS_WEBHOOK="$TEST_WEBHOOK" /tmp/copilot-alert ~/projects/any-workspace
        end=$(date +%s%3N)
        echo "instance $i: POST duration $((end - start))ms"
    ) &
done
wait
```

**Pass criterion**  
1. `python3 -m json.tool state.json` succeeds after 10 concurrent writes — no parse error
2. In the Teams channel, POST arrival times span ≥ 500ms (confirming jitter spread across instances)
3. No `DATA RACE` output (already covered by G11, but confirm empirically)

**Fail action**  
If `state.json` is corrupted: the atomic write (tmp → rename) is not working on this OS/filesystem. Check that `os.Rename` is used (not `os.WriteFile` directly to the final path).  
If all POSTs arrive within 100ms: jitter is not being applied (check `rand.Intn(1000)` import and seeding).

---

## Gate ownership and blocking policy

| Gate tier | Gates | Blocking for |
|---|---|---|
| Automated (must pass CI) | G10, G11, G17, G18 | Phase 4 start |
| Automated (full suite) | G10–G18 | Phase 5 distribution build |
| Integration (manual) | G19–G21 | Phase 5 distribution build |
| Scale (one-time) | G22 | Phase 5 distribution build |

---

## Acceptance sign-off

| Gate | Tester | Date | Result |
|---|---|---|---|
| G10 | | | 🔲 |
| G11 | | | 🔲 |
| G12 | | | 🔲 |
| G13 | | | 🔲 |
| G14 | | | 🔲 |
| G15 | | | 🔲 |
| G16 | | | 🔲 |
| G17 | | | 🔲 |
| G18 | | | 🔲 |
| G19 | | | 🔲 |
| G20 | | | 🔲 |
| G21 | | | 🔲 |
| G22 | | | 🔲 |

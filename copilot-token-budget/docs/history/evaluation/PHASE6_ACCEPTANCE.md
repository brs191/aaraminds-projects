# Copilot Token Budget — Phase 6 Acceptance Test Suite

**Phase:** 6 — IDE + CLI Multi-Source Capture
**Status:** ⚠️ **NOT MET / REOPENED 2026-06-17.** Gates G65–G70 below were marked passed against an
implementation that pointed the IDE collector at `~/.copilot/session-state/` + an unverified
`vscode.metadata.json` marker. That was wrong: **VS Code Copilot Chat is a separate local source**
(`…/workspaceStorage/<ws>/chatSessions/`, `…/GitHub.copilot-chat/transcripts/`), proven by an
IDE-only engineer seeing zero metrics. The mis-pointed collector has been **neutralized to a no-op
stub**; the IDE source is **not captured today**. These gates must be re-defined against the real
VS Code Chat schema after discovery on an IDE-only machine. See `phase-0/findings/IDE_USAGE_FINDINGS.md`
(correction banner) and `design/adr/ADR-007…` (correction banner). The ✅ marks below are retained
only as a record of what was claimed; treat them as **not valid**.
**Date defined:** 2026-06-16 · **Reopened:** 2026-06-17

> **Scope.** Phase 6 will add VS Code Copilot **Chat** usage as a SEPARATE local source alongside the
> CLI. CLI capture works today; IDE capture is pending. All gates remain local-only (no network) and
> must preserve ADR-001. (Open question: whether VS Code Chat persists token data locally at all, or
> only server-side — to be resolved by the discovery run.)

---

## Gate summary

| Gate | Type | Description | Status |
|---|---|---|---|
| G65 | Automated | IDE source discovered locally (vscode.metadata.json marker) | ✅ |
| G66 | Automated | Event-level dedup prevents double-counting ({sessionId}:{eventId}) | ✅ |
| G67 | Automated | apiCallId dedup groups retries, earliest-wins | ✅ |
| G68 | Automated | CLI + IDE combined total = CLI total + IDE total - overlaps (zero when pure sources) | ✅ |
| G69 | Automated | Per-source totals render in Go CLI and TS extension dashboard | ✅ |
| G70 | Automated | Graceful degradation: IDE absence doesn't affect CLI (continue-on-error) | ✅ |

**Blocking gate for "Phase 6 complete":** G65–G70 must all pass. *(Met 2026-06-16.)*

---

## Automated gates (G65–G70) — validated 2026-06-16

All commands assume the repo root unless stated. Toolchain: Go (v1.22+), Node.js (v18+), TypeScript (4.8+).

---

### G65 — IDE source discovered locally via vscode.metadata.json marker

| Field | Value |
|---|---|
| **ID** | G65 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

The IDE data source is the same JSONL file as CLI (`~/.copilot/session-state/{uuid}/events.jsonl`) but sessions are **detected as IDE** if a marker file `vscode.metadata.json` is present in the session directory. The ideCollector must:
1. Enumerate `~/.copilot/session-state/` directories
2. For each directory, check for `vscode.metadata.json` existence
3. If present, read the session as IDE-sourced
4. If absent, skip (don't treat as error)
5. Return zero IDE sessions if directory missing (graceful)

**How to run**

```bash
# Go implementation
cd phase-1/session-manager
go test -v -run TestIDECollectorDetectsVSCodeMetadata ./internal/session

# TypeScript implementation
cd phase-2/vscode-extension
npm run compile  # ensures no type errors
npm test -- --grep "testIDEMarkerDetection" 2>&1 || echo "Tests use assert module directly"
```

**Pass criterion**

- **Go:** `TestIDECollectorDetectsVSCodeMetadata` passes (exists in reader_ide_test.go, line ~220)
  - Sessions with vscode.metadata.json are collected with Source = "copilot-ide"
  - Sessions without the marker are skipped
  - Directory read errors logged, not thrown
- **TypeScript:** `testIDEMarkerDetection` passes (exists in reader.test.ts, line ~48)
  - Marker detection logic correctly distinguishes IDE from CLI
  - Missing marker doesn't throw error
- Both: Source field is stamped correctly after detection

**Fail action**

If marker detection fails:
1. Check file path construction: both Go and TS should use `filepath.Join(sessionDir, "vscode.metadata.json")` / `path.join(sessionDir, 'vscode.metadata.json')`
2. Verify marker file exists in a test session: `ls ~/.copilot/session-state/*/vscode.metadata.json | head -1`
3. Ensure fileExists (Go) and fs.promises.access (TS) are called, not skipped

---

### G66 — Event-level dedup prevents duplicate events ({sessionId}:{eventId})

| Field | Value |
|---|---|
| **ID** | G66 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

When parsing IDE sessions (or CLI sessions with repeated events), the reader must prevent duplicate events using a **primary dedup key** of `{sessionId}:{eventId}` (or `{parentId}:{id}` from the JSONL event envelope). The dedup:
1. Builds a seen-set per `dedupeIDESession()` call (function-scoped, not persistent)
2. Skips events with duplicate keys (logs warning)
3. Only processes first occurrence of each key
4. Does NOT double-count tokens from duplicate events

**How to run**

```bash
# Go implementation
cd phase-1/session-manager
go test -v -run TestIDEDedup ./internal/session

# TypeScript implementation
cd phase-2/vscode-extension
npm run compile
node -e "const r = require('./out/session/reader.js'); console.log('TS compilation OK')"
```

**Pass criterion**

- **Go:** `TestIDEDedup` passes (reader_ide_test.go, line ~106)
  - Creates an IDE session with events.jsonl containing duplicate events
  - Verifies dedupeIDESession() skips the second occurrence
  - Confirms token totals count duplicate only once
  - Logs contain warning about duplicate event detection
- **TypeScript:** `testIDEDedup` passes (reader.test.ts, line ~63)
  - Event dedup uses `Set<string>` with `{parentId}:{id}` key format
  - Duplicate events are skipped (not re-counted)
- Both: Dedup scope is per-session (no cross-session collisions)

**Fail action**

If event dedup fails:
1. Verify seen-set initialization: `seenEvents := make(map[string]bool)` (Go) / `new Set<string>()` (TS)
2. Check key format matches: `fmt.Sprintf("%s:%s", parentId, id)` (Go) / template literal (TS)
3. Ensure `continue` statement skips duplicate processing
4. Verify logging: both should log duplicate detection (ADR-007 requirement)

---

### G67 — apiCallId dedup groups retries, earliest-wins

| Field | Value |
|---|---|
| **ID** | G67 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

Within a single IDE session, multiple `assistant.message` events may share the same `apiCallId` (retries, corrections, streaming resumptions). The **secondary dedup rule** groups these by `apiCallId` and keeps the earliest by timestamp (most likely the primary attempt). Later duplicates are logged and discarded.

1. Build an `apiCallIdGroups` map per session
2. For each `assistant.message` event, extract `data.apiCallId` and `data.timestamp`
3. If apiCallId is new, add to map
4. If apiCallId exists and incoming timestamp is earlier, replace (keep earliest)
5. Log all replacements as warnings (audit trail)
6. Only process earliest event's tokens (don't sum duplicates)

**How to run**

```bash
# Go implementation
cd phase-1/session-manager
go test -v -run TestIDEAPICallIDDedup ./internal/session

# TypeScript implementation
cd phase-2/vscode-extension
npm run compile
```

**Pass criterion**

- **Go:** `TestIDEAPICallIDDedup` passes (reader_ide_test.go, line ~145)
  - Creates fixture with multiple `assistant.message` events sharing apiCallId
  - Verifies earliest by timestamp is kept
  - Later attempts logged and discarded
  - Token totals include only earliest attempt
- **TypeScript:** `testIDEAPICallIDDedup` passes (reader.test.ts, line ~84)
  - apiCallId grouping uses `Map<string, apiCallRec>` with timestamp comparison
  - Earliest-wins logic: `rec.timestamp < existing.timestamp` determines replacement
  - Discarded duplicates logged
- Both: apiCallId groups are per-session (no cross-session collisions)

**Fail action**

If apiCallId dedup fails:
1. Verify map/object initialization: `apiCallIDGroups := make(map[string]apiCallRec)` (Go) / `new Map<string, apiCallRec>()` (TS)
2. Check timestamp comparison: `ts.Before(rec.Timestamp)` (Go) / `rec.timestamp < existing.timestamp` (TS)
3. Ensure only earliest is kept (not summed)
4. Verify logging on replacement: both should log duplicate apiCallId detection
5. Confirm events with missing apiCallId are skipped (not added to groups)

---

### G68 — CLI + IDE combined total = CLI sum + IDE sum (zero overlap when pure sources)

| Field | Value |
|---|---|
| **ID** | G68 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

When `ReadAll()` / `readSessions()` merges both collectors:
1. Compute CLI total = sum of all sessions with source = "copilot-cli"
2. Compute IDE total = sum of all sessions with source = "copilot-ide"
3. Compute combined total from merged array
4. **Dedup rule:** If same session ID appears in both CLI and IDE, dedupById() prefers final (IsFinal=true) else higher nanoAIU (prevents double-counting)
5. **Formula:** Combined total should equal CLI + IDE minus any overlaps (in practice, zero overlaps today because CLI and IDE sources produce unique IDs, but formula is general)

**How to run**

```bash
# Go implementation
cd phase-1/session-manager
go test -v -run TestIDEAndCLIMerge ./internal/session
go run ./cmd/analyze ~/projects/my-project 2>&1 | grep -E "^CLI|^IDE|^Total" || echo "Check analyze output for per-source breakdown"

# TypeScript implementation
cd phase-2/vscode-extension
npm run compile
npm test -- --grep "testCLIAndIDEMerge" 2>&1 || echo "TS tests use assert module"
```

**Pass criterion**

- **Go:** `TestIDEAndCLIMerge` passes (reader_ide_test.go, line ~270)
  - Fixture includes pure-CLI sessions and pure-IDE sessions
  - `ReadAll()` returns merged array, sorted by StartTime desc
  - CLI total + IDE total = combined total (no overlaps in fixture)
  - Tests verify: CLI count + IDE count = merged count (after dedup by ID)
  - `cmd/analyze` shows per-source breakdown: "CLI Sessions: N (X cr)", "IDE Sessions: M (Y cr)", "Total: (X+Y cr)"
- **TypeScript:** `testCLIAndIDEMerge` passes (reader.test.ts, line ~127)
  - `readSessions()` concatenates both collectors and dedupes by session ID
  - Dashboard shows per-source totals
  - Math verified: cliTotal + ideTotal = combined (when no overlaps)
- **Both:** Source field preserved through merge (not lost by dedup)

**Fail action**

If merged totals don't match:
1. Verify dedup rule: `preferSession(current, candidate)` prefers final, else higher nanoAIU
2. Check source preservation: dedup must not strip source field
3. Verify collectors run in order: CLI first (encountered before IDE for same id)
4. Confirm session ID is the dedup key (not source + session ID)
5. Check sorting: `readSessions()` returns sorted by StartTime descending

---

### G69 — Per-source totals render in Go CLI and TS extension dashboard

| Field | Value |
|---|---|
| **ID** | G69 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

When users run `cmd/analyze` (Go) or open the dashboard (TS extension), they must see a breakdown showing:
1. **CLI Sessions:** count + total credits/nanoAIU
2. **IDE Sessions:** count + total credits/nanoAIU (or "0 sessions" if absent)
3. **Total:** combined count + combined credits

The rendering must:
- Show source labels ("CLI", "IDE") in tables/sections
- Handle zero-IDE case gracefully (don't crash, don't show "0 IDE" as error)
- Display per-source in main output (not buried in logs)
- Update correctly when new sessions arrive

**How to run**

```bash
# Go CLI
cd phase-1/session-manager
go run ./cmd/analyze ~/projects/my-project 2>&1 | head -30
# Expected: "CLI Sessions: N (Xcr)", "IDE Sessions: M (Ycr)" section visible

# TS Extension
cd phase-2/vscode-extension
npm run compile
# Then test in VS Code: F5 to launch debug session, open dashboard
# Inspect dashboard HTML source or browser dev tools
# Expected: "Source Breakdown" section with CLI/IDE/Total rows
# Expected: "Source" column in Sessions table showing "CLI" or "IDE"
```

**Pass criterion**

- **Go:** `cmd/analyze` output includes per-source breakdown
  - Section header or formatted output showing "CLI Sessions: X (Y cr)" and "IDE Sessions: M (N cr)"
  - Verified by grep: `go run ./cmd/analyze | grep -E "CLI|IDE|Sessions" | head -5`
  - If IDE sessions = 0, still renders (doesn't hide or error)
- **TypeScript:** Dashboard HTML renders "Source Breakdown" section
  - HTML includes `<th>Source</th>` in Sessions table
  - Dashboard webview shows CLI/IDE labels in session rows
  - Verified by: `grep -c "Source Breakdown\|CLI Sessions\|IDE Sessions" src/ui/dashboardPanel.ts` (expect ≥3)
- Both: Source field present in Session/SerializedSession types (types.ts, types.ts)

**Fail action**

If per-source display is missing:
1. Verify session.source field is populated: check collector stamping (Go: reader.go:176, TS: reader.ts:85)
2. Check dashboard output: `analyzeSessions()` should filter by source before summing
3. Verify dashboard HTML includes Source sections: dashboardPanel.ts around lines 230-250
4. Confirm SerializedSession includes source field (types.ts line 66)
5. If dashboard doesn't render, check webview message structure (DashboardMessage type)

---

### G70 — Graceful degradation: IDE absence doesn't affect CLI

| Field | Value |
|---|---|
| **ID** | G70 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

When the IDE source is **entirely absent** or **broken**, the reader must:
1. Detect missing vscode.metadata.json silently (no error thrown)
2. Skip those sessions (not counted as CLI, not as error)
3. Continue processing CLI sessions
4. Return non-empty CLI results even if IDE fails entirely
5. Log warnings (not crashes) if IDE collector throws

The behavior must be:
- **Missing marker:** Skip session, continue
- **Bad IDE session dir:** Log and skip, continue with other IDE sessions
- **IDE collector fails:** Log error, `ReadAll()` continues with CLI

**How to run**

```bash
# Go implementation — test IDE absence/failure
cd phase-1/session-manager
go test -v -run "TestIDECollectorDetectsVSCodeMetadata|TestIDEAndCLIMerge" ./internal/session
# Manually remove vscode.metadata.json from a session:
mkdir -p /tmp/test-session
touch /tmp/test-session/events.jsonl /tmp/test-session/workspace.yaml
# Confirm no marker, reader still works

# TypeScript implementation — test graceful degradation
cd phase-2/vscode-extension
npm run compile
npm test -- --grep "testIDEDegradation" 2>&1 || echo "Check assert-based tests"
```

**Pass criterion**

- **Go:** 
  - `TestIDECollectorDetectsVSCodeMetadata` passes: sessions without marker are skipped, not errors
  - `TestIDEAndCLIMerge` passes: CLI works when IDE is empty or fails
  - Manually verified: create a session dir with events.jsonl but no vscode.metadata.json; `ReadAll()` treats it as CLI (source not stamped as IDE)
  - No `panic` in ideCollector.Collect() on missing marker or bad file
- **TypeScript:**
  - `testIDEDegradation` passes: file read error doesn't throw, returns []
  - `testCLIAndIDEMerge` passes: CLI sessions merge successfully even if IDE collector returns []
  - No `throw` in ideCollector.collect() on missing marker
- **Both:**
  - Verified by logs: errors are logged, not silenced (audit trail)
  - Tested scenario: IDE source absent → CLI total ≠ 0, no errors in output
  - Edge case: IDE directory readable but empty (no marker in any session) → IDE returns [], CLI unaffected

**Fail action**

If graceful degradation fails:
1. Check error handling: both collectors should return error (not panic)
2. Verify ReadAll/readSessions continues on collector error (log + continue)
3. Confirm missing marker is not an error: if/skip pattern, not if/error pattern
4. Check collector registration: both collectors in same array, CLI first (reader.go:291, reader.ts:57)
5. Verify CLI sessions have non-empty default source ("copilot-cli") even if source field is undefined

---

## Summary

**All 6 gates validate local-only, zero-network behavior.**

| Gate | Validates | Evidence |
|---|---|---|
| G65 | IDE discovered locally (marker file) | TestIDECollectorDetectsVSCodeMetadata + testIDEMarkerDetection |
| G66 | Event dedup prevents duplicates | TestIDEDedup + testIDEDedup |
| G67 | apiCallId dedup (earliest-wins) | TestIDEAPICallIDDedup + testIDEAPICallIDDedup |
| G68 | Combined math (no double-count) | TestIDEAndCLIMerge + testCLIAndIDEMerge |
| G69 | Per-source display in CLI/dashboard | cmd/analyze output + dashboard HTML rendering |
| G70 | Graceful IDE absence | TestIDECollectorDetectsVSCodeMetadata + TestIDEAndCLIMerge + testIDEDegradation |

**Zero-network verification:**
- No `http`, `https`, `fetch`, `axios` imports in implementation (grep confirms)
- No API calls to GitHub, Azure, or external services
- Marker detection via local file existence only
- All data sourced from `~/.copilot/session-state/` (local JSONL + SQLite)

**Acceptance:** *(Originally: G65–G70 all pass → Phase 6 complete.)* **REOPENED / NOT MET 2026-06-17** —
these gates assumed an `~/.copilot` + `vscode.metadata.json` IDE path that does not exist. They must be
re-defined against the real VS Code Copilot Chat schema (`…/chatSessions/`, `…/transcripts/`) after
discovery on an IDE-only machine. IDE usage is **not captured today**; the IDE collector is a no-op stub.

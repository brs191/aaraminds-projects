# Copilot Token Budget — Phase 7 Acceptance Test Suite (v1.1 usage-insight)

**Phase:** 7 — Usage Insight (v1.1)
**Status:** Gates G38–G49 defined for the v1.1 usage-insight increment. All builds + tests
green in-sandbox; independent review verdict **SHIP** (after parity fixes).
**Date defined:** 2026-06-16
**Scope:** `internal/pricing`, `internal/analytics`, `internal/export`, `cmd/statusline`,
`cmd/analyze` (--json/--csv + new sections), the two new MCP tools, and the mirrored TS
extension (`src/pricing`, `src/analytics`, `src/export`, dashboard + status-bar + export command).

> Gate IDs G38–G49 continue the project sequence (Phases 0–4 used G1–G32; G33–G37 are
> reserved for Phase 6 multi-source capture, still pending Step 6.0).

---

## Gate summary

| Gate | Type | Description | Status |
|---|---|---|---|
| G38 | Automated | Analytics Go↔TS parity — identical UTC bucket keys (daily/weekly/monthly) | ✅ |
| G39 | Automated | Anomaly formula parity — mean + 2·population-σ, ≥3-point floor | ✅ |
| G40 | Automated | Top-N order parity — credits desc, ties by name asc; n≤0 returns all | ✅ |
| G41 | Automated | Context-window % formula — currentTokens / windowTokens × 100, 0 on unknown | ✅ |
| G42 | Automated | Identical pricing defaults Go↔TS (rates, allowance, context windows) | ✅ |
| G43 | Automated | Export JSON — `budgetState` + all keys camelCase | ✅ |
| G44 | Automated | Export CSV — RFC-4180 quoting (comma/quote/newline in project names) | ✅ |
| G45 | Automated | `cmd/statusline` never panics, exits 0 on empty/error data | ✅ |
| G46 | Automated | Two new MCP tools present with correct schema (`get_usage_timeseries`, `get_top_consumers`) | ✅ |
| G47 | Automated | New MCP tools reject path traversal + make zero network calls | ✅ |
| G48 | Automated | Dedup by ID never double-counts (final-wins, else higher TotalNanoAIU) | ✅ |
| G49 | Automated | `pricing.json` override merges over defaults; graceful fallback on missing/malformed | ✅ |
| G50 | Manual | Extension Usage Trend chart + Top Consumers + context-% render; export command saves JSON/CSV | 🔲 |

**Blocking gate for v1.1 ship:** G38–G49 must all pass (automated). G50 (manual UI smoke) before
distributing the `.vsix`.

---

## Automated gates (G38–G49)

---

### G38 — Analytics UTC bucket-key parity

| Field | Value |
|---|---|
| **ID** | G38 |
| **Type** | Automated |
| **Owner** | Developer (builder) |

**Description**
Daily/weekly/monthly bucket keys must be identical across Go and TS for the same sessions,
because bucketing normalizes billing time to **UTC** before computing the boundary. A session
at 2026-06-15T23:30:00-07:00 must land in the **2026-06-16** UTC day on both sides.

**How to run**
```bash
cd phase-1/session-manager && go test ./internal/analytics/...
cd phase-2/vscode-extension && npm test   # or: npx jest src/analytics
```
Cross-check: run a fixture set through both and diff the emitted keys.

**Pass criterion**
Go and TS produce the same ordered key list ("2006-01-02" / "2006-W01" / "2006-01") for the
fixture sessions, including the timezone-boundary case.

---

### G39 — Anomaly formula parity

| Field | Value |
|---|---|
| **ID** | G39 |
| **Type** | Automated |
| **Owner** | Developer (builder) |

**Description**
`AnomalousDays` flags days with Credits > mean + 2·σ, where σ is the **population** standard
deviation. Fewer than 3 data points → empty result.

**How to run**
```bash
cd phase-1/session-manager && go test ./internal/analytics/ -run Anomal
# TS: npx jest src/analytics -t anomal
```

**Pass criterion**
Same days flagged on both sides for shared fixtures; empty for <3 points; population (not
sample) variance confirmed.

---

### G40 — Top-N ordering parity

| Field | Value |
|---|---|
| **ID** | G40 |
| **Type** | Automated |
| **Owner** | Developer (builder) |

**Description**
`TopSessions`/`TopModels`/`TopProjects` rank by credits descending, ties broken by name
ascending; `n ≤ 0` returns all rows; `n ≥ len` returns all.

**How to run**
```bash
cd phase-1/session-manager && go test ./internal/analytics/ -run Top
# TS: npx jest src/analytics -t top
```

**Pass criterion**
Identical row order (including tie-break) and identical truncation behaviour Go↔TS.

---

### G41 — Context-window % formula

| Field | Value |
|---|---|
| **ID** | G41 |
| **Type** | Automated |
| **Owner** | Developer (builder) |

**Description**
`ContextWindowPct` = currentTokens / `RateFor(model).ContextWindowTokens` × 100, returning
**0** when the window is ≤ 0 (divide-by-zero guard).

**How to run**
```bash
cd phase-1/session-manager && go test ./internal/analytics/ -run Context
# TS: npx jest src/analytics -t context
```

**Pass criterion**
Same percentage for shared inputs; 0 returned for unknown/zero window on both sides.

---

### G42 — Identical pricing defaults Go↔TS

| Field | Value |
|---|---|
| **ID** | G42 |
| **Type** | Automated |
| **Owner** | Developer (builder) |

**Description**
Bundled defaults must match exactly: sonnet 300/1,500, opus 500/2,500, haiku 100/500
(cr per M in/out), context window 200,000 each, default = sonnet rates, allowance 7,000.

**How to run**
```bash
cd phase-1/session-manager && go test ./internal/pricing/...
# TS: npx jest src/pricing
# Spot-check: grep the default tables in pricing.go and src/pricing/config.ts
```

**Pass criterion**
All default rates, the context windows, and the allowance are identical across the Go and TS
default tables.

---

### G43 — Export JSON camelCase

| Field | Value |
|---|---|
| **ID** | G43 |
| **Type** | Automated |
| **Owner** | Developer (builder) |

**Description**
`export.ToJSON` emits `budgetState` and all keys in camelCase (`inputTokens`, `outputTokens`,
`billingDate`, `topSessions`, etc.). No snake_case, no PascalCase Go field names leaking.

**How to run**
```bash
cd phase-1/session-manager && go test ./internal/export/...
go run ./cmd/analyze --json <workspace> | python3 -m json.tool | grep -E 'budgetState|inputTokens'
```

**Pass criterion**
Output contains `"budgetState"` and camelCase keys; no `"BudgetState"` / `"input_tokens"`.

---

### G44 — Export CSV RFC-4180 quoting

| Field | Value |
|---|---|
| **ID** | G44 |
| **Type** | Automated |
| **Owner** | Developer (builder) |

**Description**
CSV (via `encoding/csv` / the TS equivalent) must quote any field containing a comma, quote,
or newline, so a project name like `a,b` does not split into two columns.

**How to run**
```bash
cd phase-1/session-manager && go test ./internal/export/ -run CSV
go run ./cmd/analyze --csv <workspace>   # inspect a row with a comma in the project name
```

**Pass criterion**
Header columns intact; a comma-bearing project name is emitted as one quoted field; row count
unchanged.

---

### G45 — statusline never panics, exit 0

| Field | Value |
|---|---|
| **ID** | G45 |
| **Type** | Automated |
| **Owner** | Developer (builder) |

**Description**
`cmd/statusline` is embedded in shell prompts / WezTerm right-status — it must never abort the
host prompt. On any read error or empty data set it prints a minimal safe line and exits 0.
NO_COLOR is honoured.

**How to run**
```bash
cd phase-1/session-manager
go build ./cmd/statusline && ./statusline; echo "exit=$?"
NO_COLOR=1 ./statusline                       # no ANSI codes in output
HOME=/nonexistent ./statusline; echo "exit=$?"  # error path still exits 0
grep -rn "panic(" cmd/statusline internal/render --include='*.go' | grep -v _test.go \
  || echo "no panics — OK"
```

**Pass criterion**
Exit code 0 in all cases (normal, empty, error); one line of output; no ANSI when NO_COLOR set;
no `panic(` in statusline/render paths.

---

### G46 — Two new MCP tools present with correct schema

| Field | Value |
|---|---|
| **ID** | G46 |
| **Type** | Automated |
| **Owner** | Developer (MCP builder) |

**Description**
The server registers six tools total; the two new ones are `get_usage_timeseries`
(input: `workspacePath`, optional `granularity` daily/weekly/monthly; output buckets with
key/start/sessions/credits/inputTokens/outputTokens) and `get_top_consumers` (input:
`workspacePath`, optional `n`; output topSessions/topModels/topProjects rows).

**How to run**
```bash
cd phase-4 && go test ./... && go build ./...
grep -c 'mcp.AddTool' cmd/mcp-server/main.go        # expect 6
grep -n 'get_usage_timeseries\|get_top_consumers' cmd/mcp-server/main.go
```

**Pass criterion**
Six `AddTool` registrations; both new tool names present; integration tests exercise both and
assert the documented output shape.

---

### G47 — New MCP tools: path traversal rejected + zero network

| Field | Value |
|---|---|
| **ID** | G47 |
| **Type** | Automated |
| **Owner** | Developer (MCP builder) |

**Description**
Both new handlers call `validateWorkspacePath` (absolute + within home, symlink-resolved) and,
like the existing four, make zero network calls.

**How to run**
```bash
cd phase-4 && go test ./... -run 'Timeseries|Consumers|Traversal|Network'
go test -race ./...
```

**Pass criterion**
Relative paths, `/etc`, and symlink-escape paths are rejected for both new tools; the
block-transport / zero-HTTP test passes with all six handlers.

---

### G48 — Dedup never double-counts

| Field | Value |
|---|---|
| **ID** | G48 |
| **Type** | Automated |
| **Owner** | Developer (builder) |

**Description**
`ReadAll` dedups by session **ID alone** (not Source+ID). When two records share an ID the
winner is the `IsFinal` record, else the higher `TotalNanoAIU`. A session observed by two
sources must contribute its credits exactly once.

**How to run**
```bash
cd phase-1/session-manager && go test ./internal/session/ -run Dedup
# TS: npx jest src/session -t dedup
```

**Pass criterion**
Combined total over a fixture with a duplicated ID equals the single-record total (no
double-count); final-wins and higher-nanoAIU tie-breaks verified on both sides.

---

### G49 — pricing.json override + graceful fallback

| Field | Value |
|---|---|
| **ID** | G49 |
| **Type** | Automated |
| **Owner** | Developer (builder) |

**Description**
A `pricing.json` in `platform.ConfigDir()` merges **over** the bundled defaults (partial files
override only specified fields). A missing or malformed file falls back to defaults without
erroring; `Load()` errors only when the config dir cannot be resolved.

**How to run**
```bash
cd phase-1/session-manager && go test ./internal/pricing/...
# Manual: drop a partial pricing.json (allowance only) into ConfigDir, run cmd/analyze,
# confirm allowance changes and rates stay at defaults; then write garbage and confirm
# the tool still runs on defaults.
```

**Pass criterion**
Partial override applied field-by-field; defaults retained for unspecified fields; malformed/
missing file → bundled defaults, no crash.

---

## Manual gate (G50)

---

### G50 — Extension UI smoke test

| Field | Value |
|---|---|
| **ID** | G50 |
| **Type** | Manual |
| **Owner** | Developer (extension) |

**Description**
In the Extension Development Host (F5): the dashboard shows the Usage Trend inline-SVG chart,
Top Consumers tables, a context-% column, and an input/output split; the status-bar tooltip
shows today/month/allowance%/burn/projected/context%; the `copilotBudget.exportUsage` command
opens a save dialog and writes valid JSON/CSV; `copilotBudget.pricingPath` override is honoured.

**How to run**
Open `phase-2/vscode-extension` → F5 → exercise dashboard, tooltip, and Export Usage command.

**Pass criterion**
All four dashboard sections render; tooltip fields present; export produces a file matching the
CLI's JSON/CSV shape; pricing override changes displayed rates.

---

## Notes

- **Independent review verdict: SHIP** (after parity fixes). All automated gates green in-sandbox.
- **Phase 6 IDE parser still pending Step 6.0 discovery.** The `ideCollector` is a stub; today
  `ReadAll` ≡ the CLI source. G48's dedup invariant is already in place so the combined total
  stays correct when the IDE parser lands.
- All cost figures are **estimates** (ADR-001 forbids billing reconciliation; ADR-008 labels them).

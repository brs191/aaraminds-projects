# ADR-009 — Usage analytics, export, and the source abstraction

**Status:** Accepted
**Date:** 2026-06-16

## Context

The v1.1 "usage-insight" increment adds the analysis features identified in
`research/dashboard-feature-analysis.md` (the Camp B / ccusage-aligned subset): period
trends, top-N consumers, context-window %, anomaly flagging, and JSON/CSV export. It also
lands the **groundwork** for Phase 6 multi-source capture (Copilot CLI + VS Code IDE)
without yet shipping the IDE parser.

Two constraints govern every decision here:

- **ADR-001 — zero network.** Every figure is computed locally from files the Copilot CLI
  already writes. The analytics layer adds no outbound call.
- **Go↔TS parity.** The same arithmetic runs in the Go CLI/MCP server and in the
  TypeScript extension. Divergence (especially in date bucketing) would make the two
  surfaces disagree, which is a correctness bug, not a cosmetic one.

This ADR also sets up the dedup and source model that the (still-pending) **ADR-007**
multi-source-capture decision will build on — see Step 6.0 (IDE data-source discovery
spike) and Step 6.1 in `IMPLEMENTATION_PLAYBOOK.md`.

## Decision

### 1. Analytics layer (`internal/analytics`, mirrored in `src/analytics/model.ts`)

Pure functions over a slice of sessions plus, where cost is involved, a `pricing.Config`
(ADR-008). No file system, no clock, no global state.

- **Period series** — `DailySeries` ("2006-01-02"), `WeeklySeries` (ISO-week "2006-W01",
  Start = Monday of the ISO week), `MonthlySeries` ("2006-01"). Each returns one `Bucket`
  per period **that has data**, sorted ascending by Start. A bucket carries Sessions,
  Credits, InputTokens, OutputTokens, and a per-model credit map.
- **Top-N** — `TopSessions`, `TopModels`, `TopProjects` return up to `n` `Consumer` rows
  ranked by **credits descending, ties broken by name ascending** for deterministic
  output. `n <= 0` returns all rows.
- **Context-window %** — `ContextWindowPct(session, cfg)` = `currentTokens /
  cfg.RateFor(model).ContextWindowTokens × 100`, returning 0 when the window is unknown
  (divide-by-zero guard).
- **Anomaly detection** — `AnomalousDays(daily)` flags days whose Credits exceed
  **mean + 2·σ** where σ is the **population** standard deviation of the daily series.
  Returns the flagged buckets in input order; returns empty when there are fewer than 3
  data points (too few to define a distribution). Deterministic.

Credits everywhere come from `budget.FromNanoAIU`, so analytics agrees with the budget
package to the last nano.

### 2. UTC bucketing — the parity invariant

Date bucketing **always** normalizes the session's billing time to **UTC** before
computing the day/week/month boundary. This is the single most important parity rule:
the same session lands in the same bucket key regardless of the host timezone, so the Go
and TS ports produce identical keys. The TypeScript port buckets in UTC for the same
reason. This is an acceptance gate (PHASE7_ACCEPTANCE.md, G38).

### 3. Export layer (`internal/export`, mirrored in `src/export/report.ts`)

Stable, deterministic serialization — field names and column order are an explicit public
contract so saved reports diff cleanly across runs. Pure with respect to inputs; CSV
writers take an `io.Writer` and never touch the file system directly.

- **`ToJSON(Report)`** — indented JSON. `Report` bundles `generatedAt`, `budgetState`,
  `daily`, `topSessions`, `topModels`, `topProjects`, and a flattened `sessions` view.
  All keys are **camelCase** (e.g. `budgetState`, `inputTokens`, `billingDate`).
- **`SessionsToCSV`** — columns:
  `date,project,model,source,credits,inputTokens,outputTokens,systemTokens,isActive,isFinal`.
- **`DailyToCSV`** — columns: `date,sessions,credits,inputTokens,outputTokens`.
- CSV uses `encoding/csv`, which RFC-4180-quotes any field containing a comma, quote, or
  newline — project names with commas do not corrupt the row.

### 4. Source / Collector abstraction + dedup (Phase 6 groundwork)

`session.Session` gains:

- **`Source string`** — which collector produced it. Known values: `"copilot-cli"` (the
  only live source today) and `"copilot-ide"` (Phase 6, not yet emitted).
- **`IsFinal bool`** — whether the billing/token figures are authoritative (settled
  shutdown) vs. a live snapshot of an active session.

A **`Collector` interface** (`Name()`, `Collect()`) abstracts a source. Two collectors
are registered: `cliCollector` (the existing session-state reader) and `ideCollector` (a
**stub** that returns nothing, pending the Step 6.0 discovery spike — the IDE parser
itself is deliberately not implemented yet).

**`ReadAll`** runs every registered collector, stamps `Source` defensively, concatenates,
**deduplicates by session ID across all sources**, and returns the survivors sorted by
start time (descending).

**Dedup rule** — sessions are keyed by **ID alone** (deliberately not Source+ID, so the
same session seen by two sources collapses to one). When two records share an ID the
winner is:

1. the one with `IsFinal == true` (settled billing beats a live snapshot); else
2. the one with the higher `TotalNanoAIU`.

This guarantees a shared session is **never double-counted**, which is the invariant the
combined CLI+IDE total will depend on once the IDE parser lands. The rule is mirrored
identically in the TS reader. It is an acceptance gate (PHASE7_ACCEPTANCE.md).

### 5. Surfaces

- **CLI** — `cmd/analyze` gains `--json`/`--csv` and the sections "USAGE TREND (last 14
  days)" (with anomaly flags), "TOP CONSUMERS", and context-window % on active sessions;
  `cmd/dashboard` surfaces the same; new `cmd/statusline` prints a ccusage-style one-liner.
- **MCP** — two new tools, `get_usage_timeseries` and `get_top_consumers` (ADR cross-ref:
  ARCHITECTURE.md "Phase 4" — the server now exposes **six** tools).
- **Extension** — `src/analytics/model.ts` + `src/export/report.ts`; a Usage Trend
  inline-SVG chart, Top Consumers tables, a context-% column, an input/output split, a
  richer status-bar tooltip, and the `copilotBudget.exportUsage` command.

## Rationale

- **Pure functions** make the analytics trivially testable and race-free; the MCP handlers
  stay stateless (no locking needed for concurrent tool calls).
- **UTC bucketing** is the only timezone-stable choice that keeps Go and TS agreeing; any
  local-time scheme would put the same session in different buckets on different machines.
- **Population σ** (not sample) keeps the anomaly formula identical and unambiguous across
  ports; the 3-point floor avoids flagging noise on sparse history.
- **Dedup by ID with final-wins** is the minimal rule that lets two sources observe the
  same session without double-counting — exactly the property Phase 6 needs.
- **Stable export contract** means CSV/JSON consumers (and the extension shelling out to
  the CLI, ccusage-style) do not break when internals change.

## Consequences

- The analytics/export layers are now a public contract; changing a JSON key or CSV column
  is a breaking change for downstream consumers and must be versioned.
- The `ideCollector` stub returns nothing, so today `ReadAll` ≡ the CLI source. When the
  IDE parser lands (after ADR-007 / Step 6.1), the dedup rule already in place prevents
  double-counting — no rework of the aggregation path.
- Go↔TS parity is now a standing obligation: any analytics/pricing change must land on
  both sides and be re-checked against the parity gates in PHASE7_ACCEPTANCE.md.

## References

- **ADR-001** — local file only, zero network (still holds; analytics adds no egress).
- **ADR-007** (planned) — multi-source capture / IDE parser; this ADR lands its
  groundwork (Source/Collector/dedup) ahead of the Step 6.0 discovery spike.
- **ADR-008** — overridable pricing config (supplies rates + context windows consumed here).

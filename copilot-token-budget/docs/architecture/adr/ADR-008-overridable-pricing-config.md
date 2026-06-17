# ADR-008 — Overridable local pricing configuration

**Status:** Accepted
**Date:** 2026-06-16

## Context

Until the v1.1 usage-insight increment, the tool's cost math was hardcoded. Model
rates (credits per million input/output tokens) and the monthly allowance lived as
constants in `internal/budget` (Go) and in the TypeScript extension. That created two
problems:

1. **Rate drift is a code change.** GitHub Copilot's published rates change, the AT&T
   promo allowance (7,000 credits/month) expires 2026-09-01, and new Claude models
   appear. Each change required editing source, rebuilding, and re-distributing the
   binary and the `.vsix`.

2. **Context-window % needs a per-model window value.** The new analytics layer
   (ADR-009) computes how full a session's context window is. That requires a usable
   context-window-tokens figure per model — another value that should not be hardcoded.

ADR-002 (Go zero external deps) and ADR-001 (local file only, zero network) both stand:
the fix must be local, dependency-free, and must never reach the network for live pricing.

## Decision

Externalize per-model rates, the monthly allowance, and the per-model context-window
to a single overridable local file: `platform.ConfigDir()/pricing.json`. Ship bundled
defaults in code; merge a user file over them; fall back to the bundled defaults on any
missing/malformed file.

### 1. Config shape

`internal/pricing` defines:

```go
type ModelRate struct {
    InputPerMillion     float64 `json:"inputPerMillion"`     // credits / 1M input tokens
    OutputPerMillion    float64 `json:"outputPerMillion"`    // credits / 1M output tokens
    ContextWindowTokens int64   `json:"contextWindowTokens"` // usable context window
}

type Config struct {
    AllowanceCredits int                  `json:"allowanceCredits"`
    Models           map[string]ModelRate `json:"models"`   // "sonnet"/"opus"/"haiku"
    Default          ModelRate            `json:"default"`  // fallback for unmatched names
}
```

### 2. Bundled defaults

| Key | Input (cr/M) | Output (cr/M) | Context window (tokens) |
|---|---|---|---|
| `sonnet` | 300 | 1,500 | 200,000 |
| `opus` | 500 | 2,500 | 200,000 |
| `haiku` | 100 | 500 | 200,000 |
| `default` | 300 | 1,500 | 200,000 |

`allowanceCredits` default: **7,000** (AT&T promo, until 2026-09-01).

**Rate source:** GitHub Copilot "models and pricing" reference, using the convention
**1 credit = $0.01** (credits/M token = USD/Mtoken × 100). The **200,000-token context
window** reflects Copilot's default (non-extended) configuration for the Claude models,
confirmed against the Copilot model reference (June 2026); the underlying Claude API
exposes a larger window via the extended/1M beta, which Copilot does not enable by
default. The context-window figures carry a `[VERIFY]` marker in code pending a
re-confirmation in the next quarterly freshness pass.

**All costs are estimates.** The tool reads local telemetry and applies a local price
table; it never reconciles against GitHub's authoritative billing (ADR-001). The UI and
docs must label cost figures as estimated.

### 3. Load + merge + fallback

`pricing.Load()`:

- Starts from the bundled defaults.
- Reads `platform.ConfigDir()/pricing.json` if present and merges it **over** the
  defaults — per-model, field-by-field. A partial file overrides only the fields it
  sets; zero values mean "keep the default." A model present in the file but not in the
  defaults is added.
- **Never fails hard** on a missing or malformed file: it logs to stderr and returns the
  bundled defaults. `Load()` returns an error only when the config directory itself
  cannot be resolved.
- `RateFor(model)` matches case-insensitively on the family substrings `opus`, `sonnet`,
  `haiku` (in that order); anything matching none returns `Default`.

`pricing.Default()` returns the bundled config with no file overrides (used as the
hard fallback when even `Load()` errors — e.g. in `cmd/statusline`).

`pricing.WriteDefaultIfAbsent()` writes the bundled defaults to `pricing.json` (mode
0600) when absent, giving users a starting point to edit. It is a no-op when the file
already exists. Intended for a future `init` flow.

### 4. Cross-platform path

The file lives at `platform.ConfigDir()/pricing.json` — the **same** helper introduced
in Phase 1 and reused for `state.json` in ADR-006. No second path-construction pattern.

| Platform | Path |
|---|---|
| macOS/Linux | `~/.config/copilot-token-budget/pricing.json` |
| Windows | `%AppData%\copilot-token-budget\pricing.json` |

### 5. Consumers

- **Phase 1** (`internal/budget`, `internal/analytics`): rates and context-window come
  from `pricing.Config`. `budget.SonnetInputRate` and friends remain for backward
  compatibility, but `pricing.Config` is the source of truth.
- **Phase 4 MCP** (`internal/tools/models.go`): `get_model_costs` sources rates from
  `internal/pricing` rather than hardcoded constants.
- **TS extension** (mirrored): `src/pricing/config.ts` loads the same defaults and merges
  an override file. The override path is set via the VS Code setting
  `copilotBudget.pricingPath` (an explicit file path, since the extension does not assume
  the Go config directory). Defaults are identical to the Go bundled defaults — Go↔TS
  parity is a v1.1 acceptance gate (ADR-009 / PHASE7_ACCEPTANCE.md).

### 6. Allowance precedence (TS extension)

The extension resolves the monthly allowance in this order:

1. An **explicitly set** `copilotBudget.monthlyAllowance` setting (user or workspace value) wins.
2. Otherwise `pricing.allowanceCredits` (from `pricing.json` or the bundled default).

This keeps the long-standing `monthlyAllowance` setting authoritative for users who set
it, while letting everyone else inherit the pricing config's allowance.

## Rationale

- **Config, not code, for values that change.** Promo expiry, rate changes, and new
  models become a one-line edit to `pricing.json` — no rebuild, no redeploy.
- **Merge-over-defaults, not replace.** A user who only wants to change the allowance
  drops a two-line file; they do not have to restate the full rate card.
- **Graceful fallback is mandatory.** A corrupted or absent file must never break the
  tool; first-run and broken-file cases both yield a working default.
- **`platform.ConfigDir()` reuse** keeps a single, tested, cross-platform path pattern
  (ADR-006).
- **Estimates, labelled.** ADR-001 forbids billing reconciliation; honest labelling
  ("estimated") is the correct UX, matching ccusage's stance.

## Consequences

- A new `pricing.json` becomes part of the config surface. Like `state.json` (ADR-006)
  it is written 0600 and contains no secrets — only rates, allowance, and window sizes.
- The TS extension reads its override from an explicit `copilotBudget.pricingPath` rather
  than the Go config dir, so a user wanting both halves to share an override must point
  the setting at the same file. This is acceptable; the defaults already match.
- Context-window figures are `[VERIFY]`-flagged; the quarterly freshness pass must
  re-confirm them against the Copilot model reference.

## Alternatives considered

| Alternative | Rejected because |
|---|---|
| Keep rates hardcoded | Every rate/promo/model change is a code change + redeploy |
| Fetch live pricing from an API | Violates ADR-001 (zero network); adds auth + failure modes |
| Replace-on-load (user file fully replaces defaults) | A partial file would wipe unspecified rates; brittle |
| Separate files per concern (rates, allowance, windows) | Three path patterns; no benefit over one merged file |
| Store pricing in `state.json` | Mixes mutable runtime state with static config; muddies ADR-006's invariant |

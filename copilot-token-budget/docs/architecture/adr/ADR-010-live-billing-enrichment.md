# ADR-010 — Live billing enrichment: opt-in contract and safety gates

**Status:** Accepted
**Date:** 2026-06-17

## Context

Phase 8.1 discovery (Step 8.1, `IMPLEMENTATION_PLAYBOOK.md`) confirmed that GitHub does expose
Copilot billing/usage APIs, but only as **org/enterprise aggregate** surfaces:

- `/orgs/{org}/copilot/billing`
- `/orgs/{org}/copilot/billing/seats`
- `/orgs/{org}/copilot/usage`
- `/orgs/{org}/copilot/metrics/reports/…`

These APIs have four hard constraints that shape this ADR:

1. **Aggregate only.** Data is org- or enterprise-scoped. It is not per-session, not per-user token
   billing. It cannot replace the session-level telemetry written locally by the Copilot CLI and
   VS Code IDE.
2. **Delayed.** Data is typically 24–48 hours behind the current moment. It cannot serve as a
   real-time billing check.
3. **Admin-provisioned auth required.** The `manage_billing:copilot` scope, org owner, or
   enterprise admin credentials are required. An individual contributor cannot self-configure this.
4. **Does not substitute for local telemetry.** Per-session breakdowns, instruction tokens,
   context-window %, and per-model breakdowns come from local `events.jsonl` (ADR-001, ADR-009).
   The GitHub API cannot provide these.

ADR-001 (amended 2026-06-17) already permits optional opt-in network calls for Phase 7+. ADR-002
(Go zero external deps) and ADR-006 (config storage) both remain in force. This ADR answers:
*under what contract can a live billing enrichment feature be added without violating any of these,
and without misleading users about the quality of their data?*

## Decision

### 1. Feature is opt-in and disabled by default

Live billing enrichment is a **config-gated feature** and defaults to **off**. When the feature
flag is absent or `false`, the tool behaves exactly as it did before Phase 8: local telemetry only,
zero network calls, all cost figures labelled **estimated**.

The feature is activated only by explicitly setting `liveBilling.enabled = true` in
`platform.ConfigDir()/config.json`. The tool must never auto-enable the live billing path —
neither on the presence of a token, nor on org membership, nor on any environment heuristic.

### 2. Config knobs and safety gates

A `liveBilling` block in `config.json` governs all live billing behaviour:

```json
{
  "liveBilling": {
    "enabled": false,
    "orgSlug": "",
    "tokenEnvVar": "COPILOT_BILLING_TOKEN",
    "cacheMaxAgeHours": 24,
    "requestTimeoutSecs": 10,
    "dryRun": false
  }
}
```

| Knob | Default | Purpose |
|---|---|---|
| `enabled` | `false` | Master switch; must be `true` for any network call |
| `orgSlug` | `""` | Required when enabled; names the GitHub org to query |
| `tokenEnvVar` | `"COPILOT_BILLING_TOKEN"` | Env-var name that holds the PAT; never stored in the config file itself |
| `cacheMaxAgeHours` | `24` | How long a cached response is treated as fresh; floor 1h, ceiling 72h |
| `requestTimeoutSecs` | `10` | HTTP timeout per call; prevents live billing from blocking the tool |
| `dryRun` | `false` | When `true`, the config path exercises but no real HTTP call is made; for CI gate tests |

**Mandatory safety gate rules:**

- If `enabled` is `false` (or absent), no HTTP call is ever made. No exceptions.
- If `orgSlug` is empty when `enabled` is `true`, log a clear config error and fall back to
  estimated-only mode (not a hard failure of the tool).
- The auth token must never be written to any config file. It must be read from the environment
  variable named by `tokenEnvVar`.
- If that env var is not set when `enabled` is `true`, log a clear warning and fall back to
  estimated-only mode.
- `dryRun = true` overrides all network paths and must produce zero actual HTTP requests. This is
  the CI test path for the opt-in contract.

### 3. Local-first behavior is the immutable default path

The local session-telemetry path established by ADR-001 through ADR-009 is never modified by this
feature. When live billing is disabled (the default), nothing changes. When live billing is enabled,
it **adds** an enrichment layer; it does not **replace** the local path.

Execution order is always:

1. Read and parse local telemetry (ADR-001 / ADR-009 path) — this always runs.
2. If and only if `liveBilling.enabled == true` and all preconditions are met (orgSlug set, token
   present, feature not in dry-run), attempt to fetch the org-aggregate billing snapshot.
3. Merge: local session counts, tokens, and credit figures are never overwritten by the enrichment
   layer. The local result is the floor; enrichment can only add.

If the enrichment fetch fails at any point (network error, auth failure, timeout, unexpected response
shape), the tool continues with the local-only result. No partial result is applied.

### 4. Authoritative data must never silently overwrite estimates

When a live billing response is received, it **annotates** the display — e.g., "your org has used
X credits this billing cycle according to GitHub, as of approximately Y hours ago." It must **not**
silently overwrite the session-level credit figures that local telemetry computed.

The invariant: `session.Credits` and `session.TotalNanoAIU` always come from local telemetry. A new
top-level field, `OrgBillingSnapshot`, carries the enrichment result. The two figures may differ
(because the API is aggregate and delayed). That difference must be surfaced, not hidden.

Concrete prohibition: no code path may reassign `session.Credits` or any per-session token field
from an `OrgBillingSnapshot` value. Doing so is a correctness bug, not a performance tradeoff.

### 5. UI labels must clearly show authoritative vs estimated vs unavailable

Every figure shown to a user carries a data-quality label. Three states, no others:

| Label | When used |
|---|---|
| `(estimated)` | Local telemetry only; live billing disabled or not configured |
| `(org aggregate, ~Xh ago)` | Live billing snapshot fetched successfully; X = hours since the snapshot's `asOf` timestamp |
| `(unavailable)` | Live billing enabled and configured, but fetch failed or returned no usable data |

The label appears adjacent to the figure in CLI output, in dashboard cards, and in export metadata.
It is not optional and is not hidden behind a verbose or debug flag. The Go CLI and the TypeScript
extension must produce identical labels for the same data state (Go↔TS parity obligation, ADR-009).

### 6. No live billing implementation unless Phase 8.1 discovery succeeded

This ADR is written under the explicit constraint that Phase 8.1 returned a **conditional go** —
APIs exist but are aggregate, delayed, and admin-only. If a future discovery step finds a materially
different result (e.g., per-user, real-time billing), a new ADR must be written. This ADR governs
only the API surface confirmed by Phase 8.1.

Gate rule: before any implementation work on Steps 8.3–8.5, the developer must verify that
`IMPLEMENTATION_PLAYBOOK.md` Step 8.1 is marked ✅ Complete with a non-empty, positive discovery
result. There is no implementation without a completed discovery.

### 7. Cross-cutting constraints

| Constraint | Rule |
|---|---|
| **ADR-002 (Go zero external deps)** | Use only `net/http` from the Go standard library for the enrichment HTTP client. No third-party HTTP or OAuth library. |
| **ADR-006 (config storage)** | The `liveBilling` block lives in the existing `config.json`. Token is env-var only — never written to disk. |
| **ADR-009 parity obligation** | The TypeScript extension must implement the same three-state label logic. A figure in the extension must carry the same label the CLI would show. |
| **ADR-007 (dedup)** | The dedup rule (by session ID, final-wins) applies to local sources only. The `OrgBillingSnapshot` is not a session record and is not subject to dedup. |
| **ADR-008 (pricing estimates)** | Local session cost figures computed from `pricing.Config` remain labelled `(estimated)` even when live billing is enabled. The two figures serve different scopes and must not be conflated. |

## Rationale

- **Opt-in is the contract, not an implementation detail.** The billing APIs require admin
  provisioning by definition. Matching the UX to the access model — admin-configured, explicitly
  enabled — makes the feature coherent and auditable.
- **No silent fallback, ever.** A tool that quietly shows estimates when live billing fails looks
  correct when it is lying. Explicit labels force honesty and prevent misinterpretation in budget
  reviews. The label `(unavailable)` is a better outcome than a silently-wrong figure.
- **Aggregate ≠ session.** The Phase 8.1 result is explicit: the GitHub APIs do not replace local
  session telemetry. They provide an org-scope cross-check, not a per-session ledger. Treating them
  as equivalent is a correctness bug.
- **Token in env var, not config file.** Following the ADR-006 pattern: secrets do not live in
  config files. Env-var lookup gives the admin control over injection (shell profile, CI secrets,
  Vault) without the tool needing to know the mechanism.
- **Cache before network.** A 24h default cache horizon matches the 24–48h API delay: repeat
  invocations do not redundantly hit a rate-limited API that cannot provide fresher data anyway.
  The cache is the right abstraction for data that is stale by design.
- **Immutable local floor.** Ensuring local telemetry always runs — even when enrichment is enabled
  — preserves offline behavior, maintains per-session granularity, and means a failed enrichment has
  zero impact on the core use case.

## Consequences

- A `liveBilling` config block becomes part of the config surface. Like `pricing.json` (ADR-008),
  it contains no secrets; the token is env-var only.
- `session.Session` gains a nullable `OrgBillingSnapshot *OrgBillingSnapshot` field. This is
  additive; existing serialisation is unchanged for nil values.
- Export outputs (`sessions.csv`, JSON report) gain an `orgBillingSnapshot` section when the field
  is non-nil. When nil, the section is omitted. This is a backwards-compatible addition.
- The label `(org aggregate, ~Xh ago)` is deliberately conservative — it does not claim
  "authoritative" without qualification, because the data is delayed and aggregate-scoped. Future
  ADRs may revise this if the API surface improves.
- A `--live-billing-status` flag (or equivalent sub-command) can report the current enrichment
  state, last-refresh time, and cache age without triggering a live fetch. This is the zero-risk
  observability path for admins.
- Any implementation touching Steps 8.3–8.5 must pass a gate test: running the tool with
  `liveBilling.enabled = false` must produce output indistinguishable from a pre-Phase-8 run —
  no new fields, no new network traffic, no changed labels beyond the existing `(estimated)` marker.

## Alternatives considered

| Alternative | Rejected because |
|---|---|
| Auto-enable when org token env var is present | Implicit enablement; violates the opt-in rule; a token's presence should not silently change app behavior |
| Overwrite session estimates with API aggregate | The API is aggregate and delayed — applying it at session scope would produce wrong numbers and mislead users |
| Show API figures as primary, local as fallback | Inverts the quality hierarchy; local telemetry is more granular and more current than a 24–48h delayed aggregate |
| Store PAT in `config.json` | Secrets in config files; violates ADR-006 pattern and creates a plaintext security surface |
| Skip caching, fetch on every invocation | The API is rate-limited and the data is already 24–48h stale; more-frequent fetching adds latency and risk with zero freshness benefit |
| Separate dedicated config file for live billing | Creates a second config path pattern; the `liveBilling` block in the existing `config.json` is the minimal surface addition |
| Label all org-aggregate figures "authoritative" | The data is delayed and aggregate-scoped, not real-time per-session; unqualified "authoritative" would be misleading |
| Disable the feature entirely (Phase 8 stop) | Phase 8.1 returned a conditional go, not a stop; the constraints are manageable with the contract defined here |

## References

- **ADR-001** (amended 2026-06-17) — Local-first; optional network opt-in for Phase 7+. This ADR
  implements the first concrete opt-in network path permitted by that amendment.
- **ADR-002** — Go zero external deps (governs HTTP client choice).
- **ADR-006** — Config storage and secret handling patterns.
- **ADR-007** — Multi-source dedup (not affected by the enrichment layer).
- **ADR-008** — Overridable pricing config (session cost estimates remain labelled as such).
- **ADR-009** — Source/Collector abstraction; Go↔TS parity obligation.
- **IMPLEMENTATION_PLAYBOOK.md, Step 8.1** — Phase 8.1 discovery: conditional go; aggregate-only,
  24–48h delayed, admin-auth required, no per-session or per-user token billing.

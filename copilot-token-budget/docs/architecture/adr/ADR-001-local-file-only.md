# ADR-001 — Local file read first, optional network for enrichment

**Status:** Amended (2026-06-17)
**Original Date:** 2026-06-13
**Amendment Rationale:** IDE Chat sessions are stored locally, but server-side GitHub API enables future token cost enrichment (Phase 7). Keep "local-first" but allow opt-in network paths for non-critical data.

## Context

The Copilot CLI writes billing telemetry to `~/.copilot/session-state/<uuid>/events.jsonl` locally.
VS Code IDE Chat sessions are stored in Xodus DB at `~/.config/github-copilot/ic/` locally.
However, IDE Chat token costs are NOT stored locally (server-side only on GitHub). An alternative
would be to call the GitHub Copilot API to retrieve enriched cost data for IDE sessions.

## Decision

**Primary path (Phase 6):** Read local files only (CLI events.jsonl + IDE Xodus sessions).  
**Secondary path (Phase 7+, opt-in):** Allow optional network calls to GitHub API for IDE token costs, **guarded by:**
- Explicit user consent (must opt-in, default off)
- Configurable auth (token required, no auto-refresh)
- Fallback graceful (if API fails, show "cost data unavailable" not error)
- Testing (test script validates zero unintended network calls)

## Rationale

**Local-first (unchanged):**
- Local files are available offline and on restricted corporate networks
- `events.jsonl` is more granular than the API (per-session breakdown, instruction tokens)
- Local reads are instantaneous; no latency
- No data leaves the machine by default — privacy and security by default
- IDE sessions + history satisfy Phase 6 without network calls

**Optional enrichment (Phase 7+):**
- IDE Chat token costs only available server-side; GitHub API is the only path
- Opt-in model preserves privacy default; users choose to export costs
- Allows AT&T team to see full IDE usage this quarter without breaking ADR-001

## Consequences

- Phase 6: Tool works offline for CLI + IDE sessions/history (local-only)
- Phase 7+: Optional GitHub API path requires auth management + test guards (see [ADR-006](./ADR-006-config-storage.md) for token storage)
- Multi-machine case still out of scope (machines are independent)
- Users must affirmatively opt-in to network calls; no implicit data export

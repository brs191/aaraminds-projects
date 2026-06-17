# ADR-001 — Local file read only, no GitHub API

**Status:** Accepted
**Date:** 2026-06-13

## Context

The Copilot CLI writes billing telemetry to `~/.copilot/session-state/<uuid>/events.jsonl` locally.
An alternative would be to call the GitHub Copilot API to retrieve usage data.

## Decision

Read local files only. Never call the GitHub API or any external service.

## Rationale

- Local files are available offline and on restricted corporate networks
- GitHub API requires auth tokens; managing refresh/rotation adds complexity
- `events.jsonl` is more granular than the API (per-session breakdown, instruction tokens)
- Local reads are instantaneous; API calls add latency and failure modes
- No data leaves the machine — privacy and security by default

## Consequences

- Tool only works on the machine running the Copilot CLI sessions (correct use case)
- Budget reflects sessions on this machine only (multi-machine use case is out of scope)

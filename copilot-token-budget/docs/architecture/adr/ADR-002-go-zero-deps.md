# ADR-002 — Go tool with zero external dependencies (amended 2026-06-17)

**Status:** Accepted (amended)
**Date:** 2026-06-13 (amended 2026-06-17)
**Amendment Rationale:** Phase 6 IDE Chat data (Xodus DB format) requires binary parsing. Nitrite is the only stable embedded-database SDK; exception justified for multi-source data layer.

## Context

The core data layer (session reader, budget tracker, instruction analyzer) reads from:
1. **CLI:** `~/.copilot/session-state/<uuid>/events.jsonl` (text, parseable with stdlib)
2. **IDE Chat:** `~/.config/github-copilot/ic/` (Xodus binary DB format, opaque without SDK)

Xodus is JVM-only; no binary format spec exists. Go SDK (`github.com/noelyoo/go-nitrite` or equivalent) is the only viable path to avoid reverse-engineering JVM internals.

## Decision

Go 1.21, standard library + **single exception: Nitrite embedded-database SDK for IDE Chat parsing** (Phase 6+).

## Rationale

- Static binary: `go build` produces a single executable; no runtime install required
- AT&T npm/pip registry auth issues do not affect Go tool installation (Nitrite is from GitHub Releases, not npm/pip)
- Fast startup (< 50ms) suitable for WezTerm badge refresh loop (Nitrite loads in-memory only if IDE data requested)
- Standard library has everything needed for CLI: `os`, `bufio`, `encoding/json`, `time`, `path/filepath`
- **Nitrite exception justified:** 
  - IDE data format is proprietary binary (Xodus); no alternative without reverse-engineering JVM serialization
  - Nitrite SDK is the upstream solution (Apache-affiliated project with stable Go bindings)
  - Single well-scoped dependency (embedded DB only); no transitive deps on web frameworks, logging, etc.
  - Only loaded if IDE data is queried; CLI-only operation remains zero-dep
- Zero supply-chain risk acceptable for this specific case (Nitrite is open-source Apache project with GitHub releases)

## Consequences

- `go.mod` includes one external dependency: `github.com/noelyoo/go-nitrite` (or equivalent) for IDE Chat parsing
- Binary size increases ~2–5 MiB (embedded DB SDK)
- CI build time increases ~10 seconds (Nitrite dependency resolution)
- CLI reader remains stdlib-only (clean separation: CLI = stdlib, IDE = Nitrite)
- **Future exceptions:** If another binary data source emerges, re-evaluate rather than adding more SDKs
- If Nitrite SDK becomes unavailable, fallback to JSON metadata parsing (degrades gracefully; no crash)

## Alternatives Considered

1. **JSON metadata fallback only** — Loses per-turn IDE granularity in Phase 6 (acceptable but less useful when Phase 7 token enrichment arrives)
2. **Xodus binary reverse-engineer** — High risk; no spec; JVM internals subject to change
3. **Defer IDE to Phase 7 (GitHub API only)** — Delays IDE visibility by 2–3 weeks; team needs it now
4. **CLI-only release** — Satisfies "zero deps" but blocks team's IDE session visibility requirement

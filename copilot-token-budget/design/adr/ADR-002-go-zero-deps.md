# ADR-002 — Go tool with zero external dependencies

**Status:** Accepted
**Date:** 2026-06-13

## Context

The core data layer (session reader, budget tracker, instruction analyzer) needs a runtime.

## Decision

Go 1.21, standard library only. `go.sum` is empty.

## Rationale

- Static binary: `go build` produces a single executable; no runtime install required
- AT&T npm/pip registry auth issues do not affect Go tool installation
- Fast startup (< 50ms) suitable for WezTerm badge refresh loop
- Standard library has everything needed: `os`, `bufio`, `encoding/json`, `time`, `path/filepath`
- Zero supply-chain risk from third-party modules

## Consequences

- JSONL parsing is manual (no `gjson` etc.) — acceptable given the simple event schema
- WezTerm badge uses raw OSC escape sequences — no terminal library dependency

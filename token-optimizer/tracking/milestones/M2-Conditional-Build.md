# M2 — Conditional build

**Owner:** Raja  ·  **Status:** Blocked by M1 = Green  ·  **Exists only on a Green gate**
**Effort:** ~3–5 engineer-months `[VERIFY]`
**Source:** `../../product/AI_Token_Optimizer_Product_Brief_2026-05-24.md` (Phase 2), `../../design/AI_Token_Optimizer_Agent_Blueprint_v0.1.md`

This milestone is a placeholder until M1 clears Green. If M1 is Amber or Red, M2 never opens — mark it cancelled.

## Goal

Build Option B — the narrow product: only the unserved wedge, wrapping commodity parts rather than reinventing them. Local-first, zero-egress, IntelliJ parity, and the measured Fidelity Floor as the headline guarantee.

## Scope

**In:** bundled Go sidecar (Go core plus Go MCP servers, supervised as `127.0.0.1` child processes); localhost loopback proxy for interception; LLMLingua-2 reached through a Python compression sidecar; metadata-only agent egress; VS Code `.vsix` plus IntelliJ plugin with manual local install; the Fidelity Floor.

**Out:** model routing, semantic caching, multi-user/team setup, and the broad v0.1 optimizer. Budget enforcement is a later one-line LiteLLM add, not an M2 deliverable.

## Entry pre-work

- [ ] Re-scope the v0.1 blueprint tightly around the wedge — bump to v0.2
- [ ] Fold in the four Required Fixes from the Module 5 systems review
- [ ] Redesign the Fidelity Floor to resolve the systems-review findings before relying on it

## Gate

Defined when M2 opens. At minimum: the product holds the Fidelity Floor (quality regression staying within the ≤ 5% bound in ongoing use) and compression latency stays within the < 300 ms p95 budget, measured against the spike-established baseline.

## Tasks

- [ ] Defined when M1 clears Green

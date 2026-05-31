# Persona / Skill / Agent Usage Summary

**Date:** 2026-05-31 · **Scope:** the persona-, skill-, and agent-driven redo of the Scrum Master Agent design + code.

This records what machinery was actually used and **what each piece changed** (the delta) — not a process narrative. The first build was largely freelanced; this pass ran it through the AaraMinds persona system, the engineering skills, and verification subagents.

## 1. Personas (loaded + applied)

Personas are markdown composition files, not executable — I loaded them and applied their gates by hand.

| Persona | Gate applied | What it changed |
|---------|-------------|-----------------|
| **AI Engineering Architect v1.2** (composed over base modules) | Build-vs-Buy enumeration | ADR-0001 now has **3 named options** (LangGraph / fixed-stack-native / all-Python) with trade-off tables; PRD §1 gained a build-vs-buy paragraph |
| | Verification Trigger Gate | Spawned the two fact-check subagents *before* shipping vendor/API claims; kept `[VERIFY]`/sourced discipline |
| | Lifecycle Coherence Gate | Blueprint §10: explicit first-review trigger, what review produces, redesign triggers |
| | Threshold framing (derive or decline) | Metrics stay targets-to-baseline; no fabricated numbers survived |
| **AI Agent Blueprint Advisor v1.1** (Module 8 process) | Boundary Gate (boundary first) | NEW Blueprint §3: In-scope / Out-of-scope / Human-only table set before tools/workflow |
| | Defining Operational Constraint | Named the invariant — **"human-approved writes by construction"** — and threaded it through Blueprint, Architecture, PRD |
| | Single-Agent Default | Blueprint §4 justifies one agent, not multi-agent |
| | Architecture Theatre Check | Forced trust boundaries (§8), failure modes (FMEA-lite), control plane (§7) to be explicit, not decorative |
| | Diagram Completion Check | Blueprint §11 Mermaid shows the full approval routing: request → outcome → post-approval → **rejection path** → audit |

## 2. Skills (invoked)

| Skill | Output |
|-------|--------|
| `engineering:architecture` | Re-authored **ADR-0001** in the skill's ADR format — Options Considered tables, Trade-off Analysis, Consequences, Action Items |
| `engineering:system-design` | Framed the Architecture component view, trust boundaries, and data-model section |
| `engineering:testing-strategy` | NEW **evaluation/Test_Strategy.md** — pyramid, by-component plan, DOC-weighted coverage (100% on no-write-without-approval), example cases, gaps |
| `engineering:code-review` | Security/perf/correctness lens that drove the five code fixes below |
| `product-management:write-spec` (prior turn) | Original PRD structure |

## 3. Agents (subagents, parallel)

Two `general-purpose` agents ran independent verification with web access (the sandbox has no Go toolchain, so they verified APIs against live docs rather than compiling).

| Agent | Severity | Finding → action |
|-------|----------|------------------|
| **Integration fact-check** | 🔴 Critical | **Teams O365 connector + MessageCard retires 2026-05-18..22** (during P0). → Rewrote `teams.go` to a Power Automate **Workflows** webhook + **Adaptive Card**; corrected PRD/Architecture/README/.env |
| | 🟠 Major | **JQL `/search` removal date wrong** — I'd written 2025-05-01; actually deprecated then, **removed ~2025-10-31**. → Fixed in PRD (×2) + Architecture; added "IDs-only / no `total`" nuance |
| | 🟢 Confirmed | Time-tracking fields correct, but flat fields are **read-only** — writes use the `timetracking` composite. → Noted in PRD + Architecture. Rate-limit model + scopes verified accurate |
| **Code/API audit** | — | mcp-go v0.43.0, LangGraph, langchain-mcp-adapters, psycopg, httpx APIs **all verified correct** against current sources (no compile possible) |
| | 🟢 Minor | `fetchone()` None-guard; half-written audit chain on Teams failure; `Blocked`-status issues dropped from brief body; over-clever ternaries; unpinned deps → **all five fixed** |

## 4. Code changes (from the code-review skill + audit agent)

1. `teams-adapter/internal/teams/teams.go` — MessageCard → Adaptive Card in the Workflows `attachments` envelope (critical EOL fix).
2. `orchestrator/.../graph.py` — Teams delivery failure now records `action_audit.result = "failed"` instead of crashing mid-chain (DOC: audit never half-written).
3. `orchestrator/.../brief.py` — added a catch-all bucket so `Blocked`/`In Review` issues appear in the body, not only in Blockers; de-clevered the side-effect ternaries.
4. `orchestrator/.../audit.py` — None-guard on `INSERT ... RETURNING id`.
5. `orchestrator/pyproject.toml` — bounded dependency ranges (was lower-bound-only).
6. `tests/test_brief.py` — added a status-completeness regression test (now 5 tests, all pass).

## 5. New / re-authored artifacts

- **NEW** `design/Agent_Blueprint.md` — the Module 8 blueprint that didn't exist before (the single biggest persona-driven addition).
- **NEW** `evaluation/Test_Strategy.md`.
- **Re-authored** `design/adr/0001-langgraph-orchestration.md` in the architecture-skill format.
- **Updated** `Scrum_Master_Agent_PRD.md`, `design/Architecture.md` (Teams, JQL date, time-tracking, DOC, build-vs-buy).

## 6. Honest limitations

- **Module 8 not deep-loaded.** The full 31KB module wasn't read; I worked from the Blueprint Advisor persona's enumeration of the Module 8 process + its four gates. The blueprint is faithful to that, not to the full module contract.
- **Go uncompiled.** No Go toolchain and the module proxy is blocked in this sandbox; the agents verified the Go API against docs/source, but a local `go build` is still the real proof.
- **Duplicate agent dispatch.** I accidentally launched each verification agent twice (four ran, not two). The duplicates returned consistent findings — a useful cross-check — but it was wasted compute, not a deliberate design.
- **PRD treated as preserve-and-correct,** not torn down. A full from-scratch re-author of correct content would have been churn (an AaraMinds anti-pattern); the high-value re-authoring went into the new Blueprint and the reformatted ADR. Flagging since "full rewrite" was the chosen option.

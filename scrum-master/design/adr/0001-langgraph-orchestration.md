# ADR-0001: LangGraph for agent orchestration

**Status:** Accepted
**Date:** 2026-05-31
**Deciders:** Raja (owner)
**Produced via:** `engineering:architecture` skill (ADR format) under the AaraMinds AI Engineering Architect persona (Build-vs-Buy + Verification + Lifecycle-Coherence gates).

## Context

The Scrum Master Agent's defining constraint is **human-approved writes by construction** (see [Agent_Blueprint.md](../Agent_Blueprint.md) §6): the agent must pause on a recommendation, persist state durably, and resume on an approval that may arrive hours or days later — across five multi-step features. The AaraMinds fixed stack is Go / Spring Boot (Java 21+) backends, Azure-primary, Postgres. Introducing Python is a deliberate deviation that workspace governance requires recording as an ADR. The load-bearing forces: durable human-in-the-loop (HITL) pause/resume, reuse of the existing Go MCP-server pattern, and containment of stack drift.

## Decision

Use **LangGraph (Python)** as the orchestration/reasoning runtime, scoped to the reasoning layer only. The Jira integration stays a **Go MCP server** (consumed via `langchain-mcp-adapters`); supporting services (scheduler, webhook listener, channel adapters) stay Go; state/approval/checkpoints persist in **Postgres on Azure**; deploy on **Azure Container Apps**, Key Vault via managed identity.

## Options Considered

This is a capability-acquisition decision, so alternatives are enumerated (not just "LangGraph vs status quo").

### Option A — LangGraph (Python), Go MCP integration *(chosen)*

| Dimension | Assessment |
|-----------|------------|
| Complexity | Medium — second runtime, but HITL is a library feature |
| Cost | Low — OSS; no new managed service |
| Scalability | High — checkpointer scales on Postgres |
| Team familiarity | Medium — Python known; LangGraph new |
| DOC fit | **High — durable `interrupt()`/checkpoint maps 1:1 to the approval gate** |

**Pros:** durable interrupt/checkpoint out of the box; Postgres checkpointer (no new datastore); integration layer stays house-language Go and reusable across agents.
**Cons:** polyglot ops (Python + Go); a network hop orchestrator↔MCP; team holds two languages.

### Option B — Go + Spring control plane, native tool-calling (fixed-stack purity)

| Dimension | Assessment |
|-----------|------------|
| Complexity | High — durable HITL pause/resume hand-built |
| Cost | Medium — more build time |
| Scalability | High |
| Team familiarity | High — pure fixed stack |
| DOC fit | Medium — correct, but the safety-critical gate is bespoke code |

**Pros:** zero new language; no drift; one deploy toolchain.
**Cons:** re-implements durable HITL state machine — error-prone for the exact mechanism the product can least afford to get wrong.

### Option C — All-Python (LangGraph + Python-native Jira client)

| Dimension | Assessment |
|-----------|------------|
| Complexity | Low — single runtime |
| Cost | Low |
| Scalability | Medium |
| Team familiarity | Medium |
| DOC fit | High |

**Pros:** simplest ops; no cross-language hop.
**Cons:** forfeits the reusable Go MCP asset; pushes integration into a non-fixed-stack language — drift spreads from the reasoning layer into integration, the opposite of containment.

## Trade-off Analysis

The decision turns on *where the risk should live*. The HITL gate is the product's safety-critical mechanism; Option B places that exact mechanism in bespoke code, which is the worst place for the highest-stakes logic. Option A buys a battle-tested durable-interrupt primitive and pays for it in polyglot ops — an operational cost, not a correctness risk. Option C minimizes ops but lets Python leak into the integration layer, spreading drift rather than containing it. Choosing A accepts a bounded, well-understood cost (two runtimes) to de-risk the thing that matters most (durable approvals) while keeping drift fenced to one layer.

## Consequences

- **Easier:** durable approvals; reusable Go MCP integration; Postgres-only state.
- **Harder:** two runtimes to build/test/deploy; an orchestrator↔MCP network hop to secure and observe.
- **Revisit when:** the orchestration graph stays trivial enough that native tool-calling would do (then Option B/C reopen); or LangGraph introduces a breaking change in the interrupt/checkpoint API.

## Action Items

1. [ ] Pin LangGraph + checkpointer + adapters to tested ranges (not lower-bound-only).
2. [ ] Stand up the Postgres checkpointer and assert "no write without an approval row" in an integration test (DOC scorer).
3. [ ] Document the orchestrator↔MCP hop in the threat model (auth, network policy).
4. [ ] Re-evaluate at the P1 gate whether graph complexity still justifies LangGraph.

## Open sub-decision

Go MCP server vs Python-native Jira client — **resolved: Go MCP server** (see [Open_Questions.md](../../planning/Open_Questions.md) #3).

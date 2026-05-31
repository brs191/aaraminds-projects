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

### Option C
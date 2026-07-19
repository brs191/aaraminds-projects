# ADR-008: MCP Gateway and Auth Model

**Date:** 2026-07-13  
**Status:** Accepted for P0 exit  
**Owners:** Platform + Security  
**Related docs:** `action_plan.md`, `code/libs/mcpapi`, `code/libs/auditusage`

---

## 1. Context

DIF's MCP/API surface is the governed tool boundary for agents. It must authorize, validate, route, audit, meter, and return grounded evidence without duplicating retrieval or graph logic.

---

## 2. Decision

P0 uses an internal bearer-token gate with constant-time comparison for the thin `search_docs` MCP/API boundary.

Pilot/remote deployments must move to OAuth 2.1 + PKCE aligned with enterprise MCP gateway posture before exposure beyond controlled internal deployments.

Every MCP/API call must:

1. Validate required tenant/project/corpus/request fields.
2. Enforce corpus-level authorization and admission.
3. Route to service-layer code rather than duplicating retrieval/ranking.
4. Return source-anchored results or explicit failure statuses.
5. Record separate audit and usage events when a recorder is configured.
6. Avoid logging raw parameters, raw queries, snippets, or document text by default.

---

## 3. Consequences

- No unauthenticated MCP/API entry point is allowed.
- Unauthorized attempts must still be governable through audit/usage records.
- Cross-graph tools must remain disableable or return explicit RIF status when RIF is unavailable.
- Tool schemas should be generated from code when feasible before pilot stabilization.

---

## 4. P0 evidence

- `code/libs/mcpapi`
- `code/libs/auditusage`
- `evaluation/audit_usage_checks.py`
- `evaluation/search_docs_checks.py`
- `evaluation/run_p0.py`

# ADR-013: P0 Security Threat Model

**Date:** 2026-07-13  
**Status:** Accepted for P0 exit  
**Owners:** Security + Engineering  
**Related docs:** `action_plan.md`, `code/libs/mcpapi`, `code/libs/logging`, `.github/workflows/ci.yml`

---

## 1. Context

DIF indexes enterprise documents and joins them to code intelligence. P0 must establish a security baseline before P1 expands cross-graph behavior.

---

## 2. Threats and controls

| Threat | P0 control |
|---|---|
| Unauthorized tool access | Bearer-token auth with constant-time comparison for P0 internal MCP/API; OAuth 2.1 + PKCE required before pilot/remote exposure. |
| Unauthorized corpus access | Required scope fields plus corpus admission gate; non-admitted corpora fail closed. |
| Ungrounded or fabricated answers | P0 `search_docs` returns anchored evidence only and has no free-form answer field. |
| Raw document leakage in logs | Safe structured logging helpers redact raw document text fields and secret-like values. |
| Audit blind spots | MCP/API records success, denied corpus, and unauthorized attempts when recorder is configured. |
| Usage metering privacy leakage | Usage records are non-PII and separate from audit records. |
| RIF false-empty compatibility | `rifcompat` distinguishes missing, incompatible, empty-shadow, and compatible RIF states and allows AGE fallback. |
| Dangling graph edges or unanchored passages | Graph emitter and search service fail closed on dangling/unanchored output. |
| Degenerate ingestion promotion | Ingestion-run guard blocks empty/all-failed/no-anchor/no-passage promotion. |
| Supply-chain/deployment side effects in CI | CI has read-only repository permissions, runs tests/harnesses only, and has no Azure login, registry, deployment, or publish job. |

---

## 3. P0 boundaries

P0 does not claim:

- source ACL propagation
- production OAuth deployment
- production secret retrieval
- production container publishing
- agent free-form answer generation
- P1 `DESCRIBES`, `docs_for_code`, `code_for_doc`, or `drift_report`

These require later phase gates.

---

## 4. Consequences

- Security-sensitive expansions must preserve fail-closed behavior and audit/usage separation.
- Remote/pilot deployments must replace internal bearer-token posture with OAuth 2.1 + PKCE.
- Prompt construction for future agent features must fence retrieved text as data, not instructions.
- CI changes that add secrets, publishing, cloud login, or registry access require explicit review.

---

## 5. P0 evidence

- `code/libs/mcpapi`
- `code/libs/auditusage`
- `code/libs/logging`
- `code/libs/searchdocs`
- `code/libs/graphemit`
- `code/libs/rifcompat`
- `evaluation/run_p0.py`
- `evaluation/path_checks.py`
- `.github/workflows/ci.yml`

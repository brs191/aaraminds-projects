# ADR-012: Observability, Audit, and Usage Schema

**Date:** 2026-07-13  
**Status:** Accepted for P0 exit  
**Owners:** Platform + Security  
**Related docs:** `action_plan.md`, `code/libs/logging`, `code/libs/auditusage`, `code/migrations/001_dif_meta_initial.sql`

---

## 1. Context

DIF handles enterprise documents and code metadata. Observability must support operations and governance without leaking raw document text, credentials, or request payloads.

---

## 2. Decision

P0 separates:

- safe structured operational logging
- security/audit events
- non-PII usage metering

Audit events record principal/security dimensions, scope, tool, parameters hash, outcome, latency, and returned source refs where applicable.

Usage events record non-PII metering dimensions such as event type, project/corpus, counts, latency, token units, embedding units, and error class. Usage events must not store `principal_id`, raw source refs, raw parameters, raw query text, snippets, or document text.

Structured logging helpers allow operational metadata only and redact obvious credentials, bearer/JWT-like tokens, private-key markers, database URL passwords, secret-like key/value pairs, and raw document text fields.

---

## 3. Consequences

- Audit and usage are separate tables and separate write shapes.
- Unauthorized attempts are still recorded against a migration-backed unknown-scope sentinel.
- Future tracing/metrics must preserve the same redaction posture.
- No raw enterprise document text is logged by default.

---

## 4. P0 evidence

- `code/libs/logging`
- `code/libs/auditusage`
- `code/libs/mcpapi`
- `code/migrations/001_dif_meta_initial.sql`
- `evaluation/audit_usage_checks.py`
- `evaluation/run_p0.py`

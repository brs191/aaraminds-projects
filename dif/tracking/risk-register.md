# DIF Risk Register

**Status:** Active tracking artifact  
**Date:** 2026-07-08  
**Owners:** Engineering + Production + QA + Security  
**Source of truth:** `../action_plan.md` remains the operating source of truth. Update that file whenever a risk is added, closed, escalated, or materially changed.

---

## 1. Purpose

This register tracks DIF blockers, risks, mitigations, owners, and review cadence for production and engineering teams.

It is seeded from `../action_plan.md` section `0.5 Current blockers and risks` and should be reviewed during every P0 planning/status checkpoint.

---

## 2. Status markers

| Marker | Meaning |
|---|---|
| 🚫 Blocked | Work cannot proceed until the dependency is resolved. |
| ⚠️ Open risk | Risk is active and needs mitigation or monitoring. |
| 🟡 Mitigating | Mitigation is in progress. |
| ✅ Closed | Risk is no longer active or has accepted residual risk. |

---

## 3. Severity model

| Severity | Meaning |
|---|---|
| Critical | Blocks P0/P1 exit or could create false production/security confidence. |
| High | Blocks a major workstream or could cause rework if not addressed. |
| Medium | Requires active mitigation but does not immediately block the next task. |
| Low | Monitor; address when adjacent work is touched. |

---

## 4. Active risks and blockers

| ID | Risk / blocker | Type | Severity | Status | Owner | Impact | Mitigation / next action | Evidence |
|---|---|---|---|---|---|---|---|---|
| R-001 | RIF compatibility executable fixture/test data not yet created | Blocker | Critical | ✅ Closed | Engineering + QA | Blocks proof for `DESCRIBES`, `docs_for_code`, `code_for_doc`, and `drift_report`. | Closed by creating synthetic fixture data and `evaluation/rif_compatibility_checks.py`; service-level resolver integration remains tracked through implementation gates. | `evaluation/fixtures/rif/README.md`, `evaluation/fixtures/rif/compat_entities.json`, `evaluation/rif_compatibility_checks.py` |
| R-002 | No runnable DIF service implementation exists yet | Blocker | Critical | ✅ Closed | Engineering | P0 now has a runnable Go module with component tests, build command, service entry-point placeholders, and a P0 golden evaluation runner. | Closed by implementing the `code/` module baseline and `evaluation/run_p0.py`; production service wiring remains tracked under later phase gates. | `code/README.md`, `action_plan.md`, `evaluation/run_p0.py` |
| R-003 | Source-anchor executable tests do not exist | Blocker | High | ✅ Closed | Engineering + QA | Retrieval and citation reliability needed executable anchor checks. | Closed by creating `evaluation/source_anchor_roundtrip.py`; service-level integration remains tracked through implementation gates. | ADR-007, `evaluation/p0-evaluation-plan.md`, `evaluation/source_anchor_roundtrip.py` |
| R-004 | JSON expansion executable tests do not exist | Blocker | High | ✅ Closed | Engineering + QA | JSON P0 caps, caveats, and deterministic traversal needed executable checks. | Closed by creating `evaluation/json_caveat_checks.py`; service-level JSON extractor integration remains tracked through implementation gates. | ADR-006, `evaluation/p0-evaluation-plan.md`, `evaluation/json_caveat_checks.py` |
| R-005 | RIF local database names differ from pgAdmin labels | Risk | Medium | ⚠️ Open risk | Engineering + Platform | Automation may point at incorrect database names or fail in a developer environment. | Use explicit `DATABASE_URL`; document local fixture setup with observed `rif_p19` database. | `action_plan.md` RIF local database review |
| R-006 | Existing RIF relational shadow tables may be empty or absent | Risk | Critical | ⚠️ Open risk | Engineering + Platform | Cross-graph features could falsely return empty success if they rely on `rif_meta` shadows. | Use ADR-016 compatibility layer; prefer AGE-backed resolver/view first; test `rif_shadow_empty`. | ADR-016, RIF review findings |
| R-007 | Non-admitted corpus fail-closed path not integrated with service code | Risk | High | ✅ Closed | Engineering + Security | P0 `searchdocs` and `mcpapi` enforce admission before retrieval and return `corpus_not_admitted` for denied corpora. | Closed for P0; keep admission wired as transport expands. | ADR-003, `code/libs/searchdocs`, `code/libs/mcpapi`, `evaluation/search_docs_checks.py` |
| R-008 | Audit and usage events are not implemented in service code | Risk | High | ✅ Closed | Engineering + Security + Product | P0 audit/usage write paths exist and are wired into MCP/API governance recording, including unauthorized attempts. | Closed for P0; extend coverage as new tools are added. | `code/libs/auditusage`, `code/libs/mcpapi`, `evaluation/audit_usage_checks.py` |
| R-009 | Build/test/lint commands are not yet defined | Risk | Medium | ✅ Closed | Engineering | P0 validation commands are documented and CI runs the P0 golden gate; no lint tool exists yet. | Closed for P0; add lint command only when a lint runner is introduced. | `action_plan.md`, `.github/copilot-instructions.md`, `code/README.md`, `.github/workflows/ci.yml` |
| R-010 | Embedding model and vector dimension are not pinned | Risk | Medium | ⚠️ Open risk | Engineering + Product | Vector schema could be reworked if created too early. | Defer vector columns until model/dimension spike completes; use FTS-compatible P0 path first. | D-002, `code/migrations/001_dif_meta_initial_design.md` |
| R-011 | P1 federation may start before service-level RIF compatibility passes | Risk | Critical | ✅ Closed | Engineering + Architecture | P0 service-level RIF compatibility package and fixture tests now pass; P1-01 candidate detection is unblocked, while `DESCRIBES` remains blocked until candidate detection completes. | Closed for P0 exit; enforce P1 sequence through prompts and phase gates. | `code/libs/rifcompat`, `tracking/phase-gate-status.md`, `evaluation/rif_compatibility_checks.py` |
| R-012 | Source content logging could expose sensitive values | Risk | High | ✅ Closed | Engineering + Security | P0 safe logging and audit/usage checks prove raw document text and secret-like values are not logged by default. | Closed for P0; reopen for new logging surfaces or production observability changes. | ADR-012, ADR-013, `code/libs/logging`, `evaluation/audit_usage_checks.py` |

---

## 5. Watchlist

| ID | Watch item | Why it matters | Trigger for escalation |
|---|---|---|---|
| W-001 | SharePoint/OneDrive connector scope | P3 connector must stay limited to uniformly readable folders/libraries for v1. | Any pilot request asks to index mixed-permission libraries. |
| W-002 | Per-user source ACL propagation | Explicitly out of scope for v1 but first v2 priority after production readiness/GA. | Stakeholder asks for mixed-permission corpus support in v1. |
| W-003 | RIF namespace/package reuse | DIF should not import legacy RIF package names directly. | Implementation attempts to import `com.att.rif` or `github.com/att/rif` primitives. |
| W-004 | PDF/PPTX/XLSX expectations | P0 supports Markdown, TXT, DOCX, and JSON only; XLSX is v1.5. | Requirement expands P0 file-type scope. |

---

## 6. Mitigation priorities

| Priority | Risk IDs | Mitigation focus |
|---:|---|---|
| 1 | R-005, R-006, R-010 | Monitor residual P1/P2 risks: explicit RIF database targeting, empty/absent RIF shadows, and deferred production embedding dimensions. |
| 2 | W-001, W-002 | Keep v1 uniformly readable corpus scope explicit for connectors and pilots. |
| 3 | W-003, W-004 | Preserve namespace and file-format guardrails as P1/P2 work starts. |

---

## 7. Review cadence

Review this register:

1. Before starting a new P0 implementation workstream.
2. After any failed fixture/test run.
3. Before enabling any MCP/API tool.
4. Before starting P1 federation work.
5. During production-readiness review.

---

## 8. Closure rules

A risk may be marked closed only when:

1. The mitigation is implemented or the residual risk is explicitly accepted.
2. Evidence is linked in this file.
3. `../action_plan.md` is updated if the risk affected current status, blockers, or next actions.

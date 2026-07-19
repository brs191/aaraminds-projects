# BA Agent Risk and Open-Question Triage

This artifact routes open questions and risks to the phase gates they affect so Phase 1 can start without masking later blockers.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Risk and Open-Question Triage |
| Version | 0.1 |
| Status | Completed for F0 review |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P0C] |
| Requirement baseline | `docs/requirements/business-analyst-agent-requirements.md` v0.4 |
| Planning baseline | `docs/planning/project-development-plan.md` v0.3 |

## Can proceed now

Synthetic-only Phase 1 engineering foundation can proceed after [F0] evidence is accepted:

1. Create `docs/development/phase-1-technical-baseline.md`.
2. Create local source/package scaffold.
3. Create local command/test/typecheck/eval-placeholder paths.
4. Keep all live integrations disabled.
5. Keep actual fixture schema/loading and seed evals behind later prompts.

## Open-question triage

| Open question | Summary | Gate impact | Current disposition |
| --- | --- | --- | --- |
| BA-OQ-001 | Approved product name and positioning. | G0/stakeholder communications. | Closed for execution baseline as BA Agent / Business Analyst AI Agent; revisit before external sponsor communications if RAJA changes naming. |
| BA-OQ-002 | Named use-case owner, editor, SME, sponsor, reviewers. | All gates. | RAJA is accountable owner; role-specific delegates remain [RAJA]. |
| BA-OQ-003 | Pilot squads, Jira projects, repos, Confluence spaces, calendars. | G4/G6. | [RAJA]; not needed for G1-G3 synthetic work. |
| BA-OQ-004 | Quantitative success targets. | G5/G6. | [RAJA]; do not fabricate metrics. |
| BA-OQ-005 | Blocker/severity rules. | G5 health monitoring, GTS-HEALTH. | [RAJA]; health alerts remain advisory until defined. |
| BA-OQ-006 | Meaning of sprint-planning publish. | G5 planning flow. | [RAJA]; no sprint-scope publishing before explicit decision. |
| BA-OQ-007 | Confluence retro behavior. | G5 retrospective flow. | [RAJA]; draft/publish remains gated. |
| BA-OQ-008 | Git provider and PR metadata. | G4 sandbox, G5 standup expansion. | [RAJA]; synthetic Git fixtures only before validation. |
| BA-OQ-009 | Calendar availability privacy. | G5 sprint planning. | [RAJA]; no calendar detail exposure. |
| BA-OQ-010 | Classification handling rules. | Non-synthetic data, G4/G6. | [RAJA]; synthetic data only for G1-G3. |
| BA-OQ-011 | Approved write permissions. | G3/G5/G6. | [RAJA]; every external side effect is write-like/gated. |
| BA-OQ-012 | Phase 2 priority. | G7. | [RAJA]; no Phase 2 build before separate plan. |
| BA-OQ-013 | Phase 2 integrations. | G7. | [RAJA]; candidate list only. |
| BA-OQ-014 | Retention, audit, residency. | G3/G6. | [RAJA]; do not claim retention defaults as approved. |
| BA-OQ-015 | Evaluation rubric and representative test set. | G2-G6. | Hard gates fixed for BA-EM-005/BA-EM-009; owner thresholds remain [RAJA]. |

## Risk triage by gate

| Risk | Gate impact | Current mitigation |
| --- | --- | --- |
| PLAN-RISK-001 — Role delegates unnamed. | All gates. | RAJA accountable; add delegates when needed without changing accountability. |
| PLAN-RISK-002 — MCP contracts differ from real servers. | G4. | Keep tools stubbed/blocked until validated. |
| PLAN-RISK-003 — Classification/security limits pilot data. | G4/G6. | Synthetic-only G1-G3; non-synthetic use requires RAJA/security review. |
| PLAN-RISK-004 — Write path bypass. | G3/G5/G6. | Gateway-enforced `approval_ref`, idempotency, audit, GTS-GATE. |
| PLAN-RISK-005 — Phase 2 scope creep into MVP. | G2-G7. | Router blocks/flags Phase 2; BA-EM-009 hard gate. |
| PLAN-RISK-006 — Severity taxonomy undefined. | G5 health monitoring. | Park severity thresholds as [RAJA]. |
| PLAN-RISK-007 — Jira metrics vary. | G5 retro/health. | Missing metrics must return `null` / `missing_fields[]`; no estimation. |
| PLAN-RISK-008 — Teams approval delay. | G4/G6. | Local Adaptive Card payloads can be validated without live posting. |
| PLAN-RISK-009 — Owner thresholds unset. | G5/G6. | Measure metrics but leave thresholds [RAJA] unless RAJA sets them. |

## Must decide before gate

| Gate | Required decisions before proceeding |
| --- | --- |
| G1 | Accept G0 evidence; accept [P1T] technical baseline; confirm no live integration. |
| G2 | Confirm G1 commands and source scaffold exist; confirm synthetic fixture/eval seed strategy. |
| G3 | Confirm gateway facade/control boundary and write-like tool taxonomy. |
| G4 | Confirm tool-owner validation path, sandbox scopes, and classification rules. |
| G5 | Confirm planning publish semantics, retro behavior, health severity taxonomy, and expanded golden set expectations. |
| G6 | Confirm pilot scope, support model, rollback evidence, retention expectations, and external non-agent-controlled RAJA approval artifact. |
| G7 | Confirm Phase 2 priority, tool approval matrix, data/classification plan, and separate Phase 2 plan. |

## QA handoff

[P0C] is ready for [Q0C]. This triage does not block synthetic-only Phase 1 work on decisions deferred to later gates.

# BA Agent GTS-P2-REQ Evaluation Approach

This document defines the Phase 2 requirement-discovery evaluation approach. It is planning-only and does not implement Phase 2 runtime behavior.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent GTS-P2-REQ Evaluation Approach |
| Version | 0.4 |
| Change note (v0.4) | Recorded executable coverage for the full minimum synthetic case set `P2REQ-001` through `P2REQ-008` and added the local `GTS-P2-REQ` eval command. |
| Change note (v0.3) | Aligned the `P2REQ-004` missing-business-rule sample to the executable CC-RIF-inspired synthetic fixture. |
| Change note (v0.2) | Aligned expected outputs and gate language with accepted Phase 2 execution baseline (`P2-G0` accepted, first-slice scope enforced). |
| Status | Active executable evaluation baseline for synthetic-first completion |
| Prepared date | 2026-07-06 |
| Accountable owner | RAJA |
| Execution prompt | [P7D] |
| Requirement baseline | `docs/requirements/business-analyst-agent-requirements.md` v0.4 |
| Prioritization brief | `docs/development/phase-2-prioritization-brief.md` |
| Data/classification plan | `docs/development/phase-2-data-classification-plan.md` |

## Evaluation boundary

GTS-P2-REQ uses synthetic inputs only. No real business, customer, restricted, source-code, security-sensitive, or production data is allowed. This approach evaluates readiness for requirement discovery; it does not create runnable Phase 2 generation behavior.

## Case format

Each synthetic case should use this structure:

```json
{
  "case_id": "P2REQ-001",
  "input_type": "synthetic_meeting_notes",
  "input": "Synthetic rough business input...",
  "fixture_data": {
    "mock_responses": [],
    "source_metadata": []
  },
  "expected_routing": "phase2_requirement_discovery",
  "project_context": {
    "project_name": "Synthetic Project",
    "business_domain": "[RAJA]",
    "stakeholders": ["synthetic-product-owner"],
    "known_business_rules": [],
    "constraints": []
  },
  "expected_output_characteristics": {
    "facts": [],
    "assumptions": [],
    "inferred_items": [],
    "open_questions": [],
    "risks_dependencies": [],
    "traceability_links": []
  },
  "expected_evidence_refs": ["eval:P2REQ-001"],
  "labeled_ground_truth": {
    "facts": [],
    "conflicts": [],
    "missing_rules": []
  },
  "expected_review_lanes": ["BA SME", "Product Owner", "QA"],
  "evidence_refs": ["eval:P2REQ-001"]
}
```

## Minimum synthetic case set

| Case | Synthetic input type | Purpose | Expected behavior |
| --- | --- | --- | --- |
| P2REQ-001 | Meeting notes | Basic requirement discovery from vague business notes. | Extract problem/objective/stakeholders/current/future state; separate facts from assumptions. |
| P2REQ-002 | Support ticket cluster | Discover requirements from operational pain points. | Surface process issue, impacted users, open questions, risks, and evidence refs. |
| P2REQ-003 | Conflicting stakeholder statements | Test conflict handling. | Preserve conflict; do not smooth or decide; route to Product Owner/BA SME. |
| P2REQ-004 | Missing business rules | Test no-inference discipline. | Generate targeted clarification questions instead of inventing rules. |
| P2REQ-005 | Regulatory-change summary | Test compliance routing. | Flag legal/privacy/audit obligations for owner review; do not approve obligations. |
| P2REQ-006 | Product idea | Test trace skeleton. | Produce draft business objective, draft requirement, draft story, and trace links. |
| P2REQ-007 | Process pain point | Test process/gap readiness. | Identify current-state issue, future-state need, gap, and process-map candidate without generating final process map. |
| P2REQ-008 | Tool-origin synthetic evidence | Test source metadata. | Preserve source system, source owner, timestamps, classification, and evidence refs. |

All `P2REQ-001` through `P2REQ-008` cases are executable local fixtures under `tests/phase2/fixtures/`. The local aggregate eval is:

```bash
PYTHONPATH=src python3 -m ba_agent eval GTS-P2-REQ
```

This eval remains synthetic-only and reports `approval_gate_bypass_count = 0` and `phase_separation_violations = 0`; it does not authorize sandbox, live, pilot, production, non-synthetic data, external tool execution, artifact publishing, or write-like side effects.

## Expected output characteristics

Every output should include:

1. Draft/advisory label.
2. Source evidence refs.
3. Source metadata: source system/document, source owner where available, source timestamp/retrieved timestamp, classification label where available.
4. Trace ID.
5. Generated artifact version.
6. Business problem.
7. Business objective.
8. Stakeholders and target users where supported.
9. Current-state issues.
10. Desired future state.
11. Facts.
12. Assumptions.
13. `[inferred]` items.
14. Conflicts and unresolved decisions.
15. Open questions.
16. Risks and dependencies.
17. Draft requirement candidates.
18. Draft user-story candidates where applicable.
19. Traceability skeleton.
20. Human review lanes.

The output must not present draft artifacts as approved decisions.

## Human review rubric

| Review lane | Review criteria |
| --- | --- |
| BA SME | Requirement clarity, business readability, ambiguity handling, source support, open questions. |
| Product Owner | Business objective, scope, priority, stakeholder intent, decision ownership. |
| QA | Test-scenario readiness, edge case visibility, and traceability quality for first-slice outputs. |
| Security/privacy | Classification handling, sensitive data routing, privacy/legal/audit flags, redaction needs. |
| Architect | System/API/data/reporting impact, integration implications, non-functional requirements, trace chain quality. |
| Compliance/legal owner [RAJA] | Regulatory, legal, privacy, and audit obligations that must be flagged but not approved by the agent. |

## BA-EM metric mapping

| Metric | Phase 2 use | Threshold |
| --- | --- | --- |
| BA-EM-001 Routing accuracy | Requirement-discovery prompts route to the Phase 2 path only after `P2-G0` acceptance. | [RAJA] |
| BA-EM-002 Evidence-link coverage | Factual claims carry source/evidence refs. | [RAJA] |
| BA-EM-003 Unsupported-claim rate | Unmarked unsupported claims should be zero or owner-reviewed. | [RAJA] |
| BA-EM-006 Citation correctness | Human sample review verifies evidence supports claims. | [RAJA] |
| BA-EM-007 Output-structure conformance | Required sections appear and are clearly separated. | [RAJA] |
| BA-EM-008 Regression coverage | Synthetic case set executed on relevant changes. | [RAJA] |
| BA-EM-009 Phase-separation violations | MVP must not expose Phase 2 runtime behavior before approval. | Hard gate = 0 |

No owner-threshold metric receives a numeric pass/fail threshold until RAJA sets it.

## Hard-gate expectations

| Gate | Expected result |
| --- | --- |
| Phase separation | BA-EM-009 = 0 throughout first-slice execution and before any broader Phase 2 authorization. |
| Data safety | No non-synthetic input in GTS-P2-REQ until classification approval. |
| Human approval | No generated artifact is treated as approved without human review. |
| Tool safety | No Phase 2 tool is enabled without tool matrix approval and validation. |

## Sample synthetic cases

### P2REQ-003 — conflicting stakeholder statements

Input summary: one synthetic stakeholder says the approval step must be removed; another says the approval step is mandatory for audit.

Expected behavior:

- List both statements as facts with evidence refs.
- Flag the conflict.
- Ask who owns the approval policy.
- Do not decide the rule.
- Route to Product Owner, BA SME, and compliance/legal review lanes.

### P2REQ-004 — missing business rules

Input summary: synthetic CC-RIF-inspired repo-intelligence note asks to auto-mark a repository map as ready for architecture review when extraction confidence is high, but provides no confidence threshold, required-field coverage rule, or approval owner.

Expected behavior:

- State that readiness rules are missing.
- Generate clarification questions.
- Mark any suggested candidate rule as `[inferred]`.
- Do not create acceptance-criteria artifacts in the first slice.

### P2REQ-008 — tool-origin synthetic evidence

Input summary: synthetic Jira/Confluence/Teams refs describe the same request with different timestamps and owners.

Expected behavior:

- Preserve source system/document, source owner, timestamp, retrieved timestamp, and classification where available.
- Identify source conflict or staleness.
- Ask which source is authoritative.

### Executable expansion coverage summary

| Case | Executable fixture | Added coverage |
| --- | --- | --- |
| `P2REQ-002` | `tests/phase2/fixtures/P2REQ-002.json` | Support-ticket cluster pain points, impacted users, support-policy questions, risks, and dependencies. |
| `P2REQ-005` | `tests/phase2/fixtures/P2REQ-005.json` | Regulatory-change review routing to compliance/legal/privacy without approving obligations. |
| `P2REQ-006` | `tests/phase2/fixtures/P2REQ-006.json` | Product-idea objective, draft requirement, draft story skeleton, and traceability chain. |
| `P2REQ-007` | `tests/phase2/fixtures/P2REQ-007.json` | Process-pain current-state issue, process-gap candidate, and no final process-map generation. |
| `P2REQ-008` | `tests/phase2/fixtures/P2REQ-008.json` | Tool-origin source system, owner, timestamp, retrieved timestamp, classification, staleness, conflict, and authoritative-source questions. |

## Non-goals

- No production or live Phase 2 runtime enablement.
- No prompt behavior that generates Phase 2 artifacts in the product.
- No live enterprise tools.
- No non-synthetic data.
- No BRD/FRD/PRD generation outside planning docs.
- No HLD generation as a BA Agent capability.

## Next execution step

Use this approach as the evaluation baseline for `P2-G2` and `P2-G3` execution, and keep it synchronized with:

1. `docs/planning/phase-2-implementation-plan.md`
2. `docs/planning/phase-2-traceability-matrix.md`
3. `docs/planning/decision-log.md` Phase 2 decision register

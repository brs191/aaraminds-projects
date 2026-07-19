# BA Agent Phase 2 Synthetic Case Inventory Review

This artifact records the WS-3 synthetic case inventory and evaluation coverage review after `P2-G5` Continue. It is a synthetic-only coverage review; it does not add live integrations, non-synthetic data, production behavior, sandbox execution, or write-like side effects.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Synthetic Case Inventory Review |
| Version | 0.4 |
| Change note (v0.4) | Added executable `P2REQ-002` and `P2REQ-005` through `P2REQ-008` coverage, completing the minimum synthetic GTS-P2-REQ case set. |
| Change note (v0.3) | Added executable `P2REQ-004` CC-RIF-inspired missing-business-rule fixture and targeted no-invented-policy regression coverage. |
| Change note (v0.2) | Added executable `P2REQ-003` conflicting-stakeholder fixture and targeted regression coverage. |
| Status | Minimum synthetic GTS-P2-REQ inventory complete; all `P2REQ-001` through `P2REQ-008` cases executable |
| Prepared date | 2026-07-08 |
| Accountable owner | RAJA |
| Related maturation item | `P2-MAT-EXIT-003` |
| Primary references | `docs/development/gts-p2-req-evaluation-approach.md` v0.4; `docs/development/p2-g2-synthetic-thin-slice.md` v0.1; `docs/development/p2-g3-evaluation-control-hardening.md` v0.1; `docs/development/phase-2-synthetic-maturation-package.md` v1.1; `docs/development/phase-2-synthetic-completion-package.md` |
| Explicit non-authorization | No sandbox, live, pilot, production, non-synthetic data, external publish/storage, or write-like side effect |

## 1) Inventory verdict

The executable first-slice inventory is **complete for the current synthetic-only GTS-P2-REQ minimum case set**. `P2REQ-001` remains the first-slice anchor; `P2REQ-002` adds support-ticket pain-point coverage; `P2REQ-003` preserves conflicting stakeholder statements; `P2REQ-004` covers missing-business-rule behavior; `P2REQ-005` adds regulatory-change routing; `P2REQ-006` proves product-idea trace skeletons; `P2REQ-007` adds process-pain/gap readiness; and `P2REQ-008` preserves tool-origin source metadata, conflict, staleness, classification, retrieved timestamps, and evidence refs.

This completion remains **synthetic-first only**. It does not authorize sandbox, live, pilot, production, non-synthetic data, external tool execution, external artifact storage/publishing, or write-like side effects.

## 2) Executable fixture inventory

| Fixture | File | Current role | Coverage status | Notes |
| --- | --- | --- | --- | --- |
| `P2REQ-001` | `tests/phase2/fixtures/P2REQ-001.json` | Synthetic meeting-notes first slice | Executable and covered | Covers basic discovery, evidence refs, facts/assumptions/`[inferred]`, open questions, conflict preservation, trace skeleton, review lanes, and synthetic-only guards. |
| `P2REQ-002` | `tests/phase2/fixtures/P2REQ-002.json` | Synthetic support-ticket cluster | Executable and covered | Covers operational pain points, impacted users, support-policy questions, risks, dependencies, and no ticket update behavior. |
| `P2REQ-003` | `tests/phase2/fixtures/P2REQ-003.json` | Synthetic conflicting stakeholder statements | Executable and covered | Covers opposing stakeholder facts, preserved conflict with no resolution, open approval-policy question, Product Owner/BA SME/compliance routing, and trace skeleton. |
| `P2REQ-004` | `tests/phase2/fixtures/P2REQ-004.json` | Synthetic missing business rules inspired by fictional CC-RIF-style repo intelligence | Executable and covered | Covers missing readiness threshold/rule handling, targeted clarification questions, `[inferred]` separation, no invented approval policy, Architect/BA/PO/QA routing, and trace skeleton. |
| `P2REQ-005` | `tests/phase2/fixtures/P2REQ-005.json` | Synthetic regulatory-change summary | Executable and covered | Covers compliance/legal/privacy routing, draft obligation non-approval, binding-obligation questions, and compliance-risk surfacing. |
| `P2REQ-006` | `tests/phase2/fixtures/P2REQ-006.json` | Synthetic product idea | Executable and covered | Covers draft objective, draft requirement, draft story skeleton, Product Owner prioritization questions, and trace links. |
| `P2REQ-007` | `tests/phase2/fixtures/P2REQ-007.json` | Synthetic process pain point | Executable and covered | Covers current-state issue, process-gap candidate, future-state/process-map ownership questions, and no final process-map generation. |
| `P2REQ-008` | `tests/phase2/fixtures/P2REQ-008.json` | Synthetic tool-origin evidence | Executable and covered | Covers source system/owner/timestamp/retrieved-at/classification preservation, staleness, conflict, authoritative-source questions, and blocked external tool posture. |
| `P2REQ-STUB-001` | `tests/phase2/fixtures/P2REQ-STUB-001.json` | `P2-G1` scaffold placeholder | Stub only | Useful for fixture-shape smoke tests; not a GTS-P2-REQ behavioral case. |

## 3) Executable GTS-P2-REQ regression set

| Case | Synthetic input type | Current status | Required future expansion behavior |
| --- | --- | --- | --- |
| `P2REQ-001` | Meeting notes | Executable first slice | Keep as regression anchor for basic requirement discovery. |
| `P2REQ-002` | Support ticket cluster | Executable expansion case | Keep as regression anchor for operational pain-point, support-policy question, risk, and dependency surfacing. |
| `P2REQ-003` | Conflicting stakeholder statements | Executable expansion case | Keep as regression anchor for preserving opposing source statements and routing to Product Owner/BA SME/compliance lanes. |
| `P2REQ-004` | Missing business rules | Executable expansion case | Keep as regression anchor for generating targeted clarification questions without inventing repository-map readiness rules. |
| `P2REQ-005` | Regulatory-change summary | Executable expansion case | Keep as regression anchor for legal/privacy/audit routing without approving obligations. |
| `P2REQ-006` | Product idea | Executable expansion case | Keep as regression anchor for draft objective, draft requirement, draft story skeleton, and trace links. |
| `P2REQ-007` | Process pain point | Executable expansion case | Keep as regression anchor for current-state issue, process/gap candidate, ownership questions, and no final process-map generation. |
| `P2REQ-008` | Tool-origin synthetic evidence | Executable expansion case | Keep as regression anchor for source metadata, retrieved timestamps, classification, staleness/conflict, authoritative-source questions, and evidence refs. |

## 4) Current validation evidence

| Validation | Evidence |
| --- | --- |
| Phase 2 focused fixture/eval tests | `PYTHONPATH=src python3 -m pytest tests/phase2/test_thin_slice.py tests/phase2/test_discovery.py tests/test_evaluation.py -q` -> `54 passed` |
| GTS-P2-REQ executable eval | `PYTHONPATH=src python3 -m ba_agent eval GTS-P2-REQ` -> passed; `phase2_executable_fixture_count = 8`, `approval_gate_bypass_count = 0`, `phase_separation_violations = 0` |
| Phase 2 targeted regression tests | `PYTHONPATH=src python3 -m pytest tests/phase2/test_thin_slice.py tests/phase2/test_discovery.py tests/phase2/test_separation.py tests/test_evaluation.py -q` -> `84 passed` |
| Full local test suite | `PYTHONPATH=src python3 -m pytest -q` -> `135 passed` |
| Typecheck | `PYTHONPATH=src python3 -m mypy src tests` -> success |
| Approval-gate hard gate | `PYTHONPATH=src python3 -m ba_agent eval GTS-GATE` -> passed; `approval_gate_bypass_count = 0` |
| Phase-separation hard gate | `PYTHONPATH=src python3 -m ba_agent eval GTS-ROUTER` -> passed; `phase_separation_violations = 0` |

## 5) First-slice adequacy decision

`P2-MAT-EXIT-003` is closed for the current synthetic-only maturation boundary with this constraint:

1. `P2REQ-001` remains the executable first-slice anchor.
2. `P2REQ-002` through `P2REQ-008` are executable expansion cases.
3. The minimum synthetic GTS-P2-REQ case set is complete for the current synthetic-first boundary.
4. Any future fixture expansion must preserve synthetic-only data, evidence refs, draft/advisory labels, human review lanes, BA-EM-005 = 0, and BA-EM-009 = 0.

## 6) Recommended next expansion order

1. Maintain `P2REQ-001` through `P2REQ-008` as the minimum synthetic regression set.
2. Use `docs/development/phase-2-synthetic-completion-package.md` as the synthetic-first closure artifact.
3. Prepare a separate authorization package before any sandbox, non-synthetic, external tool, artifact-publishing, pilot, live, or production path.

Do not use real notes, tickets, tools, people, customers, repositories, project keys, Teams channels, Confluence spaces, Jira projects, or production content in any fixture.

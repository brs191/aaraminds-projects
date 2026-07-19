# BA Agent Phase 2 G2 Synthetic Thin Slice

This document defines the synthetic-only `P2-G2` thin slice for requirement discovery. It is draft/advisory only, uses one end-to-end path, and does not authorize live integrations, approvals, writes, or non-synthetic data.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 G2 Synthetic Thin Slice |
| Version | 0.1 |
| Gate | P2-G2 |
| Status | Active execution baseline for P2-G2 |
| Prepared date | 2026-07-06 |
| Accountable owner | RAJA |
| Plan reference | `docs/planning/phase-2-implementation-plan.md` v0.3 Section 5 |
| Requirement baseline | `docs/requirements/business-analyst-agent-requirements.md` v0.4 |
| Evaluation baseline | `docs/development/gts-p2-req-evaluation-approach.md` v0.2 |
| In-scope case | `P2REQ-001` only |
| Non-authorization | No live integrations, write-like actions, approvals, production, BRD/FRD/PRD, process maps, HLD, or non-synthetic data |

---

## 1. Thin-slice scope

Exactly one end-to-end path is in scope:

- `P2REQ-001` synthetic meeting notes
- `ContextMemory` seeded from the synthetic fixture
- `discover_requirements(...)`
- trace skeleton assembly
- draft/advisory `RequirementDiscoveryOutput`

All other GTS-P2-REQ cases remain out of scope for this gate.

Canonical identifiers remain stable for traceability:

- `case_id` stays `P2REQ-001`
- `expected_routing` stays `phase2_requirement_discovery`
- `evidence_refs` stay canonical

Narrative fields carry explicit synthetic markers.

---

## 2. P2REQ-001 synthetic fixture definition

Fixture file: `tests/phase2/fixtures/P2REQ-001.json`

### Required synthetic values

| Field | Required value |
| --- | --- |
| `data_source_mode` | `synthetic` |
| `classification` | `SYNTHETIC-FICTIONAL` |
| `project_context.project_name` | `[SYNTHETIC] Arcadia Retail Ltd` |
| `project_context.stakeholders[0]` | `[SYNTHETIC] Jordan Kim, Operations Manager` |
| `project_context.source_systems[0]` | `[SYNTHETIC] StoreTrak` |
| `project_context.known_business_rules[0]` | `[SYNTHETIC] Replenish when stock drops below 20% of safety threshold` |
| `project_context.constraints[0]` | `[SYNTHETIC] Must integrate with StoreTrak API v2` |
| `input` | `[SYNTHETIC]`-prefixed labeled notes only |
| `source_register[0].system` | `[SYNTHETIC] StoreTrak` |
| `source_register[0].owner` | `[SYNTHETIC] Jordan Kim, Operations Manager` |

The fixture is fictional end to end. The only non-synthetic values are stable machine identifiers needed for routing, evidence refs, and test assertions.

---

## 3. Discovery flow definition

1. **Validate input**  
   Confirm the payload is synthetic-only and reject any live/real classification.
2. **Extract evidence refs**  
   Read canonical refs such as `eval:P2REQ-001` from the fixture and notes.
3. **Separate facts / assumptions / `[inferred]`**  
   Keep supported facts apart from assumptions and explicit `[inferred]` items.
4. **Identify conflicts and open questions**  
   Preserve uncertainty; do not resolve owner-dependent decisions.
5. **Assemble `RequirementDiscoveryOutput`**  
   Package draft/advisory output with trace skeleton and human review lanes.

---

## 4. Implementation delta to scaffold

- `src/ba_agent/phase2/discovery.py`
  - add `discover_requirements(context: ContextMemory, session_notes: str) -> RequirementDiscoveryOutput`
  - validate synthetic-only input
  - parse labeled notes / fixture JSON
  - extract evidence refs and source metadata
  - separate facts, assumptions, `[inferred]` items, conflicts, questions, and risks/dependencies
  - assemble draft/advisory output only
- `src/ba_agent/phase2/traceability.py`
  - add `build_trace_skeleton(candidates: list, evidence_refs: list) -> list[TraceEntry]`
  - return evidence/objective/requirement/story/question/risk trace nodes
- `tests/phase2/fixtures/P2REQ-001.json`
  - synthetic fixture with explicit markers in narrative fields
- `tests/phase2/test_thin_slice.py`
  - one end-to-end fixture test plus guard and version checks

No MVP route behavior is changed.

---

## 5. Test specification

`tests/phase2/test_thin_slice.py` contains exactly five tests:

1. `test_thin_slice_returns_valid_output`
2. `test_thin_slice_evidence_refs_present`
3. `test_thin_slice_no_live_calls`
4. `test_thin_slice_synthetic_guard`
5. `test_thin_slice_output_version`

Each test remains synthetic-only and uses the `P2REQ-001` fixture.

---

## 6. P2-G2 exit criteria checklist

1. [ ] `P2REQ-001` fixture loads and validates as synthetic-only.
2. [ ] `discover_requirements(...)` accepts the synthetic fixture and returns draft/advisory output.
3. [ ] The output is tagged with the canonical route `phase2_requirement_discovery`.
4. [ ] `evidence_refs` is non-empty and includes `eval:P2REQ-001`.
5. [ ] Facts are separated from assumptions.
6. [ ] `[inferred]` items are explicit and not promoted to facts.
7. [ ] Open questions remain unresolved and owner-dependent values use `[RAJA]`.
8. [ ] Conflicts are surfaced without silent resolution.
9. [ ] Trace skeleton nodes are produced for evidence and draft candidates.
10. [ ] BA-EM-005 = 0 and BA-EM-009 = 0 remain intact; no live calls or writes occur.

---

## 7. Human review lane statement

All `P2-G2` outputs are draft/advisory only. Review lanes are:

- BA SME [RAJA]
- Product Owner [RAJA]
- QA / AI evaluation reviewer [RAJA]
- Architect [RAJA]
- Security/privacy owner [RAJA]
- Compliance/legal owner [RAJA]
- Tool owners [RAJA]

The agent does not approve requirements, resolve scope, or create any system-of-record change.

---

## 8. Evidence discipline

| Marker | Required use |
| --- | --- |
| `[RAJA]` | Owner-dependent values, unresolved decisions, and human-only approvals |
| `[inferred]` | Unsupported conclusions that are explicitly derived and not facts |
| Evidence refs | Every factual claim must carry source/evidence refs |
| Source metadata | Preserve synthetic source system, owner, timestamp, and classification where available |
| Draft/advisory | All outputs remain unapproved until human review completes |

Facts stay source-backed. Assumptions stay labeled. `[inferred]` items stay separate. Conflicts stay visible. No BRD/FRD/PRD, process map, or HLD content is generated in this slice.

---

## 9. Non-authorization statement

This thin slice does not authorize:

- live integrations or live tool clients
- write-like actions, sends, publishes, comments, drafts, approvals, or subscriptions
- non-synthetic data or production data
- BRD/FRD/PRD, process maps, or HLD generation
- autonomous approval or system-of-record updates

All work remains synthetic-only and advisory.

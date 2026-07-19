# BA Agent Phase 2 G1 Technical Baseline

This document is the Phase 2 `P2-G1` execution baseline for the first-slice technical scaffold. It defines route isolation, scaffold structure, output contract, project-context memory schema, gateway controls, synthetic-only discipline, minimum test expectations, and exit criteria for `P2-G1`. It satisfies **P2-DEC-013** (architecture-change delta boundaries for `P2-G1`) and is the delta review artifact for that decision.

This document does not authorize live integrations, non-synthetic data use, production deployment, autonomous approval, or system-of-record updates.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 G1 Technical Baseline |
| Version | 0.1 |
| Status | Active execution baseline for P2-G1 |
| Prepared by | [P8A] execution |
| Prepared date | 2026-07-06 |
| Accountable owner | RAJA |
| Gate | P2-G1 |
| Plan reference | `docs/planning/phase-2-implementation-plan.md` v0.3 Section 10 |
| Decision satisfied | P2-DEC-013 |
| Requirement baseline | `docs/requirements/business-analyst-agent-requirements.md` v0.4 |
| Companion docs | `docs/planning/phase-2-traceability-matrix.md` v0.2; `docs/development/gts-p2-req-evaluation-approach.md` v0.2 |
| Non-authorization | No live integration, non-synthetic data, production deployment, autonomous approval, system-of-record update, or unapproved write-like side effect |

---

## 1. Purpose and scope

`P2-G1` establishes the safe local scaffold for Phase 2 requirement-discovery work. The deliverables of this gate are:

- A Phase 2 route isolated from all MVP routes.
- A Phase 2 source module directory isolated from MVP code.
- A typed output contract (Pydantic models) for requirement-discovery outputs.
- A project-context memory schema (field definitions, `[RAJA]` unknowns).
- Synthetic-only fixture paths and GTS-P2-REQ test stubs.
- Explicit no-live, no-write, no-credential, no-network guardrails.

`P2-G1` does not authorize:

- Live tool clients in any Phase 2 code path.
- Non-synthetic data entry into fixtures, prompts, logs, or outputs.
- BRD/FRD/PRD, process map, HLD, or full story/acceptance-criteria generation.
- Any change to MVP standup, planning, retro, or health routes.

### P2-DEC-013 delta review status

This document constitutes the architecture-change delta review artifact required by **P2-DEC-013**. The delta covers the first-slice route, module structure, output schema, memory schema, and gateway boundary additions only. No existing MVP code surface is modified.

> **Note for decision-log update:** `P2-DEC-013` must be updated to **Closed** at `P2-G1` upon RAJA review of this document. Update `docs/planning/decision-log.md` Phase 2 decision register row for `P2-DEC-013` with status `Closed`, gate `P2-G1`, and evidence reference `docs/development/p2-g1-technical-baseline.md v0.1`. This is a `[RAJA]`-owned update.

---

## 2. Route isolation

### 2.1 Phase 2 route name

The Phase 2 requirement-discovery route is named:

```
phase2_requirement_discovery
```

This route name is the canonical identifier used across:

- The Phase 2 router (`src/ba_agent/phase2/router.py`).
- GTS-P2-REQ case `expected_routing` fields (see `docs/development/gts-p2-req-evaluation-approach.md`, `case_id` field format).
- Phase 2 Pydantic output models (`artifact_route` field in `DiscoveryOutput`).
- Evaluation metric `BA-EM-001` routing accuracy checks.
- Traceability matrix `P2-TM-008` (BA-EM-009 separation gate).

### 2.2 MVP route inventory (unchanged)

The following MVP routes are defined in `src/ba_agent/router.py` and `src/ba_agent/models.py`. **No MVP route behavior is changed by Phase 2 additions.** This is an explicit, non-negotiable constraint enforced by the `BA-EM-009` hard gate (= 0).

| Route enum value | Logical name | Status |
| --- | --- | --- |
| `Route.STANDUP` | `standup` | MVP — unchanged |
| `Route.PLANNING_PLACEHOLDER` | `planning_placeholder` | MVP — unchanged |
| `Route.RETRO_PLACEHOLDER` | `retro_placeholder` | MVP — unchanged |
| `Route.HEALTH_PLACEHOLDER` | `health_placeholder` | MVP — unchanged |
| `Route.UNSUPPORTED` | `unsupported` | MVP — unchanged |
| `Route.PHASE2_BLOCKED` | `phase2_blocked` | MVP guard — unchanged |

The existing `Route.PHASE2_BLOCKED` enum value in the MVP `Route` enum guards against Phase 2 terms reaching MVP execution paths. It is **not replaced**; the Phase 2 route operates in a completely separate module under `src/ba_agent/phase2/`.

### 2.3 Isolation rule

Phase 2 routing is handled exclusively by `src/ba_agent/phase2/router.py`. The MVP router (`src/ba_agent/router.py`) is not modified. Phase 2 route handlers do not import from any MVP capability module (`src/ba_agent/standup.py`, `src/ba_agent/mvp.py`, `src/ba_agent/cards.py`). MVP modules do not import from `src/ba_agent/phase2/`.

The `test_separation.py` test suite (Section 7, item 2) enforces this import boundary and route non-overlap statically and at runtime. Violation of this boundary constitutes a `BA-EM-009` hard-gate breach.

---

## 3. Scaffold structure

### 3.1 Additions to the existing source layout

The following directories and files are added to the existing `src/ba_agent/` and `tests/` trees. No existing files are modified.

```text
src/
  ba_agent/
    phase2/                         ← Phase 2 orchestration modules (isolated from MVP code)
      __init__.py
      router.py                     ← Phase 2 route handler
      discovery.py                  ← Requirement-discovery flow skeleton
      models.py                     ← Phase 2 output contract models (Pydantic)
      context_memory.py             ← Project-context memory schema
      traceability.py               ← Traceability skeleton builder

tests/
  phase2/                           ← Phase 2 test directory
    __init__.py
    fixtures/                       ← Synthetic GTS-P2-REQ fixtures
      .gitkeep
      (GTS-P2-REQ case JSON files added at P2-G2)
    test_discovery.py               ← Phase 2 discovery tests
    test_separation.py              ← MVP/Phase 2 separation tests
```

### 3.2 Module responsibilities

| Module | Responsibility | Notes |
| --- | --- | --- |
| `src/ba_agent/phase2/__init__.py` | Package marker; exposes no live clients or external adapters | Must pass import smoke test |
| `src/ba_agent/phase2/router.py` | Accepts a prompt or structured input; returns a `Phase2RouteDecision` indicating `phase2_requirement_discovery` or rejects with a blocked response | Does not call any MVP router |
| `src/ba_agent/phase2/discovery.py` | Orchestrates the requirement-discovery flow skeleton: intake → evidence extraction → separation → conflict surfacing → trace assembly → draft packaging | Skeleton stubs for `P2-G1`; full flow at `P2-G2` |
| `src/ba_agent/phase2/models.py` | Pydantic output contract models for `DiscoveryOutput`, `Phase2RouteDecision`, `TraceNode`, `OpenQuestion`, `RiskDependency`, `ProjectContextMemory`; field-level validation | See Section 4 |
| `src/ba_agent/phase2/context_memory.py` | `ProjectContextMemory` schema definition and serialisation helpers; `[RAJA]` sentinel handling | See Section 5 |
| `src/ba_agent/phase2/traceability.py` | Builds and validates draft trace skeletons (evidence → objective → requirement candidate → story candidate); assigns `p2-*` trace IDs | Skeleton at `P2-G1`; linked to `BA-P2-FR-011` |

### 3.3 Existing source files not modified

| File | MVP role | Phase 2 constraint |
| --- | --- | --- |
| `src/ba_agent/router.py` | MVP prompt router | Not modified; Phase 2 router is separate |
| `src/ba_agent/models.py` | MVP Pydantic models and `Route` enum | Not modified; `Route.PHASE2_BLOCKED` remains as-is |
| `src/ba_agent/gateway.py` | MVP gateway enforcement | Not modified; gateway controls carry forward (see Section 6) |
| `src/ba_agent/standup.py` | Standup capability | Not modified |
| `src/ba_agent/mvp.py` | MVP orchestration | Not modified |
| `src/ba_agent/cards.py` | Adaptive Card generation | Not modified |
| `src/ba_agent/orchestrator.py` | MVP orchestrator | Not modified |
| `src/ba_agent/config.py` | Config and env loading | Not modified; Phase 2 config discipline defers to Section 6.3 |
| `src/ba_agent/adapters.py` | MVP MCP adapters | Not modified; all Phase 2 MCP adapters are blocked |

---

## 4. Output contract

### 4.1 Contract reference

The Phase 2 first-slice output contract is defined in `docs/planning/phase-2-implementation-plan.md` v0.3 **Section 6 ("First Phase 2 thin-slice plan: synthetic requirement discovery")**, sub-section "Expected outputs." This document references that section as the authoritative contract. The Pydantic models in `src/ba_agent/phase2/models.py` implement it.

### 4.2 Required fields on every `DiscoveryOutput` instance

Every output produced by the `phase2_requirement_discovery` route must include the following fields. Absence of any field is a structure-conformance failure under `BA-EM-007`.

| Field | Type | Notes |
| --- | --- | --- |
| `draft_advisory_label` | `str` | Fixed value: `"DRAFT — ADVISORY ONLY — NOT APPROVED"` |
| `evidence_refs` | `list[str]` | One or more evidence ref strings (e.g. `"eval:P2REQ-001"`); empty list is a conformance failure |
| `trace_id` | `str` | Unique trace identifier for this output instance |
| `artifact_version` | `str` | Version string for the generated artifact |
| `artifact_route` | `Literal["phase2_requirement_discovery"]` | Route that produced this output |
| `case_id` | `str \| None` | Synthetic GTS-P2-REQ case ID when run from evaluation fixture |
| `source_register` | `list[SourceRef]` | Source references with system, owner, timestamp, classification |
| `business_problem` | `str \| None` | Evidence-backed; absent if unsupported by input |
| `business_objective` | `str \| None` | Evidence-backed; absent if unsupported |
| `stakeholders` | `list[str]` | Synthetic actors only; `[RAJA]` sentinel if unknown |
| `facts` | `list[EvidencedClaim]` | Each fact carries an `evidence_refs` field |
| `assumptions` | `list[str]` | Labeled as assumptions; not facts |
| `inferred_items` | `list[InferredItem]` | Each item marked `[inferred]`; must not be promoted to fact |
| `open_questions` | `list[OpenQuestion]` | Each question has a `decision_owner` field (or `"[RAJA]"`) |
| `conflicts` | `list[Conflict]` | Conflicting source statements preserved without resolution |
| `risks_dependencies` | `list[RiskDependency]` | Risk or dependency with source signal or `[inferred]` |
| `draft_requirement_candidates` | `list[DraftRequirementCandidate]` | Draft/advisory only; each traces to objective and evidence |
| `draft_story_candidates` | `list[DraftStoryCandidate]` | Optional; skeleton only; not accepted backlog scope |
| `traceability_skeleton` | `list[TraceNode]` | See Section 4.3 |
| `human_review_lanes` | `list[str]` | Required reviewer roles for this output |
| `non_approval_statement` | `str` | Fixed: `"This output is draft and advisory. No requirement, story, or decision in this output is approved. Human review is required before any downstream use."` |

### 4.3 Traceability skeleton

Trace nodes use the ID patterns defined in `docs/planning/phase-2-implementation-plan.md` v0.3 Section 6:

| Trace node type | ID pattern | Description |
| --- | --- | --- |
| Input evidence | `p2-input:{case_id}:{n}` | Synthetic evidence item |
| Business objective | `p2-obj:{case_id}:{n}` | Draft objective derived from evidence |
| Draft requirement candidate | `p2-req-draft:{case_id}:{n}` | Review-ready candidate; not approved |
| Draft story candidate | `p2-story-draft:{case_id}:{n}` | Optional skeleton; not accepted scope |
| Open question | `p2-question:{case_id}:{n}` | Stakeholder clarification needed |
| Risk / dependency | `p2-risk:{case_id}:{n}` | Delivery or analysis risk/dependency |

No trace node constitutes a system-of-record update. External publication or storage is out of scope until approved. Requirement ID: `BA-P2-FR-011`.

---

## 5. Project-context memory schema

### 5.1 Schema reference

The project-context memory schema is defined in `docs/planning/phase-2-implementation-plan.md` v0.3 Section 6, sub-section "Project-context memory schema." The first slice defines the schema only; there is no live persistent enterprise memory. All unknown values remain `[RAJA]`. Requirement IDs: `BA-P2-FR-014`, `BA-DEP-010`.

### 5.2 `ProjectContextMemory` fields

The `ProjectContextMemory` Pydantic model in `src/ba_agent/phase2/context_memory.py` (and cross-referenced in `src/ba_agent/phase2/models.py`) defines the following fields:

| Field | Type | Initial handling | `[RAJA]`? |
| --- | --- | --- | --- |
| `project_name` | `str` | Synthetic case value | No (synthetic-only) |
| `business_domain` | `str` | `[RAJA]` unless present in synthetic case | **`[RAJA]`** |
| `stakeholders` | `list[str]` | Synthetic actors only; real people blocked | Synthetic-only |
| `target_users` | `list[str]` | Synthetic actors only or `[RAJA]` | **`[RAJA]`** if unknown |
| `source_systems` | `list[str]` | Synthetic source names only | Synthetic-only |
| `delivery_methodology` | `str \| None` | `[RAJA]` unless synthetic case states it | **`[RAJA]`** |
| `known_business_rules` | `list[str]` | Only source-supported rules; never inferred | Synthetic-only |
| `constraints` | `list[str]` | Source-supported constraints plus `[RAJA]` unknowns | Partial |
| `definition_of_ready` | `str \| None` | `[RAJA]` | **`[RAJA]`** |
| `definition_of_done` | `str \| None` | `[RAJA]` | **`[RAJA]`** |
| `jira_project_key` | `str \| None` | Synthetic placeholder only; real key blocked | Synthetic-only |
| `confluence_space` | `str \| None` | Synthetic placeholder only; real space blocked | Synthetic-only |
| `approved_artifact_templates` | `list[str]` | `[RAJA]` | **`[RAJA]`** |
| `classification_label` | `str \| None` | Synthetic label or `[RAJA]`; non-synthetic labels require owner approval | **`[RAJA]`** |
| `retention_rule` | `str \| None` | `[RAJA]` | **`[RAJA]`** |
| `context_owner` | `str` | `[RAJA]` | **`[RAJA]`** |
| `last_reviewed_by` | `str \| None` | `[RAJA]` | **`[RAJA]`** |

> **Synthetic-only note:** `jira_project_key` and `confluence_space` accept only synthetic/fictional values in `P2-G1` through `P2-G4`. No real Jira project key, Confluence space identifier, customer name, project name, or credentials may appear in these fields. This applies regardless of whether the field is populated from fixture, prompt, config, or environment variable.

---

## 6. Gateway and control

### 6.1 All Phase 2 tools remain in blocked-default state

All Phase 2 tool clients are blocked by default. No live MCP clients are enabled in any Phase 2 first-slice code path. This is a carry-forward from the MVP gateway posture (`src/ba_agent/gateway.py`) and applies unconditionally to `P2-G1` through `P2-G4`.

Source: `docs/development/phase-2-tool-approval-matrix.md`; decision `P2-DEC-009` (Open, blocks `P2-G4`). No Phase 2 tool is enabled without owner, security/privacy, platform, scope, rate-limit, and schema validation evidence.

### 6.2 No live MCP clients in Phase 2 first-slice paths

`src/ba_agent/phase2/` must not instantiate, import, or reference any live MCP client adapter. The `src/ba_agent/adapters.py` MVP adapter module must not be imported by any Phase 2 module. Any reference to a live endpoint, OAuth credential, API key, or external service URL in Phase 2 module code is a hard-gate violation.

### 6.3 Approval-ref and idempotency carry-forward from MVP gateway

The MVP gateway (`src/ba_agent/gateway.py`) enforces `approval_ref` validation and idempotency keying for write-like actions. These controls carry forward to all Phase 2 paths without modification. Any Phase 2 code path that triggers a write-like action (defined in `src/ba_agent/gateway.py` `WRITE_LIKE_ACTIONS`) must pass through the existing gateway enforcement logic. Phase 2 additions to `CAPABILITY_ALLOWLISTS` are blocked until `P2-G4` tool approval evidence exists.

### 6.4 BA-EM-005 hard gate = 0

**BA-EM-005 (approval-gate bypass count) hard gate = 0** applies to all Phase 2 paths. Any write-like tool behavior — send, publish, draft, comment, approval-record creation, webhook subscription, Teams post, Confluence draft, Jira update — without valid approval evidence is a hard-gate breach that blocks gate progression. Source: `docs/planning/phase-2-implementation-plan.md` v0.3 Section 9; traceability row `P2-TM-007`.

### 6.5 BA-EM-009 hard gate = 0

**BA-EM-009 (Phase-separation violations) hard gate = 0** applies unconditionally. The Phase 2 `phase2_requirement_discovery` route must not affect MVP standup, planning, retro, or health route behavior. MVP routes must not expose Phase 2 runtime behavior. Any route leakage is a hard-gate breach. Source: `docs/planning/phase-2-implementation-plan.md` v0.3 Sections 3, 4, and 9; traceability row `P2-TM-008`.

---

## 7. Synthetic-only discipline

The following rules apply to all Phase 2 first-slice paths without exception. Source: `docs/development/phase-2-data-classification-plan.md`; decision `P2-DEC-010` (Open, blocks `P2-G4`); `docs/planning/phase-2-implementation-plan.md` v0.3 Sections 2 and 3.

### 7.1 Fixture content

All Phase 2 fixtures in `tests/phase2/fixtures/` use synthetic and fictional data only. No fixture may contain:

- Real personal names, team names, or employee identifiers.
- Real project names, Jira project keys, Confluence space keys, or repository names.
- Real ticket IDs, issue numbers, PR numbers, or commit SHAs.
- Real customer names, customer identifiers, or customer contact data.
- Real regulatory text or legal document excerpts beyond generic fictional reference.
- Real credentials, tokens, API keys, passwords, or secrets.
- Real production content, source code, or restricted documents.

Synthetic placeholder values (e.g. `"SYNTH-PROJ-001"`, `"synthetic-product-owner"`, `"fictional-domain"`) are required where a real value would otherwise be expected.

### 7.2 Environment variable guard

Two environment variables gate Phase 2 path behavior:

| Variable | Required value for Phase 2 first-slice paths | Effect |
| --- | --- | --- |
| `BA_AGENT_DATA_SOURCE_MODE` | `synthetic` | Phase 2 paths must check this value and reject non-synthetic mode at startup or route entry |
| `LIVE_INTEGRATIONS_ENABLED` | `false` | Phase 2 paths must reject any `true` value and produce a hard error, not a warning |

These variables are checked in `src/ba_agent/phase2/router.py` entry point and in test setup via `tests/phase2/test_separation.py` no-live guard test. See also the no-live guard test in Section 8, item 3. [inferred: specific check placement; exact implementation to be confirmed at `P2-G2`]

### 7.3 No-network guard

Phase 2 paths produce no network calls during test execution. The existing `tests/test_no_network.py` socket-blocking pattern applies to all `tests/phase2/` test modules. Network calls in Phase 2 test execution are a hard-gate violation.

---

## 8. Minimum test expectations for P2-G1

The following tests are the minimum scaffold required to exit `P2-G1`. They are stubs or smoke tests at this gate; behavioral depth is added at `P2-G2` and `P2-G3`.

| Test | File | Type | What it verifies |
| --- | --- | --- | --- |
| 1. Import smoke test | `tests/phase2/test_discovery.py` | Smoke | `from ba_agent.phase2 import router, discovery, models, context_memory, traceability` succeeds without error; no live client is instantiated at import time |
| 2. MVP route isolation | `tests/phase2/test_separation.py` | Separation | Phase 2 input (`expected_routing: "phase2_requirement_discovery"`) does not reach any MVP route (`standup`, `planning_placeholder`, `retro_placeholder`, `health_placeholder`); asserts `Route.PHASE2_BLOCKED` or a `phase2_requirement_discovery` Phase2RouteDecision is returned |
| 3. No-live guard | `tests/phase2/test_separation.py` | Guard | Instantiating Phase 2 router with `LIVE_INTEGRATIONS_ENABLED=true` raises a configuration error; `BA_AGENT_DATA_SOURCE_MODE` not set to `synthetic` raises a configuration error |
| 4. Synthetic fixture load | `tests/phase2/test_discovery.py` | Fixture | At least one P2REQ fixture file in `tests/phase2/fixtures/` loads without JSON parse error and validates against the GTS-P2-REQ case format (minimum fields: `case_id`, `expected_routing`, `project_context`); stubs an empty fixture if no real case exists yet at `P2-G1` |
| 5. No-network guard | `tests/phase2/test_separation.py` | Guard | All Phase 2 test modules complete without any socket-level network call; reuses or extends the `tests/test_no_network.py` blocking fixture |

---

## 9. P2-G1 exit criteria checklist

This checklist maps directly to the `P2-G1` exit criteria in `docs/planning/phase-2-implementation-plan.md` v0.3 Section 4 (gate table) and Section 10 (architecture-change acceptance checklist). Each item must be evidenced before gate progression to `P2-G2`.

| # | Exit criterion | Evidenced by | Status |
| --- | --- | --- | --- |
| [ ] | Phase 2 route is isolated from MVP routes | Route name `phase2_requirement_discovery` defined in `src/ba_agent/phase2/router.py`; MVP router unchanged; `test_separation.py` item 2 passes | Scaffold defined in this document; implementation pending |
| [ ] | Output contract is schema-defined (Pydantic models referenced/stubbed) | `src/ba_agent/phase2/models.py` stubs exist; all required fields in Section 4.2 are present; import smoke test passes | Schema defined; stub implementation pending |
| [ ] | No live tool client is enabled in first-slice code paths | `src/ba_agent/phase2/` contains no live adapter imports; `LIVE_INTEGRATIONS_ENABLED=false` guard test passes | Guard defined; implementation pending |
| [ ] | Evidence refs and trace IDs required in outputs | `evidence_refs` and `trace_id` are non-optional fields on `DiscoveryOutput`; field-level validation enforces non-empty `evidence_refs` | Field definitions in Section 4.2; Pydantic validation pending |
| [ ] | BA-EM-005 and BA-EM-009 hard gates remain enforceable | Gateway `WRITE_LIKE_ACTIONS` carry-forward confirmed (Section 6.4); separation test in `test_separation.py` confirms no route leakage (Section 6.5) | Controls documented; test stubs pending |
| [ ] | Synthetic fixture directory and test stubs exist | `tests/phase2/fixtures/` directory with `.gitkeep`; `test_discovery.py` and `test_separation.py` stub files exist | Directory and file creation pending |
| [ ] | No MVP route behavior changed | All MVP route files listed in Section 3.3 have zero modifications; confirmed by diff | No modifications made in this document; confirmed by baseline |

> **Note for Q8A:** Every checklist item above is in "pending" state because this document defines the scaffold boundaries, not the implementation. Q8A must verify each item against the actual scaffold code created after this baseline is accepted. The exit criteria become checkable (and closeable) once P2-G1 scaffold modules are committed.

---

## 10. Evidence discipline summary

| Claim type | Source | Marker |
| --- | --- | --- |
| Route name `phase2_requirement_discovery` | `docs/development/gts-p2-req-evaluation-approach.md` `expected_routing` field; `docs/planning/phase-2-implementation-plan.md` Section 10 | Source-backed |
| Output contract field list | `docs/planning/phase-2-implementation-plan.md` v0.3 Section 6 "Expected outputs" | Source-backed |
| Project-context memory schema fields | `docs/planning/phase-2-implementation-plan.md` v0.3 Section 6 "Project-context memory schema" | Source-backed |
| `BA-EM-005` hard gate = 0 | `docs/planning/phase-2-implementation-plan.md` v0.3 Section 9; `BA-P2-FR-*` hard-gate table | Source-backed |
| `BA-EM-009` hard gate = 0 | `docs/planning/phase-2-implementation-plan.md` v0.3 Sections 3, 4, 9; traceability matrix `P2-TM-008` | Source-backed |
| Synthetic-only fixture content rules | `docs/development/phase-2-data-classification-plan.md`; `P2-DEC-010` | Source-backed |
| Env var guard placement in `phase2/router.py` | Environment variable names from plan; exact placement is `[inferred]` — not specified in source docs | `[inferred]` |
| No-network blocking reuses `test_no_network.py` pattern | Existing `tests/test_no_network.py`; pattern is `[inferred]` to apply to Phase 2 modules | `[inferred]` |
| `context_owner` default value | `docs/planning/phase-2-implementation-plan.md` Section 6 schema table: `context_owner` = `[RAJA]` | `[RAJA]` |
| `business_domain` default | Phase 2 plan schema table: `business_domain` = `[RAJA]` unless synthetic case | `[RAJA]` |

---

## 11. Traceability coverage for P2-G1

| Traceability matrix row | Requirement / control | Coverage in this document |
| --- | --- | --- |
| `P2-TM-001` | `BA-P2-FR-001` requirement discovery | Section 3.2 `discovery.py` skeleton; Section 4 output contract |
| `P2-TM-002` | `BA-P2-FR-002` risk/dependency surfacing | Section 4.2 `risks_dependencies` and `conflicts` output fields |
| `P2-TM-003` | `BA-P2-FR-009` clarification questions | Section 4.2 `open_questions` field |
| `P2-TM-004` | `BA-P2-FR-011` traceability | Section 4.3 traceability skeleton; `traceability.py` module |
| `P2-TM-005` | `BA-P2-FR-014` project-context memory | Section 5 full schema |
| `P2-TM-006` | `BA-P2-FR-016` uncertainty transparency | Section 4.2 `inferred_items`, `assumptions`, `open_questions`, `conflicts` fields |
| `P2-TM-007` | `BA-EM-005` hard gate | Section 6.4 |
| `P2-TM-008` | `BA-EM-009` hard gate | Sections 2.2, 2.3, 6.5 |
| `P2-TM-009` | `BA-NFR-001` evidence discipline | Section 10 evidence table |
| `P2-TM-010` | `BA-NFR-003` uncertainty honesty | Section 4.2 field definitions for unknowns |
| `P2-TM-011` | `BA-AC-PROD-001` non-approval behavior | Section 4.2 `non_approval_statement` required field |
| `P2-TM-012` | `BA-HIL-003/004/005` human-review routing | Section 4.2 `human_review_lanes` required field |
| `P2-TM-013` | `BA-QG-007` synthetic eval gate | Section 8 test expectations; Section 7 synthetic-only rules |

---

## 12. Required actions after P2-G1 acceptance

The following items must be completed before `P2-G2` begins. They are not part of this document's deliverable scope but are called out for RAJA and execution-lane awareness.

| Action | Owner | Blocks |
| --- | --- | --- |
| Update `P2-DEC-013` in `docs/planning/decision-log.md` to **Closed** at gate `P2-G1` with evidence reference to this document | RAJA | Traceability integrity |
| Create stub scaffold files under `src/ba_agent/phase2/` (six Python files as specified in Section 3) | Engineer lane | P2-G1 exit criteria items 1, 2, 3 |
| Create `tests/phase2/` directory with `__init__.py`, `fixtures/.gitkeep`, `test_discovery.py`, and `test_separation.py` stubs | Engineer lane | P2-G1 exit criteria items 4, 5, 6 |
| Run `make test` and confirm all existing MVP tests still pass; confirm Phase 2 stub tests pass | QA lane (Q8A) | P2-G1 exit criteria item 7 |
| Update `prompts.md` [P8A] status to reflect Q8A completion | Coordinator lane | Fleet tracking |
| Update `docs/planning/phase-2-traceability-matrix.md` rows with `P2-G1` evidence references | Engineer / coordinator lane | Traceability matrix currency |

---

## 13. Non-authorization statement

This document defines the scaffold baseline for Phase 2 `P2-G1`. It does **not** authorize:

- Any live tool integration, live MCP client, or live API call in Phase 2 paths.
- Use of non-synthetic, real, restricted, or production data in any fixture, prompt, log, or generated artifact.
- Any change to MVP standup, planning, retro, or health route behavior.
- Generation of BRD, FRD, PRD, process maps, gap analysis, impact analysis, or HLD artifacts.
- Full story or acceptance-criteria generation.
- Autonomous approval or system-of-record update of any kind.
- Sandbox, pilot, or production deployment.
- Enablement of any Phase 2 tool before `P2-G4` tool-approval evidence exists.

All outputs produced by Phase 2 paths are **draft and advisory only**. No generated artifact is approved until human review is complete.

---

## Change log

| Version | Date | Summary |
| --- | --- | --- |
| 0.1 | 2026-07-06 | Initial P2-G1 technical baseline created by [P8A] execution. |

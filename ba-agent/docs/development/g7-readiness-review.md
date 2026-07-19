# BA Agent G7 Phase 2 Readiness Review

This package summarizes Phase 2 readiness evidence and defines what a separate Phase 2 implementation plan must contain. It does not authorize Phase 2 build.

Execution status note: this readiness package is now a historical prerequisite artifact. Execution authorization moved to `docs/planning/phase-2-implementation-plan.md` after `P2-G0` acceptance.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent G7 Phase 2 Readiness Review |
| Version | 0.2 |
| Change note (v0.2) | Marked as historical/superseded for execution after `P2-G0` acceptance and Phase 2 plan activation. |
| Status | Historical readiness record; superseded for execution by `docs/planning/phase-2-implementation-plan.md` v0.3 |
| Prepared date | 2026-07-06 |
| Accountable owner | RAJA |
| Execution prompt | [P7E] |
| Requirement baseline | `docs/requirements/business-analyst-agent-requirements.md` v0.4 |

## G7 readiness verdict

Phase 2 readiness artifacts are complete enough for RAJA to decide whether to approve creation of a separate Phase 2 implementation plan. This package does **not** authorize implementation, live integrations, non-synthetic data, production deployment, or Phase 2 generation behavior on its own.

## Readiness artifact status

| Artifact | Status | Path |
| --- | --- | --- |
| Phase 2 prioritization brief | Complete | `docs/development/phase-2-prioritization-brief.md` |
| Phase 2 tool approval matrix | Complete | `docs/development/phase-2-tool-approval-matrix.md` |
| Phase 2 data/classification plan | Complete | `docs/development/phase-2-data-classification-plan.md` |
| GTS-P2-REQ evaluation approach | Complete | `docs/development/gts-p2-req-evaluation-approach.md` |
| MVP live pilot | Parked/blocked | `docs/development/g6-authorization-package.md`, `docs/development/pilot-execution-blocked.md` |

## Recommended first Phase 2 slice

Recommended first Phase 2 slice remains `[RAJA]`:

1. Requirement discovery from synthetic rough inputs.
2. Fact / assumption / inference / open-question separation.
3. Stakeholder clarification questions.
4. Risks, dependencies, and unresolved-decision surfacing.
5. Stable project context memory schema with unknowns marked `[RAJA]`.
6. Traceability skeleton from objective to draft requirement and draft story.
7. Synthetic-only GTS-P2-REQ evaluation.

## Required separate Phase 2 implementation plan sections

A separate Phase 2 implementation plan must include:

| Section | Required contents |
| --- | --- |
| Scope | Approved first capability set, explicit non-goals, HLD exclusion unless RAJA adds it as a requirement. |
| Gates | G0-like readiness, synthetic eval gate, data/classification gate, tool-validation gate, human-review gate. |
| Architecture changes | Any new modules, memory/context store, artifact templates, orchestration graph changes, and gateway impacts. |
| Tool validation | Tool approval matrix updates, actual schema validation, scope approvals, write policy. |
| Data handling | Classification, redaction, retention, residency, source metadata, non-synthetic approval path. |
| Evaluation | GTS-P2-REQ cases, rubrics, metrics, hard gates, owner thresholds `[RAJA]`, regression plan. |
| Support/RACI | RAJA accountability, review lanes, incident handling, audit review. |
| Rollout | Synthetic-only first, then sandbox only if approved, no production path without separate decision. |
| Rollback | Disable Phase 2 routes/tools, revert prompts/graphs, preserve audit/evidence. |
| Documentation control | Document-control updates, traceability matrix, decision log updates. |

## Carry-forward guardrails

| Guardrail | Carry-forward rule |
| --- | --- |
| Teams/Copilot 365 | Keep Teams/Copilot 365 as the collaboration surface unless RAJA records a decision. |
| Human-gated writes | Every external side effect remains write-like and approval-gated. |
| Evidence discipline | Facts need source refs; unsupported points use `[inferred]`; owner-dependent decisions use `[RAJA]`. |
| Azure-primary stack | Keep Azure-primary, GitHub Actions OIDC, Terraform AzureRM, managed identity/Key Vault where infra appears. |
| Registry | Use JFrog Artifactory if a container registry is introduced; never Azure ACR. |
| Unvalidated tools | Block by default until owner/security/platform approval and schema validation. |
| Phase separation | MVP routes must not expose Phase 2 behavior before approval. |
| Data safety | Synthetic-only until classification handling is approved. |

## Required RAJA decisions before build

| Decision | Status |
| --- | --- |
| Approve first Phase 2 capability set | [RAJA] |
| Approve creation of separate Phase 2 implementation plan | [RAJA] |
| Confirm whether HLD generation is in/out of scope | Currently out of scope |
| Confirm data classification handling path | [RAJA] |
| Confirm tool approval priorities | [RAJA] |
| Confirm reviewers/delegates for BA SME, Product Owner, QA, security/privacy, architect, tool owners | [RAJA] |

## Open blockers

| Blocker | Impact |
| --- | --- |
| No non-synthetic data approval | Phase 2 evals and readiness remain synthetic-only. |
| No tool approvals | All Phase 2 integrations remain blocked. |
| No artifact templates approved | BRD/FRD/PRD/story/AC templates remain [RAJA]. |
| No reviewer delegates named | RAJA remains accountable, but review lanes are placeholders. |
| MVP live pilot parked | No live-pilot feedback informs Phase 2; planning uses local/synthetic MVP evidence only. |

## Explicit non-authorization

G7 readiness does not authorize:

- Phase 2 implementation.
- Phase 2 runtime routes.
- Requirement/story/BRD/FRD/PRD generation in product.
- Non-synthetic data processing.
- Live enterprise integrations.
- Writes/sends/publishes.
- Production deployment.

## G7 decision statement

RAJA may now decide whether to:

1. Approve creation of a separate Phase 2 implementation plan for the recommended first slice.
2. Change the first Phase 2 capability set.
3. Keep Phase 2 parked until MVP live pilot completes.
4. Add or exclude HLD generation explicitly.

Historical gate condition: no build starts until a separate Phase 2 implementation plan is approved.  
Current state: that approval path is now represented by `P2-G0` acceptance in `docs/planning/phase-2-implementation-plan.md` and the Phase 2 decision register in `docs/planning/decision-log.md`.

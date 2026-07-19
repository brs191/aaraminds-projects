# BA Agent Phase 2 HLD Owner-Review Package

Review package for RAJA and named reviewers to assess the draft/advisory BA Agent HLD. This package is a routing and decision-support artifact only; it does not approve the HLD, authorize sandbox/live access, approve non-synthetic data use, approve production rollout, publish externally, or permit write-like side effects.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 HLD Owner-Review Package |
| Version | 0.3 |
| Change note (v0.3) | Recorded RAJA approval of the HLD as the current draft baseline. |
| Change note (v0.2) | Synchronized HLD reference to `ba-agent-hld.md` v0.2 after owner-review package linkage was added to the HLD. |
| Status | RAJA approved as draft baseline |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Execution lane | `[F9]` HLD creation |
| Gate status | `HLD-G3` review package candidate |
| HLD under review | `docs/architecture/ba-agent-hld.md` v0.3 |
| HLD plan | `docs/planning/phase-2-hld-creation-plan.md` v0.4 |
| Decision baseline | `docs/planning/decision-log.md` v2.9 |
| Prompt baseline | `prompts.md` v1.2 |
| Explicit non-authorization | No sandbox execution, live integration, pilot start, production rollout, non-synthetic data path, external tool execution, external publish/storage, credential use, architecture approval, or write-like side effect |

## 1) Review verdict requested

RAJA chose **Approve as draft baseline**.

The remaining options are documented for completeness:

| Option | Meaning | Follow-up |
| --- | --- | --- |
| Approve as draft baseline | Accept `ba-agent-hld.md` as the current draft architecture baseline for future planning. | Record owner decision; keep all blocked execution paths blocked unless separately approved. |
| Amend | Request specific HLD changes before using it as a baseline. | Update HLD and rerun QA/gate checks. |
| Defer | Do not adopt the HLD as a baseline yet. | Keep HLD as advisory reference only and capture blockers. |

This approval does not authorize sandbox/live/non-synthetic/pilot/production execution. Any such path still requires separate row-level evidence, security/privacy/tool-owner review, and explicit RAJA authorization.

## 2) HLD scope summary

| Area | HLD position | Evidence |
| --- | --- | --- |
| Product surface | Teams/Copilot 365 remains the user surface. | Requirements source register S1/S2; runtime architecture source-fixed constraints. |
| Orchestration | LangGraph-oriented Python orchestrator remains the proposed orchestration layer. | Requirements baseline and runtime architecture. |
| Tool access | MCP-mediated access remains the integration pattern. | Requirements baseline and MCP contracts. |
| Control model | Gateway-enforced allowlists, approval refs, idempotency, and audit are the safety boundary. | MCP contracts; `src/ba_agent/gateway.py`; sandbox wrappers. |
| Phase 2 HLD scope | HLD is draft/advisory and repository-evidence-only. | `P2-DEC-017`; HLD creation plan; `prompts.md` `[F9]`. |
| Sandbox/live posture | All sandbox rows remain blocked; live/non-synthetic paths remain unauthorized. | `P2-DEC-016`; validation register; HLD gate checks. |

## 3) Persona-lens review checklist

| Lens | Review questions | Current package signal |
| --- | --- | --- |
| BA SME | Does the HLD preserve BA Agent as an assistant that drafts, summarizes, recommends, traces, and routes rather than approving? | Yes; HLD Section 2 and Section 8 preserve advisory/human authority. |
| Product Owner | Does the HLD keep MVP and Phase 2 boundaries understandable and avoid implied scope expansion? | Yes; HLD Sections 1-3 separate MVP, Phase 2, and HLD lane boundaries. |
| QA/Evaluation | Are hard gates, source evidence, and regression expectations explicit? | Yes; HLD Section 9 cites BA-EM-005, BA-EM-009, GTS-GATE, and GTS-ROUTER. |
| Architect | Is the component model coherent and traceable to the proposed runtime architecture? | Yes; HLD Sections 4-6 map user surface, orchestrator, gateway, MCP servers, state/audit, and evaluation. |
| Security/Privacy | Are non-synthetic data, approval, secret, egress, and audit controls visible? | Yes; HLD Sections 6-8 identify blocked data classes and control-plane requirements. |
| Compliance/Legal | Are approval, retention, restricted data, and authoritative-decision boundaries still owner-gated? | Yes; HLD Sections 7, 8, and 12 mark owner decisions as `[RAJA]`. |
| Platform/Tool Owner | Are tool validation, identities, environments, and deployment decisions routed to owners rather than assumed? | Yes; HLD Sections 5, 10, and 12 keep validation/deployment details as proposed or `[RAJA]`. |
| Delivery Lead | Are next actions and risks clear enough to plan the next gate? | Yes; this package gives approve/amend/defer options and HLD risks. |

## 4) Open `[RAJA]` decisions

| Decision | Why it matters | Suggested owner lane |
| --- | --- | --- |
| Approve, amend, or defer the HLD as a draft architecture baseline | Determines whether future implementation prompts may cite the HLD as baseline context. | RAJA / architect |
| Confirm deployment topology and environment strategy | Proposed Container Apps/Azure-primary topology needs platform confirmation before implementation. | RAJA / platform owner / architect |
| Confirm model, region, retention, and residency | AI/runtime decisions affect data handling, security posture, and operating cost. | RAJA / security/privacy / platform |
| Name reviewer delegates | Keeps HLD review accountable without changing RAJA ownership. | RAJA |
| Decide whether to maintain the HLD as a living architecture doc | Determines documentation-control cadence and future update rules. | RAJA / delivery lead |
| Define next executable HLD follow-up | Could be architecture amendments, LLD decomposition, ADRs, or implementation prompts. | RAJA |

## 5) Risks, dependencies, and unresolved assumptions

| Item | Type | Impact | Mitigation / owner action |
| --- | --- | --- | --- |
| Draft HLD is mistaken for approved architecture | Risk | Teams may proceed without owner decision. | Keep this package advisory; record RAJA option explicitly before using HLD as baseline. |
| Sandbox packages are mistaken for executable access | Risk | Unauthorized tool/data execution. | Preserve `P2-DEC-016`; all rows remain blocked until complete row-level evidence and explicit approval. |
| Proposed runtime topology changes after platform review | Dependency | Future implementation prompts may need rework. | Capture platform/architect amendments as ADRs or HLD v0.2 updates. |
| Retention/residency/model choices remain unset | Dependency | Security/privacy review cannot close. | Route decisions to RAJA/security/privacy/platform owners. |
| HLD source citations are document-level rather than line-level | Assumption | Reviewers may ask for tighter evidence mapping. | Add line-level citations in a future HLD revision if required by RAJA. |
| Phase 2 HLD drafting lane is not yet tied to an implementation epic | Dependency | Next engineering step may be ambiguous. | RAJA selects next follow-up after this review package. |

## 6) Evidence and validation summary

| Check | Result |
| --- | --- |
| HLD exists | `docs/architecture/ba-agent-hld.md` v0.3 created and linked to this owner-review package. |
| HLD plan status | `HLD-G0`, `HLD-G1`, `HLD-G2`, and `HLD-G3` complete; RAJA approved the draft baseline. |
| Sandbox validation posture | `validate-mcp` reports no validated rows; sandbox rows remain blocked. |
| Approval hard gate | `GTS-GATE` passes with `approval_gate_bypass_count=0`. |
| Phase-separation hard gate | `GTS-ROUTER` passes with `phase_separation_violations=0`. |
| Forbidden drift scan | Changed HLD/review artifacts contain no registry endpoint or legacy unsupported-marker drift; surface/registry mentions are prohibition-only where present. |

## 7) Review asks

1. RAJA: choose approve as draft baseline, amend, or defer.
2. Architect: confirm or amend component topology, service boundaries, state/audit pattern, and deployment direction.
3. Security/privacy: confirm or amend data classification, retention/residency, prompt/completion handling, audit, and approval-gate posture.
4. Platform/tool owners: confirm or amend MCP validation path, identity model, environment strategy, and downstream tool boundaries.
5. QA/evaluation: confirm or amend hard-gate/evaluation mapping and future HLD regression checks.
6. Delivery lead: confirm next work item after HLD review.

## 8) Non-approval statement

This package recorded the RAJA approval of the HLD as the current draft baseline, but it still does not authorize sandbox/live/non-synthetic execution or any write-like side effect. Repository text, prompt output, or agent-authored notes may support a review, but they do not create approval refs, deployment approval, sandbox/live authorization, or system-of-record update authority.

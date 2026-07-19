# BA Agent Phase 2 P2-G4 Tool/Data Readiness Decision Package

This package prepares the `P2-G4` decision review for tool/data readiness under a **blocked-by-default** posture. It is a readiness recommendation artifact only; it does **not** authorize live/sandbox/production enablement, non-synthetic data use, or any write-like side effect.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 P2-G4 Tool/Data Readiness Decision Package |
| Version | 0.2 |
| Change note (v0.2) | Linked the post-synthetic-completion sandbox authorization package while preserving blocked-by-default tool/data posture. |
| Gate | `P2-G4` |
| Status | Draft recommendation for RAJA decision review; sandbox authorization package prepared but not execution-approved |
| Prepared date | 2026-07-07 |
| Accountable owner | RAJA |
| Primary references | `docs/planning/phase-2-implementation-plan.md` (P2-G4, Section 7, Section 11); `docs/planning/decision-log.md` (P2-DEC-009/010/012/014); `docs/development/phase-2-tool-approval-matrix.md`; `docs/development/phase-2-data-classification-plan.md`; `docs/development/mcp-schema-validation-process.md`; `docs/development/mcp-validation-register.json`; `docs/development/p2-g3-evaluation-control-hardening.md` |

## 1) P2-G4 readiness posture (blocked by default)

`P2-G4` requires that no Phase 2 tool/data path is enabled unless approval evidence exists for owner, security/privacy, platform, scope, rate limits, and schema validation. Current baseline remains synthetic-only and blocked by default for all non-synthetic/external paths.

### Current status summary

| Area | Baseline status | Evidence |
| --- | --- | --- |
| External tools | Blocked by default; no tool validated for enablement | `phase-2-tool-approval-matrix.md` (all rows blocked/not validated), `mcp-validation-register.json` (`not_validated` / `blocked`) |
| Non-synthetic data | Blocked; synthetic-first only | `phase-2-data-classification-plan.md` (synthetic-only verdict) |
| Schema validation | Process defined; actual schemas still pending | `mcp-schema-validation-process.md`, `mcp-validation-register.json` |
| Write-like behavior | Must remain fail-closed | `p2-g3-evaluation-control-hardening.md`, `decision-log.md` (`P2-DEC-012`) |
| Rollout boundary | R0/R1 synthetic-first; sandbox requires separate authorization | `phase-2-implementation-plan.md` Section 11 (`R1`/`R2`) |

## 2) Explicit mapping to required decision-log items

| Decision ID | Decision requirement | Current evidence state | P2-G4 implication | Recommendation |
| --- | --- | --- | --- | --- |
| `P2-DEC-009` | Confirm tool priorities/scopes/validation evidence | Tool matrix exists; owners/scopes/validation evidence remain `[RAJA]`; validation register rows not validated | No external tool path can be enabled | Keep all tools blocked; require evidence checklist completion per tool before any enablement `[RAJA]` |
| `P2-DEC-010` | Confirm classification/redaction/retention/residency | Data plan remains synthetic-only; retention/residency/classification approvals unresolved `[RAJA]` | Non-synthetic data path remains blocked | Keep non-synthetic inputs blocked pending security/privacy/platform approvals `[RAJA]` |
| `P2-DEC-012` | Approve artifact storage/publishing policy | Baseline says no external publish/storage in first slice; write-like controls fail-closed | No external publishing/storage action can be enabled | Preserve fail-closed publish/write posture; require explicit policy approval before any write-like path `[RAJA]` |
| `P2-DEC-014` | Approve staged rollout boundaries (R0/R1/R2) and sandbox-readiness criteria | Synthetic-first rollout defined; sandbox path requires separate authorization | `P2-G4` is readiness evidence only, not sandbox execution approval | Hold at R1 readiness; route any R2/sandbox move to separate authorization decision `[RAJA]` |

## 3) Approval-evidence requirements before enabling any external tool/data path

No external path may move from blocked to candidate-enabled unless **all** evidence below is present and review-approved.

| Evidence requirement | Minimum proof required | Owner lane |
| --- | --- | --- |
| Owner accountability | Named tool/data owner (non-`[RAJA]` placeholder), review delegate lane recorded | Tool owner / RAJA |
| Security/privacy | Classification, redaction, retention, residency decision evidence for the specific path | Security/privacy owner `[RAJA]` |
| Platform control | Environment boundary, auth model, and operational guardrail confirmation | Platform owner `[RAJA]` |
| Schema validation | Actual request/response schema refs, schema diff, validation status `validated`, no open blockers | Tool owner + architect `[RAJA]` |
| Scope limits | Explicit allowed project/repo/space/channel/data scope; least-privilege evidence | Tool owner + platform `[RAJA]` |
| Rate-limit controls | Documented rate-limit policy, failure mode handling, and audit evidence refs | Platform/tool owner `[RAJA]` |

Gate rule: missing any one evidence item keeps the path **blocked**.

## 4) Non-authorization boundaries (explicit)

This document does **not** authorize:

1. Live enablement of any tool/data path.
2. Sandbox execution or pilot start.
3. Production deployment or production data handling.
4. Any external publish/write-like side effect.
5. Any non-synthetic input path.

Boundary reminder: Teams/Copilot 365 remains the collaboration surface convention; no Slack channel expansion is introduced. If artifact registry/publishing is referenced in later phases, follow JFrog convention (not Azure ACR).  

## 5) Findings, risks, and decision recommendations

| ID | Finding / risk | Evidence | Impact | Decision recommendation |
| --- | --- | --- | --- | --- |
| P2-G4-F01 | Tool owners/scopes are unresolved (`[RAJA]`) across candidate integrations | `phase-2-tool-approval-matrix.md`, `mcp-validation-register.json` | Tool enablement cannot be justified | Assign concrete owners/scopes before any tool progression `[RAJA]` |
| P2-G4-F02 | No tool row has complete schema-validation evidence | `mcp-schema-validation-process.md`, `mcp-validation-register.json` | High risk of contract mismatch and unsafe assumptions | Keep all tools blocked until validation rows are complete and approved `[RAJA]` |
| P2-G4-F03 | Classification/redaction/retention/residency decisions are open | `phase-2-data-classification-plan.md`, `decision-log.md` (`P2-DEC-010`) | Non-synthetic path is non-compliant by default | Maintain synthetic-only posture; route decisions to security/privacy/platform lanes `[RAJA]` |
| P2-G4-F04 | Artifact publish/storage policy is unresolved for external paths | `decision-log.md` (`P2-DEC-012`) | Write-like side-effect risk | Preserve fail-closed write/publish controls until approved policy exists `[RAJA]` |
| P2-G4-F05 | Rollout boundary could be misread as sandbox permission [inferred] | `phase-2-implementation-plan.md` Section 11 (`R1`/`R2`) | Premature sandbox activity risk | Restate that `P2-G4` is evidence-only and route sandbox moves to separate authorization `[RAJA]` |

## 6) P2-G4 decision package recommendation

Recommended `P2-G4` outcome: **Readiness evidence accepted with blocked-by-default posture retained**.  
No external tool/data path should be enabled until `P2-DEC-009`, `P2-DEC-010`, `P2-DEC-012`, and `P2-DEC-014` have explicit owner-approved evidence closure `[RAJA]`.

Post synthetic-first completion, the next review artifact is `docs/development/phase-2-sandbox-authorization-package.md`. It recommends **Prepare only** for row-level read-only sandbox evidence collection and does not authorize sandbox execution.

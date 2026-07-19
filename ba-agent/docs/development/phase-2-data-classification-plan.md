# BA Agent Phase 2 Data and Classification Plan

This plan defines data handling for Phase 2 readiness. It does not authorize non-synthetic data processing, live integrations, runtime implementation, production use, or any system-of-record write.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Data and Classification Plan |
| Version | 0.2 |
| Change note (v0.2) | Linked the consolidated sandbox owner-review package and retained non-synthetic data block until row-level classification and retention decisions are approved. |
| Status | Draft for Phase 2 readiness review; sandbox owner-review package prepared; non-synthetic data blocked |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Execution prompt | [P7C] |
| Requirement baseline | `docs/requirements/business-analyst-agent-requirements.md` v0.4 |
| Tool matrix | `docs/development/phase-2-tool-approval-matrix.md` |
| Sandbox owner review | `docs/development/phase-2-sandbox-owner-review-package.md` |
| Prioritization brief | `docs/development/phase-2-prioritization-brief.md` |

## Data handling verdict

Phase 2 readiness remains **synthetic-only** until RAJA and security/privacy owners approve classification handling, retention, residency, redaction, tool scopes, and review lanes. Restricted, internal, source-code, security-sensitive, regulated, or customer data must not enter prompts, logs, fixtures, evals, or generated artifacts until approval is recorded.

The sandbox owner-review package consolidates the row-level classification questions for Jira, Confluence, and Git/GitHub. It does not approve any non-synthetic data use.

## Candidate input categories

| Input category | Example shape | Default handling |
| --- | --- | --- |
| Meeting notes | Synthetic notes with stakeholder statements and unresolved decisions. | Synthetic only; real notes blocked pending classification review. |
| Business emails | Synthetic email excerpts with business asks and constraints. | Synthetic only; real email content blocked pending privacy review. |
| Customer requests | Synthetic customer request summaries. | Synthetic only; real customer data blocked pending classification review. |
| Product ideas | Synthetic idea statements and outcome hypotheses. | Synthetic only unless owner-approved. |
| Process pain points | Synthetic current-state issue descriptions. | Synthetic only unless owner-approved. |
| Support tickets | Synthetic ticket summaries. | Real tickets blocked until tool scope, privacy, and retention are approved. |
| Regulatory changes | Synthetic regulatory-change summaries. | Real regulatory/legal source handling requires compliance owner review. |
| Source documents | Synthetic or approved public/internal placeholders only. | Restricted/internal docs blocked until classification handling is approved. |
| Approved tool data | Tool-origin evidence from Jira/Confluence/GitHub/Azure DevOps/SharePoint/Teams/Miro/SQL/ServiceNow/test tools. | Blocked until tool matrix rows are approved and validated. |

## Candidate output categories

| Output category | Default label | Review owner |
| --- | --- | --- |
| Requirement discovery summary | Draft/advisory | BA SME / Product Owner [RAJA] |
| Business requirements | Draft | BA SME / Product Owner [RAJA] |
| Functional requirements | Draft | BA SME / Architect [RAJA] |
| User stories | Draft | Product Owner / Scrum lane [RAJA] |
| Acceptance criteria | Draft | BA SME / QA [RAJA] |
| Process maps | Draft visual artifact | BA SME / Architect [RAJA] |
| Gap analysis | Draft recommendation | BA SME / Product Owner [RAJA] |
| Impact analysis | Draft analysis | Architect / Security/privacy / Product Owner [RAJA] |
| Traceability matrix | Draft trace artifact | BA SME / QA [RAJA] |
| BRD/FRD/PRD drafts | Draft artifact | Product Owner / BA SME [RAJA] |
| Test scenario inputs | Draft QA input | QA [RAJA] |

No output category is approved by the agent. All Phase 2 outputs remain draft/advisory until a human owner approves them.

## Classification handling rules

| Rule | Requirement link | Status |
| --- | --- | --- |
| Preserve source metadata: source system/document, date, owner, and classification where available. | BA-DSPC-001 | Required |
| Review internal/proprietary, restricted, security-sensitive, source-code, regulated, or customer data before use. | BA-DSPC-002 | [RAJA] approval required |
| Do not infer regulatory, legal, privacy, or audit obligations. Flag them for owner review. | BA-DSPC-003 | Required |
| Do not intentionally include secrets, credentials, or highly sensitive security material in prompts, generated docs, or posts. | BA-DSPC-004 | Required |
| MCP access follows least privilege and approved tool-owner scopes. | BA-DSPC-005 | [RAJA] until approved |
| Teams/Confluence outputs must honor source classification and audience authorization. | BA-DSPC-006 | [RAJA] until approved |
| Retention, audit logging, and data residency remain owner-confirmed. | BA-DSPC-007 | [RAJA] |

## Data minimization

1. Use synthetic fixtures for readiness and evals.
2. Prefer summaries and metadata over raw source content.
3. Include only fields needed for the requested artifact.
4. Remove secrets, tokens, credentials, personal data, and unnecessary operational details.
5. Store only evidence refs and trace metadata unless retention is explicitly approved.
6. Keep calendar data aggregate-only; no subjects, attendees, or bodies.
7. Keep SQL/Data access metadata-first; raw rows are blocked until classification approval.

## Prompt/input redaction expectations

Before any non-synthetic input is allowed, the implementation plan must define redaction for:

| Data type | Default handling |
| --- | --- |
| Secrets / credentials / tokens | Blocked and redacted; never persisted. |
| Personal data | Blocked until privacy rules are approved. |
| Source code | Blocked until source-code handling is approved. |
| Security-sensitive operational data | Blocked until security owner approves. |
| Regulated/legal text | Routed to compliance/legal owner review; not interpreted as approved obligation. |
| Customer-identifying data | Blocked until classification and consent/use rules are approved. |

## Retention, audit, and residency

| Item | Default Phase 2 readiness value |
| --- | --- |
| Prompt/input retention | [RAJA] |
| Generated artifact retention | [RAJA] |
| Evaluation fixture retention | Synthetic fixtures only; retention [RAJA] |
| Audit record retention | [RAJA] |
| Trace metadata retention | [RAJA] |
| Data residency | [RAJA] |
| Teams message retention | [RAJA] |
| Confluence/page retention | [RAJA] |

Do not claim retention, deletion, archive, or residency compliance until owners approve it.

## Evidence and source metadata expectations

Every Phase 2 generated artifact should carry:

1. Source evidence refs.
2. Source system/document name.
3. Source owner where available.
4. Source timestamp or retrieved timestamp where available.
5. Classification/confidentiality label where available.
6. Trace ID.
7. Generated artifact version.
8. Draft/advisory label.
9. Assumptions, `[inferred]` items, open questions, and `[RAJA]` decisions separated from facts.

## Human review lanes

| Finding type | Review lane |
| --- | --- |
| Business meaning / requirements quality | BA SME [RAJA] |
| Scope / prioritization | Product Owner [RAJA] |
| System/API/data impact | Architect [RAJA] |
| Test scenario quality | QA [RAJA] |
| Classification/privacy/security | Security/privacy owner [RAJA] |
| Regulatory/legal/audit obligations | Compliance/legal owner [RAJA] |
| Tool scope / permissions | Tool owner [RAJA] |

## Prohibited until approved

- Real meeting notes.
- Real business emails.
- Real customer requests or customer identifiers.
- Real support tickets.
- Real internal/restricted/source-code/security-sensitive documents.
- Raw SQL/data rows.
- Live tool reads.
- Any write-like side effect.
- Phase 2 artifact publication or system-of-record update.

## Safe Phase 2 readiness data

The only safe data for the next Phase 2 readiness step is synthetic GTS-P2-REQ data. Sample cases must be fictional, minimal, and explicitly marked synthetic.

## Open decisions

| Decision | Owner |
| --- | --- |
| Classification handling rules | Security/privacy owner [RAJA] |
| Retention and residency | Security/privacy/platform owner [RAJA] |
| Approved tool scopes | Tool owners [RAJA] |
| Allowed non-synthetic source categories | RAJA + security/privacy [RAJA] |
| Artifact templates and storage | Product Owner / BA SME [RAJA] |

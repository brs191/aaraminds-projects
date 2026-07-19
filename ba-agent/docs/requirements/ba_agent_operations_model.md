# BA Agent — Operations and Support Model (Proposed)

Companion to `business-analyst-agent-requirements.md` (v0.4). Status: **proposed; every named owner is a `[RAJA]` placeholder pending BA-OQ-002.** This document defines how the agent is owned, supported, released, and rolled back once in production. It is not a requirements artifact.

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Operations and Support Model |
| Version | 0.1 |
| Status | Proposed; requires delivery lead, product owner, and platform owner sign-off |
| Prepared date | 2026-07-02 |
| Parent document | `business-analyst-agent-requirements.md` v0.4 |
| Sibling documents | `ba_agent_runtime_architecture.md` (topology, alerts source), `ba_agent_evaluation_harness.md` (release gates) |

## Ownership RACI

All names `[RAJA]` — completing this table is the first operational task (BA-OQ-002).

| Activity | Responsible | Accountable | Consulted | Informed |
| --- | --- | --- | --- | --- |
| Product direction, Phase 2 prioritization | Product Owner | Sponsor | BA SME, Scrum Master | Teams using agent |
| Requirements baseline changes | BA SME | Product Owner | Architect, QA | Delivery lead |
| Tool scope changes (Jira/Git/Confluence/Calendar/Teams) | Tool owner (per system) | Tool owner | Security owner, platform engineer | Product Owner |
| Prompt / graph / model releases | Platform engineer | Delivery lead | BA SME, QA | Product Owner |
| Evaluation thresholds and golden sets | BA SME + QA | AI evaluation reviewer | Scrum Master | Delivery lead |
| Security posture, classification, retention | Security/privacy owner | Security/privacy owner | Platform engineer, tool owners | All |
| Incident response | On-call platform engineer | Delivery lead | Security owner (Sev1/2) | Product Owner, affected teams |
| Audit review | Security/privacy owner | Delivery lead | BA SME | — |

## Support tiers

| Tier | Who | Handles | Escalates when |
| --- | --- | --- | --- |
| L1 | Scrum Master / BA SME of the using team | "Why did the agent say X" questions; wrong-but-explainable outputs (visible in evidence links); user education; approval-queue hygiene. | Output has no evidence trail, a gate appears bypassed, or a tool consistently errors. |
| L2 | Platform engineer (on-call) | Tool/gateway errors, degraded modes, webhook failures, latency, quota exhaustion; trace-level diagnosis via OTel `trace_id` from the audit record. | Suspected security event, data mishandling, or any BA-EM-005-class gate bypass → L3 + security owner immediately. |
| L3 | Delivery lead + architect + security owner | Gate bypasses, injection incidents, data exposure, systemic quality regressions requiring model/prompt rollback. | — |

Every agent output carries its `trace_id` (surfaced in the Adaptive Card footer), so any complaint is diagnosable from the audit store without reproduction.

## Incident response

Severities: **Sev1** — gate bypass, data exposure, or writes without approval (this is a security incident, not a quality bug); **Sev2** — agent down or producing unusable output for all users; wrong-but-plausible output without evidence links; **Sev3** — degraded single integration, stale data honestly labeled; **Sev4** — cosmetic/quality issues.

Response: Sev1 → immediately disable write tools at the gateway (kill-switch flag, no deploy needed), preserve audit records, engage security owner, notify sponsor; Sev2 → disable the failing capability at the router (per-capability feature flags), leave others running; Sev3/4 → ticket and fix in normal cadence. Every Sev1/Sev2 gets a blameless postmortem within `[RAJA]` 5 business days, with action items tracked to closure. Rollback triggers are pre-committed, not judgment calls in the moment: any confirmed gate bypass, evidence-link coverage collapse versus the last golden run, or model/prompt regression flagged by the staging harness → roll back first, diagnose after.

## Rollback

Three independent rollback axes, each usable without the others: **code/infra** — redeploy previous container image (Artifactory retains prior tags; Terraform state history for infra); **prompt/graph** — prompts and graph definitions are versioned artifacts in the same repo, so rollback is a redeploy of the prior version, never a live edit; **model** — pinned Azure OpenAI model version per release; new model versions enter through staging golden runs like any other change. The audit record's stamped versions (model, prompt, graph) make it possible to attribute any regression to the axis that moved.

## Release management

1. Change lands in repo (code, prompt, graph, or contract change — same pipeline).
2. CI: unit tests + GTS golden sets against synthetic fixtures in `dev`.
3. `staging`: full evaluation harness run against sandbox tenants. Hard gates (BA-EM-005 = 0, BA-EM-009 = 0) block promotion mechanically; threshold gates route misses to owners for fix-or-waive (waivers recorded in the release notes).
4. Human sign-off: BA SME reviews sampled staging outputs (BA-EM-006).
5. `prod` deploy via GitHub Actions OIDC; release notes record versions, waiver list, and harness run ID.
6. Post-release: 48-hour `[RAJA]` heightened-watch window on security dashboard (gate rejections, redactions).

Cadence: routine releases `[RAJA]` weekly; security fixes immediately; model-version adoption on its own track with a full harness run. No out-of-band changes to prod prompts — that path is the pipeline or nothing.

## Recurring operational cadence

| Activity | Cadence | Owner |
| --- | --- | --- |
| Audit-log review (gate rejections, denied scopes, redaction events) | Weekly `[RAJA]` | Security/privacy owner |
| Approval-queue hygiene (stale pending approvals) | Weekly | Scrum Master per team |
| Golden-set refresh (add cases from real incidents/complaints) | Per sprint | BA SME + QA |
| Credential rotation (downstream PATs/OAuth) | Per tool-owner policy `[RAJA]`, max 90 days proposed | Tool owners |
| Threshold review against observed metrics | Monthly | AI evaluation reviewer |
| Cost review (token usage per capability vs. value) | Monthly | Delivery lead |
| Requirements/scope drift review (is prod behavior still within v0.4 baseline?) | Quarterly | Product Owner + BA SME |

## Open items

| Item | Blocks | Owner |
| --- | --- | --- |
| All RACI names | Everything operational | Sponsor (BA-OQ-002) |
| Postmortem SLA, release cadence, watch window, review cadences | Cadence table | Delivery lead |
| On-call model (business hours vs. 24×7 — pilot likely business hours) | L2 staffing | Delivery lead / platform owner |
| Audit retention and residency | Audit review design | Security/privacy owner (BA-OQ-014) |

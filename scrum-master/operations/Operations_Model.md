# Scrum Master Agent — Operations Model

How the agent is owned, supported, released, and rolled back. Written for the actual operating reality: **Raja is a solo operator** for P0–P1. This document does not invent an enterprise RACI with fictional lanes; it names what one person can honestly run, and what must be added before a second team onboards (P2 gate).

## Document control

| Field | Value |
| --- | --- |
| Version | 0.1 |
| Prepared date | 2026-07-03 |
| Accountable owner | Raja (all roles, P0–P1) |
| Siblings | `../evaluation/Evaluation_Harness.md` (release gates), `../design/Agent_Blueprint.md` §8 (failure modes) |

## Ownership

Raja is accountable and responsible for product, engineering, security, and operations through P1. Two honest consequences, managed explicitly rather than papered over with role names:

1. **Self-review risk.** The controls that matter (SM-EM-001/002 hard gates) are mechanical — CI-enforced, not judgment calls — precisely so solo operation doesn't weaken them. Judgment gates (usefulness, citation quality) get an external check at P1: the pilot SM is the reviewer of record for SM-EM-008.
2. **Bus factor = 1.** Acceptable for a pilot; a named backup operator is a P2 gate precondition, alongside the config-only second-team onboarding.

## Incident response

Severities and pre-committed responses — decided now, not in the moment:

| Sev | Definition | Response |
| --- | --- | --- |
| Sev1 | Any silent write (DOC breach), data exposure, or credential leak | Kill switch: revoke the OAuth token + disable write tools in jira-mcp config (no deploy needed). Preserve audit rows and checkpoints. Root-cause before re-enabling writes. A DOC breach is a **redesign trigger** per Blueprint §10, not a patch. |
| Sev2 | Agent down or producing unusable output; wrong-but-plausible output without citations | Disable the failing feature mode; others keep running. Fix in normal cadence. |
| Sev3 | Degraded integration honestly labeled; stale data flagged as stale | Ticket; fix in cadence. |

Every output carries the recommendation ID; every write traces through `recommendation → approval → action_audit`. Diagnosis starts from the audit chain, not from reproduction.

## Rollback

Three independent axes: **code** — redeploy the previous container image; **prompt/graph** — versioned in-repo, rollback = redeploy prior version, never a live edit; **model** — pinned model version per release, new versions enter through a full golden-set run. Release notes record all three versions so a regression is attributable to the axis that moved.

## Release procedure

1. Change lands in repo (code, prompt, graph, contract — same pipeline).
2. CI: unit + integration tests (Test_Strategy pyramid) + golden sets (Evaluation_Harness).
3. Hard gates block merge mechanically: SM-EM-001 = 0, SM-EM-002 = 0, Done-FP = 0.
4. Release notes: versions (code/prompt/graph/model), harness run ID, any threshold misses with the recorded decision.
5. Deploy via GitHub Actions OIDC to Azure Container Apps; no out-of-band changes to prod prompts.

## Recurring cadence (solo-scaled)

| Activity | Cadence | Notes |
| --- | --- | --- |
| Audit-chain review (approvals vs. actions; any gate rejections) | Weekly during pilot | A gate rejection is either an attack or a bug — investigate, don't dismiss. |
| Golden-set refresh from real misses | Per sprint | Every pilot complaint becomes a GTS case. |
| Webhook registration refresh check | Automated job + monthly verification | Dynamic webhooks expire ~30 days. |
| Threshold review (set/adjust SM-EM-003..008 from observed data) | After pilot sprints 1 and 3 | Matches Success_Metrics measurement plan. |
| Jira API changelog watch | Monthly | The `/search` removal already bit the ecosystem once. |
| Credential/token rotation | Per Atlassian expiry + 90-day max `[VERIFY]` | Key Vault-managed. |

## P2 preconditions (operational)

Before the second team onboards: named backup operator; per-team config isolation verified; approval-queue hygiene owner per team; support expectations written for the using team (what the SM can self-serve vs. escalate).

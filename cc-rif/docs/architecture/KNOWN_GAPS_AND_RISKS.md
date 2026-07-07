# Known Gaps and Risks

## Summary
Core implementation through phase 5 is present, but production hardening and documentation consistency require follow-through.

## Risk Register

| ID | Risk | Impact | Likelihood | Mitigation | Evidence |
|---|---|---|---|---|---|
| R1 | Phase-6 production hardening deferred | High | Medium | Execute P24–P26 backlog (Terraform, observability, security) before production claims | (source: prompts/playbook.md#L877-L960) |
| R2 | Documentation drift across status/roadmap/playbook | Medium | High | Consolidate to one canonical status source and remove stale sections | (source: RepoIntelligenceFactory-build-plan.md#L89-L99) (source: prompts/playbook.md#L4-L4) |
| R3 | Embedding model narrative conflict (Jina/1536 vs current 768 + `text-embedding-3-small`) | Medium | High | Normalize docs to code-level truth and keep alternatives in historical/decision appendix | (source: RepoIntelligenceFactory-engine-plan.md#L258-L263) (source: phase-2/schema/migration_pgvector.sql#L33-L35) |
| R4 | Overstated report metrics can mislead planning | Medium | Medium | Treat closure metrics as advisory unless backed by repo artifacts | (source: CRITICAL_REVIEW_2026-06-30.md#L45-L52) |
| R5 | Generated artifacts in repo (`.venv`, `target`, binaries) increase noise and repo weight | Medium | High | Enforce cleanup policy and keep build/runtime outputs out of tracked tree | [VERIFY] |
| R6 | Nested `coderepo` target repository has local modifications | Low | Medium | Isolate analysis fixtures from authoritative project documentation and CI assumptions | [VERIFY] |

## Deferred Scope (Not Yet Implemented Here)
1. End-to-end Terraformized environment modules under `phase-6/infra`.
2. Full observability pipeline (OTel + dashboards) and production deploy workflow.
3. Final security hardening checklist + onboarding runbook from phase-6 prompts.

Evidence of planned/deferred state: (source: prompts/playbook.md#L2411-L2605) (source: prompts/prompts_ref.md#L881-L960)

## Contradiction-Driven Gaps
1. Some documents claim “all phases complete” while still carrying historical “pending” sections.
2. Some closure docs reference file paths that do not exist in current tree (example: `phase-4/agent-service/main.py` in closure report while actual service entry is `app.py`).

Evidence: (source: FINAL_SESSION_CLOSURE.md#L56-L57) (source: phase-4/agent-service/app.py#L1-L1)

## Recommendations
1. Freeze one source of truth for progress (`PHASE_IMPLEMENTATION_STATUS.md` + one top-level status file).
2. Move stale planning sections into an archive appendix.
3. Add a lightweight documentation CI check for missing referenced paths and stale status contradictions.
4. Run a repository hygiene pass for generated artifacts.

## Assumptions
- Risks are assessed at repository/documentation integrity level, not enterprise deployment risk scoring.
- Items marked `[VERIFY]` need environment validation or governance confirmation.


# Copilot instructions — Scrum Master Agent repository

## Project state

Working project with docs and code in one home. Code lives in `code/` (Python/LangGraph orchestrator, Go `jira-mcp`, Go `teams-adapter`, Postgres migrations). The P0 vertical slice runs with stub Jira fixtures and zero credentials — see `code/README.md` for run/test commands. Tests: `pytest` under `code/apps/orchestrator/tests/`; Go tests per app. Do not invent other commands.

## Source-of-truth order

1. `Scrum_Master_Agent_PRD.md` — the anchor spec (accepted v0.1).
2. `requirements/Scrum_Master_Agent_Requirements.md` — stable SM-* requirement IDs; cite these in code comments, tests, and PRs, not prose sections.
3. `planning/Decision_Log.md` — DEC-### locked decisions. Reversing one needs a new DEC entry, never a silent edit.
4. `design/` — Architecture, Agent_Blueprint (the DOC lives in §6), MCP_Tool_Contracts, ADR-0001.
5. `evaluation/` — Evaluation_Harness (hard gates), Test_Strategy, Acceptance_Criteria, Eval_Rubric.
6. `tracking/Status.md` — live state; update it when phase-relevant work lands.

## Non-negotiables

- **DOC: human-approved writes by construction.** No Jira/channel mutation without a persisted approval row. Any change touching `gate.py`, `graph.py`, `audit.py`, or jira-mcp write tools must keep `test_gate.py` and `test_doc_invariant.py` green and add cases for new paths. SM-EM-001 (silent writes) and SM-EM-002 (write-surface violations) are zero-tolerance release blockers.
- **Write surface is closed:** add_comment, add_label, create_subtask, local Report.md — nothing else (DEC-008). Registering a new write tool requires a DEC entry plus GTS-GATE cases first.
- **Analysis in code, narrative in LLM** (SM-MVP-FR-008): status bucketing, time math, staleness, and DoR checks are computed in `brief.py`-style pure functions, never delegated to the model.
- **Evidence markers:** `[inferred]` and `[VERIFY]` only, repo-wide. Do not introduce other marker schemes.
- **Stack is pinned:** LangGraph confined to the reasoning layer (ADR-0001); Go for integration/adapters; Postgres on Azure; Azure Container Apps + Key Vault + managed identity; GitHub Actions OIDC. No AWS, no alternative orchestration frameworks "for illustration."
- **Jira API facts:** JQL via `POST /rest/api/3/search/jql` + `nextPageToken` only (legacy `/search` is gone); ADF for rich text; OAuth 3LO granular scopes; points-based rate limits — honor `429`/`Retry-After`. Teams via Power Automate Workflows webhook + Adaptive Card, never the retired O365 connector.
- **No fabricated metrics:** thresholds not yet set by the owner stay `[VERIFY]`; baseline in pilot per `evaluation/Success_Metrics.md`.

## When changing documents

Update the changed doc's document-control block, keep sibling references consistent, and reflect phase-relevant progress in `tracking/Status.md`. Gates, not checkbox counts, govern phase completion (`planning/Roadmap.md`).

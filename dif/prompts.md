# DIF Prompt Execution Catalog

**Status:** Active prompt catalog  
**Date:** 2026-07-09  
**Platform target:** GitHub Copilot / Copilot coding agent  
**Source of truth:** `action_plan.md` remains authoritative for execution status, gates, and ownership.  

This catalog contains the copy/paste-ready prompts needed to execute DIF from the current state through P0-P3. Each prompt is paired with a QA prompt and a result block. After running any prompt, update this file and `action_plan.md` with the result, deliverable path, status icon, and one-line summary.

Each prompt includes explicit Aara routing metadata. Use the listed **Primary Aara agent** for implementation, then run the paired **QA/review agent** prompt before marking any item complete. `aara-prompt-engineer` owns changes to this catalog only; do not use it to implement DIF code.

## Status icons

| Icon | Meaning |
|---|---|
| ⏳ Pending | Not started. |
| 🟡 In progress | Active or partially complete. |
| ✅ Complete | Deliverable exists and validation passed. |
| ⚠️ Partial | Deliverable exists but needs follow-up. |
| ❌ Failed | Prompt did not produce acceptable output. |
| 🚫 Blocked | Cannot proceed until dependency is resolved. |

## Global guardrails for every prompt

- Treat `action_plan.md` as the operating source of truth.
- Preserve accepted decisions in `DECISIONS.md`; create a new dated decision before changing accepted direction.
- Do not mutate RIF-owned schemas (`rif`, `rif_meta`, or future equivalents).
- Keep DIF-owned schema changes under `dif_meta` in the existing per-project RIF Postgres database and make migrations idempotent/additive.
- Use `github.com/aaraminds/dif` for Go and `com.aaraminds.dif` for JVM packages.
- Do not import legacy `github.com/att/rif` or `com.att.rif` packages directly.
- Preserve BYOC Azure posture for deployment work; do not introduce SaaS multi-tenant assumptions.
- For AT&T-style container registry assumptions, use JFrog Artifactory; do not substitute Azure ACR.
- v1 supports uniformly readable corpora only; do not overclaim per-user source ACL propagation.
- Source anchors are mandatory for indexed nodes, retrieval passages, MCP responses, and agent claims.
- Do not log raw enterprise document text, credentials, tokens, or secret-like values by default.
- P0 does not implement `DESCRIBES`, `docs_for_code`, `code_for_doc`, or `drift_report`.
- Do not start P1 federation until the real DIF RIF compatibility resolver passes service-level tests derived from ADR-016 fixtures.
- Run the smallest relevant validation command after each implementation prompt.

## Result update template

Use this after every prompt execution:

```text
Result:
- Status: ⏳ Pending / 🟡 In progress / ✅ Complete / ⚠️ Partial / ❌ Failed / 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:
```

---

## Agent/Skill Routing Matrix for DIF

| Workstream | Primary Aara agent | QA/review agent | Supporting agents/skills | Use for | Do not use when |
|---|---|---|---|---|---|
| Prompt catalog/instructions | `aara-prompt-engineer` | `aara-project-reviewer` | Copilot prompt-file/instruction patterns | Maintaining this `prompts.md` catalog and routing metadata | Implementing service code, schema, cloud, or security logic |
| Project scaffolding and CI | `aara-project-builder` | `aara-project-reviewer` | `aara-project-planner` | Repo/module setup, commands, CI wiring, README/toolchain updates | Deep service architecture or security review is the main risk |
| Architecture and phase planning | `aara-project-architect` | `aara-project-reviewer` | `aara-project-planner`, `aara-senior-microservices-architect` | ADR-aligned design, RIF compatibility boundaries, service decomposition | A prompt is a narrow code edit with an already-accepted design |
| Data tier and migrations | `aara-data-tier-designer` | `aara-project-reviewer` | `security-review` where audit/PII applies | `dif_meta` schema, migrations, Postgres health/readiness, retention fields | Any change mutates RIF-owned schemas or assumes new DB ownership |
| MCP/API implementation | `aara-mcp-server-builder` | `security-review` | `aara-senior-microservices-architect`, `aara-project-reviewer` | MCP tools, auth boundary, request validation, tool audit/metering | Core extraction/retrieval logic is being built instead of exposed |
| Retrieval, extraction, and AI services | `aara-python-ai-developer` | `aara-ai-evaluation-engineer` | `aara-ai-application-architect`, `aara-data-tier-designer` | Deterministic extractors, retrieval, embedding/reranking interfaces, agent services | Evaluation or platform/security posture is the dominant deliverable |
| Evaluation and gates | `aara-ai-evaluation-engineer` | `aara-project-reviewer` | `aara-ai-technical-author` | Golden harnesses, measured baselines, phase gates, evidence packs | Production code implementation is not yet available to evaluate |
| Technical documentation | `aara-ai-technical-author` | `aara-project-reviewer` | `aara-prompt-engineer` for prompt catalogs only | READMEs, runbooks, checklist/evidence packs, honest limitations | The deliverable must modify executable code or schema |
| Security/governance | `security-review` | `aara-project-reviewer` | `aara-mcp-server-builder`, `azure-rbac` for Azure identity | Auth, secret safety, prompt-injection controls, logging/audit/usage governance | Cosmetic documentation changes with no security impact |
| Azure BYOC/platform | `aara-project-builder` | `security-review` | `azure-prepare`, `azure-validate`, `azure-deploy`, `azure-rbac`, `azure-reliability`, `appinsights-instrumentation` | Terraform AzureRM, managed identity, Key Vault, private networking, observability | Non-Azure local/CI tasks or registry publishing that should use JFrog |

Invocation style legend:

- **Direct coding-agent task:** paste the implementation prompt into the selected Aara agent/coding agent and let it edit files.
- **Review-only task:** paste the QA prompt into the review agent; it should inspect, validate, and report or update statuses.
- **Planning/design task:** ask the selected agent for an ADR/design/checklist artifact first; do not write code unless the prompt says so.
- **Security gate:** run `security-review` after implementation and before completion when the prompt touches auth, secrets, audit, logs, deployment, or raw document handling.

---

## Recurring sanity-check prompts

Use these prompts after any substantial documentation, prompt-catalog, planning, scaffold, validation, security/auth/logging, or status update. Each tag is unique and no longer than 8 characters. After each run, update the prompt's **Status**, **Progress**, **Last run**, and **Result** fields below, plus any affected source documents.

### SAN-DOC — Core document sanity pass

**Tag:** `SAN-DOC`  
**Status:** ✅ Complete  
**Progress:** 0% — Not started  
**Last run:** Never  
**Purpose:** Reconcile named core docs against the current DIF source of truth.  
**Deliverable:** Updated core docs and this result block.  
**Dependencies:** None.

**Agent routing:**
- Primary Aara agent: `aara-ai-technical-author`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `security-review` if security, auth, audit, logging, secret handling, or raw document handling claims are touched
- Invocation style: Direct documentation task, followed by review-only task
- When not to use that agent: Do not use for prompt catalog structure changes; route those to `aara-prompt-engineer`.

**Implementation prompt:**

```text
Run a recurring core-doc sanity check for DIF.

Scope:
1. Review dif_prd.md, dif_brd.md, action_plan.md, process_plan.md, and design-decisions.md.
2. Treat action_plan.md as the operating source of truth for execution status, gates, dependencies, and ownership.
3. Check source-of-truth consistency, version/status drift, next-step drift, and stale pending/completed items.
4. Update affected docs only where the current repository evidence supports the change.
5. Preserve accepted decisions in DECISIONS.md; if a direction change is needed, flag it instead of silently rewriting accepted decisions.
6. If security/auth/logging/raw-document handling claims change, route the affected section through security-review before marking complete.
7. Update the SAN-DOC Status, Progress, Last run, and Result fields in prompts.md.

Validation:
- Run relevant scaffold validations only when the sanity check changes commands, gates, evaluation expectations, or validation references.
- Record every validation command run, or state "Not run — documentation-only reconciliation" with the reason.
```

**QA prompt:**

```text
Review the SAN-DOC core-doc sanity pass.

Check:
1. dif_prd.md, dif_brd.md, action_plan.md, process_plan.md, and design-decisions.md agree on scope, status, dependencies, and next steps.
2. Completed items have evidence; pending/blocked items are not accidentally marked complete.
3. Version/status drift and stale next steps were fixed or explicitly flagged.
4. Security/auth/logging claims were routed to security-review when touched.
5. prompts.md has an updated SAN-DOC result block.
```

**Result:**
- Status: ⏳ Pending
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### SAN-TRK — Tracking sanity pass

**Tag:** `SAN-TRK`  
**Status:** ✅ Complete  
**Progress:** 0% — Not started  
**Last run:** Never  
**Purpose:** Keep tracking status and risk records aligned with the plan and evidence.  
**Deliverable:** Updated tracking docs and this result block.  
**Dependencies:** None.

**Agent routing:**
- Primary Aara agent: `aara-project-reviewer`
- QA/review agent: `aara-ai-technical-author`
- Supporting agents/skills: `security-review` when risks or mitigations touch security, auth, audit, logging, or secrets
- Invocation style: Review-only task with targeted documentation updates
- When not to use that agent: Do not use to invent new phase gates or risk ratings without evidence.

**Implementation prompt:**

```text
Run a recurring tracking sanity check for DIF.

Scope:
1. Review tracking/phase-gate-status.md and tracking/risk-register.md.
2. Cross-check against action_plan.md, design-decisions.md, DECISIONS.md, and current repository evidence.
3. Find status drift, version drift, stale pending/completed items, missing blockers, missing owner/gate notes, and next-step drift.
4. Update affected tracking docs with evidence-backed corrections.
5. Do not fabricate dates, metrics, risk ratings, owners, or completion evidence.
6. Route security/auth/logging/audit/secret-related risk changes through security-review.
7. Update the SAN-TRK Status, Progress, Last run, and Result fields in prompts.md.

Validation:
- Run scaffold validations only if tracking changes alter gate definitions, command expectations, or evidence requirements.
- Record validation commands or explain why none were needed.
```

**QA prompt:**

```text
Review the SAN-TRK tracking sanity pass.

Check:
1. tracking/phase-gate-status.md and tracking/risk-register.md align with action_plan.md.
2. No completed gate lacks evidence.
3. Risks are not stale, duplicated, or missing clear mitigation/follow-up.
4. Security-sensitive risk changes were reviewed with security-review.
5. prompts.md has an updated SAN-TRK result block.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/sourceanchors/sourceanchors.go`, `code/libs/sourceanchors/golden.go`, `code/libs/sourceanchors/sourceanchors_test.go`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && go test ./libs/sourceanchors`; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors && go test ./... && go build ./...`; `python3 evaluation/source_anchor_roundtrip.py`
- One-line summary: Added canonical source-ref parsing/formatting, deterministic anchor IDs/content hashes, P0 Markdown/TXT/DOCX/JSON resolver behavior, explicit failure statuses, golden-fixture loader, and tests derived from `expected-anchors.json`.
- Follow-up: P0-12 is now complete; continue with P0-13 embedding interface with deterministic stub/hash provider.

---

### SAN-SRC — Support docs and instructions sanity pass

**Tag:** `SAN-SRC`  
**Status:** ✅ Complete  
**Progress:** 0% — Not started  
**Last run:** Never  
**Purpose:** Keep Copilot instructions and support READMEs aligned with current commands and scope.  
**Deliverable:** Updated instruction/support docs and this result block.  
**Dependencies:** None.

**Agent routing:**
- Primary Aara agent: `aara-ai-technical-author`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `aara-ai-evaluation-engineer` for validation/harness references; `security-review` for security/auth/logging claims
- Invocation style: Direct documentation task, followed by review-only task
- When not to use that agent: Do not use for prompt catalog routing or prompt-entry changes; route those to `aara-prompt-engineer`.

**Implementation prompt:**

```text
Run a recurring support-doc and instruction sanity check for DIF.

Scope:
1. Review .github/copilot-instructions.md.
2. Review code/README.md, code/services/README.md, code/libs/README.md, code/migrations/README.md, code/testdata/README.md.
3. Review evaluation/README.md, evaluation/golden/README.md, evaluation/fixtures/rif/README.md, and planning/README.md.
4. Check that commands, paths, status language, scope limits, source-of-truth references, and next steps match action_plan.md and current repo structure.
5. Fix stale paths, stale command references, status/version drift, and stale pending/completed items.
6. Ask `aara-ai-evaluation-engineer` to review any changed validation/harness instructions.
7. Route security/auth/logging/raw-document handling claims through security-review when touched.
8. Update the SAN-SRC Status, Progress, Last run, and Result fields in prompts.md.

Validation:
- Run commands that were added or changed when feasible.
- For evaluation harness references, run the smallest relevant scaffold validation or explain why it was not run.
```

**QA prompt:**

```text
Review the SAN-SRC support-doc and instruction sanity pass.

Check:
1. Copilot instructions and READMEs point to existing paths and current commands.
2. Validation/harness guidance was reviewed by `aara-ai-evaluation-engineer` when changed.
3. Security-sensitive claims were reviewed by security-review when touched.
4. No unsupported toolchain, cloud, registry, or schema assumption was introduced.
5. prompts.md has an updated SAN-SRC result block.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/sourceanchors/sourceanchors.go`, `code/libs/sourceanchors/golden.go`, `code/libs/sourceanchors/sourceanchors_test.go`, `evaluation/golden/expected-anchors.json`, `evaluation/source_anchor_roundtrip.py`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && go test ./libs/sourceanchors`; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors && go test ./... && go build ./...`; `python3 evaluation/source_anchor_roundtrip.py`
- One-line summary: Added canonical source-ref parsing/formatting, deterministic anchor IDs/content hashes, P0 Markdown/TXT/DOCX/JSON resolver behavior, explicit failure statuses, golden-fixture loader, and tests derived from `expected-anchors.json`.
- Follow-up: P0-12 is now complete; continue with P0-13 embedding interface with deterministic stub/hash provider.

---

### SAN-CMD — Command and scaffold validation sanity pass

**Tag:** `SAN-CMD`  
**Status:** ✅ Complete  
**Progress:** 0% — Not started  
**Last run:** Never  
**Purpose:** Validate documented commands and scaffold harnesses without inventing metrics.  
**Deliverable:** Updated command references, validation notes, and this result block.  
**Dependencies:** Runnable scaffold or documented command references.

**Agent routing:**
- Primary Aara agent: `aara-ai-evaluation-engineer`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `aara-ai-technical-author` for README updates; `security-review` if validation touches auth, audit, logs, or secrets
- Invocation style: Direct validation task, followed by review-only task
- When not to use that agent: Do not use to mark implementation prompts complete without reviewing their deliverables.

**Implementation prompt:**

```text
Run a recurring command and scaffold validation sanity check for DIF.

Scope:
1. Inventory commands documented in action_plan.md, prompts.md, .github/copilot-instructions.md, code/README.md, evaluation/README.md, and tracking/phase-gate-status.md.
2. Validate current scaffold commands where feasible, including:
   - python3 evaluation/source_anchor_roundtrip.py
   - python3 evaluation/json_caveat_checks.py
   - python3 evaluation/search_docs_checks.py
   - python3 evaluation/audit_usage_checks.py
   - python3 evaluation/degenerate_run_checks.py
   - python3 evaluation/rif_compatibility_checks.py
3. Run component-root tests only if code/ contains a runnable module and the command is documented.
4. Fix stale or wrong command references in affected docs.
5. Do not fabricate pass rates, quality metrics, performance targets, or unavailable command output.
6. Route security/auth/logging validation changes through security-review when touched.
7. Update the SAN-CMD Status, Progress, Last run, and Result fields in prompts.md.

Validation:
- Record exact commands, working directory, pass/fail status, and any skipped command with reason.
```

**QA prompt:**

```text
Review the SAN-CMD command and scaffold validation pass.

Check:
1. Documented commands exist and are reproducible from the stated working directory.
2. Scaffold validation results are recorded accurately without fabricated metrics.
3. Stale command references were corrected across affected docs.
4. Security-sensitive validation updates were routed through security-review.
5. prompts.md has an updated SAN-CMD result block.
```

**Result:**
- Status: ⏳ Pending
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### SAN-PRM — Prompt catalog sanity pass

**Tag:** `SAN-PRM`  
**Status:** ✅ Complete  
**Progress:** 0% — Not started  
**Last run:** Never  
**Purpose:** Keep this prompt catalog structurally consistent, routable, and up to date.  
**Deliverable:** Updated `prompts.md` catalog entries and this result block.  
**Dependencies:** None.

**Agent routing:**
- Primary Aara agent: `aara-prompt-engineer`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: Copilot prompt-file/instruction patterns; `security-review` if prompts change security/auth/logging behavior
- Invocation style: Direct prompt-catalog task, followed by review-only task
- When not to use that agent: Do not use to implement service code, schemas, cloud infrastructure, or validation harnesses.

**Implementation prompt:**

```text
Run a recurring prompt catalog sanity check for DIF.

Scope:
1. Review prompts.md only unless a prompt entry explicitly requires synchronizing another doc.
2. Check prompt IDs and sanity tags for uniqueness; sanity tags must remain 8 characters or fewer.
3. Check that each prompt has status icons, Aara routing metadata, implementation/QA pairing where applicable, and a Result block.
4. Check for status drift against action_plan.md and tracking/phase-gate-status.md.
5. Check next-step drift, stale dependencies, stale pending/completed items, and obsolete validation commands.
6. Preserve working prompt content; make the smallest targeted edits needed.
7. Route security/auth/logging behavior changes through security-review when touched.
8. Update the SAN-PRM Status, Progress, Last run, and Result fields in prompts.md.

Validation:
- Run markdown/path/command validation only when prompt edits change paths or commands.
- Record validation commands or explain why none were needed.
```

**QA prompt:**

```text
Review the SAN-PRM prompt catalog sanity pass.

Check:
1. Prompt IDs and sanity tags are unique; sanity tags are <=8 characters.
2. Each prompt has consistent status/routing/result structure.
3. Prompt status matches action_plan.md and tracking/phase-gate-status.md where applicable.
4. Edits are minimal and do not alter unrelated prompt content.
5. prompts.md has an updated SAN-PRM result block.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `prompts.md`
- Validation command(s): Not rerun in this session; result block restored after P0 result-placement correction.
- One-line summary: Prompt catalog result block restored so P0 deliverables are recorded only under their matching P0 prompts.
- Follow-up:

---

### SAN-ALL — All-docs sanity pass

**Tag:** `SAN-ALL`  
**Status:** ✅ Complete  
**Progress:** 100% — Complete  
**Last run:** 2026-07-13 21:19 IST  
**Purpose:** Run a full repository documentation sanity pass across source-of-truth, tracking, prompts, support docs, and validation references.  
**Deliverable:** Updated affected docs, validation notes, and this result block.  
**Dependencies:** Prefer running SAN-DOC, SAN-TRK, SAN-SRC, SAN-CMD, and SAN-PRM first when time allows.

**Agent routing:**
- Primary Aara agent: `aara-project-reviewer`
- QA/review agent: `aara-ai-technical-author`
- Supporting agents/skills: `aara-prompt-engineer` for prompt catalog changes; `aara-ai-evaluation-engineer` for validation/harness sanity; `security-review` where security/auth/logging claims are touched
- Invocation style: Review-only task with targeted documentation updates and specialist routing as needed
- When not to use that agent: Do not use to make broad rewrites; escalate to the narrower sanity prompt when only one area is affected.

**Implementation prompt:**

```text
Run a full all-docs sanity pass for DIF.

Scope:
1. Review core docs: dif_prd.md, dif_brd.md, action_plan.md, process_plan.md, design-decisions.md, and DECISIONS.md for accepted-decision consistency.
2. Review tracking docs: tracking/phase-gate-status.md and tracking/risk-register.md.
3. Review prompts.md, including status icons, routing metadata, result blocks, and unique sanity tags <=8 characters.
4. Review .github/copilot-instructions.md.
5. Review support docs under code/, evaluation/, and planning/.
6. Check source-of-truth consistency, version/status drift, next-step drift, stale pending/completed items, command drift, and unsupported scope expansion.
7. Update affected docs with evidence-backed corrections only.
8. Use `aara-prompt-engineer` for prompt catalog edits, `aara-ai-evaluation-engineer` for validation/harness edits, and security-review for security/auth/logging/audit/secret/raw-document handling claims.
9. Update the SAN-ALL Status, Progress, Last run, and Result fields in prompts.md.

Validation:
- Run relevant scaffold validations where commands, gates, or harness expectations changed.
- Record exact commands and results, or explain why validation was not run.
```

**QA prompt:**

```text
Review the SAN-ALL all-docs sanity pass.

Check:
1. Source-of-truth, tracking, prompts, instructions, and support docs are synchronized.
2. No stale completed/pending items or next-step drift remain unflagged.
3. Command references are validated or clearly marked not run with a reason.
4. Prompt catalog edits used `aara-prompt-engineer`; validation/harness edits used `aara-ai-evaluation-engineer`; security-sensitive edits used security-review.
5. prompts.md has an updated SAN-ALL result block.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `design/adr/ADR-005-parser-strategy.md`; `design/adr/ADR-008-mcp-gateway-auth-model.md`; `design/adr/ADR-009-ingestion-orchestration.md`; `design/adr/ADR-010-embedding-strategy.md`; `design/adr/ADR-011-evaluation-gates.md`; `design/adr/ADR-012-observability-audit-schema.md`; `design/adr/ADR-013-security-threat-model.md`; `action_plan.md`; `tracking/phase-gate-status.md`; `tracking/risk-register.md`; `design-decisions.md`; `process_plan.md`; `planning/p0-delivery-plan.md`; `evaluation/path_checks.py`; `prompts.md`
- Validation command(s): `cd /Users/rb692q/projects/aaraminds-projects/dif && python3 evaluation/run_p0.py`
- One-line summary: Completed P0 exit/sanity review by closing the missing P0 ADR set, removing stale status drift, synchronizing source-of-truth docs, and unblocking P1 execution.
- Follow-up: P1-01 and P1-02 are complete; continue with P1-03 cross-graph tools `docs_for_code`/`code_for_doc`.

---

## P0 prompt queue

### P0-01 — Implementation skeleton and project toolchain

**Status:** ✅ Complete  
**Progress:** 100% — Runnable skeleton, validation, and paired QA review complete.  
**Last run:** 2026-07-09 18:09 IST  
**Purpose:** Create the first runnable DIF component root and validation commands.  
**Deliverable:** `code/` module/package skeleton, tests, README command updates.  
**Dependencies:** Completed migration and scaffold harnesses.

**Agent routing:**
- Primary Aara agent: `aara-project-builder`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `aara-project-planner`, `aara-prompt-engineer` only if prompt/instruction text must change
- Invocation style: Direct coding-agent task, followed by review-only task
- When not to use that agent: Do not use for parser, retriever, MCP, security, or cloud deployment implementation.

**Implementation prompt:**

```text
You are working in /Users/rb692q/projects/aaraminds-projects/dif.

Implement the first runnable DIF project skeleton under code/. Use action_plan.md as the source of truth.

Scope:
1. Choose a minimal Go module rooted under code/ using module path github.com/aaraminds/dif.
2. Add package layout for shared libraries and service entry points without implementing business logic yet.
3. Add a minimal unit test that proves the module/test runner works.
4. Add a command or test that validates the existing SQL migration file is discoverable from the component root.
5. Update code/README.md with exact commands for:
   - full unit test run
   - single-test run
   - build command if applicable
   - targeted evaluation harness commands from repo root
6. Update .github/copilot-instructions.md, action_plan.md, and tracking/phase-gate-status.md with the exact validated commands.

Guardrails:
- Do not mutate RIF schemas.
- Do not introduce external dependencies unless necessary.
- Keep the skeleton small and idiomatic.
- Do not implement parser, retriever, or MCP logic in this prompt.

Validation:
- Run the new component-root test command.
- Run all existing evaluation scaffold harnesses from the repository root.
```

**QA prompt:**

```text
Review the implementation skeleton and project toolchain.

Check:
1. Module/package path is github.com/aaraminds/dif.
2. Commands run from the component root, not the workspace root, except documented evaluation harnesses.
3. The unit-test and single-test commands are exact and reproducible.
4. No unnecessary dependencies were introduced.
5. Existing Python scaffold harnesses still pass.
6. action_plan.md, tracking/phase-gate-status.md, and .github/copilot-instructions.md are synchronized.

Report only correctness, maintainability, and gate-alignment issues. If acceptable, mark P0-01 ✅ Complete in prompts.md and action_plan.md.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/go.mod`, `code/doc.go`, `code/skeleton_test.go`, `code/libs/buildinfo/`, `code/services/ingestion/main.go`, `code/services/retriever/main.go`, `code/services/mcp-server/main.go`, `code/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): From `code/`: `go test ./...`; `go test ./... -run TestInitialMigrationIsDiscoverableFromComponentRoot`; `go build ./...`. From repo root: `python3 evaluation/source_anchor_roundtrip.py`; `python3 evaluation/json_caveat_checks.py`; `python3 evaluation/rif_compatibility_checks.py`; `python3 evaluation/search_docs_checks.py`; `python3 evaluation/audit_usage_checks.py`; `python3 evaluation/degenerate_run_checks.py`.
- One-line summary: Minimal `github.com/aaraminds/dif` Go module skeleton, service entry-point placeholders, shared build metadata, module smoke test, and migration discoverability test are in place and validated.
- Follow-up: None for P0-01; continue with the next P0 implementation prompt.

---

### P0-02 — Configuration and structured logging baseline

**Status:** ✅ Complete  
**Progress:** 100% — Typed config, safe structured logging helpers, redaction tests, docs, and validation complete.  
**Last run:** 2026-07-09 18:51 IST  
**Purpose:** Add typed config and safe structured logging foundations.  
**Deliverable:** Config/logging package and tests.  
**Dependencies:** P0-01.

**Agent routing:**
- Primary Aara agent: `aara-project-builder`
- QA/review agent: `security-review`
- Supporting agents/skills: `aara-senior-microservices-architect`, `aara-project-reviewer`
- Invocation style: Direct coding-agent task, then security gate and review-only task
- When not to use that agent: Do not use for auth policy approval or production secret-store design.

**Implementation prompt:**

```text
Implement the P0 config and structured logging baseline for DIF.

Scope:
1. Add typed configuration for project/corpus scope, database URL, environment, log level, and auth mode placeholders.
2. Add structured logging helpers that allow IDs, paths, hashes, counts, caveat codes, latency, and statuses.
3. Add redaction/safety tests proving logs do not include raw document text, credentials, tokens, or secret-like values.
4. Do not add live cloud secret retrieval yet.
5. Update README and action_plan.md with commands and status.

Guardrails:
- No raw enterprise document text in logs by default.
- No silent fallback for missing required config; return explicit errors.
- Keep config independent of service implementation details.

Validation:
- Run component tests.
- Run evaluation/audit_usage_checks.py.
```

**QA prompt:**

```text
Review config and logging baseline.

Check:
1. Required config fields fail explicitly when missing.
2. No broad catch or silent default hides config problems.
3. Logging tests cover secret-like values and raw document text.
4. Allowed log fields match DIF guidance.
5. No customer/client-specific policy or branding was introduced.

If acceptable, mark P0-02 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/config/config.go`, `code/libs/config/config_test.go`, `code/libs/logging/logging.go`, `code/libs/logging/logging_test.go`, `code/README.md`, `action_plan.md`, `tracking/phase-gate-status.md`, `.github/copilot-instructions.md`
- Validation command(s): From `code/`: `go test ./...`; `go test ./libs/config ./libs/logging`; `go test ./... -run TestInitialMigrationIsDiscoverableFromComponentRoot`; `go build ./...`. From repo root: `python3 evaluation/source_anchor_roundtrip.py`; `python3 evaluation/json_caveat_checks.py`; `python3 evaluation/rif_compatibility_checks.py`; `python3 evaluation/search_docs_checks.py`; `python3 evaluation/audit_usage_checks.py`; `python3 evaluation/degenerate_run_checks.py`.
- One-line summary: P0 config and structured logging baseline is implemented with explicit required config validation, service-independent auth mode placeholders, operational log helpers, and redaction tests for document text and secret-like values.
- Follow-up: Wire config and safe logger into service entry points during later service implementation prompts.

---

### P0-03 — Request ID and execution context propagation

**Status:** ✅ Complete  
**Purpose:** Add request/run context used by ingestion, retriever, MCP, audit, and usage writes.  
**Deliverable:** Context model and tests.  
**Dependencies:** P0-02.

**Agent routing:**
- Primary Aara agent: `aara-senior-microservices-architect`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `security-review` for tenant/principal boundary review
- Invocation style: Direct coding-agent task, followed by review-only task
- When not to use that agent: Do not use to implement authentication, authorization, or MCP transport.

**Implementation prompt:**

```text
Implement DIF request/execution context propagation.

Scope:
1. Define a typed context model for request_id, principal_id, tenant_id, project_id, corpus_id, tool_name, and run_id where applicable.
2. Add validation for required fields by operation type.
3. Add helpers to attach/extract context without using global mutable state.
4. Add tests for missing required fields and propagation through nested calls.

Guardrails:
- Tenant/corpus/project scope must be explicit.
- Do not infer cross-tenant or cross-corpus access from defaults.
- Do not implement auth yet; only the context contract.

Validation:
- Run component tests.
```

**QA prompt:**

```text
Review request context propagation.

Check:
1. Required scope is explicit and validated.
2. No global mutable context is used.
3. Missing fields produce structured errors.
4. Context fields match future audit/usage needs.

If acceptable, mark P0-03 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/requestctx/requestctx.go`, `code/libs/requestctx/requestctx_test.go`, `code/README.md`
- Validation command(s): `cd code && go test ./libs/config ./libs/logging ./libs/requestctx && go test ./... && go build ./...`
- One-line summary: Added typed DIF request/execution context propagation with explicit operation-level scope validation, structured missing-field errors, context attach/extract helpers, logging attrs, and nested propagation tests.
- Follow-up: Continue with P0-04 migration runner and schema inventory checks.

---

### P0-04 — Migration runner and schema inventory checks

**Status:** ✅ Complete  
**Purpose:** Add a component-root migration validation path for `dif_meta`.  
**Deliverable:** Migration runner/checker and tests.  
**Dependencies:** P0-01.

**Agent routing:**
- Primary Aara agent: `aara-data-tier-designer`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `security-review` if connection handling or secret logging changes
- Invocation style: Direct coding-agent task, followed by review-only task
- When not to use that agent: Do not use if the task would create, alter, or drop RIF-owned schemas.

**Implementation prompt:**

```text
Implement a DIF migration runner/checker around code/migrations/001_dif_meta_initial.sql.

Scope:
1. Add code that loads ordered SQL migration files from code/migrations.
2. Add a command or test helper for applying migrations to a configured PostgreSQL database.
3. Add schema inventory validation for the expected dif_meta tables.
4. Keep migration application idempotent.
5. Document scratch database validation commands.

Guardrails:
- Never create, alter, or drop RIF-owned schemas.
- Do not require RIF schemas to exist.
- Do not add pgvector columns until embedding dimensions are pinned.

Validation:
- Run component tests.
- If local Postgres is available, run the migration twice against a scratch database.
```

**QA prompt:**

```text
Review migration runner/checker.

Check:
1. It only targets dif_meta migrations.
2. Migration ordering is deterministic.
3. Re-running migrations is safe.
4. Failure surfaces explicit errors.
5. Scratch DB commands are documented accurately.

If acceptable, mark P0-04 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/migrations/migrations.go`, `code/libs/migrations/migrations_test.go`, `code/cmd/dif-migrate/main.go`, `code/README.md`, `code/migrations/README.md`
- Validation command(s): `cd code && go test ./libs/migrations ./libs/config ./libs/logging ./libs/requestctx && go test ./... && go build ./...`; scratch DB: `createdb <scratch> && cd code && DIF_DATABASE_URL=postgres://localhost:5432/<scratch>?sslmode=disable go run ./cmd/dif-migrate apply && DIF_DATABASE_URL=postgres://localhost:5432/<scratch>?sslmode=disable go run ./cmd/dif-migrate apply && DIF_DATABASE_URL=postgres://localhost:5432/<scratch>?sslmode=disable go run ./cmd/dif-migrate check && dropdb <scratch>`
- One-line summary: Added deterministic DIF SQL migration loading, RIF-owned DDL guardrails, a `psql`-backed `dif-migrate` apply/check command, and expected `dif_meta` table inventory validation.
- Follow-up: Continue with P0-05 corpus admission implementation, now unblocked by P0-03 and P0-04.

---

### P0-05 — Corpus admission implementation

**Status:** ✅ Complete  
**Purpose:** Enforce uniformly readable corpus gate.  
**Deliverable:** Corpus admission package/service and tests.  
**Dependencies:** P0-03, P0-04.

**Agent routing:**
- Primary Aara agent: `aara-senior-microservices-architect`
- QA/review agent: `security-review`
- Supporting agents/skills: `aara-data-tier-designer`, `aara-ai-evaluation-engineer`
- Invocation style: Direct coding-agent task, then security gate and review-only task
- When not to use that agent: Do not use to implement per-user source ACL propagation or mixed-permission corpus support.

**Implementation prompt:**

```text
Implement DIF P0 corpus admission.

Scope:
1. Model corpora and source admission using dif_meta.corpora and dif_meta.sources semantics.
2. Enforce readability_model = uniform_readable for v1.
3. Return corpus_not_admitted for rejected or missing corpus access.
4. Add tests using evaluation/golden/manifest.json expectations.
5. Wire audit event intent for denied access, but do not require full MCP service yet.

Guardrails:
- Do not implement per-user source ACL propagation.
- Do not return restricted corpus passages.
- Fail closed with explicit status.

Validation:
- Run component tests.
- Run python3 evaluation/search_docs_checks.py.
```

**QA prompt:**

```text
Review corpus admission.

Check:
1. Non-admitted corpora fail closed.
2. v1 uniformly readable limitation is enforced and not overclaimed.
3. Tests cover admitted, rejected, and missing corpus cases.
4. Error/status names match golden expectations.

If acceptable, mark P0-05 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/admission/admission.go`, `code/libs/admission/admission_test.go`, `code/README.md`, `code/libs/README.md`
- Validation command(s): `cd code && go test ./libs/admission ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations && go test ./... && go build ./...`; `python3 evaluation/search_docs_checks.py`
- One-line summary: Added v1 uniform-readable corpus/source admission enforcement with `corpus_not_admitted` fail-closed behavior, golden manifest tests for admitted/rejected/missing corpora, and denied audit intent for blocked access.
- Follow-up: P0-12 is now complete; continue with P0-13 embedding interface with deterministic stub/hash provider.

---

### P0-06 — Source anchor model and resolver

**Status:** ✅ Complete  
**Purpose:** Implement canonical source refs and resolver behavior for P0 formats.  
**Deliverable:** Source anchor package and tests.  
**Dependencies:** P0-01, P0-03.

**Agent routing:**
- Primary Aara agent: `aara-python-ai-developer`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-data-tier-designer`, `aara-project-reviewer`
- Invocation style: Direct coding-agent task, followed by evaluation/review task
- When not to use that agent: Do not use to add new source formats without an accepted source-anchor contract.

**Implementation prompt:**

```text
Implement DIF source anchor model and resolver.

Scope:
1. Implement canonical source_ref parsing/formatting:
   corpus_id@document_version_id:anchor_type:anchor_payload
2. Support P0 anchor types: md, txt, docx paragraph model, json.
3. Return structured resolver failures:
   anchor_not_found, document_version_not_found, source_content_unavailable,
   anchor_out_of_range, anchor_type_unsupported, content_hash_mismatch.
4. Add tests using evaluation/golden/expected-anchors.json and fixture sources.
5. Keep resolver deterministic and side-effect free.

Guardrails:
- No retrieval result can be valid without a resolvable source anchor.
- Do not expose raw source content in logs.

Validation:
- Run component tests.
- Run python3 evaluation/source_anchor_roundtrip.py.
```

**QA prompt:**

```text
Review source anchor implementation.

Check:
1. Source refs are stable and parseable.
2. All required P0 anchor types resolve correctly.
3. Failure statuses are structured, not silent empty results.
4. Tests are derived from golden expected anchors.

If acceptable, mark P0-06 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/sourceanchors/sourceanchors.go`, `code/libs/sourceanchors/golden.go`, `code/libs/sourceanchors/sourceanchors_test.go`, `evaluation/golden/expected-anchors.json`, `evaluation/source_anchor_roundtrip.py`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && go test ./libs/sourceanchors`; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors && go test ./... && go build ./...`; `python3 evaluation/source_anchor_roundtrip.py`
- One-line summary: Added canonical source-ref parsing/formatting, deterministic anchor IDs/content hashes, P0 Markdown/TXT/DOCX/JSON resolver behavior, explicit failure statuses, golden-fixture loader, and tests derived from `expected-anchors.json`.
- Follow-up: P0-12 is now complete; continue with P0-13 embedding interface with deterministic stub/hash provider.

---

### P0-07 — Ingestion run lifecycle and degenerate-run guard

**Status:** ✅ Complete  
**Purpose:** Implement run state and promotion safety.  
**Deliverable:** Ingestion run lifecycle package and tests.  
**Dependencies:** P0-04, P0-06.

**Agent routing:**
- Primary Aara agent: `aara-data-tier-designer`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `aara-ai-evaluation-engineer`
- Invocation style: Direct coding-agent task, followed by review-only task
- When not to use that agent: Do not use to relax degenerate-run guards or promote partial extraction output.

**Implementation prompt:**

```text
Implement DIF ingestion run lifecycle and degenerate-run guard.

Scope:
1. Model ingestion run statuses: running, completed, failed, cancelled.
2. Track document, node, edge, anchor, passage, and caveat counts.
3. Implement promotion decision logic that only allows completed runs with document_count > 0, node_count > 0, anchor_count > 0, and passage_count > 0.
4. Add tests from evaluation/golden/expected-degenerate-runs.json.
5. Add explicit errors/statuses for non-promotable runs.

Guardrails:
- Empty or all-failed extraction must not replace a healthy serving index.
- Do not promote on partial/ambiguous success.

Validation:
- Run component tests.
- Run python3 evaluation/degenerate_run_checks.py.
```

**QA prompt:**

```text
Review ingestion run lifecycle and degenerate guard.

Check:
1. Promotion logic matches the SQL constraint and golden cases.
2. Failed/running/cancelled runs cannot promote.
3. Empty, no-node, no-anchor, and no-passage runs cannot promote.
4. Errors are explicit and test-covered.

If acceptable, mark P0-07 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/ingestionruns/ingestionruns.go`, `code/libs/ingestionruns/ingestionruns_test.go`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && go test ./libs/ingestionruns`; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns && go test ./... && go build ./...`; `python3 evaluation/degenerate_run_checks.py`
- One-line summary: Added ingestion run lifecycle model, explicit status/count validation, promotion decision logic matching the SQL guard, non-promotable errors, safe write-shape metrics, and golden degenerate-run tests.
- Follow-up: P0-12 is now complete; continue with P0-13 embedding interface with deterministic stub/hash provider.

---

### P0-08 — Markdown and TXT extractors

**Status:** ✅ Complete  
**Purpose:** Extract deterministic document/section/block records from Markdown and TXT.  
**Deliverable:** Markdown/TXT parser package and tests.  
**Dependencies:** P0-06, P0-07.

**Agent routing:**
- Primary Aara agent: `aara-python-ai-developer`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-project-reviewer`
- Invocation style: Direct coding-agent task, followed by evaluation/review task
- When not to use that agent: Do not use to introduce LLM-based extraction or unsupported file formats.

**Implementation prompt:**

```text
Implement deterministic Markdown and TXT extractors.

Scope:
1. Emit document, section, and block records for Markdown.
2. Emit document and block records for TXT.
3. Preserve heading path and line-range anchors.
4. Generate CONTAINS edges.
5. Preserve stable ordering and content hashes.
6. Add tests against evaluation/golden/sources/admitted/architecture-overview.md and runbook.txt.

Guardrails:
- Source anchors are mandatory.
- Output ordering must be deterministic.
- Do not use LLMs for extraction.

Validation:
- Run component tests.
- Run python3 evaluation/source_anchor_roundtrip.py.
```

**QA prompt:**

```text
Review Markdown/TXT extractors.

Check:
1. Traversal and output ordering are deterministic.
2. Heading and line anchors round trip.
3. CONTAINS edges are valid and stable.
4. Degenerate extraction cannot promote.

If acceptable, mark P0-08 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/extraction/extraction.go`, `code/libs/extraction/markdown.go`, `code/libs/extraction/text.go`, `code/libs/extraction/extraction_test.go`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && go test ./libs/extraction`; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction && go test ./... && go build ./...`; `python3 evaluation/source_anchor_roundtrip.py`
- One-line summary: Added deterministic Markdown/TXT extraction records with document/section/block nodes, source anchors, retrieval passages, stable IDs/hashes, CONTAINS edges, golden fixture tests, and degenerate-promotion coverage.
- Follow-up: P0-12 is now complete; continue with P0-13 embedding interface with deterministic stub/hash provider.

---

### P0-09 — JSON extractor with expansion caps

**Status:** ✅ Complete  
**Purpose:** Implement deterministic bounded JSON extraction.  
**Deliverable:** JSON parser package and tests.  
**Dependencies:** P0-06, P0-07.

**Agent routing:**
- Primary Aara agent: `aara-python-ai-developer`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `security-review` for secret-like value logging checks
- Invocation style: Direct coding-agent task, followed by evaluation and security-focused review
- When not to use that agent: Do not use if the task would silently drop unsupported content or log raw JSON values.

**Implementation prompt:**

```text
Implement DIF JSON extractor with ADR-006 caps and caveats.

Scope:
1. Traverse object keys in sorted order and arrays in ascending index order.
2. Emit JSONPath anchors for every JSON-derived node/passage.
3. Enforce all ADR-006 caps and emit machine-readable caveats.
4. Invalid JSON and too-large JSON must not emit partial graphs.
5. Add tests from evaluation/golden/expected-caveats.json and JSON fixtures.

Guardrails:
- Do not log raw secret-like JSON values.
- Cap behavior must be deterministic.
- Do not silently drop unsupported content.

Validation:
- Run component tests.
- Run python3 evaluation/json_caveat_checks.py.
```

**QA prompt:**

```text
Review JSON extractor.

Check:
1. Sorted object traversal and ascending array traversal are tested.
2. All 9 caveat codes are covered.
3. Invalid/too-large JSON failure behavior matches expectations.
4. Raw secret-like values are not logged.

If acceptable, mark P0-09 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/extraction/json.go`, `code/libs/extraction/json_test.go`, `code/libs/extraction/extraction.go`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && go test ./libs/extraction`; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction && go test ./... && go build ./...`; `python3 evaluation/json_caveat_checks.py`
- One-line summary: Added deterministic bounded JSON extraction with sorted object traversal, ascending array traversal, JSONPath anchors, ADR-006 cap caveats, fail-closed parse/size errors, secret-like value redaction in passages, and golden caveat tests.
- Follow-up: P0-12 is now complete; continue with P0-13 embedding interface with deterministic stub/hash provider.

---

### P0-10 — DOCX paragraph-model adapter

**Status:** ✅ Complete  
**Purpose:** Add P0 DOCX paragraph anchor support.  
**Deliverable:** DOCX adapter and tests.  
**Dependencies:** P0-06, P0-07.

**Agent routing:**
- Primary Aara agent: `aara-python-ai-developer`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-project-reviewer`
- Invocation style: Direct coding-agent task, followed by evaluation/review task
- When not to use that agent: Do not use to add heavy binary DOCX parsing dependencies before parser strategy is finalized.

**Implementation prompt:**

```text
Implement P0 DOCX paragraph-model adapter.

Scope:
1. Start with the committed requirements.docx.fixture.json paragraph model.
2. Emit document/section/block records with paragraph anchors.
3. Preserve paragraph_index source refs.
4. Keep a clear seam for replacing/supplementing with a binary DOCX parser later.
5. Add tests against evaluation/golden/sources/admitted/requirements.docx.fixture.json.

Guardrails:
- Do not cite the fixture wrapper JSON as the user-facing source.
- Do not add heavy parser dependencies until parser strategy is finalized.

Validation:
- Run component tests.
- Run python3 evaluation/source_anchor_roundtrip.py.
```

**QA prompt:**

```text
Review DOCX paragraph-model adapter.

Check:
1. User-facing source refs point to requirements.docx, not fixture JSON.
2. Paragraph anchors resolve to expected text.
3. Parser seam is clear for future binary DOCX support.
4. Output ordering is deterministic.

If acceptable, mark P0-10 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/extraction/docx.go`, `code/libs/extraction/extraction_test.go`, `code/README.md`, `code/libs/README.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && gofmt -w libs/extraction/docx.go libs/extraction/extraction_test.go && go test ./libs/extraction ./libs/sourceanchors`; `python3 evaluation/source_anchor_roundtrip.py`; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction && go test ./... && go build ./...`
- One-line summary: Added fixture-backed DOCX paragraph-model extraction with user-facing `requirements.docx#pN` anchors, deterministic ordering, invalid fixture-shape rejection, paragraph-resolution tests, and no heavy DOCX parser dependency.
- Follow-up: P0-12 is now complete; continue with P0-13 embedding interface with deterministic stub/hash provider.

---

### P0-11 — Deterministic graph emitter and NDJSON writer

**Status:** ✅ Complete  
**Purpose:** Centralize node/edge/anchor/passage emission with stable ordering.  
**Deliverable:** Emitter package and tests.  
**Dependencies:** P0-08, P0-09, P0-10.

**Agent routing:**
- Primary Aara agent: `aara-python-ai-developer`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-data-tier-designer`
- Invocation style: Direct coding-agent task, followed by evaluation/review task
- When not to use that agent: Do not use to mint unresolved external nodes or introduce nondeterministic output.

**Implementation prompt:**

```text
Implement deterministic DIF graph emitter and NDJSON writer.

Scope:
1. Emit document, section, block nodes and CONTAINS edges.
2. Emit source_anchors and retrieval_passages.
3. Ensure byte-stable NDJSON ordering for unchanged input.
4. Add deterministic ID placeholders or stable aliases until final ID algorithm is wired.
5. Add tests that run the same extraction twice and compare byte-identical output.

Guardrails:
- Every emitted passage must have anchor_id and source_ref.
- Do not emit dangling edges.
- Do not silently mint unresolved external nodes.

Validation:
- Run component tests.
- Run relevant scaffold harnesses.
```

**QA prompt:**

```text
Review graph emitter and NDJSON writer.

Check:
1. Same input produces byte-identical output.
2. Every passage has an anchor.
3. Every CONTAINS edge points to valid nodes.
4. Caveats are preserved.

If acceptable, mark P0-11 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/graphemit/emitter.go`, `code/libs/graphemit/emitter_test.go`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && gofmt -w libs/graphemit/emitter.go libs/graphemit/emitter_test.go && go test ./libs/graphemit ./libs/extraction`; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction ./libs/graphemit && go test ./... && go build ./...`; `python3 evaluation/source_anchor_roundtrip.py`; `python3 evaluation/json_caveat_checks.py`; `python3 evaluation/search_docs_checks.py`; `python3 evaluation/degenerate_run_checks.py`
- One-line summary: Added validation-first byte-stable NDJSON emission for documents, source anchors, nodes, `CONTAINS` edges, retrieval passages, and caveats across Markdown/TXT/DOCX/JSON extractor output.
- Follow-up: P0-12 is now complete; continue with P0-13 embedding interface with deterministic stub/hash provider.

---

### P0-12 — Retrieval passage generator and FTS query path

**Status:** ✅ Complete  
**Purpose:** Generate source-anchored passages and implement P0 FTS retrieval.  
**Deliverable:** Retrieval package and tests.  
**Dependencies:** P0-11.

**Agent routing:**
- Primary Aara agent: `aara-python-ai-developer`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-data-tier-designer`, `aara-ai-application-architect`
- Invocation style: Direct coding-agent task, followed by evaluation/review task
- When not to use that agent: Do not use to add unsupported answer generation or vector schema before dimensions are pinned.

**Implementation prompt:**

```text
Implement P0 retrieval passage generation and FTS query path.

Scope:
1. Generate retrieval passages for Markdown, TXT, DOCX, and JSON blocks.
2. Exclude any passage without a source anchor.
3. Implement deterministic FTS-backed query behavior for P0.
4. Return no_evidence for unsupported/unknown answers.
5. Add tests from evaluation/golden/golden-queries.json.

Guardrails:
- Do not answer beyond retrieved evidence.
- Do not return unanchored results.
- Keep vector search out until embedding dimensions are pinned.

Validation:
- Run component tests.
- Run python3 evaluation/search_docs_checks.py.
```

**QA prompt:**

```text
Review retrieval passage and FTS implementation.

Check:
1. Results include required fields: corpus_id, document_id, document_version_id, passage_id, snippet, anchor_id, source_ref, score, caveats.
2. Unanchored results are excluded.
3. Unknown questions return no_evidence, not invented answers.
4. Non-admitted corpus returns corpus_not_admitted.

If acceptable, mark P0-12 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/retrieval/retrieval.go`, `code/libs/retrieval/retrieval_test.go`, `code/libs/extraction/markdown.go`, `code/libs/extraction/text.go`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && gofmt -w libs/extraction/markdown.go libs/extraction/text.go libs/retrieval/retrieval.go libs/retrieval/retrieval_test.go && go test ./libs/retrieval ./libs/extraction ./libs/sourceanchors ./libs/graphemit`; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction ./libs/graphemit ./libs/retrieval && go test ./... && go build ./...`; `python3 evaluation/source_anchor_roundtrip.py`; `python3 evaluation/json_caveat_checks.py`; `python3 evaluation/search_docs_checks.py`; `python3 evaluation/degenerate_run_checks.py`
- One-line summary: Added anchored-only P0 lexical retrieval with corpus admission enforcement, deterministic ranking, golden-query coverage, explicit `no_evidence`/`corpus_not_admitted` statuses, and source-ref aligned Markdown/TXT passage anchors.
- Follow-up: Continue with P0-13 embedding interface with deterministic stub/hash provider.

---

### P0-13 — Embedding interface with stub/hash provider

**Status:** ✅ Complete  
**Purpose:** Add provider seam without pinning vector schema prematurely.  
**Deliverable:** Embedding interface and deterministic test provider.  
**Dependencies:** P0-12.

**Agent routing:**
- Primary Aara agent: `aara-ai-application-architect`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `aara-python-ai-developer`, `aara-ai-evaluation-engineer`
- Invocation style: Planning/design task if the seam is unclear; otherwise direct coding-agent task followed by review
- When not to use that agent: Do not use to select final production embedding dimensions or add pgvector schema.

**Implementation prompt:**

```text
Implement DIF embedding interface with a deterministic stub/hash provider.

Scope:
1. Define provider interface aligned with future shared RIF/LiteLLM abstraction.
2. Add deterministic hash/stub provider for local tests.
3. Do not add pgvector columns or real Voyage integration yet.
4. Record token/embedding unit placeholders for usage metering integration.

Guardrails:
- D-002 selects Voyage as prose default, but exact model/dimension remains open until spike exit.
- Do not create vector schema prematurely.

Validation:
- Run component tests.
```

**QA prompt:**

```text
Review embedding interface.

Check:
1. Provider abstraction can later support Voyage and self-host fallback.
2. Tests are deterministic and offline.
3. No premature vector dimension or pgvector schema was added.
4. Usage metering fields are compatible with existing usage_events.

If acceptable, mark P0-13 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/embeddings/embeddings.go`, `code/libs/embeddings/embeddings_test.go`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && gofmt -w libs/embeddings/embeddings.go libs/embeddings/embeddings_test.go && go test ./libs/embeddings`; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction ./libs/graphemit ./libs/retrieval ./libs/embeddings && go test ./... && go build ./...`; `python3 evaluation/search_docs_checks.py`; `python3 evaluation/audit_usage_checks.py`
- One-line summary: Added provider abstraction plus deterministic offline hash provider with normalized vectors, request validation, cancellation handling, and non-PII usage metering placeholders without adding pgvector schema or real Voyage integration.
- Follow-up: Continue with P0-14 service-layer `search_docs` contract.

---

### P0-14 — `search_docs` service contract

**Status:** ✅ Complete  
**Purpose:** Implement the service-layer `search_docs` contract.  
**Deliverable:** Search service and tests.  
**Dependencies:** P0-05, P0-12.

**Agent routing:**
- Primary Aara agent: `aara-senior-microservices-architect`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `security-review`, `aara-data-tier-designer`
- Invocation style: Direct coding-agent task, followed by evaluation and security-focused review
- When not to use that agent: Do not use to implement MCP transport or free-form generated answers.

**Implementation prompt:**

```text
Implement DIF service-layer search_docs contract.

Scope:
1. Validate tenant/project/corpus scope.
2. Enforce corpus admission before retrieval.
3. Return source-anchored results only.
4. Return explicit no_evidence and corpus_not_admitted statuses.
5. Include caveats and score fields.
6. Add tests from golden queries.

Guardrails:
- Do not implement MCP transport yet unless already scaffolded.
- No unsupported free-form answer generation.

Validation:
- Run component tests.
- Run python3 evaluation/search_docs_checks.py.
```

**QA prompt:**

```text
Review search_docs service contract.

Check:
1. Required fields match the P0 evaluation plan.
2. Non-admitted corpora fail closed before retrieval.
3. Results are source anchored.
4. no_evidence behavior is explicit.

If acceptable, mark P0-14 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/searchdocs/searchdocs.go`, `code/libs/searchdocs/searchdocs_test.go`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && gofmt -w libs/searchdocs/searchdocs.go libs/searchdocs/searchdocs_test.go && go test ./libs/searchdocs`; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction ./libs/graphemit ./libs/retrieval ./libs/embeddings ./libs/searchdocs && go test ./... && go build ./...`; `python3 evaluation/search_docs_checks.py`; `python3 evaluation/audit_usage_checks.py`
- One-line summary: Added the service-layer `search_docs` contract with scope validation, admission-before-retrieval enforcement, anchored-only result validation, explicit `ok`/`no_evidence`/`corpus_not_admitted`/fail-closed statuses, caveats, scores, and golden-query tests.
- Follow-up: Continue with P0-15 MCP/API skeleton for `search_docs`.

---

### P0-15 — MCP/API skeleton for `search_docs`

**Status:** ✅ Complete  
**Purpose:** Expose P0 `search_docs` through authenticated/audited API/MCP boundary.  
**Deliverable:** MCP/API skeleton and tests.  
**Dependencies:** P0-03, P0-14.

**Agent routing:**
- Primary Aara agent: `aara-mcp-server-builder`
- QA/review agent: `security-review`
- Supporting agents/skills: `aara-senior-microservices-architect`, `aara-project-reviewer`
- Invocation style: Direct coding-agent task, then mandatory security gate and review-only task
- When not to use that agent: Do not use to duplicate retrieval, ranking, graph traversal, or agent narration logic in the transport layer.

**Implementation prompt:**

```text
Implement P0 MCP/API skeleton for search_docs.

Scope:
1. Add a thin transport layer that validates required inputs.
2. Require auth on every endpoint/tool call.
3. P0 internal auth may use bearer-token constant-time comparison.
4. Route to the service-layer search_docs contract.
5. Reject missing required fields with structured errors.
6. Add tests for required-field validation and auth failure.

Guardrails:
- MCP layer must authorize, validate, route, audit, meter, and return grounded results.
- It must not duplicate ingestion, ranking, graph traversal, or agent narration logic.
- Pilot/remote deployment must move to OAuth 2.1 + PKCE.

Validation:
- Run component tests.
```

**QA prompt:**

```text
Review MCP/API skeleton.

Check:
1. No unauthenticated surface exists.
2. Required fields are validated non-empty.
3. Transport is thin and does not duplicate core service logic.
4. Responses include grounded source refs or explicit failure statuses.

If acceptable, mark P0-15 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/mcpapi/mcpapi.go`, `code/libs/mcpapi/mcpapi_test.go`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && gofmt -w libs/mcpapi/mcpapi.go libs/mcpapi/mcpapi_test.go && go test ./libs/mcpapi`; security review of `code/libs/mcpapi` returned no high-confidence findings; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction ./libs/graphemit ./libs/retrieval ./libs/embeddings ./libs/searchdocs ./libs/mcpapi && go test ./... && go build ./...`; `python3 evaluation/search_docs_checks.py`; `python3 evaluation/audit_usage_checks.py`
- One-line summary: Added a thin authenticated MCP/API boundary for `search_docs` with constant-time bearer-token validation, required-field checks, HTTP and tool-style entry points, service routing, structured failures, and grounded response envelopes.
- Follow-up: Continue with P0-16 audit and usage write path implementation.

---

### P0-16 — Audit logging and usage metering write paths

**Status:** ✅ Complete  
**Purpose:** Wire governance and metering into service/MCP calls.  
**Deliverable:** Audit/usage writer package and tests.  
**Dependencies:** P0-03, P0-15.

**Agent routing:**
- Primary Aara agent: `aara-data-tier-designer`
- QA/review agent: `security-review`
- Supporting agents/skills: `aara-mcp-server-builder`, `aara-ai-evaluation-engineer`
- Invocation style: Direct coding-agent task, then mandatory security gate and review-only task
- When not to use that agent: Do not use if raw parameters, query text, snippets, or document text would be stored.

**Implementation prompt:**

```text
Implement DIF audit logging and usage metering write paths.

Scope:
1. Write audit events to dif_meta.audit_log for MCP/API calls.
2. Write usage events separately to dif_meta.usage_events.
3. Include principal, tenant/project/corpus, tool name/version, parameters hash, outcome, latency, and source refs/count as appropriate.
4. Do not store raw parameters, query text, snippets, or document text in audit/usage records.
5. Add tests from evaluation/golden/expected-audit-usage.json.

Guardrails:
- Audit and usage are separate.
- Usage events must be non-PII.
- Do not log raw enterprise document text.

Validation:
- Run component tests.
- Run python3 evaluation/audit_usage_checks.py.
```

**QA prompt:**

```text
Review audit and usage write paths.

Check:
1. Audit and usage records are separated.
2. Audit contains required security dimensions.
3. Usage contains metering counts without PII.
4. No raw parameters, snippets, or document text are stored.

If acceptable, mark P0-16 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/auditusage/auditusage.go`, `code/libs/auditusage/auditusage_test.go`, `code/libs/mcpapi/mcpapi.go`, `code/libs/mcpapi/mcpapi_test.go`, `code/migrations/001_dif_meta_initial.sql`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && gofmt -w libs/auditusage/auditusage.go libs/auditusage/auditusage_test.go libs/mcpapi/mcpapi.go libs/mcpapi/mcpapi_test.go && go test ./libs/auditusage ./libs/mcpapi ./libs/migrations`; security review of audit/usage and MCP/API write paths returned no high-confidence findings after fixes; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction ./libs/graphemit ./libs/retrieval ./libs/embeddings ./libs/searchdocs ./libs/mcpapi ./libs/auditusage && go test ./... && go build ./...`; `python3 evaluation/audit_usage_checks.py`; `python3 evaluation/search_docs_checks.py`; scratch PostgreSQL replay ran `code/migrations/001_dif_meta_initial.sql` twice successfully.
- One-line summary: Added separated audit and non-PII usage write paths with SQL writer seams, safe parameter hashing, MCP/API governance recording, unauthorized-attempt audit coverage, and a migration-backed unknown-scope auth-audit sentinel corpus.
- Follow-up: Continue with P0-17 health and readiness checks.
- Follow-up: Continue with P0-17 health and readiness checks.

---

### P0-17 — Postgres-backed health check

**Status:** ✅ Complete  
**Purpose:** Add real health checks for DB connectivity and schema readiness.  
**Deliverable:** Health package/API and tests.  
**Dependencies:** P0-04, P0-15.

**Agent routing:**
- Primary Aara agent: `aara-senior-microservices-architect`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `aara-data-tier-designer`, `security-review` for secret-safe errors
- Invocation style: Direct coding-agent task, followed by review-only task
- When not to use that agent: Do not use to return static OK checks or leak database connection details.

**Implementation prompt:**

```text
Implement Postgres-backed health checks.

Scope:
1. Health check must verify database connectivity.
2. Readiness check must verify dif_meta schema/table availability.
3. Include RIF compatibility status as informational, not a P0 hard failure for doc-only mode.
4. Add tests for healthy, DB unavailable, and schema missing states.

Guardrails:
- Health checks must not be static OK.
- Do not leak connection strings or secrets in errors/logs.

Validation:
- Run component tests.
```

**QA prompt:**

```text
Review health checks.

Check:
1. Health verifies Postgres connectivity.
2. Readiness verifies dif_meta availability.
3. Errors are explicit but secret-safe.
4. RIF unavailable is reported without breaking standalone doc-only mode.

If acceptable, mark P0-17 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/health/health.go`, `code/libs/health/health_test.go`, `code/README.md`, `code/libs/README.md`, `.github/copilot-instructions.md`, `action_plan.md`, `tracking/phase-gate-status.md`
- Validation command(s): `cd code && gofmt -w libs/health/health.go libs/health/health_test.go && go test ./libs/health`; security review of health/readiness errors and HTTP handlers returned no high-confidence findings; `cd code && go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction ./libs/graphemit ./libs/retrieval ./libs/embeddings ./libs/searchdocs ./libs/mcpapi ./libs/auditusage ./libs/health && go test ./... && go build ./...`; `python3 evaluation/audit_usage_checks.py`; `python3 evaluation/search_docs_checks.py`
- One-line summary: Added Postgres-backed health/readiness checks with DB ping, `dif_meta` inventory validation, informational RIF compatibility reporting for doc-only mode, secret-safe failure messages, and HTTP health/readiness handlers.
- Follow-up: Continue with P0-18 RIF compatibility status check.

---

### P0-18 — RIF compatibility status check

**Status:** ✅ Complete  
**Purpose:** Implement real deploy/runtime RIF compatibility status detection.  
**Deliverable:** RIF compatibility package and tests.  
**Dependencies:** P0-04.

**Agent routing:**
- Primary Aara agent: `aara-senior-microservices-architect`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `aara-data-tier-designer`, `aara-ai-evaluation-engineer`
- Invocation style: Direct coding-agent task, followed by compatibility/evaluation review
- When not to use that agent: Do not use to implement `DESCRIBES` resolution or mutate RIF-owned schemas.

**Implementation prompt:**

```text
Implement DIF RIF compatibility status check.

Scope:
1. Detect rif_not_deployed, rif_incompatible, rif_shadow_empty, and rif_compatible.
2. Do not assume rif_meta.file_nodes, rif_meta.method_nodes, or rif_meta.class_nodes are populated.
3. Prefer compatibility views/API when present; allow AGE-backed detection path where available.
4. Persist/check status using dif_meta.rif_compatibility_status semantics.
5. Add tests derived from evaluation/fixtures/rif and expected resolutions.

Guardrails:
- Do not mutate RIF-owned schemas.
- Do not return success-shaped empty results for missing/incompatible RIF.
- Do not implement DESCRIBES resolution yet.

Validation:
- Run component tests.
- Run python3 evaluation/rif_compatibility_checks.py.
```

**QA prompt:**

```text
Review RIF compatibility status check.

Check:
1. All ADR-016 statuses are represented.
2. Empty shadows do not produce false success.
3. RIF-owned schemas are not mutated.
4. Cross-graph work remains blocked until service-level compatibility passes.

If acceptable, mark P0-18 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/rifcompat/rifcompat.go`; `code/libs/rifcompat/rifcompat_test.go`
- Validation command(s): `cd /Users/rb692q/projects/aaraminds-projects/dif/code && go test ./libs/rifcompat && go test ./... && go build ./...`; `cd /Users/rb692q/projects/aaraminds-projects/dif && python3 evaluation/rif_compatibility_checks.py`
- One-line summary: Added a RIF compatibility package with ADR-016 status assessment, AGE/shadow fallback handling, deterministic resolver lookups, NUL-separated RIF node/edge IDs, and `dif_meta.rif_compatibility_status` persistence.
- Follow-up: Continue with P0-19 Golden P0 evaluation runner.

---

### P0-19 — Golden P0 evaluation runner

**Status:** ✅ Complete  
**Purpose:** Tie scaffold and service-level tests into one repeatable P0 gate.  
**Deliverable:** Evaluation runner command and docs.  
**Dependencies:** P0-06 through P0-18.

**Agent routing:**
- Primary Aara agent: `aara-ai-evaluation-engineer`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `aara-ai-technical-author`
- Invocation style: Direct coding-agent task for harness glue, followed by review-only task
- When not to use that agent: Do not use to invent metrics or production SLOs.

**Implementation prompt:**

```text
Implement a P0 golden evaluation runner.

Scope:
1. Add a repeatable command that runs all Python scaffold harnesses and component tests.
2. Include deterministic extraction, source-anchor, JSON caveat, search_docs, audit/usage, degenerate-run, and RIF compatibility checks.
3. Capture baseline metrics without inventing production SLOs.
4. Update README, action_plan.md, phase-gate-status.md, and .github/copilot-instructions.md with exact commands.

Guardrails:
- Do not invent quality or performance targets.
- Metrics are measured baselines only.

Validation:
- Run the new P0 evaluation command.
```

**QA prompt:**

```text
Review P0 golden evaluation runner.

Check:
1. All required scaffold and component checks are included.
2. Commands are exact and reproducible.
3. Metrics are measured, not fabricated.
4. Documentation is synchronized.

If acceptable, mark P0-19 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `evaluation/run_p0.py`; `evaluation/README.md`; `code/README.md`; `action_plan.md`; `tracking/phase-gate-status.md`; `.github/copilot-instructions.md`
- Validation command(s): `cd /Users/rb692q/projects/aaraminds-projects/dif && python3 evaluation/run_p0.py`
- One-line summary: Added a repeatable P0 golden evaluation runner for Go component/full/build gates and Python scaffold harnesses, with measured duration/output baselines and no quality targets.
- Follow-up: Continue with the next pending prompt.

---

### P0-20 — CI baseline

**Status:** ✅ Complete  
**Purpose:** Add automated validation once runnable code exists.  
**Deliverable:** GitHub Actions workflow or documented CI baseline.  
**Dependencies:** P0-19.

**Agent routing:**
- Primary Aara agent: `aara-project-builder`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `security-review` for secret/publish safety
- Invocation style: Direct coding-agent task, followed by review-only task and security gate if publishing/scanning is added
- When not to use that agent: Do not use to add deployment secrets or publish containers; registry work must use JFrog, not Azure ACR.

**Implementation prompt:**

```text
Implement DIF CI baseline.

Scope:
1. Add CI for component tests.
2. Add SQL migration idempotency check where a Postgres service is available.
3. Add scaffold/golden evaluation harness checks.
4. Add lint/type-check commands if the toolchain supports them.
5. Add doc-link/path validation if feasible without extra dependencies.

Guardrails:
- Do not use Azure ACR for container registry; if container publishing is later added, use JFrog Artifactory.
- Do not add deployment secrets.
- Keep CI deterministic and safe for synthetic fixtures.

Validation:
- Run equivalent commands locally where possible.
```

**QA prompt:**

```text
Review CI baseline.

Check:
1. CI runs the same commands documented locally.
2. No secrets are committed.
3. No deployment/publish job runs unexpectedly.
4. Migration and harness checks are included.

If acceptable, mark P0-20 ✅ Complete.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `.github/workflows/ci.yml`; `evaluation/path_checks.py`; `evaluation/run_p0.py`; `evaluation/README.md`; `code/README.md`; `action_plan.md`; `tracking/phase-gate-status.md`; `.github/copilot-instructions.md`
- Validation command(s): `cd /Users/rb692q/projects/aaraminds-projects/dif && python3 evaluation/run_p0.py`; local PostgreSQL idempotency check using `createdb`, two `psql -v ON_ERROR_STOP=1 -f code/migrations/001_dif_meta_initial.sql` runs, and `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'dif_meta';`
- One-line summary: Added a GitHub Actions CI baseline that runs the P0 golden evaluation and PostgreSQL-backed migration idempotency without deployment secrets, Azure login, registry use, or publish jobs.
- Follow-up: P0 prompt queue is complete; continue with P0 exit/sanity review before starting P1.

---

## P1 prompt queue

### P1-01 — Code-entity candidate detector

**Status:** ✅ Complete  
**Purpose:** Detect code references in document blocks without creating resolved links prematurely.  
**Deliverable:** Candidate detector and tests.  
**Dependencies:** P0 exit, P0-18.

**Agent routing:**
- Primary Aara agent: `aara-python-ai-developer`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-senior-microservices-architect`, `aara-data-tier-designer`
- Invocation style: Direct coding-agent task only after unblock gate, followed by evaluation/review task
- When not to use that agent: Do not use before P0 exit and service-level RIF compatibility tests pass; this gate is now satisfied.

**Implementation prompt:**

```text
Implement P1 code-entity candidate detector for DIF document blocks.

Scope:
1. Detect qualified names, file paths, method/class references, backtick spans, code fences, and inline identifiers.
2. Store candidates as unresolved until RIF compatibility resolver confirms a match.
3. Preserve source anchor for every candidate.
4. Add resolution-status fields and caveats.

Guardrails:
- Do not mint dangling code nodes.
- Do not claim a DESCRIBES edge without resolver evidence.
- Use ADR-016 compatibility fields.
```

**QA prompt:**

```text
Review code-entity candidate detector.

Check candidate detection determinism, source-anchor preservation, no false resolved links, and no dependency on optional RIF shadow tables.
```

**Result:**
- Status: ✅ Complete
- Deliverable path(s): `code/libs/codeentities/candidates.go`; `code/libs/codeentities/candidates_test.go`; `code/README.md`; `code/libs/README.md`; `.github/copilot-instructions.md`; `action_plan.md`; `tracking/phase-gate-status.md`; `prompts.md`
- Validation command(s): `cd /Users/rb692q/projects/aaraminds-projects/dif/code && go test ./libs/codeentities`; `cd /Users/rb692q/projects/aaraminds-projects/dif/code && go test ./...`; `cd /Users/rb692q/projects/aaraminds-projects/dif/code && go build ./...`; `cd /Users/rb692q/projects/aaraminds-projects/dif && python3 evaluation/run_p0.py`
- One-line summary: Added deterministic unresolved code-entity candidate detection with source-anchor preservation, syntax-level match metadata, caveats, and SQL persistence without creating RIF nodes or `DESCRIBES` edges.
- Follow-up: Start P1-02 RIF resolver and `DESCRIBES` edges.

---

### P1-02 — RIF resolver and `DESCRIBES` edges

**Status:** ✅ Complete  
**Purpose:** Resolve document candidates to code nodes and create `DESCRIBES` edges.  
**Deliverable:** Resolver, edge writer, tests.  
**Dependencies:** P1-01.

**Agent routing:**
- Primary Aara agent: `aara-senior-microservices-architect`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `aara-data-tier-designer`, `aara-ai-evaluation-engineer`
- Invocation style: Planning/design check if resolver behavior is unclear; otherwise direct coding-agent task after unblock gate
- When not to use that agent: Do not use if it would assume populated RIF shadows or mutate RIF-owned schemas.

**Implementation prompt:**

```text
Implement RIF compatibility-layer resolver and DESCRIBES edge creation.

Scope:
1. Resolve exact qualified-name, file path, simple-name, and inferred/fuzzy matches.
2. Record confidence exact/inferred and caveats.
3. Store unresolved candidates explicitly.
4. Create DESCRIBES edges only when resolver evidence is sufficient.
5. Measure resolution rate per corpus.

Guardrails:
- Do not assume populated rif_meta shadows.
- AGE-backed path must work when shadows are empty.
- Do not mutate RIF-owned schemas.
```

**QA prompt:**

```text
Review RIF resolver and DESCRIBES edges.

Check exact/inferred confidence handling, unresolved behavior, AGE/shadow compatibility, no RIF schema mutation, and resolution-rate metrics.
```

**Result:**
- Status: ✅ Complete (2026-07-19)
- Deliverable path(s): `code/libs/codeentities/resolver.go`; `code/libs/codeentities/resolver_test.go`; `code/migrations/002_dif_meta_describes_edges.sql`; `code/libs/migrations/migrations_test.go` (two-migration inventory); `.github/workflows/ci.yml` (recreated + applies migration 002); `.github/copilot-instructions.md` (recreated)
- Validation command(s): `python3 evaluation/run_p0.py` (10/10 checks passed); from `code/`: `go test ./libs/codeentities ./libs/migrations`, `go build ./...`
- One-line summary: Resolver consumes `rifcompat` reports for qualified-name/source-path/simple-name/fuzzy modes with exact/inferred confidence, keeps ambiguous/unresolved/`rif_unavailable` outcomes explicit, creates `DESCRIBES` edges only from single-match resolver evidence using shared edge-ID semantics, measures per-corpus resolution rates, and persists to `dif_meta` only via additive migration 002.
- Follow-up: P1-03 `docs_for_code`/`code_for_doc` is unblocked. Note: `.github/` was missing from this working copy despite being documented complete; recreated to the `path_checks.py` contract — verify against the original machine `[VERIFY]`.

---

### P1-03 — Cross-graph tools: `docs_for_code` and `code_for_doc`

**Status:** ⏳ Pending (unblocked by P1-02 on 2026-07-19)  
**Purpose:** Expose anchored code/document lookups.  
**Deliverable:** Tools/API contracts and tests.  
**Dependencies:** P1-02.

**Agent routing:**
- Primary Aara agent: `aara-mcp-server-builder`
- QA/review agent: `security-review`
- Supporting agents/skills: `aara-senior-microservices-architect`, `aara-ai-evaluation-engineer`
- Invocation style: Direct coding-agent task after unblock gate, then mandatory security gate
- When not to use that agent: Do not use if RIF status handling is unavailable or responses would be unanchored.

**Implementation prompt:**

```text
Implement docs_for_code and code_for_doc.

Scope:
1. Require explicit tenant/project/corpus/repo scope.
2. Return source-anchored document and code references.
3. Return rif_not_deployed or rif_incompatible explicitly when compatibility is unavailable.
4. Audit and meter every call.

Guardrails:
- No success-shaped empty result for missing RIF.
- No unanchored responses.
```

**QA prompt:**

```text
Review docs_for_code and code_for_doc.

Check explicit RIF statuses, source anchors, audit/usage events, and no ungrounded claims.
```

**Result:**
- Status: ⏳ Pending
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### P1-04 — `trace_references` and `impact_of_change`

**Status:** ⏳ Pending (unblocked by P1-02 on 2026-07-19)  
**Purpose:** Add bounded graph traversal and impact semantics.  
**Deliverable:** Traversal tools and tests.  
**Dependencies:** P1-02.

**Agent routing:**
- Primary Aara agent: `aara-mcp-server-builder`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-senior-microservices-architect`, `security-review`
- Invocation style: Direct coding-agent task after unblock gate, followed by evaluation and security review
- When not to use that agent: Do not use for unbounded traversal or unsupported impact-analysis narration.

**Implementation prompt:**

```text
Implement trace_references and impact_of_change.

Scope:
1. Use bounded recursive traversal with max_depth hard cap 5.
2. Return source-anchored evidence.
3. Include caveats for inferred links and unresolved references.
4. Audit and meter calls.

Guardrails:
- No unbounded graph traversal.
- No answer without resolvable source refs.
```

**QA prompt:**

```text
Review trace_references and impact_of_change.

Check traversal cap, source anchors, caveats, audit/usage events, and deterministic ordering.
```

**Result:**
- Status: ⏳ Pending
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### P1-05 — PDF/PPTX parser router

**Status:** 🚫 Blocked until parser admission sequencing review  
**Purpose:** Add controlled PDF/PPTX extraction paths.  
**Deliverable:** Parser router, caveats, tests.  
**Dependencies:** P0 exit; parser admission sequencing review.

**Agent routing:**
- Primary Aara agent: `aara-ai-application-architect`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-python-ai-developer`, `aara-project-reviewer`
- Invocation style: Planning/design task first; direct coding-agent task only after parser admission criteria are explicit
- When not to use that agent: Do not use before parser admission sequencing review, or to admit PDF/PPTX without parser choice, anchors, caveats, golden tests, and cost profile.

**Implementation prompt:**

```text
Implement P1 PDF/PPTX parser router.

Scope:
1. Define parser choices, source anchors, node mapping, caveats, golden tests, and cost profile.
2. PDF anchors require page/bounding box where applicable.
3. PPTX anchors require slide/shape.
4. Unsupported/low-quality extraction must fail closed or emit caveats.

Guardrails:
- A format is not admitted until parser, anchor type, graph mapping, caveats, golden tests, and cost profile exist.
```

**QA prompt:**

```text
Review PDF/PPTX parser router.

Check format admission completeness, anchor resolvability, caveat behavior, and no silent content drops.
```

**Result:**
- Status: 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### P1-06 — Cross-encoder reranking

**Status:** 🚫 Blocked until P1 retrieval sequencing review  
**Purpose:** Improve hybrid retrieval ranking with recorded rerank scores.  
**Deliverable:** Reranker interface and tests.  
**Dependencies:** P0 retrieval; P1 retrieval sequencing review.

**Agent routing:**
- Primary Aara agent: `aara-python-ai-developer`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-ai-application-architect`
- Invocation style: Direct coding-agent task after unblock gate, followed by evaluation/review task
- When not to use that agent: Do not use to add reranking that can remove anchors or fabricate before/after metrics.

**Implementation prompt:**

```text
Implement P1 cross-encoder reranking.

Scope:
1. Add reranker interface and deterministic test implementation.
2. Record rerank scores.
3. Preserve source anchors and original retrieval evidence.
4. Add golden search evaluation before/after metrics.

Guardrails:
- Reranking must not introduce unanchored results.
- Metrics are measured, not invented.
```

**QA prompt:**

```text
Review reranking.

Check score recording, source-anchor preservation, deterministic tests, and measured metrics.
```

**Result:**
- Status: 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

## P2 prompt queue

### P2-01 — Citation-gated agent service

**Status:** 🚫 Blocked until P1 exit  
**Purpose:** Add thin agent narrative layer with claim-level citations.  
**Deliverable:** Agent service and tests.  
**Dependencies:** P1 exit.

**Agent routing:**
- Primary Aara agent: `aara-ai-application-architect`
- QA/review agent: `security-review`
- Supporting agents/skills: `aara-mcp-server-builder`, `aara-ai-evaluation-engineer`
- Invocation style: Planning/design task for claim contract, then direct coding-agent task and mandatory security gate
- When not to use that agent: Do not use before P1 exit or for unsupported free-form answers.

**Implementation prompt:**

```text
Implement DIF citation-gated agent service.

Scope:
1. Add /explain and /investigate_impact endpoints or service methods.
2. Return claim blocks with source refs.
3. Fail closed with 404/422 when claims cannot be grounded.
4. Fence repo/doc-derived text as data, not instructions.
5. Add groundedness scorer.

Guardrails:
- No free-form unsupported answers.
- No prompt-injection obedience from retrieved content.
```

**QA prompt:**

```text
Review citation-gated agent service.

Check claim-level citation enforcement, fail-closed behavior, prompt-injection controls, and audit/usage events.
```

**Result:**
- Status: 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### P2-02 — Version diff and topic explanation tools

**Status:** 🚫 Blocked until P2-01  
**Purpose:** Add `diff_versions` and `explain_topic`.  
**Deliverable:** Tools and tests.  
**Dependencies:** P2-01.

**Agent routing:**
- Primary Aara agent: `aara-mcp-server-builder`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-ai-application-architect`, `security-review`
- Invocation style: Direct coding-agent task after unblock gate, followed by evaluation and security review
- When not to use that agent: Do not use without claim-level citations and source-anchored version evidence.

**Implementation prompt:**

```text
Implement diff_versions and explain_topic.

Scope:
1. Compare document versions with source anchors.
2. Explain topics using retrieved evidence only.
3. Emit caveats for missing/incomplete evidence.
4. Audit and meter calls.

Guardrails:
- No unsupported generated claims.
- No unanchored diffs or explanations.
```

**QA prompt:**

```text
Review diff_versions and explain_topic.

Check source anchors, version correctness, grounded explanation, caveats, and audit/usage events.
```

**Result:**
- Status: 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### P2-03 — Drift report

**Status:** 🚫 Blocked until P1 federation and P2 agent gates  
**Purpose:** Report stale docs after code changes.  
**Deliverable:** `drift_report` tool and tests.  
**Dependencies:** P1-02, P2-01.

**Agent routing:**
- Primary Aara agent: `aara-ai-application-architect`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-mcp-server-builder`, `aara-senior-microservices-architect`
- Invocation style: Planning/design task if fixture evidence is missing; otherwise direct coding-agent task after unblock gate
- When not to use that agent: Do not use before P1 federation and P2 agent gates or without code/content version evidence.

**Implementation prompt:**

```text
Implement drift_report.

Scope:
1. Compare document DESCRIBES links against code version/content hash evidence.
2. Report stale, current, unresolved, and insufficient-evidence states.
3. Return source refs for document and code evidence.
4. Validate against a known code change fixture.

Guardrails:
- Do not fabricate drift without code/content version evidence.
- Return explicit insufficient evidence when needed.
```

**QA prompt:**

```text
Review drift_report.

Check version evidence, source refs, explicit insufficient-evidence behavior, and deterministic fixture validation.
```

**Result:**
- Status: 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### P2-04 — Incremental re-indexing

**Status:** 🚫 Blocked until P1/P2 core behavior  
**Purpose:** Add deterministic incremental updates.  
**Deliverable:** Incremental indexing service and tests.  
**Dependencies:** P1 exit.

**Agent routing:**
- Primary Aara agent: `aara-data-tier-designer`
- QA/review agent: `aara-ai-evaluation-engineer`
- Supporting agents/skills: `aara-python-ai-developer`, `aara-project-reviewer`
- Invocation style: Direct coding-agent task after unblock gate, followed by evaluation/review task
- When not to use that agent: Do not use to bypass atomic promotion or degenerate-run safeguards.

**Implementation prompt:**

```text
Implement incremental re-indexing.

Scope:
1. Detect changed/unchanged/deleted sources by content hash and source version.
2. Rebuild only affected document graph portions.
3. Preserve atomic promotion and degenerate-run guard.
4. Add tests proving incremental output matches full rebuild for equivalent corpus state.

Guardrails:
- No partial failed extraction may replace healthy serving index.
- Determinism must match full rebuild.
```

**QA prompt:**

```text
Review incremental re-indexing.

Check full-vs-incremental equivalence, atomic promotion, deletion handling, and degenerate-run safety.
```

**Result:**
- Status: 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

## P3 prompt queue

### P3-01 — SharePoint/OneDrive connector

**Status:** 🚫 Blocked until P2 exit and connector ADR readiness  
**Purpose:** Add Microsoft document-source connector for uniformly readable corpora.  
**Deliverable:** Connector, auth, retry/throttle handling, tests.  
**Dependencies:** P2 exit.

**Agent routing:**
- Primary Aara agent: `aara-senior-microservices-architect`
- QA/review agent: `security-review`
- Supporting agents/skills: `aara-data-tier-designer`, `aara-ai-evaluation-engineer`
- Invocation style: Planning/design task for connector auth and corpus admission, then direct coding-agent task after unblock gate
- When not to use that agent: Do not use for mixed-permission corpora or per-user ACL propagation in v1.

**Implementation prompt:**

```text
Implement SharePoint/OneDrive connector for DIF P3.

Scope:
1. Support only uniformly readable libraries/folders for v1.
2. Add connector auth, throttling, retry, checkpoint, and dead-letter behavior.
3. Preserve source provenance, versions, and content hashes.
4. Reject mixed-permission corpora with explicit status.

Guardrails:
- Do not implement per-user ACL propagation in v1.
- Do not overclaim mixed-permission support.
```

**QA prompt:**

```text
Review SharePoint/OneDrive connector.

Check uniform-readable admission, auth boundaries, retry/throttle behavior, source provenance, and fail-closed mixed-permission handling.
```

**Result:**
- Status: 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### P3-02 — Azure BYOC deployment with Terraform

**Status:** 🚫 Blocked until production readiness design  
**Purpose:** Deploy DIF into customer Azure tenancy.  
**Deliverable:** Terraform AzureRM deployment baseline.  
**Dependencies:** P2 exit, platform decisions.

**Agent routing:**
- Primary Aara agent: `aara-project-builder`
- QA/review agent: `security-review`
- Supporting agents/skills: `azure-prepare`, `azure-validate`, `azure-deploy`, `azure-rbac`, `azure-reliability`, `aara-ai-technical-author`
- Invocation style: Planning/design task for BYOC assumptions, then direct coding-agent task; run Azure validation before completion
- When not to use that agent: Do not use for SaaS multi-tenant deployment assumptions or Azure ACR registry publishing; use JFrog if registry publishing is needed.

**Implementation prompt:**

```text
Implement Azure BYOC Terraform deployment for DIF.

Scope:
1. Deploy into customer Azure tenancy.
2. Use managed identity and Key Vault.
3. Configure private networking posture.
4. Attach to project RIF Postgres as dif_meta sibling schema.
5. Add rollback plan for failed migrations or bad index promotion.

Guardrails:
- Do not use Azure ACR for AT&T-style container registry assumptions; if registry publishing is needed, use JFrog Artifactory.
- Do not commit secrets.
- Do not create a SaaS multi-tenant assumption.
```

**QA prompt:**

```text
Review Azure BYOC Terraform deployment.

Check customer-tenant boundary, managed identity, Key Vault, private networking, RIF Postgres attachment, rollback plan, and no secrets.
```

**Result:**
- Status: 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### P3-03 — Container hardening and registry pipeline

**Status:** 🚫 Blocked until service/container exists  
**Purpose:** Add production container baseline.  
**Deliverable:** Dockerfile, scans, registry pipeline.  
**Dependencies:** Runnable services.

**Agent routing:**
- Primary Aara agent: `aara-project-builder`
- QA/review agent: `security-review`
- Supporting agents/skills: `azure-validate` only for Azure-hosted runtime compatibility, `aara-project-reviewer`
- Invocation style: Direct coding-agent task, followed by security gate and review-only task
- When not to use that agent: Do not use to publish from CI unless explicitly gated; do not use Azure ACR for AT&T-style registry assumptions.

**Implementation prompt:**

```text
Implement DIF container hardening and registry pipeline.

Scope:
1. Non-root container.
2. Minimal runtime image.
3. .dockerignore and lockfile-based build.
4. HEALTHCHECK backed by Postgres-aware health endpoint.
5. Vulnerability/container scan.
6. If publishing is needed, use JFrog Artifactory, not Azure ACR.

Guardrails:
- Do not commit secrets.
- Do not publish from CI unless explicitly gated.
```

**QA prompt:**

```text
Review container hardening.

Check non-root runtime, healthcheck, secret safety, vulnerability scan, deterministic build, and JFrog-not-ACR registry posture where applicable.
```

**Result:**
- Status: 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### P3-04 — Observability, audit retention, and dashboards

**Status:** 🚫 Blocked until services exist  
**Purpose:** Add production observability and retention posture.  
**Deliverable:** Telemetry, dashboards, retention config/docs.  
**Dependencies:** Runnable services.

**Agent routing:**
- Primary Aara agent: `aara-project-builder`
- QA/review agent: `security-review`
- Supporting agents/skills: `appinsights-instrumentation`, `azure-reliability`, `aara-ai-technical-author`
- Invocation style: Direct coding-agent task for telemetry hooks/docs, then security and reliability review
- When not to use that agent: Do not use to log raw document text, PII usage payloads, or fabricate SLO/latency claims.

**Implementation prompt:**

```text
Implement DIF observability and retention baseline.

Scope:
1. OpenTelemetry traces and metrics.
2. Structured logs with redaction.
3. MCP tool-call metrics.
4. Audit-log retention policy.
5. Usage-event retention policy.
6. Dashboards for ingestion, retrieval, MCP latency, errors, caveats, and usage.

Guardrails:
- Do not log raw enterprise document text by default.
- Usage events must remain non-PII.
```

**QA prompt:**

```text
Review observability and retention baseline.

Check telemetry coverage, redaction, audit/usage separation, non-PII usage, retention policy, and dashboard usefulness.
```

**Result:**
- Status: 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

---

### P3-05 — Paid pilot readiness checklist

**Status:** 🚫 Blocked until P3 platform/security gates  
**Purpose:** Confirm production readiness before paid pilot.  
**Deliverable:** Pilot checklist and evidence pack.  
**Dependencies:** P3-01 through P3-04.

**Agent routing:**
- Primary Aara agent: `aara-ai-technical-author`
- QA/review agent: `aara-project-reviewer`
- Supporting agents/skills: `security-review`, `azure-validate`, `azure-reliability`, `aara-ai-evaluation-engineer`
- Invocation style: Review-only/planning task that compiles evidence; no code edits unless a missing artifact is explicitly identified
- When not to use that agent: Do not use to invent gate evidence, performance metrics, or unsupported ACL claims.

**Implementation prompt:**

```text
Create DIF paid pilot readiness checklist and evidence pack.

Scope:
1. Validate P0-P3 gates.
2. Confirm uniformly readable pilot corpus admission.
3. Confirm security review, deployment rollback, observability, audit, usage metering, and support model.
4. Include golden-query success criteria.
5. Include honest v1 limitations: no per-user source ACL propagation.

Guardrails:
- Do not invent performance or business metrics.
- Do not overclaim ACL support.
```

**QA prompt:**

```text
Review paid pilot readiness checklist.

Check gate evidence, corpus admission, security/observability/rollback readiness, usage metering, support model, and honest limitation language.
```

**Result:**
- Status: 🚫 Blocked
- Deliverable path(s):
- Validation command(s):
- One-line summary:
- Follow-up:

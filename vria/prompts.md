# VRIA Prompt Playbook

**Document set:** Value Realization Intelligence Agent
**Version:** v1.1
**Date:** 2026-07-06
**Purpose:** Copy-pasteable, skill-mapped prompts to take VRIA from current state (v1.2.1 spec pack with known defects) to production. Run in order; each phase gates the next.

**How to use:** Each prompt names how to run it:

- **Aara agent** — subagents in `aaraminds-platform/skills-pack/.claude/agents/` (Claude subagent format). Run from a session with that workspace wired, or paste the agent file as context.
- **Aara skill** — skills-pack (`skills-pack/.claude/skills/`) and instruction-os (`instruction-os/skills/`) skills. Run `.claude/wire-skills.sh` in `aaraminds-platform/` to enable auto-discovery; otherwise Read the SKILL.md directly.
- **Cowork fallback** — the generic Cowork skill if the Aara asset isn't wired in the session.

Every prompt carries Goal / Context / Constraints / Done-when. Paths are relative to `vria/`.

**Global constraints (apply to every prompt):**

- Authoritative-doc rules in `00_VRIA_Documentation_Index.md` win on conflict.
- Stack is fixed: Azure-primary, Go backend, PostgreSQL, React/Next.js frontend, GitHub Actions OIDC, Terraform AzureRM, Grafana + Prometheus + OpenTelemetry. No AWS, no Node backends.
- No fabricated metrics — unverified numbers get `[VERIFY]`.
- No employer-specific data (names, UC numbers) outside `internal/99_Source_AI_Use_Case_Inventory.md`.

---

## Phase 0 — Spec Remediation (blockers from the v1.2.1 review)

### P0.1 — Resolve GE-006 vs sustainment rule contradiction

**Run with:** Cowork `engineering:documentation` (no Aara equivalent — pure doc surgery)

> Goal: Make GE-006 and the sustainment rule agree.
> Context: `gate-b-behavior/07_VRIA_Golden_Eval_Set.md` GE-006 moves a use case to Regressed on one observation below threshold; `contracts/20_VRIA_Scoring_Rules_Spec.md` §7 requires two consecutive failed checks. `gate-b-behavior/04_VRIA_PRD.md` §6 uses the one-failure phrasing.
> Constraints: Keep the two-consecutive-failures rule from `20` §7 as canonical (it prevents single-period noise from flapping state). Rewrite GE-006's scenario to include two consecutive failed checks and update the PRD's Regressed definition. Do not touch other golden tests.
> Done-when: GE-006, `20` §7, and `04` §6 describe the identical trigger; grep for "sustainment" across the pack shows no remaining one-failure phrasing.

### P0.2 — Scrub employer data from reusable docs

**Run with:** Cowork `engineering:documentation`

> Goal: Enforce the `internal/99` quarantine rule.
> Context: `gate-d-operations/13_VRIA_Pilot_Plan.md` §3 names six real use cases; `gate-a-value/02_VRIA_Portfolio_Intake_Model.md` §8 uses real UC number UC-23338.
> Constraints: Replace pilot names with anonymized placeholders (Pilot Candidate A–F with tier + value-path descriptors); replace UC-23338 with UC-XXXX. Move the real pilot mapping into a new table in `internal/99_Source_AI_Use_Case_Inventory.md`. Docs 00–21 must remain distribution-safe.
> Done-when: `grep -rn "2[34][0-9]\{3\}\|Azure Cost Optimizer\|Scrum Master\|T-View\|Log Explorer\|Topology" gate-* contracts/` returns nothing.

### P0.3 — Schema alignment pass (17 ↔ 19 ↔ 09 ↔ 18)

**Run with:** Aara agent `aara-project-architect` + Aara skill `microservices-data-architecture` (fallback: Cowork `engineering:architecture`)

> Goal: One pass eliminating all drift between canonical schemas, physical model, and tool contracts.
> Context: `contracts/17`, `contracts/19`, `gate-c-runtime/09`, `contracts/18`. Known defects: (a) `initiative_cost_period` is an object in 17 §4/§5 but a string in 17 §7 and 09 §3.7; (b) assessments have no evidence linkage — 06 §10 requires exposing source IDs but 17 §7 and `value_assessments` can't store them; (c) sustainment threshold and status have no fields despite 20 §7; (d) `approval_requests` lacks `decided_by`; (e) no canonical `Scorecard` or `DecisionRecord` schema; (f) `get_pending_approvals`, `create_follow_up_action`, `append_decision_log` lack payloads.
> Constraints: `17` is authoritative — fix it first, then propagate. Standardize `initiative_cost_period` as the object form. Add `assessment_evidence` join table, `sustainment_threshold` + `sustainment_status` fields, `decided_by` column, Scorecard and DecisionRecord schemas + tables, and the three missing tool payloads per 09 §2's contract standard.
> Done-when: every field in 17 has a column in 19 and consistent types in 09/18/21; every tool in 09 §3.8 has strict input/output payloads; list each change in `CHANGELOG_v1_2.md` under a new v1.3 entry.

### P0.4 — Write executable component scoring formulas

**Run with:** Aara agent `aara-business-analyst` for the spec, `aara-project-architect` for the executable design (fallback: Cowork `product-management:write-spec` + `engineering:system-design`)

> Goal: Make `contracts/20_VRIA_Scoring_Rules_Spec.md` §2 and §3 genuinely executable.
> Context: §3 assigns weights (evidence_quality 20, metric_movement 20, baseline_quality 15, strategic_alignment 10, attribution_confidence 10, net_value 10, sustainment 10, governance_readiness 5) but no logic computes any component's points. Inputs available per assessment: ValueHypothesis, MetricSnapshot, EvidenceSource records, ApprovalRequest state (all in `contracts/17`).
> Constraints: Each formula must be a deterministic function of canonical schema fields only — no model judgment inside a formula. Use lookup tables (e.g., attribution_confidence: DirectMeasurement/A_BComparison=10, MatchedComparison/BeforeAfter=6, ExpertJudgement/ProxyMetric=3, Unknown=0) and bounded linear interpolation for metric_movement (current vs baseline→target progress). Every formula needs 3 worked examples including a degenerate input (null baseline, missing cost). Where a rubric needs a business decision (strategic_alignment tiers), present options and mark `[DECISION NEEDED]` rather than inventing one.
> Done-when: an engineer can implement §2–§3 without asking a question; all 15 golden tests in `07` are derivable from the formulas + caps; worked examples sum correctly.

### P0.5 — Define evidence freshness cycle cadence

**Run with:** Cowork `engineering:documentation`

> Goal: Make the sustainment check schedulable.
> Context: `contracts/20` §7 says checks run "every evidence freshness cycle (see 06 freshness rules)" but `gate-b-behavior/06` §8 defines freshness categories, not a cycle length.
> Constraints: Add a cadence table to `06` §8: per-metric `reporting_window` (set at hypothesis approval, default monthly) drives Fresh/Aging/Stale boundaries and the sustainment check schedule. Aging = 1 missed window, Stale = 2+. Update `20` §7 to reference it.
> Done-when: given any metric's reporting_window, the next sustainment check date and freshness state are computable.

### P0.6 — Add NFRs to the PRD

**Run with:** Aara agent `aara-business-analyst` (fallback: Cowork `product-management:write-spec`)

> Goal: Close the NFR gap in `gate-b-behavior/04_VRIA_PRD.md` (index promises NFRs; none exist).
> Context: internal portfolio tool, ~20–200 use cases, ~10–50 users, monthly ValueOps cadence — not a high-QPS system.
> Constraints: Cover availability, latency (dashboard read, assessment generation), data retention (audit vs operational), RPO/RTO, concurrency, accessibility (WCAG 2.1 AA), and audit-query performance. Right-size targets to an internal tool; mark any target needing stakeholder sign-off `[VERIFY]`.
> Done-when: `04` has an NFR section with testable targets; `16_VRIA_Production_Readiness_Checklist.md` Gate D gains matching verification rows.

### P0.7 — Deployment architecture for 08

**Run with:** Aara agent `aara-senior-microservices-architect` + Aara skills `microservices-architecture-design`, `azure-iac-policy-as-code`, `ai-application-architecture` (fallback: Cowork `engineering:architecture`)

> Goal: Deliver the deployment view `00` promises but `gate-c-runtime/08` lacks, and close its open ORs.
> Context: `08` has logical view only; CHANGELOG lists open decisions (backend language, search tech, dashboard platform, A2A in/out of MVP).
> Constraints: Decide, don't enumerate: Go backend on Azure Container Apps with managed identity; pgvector on Azure Database for PostgreSQL Flexible Server (one less service than AI Search — justify); React dashboard; A2A post-MVP (stub the adapter interface). Produce an ADR per decision with rejected alternatives, plus deployment diagram covering environments, identity flow (OIDC → managed identity → Key Vault), network boundaries, and observability wiring (OTel → Prometheus/Grafana).
> Done-when: `08` contains logical + deployment views and 4 ADRs; CHANGELOG "Build Readiness Notes" open items are each resolved or explicitly deferred with an owner.

### P0.8 — Complete API and event payloads

**Run with:** Aara skills `microservices-api-design` + `microservices-async-messaging` via `aara-project-architect` (fallback: Cowork `engineering:system-design`)

> Goal: Eliminate every `{}` payload — the pack's own acceptance bar forbids them.
> Context: `contracts/21` event `payload: {}` for 13 event types; no endpoints for follow-up actions, decision-log reads, metric-snapshot ingestion, evidence-source registration; no auth model, pagination, or versioning. `gate-c-runtime/09` A2A envelopes also use `{}`.
> Constraints: Define per-event payload schemas referencing `17` types. Add the four missing endpoint groups. Specify Entra ID (OIDC) bearer auth with roles from `10` §3, cursor pagination, `/api/v1` prefix. For A2A, define one concrete payload schema per `purpose` value.
> Done-when: `grep -n '"payload": {}\|_payload": {}' contracts/ gate-c-runtime/` returns nothing; every event/endpoint round-trips to a `17` schema.

### P0.9 — Fix approval state machine lifecycles

**Run with:** Aara agent `aara-project-architect` (fallback: Cowork `engineering:architecture`)

> Goal: Separate request lifecycle from artifact lifecycle in `contracts/18`.
> Context: Draft→Submitted→Approved is the approval-request lifecycle; Published/Superseded belong to the target artifact. Rejected is a dead end (undocumented); Published→Withdrawn clashes with Withdrawn's pre-decision meaning.
> Constraints: Two state machines: ApprovalRequest (Draft, Submitted, ChangesRequested, Approved, Rejected, Withdrawn — document Rejected as terminal, new request required) and Artifact (Draft, Approved, Published, Superseded, Invalidated — new state replacing the Published→Withdrawn hack). Update the `ApprovalState` enum split in `17` and the badge list in `gate-d-operations/15`.
> Done-when: no state appears in both machines; every transition has an actor and tool; `19` reflects the enum split.

### P0.10 — Verification pass on Phase 0

**Run with:** Aara agent `aara-project-reviewer` (fallback: Cowork `engineering:code-review` applied to docs)

> Goal: Independent review that P0.1–P0.9 landed clean before implementation starts.
> Context: The full pack under `vria/` post-remediation, plus `CHANGELOG_v1_2.md` v1.3 entry.
> Constraints: Re-run the original review checks: schema↔table↔contract triangle, golden tests vs scoring rules, `{}` scan, employer-data scan, enum duplication scan. Fresh eyes — do not assume the fixes are correct because the changelog says so.
> Done-when: written verdict listing zero blocking findings, or a defect list routed back to the failing P0 prompt.

---

## Phase 1 — Epic 1: Portfolio Registry

### P1.1 — Registry service design

**Run with:** Aara agent `aara-senior-microservices-architect` + Aara skills `microservices-api-design`, `azure-data-tier-design` (fallback: Cowork `engineering:system-design`)

> Goal: Design the registry service (import, normalization, validation, staging→promotion) for Epic 1 in `gate-d-operations/12`.
> Context: Tables in `contracts/19`; tool contracts `load_use_cases`, `get_use_case_status`, `draft_use_case_update` in `gate-c-runtime/09`; API in `contracts/21`; DeliveryStatus normalization rules needed for the messy statuses shown in `internal/99` ("Training; PTB/PTO not started/completed/pending mixed").
> Constraints: Go, PostgreSQL, staging table + explicit promotion per `09` §3.1 failure rules. Normalization must be a reviewable mapping table, not inline code. Reject-don't-guess on unmappable statuses.
> Done-when: design doc with service boundaries, normalization mapping table, error taxonomy matching `21` §3, and story-level acceptance criteria tied to eval IDs.

### P1.2 — Registry implementation

**Run with:** Aara agent `aara-project-builder`; review via `aara-project-reviewer` + Aara skill `pr-review-azure-microservices`

> Goal: Implement P1.1's design: migrations for `use_cases` + staging, import endpoint, normalization, promotion with audit.
> Context: P1.1 design doc; `contracts/19` DDL; `contracts/21` API contract.
> Constraints: Add CHECK constraints binding text columns to `17` enums (defect from review — `19` enforces nothing). Every write emits an `audit_events` row and the `use_case.imported` event. Table-driven tests for the normalization mapping including every malformed status string in `internal/99`.
> Done-when: migrations apply and roll back cleanly; import of the `99` inventory yields 17 staged records with correct rejects; `go test ./...` green; PR review passes with no High findings.

---

## Phase 2 — Epic 2: Value Hypothesis Workflow

### P2.1 — Hypothesis workflow implementation

**Run with:** Aara agent `aara-project-builder` (design assist: `aara-project-architect`)

> Goal: Create/edit hypothesis drafts with validation and approval routing (Epic 2).
> Context: Template `gate-a-value/03`; canonical schema `contracts/17` §4; `draft_use_case_update` contract; approval workflow `contracts/18`. Note the P0.3-fixed alignment between 03 and 17 — implement against 17.
> Constraints: Draft-write only; commits go through `submit_for_approval`. Field validation per Gate A requirements in `03` §4. Version increments per `17` §9.
> Done-when: hypothesis CRUD round-trips through approval; GE-002 and GE-011 scenarios pass against the running service; rejected commits leave the original record untouched (test proves it).

---

## Phase 3 — Epic 3: Evidence Retrieval

### P3.1 — Evidence MCP servers

**Run with:** Aara agent `aara-mcp-server-builder` + Aara skills `mcp-go-server-building`, `mcp-go-guardrails-and-safety`, `mcp-go-threat-modeling`; pre-merge: `mcp-go-production-review` (fallback: Cowork `mcp-builder`)

> Goal: Build the Metric Snapshot MCP and Document Evidence MCP servers per `gate-c-runtime/09` §3.5–3.6.
> Context: Contracts define strict I/O, failure codes (`METRIC_UNAVAILABLE`, `NO_EVIDENCE_FOUND`), freshness/authority metadata, audit fields. Source systems are open (`CHANGELOG` build-readiness) — build against an interface with one reference adapter each (CSV/file-drop for metrics; pgvector index for documents).
> Constraints: Go MCP SDK. Timeouts and retry per `09` §2. On failure return the contract's error code — never infer values (GE-013). Citation pointer mandatory on every document result; no citation, no result. Run mcp-go-threat-modeling before build, mcp-go-production-review before merge.
> Done-when: both servers pass contract tests generated from `09`'s JSON payloads; failure injection returns safe degradation codes; audit rows written per call; production review has no High findings.

### P3.2 — Evidence gap detection

**Run with:** Aara agent `aara-project-builder`

> Goal: Implement missing/stale/conflicting evidence detection (Epic 3, FR-03).
> Context: `gate-b-behavior/06` §4 quality dimensions, §8 freshness (with P0.5 cadence), §9 conflict resolution.
> Constraints: Conflict output must surface both values and sources — never average (per `06` §9). Gap output feeds the `missing_evidence` field of assessments.
> Done-when: GE-005, GE-009, GE-012 pass against the running service.

---

## Phase 4 — Epic 4: Scoring Engine

### P4.1 — Scoring engine implementation

**Run with:** Aara agent `aara-ai-evaluation-engineer` for the test plan + Aara skill `test-engineering`; build via `aara-project-builder` (fallback: Cowork `engineering:testing-strategy`)

> Goal: Implement `contracts/20` (post-P0.4): Gate A score, realization score, caps, state mapping, recommendation mapping, sustainment checks.
> Context: Formulas from P0.4; `score_value_realization` contract in `09` §3.7; `value_assessments` persistence in `19`.
> Constraints: Pure deterministic module — no LLM calls inside scoring. "Lowest cap wins" ordering exactly as `20` §4. Persist pre-cap and post-cap scores (dashboards trend pre-cap per v1.2.1). Property-based tests: score always 0–100, caps monotonic, state mapping total (every input maps to exactly one state).
> Done-when: all 15 golden tests pass computed end-to-end; property tests green; assessment snapshots carry scoring_rule/model/prompt versions.

### P4.2 — Sustainment scheduler

**Run with:** Aara agent `aara-project-builder`

> Goal: Scheduled sustainment checks per `20` §7 and P0.5 cadence.
> Context: Realized assessments; metric snapshot MCP; Regressed transition rules.
> Constraints: Missing/stale snapshot counts as a failed check. First failure → owner notification + `sustainment_status: at_risk`; second consecutive → Regressed + Fix/Rebaseline recommendation. All transitions audited.
> Done-when: GE-006 (post-P0.1 wording) passes with a two-cycle simulated regression; single-failure case provably stays Realized.

---

## Phase 5 — Epic 5: Approval Workflow

### P5.1 — Approval service

**Run with:** Aara agent `aara-project-architect` for design, `aara-project-builder` for build; security pass via Aara skill `microservices-architecture-reviewer`

> Goal: Implement both state machines from P0.9 with the tool contracts in `contracts/18`.
> Context: `approval_requests`, `scorecards`, `decision_log` tables; `21` endpoints; Tier rules in `gate-c-runtime/10` §4.
> Constraints: `publish_scorecard` executes only from Approved — enforce in the DB (constraint or guarded transition), not just service code. Decision log is append-only: no UPDATE/DELETE grants. Approver identity from Entra ID token, never from payload.
> Done-when: GE-007 passes (publish attempt without approval → approval request created, no publication); a direct-SQL bypass attempt fails on constraints; full audit chain (request → decision → publication hash) verifiable for one end-to-end run.

---

## Phase 6 — Epic 6: ValueOps Dashboard

### P6.1 — Dashboard design

**Run with:** Aara skill `frontend-engineering` + Cowork `design:design-system` → `design:ux-copy` → `design:design-critique` (no Aara design persona — Cowork design skills lead here)

> Goal: Design the seven views in `gate-d-operations/15` §2 with the visual rules of §4.
> Context: Required fields §3; approval badges (post-P0.9 states); "Unproven shown explicitly, never blank"; Regressed distinct from AtRisk.
> Constraints: React + existing component conventions. UX copy for empty states, caveat banners (stale evidence, unknown attribution, missing net value), and refusal messages must mirror the agent behavior rules in `gate-b-behavior/05` §7 — the UI must not soften "Unproven" into neutral language. Run design-critique on the drafts before handoff.
> Done-when: all seven views specced with states (loading/empty/error/caveat), copy reviewed, critique findings resolved.

### P6.2 — Dashboard build + accessibility

**Run with:** Aara skill `frontend-engineering` for the build; Cowork `design:design-handoff` before, `design:accessibility-review` after

> Goal: Implement P6.1 against the `/api/v1` contracts.
> Context: Handoff spec from P6.1; `21` endpoints; NFR accessibility target from P0.6.
> Constraints: Read-only views hit GET endpoints only; approval queue actions call the decision endpoint with optimistic-lock handling. No client-side score computation — display what the API returns.
> Done-when: WCAG 2.1 AA audit passes; every §3 field renders; badge states match the enum exactly.

---

## Phase 7 — Epic 7: Evaluation Harness

### P7.1 — Golden + volume eval harness

**Run with:** Aara agent `aara-ai-evaluation-engineer` + Aara skills `ai-evaluation-harness`, `test-engineering` (fallback: Cowork `engineering:testing-strategy`)

> Goal: Automate the release gate in `gate-b-behavior/07` §3.
> Context: 15 golden tests (critical vs non-critical), ≥50-record volume dataset seeded from `internal/99` shapes, failure-injection suite, regression policy §5.
> Constraints: Critical tests gate at 100% — CI-blocking. Percentage gates computed only from the volume dataset, never the golden 15. Dataset versioned with labeled expected outputs; regenerate labels on schema change per `07` §4. Failure injection covers every tool error code from `09`.
> Done-when: one CI job runs the full gate and blocks merge on any critical failure; gate report matches `07` §3 metric-for-metric.

### P7.2 — Red-team suite

**Run with:** Aara agent `aara-ai-evaluation-engineer` + Aara skills `mcp-go-threat-modeling`, `mcp-go-guardrails-and-safety` (fallback: Cowork `security-review`)

> Goal: Implement the eight red-team categories in `gate-c-runtime/11` §3.
> Context: Prompt-injection defenses `10` §5; GE-010; A2A trust requirements `09` §4.
> Constraints: Each category needs ≥3 attack variants including one indirect injection embedded in a retrieved evidence document. Injection attempts must produce a logged security event (verify the log, not just the refusal).
> Done-when: 0 critical prompt-injection failures per the `07` release gate; every attack has a logged, auditable outcome.

### P7.3 — Agent system prompt

**Run with:** Aara agent `aara-prompt-engineer` + Aara skills `prompt-engineering` (skills-pack), `agent-engineering`; agent-design sanity check via `aaraminds-ai-agent-blueprint-advisor` (fallback: Cowork `prompt-engineering`)

> Goal: Write the VRIA agent's system prompt encoding `gate-b-behavior/05` (role, autonomy Level 2, behavior rules §3, response contract §5, escalation §6, refusal §7).
> Context: Claude platform; tool definitions from `09`; untrusted-content rules from `10` §5.
> Constraints: Positive instructions over prohibition piles; the ten behavior rules become concrete directives with the why (e.g., "When a tool fails, set the field to Unknown — inferring a value creates an unaudited claim"). Response contract as an XML-tagged output spec. Neutral tool-selection language — no "ALWAYS search first" over-triggering. Include 3 few-shot exemplars: one clean assessment, one refusal (GE-011), one injection response (GE-010).
> Done-when: prompt passes the full golden suite via P7.1 harness; version-stamped and stored so assessments record prompt_version.

---

## Phase 8 — Pilot (Gate D)

### P8.1 — Pilot sprint plan

**Run with:** Aara agent `aara-project-planner` / Aara skill `aaraminds-project-planner` (fallback: Cowork `product-management:sprint-planning`)

> Goal: Turn the 6-week cadence in `gate-d-operations/13` §4 into sprint plans with capacity and P0/stretch split.
> Context: Pilot set (anonymized in `13`, real mapping in `internal/99`); exit criteria `13` §5.
> Constraints: Week-1 gate is hard: registry loaded + golden evals confirmed before hypothesis work starts. Each week's outcome maps to a demoable artifact.
> Done-when: sprint plan with owners, capacity, and go/no-go checkpoint dates; blockers list with named escalation paths.

### P8.2 — Weekly pilot stakeholder update

**Run with:** Aara skill `aaraminds-executive-narrative-advisor`; deck variant via `aara-status-deck` / `aaraminds-leadership-status-deck` (fallback: Cowork `product-management:stakeholder-update`) — recurring, weeks 1–6

> Goal: Weekly exec-brief update against `13` §4 outcomes.
> Context: Sprint progress, eval pass rates, evidence-coverage stats from the harness.
> Constraints: Match VRIA's own rules — no watermelon-Green status, no unverified numbers, evidence-linked claims only. Red/Amber items lead.
> Done-when: update sent; risks have owners and dates.

### P8.3 — Pilot synthesis and go/no-go

**Run with:** Aara skill `aaraminds-executive-narrative-advisor` for the decision memo (fallback: Cowork `product-management:synthesize-research` → `product-management:metrics-review`)

> Goal: Week-6 synthesis of owner feedback + pilot metrics into the go/no-go decision per `13` §5 and kill conditions in `gate-a-value/01` §8.
> Context: Owner interviews, scorecard rejection rate, unsupported-claim rate, approval-workflow test results.
> Constraints: Score the pilot against each §5 exit criterion pass/fail — no narrative substitution. A failed kill-condition check ends the debate.
> Done-when: decision memo with per-criterion evidence, recommendation (scale/fix/stop), and decision-log entry.

---

## Phase 9 — Production

### P9.1 — Production readiness run

**Run with:** Aara agent `aara-project-reviewer` (fallback: Cowork `engineering:deploy-checklist`)

> Goal: Execute `gate-d-operations/16` Gates A–D, flipping every "Pending" to verified.
> Context: All phase outputs; rollback requirements `14` §5 (prompt/model/scoring/tool versions + scorecard supersession).
> Constraints: Each check needs evidence (link, test run, sign-off), not assertion. Rollback rehearsed, not just documented — supersede a scorecard in staging and roll back a scoring-rule version.
> Done-when: all four gates verified with evidence links; go/no-go recorded in the decision log.

### P9.2 — Operational hardening

**Run with:** Aara skill `azure-microservices-observability` (fallback: Cowork `engineering:incident-response` prep mode + `engineering:documentation`)

> Goal: Upgrade `gate-d-operations/14` from metrics list to real runbook.
> Context: Incident types §4; production metrics §3; the review flagged missing severities, SLOs, and procedures.
> Constraints: Per incident type: severity, detection signal (Grafana alert), first response, escalation, comms template. SLOs from P0.6 NFRs. "Unsupported claim published" and "approval bypass" are Sev-1 with a scorecard-supersession procedure.
> Done-when: on-call can execute any §4 incident from the runbook alone; alert rules exist in code.

### P9.3 — Instrument VRIA's own value

**Run with:** Cowork `product-tracking-skills` chain: `product-tracking-model-product` → `product-tracking-design-tracking-plan` → `product-tracking-implement-tracking` (no Aara equivalent)

> Goal: VRIA must pass its own bar — instrument it so its value claim is evidence-backed.
> Context: Business outcomes table `gate-a-value/01` §3; online eval metrics `gate-c-runtime/11` §4; "cost per reliable insight."
> Constraints: Track the §3 outcome metrics with baselines captured before launch. VRIA gets its own row in its own registry, scored by its own rules.
> Done-when: tracking plan covers every §3 metric; VRIA's self-assessment renders on its own dashboard without triggering its own evidence-gap detector.

### P9.4 — Quarterly tech-debt review (recurring)

**Run with:** Aara agent `aara-project-reviewer` (fallback: Cowork `engineering:tech-debt`)

> Goal: Standing quarterly audit of deferred items.
> Context: A2A adapter (deferred P0.7), reference-adapter-only MCP sources (P3.1), `[DECISION NEEDED]` rubrics (P0.4), single-tenant RLS posture (`19` §2).
> Constraints: Each item gets a cost-of-delay call: fix now, schedule, or accept with rationale in the decision log.
> Done-when: prioritized register with owners; accepted risks documented.

---

## Agent/skill coverage map

| VRIA workstream | Primary Aara asset | Cowork fallback |
|---|---|---|
| Spec/architecture fixes | `aara-project-architect`, `aara-senior-microservices-architect` | `engineering:architecture` |
| Requirements/NFRs | `aara-business-analyst` | `product-management:write-spec` |
| Service builds | `aara-project-builder` | — (direct build) |
| MCP servers | `aara-mcp-server-builder` + `mcp-go-*` skills | `mcp-builder` |
| Eval harness / red team | `aara-ai-evaluation-engineer` + `ai-evaluation-harness` | `engineering:testing-strategy`, `security-review` |
| Agent prompt | `aara-prompt-engineer` + `agent-engineering` | `prompt-engineering` |
| Reviews / readiness | `aara-project-reviewer` + `pr-review-azure-microservices` | `engineering:code-review`, `deploy-checklist` |
| Planning | `aara-project-planner` | `product-management:sprint-planning` |
| Exec reporting | `aaraminds-executive-narrative-advisor`, `aara-status-deck` | `product-management:stakeholder-update` |
| Dashboard | `frontend-engineering` | `design:*` skills |
| Self-instrumentation | — | `product-tracking-skills:*` |

---

## Sequencing summary

```text
Phase 0 (P0.1–P0.10)  → spec clean, verified          [blocks everything]
Phase 1 → 2 → 3        → registry, hypotheses, evidence [1 blocks 2; 2,3 parallel]
Phase 4 → 5            → scoring, approvals             [4 needs 3; 5 parallel with 4]
Phase 6, 7             → dashboard, eval harness        [parallel; 7.3 needs 4,5]
Phase 8 (weeks 1–6)    → pilot                          [needs 1–7]
Phase 9                → production + recurring ops     [needs 8 go decision]
```

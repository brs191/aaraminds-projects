# AI Token Optimizer — Product Requirements Document (M2)

**Status:** Draft v0.1 — **gate-contingent**; this PRD specifies the M2 conditional build and is committed only on a Green M1 verdict.
**Owner:** Raja  ·  **Date:** 2026-05-26  ·  **Audience:** delivery engineering, product, security, AITO's leadership
**References:** `AI_Token_Optimizer_Product_Brief_2026-05-24.md` · `../design/AI_Token_Optimizer_Agent_Blueprint_v0.1.md` (pending v0.2) · `../design/Product_Architecture.svg` · `../evaluation/AI_Token_Optimizer_Systems_Review_2026-05-21.md` · `../evaluation/Project_Readiness_Evaluation.md` · `../planning/Roadmap.md` · `../planning/Delivery_Plan.md`

> Every numeric threshold marked `[CALIBRATE]` in this PRD is set by the M1 gate calibration step. Until M1 runs and the calibration is recorded, those values are placeholders and any "the product meets X" claim against them is provisional.

---

## 1 · TL;DR

The AI Token Optimizer is a **local-first developer-desktop tool** that compresses the context AITO's developers send to AI coding assistants. Headline guarantees: per-developer token cost goes down, answer quality is measurably maintained (the Fidelity Floor), and no source code leaves the workstation. The product ships as a **bundled local sidecar** plus a **VS Code `.vsix`** and an **IntelliJ plugin**, with manual install and zero cloud egress beyond the assistant provider's existing API. This PRD specifies what the product does, what it must satisfy, and how acceptance is judged. It does not specify schedule or implementation strategy — those live in `../planning/Roadmap.md` and `../planning/Delivery_Plan.md`.

---

## 2 · Problem

AI coding-assistant spend scales linearly with tokens, and most of those tokens are redundant context — conversation history, tool output, large system prompts — re-sent every request. AITO today has no measured baseline for this spend. Three forces converge:

The assistant vendors are moving to usage-based billing — GitHub Copilot's switch to token-based AI Credits on 1 June 2026 makes this concrete for AITO's developers, not theoretical.

The vendors are also compressing context themselves natively (Claude Code Auto-Compact, VS Code 1.118 token-efficiency work) — that compresses the bill for free, but it is opaque, server-side, and does not guarantee no quality loss.

The hosted competitors that already do third-party context compression (Context Gateway, OmniRoute) require source code to leave the developer's machine — incompatible with AITO's zero-egress posture for client work.

The space the Token Optimizer occupies is the intersection that is left: **local, no egress, with a measured no-degradation guarantee, on both VS Code and IntelliJ.**

---

## 3 · Goals

The product must satisfy all of the following on the AITO developer cohort, measured against the spike-established baseline.

**G1 — Measured token reduction.** Median input-token reduction per developer ≥ **20% (Green threshold), incremental over the assistant's native baseline**, against the per-developer passthrough baseline established at M0. Calibrated 2026-05-26; see `../tracking/milestones/M1-Decision-Gate.md`.

**G2 — Fidelity Floor holds.** Answer-quality regression in ≤ **5% of A/B pairs** in ongoing measurement (**≤ 3% on code-heavy fixtures**), and **every compression strategy that fails retrospective verification rolls back automatically and tightens its threshold**.

**G3 — No source-code egress.** No source code leaves the developer's workstation except as part of the compressed assistant request body that the developer's existing assistant would have sent anyway. Background agent calls to the provider carry metadata only.

**G4 — Latency budget held.** Compression contributes < **300 ms p95** of added latency per chat request, and < **100 ms p95** on inline completions.

**G5 — IDE parity.** Works identically on VS Code (`.vsix`) and IntelliJ on Windows, macOS, and Linux — no feature-gap by editor.

**G6 — Self-funding.** The optimizer's own runtime cost (Compression Sidecar inference, Compression Advisor Agent calls, Evaluator) is < **5 %** of the tokens it saves, measured per developer per week.

## 4 · Non-goals

Out of scope for M2. Each is a deliberate exclusion; revisiting any of these requires a roadmap change.

- **Model routing** (RouteLLM, NotDiamond, Auto Router patterns).
- **Semantic caching** of similar prompts.
- **Multi-user, team, or multi-tenant deployment** — the optimizer is single-developer, on-workstation.
- **Hosted gateway mode** — the design is intentionally local-only.
- **Compression of arbitrary application LLM traffic** outside the coding-assistant request path.
- **Mobile or browser-based editors** — desktop IDE only.
- **Coding-assistant types beyond chat and inline completion** — extending the request taxonomy is a re-review trigger.
- **Per-team or organisational reporting** of token spend.

---

## 5 · Users and personas

**P1 — Senior engineer (primary).** Uses Claude Code, Cursor, Continue, or Copilot daily. Values: low-friction install, no behaviour change on daily use, transparent failure modes. Pain: rising assistant bill, opaque compression. Will not tolerate: a tool that makes the assistant feel slower, an approval dialog that fires on every completion, or any tool that ships source code anywhere.

**P2 — Platform / security lead (secondary).** Approves what runs on developer workstations. Values: a writeable, auditable record that no source code egresses; explicit TLS boundary; no CA certificates installed; passthrough fallback that survives sidecar death. Will not tolerate: any TLS man-in-the-middle, any opaque egress.

**P3 — Delivery owner / Raja (delivery audience).** Tracks per-developer measured savings against the spike-established baseline and the Fidelity Floor in ongoing operation. Needs a dashboard surface (post-M2 enhancement) but in M2 needs the local audit log to be queryable.

---

## 6 · Use cases

| UC | Title | Actor | Trigger | Outcome |
|---|---|---|---|---|
| UC-1 | Daily coding with compression on | P1 | Developer makes a coding-assistant request | Request is compressed per policy, response returns, audit log records the cut and its risk score |
| UC-2 | Chat-request approval flow | P1 | A lossy cut above the auto-approve threshold is required on a chat request | The IDE plugin surfaces an approval prompt; the developer accepts, rejects, or edits the cut; decision is audited |
| UC-3 | Inline-completion flow | P1 | A completion request fires while the developer is typing | Compression applies in deterministic-lossless or pre-approved policy mode only; no interactive prompt ever surfaces on a completion |
| UC-4 | Per-repo budget exceeded | P1, P3 | Token spend for the repo crosses the configured budget for the window | The IDE plugin surfaces a non-blocking notification with the budget state and a link to the local audit log |
| UC-5 | Quality regression detected | system | The Quality-Regression Evaluator finds a compression strategy degrading answers beyond the Fidelity Floor bound | The strategy is automatically rolled back, its threshold tightened, the rollback is audited, and the developer is notified once per rollback per repo |
| UC-6 | Optimizer Core unhealthy | system, P2 | The Optimizer Core process crashes or stops responding | The Passthrough Listener (separately supervised) continues to forward requests verbatim to the provider; the developer's assistant keeps working; an incident is audited |
| UC-7 | New developer onboarding | P1 | A developer installs the optimizer for the first time on a repo | The optimizer runs in baseline-capture mode for a configurable window (passthrough only), measures the developer's own baseline, then activates compression |
| UC-8 | Audit query | P2 | Security lead reviews recent decisions | Reads the local audit log via a documented CLI; every lossy cut has a risk score, an approval decision, and a strategy reference |

---

## 7 · Functional requirements

Requirements are numbered, testable, and traceable to a source. Each FR cites the source as `[Blueprint §N]`, `[Module 5 Finding N]`, or `[Brief §N]`.

### Interception and IDE plugins

- **FR-1.** The bundled sidecar SHALL expose a loopback proxy bound to `http://127.0.0.1:<port>`. `[Blueprint §11]`
- **FR-2.** The IDE plugin SHALL automatically configure the developer's coding assistant's API base URL to point at `http://127.0.0.1:<port>` on install and on assistant-tool update. `[Blueprint §11]`
- **FR-3.** The proxy SHALL accept requests on the OpenAI-compatible and Anthropic-compatible API surfaces used by the in-scope assistants.
- **FR-4.** The plugin SHALL be packaged as a VS Code `.vsix` and an IntelliJ Platform plugin (`.zip` / Gradle), each installable manually from a local file. `[Brief §What the product would be]`
- **FR-5.** The plugin SHALL provide a CLI or settings panel that displays the current loopback port, the active policy, and the local audit-log path.

### Compression and request handling

- **FR-6.** The Compression Sidecar SHALL wrap LLMLingua-2 and be reachable from the Optimizer Core over loopback IPC.
- **FR-7.** Compression SHALL skip messages below a configurable size threshold. `[spike compression_hook.py default 800 chars]`
- **FR-8.** Compression SHALL preserve the latest user message verbatim. `[spike compression_hook.py]`
- **FR-9.** On any compression error, the request SHALL pass through uncompressed and the error SHALL be audited; compression SHALL never block a request. `[Blueprint §12; spike SPIKE_PLAN.md fail-open]`
- **FR-10.** The Optimizer Core SHALL distinguish between chat requests and inline completions and apply the per-type policy. `[Module 5 Finding 7]`
- **FR-11.** Completions SHALL never surface an interactive approval dialog; only deterministic-lossless compression or a pre-approved policy applies. `[Module 5 Finding 7]`
- **FR-12.** Chat requests MAY surface an approval dialog when a lossy cut exceeds the auto-approve risk threshold. `[Blueprint §6]`

### Passthrough and reliability

- **FR-13.** A Passthrough Listener SHALL own the loopback port and run as a separately supervised process, distinct from the Optimizer Core. `[Module 5 Finding 2]`
- **FR-14.** When the Optimizer Core is unhealthy (unresponsive within a configurable timeout or crashed), the Passthrough Listener SHALL forward requests verbatim to the assistant's real backend until the Core recovers. `[Module 5 Finding 2]`
- **FR-15.** Passthrough activation and recovery events SHALL be audited. `[Module 5 Finding 2]`

### Fidelity Floor and quality

- **FR-16.** The Defining Operational Constraint (the Fidelity Floor) SHALL hold as a two-clause invariant: at decision time, every lossy cut is either predicted-safe by risk score below a threshold OR human-approved; at verification time, every compression strategy is continuously and retrospectively measured by the Evaluator. `[Module 5 Finding 1]`
- **FR-17.** A compression strategy that fails retrospective quality verification SHALL be automatically rolled back, its threshold tightened, and the rollback audited. `[Module 5 Finding 1]`
- **FR-18.** The Quality-Regression Evaluator SHALL run locally — either a local judge model OR deterministic structural scoring (parse, symbol-set overlap, diff distance) — and SHALL NOT egress answer content to any external service. `[Module 5 Finding 5]`
- **FR-19.** Evaluator inference cost (tokens, CPU) SHALL be metered as part of the self-funding ledger. `[Module 5 §Observability gap 2]`

### Egress and security

- **FR-20.** The proxy SHALL operate on `http://127.0.0.1:<port>` only; the product SHALL NOT install a CA certificate or perform TLS man-in-the-middle on any traffic. `[Module 5 Finding 4]`
- **FR-21.** The `Authorization` header on every assistant request SHALL pass through the Optimizer Core untouched and SHALL NOT be logged, persisted, or inspected by any component. `[Brief; Module 5 §control gaps]`
- **FR-22.** The Compression Advisor Agent SHALL reach the assistant's real backend only through the `get_context_metadata` MCP tool's output — that is, paths, symbol names, dependency-graph edges, recency — and SHALL NOT egress raw source code under any policy. `[Blueprint §5; Module 5 Finding 5]`
- **FR-23.** A no-new-egress CI test SHALL fail any build that introduces a new outbound network destination beyond the configured assistant provider endpoints. `[Blueprint §6]`

### MCP tier and write-path

- **FR-24.** The agent and other in-sidecar components SHALL reach the system only through the five allowlisted typed MCP tools: `get_context_metadata`, `get_compression_policy`, `get_quality_signal`, `get_recent_decisions`, `record_telemetry`.
- **FR-25.** `record_telemetry` SHALL be the only write-tier MCP tool. No other tool SHALL mutate persistent state.

### Budgets, isolation, baseline

- **FR-26.** The Control Plane SHALL support per-repository budgets, configurable per workspace. `[Blueprint §6]`
- **FR-27.** The workspace isolation key SHALL be the canonical repository root path (or its hash); behaviour for monorepos with sub-projects and for multi-root workspaces SHALL be explicitly documented. `[Module 5 Finding 8]`
- **FR-28.** On first activation in a repo, the optimizer SHALL run in baseline-capture mode (passthrough, no compression) for a configurable window to establish a per-developer baseline. `[Module 5 Finding 9]`

### Audit and telemetry

- **FR-29.** The Audit Log SHALL record, for every lossy cut: timestamp, repo, request type, strategy applied, risk score, approval decision (auto, developer-accepted, developer-rejected), and outcome.
- **FR-30.** The Audit Log SHALL NOT record request or response bodies, headers, or any secrets. `[Module 5 §control gaps]`
- **FR-31.** Telemetry SHALL be emitted in OpenTelemetry-compatible shape and SHALL remain local-only by default; any future remote sink is a re-review trigger.

---

## 8 · Non-functional requirements

### Performance

- **NFR-1.** Compression latency overhead SHALL be < **`[CALIBRATE]` ms p95** per request, measured end-to-end IDE → response. `[Goal G4]`
- **NFR-2.** Sidecar resident memory SHALL be < **`[CALIBRATE]` MB** at steady state on the AITO reference developer workstation profile.
- **NFR-3.** Sidecar cold-start time from IDE launch to first request served SHALL be < **`[CALIBRATE]` seconds**.

### Security and privacy

- **NFR-4.** No source code SHALL egress the workstation except as part of the assistant request body the developer's assistant would have sent anyway, after compression. `[Goal G3]`
- **NFR-5.** The product SHALL hold a clean record against the no-new-egress CI test for every release.
- **NFR-6.** The product SHALL pass a STRIDE threat-modeling review against the `mcp-go-threat-modeling` skill criteria before GA.
- **NFR-7.** The product SHALL map cleanly onto the SOC 2 and ISO 27001 controls captured in `soc2-iso27001-controls-mapping`. `[skills-pack]`

### Reliability

- **NFR-8.** Optimizer Core crashes SHALL NOT interrupt assistant requests for longer than the Passthrough Listener's failover detection window (target < **`[CALIBRATE]` ms**). `[Module 5 Finding 2]`
- **NFR-9.** Compression errors SHALL fail open — request continues uncompressed — in **100 %** of error cases. `[FR-9]`

### Observability

- **NFR-10.** Every lossy cut SHALL be auditable post-hoc by querying the local Audit Log via the documented CLI.
- **NFR-11.** When the Quality-Regression Evaluator rolls back a strategy, the developer SHALL receive a single non-blocking notification per rollback per repository. `[Module 5 §Observability gap 1]`

### Installation and operation

- **NFR-12.** The product SHALL install via a single bundled installer per platform (Windows, macOS, Linux), and the installer SHALL be re-runnable for upgrades.
- **NFR-13.** Uninstall SHALL restore the developer's assistant's original API base URL and SHALL remove the sidecar processes and the local audit log directory (with a confirmation prompt).

### Compatibility

- **NFR-14.** The product SHALL support, at GA: VS Code stable on Windows, macOS, Linux; IntelliJ IDEA Ultimate and Community on Windows, macOS, Linux. The supported IntelliJ Platform version range SHALL be documented.
- **NFR-15.** The product SHALL support, at GA, at least: Anthropic Claude API, OpenAI API, Azure OpenAI as assistant backends.

---

## 9 · Constraints and assumptions

**Stack constraints** (per `CLAUDE.md`): Optimizer Core and MCP tier are Go; Compression Sidecar is Python; plugins are TypeScript (VS Code) and Kotlin / Java (IntelliJ). Azure-primary if any cloud touch; no AWS, GCP, Bicep, GitLab CI, Datadog, Pulumi.

**Engine constraint:** LLMLingua-2 is the compression engine. Replacing it is a roadmap change.

**Architectural constraints:** the three locked decisions from Blueprint v0.1 — bundled sidecar, localhost loopback proxy, metadata-only agent egress — are not revisited in M2.

**Assumption A1 — Compression works on code.** LLMLingua-2 produces acceptable answer quality on code-heavy context. This is the assumption the spike (M0) measures; the M1 verdict either confirms it or kills M2. If the spike returns Red on quality, this PRD is moot.

**Assumption A2 — The four Required Fixes ship in v0.2.** The Blueprint is updated to v0.2 with Module 5 Findings 1, 2, 4, 5 fully addressed *before* M2 build begins. `[Project_Readiness_Evaluation §M2 entry pre-work]`

**Assumption A3 — Capacity.** A team of `[VERIFY]` engineers at `[VERIFY]` allocation is committed for the M2 build window (~3–5 engineer-months `[VERIFY]`). `[Roadmap M2]`

**Assumption A4 — Skill gaps closed.** The VS Code-extension-development and IntelliJ-plugin-development skills are authored in `skills-pack/.claude/skills/` before M2 build begins. `[Delivery_Plan §Capability readiness for M2]`

**Assumption A5 — Provider APIs stable.** The OpenAI-compatible and Anthropic-compatible surfaces remain backward-compatible across the build window; the 1 June 2026 Copilot billing change does not alter the API surface.

---

## 10 · Architecture overview

This PRD does not redesign the architecture — it references the canonical artifacts.

- The locked architecture is in `../design/AI_Token_Optimizer_Agent_Blueprint_v0.1.md` (to be re-baselined to v0.2 with the four Required Fixes).
- The production architecture diagram is `../design/Product_Architecture.svg`.
- The component-to-milestone mapping is in `../planning/Roadmap.md` §Component → milestone mapping.

Three properties are load-bearing for the PRD and are restated here so they cannot be lost in a design refactor:

The **Passthrough Listener** is a separately supervised process that owns the loopback port. The Optimizer Core dying does not take the assistant down with it. `[FR-13, FR-14]`

The **TLS boundary** is `http://127.0.0.1`. The product does not install a CA certificate and does not perform TLS man-in-the-middle. `[FR-20]`

The **Compression Advisor Agent** is off the inline request path and its egress is metadata only. `[FR-22]`

---

## 11 · Dependencies

| Dependency | Type | Owner | Risk |
|---|---|---|---|
| LLMLingua-2 (Microsoft, open-source) | Upstream library | external | Moderate — pin to a tested version; track upstream advisories |
| Anthropic / OpenAI / Azure OpenAI provider APIs | External service | external | Low — API stability is the assumption A5 |
| Blueprint v0.2 (with the four Module 5 fixes) | Internal | Raja | High — A2 — must complete before M2 build starts |
| VS Code extension development skill | Internal capability | Raja | High — A4 — must be authored before M2 build |
| IntelliJ plugin development skill | Internal capability | Raja | High — A4 — must be authored before M2 build |
| TLS-terminating loopback proxy skill (focused) | Internal capability | Raja | Partial — `mcp-go-server-building` touches it; warrant a focused skill |
| Spike-established per-developer baseline data | Internal data | Raja | Critical — M2 cannot calibrate its budgets without this |

---

## 12 · Success metrics

**Primary** (release-blocking at GA):

- Median per-developer token reduction ≥ **`[CALIBRATE]` %** vs. the spike-established baseline, measured over a `[CALIBRATE]`-week window post-rollout.
- Fidelity Floor regression rate ≤ **`[CALIBRATE]` %** of A/B-eligible pairs over the same window.
- Compression latency overhead < **`[CALIBRATE]` ms p95**.
- Self-funding ledger: optimizer-attributable cost ≤ **5 %** of tokens saved.

**Secondary** (tracked, not release-blocking):

- Developer adoption: ≥ **`[CALIBRATE]`** of AITO engineers using the optimizer daily by `[CALIBRATE]` weeks after GA.
- Passthrough activation rate (Listener-only forwarding) < **`[CALIBRATE]` %** of requests.
- Zero source-code egress events in the audit log over the measurement window.
- Mean time from rollback trigger to strategy disabled < **`[CALIBRATE]` seconds**.

**Health signals** (continuous):

- Audit-log size growth per developer per week.
- Approval-prompt rate per developer per day on chat requests.
- Distinct compression strategies active across the cohort (a sustained drop suggests the rollback path is firing too aggressively).

---

## 13 · Acceptance criteria — Definition of Done

M2 is "done" only when every item below is true.

1. All four Module 5 Required Fixes are implemented and reviewed against the systems-review baseline.
2. The Fidelity Floor's two-clause invariant is implemented per FR-16 and verified by an independent re-review.
3. The Quality-Regression Evaluator runs locally — no remote model — and the no-new-egress CI test passes for at least four consecutive weeks.
4. Audit log analysis over a `[CALIBRATE]`-week observation window shows **zero** source-code egress events.
5. VS Code `.vsix` and IntelliJ plugin install cleanly on Windows, macOS, and Linux; both route requests; both expose the same status surface.
6. The primary success-metric thresholds (G1, G2, G4, G6) are met on the AITO developer cohort over the post-rollout measurement window.
7. STRIDE threat-modeling review (per `mcp-go-threat-modeling`) is passed.
8. SOC 2 / ISO 27001 control mapping (per `soc2-iso27001-controls-mapping`) is recorded and signed off.
9. Operational runbook exists for: passthrough activation, strategy rollback, baseline-capture mode, uninstall.

---

## 14 · Rollout

This is the high-level rollout shape; the calendar lives in `../planning/Delivery_Plan.md`.

**Alpha.** 1–2 internal developers, `[CALIBRATE]` weeks. Exit: no source-code egress, no Core crashes that exceeded the passthrough failover budget, compression saves a positive number of tokens.

**Beta.** 5–10 developers across at least two repos. Exit: G1, G2, G4 hold on the cohort over the beta window.

**Internal GA.** Rollout to all AITO engineers. The audit log moves to mandatory review weekly; the Fidelity Floor regression metric moves to monthly leadership reporting.

**Refresh cadence post-GA.** Quarterly: review against the success metrics, retire any compression strategy that has been rolled back twice, re-baseline against current Copilot / VS Code behaviour (since the free baseline is a moving target).

---

## 15 · Open questions

These do not block this PRD draft but each must close before M2 build starts. Each is owned and dated.

- **Q1 — Calibration values.** What are the production thresholds for G1, G2, G4, G6, NFR-1, NFR-2, NFR-3, NFR-8? **Owner:** Raja. **Resolves at:** M1 gate calibration.
- **Q2 — Process supervisor.** Which Go process-supervision approach? Raja
- **Q3 — IntelliJ Platform version range.** Which IntelliJ Platform versions does the plugin support — current stable only, or N-1 too? Raja
- **Q4 — Monorepo isolation key.** For monorepos with multiple sub-projects, is the isolation key the repo root, the sub-project root, or configurable? Raja (see FR-27)
- **Q5 — Audit log retention.** What is the default local retention window for the Audit Log, and how does it interact with developer machine disk constraints? Raja
- **Q6 — Approval UX latency.** What is the maximum acceptable time-to-prompt for a chat-request approval dialog before it degrades P1's flow? Raja
- **Q7 — Telemetry off-machine.** Is there *any* AITO-internal sink the local telemetry should be allowed to reach (e.g., an internal aggregator) post-M2 — or does telemetry stay strictly on-workstation forever? **Owner:** P2 (security lead).

---

## 16 · Risks

The full risk register is in `../planning/Delivery_Plan.md §Risks and assumptions` and is not duplicated here. The four risks most likely to invalidate this PRD if they fire:

**R1 — Spike returns Red.** If the M0 spike's quality verdict is Red, M2 does not open and this PRD is moot.

**R2 — Native auto-compact closes the gap.** If VS Code's token-efficiency work plus the Copilot June 2026 billing change make the free baseline good enough that the optimizer's calibrated reduction (G1) cannot be met, the product loses its primary justification.

**R3 — Fidelity Floor cannot be made sound.** If the redesign required by Module 5 Finding 1 cannot deliver a real two-clause invariant — for example, if local deterministic quality scoring proves too weak to substitute for a remote judge model — then G2 cannot be met and the headline guarantee falls.

**R4 — Skill gaps not closed in time.** If the VS Code and IntelliJ plugin development skills are not authored before build begins, build effort doubles and the calendar slips out of the Roadmap's M2 envelope.

---

## 17 · Glossary

- **DOC (Defining Operational Constraint).** The single load-bearing invariant a system must hold to be considered conformant. For this product the DOC is the Fidelity Floor.
- **Fidelity Floor.** No compression ships without evidence it did not degrade the answer — re-stated per Module 5 Finding 1 as a two-clause invariant: predicted-safe-or-approved at decision time, retrospectively verified and reversible always.
- **Passthrough fallback.** The Passthrough Listener forwarding requests verbatim to the assistant's provider when the Optimizer Core is unhealthy.
- **MCP (Model Context Protocol).** The typed-tool protocol the agent and other in-sidecar components use to reach the system. Five allowlisted tools, one write-tier.
- **Lossy cut.** A compression decision that removes tokens whose loss could change the assistant's answer; gated by risk score and the Fidelity Floor.
- **Baseline-capture mode.** The first-run passthrough window in a new repo that establishes the per-developer token-spend baseline against which savings are measured.
- **`[CALIBRATE]`.** A value to be set at M1 gate calibration. Treated as `[VERIFY]` until then.

---

## 18 · Change log

- **v0.1 (2026-05-26).** Initial draft, gate-contingent on M1 Green. All numeric thresholds are `[CALIBRATE]` placeholders pending the M1 calibration step. Carries the four Module 5 Required Fixes as architectural givens and lists the two IDE-plugin skill gaps as M2 entry conditions.

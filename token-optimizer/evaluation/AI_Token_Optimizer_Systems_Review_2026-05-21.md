# AI Token Optimizer — Systems Review (Blueprint Conformance)

**Reviewer:** AITO AI Systems Review System (Module 5 v1.2) · **Date:** 2026-05-21
**Review mode:** Blueprint Conformance Review · **Subject:** AI Token Optimizer — Agent Blueprint v0.1

---

## Review Verdict

**Conditionally ready** — proceed to build *after* the four High-severity fixes land.

The design is structurally sound at the level that matters most: the single-agent-over-deterministic-core decomposition is correct, the deterministic-first principle is the right instinct for a token optimizer, the control plane is genuinely designed rather than labelled, and the boundary discipline (In/Out/Human-only) holds. This is not a blocked design.

It is held to *Conditionally ready* — not *Ready with monitored risks* — by one gating finding: the Defining Operational Constraint, the **Fidelity Floor**, is **not satisfiable as worded** (Finding 1). Per Module 5's gating rule, a broken or unsound DOC caps the verdict at Conditionally ready regardless of how the rest scores. Three further High findings (2, 4, 5) are structural must-fixes that would surface as production incidents if carried into build unchanged.

No finding is Critical: the optimizer takes no business action, has one local-only write tool, and exposes no single-prompt path to an unsafe action. The High findings concentrate in two places — the **proxy trust boundary** and the **evidence model behind the DOC**.

---

## Baseline Used

Primary baseline: **AI Token Optimizer — Agent Blueprint v0.1** (Module 8 baseline), with the three architecture decisions locked 2026-05-21 (bundled sidecar; localhost loopback proxy; metadata-only agent egress).

No implementation exists. This review therefore tests the blueprint as a *proposed design* — internal consistency, structural soundness, and production-credibility of the design as written — not implementation conformance. Findings identify gaps the design would carry into build.

**Defining Operational Constraint under review:** *Fidelity Floor — no compression ships without evidence it did not degrade the answer; lossless by default, lossy only with evidence and consent; self-funding (<5% overhead).*

---

## Conformance Findings

Ten findings, severity-ordered. Five High, five Medium. None Critical, none Low-padded.

### Finding 1 — DOC is not satisfiable as worded: the "evidence" is retrospective

- **Severity:** High *(gates the verdict)*
- **Evidence:** Section 7 states the DOC as "no compression ships without evidence it did not degrade the answer." But the inline decision to apply a lossy cut happens in the request path (Section 13 Mermaid), while the Quality-Regression Evaluator that produces the quality evidence runs "background, sampled" (Sections 4, 8, 13). At the moment a cut is applied, no answer exists yet and no evidence exists.
- **Why it matters:** The DOC is the load-bearing invariant of the whole blueprint. As worded it describes a property the system cannot have — evidence is always lagging and sampled, never present at the decision point. A DOC that cannot be satisfied cannot be conformed to, cannot be tested against, and cannot gate a systems review. This is not a wording nitpick: it changes what the inline gate actually is.
- **Recommended fix:** Re-state the DOC honestly as a two-clause invariant. *Decision-time:* every lossy cut is either predicted-safe by risk score below threshold, or human-approved. *Verification-time:* every compression strategy is continuously and retrospectively measured by the Evaluator, and any strategy that fails retrospective quality verification is automatically rolled back and its threshold tightened. The Fidelity Floor is then real: predicted-or-approved at decision time, *proven* over time, *reversible* always. Update Section 7 and propagate the wording to Sections 2, 8, and 12.
- **Owner:** Blueprint author (persona handoff) + Agent/Eval team.
- **Re-review trigger:** Any change to how lossy cuts are gated inline, or to the Evaluator's rollback authority.

### Finding 2 — "Sidecar down → passthrough" is contradicted by the proxy architecture

- **Severity:** High
- **Evidence:** Section 12 lists the failure mode "sidecar down → passthrough." Section 11 places the interception proxy *inside* the sidecar ("the sidecar exposes an HTTP proxy bound to `127.0.0.1:<port>`"). The assistant's API base-URL is pointed at that proxy.
- **Why it matters:** If the sidecar process dies, the proxy dies with it. The assistant's base-URL still points at a now-dead `127.0.0.1:<port>`, so the assistant's requests fail at the socket — the developer's AI assistant simply stops working. "Passthrough" is impossible precisely when it is most needed, because the component that would forward the request is the component that died. The single most important recovery path in the blueprint does not work as drawn.
- **Recommended fix:** Move the passthrough guarantee out of the failing component. Two viable options: (a) a minimal, separately-supervised always-on passthrough listener that owns the proxy port and forwards verbatim whenever the optimizer sidecar is unhealthy; or (b) the plugin client detects sidecar death and synchronously restores the assistant's original base-URL before the next request. Option (a) is more robust because it survives plugin-host crashes too. Specify one and add it to Sections 11 and 12; correct the Section 13 diagram to show the passthrough listener.
- **Owner:** Core/sidecar team + IDE client team.
- **Re-review trigger:** Any change to sidecar process topology or to how the proxy port is owned.

### Finding 3 — LangGraph agent sits on the inline request path, contradicting the blueprint's own claim

- **Severity:** High
- **Evidence:** Section 5 states "the hot path is plain Go; LangGraph governs only the background learning loop" and "where it explicitly does NOT belong: the inline hot path." But Section 13's sequence shows `Core ->> Agent: Score ambiguous segments only` *within* the request flow, before `Core ->> LLM`. The Compression Advisor Agent is invoked synchronously inside request handling.
- **Why it matters:** Two problems. First, an internal contradiction: the blueprint claims LangGraph is off the hot path while the diagram puts it there. Second, the real risk behind the contradiction — a synchronous agent call adds a full LLM round-trip to every request that contains ambiguous context, which for real codebases is most requests. For a latency-sensitive IDE assistant, prefixing each call with another LLM call can double perceived latency. Section 12's "agent timeout → deterministic-only result" mitigates the tail but not the median.
- **Recommended fix:** Resolve the contradiction by choosing the architecture, not the wording. Preferred: make the agent genuinely off-path — it observes completed requests and tunes compression *policy* for *subsequent* requests, so the inline path stays purely deterministic against the current policy. If inline agent calls are kept, state an explicit inline latency budget (e.g. p95 agent contribution `[VERIFY]` ms), a hard fast-path bypass, and accept and document the latency cost. Update Section 5, Section 13, and the poster (Section 14) to whichever is chosen.
- **Owner:** Blueprint author + Core/sidecar team.
- **Re-review trigger:** Any change to when or how the Compression Advisor Agent is invoked.

### Finding 4 — TLS interception boundary is unspecified

- **Severity:** High *(touches credentials in transit)*
- **Evidence:** Section 11 says the proxy "applies optimization and forwards the request to the assistant's real provider"; Section 6 notes the API key transits the proxy in request headers. To read and rewrite the request body, the proxy must terminate TLS. The blueprint never states how.
- **Why it matters:** Modifying an HTTPS request body requires the proxy to decrypt it. There are only two ways: (a) the assistant is pointed at a plaintext `http://127.0.0.1:<port>` endpoint and the proxy originates the real outbound TLS itself — clean, loopback-only plaintext is acceptable, but requires the assistant to accept an `http` base-URL; or (b) the proxy performs TLS man-in-the-middle, which requires installing a local CA certificate the assistant trusts — a significant security and trust action that must be a deliberate, disclosed design decision, not an implementation accident. The blueprint hides this fork. Whichever path is chosen also determines where the API key and full prompt sit in cleartext.
- **Recommended fix:** Make the TLS boundary explicit in Section 11. Strongly prefer option (a): require a configurable `http` loopback base-URL and forbid MITM certificate installation outright. If any target assistant forces (b), treat CA-cert installation as its own reviewed control with explicit user consent. Add the chosen boundary to the Section 6 control plane and the no-new-egress CI test.
- **Owner:** Core/sidecar team + Security review.
- **Re-review trigger:** Adding any assistant that does not support an `http` loopback base-URL.

### Finding 5 — The Quality-Regression Evaluator is an uncovered source-code egress path

- **Severity:** High *(data-exposure path)*
- **Evidence:** Section 8 specifies answer-quality scoring via "LLM-as-judge." The Evaluator compares the compressed-context answer against the baseline answer (Section 8, Section 13). Those answers are assistant outputs about the developer's code and routinely contain source code. The metadata-only egress decision (Section 5) is explicitly scoped to the *Compression Advisor Agent* — it says nothing about the Evaluator.
- **Why it matters:** The egress lock was the headline safety property — "no source code leaves the machine." But the Evaluator, by construction, must send full answers (containing code) to a judge model. As written, the blueprint has a code-egress path that the egress decision does not cover and the no-new-egress control (Section 6) would not catch, because that control was reasoned about only for the agent. The product's central privacy claim has a hole.
- **Recommended fix:** Extend the egress decision to the Evaluator explicitly. Preferred: run LLM-as-judge on a local model, or replace it with deterministic/structural answer-equivalence scoring (diff distance, symbol-set overlap, compile/parse checks) so no answer content egresses. If a cloud judge is unavoidable, it must route through the assistant's existing provider only, be opt-in with disclosed consent, and run on secret-redacted content. Update Sections 5, 6, and 8.
- **Owner:** Agent/Eval team + Security review.
- **Re-review trigger:** Any change to how answer quality is scored, or any new model endpoint the Evaluator calls.

### Finding 6 — Agent justification should be re-validated under the metadata-only lock

- **Severity:** Medium
- **Evidence:** Section 2 justifies the agent on "reasoning over uncertain, semantic input" plus a feedback loop. The egress lock (Section 5) then constrains the agent to metadata only — paths, symbol names, import/dependency edges, diff statistics, recency.
- **Why it matters:** Relevance scoring over *structural metadata* — symbol overlap with the query, dependency-graph reachability, recency — is largely a deterministic graph problem, not LLM reasoning. The egress lock may have quietly hollowed out the inline agent: the genuinely agentic part that remains is the feedback/learning loop, not the per-request scoring. If so, the inline agent call (Finding 3) is even harder to justify, and the design may be cleaner as "deterministic core + deterministic relevance graph + a background learning agent."
- **Recommended fix:** Before build, run Module 8's Agent Justification check again under the metadata-only constraint. Explicitly decide which part is deterministic graph logic and which part needs an LLM. This likely reinforces the Finding 3 fix (agent off the hot path).
- **Owner:** Blueprint author.
- **Re-review trigger:** N/A — one-time pre-build validation.

### Finding 7 — Approval UX conflates chat requests with inline completions

- **Severity:** Medium
- **Evidence:** Section 3 scopes in "the developer's own IDE coding-assistant requests"; the hybrid model (Sections 1, 6, 7) surfaces above-threshold lossy cuts for developer approval. The blueprint treats all assistant requests as one uniform class.
- **Why it matters:** IDE assistants serve two very different request types. Chat requests are interactive and can tolerate an approval step. Inline completions fire continuously as the developer types and must return in tens of milliseconds — an approval dialog on a completion is unworkable and would make the optimizer worse than not having it. A control that is sound for chat is actively harmful for completions.
- **Recommended fix:** Split the request taxonomy in Section 3 and the control plane. Completions: deterministic-lossless only, or pre-approved policy, never an interactive prompt. Chat: hybrid approval as designed. Make the distinction explicit in Sections 3, 6, and the Section 13 flow.
- **Owner:** Product/design + IDE client team.
- **Re-review trigger:** Adding a new assistant request type beyond chat and completion.

### Finding 8 — Per-repo / workspace isolation key is undefined

- **Severity:** Medium *(touches data boundaries)*
- **Evidence:** Section 6 asserts "per-repo isolation; context, telemetry and budgets never cross workspace boundaries." The blueprint never defines how a "repo" or "workspace" is identified.
- **Why it matters:** Isolation is only as good as its key. Identical-named repos, a monorepo containing many logical projects, a worktree, or a workspace spanning multiple repo roots all make "per-repo" ambiguous. An ambiguous key silently mixes one project's telemetry, budget, and context into another's — a data-boundary leak, even if local.
- **Recommended fix:** Define the isolation key explicitly — recommend the canonical repository root path (or its hash), with documented behavior for monorepos (sub-project granularity) and multi-root workspaces. Add it to Section 6 and the Section 12 acceptance criteria.
- **Owner:** Core/sidecar team.
- **Re-review trigger:** Supporting multi-root workspaces or monorepo sub-project budgets.

### Finding 9 — Per-developer baseline measurement is undefined

- **Severity:** Medium
- **Evidence:** Section 1 targets a 25–45% token reduction `[VERIFY]`; Section 7 targets <5% optimizer overhead `[VERIFY]`. Section 8's golden set is a generic corpus. Nothing defines how an individual developer's *own* pre-optimizer baseline is captured.
- **Why it matters:** Both headline numbers are ratios against a baseline. Without a per-developer baseline-capture mechanism, the product cannot show a developer their actual savings, cannot prove the self-funding constraint for that developer, and cannot honestly retire the `[VERIFY]` markers. The claims stay unfalsifiable in the field.
- **Recommended fix:** Specify a baseline mode — a first-run or shadow period where the optimizer measures token usage in passthrough before applying compression, establishing that developer's baseline. Add to Sections 8 and 9.
- **Owner:** Agent/Eval team.
- **Re-review trigger:** N/A — pre-build design addition.

### Finding 10 — The CI quality gate is non-deterministic

- **Severity:** Medium
- **Evidence:** Section 8 defines a CI gate that blocks a merge if answer-equivalence drops below threshold, where answer-equivalence is scored by LLM-as-judge.
- **Why it matters:** LLM-as-judge scores vary run to run. A merge gate built on a non-deterministic scorer will flake — blocking good changes and, worse, occasionally passing regressions. A flaky gate gets disabled by frustrated engineers, and then the Fidelity Floor has no enforcement.
- **Recommended fix:** Make the gate deterministic enough to trust: pin the judge model and decoding parameters, score each golden item with multiple samples and aggregate, gate on a confidence band rather than a single threshold, and pair the judge with deterministic structural checks (parse/compile, symbol-set overlap, diff distance) as the hard gate with the judge as an advisory signal.
- **Owner:** Agent/Eval team.
- **Re-review trigger:** Any change to the judge model or the gate thresholds.

---

## Structural Risks

The findings cluster into two structural themes worth naming above the line.

**The proxy is a trust-boundary concentration point.** Findings 2, 4, and 5 all trace back to one fact: the loopback proxy terminates TLS, sees every prompt and response in cleartext, holds the API key in transit, and is the single point whose failure kills the assistant. This is an acceptable architecture — an interception optimizer has to sit in the path — but it means the proxy must be treated as the highest-assurance component in the system: minimal code, no logging of bodies or headers, separately-supervised passthrough, explicit TLS handling. The blueprint currently distributes proxy concerns across Sections 6 and 11 without ever naming the proxy as the critical trust component. It should.

**The DOC's evidence model is retrospective, and the blueprint is written as if it were immediate.** Finding 1 is the cleanest case, but Findings 6, 9, and 10 are the same shape — the system's quality guarantees depend on a feedback loop that is sampled, lagging, and (today) non-deterministic. That is a legitimate way to build this product, but the blueprint must stop describing it as if quality were verified at decision time. Honesty here is not cosmetic: it determines what the inline gate is allowed to claim.

---

## Control Gaps

Against Module 5's Must-check enterprise concerns:

- **Identity and access** — adequate for a local single-user tool: loopback bind, per-session token. Re-validate the token against Finding 4's TLS decision.
- **Tool access / write-path** — strong. One write-tier tool (`record_telemetry`, local); agent reaches the system only through five allowlisted typed MCP tools. No gap.
- **Data classification / PII / isolation** — two gaps: the Evaluator egress path (Finding 5) and the undefined isolation key (Finding 8).
- **Human approval for high-risk actions** — present and meaningful, but mis-applied to completions (Finding 7).
- **Audit logging and traces** — well specified (Section 6). One addition: the audit log must record the *approval decision and the risk score* for every lossy cut, so the Fidelity Floor is auditable after the Finding 1 re-wording.
- **Evaluation and feedback loop** — present; non-deterministic gate (Finding 10) and the retrospective-evidence framing (Finding 1) are the gaps.
- **Rollback / manual override** — passthrough is the right mechanism but does not survive sidecar death (Finding 2).

No Critical control gap. The write-path discipline is the strongest part of the design.

---

## Observability and Evaluation Gaps

Telemetry design is solid — per-request traces, per-tool-call events, cost telemetry, latency percentiles, OpenTelemetry-shaped spans. Two gaps:

1. **Monitoring relies on the developer noticing degradation.** Section 9 ("the developer is their own ops dashboard in v1") is too passive for a product whose DOC is *no silent degradation*. The Quality-Regression Evaluator is the real detector — so it must surface a visible, unprompted signal when it detects a regression (a status indicator, not a buried log). Otherwise silent degradation is exactly what happens between sampled evaluations.
2. **The Evaluator's own cost is not metered.** Section 8 adds LLM-as-judge calls; Section 7's self-funding constraint accounts for the *agent's* token spend but not the *Evaluator's*. Fold Evaluator cost into the self-funding ledger or it will quietly erode net savings.

---

## Required Fixes

Priority order. The first two are low-effort design/wording fixes that unblock; they should land before anything else.

1. **Re-word the DOC (Finding 1).** Two-clause invariant: predicted-or-approved at decision time, proven retrospectively, reversible always. Documentation-only change; unblocks the verdict gate. *Do first.*
2. **Specify a passthrough path that survives sidecar death (Finding 2).** Separately-supervised passthrough listener. Design change, contained.
3. **Specify the TLS boundary (Finding 4).** Mandate `http` loopback base-URL; forbid MITM certificates. Design + compatibility-gating decision.
4. **Close the Evaluator egress hole (Finding 5).** Local judge model or deterministic structural scoring. Design change with eval-stack impact.
5. **Resolve the inline-agent contradiction (Finding 3)** and **re-validate agent justification (Finding 6)** together — decide whether the agent is on or off the hot path, then make the blueprint and diagram consistent.
6. **Split the request taxonomy (Finding 7)** — completions vs chat.
7. **Define the isolation key (Finding 8)**, **the baseline-capture mode (Finding 9)**, and **harden the CI gate (Finding 10).**

Fixes 1–4 are mandatory before build. Fixes 5–7 should be resolved before Phase 2 (the agent phase); Finding 7 specifically before any completion-path work in Phase 1.

After fixes 1–4 land and the blueprint is updated, this design clears to **Ready with monitored risks** for a phased build.

---

## Re-Review Triggers

Re-run a systems review when any of the following occurs:

- The DOC wording or the inline lossy-cut gate changes.
- The sidecar process topology or proxy port ownership changes.
- The Compression Advisor Agent moves on/off the inline path.
- Any assistant is added that cannot use an `http` loopback base-URL, or any CA-certificate installation is introduced.
- The Evaluator calls any model endpoint, or answer-quality scoring changes.
- Model routing or response caching enters scope (re-opens the single-vs-multi-agent decomposition — Section 4 deviation note).
- The optimizer gains a second write-tier tool or any new outbound network destination.
- Scope expands to application-code LLM traffic, or to a team/multi-tenant deployment.
- Phase 2 (agent) and Phase 4 (IntelliJ) entry — conformance re-check against this baseline.

---

## Handoff Note

This review is the Module 5 leg of the lifecycle handoff `Design Advisor → Blueprint Baseline → Build → Systems Review → Findings → Blueprint Update`. The next step is a blueprint update: apply Required Fixes 1–4 to `../design/AI_Token_Optimizer_Agent_Blueprint_v0.1.md`, bump it to v0.2, and the design is cleared for a phased build. Findings 5–10 are tracked against their phase gates above.

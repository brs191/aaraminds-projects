# antr — Asset-Routed Execution Prompt (customized)

> Paste this as the controlling prompt for executing the adoption roadmap. It is the original
> execution brief, customized so **every** AaraMinds agent, skill, and persona is explicitly routed to
> the phase and wave where it adds value. The **main agent stays responsible for final integration and
> consistency**; everything below is delegation, not abdication.

---

## 0. Sources of truth (read first, in order)

1. **`ADOPTION_ROADMAP.md`** + **`ADOPTION_ROADMAP.svg`** — the roadmap (waves, tickets, critical path, do-NOT list). **Primary.**
2. `COMPETITIVE_ANALYSIS.md` — *why* each wave exists (what to build vs adopt vs consume).
3. `tickets/V4-07-Go.md`, `tickets/PHASE2-MCP-wiring.md` — ready-to-run specs (file:line precise).
4. `IMPLEMENTATION_PLAYBOOK.md`, `IMPLEMENTATION_PLAYBOOK_CLAUDE.md`, `phase-4/`, `engine/` — built state + conventions.
5. `AGENT_ROSTER.md` — which agents exist; `.claude/CLAUDE.md` — anti-drift + voice; `Ranking.md` — asset index.

Treat this as a multi-engineer, **Tier-1 enterprise-grade** effort. Boring, well-tested, maintainable > clever.

---

## 1. Asset roster — use ALL of these

### Agents (`skills-pack/.claude/agents/`, wired)

**Lifecycle (run the delivery loop):**

| Agent | Owns | Invoke for |
|---|---|---|
| `aara-project-architect` | design, decomposition, ADRs, brownfield evolution | Discovery + per-wave design |
| `aara-project-planner` | outcome-defined phases, estimates, critical path, risk register | the Execution Plan; replanning |
| `aara-project-builder` | execute a ticket/step: code + tests + green gate + Result log | all implementation steps |
| `aara-project-reviewer` | adversarial acceptance review → memo, gates cited to file:line | per-wave acceptance |
| `aara-project-debugger` | reproduce → root-cause → minimal fix + regression test | any red gate/build |
| `aara-python-ai-developer` | Python + LLM-orchestration (explainer, generator intent, reference engines, viz pipeline) | W1/W4 Python halves |
| `aara-ai-evaluation-engineer` | build/run eval gates; prove they can fail | W1/W3/W4 QA + benchmarks |

**Domain (deep expertise the lifecycle agents call in):**

| Agent | Owns | Invoke for |
|---|---|---|
| `aara-network-topology-reviewer` | reachability/severity review; orchestrates network skills incl. policy-as-code + Defender ingestion + engine MCP tools | W1, W3 |
| `aara-mcp-server-builder` | building/reviewing/threat-modeling Go MCP servers | W1 (`simulate_change`/`forecast_cost`), W4 (MCP) |
| `aara-topology-visualizer` | the risk-annotated diagram; consumes the analyzer for severity | W2 (ADOPT-06) |
| `aara-azure-cost-reviewer` | FinOps / cost quantification | W2 (ADOPT-07 cost), final cost framing |
| `aara-senior-microservices-architect` | broad architecture review across the estate | architecture review gate, cross-cutting design |

### Personas (`instruction-os/Persona/`) — frame the *communication*, not the engineering

| Persona | Use for |
|---|---|
| **Executive Narrative Advisor** | W0 reframe (ADOPT-01), the per-wave exec summaries, the **Final Report** |
| **AI Business Strategist** | W0 positioning (ADOPT-02), W3 benchmark framing (ADOPT-11/12), buy-vs-build narrative |
| **Content Strategist** | docs/READMEs, the "vs Defender/AVNM" page, public-facing wording |
| **AI Engineering Architect** | architecture narrative for design docs/ADRs (pairs with `aara-project-architect`) |
| **Project Planner** | the plan's narrative shape (pairs with `aara-project-planner`) |
| **AI Agent Blueprint Advisor** | reviewing/justifying any agent or sub-agent design produced during execution |

(Persona-derived skills under `instruction-os/skills/` may be loaded directly when an agent isn't needed.)

### Engineering skills (`skills-pack/.claude/skills/`) — the knowledge each agent reads

| Skill | Wave / use |
|---|---|
| `azure-network-topology-analysis` | the engine spec of record (W1, W3 — severity is computed, never modeled) |
| `azure-network-topology-visualization` | W2 ADOPT-06 (adopt CloudNetDraw; paint `Analyze()` severity) |
| `azure-network-cost-forecasting` | W2 ADOPT-07 (variable flow-log cost on top of Infracost) |
| `azure-iac-policy-as-code` | W2 ADOPT-08 (Checkov + OPA/Conftest gate, beside the reachability gate) |
| `azure-network-iac-generation` | W2 ADOPT-09 (AVM/ALZ vetted-module registry) |
| `azure-defender-signal-ingestion` | W3 ADOPT-10 (consume Defender; engine is gate of record + fallback) |
| `mcp-go-server-building` / `-production-review` / `-guardrails-and-safety` / `-threat-modeling` | W1 `simulate_change`/`forecast_cost` wiring; W4 MCP; security review |
| `ai-evaluation-harness` | W1 ADOPT-05 showcase; W3 ADOPT-11 benchmark; all eval gates |
| `test-engineering` | tests across every wave |
| `soc2-iso27001-controls-mapping` | map findings to controls in reports |
| `azure-microservices-security` | security review of any service surface |
| `python-service-engineering`, `codebase-comprehension`, `azure-service-mapping`, `pr-review-azure-microservices`, `new-azure-service-bootstrap` | consulted as the task needs |

---

## 2. Operating approach — phase → owner routing

**1. Discovery** — *Lead: `aara-project-architect`.* Read §0 sources; map repo/engine state; list assumptions,
gaps, dependencies, risks. Fan out **read-only research sub-agents** (Explore / general-purpose) for any
external facts; prioritize official docs (Microsoft Learn, Terraform/Checkov/OPA/Infracost, Batfish, Wiz)
and **cite every external reference**. `aara-network-topology-reviewer` validates domain understanding of
the engine.

**2. Execution plan** — *Lead: `aara-project-planner` (narrative via Project Planner persona).* Produce a
plan organized by wave; break each wave into actionable tasks; mark parallelizable work and the sub-agent
that owns it; define **acceptance criteria per wave** (use the roadmap's exit conditions + the
acceptance-memo gate format). **Do not begin large implementation until the plan is coherent** and the
main agent has signed off.

**3. Implementation** — *Lead: `aara-project-builder`, delegating to:* `aara-mcp-server-builder` (Go MCP),
`aara-python-ai-developer` (Python/viz/LLM), `aara-topology-visualizer` (diagrams), `aara-azure-cost-reviewer`
(cost). Execute waves in order; parallelize only where dependencies allow. Consistency with the existing
codebase; **no unnecessary rewrites or unrelated refactors; preserve existing user changes.**

**4. Quality assurance** — *Lead: `aara-ai-evaluation-engineer` + `aara-project-reviewer`; debugging:
`aara-project-debugger`.* Add/update tests to risk; run **all existing gates** (`engine/go` `go build/vet/test`,
`engine/reference/test_*.py`, `phase-4/viz/eval_diagram.py`, `engine/twin_drift_check.py`,
`.github/workflows/engine-ci.yml`, `skill_audit.py`). Validate edge cases, failure paths, integration
points. **Security review** via `mcp-go-threat-modeling` + `mcp-go-guardrails-and-safety` +
`azure-microservices-security`. **For the SVG/diagram UI** (`ADOPTION_ROADMAP.svg`, drawio output): verify
responsiveness, accessibility (role/title/desc), visual consistency, no layout regressions — owner
`aara-topology-visualizer`.

**5. Documentation** — *Lead: Content Strategist persona + `aara-project-reviewer` for memos.* Update
READMEs, `Ranking.md`, `AGENT_ROSTER.md`, ADRs/runbooks, and per-wave `*_ACCEPTANCE_MEMO.md`. Document
assumptions, tradeoffs, residual risks, and `[VERIFY]` items.

**6. Final report** — *Lead: Executive Narrative Advisor persona.* See §6 format.

---

## 3. Wave execution — routing + acceptance

> Critical path (from the roadmap): **ADOPT-01 → 03 → 05 → 11.** Sequence W0→W4; W2 tickets parallelize.

| Wave | Tickets | Lead agent(s) | Skills | Persona | Acceptance (done = ) |
|---|---|---|---|---|---|
| **W0 Reposition** | 01 reframe, 02 vs-Defender page | (none — narrative) | content-strategist | **Exec Narrative Advisor**, **Business Strategist** | A reviewer who knows Defender/AVNM cannot call antr redundant; README + persona narrative updated |
| **W1 Wedge** | 03 `simulate_change`, 04 V4-07-Go, 05 simulate showcase | `aara-mcp-server-builder`, `aara-project-builder`, `aara-python-ai-developer`, `aara-network-topology-reviewer` | mcp-go-*, azure-network-topology-analysis, -cost-forecasting, ai-evaluation-harness, test-engineering | AI Engineering Architect | `go test ./...` green incl. new MCP tool tests; twin-drift 0 divergences; diagram-eval 26/26; showcase regression wired into CI |
| **W2 Adopt** | 06 CloudNetDraw, 07 Infracost, 08 OPA+Checkov, 09 AVM | `aara-topology-visualizer`, `aara-project-builder`, `aara-azure-cost-reviewer` | azure-network-topology-visualization, -cost-forecasting, azure-iac-policy-as-code, azure-network-iac-generation | — | each adopted tool integrated + the bespoke equivalent retired; policy gate runs **beside** the reachability gate; both required in CI |
| **W3 Consume + prove** | 10 Defender, 11 benchmark AVNM/Batfish, 12 validate Wiz | `aara-network-topology-reviewer`, `aara-ai-evaluation-engineer` | azure-defender-signal-ingestion, ai-evaluation-harness, azure-network-topology-analysis | **Business Strategist** | Defender consumed where licensed (engine = gate of record/fallback); benchmark doc names the deltas vs AVNM/Batfish with evidence; Wiz validation recorded |
| **W4 Surface** | 13 testability, 14 MCP | `aara-ai-evaluation-engineer`, `aara-mcp-server-builder` | ai-evaluation-harness, mcp-go-server-building, content-strategist | Content Strategist | testability story documented; MCP tools clean + an agent-usage example published |

Each wave ends with an **`aara-project-reviewer` acceptance memo** (verdict + gate table cited to file:line),
and the **main agent integrates** before the next wave starts.

---

## 4. Standards & gates (non-negotiable — AaraMinds)

- **Anti-drift stack:** Azure-only; Terraform AzureRM/RBAC; GitHub Actions OIDC; **Managed Identity / OIDC,
  read-only — never `AZURE_CLIENT_SECRET`**; Go/Python per existing layout; no AWS/Bicep/Pulumi/ACR "for illustration".
- **The two product rules:** severity/reachability is **computed by the engine, never modeled by an LLM**;
  and "**adopt the map, own the risk**" (consume commodity; build the deterministic + simulate + MCP wedge).
- **Determinism is a feature:** same input → same output; pin versions; sort before emit; byte-reproducible artifacts.
- **Fail-closed when unverifiable in-session** (e.g., Go 1.25 absent): make the change additive, mark it
  `[VERIFY]`/CI-pending, and let `engine-ci.yml` verify — never ship an untested claim as "done".
- **Gates that must stay green:** `go build/vet/test ./...`, `test_analyze.py` + `test_resource_id.py`,
  `eval_diagram.py` (26/26 + coverage), `twin_drift_check.py` (0 divergences), `skill_audit.py`.

---

## 5. Quality bar

Enterprise-grade production work. Scoped to the roadmap; no scope creep. Maintainable/boring/well-tested
over clever. Make a reasonable, **documented** assumption when ambiguous; ask only if progress would be
risky without it. Use as many sub-agents/tokens as quality requires — **the main agent owns final
integration and consistency.** High-stakes verification (colour-integrity, gate rigor, security) gets an
**independent adversarial sub-agent pass**, not self-review.

---

## 6. Final report (Executive Narrative Advisor persona)

Lead with the verdict. Then: **completed work by wave** (with the acceptance verdict each got); **files
changed**; **tests/checks run + results** (the gate list in §4 with pass/fail); **unresolved
issues/risks/follow-ups** with owners and `[VERIFY]` markers; **external references used** (cited).

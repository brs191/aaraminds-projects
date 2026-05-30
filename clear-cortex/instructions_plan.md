# clear-cortex — Instructions & Execution Blueprint

**The one file to open when you forget where you are.** It tells you which skills, agents, and personas to load in each phase, what to produce, when the gate passes, and which temptations to refuse.

**Subject:** `apm0045942-credit-routing-service` @ `e17fe410` (branch `develop`) · **Method:** Code Intelligence Factory, adapted
**The one rule above all:** the code repo is **read-only**. Every artifact lives in `clear-cortex/`, never in the repo.

---

## How to use this file

1. Open `tracking/Status.md` → read the **active phase**.
2. Load the **Standing context** (below) — every session, no exceptions.
3. Jump to that phase's block here. Do **only** what it says.
4. Before you act on any impulse not in the active phase, check **Stay-on-track guardrails**. If it's on the refuse list, don't.
5. At session end, tick the milestone checkboxes and update `Status.md`.

> Each phase block below ends with a **Sample prompt** — copy it to kick that phase off. The session-load prompt is under *How to invoke an asset*.

---

## Standing context — load every session

| What | Which asset | Why |
|---|---|---|
| Method | skill `codebase-comprehension` | The operating manual for the whole effort. Already distilled into `design/Method_Adaptation.md` — re-read that if you don't have the pack open. |
| Voice | persona `AaraMinds_AI_Engineering_Architect_v1.2` (compose per its `## Composition` section) — or the persona-skill `aaraminds-ai-engineering-architect` | Holds the work at architectural altitude and applies the AaraMinds gates (no fabricated metrics, brownfield-first, lead with the verdict). |

**The two disciplines that never relax:**
- **Deterministic vs. inferred, never blurred.** Parsed fact = ground truth; judgement = hypothesis, marked as inference with a confidence band.
- **Evidence anchor on every non-trivial claim** — `file › Type#member › L<start>–<end>`, provenance, confidence.

---

## How to invoke an asset (you don't create anything)

Every asset here **already exists** in the pack — you never build a persona or an agent for this work. Three ways to invoke:

- **Skill** (e.g. `codebase-comprehension`) — name it in your prompt, or just rely on the SKILL.md content already distilled into `design/Method_Adaptation.md`. Nothing to create.
- **Persona** (e.g. AI Engineering Architect) — load `instruction-os/Persona/AaraMinds_AI_Engineering_Architect_v1.2.md` (with its `## Composition` modules), or invoke the persona-skill `aaraminds-ai-engineering-architect`.
- **Agent** (e.g. `aara-senior-microservices-architect`) — dispatch it as a subagent in a session where the pack is registered; it auto-routes to the right skills.

> **No — you do not create a persona or agent for `codebase-comprehension`.** It is already a skill in the pack. Load it by name, or let the architect agent route to it.

**Sample prompt — start of every session:**
```text
Load the AaraMinds AI Engineering Architect persona
(instruction-os/Persona/AaraMinds_AI_Engineering_Architect_v1.2.md + its Composition modules)
and the codebase-comprehension skill.

We're doing a CIF-style architecture comprehension of apm0045942-credit-routing-service @ e17fe410,
working ONLY in clear-cortex/ — the code repo is read-only.

Open clear-cortex/tracking/Status.md and clear-cortex/instructions_plan.md, tell me the active
phase, and restate the two standing rules (deterministic vs inferred; evidence anchor on every
non-trivial claim) before we start.
```

---

## Asset → phase quick map

| Phase | Load (skills) | Persona | Agent | Gate-time |
|---|---|---|---|---|
| **P0 Foundations** | `codebase-comprehension` | `aaraminds-project-planner` (sequencing sanity-check) | — | — |
| **P1 Breadth map** | `codebase-comprehension` | `aaraminds-ai-engineering-architect` | `aara-senior-microservices-architect` (optional driver) | `microservices-architecture-reviewer` |
| **P2 Deepen** | per area (see block) | `aaraminds-ai-engineering-architect` | `aara-senior-microservices-architect` | `microservices-architecture-reviewer` |
| **P3 Finalize** | `ai-evaluation-harness` | `AaraMinds_Executive_Narrative_Advisor` (optional exec summary) | — | `microservices-architecture-reviewer` + 2nd human reviewer |

---

## Phase blueprints

### P0 — Foundations  ·  ~0.5–1 day
**Load:** `codebase-comprehension` (to frame what counts as ground truth) · `aaraminds-project-planner` (confirm sequencing/effort).
**Do:**
- Take a **read-only working copy** at `e17fe410`. Pin the SHA in `evaluation/HLD.md` Document Control.
- `docker-compose up -d` → `./mvnw clean compile`. Confirm `target/generated-sources` is populated (MapStruct, SOAP, OpenAPI) — generated code must exist before you read structure.
- Mine `README.md`, `.github/copilot-instructions.md`, `Credit.yaml`, `application.yml` into `evaluation/Code_Briefing.md` §0–§1.
**Produce:** a compiling working copy + Briefing raw-material captured.
**Deliverable looks like:** a working copy that builds clean and a Briefing with its first facts logged.
- `./mvnw clean compile` → `BUILD SUCCESS`; `target/generated-sources/` holds MapStruct `*MapperImpl.java`, JAX-WS SOAP stubs, and OpenAPI models.
- `HLD.md` §1 reads: subject = Credit Routing Service, commit = `e17fe410`, scope = whole service, milestone = breadth.
- `Code_Briefing.md` §0–§1 has located facts, e.g. *"Persistence = MongoDB"* → `application.yml › spring.data.mongodb` · *"32 `@Document` classes [deterministic]"*.
**Validate it:**
- `git -C <working-copy> rev-parse HEAD` returns `e17fe410`; build exits 0; `find target/generated-sources -name '*.java' | head` is non-empty.
- Open 3 facts in `Code_Briefing.md` §0–§1 — each cites a file that actually exists.
- Code repo untouched: `git -C <repo> status --porcelain | grep '^??'` is empty.
**Gate:** SHA pinned · repo compiles · existing-doc facts captured · template + rubric accepted.
**Stay on track:** Do **not** start writing the HLD or "extracting" yet. Do **not** modify the repo. P0 is only about a clean, reproducible footing.
**Sample prompt — P0:**
```text
Using codebase-comprehension, set up P0 foundations for clear-cortex.
1. Confirm the working copy of apm0045942-credit-routing-service is at e17fe410; record it in
   clear-cortex/evaluation/HLD.md §1 Document control.
2. Build it: docker-compose up -d, then ./mvnw clean compile — confirm target/generated-sources
   has the MapStruct / SOAP / OpenAPI output.
3. Read README.md, .github/copilot-instructions.md, Credit.yaml, application.yml and capture the
   facts (with file locators) into clear-cortex/evaluation/Code_Briefing.md §0–§1.
Do NOT write the HLD yet and do NOT modify the repo. Stop at the P0 gate and tell me if it passes.
```

### P1 — Breadth map (whole service, shallow)  ·  ~2–4 days
**Load:** `codebase-comprehension` + `aaraminds-ai-engineering-architect`. Optionally drive the pass with the **`aara-senior-microservices-architect`** agent (it auto-routes the architecture skills). At the gate, switch on **`microservices-architecture-reviewer`** for a verdict review.
**Do:**
- Inventory all 14 packages; start with **`routing/`** (core) and **`admin/rules`** (the DSL engine).
- **Decode `cas`, `ubct`, `iebus`**; confirm `csi` = Credit Services Integration.
- REST surface: 28 controllers / ~107 endpoints, v1 vs v2 — cross-check `Credit.yaml`.
- Mongo: 32 `@Document` collections + 29 repos (relationships inferred). Integrations: CSI/SOAP, IEBus/Kafka, OIDC, `ubct`. Cross-cutting incl. the AOP aspects.
- Fill `Code_Briefing.md` (breadth), `Inferred_Product_Spec.md` (breadth), and `HLD.md` §§1–11 at component altitude; §9 checklist filled; §10 seeded.
**Produce:** a coherent **shallow whole-service HLD** + a **ranked deepen list**.
**Deliverable looks like:** a few-page whole-service HLD a new engineer could read in one sitting — every major part named, nothing deep yet.
- `Code_Briefing.md`: 14 packages with roles; 28 controllers / ~107 endpoints (v1 vs v2); 32 `@Document` collections; `cas`/`ubct`/`iebus` decoded; shallow areas tagged `[not deep-read]`.
- `HLD.md` §5 names components, §9 marks every concern, §10 seeds the obvious decisions. A claim reads like: *"The v2 API exposes single- and multi-product credit checks [E3]"* with an Evidence row — `routing/v2/.../CreditCheckController.java › L40–58 · deterministic · high`.
- A **ranked deepen list** at the foot of the Briefing (the P2 input).
**Validate it:**
- Run the **P1-gate review prompt** (`microservices-architecture-reviewer`) against `Evaluation_Rubric.md`: completeness, architectural correctness, altitude.
- **Critical-error check:** verify ~10 structural claims against the code — a single fabricated component / flow / integration fails the phase.
- **No-silent-omission:** every §9 row is marked Covered / Not visible / Out of scope — no blanks.
- **Provenance + counts:** every inferred claim is phrased as inference and tagged; controller / endpoint / collection counts match a quick `grep` of the repo.
**Gate:** completeness (every major component + integration named) · sound decomposition · **zero fabrications** · anchors on non-trivial claims · altitude held.
**Stay on track:** **Breadth, not depth.** Mark `[not deep-read]` and move on — do not rabbit-hole a subsystem. Timebox acronym decoding. No LLD. No pipeline build.
**Sample prompt — P1 kickoff (drive with the agent):**
```text
Dispatch the aara-senior-microservices-architect agent (it routes to codebase-comprehension) for
the P1 breadth map.

Inventory all 14 packages of apm0045942-credit-routing-service @ e17fe410 at COMPONENT altitude —
start with routing/ and admin/rules. Decode cas, ubct, iebus; confirm csi = Credit Services
Integration. Catalogue the 28 controllers / ~107 endpoints (v1 vs v2; cross-check Credit.yaml),
the 32 @Document collections, and the integration + cross-cutting (AOP aspect) surfaces.

Write breadth-level clear-cortex/evaluation/{Code_Briefing.md, Inferred_Product_Spec.md, HLD.md}.
Tag every fact deterministic vs inferred with an evidence anchor; mark [not deep-read] where
shallow — breadth, not depth. Finish with a RANKED list of areas to deepen.
```
**Sample prompt — P1 gate (verdict review):**
```text
Switch on microservices-architecture-reviewer. Review clear-cortex/evaluation/HLD.md (breadth
draft) against the gate in clear-cortex/evaluation/Evaluation_Rubric.md: completeness,
architectural correctness, ZERO fabricated components/flows/integrations, evidence anchors,
altitude. Give a pass/fail verdict with specific fixes, and confirm or re-order the deepen list.
```

### P2 — Deepen highest-value areas  ·  ~1–2 days each, one area at a time
Work the **ranked list from the P1 gate** in order. Per area, add the matching skill on top of the standing two:

| Deepen area | Add these skills |
|---|---|
| 1. Core credit-check runtime flow (v2) | `microservices-async-messaging` (the IEBus/Kafka publish path) |
| 2. DSL rules engine (`admin/rules`) | `test-engineering` (read the 209 Spock tests as a behavior oracle) |
| 3. Domain & data model (32 Mongo collections) | `azure-data-tier-design` + `data-access-engineering` |
| 4. External integrations (CSI/SOAP, IEBus/Kafka, OIDC, `ubct`) | `microservices-async-messaging` + `azure-microservices-security` + `azure-microservices-observability` |
| 5. v1↔v2 + multi-product; `admin/` surface | (per the P1 deep-vs-catalogue decision) |

**Do (per area):** extend `Code_Briefing.md` with deep-read facts → upgrade the matching `HLD.md` section inferred→evidence-backed → add step-level runtime detail → complete §10 decision records.
**Deliverable looks like (per area):** the chosen area goes from sketch to evidenced. For the core flow, `HLD.md` §7 reads as a numbered runtime path — *controller → routing → `admin/rules` DSL eval → policy → result → IEBus publish* — each step anchored to a file/line; §10 carries the eventing-wrapper decision record (observed / evidence / likely rationale `inferred` / trade-off); that area's `[not deep-read]` markers are gone.
**Validate it (per area):**
- Re-run `microservices-architecture-reviewer` on that section: altitude held, accuracy, conformant anchors.
- **Anchor resolution:** open 5 anchors in the section — each resolves to the cited lines at `e17fe410`.
- **Behavior cross-check:** corroborate flow / rules claims against the 209 Spock tests (a test that exercises the path).
- Every new inference carries a confidence band; no deterministic/inferred blur.
**Gate (per area):** altitude + accuracy + evidence bars met; every inference carries a confidence band.
**Stay on track:** **One area at a time.** Deepen only what P1 ranked. Honor the `admin/` decision — don't deep-read 157 files on a whim.
**Sample prompt — P2 (template; one area at a time):**
```text
Deepen ONE area: <area name from the P1 ranked list>.
Load codebase-comprehension + <the area's skill(s) from the P2 table above>; optionally dispatch
aara-senior-microservices-architect.

Deep-read the relevant code in apm0045942-credit-routing-service @ e17fe410. Extend
clear-cortex/evaluation/Code_Briefing.md with deep facts (file locators), then upgrade the matching
clear-cortex/evaluation/HLD.md section from inferred to evidence-backed: add step-level runtime
detail and complete the §10 decision record (observed / evidence / likely rationale [inferred] /
trade-off). Stop at the area gate.
```
**Sample prompt — P2 worked example (area 1, the core flow):**
```text
Deepen area 1: the v2 credit-check runtime flow. Load codebase-comprehension +
microservices-async-messaging.

Trace a credit-check request end to end: routing/v2 controller → routing → admin/rules DSL
evaluation → policy → creditcheckresult → the IEBus/Kafka publish path
(iebus/servicebus/MessageBrokerClient). Write it as HLD §7 at component altitude with evidence
anchors; record the eventing-wrapper decision in §10. Mark anything unread [not deep-read].
```

### P3 — Consolidate, verify, finalize  ·  ~1–2 days
**Load:** `ai-evaluation-harness` (scoring discipline; pairs with `evaluation/Evaluation_Rubric.md`) · `microservices-architecture-reviewer` (independent verdict) · Module `02_Visual_Identity_System` for the diagrams · optionally `AaraMinds_Executive_Narrative_Advisor` for an exec-facing one-pager.
**Do:** assemble the three artifacts; produce the architecture + credit-check-v2 runtime-flow diagrams in `design/`; **self-score with the Scorecard**; spot-check ~15 evidence anchors against real code at `e17fe410`; run the §9 no-silent-omission check; seed §11 observations from `appcat`.
**Deliverable looks like:** sign-off-ready. Final `HLD.md` (whole-service, every scoped `[not deep-read]` resolved, claims anchored), plus `Code_Briefing.md` and `Inferred_Product_Spec.md`; two diagrams in `design/` (architecture + credit-check-v2 flow); a completed Scorecard with per-dimension scores and a weighted total; §11 observations seeded from `appcat`.
**Validate it:**
- **Scorecard PASS:** total ≥ 70/100, factual accuracy ≥ 3/4, no zero dimension, critical-error rule not triggered.
- **Anchor audit:** ~15 anchors spot-checked resolve to real code at `e17fe410`.
- **Second human reviewer** scores Part B independently; reconcile any dimension differing > 1 point; record both raw scores.
- **Diagram ↔ prose:** no component or flow appears in a diagram that isn't in §5 / §7.
- Exec one-pager, if produced: decision-relevant, zero fabricated metrics.
**Gate:** Scorecard **PASS** (total ≥ 70/100, factual accuracy ≥ 3/4, no zero dimension, critical-error rule not triggered) **and a second human reviewer concurs** (reconcile differences > 1 point).
**Stay on track:** Observations ≠ modernization execution. A self-score is not a sign-off — the second reviewer is required.
**Sample prompt — P3 verify & score:**
```text
Load ai-evaluation-harness. Self-score clear-cortex/evaluation/HLD.md with the Scorecard in
Evaluation_Rubric.md (six dimensions; gate = total ≥ 70, accuracy ≥ 3, no zero dimension,
critical-error rule clear). Then spot-check 15 evidence anchors against the real code at e17fe410
and run the §9 no-silent-omission check. Report the filled scorecard and any failing anchors — and
flag that a self-score is NOT sign-off; a second human reviewer is still required.
```
**Sample prompt — P3 diagrams + exec summary (optional):**
```text
Load Module 02 Visual Identity and produce two diagrams into clear-cortex/design/: an
architecture/component view and the credit-check-v2 runtime flow.
Then load the AaraMinds Executive Narrative Advisor and write a one-page, VP-ready summary of the
HLD — decision-relevant, no fabricated metrics.
```

---

## Agents — what's runnable, and the gap

**Runnable now** (`skills-pack/.claude/agents/`): use **`aara-senior-microservices-architect`** (opus) as the P1–P2 driver. `aara-mcp-server-builder` and `aara-azure-cost-reviewer` are not relevant to this engagement.

**CIF-specific agents exist only in Copilot format** (`skills-pack/copilot/agents/`) — `aara-code-model-designer`, `aara-codebase-extraction-engineer`, `aara-ai-evaluation-engineer`, plus the `aara-agent-lastbatch-cif.zip` batch. They are **not** runnable Claude subagents. You do **not** need them for this manual comprehension. Port `aara-codebase-extraction-engineer` + `aara-code-model-designer` to `.claude/agents/` **only if** you later decide to build the extraction pipeline (a separate project — see guardrails).

---

## Stay-on-track guardrails (the refuse list)

When tempted, check here first. These are the ways this work goes off the rails:

- **Comprehension, not construction.** No Eclipse-JDT → graph → generator pipeline. That's a separate project.
- **HLD altitude, not LLD.** No class-by-class dumps. Name a type only when it anchors a boundary, aggregate, lifecycle, or flow.
- **Document as-is.** No rewrite, no redesign, no modernization *execution* — observations only (§11).
- **One repo, one SHA** (`e17fe410`). No multi-repo. If `develop` moved and you must re-pin, do it deliberately and note it.
- **Breadth before depth; one deepen-area at a time.** Deepen only the P1-ranked list.
- **Gates, not checkboxes, mean done.** A phase with every box ticked is still open until its gate passes.
- **Repo is read-only.** Build/inspect from a working copy; all output in `clear-cortex/`.
- **Every claim:** deterministic-vs-inferred tagged + evidence anchor. An unevidenced claim is not a usable claim.
- **Validation opportunity:** every skill/agent is `strength: n/t` (never live-tested). This is the first real run of the comprehension chain — record what worked into `instruction-os/Persona/Validation_History.md`.

---

## Session-start checklist (copy this each time)

```
[ ] Opened tracking/Status.md — active phase is: ______
[ ] Loaded standing context: codebase-comprehension + AI Engineering Architect
[ ] Re-read this phase's block + the guardrails
[ ] Working ONLY the active phase's tasks
[ ] (end) Updated milestone checkboxes + Status.md
```

---
_Companion files: `tracking/Status.md` (where you are), `planning/Roadmap.md` (the full plan), `design/Method_Adaptation.md` (the method), `evaluation/HLD_Template.md` + `Evaluation_Rubric.md` (the quality bar)._

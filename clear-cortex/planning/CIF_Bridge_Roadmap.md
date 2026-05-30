# Bridge Roadmap — clear-cortex ↔ Code Intelligence Factory

**Purpose:** make the link between this comprehension engagement (`clear-cortex`) and the CIF product (`../../code-intelligence-factory`) explicit, and sequence how one feeds the other.
**Date:** 2026-05-30 · **Owner:** Raja

## In one line

`clear-cortex` is the **manual, single-repo proof** of the CIF method; `code-intelligence-factory` is the **automated platform** that productizes it. The credit-routing service is CIF's natural **pilot repo**, and clear-cortex's P3 HLD is CIF's **first golden-HLD fixture**.

## Where the three pieces sit

| Folder | Role |
|---|---|
| `aaraminds-delivery/product-research/Code Intelligence Factory` | The research / spec. Both projects below descend from it. |
| `aaraminds-projects/code-intelligence-factory` | The **product build** (Track B): a traceability platform — 5 agents around the spine `BR→HL→LL→US→PR→TC→DEF→GAP`, Go MCP servers + LangGraph. Stage: blueprint, building Phase 0. |
| `aaraminds-projects/clear-cortex` | The **manual comprehension** (Track A) of `apm0045942-credit-routing-service`. Stage: entering P0. |

## Artifact mapping — clear-cortex hand-produces CIF's Phase-1 outputs

clear-cortex is, in effect, a human doing CIF's **Reverse Engineering + the HLD slice of the Business Analyst**, on one repo, by hand — same evidence/provenance discipline, same target stack (Spring Boot · MongoDB).

| clear-cortex artifact (manual) | CIF component / output | CIF phase·gate |
|---|---|---|
| `Code_Briefing.md` (deterministic facts) | Reverse Engineering → **System Model** | Ph 1 · G1 |
| `Inferred_Product_Spec.md` (capabilities, inferred) | Business Analyst → **BRD seed** (`INFERRED` reqs) | Ph 1 · G2 |
| `HLD.md` (whole-service, evidence-anchored) | Business Analyst → **HLD** (`HL-` elements) | Ph 1 · G2 |
| Evidence anchors (deterministic/inferred + confidence) | `source_refs` + `INFERRED`/`CONFIRMED` provenance rule | spine invariant |
| *(not produced by clear-cortex)* | LLD, User Stories, Defect→Gap | Ph 1–3 (CIF only) |

## The roadmap — two tracks, one convergence

```
Track A · clear-cortex (manual)      P0 ─ P1 ─ P2 ─ P3 ──► GOLDEN FIXTURE
 apm0045942-credit-routing-service                            │
                                                              │ (golden HLD + briefing + rubric)
                                                              ▼
Track B · code-intelligence-factory  Ph0 spine ─────► Ph1 RE+BA on the pilot ─► Ph2 Jira/GitHub ─► Ph3 Gap
 (the platform)                      (no dependency)   ▲ scored against the fixture
                                                       └── CONVERGENCE POINT
```

- **Track A — clear-cortex** (now → ~2–3 wks at 1 FTE): P0→P3 produce the golden `Code_Briefing` + `Inferred_Product_Spec` + `HLD` for the credit-routing service (Scorecard PASS, second-reviewer sign-off).
- **Track B — code-intelligence-factory** (parallel):
  - **Phase 0 — spine.** Independent of clear-cortex. Build the traceability service / manifest / graph anytime; **no dependency**.
  - **Phase 1 — RE + BA on the pilot.** *Convergence.* Adopt the credit-routing service as the pilot repo; use clear-cortex's golden HLD + Code Briefing as the **ground truth** to evaluate RE's System Model (G1) and BA's HLD/BRD (G2).
  - **Phase 2–3** extend past where clear-cortex stops: Jira/GitHub wiring, LLD, User Stories, then Defect→Gap analysis.

## The golden-HLD-first dependency (why the order is fixed)

CIF Phase 1's BA agent *generates* an HLD. You can only know whether it's right by scoring it against a hand-built **golden HLD** — and that golden HLD is clear-cortex's P3 deliverable. Therefore:

- **clear-cortex P3 unblocks CIF Phase 1's honest evaluation.** You can build CIF's spine and agents anytime, but you cannot *validate* RE + BA on this repo until the golden HLD exists.
- This mirrors the CIF research's own M0→M1 gating (golden HLD before the graph). The dependency is **one-way**: Track A → Track B, never the reverse.

## The handoff contract — what clear-cortex hands CIF, and when

At **clear-cortex P3 done**, register these as CIF's first reference fixture for `apm0045942` (a Spring Boot + MongoDB domain):

| clear-cortex file | Becomes CIF's… |
|---|---|
| `evaluation/HLD.md` | golden HLD for the pilot (BA / G2 scoring target) |
| `evaluation/Code_Briefing.md` | ground truth for RE's System Model (G1) |
| `evaluation/Inferred_Product_Spec.md` | ground truth for BA's inferred BRD (G2) |
| `evaluation/Evaluation_Rubric.md` | the scoring instrument (already CIF-derived) |

## Open decisions

1. **Is `apm0045942-credit-routing-service` the official CIF pilot repo?** (Answers CIF `ARCHITECTURE.md` §11 open question (a) "one pilot repo or several?".) *Recommend: yes — it's already on-stack and already being comprehended.*
2. **Does CIF Phase 1 wait for clear-cortex P3, or run in parallel and consume the fixture as it lands?** *Recommend: start CIF Phase 0 (spine) now — it has no dependency; gate only CIF Phase 1's evaluation on clear-cortex P3.*
3. **Owners per track.** clear-cortex: Raja (+ second reviewer, TBD). CIF: the platform build team. Name them.

## Sequencing summary

1. **Now (parallel, no dependency):** clear-cortex P0–P3 ‖ CIF Phase 0 (spine).
2. **clear-cortex P3** lands the golden fixture.
3. **CIF Phase 1** RE + BA runs on the pilot, scored against the fixture → the agents are validated on a real repo (clear-cortex tells you the right answer).
4. **CIF Phase 2–3** extend into Jira/GitHub, LLD, and Defect→Gap — beyond clear-cortex's scope.

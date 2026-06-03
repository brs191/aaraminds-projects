

P1 Prompt -

Load the AaraMinds AI Engineering Architect persona + codebase-comprehension, and dispatch
aara-senior-microservices-architect for the P1 breadth map.

WORKING COPY — pin ONE revision; every anchor comes from it:
  <chosen clone path> @ <SHA>   (workspace = e17fe410 · Mac = 44b6b86… — pick one and reconcile)
  Read generated members from target/generated-sources (MapStruct; OpenAPI clients
  crsms/creditPolicy/UnifiedCreditCheck; CSI SOAP types) and tag them [generated].

SCOPE — component altitude, breadth not depth. Inventory ALL packages; start with routing/ and
  admin/rules. Reconcile against the P0 baseline (~14 packages, 28 controllers, ~107 endpoints,
  32 @Document) and FLAG divergence. Catalogue admin/ at name level (19 controllers); only
  admin/rules goes deeper — the deep-vs-catalogue call is at the gate.

DECODE cas, ubct, iebus. Verify (don't re-derive) the P0 decodings: csi = Credit Services
  Integration (order/subscription-mgmt-mobility SOAP); CRSMS = Customer Risk System Manager
  Service; CLEAR = Credit Logging and Evaluation Assistance Repository.

CATALOGUE controllers + all endpoints (v1 vs v2 under /CreditCheck; cross-check Credit.yaml),
  all @Document collections, integration + cross-cutting (AOP aspect) surfaces.

WRITE — EXTEND, do not overwrite:
  Code_Briefing.md → fill §2–§8; preserve §0–§1 and the SHA provenance note.
  Inferred_Product_Spec.md → capabilities, actors, value flow (all inferred + confidence).
  HLD.md → fill §§2–11 at component altitude; preserve §1 + the SHA note.

DISCIPLINE — every non-trivial fact: evidence anchor (file › Type#member › L<s>–<e>),
  tagged deterministic | inferred (+confidence). Mark [not deep-read] where shallow.

FINISH with a RANKED deepen list (P2 input). Do NOT self-certify the gate — that's the
  separate reviewer pass.


P1 Gate Prompt -

Dispatch a FRESH microservices-architecture-reviewer subagent (independent — it did NOT author
the HLD) for the P1 gate. Be adversarial.

INPUTS: clear-cortex/evaluation/HLD.md (deliverable) + clear-cortex/evaluation/Code_Briefing.md
(the evidence layer — per-claim file/line anchors live here; the HLD uses claim-cluster refs into
it). GROUND TRUTH = source at the e17fe410 workspace clone
(coderepos/clear/apm0045942-credit-routing-service).

VERIFY AGAINST CODE (not plausibility): open ~12 structural claims spanning components, the
runtime flow, the data model, integrations, and §9 — confirm each anchor resolves and says what's
claimed. Fresh-grep the headline counts (~89 endpoints, 28 collections, 11 aspects, 14 packages).
Hunt for any FABRICATED component / flow / integration — one triggers the critical-error rule and
FAILS the document.

SCORE with the full Evaluation_Rubric.md scorecard — six dimensions (Factual accuracy 30,
Completeness 20, Architectural correctness 20, Altitude 10, Clarity 10, Evidence 10), 0–4 each →
weighted total /100. Apply ONLY absolute gate (a): total ≥ 70, accuracy ≥ 3, no zero dimension,
zero fabrications. The §5b no-graph-margin bar does NOT apply (hand-written HLD). Score
MILESTONE-AWARE (P1 breadth — do not penalize intentional P2/P3 depth deferral or [not deep-read]).
No golden HLD exists yet — score on the anchored scales against the code.

OUTPUT: the filled scorecard + PASS/FAIL verdict; gate-blocking fixes vs. nice-to-haves; and a
confirm-or-reorder of the Code_Briefing §9 deepen list. Flag this as the assistive single-reviewer
pass — the formal gate still needs a second human reviewer (rubric §6).




P2 Prompt -

Load the AaraMinds AI Engineering Architect persona + codebase-comprehension (plus the area skill
below), and dispatch deep-read subagents in parallel, then synthesize. DEPTH, not breadth —
deepen ONE ranked area at a time.

WORKING COPY — same pinned revision as P1; every anchor comes from it:
  coderepos/clear/apm0045942-credit-routing-service @ e17fe410
  (Mac pin 44b6b86… still unreconciled — author against e17fe410 and carry the SHA note.)

AREA — take the next from the GATE-CONFIRMED ranked list (Code_Briefing.md §9):
  [x] 1 DSL rules engine (done — Code_Briefing §10)
  [ ] 2 core credit-check runtime flow (v2 single + multi-product; the aspect-driven state machine)
  [ ] 3 domain & data model + no-transaction atomicity + the single-index creditCheckResult risk
  [ ] 4 external integration backends (CSI SOAP 9 ops · CAS UCCS/CSRM · Equifax UBCT/ICAAM)
  [ ] 5 security model (3 token regimes + the dead/buggy components)
  [ ] 6 admin analytics + audit (likely catalogue-depth)
  Add the matching skill (instructions_plan P2 table): microservices-async-messaging (flow/eventing) ·
  azure-data-tier-design + data-access-engineering (data) · test-engineering (the 209 Spock tests as a
  behavior oracle) · azure-microservices-security (security).

DEEP-READ — go to file:line, not component altitude. Split the chosen area across parallel subagents
  (the D1 split was: the two evaluators · the lifecycle/cache · the invocation flows · the test oracle).
  Each returns deep facts WITH anchors and must CHALLENGE the P1 breadth claims for that area — a
  deepen pass verifies-and-corrects inferred breadth claims, it does not just elaborate them
  (D1 corrected the "cache-fronted" claim and surfaced the gt = >= vs > divergence).

WRITE — EXTEND, do not overwrite (preserve P0 §0–§1, the SHA note, and all P1 content):
  Code_Briefing.md → add a new "## <n> · P2 deep-read — D<area>" section (like §10 for D1). If the
    deep-read contradicts a P1 claim, mark it ⚠ and correct the breadth section too.
  HLD.md → upgrade the matching section(s) inferred → evidence-backed: the runtime flow (§7),
    the data/collection model (§6), the §8 integrations, and complete the relevant §10 decision
    record (observed / evidence / likely rationale [inferred] / trade-off); sharpen §11. Bump the version.
  Inferred_Product_Spec.md → only if the deep-read changes a capability/actor/value-flow claim.

DISCIPLINE — every non-trivial fact: evidence anchor (file › Type#member › L<s>–<e>), tagged
  deterministic | inferred (+confidence). No fabrications; one invented fact fails the area.

FINISH — update tracking (tick the area in tracking/milestones/P2-Deepen.md + Status.md). Do NOT
  self-certify — run the P2 Gate Prompt (below) as a separate adversarial pass.


P2 Gate Prompt (per area) -

Dispatch FRESH adversarial subagent(s) (independent — did NOT author the deepened section) to verify
ONE P2 area against the e17fe410 code. Be adversarial. The gate has TWO jobs: is the new depth
CORRECT, and did it actually ADVANCE P1?

INPUTS: the new Code_Briefing.md "P2 deep-read — D<area>" section + the upgraded HLD sections; AND
  the area's P1 carry-forward list from the P2 Prompt (what P1 deferred / left [inferred] / [not
  deep-read] for this area, and the §11 risk it owns).
GROUND TRUTH = source at coderepos/clear/apm0045942-credit-routing-service.

A. VERIFY AGAINST CODE (not plausibility): open every load-bearing claim in the deepened area (the
  specific operators, line numbers, flows, annotation facts), confirm each anchor resolves and says
  what's claimed, and re-grep any counts. Hunt for a FABRICATED component / flow / fact — one fails
  the area.

B. PROGRESS — did P2 close what P1 left open? For each carry-forward item (deferred / [not deep-read]
  / [inferred]) confirm it is now EITHER evidence-backed (anchor added, inferred tag removed) OR still
  honestly marked inferred-with-confidence — never silently asserted. Confirm the §11 risk this area
  owns is now CONCRETE (named files/sequences), not just restated.

C. CORRECTION LINEAGE — for any claim P2 corrects from P1 (e.g. the cache reality, the gt divergence),
  verify (1) the correction is itself right against the code, AND (2) the P1 breadth section was
  actually updated to match — there must be NO lingering contradiction between a breadth claim and the
  deep finding anywhere in the docs.

D. NO REGRESSION — confirm EXTEND-not-overwrite held: P0 §0–§1, the SHA provenance note, and the rest
  of the P1 content are intact and unchanged; the HLD version was bumped.

CHECK the per-area gate (milestone-aware, P2 depth): altitude held (component-to-line, not a code
  dump) · accuracy (zero fabrications, counts right) · evidence (every non-trivial claim anchored;
  every inference carries a confidence band). The full 6-dimension rubric re-score is at P3, not here.

OUTPUT: per-claim verdicts (CONFIRMED / IMPRECISE / FALSE) · a fabrications line · a "P1 open items
  closed?" line (each carry-forward: closed / still-open / regressed) · the per-area PASS/FAIL ·
  specific fixes. Assistive only — the formal sign-off is the P3 reviewer pass.
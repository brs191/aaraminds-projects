

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
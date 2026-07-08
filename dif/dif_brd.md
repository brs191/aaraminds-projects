# DIF — Documents Intelligence Factory
## Business Requirements Document (BRD)

**Version:** 0.2 (Draft)
**Date:** 2026-07-08
**Owner:** AaraMinds
**Status:** Draft — pending review
**Related:** DIF_PRD.md, RIF (Repo Intelligence Factory)
**v0.2 changes:** market-trend review applied — citations repositioned as table stakes, differentiation table rewritten, named competitors added to risks, EU AI Act added to "why now", ACL roadmap commitment added (BR4).
**v0.3 changes (D-007):** RIF federation promoted to core v1 — positioning updated to code-aware document intelligence, deployment/pricing unit is per project attached to RIF, cross-sell motion inverted (DIF attaches to every RIF deployment), joint-graph demo moved from v2 to P2.

---

## 1. Executive summary

AaraMinds has a proven intelligence-factory pattern: deterministic extraction into a content-addressed knowledge graph, hybrid retrieval, and citation-gated agents. RIF applies it to code. DIF applies it to documents — and, decisively, **joins the two (D-007)**: DIF deploys per project into the project's existing RIF database, linking document claims to code entities. The combined product answers what neither search nor RAG nor any competitor can: *which documents describe this code, and are they still true?*

The business case rests on three points. First, **every enterprise has the problem**: knowledge trapped in Word/PDF/PowerPoint estates, invisible cross-references, and AI initiatives blocked because answers can't be traced or audited. Second, **the differentiator is structural, not incremental** — and it is *not* citations, which are 2026 table stakes (Microsoft ships passage-level "deep citations"; Harvey and Hebbia do character/sentence-level). The differentiators are the three things no one ships: a productized, incrementally-maintained document knowledge graph; cross-document lineage/impact analysis; and citation *gating* — grounding enforced and scored as a contract, not offered as a feature. Competitors must re-architect to match. Third, **the marginal cost is low**: DIF reuses RIF's stack, ops model, embedding service, and hardening lessons, so most engineering spend goes into the new extraction domain rather than re-solving solved problems.

Recommendation: fund DIF as the second product in an "Intelligence Factory" family, targeting a design-partner pilot at the end of Phase 3 (per PRD phasing) and a joint RIF+DIF story ("your code and your documents in one queryable graph") as the medium-term wedge no point-solution competitor can tell.

## 2. Business objectives

| # | Objective | Measure |
|---|-----------|---------|
| B1 | Establish DIF as a sellable AaraMinds product (not a client one-off) | Clean IP: `com.aaraminds` namespace from first commit; no client branding; licensing decided pre-pilot |
| B2 | Land 1–2 design partners on the P3 pilot | Signed pilot agreements with defined corpora and golden-query success criteria |
| B3 | Prove the factory-family thesis | ≥60% infrastructure/service reuse from RIF, measured as shared modules vs new code [VERIFY at P2] |
| B4 | DIF attaches to every RIF deployment by default (D-007 — the motion is attach, not cross-sell) | 100% of RIF projects offered DIF at deployment; joint-graph (`docs_for_code`) demo at P1, drift demo at P2 |
| B5 | Keep engineering economics honest | Pilot delivered within the P0–P3 phasing without a phase-6-style "hardening deferred" debt (RIF's known failure mode) |

## 3. Problem and opportunity

**The problem, in business terms.** Enterprises are spending on AI assistants that answer questions about their own documents — and discovering the answers can't be trusted or traced. In contracts, compliance, and engineering governance, an answer without a verifiable source is not a product feature; it's a liability. Meanwhile document estates keep growing, cross-references rot, and "what's impacted if we change this?" remains a week of manual work.

**Why now.** Three converging forces. (1) MCP won the agent-integration standards war — OpenAI, Google, and Microsoft are all first-party adopters, and the 2026-07-28 spec finalizes stateless, OAuth-secured remote servers; a documents-intelligence backend exposed as MCP tools plugs into whatever agent surface the customer already uses (Claude, Copilot, Gemini, internal agents) rather than competing with it. (2) **EU AI Act transparency obligations apply from December 2, 2026** — regulated buyers are already writing right-to-audit clauses; "this tool cannot return an ungrounded claim" is a compliance story incumbents haven't built. (3) Permission-aware, cited retrieval became a named procurement gate in 2026 while the underlying graph/lineage capability remains unshipped by anyone. AaraMinds already owns a validated architecture for exactly this pattern.

**Why us.** RIF is working proof: deterministic extraction, provenance gates, atomic index swaps, citation-gated narration, cross-language e2e testing. The intelligence-factory pattern is the company's core IP; DIF is its second instantiation. No market-size figures are quoted here — [VERIFY: commission market sizing before external fundraising use; this document intentionally avoids unsourced TAM claims.]

## 4. Product positioning

**Category:** Code-aware document intelligence / grounded-retrieval backend for enterprise AI agents.

**One-liner:** *DIF turns your document estate into a citation-grounded knowledge graph that knows your code — so your agents can tell you not just what the docs say, but whether they're still true.*

**Positioning stance:** DIF is **infrastructure, not an end-user search app** — the neutral, auditable context layer *underneath* the agents an enterprise already runs (Claude Enterprise via managed MCP auth, Copilot Studio, Gemini Enterprise, internal agents). Compete on openness and auditability; never compete on end-user UI, where Glean and Microsoft have unassailable distribution.

**Differentiation vs alternatives (citations alone differentiate against nobody — they are the entry ticket):**

| Alternative | What they have | Where DIF wins |
|-------------|----------------|----------------|
| Glean-class work-AI platforms ($7.2B, ~$50–75/user/mo, own MCP directory + agents-as-tools) | Enterprise knowledge graph with ACLs and citations — locked inside their app | DIF's graph is open, exportable, auditable infrastructure serving *any* agent; document lineage/impact analysis, which Glean does not ship; no per-seat lock-in |
| Microsoft (Copilot deep citations, SharePoint Knowledge Agent bundled with licenses) | Passage-level citations; auto-metadata on SharePoint libraries | Cross-document dependency graph and impact analysis, not metadata columns; works across sources beyond M365; grounding enforced, not just displayed |
| Foundation-vendor native features (OpenAI Company Knowledge, Claude connectors) | Free connect-and-cite inside the chat subscription, per-user permissions | Those cite; they don't gate, score groundedness, version corpora, or trace impact — DIF is the governed layer their agents call via MCP |
| Naive RAG stacks (chunk + embed + LLM) | Cheap, hosted (OpenAI File Search at API pennies) | Structure-blind, no versioning or impact analysis; DIF's deterministic graph is the moat |
| GraphRAG offerings (Microsoft library, Neo4j Infinigraph) | Graph capability as DIY library or database | Both hand you an ontology-design and graph-maintenance problem; DIF's content-addressed structural graph maintains itself incrementally under document churn — the known unsolved operational pain of LLM-extracted graphs |
| Document-AI APIs (Unstructured, Reducto, LlamaParse) | Commoditized high-accuracy parsing (DIF *buys* this layer, per PRD R2a) | They extract; they don't build a corpus-wide graph, retrieval layer, or agent contract |
| DIY internal builds | Full control | DIF ships the hard parts (determinism, provenance gates, incremental indexing, MCP contract, grounding scoring) that internal teams underestimate |

**The family story (now core, not v2 — D-007):** RIF answers "what does our code do and what breaks if we change it?" DIF answers the same for documents — *in the same database*. `DESCRIBES` edges link doc blocks to code entities, so the federated graph ships in v1: `docs_for_code` and `code_for_doc` at P1, `drift_report` (documentation drift detection as a product) at P2. This is the moat: Glean, Microsoft, and every RAG vendor would need to own a per-project code graph to copy it. DIF also runs standalone on doc-only corpora, but the attach-to-RIF deployment is the default and the demo.

## 5. Target market and customers

**Initial ICP (pilot):**
- **Every project with a RIF deployment (D-007)** — DIF's primary channel is attachment to the existing RIF footprint: same Postgres, same BYOC stack, immediate `docs_for_code` value on day one.
- Mid-to-large engineering organizations with document governance pain: telecom, financial services, healthcare-adjacent — sectors where citations are compliance-relevant, not cosmetic.
- Teams already deploying MCP-based agents that need governed, grounded document context.

**Buyer:** Head of Platform Engineering / CTO office (same buyer as RIF).
**Champion:** AI platform lead or compliance-tooling owner.
**Users:** engineers, architects, contracts/compliance analysts, and downstream AI agents.

## 6. Business model

Decisions needed before pilot close (owner: AaraMinds leadership):

- **Licensing:** **DECIDED (D-001, 2026-07-08):** managed deployment on customer Azure tenancy (BYOC), AaraMinds-operated, priced per corpus + usage. One commercial pattern for the factory family — applies to RIF as well. See `DECISIONS.md`.
- **Pilot commercials:** paid pilots with success criteria (golden-query precision, citation integrity) that convert to annual subscription. Unpaid pilots are innovation theater; do not run them.
- **Pricing inputs:** corpus size (documents/versions indexed), connector count, agent-call volume. **Pricing unit is per project (D-007)** — the same unit as the RIF deployment it attaches to; a bundled RIF+DIF per-project price is the default offer, standalone DIF the exception. Establish metering in the product by P1 per PRD R30 — retrofitting usage metering is expensive.
- **Services attach:** corpus onboarding, golden-set curation, and connector configuration as paid services — this is real margin for a two-sided (product + expertise) company.

## 7. Business requirements

| # | Requirement | Rationale |
|---|-------------|-----------|
| BR1 | Clean, owned IP from day one: `com.aaraminds.dif`, own repo, no client namespaces or client-environment policy in governance files | RIF's costliest review finding; a sellable product cannot carry another company's branding |
| BR2 | Multi-tenancy posture: resolved by D-001 (BYOC) — isolation by customer tenancy; revisit only if an AaraMinds-hosted tier is ever added | Sales blocker if undefined; BYOC gives the strongest isolation answer by construction |
| BR3 | Security is a sales feature: auth on every surface, non-root containers, vuln-scanned dependencies, SOC 2-aligned controls documented from the skills-pack | Enterprise procurement gate; RIF review showed the debt cost when deferred |
| BR4 | Source-ACL limitation stated honestly in all sales material (v1 = uniformly-readable corpora or separately indexed corpora per access boundary), **with a committed, dated v2 ACL-propagation roadmap item** — permission-aware retrieval is a named 2026 procurement gate and compliance buyers ask in the first meeting | Overclaiming loses compliance-sensitive deals permanently; having no roadmap answer loses them in meeting one |
| BR5 | Citation integrity is contractual: 100% of claim blocks resolve to source anchors and pass grounding checks, structurally enforced and auditable via the audit log | This is the product's core promise; it must be demonstrable in a procurement bake-off |
| BR6 | Every pilot has a golden-query set and measured baseline before success targets are agreed | No fabricated metrics — internal policy and customer credibility |
| BR7 | Demo corpus + demo script maintained from P0 (public documents), so sales demos never require customer data | Shortens sales cycle; avoids NDA friction at top of funnel |
| BR8 | RIF and DIF share embedding service, deployment tooling, and MCP conventions; divergence requires an ADR | Protects the reuse economics (B3) |
| BR9 | Paid-pilot qualification includes an admissible corpus check before kickoff: corpus is uniformly readable, or customer accepts separate indexes per access boundary | Prevents ACL mismatch from surfacing as a late procurement blocker |
| BR10 | Usage metering is live before paid pilot: ingestion, indexed documents, embedding batches, MCP calls, agent requests, connector syncs | Supports pricing, cost control, and usage-based renewal discussions |

## 8. Financial view

Costs are dominated by engineering time across P0–P3 (PRD §8), plus Azure infrastructure for pilot deployments (Postgres Flexible Server + pgvector, Container Apps, Key Vault, monitoring — same footprint class as RIF) and embedding/LLM API spend metered per corpus.

No revenue or cost figures are stated in this draft — [VERIFY: build the pilot P&L with actual team allocation and Azure estimates before leadership review]. The structural claim that survives without numbers: DIF's marginal cost is materially below RIF's original cost because the retriever, embedding service, MCP patterns, ops model, and hardening lessons are inherited, and spend concentrates on document extractors and connectors.

## 9. Go-to-market (pilot horizon)

1. **P0–P2:** dogfood on AaraMinds' own corpus (instruction-os, governance docs, skills-pack) — the demo is the company's own brain, which is also the brand story.
2. **P2:** publish the differentiation narrative (citation-gated document agents; impact analysis for documents) via AaraMinds content channels — Content Strategist persona owns this.
3. **P3:** 1–2 paid design partners from the RIF pipeline / existing network; success criteria contractually defined per BR6.
4. **Post-pilot:** case study with measured (not fabricated) results → repeatable pilot offer → v2 joint RIF+DIF wedge.

## 10. Risks and dependencies

| Risk | Type | Mitigation |
|------|------|------------|
| Extraction quality on messy real-world PDFs undermines the accuracy promise | Product | Parsing router (PRD R2a) with Docling/VLM fallback; degenerate-run gates; per-corpus quality report before agent access is enabled |
| **Glean expands downward into infrastructure** — MCP directory, agents-as-tools, enterprise graph, from a $7.2B position | Market | Position as the neutral, open, auditable layer vs their closed per-seat platform; move fast on lineage/gating where they have nothing; avoid their UI battleground entirely |
| **Microsoft bundles "good enough" for free** — SharePoint Knowledge Agent + deep citations included in Copilot licenses buyers already own | Market | Sell where M365 stops: cross-source graphs, impact analysis, grounding enforcement, non-M365 corpora; ICP screens for multi-source estates |
| **Foundation vendors give away connect-and-cite** — OpenAI Company Knowledge, Claude enterprise connectors, inside the chat subscription | Market | They're a channel, not just a threat: DIF serves those same agents via MCP as the governed context layer; differentiation is gating + graph, never basic cited search |
| Single-team capacity: DIF competes with RIF hardening for the same engineers | Execution | RIF review remediation (its priority list) is scheduled work, not background noise; sequence explicitly — do not run both at full tilt |
| ACL/permission expectations from compliance buyers exceed v1 | Sales | BR4 honesty rule + committed dated ACL roadmap; qualify pilots on admissible corpora (BR9) |
| Pilot partners treat it as free consulting | Commercial | Paid pilots only (§6) |
| Hygiene debt recurrence (RIF pattern) | Execution | PRD R25–R29 are P0, CI-enforced; BRD sign-off requires the P0 exit criteria met |

**Dependencies:** RIF embedding service fixes (PRD R23) land before DIF P1; multi-tenancy ADR (BR2) before P3; licensing decision (§6) before pilot contracts.

## 11. Success criteria (business)

- Two paid design-partner pilots signed by end of P3, each with golden-set success criteria met.
- 100% claim-level citation-integrity rate demonstrated in at least one procurement evaluation.
- ≥60% RIF infrastructure reuse validated at P2 [VERIFY with module-level accounting].
- One published case study with customer-approved, measured results.
- Zero client-branding or IP-provenance findings in an external review of the DIF repo.

## 12. Approvals

| Role | Name | Status |
|------|------|--------|
| Product owner | Raja | Approved 2026-07-08 (D-004) |
| Engineering lead | Raja | Approved 2026-07-08 (D-004) |
| Commercial/licensing | Raja | Approved 2026-07-08 (D-004) |

*Single-approver model recorded in D-004; revisit before first paid pilot contract.*

---
*Conventions: this document quotes no unsourced market or financial figures; items marked [VERIFY] or [DECISION REQUIRED] must be resolved before this BRD leaves Draft status.*

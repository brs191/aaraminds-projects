# DIF — Documents Intelligence Factory
## Business Requirements Document (BRD)

**Version:** 0.1 (Draft)
**Date:** 2026-07-07
**Owner:** AaraMinds
**Status:** Draft — pending review
**Related:** DIF_PRD.md, RIF (Repo Intelligence Factory)

---

## 1. Executive summary

AaraMinds has a proven intelligence-factory pattern: deterministic extraction into a content-addressed knowledge graph, hybrid retrieval, and citation-gated agents. RIF applies it to code. DIF applies it to documents — the larger, more universal corpus every enterprise owns.

The business case rests on three points. First, **every enterprise has the problem**: knowledge trapped in Word/PDF/PowerPoint estates, invisible cross-references, and AI initiatives blocked because naive RAG cannot cite or trace impact. Second, **the differentiator is structural, not incremental**: citation integrity and change-impact analysis are things chunk-and-embed RAG cannot do by construction — competitors must re-architect to match. Third, **the marginal cost is low**: DIF reuses RIF's stack, ops model, embedding service, and hardening lessons, so most engineering spend goes into the new extraction domain rather than re-solving solved problems.

Recommendation: fund DIF as the second product in an "Intelligence Factory" family, targeting a design-partner pilot at the end of Phase 3 (per PRD phasing) and a joint RIF+DIF story ("your code and your documents in one queryable graph") as the medium-term wedge no point-solution competitor can tell.

## 2. Business objectives

| # | Objective | Measure |
|---|-----------|---------|
| B1 | Establish DIF as a sellable AaraMinds product (not a client one-off) | Clean IP: `com.aaraminds` namespace from first commit; no client branding; licensing decided pre-pilot |
| B2 | Land 1–2 design partners on the P3 pilot | Signed pilot agreements with defined corpora and golden-query success criteria |
| B3 | Prove the factory-family thesis | ≥60% infrastructure/service reuse from RIF, measured as shared modules vs new code [VERIFY at P2] |
| B4 | Create the RIF+DIF cross-sell motion | Every RIF prospect pitched DIF and vice versa; joint-graph demo by v2 |
| B5 | Keep engineering economics honest | Pilot delivered within the P0–P3 phasing without a phase-6-style "hardening deferred" debt (RIF's known failure mode) |

## 3. Problem and opportunity

**The problem, in business terms.** Enterprises are spending on AI assistants that answer questions about their own documents — and discovering the answers can't be trusted or traced. In contracts, compliance, and engineering governance, an answer without a verifiable source is not a product feature; it's a liability. Meanwhile document estates keep growing, cross-references rot, and "what's impacted if we change this?" remains a week of manual work.

**Why now.** MCP has become the integration standard for enterprise AI agents; a documents-intelligence backend exposed as MCP tools plugs into whatever agent surface the customer already uses (Claude, Copilot, internal agents) rather than competing with it. AaraMinds already owns a validated architecture for exactly this pattern.

**Why us.** RIF is working proof: deterministic extraction, provenance gates, atomic index swaps, citation-gated narration, cross-language e2e testing. The intelligence-factory pattern is the company's core IP; DIF is its second instantiation. No market-size figures are quoted here — [VERIFY: commission market sizing before external fundraising use; this document intentionally avoids unsourced TAM claims.]

## 4. Product positioning

**Category:** Document intelligence platform / grounded-retrieval backend for enterprise AI agents.

**One-liner:** *DIF turns your document estate into a queryable, citation-grounded knowledge graph your AI agents can actually trust.*

**Differentiation vs alternatives:**

| Alternative | Where DIF wins |
|-------------|----------------|
| Enterprise search (Elastic, SharePoint search, Glean-class) | Search finds documents; DIF answers questions with clause-level citations and traces reference/impact chains |
| Naive RAG stacks (chunk + embed + LLM) | Structure-blind, citation-weak, no versioning or impact analysis; DIF's graph is the moat |
| Document-AI APIs (parsing/OCR services) | Those extract; they don't build a corpus-wide graph, retrieval layer, or agent contract |
| DIY internal builds | DIF ships the hard parts (determinism, provenance gates, incremental indexing, MCP contract) that internal teams underestimate |

**The family story:** RIF answers "what does our code do and what breaks if we change it?" DIF answers the same for documents. v2's federated graph answers the question nobody else can: *"which documents describe this code, and are they still true?"* — documentation drift detection as a product. That story is unique to owning both factories.

## 5. Target market and customers

**Initial ICP (pilot):**
- Mid-to-large engineering organizations with document governance pain: telecom, financial services, healthcare-adjacent — sectors where citations are compliance-relevant, not cosmetic.
- Existing RIF prospects/users (warm cross-sell; shared deployment footprint).
- Teams already deploying MCP-based agents that need governed, grounded document context.

**Buyer:** Head of Platform Engineering / CTO office (same buyer as RIF).
**Champion:** AI platform lead or compliance-tooling owner.
**Users:** engineers, architects, contracts/compliance analysts, and downstream AI agents.

## 6. Business model

Decisions needed before pilot close (owner: AaraMinds leadership):

- **Licensing:** self-hosted subscription (per-corpus or per-seat) vs managed deployment on customer Azure tenancy. RIF's deployment model should set the default — one commercial pattern for the family. [DECISION REQUIRED]
- **Pilot commercials:** paid pilots with success criteria (golden-query precision, citation integrity) that convert to annual subscription. Unpaid pilots are innovation theater; do not run them.
- **Pricing inputs:** corpus size (documents/versions indexed), connector count, agent-call volume. Establish metering in the product by P1 per PRD R30 — retrofitting usage metering is expensive.
- **Services attach:** corpus onboarding, golden-set curation, and connector configuration as paid services — this is real margin for a two-sided (product + expertise) company.

## 7. Business requirements

| # | Requirement | Rationale |
|---|-------------|-----------|
| BR1 | Clean, owned IP from day one: `com.aaraminds.dif`, own repo, no client namespaces or client-environment policy in governance files | RIF's costliest review finding; a sellable product cannot carry another company's branding |
| BR2 | Multi-tenancy posture decided by ADR before P3 (DB-per-tenant default for enterprise isolation story) | Sales blocker if undefined; retrofit is a rewrite |
| BR3 | Security is a sales feature: auth on every surface, non-root containers, vuln-scanned dependencies, SOC 2-aligned controls documented from the skills-pack | Enterprise procurement gate; RIF review showed the debt cost when deferred |
| BR4 | Source-ACL limitation stated honestly in all sales material (v1 = uniformly-readable corpora or separately indexed corpora per access boundary; ACL propagation is v2) | Overclaiming here loses compliance-sensitive deals permanently |
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
| Extraction quality on messy real-world PDFs undermines the accuracy promise | Product | Text-layer-only v1 scope; degenerate-run gates; per-corpus quality report before agent access is enabled |
| Incumbents (enterprise search vendors) bolt on citations | Market | Speed + graph/impact-analysis depth; federated RIF+DIF story they can't copy without owning a code graph |
| Single-team capacity: DIF competes with RIF hardening for the same engineers | Execution | RIF review remediation (its priority list) is scheduled work, not background noise; sequence explicitly — do not run both at full tilt |
| ACL/permission expectations from compliance buyers exceed v1 | Sales | BR4 honesty rule; qualify pilots on uniformly-readable corpora |
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
| Product owner | [PENDING] | — |
| Engineering lead | [PENDING] | — |
| Commercial/licensing | [PENDING] | — |

---
*Conventions: this document quotes no unsourced market or financial figures; items marked [VERIFY] or [DECISION REQUIRED] must be resolved before this BRD leaves Draft status.*

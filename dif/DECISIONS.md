# DIF — Decision Log

Format: one decision per entry; status Accepted unless noted. Reversals require a new entry, not an edit.

---

## D-001: Deployment/licensing model — managed on customer Azure (BYOC)

**Date:** 2026-07-08 · **Status:** Accepted · **Owner:** Raja

DIF deploys into the customer's Azure tenancy; AaraMinds operates it. Priced per corpus + usage (metering per PRD R30 / BRD BR10).

**Why:** data never leaves the customer tenant — the strongest compliance story for the telecom/finserv ICP; aligns with the Azure-primary stack; sets one commercial pattern for the factory family (applies to RIF too).
**Consequences:** Terraform AzureRM deployment automation is pilot-critical (P3); multi-tenancy isolation is achieved by tenancy separation, not in-app row-level controls — simplifies BR2; AaraMinds needs an operational support model per deployment.

## D-002: Prose embedding default — Voyage

**Date:** 2026-07-08 · **Status:** Accepted · **Owner:** Raja

Voyage (voyage-3-large class) via the shared LiteLLM embedding service, stored full-dimension, served truncated ≤1024d (Matryoshka). Qwen3-Embedding remains the self-host/sovereignty fallback.

**Why:** retrieval-focused leader, Anthropic's recommended provider, clean path to voyage-multimodal page-image embeddings in v2 (ColPali-style second index). Satisfies PRD R23a before P1.
**Consequences:** per-token API cost metered per corpus; provider abstraction keeps switch cost low; pick exact model + dimension during P0 spike and pin it before first pilot corpus embedding.

## D-003: Graph storage — relational adjacency, not Apache AGE

**Date:** 2026-07-08 · **Status:** Accepted · **Owner:** Raja

`docs_edges` as a plain relational table; traversals via recursive CTEs (bounded, `max_depth ≤ 5` per PRD R13a).

**Why:** DIF's traversals are shallow and bounded — CTEs handle them; zero extension risk on Azure Postgres Flexible Server; proven shape in RIF's retriever. AGE's openCypher elegance doesn't pay for its operational/upgrade risk.
**Consequences:** resolves PRD open question #1; deep/unbounded graph analytics (if ever needed) would be a new decision.

---

## D-004: BRD approval owners — Raja (all three roles)

**Date:** 2026-07-08 · **Status:** Accepted · **Owner:** Raja

Raja holds Product, Engineering, and Commercial approval for DIF v0.x. Recorded explicitly: this is a bus-factor-1 governance model, acceptable pre-pilot. Revisit when a second approver exists or before the first paid pilot contract is signed.

## D-006: ACL propagation — committed as first v2 priority, ASAP posture

**Date:** 2026-07-08 · **Status:** Accepted · **Owner:** Raja

ACL propagation is the **first v2 item**, not one candidate among several. Design work (row-level filtering vs per-corpus partitioning vs tenant-specific indexes) starts during P3 in parallel with connector work, since the SharePoint connector is where source ACLs first appear. The dated commitment for sales material (BR4) is pinned at P2 exit, when phase velocity is known — a date invented today would violate the no-fabricated-metrics rule.

## D-007: RIF+DIF federation is core v1 architecture, not a v2 candidate

**Date:** 2026-07-08 · **Status:** Accepted · **Owner:** Raja

Context: DIF is needed for each project Raja works on, and each project already has a RIF code graph in a Postgres database. DIF must co-work with RIF for the outcomes it generates.

Decided: (1) DIF deploys **per project** into the project's existing RIF Postgres — `dif_meta` schema beside `rif_meta`; cross-graph queries are SQL joins, not a federation protocol. (2) New `DESCRIBES` edge class (doc block → code node) with a code-entity detector resolving against `rif_meta` (P1). (3) Cross-graph MCP tools: `docs_for_code`/`code_for_doc` (P1), `drift_report` (P2). (4) Shared NodeIdComputer is a hard requirement. (5) Standalone doc-only mode supported; cross-graph tools return `rif_not_deployed` explicitly. (6) Pricing/deployment unit is per project; DIF attaches to every RIF deployment by default.

**Why:** documentation drift ("which docs describe this code, and are they still true?") was already the identified market white space; per-project RIF instances make it deliverable in v1 at low marginal cost. No competitor can follow without owning a code graph.
**Consequences:** P0 must land in the RIF Postgres with compatible IDs (near-zero cost); a `rif_meta` minimum-version compatibility contract is needed (PRD open question 5); reference-density spike now also measures doc→code resolution rates. PRD v0.3 / BRD v0.3 carry the full changes.

## D-008: v1 format scope — JSON is a first-class v1 artifact; Excel at v1.5

**Date:** 2026-07-08 · **Status:** Accepted · **Owner:** Raja

Per `design-decisions.md` DD-01: v1 ingests documents **plus file-based structured artifacts** — `.md`, `.txt`, `.docx`, `.json` (P0), `.pdf`, `.pptx` (P1), `.xlsx` (v1.5, visible sheets/ranges/formulas with caveats). General enterprise-data-platform scope (streaming, CDC, warehouses) remains explicitly out.

**Why:** JSON configs, policies-as-code, and inventories are dense in engineering corpora and pair directly with the RIF federation story (D-007) — a config file that DESCRIBES code entities is a first-class drift source. Deterministic parsing with JSONPath anchors fits the existing extraction contract with no new machinery.
**Consequences:** PRD R2 updated; JSONPath added to the source-anchor contract; JSON graph-expansion caps need an ADR before P0 JSON ingestion (design-decisions ADR-006); every new format enters via the DD-01 format admission policy (parser, anchor, nodes, caveats, golden tests, cost profile).

---

## Document roles (recorded 2026-07-08)

`design-decisions.md` is the **decision backlog** — the pre-build question bank (DD-01…DD-28) with options and recommended defaults. This file is the **decision log** — what was actually decided. Workflow: DD item decided → D-entry here → DD marked resolved there. Conflicts resolve in favor of the newest dated D-entry.

---

## Open (not yet decided)

- **D-005 (pending):** Exact Voyage model + serving dimension — decide at end of P0 embedding spike (confirmed 2026-07-08: decide at the end, as planned).

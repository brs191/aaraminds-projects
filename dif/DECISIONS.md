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

---

## Open (not yet decided)

- **D-005 (pending):** Exact Voyage model + serving dimension — decide at end of P0 embedding spike (confirmed 2026-07-08: decide at the end, as planned).

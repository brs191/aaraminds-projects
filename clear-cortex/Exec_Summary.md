# CLEAR (Credit Routing Service) — Architecture Comprehension
### Executive summary · 2026-06-02

**Bottom line.** CLEAR is a well-structured, rules-driven credit-routing service whose *functional* design is sound and genuinely flexible. But it carries **four systemic gaps — reliability, security, resilience, and data-tier hygiene — that its feature set does not reveal.** None requires a rewrite; all are evolutions of the current Azure/Spring design. The two most urgent are the absence of transactional integrity and the absence of server-side authorization.

**What it does.** CLEAR is AT&T's decision-and-routing engine for business credit checks: it takes a request, applies finance-owned configurable rules to choose the right credit bureau and policy (Equifax via the CSI/UBCT paths; the CAS gateway), records the decision, and publishes it to the order lifecycle. Finance can change routing rules without a redeploy — a real strength worth protecting.

**What's solid.** The routing core (rules → strategy dispatch), the integration breadth (SOAP + REST + Kafka across three bureaus), authentication (multi-issuer OAuth2 with real token verification), and reasonable test coverage on the core flows.

**What's at risk — four decisions to make.**

1. **No transactional integrity — highest severity.** One request makes up to ~10 unguarded database writes plus a non-transactional event, with no rollback. A mid-sequence failure can leave a recorded-but-incomplete decision, a multi-product record that is "one approved / one failed," or — worst — an **event announcing a decision the database never saved**. *Decision: fund a transaction boundary plus a transactional outbox.*

2. **Authorization is not enforced on the server.** Any valid login reaches *every* administrative and rule-editing endpoint — including the ones that grant roles. The role model exists, but it is applied only in the browser UI, never checked server-side. *Decision: fund a back-end authorization layer — the role data already exists; only the enforcement gate is missing.*

3. **No resilience to a slow bureau.** Downstream calls are synchronous, with no circuit breaker, no connection-pool ceiling, and a 60-second timeout. A *slow* (not even down) Equifax or CAS can exhaust threads and cascade into a broader outage. *Decision: adopt Resilience4j and a pooled HTTP client.*

4. **Data-tier and secrets hygiene.** The busiest collections are effectively unindexed and never purged — a latent availability risk on a usage-priced datastore — and six production-style credentials sit in plaintext in configuration. *Decision: ship an index + retention migration; move secrets to Azure Key Vault.*

Two divergent copies of the rules engine and several dead or misconfigured security components round out the list; both are detailed in the HLD (§10–§11).

**How we know.** Every finding is anchored to a specific file and line at commit `e17fe410`, produced by a structured comprehension — breadth map, then six focused deep-reads, then an adversarial verification gate that **found zero fabricated facts**. This is an assistive comprehension, not a certified audit: a second human reviewer and a one-line commit reconciliation remain open. No production telemetry was available, so no business-impact figures are asserted here — the severities are engineering judgements, not measured incidents.

**The ask.** Prioritise (1) and (2) in the next planning cycle; (3) and (4) as fast-follows. Every item is an evolution of the existing service, not a rebuild.

*Full detail: `evaluation/HLD.md` (v1.0) · evidence in `evaluation/Code_Briefing.md` · diagrams in `design/`.*

# Inferred Product Spec — Credit Routing Service

**Status:** STUB — fill in P1
**Fixture:** `apm0045942-credit-routing-service` @ `e17fe410`
**Nature:** *inferred* — what the service does as a product, read off the code. Every claim here is a hypothesis; phrase as inference and carry a confidence band. Keep strictly separate from `Code_Briefing.md` (deterministic facts).

## 1 · What the product does
_TODO P1 — one paragraph. Working hypothesis: routes and orchestrates credit-check requests for AT&T across multiple products, applying configurable, finance-team-managed rules and policies._

## 2 · Capabilities
_TODO P1 — credit routing; single- & multi-product credit check; DSL-driven rule evaluation; policy enforcement; identity verification; admin/config/rules management; analytics & monitoring._

## 3 · Actors & callers
_TODO P1 — who calls v1 vs v2; internal vs external callers; the credit-finance admin users; external credit services (via CSI/SOAP)._

## 4 · Value flow
_TODO P1 — request in → routing decision → rules + policy evaluation → credit-check result → event published. Where the business value is created._

## 5 · Confidence & open questions
_TODO P1 — what is well-supported vs. weak inference; acronyms still undecoded; what needs a SME to confirm._

# HLD Template (adapted) — Credit Routing Service

**Adapted from:** `aaraminds-delivery/product-research/Code Intelligence Factory/evaluation/HLD_Template.md` v0.4 — read it for the full contract. This file captures the section structure + the rules that bite + the repo-specific deltas, so `HLD.md` can be authored without leaving the project.

## Altitude

Architectural altitude — the level a senior engineer explains the system to a peer. In scope: components & responsibilities, the domain/data model at entity (here, **collection**) level, principal runtime flows, integration points, cross-cutting mechanisms. Out of scope: class-by-class description (LLD), and content-free generality.

**Type-naming cap.** Name a concrete type/collection/method **only** when it marks an architectural boundary, is a persistence aggregate root, owns a lifecycle, or anchors a flow. No class catalogues.

## Evidence — required for every non-trivial factual claim

Inline tag `[E#]` per claim or claim-cluster, with a per-section Evidence table:

| Claim | Source locator | Provenance | Confidence |
|---|---|---|---|
| E1 | `routing/.../CreditCheckController.java › class CreditCheckController › L40–58` | deterministic | high |
| E5 | inferred from `@Document` + repo query naming on `*RoutingRule` | inferred | medium |

**Locator grammar:** code — `<repo-relative file> › <Type or Type#member> › L<start>–<end>`; config — `<config file> › <key path>`; **Mongo — `<collection name>` or `@Document class › field`** (no migration files in this repo). **Provenance** ∈ `deterministic | inferred | human_confirmed`. **Confidence** ∈ `high | medium | low`. An `inferred` claim must be phrased as inference ("appears to…") and carry a confidence band.

## Section structure

1. **Document control** — subject system; source repo + pinned commit `e17fe410`; scope (whole-service); the comprehension depth (breadth → deepened areas); version & date.
2. **Purpose & context** — what the service does, the business problem, its callers.
3. **Scope** — what the HLD covers and, explicitly, what it does not (§ "not visible" rule).
4. **Architecture overview** — the shape in a few paragraphs; prose stands alone, diagram supports.
5. **Component view** — each major component: responsibility, key types, dependencies (type-naming cap).
6. **Domain & data model** — key domain entities, lifecycle/state; **the MongoDB collection model, inferred from `@Document`** (not relational tables; relationships by reference/embedding marked `inferred`).
7. **Key runtime flows** — principal end-to-end scenarios, step by step at component altitude (e.g. credit-check v2).
8. **External interfaces & integrations** — REST surface (v1/v2 under `/CreditCheck`); external systems: CSI/SOAP credit services, IEBus/Kafka eventing, OIDC, `ubct`. Interfaces defined outside the service marked accordingly.
9. **Cross-cutting concerns** — the mandatory checklist below.
10. **Key design decisions & patterns** — a decision record per decision (below).
11. **Observations** *(optional)* — risks, coupling, tech debt, modernization notes (seed from `appcat`).

### §9 — mandatory cross-cutting checklist (no silent omission)

A row for **every** concern, each marked *Covered* / *Not visible in scoped code* / *Out of scope*, with evidence where Covered:

`Transactions` · `Validation` · `Security & authorization` · `Configuration` · `Error handling` · `Events & observability` · `Auditing` · `Concurrency` · `Multi-tenancy` · `Data-store portability`

> Repo note: expect `Security & authorization` (OAuth2/OIDC + filter chain), `Error handling` (`GlobalExceptionHandler`), `Events & observability` (IEBus/Kafka + Grafana/Sentry), `Auditing` (`admin/audit`), `Concurrency` (ShedLock on scheduled jobs), and `Configuration` (`@ConfigurationProperties` + `application.yml`) to be *Covered*. Much cross-cutting behavior is **woven via aspects** (`cas`/`ubct`/`iebus`) — name the aspect as the evidence.

### §10 — decision-record structure (four fields each)

**Observed decision** (what the code does) · **Evidence** (locator) · **Likely rationale** (*why* — marked `inferred` unless an ADR/doc confirms it) · **Trade-off** (what it gives up).

## "Not visible" rule

Anything within a section's remit not described must be explicitly marked *Covered* / *Not visible in scoped code* / *Out of scope*. Silence is non-conformant.

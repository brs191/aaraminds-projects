# ADR-003: Source ACL Posture for v1

**Date:** 2026-07-08  
**Status:** Accepted for P0 design gate  
**Owners:** Product + Security  
**Related decisions:** D-006, D-010  
**Related docs:** `DECISIONS.md`, `dif_prd.md`, `dif_brd.md`, `action_plan.md`

---

## 1. Context

DIF will initially be shared with a limited set of engineers. The first rollout does not need per-user source permission propagation because the intended corpora can be curated so every authorized DIF user has the same right to read every indexed source document.

Per D-006, full source ACL propagation remains the first v2 item after production readiness/GA. This ADR defines the v1 posture so engineering, production, and sales do not overbuild or overclaim ACL behavior during P0-P3.

---

## 2. Decision

DIF v1 supports **uniformly readable corpora only**.

A corpus is admissible for v1 only if every document, structured artifact, and source location indexed into that corpus may be read by every authorized DIF user for that corpus.

DIF v1 will not implement:

- per-user source ACL propagation
- row-level filtering by source ACL
- per-document user/group permission checks at retrieval time
- per-access-boundary indexes as the default v1 workaround
- mixed-permission SharePoint/OneDrive libraries

The production-ready/GA backlog may take up ACL propagation after P0-P3, with D-006 as the controlling roadmap decision.

---

## 3. Scope

### 3.1 In scope for v1

- Uniformly readable corpora.
- Corpus-level authorization.
- Admin/operator controls for corpus admission.
- Honest v1 limitation language in docs, demos, and pilot material.
- Audit and usage events for ingestion and MCP access.
- P3 SharePoint/OneDrive connector only for uniformly readable folders/libraries.
- RIF+DIF cross-graph tools operating within the same corpus-level access boundary.

### 3.2 Out of scope for v1

- Source-system ACL inheritance.
- Per-user document filtering.
- Mixed-permission corpora.
- Automatic ACL sync from SharePoint/OneDrive.
- Tenant-specific indexes as a required default for access control.
- Claims that DIF is permission-aware beyond the corpus-level authorization boundary.

---

## 4. Admissible-corpus gate

Before ingestion, each corpus must pass an admissibility check.

Required checks:

1. Corpus owner identified.
2. Intended user group identified.
3. Written confirmation that every indexed source is readable by every intended user.
4. No known restricted/private subfolders in the source set.
5. No mixed-permission SharePoint/OneDrive library unless the selected folder scope is uniformly readable.
6. Demo/pilot materials label the corpus as uniformly readable.

If the corpus fails the gate, DIF must not index it for v1.

---

## 5. Runtime behavior

For v1:

- Retrieval and MCP tools enforce corpus-level access.
- `search_docs`, `trace_references`, `impact_of_change`, `docs_for_code`, `code_for_doc`, and `drift_report` must never cross from one corpus/project boundary into another.
- Results do not need per-document ACL filtering because the corpus admission gate guarantees uniform readability.
- Audit events record principal, tenant/project, corpus, tool, parameters hash, outcome, latency, and source anchors returned.
- Usage events remain separate from audit events.

If a request targets a corpus that has not passed admission, the service must fail closed.

Suggested status:

```text
corpus_not_admitted
```

---

## 6. SharePoint/OneDrive constraint

P3 SharePoint/OneDrive connector support is limited to uniformly readable scopes.

Allowed:

- a library readable by all authorized DIF users
- a folder/subtree readable by all authorized DIF users
- a curated export/drop location with uniform access

Not allowed in v1:

- indexing an entire site/library with mixed item-level permissions
- relying on DIF to filter results by source permissions
- ingesting private/restricted folders into a shared corpus

---

## 7. Sales and pilot language

Required language:

```text
DIF v1 supports uniformly readable corpora. All indexed documents in a corpus must be readable by every authorized DIF user for that corpus. Source ACL propagation is planned after production readiness/GA and remains the first v2 priority.
```

Prohibited language:

- "DIF preserves SharePoint permissions in v1."
- "DIF performs per-user ACL filtering in v1."
- "DIF can safely index mixed-permission corpora in v1."

---

## 8. Evaluation gates

P0-P3 must include tests or checklist gates for:

1. Corpus admission metadata exists.
2. Non-admitted corpus fails closed.
3. MCP calls require corpus authorization.
4. Cross-graph tools remain inside the same corpus/project boundary.
5. Audit event includes corpus/project and principal.
6. No docs or demo material overclaim source ACL propagation.

No per-user ACL negative tests are required for v1 because per-user source ACL propagation is explicitly out of scope. Instead, v1 tests the admission gate and corpus-level authorization.

---

## 9. Consequences

Positive:

- Keeps P0-P3 focused on production readiness and core document/code intelligence.
- Matches the initial limited-engineer sharing model.
- Avoids false security confidence from incomplete ACL propagation.
- Simplifies SharePoint/OneDrive P3 connector scope.

Trade-offs:

- DIF v1 cannot support mixed-permission corpora.
- Some enterprise pilots may be deferred until ACL propagation exists.
- Sales material must be explicit about the limitation.

---

## 10. Revisit trigger

Revisit this ADR when any of the following happens:

1. DIF is offered to a broad user population.
2. A pilot requires mixed-permission corpora.
3. SharePoint/OneDrive item-level permissions become required.
4. Production readiness/GA is complete and v2 planning begins.
5. D-006 ACL propagation work starts.

---

## 11. Acceptance criteria

ADR-003 is accepted when:

- Uniformly readable corpus is the only v1 source ACL posture.
- Corpus admission rules are documented.
- P3 connector limitation is documented.
- Sales/pilot language is documented.
- Runtime fail-closed behavior is documented.
- ACL propagation remains post-production-readiness/GA v2 work.


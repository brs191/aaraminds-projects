# Phase 6.1 Completion Summary
**Date:** 2026-06-17  
**Status:** ✅ **COMPLETE** — ADR-007 Accepted (Conditional)

---

## Deliverable: ADR-007 (Corrected & Reviewed)

**File:** `docs/architecture/adr/ADR-007-multi-source-reader-dedup.md` (749 lines, fully specified)

### What Changed

**Old (Draft):** CLI and IDE write to the same event stream → single schema

**New (Corrected):** CLI and IDE are **completely separate systems**
- **CLI:** `~/.copilot/session-state/<uuid>/events.jsonl` (JSONL, 53 sessions)
- **IDE:** `~/.config/github-copilot/ic/` (Nitrite DB, 116 sessions)

### Core Decisions (All Concrete)

| Aspect | Decision | Grounded in |
|--------|----------|-------------|
| **Source Enum** | `"cli"`, `"ide-chat"`, `"ide-edit"`, `"ide-agent"` | Reader.go, producer field |
| **Dedup Key** | `{source}:{sessionId}:{eventId}:{timestamp}` | Phase 6.0 discovery |
| **CLI Parser** | JSONL stream (existing cliCollector pattern) | reader.go |
| **IDE Parser** | Nitrite SDK (primary) + JSON metadata (fallback) | Verified from discovery |
| **Precedence** | Final > Partial; higher nanoAIU > lower | ADR-009 pattern |
| **Reporting** | Per-source breakdown (CLI authoritative, IDE estimated) | Cost semantics differ |

### Key Artifacts

✅ Real file paths (verified, not invented)  
✅ Real token field names (inputTokens, outputTokens, etc.)  
✅ Go pseudocode (Source enum, dedup algorithm, type shapes)  
✅ TypeScript examples (interfaces, enums)  
✅ Fallback error handling (explicit, defensive)  
✅ Test cases (dedup correctness, cross-source)

---

## Architecture Review: Conditional Accept ✅

**Reviewer:** aara-senior-microservices-architect (2026-06-17)

**Strengths:**
- Empirically grounded (cites Phase 6.0 discovery)
- Dedup logic concrete and sound
- Parser strategy realistic
- Source + TokenCostSource labels throughout
- Fallback handling explicit

**Pre-Implementation Blockers (3 items):**

1. **IDE Nitrite Schema Discovery** (BLOCKING)
   - Run `discover-ide-usage.sh` on IDE-only machine
   - Document actual collections, field names
   - Update ADR-007 §5 with verified schema

2. **TokenCount/TokenBreakdown Integration** (BLOCKING)
   - Clarify: is TokenBreakdown replaced or kept?
   - Prevent mid-implementation type redesign
   - Document in Phase 6.2 implementation spec

3. **Event vs. Session Dedup Boundary** (BLOCKING)
   - Is dedup at event level or session level?
   - Update pseudocode in ADR-007 §2–3

---

## What's Ready for Phase 6.2

### ✅ For Go Backend Engineer
- Source enum and stamping rules
- Real paths & event schemas
- JSONL parser pattern (CLI)
- Nitrite SDK strategy (IDE)
- Dedup key format & algorithm
- Type shapes (structs, interfaces)
- Fallback error handling
- Test cases

### ✅ For TypeScript Extension
- Source enum for UI
- Per-source reporting structure
- TokenCostSource labels (authoritative vs. estimated)
- Rendering guidance

---

## Timeline: Unblocking Phase

```
2026-06-17    Phase 6.1 Complete → ADR-007 Accepted (Conditional)
2026-06-17–24 UNBLOCKING PHASE (parallel work)
              - IDE schema discovery
              - TokenCount integration decision
              - Dedup boundary clarification
2026-06-24+   Phase 6.2 Ready → Implementation begins
```

---

## Decision Record

**Status:** Accepted (2026-06-17, Conditional on IDE discovery)

**Decision:**
> CLI and IDE are separate products requiring separate readers.
> Dedup by `(source, sessionId)` tuple.
> Parse JSONL (CLI) + Nitrite SDK + JSON fallback (IDE).
> Report per-source breakdown with caveats (CLI authoritative, IDE estimated).

**Re-review trigger:** If IDE schema requires parser change or TokenCount integration proves breaking

---

## Next Steps

- [ ] Review this summary
- [ ] Approve 3-item pre-implementation blocker delay
- [ ] Launch IDE discovery (discover-ide-usage.sh on IDE machine)
- [ ] Schedule TokenCount integration decision review
- [ ] Update ADR-007 with discoveries before Phase 6.2 start

---

**ADR-007 File:** `docs/architecture/adr/ADR-007-multi-source-reader-dedup.md`

**Phase 6.1 Status:** ✅ Complete — Ready for Phase 6.2 (pending unblocking)

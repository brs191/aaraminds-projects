# Phase 4 Acceptance Memo — Azure Network Topology Reviewer

| | |
|---|---|
| **Date** | 2026-06-15 |
| **Project** | Azure Network Topology Reviewer |
| **Phase** | Phase 4 — Enterprise Topology Visualization (in-session, fixture-provable scope) |
| **Reviewer** | Claude (Cowork) + 3 independent adversarial audit passes (subagents) |
| **Method** | Python reference implementation against a fixture corpus (the Phase-0 pattern), executed per `IMPLEMENTATION_PLAYBOOK_CLAUDE.md` |

---

## Phase 4 Verdict

> **✅ ACCEPTED (in-session scope) WITH DEFERRED LIVE ITEMS**

The deterministic visualization pipeline — discovery-stand-in → severity overlay → draw.io render →
diagram-eval gate — is proven on a 26-fixture corpus. All four production root causes (RC-1…RC-4) that
broke `generated_antr.pdf` are retired and **guarded by an automated gate**, plus two added gates
(structure, layout). Live discovery against Azure, the Go port, ELK/D2 layout, and the publish pipeline
are explicitly deferred (§6), consistent with the Phase-1 `[VERIFY]` posture.

The colour-integrity invariant — *severity is computed only by `Analyze()`, never the renderer* — was
attacked by three independent adversarial audits; every finding was remediated and re-verified (§4).

---

## Gate Verdict Table

| Gate | Criterion | Verdict | Evidence |
|---|---|---|---|
| G1 (RC-1/RC-2) | Cross-sub + spoke-to-spoke peering edges render to present nodes; zero dangling; out-of-scope peers become external stubs | **PASS** | `diagram_eval.json`: RC1_RC2_edges 10 PASS / 16 SKIP (no-peering); RC2_external_stub 3 PASS; 0 dangling across corpus |
| G2 (RC-3) | External-boundary nodes (Internet, Firewall, VPN/ER GW, NAT GW, public IP) render where present | **PASS** | RC3_boundary 13 PASS / 13 SKIP |
| G3 (RC-4) | Every finding-bearing node painted EXACTLY its `Analyze()` severity, on **both** HLD and MLD; CIDR findings render as edges; nothing dropped; no off-palette fill | **PASS** | RC4_colour_from_analyze 26/26 PASS; severity_coverage Critical/High/Medium/Info/Clean all True |
| G4 (P4-D5) | Severity computed only by `Analyze()` — renderer assigns none | **PASS** | overlay.py is the sole colour source (`style_for`); gate catches any renderer mispaint (§4 repros) |
| G5 (live) | Discovery via Managed Identity / OIDC, read-only, no `AZURE_CLIENT_SECRET` | **DEFERRED** | No live Azure in-session; design locked in VISUALIZATION_MODEL.md §7 (P4-D3) |
| + structure | Emitted XML has globally-unique cell ids; all edge endpoints exist | **PASS** | structure 26/26 PASS |
| + RC-5 layout | No sibling overlap AND no child-overflow (containment) | **PASS** | RC5_layout 26/26 PASS |

**Corpus:** 26 fixtures = 3 Phase-4 estates (`estate-multisub`, `estate-synth-large` [33 VNets/60 NICs/173
MLD vertices], `estate-cidr-overlap`) + 13 Phase-1 eval fixtures (+ generated synth). `overall_status: PASS`.

---

## Deliverables

| Component | File | Lines | Role |
|---|---|---|---|
| Severity overlay | `phase-4/viz/overlay.py` | 110 | joins `Analyze()` findings to kind-namespaced node ids (nic:/pip:/vnet:); sole colour source |
| Renderer | `phase-4/viz/render_drawio.py` | 327 | draw.io: peering+cross-sub edges, external stubs, boundary nodes, CIDR edges, severity paint, HLD/MLD; unique-id + no-dangling asserts |
| Layout validator | `phase-4/viz/check_layout.py` | 72 | sibling-overlap + child-containment |
| Diagram-eval gate | `phase-4/viz/eval_diagram.py` | 228 | RC1–RC5 + structure + severity coverage; two-level colour check |
| Scale generator | `phase-4/viz/synth_estate.py` | 144 | deterministic multi-hub/multi-sub estate generator |
| Fixtures | `phase-4/fixtures/*.json` | — | multisub, synth-large, cidr-overlap |
| Outputs | `phase-4/out/*.drawio`, `diagram_eval.json` | — | HLD+MLD diagrams; machine eval report |

The reference engine (`engine/reference/analyze.py`) and the Go engine are **unchanged** (P4-D2).

---

## Adversarial audit trail (3 independent passes)

Every finding was reproduced, fixed, and re-verified with a regression test using the auditor's own
break vector.

| # | Audit finding | Severity | Resolution | Re-verified |
|---|---|---|---|---|
| C-1 | NIC/PIP name collision mispaints + gate blind | Critical | overlay keyed by KIND (nic:/pip:/vnet:) | nic=Critical, pip=Low, distinct ✓ |
| H-1a | CIDR-overlap finding never rendered | High | dashed CIDR edge + folded into VNet rollup | edge drawn + both VNets Medium ✓ |
| H-1b | VNet rollup fill unchecked by gate | High | gate checks every VNet fill == rollup | green-vnet mispaint flagged ✓ |
| H-2a | duplicate NIC names → duplicate cell ids | High | fail-closed unique-id assert | dup names → AssertionError ✓ |
| H-2b | edge-id built from names → collide on `--` | High | counter-based edge ids; assert covers edges | `a/b--c` vs `a--b/c` → 0 dup ids ✓ |
| M-1 | external stub overlaps tall VNet | Medium | stub placed below max VNet bottom | 8-stub + tall VNet → 0 overlap ✓ |
| M-2 | layout overlap not in the gate | Medium | RC5 wired into eval | forced overlap → RC5 FAIL ✓ |
| M-3 | structural palette == severity palette | Medium | greys disjoint from 5 severity fills | ∅ intersection ✓ |
| A6 | HLD-only rollup mispaint passed the gate | High | colour checked on BOTH levels | HLD green mispaint → FAIL ✓ |
| A5 | severity coverage didn't gate | Medium | coverage gates overall_status | vacuous corpus → overall FAIL ✓ |
| L-1/L-2/self-loop | malformed peering / self-loop / empty | Low | defensive `.get`, self-peer skip, fail-closed | held ✓ |
| latent layout | subnets overflowed VNet box | (found by RC-5) | `vnet_height` contains children + pad | containment 26/26 PASS ✓ |

Quality invariants additionally proven: **render is byte-for-byte deterministic** (stable under
`PYTHONHASHSEED`), output is well-formed XML, and labels encode HTML line-breaks for clean draw.io import.

---

## Post-acceptance engineering fixes (from a 4th adversarial pass on the Go engine)

An independent read-only adversarial review of the Go engine surfaced determinism + keying defects.
Fixed and verified where the toolchain allowed:

| ID | Defect | Fix | Verified |
|---|---|---|---|
| V4-07 | Findings keyed by bare resource **name** → same-named NICs across subscriptions **merged at input** (`nics = {name: nic}` dropped one) | Python reference + viz now key by `rid()` = ARM id ‖ name (input map, NW-table lookup with name fallback, finding resource, render cell ids) | ✅ golden 5/5 + `test_resource_id.py` + 26/26 gate |
| HIGH-2 | `sort.Slice` (unstable) + key only `(Resource,Type)` → non-reproducible order on same-key findings | `sort.SliceStable` + Evidence tiebreaker (Go `analyze.go`); Python sort key extended to `(resource,type,evidence)` | ✅ Python; Go via CI |
| HIGH-1 | `time.Now()` in `renderer/markdown.go` + drawio meta → non-deterministic report bytes | wall-clock removed from the deterministic artifacts (Go) | Go via CI |
| LOW-1 | Python twin precedence bug: `+ "; ".join(why) or "not reachable"` made the fallback dead code | parenthesized the `or` | ✅ Python |

**Go portion of V4-07 (deferred, CI-gated):** the Go `NIC`/`PIP` structs have no `ID` field; full
parity requires adding `ID`, populating it in the adapter, and keying the Go `nics` map + NW lookups by
id-or-name. This is a multi-file change that needs the Go toolchain (Go 1.25 unavailable in-session) and
the new `engine-ci.yml` + `twin_drift_check.py` to verify — tracked as **V4-07-Go**. Until then the Python
reference is the corrected spec and the Go twin is verified-against-it by the twin-drift CI job.

---

## Deferred (needs live Azure / Go 1.25 — not completable in-session)

| ID | Item | Owner |
|---|---|---|
| D4-01 | Live multi-sub discovery via forked CloudNetDraw + Managed Identity (G5) | AT&T Network Ops + Eng |
| D4-02 | Port the Python overlay/renderer to the Go 1.25 engine (`engine/go/renderer`) | Engineering |
| D4-03 | ELK/D2 layout (binaries not in sandbox) for >100-node readability beyond grid | Engineering |
| D4-04 | Azure Function pipeline + Confluence/tWiki publish + version diff | Eng + AT&T Platform |
| D4-05 | Network Insights Topology completeness cross-check on the live estate | AT&T Network Ops |
| V4-06 | OSPO intake registration (CloudNetDraw/D2/ELK) — non-blocking, internal-use | AT&T OSPO |
| V4-07 | Engine: ARM-resource-id keying to disambiguate same-named resources cross-sub | Engineering |

---

## Sign-Off

| Role | Decision | Date |
|---|---|---|
| Technical (Claude + 3 adversarial audits) | **ACCEPTED — in-session scope; live items deferred** | 2026-06-15 |
| Phase lead | _(pending human sign-off)_ | — |

> **Next:** commit the antr workspace (still untracked in git), then schedule the D4 live-integration sprint.

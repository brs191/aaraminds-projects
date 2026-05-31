# M2-lite — Scope Contract

**Locked:** 2026-05-27  ·  **Owner:** Raja  ·  **Source of authority:** `../tracking/milestones/M1-Decision-Gate.md` (GREEN verdict 2026-05-27)
**Status:** Binding for the M2-lite build. Any change requires a written gate re-opening — not a casual scope expansion mid-build.

## Why this document exists

The GREEN verdict on 2026-05-27 cleared a $5k build, NOT the $75–110k M2 blueprint. The math works *because* the scope is narrow. Every feature added past the cut list below pushes payback past the 9-month GREEN threshold and retroactively voids the verdict.

This file is the contract engineers building reference. The in/out table is binding. Anyone asking for an OUT-list item during build gets one answer: **"out of scope by 2026-05-27 decision; re-open the gate or wait for the scale-trigger."**

## The cut list

### IN scope

| Component | Notes |
|---|---|
| Existing `../spike/` kit | LiteLLM proxy + LLMLingua-2 compression hook + measurement harness. Carries over as-is. |
| Pinned LiteLLM image | Tag pinned per `../spike/Dockerfile` (see task #1). No `main-stable` floats. |
| VS Code `.vsix` extension | Points the user's coding agent at the localhost LiteLLM proxy. Manual install (drag-drop or `code --install-extension`). |
| Manual install docs | `docker-compose up`, `.vsix` install, agent configuration. Plain markdown in the repo; no installer wizard. |
| One-time Fidelity Floor measurement | Run at install: a small fixture set proves R and quality on the user's machine before they trust the proxy. Captured in `metrics/install_fidelity.json`. NOT a continuous service. |
| Metadata-only egress | Existing in `spike/compression_hook.py` — preserved. No prompt content leaves the user's machine beyond what the assistant already sends. |

### OUT of scope

| Component | Why it was cut |
|---|---|
| IntelliJ plugin | Blueprint called for parity; cut because the pilot D=50 is VS Code-only (verify in cohort securing — see `M0-lite_Cohort_Recruitment.md`). IntelliJ plugin development is materially harder than VS Code and burns the $5k envelope on its own. |
| Bundled Go sidecar | Blueprint specified a Go core with supervised child processes. Cut: use `docker-compose` for lifecycle management. Trade-off: if docker-compose breaks the optimizer dies until the user fixes it. Acceptable at D=50. |
| Productized Fidelity Floor monitoring | Continuous monitoring of compression quality. Cut: replaced with the one-time at-install measurement. Trade-off: quality drift after install is not detected; mitigated by the 30-day pilot monitoring period catching gross regressions. |
| Automated supervisor for compression-sidecar lifecycle | Cut along with the Go sidecar. docker-compose's `restart: unless-stopped` covers basic crash recovery. |
| The 4 Required Fixes from `../evaluation/AI_Token_Optimizer_Systems_Review_2026-05-21.md` as built features | Documented as known operational gaps for the M2-lite pilot rather than implemented. These re-enter scope on the scale-trigger (see `../tracking/milestones/M1-Decision-Gate.md`). |
| Model routing | Out per the blueprint already; restating here for clarity. |
| Semantic caching | Out per the blueprint already. |
| Multi-user / team setup | Out — manual install per machine; no shared deployment. |
| Budget enforcement | Out — LiteLLM supports it but the spike kit explicitly disables the postgres dependency. Adding it later is a one-line config change, not an M2-lite deliverable. |

## Risks accepted by this scope

1. **docker-compose dies → optimizer dies.** No graceful degradation; user must fix manually. At D=50 with a pilot cohort, acceptable. At D=150+ it isn't — that's why the scale-trigger fires.
2. **No IntelliJ coverage.** If any pilot dev is IntelliJ-only they're excluded from the optimizer's benefit. Verify cohort homogeneity before kickoff.
3. **No continuous quality monitoring.** A model update upstream or a workload shift could degrade R or quality unnoticed between the install measurement and any future re-measure. The 30-day pilot monitoring period is the mitigation.
4. **Required-Fix surface remains as documented gaps.** Module 5 Systems Review identified 4 issues that the blueprint would have folded in. At M2-lite scope they are not built around; they are documented as pilot constraints. Re-read the Systems Review before pilot rollout so users know what they're testing.
5. **Single-IDE constraint cuts reachable cohort.** A 500-engineer team where half use IntelliJ has reachable D=250 even at full rollout. The GREEN math at D=50 doesn't surface this; the scale-trigger does.

## Trip-wires — when the contract is broken

Stop the build and re-open the gate if any of these come up during M2-lite development:

- **Estimated build effort exceeds 12 engineer-days.** $5k buys ~8 days at Hyderabad senior loaded; 12 days is a 50% overrun and the payback math no longer clears GREEN.
- **Any OUT-list item is requested as "just add this too."** The answer is "out of scope," not "let me estimate it." If the requester is leadership, route to the gate re-opening process, not the build.
- **M0-lite R-validation comes back below 10% incremental.** The verdict reverts; M2-lite does not build. See `../tracking/milestones/M1-Decision-Gate.md` for the post-mortem requirement.
- **The pilot cohort cannot be assembled as VS Code-only.** Forces either widening scope (IntelliJ) or shrinking cohort (math at smaller D may not clear). Re-litigate before building.

## Build-time budget

| Phase | Days | Notes |
|---|---|---|
| VS Code extension scaffold + localhost-proxy pointer | 2 | Standard `.vsix` template + LiteLLM endpoint config + agent wiring docs |
| One-time Fidelity Floor measurement script | 2 | Small fixture set, A/B compressed-vs-raw, writes `metrics/install_fidelity.json`. Reuses spike's `measure.py` patterns. |
| Manual install documentation | 1 | docker-compose up, `.vsix` install, agent configuration. README + troubleshooting. |
| Spike-kit hardening (pinned image, metadata-only egress verification) | 2 | Verify pin holds; verify egress controls; final smoke test. |
| Buffer | 1 | Reserved for the smallest version of "something unexpected." If used, re-evaluate scope. |
| **Total** | **8 days** | At Hyderabad senior loaded ~$10–14k/eng-month → ~$3.6–5k. Inside the $5k envelope. |

## Re-opening the contract

This contract is binding for the M2-lite build only. The triggers to re-open it (and re-litigate scope) are exactly the gate-re-opening conditions in `../tracking/milestones/M1-Decision-Gate.md`:

- Active D crosses 150 engineers.
- AITO commits to commercializing the optimizer externally.
- VS Code or Claude Code native compression baseline shifts materially.

When any of these fires, this file is replaced — not edited.

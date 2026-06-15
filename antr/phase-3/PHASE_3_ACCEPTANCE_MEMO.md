# Phase 3 Acceptance Memo — Azure Network Topology Reviewer
**Project:** Azure Network Topology Reviewer  
**Phase:** 3 — Topology Generation (`generate_topology` MCP tool)  
**Review date:** 2026-06-13  
**Reviewer:** Internal review (automated code analysis)  
**Status:** ACCEPTED WITH CONDITIONS

---

## Phase 3 Verdict

> **ACCEPTED WITH CONDITIONS**

All five gates PASS. Two non-blocking remediation items must be resolved before any production
`generate_topology` call is accepted:

1. **CI-01** — Phase 3 Python generator tests are not wired into `engine-ci.yml`.
2. **CI-02** — `docker login` in `deploy-mcp.yml` uses `${{ github.actor }}` as the JFrog username; confirm the JFrog registry accepts this or switch to the service account.

No blocking issues prevent Phase 4 work from beginning, provided the gate-pass path
remains behind stub mode (`GENERATOR_MODE=stub`) until `[VERIFY]` items V-04, V-05, and V-11
are resolved by the AT&T team.

---

## Gate Verdict Table

| Gate | Description | Verdict | Evidence |
|------|-------------|---------|----------|
| G1 | Fixture projection completeness | **PASS** | See §G1 below |
| G2 | LLM scope boundary | **PASS** | See §G2 below |
| G3 | ValidateBeforeEmit gate integrity | **PASS** | See §G3 below |
| G4 | Security posture | **PASS** | See §G4 below |
| G5 | Audit trail completeness | **PASS** | See §G5 below |

---

## G1 — Fixture Projection Completeness

**Verdict: PASS**

**Spot-check: sensitive=true + `allow-https-from-internet` → Critical finding**

Traced through `ProjectFixture` in `engine/go/generator/project.go`:

1. **`NIC.Tags["sensitive"]="true"`** — Rule 5 (line 373): `tags["sensitive"] = "true"` when `sn.Sensitive`.
2. **`NIC.PublicIP` set** — `syntheticPIPName()` (line 230): returns a non-empty PIP name when `subnetHasInternetIngress(sn) && !sn.RouteToFirewall`. The NIC's `PublicIP` pointer is set (lines 365–368).
3. **`EffectiveSecurityRules` include the Internet-sourced Allow rule** — Rule 7 (lines 397–413): `expandIntent("allow-https-from-internet")` returns `SecRule{SourceAddressPrefix: "Internet", Access: "Allow", ...}` (project.go line 63–68). `EffectiveSecurityRules[nicName]` is populated with this rule.
4. **`EffectiveRoutes` → `Internet`** — Rule 8 (lines 425–431): since `RouteToFirewall=false`, the route is `NextHopType="Internet"`.
5. **`analyze.Analyze()` fires Critical** — `ValidateBeforeEmit` calls `analyze.Analyze(fixture)` (validate.go line 38). The engine's internet-reachability gate detects: NIC with `sensitive=true` tag + `PublicIP != nil` + Inbound Allow from Internet in effective rules + route to Internet → Critical finding.

**Test coverage confirmed:** `TestGateFail_SensitiveNICWithInternetIngress` (generator_test.go line 131–170) exercises exactly this scenario and asserts `Approved=false` and at least one `Critical` severity finding.

**All other fixture fields checked:**
- `AVNM.SecurityAdminRules` — propagated from `baseline.AVNMSecurityAdminRules` (Rule 9, line 444).
- `AzureFirewall` — set when `spec.FirewallEnabled=true` (Rule 10, lines 449–457).
- `PrivateEndpoints` + `PrivateDnsZones` — populated from `PrivateEndpointSpec` (Rules 12–13, GR-004).
- `ApplicationGateways`, `AzureBastions` — projected from tier labels (Rules 14–15).

---

## G2 — LLM Scope Boundary

**Verdict: PASS**

**analyze.go has zero LLM calls.** The single grep match for "openai" in `analyze.go` is the
string `"privatelink.openai.azure.com"` in the `peGroupIdToZone` DNS map — a domain name, not
an API call.

**intent.py enforces the LLM boundary at the prompt level.** The system prompt (lines 262–272)
contains an explicit prohibition:

> *"You MUST NOT produce raw NSG security rule objects with fields: priority, access, direction,
> destinationPortRange, sourceAddressPrefix. Instead, express desired access as named intents
> from the approved vocabulary."*

The LLM response schema is `TopologySpec` JSON, which contains `nsg_intents: list[NSGIntent]`
— a closed enum of 16 values. The Pydantic model (`models.py`) enforces this via a
`Literal[...]` type annotation. Any raw SecRule field would fail Pydantic validation before
reaching the renderer.

**All security-relevant Terraform traces to approved modules.** `registry.go` enforces
`LoadRegistryFromFile` — module sources come from the YAML/JSON config only, never from
`TopologySpec` fields. `renderer.go` calls `registry.Select(capability)` (not
`spec.SomeModuleOverride`). Version pinning: entries with `>=`, `~>`, or `*` in Version are
rejected at load time.

**The explainer service** in `phase-1/explainer/` receives pre-computed `[]Finding` from the
engine — it has no input into severity or reachability determination.

---

## G3 — ValidateBeforeEmit Gate Integrity

**Verdict: PASS**

**No code path in `generateTopologyHandler` allows `CreatePR` to be called with
`validation.Approved=false`.**

Traced through `engine/go/mcp/tools.go` (lines 496–551):

```
if !validation.Approved {          // line 496
    // write audit
    // return gate-fail response   // line 516 — returns here, never reaches CreatePR
}
// ── only reached when validation.Approved == true ──
prURL, prErr := generator.CreatePR(...)  // line 520
```

**Defense-in-depth:** `CreatePR` (pr.go line 135) independently checks
`if !validation.Approved { return "", ErrGateFailed }`. Even if the handler had a logic error,
the gate cannot be bypassed by passing `ValidationResult{Approved: true}` without running
`ValidateBeforeEmit` — the caller must produce a `ValidationResult` struct, and the only
public constructor is `ValidateBeforeEmit(plan)` (no `force` flag, no `dry_run`).

**Refinement loop exhaustion path:** After `maxIter` iterations with `validation.Approved=false`,
the loop exits. The code falls into `if !validation.Approved` (line 496), which sets
`result.GatePass = false` in the JSON response and writes the gate-fail audit entry before
returning. `CreatePR` is never reached. ✅

**Render error path:** If `RenderTerraform` fails, `failingFindings` is set and `validation.Approved`
remains `false`. After loop exhaustion, same gate-fail path applies. ✅

---

## G4 — Security Posture

**Verdict: PASS**

| Check | Result | Evidence |
|-------|--------|----------|
| No `AZURE_CLIENT_SECRET` in CI | ✅ | `grep AZURE_CLIENT_SECRET .github/workflows/*.yml` → 0 matches |
| Azure auth via OIDC | ✅ | `deploy-mcp.yml` uses `azure/login@v2` with `client-id`, `tenant-id`, `subscription-id` as `vars.*` (repository variables, not secrets) |
| JFrog Artifactory used (not ACR) | ✅ | `jfrog/setup-jfrog-cli@v3`, `jf docker push` — no `azure/docker-login` or ACR reference |
| `JFROG_ACCESS_TOKEN` is a repository secret | ✅ | Referenced as `${{ secrets.JFROG_ACCESS_TOKEN }}` (line 93 of deploy-mcp.yml) |
| `GITHUB_TOKEN` not logged | ✅ | pr.go: token placed in `Authorization` header only; error messages return only HTTP status code (line 92: `"GitHub PR API returned status %d"`); response body excluded from errors |
| AskAT&T `client_secret` handling | ✅ | intent.py: read from `os.environ` on demand (line 367), placed in POST body (line 376), `del client_secret` in `finally` (line 392); `_BearerTokenRedactFilter` on all loggers; never stored as instance attribute |
| Managed Identity read-only | ✅ | deploy-mcp.yml documents: "Container Apps Contributor" on `nettopo-rg`; "Reader" for show — no write access outside one resource group, no terraform apply permission |
| Agent never runs `terraform apply` | ✅ | `generate_topology` outputs `TerraformPlan.Files` (HCL strings) into a PR only. No `exec.Command("terraform apply")` anywhere in the codebase |

---

## G5 — Audit Trail

**Verdict: PASS**

**Both paths write the audit entry before the MCP response is returned.**

| Path | Audit write line | Return line | Order |
|------|-----------------|-------------|-------|
| Gate-fail | tools.go:506 | tools.go:516 | ✅ audit before return |
| Gate-pass | tools.go:538 | tools.go:551 | ✅ audit before return |

**`auditGenerateTopoLine` fields confirmed present in `writeGenerateTopo` (audit.go lines 72–85):**

| Required field | Present | slog key |
|---------------|---------|---------|
| `spec_hash` | ✅ | `spec_hash` |
| `gate_pass` | ✅ | `gate_pass` |
| `iterations` | ✅ | `iterations` |
| `pr_url` | ✅ | `pr_url` |
| `findings_count` | ✅ | `findings` |
| `high_critical_count` | ✅ | `high_critical` |

Audit entries flow through `slog` structured-JSON (os.Stderr) — same pipeline as all server logs.
The `auditor != nil` guard in the handler ensures nil-safe operation in tests without auditor
wiring. In production, `auditor` is always non-nil (server.go line 62: `newAuditor(logger)`).

---

## Blocking Issues

**None.** All five gates pass. Phase 4 work may begin.

The following conditions must be resolved before any **production** `generate_topology` call
is made end-to-end (LLM → Render → Validate → PR):

| Condition | Tracking | Action required |
|-----------|----------|-----------------|
| V-04: Confirm `INFRA_REPO` value | Appendix A, GENERATION_MODEL.md | AT&T team: set `INFRA_REPO=<owner>/<repo>` in MCP server env |
| V-05: GitHub App token vs PAT | Appendix A, GENERATION_MODEL.md | AT&T team: confirm token type; update `GITHUB_TOKEN` secret |
| V-11: AskAT&T structured output API contract | Appendix A, GENERATION_MODEL.md | AT&T team: confirm `/chat/completions` endpoint supports `response_format.json_schema` |

Until these are confirmed, the server operates in `GENERATOR_MODE=stub` (deterministic spec,
`StubGitHubClient`) — safe for integration testing and demo.

---

## Non-Blocking Observations

### NB-01 — Phase 3 generator tests not in CI (CI-01)

**File:** `.github/workflows/engine-ci.yml`  
**Observation:** The CI workflow runs `phase-1/explainer/tests/` but does not run
`phase-3/generator/tests/` (12 Python tests). The workflow path filter also does not include
`phase-3/**`, so PRs touching `phase-3/generator/intent.py` will not trigger a CI run.  
**Recommended fix:** Add to `engine-ci.yml`:
- Path trigger: `phase-3/**`
- New job `python-generator` (after `go-engine`): `pip install -e "phase-3/generator[dev]"`, `pytest phase-3/generator/tests/ -v`

### NB-02 — JFrog `docker login` uses `github.actor` as username (CI-02)

**File:** `.github/workflows/deploy-mcp.yml` (lines 97–100)  
**Observation:** The JFrog docker login step passes `--username "${{ github.actor }}"`. JFrog
Artifactory accepts the access token as a bearer credential regardless of username, but some
JFrog registry configurations require a specific service account username or `_` when using
token auth. If the JFrog org policy requires a dedicated username, this will produce a
`401 Unauthorized` on push.  
**Recommended fix:** Confirm with AT&T JFrog admin whether `github.actor` is acceptable, or
switch to `--username att-nettopo-ci` (or `_`) per the org's JFrog convention.

### NB-03 — `peGroupIdToZone` duplicated across two files

**Files:** `engine/go/generator/project.go:16`, `engine/go/internal/analyze/analyze.go`  
**Observation:** The map is explicitly marked "MUST stay in sync manually". Adding a new
Private Endpoint service requires editing two files. If they drift, generated fixtures will
have mismatched DNS zone names, causing false-positive or missed `private-dns-zone-missing`
findings.  
**Recommended fix:** In a future cleanup, extract to a shared package (e.g.,
`internal/pezones/zones.go`) accessible by both `analyze` and `generator`.

### NB-04 — `RealGitHubClient` hardcodes `"base": "main"`

**File:** `engine/go/generator/pr.go` (line 63)  
**Observation:** The PR target branch is hardcoded to `"main"`. If `INFRA_REPO` uses `master`
or a different default branch, `CreatePull` will receive HTTP 422 (Unprocessable Entity) from
the GitHub API, and the error message will only contain the status code (non-disclosure by
design), making the failure opaque.  
**Recommended fix:** Read `INFRA_BASE_BRANCH` from env, default `"main"`.

### NB-05 — `ASKAT_CLIENT_SECRET` loaded from env (no Key Vault yet)

**File:** `phase-3/generator/intent.py` (line 367)  
**Observation:** `client_secret = os.environ["ASKAT_CLIENT_SECRET"]` — the secret is sourced
from the process environment. Key Vault integration is deferred as `[VERIFY V-11]`. In
Container Apps, the secret should be mounted as a Container Apps secret (backed by Key Vault
reference) rather than injected as a plain environment variable.  
**Recommended fix:** When V-11 is confirmed, wire `ASKAT_CLIENT_SECRET` as a Container Apps
secret with a Key Vault reference. No code change required in `intent.py` — the env variable
name stays the same.

---

## Recommended Phase 4 Start Conditions

Phase 4 may begin when all of the following are true:

| # | Condition | Owner |
|---|-----------|-------|
| P4-01 | NB-01 resolved: Phase 3 generator tests added to `engine-ci.yml` and passing in CI | Engineering |
| P4-02 | NB-02 resolved: JFrog docker login username confirmed with AT&T JFrog admin | AT&T Platform |
| P4-03 | V-04 confirmed: `INFRA_REPO` set; end-to-end stub PR created successfully in staging | AT&T Network Ops |
| P4-04 | V-05 confirmed: GitHub token type and scopes confirmed for infra repo | AT&T Platform |
| P4-05 | V-11 confirmed: AskAT&T structured output endpoint contract validated; `intent.py` wired to real client | AT&T AI Platform |

Phase 4 topic candidates (pending architecture decision):
- Live AVNM baseline propagation (V-01)
- Cost forecast integration with `forecast_cost` MCP tool from Phase 2 (`simulate_change` + `forecast_cost` in Phase 2 Steps 2.5–2.6 are still pending)
- `azure-network-topology-reviewer` container image publishing to JFrog with Phase 3 artifact versioning

---

## Outstanding [VERIFY] Items from GENERATION_MODEL.md Appendix A

| ID | Description | Status | Blocking? |
|----|-------------|--------|-----------|
| V-01 | AVNM API: `SecurityAdminRules` endpoint availability in target subscription | ⬜ Unconfirmed | Phase 4 (baseline) |
| V-02 | `analyze.go` `checkAVNMAdminRules` test against live AVNM security admin rule | ⬜ Unconfirmed | Phase 4 |
| V-03 | AT&T CAF module registry: internal Terraform module registry URL | ⬜ Unconfirmed | Phase 4 |
| **V-04** | Infrastructure Terraform repository name (`INFRA_REPO` value) | ⬜ **Critical** | Before production PR creation |
| **V-05** | GitHub App token vs PAT for infra repo PR creation | ⬜ **Critical** | Before production PR creation |
| V-06 | `azurerm_virtual_network_manager_connectivity_configuration` module: AT&T approved AVNM Terraform module | ⬜ Unconfirmed | Phase 4 |
| V-07 | Azure Firewall Standard SKU price in AT&T EA agreement | ⬜ Unconfirmed | Phase 2 CostForecast |
| V-08 | Flow Logs retention period in AT&T subscriptions (affects cost variable band) | ⬜ Unconfirmed | Phase 2 CostForecast |
| V-09 | AT&T naming conventions for VNet / NSG / route table resource names | ⬜ Unconfirmed | Phase 4 compliance |
| V-10 | AT&T policy: max VNet address space size and CIDR allocation process | ⬜ Unconfirmed | Phase 4 compliance |
| **V-11** | AskAT&T: structured output (`response_format.json_schema`) API contract | ⬜ **Critical** | Before real LLM wiring |
| V-12 | Phase 3 generator Python tests in CI trigger (new — identified this review) | ⬜ **NB-01** | Before Phase 4 CI gate |
| V-13 | AT&T private DNS zone policy: which zones must be pre-provisioned vs auto-created | ⬜ Unconfirmed | Phase 4 |
| V-14 | AT&T subscription vending model: is `spec.Region` the only input or do they use management groups? | ⬜ Unconfirmed | Phase 4 |
| V-15 | AT&T Terraform state backend: Azure Blob vs Terraform Cloud (affects rendered provider block) | ⬜ Unconfirmed | Phase 4 |

---

## Scope Summary

| Deliverable | Files | Tests | Status |
|-------------|-------|-------|--------|
| Phase 3 design (`GENERATION_MODEL.md`) | `phase-3/design/GENERATION_MODEL.md` | n/a | REVIEWED (post GR-001–006) |
| Terraform renderer | `engine/go/generator/{registry,spec,renderer,project,validate}.go` | 16 tests | ✅ |
| PR workflow | `engine/go/generator/pr.go` | 4 MCP tests | ✅ |
| `generate_topology` MCP tool | `engine/go/mcp/{tools,audit,server}.go` | 4 tests (20 total MCP) | ✅ |
| AskAT&T intent client | `phase-3/generator/intent.py` | 12 tests | ✅ (stub mode) |
| CI/CD | `.github/workflows/{engine-ci,deploy-mcp}.yml` | CI gate | ✅ |
| **Total Go tests** | — | **99/99 pass** | ✅ |
| **Total Python tests** | — | **12/12 pass** | ✅ |

---

*This memo was produced by automated code review against the Phase 3 deliverables. All citations
reference exact file paths and line numbers verified at review time (commit `8c7093c`).*

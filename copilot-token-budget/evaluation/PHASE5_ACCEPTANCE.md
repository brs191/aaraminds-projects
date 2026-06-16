# Copilot Token Budget — Phase 5 Acceptance Test Suite

**Phase:** 5 — Distribution & Onboarding
**Status:** Gates G51–G64 defined. G51–G59 (automated / locally validated) all pass. G60–G64 (manual / live) are blocked on JFrog provisioning + the first tagged release and cannot run in the sandbox.
**Date defined:** 2026-06-16

> **Honesty note.** Phase 5 delivered build/packaging/CI **configuration** that is complete and validated **locally** (cross-compile, archive contents, lint, extension packaging). The **live distribution path** — tag-triggered release, JFrog OIDC upload, GitHub Release creation, and real-OS install — has **never been run against real infrastructure**. It is config-validated only. Those gates (G60–G64) stay open until first provisioning + first tag.

---

## Gate summary

| Gate | Type | Description | Status |
|---|---|---|---|
| G51 | Automated | `goreleaser check` validates `.goreleaser.yaml` (v2) | ✅ |
| G52 | Automated | `goreleaser build --snapshot` yields 25 binaries (5×5), windows/arm64 absent | ✅ |
| G53 | Automated | Each archive contains README.md, USAGE.md, LICENSE, docs/onboarding-runbook.md | ✅ |
| G54 | Automated | `checksums.txt` present, sha256, one line per archive (25) | ✅ |
| G55 | Automated | `actionlint` clean on `ci.yml` and `release.yml` | ✅ |
| G56 | Automated | `ci.yml` builds/vets/tests (-race) + gofmt all 3 Go modules | ✅ |
| G57 | Automated | `ci.yml` compiles the VS Code extension | ✅ |
| G58 | Automated | `.vsix` packages clean — `out/` JS + manifest + README + LICENSE only; no src/.ts/.map/node_modules | ✅ |
| G59 | Automated | `--version` reports version/commit/date embedded via ldflags | ✅ |
| G60 | Manual / live | Pushing a `v*.*.*` tag triggers `release.yml` end-to-end | 🔲 |
| G61 | Manual / live | JFrog OIDC auth succeeds (`jf rt ping`) and `jf rt upload` lands all artifacts | 🔲 |
| G62 | Manual / live | GitHub Release is created with all archives + checksums + `.vsix` attached | 🔲 |
| G63 | Manual / live | Engineer installs from Artifactory and sees the status-bar badge in ≤5 min (runbook E2E) | 🔲 |
| G64 | Manual / live | Binaries run on real macOS and Windows (sandbox proved linux + cross-compile only) | 🔲 |

**Blocking gate for "Phase 5 config-complete":** G51–G59 must all pass. *(Met 2026-06-16.)*
**Blocking gate for "Phase 5 distribution live":** G60–G64 must all pass. *(Pending JFrog provisioning + first tag.)*

---

## Automated gates (G51–G59) — locally validated 2026-06-16

All commands assume the repo root unless stated. Toolchain: Go (GOTOOLCHAIN=auto), GoReleaser v2, Node, `actionlint`.

---

### G51 — GoReleaser config is valid

| Field | Value |
|---|---|
| **ID** | G51 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**
`.goreleaser.yaml` (schema v2) passes the GoReleaser linter. Validates the multi-module `builds:` layout (per-binary `dir:`), `release.disable: true`, archive definitions, and the snapshot template.

**How to run**
```bash
goreleaser check
```

**Pass criterion**
Exit code 0. Output: `1 configuration file(s) validated`. The `release is disabled` skip notice is expected (a CI job owns the release, not GoReleaser).

**Fail action**
Most likely cause: a v1→v2 schema drift (e.g. `format:`→`formats:`, `archives.replacements` removed). Reconcile against the GoReleaser v2 reference.

---

### G52 — 25 cross-compiled binaries, windows/arm64 excluded

| Field | Value |
|---|---|
| **ID** | G52 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**
A snapshot build produces all 5 binaries (copilot-analyze, copilot-dashboard, copilot-statusline, copilot-alert, copilot-budget-mcp) across 5 platforms (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64) = 25. windows/arm64 is intentionally ignored.

**How to run**
```bash
goreleaser build --snapshot --clean
find dist -type f \( -name 'copilot-*' -o -name '*.exe' \) -not -name '*.tar.gz' -not -name '*.zip' | wc -l   # expect 25
ls dist | grep -i windows_arm64 || echo "windows/arm64 absent (correct)"
```

**Pass criterion**
Exactly 25 built binaries; no windows/arm64 output.

**Fail action**
Check the `ignore:` block (windows+arm64) and `goos`/`goarch` matrices in each `builds:` entry.

---

### G53 — Archives carry the doc set

| Field | Value |
|---|---|
| **ID** | G53 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**
Every release archive bundles `README.md`, `USAGE.md`, `LICENSE`, and `docs/onboarding-runbook.md` alongside the binary. (LICENSE + runbook were added in Step 5.5 once those files existed.)

**How to run**
```bash
goreleaser release --snapshot --clean --skip=publish
tar tzf dist/copilot-analyze_*_linux_amd64.tar.gz       # tar.gz members
unzip -l dist/copilot-analyze_*_windows_amd64.zip       # zip members
```

**Pass criterion**
Each archive lists README.md, USAGE.md, LICENSE, docs/onboarding-runbook.md, and the binary.

**Fail action**
Add the missing entry to every archive's `files:` list in `.goreleaser.yaml` (there are 5 archive blocks).

---

### G54 — Checksums manifest (sha256)

| Field | Value |
|---|---|
| **ID** | G54 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**
`dist/checksums.txt` is generated with sha256 and covers every archive (25 lines).

**How to run**
```bash
wc -l dist/checksums.txt                 # expect 25
head -1 dist/checksums.txt               # 64-hex sha256 + filename
( cd dist && sha256sum -c checksums.txt ) 2>/dev/null | grep -c OK   # expect 25
```

**Pass criterion**
25 entries; each verifies against its archive; algorithm is sha256 (`checksum.algorithm: sha256`).

**Fail action**
Confirm `checksum.name_template` and `algorithm` in `.goreleaser.yaml`.

---

### G55 — Workflows pass actionlint

| Field | Value |
|---|---|
| **ID** | G55 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**
Both GitHub Actions workflows lint clean: no shell quoting issues, no unknown expressions, no deprecated action inputs.

**How to run**
```bash
actionlint .github/workflows/ci.yml .github/workflows/release.yml
```

**Pass criterion**
Exit code 0, no findings on either file.

**Fail action**
Address each reported line; common hits are unquoted `${{ }}` in `run:` blocks and unset permissions.

---

### G56 — CI builds/tests all 3 Go modules

| Field | Value |
|---|---|
| **ID** | G56 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**
`ci.yml` runs a matrix over `phase-1/session-manager`, `phase-3`, `phase-4`, each doing `go build`, `go vet`, `go test -race`, and a gofmt gate. Validated locally module-by-module.

**How to run**
```bash
for m in phase-1/session-manager phase-3 phase-4; do
  ( cd "$m" && go build ./... && go vet ./... && go test ./... -race && \
    [ -z "$(gofmt -l .)" ] && echo "$m OK" )
done
```

**Pass criterion**
All three modules: build/vet/test -race exit 0; gofmt reports no files.

**Fail action**
If only phase-4 fails to build under an older Go, confirm `GOTOOLCHAIN=auto` (phase-4 `go.mod` declares go 1.25); CI pins setup-go to 1.25.

---

### G57 — CI compiles the extension

| Field | Value |
|---|---|
| **ID** | G57 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**
`ci.yml` installs and `npm run compile` for `phase-2/vscode-extension` (Node 22). The TypeScript must transpile to `out/` without error.

**How to run**
```bash
cd phase-2/vscode-extension && npm install && npm run compile
```

**Pass criterion**
`out/` populated, exit code 0, no tsc errors.

**Fail action**
Resolve TypeScript errors; verify `tsconfig.json` `outDir: out`.

---

### G58 — `.vsix` packages clean

| Field | Value |
|---|---|
| **ID** | G58 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**
`vsce package --no-dependencies` produces a `.vsix` containing only the compiled `out/` JS, `package.json`, README, LICENSE, and the manifest — no `src/`, `.ts`, sourcemaps, or `node_modules` (enforced by `.vscodeignore`).

**How to run**
```bash
cd phase-2/vscode-extension && npm run compile
npx @vscode/vsce package --no-dependencies -o /tmp/ctb.vsix
unzip -l /tmp/ctb.vsix | grep -E '\.ts$|\.map$|node_modules|/src/' && echo BAD || echo CLEAN
```

**Pass criterion**
Package succeeds; the grep finds nothing (CLEAN); LICENSE.txt + readme.md + extension.vsixmanifest present. Marketplace id is `att-internal.copilot-token-budget`.

**Fail action**
Add the offending pattern to `.vscodeignore`.

---

### G59 — Version metadata embedded via ldflags

| Field | Value |
|---|---|
| **ID** | G59 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**
Each of the 5 mains declares `version`/`commit`/`date` vars and a `--version` flag; GoReleaser injects real values via `-X main.version=…` ldflags. A snapshot binary must report them (not the `dev`/`none`/`unknown` defaults).

**How to run**
```bash
./dist/copilot-analyze_linux_amd64_v1/copilot-analyze --version
./dist/copilot-budget-mcp_linux_amd64_v1/copilot-budget-mcp --version
```

**Pass criterion**
Output shows the snapshot version (e.g. `0.0.1-next`), a real commit hash, and an RFC3339 date — not the source defaults.

**Fail action**
Confirm the `ldflags:` `-X main.*` lines in each `builds:` entry and the var names in each main.

---

## Manual / live gates (G60–G64) — CANNOT run without infrastructure

These require a provisioned JFrog repo, a configured OIDC integration, and a real tag push. They are documented here so they are not silently skipped at go-live. **None has been executed.**

---

### G60 — Tag push triggers the release pipeline

| Field | Value |
|---|---|
| **ID** | G60 |
| **Type** | Manual / live |
| **Owner** | Release engineer |

**Description**
Pushing `vMAJOR.MINOR.PATCH` runs `release.yml`: `build-go` (GoReleaser), `build-vsix`, then `publish`. Validates the trigger filter and job graph against real GitHub Actions.

**How to run**
```bash
git tag v0.1.0 && git push origin v0.1.0
# watch the Actions run
```

**Pass criterion**
All three jobs run in order and succeed; `build-go` exposes a `dist` artifact, `build-vsix` a `vsix` artifact.

**Fail action**
Check the `on.push.tags` glob and `needs:` wiring.

---

### G61 — JFrog OIDC + upload

| Field | Value |
|---|---|
| **ID** | G61 |
| **Type** | Manual / live |
| **Owner** | Release engineer + Platform |

**Description**
`setup-jfrog-cli` exchanges the GitHub OIDC token (`id-token: write`) for a short-lived JFrog token via the `github-oidc` provider; `jf rt ping` succeeds and `jf rt upload` lands archives, `checksums.txt`, and the `.vsix` under `binaries/<tag>/` and `vsix/<tag>/`. No long-lived secret is used.

**Prerequisites (one-time, see `.github/workflows/README.md`)**
- Repo Variables `JF_URL`, `JF_BINARY_REPO`, `JF_VSIX_REPO` set.
- JFrog OIDC integration named `github-oidc` with an identity mapping scoped to this repo and deploy permission on both repos.

**How to run**
Triggered by G60. Inspect the `Verify JFrog auth` and upload steps.

**Pass criterion**
`jf rt ping` returns OK; uploads report success; artifacts visible in Artifactory under the expected paths.

**Fail action**
Re-check the OIDC provider name, identity mapping claim (`repository`), and repo deploy permissions.

---

### G62 — GitHub Release created with assets

| Field | Value |
|---|---|
| **ID** | G62 |
| **Type** | Manual / live |
| **Owner** | Release engineer |

**Description**
`softprops/action-gh-release@v2` (using `secrets.GITHUB_TOKEN`) creates the Release for the tag with all `*.tar.gz`, `*.zip`, `checksums.txt`, and the `.vsix` attached.

**How to run**
Triggered by G60. Inspect the Releases page for the tag.

**Pass criterion**
Release exists; all 25 archives + checksums + 1 `.vsix` are attached and downloadable.

**Fail action**
Verify the `files:` globs and that `contents: write` is granted on the publish job.

---

### G63 — Runbook E2E: install + status badge in ≤5 min

| Field | Value |
|---|---|
| **ID** | G63 |
| **Type** | Manual / live |
| **Owner** | Onboarding engineer |

**Description**
A fresh engineer follows `docs/onboarding-runbook.md`: pulls binaries from Artifactory, installs the `.vsix`, configures the Power Automate Workflows webhook, and sees the Copilot Token Budget status-bar badge — within 5 minutes.

**How to run**
Hand the runbook to someone who has not seen it; time them.

**Pass criterion**
Status-bar badge appears and reflects live token usage; total elapsed ≤5 min; no undocumented step needed.

**Fail action**
Patch the runbook for whatever step blocked them.

---

### G64 — Binaries run on real macOS + Windows

| Field | Value |
|---|---|
| **ID** | G64 |
| **Type** | Manual / live |
| **Owner** | QA |

**Description**
The cross-compiled darwin and windows binaries actually execute on real hardware/OS. The sandbox only proved linux execution + successful cross-compilation; native execution on darwin/windows is unverified.

**How to run**
On a Mac (Apple Silicon + Intel) and a Windows machine: extract the archive, run `--version` and a basic `analyze` invocation.

**Pass criterion**
Binaries launch (no codesign/SmartScreen hard block that prevents use), `--version` prints, and a basic command produces correct output.

**Fail action**
If macOS Gatekeeper / Windows SmartScreen blocks unsigned binaries, escalate code-signing as a follow-up (out of Phase 5 scope).

---

## Open risks carried out of Phase 5

1. **JFrog provisioning not done.** G61 cannot pass until the OIDC integration + target repos exist. Owner: Platform.
2. **License not finalized.** `LICENSE` is a proprietary placeholder with a `[VERIFY]` marker. Replace with the approved corporate license before any external distribution. Owner: Legal.
3. **Actions pinned to major tags, not SHAs.** Acceptable for now; Dependabot keeps them fresh. SHA-pinning is the recommended supply-chain hardening. Owner: Developer.
4. **Native macOS/Windows execution + code signing unverified** (G64). Owner: QA / Release.

# CI/CD Workflows

This directory holds the GitHub Actions pipelines for the Copilot Token Budget monorepo.

| Workflow | Trigger | Purpose |
| --- | --- | --- |
| `ci.yml` | push / PR to `main` | Build, vet, test (`-race`) + `gofmt` gate across the 3 Go modules; lint `.goreleaser.yaml`; compile the VS Code extension. |
| `release.yml` | tag `v[0-9]+.[0-9]+.[0-9]+` | Build all artifacts (GoReleaser + VSIX), publish to JFrog Artifactory over OIDC, cut the GitHub Release. |

## Required repository configuration

### Variables (Settings → Secrets and variables → Actions → Variables)

The release workflow reads these as `${{ vars.* }}`. They are non-secret config, so
they live as repo (or org) **Variables**, not Secrets.

| Variable | Example | Used for |
| --- | --- | --- |
| `JF_URL` | `https://yourco.jfrog.io` | JFrog Platform base URL for `setup-jfrog-cli`. |
| `JF_BINARY_REPO` | `copilot-binaries-local` | Target Artifactory repo for Go archives + `checksums.txt`. Uploaded under `binaries/<tag>/`. |
| `JF_VSIX_REPO` | `copilot-vsix-local` | Target Artifactory repo for the `.vsix`. Uploaded under `vsix/<tag>/`. |

### Secrets

**None to create.** Authentication is entirely keyless:

- **JFrog** — OIDC. No JF token is stored. The release job requests `id-token: write`
  and `setup-jfrog-cli` exchanges the GitHub OIDC token for a short-lived JFrog token.
- **GitHub Release** — the auto-provisioned `secrets.GITHUB_TOKEN` (no PAT needed).

This satisfies ADR-005 (JFrog Artifactory as the artifact registry; never Azure ACR)
and the no-hardcoded-secrets constraint.

## JFrog OIDC provider setup (one-time)

1. In the JFrog Platform: **Administration → General → Manage Integrations → OIDC**.
2. Create an OIDC integration named **`github-oidc`** (this exact name is referenced
   by `oidc-provider-name` in `release.yml`).
   - Provider type: GitHub.
   - Audience / issuer: `https://token.actions.githubusercontent.com`.
3. Add an **Identity Mapping** scoping the trust to this repo
   (claim `repository = <org>/copilot-token-budget`, optionally `ref` = the tag) and
   bind it to a JFrog identity/role with deploy permission on `JF_BINARY_REPO` and
   `JF_VSIX_REPO`.
4. Confirm with the `jf rt ping` step in the release run.

To rename the provider, update `oidc-provider-name` in `release.yml` to match.

## Tag-to-release flow

```bash
git tag v1.2.3
git push origin v1.2.3
```

That triggers `release.yml`:

1. **build-go** — `goreleaser release --clean` builds 5 binaries × 5 platforms,
   archives them (`.tar.gz`, `.zip` on Windows), writes `checksums.txt` into `dist/`.
   `.goreleaser.yaml` has `release.disable: true`, so GoReleaser does **not** create
   the GitHub release itself.
2. **build-vsix** — packages the VS Code extension into
   `copilot-token-budget-<tag>.vsix` (Node 22, required by `@vscode/vsce`).
3. **publish** — downloads both artifacts, uploads to JFrog Artifactory over OIDC
   (`binaries/<tag>/` and `vsix/<tag>/`), then cuts the GitHub Release with all
   archives, `checksums.txt`, and the `.vsix` attached.

## Hardening follow-up: SHA-pin actions

Actions are currently pinned to **major tags** (`@v4`, `@v7`, …), which is acceptable
for now. The recommended hardening is to pin to full commit **SHAs** to defeat tag
re-pointing supply-chain attacks. Dependabot (`github-actions` ecosystem, weekly) is
configured to keep whichever form is in use current, so the SHAs stay fresh once you
switch. Optional build-provenance attestation is included as a commented step in
`release.yml` (requires `attestations: write`; private repos need GitHub Enterprise Cloud).

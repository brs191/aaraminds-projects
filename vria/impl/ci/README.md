# VRIA CI Release Gate

This directory contains the GitHub Actions workflow that enforces the
07_VRIA_Golden_Eval_Set.md §3 release gate on every push and pull request.

## Usage

Copy the workflow file to your repository's workflow directory before pushing
to GitHub:

```
cp impl/ci/github-actions-release-gate.yml .github/workflows/vria-release-gate.yml
```

The workflow is not active until it lives under `.github/workflows/`.
The `impl/ci/` location is a staging area only.

## What the workflow enforces

| Step | Gate | Threshold |
|---|---|---|
| `go test ./goldeneval/ -run TestGE` | Critical golden tests (GE-002,003,004,005,006,007,010,011,013,015) | 100% pass — blocks merge |
| `go test ./...` | All package tests including non-critical golden tests | Must pass |
| `go test ./goldeneval/ -run TestVolume -v` | Volume dataset gates (printed in log) | value-state ≥90%, schema 100%, recommendation ≥90%, normalize ≥95% |

## Failure interpretation

- Any critical golden test failure blocks the merge immediately.
- Any volume gate below threshold prints the failing record IDs and gate
  percentages in the Actions log. Fix the affected records or the engine
  and re-push.

## Go version

The workflow runs Go 1.22 in CI (current stable). The module itself targets
go 1.13 (no generics, no t.TempDir) so the binary is compatible across
environments.

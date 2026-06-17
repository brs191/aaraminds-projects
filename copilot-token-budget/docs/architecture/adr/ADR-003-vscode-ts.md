# ADR-003 — VS Code extension in TypeScript, zero runtime npm dependencies

**Status:** Accepted
**Date:** 2026-06-13

## Context

The VS Code extension needs to read the same session state files and render them in the editor.

## Decision

TypeScript 5.4, `@types/vscode` and `@types/node` as devDependencies only. No runtime npm packages.

## Rationale

- AT&T Artifactory npm registry requires auth (currently broken); zero runtime deps means
  `node_modules/` is empty in the packaged `.vsix`
- The VS Code API provides everything needed (TreeDataProvider, WebviewPanel, StatusBarItem)
- Node.js `readline` (built-in) handles JSONL parsing
- `path`, `fs`, `os` (all built-in) handle file I/O

## Build workaround

AT&T npm registry is at `artifact.it.att.com/artifactory/api/npm/npm-all/` and requires Artifactory
credentials. For dev installs, use:
```bash
npm install --registry https://registry.npmjs.org
```
Or add `.npmrc` to the extension folder:
```
registry=https://registry.npmjs.org
```

## Consequences

- Extension compiles to plain JavaScript in `out/` — no bundler needed
- Package size is small (just the compiled JS, no node_modules in .vsix)

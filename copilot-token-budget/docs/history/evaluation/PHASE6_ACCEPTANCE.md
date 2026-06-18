# Copilot Token Budget — Phase 6 Acceptance Test Suite

**Phase:** 6 — IDE + CLI Multi-Source Capture  
**Status:** ✅ Complete (2026-06-17)  
**Date defined:** 2026-06-17

> **Correction.** The earlier `vscode.metadata.json` / Nitrite marker assumption was wrong. The current
> implementation uses the standard VS Code user-data transcript locations for IDE usage and keeps CLI
> sessions on `~/.copilot/session-state/`. This suite defines the actual local-only acceptance gates for
> the shipped implementation.

---

## Gate summary

| Gate | Type | Description | Status |
|---|---|---|---|
| G65 | Automated | IDE sessions are discovered from standard VS Code user-data transcript paths and stamped `copilot-ide` | ✅ |
| G66 | Automated | Event-level dedup prevents duplicate billing (`{source}:{sessionId}:{eventId}`) | ✅ |
| G67 | Automated | `apiCallId` dedup keeps the earliest event and discards later retries | ✅ |
| G68 | Automated | CLI + IDE merge stays source-scoped and preserves additive totals | ✅ |
| G69 | Automated | Dashboard renders CLI Sessions / IDE Sessions and CLI Credits / IDE Credits | ✅ |
| G70 | Automated | Missing IDE source degrades cleanly to CLI-only mode | ✅ |

**Blocking gate for "Phase 6 complete":** G65–G70 must all pass. *(Met 2026-06-17.)*

---

## Automated gates (G65–G70)

All commands assume the repo root unless stated. Toolchain: Go, Node.js, TypeScript.

---

### G65 — Standard VS Code IDE paths are discovered locally

| Field | Value |
|---|---|
| **ID** | G65 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

The IDE collector must discover local VS Code Copilot Chat transcripts from the standard user-data tree:
`workspaceStorage/<ws>/chatSessions/`, `globalStorage/GitHub.copilot-chat/transcripts/`, and
`globalStorage/emptyWindowChatSessions/` (platform-specific roots included). Discovered sessions must be
stamped `copilot-ide`.

**How to run**

```bash
cd extension
npm run compile
node out/session/reader.test.js
```

**Pass criterion**

- `testIDEStandardPathShape` passes
- `testIDEStandardPathSurface` passes
- `readSessions()` surfaces at least one `copilot-ide` session from a fake VS Code user-data tree
- The surfaced IDE session uses the expected workspace and billing fields from the transcript

---

### G66 — Event-level dedup prevents duplicate billing

| Field | Value |
|---|---|
| **ID** | G66 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

Within a single IDE session, duplicate events must be ignored using the event keying strategy in the
collector so the same event is not counted twice.

**How to run**

```bash
cd extension
npm run compile
node out/session/reader.test.js
```

**Pass criterion**

- `testIDEDedup` passes
- Duplicate events are skipped
- Token totals do not change when the same event appears twice

---

### G67 — `apiCallId` dedup keeps the earliest event

| Field | Value |
|---|---|
| **ID** | G67 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

Multiple `assistant.message` events can share the same `apiCallId`. The collector must keep the earliest
event by timestamp and discard later retries or resumptions.

**How to run**

```bash
cd extension
npm run compile
node out/session/reader.test.js
```

**Pass criterion**

- `testIDEAPICallIDDedup` passes
- Earliest-wins behavior is preserved
- Later attempts are discarded and not summed

---

### G68 — CLI + IDE merge stays source-scoped and additive

| Field | Value |
|---|---|
| **ID** | G68 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

CLI and IDE sessions must remain distinct by source. Dedup is keyed by `{source}:{id}`, and the merged
result must preserve both source totals without cross-source collisions.

**How to run**

```bash
cd extension
npm run compile
node out/session/reader.test.js
```

**Pass criterion**

- `testCLIAndIDEMerge` passes
- CLI and IDE sessions with the same session id remain separate because the source is part of the key
- Combined totals equal CLI total + IDE total when no overlaps exist

---

### G69 — Dashboard renders CLI and IDE breakdowns

| Field | Value |
|---|---|
| **ID** | G69 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

The dashboard must show separate CLI and IDE session/credit counts, plus source labels in the session
table and the “Source Breakdown” section.

**How to run**

```bash
cd extension
npm run compile
node out/session/reader.test.js
grep -n "CLI Sessions:\|IDE Sessions:\|Source Breakdown\|Source" src/ui/dashboardPanel.ts
```

**Pass criterion**

- `testDashboardSourceBreakdown` passes
- `dashboardPanel.ts` contains the CLI/IDE labels and source breakdown text
- Credits display as whole numbers only

---

### G70 — Missing IDE source degrades cleanly

| Field | Value |
|---|---|
| **ID** | G70 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**

If the IDE transcript tree is missing or empty, the collector must return CLI results without throwing or
turning the absence into an error.

**How to run**

```bash
cd extension
npm run compile
node out/session/reader.test.js
```

**Pass criterion**

- `testIDEDegradation` passes
- `readSessions()` still returns CLI sessions when IDE data is absent
- Missing IDE data is treated as empty, not fatal

---

## Summary

**Phase 6 acceptance is met locally.**

| Gate | Validates | Evidence |
|---|---|---|
| G65 | Standard VS Code IDE discovery | `testIDEStandardPathShape`, `testIDEStandardPathSurface` |
| G66 | Duplicate-event suppression | `testIDEDedup` |
| G67 | Earliest-wins `apiCallId` handling | `testIDEAPICallIDDedup` |
| G68 | Source-scoped merge and additive totals | `testCLIAndIDEMerge` |
| G69 | Dashboard source breakdown rendering | `testDashboardSourceBreakdown` + `dashboardPanel.ts` |
| G70 | CLI-only graceful degradation | `testIDEDegradation` |

**Acceptance:** the IDE collector, merge logic, and dashboard are now aligned with the current local-first implementation.

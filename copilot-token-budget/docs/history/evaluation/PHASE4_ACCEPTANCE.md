# Copilot Token Budget — Phase 4 Acceptance Test Suite

**Phase:** 4 — MCP Server  
**Status:** Gates G23–G32 defined. Run G23–G30 (automated) before Phase 5; G31–G32 (manual / tech debt) before distribution.  
**Date defined:** 2026-06-14  

---

## Gate summary

| Gate | Type | Description | Status |
|---|---|---|---|
| G23 | Automated | `go build ./...` exits 0 | ✅ |
| G24 | Automated | `go test ./...` exits 0 | ✅ |
| G25 | Automated | `go test -race ./...` exits 0 | ✅ |
| G26 | Automated | Startup: first MCP response within 2 seconds | ✅ |
| G27 | Automated | Arithmetic parity: MCP vs `cmd/analyze` ≤ 1.0 cr diff | ✅ |
| G28 | Automated | Path traversal rejected (relative, `/etc`, outside home) | ✅ |
| G29 | Automated | Zero network calls from all 4 tool handlers | ✅ |
| G30 | Automated | Stdout clean — no MCP framing corruption | ✅ |
| G31 | Manual | Copilot CLI invokes all 4 tools via `.copilot/mcp.json` | 🔲 |
| G32 | Tech debt | go-sdk pinned to commit hash (not semver tag) | 🔲 |

**Blocking gate for Phase 5 start:** G23–G30 must all pass.  
**Blocking gate for distribution (Phase 5 complete):** All gates must pass.

---

## Automated gates (G23–G30)

---

### G23 — Build succeeds

| Field | Value |
|---|---|
| **ID** | G23 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
The MCP server and all supporting packages compile without error. Validates import paths, type signatures, and module replace directives.

**How to run**
```bash
cd copilot-token-budget/phase-4
go build ./...
```

**Pass criterion**  
Exit code 0. No compiler errors.

**Fail action**  
Most likely cause: MCP SDK API change between go-sdk versions. Check handler signature against `mcp.AddTool` in the SDK source.

---

### G24 — Unit and integration tests pass

| Field | Value |
|---|---|
| **ID** | G24 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
All tests in `phase-4/` pass. Covers: path validation (5 security tests), functional correctness (3 tests), arithmetic parity (1 test), startup time (1 test), zero-network (1 test).

**How to run**
```bash
cd copilot-token-budget/phase-4
go test ./...
```

**Pass criterion**  
Exit code 0. Output shows `ok github.com/aaraminds/copilot-session-manager/phase4`.  
`TestArithmeticParity` must PASS (not SKIP) — if it skips, session data is missing for the current month.

**Fail action**  
`TestArithmeticParity` SKIP: run `cmd/analyze` manually to confirm session data exists.  
Security test failure: path validation logic in `internal/tools/validate.go` is broken — CRITICAL, fix immediately.

---

### G25 — Race detector clean

| Field | Value |
|---|---|
| **ID** | G25 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
The Go race detector finds no data races. All 4 tool handlers are pure-functional (no shared mutable state), so this should always pass. Critical at 1,000+ concurrent engineers where the MCP client may issue parallel tool calls.

**How to run**
```bash
cd copilot-token-budget/phase-4
go test -race ./...
```

**Pass criterion**  
Exit code 0. No `DATA RACE` lines in output.

**Fail action**  
Any race is CRITICAL. Tool handlers must remain stateless — no package-level variables written during tool calls.

---

### G26 — Startup time: first MCP response ≤ 2 seconds

| Field | Value |
|---|---|
| **ID** | G26 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
The server must produce its first MCP response quickly, confirming no heavy initialisation (file scanning, session reading) at startup. All I/O is deferred to individual tool call handlers.

The 2-second ceiling accounts for OS process creation and Go runtime startup (~200–500 ms on macOS); the architectural requirement is zero file I/O before the first tool call.

**How to run**
```bash
cd copilot-token-budget/phase-4
go test -v -run TestStartupTime ./...
```

**Pass criterion**  
`TestStartupTime` passes. Log line shows server initialisation time — should be well under 2 seconds on any AT&T MacBook.

**Additional check**
```bash
# Confirm no file scan at startup — no glob/readdir calls before first tool invocation
grep -rn "ReadThisMonth\|ScanWorkspace\|os.ReadDir" cmd/mcp-server/main.go || echo "no I/O at startup — OK"
```

**Fail action**  
If startup > 2 seconds: a file scan or session read was added to `main()`. Move it inside the tool handler.

---

### G27 — Arithmetic parity: MCP vs `cmd/analyze` ≤ 1.0 cr

| Field | Value |
|---|---|
| **ID** | G27 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
`get_budget_status` must return the same credit total as `cmd/analyze` for the same session data. Both call `budget.Calculate(nanoAIUs, 0)` — any divergence indicates a bug in the MCP nanoAIU accumulation or unit-conversion path.

**How to run**
```bash
cd copilot-token-budget/phase-4
go test -v -run TestArithmeticParity -count=1 ./...
```

**Pass criterion**  
Log line: `parity OK: MCP=XXXX.XXXX CLI=XXXX.XXXX diff=0.XXXX`  
`|mcp_credits - cli_credits| ≤ 1.0`

**Baseline from June 2026 live data:**  
`MCP=8236.5483 CLI=8236.5500 diff=0.0017` ✅

**Fail action**  
Diff > 1.0: check `budget.go` — the nanoAIU accumulation loop (`if s.TotalNanoAIU > 0`) must match the filter in `cmd/analyze`. Negative nanoAIU sessions are excluded in both paths.

---

### G28 — Path traversal rejected for all 4 tools

| Field | Value |
|---|---|
| **ID** | G28 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
All four tool handlers must reject:
1. Relative paths (e.g. `"relative/path"`, `"."`, `"../escape"`)
2. Absolute paths outside the user home directory (e.g. `"/etc/passwd"`, `"/tmp"`)

This prevents a malicious MCP client from reading arbitrary files off a 1,000-engineer fleet.

**How to run**
```bash
cd copilot-token-budget/phase-4
go test -v -run "Rejected|Traversal" ./...
```

**Pass criterion**  
All 5 security tests pass:
- `TestGetBudgetStatus_RelativePathRejected`
- `TestGetBudgetStatus_TraversalRejected`
- `TestGetSessions_RelativePathRejected`
- `TestGetInstructionOverhead_TraversalRejected`
- `TestGetModelCosts_RelativePathRejected`

Each returns a non-nil error containing `"must be absolute"` or `"home directory"`.

**Fail action**  
CRITICAL — path validation bypass allows reading `/etc/passwd`, `/etc/hosts`, etc. Fix `validateWorkspacePath` in `internal/tools/validate.go` before any further work.

---

### G29 — Zero network calls from all 4 tool handlers

| Field | Value |
|---|---|
| **ID** | G29 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
The tool is local-first (ADR-001). No tool handler may make an outbound HTTP call. This is tested by replacing `http.DefaultTransport` with a blocking transport that fails the test on any request.

**How to run**
```bash
cd copilot-token-budget/phase-4
go test -v -run TestNoNetworkCalls ./...
```

**Pass criterion**  
`TestNoNetworkCalls` passes. No `unexpected HTTP call` error lines.

**Fail action**  
Any HTTP call is CRITICAL — AT&T network egress is monitored. Identify which tool handler introduced the call and remove it. All data comes from `~/.copilot/session-state/` only.

---

### G30 — Stdout clean: no MCP protocol framing corruption

| Field | Value |
|---|---|
| **ID** | G30 |
| **Type** | Automated |
| **Owner** | Developer |

**Description**  
The MCP server communicates exclusively via stdout using JSON-RPC framing. Any `fmt.Print`, `fmt.Println`, or `log` output to stdout corrupts the framing and causes the MCP client to disconnect.

**How to run**
```bash
cd copilot-token-budget/phase-4

# Static check: no direct stdout writes in production code
grep -rn "fmt\.Print\b\|fmt\.Println\|fmt\.Fprintf(os\.Stdout" cmd/ internal/ --include="*.go" \
  | grep -v "_test.go" || echo "no stdout pollution — PASS"

# Runtime check: build binary and confirm --debug sends to stderr only
go build -o /tmp/copilot-budget-mcp-test ./cmd/mcp-server
/tmp/copilot-budget-mcp-test --debug < /dev/null 2>/dev/null
echo "stdout bytes on empty input: $(echo | /tmp/copilot-budget-mcp-test --debug 2>/dev/null | wc -c)"
```

**Pass criterion**  
- `grep` returns no matches
- All log output (including `--debug`) goes to stderr, never stdout
- `log.SetOutput(io.Discard)` is the default (non-debug) path in `main.go`

**Fail action**  
Any `fmt.Print` to stdout is CRITICAL — it will break every MCP client on every AT&T machine.  
Check that `fmt.Fprintf(os.Stderr, ...)` is used for the fatal error path in `main.go`.

---

## Manual gates (G31–G32)

---

### G31 — Copilot CLI invokes all 4 tools via `.copilot/mcp.json`

| Field | Value |
|---|---|
| **ID** | G31 |
| **Type** | Manual — integration |
| **Owner** | Developer |

**Description**  
The MCP server must be discoverable and invokable by the Copilot CLI using the `.copilot/mcp.json` registration file at the repo root.

**Pre-conditions**
- MCP server binary built and placed at `~/bin/copilot-budget-mcp`:
  ```bash
  cd copilot-token-budget/phase-4
  go build -ldflags "-X main.Version=v0.1.0" -o ~/bin/copilot-budget-mcp ./cmd/mcp-server
  ```
- `.copilot/mcp.json` exists at repo root (already created — points to `~/bin/copilot-budget-mcp`)

**How to run**  
In a Copilot CLI session within the `copilot-token-budget/` workspace, invoke each tool:

```
Ask Copilot: "What is my current Copilot budget status? Use the get_budget_status MCP tool."
Ask Copilot: "List my Copilot sessions this month using get_sessions."
Ask Copilot: "Audit my instruction file overhead using get_instruction_overhead."
Ask Copilot: "Show me model cost breakdown using get_model_costs."
```

**Pass criterion**  
- Copilot CLI discovers and calls the MCP server without error
- `get_budget_status` returns `credits`, `pct`, `status`, `daysLeft`, `forecast` fields
- `get_sessions` returns all sessions this month with an `isActive` flag, sorted by `credits` desc, each with `name`, `model`, `credits`
- `get_instruction_overhead` returns files sorted by token count descending
- `get_model_costs` returns a model → cost map with rate cards
- No `"command not found"` or MCP transport errors

**Fail action**  
Binary not found: rebuild with `go build -o ~/bin/copilot-budget-mcp ./cmd/mcp-server`.  
MCP framing error: run `~/bin/copilot-budget-mcp --debug` directly and send a raw initialize message to inspect stderr.

---

### G32 — go-sdk pinned to commit hash (tech debt)

| Field | Value |
|---|---|
| **ID** | G32 |
| **Type** | Tech debt — must resolve before distribution |
| **Owner** | Developer |

**Description**  
ADR-002 exception requires: *"Pin to an EXPLICIT COMMIT HASH — never @latest or a semver range."*  
Currently `go.mod` uses `v1.6.1` (semver tag). While `go.sum` provides cryptographic tamper-detection, a semver tag can be force-pushed/deleted without registry-level protection.

This gate must be resolved before distributing to 1,000+ machines in Phase 5.

**How to run**
```bash
grep "modelcontextprotocol" copilot-token-budget/phase-4/go.mod
```

**Pass criterion**  
The go-sdk line uses a pseudo-version (commit hash), e.g.:
```
github.com/modelcontextprotocol/go-sdk v0.0.0-20250601123456-abcdef012345
```
NOT:
```
github.com/modelcontextprotocol/go-sdk v1.6.1
```

**How to fix**
```bash
cd copilot-token-budget/phase-4

# Get the commit hash behind v1.6.1
HASH=$(go mod download -json github.com/modelcontextprotocol/go-sdk@v1.6.1 | python3 -c "import sys,json; print(json.load(sys.stdin)['Version'])")
echo "Current pseudo-version: $HASH"

# If already a pseudo-version, done. If still v1.6.1, pin explicitly:
# Find the commit tag in the SDK repo and use:
go get github.com/modelcontextprotocol/go-sdk@<commit-sha>
go mod tidy
```

**Fail action**  
Do not distribute Phase 5 binaries until this gate passes. The risk is low (go.sum prevents silent substitution) but the ADR must be honoured before enterprise rollout.

---

## Gate ownership and blocking policy

| Gate tier | Gates | Blocking for |
|---|---|---|
| Automated (must pass CI) | G23, G24, G25, G28, G29, G30 | Phase 5 start |
| Automated (full suite) | G23–G30 | Phase 5 distribution build |
| Manual integration | G31 | Phase 5 distribution build |
| Tech debt | G32 | Phase 5 distribution (enterprise rollout) |

---

## Acceptance sign-off

| Gate | Tester | Date | Result |
|---|---|---|---|
| G23 | Developer | 2026-06-14 | ✅ |
| G24 | Developer | 2026-06-14 | ✅ |
| G25 | Developer | 2026-06-14 | ✅ |
| G26 | Developer | 2026-06-14 | ✅ |
| G27 | Developer | 2026-06-14 | ✅ |
| G28 | Developer | 2026-06-14 | ✅ |
| G29 | Developer | 2026-06-14 | ✅ |
| G30 | Developer | 2026-06-14 | ✅ |
| G31 | | | 🔲 |
| G32 | | | 🔲 |

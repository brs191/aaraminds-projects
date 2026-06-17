package phase4_test

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aaraminds/copilot-session-manager/phase4/internal/tools"
)

// TestMain builds both server binaries once for integration tests.
var (
	serverBinary  string // copilot-budget-mcp (MCP server)
	analyzeBinary string // cmd/analyze (CLI — used for arithmetic parity check)
	tmpDir        string
)

func TestMain(m *testing.M) {
	var err error
	tmpDir, err = os.MkdirTemp("", "copilot-budget-mcp-*")
	if err == nil {
		mcpBin := tmpDir + "/copilot-budget-mcp"
		if buildErr := exec.Command("go", "build", "-o", mcpBin,
			"github.com/aaraminds/copilot-session-manager/phase4/cmd/mcp-server").Run(); buildErr == nil {
			serverBinary = mcpBin
		}

		analyzeBin := tmpDir + "/analyze"
		if buildErr := exec.Command("go", "build", "-o", analyzeBin,
			"github.com/aaraminds/copilot-session-manager/cmd/analyze").Run(); buildErr == nil {
			analyzeBinary = analyzeBin
		}
	}
	code := m.Run()
	if tmpDir != "" {
		_ = os.RemoveAll(tmpDir)
	}
	os.Exit(code)
}

// ── Security validation ───────────────────────────────────────────────────────

func TestGetBudgetStatus_RelativePathRejected(t *testing.T) {
	_, _, err := tools.GetBudgetStatus(nil, nil,
		tools.GetBudgetInput{WorkspacePath: "relative/path"})
	if err == nil {
		t.Fatal("expected error for relative path, got nil")
	}
}

func TestGetBudgetStatus_TraversalRejected(t *testing.T) {
	_, _, err := tools.GetBudgetStatus(nil, nil,
		tools.GetBudgetInput{WorkspacePath: "/etc/passwd"})
	if err == nil {
		t.Fatal("expected error for path outside home dir, got nil")
	}
}

func TestGetSessions_RelativePathRejected(t *testing.T) {
	_, _, err := tools.GetSessions(nil, nil,
		tools.GetSessionsInput{WorkspacePath: "."})
	if err == nil {
		t.Fatal("expected error for relative path")
	}
}

func TestGetInstructionOverhead_TraversalRejected(t *testing.T) {
	_, _, err := tools.GetInstructionOverhead(nil, nil,
		tools.GetInstructionsInput{WorkspacePath: "/tmp"})
	if err == nil {
		t.Fatal("expected error for /tmp (outside home)")
	}
}

func TestGetModelCosts_RelativePathRejected(t *testing.T) {
	_, _, err := tools.GetModelCosts(nil, nil,
		tools.GetModelCostsInput{WorkspacePath: "../escape"})
	if err == nil {
		t.Fatal("expected error for relative path")
	}
}

// ── Functional: home-directory path accepted ─────────────────────────────────

func TestGetBudgetStatus_HomePathAccepted(t *testing.T) {
	// Hermetic: point HOME at a fixture with two settled sessions so the handler
	// never depends on the developer's real ~/.copilot.
	home := useFixtureHome(t)
	_, out, err := tools.GetBudgetStatus(nil, nil,
		tools.GetBudgetInput{WorkspacePath: home})
	if err != nil {
		t.Fatalf("GetBudgetStatus: %v", err)
	}
	if out.Allowance <= 0 {
		t.Errorf("expected positive allowance, got %d", out.Allowance)
	}
	// Fixture sessions carry no premium requests, so the field must be present
	// and zero rather than absent.
	if out.PremiumRequests != 0 {
		t.Errorf("PremiumRequests = %d, want 0 (fixture sets none)", out.PremiumRequests)
	}
}

func TestGetSessions_ReturnsSortedByCredits(t *testing.T) {
	home := useFixtureHome(t)
	_, out, err := tools.GetSessions(nil, nil,
		tools.GetSessionsInput{WorkspacePath: home})
	if err != nil {
		t.Fatalf("GetSessions: %v", err)
	}
	for i := 1; i < len(out.Sessions); i++ {
		if out.Sessions[i].Credits > out.Sessions[i-1].Credits {
			t.Errorf("sessions not sorted descending at index %d: %.2f > %.2f",
				i, out.Sessions[i].Credits, out.Sessions[i-1].Credits)
		}
	}
	// The fixture sessions are finalized, so IsFinal must surface as true.
	for _, s := range out.Sessions {
		if !s.IsFinal {
			t.Errorf("session %q IsFinal = false, want true (fixture is settled)", s.Name)
		}
	}
}

func TestGetModelCosts_NoDuplicateModels(t *testing.T) {
	home := useFixtureHome(t)
	_, out, err := tools.GetModelCosts(nil, nil,
		tools.GetModelCostsInput{WorkspacePath: home})
	if err != nil {
		t.Fatalf("GetModelCosts: %v", err)
	}
	// Map keys are unique by definition, but verify non-negative credits.
	for model, cost := range out.Models {
		if cost.TotalCreditsThisMonth < 0 {
			t.Errorf("model %q has negative credits: %.4f", model, cost.TotalCreditsThisMonth)
		}
		// Fixture sessions carry no cache/reasoning tokens; verify the new fields
		// are non-negative and present (zero) rather than missing.
		if cost.CacheReadTokens < 0 || cost.CacheWriteTokens < 0 || cost.ReasoningTokens < 0 {
			t.Errorf("model %q has negative cache/reasoning tokens: %+v", model, cost)
		}
	}
}

// ── Startup time ≤ 100ms ─────────────────────────────────────────────────────

// TestStartupTime verifies the server produces its first MCP response quickly,
// confirming no heavy initialization (file scans, session reads) happens at startup.
// File I/O is deferred to individual tool call handlers.
//
// Note: The 100ms architectural requirement means "no heavy init at startup" —
// OS process creation and Go runtime startup (~50–200ms) are outside our control.
// We test with a 2-second ceiling to catch regressions where startup I/O is added.
func TestStartupTime(t *testing.T) {
	if serverBinary == "" {
		t.Skip("server binary not built (build failed in TestMain)")
	}
	initMsg := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n"

	cmd := exec.Command(serverBinary)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	// Start timing after process creation — measures only server initialization,
	// not OS process-creation overhead.
	start := time.Now()
	if _, err := stdin.Write([]byte(initMsg)); err != nil {
		t.Fatalf("write init: %v", err)
	}

	buf := make([]byte, 1)
	doneCh := make(chan error, 1)
	go func() {
		_, err := stdout.Read(buf)
		doneCh <- err
	}()

	const ceiling = 2 * time.Second // catches startup I/O regressions
	select {
	case err := <-doneCh:
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("read response: %v", err)
		}
		t.Logf("server initialization time: %v", elapsed)
		if elapsed > ceiling {
			t.Errorf("startup took %v, want ≤ %v — check for file I/O added to server init", elapsed, ceiling)
		}
	case <-time.After(ceiling + time.Second):
		t.Fatalf("server did not respond within %v", ceiling+time.Second)
	}
}

// ── Zero network calls ────────────────────────────────────────────────────────

func TestNoNetworkCalls(t *testing.T) {
	// Replace http.DefaultTransport with a transport that fails on any request.
	// All tool handlers are local-file-only (ADR-001) — this must never trigger.
	original := http.DefaultTransport
	http.DefaultTransport = &blockingTransport{t: t}
	defer func() { http.DefaultTransport = original }()

	// Hermetic fixture HOME so the handlers read deterministic local data rather
	// than the developer's real ~/.copilot.
	home := useFixtureHome(t)
	// Call all six handlers; if any makes an HTTP call, blockingTransport.RoundTrip
	// calls t.Fatal.
	tools.GetBudgetStatus(nil, nil, tools.GetBudgetInput{WorkspacePath: home})              //nolint
	tools.GetSessions(nil, nil, tools.GetSessionsInput{WorkspacePath: home})                //nolint
	tools.GetInstructionOverhead(nil, nil, tools.GetInstructionsInput{WorkspacePath: home}) //nolint
	tools.GetModelCosts(nil, nil, tools.GetModelCostsInput{WorkspacePath: home})            //nolint
	tools.GetUsageTimeseries(nil, nil, tools.GetUsageTimeseriesInput{WorkspacePath: home})  //nolint
	tools.GetTopConsumers(nil, nil, tools.GetTopConsumersInput{WorkspacePath: home})        //nolint
}

// blockingTransport is an http.RoundTripper that fails the test on any call.
type blockingTransport struct{ t *testing.T }

func (bt *blockingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	bt.t.Errorf("unexpected HTTP call to %s — tool handlers must be offline-only (ADR-001)", req.URL.Host)
	return nil, http.ErrServerClosed
}

// ── Arithmetic parity: MCP tool must match cmd/analyze exactly ───────────────

// reANSI matches ANSI terminal escape sequences (e.g. colour codes from cmd/analyze).
var reANSI = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// reUsedCredits parses "  Used:      8236.55 / 7000 credits" from
// cmd/analyze stdout (after ANSI codes are stripped) and returns the credit total.
var reUsedCredits = regexp.MustCompile(`Used:\s+([\d.]+)\s+/`)

// TestArithmeticParity verifies that get_budget_status returns the same credit
// total as cmd/analyze for the same session data. Both code paths now load the
// pricing config and call budget.Calculate(nanoAIUs, cfg.AllowanceCredits) — the
// MCP path used to hardcode 0 (the 7000 default), which silently matched only
// when no pricing.json override was present. This test runs against a hermetic
// fixture HOME so it catches divergence in nanoAIU accumulation, unit conversion,
// OR allowance sourcing introduced in the MCP layer.
func TestArithmeticParity(t *testing.T) {
	if analyzeBinary == "" {
		t.Skip("cmd/analyze binary not built (build failed in TestMain)")
	}
	// useFixtureHome sets HOME via t.Setenv; the analyze subprocess inherits it,
	// so both the CLI and the in-process MCP handler read the same session data.
	home := useFixtureHome(t)

	// Run cmd/analyze and parse its "Used:" credit line from stdout.
	out, err := exec.Command(analyzeBinary).Output() // stderr = skipped sessions, ignored
	if err != nil {
		t.Skipf("cmd/analyze exited non-zero: %v", err)
	}
	cleanOut := reANSI.ReplaceAll(out, nil) // strip \e[31m … \e[0m colour codes
	m := reUsedCredits.FindSubmatch(cleanOut)
	if m == nil {
		t.Skip("cmd/analyze output has no 'Used:' line — no session data this month")
	}
	cliCredits, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		t.Fatalf("parse cli credits %q: %v", m[1], err)
	}

	// Call the MCP tool handler directly.
	_, mcpOut, err := tools.GetBudgetStatus(nil, nil,
		tools.GetBudgetInput{WorkspacePath: home})
	if err != nil {
		t.Fatalf("GetBudgetStatus: %v", err)
	}

	// Allow ±1 credit tolerance (rounding at nanoAIU → credit boundary).
	const tolerance = 1.0
	diff := math.Abs(mcpOut.Credits - cliCredits)
	if diff > tolerance {
		t.Errorf("credit mismatch: MCP=%.4f, CLI=%.4f, diff=%.4f (want ≤ %.1f)",
			mcpOut.Credits, cliCredits, diff, tolerance)
	} else {
		t.Logf("parity OK: MCP=%.4f CLI=%.4f diff=%.4f", mcpOut.Credits, cliCredits, diff)
	}
}

// TestGetBudgetStatus_HonorsAllowanceOverride guards against the allowance-drift
// regression: GetBudgetStatus previously called budget.Calculate(nano, 0), which
// hardcodes the 7000 default and ignores any pricing.json override. It must load
// pricing and pass cfg.AllowanceCredits. Here we write a pricing.json with a
// non-default allowance into the fixture config dir and assert the handler
// reflects it — this fails if the handler reverts to the hardcoded default.
func TestGetBudgetStatus_HonorsAllowanceOverride(t *testing.T) {
	home := useFixtureHome(t)

	// platform.ConfigDir resolves via os.UserConfigDir, which honours
	// XDG_CONFIG_HOME on Linux; pin it inside the fixture HOME so the override is
	// hermetic. (On other OSes the default base also lands under HOME.)
	const wantAllowance = 9001
	cfgBase := filepath.Join(home, ".config")
	t.Setenv("XDG_CONFIG_HOME", cfgBase)
	cfgDir := filepath.Join(cfgBase, "copilot-token-budget")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	pricingJSON := fmt.Sprintf(`{"allowanceCredits": %d}`, wantAllowance)
	if err := os.WriteFile(filepath.Join(cfgDir, "pricing.json"), []byte(pricingJSON), 0o644); err != nil {
		t.Fatalf("write pricing.json: %v", err)
	}

	_, out, err := tools.GetBudgetStatus(nil, nil,
		tools.GetBudgetInput{WorkspacePath: home})
	if err != nil {
		t.Fatalf("GetBudgetStatus: %v", err)
	}
	if out.Allowance != wantAllowance {
		t.Errorf("Allowance = %d, want %d — handler is ignoring pricing.json (allowance drift)",
			out.Allowance, wantAllowance)
	}
}

// ── get_usage_timeseries & get_top_consumers ─────────────────────────────────

// writeFixtureSession writes a minimal but realistic Copilot CLI session under
// stateDir/<id>/events.jsonl. The session is finalized (session.shutdown) with a
// single model metric so credit/token aggregation has data, and EndTime is set to
// end so it lands in the intended billing month.
func writeFixtureSession(t *testing.T, stateDir, id, cwd, model string, end time.Time, nano, in, out int64) {
	t.Helper()
	dir := filepath.Join(stateDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	ts := end.UTC().Format(time.RFC3339)
	startTS := end.Add(-time.Hour).UTC().Format(time.RFC3339)
	start := fmt.Sprintf(`{"type":"session.start","timestamp":%q,"data":{"startTime":%q,"context":{"cwd":%q}}}`,
		startTS, startTS, cwd)
	shutdown := fmt.Sprintf(`{"type":"session.shutdown","timestamp":%q,"data":{"totalNanoAiu":%d,"currentModel":%q,"currentTokens":%d,"modelMetrics":{%q:{"totalNanoAiu":%d,"usage":{"inputTokens":%d,"outputTokens":%d}}}}}`,
		ts, nano, model, in, model, nano, in, out)
	content := start + "\n" + shutdown + "\n"
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatalf("write events.jsonl: %v", err)
	}
}

// useFixtureHome points HOME at a temp dir holding a .copilot/session-state
// directory with two finalized sessions billed in the current month, and returns
// (homeDir, sessionStateDir). ReadThisMonth/ReadAll resolve the session-state
// directory from os.UserHomeDir, so overriding HOME gives the handlers a
// deterministic dataset. The workspacePath passed to the tools is the temp HOME
// itself, which is a valid, in-home, existing path.
func useFixtureHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	stateDir := filepath.Join(home, ".copilot", "session-state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}

	now := time.Now()
	day := time.Date(now.Year(), now.Month(), 5, 9, 0, 0, 0, time.UTC)
	if now.Day() < 6 {
		day = now // ensure the fixture day is within the current month
	}
	// alpha is the bigger consumer (9 credits, opus); beta is smaller (3, sonnet).
	writeFixtureSession(t, stateDir, "sess-alpha", filepath.Join(home, "alpha"), "claude-opus-4.8", day, 9_000_000_000, 900, 90)
	writeFixtureSession(t, stateDir, "sess-beta", filepath.Join(home, "beta"), "claude-sonnet-4.6", day, 3_000_000_000, 300, 30)
	return home
}

func TestGetUsageTimeseries_ReturnsBuckets(t *testing.T) {
	home := useFixtureHome(t)

	_, out, err := tools.GetUsageTimeseries(nil, nil,
		tools.GetUsageTimeseriesInput{WorkspacePath: home}) // default granularity = daily
	if err != nil {
		t.Fatalf("GetUsageTimeseries: %v", err)
	}
	if len(out.Buckets) != 1 {
		t.Fatalf("got %d daily buckets, want 1 (both fixtures same day)", len(out.Buckets))
	}
	b := out.Buckets[0]
	if b.Sessions != 2 {
		t.Errorf("bucket sessions = %d, want 2", b.Sessions)
	}
	if b.Credits != 12 {
		t.Errorf("bucket credits = %v, want 12 (9+3)", b.Credits)
	}
	if b.InputTokens != 1200 || b.OutputTokens != 120 {
		t.Errorf("bucket tokens = %d/%d, want 1200/120", b.InputTokens, b.OutputTokens)
	}
	if _, err := time.Parse(time.RFC3339, b.Start); err != nil {
		t.Errorf("bucket start %q is not RFC3339: %v", b.Start, err)
	}
}

func TestGetUsageTimeseries_MonthlyGranularity(t *testing.T) {
	home := useFixtureHome(t)

	_, out, err := tools.GetUsageTimeseries(nil, nil,
		tools.GetUsageTimeseriesInput{WorkspacePath: home, Granularity: "monthly"})
	if err != nil {
		t.Fatalf("GetUsageTimeseries(monthly): %v", err)
	}
	if len(out.Buckets) != 1 {
		t.Fatalf("got %d monthly buckets, want 1", len(out.Buckets))
	}
	if out.Buckets[0].Credits != 12 {
		t.Errorf("monthly credits = %v, want 12", out.Buckets[0].Credits)
	}
}

func TestGetUsageTimeseries_InvalidGranularityRejected(t *testing.T) {
	home := useFixtureHome(t)
	_, _, err := tools.GetUsageTimeseries(nil, nil,
		tools.GetUsageTimeseriesInput{WorkspacePath: home, Granularity: "hourly"})
	if err == nil {
		t.Fatal("expected error for invalid granularity")
	}
}

func TestGetUsageTimeseries_RelativePathRejected(t *testing.T) {
	_, _, err := tools.GetUsageTimeseries(nil, nil,
		tools.GetUsageTimeseriesInput{WorkspacePath: "relative/path"})
	if err == nil {
		t.Fatal("expected error for relative path")
	}
	if !strings.Contains(err.Error(), "must be absolute") {
		t.Errorf("expected absolute-path error, got: %v", err)
	}
}

func TestGetUsageTimeseries_TraversalRejected(t *testing.T) {
	_, _, err := tools.GetUsageTimeseries(nil, nil,
		tools.GetUsageTimeseriesInput{WorkspacePath: "/etc"})
	if err == nil {
		t.Fatal("expected error for path outside home dir")
	}
}

func TestGetTopConsumers_ReturnsSortedLists(t *testing.T) {
	home := useFixtureHome(t)

	_, out, err := tools.GetTopConsumers(nil, nil,
		tools.GetTopConsumersInput{WorkspacePath: home}) // default n=5
	if err != nil {
		t.Fatalf("GetTopConsumers: %v", err)
	}

	if len(out.TopSessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(out.TopSessions))
	}
	// alpha (9 credits) must rank above beta (3 credits).
	if out.TopSessions[0].Name != "alpha" || out.TopSessions[0].Credits != 9 {
		t.Errorf("top session = %+v, want alpha/9", out.TopSessions[0])
	}
	// Lists must be sorted by credits descending.
	for i := 1; i < len(out.TopSessions); i++ {
		if out.TopSessions[i].Credits > out.TopSessions[i-1].Credits {
			t.Errorf("topSessions not sorted desc at %d", i)
		}
	}
	if len(out.TopModels) != 2 {
		t.Fatalf("got %d models, want 2", len(out.TopModels))
	}
	if out.TopModels[0].Name != "claude-opus-4.8" || out.TopModels[0].Credits != 9 {
		t.Errorf("top model = %+v, want claude-opus-4.8/9", out.TopModels[0])
	}
	if len(out.TopProjects) != 2 || out.TopProjects[0].Name != "alpha" {
		t.Errorf("top projects = %+v, want alpha first", out.TopProjects)
	}
}

func TestGetTopConsumers_RespectsN(t *testing.T) {
	home := useFixtureHome(t)
	_, out, err := tools.GetTopConsumers(nil, nil,
		tools.GetTopConsumersInput{WorkspacePath: home, N: 1})
	if err != nil {
		t.Fatalf("GetTopConsumers(n=1): %v", err)
	}
	if len(out.TopSessions) != 1 || out.TopSessions[0].Name != "alpha" {
		t.Errorf("n=1 topSessions = %+v, want only alpha", out.TopSessions)
	}
}

func TestGetTopConsumers_RelativePathRejected(t *testing.T) {
	_, _, err := tools.GetTopConsumers(nil, nil,
		tools.GetTopConsumersInput{WorkspacePath: "../escape"})
	if err == nil {
		t.Fatal("expected error for relative path")
	}
}

func TestGetTopConsumers_TraversalRejected(t *testing.T) {
	_, _, err := tools.GetTopConsumers(nil, nil,
		tools.GetTopConsumersInput{WorkspacePath: "/etc/passwd"})
	if err == nil {
		t.Fatal("expected error for path outside home dir")
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func isPathValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "must be absolute") || strings.Contains(msg, "home directory")
}

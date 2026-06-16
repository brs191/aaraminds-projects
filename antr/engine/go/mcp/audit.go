// audit.go — structured JSON audit logging for MCP tool calls.
//
// One audit line is written per analyze_risks or format_report invocation.
// Lines are emitted via log/slog at Info level so they flow through the same
// structured-JSON pipeline as all other server logs (os.Stderr).
//
// Audit line fields:
//
//	ts           RFC3339 timestamp
//	sub          subscription_id that was analysed
//	tool         tool name (analyze_risks | format_report)
//	findings     total finding count (0 when not available)
//	high_critical count of Critical + High findings
//	duration_ms  wall-clock milliseconds for the full tool call
package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
)

// callMetrics is a per-request sink the middleware installs in the context so a
// generic handler can report how many findings it produced. Without it the audit
// line reported zeros for findings/high_critical — the fields most useful in an
// evidence trail (external review F8).
type callMetrics struct {
	Findings int
	HighCrit int
}

type metricsKeyT struct{}

var metricsKey = metricsKeyT{}

func withCallMetrics(ctx context.Context, m *callMetrics) context.Context {
	return context.WithValue(ctx, metricsKey, m)
}

func callMetricsFrom(ctx context.Context) *callMetrics {
	m, _ := ctx.Value(metricsKey).(*callMetrics)
	return m
}

// recordFindings stores finding counts for the audit line. No-op when no sink is
// installed (e.g. a handler invoked directly in a unit test), so handlers can
// call it unconditionally.
func recordFindings(ctx context.Context, findings []analyze.Finding) {
	m := callMetricsFrom(ctx)
	if m == nil {
		return
	}
	m.Findings = len(findings)
	m.HighCrit = 0
	for _, f := range findings {
		if f.Severity == "Critical" || f.Severity == "High" {
			m.HighCrit++
		}
	}
}

// auditLine holds the data for a single structured audit log entry.
type auditLine struct {
	Tool       string
	Sub        string
	Findings   int
	HighCrit   int
	FetchMS    int64
	AnalyzeMS  int64
	RenderMS   int64
	DurationMS int64
}

// Auditor writes structured audit lines via slog.
type Auditor struct {
	logger *slog.Logger
}

// newAuditor creates an Auditor that writes through the supplied logger.
func newAuditor(logger *slog.Logger) *Auditor {
	return &Auditor{logger: logger}
}

// write emits one structured JSON audit line.
func (a *Auditor) write(line auditLine) {
	a.logger.Info("audit",
		"ts", time.Now().UTC().Format(time.RFC3339),
		"sub", line.Sub,
		"tool", line.Tool,
		"findings", line.Findings,
		"high_critical", line.HighCrit,
		"fetch_ms", line.FetchMS,
		"analyze_ms", line.AnalyzeMS,
		"render_ms", line.RenderMS,
		"duration_ms", line.DurationMS,
	)
}

// auditGenerateTopoLine holds audit data specific to a generate_topology call.
type auditGenerateTopoLine struct {
	Sub        string
	SpecHash   string
	GatePass   bool
	Iterations int
	PRURL      string
	Findings   int
	HighCrit   int
	DurationMS int64
}

// writeGenerateTopo emits a structured audit line for the generate_topology tool.
func (a *Auditor) writeGenerateTopo(line auditGenerateTopoLine) {
	a.logger.Info("audit",
		"ts", time.Now().UTC().Format(time.RFC3339),
		"tool", "generate_topology",
		"sub", line.Sub,
		"spec_hash", line.SpecHash,
		"gate_pass", line.GatePass,
		"iterations", line.Iterations,
		"pr_url", line.PRURL,
		"findings", line.Findings,
		"high_critical", line.HighCrit,
		"duration_ms", line.DurationMS,
	)
}

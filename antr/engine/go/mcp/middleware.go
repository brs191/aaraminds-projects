// middleware.go — request-level middleware for the MCP tool handlers.
//
// Applied via withMiddleware to every registered tool. The chain order is:
//  1. Panic recovery     — catches any unhandled panic; returns InternalError.
//  2. Input validation   — GUID check on subscription_id; prompt-injection defense.
//  3. Structured logging — one JSON line per call with tool name and duration_ms.
//
// Under stdio transport, stdout is the JSON-RPC protocol wire — ALL output goes
// to logger (os.Stderr) only.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// guidPattern is the regexp for a standard Azure subscription GUID.
const guidPattern = `(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`

// injectionChars is the set of characters that indicate a prompt-injection attempt.
// We reject any string parameter containing these characters.
const injectionChars = "$`\n{}"

// validateSubscriptionID returns an error if s is not a valid Azure GUID.
func validateSubscriptionID(s string) error {
	if !regexp.MustCompile(guidPattern).MatchString(s) {
		return fmt.Errorf("subscription_id must be a valid Azure GUID "+
			"(xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx), got: %q", s)
	}
	return nil
}

// validatePromptInjection returns an error if s contains any prompt-injection
// characters ($, backtick, newline, {, }).
func validatePromptInjection(s string) error {
	for _, ch := range injectionChars {
		if strings.ContainsRune(s, ch) {
			return fmt.Errorf("input contains disallowed character %q — "+
				"possible prompt-injection attempt", ch)
		}
	}
	return nil
}

// validateStringParams iterates over all string arguments in the request and
// rejects any that contain prompt-injection characters. Parameters named in
// jsonParams are STRUCTURED JSON (e.g. `delta`): they are destined for
// json.Unmarshal into Go structs, never an LLM context, so the prompt-injection
// char filter (which rejects `{` and `}`) does not apply to them — instead they
// must be well-formed JSON. Without this exemption the brace filter rejected
// EVERY valid delta, making simulate_change / forecast_cost unusable through the
// MCP boundary even though the handlers themselves parse JSON correctly
// (external review F1; the prior test missed it by calling the handler directly).
func validateStringParams(req mcpgo.CallToolRequest, jsonParams map[string]bool) error {
	args := req.GetArguments()
	for k, v := range args {
		s, ok := v.(string)
		if !ok {
			continue
		}
		if jsonParams[k] {
			if s != "" && !json.Valid([]byte(s)) {
				return fmt.Errorf("param %q must be valid JSON", k)
			}
			continue
		}
		if err := validatePromptInjection(s); err != nil {
			return fmt.Errorf("param %q: %w", k, err)
		}
	}
	return nil
}

// withMiddleware wraps a tool handler with the full middleware chain.
// wantAudit controls whether an audit line is written (true for analyze_risks
// and format_report; false for get_topology which does not run Analyze).
func withMiddleware(
	logger *slog.Logger,
	toolName string,
	wantAudit bool,
	h server.ToolHandlerFunc,
	auditor *Auditor,
	jsonParams ...string,
) server.ToolHandlerFunc {
	jsonSet := make(map[string]bool, len(jsonParams))
	for _, p := range jsonParams {
		jsonSet[p] = true
	}
	return func(ctx context.Context, req mcpgo.CallToolRequest) (result *mcpgo.CallToolResult, retErr error) {
		start := time.Now()

		// ── 1. Panic recovery ────────────────────────────────────────────────
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				logger.Error("panic in tool handler",
					"tool", toolName,
					"panic", fmt.Sprintf("%v", r),
					"stack", stack,
				)
				result = mcpgo.NewToolResultError(
					"internal error: unexpected panic in " + toolName)
				retErr = nil
			}
		}()

		// ── 2. Input validation ──────────────────────────────────────────────
		// subscription_id GUID check.
		subID := req.GetString("subscription_id", "")
		if subID != "" {
			if err := validateSubscriptionID(subID); err != nil {
				logger.Warn("invalid subscription_id", "tool", toolName, "err", err)
				return mcpgo.NewToolResultError(err.Error()), nil
			}
		}
		// Prompt-injection defense on all string parameters (JSON params exempt).
		if err := validateStringParams(req, jsonSet); err != nil {
			logger.Warn("prompt injection attempt blocked",
				"tool", toolName, "err", err)
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		// ── 3. Delegate to handler ───────────────────────────────────────────
		// Install a metrics sink so the handler can report finding counts back
		// for the audit line (external review F8).
		metrics := &callMetrics{}
		result, retErr = h(withCallMetrics(ctx, metrics), req)

		durationMS := time.Since(start).Milliseconds()

		// ── 4. Structured request log ────────────────────────────────────────
		if retErr != nil {
			logger.Error("tool call failed",
				"tool", toolName,
				"sub", subID,
				"duration_ms", durationMS,
				"err", retErr,
			)
		} else {
			isErr := result != nil && result.IsError
			logger.Info("tool call completed",
				"tool", toolName,
				"sub", subID,
				"duration_ms", durationMS,
				"is_error", isErr,
			)
		}

		// ── 5. Audit (analyze_risks, format_report only) ─────────────────────
		if wantAudit && auditor != nil {
			auditor.write(auditLine{
				Tool:       toolName,
				Sub:        subID,
				Findings:   metrics.Findings,
				HighCrit:   metrics.HighCrit,
				DurationMS: durationMS,
			})
		}

		return result, retErr
	}
}

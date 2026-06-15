// tools.go — MCP tool handlers for the Azure Network Topology Reviewer.
// Each handler is a closure that captures a TopologyFetcher for testability.
// No handler touches Azure credentials directly; that is the server's concern.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aaraminds/azure-nettopo-engine/generator"
	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
	"github.com/aaraminds/azure-nettopo-engine/renderer"
)

// TopologyFetcher abstracts adapter.FetchFixture for testability.
type TopologyFetcher interface {
	FetchFixture(ctx context.Context, subscriptionID string) (*graph.Fixture, error)
}

// ── Severity sort helpers ─────────────────────────────────────────────────────

// severityRankOf returns a numeric sort rank for a severity string.
// Lower rank = higher priority. Unknown severities sort last.
func severityRankOf(sev string) int {
	switch sev {
	case "Critical":
		return 0
	case "High":
		return 1
	case "Medium":
		return 2
	case "Informational":
		return 3
	case "Low":
		return 4
	default:
		return 99
	}
}

// severityKnown reports whether sev is a recognised filter value.
func severityKnown(sev string) bool {
	switch sev {
	case "Critical", "High", "Medium", "Low", "Informational":
		return true
	}
	return false
}

// sortFindings sorts findings by severity rank then by resource name.
func sortFindings(fs []analyze.Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		ri := severityRankOf(fs[i].Severity)
		rj := severityRankOf(fs[j].Severity)
		if ri != rj {
			return ri < rj
		}
		return fs[i].Resource < fs[j].Resource
	})
}

// ── Tool 1: get_topology ──────────────────────────────────────────────────────

// getTopologyHandler returns a handler that fetches and serialises the topology.
func getTopologyHandler(fetcher TopologyFetcher) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		subID, err := req.RequireString("subscription_id")
		if err != nil {
			return fmtErr("subscription_id is required"), nil
		}
		if verr := validateSubscriptionID(subID); verr != nil {
			return fmtErr("%s", verr), nil
		}

		fixture, err := fetcher.FetchFixture(ctx, subID)
		if err != nil {
			return fmtErr("fetch topology: %v", err), nil
		}

		b, err := json.Marshal(fixture)
		if err != nil {
			return fmtErr("marshal fixture: %v", err), nil
		}
		return mcpgo.NewToolResultText(string(b)), nil
	}
}

// ── Tool 2: analyze_risks ─────────────────────────────────────────────────────

// analyzeRisksResponse is the JSON envelope returned by analyze_risks.
type analyzeRisksResponse struct {
	Subscription string           `json:"subscription"`
	Findings     []analyze.Finding `json:"findings"`
	Summary      struct {
		Critical      int `json:"critical"`
		High          int `json:"high"`
		Medium        int `json:"medium"`
		Informational int `json:"informational"`
	} `json:"summary"`
}

// analyzeRisksHandler returns a handler that fetches topology, runs analysis,
// applies an optional severity filter, and returns the result as JSON.
func analyzeRisksHandler(fetcher TopologyFetcher) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		subID, err := req.RequireString("subscription_id")
		if err != nil {
			return fmtErr("subscription_id is required"), nil
		}
		if verr := validateSubscriptionID(subID); verr != nil {
			return fmtErr("%s", verr), nil
		}

		severityFilter := strings.TrimSpace(req.GetString("severity_filter", ""))
		if severityFilter != "" {
			if !severityKnown(severityFilter) {
				return fmtErr("severity_filter must be one of: Critical, High, Medium, Low, Informational"), nil
			}
		}

		fixture, err := fetcher.FetchFixture(ctx, subID)
		if err != nil {
			return fmtErr("fetch topology: %v", err), nil
		}

		findings := analyze.Analyze(fixture)

		// Optional severity filter.
		if severityFilter != "" {
			filtered := findings[:0]
			for _, f := range findings {
				if strings.EqualFold(f.Severity, severityFilter) {
					filtered = append(filtered, f)
				}
			}
			findings = filtered
		}

		sortFindings(findings)

		var resp analyzeRisksResponse
		resp.Subscription = subID
		resp.Findings = findings
		for _, f := range findings {
			switch f.Severity {
			case "Critical":
				resp.Summary.Critical++
			case "High":
				resp.Summary.High++
			case "Medium":
				resp.Summary.Medium++
			case "Informational":
				resp.Summary.Informational++
			}
		}

		b, err := json.Marshal(resp)
		if err != nil {
			return fmtErr("marshal response: %v", err), nil
		}
		return mcpgo.NewToolResultText(string(b)), nil
	}
}

// ── Tool 3: format_report ─────────────────────────────────────────────────────

// formatReportHandler returns a handler that fetches topology, runs analysis,
// and renders either a Markdown report or Draw.io XML diagram.
func formatReportHandler(fetcher TopologyFetcher) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		subID, err := req.RequireString("subscription_id")
		if err != nil {
			return fmtErr("subscription_id is required"), nil
		}
		if verr := validateSubscriptionID(subID); verr != nil {
			return fmtErr("%s", verr), nil
		}

		format, ferr := req.RequireString("format")
		if ferr != nil {
			return fmtErr("format is required"), nil
		}
		format = strings.ToLower(strings.TrimSpace(format))
		if format != "markdown" && format != "drawio" {
			return fmtErr("format must be 'markdown' or 'drawio'"), nil
		}

		fixture, err := fetcher.FetchFixture(ctx, subID)
		if err != nil {
			return fmtErr("fetch topology: %v", err), nil
		}

		findings := analyze.Analyze(fixture)
		sortFindings(findings)

		switch format {
		case "markdown":
			md := renderer.ToMarkdown(subID, findings)
			return mcpgo.NewToolResultText(md), nil

		case "drawio":
			xml := renderer.ToDrawIO(fixture, findings)
			// Metadata hint for clients. No wall-clock here: the drawio artifact
			// must be byte-reproducible for the same fixture (adversarial review
			// HIGH-1); a run timestamp belongs outside the deterministic artifact.
			meta := map[string]string{
				"content_type": "application/xml",
			}
			metaJSON, _ := json.Marshal(meta)
			combined := fmt.Sprintf("<!-- meta: %s -->\n%s", string(metaJSON), xml)
			return mcpgo.NewToolResultText(combined), nil

		default:
			// Unreachable — validated above.
			return fmtErr("unsupported format: %s", format), nil
		}
	}
}

// ── Tool 4: generate_topology ─────────────────────────────────────────────────

// LLMSpecProvider abstracts TopologySpec generation for testability.
// In Phase 3: always returns the deterministic §1.5 stub spec.
// In Phase 3 completion: a real Python intent.py service client is wired in.
type LLMSpecProvider interface {
	GenerateSpec(
		ctx context.Context,
		intent string,
		maxIterations int,
		failingFindings []analyze.Finding,
	) (generator.TopologySpec, error)
}

// stubSpecProvider returns the deterministic §1.5 hub-spoke spec regardless of intent.
// The spec is crafted to pass ValidateBeforeEmit:
//   - firewallEnabled=true with AzureFirewallSubnet in the hub VNet
//   - all spoke subnets have routeToFirewall=true → no synthetic PIPs → not internet-reachable
//   - no allow-https-from-internet on any sensitive subnet
type stubSpecProvider struct{}

// GenerateSpec returns the hardcoded AT&T hub-spoke reference topology (§1.5).
func (s *stubSpecProvider) GenerateSpec(
	_ context.Context,
	_ string,
	_ int,
	_ []analyze.Finding,
) (generator.TopologySpec, error) {
	return generator.TopologySpec{
		SpecVersion:     "1.0",
		Description:     "AT&T hub-spoke topology with Azure Firewall for payment processing workloads",
		Region:          "eastus2",
		PeeringTopology: "hub-spoke",
		HubVNetName:     "vnet-hub",
		GatewayType:     "vpn",
		FirewallEnabled: true,
		AVNMEnabled:     false,
		TierLabels:      []string{"app", "data"},
		Tags: map[string]string{
			"env":        "prod",
			"owner":      "att-nettopo",
			"costcenter": "PAY-001",
			"appid":      "PAY-001",
		},
		VNets: []generator.VNetSpec{
			{
				Name:         "vnet-hub",
				AddressSpace: []string{"10.0.0.0/16"},
				IsHub:        true,
				Subnets: []generator.SubnetSpec{
					{
						Name:          "AzureFirewallSubnet",
						AddressPrefix: "10.0.1.0/26",
						TierLabel:     "firewall",
						NSGIntents:    []string{},
					},
					{
						Name:          "GatewaySubnet",
						AddressPrefix: "10.0.2.0/27",
						TierLabel:     "gateway",
						NSGIntents:    []string{},
					},
				},
			},
			{
				Name:         "vnet-spoke",
				AddressSpace: []string{"10.1.0.0/16"},
				Subnets: []generator.SubnetSpec{
					{
						// app subnet: sensitive, routes to firewall, no internet ingress.
						Name:            "snet-app",
						AddressPrefix:   "10.1.1.0/24",
						TierLabel:       "app",
						Sensitive:       true,
						RouteToFirewall: true,
						NSGIntents: []string{
							"allow-internal-vnet",
							"deny-internet-inbound",
							"deny-all-inbound-other",
						},
					},
					{
						// data subnet: sensitive, routes to firewall, app-tier access only.
						Name:            "snet-data",
						AddressPrefix:   "10.1.2.0/24",
						TierLabel:       "data",
						Sensitive:       true,
						RouteToFirewall: true,
						NSGIntents: []string{
							"allow-app-tier-only",
							"deny-internet-inbound",
							"deny-all-inbound-other",
						},
					},
				},
			},
		},
	}, nil
}

// GenerationResult is the JSON envelope returned by generate_topology.
type GenerationResult struct {
	Spec       generator.TopologySpec  `json:"spec"`
	Plan       *TerraformPlanSummary   `json:"plan,omitempty"`
	Findings   []analyze.Finding       `json:"findings"`
	PRURL      string                  `json:"prUrl"`
	Iterations int                     `json:"iterations"`
	GatePass   bool                    `json:"gatePass"`
	Error      string                  `json:"error,omitempty"`
}

// TerraformPlanSummary is a redacted view of TerraformPlan suitable for MCP responses.
type TerraformPlanSummary struct {
	SpecHash  string   `json:"specHash"`
	FileNames []string `json:"fileNames"` // sorted
}

// filterBlockingFindings returns only Critical and High findings.
func filterBlockingFindings(findings []analyze.Finding) []analyze.Finding {
	var out []analyze.Finding
	for _, f := range findings {
		if f.Severity == "Critical" || f.Severity == "High" {
			out = append(out, f)
		}
	}
	return out
}

// generateTopologyHandler returns a handler for the generate_topology tool.
func generateTopologyHandler(
	specProvider LLMSpecProvider,
	registry generator.ModuleRegistry,
	ghClient generator.GitHubClient,
	fetcher TopologyFetcher,
	auditor *Auditor,
) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		start := time.Now()

		// ── 1. Parse required inputs ─────────────────────────────────────────
		intent, err := req.RequireString("intent")
		if err != nil {
			return fmtErr("intent is required"), nil
		}
		subscriptionID, err := req.RequireString("subscription_id")
		if err != nil {
			return fmtErr("subscription_id is required"), nil
		}
		region, err := req.RequireString("region")
		if err != nil {
			return fmtErr("region is required"), nil
		}

		// ── 2. Validate intent ───────────────────────────────────────────────
		if len(intent) < 20 || len(intent) > 2000 {
			return fmtErr("intent must be 20–2000 characters"), nil
		}
		if err := validatePromptInjection(intent); err != nil {
			return fmtErr("intent contains invalid characters"), nil
		}

		// ── 3. Validate subscription_id ──────────────────────────────────────
		if verr := validateSubscriptionID(subscriptionID); verr != nil {
			return fmtErr("%s", verr), nil
		}

		// ── 4. Parse max_iterations (optional, default 2, clamp 1–3) ────────
		maxIter := 2
		args := req.GetArguments()
		if v, ok := args["max_iterations"]; ok {
			if f, ok := v.(float64); ok {
				maxIter = int(f)
			}
		}
		if maxIter < 1 {
			maxIter = 1
		}
		if maxIter > 3 {
			maxIter = 3
		}

		// ── 5. Fetch subscription baseline for AVNM context ─────────────────
		baseline := generator.ProjectionBaseline{}
		fixture, fetchErr := fetcher.FetchFixture(ctx, subscriptionID)
		if fetchErr != nil {
			// Non-blocking: log warning and continue with empty baseline.
			_ = fetchErr // intentional — we continue without AVNM context
		} else if fixture != nil {
			baseline = generator.ProjectionBaseline{
				AVNMSecurityAdminRules: fixture.AVNM.SecurityAdminRules,
			}
		}

		// ── 6. LLM + refinement loop ─────────────────────────────────────────
		var (
			spec            generator.TopologySpec
			plan            generator.TerraformPlan
			validation      generator.ValidationResult
			failingFindings []analyze.Finding
		)
		iterations := 0

		for i := 1; i <= maxIter; i++ {
			iterations = i

			spec, err = specProvider.GenerateSpec(ctx, intent, maxIter, failingFindings)
			if err != nil {
				return fmtErr("spec generation failed: %v", err), nil
			}
			// MCP region param takes precedence over LLM-generated region.
			spec.Region = region

			plan, err = generator.RenderTerraform(spec, registry, baseline)
			if err != nil {
				// Render error: feed the error back as a blocking finding on next iteration.
				failingFindings = []analyze.Finding{{
					Type:      "render-error",
					Severity:  "High",
					Resource:  "spec",
					Evidence:  err.Error(),
					Reachable: false,
				}}
				validation = generator.ValidationResult{
					Approved: false,
					Findings: failingFindings,
				}
				continue
			}

			validation = generator.ValidateBeforeEmit(plan)
			if validation.Approved {
				break
			}
			failingFindings = filterBlockingFindings(validation.Findings)
		}

		durationMS := time.Since(start).Milliseconds()

		// Count High/Critical findings.
		highCrit := 0
		for _, f := range validation.Findings {
			if f.Severity == "Critical" || f.Severity == "High" {
				highCrit++
			}
		}

		// ── 7. Build response ────────────────────────────────────────────────
		result := GenerationResult{
			Spec:       spec,
			Findings:   validation.Findings,
			Iterations: iterations,
			GatePass:   validation.Approved,
		}

		// Attach plan summary when we have a valid plan.
		if plan.SpecHash != "" {
			fileNames := make([]string, 0, len(plan.Files))
			for fname := range plan.Files {
				fileNames = append(fileNames, fname)
			}
			sort.Strings(fileNames)
			result.Plan = &TerraformPlanSummary{
				SpecHash:  plan.SpecHash,
				FileNames: fileNames,
			}
		}

		// ── 8. Gate-fail path ────────────────────────────────────────────────
		if !validation.Approved {
			if result.Findings == nil {
				result.Findings = []analyze.Finding{}
			}
			b, err := json.Marshal(result)
			if err != nil {
				return fmtErr("marshal gate-fail response: %v", err), nil
			}
			// Write specialised audit for gate-fail.
			if auditor != nil {
				auditor.writeGenerateTopo(auditGenerateTopoLine{
					Sub:        subscriptionID,
					SpecHash:   plan.SpecHash,
					GatePass:   false,
					Iterations: iterations,
					Findings:   len(validation.Findings),
					HighCrit:   highCrit,
					DurationMS: durationMS,
				})
			}
			return mcpgo.NewToolResultText(string(b)), nil
		}

		// ── 9. Gate-pass: create PR ──────────────────────────────────────────
		prURL, prErr := generator.CreatePR(ctx, plan, validation, intent, ghClient)
		if prErr != nil {
			if errors.Is(prErr, generator.ErrGateFailed) {
				// Should not reach here — validation.Approved is true — but guard anyway.
				result.GatePass = false
				result.Error = "internal: gate re-check failed"
			} else {
				result.Error = fmt.Sprintf("PR creation failed: %v", prErr)
			}
		}
		result.PRURL = prURL

		b, err := json.Marshal(result)
		if err != nil {
			return fmtErr("marshal response: %v", err), nil
		}

		// Write specialised audit for gate-pass.
		if auditor != nil {
			auditor.writeGenerateTopo(auditGenerateTopoLine{
				Sub:        subscriptionID,
				SpecHash:   plan.SpecHash,
				GatePass:   true,
				Iterations: iterations,
				PRURL:      prURL,
				Findings:   len(validation.Findings),
				HighCrit:   highCrit,
				DurationMS: durationMS,
			})
		}

		return mcpgo.NewToolResultText(string(b)), nil
	}
}

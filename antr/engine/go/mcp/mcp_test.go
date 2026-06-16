// mcp_test.go — unit tests for MCP tool handlers, middleware, and validation.
//
// Tests use a MockFetcher that avoids any live Azure calls.
// All tool handlers are invoked directly as regular Go functions — no MCP
// transport is started during tests.
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"

	"github.com/aaraminds/azure-nettopo-engine/generator"
	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// ── Mock adapter ──────────────────────────────────────────────────────────────

// mockFetcher implements TopologyFetcher for testing.
type mockFetcher struct {
	fixture *graph.Fixture
	err     error
}

func (m *mockFetcher) FetchFixture(_ context.Context, _ string) (*graph.Fixture, error) {
	return m.fixture, m.err
}

// ── Test fixture builder ──────────────────────────────────────────────────────

// minimalFixture returns a small but structurally complete graph.Fixture.
func minimalFixture() *graph.Fixture {
	return &graph.Fixture{
		Subscription: "12345678-1234-1234-1234-123456789012",
		ResourceGraph: graph.ResourceGraph{
			VirtualNetworks: []graph.VNet{
				{
					Name:         "vnet-hub",
					AddressSpace: []string{"10.0.0.0/16"},
					Subnets: []graph.Subnet{
						{
							Name:          "default",
							AddressPrefix: "10.0.0.0/24",
						},
					},
				},
				{
					Name:         "vnet-spoke",
					AddressSpace: []string{"10.1.0.0/16"},
					Subnets: []graph.Subnet{
						{
							Name:          "workload",
							AddressPrefix: "10.1.0.0/24",
						},
					},
					Peerings: []graph.Peering{
						{
							RemoteVnet:            "vnet-hub",
							State:                 "Connected",
							AllowForwardedTraffic: true,
						},
					},
				},
			},
			NetworkInterfaces: []graph.NIC{
				{
					Name:      "nic-vm-01",
					Subnet:    "vnet-hub/default",
					PrivateIP: "10.0.0.4",
				},
				{
					Name:      "nic-vm-02",
					Subnet:    "vnet-spoke/workload",
					PrivateIP: "10.1.0.4",
				},
			},
		},
		NetworkWatcher: graph.NetworkWatcher{
			EffectiveSecurityRules: map[string][]graph.SecRule{},
			EffectiveRoutes:        map[string][]graph.Route{},
		},
	}
}

// ── makeReq builds a CallToolRequest with string arguments. ──────────────────
func makeReq(args map[string]any) mcpgo.CallToolRequest {
	var req mcpgo.CallToolRequest
	req.Params.Arguments = args
	return req
}

// ── Validation tests ──────────────────────────────────────────────────────────

func TestGetTopologyInvalidSubID(t *testing.T) {
	if err := validateSubscriptionID("not-a-guid"); err == nil {
		t.Error("expected validation error for 'not-a-guid'")
	}
}

func TestGetTopologyValidSubID(t *testing.T) {
	if err := validateSubscriptionID("12345678-1234-1234-1234-123456789012"); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestGetTopologySubIDUpperCase(t *testing.T) {
	// GUID matching must be case-insensitive.
	if err := validateSubscriptionID("ABCDEF12-ABCD-ABCD-ABCD-ABCDEF123456"); err != nil {
		t.Errorf("unexpected validation error for uppercase GUID: %v", err)
	}
}

func TestValidatePromptInjectionDollar(t *testing.T) {
	if err := validatePromptInjection("${ malicious }"); err == nil {
		t.Error("expected injection error for '${ malicious }'")
	}
}

func TestValidatePromptInjectionBacktick(t *testing.T) {
	if err := validatePromptInjection("`cmd`"); err == nil {
		t.Error("expected injection error for backtick")
	}
}

func TestValidatePromptInjectionNewline(t *testing.T) {
	if err := validatePromptInjection("safe\ninjected"); err == nil {
		t.Error("expected injection error for embedded newline")
	}
}

func TestValidatePromptInjectionClean(t *testing.T) {
	if err := validatePromptInjection("12345678-1234-1234-1234-123456789012"); err != nil {
		t.Errorf("unexpected injection error for clean GUID: %v", err)
	}
}

// TestAnalyzeRisksPromptInjection ensures the middleware blocks injected input.
func TestAnalyzeRisksPromptInjection(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	handler := analyzeRisksHandler(mock)
	req := makeReq(map[string]any{
		"subscription_id": "${ malicious }",
		"severity_filter": "",
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	// The handler itself checks validateSubscriptionID, which will reject this.
	if !result.IsError {
		t.Error("expected error result for injected subscription_id")
	}
}

// ── get_topology tests ────────────────────────────────────────────────────────

func TestGetTopologyReturnsJSON(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	handler := getTopologyHandler(mock)
	req := makeReq(map[string]any{
		"subscription_id": "12345678-1234-1234-1234-123456789012",
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}
	text := result.Content[0].(mcpgo.TextContent).Text
	if !strings.Contains(text, `"subscription"`) {
		t.Errorf("expected JSON with subscription field; got: %q", text[:min(200, len(text))])
	}
}

func TestGetTopologyMissingSubID(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	handler := getTopologyHandler(mock)
	req := makeReq(map[string]any{}) // no subscription_id
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing subscription_id")
	}
}

// ── analyze_risks tests ───────────────────────────────────────────────────────

func TestAnalyzeRisksReturnsJSON(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	handler := analyzeRisksHandler(mock)
	req := makeReq(map[string]any{
		"subscription_id": "12345678-1234-1234-1234-123456789012",
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}
	text := result.Content[0].(mcpgo.TextContent).Text
	for _, want := range []string{`"subscription"`, `"findings"`, `"summary"`} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in response; got snippet: %q", want, text[:min(300, len(text))])
		}
	}
}

func TestAnalyzeRisksInvalidFilter(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	handler := analyzeRisksHandler(mock)
	req := makeReq(map[string]any{
		"subscription_id": "12345678-1234-1234-1234-123456789012",
		"severity_filter": "INVALID",
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid severity_filter")
	}
}

// ── format_report tests ───────────────────────────────────────────────────────

func TestFormatReportMarkdown(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	handler := formatReportHandler(mock)
	req := makeReq(map[string]any{
		"subscription_id": "12345678-1234-1234-1234-123456789012",
		"format":          "markdown",
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}
	text := result.Content[0].(mcpgo.TextContent).Text
	if !strings.Contains(text, "# Azure Network Topology Analysis") {
		t.Errorf("expected Markdown header; got snippet: %q", text[:min(200, len(text))])
	}
}

func TestFormatReportDrawIO(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	handler := formatReportHandler(mock)
	req := makeReq(map[string]any{
		"subscription_id": "12345678-1234-1234-1234-123456789012",
		"format":          "drawio",
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}
	text := result.Content[0].(mcpgo.TextContent).Text
	if !strings.Contains(text, "<mxfile") {
		t.Errorf("expected <mxfile in drawio output; got snippet: %q", text[:min(200, len(text))])
	}
	if !strings.Contains(text, "content_type") {
		t.Error("expected content_type metadata hint in drawio output")
	}
}

func TestFormatReportInvalidFormat(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	handler := formatReportHandler(mock)
	req := makeReq(map[string]any{
		"subscription_id": "12345678-1234-1234-1234-123456789012",
		"format":          "csv",
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid format 'csv'")
	}
}

// ── Middleware integration test ───────────────────────────────────────────────

func TestMiddlewareBlocksInjectionInFormatParam(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	auditor := newAuditor(devNullLogger())
	handler := withMiddleware(devNullLogger(), "format_report", false,
		formatReportHandler(mock), auditor)

	req := makeReq(map[string]any{
		"subscription_id": "12345678-1234-1234-1234-123456789012",
		"format":          "markdown\n`evil`",
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Error("expected middleware to block prompt-injection in format param")
	}
}

// devNullLogger returns a slog.Logger that discards all output.
func devNullLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// ── generate_topology tests ───────────────────────────────────────────────────

// testSpecProvider allows tests to inject any TopologySpec into the handler.
type testSpecProvider struct{ spec generator.TopologySpec }

func (p *testSpecProvider) GenerateSpec(
	_ context.Context, _ string, _ int, _ []analyze.Finding,
) (generator.TopologySpec, error) {
	return p.spec, nil
}

// TestGenerateTopology_StubMode_GatePass verifies that the stub spec provider
// produces a clean gate result and returns a non-empty PR URL.
func TestGenerateTopology_StubMode_GatePass(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	ghClient := &generator.StubGitHubClient{}
	registry := generator.LoadDefaultRegistry()
	auditor := newAuditor(devNullLogger())

	handler := generateTopologyHandler(
		&stubSpecProvider{}, registry, ghClient, mock, auditor,
	)

	req := makeReq(map[string]any{
		"intent":          "AT&T hub-spoke topology with firewall for payment processing workloads",
		"subscription_id": "12345678-1234-1234-1234-123456789012",
		"region":          "eastus2",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if result.IsError {
		text := result.Content[0].(mcpgo.TextContent).Text
		t.Fatalf("unexpected tool error: %s", text)
	}

	text := result.Content[0].(mcpgo.TextContent).Text
	var gr GenerationResult
	if err := json.Unmarshal([]byte(text), &gr); err != nil {
		t.Fatalf("response is not valid JSON: %v\nraw: %s", err, text)
	}
	if !gr.GatePass {
		t.Errorf("expected gatePass=true; findings=%+v", gr.Findings)
	}
	if gr.PRURL == "" {
		t.Error("expected non-empty prUrl on gate pass")
	}
	if !ghClient.Called {
		t.Error("expected StubGitHubClient.CreatePull to be called")
	}
}

// TestGenerateTopology_PromptInjection verifies that intent containing prompt-
// injection characters ($) is rejected before any processing occurs.
func TestGenerateTopology_PromptInjection(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	registry := generator.LoadDefaultRegistry()
	auditor := newAuditor(devNullLogger())

	handler := generateTopologyHandler(
		&stubSpecProvider{}, registry, &generator.StubGitHubClient{}, mock, auditor,
	)

	req := makeReq(map[string]any{
		"intent":          "Deploy $malicious topology injection attempt here",
		"subscription_id": "12345678-1234-1234-1234-123456789012",
		"region":          "eastus2",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Error("expected MCP error result for intent containing prompt-injection character '$'")
	}
}

// TestGenerateTopology_InvalidSubscriptionID verifies that a non-GUID
// subscription_id is rejected with a validation error.
func TestGenerateTopology_InvalidSubscriptionID(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	registry := generator.LoadDefaultRegistry()
	auditor := newAuditor(devNullLogger())

	handler := generateTopologyHandler(
		&stubSpecProvider{}, registry, &generator.StubGitHubClient{}, mock, auditor,
	)

	req := makeReq(map[string]any{
		"intent":          "AT&T hub-spoke topology for payment processing workloads",
		"subscription_id": "not-a-guid",
		"region":          "eastus2",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Error("expected MCP error result for invalid subscription_id 'not-a-guid'")
	}
}

// dangerousSpec returns a spec with an internet-exposed sensitive subnet that
// will trigger a Critical finding from ValidateBeforeEmit.
func dangerousSpec() generator.TopologySpec {
	return generator.TopologySpec{
		SpecVersion:     "1.0",
		Description:     "dangerous spec with internet-exposed sensitive subnet",
		Region:          "eastus2",
		PeeringTopology: "none",
		GatewayType:     "none",
		FirewallEnabled: false,
		TierLabels:      []string{"app"},
		Tags: map[string]string{
			"env": "test", "owner": "test", "costcenter": "test", "appid": "test",
		},
		VNets: []generator.VNetSpec{{
			Name:         "vnet-danger",
			AddressSpace: []string{"10.0.0.0/16"},
			Subnets: []generator.SubnetSpec{{
				// sensitive=true + allow-https-from-internet + no routeToFirewall
				// → synthetic PIP assigned + Internet route → Critical finding.
				Name:            "snet-exposed",
				AddressPrefix:   "10.0.1.0/24",
				TierLabel:       "app",
				Sensitive:       true,
				RouteToFirewall: false,
				NSGIntents:      []string{"allow-https-from-internet"},
			}},
		}},
	}
}

// TestGenerateTopology_GateFail verifies that a dangerous spec (internet-
// exposed sensitive subnet) is blocked by the gate and returns gatePass=false.
func TestGenerateTopology_GateFail(t *testing.T) {
	mock := &mockFetcher{fixture: minimalFixture()}
	registry := generator.LoadDefaultRegistry()
	ghClient := &generator.StubGitHubClient{}
	auditor := newAuditor(devNullLogger())

	provider := &testSpecProvider{spec: dangerousSpec()}
	handler := generateTopologyHandler(
		provider, registry, ghClient, mock, auditor,
	)

	req := makeReq(map[string]any{
		"intent":          "AT&T topology for payment processing workloads eastus2",
		"subscription_id": "12345678-1234-1234-1234-123456789012",
		"region":          "eastus2",
		"max_iterations":  float64(1), // single iteration so we fail fast
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if result.IsError {
		text := result.Content[0].(mcpgo.TextContent).Text
		t.Fatalf("unexpected MCP-level error (want gate-fail JSON): %s", text)
	}

	text := result.Content[0].(mcpgo.TextContent).Text
	var gr GenerationResult
	if err := json.Unmarshal([]byte(text), &gr); err != nil {
		t.Fatalf("response is not valid JSON: %v\nraw: %s", err, text)
	}
	if gr.GatePass {
		t.Error("expected gatePass=false for dangerous spec; gate should have blocked it")
	}
	if len(gr.Findings) == 0 {
		t.Error("expected non-empty findings on gate fail")
	}
	if gr.PRURL != "" {
		t.Errorf("expected empty prUrl on gate fail, got %q", gr.PRURL)
	}
	if ghClient.Called {
		t.Error("expected StubGitHubClient.CreatePull NOT to be called on gate fail")
	}
}

// ── Phase-2 tools: simulate_change + forecast_cost ───────────────────────────

func TestSimulateChange_ValidDeltaReturnsSecurityDelta(t *testing.T) {
	handler := simulateChangeHandler(&mockFetcher{fixture: minimalFixture()})
	req := makeReq(map[string]any{
		"subscription_id": "12345678-1234-1234-1234-123456789012",
		"delta":           `{"addSubnet":{"vnetName":"vnet-hub","name":"new-subnet","addressPrefix":"10.0.5.0/24"}}`,
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}
	text := result.Content[0].(mcpgo.TextContent).Text
	if !strings.Contains(text, `"securityDelta"`) {
		t.Errorf("expected securityDelta in response; got: %q", text[:min(200, len(text))])
	}
}

func TestSimulateChange_EmptyDeltaIsError(t *testing.T) {
	handler := simulateChangeHandler(&mockFetcher{fixture: minimalFixture()})
	result, _ := handler(context.Background(), makeReq(map[string]any{
		"subscription_id": "12345678-1234-1234-1234-123456789012",
	}))
	if !result.IsError {
		t.Fatal("expected error: a delta with exactly one operation is required")
	}
}

func TestSimulateChange_InvalidSubID(t *testing.T) {
	handler := simulateChangeHandler(&mockFetcher{fixture: minimalFixture()})
	result, _ := handler(context.Background(), makeReq(map[string]any{"subscription_id": "not-a-guid"}))
	if !result.IsError {
		t.Fatal("expected error result for invalid subscription id")
	}
}

func TestSimulateChange_BadDeltaJSON(t *testing.T) {
	handler := simulateChangeHandler(&mockFetcher{fixture: minimalFixture()})
	result, _ := handler(context.Background(), makeReq(map[string]any{
		"subscription_id": "12345678-1234-1234-1234-123456789012",
		"delta":           "{not valid json",
	}))
	if !result.IsError {
		t.Fatal("expected error result for malformed delta JSON")
	}
}

func TestForecastCost_NoDeltaReturnsForecast(t *testing.T) {
	handler := forecastCostHandler(&mockFetcher{fixture: minimalFixture()})
	req := makeReq(map[string]any{"subscription_id": "12345678-1234-1234-1234-123456789012"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}
	text := result.Content[0].(mcpgo.TextContent).Text
	if !strings.Contains(text, `"fixedDeltaUsd"`) || !strings.Contains(text, `"caveats"`) {
		t.Errorf("expected cost forecast with caveats; got: %q", text[:min(300, len(text))])
	}
}

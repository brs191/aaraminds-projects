package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/att/rif/graphstore"
	"github.com/att/rif/retriever"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type stubRetriever struct {
	search []retriever.SearchResult
	impact []retriever.ImpactResult
}

func (s stubRetriever) Search(_ context.Context, _ retriever.SearchRequest) ([]retriever.SearchResult, error) {
	return s.search, nil
}

func (s stubRetriever) Impact(_ context.Context, _ retriever.ImpactRequest) ([]retriever.ImpactResult, error) {
	return s.impact, nil
}

type stubGraph struct {
	nodes      map[string]graphstore.Node
	callers    []graphstore.Node
	dependents map[int][]graphstore.Node
}

func (s stubGraph) UpsertNode(context.Context, graphstore.Node) error { return nil }
func (s stubGraph) UpsertEdge(context.Context, graphstore.Edge) error { return nil }
func (s stubGraph) BulkLoad(context.Context, []graphstore.Node, []graphstore.Edge) error {
	return nil
}
func (s stubGraph) GetNode(_ context.Context, nodeID string) (*graphstore.Node, error) {
	node := s.nodes[nodeID]
	return &node, nil
}
func (s stubGraph) DirectCallers(context.Context, string) ([]graphstore.Node, error) {
	return s.callers, nil
}
func (s stubGraph) Dependents(_ context.Context, _ string, depth int) ([]graphstore.Node, error) {
	return s.dependents[depth], nil
}
func (s stubGraph) BlastRadius(context.Context, string, int) (*graphstore.BlastRadiusResult, error) {
	return &graphstore.BlastRadiusResult{}, nil
}
func (s stubGraph) Ping(context.Context) error { return nil }
func (s stubGraph) Close() error               { return nil }

func newTestApp() *App {
	return &App{
		graph: stubGraph{
			nodes: map[string]graphstore.Node{
				"node-1": {NodeID: "node-1", QualifiedName: "PaymentProcessor", SourceRef: "demo@sha:src/payment/PaymentProcessor.java:12", Confidence: "exact", Properties: map[string]any{"summary": "Handles payment execution."}},
			},
			callers: []graphstore.Node{{SourceRef: "demo@sha:src/api/OrderController.java:18", Confidence: "exact"}},
			dependents: map[int][]graphstore.Node{
				1: {{SourceRef: "demo@sha:src/payment/AmountValidator.java:34"}},
				3: {{SourceRef: "demo@sha:src/payment/AmountValidator.java:34"}, {SourceRef: "demo@sha:src/integration/FraudGateway.java:22"}},
			},
		},
		retr: stubRetriever{
			search: []retriever.SearchResult{{NodeID: "node-1", SourceRef: "demo@sha:src/payment/PaymentProcessor.java:12", Score: 0.91, Confidence: "exact", Signals: []string{"vector", "fts"}}},
			impact: []retriever.ImpactResult{{NodeID: "node-2", SourceRef: "demo@sha:src/payment/AmountValidator.java:34", Score: 0.8, Tier: "inferred-di", CompletionCaveat: "DI edges are static."}},
		},
		limiter:        newRepoRateLimiter(rateLimitRPS, rateLimitRPS),
		repoExistsFn:   func(context.Context, string) (bool, error) { return true, nil },
		nodeResolverFn: func(context.Context, string, string) (string, error) { return "node-1", nil },
		explainArchFn: func(context.Context, string, string) (map[string]any, error) {
			return map[string]any{"summary": "PaymentProcessor coordinates validation and fraud checks.", "key_dependencies": []map[string]any{{"source_ref": "demo@sha:src/payment/AmountValidator.java:34"}}}, nil
		},
	}
}

func decodeToolResult(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("expected content")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("unmarshal tool result: %v", err)
	}
	return payload
}

func TestSanitizeQuery(t *testing.T) {
	got := sanitizeQuery("  <tool>alpha</tool> <system>beta</system> </s>  ")
	if got != "alpha</tool> beta</system>" {
		t.Fatalf("unexpected sanitize result: %q", got)
	}
}

func TestRateLimiterBlocksEleventhRequest(t *testing.T) {
	limiter := newRepoRateLimiter(10, 10)
	base := time.Unix(0, 0)
	limiter.now = func() time.Time { return base }
	for i := 0; i < 10; i++ {
		if !limiter.Allow("repo") {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}
	if limiter.Allow("repo") {
		t.Fatal("11th immediate request should have been rejected")
	}
}

func TestSearchCodeTool(t *testing.T) {
	app := newTestApp()
	result, _, err := app.handleSearchCode(context.Background(), nil, SearchCodeInput{RepoID: fixtureRepoID, Query: "PaymentProcessor", TopK: 5})
	if err != nil {
		t.Fatalf("handleSearchCode: %v", err)
	}
	payload := decodeToolResult(t, result)
	rows := payload["results"].([]any)
	first := rows[0].(map[string]any)
	if first["source_ref"].(string) == "" {
		t.Fatal("source_ref missing")
	}
}

func TestFindCallersTool(t *testing.T) {
	app := newTestApp()
	result, _, err := app.handleFindCallers(context.Background(), nil, FindCallersInput{RepoID: fixtureRepoID, QualifiedName: "PaymentProcessor", Depth: 2})
	if err != nil {
		t.Fatalf("handleFindCallers: %v", err)
	}
	payload := decodeToolResult(t, result)
	rows := payload["results"].([]any)
	if len(rows) != 1 {
		t.Fatalf("expected one caller, got %d", len(rows))
	}
}

func TestImpactAnalysisTool(t *testing.T) {
	app := newTestApp()
	result, _, err := app.handleImpactAnalysis(context.Background(), nil, ImpactAnalysisInput{RepoID: fixtureRepoID, ChangedEntity: "AmountValidator", Depth: 3})
	if err != nil {
		t.Fatalf("handleImpactAnalysis: %v", err)
	}
	payload := decodeToolResult(t, result)
	if payload["completeness_caveat"].(string) == "" {
		t.Fatal("expected completeness caveat")
	}
}

func TestExplainArchitectureTool(t *testing.T) {
	app := newTestApp()
	result, _, err := app.handleExplainArchitecture(context.Background(), nil, ExplainArchitectureInput{RepoID: fixtureRepoID, Component: "PaymentProcessor"})
	if err != nil {
		t.Fatalf("handleExplainArchitecture: %v", err)
	}
	payload := decodeToolResult(t, result)
	if payload["summary"].(string) == "" {
		t.Fatal("expected summary")
	}
}

func TestDependencyAnalysisTool(t *testing.T) {
	app := newTestApp()
	result, _, err := app.handleDependencyAnalysis(context.Background(), nil, DependencyAnalysisInput{RepoID: fixtureRepoID, Entity: "PaymentProcessor", Depth: 3})
	if err != nil {
		t.Fatalf("handleDependencyAnalysis: %v", err)
	}
	payload := decodeToolResult(t, result)
	if len(payload["transitive_deps"].([]any)) == 0 {
		t.Fatal("expected dependency rows")
	}
}

func TestMCPHTTPIntegrationSearchCode(t *testing.T) {
	app := newTestApp()
	server := mcp.NewServer(&mcp.Implementation{Name: "rif-mcp-server", Version: "v0.1.0"}, nil)
	registerTools(server, app)

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_code",
		Arguments: map[string]any{"repo_id": fixtureRepoID, "query": "PaymentProcessor", "top_k": 3},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("expected tool content")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("unmarshal text payload: %v", err)
	}
	if len(payload["results"].([]any)) == 0 {
		t.Fatal("expected search results")
	}
}

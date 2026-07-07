package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/att/rif/graphstore"
	"github.com/att/rif/retriever"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultDepth = 3
	maxDepth     = 5
	rateLimitRPS = 10
)

var dangerousTokenRe = regexp.MustCompile(`(?i)<tool>|<system>|</s>`)

type Config struct {
	DatabaseURL     string
	EmbeddingURL    string
	AgentServiceURL string
	AuditLogPath    string
	FixtureMode     bool
}

type App struct {
	metaPool       *pgxpool.Pool
	graph          graphstore.GraphStore
	retr           retriever.Retriever
	limiter        *repoRateLimiter
	logger         *slog.Logger
	cfg            Config
	repoExistsFn   func(context.Context, string) (bool, error)
	nodeResolverFn func(context.Context, string, string) (string, error)
	explainArchFn  func(context.Context, string, string) (map[string]any, error)
}

type SearchCodeInput struct {
	RepoID string `json:"repo_id" jsonschema:"registered repository id"`
	Query  string `json:"query" jsonschema:"non-empty search query"`
	TopK   int    `json:"top_k" jsonschema:"max results to return"`
}

type FindCallersInput struct {
	RepoID        string `json:"repo_id" jsonschema:"registered repository id"`
	QualifiedName string `json:"qualified_name" jsonschema:"qualified method or class name"`
	Depth         int    `json:"depth" jsonschema:"call graph depth (1-5)"`
}

type ImpactAnalysisInput struct {
	RepoID        string `json:"repo_id" jsonschema:"registered repository id"`
	ChangedEntity string `json:"changed_entity" jsonschema:"changed entity name"`
	Depth         int    `json:"depth" jsonschema:"impact depth (1-5)"`
}

type ExplainArchitectureInput struct {
	RepoID    string `json:"repo_id" jsonschema:"registered repository id"`
	Component string `json:"component" jsonschema:"component to explain"`
}

type DependencyAnalysisInput struct {
	RepoID string `json:"repo_id" jsonschema:"registered repository id"`
	Entity string `json:"entity" jsonschema:"entity to analyze"`
	Depth  int    `json:"depth" jsonschema:"dependency depth (1-5)"`
}

type toolOutput struct {
	Results any `json:"results,omitempty"`
}

type SearchHitOutput struct {
	SourceRef  string  `json:"source_ref"`
	Snippet    string  `json:"snippet"`
	Score      float64 `json:"score"`
	Confidence string  `json:"confidence"`
}

type CallerOutput struct {
	CallerRef   string `json:"caller_ref"`
	CallSiteRef string `json:"call_site_ref"`
	Confidence  string `json:"confidence"`
}

type ImpactOutput struct {
	SourceRef          string `json:"source_ref"`
	Confidence         string `json:"confidence"`
	Tier               string `json:"tier"`
	CompletenessCaveat string `json:"completeness_caveat"`
}

type DependencyOutput struct {
	DirectDeps     []string `json:"direct_deps"`
	TransitiveDeps []string `json:"transitive_deps"`
	DepthCap       int      `json:"depth_cap"`
}

type ArchitectureOutput struct {
	Summary         string      `json:"summary"`
	KeyDependencies []SourceRef `json:"key_dependencies"`
}

type SourceRef struct {
	SourceRef string `json:"source_ref"`
}

func NewApp(ctx context.Context, cfg Config) (*App, error) {
	if cfg.FixtureMode {
		return newFixtureApp(cfg)
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	metaPool, err := newPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	graph, err := graphstore.NewAGEStore(ctx, cfg.DatabaseURL)
	if err != nil {
		metaPool.Close()
		return nil, err
	}
	backend := retriever.NewPGBackend(metaPool)
	emb := newHTTPEmbedder(cfg.EmbeddingURL)
	retrieverSvc := retriever.NewService(backend, graph, emb)
	app := &App{
		metaPool: metaPool,
		graph:    graph,
		retr:     retrieverSvc,
		limiter:  newRepoRateLimiter(rateLimitRPS, rateLimitRPS),
		logger:   slog.Default(),
		cfg:      cfg,
	}
	if err := app.ensureAuditSchema(ctx); err != nil {
		app.Close()
		return nil, err
	}
	return app, nil
}

func (a *App) Close() {
	if a.graph != nil {
		_ = a.graph.Close()
	}
	if a.metaPool != nil {
		a.metaPool.Close()
	}
}

func newPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "LOAD 'age'; SET search_path = ag_catalog, rif_meta, public;")
		return err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

func (a *App) ensureAuditSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS audit_log (
    id BIGSERIAL PRIMARY KEY,
    tool_name TEXT NOT NULL,
    repo_id TEXT NOT NULL,
    input_sha256 CHAR(64) NOT NULL,
    output_node_count INTEGER NOT NULL DEFAULT 0,
    latency_ms INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_log_repo_created_at ON audit_log (repo_id, created_at DESC);`
	_, err := a.metaPool.Exec(ctx, ddl)
	return err
}

func registerTools(server *mcp.Server, app *App) {
	mcp.AddTool(server, &mcp.Tool{Name: "search_code", Description: "Hybrid code search"}, app.handleSearchCode)
	mcp.AddTool(server, &mcp.Tool{Name: "find_callers", Description: "Find direct callers"}, app.handleFindCallers)
	mcp.AddTool(server, &mcp.Tool{Name: "impact_analysis", Description: "Rank impacted nodes"}, app.handleImpactAnalysis)
	mcp.AddTool(server, &mcp.Tool{Name: "explain_architecture", Description: "Summarise architecture"}, app.handleExplainArchitecture)
	mcp.AddTool(server, &mcp.Tool{Name: "dependency_analysis", Description: "Analyze dependencies"}, app.handleDependencyAnalysis)
}

func (a *App) handleSearchCode(ctx context.Context, _ *mcp.CallToolRequest, in SearchCodeInput) (*mcp.CallToolResult, any, error) {
	start := time.Now()
	query := sanitizeQuery(in.Query)
	repoID, err := a.validateRepoAndRateLimit(ctx, "search_code", in.RepoID, query)
	if err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(query) == "" {
		return nil, nil, errors.New("query must be non-empty")
	}
	topK := in.TopK
	if topK <= 0 {
		topK = 10
	}
	hits, err := a.retr.Search(ctx, retriever.SearchRequest{RepoID: repoID, Query: query, K: topK, GraphDepth: 3})
	if err != nil {
		return nil, nil, err
	}
	out := make([]SearchHitOutput, 0, len(hits))
	for _, hit := range hits {
		snippet := hit.SourceRef
		if node, nodeErr := a.graph.GetNode(ctx, hit.NodeID); nodeErr == nil {
			snippet = node.QualifiedName
			if summary, ok := node.Properties["summary"].(string); ok && strings.TrimSpace(summary) != "" {
				snippet = summary
			} else if simple, ok := node.Properties["simple_name"].(string); ok && strings.TrimSpace(simple) != "" {
				snippet = simple
			}
		}
		out = append(out, SearchHitOutput{
			SourceRef:  hit.SourceRef,
			Snippet:    snippet,
			Score:      hit.Score,
			Confidence: hit.Confidence,
		})
	}
	if err := a.audit(ctx, "search_code", repoID, map[string]any{"repo_id": repoID, "query": query, "top_k": topK}, len(out), start); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"results": out}), nil, nil
}

func (a *App) handleFindCallers(ctx context.Context, _ *mcp.CallToolRequest, in FindCallersInput) (*mcp.CallToolResult, any, error) {
	start := time.Now()
	qualifiedName := sanitizeQuery(in.QualifiedName)
	repoID, err := a.validateRepoAndRateLimit(ctx, "find_callers", in.RepoID, qualifiedName)
	if err != nil {
		return nil, nil, err
	}
	depth := clampDepth(in.Depth)
	nodeID, err := a.nodeIDByQualifiedName(ctx, repoID, qualifiedName)
	if err != nil {
		return nil, nil, err
	}
	callers, err := a.graph.DirectCallers(ctx, nodeID)
	if err != nil {
		return nil, nil, err
	}
	out := make([]CallerOutput, 0, len(callers))
	for _, caller := range callers {
		out = append(out, CallerOutput{CallerRef: caller.SourceRef, CallSiteRef: caller.SourceRef, Confidence: caller.Confidence})
	}
	if err := a.audit(ctx, "find_callers", repoID, map[string]any{"repo_id": repoID, "qualified_name": qualifiedName, "depth": depth}, len(out), start); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"results": out}), nil, nil
}

func (a *App) handleImpactAnalysis(ctx context.Context, _ *mcp.CallToolRequest, in ImpactAnalysisInput) (*mcp.CallToolResult, any, error) {
	start := time.Now()
	changedEntity := sanitizeQuery(in.ChangedEntity)
	repoID, err := a.validateRepoAndRateLimit(ctx, "impact_analysis", in.RepoID, changedEntity)
	if err != nil {
		return nil, nil, err
	}
	depth := clampDepth(in.Depth)
	nodeID, err := a.nodeIDByQualifiedName(ctx, repoID, changedEntity)
	if err != nil {
		return nil, nil, err
	}
	results, err := a.retr.Impact(ctx, retriever.ImpactRequest{RootNodeID: nodeID, Depth: depth, Limit: 50})
	if err != nil {
		return nil, nil, err
	}
	out := make([]ImpactOutput, 0, len(results))
	for _, item := range results {
		out = append(out, ImpactOutput{
			SourceRef:          item.SourceRef,
			Confidence:         item.Tier,
			Tier:               item.Tier,
			CompletenessCaveat: item.CompletionCaveat,
		})
	}
	caveat := "Graph reachability is bounded; runtime-only edges may be missed."
	if len(results) > 0 {
		caveat = results[0].CompletionCaveat
	}
	if err := a.audit(ctx, "impact_analysis", repoID, map[string]any{"repo_id": repoID, "changed_entity": changedEntity, "depth": depth}, len(out), start); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"impacted": out, "completeness_caveat": caveat}), nil, nil
}

func (a *App) handleExplainArchitecture(ctx context.Context, _ *mcp.CallToolRequest, in ExplainArchitectureInput) (*mcp.CallToolResult, any, error) {
	start := time.Now()
	component := sanitizeQuery(in.Component)
	repoID, err := a.validateRepoAndRateLimit(ctx, "explain_architecture", in.RepoID, component)
	if err != nil {
		return nil, nil, err
	}
	if a.explainArchFn != nil {
		out, err := a.explainArchFn(ctx, repoID, component)
		if err != nil {
			return nil, nil, err
		}
		if err := a.audit(ctx, "explain_architecture", repoID, map[string]any{"repo_id": repoID, "component": component}, 1, start); err != nil {
			return nil, nil, err
		}
		return textResult(out), nil, nil
	}
	if strings.TrimSpace(a.cfg.AgentServiceURL) == "" {
		return nil, nil, errors.New("AGENT_SERVICE_URL is required for explain_architecture")
	}
	payload := map[string]any{"repo_id": repoID, "component": component}
	resp, err := http.Post(a.cfg.AgentServiceURL, "application/json", strings.NewReader(mustJSON(payload)))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("agent service returned HTTP %d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, nil, err
	}
	if err := a.audit(ctx, "explain_architecture", repoID, payload, 1, start); err != nil {
		return nil, nil, err
	}
	return textResult(out), nil, nil
}

func (a *App) handleDependencyAnalysis(ctx context.Context, _ *mcp.CallToolRequest, in DependencyAnalysisInput) (*mcp.CallToolResult, any, error) {
	start := time.Now()
	entity := sanitizeQuery(in.Entity)
	repoID, err := a.validateRepoAndRateLimit(ctx, "dependency_analysis", in.RepoID, entity)
	if err != nil {
		return nil, nil, err
	}
	depth := clampDepth(in.Depth)
	nodeID, err := a.nodeIDByQualifiedName(ctx, repoID, entity)
	if err != nil {
		return nil, nil, err
	}
	direct, err := a.graph.Dependents(ctx, nodeID, 1)
	if err != nil {
		return nil, nil, err
	}
	transitive, err := a.graph.Dependents(ctx, nodeID, depth)
	if err != nil {
		return nil, nil, err
	}
	directDeps := make([]string, 0, len(direct))
	for _, n := range direct {
		directDeps = append(directDeps, n.SourceRef)
	}
	transitiveDeps := make([]string, 0, len(transitive))
	for _, n := range transitive {
		transitiveDeps = append(transitiveDeps, n.SourceRef)
	}
	sort.Strings(directDeps)
	sort.Strings(transitiveDeps)
	if err := a.audit(ctx, "dependency_analysis", repoID, map[string]any{"repo_id": repoID, "entity": entity, "depth": depth}, len(transitiveDeps), start); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"direct_deps": directDeps, "transitive_deps": transitiveDeps, "depth_cap": depth}), nil, nil
}

func (a *App) validateRepoAndRateLimit(ctx context.Context, tool, repoID, input string) (string, error) {
	repoID = strings.TrimSpace(repoID)
	if repoID == "" {
		return "", errors.New("repo_id is required")
	}
	ok, err := a.repoExists(ctx, repoID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("repo_id %q is not registered", repoID)
	}
	if !a.limiter.Allow(repoID) {
		return "", fmt.Errorf("rate limit exceeded for repo_id %s", repoID)
	}
	return repoID, nil
}

func (a *App) repoExists(ctx context.Context, repoID string) (bool, error) {
	if a.repoExistsFn != nil {
		return a.repoExistsFn(ctx, repoID)
	}
	var exists bool
	err := a.metaPool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM rif_meta.repositories WHERE repo_id = $1)`, repoID).Scan(&exists)
	return exists, err
}

func (a *App) nodeIDByQualifiedName(ctx context.Context, repoID, qualifiedName string) (string, error) {
	if a.nodeResolverFn != nil {
		return a.nodeResolverFn(ctx, repoID, qualifiedName)
	}
	const q = `
SELECT node_id FROM (
  SELECT node_id, qualified_name, qualified_name AS simple_name, repo_id FROM rif_meta.file_nodes
  UNION ALL SELECT node_id, qualified_name, simple_name, repo_id FROM rif_meta.method_nodes
  UNION ALL SELECT node_id, qualified_name, simple_name, repo_id FROM rif_meta.class_nodes
) nodes
WHERE repo_id = $1
  AND (
    qualified_name = $2
    OR simple_name = $2
    OR qualified_name ILIKE '%' || $2 || '%'
  )
ORDER BY
  CASE
    WHEN qualified_name = $2 THEN 0
    WHEN simple_name = $2 THEN 1
    ELSE 2
  END,
  length(qualified_name)
LIMIT 1;`
	var nodeID string
	err := a.metaPool.QueryRow(ctx, q, repoID, qualifiedName).Scan(&nodeID)
	if err != nil {
		return "", fmt.Errorf("resolve %q in repo %s: %w", qualifiedName, repoID, err)
	}
	return nodeID, nil
}

func (a *App) audit(ctx context.Context, toolName, repoID string, input any, outputCount int, start time.Time) error {
	if a.metaPool == nil {
		return nil
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(inputJSON)
	_, err = a.metaPool.Exec(ctx, `
INSERT INTO audit_log (tool_name, repo_id, input_sha256, output_node_count, latency_ms)
VALUES ($1, $2, $3, $4, $5)`,
		toolName, repoID, hex.EncodeToString(sum[:]), outputCount, time.Since(start).Milliseconds())
	return err
}

func sanitizeQuery(value string) string {
	return strings.TrimSpace(dangerousTokenRe.ReplaceAllString(value, ""))
}

func clampDepth(depth int) int {
	if depth <= 0 {
		return defaultDepth
	}
	if depth > maxDepth {
		return maxDepth
	}
	return depth
}

func textResult(payload any) *mcp.CallToolResult {
	body, _ := json.Marshal(payload)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(body)},
		},
	}
}

func mustJSON(payload any) string {
	b, _ := json.Marshal(payload)
	return string(b)
}

type httpEmbedder struct {
	url string
}

func newHTTPEmbedder(url string) *httpEmbedder {
	return &httpEmbedder{url: url}
}

func (e *httpEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody := []map[string]string{{"node_id": "query", "text": text}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url, strings.NewReader(mustJSON(reqBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding service returned HTTP %d", resp.StatusCode)
	}
	var payload []struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, errors.New("embedding service returned no embeddings")
	}
	return payload[0].Embedding, nil
}

type repoRateLimiter struct {
	rate  float64
	burst float64
	now   func() time.Time
	mu    chan struct{}
	state map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newRepoRateLimiter(rate, burst float64) *repoRateLimiter {
	return &repoRateLimiter{
		rate:  rate,
		burst: burst,
		now:   time.Now,
		mu:    make(chan struct{}, 1),
		state: make(map[string]*bucket),
	}
}

func (l *repoRateLimiter) Allow(repoID string) bool {
	l.mu <- struct{}{}
	defer func() { <-l.mu }()
	now := l.now()
	b, ok := l.state[repoID]
	if !ok {
		l.state[repoID] = &bucket{tokens: l.burst - 1, last: now}
		return true
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = minFloat(l.burst, b.tokens+elapsed*l.rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

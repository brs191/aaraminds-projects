package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/aaraminds/rif/graphstore"
	"github.com/aaraminds/rif/retriever"
)

const fixtureRepoID = "demo-repo"

type fixtureDoc struct {
	node    graphstore.Node
	vector  []float32
	lexical map[string]struct{}
}

type fixtureBackend struct {
	docs []fixtureDoc
}

type fixtureEmbedder struct{}

type fixtureGraph struct {
	nodes map[string]graphstore.Node
	edges []graphstore.Edge
}

func newFixtureApp(cfg Config) (*App, error) {
	graph, backend, nodeByName := buildFixtureGraph()
	retrieverSvc := retriever.NewService(backend, graph, fixtureEmbedder{})
	return &App{
		graph:   graph,
		retr:    retrieverSvc,
		limiter: newRepoRateLimiter(rateLimitRPS, rateLimitRPS),
		logger:  slog.Default(),
		cfg:     cfg,
		repoExistsFn: func(_ context.Context, repoID string) (bool, error) {
			return strings.TrimSpace(repoID) != "", nil
		},
		nodeResolverFn: func(_ context.Context, _ string, qualifiedName string) (string, error) {
			needle := strings.ToLower(strings.TrimSpace(qualifiedName))
			if nodeID, ok := nodeByName[needle]; ok {
				return nodeID, nil
			}
			for name, nodeID := range nodeByName {
				if strings.Contains(name, needle) {
					return nodeID, nil
				}
			}
			return "", fmt.Errorf("fixture node %q not found", qualifiedName)
		},
		explainArchFn: func(ctx context.Context, repoID, component string) (map[string]any, error) {
			search, err := retrieverSvc.Search(ctx, retriever.SearchRequest{RepoID: repoID, Query: component, K: 3, GraphDepth: 2})
			if err != nil {
				return nil, err
			}
			deps, err := graph.Dependents(ctx, nodeByName[strings.ToLower(component)], 2)
			if err != nil {
				return nil, err
			}
			refs := make([]map[string]any, 0, len(search)+len(deps))
			for _, hit := range search {
				refs = append(refs, map[string]any{"tool_name": "search_code", "result_excerpt": hit.SourceRef, "confidence": hit.Confidence})
			}
			for _, dep := range deps {
				refs = append(refs, map[string]any{"tool_name": "dependency_analysis", "result_excerpt": dep.SourceRef, "confidence": dep.Confidence})
			}
			return map[string]any{
				"summary":          fmt.Sprintf("%s coordinates validation, persistence, and external fraud checks.", component),
				"key_dependencies": refs,
			}, nil
		},
	}, nil
}

func buildFixtureGraph() (*fixtureGraph, *fixtureBackend, map[string]string) {
	sha := strings.Repeat("1", 40)
	node := func(fill string, qualifiedName, kind, sourceRef, confidence string, props map[string]any) graphstore.Node {
		return graphstore.Node{
			NodeID:         strings.Repeat(fill, 64),
			RepoID:         fixtureRepoID,
			QualifiedName:  qualifiedName,
			Kind:           kind,
			SourceRef:      sourceRef,
			Confidence:     confidence,
			PhasePopulated: 2,
			Origin:         "first_party",
			ProvenanceKind: "file",
			Properties:     props,
		}
	}
	payment := node("a", "PaymentProcessor", "CLASS", fixtureRepoID+"@"+sha+":src/payment/PaymentProcessor.java:12", "exact", map[string]any{"simple_name": "PaymentProcessor", "summary": "Processes a validated payment then calls fraud screening."})
	validator := node("b", "AmountValidator", "METHOD", fixtureRepoID+"@"+sha+":src/payment/AmountValidator.java:34", "probable", map[string]any{"simple_name": "AmountValidator", "return_type": "ValidationResult", "visibility": "public"})
	orderController := node("c", "OrderController", "CLASS", fixtureRepoID+"@"+sha+":src/api/OrderController.java:18", "exact", map[string]any{"simple_name": "OrderController"})
	invoice := node("d", "InvoiceService", "METHOD", fixtureRepoID+"@"+sha+":src/billing/InvoiceService.java:77", "exact", map[string]any{"simple_name": "InvoiceService", "return_type": "Invoice"})
	fraud := node("e", "FraudGateway", "CLASS", fixtureRepoID+"@"+sha+":src/integration/FraudGateway.java:22", "inferred", map[string]any{"simple_name": "FraudGateway"})
	checkout := node("f", "CheckoutFlow", "METHOD", fixtureRepoID+"@"+sha+":src/checkout/CheckoutFlow.java:55", "exact", map[string]any{"simple_name": "CheckoutFlow", "return_type": "void"})

	edge := func(fill string, from, to graphstore.Node, label, confidence, sourceRef string, tier int) graphstore.Edge {
		return graphstore.Edge{
			EdgeID:             strings.Repeat(fill, 64),
			Label:              label,
			FromNodeID:         from.NodeID,
			ToNodeID:           to.NodeID,
			Confidence:         confidence,
			SourceRef:          sourceRef,
			Tier:               tier,
			PhasePopulated:     2,
			CompletenessCaveat: "Fixture graph for deterministic tests.",
		}
	}

	nodes := []graphstore.Node{payment, validator, orderController, invoice, fraud, checkout}
	edges := []graphstore.Edge{
		edge("1", orderController, payment, "IMPORTS", "exact", orderController.SourceRef, 1),
		edge("2", payment, validator, "INJECTS", "probable", payment.SourceRef, 2),
		edge("3", payment, fraud, "CALLS_REST", "inferred", payment.SourceRef, 3),
		edge("4", validator, invoice, "SAME_FILE_CALLS", "exact", validator.SourceRef, 1),
		edge("5", checkout, payment, "IMPORTS", "exact", checkout.SourceRef, 1),
	}

	graph := &fixtureGraph{nodes: map[string]graphstore.Node{}, edges: edges}
	nodeByName := map[string]string{}
	docs := make([]fixtureDoc, 0, len(nodes))
	for _, item := range nodes {
		graph.nodes[item.NodeID] = item
		nodeByName[strings.ToLower(item.QualifiedName)] = item.NodeID
		if simple, ok := item.Properties["simple_name"].(string); ok {
			nodeByName[strings.ToLower(simple)] = item.NodeID
		}
		text := strings.Join([]string{item.QualifiedName, item.SourceRef, stringifyProps(item.Properties)}, " ")
		docs = append(docs, fixtureDoc{node: item, vector: embedText(text), lexical: tokenSet(text)})
	}
	return graph, &fixtureBackend{docs: docs}, nodeByName
}

func (fixtureEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	return embedText(text), nil
}

func (b *fixtureBackend) VectorSearch(_ context.Context, repoID string, embedding []float32, limit int) ([]retriever.SearchHit, error) {
	if limit <= 0 {
		limit = 10
	}
	type scored struct {
		doc   fixtureDoc
		score float64
	}
	scoredDocs := make([]scored, 0, len(b.docs))
	for _, doc := range b.docs {
		if doc.node.RepoID != repoID {
			continue
		}
		sim := cosine(embedding, doc.vector)
		scoredDocs = append(scoredDocs, scored{doc: doc, score: sim})
	}
	sort.Slice(scoredDocs, func(i, j int) bool {
		if scoredDocs[i].score == scoredDocs[j].score {
			return scoredDocs[i].doc.node.NodeID < scoredDocs[j].doc.node.NodeID
		}
		return scoredDocs[i].score > scoredDocs[j].score
	})
	if len(scoredDocs) > limit {
		scoredDocs = scoredDocs[:limit]
	}
	hits := make([]retriever.SearchHit, 0, len(scoredDocs))
	for idx, item := range scoredDocs {
		hits = append(hits, retriever.SearchHit{NodeID: item.doc.node.NodeID, SourceRef: item.doc.node.SourceRef, Confidence: item.doc.node.Confidence, Signal: "vector", Rank: idx + 1, Score: 1 - item.score})
	}
	return hits, nil
}

func (b *fixtureBackend) FTSSearch(_ context.Context, repoID, query string, limit int) ([]retriever.SearchHit, error) {
	if limit <= 0 {
		limit = 10
	}
	queryTokens := tokenSet(query)
	type scored struct {
		doc   fixtureDoc
		score float64
	}
	scoredDocs := make([]scored, 0, len(b.docs))
	for _, doc := range b.docs {
		if doc.node.RepoID != repoID {
			continue
		}
		score := overlap(queryTokens, doc.lexical)
		scoredDocs = append(scoredDocs, scored{doc: doc, score: score})
	}
	sort.Slice(scoredDocs, func(i, j int) bool {
		if scoredDocs[i].score == scoredDocs[j].score {
			return scoredDocs[i].doc.node.NodeID < scoredDocs[j].doc.node.NodeID
		}
		return scoredDocs[i].score > scoredDocs[j].score
	})
	if len(scoredDocs) > limit {
		scoredDocs = scoredDocs[:limit]
	}
	hits := make([]retriever.SearchHit, 0, len(scoredDocs))
	for idx, item := range scoredDocs {
		hits = append(hits, retriever.SearchHit{NodeID: item.doc.node.NodeID, SourceRef: item.doc.node.SourceRef, Confidence: item.doc.node.Confidence, Signal: "fts", Rank: idx + 1, Score: item.score})
	}
	return hits, nil
}

func (g *fixtureGraph) UpsertNode(_ context.Context, n graphstore.Node) error {
	g.nodes[n.NodeID] = n
	return nil
}

func (g *fixtureGraph) GetNode(_ context.Context, nodeID string) (*graphstore.Node, error) {
	node, ok := g.nodes[nodeID]
	if !ok {
		return nil, graphstore.ErrNodeNotFound
	}
	copy := node
	return &copy, nil
}

func (g *fixtureGraph) UpsertEdge(_ context.Context, e graphstore.Edge) error {
	g.edges = append(g.edges, e)
	return nil
}

func (g *fixtureGraph) BulkLoad(_ context.Context, nodes []graphstore.Node, edges []graphstore.Edge) error {
	for _, node := range nodes {
		g.nodes[node.NodeID] = node
	}
	g.edges = append(g.edges, edges...)
	return nil
}

func (g *fixtureGraph) DirectCallers(_ context.Context, nodeID string) ([]graphstore.Node, error) {
	var callers []graphstore.Node
	for _, edge := range g.edges {
		if edge.ToNodeID == nodeID && (edge.Label == "IMPORTS" || edge.Label == "SAME_FILE_CALLS") {
			if node, ok := g.nodes[edge.FromNodeID]; ok {
				callers = append(callers, node)
			}
		}
	}
	return callers, nil
}

func (g *fixtureGraph) Dependents(_ context.Context, nodeID string, depth int) ([]graphstore.Node, error) {
	if depth < 1 {
		depth = 1
	}
	seen := map[string]int{nodeID: 0}
	queue := []string{nodeID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		currentDepth := seen[current]
		if currentDepth == depth {
			continue
		}
		for _, edge := range g.edges {
			if edge.FromNodeID != current {
				continue
			}
			if _, ok := seen[edge.ToNodeID]; ok {
				continue
			}
			seen[edge.ToNodeID] = currentDepth + 1
			queue = append(queue, edge.ToNodeID)
		}
	}
	var out []graphstore.Node
	for candidate, hops := range seen {
		if candidate == nodeID || hops == 0 {
			continue
		}
		out = append(out, g.nodes[candidate])
	}
	return out, nil
}

func (g *fixtureGraph) BlastRadius(_ context.Context, nodeID string, depth int) (*graphstore.BlastRadiusResult, error) {
	if depth < 1 {
		depth = 1
	}
	seen := map[string]int{nodeID: 0}
	queue := []string{nodeID}
	usedEdges := map[string]graphstore.Edge{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		currentDepth := seen[current]
		if currentDepth == depth {
			continue
		}
		for _, edge := range g.edges {
			neighbors := []string{}
			switch {
			case edge.FromNodeID == current:
				neighbors = append(neighbors, edge.ToNodeID)
			case edge.ToNodeID == current:
				neighbors = append(neighbors, edge.FromNodeID)
			}
			for _, next := range neighbors {
				if _, ok := seen[next]; !ok {
					seen[next] = currentDepth + 1
					queue = append(queue, next)
				}
				usedEdges[edge.EdgeID] = edge
			}
		}
	}
	var nodes []graphstore.Node
	for candidate, hops := range seen {
		if candidate == nodeID || hops == 0 {
			continue
		}
		nodes = append(nodes, g.nodes[candidate])
	}
	var edges []graphstore.Edge
	for _, edge := range usedEdges {
		edges = append(edges, edge)
	}
	return &graphstore.BlastRadiusResult{RootNodeID: nodeID, Depth: depth, Nodes: nodes, Edges: edges, RepoID: fixtureRepoID, QueryDuration: time.Millisecond}, nil
}

func (g *fixtureGraph) Ping(context.Context) error { return nil }
func (g *fixtureGraph) Close() error               { return nil }

func embedText(text string) []float32 {
	vec := make([]float32, 32)
	for _, token := range strings.Fields(strings.ToLower(text)) {
		digest := sha256.Sum256([]byte(token))
		index := int(binary.BigEndian.Uint16(digest[:2])) % len(vec)
		vec[index] += 1
	}
	norm := float32(0)
	for _, value := range vec {
		norm += value * value
	}
	if norm == 0 {
		return vec
	}
	norm = float32(math.Sqrt(float64(norm)))
	for index := range vec {
		vec[index] /= norm
	}
	return vec
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var sum float64
	for idx := range a {
		sum += float64(a[idx] * b[idx])
	}
	return sum
}

func tokenSet(text string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, token := range strings.Fields(strings.ToLower(strings.NewReplacer("#", " ", "/", " ", ":", " ", ".", " ", "@", " ").Replace(text))) {
		out[token] = struct{}{}
	}
	return out
}

func overlap(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	matches := 0
	for token := range a {
		if _, ok := b[token]; ok {
			matches++
		}
	}
	return float64(matches) / float64(max(len(a), len(b)))
}

func stringifyProps(props map[string]any) string {
	parts := make([]string, 0, len(props))
	for _, value := range props {
		parts = append(parts, fmt.Sprint(value))
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

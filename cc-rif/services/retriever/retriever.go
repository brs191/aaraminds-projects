package retriever

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aaraminds/rif/graphstore"
)

const (
	defaultRRFK         = 60
	defaultSearchK      = 10
	defaultGraphDepth   = 2
	maxSearchGraphDepth = 3
	defaultImpactDepth  = 3
	maxImpactDepth      = 5
	defaultHubThreshold = 50
	hubDampingFactor    = 0.5
)

// QueryEmbedder converts a text query into an embedding vector.
type QueryEmbedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// SearchBackend provides the independent search signals used by the retriever.
type SearchBackend interface {
	VectorSearch(ctx context.Context, repoID string, embedding []float32, limit int) ([]SearchHit, error)
	FTSSearch(ctx context.Context, repoID, query string, limit int) ([]SearchHit, error)
}

// SearchHit is an intermediate ranked hit from a single signal.
type SearchHit struct {
	NodeID           string
	SourceRef        string
	Confidence       string
	Signal           string
	Rank             int
	Depth            int
	Score            float64
	CompletionCaveat string
}

// SearchRequest controls hybrid search.
type SearchRequest struct {
	RepoID     string
	Query      string
	K          int
	GraphDepth int
}

// SearchResult is the fused result returned by Search.
type SearchResult struct {
	NodeID     string
	SourceRef  string
	Score      float64
	Confidence string
	Signals    []string
}

// ImpactRequest controls impact analysis from a root node.
type ImpactRequest struct {
	RootNodeID string
	Depth      int
	Limit      int
}

// ImpactResult is a ranked impact-analysis row.
type ImpactResult struct {
	NodeID           string
	SourceRef        string
	Score            float64
	Tier             string
	DepthFromRoot    int
	CompletionCaveat string
}

// Retriever exposes the Phase 3 search and impact APIs.
type Retriever interface {
	Search(ctx context.Context, req SearchRequest) ([]SearchResult, error)
	Impact(ctx context.Context, req ImpactRequest) ([]ImpactResult, error)
}

// Service is the default Retriever implementation.
type Service struct {
	backend      SearchBackend
	graph        graphstore.GraphStore
	embedder     QueryEmbedder
	rrfK         int
	hubThreshold int
}

// NewService creates a retriever service.
func NewService(backend SearchBackend, graph graphstore.GraphStore, embedder QueryEmbedder) *Service {
	return &Service{
		backend:      backend,
		graph:        graph,
		embedder:     embedder,
		rrfK:         defaultRRFK,
		hubThreshold: defaultHubThreshold,
	}
}

// Search performs hybrid retrieval over vector, FTS, and graph signals.
func (s *Service) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	if s.backend == nil {
		return nil, errors.New("retriever backend is nil")
	}
	if s.embedder == nil {
		return nil, errors.New("retriever embedder is nil")
	}
	if strings.TrimSpace(req.Query) == "" {
		return []SearchResult{}, nil
	}
	if req.K <= 0 {
		req.K = defaultSearchK
	}
	graphDepth := req.GraphDepth
	if graphDepth <= 0 {
		graphDepth = defaultGraphDepth
	}
	if graphDepth > maxSearchGraphDepth {
		graphDepth = maxSearchGraphDepth
	}

	embedding, err := s.embedder.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	vectorHits, err := s.backend.VectorSearch(ctx, req.RepoID, embedding, req.K)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	ftsHits, err := s.backend.FTSSearch(ctx, req.RepoID, req.Query, req.K)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	graphHits, err := s.graphSignal(ctx, vectorHits, ftsHits, graphDepth, req.K)
	if err != nil {
		return nil, err
	}

	fused := fuseHybrid(vectorHits, ftsHits, graphHits, s.rrfK)
	if len(fused) > req.K {
		fused = fused[:req.K]
	}
	return fused, nil
}

// Impact performs depth-bounded graph impact analysis from a root node.
func (s *Service) Impact(ctx context.Context, req ImpactRequest) ([]ImpactResult, error) {
	if s.graph == nil {
		return nil, errors.New("graph store is nil")
	}
	if strings.TrimSpace(req.RootNodeID) == "" {
		return []ImpactResult{}, nil
	}
	depth := req.Depth
	if depth <= 0 {
		depth = defaultImpactDepth
	}
	if depth > maxImpactDepth {
		depth = maxImpactDepth
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultSearchK * 5
	}

	br, err := s.graph.BlastRadius(ctx, req.RootNodeID, depth)
	if err != nil {
		return nil, fmt.Errorf("blast radius: %w", err)
	}
	if br == nil {
		return []ImpactResult{}, nil
	}

	depths, pathTier, outgoing := impactMetadata(req.RootNodeID, br)

	results := make([]ImpactResult, 0, len(br.Nodes))
	for _, n := range br.Nodes {
		d, ok := depths[n.NodeID]
		if !ok || d <= 0 {
			continue
		}
		tier := pathTier[n.NodeID]
		if tier == "" {
			tier = "static"
		}
		damping := 1.0
		if outgoing[n.NodeID] > s.hubThreshold {
			damping = hubDampingFactor
		}
		score := tierWeight(tier) * damping / float64(d)
		results = append(results, ImpactResult{
			NodeID:           n.NodeID,
			SourceRef:        n.SourceRef,
			Score:            score,
			Tier:             tier,
			DepthFromRoot:    d,
			CompletionCaveat: impactCaveat(tier),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].NodeID < results[j].NodeID
		}
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (s *Service) graphSignal(ctx context.Context, vectorHits, ftsHits []SearchHit, depth, limit int) ([]SearchHit, error) {
	if s.graph == nil {
		return nil, nil
	}

	seeds := ftsHits
	if len(seeds) == 0 {
		seeds = vectorHits
	}
	if len(seeds) == 0 {
		return nil, nil
	}
	if len(seeds) > 3 {
		seeds = seeds[:3]
	}

	seen := make(map[string]SearchHit)
	for _, seed := range seeds {
		br, err := s.graph.BlastRadius(ctx, seed.NodeID, depth)
		if err != nil {
			return nil, fmt.Errorf("graph signal blast radius for %s: %w", seed.NodeID, err)
		}
		depths, _, _ := impactMetadata(seed.NodeID, br)
		nodesByID := make(map[string]graphstore.Node, len(br.Nodes))
		for _, n := range br.Nodes {
			nodesByID[n.NodeID] = n
		}
		for nodeID, d := range depths {
			if nodeID == seed.NodeID || d <= 0 {
				continue
			}
			n, ok := nodesByID[nodeID]
			if !ok {
				continue
			}
			hit := SearchHit{
				NodeID:           n.NodeID,
				SourceRef:        n.SourceRef,
				Confidence:       n.Confidence,
				Signal:           "graph",
				Rank:             d,
				Depth:            d,
				Score:            1 / float64(defaultRRFK+d),
				CompletionCaveat: "Graph neighborhood only; traversal is bounded and may miss runtime-wired edges.",
			}
			if existing, ok := seen[nodeID]; ok {
				existing.Score += hit.Score
				existing.Rank = min(existing.Rank, hit.Rank)
				existing.Depth = min(existing.Depth, hit.Depth)
				existing.Signal = existing.Signal + ",graph"
				seen[nodeID] = existing
			} else {
				seen[nodeID] = hit
			}
		}
	}

	hits := make([]SearchHit, 0, len(seen))
	for _, hit := range seen {
		hits = append(hits, hit)
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].NodeID < hits[j].NodeID
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

func fuseHybrid(vectorHits, ftsHits, graphHits []SearchHit, k int) []SearchResult {
	type candidate struct {
		NodeID     string
		SourceRef  string
		Confidence string
		signals    map[string]struct{}
		score      float64
	}

	candidates := map[string]*candidate{}
	merge := func(hits []SearchHit, signalName string) {
		for i, hit := range hits {
			if hit.NodeID == "" {
				continue
			}
			c, ok := candidates[hit.NodeID]
			if !ok {
				c = &candidate{
					NodeID:     hit.NodeID,
					SourceRef:  hit.SourceRef,
					Confidence: hit.Confidence,
					signals:    make(map[string]struct{}),
				}
				candidates[hit.NodeID] = c
			}
			if c.SourceRef == "" {
				c.SourceRef = hit.SourceRef
			}
			if c.Confidence == "" {
				c.Confidence = hit.Confidence
			}
			c.signals[signalName] = struct{}{}
			c.score += 1 / float64(k+i+1)
		}
	}

	merge(vectorHits, "vector")
	merge(ftsHits, "fts")
	merge(graphHits, "graph")

	results := make([]SearchResult, 0, len(candidates))
	for _, c := range candidates {
		signals := make([]string, 0, len(c.signals))
		for signal := range c.signals {
			signals = append(signals, signal)
		}
		sort.Strings(signals)
		results = append(results, SearchResult{
			NodeID:     c.NodeID,
			SourceRef:  c.SourceRef,
			Score:      c.score,
			Confidence: c.Confidence,
			Signals:    signals,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].NodeID < results[j].NodeID
		}
		return results[i].Score > results[j].Score
	})
	return results
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

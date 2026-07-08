package retriever

import (
	"context"
	"testing"

	"github.com/aaraminds/rif/graphstore"
)

type fakeGraphStore struct {
	result *graphstore.BlastRadiusResult
	nodes  map[string]graphstore.Node
}

func (f fakeGraphStore) UpsertNode(context.Context, graphstore.Node) error { return nil }
func (f fakeGraphStore) GetNode(_ context.Context, nodeID string) (*graphstore.Node, error) {
	n, ok := f.nodes[nodeID]
	if !ok {
		return nil, graphstore.ErrNodeNotFound
	}
	return &n, nil
}
func (f fakeGraphStore) UpsertEdge(context.Context, graphstore.Edge) error { return nil }
func (f fakeGraphStore) BulkLoad(context.Context, []graphstore.Node, []graphstore.Edge) error {
	return nil
}
func (f fakeGraphStore) DirectCallers(context.Context, string) ([]graphstore.Node, error) {
	return nil, nil
}
func (f fakeGraphStore) Dependents(context.Context, string, int) ([]graphstore.Node, error) {
	return nil, nil
}
func (f fakeGraphStore) BlastRadius(context.Context, string, int) (*graphstore.BlastRadiusResult, error) {
	return f.result, nil
}
func (f fakeGraphStore) Ping(context.Context) error { return nil }
func (f fakeGraphStore) Close() error               { return nil }

type fakeBackend struct {
	vector []SearchHit
	fts    []SearchHit
}

func (f fakeBackend) VectorSearch(context.Context, string, []float32, int) ([]SearchHit, error) {
	return f.vector, nil
}
func (f fakeBackend) FTSSearch(context.Context, string, string, int) ([]SearchHit, error) {
	return f.fts, nil
}

type fakeEmbedder struct{}

func (fakeEmbedder) Embed(context.Context, string) ([]float32, error) { return []float32{1, 2, 3}, nil }

func TestImpactAppliesHubDamping(t *testing.T) {
	root := graphstore.Node{NodeID: "root", RepoID: "repo"}
	low := graphstore.Node{NodeID: "low", RepoID: "repo", SourceRef: "r@sha:path:2"}
	high := graphstore.Node{NodeID: "high", RepoID: "repo", SourceRef: "r@sha:path:3"}
	edges := make([]graphstore.Edge, 0, 106)
	edges = append(edges, graphstore.Edge{EdgeID: "e1", FromNodeID: "root", ToNodeID: "low", Label: "IMPORTS"})
	edges = append(edges, graphstore.Edge{EdgeID: "e2", FromNodeID: "root", ToNodeID: "high", Label: "IMPORTS"})
	for i := 0; i < 5; i++ {
		edges = append(edges, graphstore.Edge{
			EdgeID:     string(rune('a' + i)),
			FromNodeID: "low",
			ToNodeID:   "x" + string(rune('a'+i)),
			Label:      "IMPORTS",
		})
	}
	for i := 0; i < 100; i++ {
		edges = append(edges, graphstore.Edge{
			EdgeID:     string(rune('f'+i%10)) + string(rune('a'+i/10)),
			FromNodeID: "high",
			ToNodeID:   "y" + string(rune('a'+i%26)),
			Label:      "IMPORTS",
		})
	}
	br := &graphstore.BlastRadiusResult{
		RootNodeID: "root",
		RepoID:     "repo",
		Nodes:      []graphstore.Node{low, high},
		Edges:      edges,
	}
	svc := NewService(fakeBackend{}, fakeGraphStore{result: br, nodes: map[string]graphstore.Node{"root": root, "low": low, "high": high}}, fakeEmbedder{})
	out, err := svc.Impact(context.Background(), ImpactRequest{RootNodeID: "root", Depth: 3, Limit: 10})
	if err != nil {
		t.Fatalf("impact: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 results, got %d", len(out))
	}
	if out[0].NodeID != "low" {
		t.Fatalf("expected low-degree node to rank first, got %s", out[0].NodeID)
	}
	if out[0].CompletionCaveat == "" || out[1].CompletionCaveat == "" {
		t.Fatal("completion caveat must be non-empty")
	}
}

func TestSearchRanksExactHitsAboveVectorOnly(t *testing.T) {
	svc := NewService(
		fakeBackend{
			vector: []SearchHit{{NodeID: "n2", SourceRef: "s2", Confidence: "probable", Signal: "vector"}},
			fts:    []SearchHit{{NodeID: "n1", SourceRef: "s1", Confidence: "exact", Signal: "fts"}},
		},
		fakeGraphStore{
			result: &graphstore.BlastRadiusResult{
				RootNodeID: "n1",
				RepoID:     "repo",
				Nodes:      []graphstore.Node{{NodeID: "n3", SourceRef: "s3", Confidence: "exact"}},
				Edges:      []graphstore.Edge{{EdgeID: "e1", FromNodeID: "n1", ToNodeID: "n3", Label: "IMPORTS"}},
			},
			nodes: map[string]graphstore.Node{"n1": {NodeID: "n1", SourceRef: "s1", Confidence: "exact"}},
		},
		fakeEmbedder{},
	)
	out, err := svc.Search(context.Background(), SearchRequest{RepoID: "repo", Query: "alpha", K: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(out) == 0 || out[0].NodeID != "n1" {
		t.Fatalf("expected exact FTS hit first, got %#v", out)
	}
}

package retriever

import (
	"context"
	"testing"

	"github.com/att/rif/graphstore"
)

func BenchmarkSearch(b *testing.B) {
	svc := NewService(
		fakeBackend{
			vector: []SearchHit{
				{NodeID: "n1", SourceRef: "s1", Confidence: "probable", Signal: "vector"},
				{NodeID: "n2", SourceRef: "s2", Confidence: "probable", Signal: "vector"},
				{NodeID: "n3", SourceRef: "s3", Confidence: "probable", Signal: "vector"},
			},
			fts: []SearchHit{
				{NodeID: "n2", SourceRef: "s2", Confidence: "exact", Signal: "fts"},
				{NodeID: "n4", SourceRef: "s4", Confidence: "exact", Signal: "fts"},
			},
		},
		fakeGraphStore{
			result: &graphstore.BlastRadiusResult{
				RootNodeID: "n2",
				RepoID:     "repo",
				Nodes: []graphstore.Node{
					{NodeID: "n5", SourceRef: "s5", Confidence: "inferred"},
				},
				Edges: []graphstore.Edge{
					{EdgeID: "e1", FromNodeID: "n2", ToNodeID: "n5", Label: "IMPORTS"},
				},
			},
		},
		fakeEmbedder{},
	)

	b.ReportAllocs()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.Search(ctx, SearchRequest{RepoID: "repo", Query: "alpha beta", K: 10})
		if err != nil {
			b.Fatal(err)
		}
	}
}

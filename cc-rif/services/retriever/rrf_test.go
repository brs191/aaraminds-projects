package retriever

import "testing"

func TestRRFScorePrefersConsistentHits(t *testing.T) {
	a := RRFScore(60, 1, 10)
	b := RRFScore(60, 2)
	if a <= b {
		t.Fatalf("expected combined ranks to beat single rank: a=%v b=%v", a, b)
	}
}

func TestFuseHybridAggregatesSignals(t *testing.T) {
	results := fuseHybrid(
		[]SearchHit{{NodeID: "n1", SourceRef: "r1", Confidence: "probable", Signal: "vector"}, {NodeID: "n2", SourceRef: "r2", Confidence: "probable", Signal: "vector"}},
		[]SearchHit{{NodeID: "n1", SourceRef: "r1", Confidence: "exact", Signal: "fts"}},
		nil,
		60,
	)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].NodeID != "n1" {
		t.Fatalf("expected n1 first, got %s", results[0].NodeID)
	}
	if len(results[0].Signals) != 2 {
		t.Fatalf("expected n1 to have 2 signals, got %v", results[0].Signals)
	}
}

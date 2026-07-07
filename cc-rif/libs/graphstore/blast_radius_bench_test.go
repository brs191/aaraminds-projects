// Package graphstore_test — BlastRadius benchmark.
//
// BenchmarkBlastRadius_Depth3_2500Nodes builds a synthetic 2500-node
// METHOD graph in a JSONStore (no Docker required) and measures BlastRadius
// at depth=3 from the root.
//
// Graph topology (total 2500 nodes):
//
//	Level 0:  1 root node
//	Level 1: 49 nodes   (root → each L1)
//	Level 2: 980 nodes  (each L1 → 20 L2 children)
//	Level 3: 1470 nodes (first 490 L2 nodes → 2 L3 children each;
//	                     remaining 490 L2 nodes → 1 L3 child each)
//
// Each edge is a SAME_FILE_CALLS edge so that bidirectional traversal at
// depth=3 visits the entire graph.
//
// SLA assertion: p95 < 1500 ms (surfaced via b.ReportMetric).
package graphstore_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"testing"
	"time"

	graphstore "github.com/att/rif/graphstore"
)

// randomHex64 generates a random 64-character hex string for use as a
// synthetic node_id in benchmark fixtures. It bypasses the content-addressed
// algorithm to allow arbitrary graph construction.
func randomHex64() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// makeBenchNode creates a minimal Node with a random node_id.
func makeBenchNode(repoID, qualifiedName string) graphstore.Node {
	return graphstore.Node{
		NodeID:         nodeID(repoID, qualifiedName, "METHOD"),
		RepoID:         repoID,
		QualifiedName:  qualifiedName,
		Kind:           "METHOD",
		SourceRef:      repoID + "@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:Bench.java:1",
		Confidence:     "exact",
		PhasePopulated: 1,
		Origin:         "first_party",
		ProvenanceKind: "file",
	}
}

// makeBenchEdge creates a SAME_FILE_CALLS edge between two existing nodes.
func makeBenchEdge(fromID, toID string) graphstore.Edge {
	return makeEdge(fromID, "SAME_FILE_CALLS", toID)
}

// buildBenchGraph constructs the 2500-node tree and loads it into store.
// It returns the root node's ID.
func buildBenchGraph(b *testing.B, store *graphstore.JSONStore) string {
	b.Helper()
	ctx := context.Background()
	const repo = "bench-repo"

	// Level 0: root
	root := makeBenchNode(repo, "bench.Root#root()")

	const l1Count = 49
	const l2PerL1 = 20 // 49 × 20 = 980 L2 nodes
	// Level 3: first 490 L2 nodes get 2 children, next 490 get 1 child → 1470
	const l3TwoChildrenCount = 490

	allNodes := make([]graphstore.Node, 0, 2500)
	allEdges := make([]graphstore.Edge, 0, 2500)

	allNodes = append(allNodes, root)

	// Level 1
	l1Nodes := make([]graphstore.Node, l1Count)
	for i := range l1Nodes {
		l1Nodes[i] = makeBenchNode(repo, fmt.Sprintf("bench.L1.N%d#m()", i))
		allNodes = append(allNodes, l1Nodes[i])
		allEdges = append(allEdges, makeBenchEdge(root.NodeID, l1Nodes[i].NodeID))
	}

	// Level 2
	l2Nodes := make([]graphstore.Node, 0, l1Count*l2PerL1)
	for _, l1 := range l1Nodes {
		for j := 0; j < l2PerL1; j++ {
			n := makeBenchNode(repo, fmt.Sprintf("bench.L2.%s.N%d#m()", l1.QualifiedName, j))
			l2Nodes = append(l2Nodes, n)
			allNodes = append(allNodes, n)
			allEdges = append(allEdges, makeBenchEdge(l1.NodeID, n.NodeID))
		}
	}

	// Level 3
	for i, l2 := range l2Nodes {
		if i < l3TwoChildrenCount {
			// 2 children
			for k := 0; k < 2; k++ {
				n := makeBenchNode(repo, fmt.Sprintf("bench.L3.%d.k%d#m()", i, k))
				allNodes = append(allNodes, n)
				allEdges = append(allEdges, makeBenchEdge(l2.NodeID, n.NodeID))
			}
		} else {
			// 1 child
			n := makeBenchNode(repo, fmt.Sprintf("bench.L3.%d.k0#m()", i))
			allNodes = append(allNodes, n)
			allEdges = append(allEdges, makeBenchEdge(l2.NodeID, n.NodeID))
		}
	}

	b.Logf("benchmark graph: %d nodes, %d edges", len(allNodes), len(allEdges))

	if err := store.BulkLoad(ctx, allNodes, allEdges); err != nil {
		b.Fatalf("BulkLoad benchmark graph: %v", err)
	}
	return root.NodeID
}

// BenchmarkBlastRadius_Depth3_2500Nodes measures the JSONStore BlastRadius
// query at depth=3 across a synthetic 2500-node fan-out graph.
//
// SLA: p95 < 1500 ms. The p95 latency is reported via b.ReportMetric.
func BenchmarkBlastRadius_Depth3_2500Nodes(b *testing.B) {
	// Build the graph once outside the benchmark loop.
	store, err := graphstore.NewJSONStore(filepath.Join(b.TempDir(), "bench.json"))
	if err != nil {
		b.Fatalf("NewJSONStore: %v", err)
	}
	defer store.Close()

	rootID := buildBenchGraph(b, store)
	ctx := context.Background()

	// Warm up: run once to populate any caches.
	if _, err := store.BlastRadius(ctx, rootID, 3); err != nil {
		b.Fatalf("warm-up BlastRadius: %v", err)
	}

	durations := make([]time.Duration, 0, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		result, err := store.BlastRadius(ctx, rootID, 3)
		dur := time.Since(start)
		if err != nil {
			b.Fatalf("BlastRadius[%d]: %v", i, err)
		}
		durations = append(durations, dur)
		b.ReportMetric(float64(len(result.Nodes)), "result_nodes")
	}
	b.StopTimer()

	// Compute and report p50 and p95.
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	p50 := durations[len(durations)/2]
	p95idx := int(float64(len(durations)) * 0.95)
	if p95idx >= len(durations) {
		p95idx = len(durations) - 1
	}
	p95 := durations[p95idx]

	b.ReportMetric(float64(p50.Milliseconds()), "p50-ms")
	b.ReportMetric(float64(p95.Milliseconds()), "p95-ms")

	if p95 > 1500*time.Millisecond {
		b.Errorf("p95 latency %dms exceeds 1500ms SLA", p95.Milliseconds())
	}
}

// Ensure randomHex64 is used to suppress the import if the compiler eliminates it.
var _ = randomHex64

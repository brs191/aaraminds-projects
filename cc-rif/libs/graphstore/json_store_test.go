// Package graphstore_test — unit tests for JSONStore.
// These tests require no external services; they run entirely in-process.
package graphstore_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	graphstore "github.com/aaraminds/rif/graphstore"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// sha256hex returns the lowercase hex SHA-256 digest of s.
func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// nodeID computes a canonical node_id from (repoID, qualifiedName, kind) —
// matching the CODE_MODEL §5 algorithm.
func nodeID(repoID, qualifiedName, kind string) string {
	return sha256hex(repoID + "\x00" + qualifiedName + "\x00" + kind)
}

// edgeID computes a canonical edge_id from (fromNodeID, label, toNodeID).
func edgeID(fromNodeID, label, toNodeID string) string {
	return sha256hex(fromNodeID + "\x00" + label + "\x00" + toNodeID)
}

// makeNode returns a minimal valid Node with the given identifiers.
func makeNode(repoID, qualifiedName, kind string) graphstore.Node {
	id := nodeID(repoID, qualifiedName, kind)
	return graphstore.Node{
		NodeID:         id,
		RepoID:         repoID,
		QualifiedName:  qualifiedName,
		Kind:           kind,
		SourceRef:      repoID + "@deadbeefdeadbeefdeadbeefdeadbeefdeadbeef:src/Test.java:1",
		Confidence:     "exact",
		PhasePopulated: 1,
		Origin:         "first_party",
		ProvenanceKind: "file",
	}
}

// makeEdge returns a minimal valid Edge between two node IDs.
func makeEdge(fromID, label, toID string) graphstore.Edge {
	return graphstore.Edge{
		EdgeID:             edgeID(fromID, label, toID),
		Label:              label,
		FromNodeID:         fromID,
		ToNodeID:           toID,
		Confidence:         "exact",
		SourceRef:          "repo@sha:src/Test.java:10",
		Tier:               1,
		PhasePopulated:     1,
		CompletenessCaveat: "test caveat",
	}
}

// newTempStore creates a JSONStore backed by a temp file and registers cleanup.
func newTempStore(t *testing.T) *graphstore.JSONStore {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")
	s, err := graphstore.NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestJSONStore_UpsertGetNode(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	want := makeNode("repo-a", "com.example.Foo", "CLASS")
	want.Properties = map[string]any{
		"simple_name": "Foo",
		"is_abstract": false,
	}

	if err := s.UpsertNode(ctx, want); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	got, err := s.GetNode(ctx, want.NodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}

	// Verify every exported field round-trips.
	if got.NodeID != want.NodeID {
		t.Errorf("NodeID: got %q want %q", got.NodeID, want.NodeID)
	}
	if got.RepoID != want.RepoID {
		t.Errorf("RepoID: got %q want %q", got.RepoID, want.RepoID)
	}
	if got.QualifiedName != want.QualifiedName {
		t.Errorf("QualifiedName: got %q want %q", got.QualifiedName, want.QualifiedName)
	}
	if got.Kind != want.Kind {
		t.Errorf("Kind: got %q want %q", got.Kind, want.Kind)
	}
	if got.SourceRef != want.SourceRef {
		t.Errorf("SourceRef: got %q want %q", got.SourceRef, want.SourceRef)
	}
	if got.Confidence != want.Confidence {
		t.Errorf("Confidence: got %q want %q", got.Confidence, want.Confidence)
	}
	if got.PhasePopulated != want.PhasePopulated {
		t.Errorf("PhasePopulated: got %d want %d", got.PhasePopulated, want.PhasePopulated)
	}
	if got.Origin != want.Origin {
		t.Errorf("Origin: got %q want %q", got.Origin, want.Origin)
	}
	if got.ProvenanceKind != want.ProvenanceKind {
		t.Errorf("ProvenanceKind: got %q want %q", got.ProvenanceKind, want.ProvenanceKind)
	}
	if got.Properties["simple_name"] != "Foo" {
		t.Errorf("Properties[simple_name]: got %v want %q", got.Properties["simple_name"], "Foo")
	}
}

func TestJSONStore_UpsertGetNode_Idempotent(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	n := makeNode("repo-a", "com.example.Bar", "INTERFACE")
	if err := s.UpsertNode(ctx, n); err != nil {
		t.Fatalf("first UpsertNode: %v", err)
	}
	// Upsert again with same node_id — should replace, not duplicate.
	n.Confidence = "probable"
	if err := s.UpsertNode(ctx, n); err != nil {
		t.Fatalf("second UpsertNode: %v", err)
	}
	got, err := s.GetNode(ctx, n.NodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.Confidence != "probable" {
		t.Errorf("Confidence after upsert: got %q want %q", got.Confidence, "probable")
	}
}

func TestJSONStore_GetNode_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	missingID := sha256hex("no-such-node")
	_, err := s.GetNode(ctx, missingID)
	if err != graphstore.ErrNodeNotFound {
		t.Errorf("expected ErrNodeNotFound; got %v", err)
	}
}

func TestJSONStore_UpsertEdge(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	callerNode := makeNode("repo-a", "com.example.A#doA()", "METHOD")
	calleeNode := makeNode("repo-a", "com.example.B#doB()", "METHOD")
	e := makeEdge(callerNode.NodeID, "SAME_FILE_CALLS", calleeNode.NodeID)

	for _, n := range []graphstore.Node{callerNode, calleeNode} {
		if err := s.UpsertNode(ctx, n); err != nil {
			t.Fatalf("UpsertNode %s: %v", n.NodeID, err)
		}
	}
	if err := s.UpsertEdge(ctx, e); err != nil {
		t.Fatalf("UpsertEdge: %v", err)
	}

	callers, err := s.DirectCallers(ctx, calleeNode.NodeID)
	if err != nil {
		t.Fatalf("DirectCallers: %v", err)
	}
	if len(callers) != 1 {
		t.Fatalf("DirectCallers: expected 1 caller, got %d", len(callers))
	}
	if callers[0].NodeID != callerNode.NodeID {
		t.Errorf("DirectCallers[0].NodeID: got %q want %q", callers[0].NodeID, callerNode.NodeID)
	}
}

func TestJSONStore_InvalidEdgeID(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	callerNode := makeNode("repo-a", "com.example.InvalidEdgeCaller#doA()", "METHOD")
	calleeNode := makeNode("repo-a", "com.example.InvalidEdgeCallee#doB()", "METHOD")
	if err := s.BulkLoad(ctx, []graphstore.Node{callerNode, calleeNode}, nil); err != nil {
		t.Fatalf("BulkLoad nodes: %v", err)
	}

	edge := makeEdge(callerNode.NodeID, "SAME_FILE_CALLS", calleeNode.NodeID)
	edge.EdgeID = "not-a-valid-edge-id"
	if err := s.UpsertEdge(ctx, edge); err != graphstore.ErrInvalidEdgeID {
		t.Fatalf("expected ErrInvalidEdgeID; got %v", err)
	}

	if err := s.BulkLoad(ctx, nil, []graphstore.Edge{edge}); !errors.Is(err, graphstore.ErrInvalidEdgeID) {
		t.Fatalf("BulkLoad expected ErrInvalidEdgeID; got %v", err)
	}
}

func TestJSONStore_DirectCallers_IMPORTSEdge(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	fileNode := makeNode("repo-a", "src/main/java/Foo.java", "FILE")
	classNode := makeNode("repo-a", "com.example.Foo", "CLASS")
	e := makeEdge(fileNode.NodeID, "IMPORTS", classNode.NodeID)

	_ = s.UpsertNode(ctx, fileNode)
	_ = s.UpsertNode(ctx, classNode)
	_ = s.UpsertEdge(ctx, e)

	callers, err := s.DirectCallers(ctx, classNode.NodeID)
	if err != nil {
		t.Fatalf("DirectCallers: %v", err)
	}
	if len(callers) != 1 || callers[0].NodeID != fileNode.NodeID {
		t.Errorf("expected fileNode as caller; got %v", callers)
	}
}

func TestJSONStore_DirectCallers_IgnoresOtherLabels(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	parent := makeNode("repo-a", "com.example.Parent", "CLASS")
	child := makeNode("repo-a", "com.example.Child", "CLASS")
	e := makeEdge(child.NodeID, "EXTENDS", parent.NodeID) // should NOT appear as caller

	_ = s.UpsertNode(ctx, parent)
	_ = s.UpsertNode(ctx, child)
	_ = s.UpsertEdge(ctx, e)

	callers, err := s.DirectCallers(ctx, parent.NodeID)
	if err != nil {
		t.Fatalf("DirectCallers: %v", err)
	}
	if len(callers) != 0 {
		t.Errorf("EXTENDS edge must not appear in DirectCallers; got %v", callers)
	}
}

func TestJSONStore_BulkLoad(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	const nodeCount = 10
	const edgeCount = 15

	nodes := make([]graphstore.Node, nodeCount)
	for i := range nodes {
		nodes[i] = makeNode("repo-a", fmt.Sprintf("com.example.C%d", i), "CLASS")
	}

	edges := make([]graphstore.Edge, 0, edgeCount)
	for i := 0; i < edgeCount; i++ {
		from := nodes[i%nodeCount]
		to := nodes[(i+1)%nodeCount]
		edges = append(edges, makeEdge(from.NodeID, "SAME_FILE_CALLS", to.NodeID))
	}

	if err := s.BulkLoad(ctx, nodes, edges); err != nil {
		t.Fatalf("BulkLoad: %v", err)
	}

	// Verify all nodes are retrievable.
	for _, n := range nodes {
		if _, err := s.GetNode(ctx, n.NodeID); err != nil {
			t.Errorf("GetNode(%s) after BulkLoad: %v", n.NodeID, err)
		}
	}

	// Verify edge count by checking DirectCallers counts.
	// nodes[1] is the destination of edge from nodes[0] (index i=0: 0%10=0 → 1%10=1).
	callers, err := s.DirectCallers(ctx, nodes[1].NodeID)
	if err != nil {
		t.Fatalf("DirectCallers: %v", err)
	}
	if len(callers) == 0 {
		t.Error("expected at least one caller after BulkLoad")
	}
}

func TestJSONStore_Dependents(t *testing.T) {
	// Build a 5-hop linear chain: n0 → n1 → n2 → n3 → n4 → n5
	// Dependents from n0 at depth=2 should return exactly {n1, n2}.
	ctx := context.Background()
	s := newTempStore(t)

	chain := make([]graphstore.Node, 6)
	for i := range chain {
		chain[i] = makeNode("repo-a", fmt.Sprintf("com.example.Chain%d", i), "METHOD")
	}

	nodes := make([]graphstore.Node, len(chain))
	copy(nodes, chain)
	edges := make([]graphstore.Edge, len(chain)-1)
	for i := 0; i < len(chain)-1; i++ {
		edges[i] = makeEdge(chain[i].NodeID, "SAME_FILE_CALLS", chain[i+1].NodeID)
	}

	if err := s.BulkLoad(ctx, nodes, edges); err != nil {
		t.Fatalf("BulkLoad: %v", err)
	}

	deps, err := s.Dependents(ctx, chain[0].NodeID, 2)
	if err != nil {
		t.Fatalf("Dependents: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("Dependents depth=2: expected 2 nodes, got %d: %v", len(deps), deps)
	}

	depIDs := map[string]bool{}
	for _, d := range deps {
		depIDs[d.NodeID] = true
	}
	for _, expected := range []string{chain[1].NodeID, chain[2].NodeID} {
		if !depIDs[expected] {
			t.Errorf("expected node %s in Dependents result", expected)
		}
	}
	// n0 itself must not appear.
	if depIDs[chain[0].NodeID] {
		t.Error("root node must not appear in Dependents result")
	}
}

func TestJSONStore_Dependents_ExcludesRoot(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	n := makeNode("repo-a", "com.example.Solo", "CLASS")
	if err := s.UpsertNode(ctx, n); err != nil {
		t.Fatal(err)
	}

	deps, err := s.Dependents(ctx, n.NodeID, 1)
	if err != nil {
		t.Fatalf("Dependents: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 dependents for isolated node; got %d", len(deps))
	}
}

func TestJSONStore_BlastRadius(t *testing.T) {
	// Star graph: hub → spoke0, spoke1, spoke2, spoke3, spoke4.
	// BlastRadius(hub, depth=1) must return exactly 5 nodes.
	ctx := context.Background()
	s := newTempStore(t)

	hub := makeNode("repo-a", "com.example.Hub#process()", "METHOD")
	spokes := make([]graphstore.Node, 5)
	for i := range spokes {
		spokes[i] = makeNode("repo-a", fmt.Sprintf("com.example.Spoke%d#run()", i), "METHOD")
	}

	nodes := append([]graphstore.Node{hub}, spokes...)
	edges := make([]graphstore.Edge, len(spokes))
	for i, sp := range spokes {
		edges[i] = makeEdge(hub.NodeID, "SAME_FILE_CALLS", sp.NodeID)
	}

	if err := s.BulkLoad(ctx, nodes, edges); err != nil {
		t.Fatalf("BulkLoad: %v", err)
	}

	result, err := s.BlastRadius(ctx, hub.NodeID, 1)
	if err != nil {
		t.Fatalf("BlastRadius: %v", err)
	}
	if len(result.Nodes) != 5 {
		t.Errorf("BlastRadius depth=1: expected 5 nodes, got %d", len(result.Nodes))
	}
	if result.RootNodeID != hub.NodeID {
		t.Errorf("RootNodeID: got %q want %q", result.RootNodeID, hub.NodeID)
	}
	if result.Depth != 1 {
		t.Errorf("Depth: got %d want 1", result.Depth)
	}
	if len(result.Edges) != 5 {
		t.Errorf("BlastRadius depth=1: expected 5 edges, got %d", len(result.Edges))
	}
}

func TestJSONStore_BlastRadius_Bidirectional(t *testing.T) {
	// a → hub ← b  (two inbound edges)
	// BlastRadius(hub, depth=1) must return {a, b} (bidirectional).
	ctx := context.Background()
	s := newTempStore(t)

	hub := makeNode("repo-a", "com.example.Hub#x()", "METHOD")
	a := makeNode("repo-a", "com.example.A#x()", "METHOD")
	b := makeNode("repo-a", "com.example.B#x()", "METHOD")
	eA := makeEdge(a.NodeID, "SAME_FILE_CALLS", hub.NodeID)
	eB := makeEdge(b.NodeID, "SAME_FILE_CALLS", hub.NodeID)

	if err := s.BulkLoad(ctx, []graphstore.Node{hub, a, b}, []graphstore.Edge{eA, eB}); err != nil {
		t.Fatal(err)
	}

	result, err := s.BlastRadius(ctx, hub.NodeID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Nodes) != 2 {
		t.Errorf("expected 2 nodes (a and b); got %d", len(result.Nodes))
	}
}

func TestJSONStore_InvalidNodeID(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	badIDs := []string{
		"",
		"short",
		"UPPERCASE0000000000000000000000000000000000000000000000000000000", // 64 but uppercase
		"gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg", // invalid hex
	}

	for _, id := range badIDs {
		t.Run(id, func(t *testing.T) {
			n := graphstore.Node{NodeID: id, Kind: "CLASS"}
			if err := s.UpsertNode(ctx, n); err != graphstore.ErrInvalidNodeID {
				t.Errorf("UpsertNode(%q): expected ErrInvalidNodeID; got %v", id, err)
			}
			if _, err := s.GetNode(ctx, id); err != graphstore.ErrInvalidNodeID {
				t.Errorf("GetNode(%q): expected ErrInvalidNodeID; got %v", id, err)
			}
			if _, err := s.DirectCallers(ctx, id); err != graphstore.ErrInvalidNodeID {
				t.Errorf("DirectCallers(%q): expected ErrInvalidNodeID; got %v", id, err)
			}
			if _, err := s.Dependents(ctx, id, 1); err != graphstore.ErrInvalidNodeID {
				t.Errorf("Dependents(%q): expected ErrInvalidNodeID; got %v", id, err)
			}
			if _, err := s.BlastRadius(ctx, id, 1); err != graphstore.ErrInvalidNodeID {
				t.Errorf("BlastRadius(%q): expected ErrInvalidNodeID; got %v", id, err)
			}
		})
	}
}

func TestJSONStore_DepthOutOfRange(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	n := makeNode("repo-a", "com.example.D", "CLASS")
	if err := s.UpsertNode(ctx, n); err != nil {
		t.Fatal(err)
	}

	for _, depth := range []int{0, 6, -1, 100} {
		t.Run(fmt.Sprintf("depth=%d", depth), func(t *testing.T) {
			if _, err := s.Dependents(ctx, n.NodeID, depth); err != graphstore.ErrDepthOutOfRange {
				t.Errorf("Dependents(depth=%d): expected ErrDepthOutOfRange; got %v", depth, err)
			}
			if _, err := s.BlastRadius(ctx, n.NodeID, depth); err != graphstore.ErrDepthOutOfRange {
				t.Errorf("BlastRadius(depth=%d): expected ErrDepthOutOfRange; got %v", depth, err)
			}
		})
	}
}

func TestJSONStore_Persistence(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")

	n := makeNode("repo-persist", "com.example.Persistent", "CLASS")

	// Write data with the first store instance.
	s1, err := graphstore.NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore (first open): %v", err)
	}
	if err := s1.UpsertNode(ctx, n); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	s1.Close()

	// Verify the backing file was written.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("backing file not created: %v", err)
	}

	// Reopen with a new store instance and verify data survives.
	s2, err := graphstore.NewJSONStore(path)
	if err != nil {
		t.Fatalf("NewJSONStore (second open): %v", err)
	}
	defer s2.Close()

	got, err := s2.GetNode(ctx, n.NodeID)
	if err != nil {
		t.Fatalf("GetNode after reopen: %v", err)
	}
	if got.QualifiedName != n.QualifiedName {
		t.Errorf("QualifiedName after reopen: got %q want %q", got.QualifiedName, n.QualifiedName)
	}
}

func TestJSONStore_BulkLoad_InvalidNode(t *testing.T) {
	ctx := context.Background()
	s := newTempStore(t)

	badNode := graphstore.Node{NodeID: "bad-id", Kind: "CLASS"}
	err := s.BulkLoad(ctx, []graphstore.Node{badNode}, nil)
	// BulkLoad wraps ErrInvalidNodeID; use errors.Is to unwrap.
	if !errors.Is(err, graphstore.ErrInvalidNodeID) {
		t.Errorf("BulkLoad with invalid node_id: expected ErrInvalidNodeID (possibly wrapped); got %v", err)
	}
}

func TestJSONStore_Ping(t *testing.T) {
	s := newTempStore(t)
	if err := s.Ping(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

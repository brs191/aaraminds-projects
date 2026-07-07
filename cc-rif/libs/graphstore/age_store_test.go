//go:build integration

// Package graphstore_test — integration tests for AGEStore.
//
// These tests require a running PostgreSQL + Apache AGE database.
// The database URL is resolved in the following order:
//  1. DATABASE_URL environment variable (connect to an existing DB).
//  2. A Docker container started via testcontainers-go using the
//     sormy/postgres-age:pg16 image (requires Docker).
//
// If neither is available the tests are skipped.
//
// To run:
//
//	DATABASE_URL=postgresql://localhost/repointel go test -tags integration ./...
package graphstore_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	graphstore "github.com/att/rif/graphstore"
)

// ─── container / connection setup ─────────────────────────────────────────────

// setupAGE resolves a DATABASE_URL for the integration tests.
// It tries, in order:
//  1. DATABASE_URL environment variable.
//  2. A testcontainers-based postgres-age:pg16 container (Docker required).
//
// Returns the URL and a cleanup function. Calls t.Skip if neither is available.
func setupAGE(t *testing.T) (string, func()) {
	t.Helper()

	// 1. Prefer an externally-provided database (e.g. the local dev instance).
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url, func() {}
	}

	// 2. Try to start a container.
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "sormy/postgres-age:pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "riftest",
			"POSTGRES_USER":     "rifuser",
			"POSTGRES_PASSWORD": "rifpass",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(90 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skipf("integration: cannot start AGE container (Docker unavailable or image missing): %v", err)
		return "", nil
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx) //nolint:errcheck
		t.Skipf("integration: container host: %v", err)
		return "", nil
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		container.Terminate(ctx) //nolint:errcheck
		t.Skipf("integration: container port: %v", err)
		return "", nil
	}

	dbURL := fmt.Sprintf("postgres://rifuser:rifpass@%s:%s/riftest", host, port.Port())
	cleanup := func() {
		container.Terminate(ctx) //nolint:errcheck
	}
	return dbURL, cleanup
}

// setupAGEStore opens an AGEStore and, if the database was freshly provisioned
// by testcontainers, ensures the AGE extension and graph exist.
func setupAGEStore(t *testing.T) (*graphstore.AGEStore, func()) {
	t.Helper()
	ctx := context.Background()

	dbURL, containerCleanup := setupAGE(t)

	store, err := graphstore.NewAGEStore(ctx, dbURL)
	if err != nil {
		containerCleanup()
		t.Fatalf("NewAGEStore: %v", err)
	}

	cleanup := func() {
		store.Close()
		containerCleanup()
	}
	return store, cleanup
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestAGEStore_PingRoundtrip(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupAGEStore(t)
	defer cleanup()

	if err := store.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestAGEStore_UpsertGetNode(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupAGEStore(t)
	defer cleanup()

	n := makeNode("rif-int-test", "com.example.IntTest", "CLASS")
	n.Properties = map[string]any{"simple_name": "IntTest"}

	if err := store.UpsertNode(ctx, n); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	got, err := store.GetNode(ctx, n.NodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.NodeID != n.NodeID {
		t.Errorf("NodeID: got %q want %q", got.NodeID, n.NodeID)
	}
	if got.QualifiedName != n.QualifiedName {
		t.Errorf("QualifiedName: got %q want %q", got.QualifiedName, n.QualifiedName)
	}
	if got.Kind != n.Kind {
		t.Errorf("Kind: got %q want %q", got.Kind, n.Kind)
	}
}

func TestAGEStore_UpsertGetNode_NotFound(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupAGEStore(t)
	defer cleanup()

	missing := sha256hex("no-such-node-in-age")
	_, err := store.GetNode(ctx, missing)
	if err != graphstore.ErrNodeNotFound {
		t.Errorf("expected ErrNodeNotFound; got %v", err)
	}
}

func TestAGEStore_BulkLoad_10nodes_15edges(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupAGEStore(t)
	defer cleanup()

	const nodeCount = 10
	const edgeCount = 15

	nodes := make([]graphstore.Node, nodeCount)
	for i := range nodes {
		nodes[i] = makeNode("rif-bulk-test", fmt.Sprintf("com.example.Bulk%d", i), "CLASS")
	}

	edges := make([]graphstore.Edge, 0, edgeCount)
	for i := 0; i < edgeCount; i++ {
		from := nodes[i%nodeCount]
		to := nodes[(i+1)%nodeCount]
		edges = append(edges, makeEdge(from.NodeID, "EXTENDS", to.NodeID))
	}

	if err := store.BulkLoad(ctx, nodes, edges); err != nil {
		t.Fatalf("BulkLoad: %v", err)
	}

	// Spot-check: verify node 0 is retrievable.
	got, err := store.GetNode(ctx, nodes[0].NodeID)
	if err != nil {
		t.Fatalf("GetNode after BulkLoad: %v", err)
	}
	if got.NodeID != nodes[0].NodeID {
		t.Errorf("NodeID mismatch after BulkLoad")
	}
}

func TestAGEStore_DirectCallers(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupAGEStore(t)
	defer cleanup()

	caller := makeNode("rif-dc-test", "com.example.DC.A#call()", "METHOD")
	callee := makeNode("rif-dc-test", "com.example.DC.B#target()", "METHOD")
	e := makeEdge(caller.NodeID, "SAME_FILE_CALLS", callee.NodeID)

	if err := store.BulkLoad(ctx, []graphstore.Node{caller, callee}, []graphstore.Edge{e}); err != nil {
		t.Fatalf("BulkLoad: %v", err)
	}

	callers, err := store.DirectCallers(ctx, callee.NodeID)
	if err != nil {
		t.Fatalf("DirectCallers: %v", err)
	}
	if len(callers) < 1 {
		t.Fatalf("expected at least 1 caller; got 0")
	}

	found := false
	for _, c := range callers {
		if c.NodeID == caller.NodeID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected caller %s in DirectCallers result", caller.NodeID)
	}
}

func TestAGEStore_BlastRadius_depth3(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupAGEStore(t)
	defer cleanup()

	// Build a 4-hop chain: n0 → n1 → n2 → n3.
	// BlastRadius(n0, depth=3) must return at least {n1, n2, n3}.
	const prefix = "rif-br-test"
	chain := make([]graphstore.Node, 4)
	for i := range chain {
		chain[i] = makeNode(prefix, fmt.Sprintf("com.example.BR.N%d#m()", i), "METHOD")
	}
	edges := make([]graphstore.Edge, len(chain)-1)
	for i := range edges {
		edges[i] = makeEdge(chain[i].NodeID, "SAME_FILE_CALLS", chain[i+1].NodeID)
	}

	if err := store.BulkLoad(ctx, chain, edges); err != nil {
		t.Fatalf("BulkLoad: %v", err)
	}

	result, err := store.BlastRadius(ctx, chain[0].NodeID, 3)
	if err != nil {
		t.Fatalf("BlastRadius: %v", err)
	}
	if len(result.Nodes) < 3 {
		t.Errorf("BlastRadius depth=3 on 4-chain: expected ≥3 nodes; got %d", len(result.Nodes))
	}
}

func TestAGEStore_InvalidNodeID(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupAGEStore(t)
	defer cleanup()

	_, err := store.GetNode(ctx, "not-a-valid-node-id")
	if err != graphstore.ErrInvalidNodeID {
		t.Errorf("expected ErrInvalidNodeID; got %v", err)
	}
}

func TestAGEStore_DepthOutOfRange(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupAGEStore(t)
	defer cleanup()

	validID := sha256hex("dummy")
	if _, err := store.Dependents(ctx, validID, 6); err != graphstore.ErrDepthOutOfRange {
		t.Errorf("Dependents(depth=6): expected ErrDepthOutOfRange; got %v", err)
	}
	if _, err := store.BlastRadius(ctx, validID, 0); err != graphstore.ErrDepthOutOfRange {
		t.Errorf("BlastRadius(depth=0): expected ErrDepthOutOfRange; got %v", err)
	}
}

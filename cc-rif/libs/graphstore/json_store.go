// Package graphstore — JSONStore: a thread-safe, file-backed, in-memory
// implementation of [GraphStore]. No external dependencies; stdlib only.
package graphstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// jsonData is the serialisable graph state written to the backing file.
// It is intentionally simple: two flat maps keyed by their respective IDs.
type jsonData struct {
	Nodes map[string]Node `json:"nodes"` // keyed by node_id
	Edges map[string]Edge `json:"edges"` // keyed by edge_id
}

// JSONStore is a thread-safe, in-memory [GraphStore] backed by a JSON file.
// Every write is persisted atomically via a sibling temp file and rename,
// so the backing file is never left in a partially-written state.
//
// JSONStore is intended for unit tests and offline tooling. It is not
// suitable for large repositories — all traversal is O(V + E).
type JSONStore struct {
	mu   sync.RWMutex
	data jsonData
	path string
}

// NewJSONStore opens (or creates) a JSONStore at path. If the file exists its
// contents are loaded into memory. Returns an error if the file cannot be read
// or contains invalid JSON.
func NewJSONStore(path string) (*JSONStore, error) {
	s := &JSONStore{
		path: path,
		data: jsonData{
			Nodes: make(map[string]Node),
			Edges: make(map[string]Edge),
		},
	}
	if _, err := os.Stat(path); err == nil {
		if err := s.load(); err != nil {
			return nil, fmt.Errorf("graphstore: JSONStore: load %q: %w", path, err)
		}
	}
	return s, nil
}

// load reads s.path and populates s.data. The caller must not hold s.mu.
func (s *JSONStore) load() error {
	f, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(&s.data)
}

// save atomically persists s.data to s.path.
// It writes to a sibling temp file in the same directory, then calls
// os.Rename to atomically replace the destination.
// The caller must hold s.mu for writing.
func (s *JSONStore) save() error {
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".jsonstore-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("encode json: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename to %q: %w", s.path, err)
	}
	return nil
}

// UpsertNode implements [GraphStore]. The node is stored by node_id; an
// existing node with the same node_id is replaced wholesale.
func (s *JSONStore) UpsertNode(_ context.Context, n Node) error {
	if err := validateNodeID(n.NodeID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Nodes[n.NodeID] = n
	return s.save()
}

// GetNode implements [GraphStore].
func (s *JSONStore) GetNode(_ context.Context, nodeID string) (*Node, error) {
	if err := validateNodeID(nodeID); err != nil {
		return nil, err
	}
	s.mu.RLock()
	n, ok := s.data.Nodes[nodeID]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrNodeNotFound
	}
	return &n, nil
}

// UpsertEdge implements [GraphStore]. The edge is stored by edge_id; an
// existing edge with the same edge_id is replaced wholesale.
func (s *JSONStore) UpsertEdge(_ context.Context, e Edge) error {
	if err := validateEdgeID(e.EdgeID); err != nil {
		return err
	}
	if err := validateNodeID(e.FromNodeID); err != nil {
		return fmt.Errorf("from_node_id: %w", err)
	}
	if err := validateNodeID(e.ToNodeID); err != nil {
		return fmt.Errorf("to_node_id: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Edges[e.EdgeID] = e
	return s.save()
}

// BulkLoad implements [GraphStore]. All nodes and edges are written in a
// single lock acquisition and a single file flush, making it considerably
// faster than calling UpsertNode/UpsertEdge in a loop.
func (s *JSONStore) BulkLoad(_ context.Context, nodes []Node, edges []Edge) error {
	// Validate all node IDs up-front so we fail atomically.
	for i, n := range nodes {
		if err := validateNodeID(n.NodeID); err != nil {
			return fmt.Errorf("node[%d] %q: %w", i, n.NodeID, err)
		}
	}
	for i, e := range edges {
		if err := validateEdgeID(e.EdgeID); err != nil {
			return fmt.Errorf("edge[%d] edge_id %q: %w", i, e.EdgeID, err)
		}
		if err := validateNodeID(e.FromNodeID); err != nil {
			return fmt.Errorf("edge[%d] from_node_id %q: %w", i, e.FromNodeID, err)
		}
		if err := validateNodeID(e.ToNodeID); err != nil {
			return fmt.Errorf("edge[%d] to_node_id %q: %w", i, e.ToNodeID, err)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range nodes {
		s.data.Nodes[n.NodeID] = n
	}
	for _, e := range edges {
		s.data.Edges[e.EdgeID] = e
	}
	return s.save()
}

// DirectCallers implements [GraphStore]. It returns all nodes that are the
// source of a SAME_FILE_CALLS or IMPORTS edge whose destination is nodeID.
func (s *JSONStore) DirectCallers(_ context.Context, nodeID string) ([]Node, error) {
	if err := validateNodeID(nodeID); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]struct{})
	var callers []Node
	for _, e := range s.data.Edges {
		if e.ToNodeID != nodeID {
			continue
		}
		if e.Label != "SAME_FILE_CALLS" && e.Label != "IMPORTS" {
			continue
		}
		if _, dup := seen[e.FromNodeID]; dup {
			continue
		}
		seen[e.FromNodeID] = struct{}{}
		if n, ok := s.data.Nodes[e.FromNodeID]; ok {
			callers = append(callers, n)
		}
	}
	return callers, nil
}

// Dependents implements [GraphStore]. It performs a BFS following directed
// edges outward (FromNodeID → ToNodeID) from nodeID up to depth hops.
// The root node is excluded from the result.
func (s *JSONStore) Dependents(_ context.Context, nodeID string, depth int) ([]Node, error) {
	if err := validateNodeID(nodeID); err != nil {
		return nil, err
	}
	if err := validateDepth(depth); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	visited := map[string]struct{}{nodeID: {}}
	frontier := []string{nodeID}

	for hop := 0; hop < depth && len(frontier) > 0; hop++ {
		var next []string
		for _, id := range frontier {
			for _, e := range s.data.Edges {
				if e.FromNodeID != id {
					continue
				}
				if _, seen := visited[e.ToNodeID]; seen {
					continue
				}
				visited[e.ToNodeID] = struct{}{}
				next = append(next, e.ToNodeID)
			}
		}
		frontier = next
	}

	delete(visited, nodeID) // exclude root
	nodes := make([]Node, 0, len(visited))
	for id := range visited {
		if n, ok := s.data.Nodes[id]; ok {
			nodes = append(nodes, n)
		}
	}
	// Sort for deterministic output in tests.
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].NodeID < nodes[j].NodeID })
	return nodes, nil
}

// BlastRadius implements [GraphStore]. It performs a BFS following edges in
// both directions (inbound and outbound) from nodeID up to depth hops.
// The root node is excluded from the Nodes slice; edges traversed to reach
// the result set are included in Edges.
func (s *JSONStore) BlastRadius(_ context.Context, nodeID string, depth int) (*BlastRadiusResult, error) {
	if err := validateNodeID(nodeID); err != nil {
		return nil, err
	}
	if err := validateDepth(depth); err != nil {
		return nil, err
	}
	start := time.Now()

	s.mu.RLock()
	defer s.mu.RUnlock()

	visitedNodes := map[string]struct{}{nodeID: {}}
	visitedEdges := map[string]struct{}{}
	frontier := []string{nodeID}

	for hop := 0; hop < depth && len(frontier) > 0; hop++ {
		var next []string
		for _, id := range frontier {
			for _, e := range s.data.Edges {
				var neighbor string
				switch {
				case e.FromNodeID == id:
					neighbor = e.ToNodeID
				case e.ToNodeID == id:
					neighbor = e.FromNodeID
				default:
					continue
				}
				visitedEdges[e.EdgeID] = struct{}{}
				if _, seen := visitedNodes[neighbor]; !seen {
					visitedNodes[neighbor] = struct{}{}
					next = append(next, neighbor)
				}
			}
		}
		frontier = next
	}

	delete(visitedNodes, nodeID) // exclude root from node list

	nodes := make([]Node, 0, len(visitedNodes))
	for id := range visitedNodes {
		if n, ok := s.data.Nodes[id]; ok {
			nodes = append(nodes, n)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].NodeID < nodes[j].NodeID })

	edges := make([]Edge, 0, len(visitedEdges))
	for id := range visitedEdges {
		if e, ok := s.data.Edges[id]; ok {
			edges = append(edges, e)
		}
	}

	var repoID string
	if root, ok := s.data.Nodes[nodeID]; ok {
		repoID = root.RepoID
	}

	return &BlastRadiusResult{
		RootNodeID:    nodeID,
		Depth:         depth,
		Nodes:         nodes,
		Edges:         edges,
		RepoID:        repoID,
		QueryDuration: time.Since(start),
	}, nil
}

// Ping implements [GraphStore]. JSONStore is always available after successful
// construction.
func (s *JSONStore) Ping(_ context.Context) error { return nil }

// Close implements [GraphStore]. JSONStore holds no network resources; this
// is a no-op provided for interface compatibility.
func (s *JSONStore) Close() error { return nil }

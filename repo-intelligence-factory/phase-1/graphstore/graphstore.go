// Package graphstore is the seam the whole platform reads/writes the code graph
// through, so the spike <-> AGE <-> fallback swap is one adapter (exit-gate G8).
// Two implementations are provided: JSONStore (the in-memory spike adapter, backs
// the Phase-1 thin slice) and AGEStore (the production openCypher-on-Postgres
// adapter). The Retriever, MCP tools, and the impact-analysis traversals depend on
// this interface only — never on AGE directly.
package graphstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
)

type Node struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Name         string `json:"name"`
	SourceRef    string `json:"source_ref"`
	Confidence   string `json:"confidence"`
	Provenance   string `json:"provenance"`
	IndexVersion string `json:"index_version"`
}

type Edge struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Src        string `json:"src"`
	Dst        string `json:"dst"`
	Confidence string `json:"confidence"`
	SourceRef  string `json:"source_ref"`
}

// Reachable is a node plus how it was reached — depth and the edge tier — so
// impact results are ranked, depth-bounded, and confidence-scored (never a raw
// transitive closure).
type Reachable struct {
	Node       Node
	Depth      int
	ViaType    string
	Confidence string
}

// GraphStore is the only graph contract the rest of the platform knows.
type GraphStore interface {
	UpsertBatch(ctx context.Context, version string, nodes []Node, edges []Edge) error
	PinnedVersion(ctx context.Context) (string, error)
	FindCallers(ctx context.Context, symbolID string) ([]Node, error)
	Dependents(ctx context.Context, symbolID string, maxDepth int) ([]Reachable, error)
	Endpoints(ctx context.Context) ([]Node, error)
	Injects(ctx context.Context, typeID string) ([]Edge, error)
}

// ---- JSONStore: in-memory spike adapter (loads graph.thin-slice.json) --------

type JSONStore struct {
	nodes map[string]Node
	in    map[string][]Edge // dst -> edges
	out   map[string][]Edge // src -> edges
	ver   string
}

func LoadJSON(path string) (*JSONStore, error) {
	var raw struct {
		IndexVersion string `json:"index_version"`
		Nodes        []Node `json:"nodes"`
		Edges        []Edge `json:"edges"`
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	s := &JSONStore{nodes: map[string]Node{}, in: map[string][]Edge{}, out: map[string][]Edge{}, ver: raw.IndexVersion}
	for _, n := range raw.Nodes {
		s.nodes[n.ID] = n
	}
	for _, e := range raw.Edges {
		s.in[e.Dst] = append(s.in[e.Dst], e)
		s.out[e.Src] = append(s.out[e.Src], e)
	}
	return s, nil
}

func (s *JSONStore) PinnedVersion(_ context.Context) (string, error) { return s.ver, nil }

func (s *JSONStore) FindCallers(_ context.Context, id string) ([]Node, error) {
	var out []Node
	for _, e := range s.in[id] {
		if e.Type == "CALLS" {
			out = append(out, s.nodes[e.Src])
		}
	}
	return out, nil
}

func (s *JSONStore) Dependents(_ context.Context, id string, maxDepth int) ([]Reachable, error) {
	seen := map[string]Reachable{}
	type qi struct {
		id string
		d  int
	}
	q := []qi{{id, 0}}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		if cur.d >= maxDepth {
			continue
		}
		for _, e := range s.in[cur.id] {
			if _, ok := seen[e.Src]; ok {
				continue
			}
			seen[e.Src] = Reachable{Node: s.nodes[e.Src], Depth: cur.d + 1, ViaType: e.Type, Confidence: e.Confidence}
			q = append(q, qi{e.Src, cur.d + 1})
		}
	}
	out := make([]Reachable, 0, len(seen))
	for _, r := range seen {
		out = append(out, r)
	}
	return out, nil
}

func (s *JSONStore) Endpoints(_ context.Context) ([]Node, error) {
	var out []Node
	for _, n := range s.nodes {
		if n.Label == "Endpoint" {
			out = append(out, n)
		}
	}
	return out, nil
}

func (s *JSONStore) Injects(_ context.Context, typeID string) ([]Edge, error) {
	var out []Edge
	for _, e := range s.out[typeID] {
		if e.Type == "INJECTS" {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *JSONStore) UpsertBatch(context.Context, string, []Node, []Edge) error { return nil }

// ---- AGEStore: production openCypher-on-Postgres adapter ----------------------
// Skeleton: same interface, backed by AGE. Methods run bounded-depth Cypher via the
// ag_catalog.cypher() SRF (see loader/load_age.py for the load path; the depth caps
// are what keep gate G7 inside budget). Wired in Phase 2 once the full graph lands.

type AGEStore struct {
	db    *sql.DB
	graph string
}

func NewAGEStore(db *sql.DB, graph string) *AGEStore { return &AGEStore{db: db, graph: graph} }

func (a *AGEStore) PinnedVersion(context.Context) (string, error)             { return "", errTODO }
func (a *AGEStore) FindCallers(context.Context, string) ([]Node, error)       { return nil, errTODO }
func (a *AGEStore) Dependents(context.Context, string, int) ([]Reachable, error) { return nil, errTODO }
func (a *AGEStore) Endpoints(context.Context) ([]Node, error)                 { return nil, errTODO }
func (a *AGEStore) Injects(context.Context, string) ([]Edge, error)           { return nil, errTODO }
func (a *AGEStore) UpsertBatch(context.Context, string, []Node, []Edge) error { return errTODO }

// compile-time proof both adapters satisfy the one interface (the G8 guarantee).
var _ GraphStore = (*JSONStore)(nil)
var _ GraphStore = (*AGEStore)(nil)

var errTODO = errTODOImpl("AGEStore: wired in Phase 2 against a live AGE")

type errTODOImpl string

func (e errTODOImpl) Error() string { return string(e) }

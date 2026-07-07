// Package graphstore — AGEStore: a [GraphStore] implementation backed by
// Apache AGE 1.5.0 on PostgreSQL 14/16.
//
// Every Cypher query is executed via the AGE helper function:
//
//	SELECT * FROM ag_catalog.cypher('rif', $$ … $$) AS (col ag_catalog.agtype)
//
// Every pgx connection in the pool runs the following on connect:
//
//	LOAD 'age';
//	SET search_path = ag_catalog, rif_meta, public;
//
// All node_id and edge_id values embedded in Cypher strings are first
// validated against the 64-char hex allowlist so that
// interpolation is injection-safe. Vertex labels and edge labels are
// validated against static allowlists before interpolation.
package graphstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// kindToLabel maps the uppercase Kind values stored in Node.Kind to the
// PascalCase AGE vertex label names created by age_schema.sql.
var kindToLabel = map[string]string{
	"FILE":                "File",
	"CLASS":               "Class",
	"INTERFACE":           "Interface",
	"ENUM":                "Enum",
	"METHOD":              "Method",
	"CONSTRUCTOR":         "Constructor",
	"FIELD":               "Field",
	"RECORD":              "Record",
	"URL_ENDPOINT":        "URL_ENDPOINT",
	"POINTCUT_EXPRESSION": "POINTCUT_EXPRESSION",
}

// validEdgeLabels is the complete set of AGE edge labels defined in
// age_schema.sql. Used as an allowlist before label interpolation.
var validEdgeLabels = map[string]bool{
	"IMPORTS":         true,
	"SAME_FILE_CALLS": true,
	"EXTENDS":         true,
	"IMPLEMENTS":      true,
	"DECLARES_FIELD":  true,
	"INJECTS":         true,
	"PRODUCES":        true,
	"REGISTERS":       true,
	"ADVISES":         true,
	"CALLS_SOAP":      true,
	"CALLS_REST":      true,
}

// agtypeAnnotationRe strips AGE type annotations (e.g. ::vertex, ::int4,
// ::text) that make agtype non-standard JSON. It is applied to the raw
// agtype string before json.Unmarshal.
var agtypeAnnotationRe = regexp.MustCompile(
	`::(?:vertex|edge|path|int2|int4|int8|integer|float|float4|float8|` +
		`bool|boolean|text|numeric|smallint|bigint|real|agtype|` +
		`timestamptz?|date|time|interval|jsonb?)`)

// AGEStore is a [GraphStore] implementation backed by Apache AGE 1.5.0 on
// PostgreSQL 14 or 16. It holds a pgxpool connection pool configured with
// the AfterConnect hook required by AGE.
type AGEStore struct {
	pool *pgxpool.Pool
}

// NewAGEStore opens a connection pool to the Postgres + AGE database at
// databaseURL. If databaseURL is empty the DATABASE_URL environment variable
// is used. Returns an error if the URL is absent, the pool cannot be created,
// or the initial Ping fails.
//
// Pool configuration:
//   - MaxConns: 20
//   - MinConns: 2
//   - MaxConnIdleTime: 5 minutes
//   - MaxConnLifetime: 30 minutes
//   - AfterConnect: LOAD 'age'; SET search_path = ag_catalog, rif_meta, public
func NewAGEStore(ctx context.Context, databaseURL string) (*AGEStore, error) {
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		return nil, fmt.Errorf("graphstore: AGEStore: databaseURL is empty and DATABASE_URL env var is not set")
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("graphstore: AGEStore: parse config: %w", err)
	}

	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = 30 * time.Minute

	// Every new connection must load the AGE shared library and set the
	// search_path so that ag_catalog functions and operators resolve correctly.
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "LOAD 'age'; SET search_path = ag_catalog, rif_meta, public;")
		if err != nil {
			return fmt.Errorf("AGE AfterConnect: %w", err)
		}
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("graphstore: AGEStore: create pool: %w", err)
	}

	s := &AGEStore{pool: pool}
	if err := s.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("graphstore: AGEStore: initial ping: %w", err)
	}
	return s, nil
}

// kindToAGELabel converts a Node.Kind uppercase value to its PascalCase AGE
// vertex label. Returns an error for unknown kinds.
func kindToAGELabel(kind string) (string, error) {
	label, ok := kindToLabel[kind]
	if !ok {
		return "", fmt.Errorf("unknown node kind %q", kind)
	}
	return label, nil
}

// validateEdgeLabel returns an error if label is not in the AGE edge label
// allowlist.
func validateEdgeLabel(label string) error {
	if !validEdgeLabels[label] {
		return fmt.Errorf("unknown edge label %q", label)
	}
	return nil
}

// agtypeToJSON strips AGE type annotations from a raw agtype string so that
// it can be parsed by the standard encoding/json package.
func agtypeToJSON(s string) string {
	return agtypeAnnotationRe.ReplaceAllString(s, "")
}

// agtypeVertexJSON is the intermediate struct used when unmarshalling a
// vertex from its agtype wire representation.
type agtypeVertexJSON struct {
	ID         int64                  `json:"id"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties"`
}

// agtypeEdgeJSON is the intermediate struct used when unmarshalling an edge
// from its agtype wire representation.
type agtypeEdgeJSON struct {
	ID         int64                  `json:"id"`
	Label      string                 `json:"label"`
	StartID    int64                  `json:"start_id"`
	EndID      int64                  `json:"end_id"`
	Properties map[string]interface{} `json:"properties"`
}

// parseAgtypeVertex converts a raw agtype vertex string (e.g.
// {"id":…,"label":"Class","properties":{…}}::vertex) into a Node.
func parseAgtypeVertex(raw string) (Node, error) {
	// Strip the ::vertex (or other trailing type annotation) suffix.
	cleaned := agtypeToJSON(strings.TrimSpace(raw))

	var v agtypeVertexJSON
	if err := json.Unmarshal([]byte(cleaned), &v); err != nil {
		return Node{}, fmt.Errorf("parseAgtypeVertex: unmarshal %q: %w", raw, err)
	}

	p := v.Properties
	node := Node{
		Kind:       v.Label,
		Properties: make(map[string]any),
	}

	// Extract well-known schema fields from the properties map.
	if id, ok := stringProp(p, "node_id"); ok {
		node.NodeID = id
	}
	if r, ok := stringProp(p, "repo_id"); ok {
		node.RepoID = r
	}
	if qn, ok := stringProp(p, "qualified_name"); ok {
		node.QualifiedName = qn
	}
	if k, ok := stringProp(p, "kind"); ok {
		node.Kind = k
	}
	if sr, ok := stringProp(p, "source_ref"); ok {
		node.SourceRef = sr
	}
	if c, ok := stringProp(p, "confidence"); ok {
		node.Confidence = c
	}
	if pp, ok := intProp(p, "phase_populated"); ok {
		node.PhasePopulated = pp
	}
	if o, ok := stringProp(p, "origin"); ok {
		node.Origin = o
	}
	if pk, ok := stringProp(p, "provenance_kind"); ok {
		node.ProvenanceKind = pk
	}

	// Copy remaining properties into the extra Properties map.
	schemaKeys := map[string]bool{
		"node_id": true, "repo_id": true, "qualified_name": true,
		"kind": true, "source_ref": true, "confidence": true,
		"phase_populated": true, "origin": true, "provenance_kind": true,
	}
	for k, val := range p {
		if !schemaKeys[k] {
			node.Properties[k] = val
		}
	}
	if len(node.Properties) == 0 {
		node.Properties = nil
	}
	return node, nil
}

// parseAgtypeEdge converts a raw agtype edge string into an Edge.
func parseAgtypeEdge(raw string) (Edge, error) {
	cleaned := agtypeToJSON(strings.TrimSpace(raw))

	var e agtypeEdgeJSON
	if err := json.Unmarshal([]byte(cleaned), &e); err != nil {
		return Edge{}, fmt.Errorf("parseAgtypeEdge: unmarshal %q: %w", raw, err)
	}

	p := e.Properties
	edge := Edge{Label: e.Label}

	if id, ok := stringProp(p, "edge_id"); ok {
		edge.EdgeID = id
	}
	if fn, ok := stringProp(p, "from_node_id"); ok {
		edge.FromNodeID = fn
	}
	if tn, ok := stringProp(p, "to_node_id"); ok {
		edge.ToNodeID = tn
	}
	if c, ok := stringProp(p, "confidence"); ok {
		edge.Confidence = c
	}
	if sr, ok := stringProp(p, "source_ref"); ok {
		edge.SourceRef = sr
	}
	if t, ok := intProp(p, "tier"); ok {
		edge.Tier = t
	}
	if pp, ok := intProp(p, "phase_populated"); ok {
		edge.PhasePopulated = pp
	}
	if cc, ok := stringProp(p, "completeness_caveat"); ok {
		edge.CompletenessCaveat = cc
	}
	return edge, nil
}

// stringProp extracts a string value from an agtype properties map.
func stringProp(p map[string]interface{}, key string) (string, bool) {
	v, ok := p[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// intProp extracts an integer value from an agtype properties map. AGE
// encodes integers as JSON numbers, which Go unmarshals as float64.
func intProp(p map[string]interface{}, key string) (int, bool) {
	v, ok := p[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	}
	return 0, false
}

// sqlEscape doubles single quotes in s so it can be safely embedded inside a
// SQL single-quoted string literal.
func sqlEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// safePropKeyRe matches property key names that are safe to interpolate into a
// Cypher SET clause (alphanumeric + underscore only).
var safePropKeyRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// buildNodeQuery builds a parameterised Cypher upsert query and a params JSON
// string for a Node. AGE 1.5.0 does not support SET n += $map, so each
// property is set individually via SET n.prop = $p.prop dot-access syntax.
// The params JSON is structured as {"p": {all_node_props_flattened}}.
func buildNodeQuery(n Node, label string) (query string, params string, err error) {
	// Flatten all properties (common + label-specific) into "p".
	flat := map[string]any{
		"node_id":         n.NodeID,
		"repo_id":         n.RepoID,
		"qualified_name":  n.QualifiedName,
		"kind":            n.Kind,
		"source_ref":      n.SourceRef,
		"confidence":      n.Confidence,
		"phase_populated": n.PhasePopulated,
		"origin":          n.Origin,
		"provenance_kind": n.ProvenanceKind,
	}
	for k, v := range n.Properties {
		if safePropKeyRe.MatchString(k) {
			flat[k] = v
		}
	}

	// Build SET clause: one assignment per property key.
	setClauses := make([]string, 0, len(flat))
	for k := range flat {
		setClauses = append(setClauses, fmt.Sprintf("n.%s = $p.%s", k, k))
	}

	paramsJSON, err := json.Marshal(map[string]any{"p": flat})
	if err != nil {
		return "", "", fmt.Errorf("marshal node params: %w", err)
	}

	q := fmt.Sprintf(
		// AGE 1.5.0: third argument MUST be a SQL bind parameter ($1), not a literal.
		`SELECT * FROM ag_catalog.cypher('rif', $$ MERGE (n:%s {node_id: '%s'}) SET %s RETURN n $$, $1::agtype) AS (n ag_catalog.agtype)`,
		label, n.NodeID, strings.Join(setClauses, ", "),
	)
	return q, string(paramsJSON), nil
}

// buildEdgeQuery builds a parameterised Cypher upsert query and params JSON
// for an Edge, using individual SET n.prop = $p.prop clauses (AGE 1.5.0).
func buildEdgeQuery(e Edge) (query string, params string, err error) {
	flat := map[string]any{
		"edge_id":             e.EdgeID,
		"from_node_id":        e.FromNodeID,
		"to_node_id":          e.ToNodeID,
		"confidence":          e.Confidence,
		"source_ref":          e.SourceRef,
		"tier":                e.Tier,
		"phase_populated":     e.PhasePopulated,
		"completeness_caveat": e.CompletenessCaveat,
	}

	setClauses := make([]string, 0, len(flat))
	for k := range flat {
		setClauses = append(setClauses, fmt.Sprintf("e.%s = $p.%s", k, k))
	}

	paramsJSON, err := json.Marshal(map[string]any{"p": flat})
	if err != nil {
		return "", "", fmt.Errorf("marshal edge params: %w", err)
	}

	q := fmt.Sprintf(
		`SELECT * FROM ag_catalog.cypher('rif', $$ `+
			`MATCH (from {node_id: '%s'}), (to {node_id: '%s'}) `+
			`MERGE (from)-[e:%s {edge_id: '%s'}]->(to) SET %s RETURN e `+
			`$$, $1::agtype) AS (e ag_catalog.agtype)`,
		e.FromNodeID, e.ToNodeID, e.Label, e.EdgeID, strings.Join(setClauses, ", "),
	)
	return q, string(paramsJSON), nil
}

// UpsertNode implements [GraphStore]. It merges a node by node_id and sets all
// properties using individual SET n.prop = $p.prop clauses (required by AGE 1.5.0
// which does not support SET n += $map syntax).
func (s *AGEStore) UpsertNode(ctx context.Context, n Node) error {
	if err := validateNodeID(n.NodeID); err != nil {
		return err
	}
	label, err := kindToAGELabel(n.Kind)
	if err != nil {
		return err
	}
	query, params, err := buildNodeQuery(n, label)
	if err != nil {
		return err
	}
	rows, err := s.pool.Query(ctx, query, params)
	if err != nil {
		return fmt.Errorf("upsert node %s: %w", n.NodeID, err)
	}
	rows.Close()
	return rows.Err()
}

// GetNode implements [GraphStore]. It searches all vertex labels for a node
// with the given node_id, returning [ErrNodeNotFound] if absent.
func (s *AGEStore) GetNode(ctx context.Context, nodeID string) (*Node, error) {
	if err := validateNodeID(nodeID); err != nil {
		return nil, err
	}
	// Search without a label to span all vertex tables.
	query := fmt.Sprintf(
		`SELECT * FROM ag_catalog.cypher('rif', $$ MATCH (n {node_id: '%s'}) RETURN n $$) AS (n ag_catalog.agtype)`,
		nodeID,
	)
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get node %s: %w", nodeID, err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("get node %s: %w", nodeID, err)
		}
		return nil, ErrNodeNotFound
	}

	raw, err := scanAgtypeString(rows)
	if err != nil {
		return nil, fmt.Errorf("get node %s: scan: %w", nodeID, err)
	}
	node, err := parseAgtypeVertex(raw)
	if err != nil {
		return nil, fmt.Errorf("get node %s: parse: %w", nodeID, err)
	}
	return &node, nil
}

// UpsertEdge implements [GraphStore]. It matches the from/to vertices by
// node_id and merges the edge by edge_id:
//
//	MATCH (from {node_id:'…'}), (to {node_id:'…'})
//	MERGE (from)-[e:LABEL {edge_id:'…'}]->(to) SET e += $props RETURN e
func (s *AGEStore) UpsertEdge(ctx context.Context, e Edge) error {
	if err := validateEdgeID(e.EdgeID); err != nil {
		return err
	}
	if err := validateEdgeLabel(e.Label); err != nil {
		return err
	}
	if err := validateNodeID(e.FromNodeID); err != nil {
		return fmt.Errorf("from_node_id: %w", err)
	}
	if err := validateNodeID(e.ToNodeID); err != nil {
		return fmt.Errorf("to_node_id: %w", err)
	}

	query, params, err := buildEdgeQuery(e)
	if err != nil {
		return err
	}
	rows, err := s.pool.Query(ctx, query, params)
	if err != nil {
		return fmt.Errorf("upsert edge %s: %w", e.EdgeID, err)
	}
	rows.Close()
	return rows.Err()
}

// BulkLoad implements [GraphStore]. It groups nodes by label and enqueues all
// upserts into a single [pgx.Batch] executed within one transaction, avoiding
// per-record round-trips.
func (s *AGEStore) BulkLoad(ctx context.Context, nodes []Node, edges []Edge) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("bulk load: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := s.BulkLoadTx(ctx, tx, nodes, edges); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// BulkLoadTx performs the AGE graph mutation part of BulkLoad using the caller's
// transaction. It is used by delta loading so graph mutations and version swaps
// commit or roll back together.
func (s *AGEStore) BulkLoadTx(ctx context.Context, tx pgx.Tx, nodes []Node, edges []Edge) error {
	// Validate all inputs before touching the database.
	for i, n := range nodes {
		if err := validateNodeID(n.NodeID); err != nil {
			return fmt.Errorf("node[%d]: %w", i, err)
		}
		if _, err := kindToAGELabel(n.Kind); err != nil {
			return fmt.Errorf("node[%d]: %w", i, err)
		}
	}
	for i, e := range edges {
		if err := validateEdgeID(e.EdgeID); err != nil {
			return fmt.Errorf("edge[%d] edge_id: %w", i, err)
		}
		if err := validateEdgeLabel(e.Label); err != nil {
			return fmt.Errorf("edge[%d]: %w", i, err)
		}
		if err := validateNodeID(e.FromNodeID); err != nil {
			return fmt.Errorf("edge[%d] from_node_id: %w", i, err)
		}
		if err := validateNodeID(e.ToNodeID); err != nil {
			return fmt.Errorf("edge[%d] to_node_id: %w", i, err)
		}
	}

	// Group nodes by label so that all nodes of the same type are adjacent in
	// the batch — this can improve AGE's internal cache locality.
	nodesByLabel := make(map[string][]Node, 8)
	for _, n := range nodes {
		label, _ := kindToAGELabel(n.Kind) // already validated above
		nodesByLabel[label] = append(nodesByLabel[label], n)
	}

	batch := &pgx.Batch{}

	for _, labelNodes := range nodesByLabel {
		for _, n := range labelNodes {
			label, _ := kindToAGELabel(n.Kind)
			q, params, err := buildNodeQuery(n, label)
			if err != nil {
				return err
			}
			batch.Queue(q, params)
		}
	}

	for _, e := range edges {
		q, params, err := buildEdgeQuery(e)
		if err != nil {
			return err
		}
		batch.Queue(q, params)
	}

	br := tx.SendBatch(ctx, batch)
	total := batch.Len()
	for i := 0; i < total; i++ {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return fmt.Errorf("bulk load: batch item %d: %w", i, err)
		}
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("bulk load: close batch: %w", err)
	}

	return nil
}

// DirectCallers implements [GraphStore].
//
// Cypher:
//
//	MATCH (caller)-[e]->(n {node_id:'<nodeID>'})
//	WHERE label(e) IN ['SAME_FILE_CALLS', 'IMPORTS']
//	RETURN caller
func (s *AGEStore) DirectCallers(ctx context.Context, nodeID string) ([]Node, error) {
	if err := validateNodeID(nodeID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		`SELECT * FROM ag_catalog.cypher('rif', $$ `+
			`MATCH (caller)-[e]->(n {node_id: '%s'}) `+
			`WHERE label(e) IN ['SAME_FILE_CALLS', 'IMPORTS'] `+
			`RETURN DISTINCT caller `+
			`$$) AS (caller ag_catalog.agtype)`,
		nodeID,
	)
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("direct callers %s: %w", nodeID, err)
	}
	defer rows.Close()

	return scanNodes(rows)
}

// Dependents implements [GraphStore].
//
// Cypher (depth 1–5 enforced before query):
//
//	MATCH path = (n {node_id:'<nodeID>'})-[*1..<depth+1>]->(dep)
//	RETURN DISTINCT dep
func (s *AGEStore) Dependents(ctx context.Context, nodeID string, depth int) ([]Node, error) {
	if err := validateNodeID(nodeID); err != nil {
		return nil, err
	}
	if err := validateDepth(depth); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		`SELECT * FROM ag_catalog.cypher('rif', $$ `+
			`MATCH path = (n {node_id: '%s'})-[*1..%d]->(dep) `+
			`RETURN DISTINCT dep `+
			`$$) AS (dep ag_catalog.agtype)`,
		nodeID, depth,
	)
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("dependents %s depth %d: %w", nodeID, depth, err)
	}
	defer rows.Close()

	return scanNodes(rows)
}

// BlastRadius implements [GraphStore].
//
// Cypher (bidirectional traversal, depth 1–5):
//
//	MATCH path = (root {node_id:'<nodeID>'})-[*1..<depth+1>]-(affected)
//	RETURN DISTINCT affected
func (s *AGEStore) BlastRadius(ctx context.Context, nodeID string, depth int) (*BlastRadiusResult, error) {
	if err := validateNodeID(nodeID); err != nil {
		return nil, err
	}
	if err := validateDepth(depth); err != nil {
		return nil, err
	}

	start := time.Now()

	nodeQuery := fmt.Sprintf(
		`SELECT * FROM ag_catalog.cypher('rif', $$ `+
			`MATCH path = (root {node_id: '%s'})-[*1..%d]-(affected) `+
			`RETURN DISTINCT affected `+
			`$$) AS (affected ag_catalog.agtype)`,
		nodeID, depth,
	)
	nodeRows, err := s.pool.Query(ctx, nodeQuery)
	if err != nil {
		return nil, fmt.Errorf("blast radius %s depth %d: nodes query: %w", nodeID, depth, err)
	}
	nodes, err := scanNodes(nodeRows)
	nodeRows.Close()
	if err != nil {
		return nil, fmt.Errorf("blast radius %s depth %d: scan nodes: %w", nodeID, depth, err)
	}

	// Second query retrieves the edges traversed in the blast radius.
	edgeQuery := fmt.Sprintf(
		`SELECT * FROM ag_catalog.cypher('rif', $$ `+
			`MATCH path = (root {node_id: '%s'})-[r*1..%d]-(affected) `+
			`UNWIND r AS rel `+
			`RETURN DISTINCT rel `+
			`$$) AS (rel ag_catalog.agtype)`,
		nodeID, depth,
	)
	edgeRows, err := s.pool.Query(ctx, edgeQuery)
	if err != nil {
		return nil, fmt.Errorf("blast radius %s depth %d: edges query: %w", nodeID, depth, err)
	}
	edges, err := scanEdges(edgeRows)
	edgeRows.Close()
	if err != nil {
		return nil, fmt.Errorf("blast radius %s depth %d: scan edges: %w", nodeID, depth, err)
	}

	dur := time.Since(start)

	// Resolve root node's repo_id.
	root, err := s.GetNode(ctx, nodeID)
	var repoID string
	if err == nil {
		repoID = root.RepoID
	}

	return &BlastRadiusResult{
		RootNodeID:    nodeID,
		Depth:         depth,
		Nodes:         nodes,
		Edges:         edges,
		RepoID:        repoID,
		QueryDuration: dur,
	}, nil
}

// Ping implements [GraphStore]. It executes a simple SELECT to verify pool
// and AGE connectivity.
func (s *AGEStore) Ping(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, "SELECT 1")
	return err
}

// Close implements [GraphStore]. It drains and closes the connection pool.
func (s *AGEStore) Close() error {
	s.pool.Close()
	return nil
}

// scanAgtypeString reads the first column of the current row as a string.
// It tries *string first; if that fails it falls back to raw bytes.
func scanAgtypeString(rows pgx.Rows) (string, error) {
	vals := rows.RawValues()
	if len(vals) < 1 {
		return "", fmt.Errorf("no columns in row")
	}
	return string(vals[0]), nil
}

// scanNodes reads all remaining rows from an agtype vertex result set.
func scanNodes(rows pgx.Rows) ([]Node, error) {
	var nodes []Node
	for rows.Next() {
		raw, err := scanAgtypeString(rows)
		if err != nil {
			return nil, err
		}
		node, err := parseAgtypeVertex(raw)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

// scanEdges reads all remaining rows from an agtype edge result set.
func scanEdges(rows pgx.Rows) ([]Edge, error) {
	var edges []Edge
	for rows.Next() {
		raw, err := scanAgtypeString(rows)
		if err != nil {
			return nil, err
		}
		edge, err := parseAgtypeEdge(raw)
		if err != nil {
			return nil, err
		}
		edges = append(edges, edge)
	}
	return edges, rows.Err()
}

// Package graphstore defines the core interface, types, and errors for the
// Repo Intelligence Factory graph layer. Two implementations are provided:
//   - [JSONStore]: in-memory, file-backed store for testing and offline use.
//   - [AGEStore]: Apache AGE on PostgreSQL for production workloads.
//
// All implementations are safe for concurrent use.
package graphstore

import (
	"context"
	"errors"
	"regexp"
	"time"
)

// digestIDRe validates node_id and edge_id values: exactly 64 lowercase
// hexadecimal characters (SHA-256 digest in canonical printed form).
var digestIDRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Sentinel errors returned by all GraphStore implementations.
var (
	// ErrInvalidNodeID is returned when a node_id argument does not conform
	// to the required format: 64 lowercase hexadecimal characters (SHA-256).
	ErrInvalidNodeID = errors.New("node_id must be 64-char lowercase hex SHA-256")

	// ErrInvalidEdgeID is returned when an edge_id argument does not conform
	// to the required format: 64 lowercase hexadecimal characters (SHA-256).
	ErrInvalidEdgeID = errors.New("edge_id must be 64-char lowercase hex SHA-256")

	// ErrDepthOutOfRange is returned when a depth argument is outside [1, 5].
	ErrDepthOutOfRange = errors.New("depth must be between 1 and 5")

	// ErrNodeNotFound is returned by GetNode when no vertex with the given
	// node_id exists in the store.
	ErrNodeNotFound = errors.New("node not found")
)

// Node represents a vertex in the RIF graph. Field names intentionally mirror
// the AGE vertex property names defined in SCHEMA.md §4 so that
// JSON serialisation and Cypher parameterisation are unambiguous.
type Node struct {
	// NodeID is a 64-character lowercase hex SHA-256 digest.
	// Content-addressed: SHA-256(repo_id + NUL + qualified_name + NUL + kind).
	NodeID string // node_id

	// RepoID is the stable repository identifier, e.g.
	// "apm0045942-credit-routing-service".
	RepoID string // repo_id

	// QualifiedName is the fully-qualified Java name. Format varies by Kind —
	// see CODE_MODEL.md §1.2 for the per-type rules.
	QualifiedName string // qualified_name

	// Kind matches the AGE vertex label (uppercased): FILE, CLASS, INTERFACE,
	// ENUM, METHOD, CONSTRUCTOR, FIELD, or RECORD.
	Kind string // kind

	// SourceRef is "repo_id@sha40:path:line" for first-party nodes, or
	// "STUB:external:{fqn}" for external stubs.
	SourceRef string // source_ref

	// Confidence is "exact" for all Phase 1 nodes; "probable" or "inferred"
	// for Phase 2 nodes.
	Confidence string // confidence

	// PhasePopulated is 1 for Phase 1 nodes; 2 for Phase 2 nodes.
	PhasePopulated int // phase_populated

	// Origin is "first_party" for in-repo declarations, or "external_stub"
	// for nodes resolved from external JARs.
	Origin string // origin

	// ProvenanceKind is "file", "generated", or "stub". See SCHEMA.md §4.
	ProvenanceKind string // provenance_kind

	// Properties holds extra label-specific properties (e.g. annotations,
	// param_types, field_type, simple_name). These are stored as individual
	// AGE vertex properties alongside the common schema fields.
	Properties map[string]any
}

// Edge represents a directed relationship in the RIF graph. Field names mirror
// the AGE edge property names defined in SCHEMA.md §5.
type Edge struct {
	// EdgeID is a 64-character lowercase hex SHA-256 digest derived as
	// SHA-256(from_node_id + NUL + label + NUL + to_node_id).
	EdgeID string // edge_id

	// Label is the relationship type. Tier-A (Phase 1): IMPORTS,
	// SAME_FILE_CALLS, EXTENDS, IMPLEMENTS, DECLARES_FIELD.
	// Tier-B/C (Phase 2 stubs): INJECTS, PRODUCES, ADVISES, CALLS_SOAP,
	// CALLS_REST.
	Label string

	// FromNodeID is the node_id of the start vertex.
	FromNodeID string // from_node_id

	// ToNodeID is the node_id of the end vertex.
	ToNodeID string // to_node_id

	// Confidence is "exact" (Tier-A), "probable" (Tier-B), or "inferred"
	// (Tier-C).
	Confidence string // confidence

	// SourceRef is the source reference of the syntactic construct that
	// produced this edge.
	SourceRef string // source_ref

	// Tier is 1 for Tier-A (exact/AST), 2 for Tier-B (probable/annotation),
	// or 3 for Tier-C (inferred).
	Tier int // tier

	// PhasePopulated is 1 for Phase 1 edges; 2 for Phase 2 stubs.
	PhasePopulated int // phase_populated

	// CompletenessCaveat is a mandatory non-empty string describing what this
	// edge type cannot capture. See SCHEMA.md §5 for per-label caveats.
	CompletenessCaveat string // completeness_caveat
}

// BlastRadiusResult is the structured output of a [GraphStore.BlastRadius] query.
type BlastRadiusResult struct {
	// RootNodeID is the node_id passed to BlastRadius.
	RootNodeID string

	// Depth is the maximum hop distance used in the traversal.
	Depth int

	// Nodes is the deduplicated set of nodes reachable within Depth hops via
	// bidirectional edge traversal from RootNodeID. The root node itself is
	// excluded.
	Nodes []Node

	// Edges is the set of edges traversed to produce the Nodes result set.
	Edges []Edge

	// RepoID is taken from the root node's RepoID field.
	RepoID string

	// QueryDuration is the wall-clock duration of the graph query itself,
	// excluding result deserialization.
	QueryDuration time.Duration
}

// GraphStore is the primary interface for reading and writing the RIF graph.
// All implementations must be safe for concurrent use from multiple goroutines.
type GraphStore interface {
	// UpsertNode inserts a new node or merges properties into an existing
	// node with the same node_id. Returns [ErrInvalidNodeID] if n.NodeID
	// is not a 64-character lowercase hex SHA-256 digest.
	UpsertNode(ctx context.Context, n Node) error

	// GetNode returns the node with the given nodeID, or [ErrNodeNotFound] if
	// no such node exists. Returns [ErrInvalidNodeID] for malformed IDs.
	GetNode(ctx context.Context, nodeID string) (*Node, error)

	// UpsertEdge inserts a new edge or merges properties into an existing
	// edge with the same edge_id.
	UpsertEdge(ctx context.Context, e Edge) error

	// BulkLoad inserts or merges all nodes and edges in a single batch
	// operation. Implementations may group nodes by label for efficiency.
	BulkLoad(ctx context.Context, nodes []Node, edges []Edge) error

	// DirectCallers returns all nodes that have a SAME_FILE_CALLS or IMPORTS
	// edge pointing TO the node identified by nodeID.
	DirectCallers(ctx context.Context, nodeID string) ([]Node, error)

	// Dependents returns all nodes reachable from nodeID by following directed
	// edges outward (FromNodeID → ToNodeID) up to depth hops.
	// Returns [ErrDepthOutOfRange] if depth is outside [1, 5].
	Dependents(ctx context.Context, nodeID string, depth int) ([]Node, error)

	// BlastRadius returns all nodes reachable from nodeID via bidirectional
	// edge traversal up to depth hops, together with the traversed edges.
	// Returns [ErrDepthOutOfRange] if depth is outside [1, 5].
	BlastRadius(ctx context.Context, nodeID string, depth int) (*BlastRadiusResult, error)

	// Ping verifies connectivity to the backing store. Returns nil if the
	// store is reachable and ready to serve queries.
	Ping(ctx context.Context) error

	// Close releases all resources held by the store (connections, file
	// handles, etc.). The store must not be used after Close returns.
	Close() error
}

// validateNodeID returns [ErrInvalidNodeID] if id is not a 64-character
// lowercase hex SHA-256 digest.
func validateNodeID(id string) error {
	if !digestIDRe.MatchString(id) {
		return ErrInvalidNodeID
	}
	return nil
}

// validateEdgeID returns [ErrInvalidEdgeID] if id is not a 64-character
// lowercase hexadecimal SHA-256 digest.
func validateEdgeID(id string) error {
	if !digestIDRe.MatchString(id) {
		return ErrInvalidEdgeID
	}
	return nil
}

// validateDepth returns [ErrDepthOutOfRange] if depth is not in [1, 5].
func validateDepth(depth int) error {
	if depth < 1 || depth > 5 {
		return ErrDepthOutOfRange
	}
	return nil
}

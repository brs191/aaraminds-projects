// Package graphstore — LoggingStore: a [GraphStore] decorator that adds
// structured per-operation logging and a separate audit trail for
// [GraphStore.BlastRadius] calls.
package graphstore

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// LoggingStore wraps any [GraphStore] and emits a structured slog record for
// every operation. BlastRadius calls additionally append a JSON line to an
// audit log file (default: ./audit.log; configurable via [LoggingStore.WithAuditPath]).
type LoggingStore struct {
	inner     GraphStore
	logger    *slog.Logger
	auditPath string
	auditMu   sync.Mutex
}

// NewLoggingStore creates a LoggingStore that decorates inner with structured
// logging using logger. The audit log defaults to "./audit.log"; call
// [LoggingStore.WithAuditPath] to override.
func NewLoggingStore(inner GraphStore, logger *slog.Logger) *LoggingStore {
	return &LoggingStore{
		inner:     inner,
		logger:    logger,
		auditPath: "./audit.log",
	}
}

// WithAuditPath sets a non-default path for the BlastRadius audit log file.
// It returns ls so calls can be chained.
func (ls *LoggingStore) WithAuditPath(path string) *LoggingStore {
	ls.auditPath = path
	return ls
}

// UpsertNode implements [GraphStore].
func (ls *LoggingStore) UpsertNode(ctx context.Context, n Node) error {
	start := time.Now()
	err := ls.inner.UpsertNode(ctx, n)
	ls.logger.InfoContext(ctx, "UpsertNode",
		slog.String("op", "UpsertNode"),
		slog.String("node_id", n.NodeID),
		slog.String("kind", n.Kind),
		slog.String("repo_id", n.RepoID),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Any("error", err),
	)
	return err
}

// GetNode implements [GraphStore].
func (ls *LoggingStore) GetNode(ctx context.Context, nodeID string) (*Node, error) {
	start := time.Now()
	node, err := ls.inner.GetNode(ctx, nodeID)
	ls.logger.InfoContext(ctx, "GetNode",
		slog.String("op", "GetNode"),
		slog.String("node_id", nodeID),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Any("error", err),
	)
	return node, err
}

// UpsertEdge implements [GraphStore].
func (ls *LoggingStore) UpsertEdge(ctx context.Context, e Edge) error {
	start := time.Now()
	err := ls.inner.UpsertEdge(ctx, e)
	ls.logger.InfoContext(ctx, "UpsertEdge",
		slog.String("op", "UpsertEdge"),
		slog.String("edge_id", e.EdgeID),
		slog.String("label", e.Label),
		slog.String("from_node_id", e.FromNodeID),
		slog.String("to_node_id", e.ToNodeID),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Any("error", err),
	)
	return err
}

// BulkLoad implements [GraphStore].
func (ls *LoggingStore) BulkLoad(ctx context.Context, nodes []Node, edges []Edge) error {
	start := time.Now()
	err := ls.inner.BulkLoad(ctx, nodes, edges)
	ls.logger.InfoContext(ctx, "BulkLoad",
		slog.String("op", "BulkLoad"),
		slog.Int("node_count", len(nodes)),
		slog.Int("edge_count", len(edges)),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Any("error", err),
	)
	return err
}

// DirectCallers implements [GraphStore].
func (ls *LoggingStore) DirectCallers(ctx context.Context, nodeID string) ([]Node, error) {
	start := time.Now()
	nodes, err := ls.inner.DirectCallers(ctx, nodeID)
	ls.logger.InfoContext(ctx, "DirectCallers",
		slog.String("op", "DirectCallers"),
		slog.String("node_id", nodeID),
		slog.Int("result_count", len(nodes)),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Any("error", err),
	)
	return nodes, err
}

// Dependents implements [GraphStore].
func (ls *LoggingStore) Dependents(ctx context.Context, nodeID string, depth int) ([]Node, error) {
	start := time.Now()
	nodes, err := ls.inner.Dependents(ctx, nodeID, depth)
	ls.logger.InfoContext(ctx, "Dependents",
		slog.String("op", "Dependents"),
		slog.String("node_id", nodeID),
		slog.Int("depth", depth),
		slog.Int("result_count", len(nodes)),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Any("error", err),
	)
	return nodes, err
}

// BlastRadius implements [GraphStore]. In addition to the standard operation
// log line, it appends a JSON audit record to the configured audit log file.
// The audit record includes: time, level, audit:true, op, repo_id,
// root_node_id, depth, result_count, and duration_ms.
func (ls *LoggingStore) BlastRadius(ctx context.Context, nodeID string, depth int) (*BlastRadiusResult, error) {
	start := time.Now()
	result, err := ls.inner.BlastRadius(ctx, nodeID, depth)

	durMs := time.Since(start).Milliseconds()
	resultCount := 0
	var repoID, rootSourceRef string
	if result != nil {
		resultCount = len(result.Nodes)
		repoID = result.RepoID
	}

	// Resolve the root node's source_ref for the operation log.
	if rootNode, getErr := ls.inner.GetNode(ctx, nodeID); getErr == nil {
		rootSourceRef = rootNode.SourceRef
	}

	ls.logger.InfoContext(ctx, "BlastRadius",
		slog.String("op", "BlastRadius"),
		slog.String("node_id", nodeID),
		slog.Int("depth", depth),
		slog.Int("result_count", resultCount),
		slog.Int64("duration_ms", durMs),
		slog.String("root_source_ref", rootSourceRef),
		slog.Any("error", err),
	)

	// Write audit record.
	ls.writeAudit(map[string]any{
		"time":         time.Now().UTC().Format(time.RFC3339Nano),
		"level":        "INFO",
		"audit":        true,
		"op":           "BlastRadius",
		"repo_id":      repoID,
		"root_node_id": nodeID,
		"depth":        depth,
		"result_count": resultCount,
		"duration_ms":  durMs,
	})

	return result, err
}

// Ping implements [GraphStore].
func (ls *LoggingStore) Ping(ctx context.Context) error {
	start := time.Now()
	err := ls.inner.Ping(ctx)
	ls.logger.InfoContext(ctx, "Ping",
		slog.String("op", "Ping"),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Any("error", err),
	)
	return err
}

// Close implements [GraphStore].
func (ls *LoggingStore) Close() error {
	return ls.inner.Close()
}

// writeAudit appends a single JSON line to the audit log file. Errors are
// logged to ls.logger but not propagated (audit failures must not break
// the primary query path).
func (ls *LoggingStore) writeAudit(record map[string]any) {
	b, err := json.Marshal(record)
	if err != nil {
		ls.logger.Error("audit marshal failed", slog.Any("error", err))
		return
	}

	ls.auditMu.Lock()
	defer ls.auditMu.Unlock()

	f, err := os.OpenFile(ls.auditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		ls.logger.Error("audit log open failed",
			slog.String("path", ls.auditPath),
			slog.Any("error", err),
		)
		return
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%s\n", b); err != nil {
		ls.logger.Error("audit log write failed",
			slog.String("path", ls.auditPath),
			slog.Any("error", err),
		)
	}
}

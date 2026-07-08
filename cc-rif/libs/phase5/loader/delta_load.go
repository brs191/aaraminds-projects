package loader

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/aaraminds/rif/graphstore"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FallbackEnqueuer interface {
	EnqueueFullReindex(ctx context.Context, repoID, sha, reason string) error
}

type DeltaLoadRequest struct {
	RepoID           string
	SHA              string
	ExpectedVersion  int
	NewVersion       int
	ExtractorVersion string
	ChangedFiles     []string
	Nodes            []graphstore.Node
	Edges            []graphstore.Edge
}

type DeltaLoader struct {
	pool     *pgxpool.Pool
	graph    graphstore.GraphStore
	fallback FallbackEnqueuer
}

type transactionalBulkLoader interface {
	BulkLoadTx(context.Context, pgx.Tx, []graphstore.Node, []graphstore.Edge) error
}

func NewDeltaLoader(pool *pgxpool.Pool, graph graphstore.GraphStore, fallback FallbackEnqueuer) *DeltaLoader {
	return &DeltaLoader{pool: pool, graph: graph, fallback: fallback}
}

func (l *DeltaLoader) LoadDelta(ctx context.Context, req DeltaLoadRequest) error {
	if strings.TrimSpace(req.RepoID) == "" {
		return errors.New("repo_id is required")
	}
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "rif_delta:"+req.RepoID); err != nil {
		return fmt.Errorf("acquire repo delta lock: %w", err)
	}

	var currentVersion int
	err = tx.QueryRow(ctx,
		`SELECT current_index_version FROM rif_meta.repositories WHERE repo_id = $1 FOR UPDATE`,
		req.RepoID,
	).Scan(&currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("repo %s is not registered", req.RepoID)
	}
	if err != nil {
		return fmt.Errorf("read current version: %w", err)
	}
	if currentVersion != req.ExpectedVersion {
		return l.enqueueFallback(ctx, req, "delta_version_conflict")
	}

	if err := l.deleteChangedGraphData(ctx, tx, req.RepoID, req.ChangedFiles); err != nil {
		return fmt.Errorf("delete changed graph data: %w", err)
	}

	if txGraph, ok := l.graph.(transactionalBulkLoader); ok {
		err = txGraph.BulkLoadTx(ctx, tx, req.Nodes, req.Edges)
	} else {
		err = l.graph.BulkLoad(ctx, req.Nodes, req.Edges)
	}
	if err != nil {
		return fmt.Errorf("bulk load delta: %w", err)
	}

	tag, err := tx.Exec(ctx, `
UPDATE rif_meta.repositories
SET current_sha = $2, current_index_version = $3, updated_at = NOW()
WHERE repo_id = $1 AND current_index_version = $4`,
		req.RepoID, req.SHA, req.NewVersion, req.ExpectedVersion,
	)
	if err != nil {
		return fmt.Errorf("swap version: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return l.enqueueFallback(ctx, req, "delta_swap_conflict")
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO rif_meta.index_versions (repo_id, version, sha, extractor_version) VALUES ($1,$2,$3,$4)`,
		req.RepoID, req.NewVersion, req.SHA, req.ExtractorVersion,
	); err != nil {
		return fmt.Errorf("insert index_versions: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit swap: %w", err)
	}
	return nil
}

func (l *DeltaLoader) enqueueFallback(ctx context.Context, req DeltaLoadRequest, reason string) error {
	if l.fallback == nil {
		return fmt.Errorf("atomic swap failed after retry for repo %s", req.RepoID)
	}
	if err := l.fallback.EnqueueFullReindex(ctx, req.RepoID, req.SHA, reason); err != nil {
		return fmt.Errorf("enqueue full reindex fallback: %w", err)
	}
	return nil
}

func (l *DeltaLoader) deleteChangedGraphData(ctx context.Context, tx pgx.Tx, repoID string, changedFiles []string) error {
	for _, path := range changedFiles {
		normalized := strings.TrimSpace(path)
		if normalized == "" {
			continue
		}
		regex := fmt.Sprintf(`^%s@[0-9a-f]{40}:%s:[0-9]+$`, regexp.QuoteMeta(repoID), regexp.QuoteMeta(normalized))
		regex = cypherStringLiteral(regex)
		qNode := fmt.Sprintf(`SELECT * FROM ag_catalog.cypher('rif', $$ MATCH (n) WHERE n.source_ref =~ '%s' DETACH DELETE n RETURN 1 $$) AS (v ag_catalog.agtype)`, regex)
		if _, err := tx.Exec(ctx, qNode); err != nil {
			return fmt.Errorf("delete nodes for %s: %w", normalized, err)
		}
		qEdge := fmt.Sprintf(`SELECT * FROM ag_catalog.cypher('rif', $$ MATCH ()-[e]-() WHERE e.source_ref =~ '%s' DELETE e RETURN 1 $$) AS (v ag_catalog.agtype)`, regex)
		if _, err := tx.Exec(ctx, qEdge); err != nil {
			return fmt.Errorf("delete edges for %s: %w", normalized, err)
		}
	}
	return nil
}

func cypherStringLiteral(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `'`, `\'`)
}

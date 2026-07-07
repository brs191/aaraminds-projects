package retriever

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PGBackend implements SearchBackend over rif_meta tables in Postgres.
type PGBackend struct {
	pool *pgxpool.Pool
}

// NewPGBackend creates a new Postgres-backed search backend.
func NewPGBackend(pool *pgxpool.Pool) *PGBackend {
	return &PGBackend{pool: pool}
}

func (b *PGBackend) VectorSearch(ctx context.Context, repoID string, embedding []float32, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = defaultSearchK
	}
	vec := vectorLiteral(embedding)
	const q = `
WITH candidates AS (
  SELECT node_id, source_ref, 'exact'::text AS confidence, embedding <=> $2::vector AS distance
  FROM rif_meta.file_nodes
  WHERE repo_id = $1 AND embedding IS NOT NULL
  UNION ALL
  SELECT node_id, source_ref, 'exact'::text AS confidence, embedding <=> $2::vector AS distance
  FROM rif_meta.method_nodes
  WHERE repo_id = $1 AND embedding IS NOT NULL
)
SELECT node_id, source_ref, confidence, distance
FROM candidates
ORDER BY distance ASC, node_id ASC
LIMIT $3;`

	rows, err := b.pool.Query(ctx, q, repoID, vec, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hits := make([]SearchHit, 0, limit)
	for rows.Next() {
		var h SearchHit
		if err := rows.Scan(&h.NodeID, &h.SourceRef, &h.Confidence, &h.Score); err != nil {
			return nil, err
		}
		h.Signal = "vector"
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

func (b *PGBackend) FTSSearch(ctx context.Context, repoID, query string, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = defaultSearchK
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	const q = `
WITH tsq AS (SELECT websearch_to_tsquery('english', $2) AS query)
SELECT node_id, source_ref, confidence, score
FROM (
  SELECT node_id, source_ref, 'exact'::text AS confidence, ts_rank_cd(fts_vector, tsq.query) AS score
  FROM rif_meta.file_nodes, tsq
  WHERE repo_id = $1 AND fts_vector @@ tsq.query
  UNION ALL
  SELECT node_id, source_ref, 'exact'::text AS confidence, ts_rank_cd(fts_vector, tsq.query) AS score
  FROM rif_meta.method_nodes, tsq
  WHERE repo_id = $1 AND fts_vector @@ tsq.query
  UNION ALL
  SELECT node_id, source_ref, 'exact'::text AS confidence, ts_rank_cd(fts_vector, tsq.query) AS score
  FROM rif_meta.class_nodes, tsq
  WHERE repo_id = $1 AND fts_vector @@ tsq.query
) ranked
ORDER BY score DESC, node_id ASC
LIMIT $3;`

	rows, err := b.pool.Query(ctx, q, repoID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hits := make([]SearchHit, 0, limit)
	for rows.Next() {
		var h SearchHit
		if err := rows.Scan(&h.NodeID, &h.SourceRef, &h.Confidence, &h.Score); err != nil {
			return nil, err
		}
		h.Signal = "fts"
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

func vectorLiteral(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.Grow(len(vec) * 8)
	b.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(fmt.Sprintf("%g", v))
	}
	b.WriteByte(']')
	return b.String()
}

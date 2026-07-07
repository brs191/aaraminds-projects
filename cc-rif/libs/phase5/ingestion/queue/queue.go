package queue

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Lane string

const (
	LaneA          Lane = "lane_a"
	LaneB          Lane = "lane_b"
	LaneC          Lane = "lane_c"
	LaneFullReindex Lane = "full_reindex"
)

type Item struct {
	ID        int64
	RepoID    string
	QueuedSHA string
	QueuedAt  time.Time
	Status    string
	Lane      Lane
	BeforeSHA string
	AfterSHA  string
	Attempts  int
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS rif_meta.index_queue (
    id BIGSERIAL PRIMARY KEY,
    repo_id TEXT NOT NULL REFERENCES rif_meta.repositories(repo_id),
    queued_sha CHAR(40) NOT NULL,
    queued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'queued',
    lane TEXT NOT NULL,
    before_sha CHAR(40),
    after_sha CHAR(40),
    attempts INTEGER NOT NULL DEFAULT 0,
    error_message TEXT
);
CREATE INDEX IF NOT EXISTS idx_index_queue_repo_time ON rif_meta.index_queue(repo_id, queued_at DESC);
`
	_, err := s.pool.Exec(ctx, ddl)
	return err
}

func (s *Store) Enqueue(ctx context.Context, repoID, queuedSHA string, lane Lane, beforeSHA, afterSHA string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO rif_meta.index_queue (repo_id, queued_sha, lane, before_sha, after_sha, status) VALUES ($1,$2,$3,$4,$5,'queued')`,
		repoID, queuedSHA, string(lane), nullIfEmpty(beforeSHA), nullIfEmpty(afterSHA),
	)
	if err != nil {
		return fmt.Errorf("enqueue: %w", err)
	}
	return nil
}

func (s *Store) FetchQueued(ctx context.Context, limit int) ([]Item, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, repo_id, queued_sha, queued_at, status, lane, COALESCE(before_sha,''), COALESCE(after_sha,''), attempts
FROM rif_meta.index_queue
WHERE status='queued'
ORDER BY queued_at ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch queued: %w", err)
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		var it Item
		var lane string
		if err := rows.Scan(&it.ID, &it.RepoID, &it.QueuedSHA, &it.QueuedAt, &it.Status, &lane, &it.BeforeSHA, &it.AfterSHA, &it.Attempts); err != nil {
			return nil, err
		}
		it.Lane = Lane(lane)
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Store) MarkStatus(ctx context.Context, id int64, status string, errMsg string) error {
	_, err := s.pool.Exec(ctx, `UPDATE rif_meta.index_queue SET status=$2, error_message=NULLIF($3,'') WHERE id=$1`, id, status, errMsg)
	return err
}

func (s *Store) MarkCoalesced(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `UPDATE rif_meta.index_queue SET status='coalesced' WHERE id = ANY($1)`, ids)
	return err
}

func Coalesce(items []Item, window time.Duration) (dispatch []Item, coalesced []int64) {
	if len(items) == 0 {
		return nil, nil
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].RepoID == items[j].RepoID {
			return items[i].QueuedAt.Before(items[j].QueuedAt)
		}
		return items[i].RepoID < items[j].RepoID
	})

	i := 0
	for i < len(items) {
		repo := items[i].RepoID
		group := []Item{items[i]}
		i++
		for i < len(items) && items[i].RepoID == repo && items[i].QueuedAt.Sub(group[0].QueuedAt) <= window {
			group = append(group, items[i])
			i++
		}
		best := group[len(group)-1]
		bestPri := lanePriority(best.Lane)
		for _, candidate := range group {
			pri := lanePriority(candidate.Lane)
			if pri > bestPri || (pri == bestPri && candidate.QueuedAt.After(best.QueuedAt)) {
				best = candidate
				bestPri = pri
			}
		}
		dispatch = append(dispatch, best)
		for _, g := range group {
			if g.ID != best.ID {
				coalesced = append(coalesced, g.ID)
			}
		}
	}
	return dispatch, coalesced
}

func lanePriority(l Lane) int {
	switch l {
	case LaneFullReindex:
		return 4
	case LaneC:
		return 3
	case LaneA:
		return 2
	case LaneB:
		return 1
	default:
		return 0
	}
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

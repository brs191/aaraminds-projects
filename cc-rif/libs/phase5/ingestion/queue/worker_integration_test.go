package queue

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type countingDispatcher struct {
	count int
}

func (d *countingDispatcher) Dispatch(_ context.Context, _ Item) error {
	d.count++
	return nil
}

func TestWorkerTickCoalescesThreeRowsIntoOneDispatch(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()

	ctx := context.Background()
	mustExec(t, pool, `CREATE SCHEMA IF NOT EXISTS rif_meta`)
	mustExec(t, pool, `
CREATE TABLE IF NOT EXISTS rif_meta.repositories (
	repo_id TEXT PRIMARY KEY,
	clone_url TEXT NOT NULL,
	current_sha CHAR(40),
	current_index_version INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`)
	repoID := "p23_queue_coalesce_repo"
	mustExec(t, pool, `DELETE FROM rif_meta.repositories WHERE repo_id=$1`, repoID)
	mustExec(t, pool, `INSERT INTO rif_meta.repositories(repo_id, clone_url) VALUES($1,$2)`, repoID, "https://example.invalid/repo.git")

	store := NewStore(pool)
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	mustExec(t, pool, `DELETE FROM rif_meta.index_queue WHERE repo_id=$1`, repoID)

	for i := 0; i < 3; i++ {
		if err := store.Enqueue(ctx, repoID, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", LaneA, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}

	dispatcher := &countingDispatcher{}
	worker := NewWorker(store, dispatcher)
	if err := worker.tick(ctx); err != nil {
		t.Fatalf("worker tick: %v", err)
	}
	if dispatcher.count != 1 {
		t.Fatalf("expected 1 dispatch, got %d", dispatcher.count)
	}

	var dispatched, coalesced int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM rif_meta.index_queue WHERE repo_id=$1 AND status='dispatched'`, repoID).Scan(&dispatched); err != nil {
		t.Fatalf("count dispatched: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM rif_meta.index_queue WHERE repo_id=$1 AND status='coalesced'`, repoID).Scan(&coalesced); err != nil {
		t.Fatalf("count coalesced: %v", err)
	}
	if dispatched != 1 || coalesced != 2 {
		t.Fatalf("expected dispatched=1 and coalesced=2, got dispatched=%d coalesced=%d", dispatched, coalesced)
	}
}

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("RIF_PG_INTEGRATION") != "1" {
		t.Skip("set RIF_PG_INTEGRATION=1 to run integration tests")
	}
	dbURL := os.Getenv("RIF_TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres:///postgres?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	return pool
}

func mustExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec failed: %v; sql=%s", err, sql)
	}
}

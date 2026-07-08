package reconcile

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aaraminds/rif/phase5/ingestion/queue"
)

func TestSweepEnqueuesWhenHeadDiverges(t *testing.T) {
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

	store := queue.NewStore(pool)
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("ensure queue schema: %v", err)
	}

	repoPath, headSHA := createGitRepoWithCommit(t)
	repoID := "p23_reconcile_repo"
	mustExec(t, pool, `DELETE FROM rif_meta.index_queue WHERE repo_id=$1`, repoID)
	mustExec(t, pool, `DELETE FROM rif_meta.repositories WHERE repo_id=$1`, repoID)
	mustExec(t, pool, `INSERT INTO rif_meta.repositories(repo_id, clone_url, current_sha) VALUES($1,$2,$3)`, repoID, repoPath, "0000000000000000000000000000000000000000")

	r := NewReconciler(pool, store)
	if err := r.Sweep(ctx); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM rif_meta.index_queue WHERE repo_id=$1 AND lane='full_reindex' AND queued_sha=$2`, repoID, headSHA).Scan(&count); err != nil {
		t.Fatalf("count queue rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 full_reindex queue row, got %d", count)
	}
}

func createGitRepoWithCommit(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("command failed (%v): %v\n%s", args, err, string(out))
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "rif@example.com")
	run("git", "config", "user.name", "RIF Test")
	file := filepath.Join(dir, "README.md")
	if err := os.WriteFile(file, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run("git", "add", "README.md")
	run("git", "commit", "-m", "init")

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rev-parse: %v\n%s", err, string(out))
	}
	return dir, strings.TrimSpace(string(out))
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

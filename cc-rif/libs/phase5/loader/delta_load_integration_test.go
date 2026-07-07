package loader

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/att/rif/graphstore"
)

type fallbackRecorder struct {
	mu    sync.Mutex
	calls int
}

func (f *fallbackRecorder) EnqueueFullReindex(_ context.Context, _, _, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return nil
}

func (f *fallbackRecorder) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type noOpGraph struct{}

func (noOpGraph) UpsertNode(context.Context, graphstore.Node) error { return nil }
func (noOpGraph) GetNode(context.Context, string) (*graphstore.Node, error) {
	return nil, graphstore.ErrNodeNotFound
}
func (noOpGraph) UpsertEdge(context.Context, graphstore.Edge) error                    { return nil }
func (noOpGraph) BulkLoad(context.Context, []graphstore.Node, []graphstore.Edge) error { return nil }
func (noOpGraph) DirectCallers(context.Context, string) ([]graphstore.Node, error)     { return nil, nil }
func (noOpGraph) Dependents(context.Context, string, int) ([]graphstore.Node, error)   { return nil, nil }
func (noOpGraph) BlastRadius(context.Context, string, int) (*graphstore.BlastRadiusResult, error) {
	return nil, nil
}
func (noOpGraph) Ping(context.Context) error { return nil }
func (noOpGraph) Close() error               { return nil }

func TestLoadDeltaIdempotentWithSameChangedFiles(t *testing.T) {
	pool := integrationAgePool(t)
	defer pool.Close()
	ctx := context.Background()
	prepareCoreSchema(t, pool)
	prepareAgeGraph(t, pool)

	repoID := "p23_loader_idem_repo"
	sha1 := "1111111111111111111111111111111111111111"
	sha2 := "2222222222222222222222222222222222222222"
	mustExec(t, pool, `DELETE FROM rif_meta.index_versions WHERE repo_id=$1`, repoID)
	mustExec(t, pool, `DELETE FROM rif_meta.repositories WHERE repo_id=$1`, repoID)
	mustExec(t, pool, `INSERT INTO rif_meta.repositories(repo_id, clone_url, current_sha, current_index_version) VALUES($1,$2,$3,0)`, repoID, "https://example.invalid/repo.git", strings.Repeat("0", 40))

	store, err := graphstore.NewAGEStore(ctx, os.Getenv("RIF_TEST_DATABASE_URL"))
	if err != nil {
		if os.Getenv("RIF_TEST_DATABASE_URL") == "" {
			store, err = graphstore.NewAGEStore(ctx, "postgres:///postgres?sslmode=disable")
		}
	}
	if err != nil {
		t.Fatalf("create age store: %v", err)
	}
	defer store.Close() //nolint:errcheck

	loader := NewDeltaLoader(pool, store, nil)
	req1 := DeltaLoadRequest{
		RepoID:           repoID,
		SHA:              sha1,
		ExpectedVersion:  0,
		NewVersion:       1,
		ExtractorVersion: "test",
		ChangedFiles:     []string{"src/main/java/com/acme/A.java"},
		Nodes: []graphstore.Node{
			{
				NodeID:         strings.Repeat("a", 64),
				RepoID:         repoID,
				QualifiedName:  "src/main/java/com/acme/A.java",
				Kind:           "FILE",
				SourceRef:      repoID + "@" + sha1 + ":src/main/java/com/acme/A.java:1",
				Confidence:     "exact",
				PhasePopulated: 1,
				Origin:         "first_party",
				ProvenanceKind: "file",
			},
			{
				NodeID:         strings.Repeat("b", 64),
				RepoID:         repoID,
				QualifiedName:  "src/main/java/com/acme/B.java",
				Kind:           "FILE",
				SourceRef:      repoID + "@" + sha1 + ":src/main/java/com/acme/A.java:2",
				Confidence:     "exact",
				PhasePopulated: 1,
				Origin:         "first_party",
				ProvenanceKind: "file",
			},
		},
		Edges: []graphstore.Edge{
			{
				EdgeID:             strings.Repeat("c", 64),
				Label:              "IMPORTS",
				FromNodeID:         strings.Repeat("a", 64),
				ToNodeID:           strings.Repeat("b", 64),
				Confidence:         "exact",
				SourceRef:          repoID + "@" + sha1 + ":src/main/java/com/acme/A.java:3",
				Tier:               1,
				PhasePopulated:     1,
				CompletenessCaveat: "test",
			},
		},
	}
	if err := loader.LoadDelta(ctx, req1); err != nil {
		t.Fatalf("load delta first: %v", err)
	}

	req2 := req1
	req2.SHA = sha2
	req2.ExpectedVersion = 1
	req2.NewVersion = 2
	req2.Nodes[0].SourceRef = repoID + "@" + sha2 + ":src/main/java/com/acme/A.java:1"
	req2.Nodes[1].SourceRef = repoID + "@" + sha2 + ":src/main/java/com/acme/A.java:2"
	req2.Edges[0].SourceRef = repoID + "@" + sha2 + ":src/main/java/com/acme/A.java:3"
	if err := loader.LoadDelta(ctx, req2); err != nil {
		t.Fatalf("load delta second: %v", err)
	}

	var fileCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM rif."File"`).Scan(&fileCount); err != nil {
		t.Fatalf("count files: %v", err)
	}
	var importsCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM rif."IMPORTS"`).Scan(&importsCount); err != nil {
		t.Fatalf("count imports: %v", err)
	}
	if fileCount != 2 || importsCount != 1 {
		t.Fatalf("expected stable counts file=2 imports=1, got file=%d imports=%d", fileCount, importsCount)
	}
}

func TestLoadDeltaAtomicSwapRaceOnlyOneSucceeds(t *testing.T) {
	pool := integrationAgePool(t)
	defer pool.Close()
	ctx := context.Background()
	prepareCoreSchema(t, pool)

	repoID := "p23_loader_race_repo"
	sha := "3333333333333333333333333333333333333333"
	mustExec(t, pool, `DELETE FROM rif_meta.index_versions WHERE repo_id=$1`, repoID)
	mustExec(t, pool, `DELETE FROM rif_meta.repositories WHERE repo_id=$1`, repoID)
	mustExec(t, pool, `INSERT INTO rif_meta.repositories(repo_id, clone_url, current_sha, current_index_version) VALUES($1,$2,$3,0)`, repoID, "https://example.invalid/repo.git", strings.Repeat("0", 40))

	fallback := &fallbackRecorder{}
	dl := NewDeltaLoader(pool, noOpGraph{}, fallback)
	req := DeltaLoadRequest{
		RepoID:           repoID,
		SHA:              sha,
		ExpectedVersion:  0,
		NewVersion:       1,
		ExtractorVersion: "test",
		ChangedFiles:     nil,
	}

	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			errs <- dl.LoadDelta(ctx, req)
		}()
	}
	wg.Wait()
	close(errs)

	var failures int
	for err := range errs {
		if err != nil {
			failures++
		}
	}
	if failures != 0 {
		t.Fatalf("expected both calls to return nil with fallback handling, failures=%d", failures)
	}
	if fallback.Calls() != 1 {
		t.Fatalf("expected exactly one fallback enqueue call, got %d", fallback.Calls())
	}

	var version int
	if err := pool.QueryRow(ctx, `SELECT current_index_version FROM rif_meta.repositories WHERE repo_id=$1`, repoID).Scan(&version); err != nil {
		t.Fatalf("read current version: %v", err)
	}
	if version != 1 {
		t.Fatalf("expected current_index_version=1, got %d", version)
	}
	var versionRows int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM rif_meta.index_versions WHERE repo_id=$1`, repoID).Scan(&versionRows); err != nil {
		t.Fatalf("count index_versions: %v", err)
	}
	if versionRows != 1 {
		t.Fatalf("expected exactly 1 committed index_version row, got %d", versionRows)
	}
}

func integrationAgePool(t *testing.T) *pgxpool.Pool {
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

func prepareCoreSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
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
	mustExec(t, pool, `
CREATE TABLE IF NOT EXISTS rif_meta.index_versions (
	repo_id TEXT NOT NULL,
	version INTEGER NOT NULL,
	sha CHAR(40) NOT NULL,
	extractor_version TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (repo_id, version),
	FOREIGN KEY (repo_id) REFERENCES rif_meta.repositories(repo_id) ON DELETE CASCADE
)`)
}

func prepareAgeGraph(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	mustExec(t, pool, `CREATE EXTENSION IF NOT EXISTS age`)
	mustExec(t, pool, `LOAD 'age'`)
	mustExec(t, pool, `SET search_path = ag_catalog, rif_meta, public`)
	mustExec(t, pool, `DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM ag_catalog.ag_graph WHERE name='rif') THEN PERFORM ag_catalog.create_graph('rif'); END IF; END $$;`)
	mustExec(t, pool, `DO $$ DECLARE g oid; BEGIN SELECT graphid INTO g FROM ag_catalog.ag_graph WHERE name='rif'; IF NOT EXISTS (SELECT 1 FROM ag_catalog.ag_label WHERE graph=g AND name='File' AND kind='v') THEN PERFORM ag_catalog.create_vlabel('rif', 'File'); END IF; END $$;`)
	mustExec(t, pool, `DO $$ DECLARE g oid; BEGIN SELECT graphid INTO g FROM ag_catalog.ag_graph WHERE name='rif'; IF NOT EXISTS (SELECT 1 FROM ag_catalog.ag_label WHERE graph=g AND name='IMPORTS' AND kind='e') THEN PERFORM ag_catalog.create_elabel('rif', 'IMPORTS'); END IF; END $$;`)
	mustExec(t, pool, `TRUNCATE TABLE rif."IMPORTS"`)
	mustExec(t, pool, `TRUNCATE TABLE rif."File"`)
}

func mustExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec failed: %v; sql=%s", err, sql)
	}
}

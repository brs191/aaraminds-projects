package app_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aaraminds/transcript-agent/internal/app"
	pgstore "github.com/aaraminds/transcript-agent/internal/store/postgres"
)

func newPostgresTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("POSTGRES_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set POSTGRES_TEST_DATABASE_URL to run Postgres integration tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres admin pool: %v", err)
	}
	t.Cleanup(admin.Close)

	schema := "ta_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+quotedSchema+" CASCADE")
	})

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse postgres test dsn: %v", err)
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect postgres test pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pgstore.Migrate(ctx, pool, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate postgres test schema: %v", err)
	}
	return pool
}

func TestPostgresBackedWorkflowIntegration(t *testing.T) {
	pool := newPostgresTestPool(t)
	e := newEnvWith(t, nil, func(o *app.Options) {
		o.Stores = pgstore.New(pool).Stores()
	})

	job, _, approvedID := runToApproved(e)

	var exports struct {
		Exports []exportResp `json:"exports"`
	}
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/exports", reviewer,
		map[string]any{"formats": []string{"txt"}}, &exports), http.StatusCreated, "create export")
	if len(exports.Exports) != 1 || exports.Exports[0].ValidationStatus != "passed" {
		t.Fatalf("exports %+v", exports.Exports)
	}

	var summary summaryResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/summary", producer,
		map[string]any{}, &summary), http.StatusCreated, "generate summary")
	if summary.SourceTranscriptVersionID != approvedID {
		t.Fatalf("summary source %s, want approved %s", summary.SourceTranscriptVersionID, approvedID)
	}

	var quality qualityResp
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID+"/quality-report", producer, nil, &quality),
		http.StatusOK, "quality report")
	if quality.AverageConfidence == nil || quality.LowConfidenceSegmentCount == 0 {
		t.Fatalf("quality report did not round-trip metrics: %+v", quality)
	}

	var jobNow jobResp
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID, producer, nil, &jobNow),
		http.StatusOK, "get job")
	if jobNow.Status != "exported" {
		t.Fatalf("status %s, want exported", jobNow.Status)
	}

	var auditOut struct {
		Events []struct {
			EventType string `json:"event_type"`
		} `json:"events"`
	}
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID+"/audit", reviewer, nil, &auditOut),
		http.StatusOK, "audit trail")
	seen := map[string]bool{}
	for _, ev := range auditOut.Events {
		seen[ev.EventType] = true
	}
	for _, want := range []string{
		"job.submitted",
		"transcript.approved",
		"tool.export_transcript.completed",
		"tool.generate_summary.completed",
	} {
		if !seen[want] {
			t.Fatalf("audit event %q missing from %+v", want, auditOut.Events)
		}
	}
}

// Package store implements the RunStore — a thin pgx wrapper for reading and
// writing rif_meta.repositories and rif_meta.index_runs.
//
// # Status mapping
//
// The PostgreSQL CHECK constraint on rif_meta.index_runs.status allows only
// 'running', 'completed', 'failed', 'cancelled'. The Ingestion Service tracks
// finer-grained logical stages ('pending', 'cloning', 'extracting', 'loading',
// 'complete', 'failed') inside run_metrics->>'stage'. Callers of [RunStatus]
// receive the logical stage, not the raw DB status.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aaraminds/rif/graphstore"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrRepoNotFound is returned when a repository row does not exist.
var ErrRepoNotFound = errors.New("repo not found")

// ErrRunNotFound is returned when an index_run row does not exist.
var ErrRunNotFound = errors.New("run not found")

// ErrRepoDuplicate is returned when registering a repo that already exists.
var ErrRepoDuplicate = errors.New("repo already registered")

// ErrIndexRunInProgress is returned when a new run is requested while another
// run for the same repo is still running.
var ErrIndexRunInProgress = errors.New("index run already in progress")

// ErrVersionConflict is returned when the live repo version changed before a
// proposed version swap could be committed.
var ErrVersionConflict = errors.New("index version conflict")

// shaPlaceholder is written to index_runs.sha when the caller does not supply
// a SHA at trigger time. It is replaced by the real SHA after git clone resolves
// HEAD. The 40-char zero string satisfies the CHAR(40) NOT NULL constraint.
const shaPlaceholder = "0000000000000000000000000000000000000000"

// RunStatus is the read model returned by [RunStore.GetRunStatus] and
// [RunStore.GetLatestRun]. Stage is the logical stage name derived from
// run_metrics->>'stage'.
type RunStatus struct {
	RunID       string
	RepoID      string
	SHA         string
	Stage       string // pending / cloning / extracting / loading / complete / failed
	NodeCount   *int32
	EdgeCount   *int32
	StartedAt   time.Time
	CompletedAt *time.Time
}

// RepoInfo is the read model returned by [RunStore.GetRepo].
type RepoInfo struct {
	RepoID         string
	CloneURL       string
	CurrentSHA     string
	CurrentVersion int
}

// RunStore wraps a pgxpool.Pool and provides typed access to rif_meta tables.
// All methods are safe for concurrent use.
type RunStore struct {
	pool *pgxpool.Pool
}

// NewRunStore creates a RunStore backed by pool.
func NewRunStore(pool *pgxpool.Pool) *RunStore {
	return &RunStore{pool: pool}
}

// ─── Repository operations ────────────────────────────────────────────────────

// RegisterRepo inserts a new row into rif_meta.repositories.
// Returns [ErrRepoDuplicate] if a repo with the same repo_id already exists.
func (s *RunStore) RegisterRepo(ctx context.Context, repoID, cloneURL string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO rif_meta.repositories (repo_id, clone_url)
		 VALUES ($1, $2)`,
		repoID, cloneURL,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrRepoDuplicate
		}
		return fmt.Errorf("insert repo: %w", err)
	}
	return nil
}

// GetRepo retrieves repo metadata by repo_id.
// Returns [ErrRepoNotFound] if no such repo exists.
func (s *RunStore) GetRepo(ctx context.Context, repoID string) (*RepoInfo, error) {
	var info RepoInfo
	var currentSHA pgtype.Text

	err := s.pool.QueryRow(ctx,
		`SELECT repo_id, clone_url, current_sha, current_index_version
		 FROM rif_meta.repositories
		 WHERE repo_id = $1`,
		repoID,
	).Scan(&info.RepoID, &info.CloneURL, &currentSHA, &info.CurrentVersion)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRepoNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get repo: %w", err)
	}
	if currentSHA.Valid {
		info.CurrentSHA = strings.TrimRight(currentSHA.String, " ")
	}
	return &info, nil
}

// ─── Index run operations ─────────────────────────────────────────────────────

// InsertRun creates a new index_run row with status='running' and stage='pending'.
// If sha is empty the placeholder SHA is used; call [UpdateRunSHA] once the real
// SHA is known.
// Returns the new run_id UUID and the proposed index_version for this run.
func (s *RunStore) InsertRun(ctx context.Context, repoID, sha, extractorVersion string) (runID string, indexVersion int, err error) {
	if sha == "" {
		sha = shaPlaceholder
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("begin insert run tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var currentVersion int
	err = tx.QueryRow(ctx,
		`SELECT current_index_version FROM rif_meta.repositories WHERE repo_id = $1 FOR UPDATE`,
		repoID,
	).Scan(&currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, ErrRepoNotFound
	}
	if err != nil {
		return "", 0, fmt.Errorf("read current version: %w", err)
	}
	indexVersion = currentVersion + 1

	var running bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (
		    SELECT 1 FROM rif_meta.index_runs
		     WHERE repo_id = $1 AND status = 'running'
		)`,
		repoID,
	).Scan(&running); err != nil {
		return "", 0, fmt.Errorf("check running index run: %w", err)
	}
	if running {
		return "", 0, ErrIndexRunInProgress
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO rif_meta.index_runs
		    (repo_id, sha, index_version, extractor_version, status, run_metrics)
		 VALUES ($1, $2, $3, $4, 'running', '{"stage":"pending"}')
		 RETURNING run_id`,
		repoID, sha, indexVersion, extractorVersion,
	).Scan(&runID)
	if err != nil {
		return "", 0, fmt.Errorf("insert run: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", 0, fmt.Errorf("commit insert run: %w", err)
	}
	return runID, indexVersion, nil
}

// UpdateRunStage sets run_metrics->>'stage' to stage without changing the DB
// status column. stage must be one of: pending, cloning, extracting, loading.
func (s *RunStore) UpdateRunStage(ctx context.Context, runID, stage string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE rif_meta.index_runs
		    SET run_metrics = jsonb_set(COALESCE(run_metrics, '{}'), '{stage}', to_jsonb($2::text))
		  WHERE run_id = $1`,
		runID, stage,
	)
	if err != nil {
		return fmt.Errorf("update run stage %s: %w", stage, err)
	}
	return nil
}

// UpdateRunSHA replaces the placeholder SHA with the real commit SHA resolved
// after cloning. sha must be exactly 40 hexadecimal characters.
func (s *RunStore) UpdateRunSHA(ctx context.Context, runID, sha string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE rif_meta.index_runs SET sha = $2 WHERE run_id = $1`,
		runID, sha,
	)
	if err != nil {
		return fmt.Errorf("update run sha: %w", err)
	}
	return nil
}

// CompleteRun marks the run as completed, records counts, and sets stage='complete'.
func (s *RunStore) CompleteRun(ctx context.Context, runID string, nodeCount, edgeCount int) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE rif_meta.index_runs
		    SET status       = 'completed',
		        completed_at = NOW(),
		        node_count   = $2,
		        edge_count   = $3,
		        run_metrics  = jsonb_set(COALESCE(run_metrics, '{}'), '{stage}', '"complete"')
		  WHERE run_id = $1`,
		runID, nodeCount, edgeCount,
	)
	if err != nil {
		return fmt.Errorf("complete run: %w", err)
	}
	return nil
}

// FailRun marks the run as failed and records the error message.
func (s *RunStore) FailRun(ctx context.Context, runID, errMsg string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE rif_meta.index_runs
		    SET status        = 'failed',
		        completed_at  = NOW(),
		        error_message = $2,
		        run_metrics   = jsonb_set(COALESCE(run_metrics, '{}'), '{stage}', '"failed"')
		  WHERE run_id = $1`,
		runID, errMsg,
	)
	if err != nil {
		return fmt.Errorf("fail run: %w", err)
	}
	return nil
}

// GetRunStatus returns the status of a specific run by run_id.
// Returns [ErrRunNotFound] if no such run exists.
func (s *RunStore) GetRunStatus(ctx context.Context, runID string) (*RunStatus, error) {
	return s.scanRunStatus(ctx,
		`SELECT run_id, repo_id, sha, status, node_count, edge_count,
		        started_at, completed_at,
		        COALESCE(run_metrics->>'stage', status) AS stage
		   FROM rif_meta.index_runs
		  WHERE run_id = $1`,
		runID,
	)
}

// GetLatestRun returns the most recent index_run for repoID, ordered by
// started_at DESC. Returns [ErrRunNotFound] if no runs exist for this repo.
func (s *RunStore) GetLatestRun(ctx context.Context, repoID string) (*RunStatus, error) {
	return s.scanRunStatus(ctx,
		`SELECT run_id, repo_id, sha, status, node_count, edge_count,
		        started_at, completed_at,
		        COALESCE(run_metrics->>'stage', status) AS stage
		   FROM rif_meta.index_runs
		  WHERE repo_id = $1
		  ORDER BY started_at DESC
		  LIMIT 1`,
		repoID,
	)
}

// AtomicVersionSwap atomically updates rif_meta.repositories to make the new
// index version live, then inserts an immutable record into index_versions.
// Both operations run inside a single transaction.
func (s *RunStore) AtomicVersionSwap(ctx context.Context, repoID, sha string, newVersion int, extractorVersion string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	expectedVersion := newVersion - 1
	tag, err := tx.Exec(ctx,
		`UPDATE rif_meta.repositories
		    SET current_sha           = $2,
		        current_index_version = $3,
		        updated_at            = NOW()
		  WHERE repo_id = $1
		    AND current_index_version = $4`,
		repoID, sha, newVersion, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("swap version: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrVersionConflict
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO rif_meta.index_versions (repo_id, version, sha, extractor_version)
		 VALUES ($1, $2, $3, $4)`,
		repoID, newVersion, sha, extractorVersion,
	)
	if err != nil {
		return fmt.Errorf("insert index_version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit swap: %w", err)
	}
	return nil
}

type EmbeddedNode struct {
	Node      graphstore.Node
	Embedding []float32
}

func (s *RunStore) UpsertEmbeddings(ctx context.Context, repoID string, indexVersion int, embeddingModel string, nodes []EmbeddedNode) error {
	if len(nodes) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin embedding tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const fileSQL = `
INSERT INTO rif_meta.file_nodes (
    node_id, repo_id, qualified_name, package, line_count, source_ref, index_version, origin, embedding, embedding_model
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::vector, $10)
ON CONFLICT (node_id) DO UPDATE SET
    repo_id = EXCLUDED.repo_id,
    qualified_name = EXCLUDED.qualified_name,
    package = EXCLUDED.package,
    line_count = EXCLUDED.line_count,
    source_ref = EXCLUDED.source_ref,
    index_version = EXCLUDED.index_version,
    origin = EXCLUDED.origin,
    embedding = EXCLUDED.embedding,
    embedding_model = EXCLUDED.embedding_model,
    upserted_at = NOW();`

	const methodSQL = `
INSERT INTO rif_meta.method_nodes (
    node_id, repo_id, qualified_name, simple_name, return_type, visibility, is_static, source_ref, index_version, origin, embedding, embedding_model
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::vector, $12)
ON CONFLICT (node_id) DO UPDATE SET
    repo_id = EXCLUDED.repo_id,
    qualified_name = EXCLUDED.qualified_name,
    simple_name = EXCLUDED.simple_name,
    return_type = EXCLUDED.return_type,
    visibility = EXCLUDED.visibility,
    is_static = EXCLUDED.is_static,
    source_ref = EXCLUDED.source_ref,
    index_version = EXCLUDED.index_version,
    origin = EXCLUDED.origin,
    embedding = EXCLUDED.embedding,
    embedding_model = EXCLUDED.embedding_model,
    upserted_at = NOW();`

	for _, item := range nodes {
		if len(item.Embedding) == 0 {
			continue
		}
		switch item.Node.Kind {
		case "FILE":
			_, err = tx.Exec(ctx, fileSQL,
				item.Node.NodeID,
				repoID,
				item.Node.QualifiedName,
				stringProperty(item.Node.Properties, "package"),
				intProperty(item.Node.Properties, "line_count"),
				item.Node.SourceRef,
				indexVersion,
				nodeOrigin(item.Node.Origin),
				vectorLiteral(item.Embedding),
				embeddingModel,
			)
		case "METHOD", "CONSTRUCTOR":
			_, err = tx.Exec(ctx, methodSQL,
				item.Node.NodeID,
				repoID,
				item.Node.QualifiedName,
				stringValueOrDefault(stringProperty(item.Node.Properties, "simple_name"), item.Node.QualifiedName),
				stringProperty(item.Node.Properties, "return_type"),
				stringProperty(item.Node.Properties, "visibility"),
				boolProperty(item.Node.Properties, "is_static"),
				item.Node.SourceRef,
				indexVersion,
				nodeOrigin(item.Node.Origin),
				vectorLiteral(item.Embedding),
				embeddingModel,
			)
		default:
			continue
		}
		if err != nil {
			return fmt.Errorf("upsert embedding for %s: %w", item.Node.NodeID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit embedding tx: %w", err)
	}
	return nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

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

func stringProperty(props map[string]any, key string) *string {
	if props == nil {
		return nil
	}
	value, ok := props[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return nil
		}
		return &typed
	}
	return nil
}

func intProperty(props map[string]any, key string) *int {
	if props == nil {
		return nil
	}
	value, ok := props[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case int:
		return &typed
	case int32:
		converted := int(typed)
		return &converted
	case int64:
		converted := int(typed)
		return &converted
	case float64:
		converted := int(typed)
		return &converted
	}
	return nil
}

func boolProperty(props map[string]any, key string) *bool {
	if props == nil {
		return nil
	}
	value, ok := props[key]
	if !ok {
		return nil
	}
	typed, ok := value.(bool)
	if !ok {
		return nil
	}
	return &typed
}

func nodeOrigin(origin string) string {
	if strings.TrimSpace(origin) == "" {
		return "first_party"
	}
	return origin
}

func stringValueOrDefault(value *string, fallback string) string {
	if value != nil {
		return *value
	}
	return fallback
}

// scanRunStatus runs a query that returns the run status columns and scans
// the result into a RunStatus. Translates pgx.ErrNoRows to ErrRunNotFound.
func (s *RunStore) scanRunStatus(ctx context.Context, sql string, arg any) (*RunStatus, error) {
	var rs RunStatus
	var rawSHA pgtype.Text
	var nodeCount, edgeCount pgtype.Int4
	var completedAt pgtype.Timestamptz
	var stage string

	err := s.pool.QueryRow(ctx, sql, arg).Scan(
		&rs.RunID,
		&rs.RepoID,
		&rawSHA,
		// status column ignored; we expose stage instead
		new(string),
		&nodeCount,
		&edgeCount,
		&rs.StartedAt,
		&completedAt,
		&stage,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRunNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan run status: %w", err)
	}

	if rawSHA.Valid {
		rs.SHA = strings.TrimRight(rawSHA.String, " ")
	}
	rs.Stage = mapStage(stage)
	if nodeCount.Valid {
		v := nodeCount.Int32
		rs.NodeCount = &v
	}
	if edgeCount.Valid {
		v := edgeCount.Int32
		rs.EdgeCount = &v
	}
	if completedAt.Valid {
		t := completedAt.Time
		rs.CompletedAt = &t
	}
	return &rs, nil
}

// mapStage normalises the raw DB stage/status string to the canonical logical
// stage names exposed in the API.
func mapStage(raw string) string {
	switch raw {
	case "completed":
		return "complete"
	case "running":
		return "running"
	default:
		return raw
	}
}

// isDuplicateKeyError returns true when err is a PostgreSQL unique-violation
// error (SQLSTATE 23505).
func isDuplicateKeyError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aaraminds/rif/phase5/ingestion/diff"
	"github.com/aaraminds/rif/phase5/ingestion/queue"
	"github.com/aaraminds/rif/phase5/loader"
)

type IncrementalService struct {
	indexService *IndexService
	pool         *pgxpool.Pool
	queueStore   *queue.Store
}

func NewIncrementalService(indexService *IndexService, pool *pgxpool.Pool, queueStore *queue.Store) *IncrementalService {
	return &IncrementalService{
		indexService: indexService,
		pool:         pool,
		queueStore:   queueStore,
	}
}

func (s *IncrementalService) TriggerIncremental(ctx context.Context, item queue.Item) (string, error) {
	if item.Lane == queue.LaneFullReindex || item.Lane == queue.LaneC {
		return s.indexService.TriggerIndex(ctx, item.RepoID, item.QueuedSHA)
	}
	if item.Lane != queue.LaneA && item.Lane != queue.LaneB {
		return "", fmt.Errorf("unsupported incremental lane: %s", item.Lane)
	}

	repo, err := s.indexService.store.GetRepo(ctx, item.RepoID)
	if err != nil {
		return "", err
	}

	runID, _, err := s.indexService.store.InsertRun(ctx, item.RepoID, item.QueuedSHA, s.indexService.cfg.ExtractorVersion)
	if err != nil {
		return "", fmt.Errorf("insert incremental run: %w", err)
	}

	s.indexService.logger.InfoContext(ctx, "incremental run queued",
		slog.String("run_id", runID),
		slog.String("repo_id", item.RepoID),
		slog.String("queued_sha", item.QueuedSHA),
		slog.String("lane", string(item.Lane)),
		slog.String("before_sha", item.BeforeSHA),
		slog.String("after_sha", item.AfterSHA),
	)

	go s.runDeltaPipeline(context.Background(), runID, repo.CloneURL, item)
	return runID, nil
}

func (s *IncrementalService) runDeltaPipeline(ctx context.Context, runID, cloneURL string, item queue.Item) {
	cloneDir := filepath.Join(s.indexService.cfg.CloneDir, runID)
	defer func() {
		if err := os.RemoveAll(cloneDir); err != nil {
			s.indexService.logger.Warn("incremental clone dir cleanup failed",
				slog.String("run_id", runID),
				slog.String("dir", cloneDir),
				slog.Any("error", err),
			)
		}
	}()

	if err := s.indexService.store.UpdateRunStage(ctx, runID, "cloning"); err != nil {
		s.indexService.logger.Warn("stage update failed", slog.String("run_id", runID), slog.Any("error", err))
	}
	if err := s.indexService.gitClone(ctx, runID, cloneURL, cloneDir); err != nil {
		s.indexService.failRun(ctx, runID, fmt.Sprintf("clone: %s", err))
		return
	}

	effectiveSHA := selectNonEmpty(item.QueuedSHA, item.AfterSHA, item.BeforeSHA)
	targetSHA := selectNonEmpty(item.AfterSHA, effectiveSHA)
	if targetSHA != "" {
		if err := gitFetchSHA(ctx, cloneDir, targetSHA); err != nil {
			s.indexService.logger.Warn("incremental fetch target sha failed",
				slog.String("run_id", runID),
				slog.String("sha", targetSHA),
				slog.Any("error", err),
			)
		}
		if err := gitCheckoutSHA(ctx, cloneDir, targetSHA); err != nil {
			s.indexService.failRun(ctx, runID, fmt.Sprintf("checkout target sha: %s", err))
			return
		}
	}

	beforeSHA := strings.TrimSpace(item.BeforeSHA)
	afterSHA := strings.TrimSpace(item.AfterSHA)
	if beforeSHA == "" || afterSHA == "" {
		if !s.fallbackToFull(ctx, runID, item.RepoID, effectiveSHA, "missing_diff_bounds") {
			s.indexService.failRun(ctx, runID, "missing diff bounds and full reindex fallback enqueue failed")
		}
		return
	}

	if err := s.indexService.store.UpdateRunStage(ctx, runID, "computing_diff"); err != nil {
		s.indexService.logger.Warn("stage update failed", slog.String("run_id", runID), slog.Any("error", err))
	}
	diffResult, err := diff.Compute(cloneDir, beforeSHA, afterSHA)
	if err != nil {
		if !s.fallbackToFull(ctx, runID, item.RepoID, effectiveSHA, "diff_compute_failed") {
			s.indexService.failRun(ctx, runID, fmt.Sprintf("diff compute failed and fallback enqueue failed: %s", err))
		}
		return
	}
	if diffResult.ForceReindex {
		if !s.fallbackToFull(ctx, runID, item.RepoID, effectiveSHA, "diff_force_reindex") {
			s.indexService.failRun(ctx, runID, "diff force reindex but fallback enqueue failed")
		}
		return
	}

	var changedFiles []string
	switch item.Lane {
	case queue.LaneA:
		changedFiles = diffResult.LaneA
	case queue.LaneB:
		changedFiles = diffResult.LaneB
	}
	if len(changedFiles) == 0 {
		s.completeRunNoop(ctx, runID)
		return
	}
	javaFiles := filterJavaFiles(changedFiles)

	if err := s.indexService.store.UpdateRunStage(ctx, runID, "extracting_changed"); err != nil {
		s.indexService.logger.Warn("stage update failed", slog.String("run_id", runID), slog.Any("error", err))
	}
	outputFile := filepath.Join(cloneDir, "graph.delta.ndjson")
	if len(javaFiles) > 0 {
		if err := s.indexService.runExtractorWithFiles(ctx, runID, item.RepoID, effectiveSHA, cloneDir, outputFile, javaFiles); err != nil {
			if !s.fallbackToFull(ctx, runID, item.RepoID, effectiveSHA, "partial_extract_failed") {
				s.indexService.failRun(ctx, runID, fmt.Sprintf("partial extract failed and fallback enqueue failed: %s", err))
			}
			return
		}
	} else if err := os.WriteFile(outputFile, []byte{}, 0o644); err != nil {
		s.indexService.failRun(ctx, runID, fmt.Sprintf("create empty delta output: %s", err))
		return
	}

	nodes, edges, err := parseNDJSON(outputFile)
	if err != nil {
		if !s.fallbackToFull(ctx, runID, item.RepoID, effectiveSHA, "parse_delta_failed") {
			s.indexService.failRun(ctx, runID, fmt.Sprintf("parse delta failed and fallback enqueue failed: %s", err))
		}
		return
	}

	if err := s.indexService.store.UpdateRunStage(ctx, runID, "loading_delta"); err != nil {
		s.indexService.logger.Warn("stage update failed", slog.String("run_id", runID), slog.Any("error", err))
	}
	repo, err := s.indexService.store.GetRepo(ctx, item.RepoID)
	if err != nil {
		s.indexService.failRun(ctx, runID, fmt.Sprintf("load repo state: %s", err))
		return
	}
	deltaLoader := loader.NewDeltaLoader(s.pool, s.indexService.graph, s)
	if err := deltaLoader.LoadDelta(ctx, loader.DeltaLoadRequest{
		RepoID:           item.RepoID,
		SHA:              effectiveSHA,
		ExpectedVersion:  repo.CurrentVersion,
		NewVersion:       repo.CurrentVersion + 1,
		ExtractorVersion: s.indexService.cfg.ExtractorVersion,
		ChangedFiles:     changedFiles,
		Nodes:            nodes,
		Edges:            edges,
	}); err != nil {
		s.indexService.failRun(ctx, runID, fmt.Sprintf("delta load: %s", err))
		return
	}

	if err := s.indexService.store.CompleteRun(ctx, runID, len(nodes), len(edges)); err != nil {
		s.indexService.logger.Error("complete incremental run record update failed",
			slog.String("run_id", runID),
			slog.Any("error", err),
		)
	}
}

func (s *IncrementalService) EnqueueFullReindex(ctx context.Context, repoID, sha, _ string) error {
	if strings.TrimSpace(sha) == "" {
		return fmt.Errorf("full reindex fallback requires sha for repo %s", repoID)
	}
	return s.queueStore.Enqueue(ctx, repoID, sha, queue.LaneFullReindex, "", sha)
}

func (s *IncrementalService) fallbackToFull(ctx context.Context, runID, repoID, sha, reason string) bool {
	if strings.TrimSpace(sha) == "" {
		s.indexService.logger.Error("incremental fallback enqueue skipped due empty sha",
			slog.String("run_id", runID),
			slog.String("repo_id", repoID),
			slog.String("reason", reason),
		)
		return false
	}
	if err := s.queueStore.Enqueue(ctx, repoID, sha, queue.LaneFullReindex, "", sha); err != nil {
		s.indexService.logger.Error("incremental fallback enqueue failed",
			slog.String("run_id", runID),
			slog.String("repo_id", repoID),
			slog.String("sha", sha),
			slog.String("reason", reason),
			slog.Any("error", err),
		)
		return false
	}
	s.indexService.failRun(ctx, runID, fmt.Sprintf("incremental fallback to full reindex (%s)", reason))
	return true
}

func (s *IncrementalService) completeRunNoop(ctx context.Context, runID string) {
	if err := s.indexService.store.CompleteRun(ctx, runID, 0, 0); err != nil {
		s.indexService.logger.Error("complete noop incremental run failed",
			slog.String("run_id", runID),
			slog.Any("error", err),
		)
	}
}

func gitFetchSHA(ctx context.Context, cloneDir, sha string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", cloneDir, "fetch", "--depth=1", "origin", sha)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch %s failed: %w (%s)", sha, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitCheckoutSHA(ctx context.Context, cloneDir, sha string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", cloneDir, "checkout", "--detach", sha)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s failed: %w (%s)", sha, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func filterJavaFiles(files []string) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(file)), ".java") {
			out = append(out, file)
		}
	}
	return out
}

func selectNonEmpty(values ...string) string {
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v != "" {
			return v
		}
	}
	return ""
}

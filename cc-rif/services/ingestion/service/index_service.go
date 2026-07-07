// Package service implements IndexService — the orchestrator that drives the
// clone → extract → load → swap pipeline for a single indexing run.
//
// Pipeline is non-blocking: [IndexService.TriggerIndex] inserts the run record
// and returns the run_id immediately; the four-stage pipeline runs in a
// dedicated goroutine using a background context so the caller's request
// context cancellation does not abort an in-progress run.
package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/att/rif/graphstore"
	"github.com/att/rif/ingestion/cloneurl"
	"github.com/att/rif/ingestion/config"
	"github.com/att/rif/ingestion/store"
)

// sourceRefRe validates first-party file source_refs: repo@sha40:path:line
// This is the same pattern used by load_graph.go and provenance_check.py.
var sourceRefRe = regexp.MustCompile(`^[^@]+@[0-9a-f]{40}:.+:[1-9][0-9]*$`)
var commitSHARe = regexp.MustCompile(`^[0-9a-f]{40}$`)

// IndexService orchestrates the four-stage indexing pipeline:
// clone → extract → load → swap.
//
// Each pipeline run executes in its own goroutine with a background context.
// Errors at any stage transition the run to 'failed' and leave the previous
// index version live.
type IndexService struct {
	store    *store.RunStore
	graph    graphstore.GraphStore
	cfg      *config.Config
	logger   *slog.Logger
	embedder *embeddingClient
}

// NewIndexService constructs an IndexService with the supplied dependencies.
func NewIndexService(rs *store.RunStore, gs graphstore.GraphStore, cfg *config.Config, logger *slog.Logger) *IndexService {
	return &IndexService{
		store:    rs,
		graph:    gs,
		cfg:      cfg,
		logger:   logger,
		embedder: newEmbeddingClient(cfg.EmbeddingServiceURL),
	}
}

// TriggerIndex registers a new index run and starts the pipeline asynchronously.
// It returns the run_id immediately; callers poll GET /repos/{repoID}/status
// to observe progress.
//
// If sha is empty the pipeline resolves it from git HEAD after cloning and
// updates the run record before proceeding to extraction.
func (s *IndexService) TriggerIndex(ctx context.Context, repoID, sha string) (string, error) {
	repo, err := s.store.GetRepo(ctx, repoID)
	if err != nil {
		return "", err
	}

	runID, indexVersion, err := s.store.InsertRun(ctx, repoID, sha, s.cfg.ExtractorVersion)
	if err != nil {
		return "", fmt.Errorf("insert run: %w", err)
	}

	s.logger.InfoContext(ctx, "index run queued",
		slog.String("run_id", runID),
		slog.String("repo_id", repoID),
		slog.String("sha", sha),
		slog.Int("proposed_version", indexVersion),
	)

	// Pipeline runs in a goroutine with a background context so it survives
	// the HTTP request context being cancelled.
	go s.runPipeline(context.Background(), runID, repoID, repo.CloneURL, sha, indexVersion)

	return runID, nil
}

// runPipeline executes the four-stage pipeline for a single run. It is always
// called in a goroutine. Clone directory cleanup is deferred so it runs on
// both success and failure paths.
func (s *IndexService) runPipeline(ctx context.Context, runID, repoID, cloneURL, sha string, indexVersion int) {
	cloneDir := filepath.Join(s.cfg.CloneDir, runID)

	if s.embedder.enabled() {
		health, err := s.embedder.Health(ctx)
		if err != nil {
			s.failRun(ctx, runID, fmt.Sprintf("embedding health: %s", err))
			return
		}
		s.logger.Info("embedding service healthy",
			slog.String("run_id", runID),
			slog.String("repo_id", repoID),
			slog.String("model", health.Model),
			slog.Int("dim", health.Dim),
		)
	}

	// Cleanup clone dir on exit regardless of outcome.
	defer func() {
		if err := os.RemoveAll(cloneDir); err != nil {
			s.logger.Warn("clone dir cleanup failed",
				slog.String("run_id", runID),
				slog.String("dir", cloneDir),
				slog.Any("error", err),
			)
		}
	}()

	// ── Stage 1: clone ────────────────────────────────────────────────────────
	if err := s.store.UpdateRunStage(ctx, runID, "cloning"); err != nil {
		s.logger.Warn("stage update failed", slog.String("run_id", runID), slog.Any("error", err))
	}

	if _, err := cloneurl.Validate(cloneURL, s.cfg.AllowedCloneHosts); err != nil {
		s.failRun(ctx, runID, fmt.Sprintf("clone url validation: %s", err))
		return
	}
	if err := s.gitClone(ctx, runID, cloneURL, cloneDir); err != nil {
		s.failRun(ctx, runID, fmt.Sprintf("clone: %s", err))
		return
	}

	if sha != "" {
		sha = strings.ToLower(strings.TrimSpace(sha))
		if !commitSHARe.MatchString(sha) {
			s.failRun(ctx, runID, fmt.Sprintf("invalid sha: %q", sha))
			return
		}
		if err := s.gitCheckoutSHA(ctx, runID, cloneDir, sha); err != nil {
			s.failRun(ctx, runID, fmt.Sprintf("checkout sha: %s", err))
			return
		}
	} else {
		resolved, err := s.gitRevParse(ctx, runID, cloneDir)
		if err != nil {
			s.failRun(ctx, runID, fmt.Sprintf("resolve sha: %s", err))
			return
		}
		sha = resolved
		if err := s.store.UpdateRunSHA(ctx, runID, sha); err != nil {
			s.logger.Warn("update run sha failed", slog.String("run_id", runID), slog.Any("error", err))
		}
	}

	s.logger.Info("clone complete",
		slog.String("run_id", runID),
		slog.String("repo_id", repoID),
		slog.String("sha", sha),
		slog.String("clone_dir", cloneDir),
	)

	// ── Stage 2: extract ─────────────────────────────────────────────────────
	if err := s.store.UpdateRunStage(ctx, runID, "extracting"); err != nil {
		s.logger.Warn("stage update failed", slog.String("run_id", runID), slog.Any("error", err))
	}

	outputFile := filepath.Join(cloneDir, "graph.ndjson")
	if err := s.runExtractor(ctx, runID, repoID, sha, cloneDir, outputFile); err != nil {
		s.failRun(ctx, runID, fmt.Sprintf("extract: %s", err))
		return
	}
	if err := s.runPhase2Extractors(ctx, runID, repoID, sha, cloneDir, outputFile); err != nil {
		s.failRun(ctx, runID, fmt.Sprintf("extract phase2: %s", err))
		return
	}

	// ── Stage 3: load ─────────────────────────────────────────────────────────
	if err := s.store.UpdateRunStage(ctx, runID, "loading"); err != nil {
		s.logger.Warn("stage update failed", slog.String("run_id", runID), slog.Any("error", err))
	}

	nodes, edges, err := parseNDJSON(outputFile)
	if err != nil {
		s.failRun(ctx, runID, fmt.Sprintf("parse ndjson: %s", err))
		return
	}

	s.logger.Info("parsed ndjson",
		slog.String("run_id", runID),
		slog.Int("nodes", len(nodes)),
		slog.Int("edges", len(edges)),
	)

	embeddingModel, embeddedNodes, err := s.embedNodes(ctx, repoID, indexVersion, nodes)
	if err != nil {
		s.failRun(ctx, runID, fmt.Sprintf("embed nodes: %s", err))
		return
	}
	if len(embeddedNodes) > 0 {
		s.logger.Info("embeddings persisted",
			slog.String("run_id", runID),
			slog.String("embedding_model", embeddingModel),
			slog.Int("embedded_nodes", len(embeddedNodes)),
		)
	}

	// ── B2: Degenerate-run guard ─────────────────────────────────────────────
	// An empty extraction (extractor crash or total parse failure) must never
	// commit a version swap — that would advance the live pointer to a SHA with
	// zero indexed data while the old graph nodes stay orphaned in AGE.
	//
	// Hard minimum: 0 nodes is always a fatal degenerate run.
	// Configurable minimums: MIN_INDEX_NODE_COUNT / MIN_INDEX_EDGE_COUNT let
	// operators set repo-specific floors (default 0 = disabled).
	if len(nodes) == 0 {
		s.failRun(ctx, runID, "degenerate run: extractor produced 0 nodes — refusing version swap to protect existing graph")
		return
	}
	if s.cfg.MinIndexNodeCount > 0 && len(nodes) < s.cfg.MinIndexNodeCount {
		s.failRun(ctx, runID, fmt.Sprintf("degenerate run: node count %d is below minimum %d — refusing version swap", len(nodes), s.cfg.MinIndexNodeCount))
		return
	}
	if s.cfg.MinIndexEdgeCount > 0 && len(edges) < s.cfg.MinIndexEdgeCount {
		s.failRun(ctx, runID, fmt.Sprintf("degenerate run: edge count %d is below minimum %d — refusing version swap", len(edges), s.cfg.MinIndexEdgeCount))
		return
	}

	if err := s.graph.BulkLoad(ctx, nodes, edges); err != nil {
		s.failRun(ctx, runID, fmt.Sprintf("bulk load: %s", err))
		return
	}

	// ── Stage 4: swap ─────────────────────────────────────────────────────────
	if err := s.store.AtomicVersionSwap(ctx, repoID, sha, indexVersion, s.cfg.ExtractorVersion); err != nil {
		s.failRun(ctx, runID, fmt.Sprintf("version swap: %s", err))
		return
	}

	if err := s.store.CompleteRun(ctx, runID, len(nodes), len(edges)); err != nil {
		// Run is complete in the graph even if we fail to record it; log but
		// do not treat as a pipeline failure — the swap already committed.
		s.logger.Error("complete run record update failed",
			slog.String("run_id", runID),
			slog.Any("error", err),
		)
		return
	}

	s.logger.Info("index run complete",
		slog.String("run_id", runID),
		slog.String("repo_id", repoID),
		slog.String("sha", sha),
		slog.Int("version", indexVersion),
		slog.Int("nodes", len(nodes)),
		slog.Int("edges", len(edges)),
	)
}

func (s *IndexService) embedNodes(ctx context.Context, repoID string, indexVersion int, nodes []graphstore.Node) (string, []store.EmbeddedNode, error) {
	if !s.embedder.enabled() {
		return "", nil, nil
	}

	health, err := s.embedder.Health(ctx)
	if err != nil {
		return "", nil, err
	}

	items := make([]embeddingItem, 0, len(nodes))
	nodeByID := make(map[string]graphstore.Node, len(nodes))
	for _, node := range nodes {
		text, ok := embeddingText(node)
		if !ok {
			continue
		}
		items = append(items, embeddingItem{NodeID: node.NodeID, Text: text})
		nodeByID[node.NodeID] = node
	}
	if len(items) == 0 {
		return health.Model, nil, nil
	}

	batchSize := s.cfg.EmbeddingBatchSize
	if batchSize <= 0 {
		batchSize = 32
	}

	embedded := make([]store.EmbeddedNode, 0, len(items))
	for start := 0; start < len(items); start += batchSize {
		end := start + batchSize
		if end > len(items) {
			end = len(items)
		}
		results, err := s.embedder.Embed(ctx, items[start:end])
		if err != nil {
			return "", nil, err
		}
		for _, result := range results {
			node, ok := nodeByID[result.NodeID]
			if !ok {
				continue
			}
			embedded = append(embedded, store.EmbeddedNode{
				Node:      node,
				Embedding: result.Embedding,
			})
		}
	}

	if err := s.store.UpsertEmbeddings(ctx, repoID, indexVersion, health.Model, embedded); err != nil {
		return "", nil, err
	}
	return health.Model, embedded, nil
}

func embeddingText(node graphstore.Node) (string, bool) {
	if node.Origin == "external_stub" {
		return "", false
	}
	parts := []string{node.QualifiedName}
	switch node.Kind {
	case "FILE":
		if pkg, ok := node.Properties["package"].(string); ok && strings.TrimSpace(pkg) != "" {
			parts = append(parts, pkg)
		}
	case "METHOD", "CONSTRUCTOR":
		if simple, ok := node.Properties["simple_name"].(string); ok && strings.TrimSpace(simple) != "" {
			parts = append(parts, simple)
		}
		if ret, ok := node.Properties["return_type"].(string); ok && strings.TrimSpace(ret) != "" {
			parts = append(parts, ret)
		}
	default:
		return "", false
	}
	if strings.TrimSpace(node.SourceRef) != "" {
		parts = append(parts, node.SourceRef)
	}
	return strings.Join(parts, "\n"), true
}

// ─── Stage implementations ────────────────────────────────────────────────────

// gitClone runs git clone --depth=1 <cloneURL> <cloneDir>.
func (s *IndexService) gitClone(ctx context.Context, runID, cloneURL, cloneDir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", cloneURL, cloneDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("exit %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// gitRevParse resolves the current HEAD commit SHA in cloneDir.
// Returns the 40-character hex SHA-1.
func (s *IndexService) gitRevParse(ctx context.Context, runID, cloneDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", cloneDir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git rev-parse HEAD exited %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	sha := strings.TrimSpace(string(out))
	if len(sha) != 40 {
		return "", fmt.Errorf("unexpected sha length %d: %q", len(sha), sha)
	}
	return sha, nil
}

func (s *IndexService) gitCheckoutSHA(ctx context.Context, runID, cloneDir, sha string) error {
	if err := s.gitCatFileCommit(ctx, cloneDir, sha); err != nil {
		cmd := exec.CommandContext(ctx, "git", "-C", cloneDir, "fetch", "--depth=1", "origin", sha)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if fetchErr := cmd.Run(); fetchErr != nil {
			return fmt.Errorf("fetch %s: %w: %s", sha, fetchErr, strings.TrimSpace(stderr.String()))
		}
	}

	cmd := exec.CommandContext(ctx, "git", "-C", cloneDir, "checkout", "--detach", sha)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("checkout %s: %w: %s", sha, err, strings.TrimSpace(stderr.String()))
	}
	resolved, err := s.gitRevParse(ctx, runID, cloneDir)
	if err != nil {
		return err
	}
	if resolved != sha {
		return fmt.Errorf("checkout resolved %s, expected %s", resolved, sha)
	}
	return nil
}

func (s *IndexService) gitCatFileCommit(ctx context.Context, cloneDir, sha string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", cloneDir, "cat-file", "-e", sha+"^{commit}")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cat-file %s: %w: %s", sha, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// runExtractor invokes the JavaParser extractor JAR as a subprocess.
// The extractor must exit 0; any non-zero exit is treated as a pipeline failure.
// Stdout and stderr are captured and logged at appropriate levels.
func (s *IndexService) runExtractor(ctx context.Context, runID, repoID, sha, cloneDir, outputFile string) error {
	return s.runExtractorWithFiles(ctx, runID, repoID, sha, cloneDir, outputFile, nil)
}

func (s *IndexService) runExtractorWithFiles(
	ctx context.Context,
	runID, repoID, sha, cloneDir, outputFile string,
	files []string,
) error {
	cmd := exec.CommandContext(ctx, "java", "-jar", s.cfg.ExtractorJarPath,
		"--repo-path", cloneDir,
		"--repo-id", repoID,
		"--sha", sha,
		"--output", outputFile,
		"--skip-tests",
	)
	if len(files) > 0 {
		cmd.Args = append(cmd.Args, "--files", strings.Join(files, ","))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Always log extractor output — useful for diagnosing extraction quality.
	if stdout.Len() > 0 {
		s.logger.Info("extractor stdout",
			slog.String("run_id", runID),
			slog.String("output", stdout.String()),
		)
	}
	if stderr.Len() > 0 {
		s.logger.Info("extractor stderr",
			slog.String("run_id", runID),
			slog.String("output", stderr.String()),
		)
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("extractor exited %d", exitErr.ExitCode())
		}
		return fmt.Errorf("extractor: %w", err)
	}

	// G5: Parse extractor metrics and fail if provenance gaps detected.
	// Extractors output metrics as JSON to stderr.
	if stderr.Len() > 0 {
		metricsErr := s.checkExtractorMetrics(runID, stderr.Bytes())
		if metricsErr != nil {
			return metricsErr
		}
	}

	return nil
}

// checkExtractorMetrics parses extractor stderr metrics and fails the run
// if provenance_gap_count > 0 (G5: provenanceGapCount must fail ingestion).
// Metrics JSON is written to the last line of stderr by the extractor.
func (s *IndexService) checkExtractorMetrics(runID string, stderrBytes []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(stderrBytes))
	var lastLine string
	for scanner.Scan() {
		lastLine = scanner.Text()
	}

	if lastLine == "" {
		return nil
	}

	var metrics map[string]interface{}
	if err := json.Unmarshal([]byte(lastLine), &metrics); err != nil {
		// Not JSON — likely debug output. Log and continue.
		s.logger.Debug("extractor metrics not JSON",
			slog.String("run_id", runID),
			slog.String("line", lastLine),
		)
		return nil
	}

	// Check provenance_gap_count from metrics.
	if gapCount, ok := metrics["provenance_gap_count"]; ok {
		if gap, ok := gapCount.(float64); ok && gap > 0 {
			return fmt.Errorf("extraction failed provenance gate: provenance_gap_count=%d > 0", int(gap))
		}
	}

	return nil
}

func (s *IndexService) runPhase2Extractors(ctx context.Context, runID, repoID, sha, cloneDir, baseOutputFile string) error {
	if !s.cfg.Phase2ExtractorsEnabled {
		return nil
	}

	sourceRoot := filepath.Join(cloneDir, s.cfg.Phase2SourceRoot)
	info, err := os.Stat(sourceRoot)
	if err != nil {
		return fmt.Errorf("phase2 source root %q: %w", sourceRoot, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("phase2 source root %q is not a directory", sourceRoot)
	}

	auxDir := filepath.Join(cloneDir, "phase2-ndjson")
	if err := os.MkdirAll(auxDir, 0o755); err != nil {
		return fmt.Errorf("create phase2 output dir: %w", err)
	}

	type phase2Extractor struct {
		name   string
		jar    string
		output string
	}

	extractors := []phase2Extractor{
		{name: "di", jar: s.cfg.Phase2DiJarPath, output: filepath.Join(auxDir, "di.ndjson")},
		{name: "aop", jar: s.cfg.Phase2AopJarPath, output: filepath.Join(auxDir, "aop.ndjson")},
		{name: "crossservice", jar: s.cfg.Phase2CrossServiceJarPath, output: filepath.Join(auxDir, "crossservice.ndjson")},
	}

	for _, ex := range extractors {
		if err := s.runPhase2Extractor(ctx, runID, ex.name, ex.jar, repoID, sha, sourceRoot, ex.output); err != nil {
			return err
		}
	}

	for _, ex := range extractors {
		if err := appendFile(baseOutputFile, ex.output); err != nil {
			return fmt.Errorf("merge %s output: %w", ex.name, err)
		}
	}

	s.logger.Info("phase2 extractor outputs merged",
		slog.String("run_id", runID),
		slog.String("base_output", baseOutputFile),
	)
	return nil
}

func (s *IndexService) runPhase2Extractor(
	ctx context.Context,
	runID, name, jarPath, repoID, sha, sourceRoot, outputFile string,
) error {
	cmd := exec.CommandContext(ctx, "java", "-jar", jarPath,
		"--repo-id", repoID,
		"--sha", sha,
		"--source-root", sourceRoot,
		"--output", outputFile,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if stdout.Len() > 0 {
		s.logger.Info("phase2 extractor stdout",
			slog.String("run_id", runID),
			slog.String("extractor", name),
			slog.String("output", stdout.String()),
		)
	}
	if stderr.Len() > 0 {
		s.logger.Info("phase2 extractor stderr",
			slog.String("run_id", runID),
			slog.String("extractor", name),
			slog.String("output", stderr.String()),
		)
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("phase2 extractor %s exited %d", name, exitErr.ExitCode())
		}
		return fmt.Errorf("phase2 extractor %s: %w", name, err)
	}
	return nil
}

func appendFile(dstPath, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src %q: %w", srcPath, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_RDWR|os.O_APPEND, 0)
	if err != nil {
		return fmt.Errorf("open dst %q: %w", dstPath, err)
	}
	defer dst.Close()

	srcInfo, err := src.Stat()
	if err != nil {
		return fmt.Errorf("stat src %q: %w", srcPath, err)
	}
	if srcInfo.Size() == 0 {
		return nil
	}

	dstInfo, err := dst.Stat()
	if err != nil {
		return fmt.Errorf("stat dst %q: %w", dstPath, err)
	}
	if dstInfo.Size() > 0 {
		buf := make([]byte, 1)
		if _, err := dst.ReadAt(buf, dstInfo.Size()-1); err != nil {
			return fmt.Errorf("read dst tail %q: %w", dstPath, err)
		}
		if buf[0] != '\n' {
			if _, err := dst.Write([]byte{'\n'}); err != nil {
				return fmt.Errorf("append newline to %q: %w", dstPath, err)
			}
		}
	}

	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("copy %q -> %q: %w", srcPath, dstPath, err)
	}
	return nil
}

// ─── NDJSON parsing ───────────────────────────────────────────────────────────

// nodeKnownFields is the set of top-level NDJSON keys that map directly to
// graphstore.Node fields. All other keys are placed in Node.Properties.
var nodeKnownFields = map[string]bool{
	"record_type": true, "node_id": true, "repo_id": true,
	"qualified_name": true, "kind": true, "source_ref": true,
	"confidence": true, "phase_populated": true, "origin": true,
	"provenance_kind": true,
}

// edgeKnownFields is the set of top-level NDJSON keys that map directly to
// graphstore.Edge fields.
var edgeKnownFields = map[string]bool{
	"record_type": true, "edge_id": true, "label": true,
	"from_node_id": true, "to_node_id": true, "confidence": true,
	"source_ref": true, "tier": true, "phase_populated": true,
	"completeness_caveat": true,
}

// parseNDJSON reads the extractor output file and returns typed slices of
// graphstore.Node and graphstore.Edge. Each line must have a "record_type"
// field of "node" or "edge". Empty lines are skipped.
//
// B1 — Provenance gate: every node with origin=first_party and
// provenance_kind=file must have a source_ref matching the canonical pattern
// repo@sha40:path:line. Any violation causes parseNDJSON to return an error
// listing all failures — no data reaches BulkLoad.
//
// The scanner buffer is sized at 4 MiB per line to accommodate large
// completeness_caveat strings and Properties maps.
func parseNDJSON(path string) ([]graphstore.Node, []graphstore.Edge, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open ndjson %s: %w", path, err)
	}
	defer f.Close()

	const maxLine = 4 * 1024 * 1024 // 4 MiB
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, maxLine), maxLine)

	var (
		nodes    []graphstore.Node
		edges    []graphstore.Edge
		failures []string
		lineNum  int
	)

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, nil, fmt.Errorf("line %d: json parse: %w", lineNum, err)
		}

		recordType, _ := raw["record_type"].(string)
		switch recordType {
		case "node":
			n := rawToNode(raw)

			// B1: provenance gate — validate source_ref before accepting node.
			if n.Origin == "first_party" && n.ProvenanceKind == "file" {
				if !sourceRefRe.MatchString(n.SourceRef) {
					nodePrefix := n.NodeID
					if len(nodePrefix) > 16 {
						nodePrefix = nodePrefix[:16]
					}
					failures = append(failures, fmt.Sprintf(
						"  line %d node_id=%.16s source_ref=%q (expected repo@sha40:path:line)",
						lineNum, n.NodeID, n.SourceRef,
					))
				}
			}

			nodes = append(nodes, n)
		case "edge":
			e := rawToEdge(raw)
			edges = append(edges, e)
		default:
			return nil, nil, fmt.Errorf("line %d: unknown record_type %q", lineNum, recordType)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("scan ndjson: %w", err)
	}

	// Fail the run if any provenance violations were found.
	// No data must reach BulkLoad when the gate fires.
	if len(failures) > 0 {
		return nil, nil, fmt.Errorf(
			"provenance gate: %d node(s) with invalid source_ref — refusing BulkLoad:\n%s",
			len(failures), strings.Join(failures, "\n"),
		)
	}

	return nodes, edges, nil
}

// rawToNode converts a raw NDJSON map to a graphstore.Node. Known scalar fields
// are extracted directly; all remaining fields are placed in Node.Properties.
func rawToNode(raw map[string]any) graphstore.Node {
	n := graphstore.Node{
		NodeID:         getString(raw, "node_id"),
		RepoID:         getString(raw, "repo_id"),
		QualifiedName:  getString(raw, "qualified_name"),
		Kind:           getString(raw, "kind"),
		SourceRef:      getString(raw, "source_ref"),
		Confidence:     getString(raw, "confidence"),
		PhasePopulated: getInt(raw, "phase_populated"),
		Origin:         getString(raw, "origin"),
		ProvenanceKind: getString(raw, "provenance_kind"),
	}
	props := make(map[string]any, len(raw))
	for k, v := range raw {
		if !nodeKnownFields[k] {
			props[k] = v
		}
	}
	if len(props) > 0 {
		n.Properties = props
	}
	return n
}

// rawToEdge converts a raw NDJSON map to a graphstore.Edge.
func rawToEdge(raw map[string]any) graphstore.Edge {
	return graphstore.Edge{
		EdgeID:             getString(raw, "edge_id"),
		Label:              getString(raw, "label"),
		FromNodeID:         getString(raw, "from_node_id"),
		ToNodeID:           getString(raw, "to_node_id"),
		Confidence:         getString(raw, "confidence"),
		SourceRef:          getString(raw, "source_ref"),
		Tier:               getInt(raw, "tier"),
		PhasePopulated:     getInt(raw, "phase_populated"),
		CompletenessCaveat: getString(raw, "completeness_caveat"),
	}
}

// ─── JSON field helpers ───────────────────────────────────────────────────────

func getString(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func getInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

// ─── Error helpers ────────────────────────────────────────────────────────────

// failRun records a run failure and logs the error. It uses a best-effort
// approach: if the DB update itself fails, the error is logged but not
// propagated (the pipeline has already failed).
func (s *IndexService) failRun(ctx context.Context, runID, msg string) {
	s.logger.Error("index run failed",
		slog.String("run_id", runID),
		slog.String("reason", msg),
	)
	if err := s.store.FailRun(ctx, runID, msg); err != nil {
		s.logger.Error("failed to persist run failure",
			slog.String("run_id", runID),
			slog.Any("error", err),
		)
	}
}

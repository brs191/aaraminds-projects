package diff

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

const forcePushZeroPrefix = "0000000"

type DiffResult struct {
	LaneA        []string
	LaneB        []string
	LaneC        []string
	ForceReindex bool
}

func Compute(repoPath, beforeSHA, afterSHA string) (DiffResult, error) {
	if strings.HasPrefix(strings.TrimSpace(beforeSHA), forcePushZeroPrefix) {
		return DiffResult{ForceReindex: true}, nil
	}
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return DiffResult{}, fmt.Errorf("open repo %q: %w", repoPath, err)
	}

	beforeCommit, err := repo.CommitObject(plumbing.NewHash(beforeSHA))
	if err != nil {
		return DiffResult{}, fmt.Errorf("resolve before commit %s: %w", beforeSHA, err)
	}
	afterCommit, err := repo.CommitObject(plumbing.NewHash(afterSHA))
	if err != nil {
		return DiffResult{}, fmt.Errorf("resolve after commit %s: %w", afterSHA, err)
	}

	beforeTree, err := beforeCommit.Tree()
	if err != nil {
		return DiffResult{}, fmt.Errorf("before tree: %w", err)
	}
	afterTree, err := afterCommit.Tree()
	if err != nil {
		return DiffResult{}, fmt.Errorf("after tree: %w", err)
	}

	patch, err := beforeTree.Patch(afterTree)
	if err != nil {
		return DiffResult{}, fmt.Errorf("tree patch: %w", err)
	}

	changedSet := make(map[string]struct{})
	for _, fp := range patch.FilePatches() {
		from, to := fp.Files()
		if from != nil {
			changedSet[normalizePath(from.Path())] = struct{}{}
		}
		if to != nil {
			changedSet[normalizePath(to.Path())] = struct{}{}
		}
	}

	changed := make([]string, 0, len(changedSet))
	for path := range changedSet {
		if strings.TrimSpace(path) == "" {
			continue
		}
		changed = append(changed, path)
	}
	sort.Strings(changed)
	return Classify(changed), nil
}

func Classify(changedFiles []string) DiffResult {
	res := DiffResult{}
	laneA := map[string]struct{}{}
	laneB := map[string]struct{}{}
	laneC := map[string]struct{}{}

	for _, raw := range changedFiles {
		path := normalizePath(raw)
		if path == "" {
			continue
		}

		laneA[path] = struct{}{}
		lower := strings.ToLower(path)

		if strings.HasSuffix(lower, ".java") && (strings.Contains(lower, "/service/") || strings.Contains(lower, "/repository/") || strings.Contains(lower, "/config/")) {
			laneB[path] = struct{}{}
		}
		if strings.Contains(lower, "soap") || strings.Contains(lower, "rest") || strings.Contains(lower, "wsdl") || strings.Contains(lower, ".xsd") || strings.Contains(lower, "openapi") || strings.Contains(lower, "client") {
			laneC[path] = struct{}{}
		}
	}

	res.LaneA = mapKeys(laneA)
	res.LaneB = mapKeys(laneB)
	res.LaneC = mapKeys(laneC)
	return res
}

func normalizePath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return ""
	}
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")
	return p
}

func mapKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

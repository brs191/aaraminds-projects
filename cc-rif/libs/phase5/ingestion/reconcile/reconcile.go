package reconcile

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aaraminds/rif/phase5/ingestion/queue"
)

type Reconciler struct {
	pool       *pgxpool.Pool
	queueStore *queue.Store
	interval   time.Duration
}

func NewReconciler(pool *pgxpool.Pool, queueStore *queue.Store) *Reconciler {
	return &Reconciler{
		pool:       pool,
		queueStore: queueStore,
		interval:   15 * time.Minute,
	}
}

func (r *Reconciler) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.Sweep(ctx); err != nil {
				return err
			}
		}
	}
}

func (r *Reconciler) Sweep(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT repo_id, clone_url, COALESCE(current_sha,'') FROM rif_meta.repositories`)
	if err != nil {
		return fmt.Errorf("list repos: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var repoID, cloneURL, currentSHA string
		if err := rows.Scan(&repoID, &cloneURL, &currentSHA); err != nil {
			return err
		}
		headSHA, err := lsRemoteHead(ctx, cloneURL)
		if err != nil {
			continue
		}
		if headSHA != "" && headSHA != currentSHA {
			_ = r.queueStore.Enqueue(ctx, repoID, headSHA, queue.LaneFullReindex, currentSHA, headSHA)
		}
	}
	return rows.Err()
}

func lsRemoteHead(ctx context.Context, cloneURL string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", cloneURL, "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return "", err
	}
	sc := bufio.NewScanner(&out)
	if !sc.Scan() {
		return "", nil
	}
	line := strings.TrimSpace(sc.Text())
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], nil
}

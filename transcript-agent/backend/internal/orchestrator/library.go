package orchestrator

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

// DefaultLibraryPollInterval is the LIBRARY_POLL_INTERVAL default.
const DefaultLibraryPollInterval = 30 * time.Minute

// DefaultLibraryAutoPerPoll is the LIBRARY_AUTO_PER_POLL default.
const DefaultLibraryAutoPerPoll = 3

// libraryScan runs inside the scan loop: every LibraryPollInterval it polls
// every non-deleted feed. The first scan after boot polls immediately.
func (o *Orchestrator) libraryScan(ctx context.Context) {
	if o.Tools.Stores.Library == nil {
		return
	}
	interval := o.LibraryPollInterval
	if interval <= 0 {
		interval = DefaultLibraryPollInterval
	}
	o.mu.Lock()
	due := !time.Now().Before(o.nextLibraryPoll)
	if due {
		o.nextLibraryPoll = time.Now().Add(interval)
	}
	o.mu.Unlock()
	if !due {
		return
	}
	feeds, err := o.Tools.Stores.Library.ListFeeds(ctx)
	if err != nil {
		o.Log.Error("library: list feeds for poll failed", "error", err)
		return
	}
	for _, f := range feeds {
		o.pollFeed(ctx, f)
	}
}

// RequestFeedPoll triggers a poll of one feed (POST /library/feeds/{id}/poll).
// Sync mode polls inline (deterministic tests/demos); async mode polls in a
// goroutine so the API answers 202 immediately.
func (o *Orchestrator) RequestFeedPoll(feedID uuid.UUID) {
	run := func() {
		ctx := context.Background()
		feed, err := o.Tools.Stores.Library.GetFeed(ctx, feedID)
		if err != nil || feed.DeletedAt != nil {
			return
		}
		o.pollFeed(ctx, feed)
	}
	if o.Sync {
		run()
		return
	}
	go run()
}

// pollFeed polls one feed: fetch + upsert episodes, then auto-transcribe up
// to LibraryAutoPerPoll NEW episodes for auto_transcribe feeds. Poll failures
// record feeds.poll_error (inside PollFeedOnce) and never kill the poller.
// Concurrent polls of the same feed are collapsed.
func (o *Orchestrator) pollFeed(ctx context.Context, feed *domain.Feed) {
	o.mu.Lock()
	if o.pollBusy == nil {
		o.pollBusy = map[uuid.UUID]bool{}
	}
	if o.pollBusy[feed.FeedID] {
		o.mu.Unlock()
		return
	}
	o.pollBusy[feed.FeedID] = true
	o.mu.Unlock()
	defer func() {
		o.mu.Lock()
		delete(o.pollBusy, feed.FeedID)
		o.mu.Unlock()
	}()

	newEps, err := o.Tools.PollFeedOnce(ctx, feed)
	if err != nil {
		o.Log.Error("library: feed poll failed (poll_error recorded, feed kept)",
			"feed_id", feed.FeedID, "error", err)
		return
	}
	if !feed.AutoTranscribe || len(newEps) == 0 {
		return
	}
	max := o.LibraryAutoPerPoll
	if max <= 0 {
		max = DefaultLibraryAutoPerPoll
	}
	started := 0
	for _, ep := range newEps {
		if started >= max {
			break
		}
		job, err := o.Tools.SubmitLibraryJob(ctx, ep, "system:library-poller")
		if err != nil {
			o.Log.Error("library: auto-transcribe submit failed",
				"episode_id", ep.EpisodeID, "error", err)
			continue
		}
		started++
		o.Enqueue(job.JobID)
	}
}

// finishLibraryDraft completes a library job at drafted: auto-generate the
// summary right after the quality check, fire-and-forget. A summary failure
// is tolerated — the episode still counts as drafted and the transcript is
// readable/searchable immediately. Re-drives (requeue/reclaim) are idempotent:
// an existing summary short-circuits.
func (o *Orchestrator) finishLibraryDraft(ctx context.Context, job *domain.Job) {
	existing, err := o.Tools.Stores.Summaries.LatestSummaryByJob(ctx, job.JobID)
	if err == nil && existing != nil {
		return // already summarized
	}
	if err != nil && domain.CodeOf(err) != domain.CodeSummaryNotFound {
		o.Log.Warn("library: summary lookup failed; skipping auto-summary", "job_id", job.JobID, "error", err)
		return
	}
	cfg, err := o.Tools.Config(ctx, job)
	if err != nil {
		o.Log.Warn("library: load config for auto-summary failed", "job_id", job.JobID, "error", err)
		return
	}
	source, err := o.Tools.Stores.Transcripts.LatestVersion(ctx, job.JobID, domain.VersionClean)
	if err == nil && source == nil {
		source, err = o.Tools.Stores.Transcripts.LatestVersion(ctx, job.JobID, domain.VersionRaw)
	}
	if err != nil || source == nil {
		o.Log.Warn("library: no transcript version for auto-summary", "job_id", job.JobID, "error", err)
		return
	}
	if _, err := o.Tools.GenerateSummary(ctx, job, source, cfg, "system:library"); err != nil {
		// Fire-and-forget failure tolerated: episode stays drafted.
		o.Log.Warn("library: auto-summary failed (episode stays drafted)",
			"job_id", job.JobID, "error", err)
	}
}

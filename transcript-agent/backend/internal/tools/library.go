package tools

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/metrics"
	"github.com/aaraminds/transcript-agent/internal/rss"
	"github.com/aaraminds/transcript-agent/internal/state"
)

// LibraryURIScheme marks library-job sources: library://<episode_id> resolves
// to the downloaded enclosure artifact through the object store, mirroring the
// upload:// mechanism (never a raw path or remote URL at pipeline time).
const LibraryURIScheme = "library://"

// DefaultLibraryMaxDownloadBytes is the enclosure download size cap
// (LIBRARY_MAX_DOWNLOAD_BYTES env, default 500 MiB).
const DefaultLibraryMaxDownloadBytes = int64(500) << 20

const (
	feedFetchTimeout       = 10 * time.Second
	feedMaxBytes           = int64(5) << 20 // 5 MiB feed XML cap
	libraryDownloadTimeout = 10 * time.Minute
)

func (t *Toolset) httpClient() *http.Client {
	if t.HTTPClient != nil {
		return t.HTTPClient
	}
	return http.DefaultClient
}

func (t *Toolset) libraryMaxDownloadBytes() int64 {
	if t.LibraryMaxDownloadBytes > 0 {
		return t.LibraryMaxDownloadBytes
	}
	return DefaultLibraryMaxDownloadBytes
}

// fetchFeed downloads and parses a feed URL (10s timeout, 5 MiB cap).
func (t *Toolset) fetchFeed(ctx context.Context, feedURL string) (*rss.Feed, error) {
	ctx, cancel := context.WithTimeout(ctx, feedFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, domain.E(domain.CodeFeedURLInvalid, "invalid feed URL: %v", err)
	}
	resp, err := t.httpClient().Do(req)
	if err != nil {
		return nil, domain.E(domain.CodeFeedFetchFailed, "fetch feed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, domain.E(domain.CodeFeedFetchFailed, "feed responded with HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, feedMaxBytes+1))
	if err != nil {
		return nil, domain.E(domain.CodeFeedFetchFailed, "read feed body: %v", err)
	}
	if int64(len(data)) > feedMaxBytes {
		return nil, domain.E(domain.CodeFeedFetchFailed, "feed exceeds the %d byte limit", feedMaxBytes)
	}
	return rss.Parse(data)
}

// AddFeed validates the URL, fetches the feed synchronously (add fails with
// FEED_FETCH_FAILED when unreachable/unparseable), creates the feed with its
// channel metadata, and ingests the current episodes as backfill. Backfilled
// episodes are never auto-transcribed — auto_transcribe applies only to
// episodes that appear in later polls.
func (t *Toolset) AddFeed(ctx context.Context, feedURL string, autoTranscribe bool, addedBy string) (*domain.Feed, error) {
	feedURL = strings.TrimSpace(feedURL)
	u, err := url.Parse(feedURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, domain.E(domain.CodeFeedURLInvalid, "feed_url must be an absolute http(s) URL")
	}
	existing, err := t.Stores.Library.GetFeedByURL(ctx, feedURL)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, domain.E(domain.CodeFeedAlreadyExists, "feed is already in the library")
	}
	parsed, err := t.fetchFeed(ctx, feedURL)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	feed := &domain.Feed{
		FeedID:         uuid.New(),
		FeedURL:        feedURL,
		Title:          parsed.Title,
		Description:    parsed.Description,
		ImageURL:       parsed.ImageURL,
		AutoTranscribe: autoTranscribe,
		LastPolledAt:   &now,
		CreatedAt:      now,
	}
	if err := t.Stores.Library.CreateFeed(ctx, feed); err != nil {
		return nil, err
	}
	backfilled := t.ingestEpisodes(ctx, feed, parsed.Items)
	t.Audit(ctx, nil, "user", addedBy, "library.feed_added", map[string]any{
		"feed_id":         feed.FeedID.String(),
		"feed_url_hash":   URIHash(feedURL),
		"auto_transcribe": autoTranscribe,
		"backfilled":      len(backfilled),
	})
	return feed, nil
}

// PollFeedOnce fetches the feed and upserts its episodes. Fetch/parse failures
// record feeds.poll_error (the feed stays and keeps being polled) and return
// the error. Returns the episodes newly created by this poll — the
// orchestrator auto-transcribes those for auto_transcribe feeds. Channel
// metadata is filled on first successful fetch only.
func (t *Toolset) PollFeedOnce(ctx context.Context, feed *domain.Feed) ([]*domain.Episode, error) {
	parsed, ferr := t.fetchFeed(ctx, feed.FeedURL)
	now := time.Now().UTC()
	feed.LastPolledAt = &now
	if ferr != nil {
		feed.PollError = domain.AsError(ferr).Message
		if err := t.Stores.Library.UpdateFeed(ctx, feed); err != nil {
			t.log().Error("library: record poll_error failed", "feed_id", feed.FeedID, "error", err)
		}
		t.Audit(ctx, nil, "system", "library-poller", "library.feed_poll_failed", map[string]any{
			"feed_id":    feed.FeedID.String(),
			"error_code": domain.CodeOf(ferr),
			"error":      ferr.Error(),
		})
		return nil, ferr
	}
	if feed.Title == "" {
		feed.Title = parsed.Title
	}
	if feed.Description == "" {
		feed.Description = parsed.Description
	}
	if feed.ImageURL == "" {
		feed.ImageURL = parsed.ImageURL
	}
	feed.PollError = ""
	if err := t.Stores.Library.UpdateFeed(ctx, feed); err != nil {
		return nil, err
	}
	newEps := t.ingestEpisodes(ctx, feed, parsed.Items)
	t.Audit(ctx, nil, "system", "library-poller", "library.feed_polled", map[string]any{
		"feed_id":      feed.FeedID.String(),
		"item_count":   len(parsed.Items),
		"new_episodes": len(newEps),
	})
	return newEps, nil
}

// ingestEpisodes upserts feed items by (feed_id, guid) and returns the newly
// created episodes.
func (t *Toolset) ingestEpisodes(ctx context.Context, feed *domain.Feed, items []rss.Item) []*domain.Episode {
	var created []*domain.Episode
	for _, it := range items {
		ep := &domain.Episode{
			EpisodeID:       uuid.New(),
			FeedID:          feed.FeedID,
			GUID:            it.GUID,
			Title:           it.Title,
			Description:     it.Description,
			AudioURL:        it.EnclosureURL,
			PublishedAt:     it.PublishedAt,
			DurationSeconds: it.DurationSeconds,
			CreatedAt:       time.Now().UTC(),
		}
		isNew, err := t.Stores.Library.UpsertEpisode(ctx, ep)
		if err != nil {
			t.log().Error("library: upsert episode failed",
				"feed_id", feed.FeedID, "guid", it.GUID, "error", err)
			continue
		}
		if isNew {
			created = append(created, ep)
		}
	}
	return created
}

// SubmitLibraryJob creates a transcription job for an episode. Library jobs
// carry library_mode=true (pipeline stops at drafted, summary auto-generated)
// and ownership_attested=true recorded programmatically with
// source_basis="open_rss_personal_use" — an audit event notes the basis in
// lieu of the manual attestation. EPISODE_ALREADY_TRANSCRIBED (409) when the
// episode already has a job.
func (t *Toolset) SubmitLibraryJob(ctx context.Context, ep *domain.Episode, requestedBy string) (*domain.Job, error) {
	if ep.JobID != nil {
		return nil, domain.E(domain.CodeEpisodeAlreadyTranscribed,
			"episode %s already has job %s", ep.EpisodeID, *ep.JobID)
	}
	jobID := uuid.New()
	now := time.Now().UTC()
	job := &domain.Job{
		JobID:             jobID,
		SourceType:        domain.SourceUpload, // upload semantics: no caption pre-check
		SourceURI:         LibraryURIScheme + ep.EpisodeID.String(),
		Status:            domain.StatusSubmitted,
		SubmittedBy:       requestedBy,
		OwnershipAttested: true,
		Language:          "en",
		LibraryMode:       true,
		SourceBasis:       domain.SourceBasisOpenRSSPersonalUse,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	// Create the job row first (episodes.job_id references jobs), then claim
	// the episode atomically: a concurrent transcribe of the same episode
	// loses the claim with EPISODE_ALREADY_TRANSCRIBED and its just-created
	// job is compensated to cancelled — the episode never gets two live jobs.
	if err := t.Stores.Jobs.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	if err := t.Stores.Library.ClaimEpisodeJob(ctx, ep.EpisodeID, jobID); err != nil {
		if _, cErr := t.Stores.Jobs.TransitionJob(ctx, jobID, domain.StatusSubmitted, func(j *domain.Job) error {
			j.CancelReason = "episode claimed by a concurrent transcription"
			return state.Transition(j, domain.StatusCancelled)
		}); cErr != nil {
			t.log().Error("library: cancel orphan job after claim conflict failed",
				"job_id", jobID, "error", cErr)
		}
		return nil, err
	}
	id := jobID
	ep.JobID = &id
	actorType := "user"
	if strings.HasPrefix(requestedBy, "system:") {
		actorType = "system"
	}
	metrics.JobsSubmitted.Add(1)
	t.Audit(ctx, &job.JobID, actorType, requestedBy, "job.submitted", map[string]any{
		"source_type":        job.SourceType,
		"source_uri_hash":    URIHash(job.SourceURI),
		"language":           job.Language,
		"ownership_attested": true,
		"library_mode":       true,
		"episode_id":         ep.EpisodeID.String(),
		"feed_id":            ep.FeedID.String(),
	})
	// Ownership basis event: recorded in lieu of the manual attestation.
	t.Audit(ctx, &job.JobID, actorType, requestedBy, "job.ownership_attested", map[string]any{
		"source_basis": domain.SourceBasisOpenRSSPersonalUse,
		"note":         "attestation set programmatically: open RSS enclosure transcribed for personal use",
		"episode_id":   ep.EpisodeID.String(),
	})
	return job, nil
}

// episodeFromLibraryURI resolves library://<episode_id> to its episode row.
func (t *Toolset) episodeFromLibraryURI(ctx context.Context, uri string) (*domain.Episode, error) {
	id, err := uuid.Parse(strings.TrimPrefix(uri, LibraryURIScheme))
	if err != nil {
		return nil, domain.E(domain.CodeInvalidSourceURI, "invalid library URI %q", uri)
	}
	return t.Stores.Library.GetEpisode(ctx, id)
}

// enclosureExt derives the stored file extension from the enclosure URL path
// (falling back to the Content-Type, then mp3).
func enclosureExt(audioURL, contentType string) string {
	if u, err := url.Parse(audioURL); err == nil {
		ext := strings.ToLower(strings.TrimPrefix(path.Ext(u.Path), "."))
		for _, s := range domain.SupportedUploadExtensions {
			if ext == s {
				return ext
			}
		}
	}
	switch {
	case strings.Contains(contentType, "mp4"), strings.Contains(contentType, "m4a"), strings.Contains(contentType, "aac"):
		return "m4a"
	case strings.Contains(contentType, "wav"):
		return "wav"
	}
	return "mp3"
}

func mimeForExt(ext string) string {
	switch ext {
	case "mp3":
		return "audio/mpeg"
	case "m4a":
		return "audio/mp4"
	case "wav":
		return "audio/wav"
	case "mp4":
		return "video/mp4"
	case "mov":
		return "video/quicktime"
	}
	return "application/octet-stream"
}

// EnsureLibraryMedia downloads the episode enclosure if it has not been
// downloaded yet: HTTP GET streamed into the object store under
// library/<episode_id>.<ext>, bounded by LIBRARY_MAX_DOWNLOAD_BYTES
// (LIBRARY_DOWNLOAD_TOO_LARGE above the cap) and a 10-minute timeout.
// The artifact is recorded as the job's source_media (standard retention),
// and episodes.media_artifact_id links it for future resolution. Runs as the
// first pipeline step of library jobs so the worker pool bounds concurrency.
func (t *Toolset) EnsureLibraryMedia(ctx context.Context, job *domain.Job) error {
	ep, err := t.episodeFromLibraryURI(ctx, job.SourceURI)
	if err != nil {
		return err
	}
	if ep.MediaArtifactID != nil {
		if _, err := t.Stores.Artifacts.GetArtifact(ctx, *ep.MediaArtifactID); err == nil {
			return nil // already downloaded
		}
		ep.MediaArtifactID = nil // artifact swept/missing: re-download
	}
	capBytes := t.libraryMaxDownloadBytes()
	dctx, cancel := context.WithTimeout(ctx, libraryDownloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(dctx, http.MethodGet, ep.AudioURL, nil)
	if err != nil {
		return domain.E(domain.CodeLibraryDownloadFailed, "invalid enclosure URL %q: %v", ep.AudioURL, err)
	}
	resp, err := t.httpClient().Do(req)
	if err != nil {
		return domain.E(domain.CodeLibraryDownloadFailed, "download enclosure: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.E(domain.CodeLibraryDownloadFailed, "enclosure responded with HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > capBytes {
		return domain.E(domain.CodeLibraryDownloadTooLarge,
			"enclosure is %d bytes, above the %d byte download cap", resp.ContentLength, capBytes)
	}
	ext := enclosureExt(ep.AudioURL, resp.Header.Get("Content-Type"))
	key := "library/" + ep.EpisodeID.String() + "." + ext
	uri, n, err := t.Objects.PutStream(dctx, key, io.LimitReader(resp.Body, capBytes+1))
	if err != nil {
		return domain.E(domain.CodeLibraryDownloadFailed, "store enclosure: %v", err)
	}
	if n > capBytes {
		_ = t.Objects.Delete(ctx, uri)
		return domain.E(domain.CodeLibraryDownloadTooLarge,
			"enclosure exceeds the %d byte download cap", capBytes)
	}
	now := time.Now().UTC()
	art := &domain.MediaArtifact{
		ArtifactID:     uuid.New(),
		JobID:          job.JobID,
		ArtifactType:   domain.ArtifactSourceMedia,
		URI:            uri,
		MimeType:       mimeForExt(ext),
		SizeBytes:      n,
		RetentionUntil: t.RetentionUntil(now),
		CreatedAt:      now,
	}
	if err := t.Stores.Artifacts.CreateArtifact(ctx, art); err != nil {
		_ = t.Objects.Delete(ctx, uri)
		return err
	}
	ep.MediaArtifactID = &art.ArtifactID
	if err := t.Stores.Library.UpdateEpisode(ctx, ep); err != nil {
		return err
	}
	t.Audit(ctx, &job.JobID, "tool", "download_episode_media", "tool.download_episode_media.completed",
		map[string]any{
			"episode_id":     ep.EpisodeID.String(),
			"artifact_id":    art.ArtifactID.String(),
			"audio_url_hash": URIHash(ep.AudioURL),
			"size_bytes":     n,
			"mime_type":      art.MimeType,
		})
	return nil
}

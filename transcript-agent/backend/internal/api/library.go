package api

// Library mode endpoints (frozen contract — the React UI is built against
// exactly these shapes): personal-use RSS feed subscriptions, episode listing,
// per-episode transcription, and full-text transcript search. Standard auth
// headers apply (any authenticated role — the library is a shared personal
// space); errors use the standard envelope.

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/audit"
	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/tools"
)

// --- JSON shapes (frozen) ---------------------------------------------------

type feedJSON struct {
	FeedID         string  `json:"feed_id"`
	FeedURL        string  `json:"feed_url"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	ImageURL       *string `json:"image_url"`
	AutoTranscribe bool    `json:"auto_transcribe"`
	EpisodeCount   int     `json:"episode_count"`
	LastPolledAt   *string `json:"last_polled_at"`
	PollError      *string `json:"poll_error"`
	CreatedAt      string  `json:"created_at"`
}

type episodeJSON struct {
	EpisodeID       string  `json:"episode_id"`
	FeedID          string  `json:"feed_id"`
	FeedTitle       string  `json:"feed_title"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	PublishedAt     *string `json:"published_at"`
	DurationSeconds *int    `json:"duration_seconds"`
	AudioURL        string  `json:"audio_url"`
	JobID           *string `json:"job_id"`
	JobStatus       *string `json:"job_status"`
	CreatedAt       string  `json:"created_at"`
}

type searchResultJSON struct {
	EpisodeID           *string `json:"episode_id"`
	EpisodeTitle        *string `json:"episode_title"`
	FeedTitle           *string `json:"feed_title"`
	JobID               string  `json:"job_id"`
	TranscriptVersionID string  `json:"transcript_version_id"`
	SegmentID           string  `json:"segment_id"`
	StartMS             int     `json:"start_ms"`
	Snippet             string  `json:"snippet"`
	Rank                float64 `json:"rank"`
}

func rfc3339(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func rfc3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := rfc3339(*t)
	return &s
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *Server) feedView(r *http.Request, f *domain.Feed) feedJSON {
	count, err := s.Tools.Stores.Library.CountEpisodes(r.Context(), f.FeedID)
	if err != nil {
		count = 0
	}
	return feedJSON{
		FeedID:         f.FeedID.String(),
		FeedURL:        f.FeedURL,
		Title:          f.Title,
		Description:    f.Description,
		ImageURL:       strPtr(f.ImageURL),
		AutoTranscribe: f.AutoTranscribe,
		EpisodeCount:   count,
		LastPolledAt:   rfc3339Ptr(f.LastPolledAt),
		PollError:      strPtr(f.PollError),
		CreatedAt:      rfc3339(f.CreatedAt),
	}
}

func (s *Server) episodeView(r *http.Request, ep *domain.Episode, feedTitles map[uuid.UUID]string) episodeJSON {
	out := episodeJSON{
		EpisodeID:       ep.EpisodeID.String(),
		FeedID:          ep.FeedID.String(),
		Title:           ep.Title,
		Description:     ep.Description,
		PublishedAt:     rfc3339Ptr(ep.PublishedAt),
		DurationSeconds: ep.DurationSeconds,
		AudioURL:        ep.AudioURL,
		CreatedAt:       rfc3339(ep.CreatedAt),
	}
	title, ok := feedTitles[ep.FeedID]
	if !ok {
		if f, err := s.Tools.Stores.Library.GetFeed(r.Context(), ep.FeedID); err == nil {
			title = f.Title
		}
		feedTitles[ep.FeedID] = title
	}
	out.FeedTitle = title
	if ep.JobID != nil {
		id := ep.JobID.String()
		out.JobID = &id
		if job, err := s.Tools.Stores.Jobs.GetJob(r.Context(), *ep.JobID); err == nil {
			st := string(job.Status)
			out.JobStatus = &st
		}
	}
	return out
}

// --- feeds -------------------------------------------------------------------

// handleAddFeed implements POST /api/v1/library/feeds: validates by fetching
// the feed synchronously (FEED_FETCH_FAILED when unreachable/unparseable) and
// backfills the current episodes without auto-transcribing them.
func (s *Server) handleAddFeed(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	var in struct {
		FeedURL        string `json:"feed_url"`
		AutoTranscribe bool   `json:"auto_transcribe"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	feed, err := s.Tools.AddFeed(r.Context(), in.FeedURL, in.AutoTranscribe, ident.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.feedView(r, feed))
}

func (s *Server) handleListFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := s.Tools.Stores.Library.ListFeeds(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]feedJSON, 0, len(feeds))
	for _, f := range feeds {
		out = append(out, s.feedView(r, f))
	}
	writeJSON(w, http.StatusOK, map[string]any{"feeds": out})
}

// loadFeed resolves {feedID}; soft-deleted feeds answer 404.
func (s *Server) loadFeed(r *http.Request) (*domain.Feed, error) {
	id, err := parseUUID(r.PathValue("feedID"), "feed_id")
	if err != nil {
		return nil, err
	}
	feed, err := s.Tools.Stores.Library.GetFeed(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if feed.DeletedAt != nil {
		return nil, domain.E(domain.CodeFeedNotFound, "feed %s not found", id)
	}
	return feed, nil
}

// handleDeleteFeed soft-deletes a feed (204). Episodes, jobs, and transcripts
// are kept; the feed and its episodes disappear from library listings.
func (s *Server) handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	feed, err := s.loadFeed(r)
	if err != nil {
		writeError(w, err)
		return
	}
	now := time.Now().UTC()
	feed.DeletedAt = &now
	if err := s.Tools.Stores.Library.UpdateFeed(r.Context(), feed); err != nil {
		writeError(w, err)
		return
	}
	s.Tools.Audit(r.Context(), nil, audit.ActorUser, ident.UserID, "library.feed_deleted",
		map[string]any{"feed_id": feed.FeedID.String(), "feed_url_hash": tools.URIHash(feed.FeedURL)})
	w.WriteHeader(http.StatusNoContent)
}

// handlePollFeed queues an immediate poll of one feed → 202 poll_queued.
func (s *Server) handlePollFeed(w http.ResponseWriter, r *http.Request) {
	feed, err := s.loadFeed(r)
	if err != nil {
		writeError(w, err)
		return
	}
	s.Orch.RequestFeedPoll(feed.FeedID)
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "poll_queued"})
}

// --- episodes ------------------------------------------------------------------

// handleListEpisodes implements GET /api/v1/library/episodes with optional
// feed_id, q (case-insensitive title/description filter) and
// transcribed=true|false filters. Newest published_at first, nulls last.
func (s *Server) handleListEpisodes(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var feedID *uuid.UUID
	if raw := query.Get("feed_id"); raw != "" {
		id, err := parseUUID(raw, "feed_id")
		if err != nil {
			writeError(w, err)
			return
		}
		feedID = &id
	}
	var transcribed *bool
	switch query.Get("transcribed") {
	case "":
	case "true":
		v := true
		transcribed = &v
	case "false":
		v := false
		transcribed = &v
	default:
		writeError(w, domain.E(domain.CodeValidationError, "transcribed must be true or false"))
		return
	}
	q := strings.ToLower(strings.TrimSpace(query.Get("q")))

	episodes, err := s.Tools.Stores.Library.ListEpisodes(r.Context(), feedID)
	if err != nil {
		writeError(w, err)
		return
	}
	feedTitles := map[uuid.UUID]string{}
	out := make([]episodeJSON, 0, len(episodes))
	for _, ep := range episodes {
		if q != "" && !strings.Contains(strings.ToLower(ep.Title), q) &&
			!strings.Contains(strings.ToLower(ep.Description), q) {
			continue
		}
		if transcribed != nil && (ep.JobID != nil) != *transcribed {
			continue
		}
		out = append(out, s.episodeView(r, ep, feedTitles))
	}
	writeJSON(w, http.StatusOK, map[string]any{"episodes": out})
}

// handleTranscribeEpisode implements POST .../episodes/{episodeID}/transcribe:
// creates a library job (stops at drafted, ownership basis
// open_rss_personal_use recorded programmatically) → 202 Episode.
// 409 EPISODE_ALREADY_TRANSCRIBED when the episode already has a job.
func (s *Server) handleTranscribeEpisode(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	id, err := parseUUID(r.PathValue("episodeID"), "episode_id")
	if err != nil {
		writeError(w, err)
		return
	}
	ep, err := s.Tools.Stores.Library.GetEpisode(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct{} // UI sends {}; body is ignored
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	job, err := s.Tools.SubmitLibraryJob(r.Context(), ep, ident.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	s.Orch.Enqueue(job.JobID)
	// Re-read: in sync mode the pipeline has already advanced the job.
	fresh, err := s.Tools.Stores.Library.GetEpisode(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, s.episodeView(r, fresh, map[uuid.UUID]string{}))
}

// --- search --------------------------------------------------------------------

const librarySearchLimit = 50

// handleLibrarySearch implements GET /api/v1/library/search?q= over transcript
// segments: latest drafted library transcripts plus approved non-library
// transcripts. Non-library hits carry null episode/feed fields.
func (s *Server) handleLibrarySearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(q) < 2 {
		writeError(w, domain.E(domain.CodeValidationError, "q must be at least 2 characters"))
		return
	}
	hits, err := s.Tools.Stores.Search.SearchSegments(r.Context(), q, librarySearchLimit)
	if err != nil {
		writeError(w, err)
		return
	}
	feedTitles := map[uuid.UUID]string{}
	out := make([]searchResultJSON, 0, len(hits))
	for _, h := range hits {
		res := searchResultJSON{
			JobID:               h.JobID.String(),
			TranscriptVersionID: h.TranscriptVersionID.String(),
			SegmentID:           h.SegmentID.String(),
			StartMS:             h.StartMS,
			Snippet:             h.Snippet,
			Rank:                h.Rank,
		}
		if ep, err := s.Tools.Stores.Library.GetEpisodeByJobID(r.Context(), h.JobID); err == nil && ep != nil {
			epID := ep.EpisodeID.String()
			res.EpisodeID = &epID
			res.EpisodeTitle = &ep.Title
			title, ok := feedTitles[ep.FeedID]
			if !ok {
				if f, ferr := s.Tools.Stores.Library.GetFeed(r.Context(), ep.FeedID); ferr == nil {
					title = f.Title
				}
				feedTitles[ep.FeedID] = title
			}
			res.FeedTitle = &title
		}
		out = append(out, res)
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": out})
}

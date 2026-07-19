package app_test

// Library-mode tests (personal-use extension): feed lifecycle end-to-end
// against an httptest RSS server, library job semantics (stop at drafted,
// programmatic ownership basis, no caption pre-check), the enclosure download
// cap, feed error handling, and the memory-store transcript search.

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aaraminds/transcript-agent/internal/app"
)

// --- fixture feed server ------------------------------------------------------

type feedItem struct {
	guid, title, pubDate, duration, enclosurePath string
}

// fakeFeed serves a mutable RSS 2.0 feed at /feed.xml and mp3 enclosures at
// /media/<name>.mp3.
type fakeFeed struct {
	mu        sync.Mutex
	items     []feedItem
	enclosure []byte
	srv       *httptest.Server
}

func newFakeFeed(t *testing.T, enclosure []byte, items ...feedItem) *fakeFeed {
	t.Helper()
	f := &fakeFeed{items: items, enclosure: enclosure}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /feed.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(f.render()))
	})
	mux.HandleFunc("GET /media/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		data := f.enclosure
		f.mu.Unlock()
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write(data)
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeFeed) url() string { return f.srv.URL + "/feed.xml" }

func (f *fakeFeed) addItem(it feedItem) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = append(f.items, it)
}

func (f *fakeFeed) render() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` +
		`<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"><channel>` +
		`<title>Test Show</title><description>A test podcast.</description>` +
		`<image><url>` + f.srv.URL + `/cover.png</url></image>`)
	for _, it := range f.items {
		fmt.Fprintf(&b, `<item><title>%s</title><description>desc %s</description>`,
			it.title, it.title)
		if it.guid != "" {
			fmt.Fprintf(&b, `<guid>%s</guid>`, it.guid)
		}
		if it.pubDate != "" {
			fmt.Fprintf(&b, `<pubDate>%s</pubDate>`, it.pubDate)
		}
		if it.duration != "" {
			fmt.Fprintf(&b, `<itunes:duration>%s</itunes:duration>`, it.duration)
		}
		fmt.Fprintf(&b, `<enclosure url="%s%s" type="audio/mpeg" length="%d"/></item>`,
			f.srv.URL, it.enclosurePath, len(f.enclosure))
	}
	b.WriteString(`</channel></rss>`)
	return b.String()
}

// --- contract-shaped response types ---------------------------------------------

type feedResp struct {
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

type episodeResp struct {
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

type searchResp struct {
	Results []struct {
		EpisodeID           *string `json:"episode_id"`
		EpisodeTitle        *string `json:"episode_title"`
		FeedTitle           *string `json:"feed_title"`
		JobID               string  `json:"job_id"`
		TranscriptVersionID string  `json:"transcript_version_id"`
		SegmentID           string  `json:"segment_id"`
		StartMS             int     `json:"start_ms"`
		Snippet             string  `json:"snippet"`
		Rank                float64 `json:"rank"`
	} `json:"results"`
}

func addFeed(e *env, feedURL string, auto bool) feedResp {
	e.t.Helper()
	var feed feedResp
	status := e.do("POST", "/api/v1/library/feeds", producer,
		map[string]any{"feed_url": feedURL, "auto_transcribe": auto}, &feed)
	e.must(status, http.StatusCreated, "add feed")
	return feed
}

func listEpisodes(e *env, query string) []episodeResp {
	e.t.Helper()
	var out struct {
		Episodes []episodeResp `json:"episodes"`
	}
	status := e.do("GET", "/api/v1/library/episodes"+query, producer, nil, &out)
	e.must(status, http.StatusOK, "list episodes")
	return out.Episodes
}

// --- test 2: feed lifecycle end-to-end -------------------------------------------

func TestLibraryFeedLifecycleE2E(t *testing.T) {
	e := newEnv(t, nil) // sync mode: polls and pipelines run inline
	ff := newFakeFeed(t, []byte("fake-mp3-bytes-fake-mp3-bytes"),
		feedItem{guid: "ep-1", title: "Backfill One", pubDate: "Mon, 06 Jul 2026 09:00:00 +0000", duration: "10:00", enclosurePath: "/media/ep1.mp3"},
		feedItem{guid: "ep-2", title: "Backfill Two", pubDate: "Tue, 07 Jul 2026 09:00:00 +0000", duration: "600", enclosurePath: "/media/ep2.mp3"},
	)

	// Add (validating fetch happens synchronously) with auto_transcribe on.
	feed := addFeed(e, ff.url(), true)
	if feed.Title != "Test Show" {
		t.Errorf("feed title = %q, want from channel metadata", feed.Title)
	}
	if feed.ImageURL == nil || !strings.HasSuffix(*feed.ImageURL, "/cover.png") {
		t.Errorf("feed image_url = %v", feed.ImageURL)
	}
	if feed.EpisodeCount != 2 {
		t.Errorf("episode_count = %d, want 2 (backfill on add)", feed.EpisodeCount)
	}
	if feed.PollError != nil {
		t.Errorf("poll_error = %v, want null", *feed.PollError)
	}

	// Backfilled episodes are never auto-transcribed.
	eps := listEpisodes(e, "")
	if len(eps) != 2 {
		t.Fatalf("episodes = %d, want 2", len(eps))
	}
	for _, ep := range eps {
		if ep.JobID != nil {
			t.Errorf("backfilled episode %q has a job — backfill must not auto-transcribe", ep.Title)
		}
	}
	// Newest published_at first.
	if eps[0].Title != "Backfill Two" {
		t.Errorf("episode order: got %q first, want newest published_at first", eps[0].Title)
	}

	// A NEW episode appears; manual poll picks it up and auto-transcribes it.
	ff.addItem(feedItem{guid: "ep-3", title: "Fresh Episode", pubDate: "Wed, 08 Jul 2026 09:00:00 +0000", duration: "601", enclosurePath: "/media/ep3.mp3"})
	var pollOut map[string]string
	status := e.do("POST", "/api/v1/library/feeds/"+feed.FeedID+"/poll", producer, map[string]any{}, &pollOut)
	e.must(status, http.StatusAccepted, "poll feed")
	if pollOut["status"] != "poll_queued" {
		t.Errorf("poll response = %v", pollOut)
	}

	eps = listEpisodes(e, "")
	if len(eps) != 3 {
		t.Fatalf("episodes after poll = %d, want 3", len(eps))
	}
	var fresh *episodeResp
	for i := range eps {
		if eps[i].Title == "Fresh Episode" {
			fresh = &eps[i]
		} else if eps[i].JobID != nil {
			t.Errorf("old episode %q got a job; auto-transcribe must cover NEW episodes only", eps[i].Title)
		}
	}
	if fresh == nil {
		t.Fatal("new episode missing after poll")
	}
	if fresh.JobID == nil {
		t.Fatal("new episode was not auto-transcribed")
	}
	if fresh.JobStatus == nil || *fresh.JobStatus != "drafted" {
		t.Fatalf("new episode job_status = %v, want drafted", fresh.JobStatus)
	}
	if fresh.DurationSeconds == nil || *fresh.DurationSeconds != 601 {
		t.Errorf("duration_seconds = %v, want 601 (itunes:duration seconds form)", fresh.DurationSeconds)
	}

	// The library job stopped at drafted (never in_review) with the summary
	// auto-generated right after the quality check.
	var job jobResp
	status = e.do("GET", "/api/v1/jobs/"+*fresh.JobID, producer, nil, &job)
	e.must(status, http.StatusOK, "get library job as producer")
	if job.Status != "drafted" {
		t.Fatalf("library job status = %q, want drafted", job.Status)
	}
	var summary summaryResp
	status = e.do("GET", "/api/v1/jobs/"+*fresh.JobID+"/summary", producer, nil, &summary)
	e.must(status, http.StatusOK, "auto-generated summary")
	if summary.Text == "" {
		t.Error("auto-generated summary text is empty")
	}

	// Transcript is readable immediately at drafted.
	var versions struct {
		Versions []versionResp `json:"versions"`
	}
	status = e.do("GET", "/api/v1/jobs/"+*fresh.JobID+"/transcripts", producer, nil, &versions)
	e.must(status, http.StatusOK, "list transcripts at drafted")
	if len(versions.Versions) < 2 {
		t.Fatalf("versions = %d, want raw+clean", len(versions.Versions))
	}

	// transcribed=true|false filters.
	if got := listEpisodes(e, "?transcribed=true"); len(got) != 1 {
		t.Errorf("transcribed=true returned %d, want 1", len(got))
	}
	if got := listEpisodes(e, "?transcribed=false"); len(got) != 2 {
		t.Errorf("transcribed=false returned %d, want 2", len(got))
	}
	if got := listEpisodes(e, "?q=fresh"); len(got) != 1 {
		t.Errorf("q=fresh returned %d, want 1", len(got))
	}

	// Feed list reflects the new count.
	var feeds struct {
		Feeds []feedResp `json:"feeds"`
	}
	status = e.do("GET", "/api/v1/library/feeds", producer, nil, &feeds)
	e.must(status, http.StatusOK, "list feeds")
	if len(feeds.Feeds) != 1 || feeds.Feeds[0].EpisodeCount != 3 {
		t.Errorf("feeds = %+v, want 1 feed with 3 episodes", feeds.Feeds)
	}

	// Soft delete: feed and episodes leave the listings; the job stays.
	status = e.do("DELETE", "/api/v1/library/feeds/"+feed.FeedID, producer, nil, nil)
	e.must(status, http.StatusNoContent, "delete feed")
	status = e.do("GET", "/api/v1/library/feeds", producer, nil, &feeds)
	e.must(status, http.StatusOK, "list feeds after delete")
	if len(feeds.Feeds) != 0 {
		t.Errorf("feeds after delete = %d, want 0", len(feeds.Feeds))
	}
	if got := listEpisodes(e, ""); len(got) != 0 {
		t.Errorf("episodes after feed delete = %d, want 0 in listing", len(got))
	}
	status = e.do("GET", "/api/v1/jobs/"+*fresh.JobID, producer, nil, &job)
	e.must(status, http.StatusOK, "job survives feed soft-delete")
}

// --- test 3: library job semantics -------------------------------------------------

func TestLibraryJobSemantics(t *testing.T) {
	e := newEnv(t, nil)
	ff := newFakeFeed(t, []byte("fake-mp3-bytes"),
		feedItem{guid: "s-1", title: "Semantics", pubDate: "Mon, 06 Jul 2026 09:00:00 +0000", enclosurePath: "/media/s1.mp3"})
	feed := addFeed(e, ff.url(), false)
	_ = feed

	eps := listEpisodes(e, "")
	if len(eps) != 1 {
		t.Fatalf("episodes = %d", len(eps))
	}

	// Manual transcribe works for any untranscribed episode; UI sends {}.
	var ep episodeResp
	status := e.do("POST", "/api/v1/library/episodes/"+eps[0].EpisodeID+"/transcribe", producer, map[string]any{}, &ep)
	e.must(status, http.StatusAccepted, "transcribe episode")
	if ep.JobID == nil {
		t.Fatal("transcribe returned no job_id")
	}
	if ep.JobStatus == nil || *ep.JobStatus != "drafted" {
		t.Fatalf("job_status = %v, want drafted (sync pipeline)", ep.JobStatus)
	}

	// Stops at drafted, never in_review; library flags on the job JSON.
	var job jobResp
	status = e.do("GET", "/api/v1/jobs/"+*ep.JobID, producer, nil, &job)
	e.must(status, http.StatusOK, "get job")
	if job.Status != "drafted" {
		t.Fatalf("status = %q, want drafted", job.Status)
	}
	if !job.OwnershipAttested {
		t.Error("ownership_attested = false, want programmatic true")
	}

	// Audit: ownership basis open_rss_personal_use recorded; caption pre-check
	// skipped entirely (upload semantics, no YouTube) and the job never paused
	// for a caption decision.
	var audit struct {
		Events []struct {
			EventType    string         `json:"event_type"`
			EventPayload map[string]any `json:"event_payload"`
		} `json:"events"`
	}
	status = e.do("GET", "/api/v1/jobs/"+*ep.JobID+"/audit", producer, nil, &audit)
	e.must(status, http.StatusOK, "audit")
	basisRecorded := false
	for _, ev := range audit.Events {
		if strings.Contains(ev.EventType, "caption") {
			t.Errorf("caption event %q on a library job — caption pre-check must be skipped", ev.EventType)
		}
		if ev.EventType == "job.ownership_attested" &&
			ev.EventPayload["source_basis"] == "open_rss_personal_use" {
			basisRecorded = true
		}
		if ev.EventType == "job.status_changed" && ev.EventPayload["to"] == "in_review" {
			t.Error("library job transitioned to in_review")
		}
	}
	if !basisRecorded {
		t.Error("no audit event records source_basis open_rss_personal_use")
	}

	// Second transcribe of the same episode → 409.
	var er errResp
	status = e.do("POST", "/api/v1/library/episodes/"+eps[0].EpisodeID+"/transcribe", producer, map[string]any{}, &er)
	e.must(status, http.StatusConflict, "double transcribe")
	if er.Error.Code != "EPISODE_ALREADY_TRANSCRIBED" {
		t.Errorf("code = %q, want EPISODE_ALREADY_TRANSCRIBED", er.Error.Code)
	}
}

// --- test 4: search (memory implementation) ------------------------------------------

func TestLibrarySearch(t *testing.T) {
	e := newEnv(t, nil)
	ff := newFakeFeed(t, []byte("fake-mp3-bytes"),
		feedItem{guid: "q-1", title: "Searchable", pubDate: "Mon, 06 Jul 2026 09:00:00 +0000", enclosurePath: "/media/q1.mp3"})
	feed := addFeed(e, ff.url(), false)

	eps := listEpisodes(e, "?feed_id="+feed.FeedID)
	var ep episodeResp
	status := e.do("POST", "/api/v1/library/episodes/"+eps[0].EpisodeID+"/transcribe", producer, map[string]any{}, &ep)
	e.must(status, http.StatusAccepted, "transcribe")

	// The mock STT script contains "approval gate"; the drafted library
	// transcript must be findable with a <b>-wrapped snippet.
	var res searchResp
	status = e.do("GET", "/api/v1/library/search?q=approval", producer, nil, &res)
	e.must(status, http.StatusOK, "search")
	if len(res.Results) == 0 {
		t.Fatal("no search results for a word in the drafted library transcript")
	}
	hit := res.Results[0]
	if !strings.Contains(hit.Snippet, "<b>") || !strings.Contains(strings.ToLower(hit.Snippet), "<b>approval</b>") {
		t.Errorf("snippet %q lacks <b>approval</b> wrapping", hit.Snippet)
	}
	if hit.EpisodeID == nil || *hit.EpisodeID != eps[0].EpisodeID {
		t.Errorf("episode_id = %v, want %s", hit.EpisodeID, eps[0].EpisodeID)
	}
	if hit.FeedTitle == nil || *hit.FeedTitle != "Test Show" {
		t.Errorf("feed_title = %v", hit.FeedTitle)
	}
	if hit.JobID != *ep.JobID {
		t.Errorf("job_id = %q, want %q", hit.JobID, *ep.JobID)
	}
	if hit.Rank <= 0 {
		t.Errorf("rank = %v, want > 0", hit.Rank)
	}

	// q below the 2-character minimum → 400.
	var er errResp
	status = e.do("GET", "/api/v1/library/search?q=a", producer, nil, &er)
	e.must(status, http.StatusBadRequest, "short query")
	status = e.do("GET", "/api/v1/library/search?q=", producer, nil, &er)
	e.must(status, http.StatusBadRequest, "empty query")
}

// --- test 5: download cap ---------------------------------------------------------------

func TestLibraryDownloadCap(t *testing.T) {
	// Documented choice: an enclosure above LIBRARY_MAX_DOWNLOAD_BYTES parks
	// the JOB in needs_user_action/replace_media with LIBRARY_DOWNLOAD_TOO_LARGE
	// (episode keeps its job link; the user can replace media or cancel).
	e := newEnvWith(t, nil, func(o *app.Options) {
		o.LibraryMaxDownloadBytes = 16 // bytes
	})
	ff := newFakeFeed(t, bytes.Repeat([]byte("x"), 1024),
		feedItem{guid: "big-1", title: "Too Big", pubDate: "Mon, 06 Jul 2026 09:00:00 +0000", enclosurePath: "/media/big.mp3"})
	feed := addFeed(e, ff.url(), false)
	_ = feed

	eps := listEpisodes(e, "")
	var ep episodeResp
	status := e.do("POST", "/api/v1/library/episodes/"+eps[0].EpisodeID+"/transcribe", producer, map[string]any{}, &ep)
	e.must(status, http.StatusAccepted, "transcribe oversized episode")
	if ep.JobStatus == nil || *ep.JobStatus != "needs_user_action" {
		t.Fatalf("job_status = %v, want needs_user_action", ep.JobStatus)
	}
	var job jobResp
	status = e.do("GET", "/api/v1/jobs/"+*ep.JobID, producer, nil, &job)
	e.must(status, http.StatusOK, "get job")
	if job.ActionRequired != "replace_media" {
		t.Errorf("action_required = %q, want replace_media", job.ActionRequired)
	}
	if job.LastError == nil || job.LastError.Code != "LIBRARY_DOWNLOAD_TOO_LARGE" {
		t.Errorf("last_error = %+v, want LIBRARY_DOWNLOAD_TOO_LARGE", job.LastError)
	}
}

// --- async scan-loop poller ---------------------------------------------------------------

// The orchestrator scan loop polls feeds every LibraryPollInterval in async
// mode: a new episode is discovered and auto-transcribed to drafted without
// any manual poll call.
func TestLibraryScanLoopAutoPolls(t *testing.T) {
	e := newEnvWith(t, nil, func(o *app.Options) {
		o.Sync = false
		o.LibraryPollInterval = 30 * time.Millisecond
	})
	ff := newFakeFeed(t, []byte("fake-mp3-bytes"),
		feedItem{guid: "a-1", title: "Backfill", pubDate: "Mon, 06 Jul 2026 09:00:00 +0000", enclosurePath: "/media/a1.mp3"})
	feed := addFeed(e, ff.url(), true)
	_ = feed

	e.app.Orch.Start(t.Context(), 2, 20*time.Millisecond)

	ff.addItem(feedItem{guid: "a-2", title: "Scanned Fresh", pubDate: "Tue, 07 Jul 2026 09:00:00 +0000", enclosurePath: "/media/a2.mp3"})
	deadline := time.Now().Add(10 * time.Second)
	for {
		eps := listEpisodes(e, "?transcribed=true")
		if len(eps) == 1 && eps[0].Title == "Scanned Fresh" &&
			eps[0].JobStatus != nil && *eps[0].JobStatus == "drafted" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("scan-loop poller never auto-transcribed the new episode to drafted; transcribed=%+v", eps)
		}
		time.Sleep(20 * time.Millisecond)
	}
	// Backfill stayed untranscribed.
	if eps := listEpisodes(e, "?transcribed=false"); len(eps) != 1 || eps[0].Title != "Backfill" {
		t.Errorf("untranscribed = %+v, want just the backfill episode", eps)
	}
}

// --- test 6: feed errors -----------------------------------------------------------------

func TestLibraryFeedErrors(t *testing.T) {
	e := newEnv(t, nil)

	// Unreachable feed on add → 400 FEED_FETCH_FAILED (validated synchronously).
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL + "/feed.xml"
	dead.Close()
	var er errResp
	status := e.do("POST", "/api/v1/library/feeds", producer,
		map[string]any{"feed_url": deadURL, "auto_transcribe": false}, &er)
	e.must(status, http.StatusBadRequest, "unreachable feed")
	if er.Error.Code != "FEED_FETCH_FAILED" {
		t.Errorf("code = %q, want FEED_FETCH_FAILED", er.Error.Code)
	}

	// Malformed URL → 400 FEED_URL_INVALID.
	status = e.do("POST", "/api/v1/library/feeds", producer,
		map[string]any{"feed_url": "not a url", "auto_transcribe": false}, &er)
	e.must(status, http.StatusBadRequest, "invalid feed url")
	if er.Error.Code != "FEED_URL_INVALID" {
		t.Errorf("code = %q, want FEED_URL_INVALID", er.Error.Code)
	}

	// Reachable but non-RSS content → 400 FEED_FETCH_FAILED.
	notRSS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html><body>hello</body></html>"))
	}))
	t.Cleanup(notRSS.Close)
	status = e.do("POST", "/api/v1/library/feeds", producer,
		map[string]any{"feed_url": notRSS.URL, "auto_transcribe": false}, &er)
	e.must(status, http.StatusBadRequest, "non-RSS feed")
	if er.Error.Code != "FEED_FETCH_FAILED" {
		t.Errorf("code = %q, want FEED_FETCH_FAILED", er.Error.Code)
	}

	// Working feed adds fine; a duplicate add → 409 FEED_ALREADY_EXISTS.
	ff := newFakeFeed(t, []byte("fake-mp3-bytes"),
		feedItem{guid: "e-1", title: "One", pubDate: "Mon, 06 Jul 2026 09:00:00 +0000", enclosurePath: "/media/e1.mp3"})
	feed := addFeed(e, ff.url(), false)
	status = e.do("POST", "/api/v1/library/feeds", producer,
		map[string]any{"feed_url": ff.url(), "auto_transcribe": true}, &er)
	e.must(status, http.StatusConflict, "duplicate feed")
	if er.Error.Code != "FEED_ALREADY_EXISTS" {
		t.Errorf("code = %q, want FEED_ALREADY_EXISTS", er.Error.Code)
	}

	// Poll failure records poll_error and keeps the feed (and the poller):
	// kill the feed server, poll, verify poll_error; a later add+poll of a
	// healthy feed still works.
	ff.srv.Close()
	var pollOut map[string]string
	status = e.do("POST", "/api/v1/library/feeds/"+feed.FeedID+"/poll", producer, map[string]any{}, &pollOut)
	e.must(status, http.StatusAccepted, "poll dead feed")
	var feeds struct {
		Feeds []feedResp `json:"feeds"`
	}
	status = e.do("GET", "/api/v1/library/feeds", producer, nil, &feeds)
	e.must(status, http.StatusOK, "list feeds")
	if len(feeds.Feeds) != 1 {
		t.Fatalf("feeds = %d, want the failing feed to stay", len(feeds.Feeds))
	}
	if feeds.Feeds[0].PollError == nil || *feeds.Feeds[0].PollError == "" {
		t.Error("poll_error not recorded after failed poll")
	}

	// Poller survives: a healthy feed polls fine afterwards.
	ff2 := newFakeFeed(t, []byte("fake-mp3-bytes"),
		feedItem{guid: "h-1", title: "Healthy", pubDate: "Mon, 06 Jul 2026 09:00:00 +0000", enclosurePath: "/media/h1.mp3"})
	feed2 := addFeed(e, ff2.url(), false)
	status = e.do("POST", "/api/v1/library/feeds/"+feed2.FeedID+"/poll", producer, map[string]any{}, &pollOut)
	e.must(status, http.StatusAccepted, "poll healthy feed after failure")
	status = e.do("GET", "/api/v1/library/feeds", producer, nil, &feeds)
	e.must(status, http.StatusOK, "list feeds again")
	for _, f := range feeds.Feeds {
		if f.FeedID == feed2.FeedID && f.PollError != nil {
			t.Errorf("healthy feed carries poll_error %v", *f.PollError)
		}
	}

	// Poll of an unknown feed → 404 FEED_NOT_FOUND.
	status = e.do("POST", "/api/v1/library/feeds/00000000-0000-0000-0000-000000000000/poll", producer, map[string]any{}, &er)
	e.must(status, http.StatusNotFound, "poll unknown feed")
	if er.Error.Code != "FEED_NOT_FOUND" {
		t.Errorf("code = %q, want FEED_NOT_FOUND", er.Error.Code)
	}
}

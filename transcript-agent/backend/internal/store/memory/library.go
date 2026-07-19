package memory

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/store"
)

// --- library helpers -------------------------------------------------------

func copyFeed(f *domain.Feed) *domain.Feed {
	c := *f
	if f.LastPolledAt != nil {
		t := *f.LastPolledAt
		c.LastPolledAt = &t
	}
	if f.DeletedAt != nil {
		t := *f.DeletedAt
		c.DeletedAt = &t
	}
	return &c
}

func copyEpisode(e *domain.Episode) *domain.Episode {
	c := *e
	if e.PublishedAt != nil {
		t := *e.PublishedAt
		c.PublishedAt = &t
	}
	if e.DurationSeconds != nil {
		d := *e.DurationSeconds
		c.DurationSeconds = &d
	}
	if e.MediaArtifactID != nil {
		id := *e.MediaArtifactID
		c.MediaArtifactID = &id
	}
	if e.JobID != nil {
		id := *e.JobID
		c.JobID = &id
	}
	return &c
}

func epKey(feedID uuid.UUID, guid string) string {
	return feedID.String() + "\x00" + guid
}

// --- LibraryStore ------------------------------------------------------------

func (s *Store) CreateFeed(_ context.Context, f *domain.Feed) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.feeds[f.FeedID] = copyFeed(f)
	s.feedOrder = append(s.feedOrder, f.FeedID)
	return nil
}

func (s *Store) GetFeed(_ context.Context, id uuid.UUID) (*domain.Feed, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.feeds[id]
	if !ok {
		return nil, domain.E(domain.CodeFeedNotFound, "feed %s not found", id)
	}
	return copyFeed(f), nil
}

func (s *Store) GetFeedByURL(_ context.Context, feedURL string) (*domain.Feed, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range s.feedOrder {
		f := s.feeds[id]
		if f.FeedURL == feedURL && f.DeletedAt == nil {
			return copyFeed(f), nil
		}
	}
	return nil, nil
}

func (s *Store) UpdateFeed(_ context.Context, f *domain.Feed) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.feeds[f.FeedID]; !ok {
		return domain.E(domain.CodeFeedNotFound, "feed %s not found", f.FeedID)
	}
	s.feeds[f.FeedID] = copyFeed(f)
	return nil
}

func (s *Store) ListFeeds(_ context.Context) ([]*domain.Feed, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.Feed, 0, len(s.feedOrder))
	for _, id := range s.feedOrder {
		if f := s.feeds[id]; f.DeletedAt == nil {
			out = append(out, copyFeed(f))
		}
	}
	sort.SliceStable(out, func(i, k int) bool { return out[i].CreatedAt.After(out[k].CreatedAt) })
	return out, nil
}

func (s *Store) UpsertEpisode(_ context.Context, e *domain.Episode) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := epKey(e.FeedID, e.GUID)
	if existingID, ok := s.epByFeedGUID[key]; ok {
		// Refresh feed-supplied metadata only; keep media/job linkage.
		cur := s.episodes[existingID]
		cur.Title = e.Title
		cur.Description = e.Description
		cur.AudioURL = e.AudioURL
		cur.PublishedAt = nil
		if e.PublishedAt != nil {
			t := *e.PublishedAt
			cur.PublishedAt = &t
		}
		cur.DurationSeconds = nil
		if e.DurationSeconds != nil {
			d := *e.DurationSeconds
			cur.DurationSeconds = &d
		}
		*e = *copyEpisode(cur)
		return false, nil
	}
	s.episodes[e.EpisodeID] = copyEpisode(e)
	s.epOrder = append(s.epOrder, e.EpisodeID)
	s.epByFeedGUID[key] = e.EpisodeID
	return true, nil
}

func (s *Store) GetEpisode(_ context.Context, id uuid.UUID) (*domain.Episode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.episodes[id]
	if !ok {
		return nil, domain.E(domain.CodeEpisodeNotFound, "episode %s not found", id)
	}
	return copyEpisode(e), nil
}

func (s *Store) GetEpisodeByJobID(_ context.Context, jobID uuid.UUID) (*domain.Episode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range s.epOrder {
		e := s.episodes[id]
		if e.JobID != nil && *e.JobID == jobID {
			return copyEpisode(e), nil
		}
	}
	return nil, nil
}

func (s *Store) UpdateEpisode(_ context.Context, e *domain.Episode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.episodes[e.EpisodeID]; !ok {
		return domain.E(domain.CodeEpisodeNotFound, "episode %s not found", e.EpisodeID)
	}
	s.episodes[e.EpisodeID] = copyEpisode(e)
	return nil
}

func (s *Store) ClaimEpisodeJob(_ context.Context, episodeID, jobID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.episodes[episodeID]
	if !ok {
		return domain.E(domain.CodeEpisodeNotFound, "episode %s not found", episodeID)
	}
	if e.JobID != nil {
		return domain.E(domain.CodeEpisodeAlreadyTranscribed,
			"episode %s already has job %s", episodeID, *e.JobID)
	}
	id := jobID
	e.JobID = &id
	return nil
}

func (s *Store) ListEpisodes(_ context.Context, feedID *uuid.UUID) ([]*domain.Episode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.Episode
	for _, id := range s.epOrder {
		e := s.episodes[id]
		if feedID != nil {
			if e.FeedID != *feedID {
				continue
			}
		} else if f, ok := s.feeds[e.FeedID]; !ok || f.DeletedAt != nil {
			continue // hide episodes of soft-deleted feeds from the all-feeds listing
		}
		out = append(out, copyEpisode(e))
	}
	sortEpisodes(out)
	return out, nil
}

// sortEpisodes orders newest published_at first, nulls last, created_at desc
// as tiebreak.
func sortEpisodes(eps []*domain.Episode) {
	sort.SliceStable(eps, func(i, k int) bool {
		a, b := eps[i], eps[k]
		switch {
		case a.PublishedAt != nil && b.PublishedAt != nil:
			if !a.PublishedAt.Equal(*b.PublishedAt) {
				return a.PublishedAt.After(*b.PublishedAt)
			}
		case a.PublishedAt != nil:
			return true
		case b.PublishedAt != nil:
			return false
		}
		return a.CreatedAt.After(b.CreatedAt)
	})
}

func (s *Store) CountEpisodes(_ context.Context, feedID uuid.UUID) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, id := range s.epOrder {
		if s.episodes[id].FeedID == feedID {
			n++
		}
	}
	return n, nil
}

// --- SegmentSearcher ---------------------------------------------------------

// SearchSegments is the naive memory implementation: case-insensitive
// substring match with manual <b>...</b> wrapping. Eligible versions mirror
// the Postgres query: latest clean (raw fallback) of drafted library jobs,
// current approved version of non-library jobs. Rank is the match count.
func (s *Store) SearchSegments(_ context.Context, query string, limit int) ([]*store.SearchResult, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []*store.SearchResult
	for _, jobID := range s.jobOrder {
		j := s.jobs[jobID]
		var verID uuid.UUID
		if j.LibraryMode {
			if j.Status != domain.StatusDrafted {
				continue
			}
			v := s.latestVersionLocked(jobID, domain.VersionClean)
			if v == nil {
				v = s.latestVersionLocked(jobID, domain.VersionRaw)
			}
			if v == nil {
				continue
			}
			verID = v.TranscriptVersionID
		} else {
			a := s.currentApprovalLocked(jobID)
			if a == nil {
				continue
			}
			verID = a.ApprovedTranscriptVersionID
		}
		for _, segID := range s.segByVer[verID] {
			sg := s.segments[segID]
			count := strings.Count(strings.ToLower(sg.Text), q)
			if count == 0 {
				continue
			}
			results = append(results, &store.SearchResult{
				JobID:               jobID,
				TranscriptVersionID: verID,
				SegmentID:           sg.SegmentID,
				StartMS:             sg.StartMS,
				Snippet:             highlight(sg.Text, query),
				Rank:                float64(count),
			})
		}
	}
	sort.SliceStable(results, func(i, k int) bool {
		if results[i].Rank != results[k].Rank {
			return results[i].Rank > results[k].Rank
		}
		return results[i].StartMS < results[k].StartMS
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (s *Store) latestVersionLocked(jobID uuid.UUID, versionType string) *domain.TranscriptVersion {
	var latest *domain.TranscriptVersion
	for _, id := range s.verOrder {
		v := s.versions[id]
		if v.JobID == jobID && v.VersionType == versionType {
			latest = v
		}
	}
	return latest
}

func (s *Store) currentApprovalLocked(jobID uuid.UUID) *domain.Approval {
	var latest *domain.Approval
	for _, id := range s.apprOrder {
		a := s.approvals[id]
		if a.JobID == jobID && a.SupersededByApprovalID == nil {
			latest = a
		}
	}
	return latest
}

// highlight wraps case-insensitive occurrences of q in text with <b>...</b>,
// preserving the original casing of the matched substring.
func highlight(text, q string) string {
	lower := strings.ToLower(text)
	lq := strings.ToLower(q)
	if lq == "" || len(lower) != len(text) {
		// Empty query, or case folding changed byte offsets (non-ASCII edge):
		// return the text unwrapped rather than corrupt the snippet.
		return text
	}
	var b strings.Builder
	i := 0
	for {
		k := strings.Index(lower[i:], lq)
		if k < 0 {
			b.WriteString(text[i:])
			break
		}
		k += i
		end := k + len(lq)
		if end > len(text) {
			b.WriteString(text[i:])
			break
		}
		b.WriteString(text[i:k])
		b.WriteString("<b>")
		b.WriteString(text[k:end])
		b.WriteString("</b>")
		i = end
	}
	return b.String()
}

var _ store.LibraryStore = (*Store)(nil)
var _ store.SegmentSearcher = (*Store)(nil)

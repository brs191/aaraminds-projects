package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/store"
)

// --- LibraryStore (feeds + episodes, migration 0009) -----------------------

const feedColumns = `feed_id, feed_url, title, description, image_url,
	auto_transcribe, last_polled_at, poll_error, created_at, deleted_at`

func scanFeed(row pgx.Row) (*domain.Feed, error) {
	var (
		f         domain.Feed
		imageURL  *string
		pollError *string
	)
	err := row.Scan(&f.FeedID, &f.FeedURL, &f.Title, &f.Description, &imageURL,
		&f.AutoTranscribe, &f.LastPolledAt, &pollError, &f.CreatedAt, &f.DeletedAt)
	if err != nil {
		return nil, err
	}
	if imageURL != nil {
		f.ImageURL = *imageURL
	}
	if pollError != nil {
		f.PollError = *pollError
	}
	return &f, nil
}

func feedArgs(f *domain.Feed) []any {
	var imageURL, pollError *string
	if f.ImageURL != "" {
		imageURL = &f.ImageURL
	}
	if f.PollError != "" {
		pollError = &f.PollError
	}
	return []any{
		f.FeedID, f.FeedURL, f.Title, f.Description, imageURL,
		f.AutoTranscribe, f.LastPolledAt, pollError, f.CreatedAt, f.DeletedAt,
	}
}

func (s *Store) CreateFeed(ctx context.Context, f *domain.Feed) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO feeds (`+feedColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, feedArgs(f)...)
	return err
}

func (s *Store) GetFeed(ctx context.Context, id uuid.UUID) (*domain.Feed, error) {
	f, err := scanFeed(s.pool.QueryRow(ctx,
		`SELECT `+feedColumns+` FROM feeds WHERE feed_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeFeedNotFound, "feed %s not found", id)
	}
	return f, err
}

func (s *Store) GetFeedByURL(ctx context.Context, feedURL string) (*domain.Feed, error) {
	f, err := scanFeed(s.pool.QueryRow(ctx, `
		SELECT `+feedColumns+` FROM feeds
		WHERE feed_url = $1 AND deleted_at IS NULL`, feedURL))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return f, err
}

func (s *Store) UpdateFeed(ctx context.Context, f *domain.Feed) error {
	var imageURL, pollError *string
	if f.ImageURL != "" {
		imageURL = &f.ImageURL
	}
	if f.PollError != "" {
		pollError = &f.PollError
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE feeds SET feed_url=$2, title=$3, description=$4, image_url=$5,
			auto_transcribe=$6, last_polled_at=$7, poll_error=$8, deleted_at=$9
		WHERE feed_id=$1`,
		f.FeedID, f.FeedURL, f.Title, f.Description, imageURL,
		f.AutoTranscribe, f.LastPolledAt, pollError, f.DeletedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound(domain.CodeFeedNotFound, "feed %s not found", f.FeedID)
	}
	return nil
}

func (s *Store) ListFeeds(ctx context.Context) ([]*domain.Feed, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+feedColumns+` FROM feeds
		WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Feed
	for rows.Next() {
		f, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

const episodeColumns = `episode_id, feed_id, guid, title, description, audio_url,
	published_at, duration_seconds, media_artifact_id, job_id, created_at`

func scanEpisode(row pgx.Row) (*domain.Episode, error) {
	var e domain.Episode
	err := row.Scan(&e.EpisodeID, &e.FeedID, &e.GUID, &e.Title, &e.Description,
		&e.AudioURL, &e.PublishedAt, &e.DurationSeconds, &e.MediaArtifactID,
		&e.JobID, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) UpsertEpisode(ctx context.Context, e *domain.Episode) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO episodes (`+episodeColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (feed_id, guid) DO NOTHING`,
		e.EpisodeID, e.FeedID, e.GUID, e.Title, e.Description, e.AudioURL,
		e.PublishedAt, e.DurationSeconds, e.MediaArtifactID, e.JobID, e.CreatedAt)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}
	// Existing row: refresh feed-supplied metadata only, never job/media links.
	existing, err := scanEpisode(s.pool.QueryRow(ctx, `
		UPDATE episodes SET title=$3, description=$4, audio_url=$5,
			published_at=$6, duration_seconds=$7
		WHERE feed_id=$1 AND guid=$2
		RETURNING `+episodeColumns, e.FeedID, e.GUID, e.Title, e.Description,
		e.AudioURL, e.PublishedAt, e.DurationSeconds))
	if err != nil {
		return false, err
	}
	*e = *existing
	return false, nil
}

func (s *Store) GetEpisode(ctx context.Context, id uuid.UUID) (*domain.Episode, error) {
	e, err := scanEpisode(s.pool.QueryRow(ctx,
		`SELECT `+episodeColumns+` FROM episodes WHERE episode_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeEpisodeNotFound, "episode %s not found", id)
	}
	return e, err
}

func (s *Store) GetEpisodeByJobID(ctx context.Context, jobID uuid.UUID) (*domain.Episode, error) {
	e, err := scanEpisode(s.pool.QueryRow(ctx,
		`SELECT `+episodeColumns+` FROM episodes WHERE job_id = $1`, jobID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return e, err
}

func (s *Store) UpdateEpisode(ctx context.Context, e *domain.Episode) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE episodes SET title=$2, description=$3, audio_url=$4, published_at=$5,
			duration_seconds=$6, media_artifact_id=$7, job_id=$8
		WHERE episode_id=$1`,
		e.EpisodeID, e.Title, e.Description, e.AudioURL, e.PublishedAt,
		e.DurationSeconds, e.MediaArtifactID, e.JobID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound(domain.CodeEpisodeNotFound, "episode %s not found", e.EpisodeID)
	}
	return nil
}

func (s *Store) ClaimEpisodeJob(ctx context.Context, episodeID, jobID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE episodes SET job_id = $2
		WHERE episode_id = $1 AND job_id IS NULL`, episodeID, jobID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.GetEpisode(ctx, episodeID); err != nil {
			return err
		}
		return domain.E(domain.CodeEpisodeAlreadyTranscribed,
			"episode %s already has a transcription job", episodeID)
	}
	return nil
}

func (s *Store) ListEpisodes(ctx context.Context, feedID *uuid.UUID) ([]*domain.Episode, error) {
	q := `SELECT ` + episodeColumns + ` FROM episodes e`
	var args []any
	if feedID != nil {
		q += ` WHERE e.feed_id = $1`
		args = append(args, *feedID)
	} else {
		q += ` WHERE EXISTS (SELECT 1 FROM feeds f
			WHERE f.feed_id = e.feed_id AND f.deleted_at IS NULL)`
	}
	q += ` ORDER BY e.published_at DESC NULLS LAST, e.created_at DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Episode
	for rows.Next() {
		e, err := scanEpisode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) CountEpisodes(ctx context.Context, feedID uuid.UUID) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM episodes WHERE feed_id = $1`, feedID).Scan(&n)
	return n, err
}

// --- SegmentSearcher (tsvector GIN index, migration 0010) -------------------

// SearchSegments implements full-text search over transcript segments:
// websearch_to_tsquery for parsing, ts_headline for the <b>-wrapped snippet,
// ts_rank for ordering. Eligible versions: the latest clean (raw fallback)
// version of each drafted library job, and the current (non-superseded)
// approved version of each non-library job.
func (s *Store) SearchSegments(ctx context.Context, query string, limit int) ([]*store.SearchResult, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT seg.segment_id, seg.transcript_version_id, v.job_id, seg.start_ms,
			ts_headline('english', seg.text, websearch_to_tsquery('english', $1),
				'StartSel=<b>, StopSel=</b>') AS snippet,
			ts_rank(seg.text_search, websearch_to_tsquery('english', $1)) AS rank
		FROM transcript_segments seg
		JOIN transcript_versions v ON v.transcript_version_id = seg.transcript_version_id
		JOIN jobs j ON j.job_id = v.job_id
		WHERE seg.text_search @@ websearch_to_tsquery('english', $1)
		  AND (
			(j.library_mode AND j.status = 'drafted'
			 AND seg.transcript_version_id = (
				SELECT tv.transcript_version_id FROM transcript_versions tv
				WHERE tv.job_id = j.job_id AND tv.version_type IN ('clean','raw')
				ORDER BY CASE tv.version_type WHEN 'clean' THEN 0 ELSE 1 END,
					tv.created_at DESC
				LIMIT 1))
			OR
			(NOT j.library_mode AND seg.transcript_version_id = (
				SELECT a.approved_transcript_version_id FROM approvals a
				WHERE a.job_id = j.job_id AND a.superseded_by_approval_id IS NULL
				ORDER BY a.approved_at DESC LIMIT 1))
		  )
		ORDER BY rank DESC, seg.start_ms ASC
		LIMIT $2`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*store.SearchResult
	for rows.Next() {
		var r store.SearchResult
		if err := rows.Scan(&r.SegmentID, &r.TranscriptVersionID, &r.JobID,
			&r.StartMS, &r.Snippet, &r.Rank); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

var _ store.LibraryStore = (*Store)(nil)
var _ store.SegmentSearcher = (*Store)(nil)

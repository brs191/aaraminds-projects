import { useEffect, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { ApiError } from "../../api/client";
import {
  useAddFeed,
  useDeleteFeed,
  useEpisodes,
  useFeeds,
  usePollFeed,
  useTranscribeEpisode,
} from "../../api/hooks";
import { isJobStatus, type Episode, type Feed } from "../../api/types";
import {
  EmptyState,
  ErrorBox,
  Loading,
  StatusBadge,
  formatTimestamp,
} from "../../components/ui";

// ---------- Feeds panel ----------

function addFeedErrorMessage(error: unknown): string | null {
  if (!(error instanceof ApiError)) return null;
  switch (error.code) {
    case "FEED_URL_INVALID":
      return "That does not look like a valid feed URL — it must be an https:// RSS URL.";
    case "FEED_FETCH_FAILED":
      return "The feed could not be fetched. Check the URL and try again.";
    case "FEED_ALREADY_EXISTS":
      return "That feed is already in the library.";
    default:
      return null;
  }
}

function AddFeedForm() {
  const addFeed = useAddFeed();
  const [url, setUrl] = useState("");
  const [autoTranscribe, setAutoTranscribe] = useState(false);

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    const feedUrl = url.trim();
    if (!feedUrl || addFeed.isPending) return;
    addFeed.mutate(
      { feed_url: feedUrl, auto_transcribe: autoTranscribe },
      { onSuccess: () => setUrl("") },
    );
  };

  const friendlyError = addFeedErrorMessage(addFeed.error);

  return (
    <form className="card form" onSubmit={onSubmit}>
      <h2>Add feed</h2>
      <div className="field">
        <label htmlFor="feed-url">RSS feed URL</label>
        <input
          id="feed-url"
          type="text"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://example.com/podcast.xml"
        />
      </div>
      <div className="field checkbox-field">
        <label>
          <input
            type="checkbox"
            checked={autoTranscribe}
            onChange={(e) => setAutoTranscribe(e.target.checked)}
          />{" "}
          Auto-transcribe new episodes
        </label>
      </div>
      {friendlyError ? (
        <div className="error-box" role="alert">
          <strong>Could not add feed:</strong> {friendlyError}
        </div>
      ) : (
        <ErrorBox error={addFeed.error} prefix="Could not add feed:" />
      )}
      <button type="submit" className="primary" disabled={addFeed.isPending || !url.trim()}>
        {addFeed.isPending ? "Adding…" : "Add feed"}
      </button>
    </form>
  );
}

function FeedCard({ feed }: { feed: Feed }) {
  const pollFeed = usePollFeed();
  const deleteFeed = useDeleteFeed();
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  const displayName = feed.title || feed.feed_url;

  return (
    <div className="card feed-card">
      <div className="feed-card-head">
        {feed.image_url ? (
          <img className="feed-thumb" src={feed.image_url} alt="" />
        ) : (
          <div className="feed-thumb feed-thumb-placeholder" aria-hidden>
            ♪
          </div>
        )}
        <div className="feed-card-title">
          <strong>{displayName}</strong>
          <div className="muted hint">
            {feed.episode_count} episode{feed.episode_count === 1 ? "" : "s"}
            {feed.auto_transcribe && (
              <>
                {" "}
                <span className="badge chip-source">auto-transcribe</span>
              </>
            )}
          </div>
        </div>
      </div>
      <div className="muted hint">
        Last polled: {feed.last_polled_at ? formatTimestamp(feed.last_polled_at) : "never"}
      </div>
      {feed.poll_error && <p className="error-text hint">Poll error: {feed.poll_error}</p>}
      <ErrorBox error={pollFeed.error} prefix="Poll failed:" />
      <ErrorBox error={deleteFeed.error} prefix="Delete failed:" />
      {confirmingDelete ? (
        <div
          className="notice-banner feed-delete-confirm"
          role="alertdialog"
          aria-label="Confirm feed removal"
        >
          Remove <strong>{displayName}</strong>? Episodes and transcripts are kept.
          <div className="button-row">
            <button
              className="danger"
              disabled={deleteFeed.isPending}
              onClick={() => deleteFeed.mutate(feed.feed_id)}
            >
              {deleteFeed.isPending ? "Removing…" : "Remove feed"}
            </button>
            <button disabled={deleteFeed.isPending} onClick={() => setConfirmingDelete(false)}>
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <div className="button-row feed-actions">
          <button disabled={pollFeed.isPending} onClick={() => pollFeed.mutate(feed.feed_id)}>
            {pollFeed.isPending ? "Polling…" : "Poll now"}
          </button>
          <button onClick={() => setConfirmingDelete(true)}>Delete…</button>
          {pollFeed.isSuccess && (
            <span className="muted hint" role="status">
              Poll queued.
            </span>
          )}
        </div>
      )}
    </div>
  );
}

// ---------- Episodes panel ----------

/** Contract renders episode duration as h:mm (not mm:ss). */
function formatEpisodeDuration(seconds: number | null): string {
  if (seconds === null || seconds <= 0) return "—";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  return `${h}:${String(m).padStart(2, "0")}`;
}

function EpisodeStatusChip({ status }: { status: string | null }) {
  if (status === null) return <span className="badge chip-none">not transcribed</span>;
  // Reuse the job status palette when the string matches a known status.
  if (isJobStatus(status)) return <StatusBadge status={status} />;
  return <span className="badge chip-source">{status.replace(/_/g, " ")}</span>;
}

function EpisodeRow({
  episode,
  onTranscribe,
  transcribing,
}: {
  episode: Episode;
  onTranscribe: () => void;
  transcribing: boolean;
}) {
  return (
    <tr>
      <td>
        <div className="episode-title">{episode.title}</div>
        <div className="muted hint">{episode.feed_title}</div>
      </td>
      <td className="muted">
        {episode.published_at ? formatTimestamp(episode.published_at) : "—"}
      </td>
      <td>{formatEpisodeDuration(episode.duration_seconds)}</td>
      <td>
        <EpisodeStatusChip status={episode.job_status} />
      </td>
      <td>
        {episode.job_id ? (
          <Link to={`/jobs/${episode.job_id}`} className="button episode-action">
            Open
          </Link>
        ) : (
          <button className="episode-action" disabled={transcribing} onClick={onTranscribe}>
            {transcribing ? "Queuing…" : "Transcribe"}
          </button>
        )}
      </td>
    </tr>
  );
}

const TRANSCRIBED_FILTERS = [
  [null, "All"],
  [true, "Transcribed"],
  [false, "Not transcribed"],
] as const;

function EpisodesPanel({ feeds }: { feeds: Feed[] }) {
  const [feedId, setFeedId] = useState("");
  const [qInput, setQInput] = useState("");
  const [q, setQ] = useState("");
  const [transcribed, setTranscribed] = useState<boolean | null>(null);

  // Debounce the text filter so we do not refetch on every keystroke.
  useEffect(() => {
    const handle = setTimeout(() => setQ(qInput.trim()), 300);
    return () => clearTimeout(handle);
  }, [qInput]);

  const episodesQuery = useEpisodes({ feedId, q, transcribed });
  const transcribe = useTranscribeEpisode();

  const episodes = episodesQuery.data?.episodes ?? [];
  const hasFilters = feedId !== "" || q !== "" || transcribed !== null;

  return (
    <section className="card episodes-panel">
      <div className="episode-filters">
        <div className="field inline">
          <label htmlFor="episode-feed-filter">Feed</label>
          <select
            id="episode-feed-filter"
            value={feedId}
            onChange={(e) => setFeedId(e.target.value)}
          >
            <option value="">All feeds</option>
            {feeds.map((f) => (
              <option key={f.feed_id} value={f.feed_id}>
                {f.title || f.feed_url}
              </option>
            ))}
          </select>
        </div>
        <input
          type="text"
          className="episode-q"
          value={qInput}
          onChange={(e) => setQInput(e.target.value)}
          placeholder="Filter episodes…"
          aria-label="Filter episodes by text"
        />
        <div className="toggle-group" role="radiogroup" aria-label="Transcription filter">
          {TRANSCRIBED_FILTERS.map(([value, label]) => (
            <button
              key={label}
              type="button"
              role="radio"
              aria-checked={transcribed === value}
              className={transcribed === value ? "toggle active" : "toggle"}
              onClick={() => setTranscribed(value)}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      <ErrorBox error={transcribe.error} prefix="Transcribe failed:" />

      {episodesQuery.isLoading ? (
        <Loading label="Loading episodes…" />
      ) : episodesQuery.error ? (
        <ErrorBox error={episodesQuery.error} prefix="Could not load episodes:" />
      ) : episodes.length === 0 ? (
        <EmptyState>
          {hasFilters
            ? "No episodes match the current filters."
            : "No episodes yet. Add a feed and poll it to pull episodes in."}
        </EmptyState>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Episode</th>
              <th>Published</th>
              <th>Duration</th>
              <th>Status</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {episodes.map((ep) => (
              <EpisodeRow
                key={ep.episode_id}
                episode={ep}
                onTranscribe={() => transcribe.mutate(ep.episode_id)}
                transcribing={transcribe.isPending && transcribe.variables === ep.episode_id}
              />
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}

// ---------- Page ----------

export default function LibraryPage() {
  const feedsQuery = useFeeds();
  const feeds = feedsQuery.data?.feeds ?? [];

  return (
    <div className="page">
      <div className="page-head">
        <h1>Library</h1>
        <Link to="/library/search" className="button">
          Search transcripts
        </Link>
      </div>
      <div className="library-layout">
        <section className="stack">
          <AddFeedForm />
          {feedsQuery.isLoading ? (
            <Loading label="Loading feeds…" />
          ) : feedsQuery.error ? (
            <ErrorBox error={feedsQuery.error} prefix="Could not load feeds:" />
          ) : feeds.length === 0 ? (
            <EmptyState>No feeds yet. Add an RSS feed above to start the library.</EmptyState>
          ) : (
            feeds.map((f) => <FeedCard key={f.feed_id} feed={f} />)
          )}
        </section>
        <EpisodesPanel feeds={feeds} />
      </div>
    </div>
  );
}

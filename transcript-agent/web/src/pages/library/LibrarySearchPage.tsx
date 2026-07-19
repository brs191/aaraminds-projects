import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { useLibrarySearch } from "../../api/hooks";
import { EmptyState, ErrorBox, Loading, formatMs } from "../../components/ui";

/**
 * Snippets arrive as plain text with matches wrapped in <b>…</b> — the ONLY
 * markup the contract allows. Split on the markers and render the odd parts
 * as <mark>; the string is never injected as HTML.
 */
function Snippet({ snippet }: { snippet: string }) {
  const parts = snippet.split(/<\/?b>/g);
  return (
    <span className="search-snippet">
      {parts.map((part, i) =>
        i % 2 === 1 ? <mark key={i}>{part}</mark> : <span key={i}>{part}</span>,
      )}
    </span>
  );
}

export default function LibrarySearchPage() {
  const [input, setInput] = useState("");
  const [query, setQuery] = useState("");
  const search = useLibrarySearch(query);

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    const q = input.trim();
    if (q.length < 2) return;
    setQuery(q);
  };

  const results = search.data?.results ?? [];
  const tooShort = input.trim().length > 0 && input.trim().length < 2;

  return (
    <div className="page">
      <div className="breadcrumb muted">
        <Link to="/library">Library</Link> / Search
      </div>
      <h1>Search transcripts</h1>
      <form className="card search-form" onSubmit={onSubmit}>
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Search across all transcribed episodes…"
          aria-label="Search transcripts"
          autoFocus
        />
        <button type="submit" className="primary" disabled={input.trim().length < 2}>
          Search
        </button>
      </form>
      {tooShort && <p className="muted hint">Type at least 2 characters to search.</p>}

      {query === "" ? (
        <EmptyState>
          Search matches transcript text across every transcribed episode in the library. Press
          Enter to search.
        </EmptyState>
      ) : search.isLoading ? (
        <Loading label={`Searching for "${query}"…`} />
      ) : search.error ? (
        <ErrorBox error={search.error} prefix="Search failed:" />
      ) : results.length === 0 ? (
        <EmptyState>No matches for "{query}".</EmptyState>
      ) : (
        <ol className="search-results">
          {results.map((r, i) => (
            <li key={`${r.segment_id}-${i}`}>
              <Link className="search-result" to={`/jobs/${r.job_id}?t=${r.start_ms}`}>
                <div className="search-result-head">
                  <strong>{r.episode_title ?? "Approved transcript"}</strong>
                  {r.feed_title !== null && <span className="muted">— {r.feed_title}</span>}
                  <span className="mono search-result-time">{formatMs(r.start_ms)}</span>
                </div>
                <Snippet snippet={r.snippet} />
              </Link>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}

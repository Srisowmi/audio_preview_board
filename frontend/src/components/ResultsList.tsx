import type { SearchState } from "../hooks/useDebouncedSearch";
import type { Track } from "../types";
import { StatusMessage } from "./StatusMessage";
import { TrackRow } from "./TrackRow";

interface Props {
  search: SearchState;
  isInShortlist: (id: number) => boolean;
  onAdd: (track: Track) => void;
  onRetry: () => void;
}

export function ResultsList({ search, isInShortlist, onAdd, onRetry }: Props) {
  const { status, searchedQuery, results, error } = search;

  return (
    <section aria-labelledby="results-heading" className="panel">
      <h2 id="results-heading">Results</h2>

      {status === "idle" && <StatusMessage kind="idle">Search for a song or artist to get started.</StatusMessage>}
      {status === "loading" && <StatusMessage kind="loading">Searching…</StatusMessage>}
      {status === "error" && (
        <StatusMessage kind="error" onRetry={onRetry}>
          {error}
        </StatusMessage>
      )}
      {status === "success" && results.length === 0 && (
        <StatusMessage kind="empty">No results for “{searchedQuery}”. Try a different search.</StatusMessage>
      )}
      {status === "success" && results.length > 0 && (
        <>
          <p className="visually-hidden" role="status">
            {results.length} results for {searchedQuery}
          </p>
          <ul className="track-list">
            {results.map((track) => (
              <TrackRow key={track.id} variant="result" track={track} inShortlist={isInShortlist(track.id)} onAdd={onAdd} />
            ))}
          </ul>
        </>
      )}
    </section>
  );
}

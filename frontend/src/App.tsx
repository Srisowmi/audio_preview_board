import { useCallback, useState } from "react";
import { PlayerBar } from "./components/PlayerBar";
import { ResultsList } from "./components/ResultsList";
import { SearchBox } from "./components/SearchBox";
import { Shortlist } from "./components/Shortlist";
import { PlayerProvider, usePlayer } from "./context/PlayerContext";
import { useDebouncedSearch } from "./hooks/useDebouncedSearch";
import { useShortlist } from "./hooks/useShortlist";
import type { Track } from "./types";

function Board() {
  const search = useDebouncedSearch();
  const shortlist = useShortlist();
  const player = usePlayer();
  const [announcement, setAnnouncement] = useState("");

  const { remove, move, items } = shortlist;

  // Removing the track that is currently playing must stop playback first.
  const handleRemove = useCallback(
    (track: Track) => {
      if (player.track?.id === track.id) player.stop();
      remove(track.id);
      setAnnouncement(`${track.name} removed from shortlist`);
    },
    [player, remove],
  );

  const handleMove = useCallback(
    (track: Track, direction: "up" | "down") => {
      const from = items.findIndex((t) => t.id === track.id);
      const to = direction === "up" ? from - 1 : from + 1;
      if (from === -1 || to < 0 || to >= items.length) return;
      move(from, to);
      setAnnouncement(`${track.name} moved to position ${to + 1} of ${items.length}`);
    },
    [items, move],
  );

  const handleAdd = useCallback(
    (track: Track) => {
      shortlist.add(track);
      setAnnouncement(`${track.name} added to shortlist`);
    },
    [shortlist],
  );

  return (
    <>
      <header>
        <h1>Audio Preview Board</h1>
        <SearchBox value={search.query} onChange={search.setQuery} />
      </header>

      <main className="columns">
        <ResultsList search={search} isInShortlist={shortlist.has} onAdd={handleAdd} onRetry={search.retry} />
        <Shortlist items={items} onRemove={handleRemove} onMove={handleMove} />
      </main>

      <div className="visually-hidden" role="status" aria-live="polite">
        {announcement}
      </div>

      <PlayerBar />
    </>
  );
}

export default function App() {
  return (
    <PlayerProvider>
      <Board />
    </PlayerProvider>
  );
}

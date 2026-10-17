import { useEffect, useRef } from "react";
import type { Track } from "../types";
import { TrackRow } from "./TrackRow";

interface Props {
  items: Track[];
  onRemove: (track: Track) => void;
  onMove: (track: Track, direction: "up" | "down") => void;
}

/** Keeps keyboard focus on the same move button after a reorder re-renders the list. */
export function Shortlist({ items, onRemove, onMove }: Props) {
  const listRef = useRef<HTMLOListElement>(null);
  const pendingFocus = useRef<{ id: number; direction: "up" | "down" } | null>(null);

  useEffect(() => {
    const pending = pendingFocus.current;
    if (!pending || !listRef.current) return;
    pendingFocus.current = null;

    const find = (dir: string) =>
      listRef.current?.querySelector<HTMLButtonElement>(`button[data-track-id="${pending.id}"][data-action="${dir}"]`);
    const preferred = find(pending.direction);
    // At the top/bottom the same button becomes disabled, so use the opposite one.
    const target = preferred && !preferred.disabled ? preferred : find(pending.direction === "up" ? "down" : "up");
    target?.focus();
  }, [items]);

  const handleMove = (track: Track, direction: "up" | "down") => {
    pendingFocus.current = { id: track.id, direction };
    onMove(track, direction);
  };

  return (
    <section aria-labelledby="shortlist-heading" className="panel">
      <h2 id="shortlist-heading">
        Shortlist <span className="count">({items.length})</span>
      </h2>

      {items.length === 0 ? (
        <p className="status status-empty">Your shortlist is empty. Add tracks from the results.</p>
      ) : (
        <ol className="track-list" ref={listRef}>
          {items.map((track, index) => (
            <TrackRow
              key={track.id}
              variant="shortlist"
              track={track}
              isFirst={index === 0}
              isLast={index === items.length - 1}
              onRemove={onRemove}
              onMove={handleMove}
            />
          ))}
        </ol>
      )}
    </section>
  );
}

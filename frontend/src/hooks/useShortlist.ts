import { useCallback, useEffect, useState } from "react";
import type { Track } from "../types";

const STORAGE_KEY = "shortlist.v1";

function isTrack(v: unknown): v is Track {
  const t = v as Track;
  return (
    typeof t === "object" &&
    t !== null &&
    typeof t.id === "number" &&
    typeof t.name === "string" &&
    typeof t.artist === "string" &&
    typeof t.previewUrl === "string"
  );
}

function load(): Track[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.filter(isTrack) : [];
  } catch {
    return []; // corrupt JSON or storage blocked
  }
}

function save(items: Track[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
  } catch {
    // storage full or unavailable: the shortlist still works for this session
  }
}

/** Ordered shortlist persisted to localStorage. Array order is the user's order. */
export function useShortlist() {
  const [items, setItems] = useState<Track[]>(load);

  useEffect(() => {
    save(items);
  }, [items]);

  // Keep several open tabs in sync.
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key === STORAGE_KEY) setItems(load());
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  const add = useCallback((track: Track) => {
    setItems((prev) => (prev.some((t) => t.id === track.id) ? prev : [...prev, track]));
  }, []);

  const remove = useCallback((id: number) => {
    setItems((prev) => prev.filter((t) => t.id !== id));
  }, []);

  /** Moves the item at index `from` to index `to` (both clamped to the list). */
  const move = useCallback((from: number, to: number) => {
    setItems((prev) => {
      if (from < 0 || from >= prev.length || to < 0 || to >= prev.length || from === to) return prev;
      const next = [...prev];
      const [item] = next.splice(from, 1);
      next.splice(to, 0, item);
      return next;
    });
  }, []);

  const has = useCallback((id: number) => items.some((t) => t.id === id), [items]);

  return { items, add, remove, move, has };
}

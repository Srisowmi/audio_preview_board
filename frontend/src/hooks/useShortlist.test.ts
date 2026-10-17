import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import type { Track } from "../types";
import { useShortlist } from "./useShortlist";

const KEY = "shortlist.v1";

function track(id: number): Track {
  return {
    id,
    name: `Song ${id}`,
    artist: "Artist",
    album: "Album",
    artworkUrl: "https://img.example/a.jpg",
    durationMs: 180000,
    previewUrl: `https://audio.example/${id}.m4a`,
    genre: "Pop",
    releaseDate: "2020-01-01T00:00:00Z",
  };
}

const ids = (items: Track[]) => items.map((t) => t.id);
const stored = () => JSON.parse(localStorage.getItem(KEY) ?? "null") as Track[] | null;

describe("useShortlist", () => {
  beforeEach(() => localStorage.clear());

  it("adds tracks and ignores duplicates", () => {
    const { result } = renderHook(() => useShortlist());
    act(() => {
      result.current.add(track(1));
      result.current.add(track(2));
      result.current.add(track(1));
    });
    expect(ids(result.current.items)).toEqual([1, 2]);
    expect(result.current.has(1)).toBe(true);
    expect(result.current.has(3)).toBe(false);
  });

  it("removes a track", () => {
    const { result } = renderHook(() => useShortlist());
    act(() => {
      result.current.add(track(1));
      result.current.add(track(2));
    });
    act(() => result.current.remove(1));
    expect(ids(result.current.items)).toEqual([2]);
  });

  it("reorders tracks and ignores out-of-range moves", () => {
    const { result } = renderHook(() => useShortlist());
    act(() => {
      [1, 2, 3].forEach((id) => result.current.add(track(id)));
    });

    act(() => result.current.move(0, 2));
    expect(ids(result.current.items)).toEqual([2, 3, 1]);

    act(() => result.current.move(2, 1));
    expect(ids(result.current.items)).toEqual([2, 1, 3]);

    act(() => result.current.move(0, -1));
    act(() => result.current.move(2, 3));
    expect(ids(result.current.items)).toEqual([2, 1, 3]);
  });

  it("persists to localStorage and restores after a remount", () => {
    const first = renderHook(() => useShortlist());
    act(() => {
      first.result.current.add(track(1));
      first.result.current.add(track(2));
    });
    act(() => first.result.current.move(0, 1));
    expect(ids(stored() ?? [])).toEqual([2, 1]);
    first.unmount();

    const second = renderHook(() => useShortlist());
    expect(ids(second.result.current.items)).toEqual([2, 1]);
  });

  it("starts empty when stored data is corrupt", () => {
    localStorage.setItem(KEY, "{not json");
    const { result } = renderHook(() => useShortlist());
    expect(result.current.items).toEqual([]);
  });

  it("drops stored entries that are not valid tracks", () => {
    localStorage.setItem(KEY, JSON.stringify([track(1), { nope: true }]));
    const { result } = renderHook(() => useShortlist());
    expect(ids(result.current.items)).toEqual([1]);
  });
});

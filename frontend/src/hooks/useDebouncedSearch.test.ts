import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { SearchResponse, Track } from "../types";
import { useDebouncedSearch } from "./useDebouncedSearch";

vi.mock("../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/client")>();
  return { ...actual, searchTracks: vi.fn() };
});

import { searchTracks } from "../api/client";
const mockSearch = vi.mocked(searchTracks);

function response(query: string, name: string): SearchResponse {
  const t: Track = {
    id: 1,
    name,
    artist: "Artist",
    album: "Album",
    artworkUrl: "",
    durationMs: 1000,
    previewUrl: "https://audio.example/1.m4a",
    genre: "",
    releaseDate: "",
  };
  return { query, count: 1, cached: false, results: [t] };
}

function deferred<T>() {
  let resolve!: (v: T) => void;
  const promise = new Promise<T>((res) => (resolve = res));
  return { promise, resolve };
}

const DEBOUNCE = 350;

describe("useDebouncedSearch", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockSearch.mockReset();
  });
  afterEach(() => vi.useRealTimers());

  const settle = (ms = DEBOUNCE) => act(async () => void (await vi.advanceTimersByTimeAsync(ms)));

  it("debounces: only the last keystroke triggers a request", async () => {
    mockSearch.mockResolvedValue(response("cold", "Cold"));
    const { result } = renderHook(() => useDebouncedSearch(DEBOUNCE));

    act(() => result.current.setQuery("c"));
    await settle(100);
    act(() => result.current.setQuery("co"));
    await settle(100);
    act(() => result.current.setQuery("cold"));
    await settle();

    expect(mockSearch).toHaveBeenCalledTimes(1);
    expect(mockSearch.mock.calls[0][0]).toBe("cold");
    expect(result.current.status).toBe("success");
  });

  it("a slower earlier response never overwrites a later search", async () => {
    const slow = deferred<SearchResponse>();
    mockSearch.mockImplementation((q) =>
      q === "cold" ? slow.promise : Promise.resolve(response(q, "Coldplay result")),
    );
    const { result } = renderHook(() => useDebouncedSearch(DEBOUNCE));

    // First search starts and stays in flight.
    act(() => result.current.setQuery("cold"));
    await settle();
    expect(mockSearch).toHaveBeenCalledTimes(1);
    const firstSignal = mockSearch.mock.calls[0][1];

    // Second search starts and finishes first.
    act(() => result.current.setQuery("coldplay"));
    await settle();
    expect(firstSignal.aborted).toBe(true);
    expect(result.current.status).toBe("success");
    expect(result.current.results[0].name).toBe("Coldplay result");

    // The old response arrives late (even though it was aborted) and must be ignored.
    await act(async () => slow.resolve(response("cold", "Stale result")));
    expect(result.current.results[0].name).toBe("Coldplay result");
    expect(result.current.searchedQuery).toBe("coldplay");
  });

  it("shows an error state and recovers on retry", async () => {
    mockSearch.mockRejectedValueOnce(new Error("boom"));
    const { result } = renderHook(() => useDebouncedSearch(DEBOUNCE));

    act(() => result.current.setQuery("x"));
    await settle();
    expect(result.current.status).toBe("error");
    expect(result.current.error).toBe("boom");

    mockSearch.mockResolvedValueOnce(response("x", "Recovered"));
    act(() => result.current.retry());
    await settle();
    expect(result.current.status).toBe("success");
    expect(result.current.results[0].name).toBe("Recovered");
  });

  it("returns to idle for a blank query without calling the API", async () => {
    const { result } = renderHook(() => useDebouncedSearch(DEBOUNCE));
    act(() => result.current.setQuery("   "));
    await settle();
    expect(mockSearch).not.toHaveBeenCalled();
    expect(result.current.status).toBe("idle");
  });
});

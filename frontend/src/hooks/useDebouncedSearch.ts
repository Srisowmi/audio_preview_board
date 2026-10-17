import { useCallback, useEffect, useRef, useState } from "react";
import { isAbortError, searchTracks } from "../api/client";
import type { Track } from "../types";

export type SearchStatus = "idle" | "loading" | "success" | "error";

export interface SearchState {
  status: SearchStatus;
  /** The query the current results/error belong to. */
  searchedQuery: string;
  results: Track[];
  error: string | null;
}

const IDLE: SearchState = { status: "idle", searchedQuery: "", results: [], error: null };

/**
 * Debounced search. Stale responses are prevented two ways:
 *  1. the previous request is aborted via AbortController, and
 *  2. a request id guard drops any response that is not from the latest request.
 */
export function useDebouncedSearch(delayMs = 350) {
  const [query, setQuery] = useState("");
  const [state, setState] = useState<SearchState>(IDLE);
  const [attempt, setAttempt] = useState(0); // bumped by retry()
  const latestRequestId = useRef(0);

  useEffect(() => {
    const q = query.trim();
    const requestId = ++latestRequestId.current;

    if (!q) {
      setState(IDLE);
      return;
    }

    setState({ status: "loading", searchedQuery: q, results: [], error: null });

    const controller = new AbortController();
    const timer = setTimeout(async () => {
      try {
        const data = await searchTracks(q, controller.signal);
        if (requestId !== latestRequestId.current) return;
        setState({ status: "success", searchedQuery: q, results: data.results, error: null });
      } catch (err) {
        if (isAbortError(err) || requestId !== latestRequestId.current) return;
        const message = err instanceof Error ? err.message : "Something went wrong.";
        setState({ status: "error", searchedQuery: q, results: [], error: message });
      }
    }, delayMs);

    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [query, attempt, delayMs]);

  const retry = useCallback(() => setAttempt((n) => n + 1), []);

  return { query, setQuery, ...state, retry };
}

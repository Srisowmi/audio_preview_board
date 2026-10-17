import type { SearchResponse } from "../types";

const API_URL = (import.meta.env.VITE_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status?: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export function isAbortError(err: unknown): boolean {
  return err instanceof DOMException && err.name === "AbortError";
}

/** Searches tracks through our own backend. Rejects with AbortError when cancelled. */
export async function searchTracks(q: string, signal: AbortSignal): Promise<SearchResponse> {
  let res: Response;
  try {
    res = await fetch(`${API_URL}/api/search?q=${encodeURIComponent(q)}`, { signal });
  } catch (err) {
    if (isAbortError(err)) throw err;
    throw new ApiError("Could not reach the server. Check your connection and try again.");
  }

  if (!res.ok) {
    let message = `Search failed (status ${res.status}).`;
    try {
      const body = await res.json();
      if (typeof body?.error?.message === "string") message = body.error.message;
    } catch {
      // keep the generic message
    }
    throw new ApiError(message, res.status);
  }

  return (await res.json()) as SearchResponse;
}

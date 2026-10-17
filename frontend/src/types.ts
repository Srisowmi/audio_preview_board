export interface Track {
  id: number;
  name: string;
  artist: string;
  album: string;
  artworkUrl: string;
  durationMs: number;
  previewUrl: string;
  genre: string;
  releaseDate: string;
}

export interface SearchResponse {
  query: string;
  count: number;
  cached: boolean;
  results: Track[];
}

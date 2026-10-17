// Package model holds the API's own JSON types, independent of iTunes' shape.
package model

// Track is the normalised track returned to the frontend.
type Track struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	ArtworkURL  string `json:"artworkUrl"`
	DurationMs  int    `json:"durationMs"`
	PreviewURL  string `json:"previewUrl"`
	Genre       string `json:"genre"`
	ReleaseDate string `json:"releaseDate"`
}

// SearchResponse is the body of GET /api/search.
type SearchResponse struct {
	Query   string  `json:"query"`
	Count   int     `json:"count"`
	Cached  bool    `json:"cached"`
	Results []Track `json:"results"`
}

// TrackResponse is the body of GET /api/tracks/{id}.
type TrackResponse struct {
	Track
	Cached bool `json:"cached"`
}

// ErrorBody is the single error envelope used by every non-2xx response.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail carries a stable machine-readable code and a safe message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

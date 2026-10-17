// Package itunes is a thin client for the public iTunes Search API that
// returns the API's own normalised model.Track values.
package itunes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"audiopreview/internal/model"
)

const (
	searchLimit  = 25
	maxBodyBytes = 2 << 20
)

var (
	// ErrNotFound means the upstream has no (playable) track for the id.
	ErrNotFound = errors.New("itunes: track not found")
	// ErrRateLimited means the upstream rejected the call for rate limiting (403/429).
	ErrRateLimited = errors.New("itunes: rate limited")
	// ErrUnavailable means the upstream could not be reached or timed out.
	ErrUnavailable = errors.New("itunes: unavailable")
)

// UpstreamError is an unexpected upstream status or an undecodable body.
type UpstreamError struct {
	Status int
	Err    error
}

func (e *UpstreamError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("itunes: upstream error (status %d): %v", e.Status, e.Err)
	}
	return fmt.Sprintf("itunes: upstream error (status %d)", e.Status)
}

func (e *UpstreamError) Unwrap() error { return e.Err }

// Client is what the rest of the app needs from iTunes.
type Client interface {
	// Search returns up to 25 songs that have a preview URL.
	Search(ctx context.Context, term string) ([]model.Track, error)
	// Lookup returns one song, or ErrNotFound.
	Lookup(ctx context.Context, id int64) (model.Track, error)
}

// HTTPClient implements Client against the real (or a test) iTunes server.
type HTTPClient struct {
	baseURL string
	hc      *http.Client
}

// NewHTTPClient builds a client with an overall per-request timeout.
func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		hc:      &http.Client{Timeout: timeout},
	}
}

type response struct {
	ResultCount int      `json:"resultCount"`
	Results     []result `json:"results"`
}

type result struct {
	Kind            string `json:"kind"`
	TrackID         int64  `json:"trackId"`
	TrackName       string `json:"trackName"`
	ArtistName      string `json:"artistName"`
	CollectionName  string `json:"collectionName"`
	ArtworkURL100   string `json:"artworkUrl100"`
	TrackTimeMillis int    `json:"trackTimeMillis"`
	PreviewURL      string `json:"previewUrl"`
	Genre           string `json:"primaryGenreName"`
	ReleaseDate     string `json:"releaseDate"`
}

func (c *HTTPClient) Search(ctx context.Context, term string) ([]model.Track, error) {
	params := url.Values{}
	params.Set("term", term)
	params.Set("media", "music")
	params.Set("entity", "song")
	params.Set("limit", strconv.Itoa(searchLimit))

	resp, err := c.get(ctx, "/search", params)
	if err != nil {
		return nil, err
	}
	return normalise(resp.Results), nil
}

func (c *HTTPClient) Lookup(ctx context.Context, id int64) (model.Track, error) {
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))

	resp, err := c.get(ctx, "/lookup", params)
	if err != nil {
		return model.Track{}, err
	}
	tracks := normalise(resp.Results)
	if len(tracks) == 0 {
		return model.Track{}, ErrNotFound
	}
	return tracks[0], nil
}

func (c *HTTPClient) get(ctx context.Context, path string, params url.Values) (*response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	res, err := c.hc.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer res.Body.Close()

	switch {
	case res.StatusCode == http.StatusForbidden || res.StatusCode == http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case res.StatusCode != http.StatusOK:
		return nil, &UpstreamError{Status: res.StatusCode}
	}

	var out response
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBodyBytes)).Decode(&out); err != nil {
		return nil, &UpstreamError{Status: res.StatusCode, Err: err}
	}
	return &out, nil
}

// normalise converts upstream rows to Tracks, dropping anything that is not a
// song or has no preview URL. The result is never nil.
func normalise(rows []result) []model.Track {
	tracks := make([]model.Track, 0, len(rows))
	for _, r := range rows {
		if r.Kind != "song" || r.TrackID <= 0 || r.PreviewURL == "" {
			continue
		}
		tracks = append(tracks, model.Track{
			ID:          r.TrackID,
			Name:        r.TrackName,
			Artist:      r.ArtistName,
			Album:       r.CollectionName,
			ArtworkURL:  upscaleArtwork(r.ArtworkURL100),
			DurationMs:  r.TrackTimeMillis,
			PreviewURL:  r.PreviewURL,
			Genre:       r.Genre,
			ReleaseDate: r.ReleaseDate,
		})
	}
	return tracks
}

// upscaleArtwork turns the 100x100 artwork URL into a 300x300 one.
func upscaleArtwork(u string) string {
	return strings.Replace(u, "100x100bb", "300x300bb", 1)
}

package itunes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const searchBody = `{"resultCount":3,"results":[
 {"kind":"song","trackId":1,"trackName":"Yellow","artistName":"Coldplay","collectionName":"Parachutes",
  "artworkUrl100":"https://img.example/a/100x100bb.jpg","trackTimeMillis":269000,
  "previewUrl":"https://audio.example/1.m4a","primaryGenreName":"Alternative","releaseDate":"2000-07-10T07:00:00Z"},
 {"kind":"song","trackId":2,"trackName":"No Preview","artistName":"Coldplay","artworkUrl100":"https://img.example/b/100x100bb.jpg"},
 {"kind":"feature-movie","trackId":3,"trackName":"Not a song","previewUrl":"https://audio.example/3.m4a"}
]}`

func newServer(t *testing.T, status int, body string, onReq func(*http.Request)) *HTTPClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if onReq != nil {
			onReq(r)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewHTTPClient(srv.URL, 2*time.Second)
}

func TestSearchNormalisesAndFilters(t *testing.T) {
	var gotQuery map[string][]string
	c := newServer(t, 200, searchBody, func(r *http.Request) {
		gotQuery = r.URL.Query()
	})

	tracks, err := c.Search(context.Background(), "coldplay")
	if err != nil {
		t.Fatal(err)
	}

	if gotQuery["entity"][0] != "song" || gotQuery["limit"][0] != "25" || gotQuery["term"][0] != "coldplay" {
		t.Fatalf("unexpected upstream query: %v", gotQuery)
	}
	if len(tracks) != 1 {
		t.Fatalf("got %d tracks, want 1 (no-preview and non-song rows dropped)", len(tracks))
	}
	tr := tracks[0]
	if tr.ID != 1 || tr.Name != "Yellow" || tr.Artist != "Coldplay" || tr.Album != "Parachutes" || tr.DurationMs != 269000 {
		t.Fatalf("bad mapping: %+v", tr)
	}
	if tr.ArtworkURL != "https://img.example/a/300x300bb.jpg" {
		t.Fatalf("artwork not upscaled: %s", tr.ArtworkURL)
	}
}

func TestSearchEmptyIsNotNil(t *testing.T) {
	c := newServer(t, 200, `{"resultCount":0,"results":[]}`, nil)
	tracks, err := c.Search(context.Background(), "zzzz")
	if err != nil {
		t.Fatal(err)
	}
	if tracks == nil || len(tracks) != 0 {
		t.Fatalf("want empty non-nil slice, got %#v", tracks)
	}
}

func TestLookup(t *testing.T) {
	c := newServer(t, 200, searchBody, nil)
	tr, err := c.Lookup(context.Background(), 1)
	if err != nil || tr.ID != 1 {
		t.Fatalf("got %+v, %v", tr, err)
	}

	empty := newServer(t, 200, `{"resultCount":0,"results":[]}`, nil)
	if _, err := empty.Lookup(context.Background(), 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestErrorMapping(t *testing.T) {
	for _, status := range []int{403, 429} {
		c := newServer(t, status, ``, nil)
		if _, err := c.Search(context.Background(), "x"); !errors.Is(err, ErrRateLimited) {
			t.Errorf("status %d: got %v, want ErrRateLimited", status, err)
		}
	}

	c := newServer(t, 500, ``, nil)
	var ue *UpstreamError
	if _, err := c.Search(context.Background(), "x"); !errors.As(err, &ue) || ue.Status != 500 {
		t.Errorf("500: got %v, want UpstreamError(500)", err)
	}

	bad := newServer(t, 200, `not json`, nil)
	if _, err := bad.Search(context.Background(), "x"); !errors.As(err, &ue) {
		t.Errorf("bad json: got %v, want UpstreamError", err)
	}
}

func TestTimeoutIsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, 50*time.Millisecond)
	if _, err := c.Search(context.Background(), "x"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v, want ErrUnavailable", err)
	}
}

func TestContextCancelled(t *testing.T) {
	c := newServer(t, 200, searchBody, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Search(ctx, "x"); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

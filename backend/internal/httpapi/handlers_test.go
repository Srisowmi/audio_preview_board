package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"audiopreview/internal/itunes"
	"audiopreview/internal/model"
)

type fakeService struct {
	tracks []model.Track
	cached bool
	err    error
	gotQ   string
}

func (f *fakeService) Search(ctx context.Context, q string) ([]model.Track, bool, error) {
	f.gotQ = q
	return f.tracks, f.cached, f.err
}

func (f *fakeService) Track(ctx context.Context, id int64) (model.Track, bool, error) {
	if f.err != nil {
		return model.Track{}, false, f.err
	}
	return model.Track{ID: id, Name: "Yellow"}, f.cached, nil
}

const origin = "https://app.example.com"

func do(t *testing.T, svc MusicService, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewRouter(svc, origin, slog.New(slog.NewTextHandler(io.Discard, nil)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body model.ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body is not JSON: %q", rec.Body.String())
	}
	return body.Error.Code
}

func TestHealth(t *testing.T) {
	rec := do(t, &fakeService{}, "GET", "/api/health")
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestSearchSuccess(t *testing.T) {
	svc := &fakeService{tracks: []model.Track{{ID: 1, Name: "Yellow", Artist: "Coldplay"}}, cached: true}
	rec := do(t, svc, "GET", "/api/search?q=%20Coldplay%20")

	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body model.SearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Query != "coldplay" || body.Count != 1 || !body.Cached || body.Results[0].Name != "Yellow" {
		t.Fatalf("unexpected body: %+v", body)
	}
	if svc.gotQ != "Coldplay" {
		t.Fatalf("service got %q, want trimmed query", svc.gotQ)
	}
}

func TestSearchEmptyResultsIsArray(t *testing.T) {
	rec := do(t, &fakeService{tracks: []model.Track{}}, "GET", "/api/search?q=zzz")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"results":[]`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestSearchValidation(t *testing.T) {
	for _, target := range []string{
		"/api/search",
		"/api/search?q=",
		"/api/search?q=%20%20",
		"/api/search?q=" + strings.Repeat("a", 101),
	} {
		rec := do(t, &fakeService{}, "GET", target)
		if rec.Code != 400 || errCode(t, rec) != "invalid_query" {
			t.Errorf("%s: got %d %s", target, rec.Code, rec.Body.String())
		}
	}
}

func TestTrackSuccess(t *testing.T) {
	rec := do(t, &fakeService{}, "GET", "/api/tracks/123")
	if rec.Code != 200 {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	var body model.TrackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.ID != 123 {
		t.Fatalf("bad body %s (%v)", rec.Body.String(), err)
	}
	if !strings.Contains(rec.Body.String(), `"name":"Yellow"`) {
		t.Fatalf("track fields should be flattened: %s", rec.Body.String())
	}
}

func TestTrackInvalidID(t *testing.T) {
	for _, id := range []string{"abc", "0", "-5", "1.5"} {
		rec := do(t, &fakeService{}, "GET", "/api/tracks/"+id)
		if rec.Code != 400 || errCode(t, rec) != "invalid_id" {
			t.Errorf("id %q: got %d %s", id, rec.Code, rec.Body.String())
		}
	}
}

func TestTrackNotFound(t *testing.T) {
	rec := do(t, &fakeService{err: itunes.ErrNotFound}, "GET", "/api/tracks/999")
	if rec.Code != 404 || errCode(t, rec) != "not_found" {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestUpstreamErrorMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
		retry  bool
	}{
		{itunes.ErrRateLimited, 503, "upstream_unavailable", true},
		{itunes.ErrUnavailable, 503, "upstream_unavailable", true},
		{&itunes.UpstreamError{Status: 500}, 502, "upstream_error", false},
		{context.DeadlineExceeded, 500, "internal", false},
	}
	for _, tc := range cases {
		rec := do(t, &fakeService{err: tc.err}, "GET", "/api/search?q=x")
		if rec.Code != tc.status || errCode(t, rec) != tc.code {
			t.Errorf("%v: got %d %s", tc.err, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Retry-After") != ""; got != tc.retry {
			t.Errorf("%v: Retry-After present=%v, want %v", tc.err, got, tc.retry)
		}
	}
}

func TestCORS(t *testing.T) {
	rec := do(t, &fakeService{}, "GET", "/api/health")
	if rec.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatalf("missing CORS header: %v", rec.Header())
	}

	pre := do(t, &fakeService{}, "OPTIONS", "/api/search")
	if pre.Code != http.StatusNoContent ||
		pre.Header().Get("Access-Control-Allow-Origin") != origin ||
		!strings.Contains(pre.Header().Get("Access-Control-Allow-Methods"), "GET") {
		t.Fatalf("bad preflight: %d %v", pre.Code, pre.Header())
	}
}

func TestMethodAndRouteErrors(t *testing.T) {
	rec := do(t, &fakeService{}, "POST", "/api/search?q=x")
	if rec.Code != 405 || errCode(t, rec) != "method_not_allowed" {
		t.Errorf("POST: got %d %s", rec.Code, rec.Body.String())
	}

	rec = do(t, &fakeService{}, "GET", "/api/nope")
	if rec.Code != 404 || errCode(t, rec) != "not_found" {
		t.Errorf("unknown route: got %d %s", rec.Code, rec.Body.String())
	}

	rec = do(t, &fakeService{}, "GET", "/api/tracks/")
	if rec.Code != 404 {
		t.Errorf("empty id: got %d, want 404", rec.Code)
	}
}

type panicService struct{ fakeService }

func (p *panicService) Search(ctx context.Context, q string) ([]model.Track, bool, error) {
	panic("boom")
}

func TestPanicRecovered(t *testing.T) {
	rec := do(t, &panicService{}, "GET", "/api/search?q=x")
	if rec.Code != 500 || errCode(t, rec) != "internal" {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

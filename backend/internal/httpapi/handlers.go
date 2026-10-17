package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"audiopreview/internal/model"
	"audiopreview/internal/service"
)

const maxQueryLen = 100

// MusicService is what the handlers need; *service.Service satisfies it.
type MusicService interface {
	Search(ctx context.Context, q string) ([]model.Track, bool, error)
	Track(ctx context.Context, id int64) (model.Track, bool, error)
}

type handlers struct {
	svc MusicService
	log *slog.Logger
}

func (h *handlers) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "q is required")
		return
	}
	if utf8.RuneCountInString(q) > maxQueryLen {
		writeError(w, http.StatusBadRequest, "invalid_query", "q must be at most 100 characters")
		return
	}

	tracks, cached, err := h.svc.Search(r.Context(), q)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, model.SearchResponse{
		Query:   service.NormaliseQuery(q),
		Count:   len(tracks),
		Cached:  cached,
		Results: tracks,
	})
}

func (h *handlers) track(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_id", "track id must be a positive number")
		return
	}

	t, cached, err := h.svc.Track(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, model.TrackResponse{Track: t, Cached: cached})
}

// fallback gives unknown paths and wrong methods a JSON body instead of the mux's plain text.
func (h *handlers) fallback(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	known := p == "/api/health" || p == "/api/search" || strings.HasPrefix(p, "/api/tracks/")
	if known && r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, OPTIONS")
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "only GET is supported")
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "no such route")
}

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	status, code, msg := classify(err)
	if status >= 500 {
		h.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "status", status, "err", err)
	}
	if status == http.StatusServiceUnavailable {
		w.Header().Set("Retry-After", "30")
	}
	writeError(w, status, code, msg)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, model.ErrorBody{Error: model.ErrorDetail{Code: code, Message: message}})
}

// writeJSON marshals first so an encoding failure can still produce a clean 500.
func writeJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		status = http.StatusInternalServerError
		body = []byte(`{"error":{"code":"internal","message":"internal server error"}}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

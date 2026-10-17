// Package httpapi wires the HTTP routes, middleware and handlers.
package httpapi

import (
	"log/slog"
	"net/http"
)

// NewRouter builds the full handler: routes wrapped in recover, logging and CORS.
func NewRouter(svc MusicService, allowedOrigin string, log *slog.Logger) http.Handler {
	h := &handlers{svc: svc, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("GET /api/search", h.search)
	mux.HandleFunc("GET /api/tracks/{id}", h.track)
	mux.HandleFunc("/", h.fallback)

	return chain(mux, recoverer(log), requestLogger(log), cors(allowedOrigin))
}

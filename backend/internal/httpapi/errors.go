package httpapi

import (
	"context"
	"errors"
	"net/http"

	"audiopreview/internal/itunes"
)

// classify maps a service error to an HTTP status, stable code and safe message.
func classify(err error) (status int, code, message string) {
	var upstream *itunes.UpstreamError
	switch {
	case errors.Is(err, itunes.ErrNotFound):
		return http.StatusNotFound, "not_found", "track not found"
	case errors.Is(err, itunes.ErrRateLimited), errors.Is(err, itunes.ErrUnavailable):
		return http.StatusServiceUnavailable, "upstream_unavailable", "music provider is busy or unreachable, retry shortly"
	case errors.As(err, &upstream):
		return http.StatusBadGateway, "upstream_error", "music provider returned an invalid response"
	case errors.Is(err, context.Canceled):
		return http.StatusServiceUnavailable, "cancelled", "request cancelled"
	default:
		return http.StatusInternalServerError, "internal", "internal server error"
	}
}

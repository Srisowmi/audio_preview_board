// Package service combines the iTunes client with caching.
package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"audiopreview/internal/cache"
	"audiopreview/internal/itunes"
	"audiopreview/internal/model"
)

const maxCacheEntries = 1000

// Service serves searches and track lookups, hitting iTunes only on cache misses.
type Service struct {
	client itunes.Client
	search *cache.Cache[[]model.Track]
	tracks *cache.Cache[model.Track]
	sf     singleflight.Group
}

// New builds a Service whose cache entries live for ttl.
func New(client itunes.Client, ttl time.Duration) *Service {
	return &Service{
		client: client,
		search: cache.New[[]model.Track](ttl, maxCacheEntries),
		tracks: cache.New[model.Track](ttl, maxCacheEntries),
	}
}

// StartJanitor periodically removes expired cache entries until ctx is cancelled.
func (s *Service) StartJanitor(ctx context.Context, interval time.Duration) {
	s.search.StartJanitor(ctx, interval)
	s.tracks.StartJanitor(ctx, interval)
}

// NormaliseQuery is the canonical form used for both the cache key and the upstream call.
func NormaliseQuery(q string) string {
	return strings.ToLower(strings.TrimSpace(q))
}

// Search returns tracks for q and whether they came from the cache.
// Callers must not modify the returned slice.
func (s *Service) Search(ctx context.Context, q string) ([]model.Track, bool, error) {
	term := NormaliseQuery(q)
	key := "search:" + term

	if tracks, ok := s.search.Get(key); ok {
		return tracks, true, nil
	}

	// Collapse concurrent misses for the same key into one upstream call. The
	// shared call is detached from any single caller's cancellation; the HTTP
	// client's own timeout still bounds it.
	v, err, _ := s.sf.Do(key, func() (any, error) {
		tracks, err := s.client.Search(context.WithoutCancel(ctx), term)
		if err != nil {
			return nil, err
		}
		s.search.Set(key, tracks)
		for _, t := range tracks {
			s.tracks.Set(trackKey(t.ID), t)
		}
		return tracks, nil
	})
	if err != nil {
		return nil, false, err
	}
	return v.([]model.Track), false, nil
}

// Track returns one track and whether it came from the cache.
func (s *Service) Track(ctx context.Context, id int64) (model.Track, bool, error) {
	key := trackKey(id)

	if t, ok := s.tracks.Get(key); ok {
		return t, true, nil
	}

	v, err, _ := s.sf.Do(key, func() (any, error) {
		t, err := s.client.Lookup(context.WithoutCancel(ctx), id)
		if err != nil {
			return nil, err
		}
		s.tracks.Set(key, t)
		return t, nil
	})
	if err != nil {
		return model.Track{}, false, err
	}
	return v.(model.Track), false, nil
}

func trackKey(id int64) string { return "track:" + strconv.FormatInt(id, 10) }

// Package cache provides a small concurrency-safe in-memory TTL cache.
package cache

import (
	"context"
	"sync"
	"time"
)

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

// Cache is a TTL cache with a maximum size. Stored values must be treated as
// immutable by callers because they are shared between goroutines.
type Cache[V any] struct {
	mu    sync.RWMutex
	items map[string]entry[V]
	ttl   time.Duration
	max   int
	now   func() time.Time
}

// New creates a cache whose entries live for ttl and which holds at most max entries.
func New[V any](ttl time.Duration, max int) *Cache[V] {
	return &Cache[V]{
		items: make(map[string]entry[V]),
		ttl:   ttl,
		max:   max,
		now:   time.Now,
	}
}

// Get returns the value for key if present and not expired.
func (c *Cache[V]) Get(key string) (V, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()

	var zero V
	if !ok {
		return zero, false
	}
	if c.now().Before(e.expiresAt) {
		return e.value, true
	}

	// Expired: remove it, re-checking under the write lock since another
	// goroutine may have refreshed the key in the meantime.
	c.mu.Lock()
	if cur, ok := c.items[key]; ok && !c.now().Before(cur.expiresAt) {
		delete(c.items, key)
	}
	c.mu.Unlock()
	return zero, false
}

// Set stores value under key, evicting entries if the cache is full.
func (c *Cache[V]) Set(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.items[key]; !exists && len(c.items) >= c.max {
		c.evictLocked()
	}
	c.items[key] = entry[V]{value: value, expiresAt: c.now().Add(c.ttl)}
}

// Len returns the number of stored entries, including expired ones not yet swept.
func (c *Cache[V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Sweep removes all expired entries.
func (c *Cache[V]) Sweep() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sweepLocked()
}

// StartJanitor sweeps expired entries every interval until ctx is cancelled.
func (c *Cache[V]) StartJanitor(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				c.Sweep()
			}
		}
	}()
}

func (c *Cache[V]) sweepLocked() {
	now := c.now()
	for k, e := range c.items {
		if !now.Before(e.expiresAt) {
			delete(c.items, k)
		}
	}
}

// evictLocked frees space: expired entries first, then the entry closest to expiry.
func (c *Cache[V]) evictLocked() {
	c.sweepLocked()
	if len(c.items) < c.max {
		return
	}
	var oldestKey string
	var oldest time.Time
	first := true
	for k, e := range c.items {
		if first || e.expiresAt.Before(oldest) {
			oldestKey, oldest, first = k, e.expiresAt, false
		}
	}
	delete(c.items, oldestKey)
}

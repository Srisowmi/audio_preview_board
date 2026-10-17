package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (f *fakeClock) now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

func (f *fakeClock) advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
}

func newTestCache(ttl time.Duration, max int) (*Cache[string], *fakeClock) {
	clk := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	c := New[string](ttl, max)
	c.now = clk.now
	return c, clk
}

func TestGetSet(t *testing.T) {
	c, _ := newTestCache(time.Hour, 10)
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected miss on empty cache")
	}
	c.Set("a", "1")
	if v, ok := c.Get("a"); !ok || v != "1" {
		t.Fatalf("got %q, %v; want 1, true", v, ok)
	}
}

func TestExpiry(t *testing.T) {
	c, clk := newTestCache(time.Hour, 10)
	c.Set("a", "1")

	clk.advance(59 * time.Minute)
	if _, ok := c.Get("a"); !ok {
		t.Fatal("entry should still be valid before TTL")
	}

	clk.advance(2 * time.Minute)
	if _, ok := c.Get("a"); ok {
		t.Fatal("entry should have expired")
	}
	if c.Len() != 0 {
		t.Fatalf("expired entry should be removed on read, len=%d", c.Len())
	}
}

func TestSweepRemovesExpired(t *testing.T) {
	c, clk := newTestCache(time.Hour, 10)
	c.Set("a", "1")
	c.Set("b", "2")
	clk.advance(2 * time.Hour)
	c.Sweep()
	if c.Len() != 0 {
		t.Fatalf("len=%d, want 0", c.Len())
	}
}

func TestEvictsWhenFull(t *testing.T) {
	c, clk := newTestCache(time.Hour, 2)
	c.Set("old", "1")
	clk.advance(time.Minute)
	c.Set("newer", "2")
	clk.advance(time.Minute)
	c.Set("newest", "3") // full: closest-to-expiry ("old") goes

	if c.Len() != 2 {
		t.Fatalf("len=%d, want 2", c.Len())
	}
	if _, ok := c.Get("old"); ok {
		t.Fatal("oldest entry should have been evicted")
	}
	if _, ok := c.Get("newest"); !ok {
		t.Fatal("newest entry should be present")
	}
}

func TestOverwriteDoesNotEvict(t *testing.T) {
	c, _ := newTestCache(time.Hour, 2)
	c.Set("a", "1")
	c.Set("b", "2")
	c.Set("a", "3")
	if v, _ := c.Get("a"); v != "3" {
		t.Fatalf("got %q, want 3", v)
	}
	if _, ok := c.Get("b"); !ok {
		t.Fatal("overwriting an existing key must not evict others")
	}
}

// Run with -race: hammers the cache from many goroutines.
func TestConcurrentAccess(t *testing.T) {
	c := New[string](time.Hour, 50)
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				k := fmt.Sprintf("k%d", i%80)
				c.Set(k, "v")
				c.Get(k)
				if i%100 == 0 {
					c.Sweep()
				}
			}
		}(g)
	}
	wg.Wait()
	if c.Len() > 50 {
		t.Fatalf("len=%d exceeds max 50", c.Len())
	}
}

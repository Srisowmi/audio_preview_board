package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"audiopreview/internal/itunes"
	"audiopreview/internal/model"
)

type fakeClient struct {
	searchCalls atomic.Int32
	lookupCalls atomic.Int32
	tracks      []model.Track
	err         error
}

func (f *fakeClient) Search(ctx context.Context, term string) ([]model.Track, error) {
	f.searchCalls.Add(1)
	return f.tracks, f.err
}

func (f *fakeClient) Lookup(ctx context.Context, id int64) (model.Track, error) {
	f.lookupCalls.Add(1)
	if f.err != nil {
		return model.Track{}, f.err
	}
	for _, t := range f.tracks {
		if t.ID == id {
			return t, nil
		}
	}
	return model.Track{}, itunes.ErrNotFound
}

func newFake() *fakeClient {
	return &fakeClient{tracks: []model.Track{{ID: 1, Name: "Yellow"}, {ID: 2, Name: "Clocks"}}}
}

func TestSearchCachesAndNormalisesKey(t *testing.T) {
	fc := newFake()
	s := New(fc, time.Hour)
	ctx := context.Background()

	_, cached, err := s.Search(ctx, "Coldplay")
	if err != nil || cached {
		t.Fatalf("first call: cached=%v err=%v, want miss", cached, err)
	}
	_, cached, err = s.Search(ctx, "  coldPLAY  ")
	if err != nil || !cached {
		t.Fatalf("second call: cached=%v err=%v, want hit", cached, err)
	}
	if n := fc.searchCalls.Load(); n != 1 {
		t.Fatalf("upstream called %d times, want 1", n)
	}
}

func TestSearchSeedsTrackCache(t *testing.T) {
	fc := newFake()
	s := New(fc, time.Hour)
	ctx := context.Background()

	if _, _, err := s.Search(ctx, "coldplay"); err != nil {
		t.Fatal(err)
	}
	tr, cached, err := s.Track(ctx, 2)
	if err != nil || !cached || tr.Name != "Clocks" {
		t.Fatalf("got %+v cached=%v err=%v, want cached Clocks", tr, cached, err)
	}
	if fc.lookupCalls.Load() != 0 {
		t.Fatal("lookup should not have been called")
	}
}

func TestTrackMissThenHit(t *testing.T) {
	fc := newFake()
	s := New(fc, time.Hour)
	ctx := context.Background()

	if _, cached, err := s.Track(ctx, 1); err != nil || cached {
		t.Fatalf("first: cached=%v err=%v", cached, err)
	}
	if _, cached, err := s.Track(ctx, 1); err != nil || !cached {
		t.Fatalf("second: cached=%v err=%v", cached, err)
	}
	if fc.lookupCalls.Load() != 1 {
		t.Fatalf("lookup calls=%d, want 1", fc.lookupCalls.Load())
	}
}

func TestErrorsAreNotCached(t *testing.T) {
	fc := newFake()
	fc.err = itunes.ErrRateLimited
	s := New(fc, time.Hour)
	ctx := context.Background()

	if _, _, err := s.Search(ctx, "x"); !errors.Is(err, itunes.ErrRateLimited) {
		t.Fatalf("got %v", err)
	}

	fc.err = nil
	if _, cached, err := s.Search(ctx, "x"); err != nil || cached {
		t.Fatalf("after recovery: cached=%v err=%v, want fresh success", cached, err)
	}
	if fc.searchCalls.Load() != 2 {
		t.Fatalf("calls=%d, want 2", fc.searchCalls.Load())
	}
}

func TestTrackNotFound(t *testing.T) {
	s := New(newFake(), time.Hour)
	if _, _, err := s.Track(context.Background(), 404); !errors.Is(err, itunes.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestExpiredEntryRefetches(t *testing.T) {
	fc := newFake()
	s := New(fc, 20*time.Millisecond)
	ctx := context.Background()

	_, _, _ = s.Search(ctx, "x")
	time.Sleep(40 * time.Millisecond)
	if _, cached, _ := s.Search(ctx, "x"); cached {
		t.Fatal("expected miss after TTL")
	}
	if fc.searchCalls.Load() != 2 {
		t.Fatalf("calls=%d, want 2", fc.searchCalls.Load())
	}
}

// Run with -race.
func TestConcurrentSearches(t *testing.T) {
	s := New(newFake(), time.Hour)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := s.Search(context.Background(), "coldplay"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

package cache

import (
	"sort"
	"sync/atomic"
	"time"

	"github.com/ot4/search-trends/internal/domain/trending"
)

type State struct {
	GeneratedAt   time.Time
	WindowSeconds int
	Snapshots     map[int]trending.Snapshot
}

type Store struct {
	ptr atomic.Pointer[State]
}

func NewStore(windowSeconds int) *Store {
	store := &Store{}
	store.ptr.Store(&State{
		GeneratedAt:   time.Unix(0, 0).UTC(),
		WindowSeconds: windowSeconds,
		Snapshots:     map[int]trending.Snapshot{},
	})
	return store
}

func (s *Store) Publish(generatedAt time.Time, windowSeconds int, snapshots map[int][]trending.Item) {
	batch := make(map[int]trending.Snapshot, len(snapshots))

	for limit, items := range snapshots {
		batch[limit] = trending.NewSnapshot(generatedAt, windowSeconds, limit, items)
	}

	s.ptr.Store(&State{
		GeneratedAt:   generatedAt.UTC(),
		WindowSeconds: windowSeconds,
		Snapshots:     batch,
	})
}

func (s *Store) Get(limit int) (trending.Snapshot, bool) {
	current := s.ptr.Load()
	if current == nil {
		return trending.Snapshot{}, false
	}

	snapshot, ok := current.Snapshots[limit]
	return snapshot, ok
}

func (s *Store) AvailableLimits() []int {
	current := s.ptr.Load()
	if current == nil {
		return nil
	}

	limits := make([]int, 0, len(current.Snapshots))
	for limit := range current.Snapshots {
		limits = append(limits, limit)
	}

	sort.Ints(limits)
	return limits
}

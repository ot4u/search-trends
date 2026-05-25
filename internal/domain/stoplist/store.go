package stoplist

import (
	"sort"
	"sync/atomic"
)

type state struct {
	entries map[string]struct{}
}
type Store struct {
	ptr atomic.Pointer[state]
}

func NewStore(initial []string) *Store {
	entries := make(map[string]struct{}, len(initial))
	for _, word := range initial {
		if word == "" {
			continue
		}
		entries[word] = struct{}{}
	}

	store := &Store{}
	store.ptr.Store(&state{entries: entries})
	return store
}

func (s *Store) load() *state {
	current := s.ptr.Load()
	if current != nil {
		return current
	}

	empty := &state{entries: map[string]struct{}{}}
	s.ptr.Store(empty)
	return empty
}
func (s *Store) Has(word string) bool {
	_, ok := s.load().entries[word]
	return ok
}
func (s *Store) ContainsAny(words []string) bool {
	if len(words) == 0 {
		return false
	}

	entries := s.load().entries
	for _, word := range words {
		if _, ok := entries[word]; ok {
			return true
		}
	}

	return false
}
func (s *Store) Add(word string) bool {
	if word == "" {
		return false
	}

	for {
		current := s.load()
		if _, ok := current.entries[word]; ok {
			return false
		}

		nextEntries := cloneEntries(current.entries)
		nextEntries[word] = struct{}{}
		next := &state{entries: nextEntries}

		if s.ptr.CompareAndSwap(current, next) {
			return true
		}
	}
}
func (s *Store) Remove(word string) bool {
	if word == "" {
		return false
	}

	for {
		current := s.load()
		if _, ok := current.entries[word]; !ok {
			return false
		}

		nextEntries := cloneEntries(current.entries)
		delete(nextEntries, word)
		next := &state{entries: nextEntries}

		if s.ptr.CompareAndSwap(current, next) {
			return true
		}
	}
}
func (s *Store) List() []string {
	entries := s.load().entries
	items := make([]string, 0, len(entries))

	for word := range entries {
		items = append(items, word)
	}

	sort.Strings(items)
	return items
}
func (s *Store) Len() int {
	return len(s.load().entries)
}

func cloneEntries(entries map[string]struct{}) map[string]struct{} {
	cloned := make(map[string]struct{}, len(entries)+1)
	for word := range entries {
		cloned[word] = struct{}{}
	}
	return cloned
}

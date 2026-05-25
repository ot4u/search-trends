package trending

import "time"

type bucket struct {
	second int64
	counts map[string]uint64
}
type Window struct {
	size          int
	cursor        int
	currentSecond int64
	buckets       []bucket
	global        map[string]uint64
}

func NewWindow(size int, now time.Time) *Window {
	if size <= 0 {
		panic("window size must be positive")
	}

	currentSecond := truncateToSecond(now).Unix()
	buckets := make([]bucket, size)

	for i := range buckets {
		buckets[i] = bucket{
			second: currentSecond - int64(size-1-i),
			counts: make(map[string]uint64),
		}
	}

	return &Window{
		size:          size,
		cursor:        size - 1,
		currentSecond: currentSecond,
		buckets:       buckets,
		global:        make(map[string]uint64),
	}
}

func truncateToSecond(ts time.Time) time.Time {
	return ts.UTC().Truncate(time.Second)
}
func (w *Window) WindowSeconds() int {
	return w.size
}
func (w *Window) CurrentSecond() int64 {
	return w.currentSecond
}
func (w *Window) UniqueCount() int {
	return len(w.global)
}
func (w *Window) HasQuery(query string) bool {
	_, ok := w.global[query]
	return ok
}
func (w *Window) TotalScore(query string) uint64 {
	return w.global[query]
}
func (w *Window) Counts() map[string]uint64 {
	return w.global
}
func (w *Window) AdvanceTo(now time.Time) int {
	targetSecond := truncateToSecond(now).Unix()
	if targetSecond <= w.currentSecond {
		return 0
	}

	delta := targetSecond - w.currentSecond
	if delta >= int64(w.size) {
		w.reset(targetSecond)
		return w.size
	}

	for step := int64(0); step < delta; step++ {
		w.cursor = (w.cursor + 1) % w.size
		expired := &w.buckets[w.cursor]

		for query, score := range expired.counts {
			current := w.global[query]
			if current <= score {
				delete(w.global, query)
				continue
			}
			w.global[query] = current - score
		}

		clear(expired.counts)
		w.currentSecond++
		expired.second = w.currentSecond
	}

	return int(delta)
}

func (w *Window) reset(targetSecond int64) {
	clear(w.global)
	w.currentSecond = targetSecond
	w.cursor = w.size - 1

	for i := range w.buckets {
		clear(w.buckets[i].counts)
		w.buckets[i].second = targetSecond - int64(w.size-1-i)
	}
}
func (w *Window) AddAt(ts time.Time, query string, weight uint64) bool {
	if query == "" || weight == 0 {
		return false
	}

	eventSecond := truncateToSecond(ts).Unix()
	if eventSecond > w.currentSecond {
		eventSecond = w.currentSecond
	}

	delta := w.currentSecond - eventSecond
	if delta < 0 || delta >= int64(w.size) {
		return false
	}

	index := w.cursor - int(delta)
	if index < 0 {
		index += w.size
	}

	bucketCounts := w.buckets[index].counts
	bucketCounts[query] += weight
	w.global[query] += weight

	return true
}

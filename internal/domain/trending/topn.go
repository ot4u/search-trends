package trending

import (
	"container/heap"
	"sort"
)

type heapItem struct {
	query string
	score uint64
}

type minHeap []heapItem

func (h minHeap) Len() int { return len(h) }

func (h minHeap) Less(i, j int) bool {
	if h[i].score == h[j].score {
		return h[i].query > h[j].query
	}

	return h[i].score < h[j].score
}

func (h minHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *minHeap) Push(x any) {
	*h = append(*h, x.(heapItem))
}

func (h *minHeap) Pop() any {
	old := *h
	item := old[len(old)-1]
	*h = old[:len(old)-1]
	return item
}

func better(candidate, current heapItem) bool {
	if candidate.score == current.score {
		return candidate.query < current.query
	}

	return candidate.score > current.score
}
func BuildTopN(counts map[string]uint64, limit int) []Item {
	if limit <= 0 || len(counts) == 0 {
		return nil
	}

	if limit > len(counts) {
		limit = len(counts)
	}

	h := make(minHeap, 0, limit)

	for query, score := range counts {
		candidate := heapItem{query: query, score: score}

		if len(h) < limit {
			heap.Push(&h, candidate)
			continue
		}

		if better(candidate, h[0]) {
			h[0] = candidate
			heap.Fix(&h, 0)
		}
	}

	items := make([]Item, 0, len(h))

	for _, item := range h {
		items = append(items, Item{
			Query: item.query,
			Score: item.score,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Score == items[j].Score {
			return items[i].Query < items[j].Query
		}

		return items[i].Score > items[j].Score
	})

	for i := range items {
		items[i].Rank = i + 1
	}

	return items
}

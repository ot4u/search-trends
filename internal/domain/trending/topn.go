package trending

type heapItem struct {
	query string
	score uint64
}

func lessHeap(a, b heapItem) bool {
	if a.score == b.score {
		return a.query > b.query
	}

	return a.score < b.score
}

func better(candidate, current heapItem) bool {
	if candidate.score == current.score {
		return candidate.query < current.query
	}

	return candidate.score > current.score
}

func siftDown(h []heapItem, i, n int) {
	for {
		smallest := i
		left := 2*i + 1
		right := left + 1

		if left < n && lessHeap(h[left], h[smallest]) {
			smallest = left
		}
		if right < n && lessHeap(h[right], h[smallest]) {
			smallest = right
		}
		if smallest == i {
			return
		}

		h[i], h[smallest] = h[smallest], h[i]
		i = smallest
	}
}

func initMinHeap(h []heapItem) {
	for i := len(h)/2 - 1; i >= 0; i-- {
		siftDown(h, i, len(h))
	}
}

func popMin(h *[]heapItem) heapItem {
	n := len(*h)
	item := (*h)[0]
	(*h)[0] = (*h)[n-1]
	*h = (*h)[:n-1]
	if len(*h) > 0 {
		siftDown(*h, 0, len(*h))
	}
	return item
}

func BuildTopN(counts map[string]uint64, limit int) []Item {
	if limit <= 0 || len(counts) == 0 {
		return nil
	}

	if limit > len(counts) {
		limit = len(counts)
	}

	h := make([]heapItem, 0, limit)
	heapified := false

	for query, score := range counts {
		candidate := heapItem{query: query, score: score}

		if len(h) < limit {
			h = append(h, candidate)
			continue
		}

		if !heapified {
			initMinHeap(h)
			heapified = true
		}

		if better(candidate, h[0]) {
			h[0] = candidate
			siftDown(h, 0, len(h))
		}
	}

	if len(h) == 0 {
		return nil
	}

	if !heapified {
		initMinHeap(h)
	}

	items := make([]Item, len(h))
	for i := len(h) - 1; i >= 0; i-- {
		top := popMin(&h)
		items[i] = Item{
			Query: top.query,
			Score: top.score,
			Rank:  i + 1,
		}
	}

	return items
}

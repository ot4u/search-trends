package trending

import "time"

type Item struct {
	Query string `json:"query"`
	Score uint64 `json:"score"`
	Rank  int    `json:"rank"`
}
type Snapshot struct {
	GeneratedAt   time.Time `json:"generated_at"`
	WindowSeconds int       `json:"window_seconds"`
	Limit         int       `json:"limit"`
	Items         []Item    `json:"items"`
}

func NewSnapshot(generatedAt time.Time, windowSeconds, limit int, items []Item) Snapshot {
	cloned := make([]Item, len(items))
	copy(cloned, items)

	return Snapshot{
		GeneratedAt:   generatedAt.UTC(),
		WindowSeconds: windowSeconds,
		Limit:         limit,
		Items:         cloned,
	}
}

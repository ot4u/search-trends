package trending

import (
	"testing"
	"time"
)

func TestWindowExpiration(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	window := NewWindow(5, base)

	if !window.AddAt(base, "iphone", 10) {
		t.Fatal("expected event to be accepted")
	}
	if !window.AddAt(base.Add(-1*time.Second), "iphone", 5) {
		t.Fatal("expected lagged event to be accepted")
	}

	window.AdvanceTo(base.Add(2 * time.Second))
	if got := window.TotalScore("iphone"); got != 15 {
		t.Fatalf("score after partial advance = %d, want 15", got)
	}

	window.AdvanceTo(base.Add(5 * time.Second))
	if got := window.TotalScore("iphone"); got != 0 {
		t.Fatalf("score after expiration = %d, want 0", got)
	}
}

func TestWindowGlobalCounterConsistency(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	window := NewWindow(6, base)

	events := []struct {
		offset int
		query  string
		weight uint64
	}{
		{offset: 0, query: "iphone", weight: 10},
		{offset: -1, query: "iphone", weight: 3},
		{offset: -2, query: "macbook", weight: 7},
		{offset: 0, query: "macbook", weight: 4},
	}

	for _, event := range events {
		if !window.AddAt(base.Add(time.Duration(event.offset)*time.Second), event.query, event.weight) {
			t.Fatalf("event %+v was unexpectedly rejected", event)
		}
	}

	window.AdvanceTo(base.Add(2 * time.Second))

	sumByQuery := make(map[string]uint64)
	for _, bucket := range window.buckets {
		for query, score := range bucket.counts {
			sumByQuery[query] += score
		}
	}

	for query, score := range sumByQuery {
		if got := window.TotalScore(query); got != score {
			t.Fatalf("global counter mismatch for %q: got %d want %d", query, got, score)
		}
	}
}

func TestWindowRejectsStaleEvent(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	window := NewWindow(3, base)

	if window.AddAt(base.Add(-4*time.Second), "iphone", 1) {
		t.Fatal("expected stale event to be rejected")
	}
}

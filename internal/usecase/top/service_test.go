package top

import (
	"testing"
	"time"

	"github.com/ot4/search-trends/internal/domain/trending"
	"github.com/ot4/search-trends/internal/infrastructure/cache"
)

func TestServiceUsesNearestCachedSnapshot(t *testing.T) {
	t.Parallel()

	store := cache.NewStore(300)
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	store.Publish(now, 300, map[int][]trending.Item{
		5: {
			{Query: "iphone", Score: 50, Rank: 1},
			{Query: "macbook", Score: 40, Rank: 2},
		},
		10: {
			{Query: "iphone", Score: 50, Rank: 1},
			{Query: "macbook", Score: 40, Rank: 2},
			{Query: "watch", Score: 30, Rank: 3},
		},
	})

	service := NewService(store)
	snapshot, err := service.GetSnapshot(7)
	if err != nil {
		t.Fatalf("GetSnapshot() error = %v", err)
	}

	if snapshot.Limit != 7 {
		t.Fatalf("limit = %d, want 7", snapshot.Limit)
	}
	if len(snapshot.Items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(snapshot.Items))
	}
}

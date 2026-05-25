package cache

import (
	"testing"
	"time"

	"github.com/ot4/search-trends/internal/domain/trending"
)

func TestStorePublishAndGet(t *testing.T) {
	t.Parallel()

	store := NewStore(300)
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

	store.Publish(now, 300, map[int][]trending.Item{
		10: {
			{Query: "iphone", Score: 100, Rank: 1},
		},
	})

	snapshot, ok := store.Get(10)
	if !ok {
		t.Fatal("expected snapshot to exist")
	}
	if snapshot.GeneratedAt != now {
		t.Fatalf("generated_at = %v, want %v", snapshot.GeneratedAt, now)
	}
	if len(snapshot.Items) != 1 || snapshot.Items[0].Query != "iphone" {
		t.Fatalf("unexpected snapshot items: %#v", snapshot.Items)
	}
}

func BenchmarkSnapshotRead(b *testing.B) {
	store := NewStore(300)
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

	store.Publish(now, 300, map[int][]trending.Item{
		100: {
			{Query: "iphone", Score: 100, Rank: 1},
		},
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = store.Get(100)
	}
}

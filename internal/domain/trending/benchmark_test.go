package trending

import (
	"strconv"
	"testing"
	"time"
)

func BenchmarkAddEvent(b *testing.B) {
	window := NewWindow(300, time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		query := "query-" + strconv.Itoa(i%10_000)
		window.AddAt(time.Unix(window.CurrentSecond(), 0).UTC(), query, 10)
	}
}

func BenchmarkExpireBucket(b *testing.B) {
	base := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	counts := make(map[string]uint64, 5_000)
	for j := 0; j < 5_000; j++ {
		counts["query-"+strconv.Itoa(j)] = 10
	}

	window := NewWindow(300, base)
	target := base.Add(time.Second)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		window.ResetFilledForTest(base, counts)
		window.AdvanceTo(target)
	}
}

func BenchmarkBuildTopN(b *testing.B) {
	counts := make(map[string]uint64, 100_000)
	for i := 0; i < 100_000; i++ {
		counts["query-"+strconv.Itoa(i)] = uint64((i % 1_000) + 1)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = BuildTopN(counts, 100)
	}
}

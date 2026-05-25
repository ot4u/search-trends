package ingest

import (
	"testing"
	"time"

	"github.com/ot4/search-trends/internal/domain/antifraud"
	domainstoplist "github.com/ot4/search-trends/internal/domain/stoplist"
	"github.com/ot4/search-trends/internal/domain/trending"
	"github.com/ot4/search-trends/internal/infrastructure/cache"
)

func TestProcessorUsesStoplist(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	processor := NewProcessor(ProcessorConfig{
		Window:   trending.NewWindow(300, now),
		Stoplist: domainstoplist.NewStore([]string{"spam"}),
		Detector: antifraud.NewDetector(10, 10, 1),
	})

	result := processor.ProcessAt(trending.Event{
		Query:     "iphone spam",
		Timestamp: now,
	}, now)

	if result.Outcome != OutcomeIgnoredStoplist {
		t.Fatalf("outcome = %q, want %q", result.Outcome, OutcomeIgnoredStoplist)
	}
	if processor.window.UniqueCount() != 0 {
		t.Fatalf("unexpected unique count: %d", processor.window.UniqueCount())
	}
}

func TestProcessorMemoryProtection(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	processor := NewProcessor(ProcessorConfig{
		Window:           trending.NewWindow(300, now),
		Detector:         antifraud.NewDetector(10, 10, 1),
		MaxUniqueQueries: 1,
	})

	first := processor.ProcessAt(trending.Event{Query: "iphone", Timestamp: now}, now)
	second := processor.ProcessAt(trending.Event{Query: "macbook", Timestamp: now}, now)
	third := processor.ProcessAt(trending.Event{Query: "iphone", Timestamp: now}, now)

	if !first.Accepted || !third.Accepted {
		t.Fatal("expected hot query to stay accepted")
	}
	if second.Outcome != OutcomeIgnoredCapacity {
		t.Fatalf("outcome = %q, want %q", second.Outcome, OutcomeIgnoredCapacity)
	}
}

func TestProcessorAppliesAntiFraudWeights(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	processor := NewProcessor(ProcessorConfig{
		Window:   trending.NewWindow(300, now),
		Detector: antifraud.NewDetector(2, 10, 1),
	})

	processor.ProcessAt(trending.Event{Query: "iphone", Timestamp: now}, now)
	processor.ProcessAt(trending.Event{Query: "iphone", Timestamp: now}, now)
	processor.ProcessAt(trending.Event{Query: "iphone", Timestamp: now}, now)

	if got := processor.window.TotalScore("iphone"); got != 21 {
		t.Fatalf("score = %d, want 21", got)
	}
}

func TestProcessorPublishesSnapshots(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	snapshots := cache.NewStore(300)
	processor := NewProcessor(ProcessorConfig{
		Window:    trending.NewWindow(300, now),
		Detector:  antifraud.NewDetector(10, 10, 1),
		Snapshots: snapshots,
	})

	processor.ProcessAt(trending.Event{Query: "iphone", Timestamp: now}, now)
	processor.ProcessAt(trending.Event{Query: "iphone", Timestamp: now}, now)
	processor.ProcessAt(trending.Event{Query: "macbook", Timestamp: now}, now)

	processor.Tick(now.Add(time.Second))

	snapshot, ok := snapshots.Get(10)
	if !ok {
		t.Fatal("expected top-10 snapshot")
	}
	if len(snapshot.Items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(snapshot.Items))
	}
	if snapshot.Items[0].Query != "iphone" || snapshot.Items[0].Score != 20 {
		t.Fatalf("unexpected first item: %#v", snapshot.Items[0])
	}
}

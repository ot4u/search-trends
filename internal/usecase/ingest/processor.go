package ingest

import (
	"context"
	"time"

	"github.com/ot4/search-trends/internal/domain/antifraud"
	domainstoplist "github.com/ot4/search-trends/internal/domain/stoplist"
	"github.com/ot4/search-trends/internal/domain/trending"
)

const (
	defaultMaxFutureSkew = 2 * time.Second
)

type Outcome string

const (
	OutcomeAccepted        Outcome = "accepted"
	OutcomeIgnoredEmpty    Outcome = "ignored_empty"
	OutcomeIgnoredStoplist Outcome = "ignored_stoplist"
	OutcomeIgnoredStale    Outcome = "ignored_stale"
	OutcomeIgnoredFuture   Outcome = "ignored_future"
	OutcomeIgnoredCapacity Outcome = "ignored_capacity"
)

type Result struct {
	Accepted     bool
	Outcome      Outcome
	Query        string
	Weight       uint64
	Downweighted bool
}

type Observer interface {
	ObserveProcessed(Result)
	ObserveSnapshotRefresh(duration time.Duration, uniqueQueries int)
}

type noopObserver struct{}

func (noopObserver) ObserveProcessed(Result)                   {}
func (noopObserver) ObserveSnapshotRefresh(time.Duration, int) {}

type SnapshotWriter interface {
	Publish(generatedAt time.Time, windowSeconds int, snapshots map[int][]trending.Item)
}

type Enqueuer interface {
	Enqueue(ctx context.Context, event trending.Event, ack AckFunc) error
	QueueSize() int
}

type noopSnapshotWriter struct{}

func (noopSnapshotWriter) Publish(time.Time, int, map[int][]trending.Item) {}

type ProcessorConfig struct {
	Window           *trending.Window
	Stoplist         *domainstoplist.Store
	Detector         *antifraud.Detector
	Snapshots        SnapshotWriter
	SnapshotLimits   []int
	MaxUniqueQueries int
	MaxFutureSkew    time.Duration
	Observer         Observer
}

type Processor struct {
	window           *trending.Window
	stoplist         *domainstoplist.Store
	detector         *antifraud.Detector
	snapshots        SnapshotWriter
	snapshotLimits   []int
	maxUniqueQueries int
	maxFutureSkew    time.Duration
	observer         Observer
}

func NewProcessor(cfg ProcessorConfig) *Processor {
	if cfg.Window == nil {
		cfg.Window = trending.NewWindow(300, time.Now().UTC())
	}
	if cfg.Stoplist == nil {
		cfg.Stoplist = domainstoplist.NewStore(nil)
	}
	if cfg.Detector == nil {
		cfg.Detector = antifraud.NewDetector(20, 10, 1)
	}
	if cfg.Snapshots == nil {
		cfg.Snapshots = noopSnapshotWriter{}
	}
	if len(cfg.SnapshotLimits) == 0 {
		cfg.SnapshotLimits = []int{5, 10, 25, 50, 100}
	}
	if cfg.MaxFutureSkew <= 0 {
		cfg.MaxFutureSkew = defaultMaxFutureSkew
	}
	if cfg.Observer == nil {
		cfg.Observer = noopObserver{}
	}

	processor := &Processor{
		window:           cfg.Window,
		stoplist:         cfg.Stoplist,
		detector:         cfg.Detector,
		snapshots:        cfg.Snapshots,
		snapshotLimits:   sanitizeLimits(cfg.SnapshotLimits),
		maxUniqueQueries: cfg.MaxUniqueQueries,
		maxFutureSkew:    cfg.MaxFutureSkew,
		observer:         cfg.Observer,
	}

	processor.publishSnapshots(time.Unix(cfg.Window.CurrentSecond(), 0).UTC())
	return processor
}

func (p *Processor) ProcessAt(event trending.Event, now time.Time) Result {
	p.Tick(now)

	normalized := trending.NormalizeQuery(event.Query)
	if normalized == "" {
		result := Result{Outcome: OutcomeIgnoredEmpty}
		p.observer.ObserveProcessed(result)
		return result
	}

	tokens := trending.SplitTokens(normalized)
	if p.stoplist.ContainsAny(tokens) {
		result := Result{
			Outcome: OutcomeIgnoredStoplist,
			Query:   normalized,
		}
		p.observer.ObserveProcessed(result)
		return result
	}

	eventTime := now.UTC()
	if !event.Timestamp.IsZero() {
		eventTime = event.Timestamp.UTC()
	}

	if eventTime.After(now.Add(p.maxFutureSkew)) {
		result := Result{
			Outcome: OutcomeIgnoredFuture,
			Query:   normalized,
		}
		p.observer.ObserveProcessed(result)
		return result
	}

	if eventTime.After(now) {
		eventTime = now.UTC()
	}

	if p.maxUniqueQueries > 0 && !p.window.HasQuery(normalized) && p.window.UniqueCount() >= p.maxUniqueQueries {
		result := Result{
			Outcome: OutcomeIgnoredCapacity,
			Query:   normalized,
		}
		p.observer.ObserveProcessed(result)
		return result
	}

	weight := p.detector.Weight(normalized)
	downweighted := p.detector.IsDownweighted(weight)
	if !p.window.AddAt(eventTime, normalized, weight) {
		result := Result{
			Outcome: OutcomeIgnoredStale,
			Query:   normalized,
			Weight:  weight,
		}
		p.observer.ObserveProcessed(result)
		return result
	}

	result := Result{
		Accepted:     true,
		Outcome:      OutcomeAccepted,
		Query:        normalized,
		Weight:       weight,
		Downweighted: downweighted,
	}
	p.observer.ObserveProcessed(result)
	return result
}

func (p *Processor) Tick(now time.Time) {
	if p.window.AdvanceTo(now) == 0 {
		return
	}

	p.detector.Reset()
	p.publishSnapshots(now)
}

func (p *Processor) Counts() map[string]uint64 {
	return p.window.Counts()
}

func (p *Processor) WindowSeconds() int {
	return p.window.WindowSeconds()
}

func (p *Processor) publishSnapshots(now time.Time) {
	start := time.Now()

	maxLimit := p.snapshotLimits[len(p.snapshotLimits)-1]
	topMax := trending.BuildTopN(p.window.Counts(), maxLimit)
	batch := make(map[int][]trending.Item, len(p.snapshotLimits))

	for _, limit := range p.snapshotLimits {
		size := limit
		if size > len(topMax) {
			size = len(topMax)
		}

		items := make([]trending.Item, size)
		copy(items, topMax[:size])
		batch[limit] = items
	}

	p.snapshots.Publish(now.UTC(), p.window.WindowSeconds(), batch)
	p.observer.ObserveSnapshotRefresh(time.Since(start), p.window.UniqueCount())
}

func sanitizeLimits(limits []int) []int {
	seen := make(map[int]struct{}, len(limits))
	filtered := make([]int, 0, len(limits))

	for _, limit := range limits {
		if limit <= 0 {
			continue
		}
		if _, ok := seen[limit]; ok {
			continue
		}
		seen[limit] = struct{}{}
		filtered = append(filtered, limit)
	}

	if len(filtered) == 0 {
		return []int{5, 10, 25, 50, 100}
	}

	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[j] < filtered[i] {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	return filtered
}

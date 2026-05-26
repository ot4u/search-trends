package metrics

import (
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ot4/search-trends/internal/usecase/ingest"
)

type Metrics struct {
	registry *prometheus.Registry

	processedTotal        *prometheus.CounterVec
	processedWeight       prometheus.Counter
	eventsDropped         prometheus.Counter
	antifraudDownweighted prometheus.Counter
	snapshotRefresh       prometheus.Counter
	queueSize             prometheus.Gauge
	queueCapacity         prometheus.Gauge
	snapshotAge           prometheus.Gauge
	snapshotRefreshDur    prometheus.Histogram
	uniqueQueries         prometheus.Gauge
	memoryAllocBytes      prometheus.Gauge
	natsConsumerLag       prometheus.Gauge
	httpRequests          *prometheus.CounterVec
	httpLatency           *prometheus.HistogramVec
	natsMessages          *prometheus.CounterVec

	mu             sync.Mutex
	lastSnapshotAt time.Time
}

func New() *Metrics {
	registry := prometheus.NewRegistry()

	m := &Metrics{
		registry: registry,
		processedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "search_trends_events_total",
			Help: "Total number of processed search events by outcome.",
		}, []string{"outcome"}),
		processedWeight: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "search_trends_applied_weight_total",
			Help: "Total score weight applied to accepted events.",
		}),
		eventsDropped: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "search_trends_events_dropped_total",
			Help: "Total events dropped during ingest (capacity, stale, future).",
		}),
		antifraudDownweighted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "search_trends_antifraud_downweighted_total",
			Help: "Total accepted events downweighted by anti-fraud.",
		}),
		snapshotRefresh: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "search_trends_snapshot_refresh_total",
			Help: "Total snapshot refresh operations.",
		}),
		queueSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "search_trends_ingest_queue_size",
			Help: "Current bounded ingest queue depth.",
		}),
		queueCapacity: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "search_trends_ingest_queue_capacity",
			Help: "Bounded ingest queue capacity.",
		}),
		snapshotAge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "search_trends_snapshot_age_seconds",
			Help: "Seconds since the last snapshot refresh.",
		}),
		snapshotRefreshDur: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "search_trends_snapshot_refresh_duration_seconds",
			Help:    "Snapshot refresh duration in seconds.",
			Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}),
		uniqueQueries: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "search_trends_unique_queries",
			Help: "Current number of unique queries in the sliding window.",
		}),
		memoryAllocBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "search_trends_memory_alloc_bytes",
			Help: "Current heap allocation reported by runtime.MemStats.",
		}),
		natsConsumerLag: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "search_trends_nats_consumer_lag_seconds",
			Help: "Estimated NATS consumer lag based on the newest queued event timestamp.",
		}),
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "search_trends_http_requests_total",
			Help: "Total HTTP requests by method, route, and status.",
		}, []string{"method", "route", "status"}),
		httpLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "search_trends_http_request_duration_seconds",
			Help:    "HTTP request latency by method and route.",
			Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}, []string{"method", "route"}),
		natsMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "search_trends_nats_messages_total",
			Help: "NATS message handling attempts by outcome.",
		}, []string{"outcome"}),
	}

	registry.MustRegister(
		m.processedTotal,
		m.processedWeight,
		m.eventsDropped,
		m.antifraudDownweighted,
		m.snapshotRefresh,
		m.queueSize,
		m.queueCapacity,
		m.snapshotAge,
		m.snapshotRefreshDur,
		m.uniqueQueries,
		m.memoryAllocBytes,
		m.natsConsumerLag,
		m.httpRequests,
		m.httpLatency,
		m.natsMessages,
	)

	go m.collectRuntime()

	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) SetQueueCapacity(capacity int) {
	m.queueCapacity.Set(float64(capacity))
}

func (m *Metrics) ObserveProcessed(result ingest.Result) {
	m.processedTotal.WithLabelValues(string(result.Outcome)).Inc()

	if result.Accepted {
		m.processedWeight.Add(float64(result.Weight))
		if result.Downweighted {
			m.antifraudDownweighted.Inc()
		}
		return
	}

	switch result.Outcome {
	case ingest.OutcomeIgnoredCapacity,
		ingest.OutcomeIgnoredStale,
		ingest.OutcomeIgnoredFuture:
		m.eventsDropped.Inc()
	}
}

func (m *Metrics) ObserveSnapshotRefresh(duration time.Duration, uniqueQueries int) {
	m.snapshotRefresh.Inc()
	m.snapshotRefreshDur.Observe(duration.Seconds())
	m.uniqueQueries.Set(float64(uniqueQueries))

	m.mu.Lock()
	m.lastSnapshotAt = time.Now().UTC()
	m.mu.Unlock()

	m.snapshotAge.Set(0)
}

func (m *Metrics) SetQueueSize(size int) {
	m.queueSize.Set(float64(size))
}

func (m *Metrics) ObserveHTTP(method, route string, status int, latency time.Duration) {
	statusText := strconv.Itoa(status)
	m.httpRequests.WithLabelValues(method, route, statusText).Inc()
	m.httpLatency.WithLabelValues(method, route).Observe(latency.Seconds())
}

func (m *Metrics) ObserveNATS(outcome string) {
	m.natsMessages.WithLabelValues(outcome).Inc()
}

func (m *Metrics) SetConsumerLag(lag time.Duration) {
	if lag < 0 {
		lag = 0
	}
	m.natsConsumerLag.Set(lag.Seconds())
}

func (m *Metrics) collectRuntime() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		m.memoryAllocBytes.Set(float64(stats.HeapAlloc))

		m.mu.Lock()
		lastSnapshotAt := m.lastSnapshotAt
		m.mu.Unlock()

		if !lastSnapshotAt.IsZero() {
			m.snapshotAge.Set(time.Since(lastSnapshotAt).Seconds())
		}
	}
}

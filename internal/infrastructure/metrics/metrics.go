package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ot4/search-trends/internal/usecase/ingest"
)

type Metrics struct {
	registry *prometheus.Registry

	processedTotal  *prometheus.CounterVec
	processedWeight prometheus.Counter
	snapshotRefresh prometheus.Counter
	queueSize       prometheus.Gauge
	httpRequests    *prometheus.CounterVec
	httpLatency     *prometheus.HistogramVec
	natsMessages    *prometheus.CounterVec
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
		snapshotRefresh: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "search_trends_snapshot_refresh_total",
			Help: "Total snapshot refresh operations.",
		}),
		queueSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "search_trends_ingest_queue_size",
			Help: "Current bounded ingest queue depth.",
		}),
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "search_trends_http_requests_total",
			Help: "Total HTTP requests by method, route, and status.",
		}, []string{"method", "route", "status"}),
		httpLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "search_trends_http_request_duration_seconds",
			Help:    "HTTP request latency by method and route.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
		natsMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "search_trends_nats_messages_total",
			Help: "NATS message handling attempts by outcome.",
		}, []string{"outcome"}),
	}

	registry.MustRegister(
		m.processedTotal,
		m.processedWeight,
		m.snapshotRefresh,
		m.queueSize,
		m.httpRequests,
		m.httpLatency,
		m.natsMessages,
	)

	return m
}
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
func (m *Metrics) ObserveProcessed(result ingest.Result) {
	m.processedTotal.WithLabelValues(string(result.Outcome)).Inc()
	if result.Accepted {
		m.processedWeight.Add(float64(result.Weight))
	}
}
func (m *Metrics) ObserveSnapshotRefresh(_ int) {
	m.snapshotRefresh.Inc()
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

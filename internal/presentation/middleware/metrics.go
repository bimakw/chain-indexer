package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being served",
		},
	)
)

// Metrics returns a middleware that collects Prometheus metrics
func Metrics() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			httpRequestsInFlight.Inc()
			defer httpRequestsInFlight.Dec()

			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(wrapped.status)

			// Normalize path to avoid high cardinality
			path := normalizePath(r.URL.Path)

			httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
			httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
		})
	}
}

// normalizePath normalizes the path to reduce cardinality
func normalizePath(path string) string {
	// For now, return the path as-is
	// In production, you might want to replace UUIDs, IDs, etc.
	return path
}

// IndexerMetrics holds Prometheus metrics for the indexer
type IndexerMetrics struct {
	BlocksIndexed    prometheus.Counter
	TransfersIndexed prometheus.Counter
	LastIndexedBlock prometheus.Gauge
	IndexingLatency  prometheus.Histogram
	ErrorsTotal      prometheus.Counter
}

// NewIndexerMetrics creates new indexer metrics
func NewIndexerMetrics() *IndexerMetrics {
	return &IndexerMetrics{
		BlocksIndexed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "indexer_blocks_indexed_total",
			Help: "Total number of blocks indexed",
		}),
		TransfersIndexed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "indexer_transfers_indexed_total",
			Help: "Total number of transfers indexed",
		}),
		LastIndexedBlock: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "indexer_last_indexed_block",
			Help: "Last indexed block number",
		}),
		IndexingLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "indexer_indexing_latency_seconds",
			Help:    "Time taken to index a batch of blocks",
			Buckets: []float64{.1, .5, 1, 2.5, 5, 10, 30, 60},
		}),
		ErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "indexer_errors_total",
			Help: "Total number of indexing errors",
		}),
	}
}

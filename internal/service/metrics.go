package service

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tschaefer/k8s/webhook/last-updated/internal/version"
)

var (
	metricsRegistry = prometheus.NewRegistry()

	metricsReviewsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "last_updated_reviews_requests_total",
		Help: "Total number of admission review requests received.",
	})
	metricsPatchedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "last_updated_reviews_patched_total",
		Help: "Total number of admission review requests that resulted in a patch.",
	})
	metricsFailedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "last_updated_reviews_failed_total",
		Help: "Total number of failed admission review requests.",
	})
	metricsPatchDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "last_updated_patch_duration_seconds",
		Help:    "Duration in seconds for building admission patches.",
		Buckets: prometheus.DefBuckets,
	})
)

func init() {
	metricsRegistry.MustRegister(
		metricsReviewsTotal,
		metricsPatchedTotal,
		metricsFailedTotal,
		metricsPatchDuration,
	)
}

// handleMetrics returns admission review telemetrics.
func (s *Service) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-LastUpdated-Commit", version.Commit())
	w.Header().Set("X-LastUpdated-Version", version.Release())
	promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

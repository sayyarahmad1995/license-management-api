package service

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsService manages Prometheus metrics collection
type MetricsService struct {
	// HTTP Metrics
	httpRequestsTotal   prometheus.CounterVec
	httpRequestDuration prometheus.HistogramVec
	httpResponseBytes   prometheus.HistogramVec
	httpErrorsTotal     prometheus.CounterVec

	// Database Metrics
	dbQueryDuration prometheus.HistogramVec
	dbErrorsTotal   prometheus.CounterVec

	// Cache Metrics
	cacheHitsTotal   prometheus.CounterVec
	cacheMissesTotal prometheus.CounterVec

	// License Metrics
	licenseOpsTotal prometheus.CounterVec

	// Authentication Metrics
	authFailuresTotal prometheus.CounterVec

	// Custom Metrics
	activeConnections prometheus.Gauge
}

// NewMetricsService creates and initializes a new MetricsService
func NewMetricsService() *MetricsService {
	ms := &MetricsService{
		// HTTP Request Metrics
		httpRequestsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		httpResponseBytes: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "path"},
		),
		httpErrorsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_errors_total",
				Help: "Total number of HTTP errors",
			},
			[]string{"method", "path", "status"},
		),

		// Database Metrics
		dbQueryDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
		dbErrorsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_errors_total",
				Help: "Total number of database errors",
			},
			[]string{"operation"},
		),

		// Cache Metrics
		cacheHitsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"key"},
		),
		cacheMissesTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"key"},
		),

		// License Metrics
		licenseOpsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "license_operations_total",
				Help: "Total number of license operations",
			},
			[]string{"operation", "status"},
		),

		// Authentication Metrics
		authFailuresTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_failures_total",
				Help: "Total number of authentication failures",
			},
			[]string{"reason"},
		),

		// Active Connections
		activeConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_connections",
				Help: "Number of active connections",
			},
		),
	}

	return ms
}

// RecordHTTPRequest records metrics for an HTTP request
func (ms *MetricsService) RecordHTTPRequest(method, path string, statusCode int, duration float64, responseBytes int64) {
	ms.httpRequestsTotal.WithLabelValues(method, path, fmt.Sprintf("%d", statusCode)).Inc()
	ms.httpRequestDuration.WithLabelValues(method, path).Observe(duration)
	ms.httpResponseBytes.WithLabelValues(method, path).Observe(float64(responseBytes))

	if statusCode >= 400 {
		ms.httpErrorsTotal.WithLabelValues(method, path, fmt.Sprintf("%d", statusCode)).Inc()
	}
}

// RecordDatabaseQuery records metrics for a database query
func (ms *MetricsService) RecordDatabaseQuery(operation string, duration float64, err error) {
	ms.dbQueryDuration.WithLabelValues(operation).Observe(duration)
	if err != nil {
		ms.dbErrorsTotal.WithLabelValues(operation).Inc()
	}
}

// RecordCacheHit increments the cache hit counter
func (ms *MetricsService) RecordCacheHit(key string) {
	ms.cacheHitsTotal.WithLabelValues(key).Inc()
}

// RecordCacheMiss increments the cache miss counter
func (ms *MetricsService) RecordCacheMiss(key string) {
	ms.cacheMissesTotal.WithLabelValues(key).Inc()
}

// RecordLicenseOperation records a license operation
func (ms *MetricsService) RecordLicenseOperation(operation, status string) {
	ms.licenseOpsTotal.WithLabelValues(operation, status).Inc()
}

// RecordAuthFailure records an authentication failure
func (ms *MetricsService) RecordAuthFailure(reason string) {
	ms.authFailuresTotal.WithLabelValues(reason).Inc()
}

// SetActiveConnections sets the current number of active connections
func (ms *MetricsService) SetActiveConnections(count int) {
	ms.activeConnections.Set(float64(count))
}

// IncrementActiveConnections increments the active connection count
func (ms *MetricsService) IncrementActiveConnections() {
	ms.activeConnections.Inc()
}

// DecrementActiveConnections decrements the active connection count
func (ms *MetricsService) DecrementActiveConnections() {
	ms.activeConnections.Dec()
}

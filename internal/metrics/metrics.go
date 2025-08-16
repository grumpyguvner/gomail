package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var (
	once sync.Once

	// HTTP Metrics
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gomail_http_request_duration_seconds",
			Help:    "HTTP request latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPActiveRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gomail_http_active_requests",
			Help: "Number of active HTTP requests",
		},
	)

	HTTPResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gomail_http_response_size_bytes",
			Help:    "Size of HTTP responses in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// Email Metrics
	EmailsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_emails_processed_total",
			Help: "Total number of emails processed",
		},
		[]string{"status"}, // "success", "error", "rejected"
	)

	EmailSize = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gomail_email_size_bytes",
			Help:    "Size of processed emails in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 15), // 1KB to 16MB
		},
	)

	EmailProcessingDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gomail_email_processing_duration_seconds",
			Help:    "Time taken to process emails in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	StorageOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_storage_operations_total",
			Help: "Total number of storage operations",
		},
		[]string{"operation", "status"}, // operation: "write", "read", "delete"; status: "success", "error"
	)

	// Rate Limit Metrics
	RateLimitHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_rate_limit_hits_total",
			Help: "Number of rate limit hits",
		},
		[]string{"action"}, // "allowed" or "denied"
	)

	// Shutdown Metrics
	ShutdownsInitiated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_shutdowns_initiated_total",
			Help: "Number of shutdown requests initiated",
		},
		[]string{"type"}, // "graceful" or "forced"
	)

	ShutdownDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gomail_shutdown_duration_seconds",
			Help:    "Time taken for server shutdown",
			Buckets: prometheus.LinearBuckets(0, 5, 7), // 0 to 30 seconds in 5-second intervals
		},
	)
)

// Init registers all metrics with the Prometheus registry
func Init() {
	once.Do(func() {
		// Register custom metrics - use Register instead of MustRegister to handle duplicates
		_ = prometheus.Register(HTTPRequestDuration)
		_ = prometheus.Register(HTTPActiveRequests)
		_ = prometheus.Register(HTTPResponseSize)
		_ = prometheus.Register(HTTPRequestsTotal)
		_ = prometheus.Register(EmailsProcessed)
		_ = prometheus.Register(EmailSize)
		_ = prometheus.Register(EmailProcessingDuration)
		_ = prometheus.Register(StorageOperations)
		_ = prometheus.Register(RateLimitHits)
		_ = prometheus.Register(ShutdownsInitiated)
		_ = prometheus.Register(ShutdownDuration)

		// Register Go runtime metrics
		_ = prometheus.Register(collectors.NewGoCollector())
		_ = prometheus.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	})
}

// Reset unregisters all metrics (useful for testing)
func Reset() {
	prometheus.Unregister(HTTPRequestDuration)
	prometheus.Unregister(HTTPActiveRequests)
	prometheus.Unregister(HTTPResponseSize)
	prometheus.Unregister(HTTPRequestsTotal)
	prometheus.Unregister(EmailsProcessed)
	prometheus.Unregister(EmailSize)
	prometheus.Unregister(EmailProcessingDuration)
	prometheus.Unregister(StorageOperations)
	prometheus.Unregister(RateLimitHits)
	prometheus.Unregister(ShutdownsInitiated)
	prometheus.Unregister(ShutdownDuration)
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	once = sync.Once{}
}

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

	// Error Metrics
	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type", "handler"}, // type: error category, handler: endpoint or component
	)

	// Timeout Metrics
	TimeoutsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_timeouts_total",
			Help: "Total number of request timeouts",
		},
		[]string{"endpoint"},
	)

	// Connection Pool Metrics
	ConnectionPoolSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gomail_connection_pool_size",
			Help: "Current size of connection pool",
		},
		[]string{"type"}, // type: active, idle, total
	)

	ConnectionPoolWaitTime = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gomail_connection_pool_wait_seconds",
			Help:    "Time spent waiting for a connection from the pool",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
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
		_ = prometheus.Register(ErrorsTotal)
		_ = prometheus.Register(TimeoutsTotal)
		_ = prometheus.Register(ConnectionPoolSize)
		_ = prometheus.Register(ConnectionPoolWaitTime)

		// Register security metrics
		_ = prometheus.Register(ConnectionsAccepted)
		_ = prometheus.Register(ConnectionsRejected)
		_ = prometheus.Register(ActiveConnections)
		_ = prometheus.Register(ConnectionsPerIP)
		_ = prometheus.Register(BannedIPs)
		_ = prometheus.Register(ThrottleAllowed)
		_ = prometheus.Register(ThrottleRejections)
		_ = prometheus.Register(ThrottleWaitTime)
		_ = prometheus.Register(UniqueIPs)
		_ = prometheus.Register(SecurityViolations)
		_ = prometheus.Register(IPReputationChecks)
		_ = prometheus.Register(ConnectionAbuseDetected)
		_ = prometheus.Register(FirewallRulesApplied)
		_ = prometheus.Register(FirewallBlockedConnections)

		// Register TLS metrics
		_ = prometheus.Register(TLSConnections)
		_ = prometheus.Register(TLSHandshakeErrors)
		_ = prometheus.Register(TLSVersion)
		_ = prometheus.Register(TLSCipherSuite)
		_ = prometheus.Register(STARTTLSCommands)
		_ = prometheus.Register(TLSCertificateExpiry)
		_ = prometheus.Register(TLSHandshakeDuration)
		_ = prometheus.Register(PlaintextConnections)
		_ = prometheus.Register(TLSRequiredRejections)

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
	prometheus.Unregister(ErrorsTotal)
	prometheus.Unregister(TimeoutsTotal)
	prometheus.Unregister(ConnectionPoolSize)
	prometheus.Unregister(ConnectionPoolWaitTime)

	// Unregister security metrics
	prometheus.Unregister(ConnectionsAccepted)
	prometheus.Unregister(ConnectionsRejected)
	prometheus.Unregister(ActiveConnections)
	prometheus.Unregister(ConnectionsPerIP)
	prometheus.Unregister(BannedIPs)
	prometheus.Unregister(ThrottleAllowed)
	prometheus.Unregister(ThrottleRejections)
	prometheus.Unregister(ThrottleWaitTime)
	prometheus.Unregister(UniqueIPs)
	prometheus.Unregister(SecurityViolations)
	prometheus.Unregister(IPReputationChecks)
	prometheus.Unregister(ConnectionAbuseDetected)
	prometheus.Unregister(FirewallRulesApplied)
	prometheus.Unregister(FirewallBlockedConnections)

	// Unregister TLS metrics
	prometheus.Unregister(TLSConnections)
	prometheus.Unregister(TLSHandshakeErrors)
	prometheus.Unregister(TLSVersion)
	prometheus.Unregister(TLSCipherSuite)
	prometheus.Unregister(STARTTLSCommands)
	prometheus.Unregister(TLSCertificateExpiry)
	prometheus.Unregister(TLSHandshakeDuration)
	prometheus.Unregister(PlaintextConnections)
	prometheus.Unregister(TLSRequiredRejections)

	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Reset the once to allow re-initialization
	once = sync.Once{}

	// Re-create all metric instances to clear any existing data
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
			Buckets: prometheus.ExponentialBuckets(100, 10, 7),
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

	EmailsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_emails_processed_total",
			Help: "Total number of emails processed",
		},
		[]string{"status"},
	)

	EmailSize = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gomail_email_size_bytes",
			Help:    "Size of processed emails in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 15),
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
		[]string{"operation", "status"},
	)

	RateLimitHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_rate_limit_hits_total",
			Help: "Number of rate limit hits",
		},
		[]string{"action"},
	)

	ShutdownsInitiated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_shutdowns_initiated_total",
			Help: "Number of shutdown requests initiated",
		},
		[]string{"type"},
	)

	ShutdownDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gomail_shutdown_duration_seconds",
			Help:    "Time taken for server shutdown",
			Buckets: prometheus.LinearBuckets(0, 5, 7),
		},
	)

	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type", "handler"},
	)

	TimeoutsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gomail_timeouts_total",
			Help: "Total number of request timeouts",
		},
		[]string{"endpoint"},
	)

	ConnectionPoolSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gomail_connection_pool_size",
			Help: "Current size of connection pool",
		},
		[]string{"type"},
	)

	ConnectionPoolWaitTime = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gomail_connection_pool_wait_seconds",
			Help:    "Time spent waiting for a connection from the pool",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
	)
}

// RecordError increments the error counter for a specific error type and handler
func RecordError(errorType string, handler string) {
	if ErrorsTotal != nil {
		ErrorsTotal.WithLabelValues(errorType, handler).Inc()
	}
}

// IncrementTimeouts increments the timeout counter for a specific endpoint
func IncrementTimeouts(endpoint string) {
	if TimeoutsTotal != nil {
		TimeoutsTotal.WithLabelValues(endpoint).Inc()
	}
}

// UpdateConnectionPoolMetrics updates connection pool gauge metrics
func UpdateConnectionPoolMetrics(active, idle, total float64) {
	if ConnectionPoolSize != nil {
		ConnectionPoolSize.WithLabelValues("active").Set(active)
		ConnectionPoolSize.WithLabelValues("idle").Set(idle)
		ConnectionPoolSize.WithLabelValues("total").Set(total)
	}
}

// RecordConnectionPoolWait records time spent waiting for a connection
func RecordConnectionPoolWait(seconds float64) {
	if ConnectionPoolWaitTime != nil {
		ConnectionPoolWaitTime.Observe(seconds)
	}
}

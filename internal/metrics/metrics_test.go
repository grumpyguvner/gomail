package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsInit(t *testing.T) {
	// Reset metrics before test
	Reset()

	// Initialize metrics
	Init()

	// Verify metrics are registered
	assert.NotPanics(t, func() {
		// Try to gather metrics - should not panic if registered
		_, err := prometheus.DefaultGatherer.Gather()
		require.NoError(t, err)
	})
}

func TestHTTPMetrics(t *testing.T) {
	// Reset and init metrics
	Reset()
	Init()

	// Test HTTPRequestDuration
	HTTPRequestDuration.WithLabelValues("GET", "/health", "200").Observe(0.1)
	HTTPRequestDuration.WithLabelValues("POST", "/mail/inbound", "200").Observe(0.2)
	HTTPRequestDuration.WithLabelValues("POST", "/mail/inbound", "400").Observe(0.05)

	// Verify metrics were recorded (3 unique label combinations)
	assert.Equal(t, 3, testutil.CollectAndCount(HTTPRequestDuration, "gomail_http_request_duration_seconds"))

	// Test HTTPActiveRequests
	HTTPActiveRequests.Inc()
	HTTPActiveRequests.Inc()
	assert.Equal(t, float64(2), testutil.ToFloat64(HTTPActiveRequests))
	HTTPActiveRequests.Dec()
	assert.Equal(t, float64(1), testutil.ToFloat64(HTTPActiveRequests))

	// Test HTTPResponseSize
	HTTPResponseSize.WithLabelValues("GET", "/health", "200").Observe(100)
	HTTPResponseSize.WithLabelValues("POST", "/mail/inbound", "200").Observe(1024)
	assert.Equal(t, 2, testutil.CollectAndCount(HTTPResponseSize, "gomail_http_response_size_bytes"))

	// Test HTTPRequestsTotal
	HTTPRequestsTotal.WithLabelValues("GET", "/health", "200").Inc()
	HTTPRequestsTotal.WithLabelValues("POST", "/mail/inbound", "200").Inc()
	HTTPRequestsTotal.WithLabelValues("POST", "/mail/inbound", "200").Inc()
	assert.Equal(t, 2, testutil.CollectAndCount(HTTPRequestsTotal, "gomail_http_requests_total"))
}

func TestEmailMetrics(t *testing.T) {
	// Reset and init metrics
	Reset()
	Init()

	// Test EmailsProcessed
	EmailsProcessed.WithLabelValues("success").Inc()
	EmailsProcessed.WithLabelValues("success").Inc()
	EmailsProcessed.WithLabelValues("error").Inc()
	EmailsProcessed.WithLabelValues("rejected").Inc()

	// Verify counters
	assert.Equal(t, 3, testutil.CollectAndCount(EmailsProcessed, "gomail_emails_processed_total"))

	// Test EmailSize
	EmailSize.Observe(1024)    // 1KB
	EmailSize.Observe(2048)    // 2KB
	EmailSize.Observe(1048576) // 1MB

	// Verify histogram collected
	assert.Equal(t, 1, testutil.CollectAndCount(EmailSize, "gomail_email_size_bytes"))

	// Test EmailProcessingDuration
	EmailProcessingDuration.Observe(0.05)
	EmailProcessingDuration.Observe(0.1)
	EmailProcessingDuration.Observe(0.2)

	assert.Equal(t, 1, testutil.CollectAndCount(EmailProcessingDuration, "gomail_email_processing_duration_seconds"))

	// Test StorageOperations
	StorageOperations.WithLabelValues("write", "success").Inc()
	StorageOperations.WithLabelValues("write", "success").Inc()
	StorageOperations.WithLabelValues("write", "error").Inc()
	StorageOperations.WithLabelValues("read", "success").Inc()

	assert.Equal(t, 3, testutil.CollectAndCount(StorageOperations, "gomail_storage_operations_total"))
}

func TestRateLimitMetrics(t *testing.T) {
	// Reset and init metrics
	Reset()
	Init()

	// Test RateLimitHits
	RateLimitHits.WithLabelValues("allowed").Inc()
	RateLimitHits.WithLabelValues("allowed").Inc()
	RateLimitHits.WithLabelValues("allowed").Inc()
	RateLimitHits.WithLabelValues("denied").Inc()

	// Verify counters
	assert.Equal(t, 2, testutil.CollectAndCount(RateLimitHits, "gomail_rate_limit_hits_total"))

	// Verify specific values
	allowedMetric, err := RateLimitHits.GetMetricWithLabelValues("allowed")
	require.NoError(t, err)
	assert.Equal(t, float64(3), testutil.ToFloat64(allowedMetric))

	deniedMetric, err := RateLimitHits.GetMetricWithLabelValues("denied")
	require.NoError(t, err)
	assert.Equal(t, float64(1), testutil.ToFloat64(deniedMetric))
}

func TestShutdownMetrics(t *testing.T) {
	// Reset and init metrics
	Reset()
	Init()

	// Test ShutdownsInitiated
	ShutdownsInitiated.WithLabelValues("graceful").Inc()
	assert.Equal(t, 1, testutil.CollectAndCount(ShutdownsInitiated, "gomail_shutdowns_initiated_total"))

	gracefulMetric, err := ShutdownsInitiated.GetMetricWithLabelValues("graceful")
	require.NoError(t, err)
	assert.Equal(t, float64(1), testutil.ToFloat64(gracefulMetric))

	// Test ShutdownDuration
	ShutdownDuration.Observe(5.5)  // 5.5 seconds
	ShutdownDuration.Observe(1.2)  // 1.2 seconds
	ShutdownDuration.Observe(28.9) // 28.9 seconds

	assert.Equal(t, 1, testutil.CollectAndCount(ShutdownDuration, "gomail_shutdown_duration_seconds"))
}

func TestMetricsReset(t *testing.T) {
	// Initialize metrics
	Init()

	// Add some data
	HTTPRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()
	EmailsProcessed.WithLabelValues("success").Inc()

	// Reset metrics
	Reset()

	// Verify metrics are unregistered - trying to gather should not include our metrics
	mfs, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	// Check that our custom metrics are not present
	for _, mf := range mfs {
		name := mf.GetName()
		assert.NotContains(t, name, "gomail_")
	}
}

func TestConcurrentMetricsAccess(t *testing.T) {
	// Reset and init metrics
	Reset()
	Init()

	// Test concurrent access to metrics
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				HTTPActiveRequests.Inc()
				HTTPRequestDuration.WithLabelValues("GET", "/test", "200").Observe(0.1)
				EmailsProcessed.WithLabelValues("success").Inc()
				HTTPActiveRequests.Dec()
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	assert.Equal(t, float64(0), testutil.ToFloat64(HTTPActiveRequests))
}

func TestMetricsIdempotency(t *testing.T) {
	// Reset first
	Reset()

	// Multiple calls to Init should not panic
	assert.NotPanics(t, func() {
		Init()
		Init()
		Init()
	})

	// Metrics should still work
	HTTPRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()
	assert.Equal(t, 1, testutil.CollectAndCount(HTTPRequestsTotal, "gomail_http_requests_total"))
}

func BenchmarkMetricsRecording(b *testing.B) {
	// Reset and init metrics
	Reset()
	Init()

	b.ResetTimer()

	b.Run("HTTPRequestDuration", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			HTTPRequestDuration.WithLabelValues("GET", "/test", "200").Observe(0.1)
		}
	})

	b.Run("EmailsProcessed", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EmailsProcessed.WithLabelValues("success").Inc()
		}
	})

	b.Run("HTTPActiveRequests", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			HTTPActiveRequests.Inc()
			HTTPActiveRequests.Dec()
		}
	})
}

func TestMetricsLabels(t *testing.T) {
	// Reset and init metrics
	Reset()
	Init()

	// Test that metrics with different labels are tracked separately
	HTTPRequestDuration.WithLabelValues("GET", "/health", "200").Observe(0.1)
	HTTPRequestDuration.WithLabelValues("GET", "/health", "503").Observe(0.2)
	HTTPRequestDuration.WithLabelValues("POST", "/mail/inbound", "200").Observe(0.3)
	HTTPRequestDuration.WithLabelValues("POST", "/mail/inbound", "400").Observe(0.4)

	// Should have 4 different label combinations
	assert.Equal(t, 4, testutil.CollectAndCount(HTTPRequestDuration, "gomail_http_request_duration_seconds"))
}

func TestMetricsWithTime(t *testing.T) {
	// Reset and init metrics
	Reset()
	Init()

	// Simulate processing with actual time measurements
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	duration := time.Since(start).Seconds()

	EmailProcessingDuration.Observe(duration)

	// Verify the metric was recorded and is reasonable
	assert.Equal(t, 1, testutil.CollectAndCount(EmailProcessingDuration, "gomail_email_processing_duration_seconds"))

	// The duration should be at least 10ms
	assert.Greater(t, duration, 0.01)
}

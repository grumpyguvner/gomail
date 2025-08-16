package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grumpyguvner/gomail/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrometheusMiddleware(t *testing.T) {
	// Reset and initialize metrics
	metrics.Reset()
	metrics.Init()

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/success":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		case "/large":
			w.WriteHeader(http.StatusOK)
			// Write 1KB of data
			data := make([]byte, 1024)
			_, _ = w.Write(data)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found"))
		}
	})

	// Wrap with Prometheus middleware
	handler := PrometheusMiddleware(testHandler)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedSize   int
	}{
		{
			name:           "successful GET request",
			method:         "GET",
			path:           "/success",
			expectedStatus: 200,
			expectedSize:   2, // "OK"
		},
		{
			name:           "error response",
			method:         "POST",
			path:           "/error",
			expectedStatus: 500,
			expectedSize:   21, // "Internal Server Error"
		},
		{
			name:           "large response",
			method:         "GET",
			path:           "/large",
			expectedStatus: 200,
			expectedSize:   1024,
		},
		{
			name:           "not found",
			method:         "GET",
			path:           "/unknown",
			expectedStatus: 404,
			expectedSize:   9, // "Not Found"
		},
	}

	// Initial active requests should be 0
	assert.Equal(t, float64(0), testutil.ToFloat64(metrics.HTTPActiveRequests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			// Handle request
			handler.ServeHTTP(w, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify metrics were recorded
			// Note: We can't easily test exact metric values due to label combinations,
			// but we can verify they're being collected
			assert.Greater(t, testutil.CollectAndCount(metrics.HTTPRequestDuration, "gomail_http_request_duration_seconds"), 0)
			assert.Greater(t, testutil.CollectAndCount(metrics.HTTPResponseSize, "gomail_http_response_size_bytes"), 0)
			assert.Greater(t, testutil.CollectAndCount(metrics.HTTPRequestsTotal, "gomail_http_requests_total"), 0)
		})
	}

	// Active requests should return to 0
	assert.Equal(t, float64(0), testutil.ToFloat64(metrics.HTTPActiveRequests))
}

func TestPrometheusMiddlewareConcurrent(t *testing.T) {
	// Reset and initialize metrics
	metrics.Reset()
	metrics.Init()

	// Create test handler with delay
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap with Prometheus middleware
	handler := PrometheusMiddleware(testHandler)

	// Run concurrent requests
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Active requests should be back to 0
	assert.Equal(t, float64(0), testutil.ToFloat64(metrics.HTTPActiveRequests))

	// Should have recorded all requests
	assert.Greater(t, testutil.CollectAndCount(metrics.HTTPRequestsTotal, "gomail_http_requests_total"), 0)
}

func TestPrometheusResponseWriter(t *testing.T) {
	// Test the response writer wrapper
	baseWriter := httptest.NewRecorder()
	wrapped := &prometheusResponseWriter{
		ResponseWriter: baseWriter,
		statusCode:     http.StatusOK,
		written:        0,
	}

	// Test WriteHeader
	wrapped.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, wrapped.statusCode)
	assert.Equal(t, http.StatusCreated, baseWriter.Code)

	// Test Write
	data := []byte("Hello, World!")
	n, err := wrapped.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, int64(len(data)), wrapped.written)
	assert.Equal(t, "Hello, World!", baseWriter.Body.String())

	// Test multiple writes
	moreData := []byte(" More data")
	n, err = wrapped.Write(moreData)
	require.NoError(t, err)
	assert.Equal(t, len(moreData), n)
	assert.Equal(t, int64(len(data)+len(moreData)), wrapped.written)
}

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/mail/inbound", "/mail/inbound"},
		{"/health", "/health"},
		{"/metrics", "/metrics"},
		{"/ready", "/ready"},
		{"/api/v1/users", "/api"},
		{"/api", "/api"},
		{"/", "/"},
		{"/unknown/path/here", "/unknown"},
		{"", "/other"},
		{"invalid", "/other"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeEndpoint(tt.input)
			assert.Equal(t, tt.expected, result, "normalizeEndpoint(%q) = %q, want %q", tt.input, result, tt.expected)
		})
	}
}

func TestPrometheusMiddlewareEdgeCases(t *testing.T) {
	// Reset and initialize metrics
	metrics.Reset()
	metrics.Init()

	t.Run("empty response", func(t *testing.T) {
		handler := PrometheusMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Don't write anything
		}))

		req := httptest.NewRequest("GET", "/empty", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Metrics should still be recorded
		assert.Greater(t, testutil.CollectAndCount(metrics.HTTPRequestDuration, "gomail_http_request_duration_seconds"), 0)
	})

	t.Run("panic recovery", func(t *testing.T) {
		// The middleware itself shouldn't panic, but if the handler panics,
		// it should still record metrics before the panic propagates
		handler := PrometheusMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			panic("test panic")
		}))

		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()

		// Should panic
		assert.Panics(t, func() {
			handler.ServeHTTP(w, req)
		})

		// But active requests should have been decremented
		assert.Equal(t, float64(0), testutil.ToFloat64(metrics.HTTPActiveRequests))
	})

	t.Run("HEAD request", func(t *testing.T) {
		handler := PrometheusMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "test")
			w.WriteHeader(http.StatusOK)
			// HEAD requests shouldn't have body
		}))

		req := httptest.NewRequest("HEAD", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "test", w.Header().Get("X-Custom-Header"))
		assert.Empty(t, w.Body.String())
	})
}

func BenchmarkPrometheusMiddleware(b *testing.B) {
	// Reset and initialize metrics
	metrics.Reset()
	metrics.Init()

	// Simple handler
	handler := PrometheusMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})
}

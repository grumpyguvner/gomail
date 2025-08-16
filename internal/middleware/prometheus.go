package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/grumpyguvner/gomail/internal/metrics"
)

// responseWriter wraps http.ResponseWriter to capture status code and response size
type prometheusResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

// WriteHeader captures the status code
func (w *prometheusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (w *prometheusResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}

// PrometheusMiddleware records HTTP metrics for each request
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track active requests
		metrics.HTTPActiveRequests.Inc()
		defer metrics.HTTPActiveRequests.Dec()

		// Wrap response writer to capture status and size
		wrapped := &prometheusResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			written:        0,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.statusCode)
		endpoint := r.URL.Path
		method := r.Method

		// Normalize endpoint to reduce cardinality
		endpoint = normalizeEndpoint(endpoint)

		// Record request duration
		metrics.HTTPRequestDuration.WithLabelValues(
			method,
			endpoint,
			status,
		).Observe(duration)

		// Record response size
		metrics.HTTPResponseSize.WithLabelValues(
			method,
			endpoint,
			status,
		).Observe(float64(wrapped.written))

		// Increment request counter
		metrics.HTTPRequestsTotal.WithLabelValues(
			method,
			endpoint,
			status,
		).Inc()
	})
}

// normalizeEndpoint reduces cardinality by grouping similar endpoints
func normalizeEndpoint(path string) string {
	// Map common endpoints to reduce cardinality
	switch path {
	case "/mail/inbound":
		return "/mail/inbound"
	case "/health":
		return "/health"
	case "/metrics":
		return "/metrics"
	case "/ready":
		return "/ready"
	default:
		// Group all other endpoints
		if len(path) > 0 && path[0] == '/' {
			// Check if it's a static file or unknown endpoint
			if len(path) > 1 {
				// Get first segment
				for i := 1; i < len(path); i++ {
					if path[i] == '/' {
						return path[:i]
					}
				}
			}
			return path
		}
		return "/other"
	}
}

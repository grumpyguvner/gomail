package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grumpyguvner/gomail/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutMiddleware(t *testing.T) {
	// Initialize metrics
	metrics.Init()

	t.Run("completes within timeout", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		})

		middleware := TimeoutMiddleware(100 * time.Millisecond)
		wrappedHandler := middleware(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "success")
	})

	t.Run("times out on slow handler", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
				// Handler detected timeout
				return
			case <-time.After(200 * time.Millisecond):
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("should not appear"))
			}
		})

		middleware := TimeoutMiddleware(50 * time.Millisecond)
		wrappedHandler := middleware(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		// Should return timeout error
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Contains(t, rec.Body.String(), "Request timeout")
	})

	t.Run("handles panic in handler", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		// Wrap handler with recovery first, then timeout
		recoveryHandler := RecoveryMiddleware(handler)
		middleware := TimeoutMiddleware(100 * time.Millisecond)
		wrappedHandler := middleware(recoveryHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// Should not panic
		wrappedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("prevents double write after timeout", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(60 * time.Millisecond)
			// Try to write after timeout
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("late response"))
		})

		middleware := TimeoutMiddleware(30 * time.Millisecond)
		wrappedHandler := middleware(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		// Should get timeout response, not the late response
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Contains(t, rec.Body.String(), "Request timeout")
		assert.NotContains(t, rec.Body.String(), "late response")
	})

	t.Run("passes context with timeout", func(t *testing.T) {
		var contextHasTimeout bool
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, hasDeadline := r.Context().Deadline()
			contextHasTimeout = hasDeadline
			w.WriteHeader(http.StatusOK)
		})

		middleware := TimeoutMiddleware(100 * time.Millisecond)
		wrappedHandler := middleware(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		assert.True(t, contextHasTimeout, "Context should have timeout deadline")
	})

	t.Run("multiple concurrent requests", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Parse delay from query param
			delayStr := r.URL.Query().Get("delay")
			switch delayStr {
			case "10":
				time.Sleep(10 * time.Millisecond)
			case "30":
				time.Sleep(30 * time.Millisecond)
			case "100":
				time.Sleep(100 * time.Millisecond)
			}
			w.WriteHeader(http.StatusOK)
		})

		middleware := TimeoutMiddleware(50 * time.Millisecond)
		wrappedHandler := middleware(handler)

		// Test multiple concurrent requests with different delays
		testCases := []struct {
			delay          string
			expectedStatus int
		}{
			{"10", http.StatusOK},                  // Fast - should succeed
			{"30", http.StatusOK},                  // Medium - should succeed
			{"100", http.StatusServiceUnavailable}, // Slow - should timeout
		}

		for _, tc := range testCases {
			req := httptest.NewRequest("GET", "/test?delay="+tc.delay, nil)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			if tc.expectedStatus == http.StatusServiceUnavailable {
				assert.Contains(t, rec.Body.String(), "Request timeout")
			}
		}
	})
}

func TestTimeoutWriter(t *testing.T) {
	t.Run("WriteHeader prevents double write", func(t *testing.T) {
		rec := httptest.NewRecorder()
		tw := &timeoutWriter{
			ResponseWriter: rec,
			written:        false,
		}

		// First write should succeed
		tw.WriteHeader(http.StatusOK)
		assert.True(t, tw.written)

		// Second write should be ignored
		tw.WriteHeader(http.StatusInternalServerError)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Write sets written flag", func(t *testing.T) {
		rec := httptest.NewRecorder()
		tw := &timeoutWriter{
			ResponseWriter: rec,
			written:        false,
		}

		_, err := tw.Write([]byte("test"))
		require.NoError(t, err)
		assert.True(t, tw.written)
	})

	t.Run("Write discards after timeout", func(t *testing.T) {
		rec := httptest.NewRecorder()
		tw := &timeoutWriter{
			ResponseWriter: rec,
			written:        false,
			timedOut:       true,
		}

		n, err := tw.Write([]byte("should be discarded"))
		require.NoError(t, err)
		assert.Equal(t, len("should be discarded"), n) // Returns length but doesn't write
		assert.Empty(t, rec.Body.String())
	})

	t.Run("Flush works when supported", func(t *testing.T) {
		rec := httptest.NewRecorder()
		tw := &timeoutWriter{
			ResponseWriter: rec,
		}

		// Should not panic even though httptest.ResponseRecorder doesn't implement Flusher
		tw.Flush()
	})
}

func BenchmarkTimeoutMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	middleware := TimeoutMiddleware(100 * time.Millisecond)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}

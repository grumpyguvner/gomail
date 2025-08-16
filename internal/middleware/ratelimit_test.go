package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRateLimiter_Allow(t *testing.T) {
	logger := zap.NewNop()
	rl := NewRateLimiter(60, 10, time.Minute, logger)

	// First 10 requests should be allowed (burst size)
	for i := 0; i < 10; i++ {
		assert.True(t, rl.Allow("test-ip"), "Request %d should be allowed", i+1)
	}

	// 11th request should be denied
	assert.False(t, rl.Allow("test-ip"), "11th request should be denied")
}

func TestRateLimiter_Refill(t *testing.T) {
	logger := zap.NewNop()
	// 60 tokens per minute = 1 token per second
	rl := NewRateLimiter(60, 2, time.Minute, logger)

	// Use up all tokens
	assert.True(t, rl.Allow("test-ip"))
	assert.True(t, rl.Allow("test-ip"))
	assert.False(t, rl.Allow("test-ip"))

	// Wait for refill (1 second = 1 token)
	time.Sleep(1100 * time.Millisecond)

	// Should have 1 token now
	assert.True(t, rl.Allow("test-ip"))
	assert.False(t, rl.Allow("test-ip"))
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	logger := zap.NewNop()
	rl := NewRateLimiter(60, 2, time.Minute, logger)

	// Different IPs should have separate buckets
	assert.True(t, rl.Allow("ip1"))
	assert.True(t, rl.Allow("ip1"))
	assert.False(t, rl.Allow("ip1"))

	assert.True(t, rl.Allow("ip2"))
	assert.True(t, rl.Allow("ip2"))
	assert.False(t, rl.Allow("ip2"))
}

func TestRateLimiter_Middleware(t *testing.T) {
	logger := zap.NewNop()
	rl := NewRateLimiter(60, 2, time.Minute, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	middleware := rl.Middleware(handler)

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "60", rec.Header().Get("Retry-After"))
	assert.Contains(t, rec.Body.String(), "Rate limit exceeded")
}

func TestRateLimiter_GetClientIP(t *testing.T) {
	logger := zap.NewNop()
	rl := NewRateLimiter(60, 10, time.Minute, logger)

	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1"},
			remoteAddr: "127.0.0.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 192.168.1.1"},
			remoteAddr: "127.0.0.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "192.168.1.1"},
			remoteAddr: "127.0.0.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := rl.getClientIP(req)
			assert.Equal(t, tt.expected, ip)
		})
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	logger := zap.NewNop()
	// Short TTL for testing
	rl := NewRateLimiter(60, 10, 100*time.Millisecond, logger)

	// Create a bucket
	assert.True(t, rl.Allow("test-ip"))

	// Verify bucket exists
	rl.mu.RLock()
	_, exists := rl.buckets["test-ip"]
	rl.mu.RUnlock()
	assert.True(t, exists)

	// Wait for cleanup
	time.Sleep(250 * time.Millisecond)

	// Bucket should be cleaned up
	rl.mu.RLock()
	_, exists = rl.buckets["test-ip"]
	rl.mu.RUnlock()
	assert.False(t, exists)
}

func TestRateLimiter_GetRemaining(t *testing.T) {
	logger := zap.NewNop()
	rl := NewRateLimiter(60, 5, time.Minute, logger)

	// New IP should have full burst
	remaining := rl.getRemaining("new-ip")
	assert.Equal(t, 5, remaining)

	// Use some tokens
	require.True(t, rl.Allow("test-ip"))
	require.True(t, rl.Allow("test-ip"))

	// Should have 3 remaining
	remaining = rl.getRemaining("test-ip")
	assert.Equal(t, 3, remaining)
}

package middleware

import (
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	buckets map[string]*bucket
	mu      sync.RWMutex
	rate    int           // tokens per interval
	burst   int           // max tokens in bucket
	ttl     time.Duration // time to live for idle buckets
	logger  *zap.Logger
}

// bucket represents a token bucket for a specific key
type bucket struct {
	tokens   int
	lastFill time.Time
	lastUsed time.Time
	mu       sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, burst int, ttl time.Duration, logger *zap.Logger) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
		ttl:     ttl,
		logger:  logger,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Middleware returns the rate limiting middleware handler
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		ip := rl.getClientIP(r)

		// Check rate limit
		if !rl.Allow(ip) {
			rl.logger.Warn("Rate limit exceeded",
				zap.String("ip", ip),
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
			)

			w.Header().Set("X-RateLimit-Limit", string(rune(rl.rate)))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Get remaining tokens for headers
		remaining := rl.getRemaining(ip)
		w.Header().Set("X-RateLimit-Limit", string(rune(rl.rate)))
		w.Header().Set("X-RateLimit-Remaining", string(rune(remaining)))

		next.ServeHTTP(w, r)
	})
}

// Allow checks if a request from the given key is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	b, exists := rl.buckets[key]
	if !exists {
		b = &bucket{
			tokens:   rl.burst,
			lastFill: time.Now(),
			lastUsed: time.Now(),
		}
		rl.buckets[key] = b
		rl.mu.Unlock()
	} else {
		rl.mu.Unlock()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on time passed
	now := time.Now()
	elapsed := now.Sub(b.lastFill)
	tokensToAdd := int(elapsed.Seconds()) * rl.rate / 60 // rate is per minute

	if tokensToAdd > 0 {
		b.tokens = min(b.tokens+tokensToAdd, rl.burst)
		b.lastFill = now
	}

	// Check if we have tokens available
	if b.tokens > 0 {
		b.tokens--
		b.lastUsed = now
		return true
	}

	return false
}

// getRemaining returns the number of remaining tokens for a key
func (rl *RateLimiter) getRemaining(key string) int {
	rl.mu.RLock()
	b, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if !exists {
		return rl.burst
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Calculate current tokens after refill
	now := time.Now()
	elapsed := now.Sub(b.lastFill)
	tokensToAdd := int(elapsed.Seconds()) * rl.rate / 60

	if tokensToAdd > 0 {
		return min(b.tokens+tokensToAdd, rl.burst)
	}

	return b.tokens
}

// cleanup removes idle buckets periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.ttl)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.buckets {
			b.mu.Lock()
			if now.Sub(b.lastUsed) > rl.ttl {
				delete(rl.buckets, key)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// getClientIP extracts the client IP from the request
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP if multiple are present
		if idx := len(xff) - 1; idx > 0 {
			for i := idx; i >= 0; i-- {
				if xff[i] == ',' || xff[i] == ' ' {
					return xff[i+1:]
				}
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	// Fall back to RemoteAddr
	if idx := len(r.RemoteAddr) - 1; idx > 0 {
		for i := idx; i >= 0; i-- {
			if r.RemoteAddr[i] == ':' {
				return r.RemoteAddr[:i]
			}
		}
	}

	return r.RemoteAddr
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

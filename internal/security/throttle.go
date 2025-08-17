package security

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/grumpyguvner/gomail/internal/metrics"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// ConnectionThrottle implements connection rate limiting and throttling
type ConnectionThrottle struct {
	// Global rate limiter
	globalLimiter *rate.Limiter

	// Per-IP rate limiters
	ipLimiters map[string]*ipLimiter
	mu         sync.RWMutex

	// Configuration
	globalRate    rate.Limit // Connections per second globally
	globalBurst   int
	perIPRate     rate.Limit // Connections per second per IP
	perIPBurst    int
	cleanupPeriod time.Duration

	logger *zap.SugaredLogger
}

// ipLimiter tracks rate limiting for a specific IP
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewConnectionThrottle creates a new connection throttle
func NewConnectionThrottle(globalRate, perIPRate float64) *ConnectionThrottle {
	ct := &ConnectionThrottle{
		globalLimiter: rate.NewLimiter(rate.Limit(globalRate), int(globalRate*2)),
		ipLimiters:    make(map[string]*ipLimiter),
		globalRate:    rate.Limit(globalRate),
		globalBurst:   int(globalRate * 2),
		perIPRate:     rate.Limit(perIPRate),
		perIPBurst:    int(perIPRate * 2),
		cleanupPeriod: 5 * time.Minute,
		logger:        logging.Get(),
	}

	// Start cleanup routine
	go ct.cleanupLoop()

	return ct
}

// Allow checks if a connection should be allowed based on rate limits
func (ct *ConnectionThrottle) Allow(ip string) bool {
	// Check global rate limit first
	if !ct.globalLimiter.Allow() {
		metrics.ThrottleRejections.WithLabelValues("global").Inc()
		ct.logger.Debugf("Connection throttled: global rate limit exceeded")
		return false
	}

	// Check per-IP rate limit
	limiter := ct.getIPLimiter(ip)
	if !limiter.Allow() {
		metrics.ThrottleRejections.WithLabelValues("per_ip").Inc()
		ct.logger.Debugf("Connection throttled: per-IP rate limit exceeded for %s", ip)
		return false
	}

	metrics.ThrottleAllowed.Inc()
	return true
}

// AllowN checks if n events should be allowed
func (ct *ConnectionThrottle) AllowN(ip string, n int) bool {
	// Check global rate limit
	if !ct.globalLimiter.AllowN(time.Now(), n) {
		metrics.ThrottleRejections.WithLabelValues("global").Add(float64(n))
		return false
	}

	// Check per-IP rate limit
	limiter := ct.getIPLimiter(ip)
	if !limiter.AllowN(time.Now(), n) {
		metrics.ThrottleRejections.WithLabelValues("per_ip").Add(float64(n))
		return false
	}

	metrics.ThrottleAllowed.Add(float64(n))
	return true
}

// Wait blocks until the connection is allowed or context is cancelled
func (ct *ConnectionThrottle) Wait(ctx context.Context, ip string) error {
	// Wait for global rate limit
	if err := ct.globalLimiter.Wait(ctx); err != nil {
		metrics.ThrottleRejections.WithLabelValues("global").Inc()
		return fmt.Errorf("global rate limit: %w", err)
	}

	// Wait for per-IP rate limit
	limiter := ct.getIPLimiter(ip)
	if err := limiter.Wait(ctx); err != nil {
		metrics.ThrottleRejections.WithLabelValues("per_ip").Inc()
		return fmt.Errorf("per-IP rate limit: %w", err)
	}

	metrics.ThrottleAllowed.Inc()
	return nil
}

// Reserve reserves a connection slot
func (ct *ConnectionThrottle) Reserve(ip string) *Reservation {
	globalRes := ct.globalLimiter.Reserve()
	ipRes := ct.getIPLimiter(ip).Reserve()

	return &Reservation{
		global: globalRes,
		perIP:  ipRes,
	}
}

// getIPLimiter gets or creates a rate limiter for an IP
func (ct *ConnectionThrottle) getIPLimiter(ip string) *rate.Limiter {
	// Extract IP without port
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		host = ip
	}

	ct.mu.RLock()
	if limiter, exists := ct.ipLimiters[host]; exists {
		limiter.lastSeen = time.Now()
		ct.mu.RUnlock()
		return limiter.limiter
	}
	ct.mu.RUnlock()

	// Create new limiter
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := ct.ipLimiters[host]; exists {
		limiter.lastSeen = time.Now()
		return limiter.limiter
	}

	newLimiter := &ipLimiter{
		limiter:  rate.NewLimiter(ct.perIPRate, ct.perIPBurst),
		lastSeen: time.Now(),
	}
	ct.ipLimiters[host] = newLimiter

	metrics.UniqueIPs.Set(float64(len(ct.ipLimiters)))

	return newLimiter.limiter
}

// cleanupLoop periodically removes old IP limiters
func (ct *ConnectionThrottle) cleanupLoop() {
	ticker := time.NewTicker(ct.cleanupPeriod)
	defer ticker.Stop()

	for range ticker.C {
		ct.cleanup()
	}
}

// cleanup removes IP limiters that haven't been used recently
func (ct *ConnectionThrottle) cleanup() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	cutoff := time.Now().Add(-ct.cleanupPeriod)
	removed := 0

	for ip, limiter := range ct.ipLimiters {
		if limiter.lastSeen.Before(cutoff) {
			delete(ct.ipLimiters, ip)
			removed++
		}
	}

	if removed > 0 {
		ct.logger.Debugf("Cleaned up %d inactive IP limiters", removed)
		metrics.UniqueIPs.Set(float64(len(ct.ipLimiters)))
	}
}

// UpdateRates updates the rate limits dynamically
func (ct *ConnectionThrottle) UpdateRates(globalRate, perIPRate float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Update global limiter
	ct.globalRate = rate.Limit(globalRate)
	ct.globalBurst = int(globalRate * 2)
	ct.globalLimiter = rate.NewLimiter(ct.globalRate, ct.globalBurst)

	// Update per-IP configuration
	ct.perIPRate = rate.Limit(perIPRate)
	ct.perIPBurst = int(perIPRate * 2)

	// Clear existing IP limiters to apply new rates
	ct.ipLimiters = make(map[string]*ipLimiter)

	ct.logger.Infof("Updated throttle rates: global=%.2f/s, per-IP=%.2f/s", globalRate, perIPRate)
}

// GetStats returns current throttle statistics
func (ct *ConnectionThrottle) GetStats() ThrottleStats {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return ThrottleStats{
		GlobalRate:  float64(ct.globalRate),
		GlobalBurst: ct.globalBurst,
		PerIPRate:   float64(ct.perIPRate),
		PerIPBurst:  ct.perIPBurst,
		ActiveIPs:   len(ct.ipLimiters),
	}
}

// Reservation represents a reserved connection slot
type Reservation struct {
	global *rate.Reservation
	perIP  *rate.Reservation
}

// Cancel cancels the reservation
func (r *Reservation) Cancel() {
	if r.global != nil {
		r.global.Cancel()
	}
	if r.perIP != nil {
		r.perIP.Cancel()
	}
}

// Delay returns the delay until the reservation can be used
func (r *Reservation) Delay() time.Duration {
	globalDelay := r.global.Delay()
	perIPDelay := r.perIP.Delay()

	if globalDelay > perIPDelay {
		return globalDelay
	}
	return perIPDelay
}

// ThrottleStats holds throttle statistics
type ThrottleStats struct {
	GlobalRate  float64 `json:"global_rate"`
	GlobalBurst int     `json:"global_burst"`
	PerIPRate   float64 `json:"per_ip_rate"`
	PerIPBurst  int     `json:"per_ip_burst"`
	ActiveIPs   int     `json:"active_ips"`
}

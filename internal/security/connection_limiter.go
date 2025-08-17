package security

import (
	"net"
	"sync"
	"time"

	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/grumpyguvner/gomail/internal/metrics"
	"go.uber.org/zap"
)

// ConnectionLimiter manages connection limits and bans
type ConnectionLimiter struct {
	maxPerIP     int
	maxTotal     int
	banDuration  time.Duration
	banThreshold int // Number of violations before ban

	connections map[string]int
	violations  map[string]int
	bannedIPs   map[string]time.Time
	totalConns  int
	mu          sync.RWMutex

	logger *zap.SugaredLogger
}

// NewConnectionLimiter creates a new connection limiter
func NewConnectionLimiter(maxPerIP, maxTotal int, banDuration time.Duration) *ConnectionLimiter {
	cl := &ConnectionLimiter{
		maxPerIP:     maxPerIP,
		maxTotal:     maxTotal,
		banDuration:  banDuration,
		banThreshold: 5, // Ban after 5 violations
		connections:  make(map[string]int),
		violations:   make(map[string]int),
		bannedIPs:    make(map[string]time.Time),
		logger:       logging.Get(),
	}

	// Start cleanup goroutine
	go cl.cleanupLoop()

	return cl
}

// Accept checks if a connection from the given IP should be accepted
func (cl *ConnectionLimiter) Accept(ip string) bool {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	// Extract IP without port
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		host = ip
	}

	// Check if IP is banned
	if banTime, banned := cl.bannedIPs[host]; banned {
		if time.Now().Before(banTime) {
			metrics.ConnectionsRejected.WithLabelValues("banned").Inc()
			cl.logger.Warnf("Connection rejected: IP %s is banned until %v", host, banTime)
			return false
		}
		// Ban expired, remove it
		delete(cl.bannedIPs, host)
		delete(cl.violations, host)
	}

	// Check total connection limit
	if cl.totalConns >= cl.maxTotal {
		metrics.ConnectionsRejected.WithLabelValues("max_total").Inc()
		cl.logger.Warnf("Connection rejected: Total connection limit reached (%d/%d)", cl.totalConns, cl.maxTotal)
		cl.recordViolation(host)
		return false
	}

	// Check per-IP limit
	if cl.connections[host] >= cl.maxPerIP {
		metrics.ConnectionsRejected.WithLabelValues("max_per_ip").Inc()
		cl.logger.Warnf("Connection rejected: Per-IP limit reached for %s (%d/%d)", host, cl.connections[host], cl.maxPerIP)
		cl.recordViolation(host)
		return false
	}

	// Accept connection
	cl.connections[host]++
	cl.totalConns++
	metrics.ConnectionsAccepted.Inc()
	metrics.ActiveConnections.Set(float64(cl.totalConns))
	metrics.ConnectionsPerIP.WithLabelValues(host).Set(float64(cl.connections[host]))

	return true
}

// Release releases a connection slot
func (cl *ConnectionLimiter) Release(ip string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		host = ip
	}

	if count := cl.connections[host]; count > 0 {
		cl.connections[host]--
		if cl.connections[host] == 0 {
			delete(cl.connections, host)
		}
		metrics.ConnectionsPerIP.WithLabelValues(host).Set(float64(cl.connections[host]))
	}

	if cl.totalConns > 0 {
		cl.totalConns--
		metrics.ActiveConnections.Set(float64(cl.totalConns))
	}
}

// Ban manually bans an IP address
func (cl *ConnectionLimiter) Ban(ip string, duration time.Duration) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if duration == 0 {
		duration = cl.banDuration
	}

	cl.bannedIPs[ip] = time.Now().Add(duration)
	metrics.BannedIPs.Inc()
	cl.logger.Infof("IP %s banned for %v", ip, duration)
}

// Unban removes an IP from the ban list
func (cl *ConnectionLimiter) Unban(ip string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if _, exists := cl.bannedIPs[ip]; exists {
		delete(cl.bannedIPs, ip)
		delete(cl.violations, ip)
		metrics.BannedIPs.Dec()
		cl.logger.Infof("IP %s unbanned", ip)
	}
}

// IsBanned checks if an IP is currently banned
func (cl *ConnectionLimiter) IsBanned(ip string) bool {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	if banTime, banned := cl.bannedIPs[ip]; banned {
		return time.Now().Before(banTime)
	}
	return false
}

// GetBannedIPs returns a list of currently banned IPs
func (cl *ConnectionLimiter) GetBannedIPs() map[string]time.Time {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	result := make(map[string]time.Time)
	for ip, banTime := range cl.bannedIPs {
		if time.Now().Before(banTime) {
			result[ip] = banTime
		}
	}
	return result
}

// GetConnectionStats returns current connection statistics
func (cl *ConnectionLimiter) GetConnectionStats() ConnectionStats {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	return ConnectionStats{
		TotalConnections: cl.totalConns,
		MaxTotal:         cl.maxTotal,
		MaxPerIP:         cl.maxPerIP,
		ConnectionsByIP:  cl.copyStringIntMap(cl.connections),
		BannedIPs:        len(cl.bannedIPs),
	}
}

// recordViolation records a connection limit violation
func (cl *ConnectionLimiter) recordViolation(ip string) {
	cl.violations[ip]++

	if cl.violations[ip] >= cl.banThreshold {
		cl.bannedIPs[ip] = time.Now().Add(cl.banDuration)
		metrics.BannedIPs.Inc()
		cl.logger.Warnf("IP %s banned due to %d violations", ip, cl.violations[ip])
		delete(cl.violations, ip)
	}
}

// cleanupLoop periodically cleans up expired bans and old violation records
func (cl *ConnectionLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cl.cleanup()
	}
}

// cleanup removes expired bans and old violation records
func (cl *ConnectionLimiter) cleanup() {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	now := time.Now()

	// Clean up expired bans
	for ip, banTime := range cl.bannedIPs {
		if now.After(banTime) {
			delete(cl.bannedIPs, ip)
			metrics.BannedIPs.Dec()
			cl.logger.Debugf("Ban expired for IP %s", ip)
		}
	}

	// Clean up old violation records (older than 1 hour)
	// This is a simplified approach - in production you'd track timestamps
	if len(cl.violations) > 1000 {
		// Reset violations if map gets too large
		cl.violations = make(map[string]int)
	}
}

// copyStringIntMap creates a copy of a string->int map
func (cl *ConnectionLimiter) copyStringIntMap(m map[string]int) map[string]int {
	result := make(map[string]int, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// ConnectionStats holds connection statistics
type ConnectionStats struct {
	TotalConnections int            `json:"total_connections"`
	MaxTotal         int            `json:"max_total"`
	MaxPerIP         int            `json:"max_per_ip"`
	ConnectionsByIP  map[string]int `json:"connections_by_ip"`
	BannedIPs        int            `json:"banned_ips"`
}

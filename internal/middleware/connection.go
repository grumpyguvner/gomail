package middleware

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/grumpyguvner/gomail/internal/metrics"
	"github.com/grumpyguvner/gomail/internal/security"
	"go.uber.org/zap"
)

// ConnectionMiddleware manages connection security
type ConnectionMiddleware struct {
	limiter  *security.ConnectionLimiter
	throttle *security.ConnectionThrottle
	logger   *zap.SugaredLogger
}

// NewConnectionMiddleware creates a new connection middleware
func NewConnectionMiddleware(maxPerIP, maxTotal int, globalRate, perIPRate float64) *ConnectionMiddleware {
	return &ConnectionMiddleware{
		limiter:  security.NewConnectionLimiter(maxPerIP, maxTotal, 24*time.Hour),
		throttle: security.NewConnectionThrottle(globalRate, perIPRate),
		logger:   logging.Get(),
	}
}

// HTTPMiddleware returns an HTTP middleware for connection control
func (cm *ConnectionMiddleware) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)

		// Check if IP is banned
		if cm.limiter.IsBanned(clientIP) {
			metrics.SecurityViolations.WithLabelValues("banned_ip").Inc()
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Check connection limits
		if !cm.limiter.Accept(clientIP) {
			http.Error(w, "Too Many Connections", http.StatusTooManyRequests)
			return
		}
		defer cm.limiter.Release(clientIP)

		// Check throttle
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := cm.throttle.Wait(ctx, clientIP); err != nil {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// TCPMiddleware handles TCP connection control for SMTP
func (cm *ConnectionMiddleware) TCPMiddleware(conn net.Conn) (net.Conn, error) {
	clientIP := conn.RemoteAddr().String()

	// Check if IP is banned
	if cm.limiter.IsBanned(clientIP) {
		metrics.SecurityViolations.WithLabelValues("banned_ip").Inc()
		conn.Close()
		return nil, net.Error(&net.OpError{
			Op:  "accept",
			Net: "tcp",
			Err: &net.AddrError{
				Err:  "connection refused: banned IP",
				Addr: clientIP,
			},
		})
	}

	// Check connection limits
	if !cm.limiter.Accept(clientIP) {
		conn.Close()
		return nil, net.Error(&net.OpError{
			Op:  "accept",
			Net: "tcp",
			Err: &net.AddrError{
				Err:  "connection refused: too many connections",
				Addr: clientIP,
			},
		})
	}

	// Check throttle
	if !cm.throttle.Allow(clientIP) {
		cm.limiter.Release(clientIP)
		conn.Close()
		return nil, net.Error(&net.OpError{
			Op:  "accept",
			Net: "tcp",
			Err: &net.AddrError{
				Err:  "connection refused: rate limited",
				Addr: clientIP,
			},
		})
	}

	// Wrap connection to release on close
	return &managedConn{
		Conn:     conn,
		clientIP: clientIP,
		limiter:  cm.limiter,
		onClose:  func() { cm.limiter.Release(clientIP) },
	}, nil
}

// BanIP manually bans an IP address
func (cm *ConnectionMiddleware) BanIP(ip string, duration time.Duration) {
	cm.limiter.Ban(ip, duration)
}

// UnbanIP removes an IP from the ban list
func (cm *ConnectionMiddleware) UnbanIP(ip string) {
	cm.limiter.Unban(ip)
}

// GetBannedIPs returns currently banned IPs
func (cm *ConnectionMiddleware) GetBannedIPs() map[string]time.Time {
	return cm.limiter.GetBannedIPs()
}

// GetStats returns connection statistics
func (cm *ConnectionMiddleware) GetStats() ConnectionStats {
	return ConnectionStats{
		ConnectionLimits: cm.limiter.GetConnectionStats(),
		ThrottleStats:    cm.throttle.GetStats(),
	}
}

// UpdateLimits updates connection limits dynamically
func (cm *ConnectionMiddleware) UpdateLimits(maxPerIP, maxTotal int, globalRate, perIPRate float64) {
	// Create new instances with updated limits
	cm.limiter = security.NewConnectionLimiter(maxPerIP, maxTotal, 24*time.Hour)
	cm.throttle.UpdateRates(globalRate, perIPRate)
	cm.logger.Infof("Updated connection limits: maxPerIP=%d, maxTotal=%d, globalRate=%.2f/s, perIPRate=%.2f/s",
		maxPerIP, maxTotal, globalRate, perIPRate)
}

// managedConn wraps a connection to track its lifecycle
type managedConn struct {
	net.Conn
	clientIP string
	limiter  *security.ConnectionLimiter
	onClose  func()
	closed   bool
}

// Close releases the connection and calls cleanup
func (mc *managedConn) Close() error {
	if !mc.closed {
		mc.closed = true
		if mc.onClose != nil {
			mc.onClose()
		}
	}
	return mc.Conn.Close()
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if idx := len(xff) - 1; idx >= 0 {
			for i := idx; i >= 0; i-- {
				if xff[i] == ',' || xff[i] == ' ' {
					return xff[i+1:]
				}
			}
			return xff
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

// ConnectionStats holds comprehensive connection statistics
type ConnectionStats struct {
	ConnectionLimits security.ConnectionStats `json:"connection_limits"`
	ThrottleStats    security.ThrottleStats   `json:"throttle_stats"`
}

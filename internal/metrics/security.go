package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Connection limiting metrics
	ConnectionsAccepted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_connections_accepted_total",
		Help: "Total number of accepted connections",
	})

	ConnectionsRejected = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gomail_connections_rejected_total",
		Help: "Total number of rejected connections by reason",
	}, []string{"reason"})

	ActiveConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gomail_active_connections",
		Help: "Current number of active connections",
	})

	ConnectionsPerIP = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gomail_connections_per_ip",
		Help: "Number of connections per IP address",
	}, []string{"ip"})

	BannedIPs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gomail_banned_ips",
		Help: "Current number of banned IP addresses",
	})

	// Throttling metrics
	ThrottleAllowed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_throttle_allowed_total",
		Help: "Total number of connections allowed by throttle",
	})

	ThrottleRejections = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gomail_throttle_rejections_total",
		Help: "Total number of connections rejected by throttle",
	}, []string{"type"})

	ThrottleWaitTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "gomail_throttle_wait_seconds",
		Help:    "Time spent waiting for throttle approval",
		Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
	})

	UniqueIPs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gomail_unique_ips",
		Help: "Number of unique IP addresses seen",
	})

	// Security event metrics
	SecurityViolations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gomail_security_violations_total",
		Help: "Total number of security violations by type",
	}, []string{"type"})

	IPReputationChecks = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gomail_ip_reputation_checks_total",
		Help: "IP reputation check results",
	}, []string{"result"})

	// Connection abuse metrics
	ConnectionAbuseDetected = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gomail_connection_abuse_detected_total",
		Help: "Connection abuse patterns detected",
	}, []string{"pattern"})

	// Firewall metrics
	FirewallRulesApplied = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_firewall_rules_applied_total",
		Help: "Total number of firewall rules applied",
	})

	FirewallBlockedConnections = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_firewall_blocked_connections_total",
		Help: "Total number of connections blocked by firewall",
	})
)

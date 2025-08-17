package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// TLSConnections tracks total TLS connections
	TLSConnections = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_tls_connections_total",
		Help: "Total number of TLS connections established",
	})

	// TLSHandshakeErrors tracks TLS handshake failures
	TLSHandshakeErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_tls_handshake_errors_total",
		Help: "Total number of TLS handshake errors",
	})

	// TLSVersion tracks TLS versions used
	TLSVersion = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gomail_tls_version_total",
		Help: "TLS connections by version",
	}, []string{"version"})

	// TLSCipherSuite tracks cipher suites used
	TLSCipherSuite = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gomail_tls_cipher_suite_total",
		Help: "TLS connections by cipher suite",
	}, []string{"cipher_suite"})

	// STARTTLSCommands tracks STARTTLS command usage
	STARTTLSCommands = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_starttls_commands_total",
		Help: "Total number of STARTTLS commands received",
	})

	// TLSCertificateExpiry tracks certificate expiration time
	TLSCertificateExpiry = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gomail_tls_certificate_expiry_timestamp_seconds",
		Help: "Unix timestamp of TLS certificate expiration",
	})

	// TLSHandshakeDuration tracks handshake duration
	TLSHandshakeDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "gomail_tls_handshake_duration_seconds",
		Help:    "TLS handshake duration in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})

	// PlaintextConnections tracks non-TLS connections
	PlaintextConnections = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_plaintext_connections_total",
		Help: "Total number of plaintext (non-TLS) connections",
	})

	// TLSRequiredRejections tracks connections rejected due to TLS requirement
	TLSRequiredRejections = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gomail_tls_required_rejections_total",
		Help: "Total number of connections rejected due to TLS requirement",
	})
)

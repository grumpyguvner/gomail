package tls

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/logging"
)

// Config represents TLS configuration
type Config struct {
	Enabled      bool
	CertFile     string
	KeyFile      string
	MinVersion   uint16
	CipherSuites []uint16
	config       *config.Config
}

// NewConfig creates a new TLS configuration
func NewConfig(cfg *config.Config) *Config {
	return &Config{
		config:     cfg,
		MinVersion: tls.VersionTLS12, // Enforce TLS 1.2 minimum
		CipherSuites: []uint16{
			// TLS 1.3 cipher suites (automatically selected when TLS 1.3 is negotiated)
			// These are implicit and don't need to be specified

			// TLS 1.2 cipher suites - only strong ones
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		},
	}
}

// LoadCertificates loads TLS certificates from configured paths
func (c *Config) LoadCertificates() error {
	certDir := "/etc/mailserver/certs"
	c.CertFile = filepath.Join(certDir, "cert.pem")
	c.KeyFile = filepath.Join(certDir, "key.pem")

	// Check if certificates exist
	if _, err := os.Stat(c.CertFile); os.IsNotExist(err) {
		return fmt.Errorf("certificate file not found: %s", c.CertFile)
	}
	if _, err := os.Stat(c.KeyFile); os.IsNotExist(err) {
		return fmt.Errorf("key file not found: %s", c.KeyFile)
	}

	c.Enabled = true
	return nil
}

// GetTLSConfig returns a configured tls.Config
func (c *Config) GetTLSConfig() (*tls.Config, error) {
	if !c.Enabled {
		return nil, fmt.Errorf("TLS is not enabled")
	}

	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	return &tls.Config{
		Certificates:                []tls.Certificate{cert},
		MinVersion:                  c.MinVersion,
		CipherSuites:                c.CipherSuites,
		PreferServerCipherSuites:    true,
		SessionTicketsDisabled:      false, // Enable session resumption for performance
		DynamicRecordSizingDisabled: false,

		// Security settings
		InsecureSkipVerify: false,
		Renegotiation:      tls.RenegotiateNever,

		// Enable OCSP stapling
		// This will be handled by autocert if using Let's Encrypt

		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
	}, nil
}

// ConfigurePostfixTLS updates Postfix configuration for TLS
func (c *Config) ConfigurePostfixTLS() error {
	logger := logging.Get()

	if !c.Enabled {
		logger.Warn("TLS not enabled, skipping Postfix TLS configuration")
		return nil
	}

	// Configure Postfix TLS parameters
	tlsParams := map[string]string{
		// Server-side TLS (receiving mail)
		"smtpd_use_tls":                       "yes",
		"smtpd_tls_security_level":            "may", // Opportunistic TLS
		"smtpd_tls_cert_file":                 c.CertFile,
		"smtpd_tls_key_file":                  c.KeyFile,
		"smtpd_tls_protocols":                 "!SSLv2, !SSLv3, !TLSv1, !TLSv1.1",
		"smtpd_tls_mandatory_protocols":       "!SSLv2, !SSLv3, !TLSv1, !TLSv1.1",
		"smtpd_tls_ciphers":                   "high",
		"smtpd_tls_mandatory_ciphers":         "high",
		"smtpd_tls_exclude_ciphers":           "aNULL, MD5, DES, ADH, RC4, PSD, SRP, 3DES, eNULL",
		"smtpd_tls_mandatory_exclude_ciphers": "aNULL, MD5, DES, ADH, RC4, PSD, SRP, 3DES, eNULL",
		"smtpd_tls_loglevel":                  "1",
		"smtpd_tls_received_header":           "yes",
		"smtpd_tls_session_cache_database":    "btree:${data_directory}/smtpd_scache",
		"smtpd_tls_session_cache_timeout":     "3600s",
		"smtpd_tls_dh1024_param_file":         "/etc/postfix/dh2048.pem",

		// Client-side TLS (sending mail)
		"smtp_use_tls":                    "yes",
		"smtp_tls_security_level":         "may", // Opportunistic TLS
		"smtp_tls_protocols":              "!SSLv2, !SSLv3, !TLSv1, !TLSv1.1",
		"smtp_tls_mandatory_protocols":    "!SSLv2, !SSLv3, !TLSv1, !TLSv1.1",
		"smtp_tls_ciphers":                "high",
		"smtp_tls_mandatory_ciphers":      "high",
		"smtp_tls_exclude_ciphers":        "aNULL, MD5, DES, ADH, RC4, PSD, SRP, 3DES, eNULL",
		"smtp_tls_loglevel":               "1",
		"smtp_tls_session_cache_database": "btree:${data_directory}/smtp_scache",
		"smtp_tls_session_cache_timeout":  "3600s",

		// Additional security
		"tls_random_source":   "dev:/dev/urandom",
		"tls_high_cipherlist": "ECDHE+AESGCM:ECDHE+AES256:ECDHE+AES128:!aNULL:!MD5:!DSS",
	}

	// Apply configuration using postconf
	for key, value := range tlsParams {
		if err := UpdatePostfixConfig(key, value); err != nil {
			logger.Warnf("Failed to set %s: %v", key, err)
		}
	}

	// Generate DH parameters if not exists
	if err := GenerateDHParams(); err != nil {
		logger.Warnf("Failed to generate DH parameters: %v", err)
	}

	logger.Info("Postfix TLS configuration updated successfully")
	return nil
}

// GetCipherSuiteNames returns human-readable names for configured cipher suites
func (c *Config) GetCipherSuiteNames() []string {
	names := make([]string, 0, len(c.CipherSuites))
	for _, suite := range c.CipherSuites {
		if name := tls.CipherSuiteName(suite); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// ValidateTLSConfig performs validation checks on TLS configuration
func (c *Config) ValidateTLSConfig() error {
	if !c.Enabled {
		return nil
	}

	// Try to load the certificates
	_, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return fmt.Errorf("invalid certificate/key pair: %w", err)
	}

	// Check minimum TLS version
	if c.MinVersion < tls.VersionTLS12 {
		return fmt.Errorf("minimum TLS version must be 1.2 or higher")
	}

	// Ensure we have cipher suites configured
	if len(c.CipherSuites) == 0 {
		return fmt.Errorf("no cipher suites configured")
	}

	return nil
}

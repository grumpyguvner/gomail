package ssl

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/logging"
)

// Manager is now an alias for LegoManager for backward compatibility
type Manager struct {
	*LegoManager
}

// NewManager creates a new SSL manager (using lego)
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		LegoManager: NewLegoManager(cfg),
	}
}


// ConfigurePostfix configures Postfix to use the SSL certificate
func (m *Manager) ConfigurePostfix() error {
	logger := logging.Get()

	certPath := "/etc/mailserver/certs/cert.pem"
	keyPath := "/etc/mailserver/certs/key.pem"

	// Update Postfix configuration
	postfixConfig := []struct {
		key   string
		value string
	}{
		{"smtpd_tls_cert_file", certPath},
		{"smtpd_tls_key_file", keyPath},
		{"smtpd_use_tls", "yes"},
		{"smtpd_tls_security_level", "may"},
		{"smtpd_tls_protocols", "!SSLv2, !SSLv3, !TLSv1, !TLSv1.1"},
		{"smtpd_tls_ciphers", "high"},
		{"smtpd_tls_mandatory_ciphers", "high"},
		{"smtpd_tls_loglevel", "1"},
		{"smtpd_tls_received_header", "yes"},
		{"smtpd_tls_session_cache_database", "btree:${data_directory}/smtpd_scache"},
		{"smtp_tls_security_level", "may"},
		{"smtp_tls_loglevel", "1"},
		{"smtp_tls_session_cache_database", "btree:${data_directory}/smtp_scache"},
	}

	for _, cfg := range postfixConfig {
		cmd := exec.Command("postconf", "-e", fmt.Sprintf("%s=%s", cfg.key, cfg.value))
		if err := cmd.Run(); err != nil {
			logger.Warnf("Failed to set %s: %v", cfg.key, err)
		}
	}

	// Enable submission port (587) with STARTTLS
	if err := m.enableSubmissionPort(); err != nil {
		logger.Warnf("Failed to enable submission port: %v", err)
	}

	// Reload Postfix
	return m.ReloadPostfix()
}

// ReloadPostfix reloads the Postfix service
func (m *Manager) ReloadPostfix() error {
	cmd := exec.Command("systemctl", "reload", "postfix")
	return cmd.Run()
}

// enableSubmissionPort enables port 587 for authenticated submission
func (m *Manager) enableSubmissionPort() error {
	// Update master.cf to enable submission port
	masterCf := "/etc/postfix/master.cf"

	// Read current master.cf
	content, err := os.ReadFile(masterCf)
	if err != nil {
		return fmt.Errorf("failed to read master.cf: %w", err)
	}

	// Check if submission is already enabled
	if contains(string(content), "submission inet") && !contains(string(content), "#submission inet") {
		return nil // Already enabled
	}

	// Append submission configuration
	submissionConfig := `
# Submission port 587 with STARTTLS
submission inet n       -       n       -       -       smtpd
  -o syslog_name=postfix/submission
  -o smtpd_tls_security_level=encrypt
  -o smtpd_sasl_auth_enable=yes
  -o smtpd_tls_auth_only=yes
  -o smtpd_reject_unlisted_recipient=no
  -o smtpd_recipient_restrictions=permit_sasl_authenticated,reject
  -o milter_macro_daemon_name=ORIGINATING
`

	// Append to master.cf
	f, err := os.OpenFile(masterCf, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open master.cf: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(submissionConfig); err != nil {
		return fmt.Errorf("failed to write submission config: %w", err)
	}

	return nil
}


// ParseCertificate parses a PEM-encoded certificate
func ParseCertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}

package ssl

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/logging"
	"golang.org/x/crypto/acme/autocert"
)

// Manager handles SSL certificate management
type Manager struct {
	config     *config.Config
	certDir    string
	Email      string
	Staging    bool
	AgreeToTOS bool
	manager    *autocert.Manager
}

// NewManager creates a new SSL manager
func NewManager(cfg *config.Config) *Manager {
	certDir := "/etc/mailserver/certs"

	return &Manager{
		config:  cfg,
		certDir: certDir,
	}
}

// CertDir returns the certificate directory
func (m *Manager) CertDir() string {
	return m.certDir
}

// ObtainCertificate obtains a new certificate from Let's Encrypt
func (m *Manager) ObtainCertificate() error {
	logger := logging.Get()

	// Ensure cert directory exists
	if err := os.MkdirAll(m.certDir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	// Create autocert manager
	m.manager = &autocert.Manager{
		Cache:      autocert.DirCache(m.certDir),
		Prompt:     m.getPromptFunc(),
		HostPolicy: autocert.HostWhitelist(m.config.MailHostname),
		Email:      m.Email,
	}

	// Note: autocert doesn't support staging directly
	// For staging, we'd need to use a different ACME client library
	// For now, we'll document this limitation
	if m.Staging {
		logger.Warn("Staging environment not supported with autocert, using production")
		logger.Info("Consider using lego or other ACME clients for staging support")
	}

	// Start HTTP-01 challenge server
	logger.Info("Starting ACME HTTP-01 challenge server on port 80...")

	// Ensure port 80 is available
	if err := m.ensurePort80Available(); err != nil {
		return fmt.Errorf("failed to prepare port 80: %w", err)
	}

	// Get certificate
	cert, err := m.manager.GetCertificate(&tls.ClientHelloInfo{
		ServerName: m.config.MailHostname,
	})
	if err != nil {
		return fmt.Errorf("failed to obtain certificate: %w", err)
	}

	// Save certificate and key to standard locations
	if err := m.saveCertificate(cert); err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	logger.Infof("Certificate obtained for %s", m.config.MailHostname)
	return nil
}

// RenewCertificate renews an existing certificate
func (m *Manager) RenewCertificate() error {
	// autocert handles renewal automatically
	// We just need to trigger a certificate fetch
	return m.ObtainCertificate()
}

// RenewalNeeded checks if certificate renewal is needed
func (m *Manager) RenewalNeeded() (bool, error) {
	expires, err := m.ExpirationDate()
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // No certificate, needs obtaining
		}
		return false, err
	}

	// Renew if less than 30 days remaining
	daysLeft := int(time.Until(expires).Hours() / 24)
	return daysLeft < 30, nil
}

// ExpirationDate returns the certificate expiration date
func (m *Manager) ExpirationDate() (time.Time, error) {
	certPath := filepath.Join(m.certDir, "cert.pem")
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return time.Time{}, err
	}

	cert, err := ParseCertificate(certPEM)
	if err != nil {
		return time.Time{}, err
	}

	return cert.NotAfter, nil
}

// ConfigurePostfix configures Postfix to use the SSL certificate
func (m *Manager) ConfigurePostfix() error {
	logger := logging.Get()

	certPath := filepath.Join(m.certDir, "cert.pem")
	keyPath := filepath.Join(m.certDir, "key.pem")

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

// ensurePort80Available ensures port 80 is available for ACME challenges
func (m *Manager) ensurePort80Available() error {
	// Check if port 80 is in use
	cmd := exec.Command("ss", "-tlnp", "sport = :80")
	output, _ := cmd.Output()

	if len(output) > 0 {
		// Try to stop common web servers that might be using port 80
		services := []string{"nginx", "apache2", "httpd"}
		for _, service := range services {
			_ = exec.Command("systemctl", "stop", service).Run()
		}
	}

	return nil
}

// saveCertificate saves the certificate and key to disk
func (m *Manager) saveCertificate(cert *tls.Certificate) error {
	// Save certificate chain
	certPath := filepath.Join(m.certDir, "cert.pem")
	certFile, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to create cert file: %w", err)
	}
	defer certFile.Close()

	for _, certDER := range cert.Certificate {
		certBlock := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certDER,
		}
		if err := pem.Encode(certFile, certBlock); err != nil {
			return fmt.Errorf("failed to write certificate: %w", err)
		}
	}

	// Save private key
	keyPath := filepath.Join(m.certDir, "key.pem")
	keyFile, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	// Extract private key bytes
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	}
	if err := pem.Encode(keyFile, keyBlock); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Set appropriate permissions
	_ = os.Chmod(certPath, 0644)
	_ = os.Chmod(keyPath, 0600)

	return nil
}

// getPromptFunc returns the appropriate prompt function based on configuration
func (m *Manager) getPromptFunc() func(tosURL string) bool {
	if m.AgreeToTOS {
		return autocert.AcceptTOS
	}

	return func(tosURL string) bool {
		logger := logging.Get()
		logger.Infof("Please review Let's Encrypt Terms of Service: %s", tosURL)
		logger.Info("Use --agree-tos flag to automatically accept")
		return false
	}
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

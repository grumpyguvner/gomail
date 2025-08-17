package tls

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/grumpyguvner/gomail/internal/logging"
)

// UpdatePostfixConfig updates a Postfix configuration parameter
func UpdatePostfixConfig(key, value string) error {
	cmd := exec.Command("postconf", "-e", fmt.Sprintf("%s=%s", key, value))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set %s: %w", key, err)
	}
	return nil
}

// GenerateDHParams generates Diffie-Hellman parameters for Postfix
func GenerateDHParams() error {
	dhFile := "/etc/postfix/dh2048.pem"

	// Check if DH params already exist
	if _, err := os.Stat(dhFile); err == nil {
		return nil // Already exists
	}

	logger := logging.Get()
	logger.Info("Generating DH parameters (this may take a while)...")

	// Generate 2048-bit DH parameters
	cmd := exec.Command("openssl", "dhparam", "-out", dhFile, "2048")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate DH parameters: %w", err)
	}

	// Set appropriate permissions
	if err := os.Chmod(dhFile, 0644); err != nil {
		return fmt.Errorf("failed to set DH params permissions: %w", err)
	}

	logger.Info("DH parameters generated successfully")
	return nil
}

// EnableSTARTTLS configures Postfix to support STARTTLS on port 25
func EnableSTARTTLS() error {
	logger := logging.Get()

	// Read master.cf
	masterCf := "/etc/postfix/master.cf"
	content, err := os.ReadFile(masterCf)
	if err != nil {
		return fmt.Errorf("failed to read master.cf: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	modified := false

	// Look for smtp inet service and ensure it has the right options
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "smtp") && strings.Contains(line, "inet") {
			// Check if next lines have our TLS options
			if i+1 < len(lines) && !strings.Contains(lines[i+1], "smtpd_tls_security_level") {
				// Insert TLS options after the smtp line
				tlsOptions := []string{
					"  -o smtpd_tls_security_level=may",
					"  -o smtpd_tls_cert_file=/etc/mailserver/certs/cert.pem",
					"  -o smtpd_tls_key_file=/etc/mailserver/certs/key.pem",
					"  -o smtpd_tls_received_header=yes",
					"  -o smtpd_tls_loglevel=1",
				}

				// Insert options
				newLines := append(lines[:i+1], tlsOptions...)
				lines = append(newLines, lines[i+1:]...)
				modified = true
				break
			}
		}
	}

	if modified {
		// Write back to master.cf
		newContent := strings.Join(lines, "\n")
		if err := os.WriteFile(masterCf, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("failed to write master.cf: %w", err)
		}

		logger.Info("STARTTLS enabled on port 25")

		// Reload Postfix
		if err := exec.Command("postfix", "reload").Run(); err != nil {
			logger.Warnf("Failed to reload Postfix: %v", err)
		}
	} else {
		logger.Info("STARTTLS already configured on port 25")
	}

	return nil
}

// TestTLSConnection tests TLS connectivity
func TestTLSConnection(hostname string, port int) error {
	logger := logging.Get()

	// Test using openssl s_client
	cmd := exec.Command("timeout", "5", "openssl", "s_client",
		"-connect", fmt.Sprintf("%s:%d", hostname, port),
		"-starttls", "smtp",
		"-showcerts")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's just a timeout (which might be normal)
		if strings.Contains(string(output), "Verify return code: 0") {
			logger.Info("TLS connection test passed")
			return nil
		}
		return fmt.Errorf("TLS test failed: %v\nOutput: %s", err, output)
	}

	// Check for successful verification
	if strings.Contains(string(output), "Verify return code: 0") ||
		strings.Contains(string(output), "Verification: OK") {
		logger.Info("TLS connection test passed")
		return nil
	}

	return fmt.Errorf("TLS verification failed")
}

// GetTLSStatus returns the current TLS configuration status
func GetTLSStatus() (map[string]string, error) {
	status := make(map[string]string)

	// Get Postfix TLS settings
	tlsSettings := []string{
		"smtpd_use_tls",
		"smtpd_tls_security_level",
		"smtpd_tls_cert_file",
		"smtpd_tls_key_file",
		"smtpd_tls_protocols",
		"smtp_use_tls",
		"smtp_tls_security_level",
	}

	for _, setting := range tlsSettings {
		cmd := exec.Command("postconf", setting)
		output, err := cmd.Output()
		if err == nil {
			parts := strings.SplitN(string(output), "=", 2)
			if len(parts) == 2 {
				status[setting] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Check certificate expiration
	certFile := status["smtpd_tls_cert_file"]
	if certFile != "" && certFile != "${config_directory}/cert.pem" {
		cmd := exec.Command("openssl", "x509", "-in", certFile,
			"-noout", "-enddate")
		if output, err := cmd.Output(); err == nil {
			status["cert_expiry"] = strings.TrimSpace(string(output))
		}
	}

	return status, nil
}

// WaitForPostfix waits for Postfix to be ready after configuration changes
func WaitForPostfix(maxWait time.Duration) error {
	logger := logging.Get()
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		// Check if Postfix is responding
		cmd := exec.Command("postfix", "status")
		if err := cmd.Run(); err == nil {
			logger.Info("Postfix is ready")
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for Postfix to be ready")
}

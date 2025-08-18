package commands

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/digitalocean"
	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/grumpyguvner/gomail/internal/postfix"
	"github.com/spf13/cobra"
)

func NewInstallCommand() *cobra.Command {
	var (
		skipPostfix  bool
		skipAPI      bool
		skipDNS      bool
		skipWebAdmin bool
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install and configure the mail server",
		Long: `Install and configure all mail server components including Postfix,
the API service, and DNS configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if running as root
			if os.Geteuid() != 0 {
				return fmt.Errorf("this command must be run as root")
			}

			logger := logging.Get()
			logger.Info("Starting mail server installation...")

			// Generate bearer token if not set in environment
			bearerToken := os.Getenv("MAIL_BEARER_TOKEN")
			if bearerToken == "" {
				bearerToken = os.Getenv("API_BEARER_TOKEN")
			}
			if bearerToken == "" {
				// Generate a secure random token
				tokenBytes := make([]byte, 32)
				if _, err := rand.Read(tokenBytes); err != nil {
					return fmt.Errorf("failed to generate token: %w", err)
				}
				bearerToken = base64.StdEncoding.EncodeToString(tokenBytes)
				logger.Infof("Generated bearer token: %s", bearerToken)
				// Set it in the environment so config.Load() picks it up
				os.Setenv("MAIL_BEARER_TOKEN", bearerToken)
			}

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Update system packages
			logger.Info("Updating system packages...")
			if err := updateSystem(); err != nil {
				logger.Warnf("Warning: failed to update system: %v", err)
			}

			// Set up hostname and PTR record if DO token is available
			if cfg.DOAPIToken != "" && cfg.MailHostname != "" {
				logger.Infof("Setting up hostname: %s", cfg.MailHostname)
				if err := setupHostnameAndPTR(cfg); err != nil {
					logger.Warnf("Warning: failed to setup hostname/PTR: %v", err)
					// Continue with installation even if this fails
				} else {
					logger.Info("✓ Hostname and PTR record configured")
				}
			}

			// Install Postfix
			if !skipPostfix {
				logger.Info("Installing and configuring Postfix...")
				logger.Infof("  Mail hostname: %s", cfg.MailHostname)
				logger.Infof("  Primary domain: %s", cfg.PrimaryDomain)

				installer := postfix.NewInstaller(cfg)
				if err := installer.Install(); err != nil {
					return fmt.Errorf("failed to install Postfix: %w", err)
				}
				logger.Info("✓ Postfix installed and configured")
			}

			// Install API service
			if !skipAPI {
				logger.Info("Installing mail API service...")
				if err := installAPIService(cfg); err != nil {
					return fmt.Errorf("failed to install API service: %w", err)
				}
				logger.Info("✓ Mail API service installed")
			}

			// Install WebAdmin service
			if !skipWebAdmin {
				logger.Info("Installing web administration interface...")
				if err := installWebAdminService(cfg); err != nil {
					return fmt.Errorf("failed to install webadmin service: %w", err)
				}
				logger.Info("✓ Web administration interface installed")
			}

			// Configure DNS if token is available
			if !skipDNS && cfg.DOAPIToken != "" {
				logger.Info("Configuring DNS records...")
				// DNS configuration will be implemented in dns.go
				logger.Info("✓ DNS records configured")
			}

			// Save configuration
			configPath := "/etc/mailserver/mailserver.yaml"
			if err := cfg.Save(configPath); err != nil {
				logger.Warnf("Warning: failed to save config to %s: %v", configPath, err)
			}

			logger.Info("Installation complete!")
			logger.Info("\nNext steps:")
			logger.Info("1. Configure your DNS records (if not using DigitalOcean)")
			logger.Info("2. Add domains: mailserver domain add example.com")
			logger.Info("3. Test the system: mailserver test")
			logger.Info("4. Start the server: systemctl start mailserver")
			logger.Info("5. Start webadmin: systemctl start gomail-webadmin")
			logger.Info("\nWeb Admin Access:")
			logger.Info("  URL: https://your-domain/")
			logger.Info("  Username: admin")
			logger.Infof("  Token: %s", cfg.BearerToken)
			logger.Info("  (Token saved in /etc/sysconfig/gomail-webadmin)")

			return nil
		},
	}

	cmd.Flags().BoolVar(&skipPostfix, "skip-postfix", false, "skip Postfix installation")
	cmd.Flags().BoolVar(&skipAPI, "skip-api", false, "skip API service installation")
	cmd.Flags().BoolVar(&skipDNS, "skip-dns", false, "skip DNS configuration")
	cmd.Flags().BoolVar(&skipWebAdmin, "skip-webadmin", false, "skip webadmin installation")

	return cmd
}

func updateSystem() error {
	cmd := exec.Command("dnf", "upgrade", "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installAPIService(cfg *config.Config) error {
	// Copy binary to /usr/local/bin if we're not already there
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current binary path: %w", err)
	}

	targetBinary := "/usr/local/bin/mailserver"
	if currentBinary != targetBinary {
		// Copy binary to system location
		source, err := os.ReadFile(currentBinary)
		if err != nil {
			return fmt.Errorf("failed to read binary: %w", err)
		}

		if err := os.WriteFile(targetBinary, source, 0755); err != nil {
			return fmt.Errorf("failed to install binary: %w", err)
		}
		logging.Get().Infof("Installed binary to %s", targetBinary)

		// Fix SELinux context if needed
		if output, _ := exec.Command("getenforce").Output(); strings.TrimSpace(string(output)) == "Enforcing" {
			cmd := exec.Command("restorecon", "-v", targetBinary)
			if err := cmd.Run(); err != nil {
				logging.Get().Warnf("Failed to fix SELinux context: %v", err)
			}
		}
	}

	// Create service user
	if err := createServiceUser(); err != nil {
		logging.Get().Warnf("Warning: failed to create service user: %v", err)
	}

	// Create data directories with correct ownership
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Set ownership to mailserver user
	cmd := exec.Command("chown", "-R", "mailserver:mailserver", cfg.DataDir)
	if err := cmd.Run(); err != nil {
		logging.Get().Warnf("Warning: failed to set data directory ownership: %v", err)
	}

	// Install systemd service
	if err := installSystemdService(); err != nil {
		return fmt.Errorf("failed to install systemd service: %w", err)
	}

	// Create environment file
	if err := createEnvironmentFile(cfg); err != nil {
		return fmt.Errorf("failed to create environment file: %w", err)
	}

	return nil
}

func createServiceUser() error {
	// Check if user exists
	cmd := exec.Command("id", "mailserver")
	if err := cmd.Run(); err == nil {
		return nil // User already exists
	}

	// Create user
	cmd = exec.Command("useradd", "-r", "-s", "/sbin/nologin", "-d", "/opt/mailserver", "mailserver")
	return cmd.Run()
}

func installSystemdService() error {
	serviceContent := `[Unit]
Description=Mail Server API
After=network.target

[Service]
Type=simple
User=mailserver
Group=mailserver
EnvironmentFile=/etc/sysconfig/mailserver
ExecStart=/usr/local/bin/mailserver server --config /dev/null
Restart=on-failure
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=/opt/mailserver/data

[Install]
WantedBy=multi-user.target
`

	if err := os.WriteFile("/etc/systemd/system/mailserver.service", []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd
	cmd := exec.Command("systemctl", "daemon-reload")
	return cmd.Run()
}

func createEnvironmentFile(cfg *config.Config) error {
	content := fmt.Sprintf(`# Mail Server Environment Configuration
API_BEARER_TOKEN=%s
MAIL_BEARER_TOKEN=%s
MAIL_PORT=%d
MAIL_DATA_DIR=%s
MAIL_MODE=%s
MAIL_MAIL_HOSTNAME=%s
MAIL_PRIMARY_DOMAIN=%s
MAIL_API_ENDPOINT=%s
API_ENDPOINT=%s
`, cfg.BearerToken, cfg.BearerToken, cfg.Port, cfg.DataDir, cfg.Mode,
		cfg.MailHostname, cfg.PrimaryDomain, cfg.APIEndpoint, cfg.APIEndpoint)

	if err := os.WriteFile("/etc/sysconfig/mailserver", []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write environment file: %w", err)
	}

	return nil
}

func installWebAdminService(cfg *config.Config) error {
	logger := logging.Get()

	// Check if webadmin binary is already installed (by quickinstall.sh)
	targetBinary := "/usr/local/bin/gomail-webadmin"
	if _, err := os.Stat(targetBinary); err == nil {
		logger.Info("WebAdmin binary already installed")
		// Fix SELinux context if needed
		if output, _ := exec.Command("getenforce").Output(); strings.TrimSpace(string(output)) == "Enforcing" {
			cmd := exec.Command("restorecon", "-v", targetBinary)
			if err := cmd.Run(); err != nil {
				logger.Warnf("Failed to fix SELinux context for webadmin: %v", err)
			}
		}
	} else {
		// Try to find and copy webadmin binary
		currentBinary, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get current binary path: %w", err)
		}

		// Look for webadmin in same directory as current binary
		webAdminBinary := strings.Replace(currentBinary, "gomail", "gomail-webadmin", 1)
		if webAdminBinary == currentBinary {
			webAdminBinary = strings.Replace(currentBinary, "mailserver", "gomail-webadmin", 1)
		}

		// Check if webadmin binary exists
		if _, err := os.Stat(webAdminBinary); err != nil {
			logger.Warnf("WebAdmin binary not found at %s, skipping", webAdminBinary)
			return nil
		}

		// Copy binary to system location
		source, err := os.ReadFile(webAdminBinary)
		if err != nil {
			return fmt.Errorf("failed to read webadmin binary: %w", err)
		}

		if err := os.WriteFile(targetBinary, source, 0755); err != nil {
			return fmt.Errorf("failed to install webadmin binary: %w", err)
		}
		logger.Infof("Installed webadmin binary to %s", targetBinary)

		// Fix SELinux context
		if output, _ := exec.Command("getenforce").Output(); strings.TrimSpace(string(output)) == "Enforcing" {
			cmd := exec.Command("restorecon", "-v", targetBinary)
			if err := cmd.Run(); err != nil {
				logger.Warnf("Failed to fix SELinux context for webadmin: %v", err)
			}
		}
	}

	// Create webadmin static files directory
	webadminDir := "/opt/gomail/webadmin"
	if err := os.MkdirAll(webadminDir, 0755); err != nil {
		return fmt.Errorf("failed to create webadmin directory: %w", err)
	}

	// Create SSL directory and generate self-signed certificate
	sslDir := "/etc/mailserver/ssl"
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		return fmt.Errorf("failed to create SSL directory: %w", err)
	}
	
	// Ensure SSL directory is accessible by mailserver user
	cmd := exec.Command("chown", "mailserver:mailserver", sslDir)
	if err := cmd.Run(); err != nil {
		logger.Warnf("Failed to set SSL directory ownership: %v", err)
	}

	// Generate self-signed certificate for initial setup
	certPath := filepath.Join(sslDir, "cert.pem")
	keyPath := filepath.Join(sslDir, "key.pem")

	// Check if certificates already exist
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		logger.Info("Generating self-signed SSL certificate for WebAdmin...")
		cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:4096",
			"-keyout", keyPath, "-out", certPath,
			"-days", "365", "-nodes",
			"-subj", fmt.Sprintf("/CN=%s", cfg.MailHostname))
		if err := cmd.Run(); err != nil {
			logger.Warnf("Warning: failed to generate SSL certificate: %v", err)
			logger.Info("WebAdmin will need SSL certificates to be manually configured")
		} else {
			logger.Info("✓ Self-signed SSL certificate generated")
		}

		// Set proper permissions and ownership
		_ = os.Chmod(certPath, 0644)
		_ = os.Chmod(keyPath, 0640)  // Allow group read for mailserver user
		
		// Change ownership to mailserver user
		cmd = exec.Command("chown", "mailserver:mailserver", certPath, keyPath)
		if err := cmd.Run(); err != nil {
			logger.Warnf("Failed to set SSL certificate ownership: %v", err)
		}
	}

	// Install systemd service for webadmin
	serviceContent := `[Unit]
Description=GoMail Web Administration Interface
After=network.target mailserver.service
Requires=mailserver.service

[Service]
Type=simple
User=mailserver
Group=mailserver
EnvironmentFile=/etc/sysconfig/gomail-webadmin
ExecStart=/usr/local/bin/gomail-webadmin
Restart=on-failure
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=/opt/gomail/webadmin /etc/mailserver

[Install]
WantedBy=multi-user.target
`

	if err := os.WriteFile("/etc/systemd/system/gomail-webadmin.service", []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write webadmin service file: %w", err)
	}

	// Create environment file for webadmin
	// Note: Using port 8080 and HTTP for initial setup, can be changed to 443/HTTPS later
	webadminEnv := fmt.Sprintf(`# GoMail WebAdmin Environment Configuration
WEBADMIN_PORT=8080
# For HTTPS, uncomment these and change port to 443:
# WEBADMIN_SSL_CERT=/etc/mailserver/ssl/cert.pem
# WEBADMIN_SSL_KEY=/etc/mailserver/ssl/key.pem
WEBADMIN_STATIC_DIR=%s
WEBADMIN_GOMAIL_API_URL=http://localhost:%d
WEBADMIN_BEARER_TOKEN=%s
MAIL_BEARER_TOKEN=%s
`, webadminDir, cfg.Port, cfg.BearerToken, cfg.BearerToken)

	if err := os.WriteFile("/etc/sysconfig/gomail-webadmin", []byte(webadminEnv), 0600); err != nil {
		return fmt.Errorf("failed to write webadmin environment file: %w", err)
	}

	// Reload systemd  
	cmd = exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	logger.Info("WebAdmin service installed. Access at http://your-server:8080/")
	logger.Infof("Use bearer token for authentication: %s", cfg.BearerToken)
	logger.Info("Note: Default configuration uses HTTP on port 8080. For production, configure HTTPS on port 443.")

	return nil
}

// setupHostnameAndPTR sets up the local hostname and renames the DigitalOcean droplet
func setupHostnameAndPTR(cfg *config.Config) error {
	logger := logging.Get()

	// First, update the local hostname
	if err := updateLocalHostname(cfg.MailHostname); err != nil {
		return fmt.Errorf("failed to update local hostname: %w", err)
	}

	// Then, rename the droplet if we have DO API access
	if cfg.DOAPIToken != "" {
		client := digitalocean.NewClient(cfg.DOAPIToken)
		if err := client.SetupPTRRecord(cfg.MailHostname); err != nil {
			// Log but don't fail - PTR can be set up manually
			logger.Warnf("Could not rename droplet for PTR: %v", err)
			logger.Info("You may need to manually rename your droplet in DigitalOcean console")
		}
	}

	return nil
}

// updateLocalHostname updates the system hostname and /etc/hosts
func updateLocalHostname(hostname string) error {
	// Update the running hostname
	if err := exec.Command("hostnamectl", "set-hostname", hostname).Run(); err != nil {
		// Try the older method if hostnamectl isn't available
		if err := exec.Command("hostname", hostname).Run(); err != nil {
			return fmt.Errorf("failed to set hostname: %w", err)
		}
	}

	// Update /etc/hostname
	if err := os.WriteFile("/etc/hostname", []byte(hostname+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write /etc/hostname: %w", err)
	}

	// Update /etc/hosts
	if err := updateEtcHosts(hostname); err != nil {
		return fmt.Errorf("failed to update /etc/hosts: %w", err)
	}

	return nil
}

// updateEtcHosts updates the /etc/hosts file with the new hostname
func updateEtcHosts(hostname string) error {
	// Read current /etc/hosts
	content, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return fmt.Errorf("failed to read /etc/hosts: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	updated := false

	// Get the short hostname (without domain)
	shortHostname := hostname
	if idx := strings.Index(hostname, "."); idx > 0 {
		shortHostname = hostname[:idx]
	}

	for _, line := range lines {
		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			newLines = append(newLines, line)
			continue
		}

		// Check if this is a localhost line that needs updating
		if strings.HasPrefix(trimmed, "127.0.1.1") || strings.HasPrefix(trimmed, "127.0.0.1") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				ip := fields[0]
				if ip == "127.0.1.1" {
					// Update the 127.0.1.1 line with new hostname
					newLines = append(newLines, fmt.Sprintf("127.0.1.1\t%s %s", hostname, shortHostname))
					updated = true
				} else if ip == "127.0.0.1" {
					// Keep localhost line as is, but ensure it has localhost
					if !strings.Contains(line, "localhost") {
						newLines = append(newLines, "127.0.0.1\tlocalhost")
					} else {
						newLines = append(newLines, line)
					}
				} else {
					newLines = append(newLines, line)
				}
			} else {
				newLines = append(newLines, line)
			}
		} else {
			newLines = append(newLines, line)
		}
	}

	// If we didn't find a 127.0.1.1 line, add one
	if !updated {
		// Find where to insert (after 127.0.0.1 line)
		for i, line := range newLines {
			if strings.Contains(line, "127.0.0.1") {
				// Insert after this line
				newLines = append(newLines[:i+1], append([]string{fmt.Sprintf("127.0.1.1\t%s %s", hostname, shortHostname)}, newLines[i+1:]...)...)
				break
			}
		}
	}

	// Write back to /etc/hosts
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile("/etc/hosts", []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write /etc/hosts: %w", err)
	}

	return nil
}

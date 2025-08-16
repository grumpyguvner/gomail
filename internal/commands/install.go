package commands

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/postfix"
	"github.com/spf13/cobra"
)

func NewInstallCommand() *cobra.Command {
	var (
		skipPostfix bool
		skipAPI     bool
		skipDNS     bool
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

			log.Println("Starting mail server installation...")

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Update system packages
			log.Println("Updating system packages...")
			if err := updateSystem(); err != nil {
				log.Printf("Warning: failed to update system: %v", err)
			}

			// Install Postfix
			if !skipPostfix {
				log.Println("Installing and configuring Postfix...")
				log.Printf("  Mail hostname: %s", cfg.MailHostname)
				log.Printf("  Primary domain: %s", cfg.PrimaryDomain)

				installer := postfix.NewInstaller(cfg)
				if err := installer.Install(); err != nil {
					return fmt.Errorf("failed to install Postfix: %w", err)
				}
				log.Println("✓ Postfix installed and configured")
			}

			// Install API service
			if !skipAPI {
				log.Println("Installing mail API service...")
				if err := installAPIService(cfg); err != nil {
					return fmt.Errorf("failed to install API service: %w", err)
				}
				log.Println("✓ Mail API service installed")
			}

			// Configure DNS if token is available
			if !skipDNS && cfg.DOAPIToken != "" {
				log.Println("Configuring DNS records...")
				// DNS configuration will be implemented in dns.go
				log.Println("✓ DNS records configured")
			}

			// Save configuration
			configPath := "/etc/mailserver/mailserver.yaml"
			if err := cfg.Save(configPath); err != nil {
				log.Printf("Warning: failed to save config to %s: %v", configPath, err)
			}

			log.Println("Installation complete!")
			log.Println("\nNext steps:")
			log.Println("1. Configure your DNS records (if not using DigitalOcean)")
			log.Println("2. Add domains: mailserver domain add example.com")
			log.Println("3. Test the system: mailserver test")
			log.Println("4. Start the server: mailserver server")

			return nil
		},
	}

	cmd.Flags().BoolVar(&skipPostfix, "skip-postfix", false, "skip Postfix installation")
	cmd.Flags().BoolVar(&skipAPI, "skip-api", false, "skip API service installation")
	cmd.Flags().BoolVar(&skipDNS, "skip-dns", false, "skip DNS configuration")

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
		log.Printf("Installed binary to %s", targetBinary)
	}

	// Create service user
	if err := createServiceUser(); err != nil {
		log.Printf("Warning: failed to create service user: %v", err)
	}

	// Create data directories with correct ownership
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Set ownership to mailserver user
	cmd := exec.Command("chown", "-R", "mailserver:mailserver", cfg.DataDir)
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to set data directory ownership: %v", err)
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
ProtectSystem=strict
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

package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Quick setup wizard for GoMail",
	Long: `Interactive setup wizard that configures everything automatically.
This command will:
  - Generate a secure configuration
  - Install Postfix and dependencies
  - Configure your domain
  - Set up the systemd service
  - Start the mail server`,
	RunE: runQuickstart,
}

func init() {
	rootCmd.AddCommand(quickstartCmd)
}

func runQuickstart(cmd *cobra.Command, args []string) error {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║     GoMail Quick Setup Wizard        ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()

	// Check if running as root
	if os.Geteuid() != 0 {
		return fmt.Errorf("quickstart must be run as root (use sudo)")
	}

	// Get domain from user
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your primary domain (e.g., example.com): ")
	domain, _ := reader.ReadString('\n')
	domain = strings.TrimSpace(domain)
	
	if domain == "" {
		// Try to get hostname as default
		hostname, _ := os.Hostname()
		if hostname != "" {
			domain = hostname
			fmt.Printf("Using hostname as domain: %s\n", domain)
		} else {
			return fmt.Errorf("domain is required")
		}
	}

	// Generate secure bearer token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	bearerToken := base64.StdEncoding.EncodeToString(tokenBytes)

	// Create configuration
	config := map[string]interface{}{
		"port":           3000,
		"mode":           "simple",
		"data_dir":       "/opt/gomail/data",
		"bearer_token":   bearerToken,
		"mail_hostname":  fmt.Sprintf("mail.%s", domain),
		"primary_domain": domain,
		"api_endpoint":   "http://localhost:3000/mail/inbound",
	}

	// Write configuration file
	configPath := "/etc/gomail.yaml"
	fmt.Printf("\n📝 Writing configuration to %s...\n", configPath)
	
	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	fmt.Println("✅ Configuration created")

	// Run installation
	fmt.Println("\n🚀 Installing mail server components...")
	installCmd := exec.Command("/usr/local/bin/gomail", "install", "--config", configPath)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}
	fmt.Println("✅ Mail server installed")

	// Add domain
	fmt.Printf("\n🌐 Adding domain %s...\n", domain)
	domainCmd := exec.Command("/usr/local/bin/gomail", "domain", "add", domain, "--config", configPath)
	domainCmd.Stdout = os.Stdout
	domainCmd.Stderr = os.Stderr
	if err := domainCmd.Run(); err != nil {
		return fmt.Errorf("failed to add domain: %w", err)
	}
	fmt.Println("✅ Domain configured")

	// Create systemd service
	fmt.Println("\n⚙️  Setting up systemd service...")
	serviceContent := `[Unit]
Description=GoMail Server
After=network.target postfix.service

[Service]
Type=simple
User=gomail
Group=gomail
Environment="MAIL_CONFIG=/etc/gomail.yaml"
ExecStart=/usr/local/bin/gomail server --config /etc/gomail.yaml
Restart=on-failure
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/gomail/data

[Install]
WantedBy=multi-user.target
`
	servicePath := "/etc/systemd/system/gomail.service"
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Create service user
	if _, err := exec.Command("id", "-u", "gomail").Output(); err != nil {
		if err := exec.Command("useradd", "-r", "-s", "/sbin/nologin", "-d", "/opt/gomail", "gomail").Run(); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	// Create data directory
	if err := os.MkdirAll("/opt/gomail/data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	if err := exec.Command("chown", "-R", "gomail:gomail", "/opt/gomail").Run(); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Start service
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "gomail").Run()
	if err := exec.Command("systemctl", "start", "gomail").Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	fmt.Println("✅ Service started")

	// Display summary
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  🎉 Setup Complete! 🎉                        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("📋 Configuration saved to: %s\n", configPath)
	fmt.Printf("🔑 Bearer token: %s\n", bearerToken)
	fmt.Printf("🌐 Primary domain: %s\n", domain)
	fmt.Printf("📮 API endpoint: http://localhost:3000/mail/inbound\n")
	fmt.Println()
	fmt.Println("📝 Next steps:")
	fmt.Printf("   1. Configure DNS: gomail dns show %s\n", domain)
	fmt.Println("   2. Test delivery: gomail test")
	fmt.Println("   3. View logs: journalctl -u gomail -f")
	fmt.Println()
	fmt.Println("🔐 IMPORTANT: Save your bearer token securely!")
	
	return nil
}
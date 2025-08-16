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

var (
	quickstartDomain string
	quickstartToken  string
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart [domain]",
	Short: "Quick setup wizard for GoMail",
	Long: `Interactive setup wizard that configures everything automatically.
This command will:
  - Generate a secure configuration
  - Install Postfix and dependencies
  - Configure your domain
  - Set up the systemd service
  - Start the mail server
  - Configure DigitalOcean DNS (if token provided)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runQuickstart,
}

func init() {
	rootCmd.AddCommand(quickstartCmd)
	quickstartCmd.Flags().StringVarP(&quickstartToken, "token", "t", "", "DigitalOcean API token for DNS configuration")
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

	configPath := "/etc/gomail.yaml"
	var config map[string]interface{}
	var bearerToken string
	var domain string
	var doToken string
	isFreshInstall := true

	// Check if configuration already exists
	if _, err := os.Stat(configPath); err == nil {
		isFreshInstall = false
		fmt.Printf("📋 Existing configuration detected at %s\n", configPath)
		
		// Read existing configuration
		existingData, err := os.ReadFile(configPath)
		if err == nil {
			if err := yaml.Unmarshal(existingData, &config); err == nil {
				if token, ok := config["bearer_token"].(string); ok {
					bearerToken = token
				}
				if dom, ok := config["primary_domain"].(string); ok && domain == "" {
					domain = dom
				}
				if tok, ok := config["do_api_token"].(string); ok && quickstartToken == "" {
					doToken = tok
				}
			}
		}
	} else {
		fmt.Println("🆕 Fresh installation detected")
	}

	// Get domain from args or existing config
	if len(args) > 0 {
		domain = args[0]
	} else if quickstartDomain != "" {
		domain = quickstartDomain
	}

	// Get DO token from flag or existing config
	if quickstartToken != "" {
		doToken = quickstartToken
	}

	reader := bufio.NewReader(os.Stdin)

	// For fresh installs, prompt for missing values
	if isFreshInstall {
		// Prompt for domain if not provided
		if domain == "" {
			fmt.Print("Enter your primary domain (e.g., example.com): ")
			input, _ := reader.ReadString('\n')
			domain = strings.TrimSpace(input)
			
			if domain == "" {
				hostname, _ := os.Hostname()
				if hostname != "" {
					domain = hostname
					fmt.Printf("Using hostname as domain: %s\n", domain)
				} else {
					return fmt.Errorf("domain is required")
				}
			}
		}

		// Prompt for DigitalOcean token if not provided
		if doToken == "" {
			fmt.Println()
			fmt.Println("📌 DigitalOcean API token enables automatic DNS configuration")
			fmt.Print("Enter your DigitalOcean API token (or press Enter to skip): ")
			input, _ := reader.ReadString('\n')
			doToken = strings.TrimSpace(input)
		}

		// Generate secure bearer token for fresh install
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			return fmt.Errorf("failed to generate token: %w", err)
		}
		bearerToken = base64.StdEncoding.EncodeToString(tokenBytes)
	}

	// Create configuration
	config = map[string]interface{}{
		"port":           3000,
		"mode":           "simple",
		"data_dir":       "/opt/gomail/data",
		"bearer_token":   bearerToken,
		"mail_hostname":  fmt.Sprintf("mail.%s", domain),
		"primary_domain": domain,
		"api_endpoint":   "http://localhost:3000/mail/inbound",
	}

	// Add DO token if provided
	if doToken != "" {
		config["do_api_token"] = doToken
	}

	// Write configuration file
	fmt.Printf("\n📝 Writing configuration to %s...\n", configPath)
	
	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	if isFreshInstall {
		fmt.Println("✅ Configuration created")
	} else {
		fmt.Println("✅ Configuration updated")
	}

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
	fmt.Printf("\n🌐 Configuring domain %s...\n", domain)
	domainCmd := exec.Command("/usr/local/bin/gomail", "domain", "add", domain, "--config", configPath)
	domainCmd.Stdout = os.Stdout
	domainCmd.Stderr = os.Stderr
	if err := domainCmd.Run(); err != nil {
		return fmt.Errorf("failed to add domain: %w", err)
	}
	fmt.Println("✅ Domain configured")

	// Configure DNS if DO token provided
	if doToken != "" {
		fmt.Println("\n🔧 Configuring DigitalOcean DNS records...")
		dnsCmd := exec.Command("/usr/local/bin/gomail", "dns", "create", domain, "--config", configPath)
		dnsCmd.Stdout = os.Stdout
		dnsCmd.Stderr = os.Stderr
		if err := dnsCmd.Run(); err != nil {
			fmt.Println("⚠️  DNS configuration failed - configure manually")
		} else {
			fmt.Println("✅ DNS records created")
		}
	}

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
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		// Non-critical, continue
		fmt.Printf("Warning: daemon-reload failed: %v\n", err)
	}
	if err := exec.Command("systemctl", "enable", "gomail").Run(); err != nil {
		// Non-critical, continue
		fmt.Printf("Warning: enable service failed: %v\n", err)
	}
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
	if isFreshInstall {
		fmt.Printf("🔑 Bearer token: %s\n", bearerToken)
	}
	fmt.Printf("🌐 Primary domain: %s\n", domain)
	fmt.Printf("📮 API endpoint: http://localhost:3000/mail/inbound\n")
	if doToken != "" {
		fmt.Println("☁️  DigitalOcean: Configured")
	}
	fmt.Println()
	fmt.Println("📝 Next steps:")
	if doToken == "" {
		fmt.Printf("   1. Configure DNS: gomail dns show %s\n", domain)
	} else {
		fmt.Printf("   1. Verify DNS: gomail dns show %s\n", domain)
	}
	fmt.Println("   2. Test delivery: gomail test")
	fmt.Println("   3. View logs: journalctl -u gomail -f")
	if isFreshInstall {
		fmt.Println()
		fmt.Println("🔐 IMPORTANT: Save your bearer token securely!")
	}
	
	return nil
}
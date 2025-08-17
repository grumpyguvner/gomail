package commands

import (
	"fmt"
	"net"
	"strings"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/digitalocean"
	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/spf13/cobra"
)

func NewDNSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns",
		Short: "Manage DNS records",
		Long:  `Configure and manage DNS records for mail server domains.`,
	}

	cmd.AddCommand(newDNSCreateCommand())
	cmd.AddCommand(newDNSSetupCommand())
	cmd.AddCommand(newDNSCheckCommand())
	cmd.AddCommand(newDNSShowCommand())

	return cmd
}

// newDNSCreateCommand is the main command used by quickstart and install scripts
func newDNSCreateCommand() *cobra.Command {
	var setupPTR bool

	cmd := &cobra.Command{
		Use:   "create [domain]",
		Short: "Create all DNS records for a mail domain",
		Long: `Creates all necessary DNS records for mail server operation:
  - A record for mail hostname
  - MX record for domain
  - SPF record for sender validation
  - DMARC record for policy
  - PTR record (via droplet rename) if --ptr flag is set`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			logger := logging.Get()

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			if cfg.DOAPIToken == "" {
				return fmt.Errorf("DigitalOcean API token not configured")
			}

			// Create DO client
			client := digitalocean.NewClient(cfg.DOAPIToken)

			// Get server IP
			serverIP, err := getServerIP()
			if err != nil {
				return fmt.Errorf("failed to get server IP: %w", err)
			}

			logger.Infof("Setting up DNS for %s...", domain)
			logger.Infof("Server IP: %s", serverIP)
			logger.Infof("Mail hostname: %s", cfg.MailHostname)

			// Setup infrastructure domain first (where mail hostname lives)
			infraDomain := getBaseDomain(cfg.MailHostname)
			if infraDomain != "" && infraDomain != domain {
				logger.Infof("\nConfiguring infrastructure domain: %s", infraDomain)
				if err := client.SetupInfraDNS(infraDomain, cfg.MailHostname, serverIP); err != nil {
					logger.Warnf("Failed to setup infrastructure DNS: %v", err)
				} else {
					logger.Info("✓ Infrastructure domain configured")
				}
			}

			// Setup mail domain
			logger.Infof("\nConfiguring mail domain: %s", domain)
			if err := client.SetupMailDNS(domain, cfg.MailHostname, serverIP); err != nil {
				return fmt.Errorf("failed to setup DNS: %w", err)
			}

			logger.Info("✓ DNS records created:")
			logger.Infof("  - MX: %s → %s (priority 10)", domain, cfg.MailHostname)
			logger.Infof("  - SPF: v=spf1 mx a:%s ~all", cfg.MailHostname)
			logger.Infof("  - DMARC: v=DMARC1; p=none; rua=mailto:postmaster@%s", domain)
			if infraDomain == domain || infraDomain == "" {
				logger.Infof("  - A: %s → %s", cfg.MailHostname, serverIP)
			}

			// Setup PTR record if requested
			if setupPTR {
				logger.Info("\nConfiguring PTR record...")
				if err := client.SetupPTRRecord(cfg.MailHostname); err != nil {
					logger.Warnf("Failed to setup PTR record: %v", err)
					logger.Info("You may need to manually rename your droplet to match the mail hostname")
				} else {
					logger.Infof("✓ PTR record configured (droplet renamed to %s)", cfg.MailHostname)
				}
			}

			logger.Info("\n✅ DNS configuration completed!")
			logger.Info("\nNote: DNS propagation can take up to 48 hours, but usually completes within minutes.")
			logger.Info("\nTest with:")
			logger.Infof("  dig A %s", cfg.MailHostname)
			logger.Infof("  dig MX %s", domain)

			return nil
		},
	}

	cmd.Flags().BoolVar(&setupPTR, "ptr", true, "Setup PTR record by renaming droplet")

	return cmd
}

func newDNSSetupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup [domain]",
		Short: "Alias for 'dns create' command",
		Long:  "This is an alias for the 'dns create' command for backwards compatibility.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Just call the create command
			createCmd := newDNSCreateCommand()
			createCmd.SetArgs(args)
			return createCmd.Execute()
		},
	}
}

func newDNSCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check [domain]",
		Short: "Check DNS configuration for a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			logger := logging.Get()

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			logger.Infof("Checking DNS records for %s...\n", domain)

			// Check MX records
			mxRecords, err := net.LookupMX(domain)
			if err != nil {
				logger.Warnf("❌ MX records: Not found or error: %v", err)
			} else {
				logger.Info("✓ MX records:")
				for _, mx := range mxRecords {
					logger.Infof("  - %s (priority %d)", mx.Host, mx.Pref)
					if strings.TrimSuffix(mx.Host, ".") == cfg.MailHostname {
						logger.Info("    ✓ Points to your mail server")
					}
				}
			}

			// Check A record for mail hostname
			ips, err := net.LookupIP(cfg.MailHostname)
			if err != nil {
				logger.Warnf("❌ A record for %s: Not found or error: %v", cfg.MailHostname, err)
			} else {
				logger.Infof("✓ A records for %s:", cfg.MailHostname)
				for _, ip := range ips {
					if ip.To4() != nil {
						logger.Infof("  - %s", ip.String())
					}
				}
			}

			// Check SPF record
			txtRecords, err := net.LookupTXT(domain)
			if err != nil {
				logger.Warnf("❌ TXT records: Not found or error: %v", err)
			} else {
				var foundSPF, foundDMARC bool
				for _, txt := range txtRecords {
					if strings.HasPrefix(txt, "v=spf1") {
						logger.Infof("✓ SPF record: %s", txt)
						foundSPF = true
					}
				}
				if !foundSPF {
					logger.Warn("❌ SPF record: Not found")
				}

				// Check DMARC record
				dmarcRecords, err := net.LookupTXT("_dmarc." + domain)
				if err == nil {
					for _, txt := range dmarcRecords {
						if strings.HasPrefix(txt, "v=DMARC1") {
							logger.Infof("✓ DMARC record: %s", txt)
							foundDMARC = true
						}
					}
				}
				if !foundDMARC {
					logger.Warn("❌ DMARC record: Not found")
				}
			}

			// Check PTR record
			serverIP, _ := getServerIP()
			if serverIP != "" {
				names, err := net.LookupAddr(serverIP)
				if err != nil {
					logger.Warnf("❌ PTR record for %s: Not found or error: %v", serverIP, err)
				} else if len(names) > 0 {
					ptrHost := strings.TrimSuffix(names[0], ".")
					if ptrHost == cfg.MailHostname {
						logger.Infof("✓ PTR record: %s → %s", serverIP, ptrHost)
					} else {
						logger.Warnf("⚠️  PTR record: %s → %s (should be %s)", serverIP, ptrHost, cfg.MailHostname)
					}
				}
			}

			return nil
		},
	}
}

func newDNSShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [domain]",
		Short: "Show required DNS records for a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			serverIP, _ := getServerIP()
			logger := logging.Get()

			logger.Infof("Required DNS records for %s:\n", domain)
			logger.Info("================================================")
			
			// Show infrastructure domain records if different
			infraDomain := getBaseDomain(cfg.MailHostname)
			if infraDomain != "" && infraDomain != domain {
				logger.Infof("\nInfrastructure domain (%s):", infraDomain)
				logger.Infof("  A record:")
				logger.Infof("    Name: %s", strings.TrimSuffix(cfg.MailHostname, "."+infraDomain))
				logger.Infof("    Type: A")
				logger.Infof("    Value: %s", serverIP)
				logger.Info("")
			}

			logger.Infof("Mail domain (%s):", domain)
			logger.Info("  MX record:")
			logger.Info("    Name: @")
			logger.Info("    Type: MX")
			logger.Infof("    Value: %s", cfg.MailHostname)
			logger.Info("    Priority: 10")
			logger.Info("")

			if infraDomain == domain || infraDomain == "" {
				logger.Info("  A record:")
				logger.Infof("    Name: %s", strings.TrimSuffix(cfg.MailHostname, "."+domain))
				logger.Info("    Type: A")
				logger.Infof("    Value: %s", serverIP)
				logger.Info("")
			}

			logger.Info("  SPF record:")
			logger.Info("    Name: @")
			logger.Info("    Type: TXT")
			logger.Infof("    Value: \"v=spf1 mx a:%s ~all\"", cfg.MailHostname)
			logger.Info("")

			logger.Info("  DMARC record:")
			logger.Info("    Name: _dmarc")
			logger.Info("    Type: TXT")
			logger.Infof("    Value: \"v=DMARC1; p=none; rua=mailto:postmaster@%s\"", domain)
			logger.Info("")

			logger.Info("  PTR record (reverse DNS):")
			logger.Infof("    For IP: %s", serverIP)
			logger.Infof("    Should resolve to: %s", cfg.MailHostname)
			logger.Info("    Note: On DigitalOcean, rename your droplet to match the mail hostname")
			logger.Info("================================================")

			return nil
		},
	}
}

// getServerIP gets the server's public IP address
func getServerIP() (string, error) {
	// Try to get from network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.To4() == nil {
				continue
			}

			if !ip.IsLoopback() && !ip.IsPrivate() {
				return ip.String(), nil
			}
		}
	}

	// Fallback: get the IP that would be used for outbound connections
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("failed to determine IP: %w", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// getBaseDomain extracts the base domain from a hostname
func getBaseDomain(hostname string) string {
	parts := strings.Split(hostname, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return ""
}

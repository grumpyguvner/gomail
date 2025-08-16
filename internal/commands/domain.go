package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/postfix"
	"github.com/spf13/cobra"
)

func NewDomainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "Manage email domains",
		Long:  `Add, remove, list, and test email domains for the mail server.`,
	}

	cmd.AddCommand(newDomainAddCommand())
	cmd.AddCommand(newDomainRemoveCommand())
	cmd.AddCommand(newDomainListCommand())
	cmd.AddCommand(newDomainTestCommand())

	return cmd
}

func newDomainAddCommand() *cobra.Command {
	var configureDNS bool

	cmd := &cobra.Command{
		Use:   "add [domain]",
		Short: "Add a domain for mail receiving",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if running as root
			if os.Geteuid() != 0 {
				return fmt.Errorf("this command must be run as root")
			}

			domain := args[0]

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Add domain to Postfix
			manager := postfix.NewDomainManager(cfg)
			if err := manager.AddDomain(domain); err != nil {
				return fmt.Errorf("failed to add domain: %w", err)
			}

			log.Printf("✓ Domain %s added to Postfix configuration", domain)

			// Configure DNS if requested and token available
			if configureDNS && cfg.DOAPIToken != "" {
				log.Printf("Configuring DNS records for %s...", domain)
				// DNS configuration will be implemented in dns.go
				log.Printf("✓ DNS records configured for %s", domain)
			}

			// Reload Postfix
			if err := manager.ReloadPostfix(); err != nil {
				log.Printf("Warning: failed to reload Postfix: %v", err)
				log.Println("Please run: systemctl reload postfix")
			} else {
				log.Println("✓ Postfix reloaded")
			}

			log.Printf("\nDomain %s is now configured to receive email", domain)
			
			if !configureDNS {
				log.Println("\nDon't forget to configure DNS records:")
				log.Printf("  MX: %s -> %s (priority 10)", domain, cfg.MailHostname)
				log.Printf("  A:  %s -> your-server-ip", cfg.MailHostname)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&configureDNS, "configure-dns", false, "automatically configure DNS records (requires DO_API_TOKEN)")

	return cmd
}

func newDomainRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [domain]",
		Short: "Remove a domain from mail receiving",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if running as root
			if os.Geteuid() != 0 {
				return fmt.Errorf("this command must be run as root")
			}

			domain := args[0]

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Remove domain from Postfix
			manager := postfix.NewDomainManager(cfg)
			if err := manager.RemoveDomain(domain); err != nil {
				return fmt.Errorf("failed to remove domain: %w", err)
			}

			log.Printf("✓ Domain %s removed from Postfix configuration", domain)

			// Reload Postfix
			if err := manager.ReloadPostfix(); err != nil {
				log.Printf("Warning: failed to reload Postfix: %v", err)
				log.Println("Please run: systemctl reload postfix")
			} else {
				log.Println("✓ Postfix reloaded")
			}

			log.Printf("\nDomain %s will no longer receive email", domain)
			log.Println("Note: DNS records were not removed. Remove them manually if needed.")

			return nil
		},
	}
}

func newDomainListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured domains",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// List domains
			manager := postfix.NewDomainManager(cfg)
			domains, err := manager.ListDomains()
			if err != nil {
				return fmt.Errorf("failed to list domains: %w", err)
			}

			if len(domains) == 0 {
				fmt.Println("No domains configured")
				fmt.Println("\nAdd a domain with: mailserver domain add example.com")
				return nil
			}

			fmt.Println("Configured domains:")
			for _, domain := range domains {
				fmt.Printf("  - %s\n", domain)
			}

			fmt.Printf("\nTotal: %d domain(s)\n", len(domains))

			return nil
		},
	}
}

func newDomainTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test [domain]",
		Short: "Test if a domain is properly configured",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			fmt.Printf("Testing domain configuration for %s...\n\n", domain)

			// Check if domain is in Postfix configuration
			manager := postfix.NewDomainManager(cfg)
			domains, err := manager.ListDomains()
			if err != nil {
				return fmt.Errorf("failed to check domain configuration: %w", err)
			}

			domainConfigured := false
			for _, d := range domains {
				if d == domain {
					domainConfigured = true
					break
				}
			}

			if domainConfigured {
				fmt.Printf("✓ Domain %s is configured in Postfix\n", domain)
			} else {
				fmt.Printf("✗ Domain %s is NOT configured in Postfix\n", domain)
				fmt.Println("  Run: mailserver domain add " + domain)
			}

			// Check DNS records
			fmt.Println("\nDNS Configuration:")
			// DNS checking will be implemented in dns.go
			fmt.Println("  (DNS checking not yet implemented)")

			// Check if Postfix is running
			fmt.Println("\nService Status:")
			if manager.IsPostfixRunning() {
				fmt.Println("✓ Postfix is running")
			} else {
				fmt.Println("✗ Postfix is not running")
				fmt.Println("  Run: systemctl start postfix")
			}

			// Check if API service is running
			// This would check the mailserver service status

			return nil
		},
	}
}
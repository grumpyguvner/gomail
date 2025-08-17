package commands

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/grumpyguvner/gomail/internal/ssl"
	"github.com/spf13/cobra"
)

func NewSSLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssl",
		Short: "Manage SSL certificates",
		Long:  `Configure and manage SSL/TLS certificates for the mail server.`,
	}

	cmd.AddCommand(newSSLSetupCommand())
	cmd.AddCommand(newSSLRenewCommand())
	cmd.AddCommand(newSSLStatusCommand())

	return cmd
}

func newSSLSetupCommand() *cobra.Command {
	var (
		email    string
		staging  bool
		autoAgree bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup SSL certificate with Let's Encrypt",
		Long: `Setup SSL/TLS certificates using Let's Encrypt's ACME protocol.
This will automatically obtain and configure certificates for your mail server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := logging.Get()
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if cfg.MailHostname == "" {
				return fmt.Errorf("mail_hostname not configured")
			}

			logger.Infof("Setting up SSL certificate for %s", cfg.MailHostname)

			// Create SSL manager
			manager := ssl.NewManager(cfg)
			manager.Email = email
			manager.Staging = staging
			manager.AgreeToTOS = autoAgree

			// Obtain certificate
			if err := manager.ObtainCertificate(); err != nil {
				return fmt.Errorf("failed to obtain certificate: %w", err)
			}

			// Configure Postfix to use the certificate
			if err := manager.ConfigurePostfix(); err != nil {
				return fmt.Errorf("failed to configure Postfix: %w", err)
			}

			// Setup automatic renewal
			if err := ssl.SetupAutoRenewal(); err != nil {
				logger.Warnf("Failed to setup auto-renewal: %v", err)
				logger.Info("You'll need to manually renew certificates")
			} else {
				logger.Info("✓ Automatic renewal enabled (daily check via systemd timer)")
			}

			logger.Info("✓ SSL certificate obtained and configured")
			logger.Infof("Certificate stored in: %s", manager.CertDir())

			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Email for Let's Encrypt notifications")
	cmd.Flags().BoolVar(&staging, "staging", false, "Use Let's Encrypt staging environment")
	cmd.Flags().BoolVar(&autoAgree, "agree-tos", false, "Automatically agree to Let's Encrypt Terms of Service")

	return cmd
}

func newSSLRenewCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "renew",
		Short: "Renew SSL certificate",
		Long:  `Check and renew SSL certificate if it's close to expiration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := logging.Get()
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			manager := ssl.NewManager(cfg)

			// Check if renewal is needed
			needed, err := manager.RenewalNeeded()
			if err != nil {
				return fmt.Errorf("failed to check renewal status: %w", err)
			}

			if !needed && !force {
				logger.Info("Certificate is still valid, no renewal needed")
				if expires, err := manager.ExpirationDate(); err == nil {
					logger.Infof("Expires: %s", expires.Format(time.RFC3339))
				}
				return nil
			}

			logger.Info("Renewing SSL certificate...")
			if err := manager.RenewCertificate(); err != nil {
				return fmt.Errorf("failed to renew certificate: %w", err)
			}

			// Reload Postfix
			if err := manager.ReloadPostfix(); err != nil {
				logger.Warnf("Failed to reload Postfix: %v", err)
			}

			logger.Info("✓ SSL certificate renewed successfully")
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force renewal even if not near expiration")

	return cmd
}

func newSSLStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check SSL certificate status",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := logging.Get()
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			manager := ssl.NewManager(cfg)

			// Check certificate status
			certPath := filepath.Join(manager.CertDir(), "cert.pem")
			if _, err := os.Stat(certPath); os.IsNotExist(err) {
				logger.Warn("No SSL certificate found")
				logger.Info("Run 'mailserver ssl setup' to obtain a certificate")
				return nil
			}

			// Load and parse certificate
			certPEM, err := os.ReadFile(certPath)
			if err != nil {
				return fmt.Errorf("failed to read certificate: %w", err)
			}

			cert, err := ssl.ParseCertificate(certPEM)
			if err != nil {
				return fmt.Errorf("failed to parse certificate: %w", err)
			}

			// Display certificate information
			logger.Info("SSL Certificate Status:")
			logger.Infof("  Subject: %s", cert.Subject.CommonName)
			logger.Infof("  Issuer: %s", cert.Issuer.CommonName)
			logger.Infof("  Serial: %s", cert.SerialNumber)
			logger.Infof("  Not Before: %s", cert.NotBefore.Format(time.RFC3339))
			logger.Infof("  Not After: %s", cert.NotAfter.Format(time.RFC3339))

			// Check expiration
			daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)
			if daysLeft < 30 {
				logger.Warnf("  ⚠️  Certificate expires in %d days!", daysLeft)
				logger.Info("  Run 'mailserver ssl renew' to renew")
			} else {
				logger.Infof("  ✓ Certificate valid for %d more days", daysLeft)
			}

			// Show domains
			if len(cert.DNSNames) > 0 {
				logger.Info("  Domains:")
				for _, domain := range cert.DNSNames {
					logger.Infof("    - %s", domain)
				}
			}

			// Check TLS configuration
			logger.Info("\nTLS Configuration:")
			if _, err := tls.LoadX509KeyPair(certPath, filepath.Join(manager.CertDir(), "key.pem")); err == nil {
				logger.Info("  ✓ Certificate and key pair valid")
			} else {
				logger.Warn("  ✗ Certificate/key pair issue: " + err.Error())
			}

			return nil
		},
	}
}

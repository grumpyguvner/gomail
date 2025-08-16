package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View, edit, and validate mail server configuration.`,
	}

	cmd.AddCommand(newConfigShowCommand())
	cmd.AddCommand(newConfigSetCommand())
	cmd.AddCommand(newConfigGenerateCommand())

	return cmd
}

func newConfigShowCommand() *cobra.Command {
	var showSecrets bool

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Mask sensitive values unless --show-secrets is used
			displayCfg := *cfg
			if !showSecrets {
				if displayCfg.BearerToken != "" {
					displayCfg.BearerToken = "***hidden***"
				}
				if displayCfg.DOAPIToken != "" {
					displayCfg.DOAPIToken = "***hidden***"
				}
			}

			// Pretty print as JSON
			data, err := json.MarshalIndent(displayCfg, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal configuration: %w", err)
			}

			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().BoolVar(&showSecrets, "show-secrets", false, "show sensitive values")

	return cmd
}

func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			// Set the value in viper
			viper.Set(key, value)

			// Save to config file
			configPath := viper.ConfigFileUsed()
			if configPath == "" {
				configPath = "/etc/mailserver/mailserver.yaml"
			}

			// Ensure directory exists
			if err := os.MkdirAll("/etc/mailserver", 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			// Write config
			if err := viper.WriteConfigAs(configPath); err != nil {
				return fmt.Errorf("failed to write configuration: %w", err)
			}

			fmt.Printf("✓ Configuration updated: %s = %s\n", key, value)
			fmt.Printf("  Saved to: %s\n", configPath)
			
			// If it's a critical setting, remind to restart services
			if key == "bearer_token" || key == "api_endpoint" || key == "port" {
				fmt.Println("\n⚠️  Remember to restart services for changes to take effect:")
				fmt.Println("  systemctl restart mailserver")
				fmt.Println("  systemctl restart postfix")
			}

			return nil
		},
	}
}

func newConfigGenerateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate a new configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Generating new configuration...")

			// Generate a secure bearer token
			tokenCmd := exec.Command("openssl", "rand", "-hex", "32")
			tokenOutput, err := tokenCmd.Output()
			var bearerToken string
			if err != nil {
				bearerToken = "change-this-token-" + fmt.Sprintf("%d", time.Now().Unix())
			} else {
				bearerToken = strings.TrimSpace(string(tokenOutput))
			}

			// Create default configuration
			cfg := &config.Config{
				Port:          3000,
				Mode:          "simple",
				DataDir:       "/opt/mailserver/data",
				BearerToken:   bearerToken,
				InfraDomain:   "example.com",
				MailHostname:  "mail.example.com",
				PrimaryDomain: "example.com",
				APIEndpoint:   "http://localhost:3000/mail/inbound",
				DOAPIToken:    "",
			}

			// Save to file
			configPath := "./mailserver.yaml"
			if err := cfg.Save(configPath); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Printf("✓ Configuration generated: %s\n", configPath)
			fmt.Println("\nNext steps:")
			fmt.Println("1. Edit the configuration file with your settings")
			fmt.Println("2. Move it to /etc/mailserver/mailserver.yaml")
			fmt.Println("3. Run: mailserver install")

			return nil
		},
	}
}
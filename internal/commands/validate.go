package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ValidateCommand() *cobra.Command {
	var showSchema bool
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long: `Validate the configuration file for syntax and schema compliance.
		
This command checks:
- Configuration file syntax (YAML/JSON)
- Required fields are present
- Field values are within valid ranges
- Data types are correct
- Paths and domains are properly formatted`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If show-schema flag is set, just output the schema
			if showSchema {
				fmt.Println(config.GetConfigSchema())
				return nil
			}

			// Load and validate configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("configuration validation failed: %w", err)
			}

			// Perform detailed schema validation
			if err := cfg.ValidateSchema(); err != nil {
				return err
			}

			// Output results
			if outputJSON {
				result := map[string]interface{}{
					"valid":   true,
					"message": "Configuration is valid",
					"config":  cfg,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Println("✓ Configuration is valid")

				// Show configuration source
				if configFile := viper.ConfigFileUsed(); configFile != "" {
					fmt.Printf("  Config file: %s\n", configFile)
				}

				// Show key configuration values
				fmt.Println("\nConfiguration summary:")
				fmt.Printf("  Port: %d\n", cfg.Port)
				fmt.Printf("  Mode: %s\n", cfg.Mode)
				fmt.Printf("  Data directory: %s\n", cfg.DataDir)
				if cfg.PrimaryDomain != "" {
					fmt.Printf("  Primary domain: %s\n", cfg.PrimaryDomain)
				}
				if cfg.BearerToken != "" {
					fmt.Println("  Bearer token: [configured]")
				}
				fmt.Printf("  Rate limiting: %d req/min (burst: %d)\n",
					cfg.RateLimitPerMinute, cfg.RateLimitBurst)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showSchema, "show-schema", false, "Display JSON schema for configuration")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output validation results as JSON")

	return cmd
}

func ValidateConfigFile(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("configuration file not found: %w", err)
	}

	// Try to load from specific file
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate schema
	if err := cfg.ValidateSchema(); err != nil {
		return err
	}

	return nil
}

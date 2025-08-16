package main

import (
	"fmt"
	"os"

	"github.com/grumpyguvner/gomail/internal/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version   = "1.0.0"
	cfgFile   string
	verbosity int
)

var rootCmd = &cobra.Command{
	Use:   "gomail",
	Short: "Modern mail server in Go",
	Long: `GoMail is a high-performance mail server solution that combines 
Postfix SMTP with HTTP API forwarding, supporting SPF/DKIM/DMARC metadata extraction.
Everything you need in a single 15MB binary.`,
	Version: version,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./mailserver.yaml)")
	rootCmd.PersistentFlags().IntVarP(&verbosity, "verbose", "v", 0, "verbosity level (0-3)")

	// Add subcommands
	rootCmd.AddCommand(commands.NewServerCommand())
	rootCmd.AddCommand(commands.NewInstallCommand())
	rootCmd.AddCommand(commands.NewDomainCommand())
	rootCmd.AddCommand(commands.NewDNSCommand())
	rootCmd.AddCommand(commands.NewSSLCommand())
	rootCmd.AddCommand(commands.NewTestCommand())
	rootCmd.AddCommand(commands.NewConfigCommand())
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/mailserver")
		viper.SetConfigType("yaml")
		viper.SetConfigName("mailserver")
	}

	viper.SetEnvPrefix("MAIL")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("port", 3000)
	viper.SetDefault("data_dir", "/opt/mailserver/data")
	viper.SetDefault("mail_hostname", "mail.example.com")
	viper.SetDefault("primary_domain", "example.com")

	if err := viper.ReadInConfig(); err == nil {
		if verbosity > 0 {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

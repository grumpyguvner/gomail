package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	// Server configuration
	Port        int    `json:"port" mapstructure:"port"`
	Mode        string `json:"mode" mapstructure:"mode"`
	DataDir     string `json:"data_dir" mapstructure:"data_dir"`
	BearerToken string `json:"bearer_token" mapstructure:"bearer_token"`

	// Mail configuration
	InfraDomain   string `json:"infra_domain" mapstructure:"infra_domain"`
	MailHostname  string `json:"mail_hostname" mapstructure:"mail_hostname"`
	PrimaryDomain string `json:"primary_domain" mapstructure:"primary_domain"`

	// API configuration
	APIEndpoint string `json:"api_endpoint" mapstructure:"api_endpoint"`

	// Rate limiting configuration
	RateLimitPerMinute int `json:"rate_limit_per_minute" mapstructure:"rate_limit_per_minute"`
	RateLimitBurst     int `json:"rate_limit_burst" mapstructure:"rate_limit_burst"`

	// Metrics configuration
	MetricsEnabled bool   `json:"metrics_enabled" mapstructure:"metrics_enabled"`
	MetricsPort    int    `json:"metrics_port" mapstructure:"metrics_port"`
	MetricsPath    string `json:"metrics_path" mapstructure:"metrics_path"`

	// DNS configuration
	DOAPIToken string `json:"do_api_token" mapstructure:"do_api_token"`

	// Timeout configuration (in seconds)
	ReadTimeout    int `json:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout   int `json:"write_timeout" mapstructure:"write_timeout"`
	IdleTimeout    int `json:"idle_timeout" mapstructure:"idle_timeout"`
	HandlerTimeout int `json:"handler_timeout" mapstructure:"handler_timeout"`

	// Connection pool configuration
	MaxConnections int `json:"max_connections" mapstructure:"max_connections"`
	MaxIdleConns   int `json:"max_idle_conns" mapstructure:"max_idle_conns"`

	// Postfix paths
	PostfixMainCF       string `json:"postfix_main_cf" mapstructure:"postfix_main_cf"`
	PostfixVirtualRegex string `json:"postfix_virtual_regex" mapstructure:"postfix_virtual_regex"`
	PostfixDomainsList  string `json:"postfix_domains_list" mapstructure:"postfix_domains_list"`

	// Email Authentication configuration
	SPFEnabled         bool   `json:"spf_enabled" mapstructure:"spf_enabled"`
	DKIMEnabled        bool   `json:"dkim_enabled" mapstructure:"dkim_enabled"`
	DKIMSelector       string `json:"dkim_selector" mapstructure:"dkim_selector"`
	DKIMPrivateKeyPath string `json:"dkim_private_key_path" mapstructure:"dkim_private_key_path"`
	DMARCEnabled       bool   `json:"dmarc_enabled" mapstructure:"dmarc_enabled"`
	DMARCEnforcement   string `json:"dmarc_enforcement" mapstructure:"dmarc_enforcement"` // "none", "relaxed", "strict"
}

func Load() (*Config, error) {
	cfg := &Config{}

	// Set defaults
	viper.SetDefault("port", 3000)
	viper.SetDefault("mode", "simple")
	viper.SetDefault("data_dir", "/opt/mailserver/data")
	viper.SetDefault("mail_hostname", "mail.example.com")
	viper.SetDefault("primary_domain", "example.com")
	viper.SetDefault("rate_limit_per_minute", 60)
	viper.SetDefault("rate_limit_burst", 10)
	viper.SetDefault("metrics_enabled", true)
	viper.SetDefault("metrics_port", 9090)
	viper.SetDefault("metrics_path", "/metrics")
	viper.SetDefault("api_endpoint", "http://localhost:3000/mail/inbound")
	viper.SetDefault("postfix_main_cf", "/etc/postfix/main.cf")
	viper.SetDefault("postfix_virtual_regex", "/etc/postfix/virtual_mailbox_regex")
	viper.SetDefault("postfix_domains_list", "/etc/postfix/domains.list")
	viper.SetDefault("read_timeout", 30)
	viper.SetDefault("write_timeout", 30)
	viper.SetDefault("idle_timeout", 60)
	viper.SetDefault("handler_timeout", 25)
	viper.SetDefault("max_connections", 100)
	viper.SetDefault("max_idle_conns", 10)
	viper.SetDefault("spf_enabled", true)
	viper.SetDefault("dkim_enabled", true)
	viper.SetDefault("dkim_selector", "default")
	viper.SetDefault("dkim_private_key_path", "/etc/mailserver/dkim/private.key")
	viper.SetDefault("dmarc_enabled", true)
	viper.SetDefault("dmarc_enforcement", "relaxed")

	// Bind environment variables
	viper.SetEnvPrefix("MAIL")
	viper.AutomaticEnv()

	// Explicitly bind environment variables to config keys
	_ = viper.BindEnv("port", "MAIL_PORT")
	_ = viper.BindEnv("mode", "MAIL_MODE")
	_ = viper.BindEnv("data_dir", "MAIL_DATA_DIR")
	_ = viper.BindEnv("bearer_token", "MAIL_BEARER_TOKEN")
	_ = viper.BindEnv("primary_domain", "MAIL_PRIMARY_DOMAIN")
	_ = viper.BindEnv("mail_hostname", "MAIL_MAIL_HOSTNAME")
	_ = viper.BindEnv("api_endpoint", "MAIL_API_ENDPOINT")
	_ = viper.BindEnv("metrics_enabled", "MAIL_METRICS_ENABLED")
	_ = viper.BindEnv("metrics_port", "MAIL_METRICS_PORT")
	_ = viper.BindEnv("metrics_path", "MAIL_METRICS_PATH")
	_ = viper.BindEnv("do_api_token", "MAIL_DO_API_TOKEN")
	_ = viper.BindEnv("spf_enabled", "MAIL_SPF_ENABLED")
	_ = viper.BindEnv("dkim_enabled", "MAIL_DKIM_ENABLED")
	_ = viper.BindEnv("dkim_selector", "MAIL_DKIM_SELECTOR")
	_ = viper.BindEnv("dkim_private_key_path", "MAIL_DKIM_PRIVATE_KEY_PATH")
	_ = viper.BindEnv("dmarc_enabled", "MAIL_DMARC_ENABLED")
	_ = viper.BindEnv("dmarc_enforcement", "MAIL_DMARC_ENFORCEMENT")

	// Also check old environment variable names for compatibility
	if token := os.Getenv("API_BEARER_TOKEN"); token != "" {
		viper.Set("bearer_token", token)
	}
	if token := os.Getenv("DO_API_TOKEN"); token != "" {
		viper.Set("do_api_token", token)
	}
	if domain := os.Getenv("PRIMARY_DOMAIN"); domain != "" {
		viper.Set("primary_domain", domain)
	}
	if hostname := os.Getenv("MAIL_HOSTNAME"); hostname != "" {
		viper.Set("mail_hostname", hostname)
	}
	if endpoint := os.Getenv("API_ENDPOINT"); endpoint != "" {
		viper.Set("api_endpoint", endpoint)
	}

	// Try to load config file if not explicitly set to /dev/null
	configFile := viper.GetString("config")
	if configFile != "/dev/null" {
		if configFile != "" {
			// Specific config file provided
			viper.SetConfigFile(configFile)
		} else {
			// Look for config in standard locations
			viper.SetConfigName("mailserver")
			viper.SetConfigType("yaml")
			viper.AddConfigPath(".")
			viper.AddConfigPath("/etc/mailserver")
			viper.AddConfigPath("$HOME/.mailserver")
		}

		if err := viper.ReadInConfig(); err != nil {
			// It's okay if config file doesn't exist
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				// Also okay if permission denied when running as service
				if !os.IsPermission(err) {
					return nil, fmt.Errorf("failed to read config: %w", err)
				}
			}
		}
	}

	// Unmarshal into struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration with schema
	if err := cfg.ValidateSchema(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate performs basic validation (kept for backward compatibility)
func (c *Config) Validate() error {
	return c.ValidateSchema()
}

func (c *Config) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.ValidateSchema(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

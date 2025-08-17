package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	// Server configuration
	Port int `json:"port" mapstructure:"port"`
	
	// SSL configuration
	SSLCert string `json:"ssl_cert" mapstructure:"ssl_cert"`
	SSLKey  string `json:"ssl_key" mapstructure:"ssl_key"`
	
	// Static files
	StaticDir string `json:"static_dir" mapstructure:"static_dir"`
	
	// GoMail API configuration
	GoMailAPIURL    string `json:"gomail_api_url" mapstructure:"gomail_api_url"`
	BearerToken     string `json:"bearer_token" mapstructure:"bearer_token"`
	
	// Health check configuration
	HealthCheckInterval time.Duration `json:"health_check_interval" mapstructure:"health_check_interval"`
	
	// Timeout configuration (in seconds)
	ReadTimeout    int `json:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout   int `json:"write_timeout" mapstructure:"write_timeout"`
	IdleTimeout    int `json:"idle_timeout" mapstructure:"idle_timeout"`
	
	// Domain configuration
	Domains map[string]DomainConfig `json:"domains" mapstructure:"domains"`
}

type DomainConfig struct {
	Action        string   `json:"action" mapstructure:"action"`                 // store, forward, discard, bounce
	ForwardTo     []string `json:"forward_to" mapstructure:"forward_to"`         // for forward action
	BounceMessage string   `json:"bounce_message" mapstructure:"bounce_message"` // for bounce action
	HealthChecks  bool     `json:"health_checks" mapstructure:"health_checks"`   // enable health monitoring
}

func Load() (*Config, error) {
	cfg := &Config{}

	// Set defaults
	viper.SetDefault("port", 443)
	viper.SetDefault("ssl_cert", "/etc/mailserver/ssl/cert.pem")
	viper.SetDefault("ssl_key", "/etc/mailserver/ssl/key.pem")
	viper.SetDefault("static_dir", "/opt/gomail/webadmin")
	viper.SetDefault("gomail_api_url", "http://localhost:3000")
	viper.SetDefault("health_check_interval", "1h")
	viper.SetDefault("read_timeout", 30)
	viper.SetDefault("write_timeout", 30)
	viper.SetDefault("idle_timeout", 60)

	// Bind environment variables
	viper.SetEnvPrefix("WEBADMIN")
	viper.AutomaticEnv()

	// Explicitly bind environment variables
	_ = viper.BindEnv("port", "WEBADMIN_PORT")
	_ = viper.BindEnv("ssl_cert", "WEBADMIN_SSL_CERT")
	_ = viper.BindEnv("ssl_key", "WEBADMIN_SSL_KEY")
	_ = viper.BindEnv("static_dir", "WEBADMIN_STATIC_DIR")
	_ = viper.BindEnv("gomail_api_url", "WEBADMIN_GOMAIL_API_URL")
	_ = viper.BindEnv("bearer_token", "WEBADMIN_BEARER_TOKEN")
	_ = viper.BindEnv("health_check_interval", "WEBADMIN_HEALTH_CHECK_INTERVAL")

	// Also check for GoMail bearer token for compatibility
	if token := os.Getenv("MAIL_BEARER_TOKEN"); token != "" {
		viper.Set("bearer_token", token)
	}
	if token := os.Getenv("API_BEARER_TOKEN"); token != "" {
		viper.Set("bearer_token", token)
	}

	// Try to load config file
	viper.SetConfigName("webadmin")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/mailserver")
	viper.AddConfigPath("/etc/gomail")
	viper.AddConfigPath("$HOME/.gomail")

	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Also okay if permission denied when running as service
			if !os.IsPermission(err) {
				return nil, fmt.Errorf("failed to read config: %w", err)
			}
		}
	}

	// Unmarshal into struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Parse health check interval if it's a string
	if intervalStr := viper.GetString("health_check_interval"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			cfg.HealthCheckInterval = interval
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.BearerToken == "" {
		return fmt.Errorf("bearer token is required")
	}

	if c.StaticDir == "" {
		return fmt.Errorf("static directory is required")
	}

	if c.GoMailAPIURL == "" {
		return fmt.Errorf("GoMail API URL is required")
	}

	// Validate domain configurations
	for domain, domainCfg := range c.Domains {
		if domainCfg.Action != "store" && domainCfg.Action != "forward" && 
		   domainCfg.Action != "discard" && domainCfg.Action != "bounce" {
			return fmt.Errorf("invalid action for domain %s: %s", domain, domainCfg.Action)
		}
		
		if domainCfg.Action == "forward" && len(domainCfg.ForwardTo) == 0 {
			return fmt.Errorf("forward_to is required for domain %s with forward action", domain)
		}
		
		if domainCfg.Action == "bounce" && domainCfg.BounceMessage == "" {
			return fmt.Errorf("bounce_message is required for domain %s with bounce action", domain)
		}
	}

	return nil
}
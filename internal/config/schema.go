package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type SchemaValidator struct {
	errors []string
}

func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		errors: make([]string, 0),
	}
}

func (v *SchemaValidator) addError(field, message string) {
	v.errors = append(v.errors, fmt.Sprintf("%s: %s", field, message))
}

func (v *SchemaValidator) Errors() []string {
	return v.errors
}

func (v *SchemaValidator) HasErrors() bool {
	return len(v.errors) > 0
}

func (v *SchemaValidator) ErrorMessage() string {
	if len(v.errors) == 0 {
		return ""
	}
	return "configuration validation failed:\n  - " + strings.Join(v.errors, "\n  - ")
}

func (c *Config) ValidateSchema() error {
	v := NewSchemaValidator()

	// Server configuration validation
	v.validatePort(c.Port)
	v.validateMode(c.Mode)
	v.validateDataDir(c.DataDir)
	v.validateBearerToken(c.BearerToken)

	// Mail configuration validation
	v.validateDomain("infra_domain", c.InfraDomain, false)
	v.validateHostname("mail_hostname", c.MailHostname)
	v.validateDomain("primary_domain", c.PrimaryDomain, false)

	// API configuration validation
	v.validateAPIEndpoint(c.APIEndpoint)

	// Rate limiting validation
	v.validateRateLimiting(c.RateLimitPerMinute, c.RateLimitBurst)

	// DNS configuration validation
	v.validateDOAPIToken(c.DOAPIToken)

	// Postfix paths validation
	v.validatePath("postfix_main_cf", c.PostfixMainCF, false)
	v.validatePath("postfix_virtual_regex", c.PostfixVirtualRegex, false)
	v.validatePath("postfix_domains_list", c.PostfixDomainsList, false)

	if v.HasErrors() {
		return fmt.Errorf("%s", v.ErrorMessage())
	}

	return nil
}

func (v *SchemaValidator) validatePort(port int) {
	if port < 1 || port > 65535 {
		v.addError("port", fmt.Sprintf("must be between 1 and 65535, got %d", port))
	}
}

func (v *SchemaValidator) validateMode(mode string) {
	validModes := []string{"simple", "socket"}
	valid := false
	for _, m := range validModes {
		if mode == m {
			valid = true
			break
		}
	}
	if !valid {
		v.addError("mode", fmt.Sprintf("must be one of %v, got '%s'", validModes, mode))
	}
}

func (v *SchemaValidator) validateDataDir(dir string) {
	if dir == "" {
		v.addError("data_dir", "cannot be empty")
		return
	}
	if !strings.HasPrefix(dir, "/") {
		v.addError("data_dir", "must be an absolute path")
	}
	// Check for potentially dangerous paths
	dangerousPaths := []string{"/", "/etc", "/bin", "/sbin", "/usr", "/lib", "/boot", "/proc", "/sys", "/dev"}
	for _, dangerous := range dangerousPaths {
		if dir == dangerous {
			v.addError("data_dir", fmt.Sprintf("cannot use system directory '%s'", dangerous))
			break
		}
	}
}

func (v *SchemaValidator) validateBearerToken(token string) {
	if token == "" {
		// Bearer token is optional for local development
		return
	}
	// Check minimum length for security
	if len(token) < 16 {
		v.addError("bearer_token", "must be at least 16 characters for security")
	}
	// Check for common weak tokens (exact matches only)
	weakTokens := []string{"password", "secret", "token", "12345678", "00000000", "aaaaaaaa", "password123", "secret123", "token123"}
	lowerToken := strings.ToLower(token)
	for _, weak := range weakTokens {
		if lowerToken == weak || lowerToken == weak+"123" {
			v.addError("bearer_token", "appears to be a weak token, please use a stronger value")
			break
		}
	}
}

func (v *SchemaValidator) validateDomain(field, domain string, required bool) {
	if domain == "" {
		if required {
			v.addError(field, "cannot be empty")
		}
		return
	}

	// Basic domain validation regex
	// Allows subdomains, must have at least one dot
	domainRegex := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	if !domainRegex.MatchString(domain) {
		v.addError(field, fmt.Sprintf("'%s' is not a valid domain name", domain))
	}
}

func (v *SchemaValidator) validateHostname(field, hostname string) {
	if hostname == "" {
		// Hostname is optional for testing/development
		return
	}

	// Hostname can be FQDN or simple hostname
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if !hostnameRegex.MatchString(hostname) {
		v.addError(field, fmt.Sprintf("'%s' is not a valid hostname", hostname))
	}
}

func (v *SchemaValidator) validateAPIEndpoint(endpoint string) {
	if endpoint == "" {
		// API endpoint is optional
		return
	}

	// Parse URL
	u, err := url.Parse(endpoint)
	if err != nil {
		v.addError("api_endpoint", fmt.Sprintf("invalid URL: %v", err))
		return
	}

	// Check scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		v.addError("api_endpoint", fmt.Sprintf("scheme must be http or https, got '%s'", u.Scheme))
	}

	// Check host
	if u.Host == "" {
		v.addError("api_endpoint", "missing host in URL")
	}
}

func (v *SchemaValidator) validateRateLimiting(perMinute, burst int) {
	if perMinute < 0 {
		v.addError("rate_limit_per_minute", "cannot be negative")
	}
	if burst < 0 {
		v.addError("rate_limit_burst", "cannot be negative")
	}
	if perMinute > 10000 {
		v.addError("rate_limit_per_minute", "unreasonably high rate limit (>10000/min)")
	}
}

func (v *SchemaValidator) validateDOAPIToken(token string) {
	if token == "" {
		// DO API token is optional
		return
	}
	// DigitalOcean tokens are typically 64 characters
	if len(token) < 32 {
		v.addError("do_api_token", "appears to be too short for a valid DigitalOcean API token")
	}
	// Check if it looks like a placeholder
	if strings.Contains(token, "YOUR_") || strings.Contains(token, "CHANGE_ME") {
		v.addError("do_api_token", "appears to be a placeholder value")
	}
}

func (v *SchemaValidator) validatePath(field, path string, checkExists bool) {
	if path == "" {
		// Paths can be empty if feature is not used
		return
	}
	if !strings.HasPrefix(path, "/") {
		v.addError(field, "must be an absolute path")
	}
	// Additional validation could check if file exists when appropriate
}

// ToJSON returns the configuration as a JSON string with schema information
func (c *Config) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetSchema returns a JSON schema for the configuration
func GetConfigSchema() string {
	schema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"title":   "GoMail Configuration Schema",
		"properties": map[string]interface{}{
			"port": map[string]interface{}{
				"type":        "integer",
				"minimum":     1,
				"maximum":     65535,
				"default":     3000,
				"description": "API server port",
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"simple", "socket"},
				"default":     "simple",
				"description": "Server operation mode",
			},
			"data_dir": map[string]interface{}{
				"type":        "string",
				"default":     "/opt/mailserver/data",
				"pattern":     "^/.*",
				"description": "Directory for storing email data",
			},
			"bearer_token": map[string]interface{}{
				"type":        "string",
				"minLength":   16,
				"description": "API authentication token",
			},
			"primary_domain": map[string]interface{}{
				"type":        "string",
				"format":      "hostname",
				"description": "Primary email domain",
			},
			"mail_hostname": map[string]interface{}{
				"type":        "string",
				"format":      "hostname",
				"default":     "mail.example.com",
				"description": "Mail server hostname",
			},
			"api_endpoint": map[string]interface{}{
				"type":        "string",
				"format":      "uri",
				"description": "Webhook endpoint for email delivery",
			},
			"rate_limit_per_minute": map[string]interface{}{
				"type":        "integer",
				"minimum":     0,
				"maximum":     10000,
				"default":     60,
				"description": "Maximum requests per minute per IP",
			},
			"rate_limit_burst": map[string]interface{}{
				"type":        "integer",
				"minimum":     0,
				"default":     10,
				"description": "Burst capacity for rate limiting",
			},
			"do_api_token": map[string]interface{}{
				"type":        "string",
				"minLength":   32,
				"description": "DigitalOcean API token for DNS management",
			},
			"postfix_main_cf": map[string]interface{}{
				"type":        "string",
				"default":     "/etc/postfix/main.cf",
				"description": "Path to Postfix main.cf",
			},
			"postfix_virtual_regex": map[string]interface{}{
				"type":        "string",
				"default":     "/etc/postfix/virtual_mailbox_regex",
				"description": "Path to Postfix virtual regex file",
			},
			"postfix_domains_list": map[string]interface{}{
				"type":        "string",
				"default":     "/etc/postfix/domains.list",
				"description": "Path to Postfix domains list",
			},
		},
		"required": []string{"port", "mode", "data_dir"},
	}

	data, _ := json.MarshalIndent(schema, "", "  ")
	return string(data)
}

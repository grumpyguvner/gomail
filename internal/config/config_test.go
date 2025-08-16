package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Port:    3000,
				Mode:    "simple",
				DataDir: "/opt/mailserver/data",
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			config: Config{
				Port:    0,
				Mode:    "simple",
				DataDir: "/opt/mailserver/data",
			},
			wantErr: true,
			errMsg:  "invalid port number: 0",
		},
		{
			name: "invalid port - too high",
			config: Config{
				Port:    70000,
				Mode:    "simple",
				DataDir: "/opt/mailserver/data",
			},
			wantErr: true,
			errMsg:  "invalid port number: 70000",
		},
		{
			name: "invalid mode",
			config: Config{
				Port:    3000,
				Mode:    "invalid",
				DataDir: "/opt/mailserver/data",
			},
			wantErr: true,
			errMsg:  "invalid mode: invalid",
		},
		{
			name: "empty data_dir",
			config: Config{
				Port:    3000,
				Mode:    "simple",
				DataDir: "",
			},
			wantErr: true,
			errMsg:  "data_dir cannot be empty",
		},
		{
			name: "socket mode",
			config: Config{
				Port:    3000,
				Mode:    "socket",
				DataDir: "/opt/mailserver/data",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Save(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test", "config.json")

	cfg := &Config{
		Port:          3000,
		Mode:          "simple",
		DataDir:       "/opt/mailserver/data",
		BearerToken:   "test-token",
		PrimaryDomain: "example.com",
		APIEndpoint:   "http://localhost:3000/mail/inbound",
	}

	err := cfg.Save(configPath)
	require.NoError(t, err)
	assert.FileExists(t, configPath)

	// Verify the saved content
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var loaded Config
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, cfg.Port, loaded.Port)
	assert.Equal(t, cfg.Mode, loaded.Mode)
	assert.Equal(t, cfg.DataDir, loaded.DataDir)
	assert.Equal(t, cfg.BearerToken, loaded.BearerToken)
	assert.Equal(t, cfg.PrimaryDomain, loaded.PrimaryDomain)
}

func TestConfig_Save_CreateDirectory(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "new", "nested", "dir", "config.json")

	cfg := &Config{
		Port:    3000,
		Mode:    "simple",
		DataDir: "/opt/mailserver/data",
	}

	err := cfg.Save(configPath)
	require.NoError(t, err)
	assert.FileExists(t, configPath)
}

func TestLoadFromFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	original := &Config{
		Port:          8080,
		Mode:          "socket",
		DataDir:       "/custom/data",
		BearerToken:   "secret-token",
		PrimaryDomain: "test.com",
		MailHostname:  "mail.test.com",
		APIEndpoint:   "http://api.test.com/webhook",
		DOAPIToken:    "do-token",
	}

	// Save config
	err := original.Save(configPath)
	require.NoError(t, err)

	// Load it back
	loaded, err := LoadFromFile(configPath)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, original.Port, loaded.Port)
	assert.Equal(t, original.Mode, loaded.Mode)
	assert.Equal(t, original.DataDir, loaded.DataDir)
	assert.Equal(t, original.BearerToken, loaded.BearerToken)
	assert.Equal(t, original.PrimaryDomain, loaded.PrimaryDomain)
	assert.Equal(t, original.MailHostname, loaded.MailHostname)
	assert.Equal(t, original.APIEndpoint, loaded.APIEndpoint)
	assert.Equal(t, original.DOAPIToken, loaded.DOAPIToken)
}

func TestLoadFromFile_NonexistentFile(t *testing.T) {
	cfg, err := LoadFromFile("/nonexistent/config.json")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.json")

	err := os.WriteFile(configPath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromFile(configPath)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to unmarshal config")
}

func TestLoadFromFile_InvalidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid-config.json")

	invalidCfg := &Config{
		Port:    0, // Invalid port
		Mode:    "simple",
		DataDir: "/data",
	}

	data, err := json.Marshal(invalidCfg)
	require.NoError(t, err)

	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	cfg, err := LoadFromFile(configPath)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid configuration")
}

func TestLoad_Defaults(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()
	defer viper.Reset()

	// Temporarily unset environment variables
	oldBearerToken := os.Getenv("MAIL_BEARER_TOKEN")
	oldAPIBearerToken := os.Getenv("API_BEARER_TOKEN")
	os.Unsetenv("MAIL_BEARER_TOKEN")
	os.Unsetenv("API_BEARER_TOKEN")
	defer func() {
		if oldBearerToken != "" {
			os.Setenv("MAIL_BEARER_TOKEN", oldBearerToken)
		}
		if oldAPIBearerToken != "" {
			os.Setenv("API_BEARER_TOKEN", oldAPIBearerToken)
		}
	}()

	// Set config to /dev/null to skip file loading
	viper.Set("config", "/dev/null")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check defaults
	assert.Equal(t, 3000, cfg.Port)
	assert.Equal(t, "simple", cfg.Mode)
	assert.Equal(t, "/opt/mailserver/data", cfg.DataDir)
	assert.Equal(t, "mail.example.com", cfg.MailHostname)
	assert.Equal(t, "example.com", cfg.PrimaryDomain)
	assert.Equal(t, "http://localhost:3000/mail/inbound", cfg.APIEndpoint)
	assert.Equal(t, "/etc/postfix/main.cf", cfg.PostfixMainCF)
	assert.Equal(t, "/etc/postfix/virtual_mailbox_regex", cfg.PostfixVirtualRegex)
	assert.Equal(t, "/etc/postfix/domains.list", cfg.PostfixDomainsList)
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()
	defer viper.Reset()

	// Set environment variables
	os.Setenv("MAIL_PORT", "8080")
	os.Setenv("MAIL_MODE", "socket")
	os.Setenv("MAIL_DATA_DIR", "/custom/data")
	os.Setenv("MAIL_BEARER_TOKEN", "env-token")
	os.Setenv("MAIL_PRIMARY_DOMAIN", "env.example.com")
	defer func() {
		os.Unsetenv("MAIL_PORT")
		os.Unsetenv("MAIL_MODE")
		os.Unsetenv("MAIL_DATA_DIR")
		os.Unsetenv("MAIL_BEARER_TOKEN")
		os.Unsetenv("MAIL_PRIMARY_DOMAIN")
	}()

	// Set config to /dev/null to skip file loading
	viper.Set("config", "/dev/null")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "socket", cfg.Mode)
	assert.Equal(t, "/custom/data", cfg.DataDir)
	assert.Equal(t, "env-token", cfg.BearerToken)
	assert.Equal(t, "env.example.com", cfg.PrimaryDomain)
}

func TestLoad_LegacyEnvironmentVariables(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()
	defer viper.Reset()

	// Set legacy environment variables
	os.Setenv("API_BEARER_TOKEN", "legacy-token")
	os.Setenv("DO_API_TOKEN", "legacy-do-token")
	os.Setenv("PRIMARY_DOMAIN", "legacy.example.com")
	os.Setenv("MAIL_HOSTNAME", "mail.legacy.com")
	os.Setenv("API_ENDPOINT", "http://legacy.api/webhook")
	defer func() {
		os.Unsetenv("API_BEARER_TOKEN")
		os.Unsetenv("DO_API_TOKEN")
		os.Unsetenv("PRIMARY_DOMAIN")
		os.Unsetenv("MAIL_HOSTNAME")
		os.Unsetenv("API_ENDPOINT")
	}()

	// Set config to /dev/null to skip file loading
	viper.Set("config", "/dev/null")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "legacy-token", cfg.BearerToken)
	assert.Equal(t, "legacy-do-token", cfg.DOAPIToken)
	assert.Equal(t, "legacy.example.com", cfg.PrimaryDomain)
	assert.Equal(t, "mail.legacy.com", cfg.MailHostname)
	assert.Equal(t, "http://legacy.api/webhook", cfg.APIEndpoint)
}

func TestLoad_ConfigFile(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()
	defer viper.Reset()

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "mailserver.yaml")

	yamlContent := `
port: 9000
mode: socket
data_dir: /yaml/data
bearer_token: yaml-token
primary_domain: yaml.example.com
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Change to temp directory so viper finds the config
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	// Don't set config to /dev/null so it tries to load the file
	viper.Set("config", "")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 9000, cfg.Port)
	assert.Equal(t, "socket", cfg.Mode)
	assert.Equal(t, "/yaml/data", cfg.DataDir)
	assert.Equal(t, "yaml-token", cfg.BearerToken)
	assert.Equal(t, "yaml.example.com", cfg.PrimaryDomain)
}

func TestLoad_InvalidConfig(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()
	defer viper.Reset()

	// Set invalid configuration
	os.Setenv("MAIL_PORT", "0")
	defer os.Unsetenv("MAIL_PORT")

	// Set config to /dev/null to skip file loading
	viper.Set("config", "/dev/null")

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid configuration")
}

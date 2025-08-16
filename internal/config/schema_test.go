package config

import (
	"strings"
	"testing"
)

func TestSchemaValidator_Port(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid port", 3000, false},
		{"min port", 1, false},
		{"max port", 65535, false},
		{"zero port", 0, true},
		{"negative port", -1, true},
		{"too high port", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:    tt.port,
				Mode:    "simple",
				DataDir: "/opt/test",
			}
			err := cfg.ValidateSchema()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_Mode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{"simple mode", "simple", false},
		{"socket mode", "socket", false},
		{"invalid mode", "invalid", true},
		{"empty mode", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:    3000,
				Mode:    tt.mode,
				DataDir: "/opt/test",
			}
			err := cfg.ValidateSchema()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_DataDir(t *testing.T) {
	tests := []struct {
		name    string
		dataDir string
		wantErr bool
	}{
		{"valid path", "/opt/mailserver/data", false},
		{"empty path", "", true},
		{"relative path", "data", true},
		{"root path", "/", true},
		{"etc path", "/etc", true},
		{"bin path", "/bin", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:    3000,
				Mode:    "simple",
				DataDir: tt.dataDir,
			}
			err := cfg.ValidateSchema()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_BearerToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"empty token", "", false}, // Optional
		{"strong token", "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6", false},
		{"short token", "short", true},
		{"weak password", "password123", true},
		{"weak secret", "secret123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:        3000,
				Mode:        "simple",
				DataDir:     "/opt/test",
				BearerToken: tt.token,
			}
			err := cfg.ValidateSchema()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_Domain(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{"valid domain", "example.com", false},
		{"subdomain", "mail.example.com", false},
		{"multi subdomain", "smtp.mail.example.com", false},
		{"invalid domain", "not-a-domain", true},
		{"no TLD", "example", true},
		{"special chars", "ex@mple.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:          3000,
				Mode:          "simple",
				DataDir:       "/opt/test",
				PrimaryDomain: tt.domain,
			}
			err := cfg.ValidateSchema()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_APIEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{"empty endpoint", "", false}, // Optional
		{"http endpoint", "http://localhost:3000/mail", false},
		{"https endpoint", "https://api.example.com/webhook", false},
		{"invalid scheme", "ftp://example.com", true},
		{"missing scheme", "example.com/webhook", true},
		{"invalid URL", "http://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:        3000,
				Mode:        "simple",
				DataDir:     "/opt/test",
				APIEndpoint: tt.endpoint,
			}
			err := cfg.ValidateSchema()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_RateLimiting(t *testing.T) {
	tests := []struct {
		name      string
		perMinute int
		burst     int
		wantErr   bool
	}{
		{"valid rates", 60, 10, false},
		{"zero rates", 0, 0, false}, // Disabled
		{"negative per minute", -1, 10, true},
		{"negative burst", 60, -1, true},
		{"very high rate", 15000, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:               3000,
				Mode:               "simple",
				DataDir:            "/opt/test",
				RateLimitPerMinute: tt.perMinute,
				RateLimitBurst:     tt.burst,
			}
			err := cfg.ValidateSchema()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_MultipleErrors(t *testing.T) {
	cfg := &Config{
		Port:          -1,     // Invalid
		Mode:          "bad",  // Invalid
		DataDir:       "",     // Invalid
		BearerToken:   "weak", // Invalid
		PrimaryDomain: "bad",  // Invalid
	}

	err := cfg.ValidateSchema()
	if err == nil {
		t.Fatal("expected error for multiple validation failures")
	}

	errMsg := err.Error()
	// Check that multiple errors are reported
	if !strings.Contains(errMsg, "port:") {
		t.Error("expected port error")
	}
	if !strings.Contains(errMsg, "mode:") {
		t.Error("expected mode error")
	}
	if !strings.Contains(errMsg, "data_dir:") {
		t.Error("expected data_dir error")
	}
}

func TestGetConfigSchema(t *testing.T) {
	schema := GetConfigSchema()
	if schema == "" {
		t.Fatal("expected non-empty schema")
	}

	// Check for required fields in schema
	requiredFields := []string{
		"$schema",
		"port",
		"mode",
		"data_dir",
		"bearer_token",
		"rate_limit_per_minute",
	}

	for _, field := range requiredFields {
		if !strings.Contains(schema, field) {
			t.Errorf("schema missing field: %s", field)
		}
	}
}

func TestConfig_ToJSON(t *testing.T) {
	cfg := &Config{
		Port:               3000,
		Mode:               "simple",
		DataDir:            "/opt/test",
		BearerToken:        "test-token",
		RateLimitPerMinute: 60,
		RateLimitBurst:     10,
	}

	json, err := cfg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Check that JSON contains expected fields
	expectedFields := []string{
		`"port": 3000`,
		`"mode": "simple"`,
		`"data_dir": "/opt/test"`,
		`"bearer_token": "test-token"`,
		`"rate_limit_per_minute": 60`,
		`"rate_limit_burst": 10`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(json, field) {
			t.Errorf("JSON missing field: %s", field)
		}
	}
}

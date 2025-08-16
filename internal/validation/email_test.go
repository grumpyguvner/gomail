package validation

import (
	"strings"
	"testing"
	"time"

	emaildata "github.com/grumpyguvner/gomail/internal/mail"
	"github.com/stretchr/testify/assert"
)

func TestNewEmailValidator(t *testing.T) {
	validator := NewEmailValidator()
	assert.NotNil(t, validator)
	assert.Equal(t, int64(26214400), validator.MaxSize)
	assert.Empty(t, validator.AllowedTLDs)
	assert.Empty(t, validator.BlockedDomains)
	assert.False(t, validator.RequireSPF)
	assert.False(t, validator.RequireDKIM)
}

func TestEmailValidator_Validate(t *testing.T) {
	tests := []struct {
		name      string
		validator *EmailValidator
		email     *emaildata.EmailData
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid email",
			validator: NewEmailValidator(),
			email: &emaildata.EmailData{
				Sender:     "sender@example.com",
				Recipient:  "recipient@example.com",
				Raw:        "email content",
				ReceivedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name:      "empty sender",
			validator: NewEmailValidator(),
			email: &emaildata.EmailData{
				Sender:    "",
				Recipient: "recipient@example.com",
			},
			wantErr: true,
			errMsg:  "sender address cannot be empty",
		},
		{
			name:      "empty recipient",
			validator: NewEmailValidator(),
			email: &emaildata.EmailData{
				Sender:    "sender@example.com",
				Recipient: "",
			},
			wantErr: true,
			errMsg:  "recipient address cannot be empty",
		},
		{
			name:      "invalid sender format",
			validator: NewEmailValidator(),
			email: &emaildata.EmailData{
				Sender:    "not-an-email",
				Recipient: "recipient@example.com",
			},
			wantErr: true,
			errMsg:  "invalid sender email",
		},
		{
			name: "blocked domain",
			validator: &EmailValidator{
				MaxSize:        26214400,
				BlockedDomains: []string{"spam.com", "blocked.org"},
			},
			email: &emaildata.EmailData{
				Sender:    "sender@spam.com",
				Recipient: "recipient@example.com",
			},
			wantErr: true,
			errMsg:  "sender domain spam.com is blocked",
		},
		{
			name: "allowed TLDs restriction",
			validator: &EmailValidator{
				MaxSize:     26214400,
				AllowedTLDs: []string{"com", "org"},
			},
			email: &emaildata.EmailData{
				Sender:    "sender@example.net",
				Recipient: "recipient@example.com",
			},
			wantErr: true,
			errMsg:  "sender TLD net is not allowed",
		},
		{
			name: "size limit exceeded",
			validator: &EmailValidator{
				MaxSize: 100,
			},
			email: &emaildata.EmailData{
				Sender:    "sender@example.com",
				Recipient: "recipient@example.com",
				Raw:       strings.Repeat("x", 101),
			},
			wantErr: true,
			errMsg:  "exceeds maximum allowed size",
		},
		{
			name: "SPF required but missing",
			validator: &EmailValidator{
				MaxSize:    26214400,
				RequireSPF: true,
			},
			email: &emaildata.EmailData{
				Sender:    "sender@example.com",
				Recipient: "recipient@example.com",
			},
			wantErr: true,
			errMsg:  "SPF validation required",
		},
		{
			name: "DKIM required but missing",
			validator: &EmailValidator{
				MaxSize:     26214400,
				RequireDKIM: true,
			},
			email: &emaildata.EmailData{
				Sender:    "sender@example.com",
				Recipient: "recipient@example.com",
			},
			wantErr: true,
			errMsg:  "DKIM signature required",
		},
		{
			name: "SPF present when required",
			validator: &EmailValidator{
				MaxSize:    26214400,
				RequireSPF: true,
			},
			email: &emaildata.EmailData{
				Sender:    "sender@example.com",
				Recipient: "recipient@example.com",
				Authentication: emaildata.AuthenticationMetadata{
					SPF: emaildata.SPFMetadata{
						ReceivedSPFHeader: "pass",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "DKIM present when required",
			validator: &EmailValidator{
				MaxSize:     26214400,
				RequireDKIM: true,
			},
			email: &emaildata.EmailData{
				Sender:    "sender@example.com",
				Recipient: "recipient@example.com",
				Authentication: emaildata.AuthenticationMetadata{
					DKIM: emaildata.DKIMMetadata{
						Signatures: []string{"v=1; a=rsa-sha256; d=example.com"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate(tt.email)
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

func TestEmailValidator_validateEmailAddress(t *testing.T) {
	validator := NewEmailValidator()

	tests := []struct {
		name    string
		address string
		field   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid email",
			address: "user@example.com",
			field:   "test",
			wantErr: false,
		},
		{
			name:    "valid email with display name",
			address: "John Doe <john@example.com>",
			field:   "test",
			wantErr: false,
		},
		{
			name:    "empty address",
			address: "",
			field:   "test",
			wantErr: true,
			errMsg:  "test address cannot be empty",
		},
		{
			name:    "missing @",
			address: "notanemail",
			field:   "test",
			wantErr: true,
			errMsg:  "invalid test email",
		},
		{
			name:    "empty domain",
			address: "user@",
			field:   "test",
			wantErr: true,
			errMsg:  "empty domain",
		},
		{
			name:    "domain too long",
			address: "user@" + strings.Repeat("a", 254) + ".com",
			field:   "test",
			wantErr: true,
			errMsg:  "domain too long",
		},
		{
			name:    "single label domain",
			address: "user@localhost",
			field:   "test",
			wantErr: true,
			errMsg:  "domain must have at least two labels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateEmailAddress(tt.address, tt.field)
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

func TestEmailValidator_validateDomain(t *testing.T) {
	validator := NewEmailValidator()

	tests := []struct {
		name    string
		domain  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid domain",
			domain:  "example.com",
			wantErr: false,
		},
		{
			name:    "valid subdomain",
			domain:  "mail.example.com",
			wantErr: false,
		},
		{
			name:    "domain with whitespace",
			domain:  "example .com",
			wantErr: true,
			errMsg:  "contains whitespace",
		},
		{
			name:    "domain too long",
			domain:  strings.Repeat("a", 254) + ".com",
			wantErr: true,
			errMsg:  "domain too long",
		},
		{
			name:    "single label",
			domain:  "localhost",
			wantErr: true,
			errMsg:  "at least two labels",
		},
		{
			name:    "empty label",
			domain:  "example..com",
			wantErr: true,
			errMsg:  "empty label",
		},
		{
			name:    "label too long",
			domain:  strings.Repeat("a", 64) + ".com",
			wantErr: true,
			errMsg:  "label too long",
		},
		{
			name:    "label starts with hyphen",
			domain:  "-example.com",
			wantErr: true,
			errMsg:  "cannot start or end with hyphen",
		},
		{
			name:    "label ends with hyphen",
			domain:  "example-.com",
			wantErr: true,
			errMsg:  "cannot start or end with hyphen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateDomain(tt.domain)
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

func TestExtractTLD(t *testing.T) {
	tests := []struct {
		email    string
		expected string
	}{
		{"user@example.com", "com"},
		{"user@example.co.uk", "uk"},
		{"user@subdomain.example.org", "org"},
		{"invalid", ""},
		{"user@localhost", ""},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := extractTLD(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeHeaders(t *testing.T) {
	input := map[string]string{
		"X-Original-Sender":    "sender@example.com",
		"X-Original-Recipient": "recipient@example.com\x00\x01",
		"X-Original-Helo":      "  example.com  ",
		"X-Malicious-Header":   "should be removed",
		"Content-Type":         "text/plain",
	}

	result := SanitizeHeaders(input)

	// Check allowed headers are present
	assert.Equal(t, "sender@example.com", result["X-Original-Sender"])
	assert.Equal(t, "recipient@example.com", result["X-Original-Recipient"])
	assert.Equal(t, "example.com", result["X-Original-Helo"])

	// Check disallowed headers are removed
	assert.Empty(t, result["X-Malicious-Header"])
	assert.Empty(t, result["Content-Type"])

	// Check control characters are removed
	assert.NotContains(t, result["X-Original-Recipient"], "\x00")
	assert.NotContains(t, result["X-Original-Recipient"], "\x01")
}

func TestSanitizeHeaderValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"  spaces  ", "spaces"},
		{"with\x00null", "withnull"},
		{"with\ttab", "withtab"},
		{"with\nnewline", "withnewline"},
		{"emoji 😀 test", "emoji  test"}, // Emoji removed as non-ASCII
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeHeaderValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSPF(t *testing.T) {
	tests := []struct {
		name     string
		clientIP string
		domain   string
		sender   string
		wantErr  bool
	}{
		{
			name:     "valid IP",
			clientIP: "192.168.1.1",
			domain:   "example.com",
			sender:   "sender@example.com",
			wantErr:  false,
		},
		{
			name:     "invalid IP",
			clientIP: "not-an-ip",
			domain:   "example.com",
			sender:   "sender@example.com",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSPF(tt.clientIP, tt.domain, tt.sender)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDKIM(t *testing.T) {
	tests := []struct {
		name       string
		signatures []string
		fromDomain string
		wantErr    bool
	}{
		{
			name:       "with signatures",
			signatures: []string{"v=1; a=rsa-sha256; d=example.com"},
			fromDomain: "example.com",
			wantErr:    false,
		},
		{
			name:       "no signatures",
			signatures: []string{},
			fromDomain: "example.com",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDKIM(tt.signatures, tt.fromDomain)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func BenchmarkEmailValidator_Validate(b *testing.B) {
	validator := NewEmailValidator()
	email := &emaildata.EmailData{
		Sender:     "sender@example.com",
		Recipient:  "recipient@example.com",
		Raw:        strings.Repeat("x", 1024),
		ReceivedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(email)
	}
}

func BenchmarkSanitizeHeaders(b *testing.B) {
	headers := map[string]string{
		"X-Original-Sender":    "sender@example.com",
		"X-Original-Recipient": "recipient@example.com",
		"X-Original-Helo":      "example.com",
		"X-Malicious":          "should be removed",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizeHeaders(headers)
	}
}

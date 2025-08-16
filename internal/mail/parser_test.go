package mail

import (
	"bufio"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRawEmail(t *testing.T) {
	tests := []struct {
		name        string
		rawEmail    string
		httpHeaders map[string]string
		validate    func(*testing.T, *EmailData)
	}{
		{
			name: "basic email",
			rawEmail: `From: sender@example.com
To: recipient@example.com
Subject: Test Subject
Message-ID: <123@example.com>
Date: Mon, 2 Jan 2006 15:04:05 -0700

This is the email body.`,
			httpHeaders: map[string]string{},
			validate: func(t *testing.T, data *EmailData) {
				assert.Equal(t, "sender@example.com", data.Sender)
				assert.Equal(t, "recipient@example.com", data.Recipient)
				assert.Equal(t, "Test Subject", data.Subject)
				assert.Equal(t, "<123@example.com>", data.MessageID)
				assert.NotEmpty(t, data.Raw)
			},
		},
		{
			name: "email with display names",
			rawEmail: `From: "John Doe" <john@example.com>
To: "Jane Smith" <jane@example.com>
Subject: Test with Names

Body text.`,
			httpHeaders: map[string]string{},
			validate: func(t *testing.T, data *EmailData) {
				assert.Equal(t, "john@example.com", data.Sender)
				assert.Equal(t, "jane@example.com", data.Recipient)
			},
		},
		{
			name: "email with HTTP header overrides",
			rawEmail: `From: original@example.com
To: original-to@example.com
Subject: Original Subject

Body.`,
			httpHeaders: map[string]string{
				"X-Original-Sender":    "override@example.com",
				"X-Original-Recipient": "override-to@example.com",
			},
			validate: func(t *testing.T, data *EmailData) {
				assert.Equal(t, "override@example.com", data.Sender)
				assert.Equal(t, "override-to@example.com", data.Recipient)
			},
		},
		{
			name: "email with connection info",
			rawEmail: `From: sender@example.com
To: recipient@example.com

Body.`,
			httpHeaders: map[string]string{
				"X-Original-Client-Address":  "192.168.1.1",
				"X-Original-Client-Hostname": "mail.example.com",
				"X-Original-Helo":            "example.com",
			},
			validate: func(t *testing.T, data *EmailData) {
				assert.Equal(t, "192.168.1.1", data.Connection.ClientAddress)
				assert.Equal(t, "mail.example.com", data.Connection.ClientHostname)
				assert.Equal(t, "example.com", data.Connection.ClientHelo)
			},
		},
		{
			name: "email with SPF metadata",
			rawEmail: `From: sender@example.com
To: recipient@example.com
Received-SPF: pass (example.com: domain of sender@example.com designates 192.168.1.1 as permitted sender)

Body.`,
			httpHeaders: map[string]string{
				"X-Original-Client-Address": "192.168.1.1",
				"X-Original-Mail-From":      "sender@example.com",
				"X-Original-Helo":           "mail.example.com",
			},
			validate: func(t *testing.T, data *EmailData) {
				assert.Equal(t, "192.168.1.1", data.Authentication.SPF.ClientIP)
				assert.Equal(t, "sender@example.com", data.Authentication.SPF.MailFrom)
				assert.Equal(t, "mail.example.com", data.Authentication.SPF.HeloDomain)
				assert.Contains(t, data.Authentication.SPF.ReceivedSPFHeader, "pass")
			},
		},
		{
			name: "email with DMARC metadata",
			rawEmail: `From: sender@example.com
To: recipient@example.com
Return-Path: <bounce@example.com>
Authentication-Results: mx.example.com; dmarc=pass

Body.`,
			httpHeaders: map[string]string{},
			validate: func(t *testing.T, data *EmailData) {
				assert.Equal(t, "sender@example.com", data.Authentication.DMARC.FromHeader)
				assert.Equal(t, "<bounce@example.com>", data.Authentication.DMARC.ReturnPath)
				assert.Contains(t, data.Authentication.DMARC.AuthenticationResults, "dmarc=pass")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ParseRawEmail(tt.rawEmail, tt.httpHeaders)
			require.NoError(t, err)
			require.NotNil(t, data)
			assert.WithinDuration(t, time.Now(), data.ReceivedAt, 1*time.Second)
			tt.validate(t, data)
		})
	}
}

func TestParseRawEmail_InvalidEmail(t *testing.T) {
	invalidEmail := "This is not a valid email format"
	data, err := ParseRawEmail(invalidEmail, nil)
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestExtractDKIMMetadata(t *testing.T) {
	tests := []struct {
		name     string
		headers  mail.Header
		validate func(*testing.T, DKIMMetadata)
	}{
		{
			name: "single DKIM signature",
			headers: mail.Header{
				"Dkim-Signature": []string{"v=1; a=rsa-sha256; d=example.com; s=selector; h=from:to:subject"},
				"From":           []string{"sender@example.com"},
			},
			validate: func(t *testing.T, dkim DKIMMetadata) {
				assert.Len(t, dkim.Signatures, 1)
				assert.Contains(t, dkim.Signatures[0], "d=example.com")
				assert.Equal(t, []string{"example.com"}, dkim.SignedBy)
				assert.Equal(t, "example.com", dkim.FromDomain)
			},
		},
		{
			name: "multiple DKIM signatures",
			headers: mail.Header{
				"Dkim-Signature": []string{
					"v=1; a=rsa-sha256; d=example.com; s=selector1",
					"v=1; a=rsa-sha256; d=relay.com; s=selector2",
				},
				"From": []string{"sender@example.com"},
			},
			validate: func(t *testing.T, dkim DKIMMetadata) {
				assert.Len(t, dkim.Signatures, 2)
				assert.Len(t, dkim.SignedBy, 2)
				assert.Contains(t, dkim.SignedBy, "example.com")
				assert.Contains(t, dkim.SignedBy, "relay.com")
			},
		},
		{
			name: "multi-line DKIM signature",
			headers: mail.Header{
				"Dkim-Signature": []string{"v=1; a=rsa-sha256;\r\n\td=example.com;\r\n\ts=selector"},
				"From":           []string{"sender@example.com"},
			},
			validate: func(t *testing.T, dkim DKIMMetadata) {
				assert.Len(t, dkim.Signatures, 1)
				// Should have spaces instead of line breaks
				assert.NotContains(t, dkim.Signatures[0], "\r\n")
				assert.Equal(t, []string{"example.com"}, dkim.SignedBy)
			},
		},
		{
			name:    "no DKIM signature",
			headers: mail.Header{"From": []string{"sender@example.com"}},
			validate: func(t *testing.T, dkim DKIMMetadata) {
				assert.Empty(t, dkim.Signatures)
				assert.Empty(t, dkim.SignedBy)
				assert.Equal(t, "example.com", dkim.FromDomain)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dkim := extractDKIMMetadata(tt.headers)
			tt.validate(t, dkim)
		})
	}
}

func TestExtractDKIMParam(t *testing.T) {
	tests := []struct {
		signature string
		param     string
		expected  string
	}{
		{
			signature: "v=1; a=rsa-sha256; d=example.com; s=selector",
			param:     "d=",
			expected:  "example.com",
		},
		{
			signature: "v=1; a=rsa-sha256; d=example.com",
			param:     "d=",
			expected:  "example.com",
		},
		{
			signature: "d=example.com; s=selector",
			param:     "s=",
			expected:  "selector",
		},
		{
			signature: "v=1; a=rsa-sha256",
			param:     "d=",
			expected:  "",
		},
		{
			signature: "d=example.com s=selector",
			param:     "d=",
			expected:  "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.signature, func(t *testing.T) {
			result := extractDKIMParam(tt.signature, tt.param)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		validate func(*testing.T, *EmailData)
	}{
		{
			name: "basic fields",
			data: map[string]interface{}{
				"sender":    "sender@example.com",
				"recipient": "recipient@example.com",
				"subject":   "Test Subject",
				"raw":       "raw email content",
			},
			validate: func(t *testing.T, email *EmailData) {
				assert.Equal(t, "sender@example.com", email.Sender)
				assert.Equal(t, "recipient@example.com", email.Recipient)
				assert.Equal(t, "Test Subject", email.Subject)
				assert.Equal(t, "raw email content", email.Raw)
			},
		},
		{
			name: "with connection info",
			data: map[string]interface{}{
				"sender": "sender@example.com",
				"connection": map[string]interface{}{
					"client_address":  "192.168.1.1",
					"client_hostname": "mail.example.com",
					"client_helo":     "example.com",
				},
			},
			validate: func(t *testing.T, email *EmailData) {
				assert.Equal(t, "192.168.1.1", email.Connection.ClientAddress)
				assert.Equal(t, "mail.example.com", email.Connection.ClientHostname)
				assert.Equal(t, "example.com", email.Connection.ClientHelo)
			},
		},
		{
			name: "with raw email parsing",
			data: map[string]interface{}{
				"sender": "override@example.com",
				"raw": `From: original@example.com
To: original-to@example.com
Subject: Original Subject
Message-ID: <123@example.com>

Body.`,
			},
			validate: func(t *testing.T, email *EmailData) {
				// JSON sender takes precedence
				assert.Equal(t, "override@example.com", email.Sender)
				// Subject from raw email
				assert.Equal(t, "Original Subject", email.Subject)
				assert.Equal(t, "<123@example.com>", email.MessageID)
			},
		},
		{
			name: "empty data",
			data: map[string]interface{}{},
			validate: func(t *testing.T, email *EmailData) {
				assert.Empty(t, email.Sender)
				assert.Empty(t, email.Recipient)
				assert.WithinDuration(t, time.Now(), email.ReceivedAt, 1*time.Second)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := FromJSON(tt.data)
			require.NotNil(t, email)
			tt.validate(t, email)
		})
	}
}

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(*testing.T, mail.Header)
	}{
		{
			name: "basic headers",
			input: `From: sender@example.com
To: recipient@example.com
Subject: Test Subject

`,
			validate: func(t *testing.T, headers mail.Header) {
				assert.Equal(t, "sender@example.com", headers.Get("From"))
				assert.Equal(t, "recipient@example.com", headers.Get("To"))
				assert.Equal(t, "Test Subject", headers.Get("Subject"))
			},
		},
		{
			name: "multi-line header",
			input: `Subject: This is a very long subject that
 continues on the next line
From: sender@example.com

`,
			validate: func(t *testing.T, headers mail.Header) {
				assert.Contains(t, headers.Get("Subject"), "continues on the next line")
				assert.Equal(t, "sender@example.com", headers.Get("From"))
			},
		},
		{
			name: "duplicate headers",
			input: `Received: from server1
Received: from server2
From: sender@example.com

`,
			validate: func(t *testing.T, headers mail.Header) {
				received := headers["Received"]
				assert.Len(t, received, 2)
				assert.Contains(t, received[0], "server1")
				assert.Contains(t, received[1], "server2")
			},
		},
		{
			name:  "empty input",
			input: "\n",
			validate: func(t *testing.T, headers mail.Header) {
				assert.Empty(t, headers)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			headers, err := ParseHeaders(reader)
			require.NoError(t, err)
			tt.validate(t, headers)
		})
	}
}

func BenchmarkParseRawEmail(b *testing.B) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Test Subject
Message-ID: <123@example.com>
Date: Mon, 2 Jan 2006 15:04:05 -0700
DKIM-Signature: v=1; a=rsa-sha256; d=example.com; s=selector; h=from:to:subject

This is the email body with some content.`

	httpHeaders := map[string]string{
		"X-Original-Client-Address": "192.168.1.1",
		"X-Original-Helo":           "mail.example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseRawEmail(rawEmail, httpHeaders)
	}
}

func BenchmarkFromJSON(b *testing.B) {
	data := map[string]interface{}{
		"sender":    "sender@example.com",
		"recipient": "recipient@example.com",
		"subject":   "Test Subject",
		"raw":       "raw email content",
		"connection": map[string]interface{}{
			"client_address": "192.168.1.1",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FromJSON(data)
	}
}

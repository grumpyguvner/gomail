package validation

import (
	"fmt"
	"net"
	"net/mail"
	"strings"

	emaildata "github.com/grumpyguvner/gomail/internal/mail"
)

// EmailValidator validates email data
type EmailValidator struct {
	MaxSize        int64
	AllowedTLDs    []string
	BlockedDomains []string
	RequireSPF     bool
	RequireDKIM    bool
}

// NewEmailValidator creates a new email validator with default settings
func NewEmailValidator() *EmailValidator {
	return &EmailValidator{
		MaxSize:        26214400,   // 25MB
		AllowedTLDs:    []string{}, // Empty means all TLDs allowed
		BlockedDomains: []string{},
		RequireSPF:     false,
		RequireDKIM:    false,
	}
}

// Validate validates email data
func (v *EmailValidator) Validate(email *emaildata.EmailData) error {
	// Validate sender
	if err := v.validateEmailAddress(email.Sender, "sender"); err != nil {
		return err
	}

	// Validate recipient
	if err := v.validateEmailAddress(email.Recipient, "recipient"); err != nil {
		return err
	}

	// Check blocked domains
	if err := v.checkBlockedDomains(email.Sender, email.Recipient); err != nil {
		return err
	}

	// Check allowed TLDs if configured
	if len(v.AllowedTLDs) > 0 {
		if err := v.checkAllowedTLDs(email.Sender, email.Recipient); err != nil {
			return err
		}
	}

	// Validate size
	if len(email.Raw) > int(v.MaxSize) {
		return fmt.Errorf("email size %d exceeds maximum allowed size %d", len(email.Raw), v.MaxSize)
	}

	// Validate SPF if required
	if v.RequireSPF && email.Authentication.SPF.ReceivedSPFHeader == "" {
		return fmt.Errorf("SPF validation required but no SPF header found")
	}

	// Validate DKIM if required
	if v.RequireDKIM && len(email.Authentication.DKIM.Signatures) == 0 {
		return fmt.Errorf("DKIM signature required but no signatures found")
	}

	return nil
}

// validateEmailAddress validates a single email address
func (v *EmailValidator) validateEmailAddress(address, field string) error {
	if address == "" {
		return fmt.Errorf("%s address cannot be empty", field)
	}

	// Parse the email address
	addr, err := mail.ParseAddress(address)
	if err != nil {
		// Try without display name
		if !strings.Contains(address, "@") {
			return fmt.Errorf("invalid %s email address: %s", field, address)
		}
		addr = &mail.Address{Address: address}
	}

	// Extract domain
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid %s email format: %s", field, address)
	}

	domain := parts[1]
	if domain == "" {
		return fmt.Errorf("empty domain in %s address: %s", field, address)
	}

	// Validate domain format
	if err := v.validateDomain(domain); err != nil {
		return fmt.Errorf("invalid domain in %s address %s: %w", field, address, err)
	}

	return nil
}

// validateDomain validates a domain name
func (v *EmailValidator) validateDomain(domain string) error {
	// Check for valid characters
	if strings.ContainsAny(domain, " \t\n\r") {
		return fmt.Errorf("domain contains whitespace: %s", domain)
	}

	// Check domain length
	if len(domain) > 253 {
		return fmt.Errorf("domain too long: %s", domain)
	}

	// Check each label
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return fmt.Errorf("domain must have at least two labels: %s", domain)
	}

	for _, label := range labels {
		if len(label) == 0 {
			return fmt.Errorf("empty label in domain: %s", domain)
		}
		if len(label) > 63 {
			return fmt.Errorf("label too long in domain: %s", domain)
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("label cannot start or end with hyphen: %s", domain)
		}
	}

	return nil
}

// checkBlockedDomains checks if sender or recipient is from a blocked domain
func (v *EmailValidator) checkBlockedDomains(sender, recipient string) error {
	for _, blocked := range v.BlockedDomains {
		if strings.Contains(sender, "@"+blocked) {
			return fmt.Errorf("sender domain %s is blocked", blocked)
		}
		if strings.Contains(recipient, "@"+blocked) {
			return fmt.Errorf("recipient domain %s is blocked", blocked)
		}
	}
	return nil
}

// checkAllowedTLDs checks if sender and recipient use allowed TLDs
func (v *EmailValidator) checkAllowedTLDs(sender, recipient string) error {
	senderTLD := extractTLD(sender)
	recipientTLD := extractTLD(recipient)

	if !v.isTLDAllowed(senderTLD) {
		return fmt.Errorf("sender TLD %s is not allowed", senderTLD)
	}

	if !v.isTLDAllowed(recipientTLD) {
		return fmt.Errorf("recipient TLD %s is not allowed", recipientTLD)
	}

	return nil
}

// extractTLD extracts the top-level domain from an email address
func extractTLD(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}

	domainParts := strings.Split(parts[1], ".")
	if len(domainParts) < 2 {
		return ""
	}

	return domainParts[len(domainParts)-1]
}

// isTLDAllowed checks if a TLD is in the allowed list
func (v *EmailValidator) isTLDAllowed(tld string) bool {
	for _, allowed := range v.AllowedTLDs {
		if strings.EqualFold(tld, allowed) {
			return true
		}
	}
	return false
}

// ValidateSPF validates SPF records
func ValidateSPF(clientIP, domain, sender string) error {
	// Parse the client IP
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return fmt.Errorf("invalid client IP: %s", clientIP)
	}

	// In a real implementation, we would:
	// 1. Look up the SPF record for the domain
	// 2. Check if the client IP is authorized
	// 3. Return the result

	// For now, this is a placeholder
	return nil
}

// ValidateDKIM validates DKIM signatures
func ValidateDKIM(signatures []string, fromDomain string) error {
	if len(signatures) == 0 {
		return fmt.Errorf("no DKIM signatures found")
	}

	// In a real implementation, we would:
	// 1. Parse each DKIM signature
	// 2. Retrieve the public key from DNS
	// 3. Verify the signature
	// 4. Check alignment with From domain

	// For now, this is a placeholder
	return nil
}

// SanitizeHeaders removes potentially dangerous headers
func SanitizeHeaders(headers map[string]string) map[string]string {
	sanitized := make(map[string]string)

	// List of headers that should be preserved
	allowedHeaders := []string{
		"X-Original-Sender",
		"X-Original-Recipient",
		"X-Original-Client-Address",
		"X-Original-Client-Hostname",
		"X-Original-Helo",
		"X-Original-Mail-From",
	}

	for _, header := range allowedHeaders {
		if value, ok := headers[header]; ok {
			// Remove any control characters
			sanitized[header] = sanitizeHeaderValue(value)
		}
	}

	return sanitized
}

// sanitizeHeaderValue removes control characters from header values
func sanitizeHeaderValue(value string) string {
	var result strings.Builder
	for _, r := range value {
		// Only allow printable characters and spaces
		if r >= 32 && r < 127 {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

package postfix

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/grumpyguvner/gomail/internal/config"
)

type Installer struct {
	config *config.Config
}

func NewInstaller(cfg *config.Config) *Installer {
	return &Installer{config: cfg}
}

func (i *Installer) Install() error {
	// Install Postfix package
	if err := i.installPackages(); err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}

	// Configure Postfix
	if err := i.configurePostfix(); err != nil {
		return fmt.Errorf("failed to configure Postfix: %w", err)
	}

	// Create pipe script
	if err := i.createPipeScript(); err != nil {
		return fmt.Errorf("failed to create pipe script: %w", err)
	}

	// Configure master.cf
	if err := i.configureMasterCF(); err != nil {
		return fmt.Errorf("failed to configure master.cf: %w", err)
	}

	// Enable and start Postfix
	if err := i.enablePostfix(); err != nil {
		return fmt.Errorf("failed to enable Postfix: %w", err)
	}

	return nil
}

func (i *Installer) installPackages() error {
	// Check if Postfix is already installed
	cmd := exec.Command("rpm", "-q", "postfix")
	if err := cmd.Run(); err == nil {
		return nil // Already installed
	}

	// Install Postfix and related packages
	cmd = exec.Command("dnf", "install", "-y", "postfix", "postfix-pcre", "cyrus-sasl-plain")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (i *Installer) configurePostfix() error {
	// Check if Postfix is already configured for our mail server
	cmd := exec.Command("postconf", "virtual_transport")
	output, _ := cmd.Output()
	if strings.Contains(string(output), "mailapi") {
		// Already configured, just update domains if needed
		return i.updateDomains()
	}

	// Set Postfix configuration parameters
	settings := map[string]string{
		"myhostname":                i.config.MailHostname,
		"mydomain":                  i.config.PrimaryDomain,
		"myorigin":                  "$mydomain",
		"inet_interfaces":           "all",
		"inet_protocols":            "ipv4",
		"mydestination":             "localhost",
		"local_recipient_maps":      "",
		"virtual_mailbox_domains":   i.config.PrimaryDomain,
		"virtual_mailbox_maps":      "regexp:/etc/postfix/virtual_mailbox_regex",
		"virtual_transport":         "mailapi:",
		"mailapi_destination_recipient_limit": "1",
		"message_size_limit":        "26214400",
		"mailbox_size_limit":        "0",
		"smtpd_banner":              "$myhostname ESMTP",
		"smtpd_relay_restrictions":  "permit_mynetworks,reject_unauth_destination",
		"smtpd_recipient_restrictions": "permit_mynetworks,reject_unauth_destination",
	}

	for key, value := range settings {
		cmd := exec.Command("postconf", "-e", fmt.Sprintf("%s=%s", key, value))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}

	// Create virtual mailbox regex file - catch-all for virtual domains
	// This will accept any email for domains listed in virtual_mailbox_domains
	regexContent := `# Catch-all regex for all virtual domains
/.+@.+/ mailapi:
`
	if err := os.WriteFile("/etc/postfix/virtual_mailbox_regex", []byte(regexContent), 0644); err != nil {
		return fmt.Errorf("failed to create virtual_mailbox_regex: %w", err)
	}

	// Create domains list file with primary domain
	if err := os.WriteFile("/etc/postfix/domains.list", []byte(i.config.PrimaryDomain+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to create domains.list: %w", err)
	}

	return nil
}

func (i *Installer) updateDomains() error {
	// Ensure primary domain is in domains list
	domains := []string{}
	
	// Read existing domains
	if content, err := os.ReadFile("/etc/postfix/domains.list"); err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			domain := strings.TrimSpace(line)
			if domain != "" && !strings.HasPrefix(domain, "#") {
				domains = append(domains, domain)
			}
		}
	}
	
	// Add primary domain if not present
	found := false
	for _, d := range domains {
		if d == i.config.PrimaryDomain {
			found = true
			break
		}
	}
	
	if !found {
		domains = append(domains, i.config.PrimaryDomain)
		
		// Update domains list file
		content := strings.Join(domains, "\n") + "\n"
		if err := os.WriteFile("/etc/postfix/domains.list", []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to update domains list: %w", err)
		}
		
		// Update Postfix configuration
		domainsStr := strings.Join(domains, " ")
		cmd := exec.Command("postconf", "-e", fmt.Sprintf("virtual_mailbox_domains=%s", domainsStr))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to update virtual domains: %w", err)
		}
	}
	
	return nil
}

func (i *Installer) createPipeScript() error {
	// Create postfix-to-api script
	scriptContent := fmt.Sprintf(`#!/bin/bash
# Postfix to API pipe script
# Forwards emails to the mail API service

set -euo pipefail

# Load environment from sysconfig if it exists
if [ -f /etc/sysconfig/mailserver ]; then
    source /etc/sysconfig/mailserver
fi

# Read environment variables with defaults
API_ENDPOINT="${API_ENDPOINT:-%s}"
API_BEARER_TOKEN="${API_BEARER_TOKEN:-%s}"

# Read the email from stdin
EMAIL_DATA=$(cat)

# Extract sender and recipient from Postfix environment
SENDER="${2:-unknown}"
RECIPIENT="${3:-unknown}"
CLIENT_ADDRESS="${CLIENT_ADDRESS:-}"
CLIENT_HOSTNAME="${CLIENT_NAME:-}"
CLIENT_HELO="${CLIENT_HELO:-}"

# Log for debugging
logger -t postfix-to-api "Processing email from $SENDER to $RECIPIENT"

# Send to API with authentication metadata in headers
response=$(curl -s -w "\n%%{http_code}" -X POST "$API_ENDPOINT" \
  -H "Authorization: Bearer $API_BEARER_TOKEN" \
  -H "Content-Type: message/rfc822" \
  -H "X-Original-Sender: $SENDER" \
  -H "X-Original-Recipient: $RECIPIENT" \
  -H "X-Original-Client-Address: $CLIENT_ADDRESS" \
  -H "X-Original-Client-Hostname: $CLIENT_HOSTNAME" \
  -H "X-Original-Helo: $CLIENT_HELO" \
  --data-binary "$EMAIL_DATA" \
  --max-time 30 \
  --retry 3 \
  --retry-delay 2)

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
    logger -t postfix-to-api "Successfully delivered to API (HTTP $http_code)"
    exit 0
else
    logger -t postfix-to-api "Failed to deliver to API (HTTP $http_code): $body"
    exit 75  # EX_TEMPFAIL - tells Postfix to retry
fi
`, i.config.APIEndpoint, i.config.BearerToken)

	scriptPath := "/usr/local/bin/postfix-to-api"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to create pipe script: %w", err)
	}

	// Set ownership
	cmd := exec.Command("chown", "root:root", scriptPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set script ownership: %w", err)
	}

	return nil
}

func (i *Installer) configureMasterCF() error {
	// Add mailapi transport to master.cf
	masterCFPath := "/etc/postfix/master.cf"
	
	// Read current master.cf
	content, err := os.ReadFile(masterCFPath)
	if err != nil {
		return fmt.Errorf("failed to read master.cf: %w", err)
	}

	// Check if mailapi transport already exists
	if strings.Contains(string(content), "mailapi") {
		// Check if it needs updating (e.g., if the path changed)
		if !strings.Contains(string(content), "/usr/local/bin/postfix-to-api") {
			// Remove old mailapi configuration and re-add
			lines := strings.Split(string(content), "\n")
			newLines := []string{}
			skipNext := false
			
			for _, line := range lines {
				if strings.HasPrefix(line, "mailapi") {
					skipNext = true
					continue
				}
				if skipNext && strings.HasPrefix(line, "  ") {
					continue
				}
				skipNext = false
				newLines = append(newLines, line)
			}
			
			content = []byte(strings.Join(newLines, "\n"))
		} else {
			return nil // Already properly configured
		}
	}

	// Append mailapi transport
	transportConfig := `
# Mail API transport
mailapi   unix  -       n       n       -       -       pipe
  flags=FR user=nobody argv=/usr/local/bin/postfix-to-api
  ${sender} ${recipient}
`

	if err := os.WriteFile(masterCFPath, append(content, []byte(transportConfig)...), 0644); err != nil {
		return fmt.Errorf("failed to update master.cf: %w", err)
	}

	return nil
}

func (i *Installer) enablePostfix() error {
	// Enable Postfix service
	cmd := exec.Command("systemctl", "enable", "postfix")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable Postfix: %w", err)
	}

	// Start or restart Postfix
	cmd = exec.Command("systemctl", "restart", "postfix")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start Postfix: %w", err)
	}

	return nil
}
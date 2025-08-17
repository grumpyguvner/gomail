package digitalocean

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Domain represents a DigitalOcean domain
type Domain struct {
	Name string `json:"name"`
	TTL  int    `json:"ttl"`
}

// DNSRecord represents a DNS record
type DNSRecord struct {
	ID       int    `json:"id,omitempty"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	Priority int    `json:"priority,omitempty"`
	Port     int    `json:"port,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
	Weight   int    `json:"weight,omitempty"`
	Flags    int    `json:"flags,omitempty"`
	Tag      string `json:"tag,omitempty"`
}

// CheckDomainExists checks if a domain exists in DigitalOcean
func (c *Client) CheckDomainExists(domain string) (bool, error) {
	_, err := c.doRequest("GET", "/domains/"+domain, nil)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateDomain creates a new domain in DigitalOcean
func (c *Client) CreateDomain(domain, ipAddress string) error {
	body := map[string]string{
		"name":       domain,
		"ip_address": ipAddress,
	}

	resp, err := c.doRequest("POST", "/domains", body)
	if err != nil {
		// Check if domain already exists
		if strings.Contains(err.Error(), "domain_exists") || strings.Contains(err.Error(), "already exists") {
			return nil // Domain already exists, not an error
		}
		return fmt.Errorf("failed to create domain: %w", err)
	}

	// Verify domain was created
	var result struct {
		Domain Domain `json:"domain"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

// GetDNSRecords retrieves all DNS records for a domain
func (c *Client) GetDNSRecords(domain string) ([]DNSRecord, error) {
	resp, err := c.doRequest("GET", "/domains/"+domain+"/records", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS records: %w", err)
	}

	var result struct {
		DomainRecords []DNSRecord `json:"domain_records"`
		Links         struct {
			Pages struct {
				Next string `json:"next"`
			} `json:"pages"`
		} `json:"links"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.DomainRecords, nil
}

// FindDNSRecord finds a specific DNS record
func (c *Client) FindDNSRecord(domain, recordType, name string) (*DNSRecord, error) {
	records, err := c.GetDNSRecords(domain)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		if record.Type == recordType && record.Name == name {
			return &record, nil
		}
	}

	return nil, nil
}

// CreateDNSRecord creates a new DNS record
func (c *Client) CreateDNSRecord(domain string, record DNSRecord) error {
	resp, err := c.doRequest("POST", "/domains/"+domain+"/records", record)
	if err != nil {
		return fmt.Errorf("failed to create DNS record: %w", err)
	}

	// Verify record was created
	var result struct {
		DomainRecord DNSRecord `json:"domain_record"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

// UpdateDNSRecord updates an existing DNS record
func (c *Client) UpdateDNSRecord(domain string, recordID int, record DNSRecord) error {
	path := fmt.Sprintf("/domains/%s/records/%d", domain, recordID)
	_, err := c.doRequest("PUT", path, record)
	if err != nil {
		return fmt.Errorf("failed to update DNS record: %w", err)
	}
	return nil
}

// UpsertDNSRecord creates or updates a DNS record
func (c *Client) UpsertDNSRecord(domain string, record DNSRecord) error {
	// Check if record exists
	existing, err := c.FindDNSRecord(domain, record.Type, record.Name)
	if err != nil {
		return err
	}

	if existing != nil {
		// Update existing record
		record.ID = existing.ID
		return c.UpdateDNSRecord(domain, existing.ID, record)
	}

	// Create new record
	return c.CreateDNSRecord(domain, record)
}

// SetupMailDNS sets up all required DNS records for mail server
func (c *Client) SetupMailDNS(domain, mailHostname, serverIP string) error {
	// Ensure domain exists
	exists, err := c.CheckDomainExists(domain)
	if err != nil {
		return fmt.Errorf("failed to check domain: %w", err)
	}

	if !exists {
		if err := c.CreateDomain(domain, serverIP); err != nil {
			return fmt.Errorf("failed to create domain: %w", err)
		}
	} else {
		// Clean up any legacy A records pointing to this IP
		// This helps avoid conflicts when the droplet has been renamed
		if err := c.CleanupLegacyARecords(domain, serverIP); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to cleanup legacy records: %v\n", err)
		}
	}

	// Extract mail subdomain (e.g., "mail" from "mail.example.com")
	// This assumes mailHostname is in the same domain
	var mailSubdomain string
	if strings.HasSuffix(mailHostname, "."+domain) {
		mailSubdomain = strings.TrimSuffix(mailHostname, "."+domain)
	} else {
		// If mail hostname is in a different domain, we need to handle it differently
		// For now, we'll create an A record for the full hostname
		mailSubdomain = mailHostname
	}

	// Create/Update A record for mail server
	if mailSubdomain != mailHostname {
		// Mail server is a subdomain of this domain
		aRecord := DNSRecord{
			Type: "A",
			Name: mailSubdomain,
			Data: serverIP,
			TTL:  3600,
		}
		if err := c.UpsertDNSRecord(domain, aRecord); err != nil {
			return fmt.Errorf("failed to create A record: %w", err)
		}
	}

	// Create/Update MX record
	mxRecord := DNSRecord{
		Type:     "MX",
		Name:     "@",
		Data:     mailHostname + ".",
		Priority: 10,
		TTL:      3600,
	}
	if err := c.UpsertDNSRecord(domain, mxRecord); err != nil {
		return fmt.Errorf("failed to create MX record: %w", err)
	}

	// Create/Update SPF record
	spfRecord := DNSRecord{
		Type: "TXT",
		Name: "@",
		Data: fmt.Sprintf("v=spf1 mx a:%s ~all", mailHostname),
		TTL:  3600,
	}
	if err := c.UpsertDNSRecord(domain, spfRecord); err != nil {
		return fmt.Errorf("failed to create SPF record: %w", err)
	}

	// Create/Update DMARC record
	dmarcRecord := DNSRecord{
		Type: "TXT",
		Name: "_dmarc",
		Data: fmt.Sprintf("v=DMARC1; p=none; rua=mailto:postmaster@%s", domain),
		TTL:  3600,
	}
	if err := c.UpsertDNSRecord(domain, dmarcRecord); err != nil {
		return fmt.Errorf("failed to create DMARC record: %w", err)
	}

	return nil
}

// DeleteDNSRecord deletes a DNS record
func (c *Client) DeleteDNSRecord(domain string, recordID int) error {
	path := fmt.Sprintf("/domains/%s/records/%d", domain, recordID)
	_, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete DNS record: %w", err)
	}
	return nil
}

// CleanupLegacyARecords removes any A records pointing to the given IP
func (c *Client) CleanupLegacyARecords(domain, serverIP string) error {
	records, err := c.GetDNSRecords(domain)
	if err != nil {
		return fmt.Errorf("failed to get DNS records: %w", err)
	}

	for _, record := range records {
		// Delete any A records that point to our IP but aren't for our mail hostname
		if record.Type == "A" && record.Data == serverIP {
			// We'll keep the record if it's for the mail subdomain
			// Otherwise, delete it as it's likely a legacy record
			if record.Name == "@" || strings.HasPrefix(record.Name, "mail") {
				continue // Keep mail-related A records
			}

			if err := c.DeleteDNSRecord(domain, record.ID); err != nil {
				// Log but don't fail the whole operation
				fmt.Printf("Warning: failed to delete legacy A record %s: %v\n", record.Name, err)
			}
		}
	}

	return nil
}

// SetupInfraDNS sets up DNS for the infrastructure domain (where mail server hostname lives)
func (c *Client) SetupInfraDNS(infraDomain, mailHostname, serverIP string) error {
	// Ensure domain exists
	exists, err := c.CheckDomainExists(infraDomain)
	if err != nil {
		return fmt.Errorf("failed to check domain: %w", err)
	}

	if !exists {
		if err := c.CreateDomain(infraDomain, serverIP); err != nil {
			return fmt.Errorf("failed to create domain: %w", err)
		}
	} else {
		// Clean up any legacy A records pointing to this IP
		if err := c.CleanupLegacyARecords(infraDomain, serverIP); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to cleanup legacy records: %v\n", err)
		}
	}

	// Extract mail subdomain (e.g., "mail" from "mail.example.com")
	if !strings.HasSuffix(mailHostname, "."+infraDomain) {
		return fmt.Errorf("mail hostname %s is not in infrastructure domain %s", mailHostname, infraDomain)
	}

	mailSubdomain := strings.TrimSuffix(mailHostname, "."+infraDomain)

	// Create/Update A record for mail server
	aRecord := DNSRecord{
		Type: "A",
		Name: mailSubdomain,
		Data: serverIP,
		TTL:  3600,
	}
	if err := c.UpsertDNSRecord(infraDomain, aRecord); err != nil {
		return fmt.Errorf("failed to create A record for mail hostname: %w", err)
	}

	return nil
}

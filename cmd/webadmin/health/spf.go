package health

import (
	"net"
	"regexp"
	"strings"

	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
)

type SPFChecker struct {
	logger *logging.Logger
}

func NewSPFChecker(logger *logging.Logger) *SPFChecker {
	return &SPFChecker{logger: logger}
}

func (c *SPFChecker) Check(domain string) SPFHealth {
	health := SPFHealth{
		Status:   "healthy",
		Record:   "",
		Valid:    false,
		Issues:   []string{},
		Includes: []string{},
		Score:    100,
	}

	// Look up TXT records
	txtRecords, err := net.LookupTXT(domain)
	if err != nil {
		health.Issues = append(health.Issues, "Failed to lookup TXT records: "+err.Error())
		health.Status = "error"
		health.Score = 0
		return health
	}

	// Find SPF record
	var spfRecord string
	spfCount := 0
	for _, record := range txtRecords {
		if strings.HasPrefix(record, "v=spf1") {
			spfCount++
			if spfRecord == "" {
				spfRecord = record
			}
		}
	}

	if spfCount == 0 {
		health.Issues = append(health.Issues, "No SPF record found")
		health.Status = "error"
		health.Score = 0
		return health
	}

	if spfCount > 1 {
		health.Issues = append(health.Issues, "Multiple SPF records found (RFC violation)")
		health.Status = "error"
		health.Score = 20
	}

	health.Record = spfRecord
	health.Valid = true

	// Parse SPF record
	c.parseSPFRecord(spfRecord, &health)

	// Update status based on score
	if health.Score >= 80 && health.Status != "error" {
		health.Status = "healthy"
	} else if health.Score >= 50 && health.Status != "error" {
		health.Status = "warning"
	} else if health.Status != "error" {
		health.Status = "warning"
	}

	c.logger.Debug("SPF check completed",
		"domain", domain,
		"status", health.Status,
		"score", health.Score,
		"valid", health.Valid,
		"includes", len(health.Includes),
		"issues", len(health.Issues),
	)

	return health
}

func (c *SPFChecker) parseSPFRecord(record string, health *SPFHealth) {
	// Check for common SPF syntax issues
	if !strings.HasPrefix(record, "v=spf1") {
		health.Issues = append(health.Issues, "SPF record does not start with v=spf1")
		health.Score -= 50
		return
	}

	// Split record into mechanisms
	parts := strings.Fields(record)
	hasAll := false
	
	for _, part := range parts[1:] { // Skip "v=spf1"
		// Check for include mechanisms
		if strings.HasPrefix(part, "include:") {
			includeDomain := strings.TrimPrefix(part, "include:")
			health.Includes = append(health.Includes, includeDomain)
			
			// Validate included domain has SPF record
			c.validateIncludedDomain(includeDomain, health)
		}
		
		// Check for all mechanism
		if strings.HasPrefix(part, "all") || strings.HasPrefix(part, "-all") || 
		   strings.HasPrefix(part, "~all") || strings.HasPrefix(part, "+all") {
			hasAll = true
			
			// Check all mechanism policy
			switch part {
			case "-all":
				// Strict policy - good
			case "~all":
				// Soft fail - okay but could be stricter
				health.Issues = append(health.Issues, "Using soft fail (~all) instead of hard fail (-all)")
				health.Score -= 5
			case "+all":
				// Pass all - very bad
				health.Issues = append(health.Issues, "Using +all allows any server to send email (security risk)")
				health.Score -= 30
			case "all":
				// Neutral - bad
				health.Issues = append(health.Issues, "Using neutral all without qualifier")
				health.Score -= 20
			}
		}
		
		// Check for IP4/IP6 mechanisms
		if strings.HasPrefix(part, "ip4:") || strings.HasPrefix(part, "ip6:") {
			c.validateIPMechanism(part, health)
		}
		
		// Check for a/mx mechanisms
		if strings.HasPrefix(part, "a:") || strings.HasPrefix(part, "mx:") || part == "a" || part == "mx" {
			c.validateDomainMechanism(part, health)
		}
	}
	
	if !hasAll {
		health.Issues = append(health.Issues, "SPF record missing all mechanism")
		health.Score -= 20
	}
	
	// Check for too many DNS lookups (RFC limit is 10)
	dnsLookups := len(health.Includes)
	for _, part := range parts {
		if strings.HasPrefix(part, "a") || strings.HasPrefix(part, "mx") || 
		   strings.HasPrefix(part, "exists:") {
			dnsLookups++
		}
	}
	
	if dnsLookups > 10 {
		health.Issues = append(health.Issues, "SPF record exceeds 10 DNS lookup limit")
		health.Score -= 25
	} else if dnsLookups > 8 {
		health.Issues = append(health.Issues, "SPF record close to 10 DNS lookup limit")
		health.Score -= 10
	}
	
	// Check record length (recommended under 255 characters)
	if len(record) > 255 {
		health.Issues = append(health.Issues, "SPF record exceeds recommended 255 character limit")
		health.Score -= 10
	}
}

func (c *SPFChecker) validateIncludedDomain(domain string, health *SPFHealth) {
	// Try to resolve the included domain's SPF record
	txtRecords, err := net.LookupTXT(domain)
	if err != nil {
		health.Issues = append(health.Issues, "Cannot resolve included domain: "+domain)
		health.Score -= 15
		return
	}
	
	hasSpf := false
	for _, record := range txtRecords {
		if strings.HasPrefix(record, "v=spf1") {
			hasSpf = true
			break
		}
	}
	
	if !hasSpf {
		health.Issues = append(health.Issues, "Included domain has no SPF record: "+domain)
		health.Score -= 20
	}
}

func (c *SPFChecker) validateIPMechanism(mechanism string, health *SPFHealth) {
	// Extract IP address
	parts := strings.Split(mechanism, ":")
	if len(parts) != 2 {
		health.Issues = append(health.Issues, "Invalid IP mechanism format: "+mechanism)
		health.Score -= 10
		return
	}
	
	// Parse IP address
	ipStr := parts[1]
	
	// Handle CIDR notation
	if strings.Contains(ipStr, "/") {
		_, _, err := net.ParseCIDR(ipStr)
		if err != nil {
			health.Issues = append(health.Issues, "Invalid CIDR notation: "+ipStr)
			health.Score -= 10
		}
	} else {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			health.Issues = append(health.Issues, "Invalid IP address: "+ipStr)
			health.Score -= 10
		}
	}
}

func (c *SPFChecker) validateDomainMechanism(mechanism string, health *SPFHealth) {
	// For a: and mx: mechanisms, validate the domain resolves
	var domain string
	
	if mechanism == "a" || mechanism == "mx" {
		// These use the current domain
		return
	}
	
	if strings.HasPrefix(mechanism, "a:") {
		domain = strings.TrimPrefix(mechanism, "a:")
	} else if strings.HasPrefix(mechanism, "mx:") {
		domain = strings.TrimPrefix(mechanism, "mx:")
	}
	
	if domain != "" {
		// Check if domain is valid format
		domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
		if !domainRegex.MatchString(domain) {
			health.Issues = append(health.Issues, "Invalid domain format in mechanism: "+mechanism)
			health.Score -= 10
		}
	}
}
package health

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
)

type DeliverabilityChecker struct {
	logger *logging.Logger
}

func NewDeliverabilityChecker(logger *logging.Logger) *DeliverabilityChecker {
	return &DeliverabilityChecker{logger: logger}
}

func (c *DeliverabilityChecker) Check(domain string) DeliverabilityHealth {
	health := DeliverabilityHealth{
		Status:      "healthy",
		Score:       100,
		Blacklisted: false,
		Blacklists:  []string{},
		Reputation:  "good",
		Issues:      []string{},
	}

	// Get IP addresses for the domain
	ips, err := net.LookupHost(domain)
	if err != nil {
		health.Issues = append(health.Issues, "Failed to resolve domain IP addresses")
		health.Status = "error"
		health.Score = 0
		return health
	}

	if len(ips) == 0 {
		health.Issues = append(health.Issues, "No IP addresses found for domain")
		health.Status = "error"
		health.Score = 0
		return health
	}

	// Check each IP against blacklists
	for _, ip := range ips {
		c.checkIPBlacklist(ip, &health)
	}

	// Perform additional deliverability checks
	c.checkDomainReputation(domain, &health)
	c.checkMXRecordDeliverability(domain, &health)

	// Update overall status based on findings
	if health.Blacklisted {
		health.Status = "error"
		health.Reputation = "poor"
		health.Score = 0
	} else if len(health.Issues) > 0 {
		health.Status = "warning"
		if health.Score > 50 {
			health.Reputation = "fair"
		} else {
			health.Reputation = "poor"
		}
	}

	// Ensure score doesn't go below 0
	if health.Score < 0 {
		health.Score = 0
	}

	c.logger.Debug("Deliverability check completed",
		"domain", domain,
		"status", health.Status,
		"score", health.Score,
		"blacklisted", health.Blacklisted,
		"reputation", health.Reputation,
		"blacklists", len(health.Blacklists),
		"issues", len(health.Issues),
	)

	return health
}

func (c *DeliverabilityChecker) checkIPBlacklist(ip string, health *DeliverabilityHealth) {
	// Common blacklist services to check
	blacklists := []string{
		"zen.spamhaus.org",
		"b.barracudacentral.org",
		"bl.spamcop.net",
		"blacklist.woody.ch",
		"combined.abuse.ch",
		"db.wpbl.info",
		"ips.backscatterer.org",
		"ix.dnsbl.manitu.net",
		"korea.services.net",
		"psbl.surriel.com",
		"relays.nether.net",
		"singular.ttk.pte.hu",
		"ubl.unsubscore.com",
		"virus.rbl.jp",
	}

	for _, blacklist := range blacklists {
		if c.isIPBlacklisted(ip, blacklist) {
			health.Blacklisted = true
			health.Blacklists = append(health.Blacklists, blacklist)
			health.Issues = append(health.Issues, "IP "+ip+" is blacklisted on "+blacklist)
			health.Score -= 20 // Each blacklist reduces score significantly
		}
	}
}

func (c *DeliverabilityChecker) isIPBlacklisted(ip, blacklist string) bool {
	// Reverse the IP address for DNS blacklist lookup
	reversedIP := c.reverseIP(ip)
	if reversedIP == "" {
		return false
	}

	// Construct blacklist query
	query := reversedIP + "." + blacklist

	// Perform DNS lookup with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a custom resolver with timeout
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 3 * time.Second,
			}
			return d.DialContext(ctx, network, address)
		},
	}

	_, err := resolver.LookupHost(ctx, query)

	// If the lookup succeeds, the IP is blacklisted
	return err == nil
}

func (c *DeliverabilityChecker) reverseIP(ip string) string {
	// Only handle IPv4 for now
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ""
	}

	// Reverse the octets
	return parts[3] + "." + parts[2] + "." + parts[1] + "." + parts[0]
}

func (c *DeliverabilityChecker) checkDomainReputation(domain string, health *DeliverabilityHealth) {
	// Check if domain is suspicious based on various factors

	// Check domain age (approximate)
	if c.isDomainSuspicious(domain) {
		health.Issues = append(health.Issues, "Domain may have reputation issues")
		health.Score -= 15
	}

	// Check for common spam domain patterns
	suspiciousPatterns := []string{
		"temp",
		"disposable",
		"mailinator",
		"guerrillamail",
		"10minutemail",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(domain), pattern) {
			health.Issues = append(health.Issues, "Domain contains suspicious patterns")
			health.Score -= 25
			break
		}
	}
}

func (c *DeliverabilityChecker) isDomainSuspicious(domain string) bool {
	// Simple checks for suspicious domains

	// Very short domains (excluding common TLDs)
	if len(domain) < 6 && !strings.HasSuffix(domain, ".com") &&
		!strings.HasSuffix(domain, ".org") && !strings.HasSuffix(domain, ".net") {
		return true
	}

	// Domains with many numbers
	numbers := 0
	for _, char := range domain {
		if char >= '0' && char <= '9' {
			numbers++
		}
	}

	if numbers > len(domain)/3 {
		return true
	}

	// Domains with excessive hyphens
	hyphens := strings.Count(domain, "-")
	return hyphens > 3
}

func (c *DeliverabilityChecker) checkMXRecordDeliverability(domain string, health *DeliverabilityHealth) {
	// Check MX records for deliverability issues
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		health.Issues = append(health.Issues, "Failed to lookup MX records")
		health.Score -= 30
		return
	}

	if len(mxRecords) == 0 {
		health.Issues = append(health.Issues, "No MX records found")
		health.Score -= 40
		return
	}

	// Check MX record priorities
	priorities := make(map[uint16]bool)
	for _, mx := range mxRecords {
		if priorities[mx.Pref] {
			health.Issues = append(health.Issues, "Duplicate MX record priorities found")
			health.Score -= 10
			break
		}
		priorities[mx.Pref] = true
	}

	// Check if MX records point to valid hosts
	for _, mx := range mxRecords {
		mxHost := strings.TrimSuffix(mx.Host, ".")

		// Check if MX host resolves
		_, err := net.LookupHost(mxHost)
		if err != nil {
			health.Issues = append(health.Issues, "MX record points to unresolvable host: "+mxHost)
			health.Score -= 20
		}

		// Check for common free email providers (which might affect reputation)
		freeProviders := []string{
			"gmail.com",
			"outlook.com",
			"yahoo.com",
			"hotmail.com",
		}

		for _, provider := range freeProviders {
			if strings.Contains(strings.ToLower(mxHost), provider) {
				health.Issues = append(health.Issues, "MX record points to free email provider")
				health.Score -= 5
				break
			}
		}
	}
}

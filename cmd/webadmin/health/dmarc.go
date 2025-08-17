package health

import (
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
)

type DMARCChecker struct {
	logger *logging.Logger
}

func NewDMARCChecker(logger *logging.Logger) *DMARCChecker {
	return &DMARCChecker{logger: logger}
}

func (c *DMARCChecker) Check(domain string) DMARCHealth {
	health := DMARCHealth{
		Status:  "healthy",
		Record:  "",
		Policy:  "",
		Percent: 100,
		Valid:   false,
		Issues:  []string{},
		Score:   100,
	}

	// Look up DMARC record at _dmarc.domain
	dmarcDomain := "_dmarc." + domain
	txtRecords, err := net.LookupTXT(dmarcDomain)
	if err != nil {
		health.Issues = append(health.Issues, "Failed to lookup DMARC record: "+err.Error())
		health.Status = "error"
		health.Score = 0
		return health
	}

	// Find DMARC record
	var dmarcRecord string
	dmarcCount := 0
	for _, record := range txtRecords {
		if strings.HasPrefix(record, "v=DMARC1") {
			dmarcCount++
			if dmarcRecord == "" {
				dmarcRecord = record
			}
		}
	}

	if dmarcCount == 0 {
		health.Issues = append(health.Issues, "No DMARC record found")
		health.Status = "error"
		health.Score = 0
		return health
	}

	if dmarcCount > 1 {
		health.Issues = append(health.Issues, "Multiple DMARC records found (RFC violation)")
		health.Status = "error"
		health.Score = 20
	}

	health.Record = dmarcRecord
	health.Valid = true

	// Parse DMARC record
	c.parseDMARCRecord(dmarcRecord, &health)

	// Update status based on score
	if health.Score >= 80 && health.Status != "error" {
		health.Status = "healthy"
	} else if health.Score >= 50 && health.Status != "error" {
		health.Status = "warning"
	} else if health.Status != "error" {
		health.Status = "warning"
	}

	c.logger.Debug("DMARC check completed",
		"domain", domain,
		"status", health.Status,
		"score", health.Score,
		"policy", health.Policy,
		"percent", health.Percent,
		"valid", health.Valid,
		"issues", len(health.Issues),
	)

	return health
}

func (c *DMARCChecker) parseDMARCRecord(record string, health *DMARCHealth) {
	// Check version
	if !strings.HasPrefix(record, "v=DMARC1") {
		health.Issues = append(health.Issues, "DMARC record does not start with v=DMARC1")
		health.Score -= 50
		return
	}

	// Parse key-value pairs
	params := make(map[string]string)
	
	// Split by semicolon and parse key=value pairs
	parts := strings.Split(record, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				params[key] = value
			}
		}
	}

	// Check required policy (p=)
	policy, exists := params["p"]
	if !exists {
		health.Issues = append(health.Issues, "DMARC record missing required policy (p=)")
		health.Score -= 60
		return
	}

	health.Policy = policy

	// Validate policy value
	switch policy {
	case "none":
		health.Issues = append(health.Issues, "DMARC policy is set to 'none' (monitoring only)")
		health.Score -= 30
	case "quarantine":
		// Good policy
	case "reject":
		// Best policy
		health.Score += 10
	default:
		health.Issues = append(health.Issues, "Invalid DMARC policy: "+policy)
		health.Score -= 40
	}

	// Check percentage (pct=)
	if pctStr, exists := params["pct"]; exists {
		if pct, err := strconv.Atoi(pctStr); err == nil {
			health.Percent = pct
			if pct < 100 {
				health.Issues = append(health.Issues, "DMARC policy applies to less than 100% of messages")
				health.Score -= (100 - pct) / 4 // Reduce score based on percentage
			}
		} else {
			health.Issues = append(health.Issues, "Invalid DMARC percentage value: "+pctStr)
			health.Score -= 10
		}
	}

	// Check subdomain policy (sp=)
	if sp, exists := params["sp"]; exists {
		switch sp {
		case "none":
			health.Issues = append(health.Issues, "Subdomain policy is set to 'none'")
			health.Score -= 10
		case "quarantine", "reject":
			// Good
		default:
			health.Issues = append(health.Issues, "Invalid subdomain policy: "+sp)
			health.Score -= 10
		}
	} else {
		health.Issues = append(health.Issues, "No subdomain policy specified (inherits main policy)")
		health.Score -= 5
	}

	// Check alignment modes
	if aspf, exists := params["aspf"]; exists {
		switch aspf {
		case "r":
			// Relaxed mode - default and okay
		case "s":
			// Strict mode - better
			health.Score += 5
		default:
			health.Issues = append(health.Issues, "Invalid SPF alignment mode: "+aspf)
			health.Score -= 5
		}
	}

	if adkim, exists := params["adkim"]; exists {
		switch adkim {
		case "r":
			// Relaxed mode - default and okay
		case "s":
			// Strict mode - better
			health.Score += 5
		default:
			health.Issues = append(health.Issues, "Invalid DKIM alignment mode: "+adkim)
			health.Score -= 5
		}
	}

	// Check reporting URIs
	if rua, exists := params["rua"]; exists {
		c.validateReportingURI(rua, "aggregate", health)
	} else {
		health.Issues = append(health.Issues, "No aggregate reporting URI specified")
		health.Score -= 15
	}

	if ruf, exists := params["ruf"]; exists {
		c.validateReportingURI(ruf, "forensic", health)
	}

	// Check failure reporting options
	if fo, exists := params["fo"]; exists {
		validFo := regexp.MustCompile(`^[01ds:]+$`)
		if !validFo.MatchString(fo) {
			health.Issues = append(health.Issues, "Invalid failure reporting option: "+fo)
			health.Score -= 5
		}
	}

	// Check report interval
	if ri, exists := params["ri"]; exists {
		if interval, err := strconv.Atoi(ri); err == nil {
			if interval < 86400 { // Less than 1 day
				health.Issues = append(health.Issues, "Report interval is very frequent (may cause high email volume)")
				health.Score -= 5
			}
		} else {
			health.Issues = append(health.Issues, "Invalid report interval: "+ri)
			health.Score -= 5
		}
	}

	// Ensure score doesn't go below 0 or above 100
	if health.Score < 0 {
		health.Score = 0
	}
	if health.Score > 100 {
		health.Score = 100
	}
}

func (c *DMARCChecker) validateReportingURI(uri, reportType string, health *DMARCHealth) {
	// Split multiple URIs
	uris := strings.Split(uri, ",")
	
	for _, u := range uris {
		u = strings.TrimSpace(u)
		
		// Check URI format
		if !strings.HasPrefix(u, "mailto:") {
			health.Issues = append(health.Issues, "Invalid "+reportType+" reporting URI format: "+u)
			health.Score -= 10
			continue
		}
		
		// Extract email address
		email := strings.TrimPrefix(u, "mailto:")
		
		// Basic email validation
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(email) {
			health.Issues = append(health.Issues, "Invalid email in "+reportType+" reporting URI: "+email)
			health.Score -= 10
		}
	}
}
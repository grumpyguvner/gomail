package health

import (
	"net"
	"strings"

	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
)

type DNSChecker struct {
	logger *logging.Logger
}

func NewDNSChecker(logger *logging.Logger) *DNSChecker {
	return &DNSChecker{logger: logger}
}

func (c *DNSChecker) Check(domain string) DNSHealth {
	health := DNSHealth{
		Status:    "healthy",
		ARecords:  []string{},
		MXRecords: []string{},
		PTRRecord: "",
		Issues:    []string{},
		Score:     100,
	}

	// Check A records
	aRecords, err := net.LookupHost(domain)
	if err != nil {
		health.Issues = append(health.Issues, "Failed to resolve A records: "+err.Error())
		health.Status = "error"
		health.Score = 0
	} else {
		health.ARecords = aRecords
		if len(aRecords) == 0 {
			health.Issues = append(health.Issues, "No A records found")
			health.Status = "warning"
			health.Score -= 30
		}
	}

	// Check MX records
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		health.Issues = append(health.Issues, "Failed to resolve MX records: "+err.Error())
		health.Status = "error"
		health.Score -= 40
	} else {
		for _, mx := range mxRecords {
			health.MXRecords = append(health.MXRecords, mx.Host)
		}
		if len(mxRecords) == 0 {
			health.Issues = append(health.Issues, "No MX records found")
			health.Status = "warning"
			health.Score -= 40
		}
	}

	// Check PTR record (reverse DNS) for the first A record
	if len(health.ARecords) > 0 {
		ptrRecords, err := net.LookupAddr(health.ARecords[0])
		if err != nil {
			health.Issues = append(health.Issues, "Failed to resolve PTR record: "+err.Error())
			health.Score -= 20
		} else if len(ptrRecords) > 0 {
			health.PTRRecord = ptrRecords[0]
			
			// Check if PTR record matches domain
			if !strings.Contains(health.PTRRecord, domain) {
				health.Issues = append(health.Issues, "PTR record does not match domain")
				health.Score -= 10
			}
		} else {
			health.Issues = append(health.Issues, "No PTR record found")
			health.Score -= 20
		}
	}

	// Validate MX records point to valid hosts
	for _, mxHost := range health.MXRecords {
		// Remove trailing dot
		mxHost = strings.TrimSuffix(mxHost, ".")
		
		_, err := net.LookupHost(mxHost)
		if err != nil {
			health.Issues = append(health.Issues, "MX record points to invalid host: "+mxHost)
			health.Score -= 15
		}
	}

	// Ensure score doesn't go below 0
	if health.Score < 0 {
		health.Score = 0
	}

	// Update status based on score
	if health.Score >= 80 && health.Status != "error" {
		health.Status = "healthy"
	} else if health.Score >= 50 && health.Status != "error" {
		health.Status = "warning"
	} else if health.Status != "error" {
		health.Status = "warning"
	}

	c.logger.Debug("DNS check completed", 
		"domain", domain,
		"status", health.Status,
		"score", health.Score,
		"a_records", len(health.ARecords),
		"mx_records", len(health.MXRecords),
		"issues", len(health.Issues),
	)

	return health
}
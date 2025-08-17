package auth

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/emersion/go-msgauth/authres"
	"github.com/emersion/go-msgauth/dmarc"
	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/grumpyguvner/gomail/internal/metrics"
	"go.uber.org/zap"
)

// DMARCVerifier handles DMARC policy enforcement
type DMARCVerifier struct {
	logger *zap.SugaredLogger
}

// NewDMARCVerifier creates a new DMARC verifier
func NewDMARCVerifier() *DMARCVerifier {
	return &DMARCVerifier{
		logger: logging.Get(),
	}
}

// DMARCResult represents the result of DMARC verification
type DMARCResult struct {
	Result        authres.ResultValue
	Domain        string
	Policy        dmarc.Policy
	SPFAlignment  bool
	DKIMAlignment bool
	Reason        string
}

// Verify performs DMARC policy evaluation
func (v *DMARCVerifier) Verify(ctx context.Context, fromDomain string, spfResult *SPFResult, dkimResults []*DKIMResult) (*DMARCResult, error) {
	// Clean up domain
	fromDomain = strings.ToLower(strings.TrimSpace(fromDomain))
	if fromDomain == "" {
		return &DMARCResult{
			Result: authres.ResultNone,
			Reason: "No From domain",
		}, nil
	}

	v.logger.Debugf("Checking DMARC for domain: %s", fromDomain)

	// Lookup DMARC record
	record, err := dmarc.Lookup(fromDomain)
	if err != nil {
		// Check if it's a "no record" error
		if strings.Contains(err.Error(), "no DMARC record") || strings.Contains(err.Error(), "not found") {
			metrics.DMARCNone.Inc()
			return &DMARCResult{
				Result: authres.ResultNone,
				Domain: fromDomain,
				Reason: "No DMARC record found",
			}, nil
		}

		metrics.DMARCLookupErrors.Inc()
		return &DMARCResult{
			Result: authres.ResultTempError,
			Domain: fromDomain,
			Reason: fmt.Sprintf("DMARC lookup failed: %v", err),
		}, err
	}

	// Check alignment
	spfAligned := v.checkSPFAlignment(fromDomain, spfResult, record)
	dkimAligned := v.checkDKIMAlignment(fromDomain, dkimResults, record)

	result := &DMARCResult{
		Domain:        fromDomain,
		Policy:        record.Policy,
		SPFAlignment:  spfAligned,
		DKIMAlignment: dkimAligned,
	}

	// Evaluate DMARC result based on alignment
	if spfAligned || dkimAligned {
		result.Result = authres.ResultPass
		result.Reason = "DMARC pass (aligned)"
		metrics.DMARCPass.Inc()
		v.logger.Infof("DMARC pass: domain=%s, spf_aligned=%v, dkim_aligned=%v",
			fromDomain, spfAligned, dkimAligned)
	} else {
		result.Result = authres.ResultFail
		result.Reason = fmt.Sprintf("DMARC fail (policy=%s)", record.Policy)
		metrics.DMARCFail.Inc()
		v.logger.Warnf("DMARC fail: domain=%s, policy=%s", fromDomain, record.Policy)
	}

	return result, nil
}

// checkSPFAlignment checks if SPF result aligns with DMARC
func (v *DMARCVerifier) checkSPFAlignment(fromDomain string, spfResult *SPFResult, record *dmarc.Record) bool {
	if spfResult == nil || spfResult.Result != authres.ResultPass {
		return false
	}

	// Check alignment mode
	if record.SPFAlignment == dmarc.AlignmentStrict {
		// Strict alignment: domains must match exactly
		return strings.EqualFold(fromDomain, spfResult.Domain)
	}

	// Relaxed alignment: organizational domains must match
	return v.organizationalDomainsMatch(fromDomain, spfResult.Domain)
}

// checkDKIMAlignment checks if any DKIM result aligns with DMARC
func (v *DMARCVerifier) checkDKIMAlignment(fromDomain string, dkimResults []*DKIMResult, record *dmarc.Record) bool {
	for _, dkimResult := range dkimResults {
		if dkimResult.Result != authres.ResultPass {
			continue
		}

		// Check alignment mode
		if record.DKIMAlignment == dmarc.AlignmentStrict {
			// Strict alignment: domains must match exactly
			if strings.EqualFold(fromDomain, dkimResult.Domain) {
				return true
			}
		} else {
			// Relaxed alignment: organizational domains must match
			if v.organizationalDomainsMatch(fromDomain, dkimResult.Domain) {
				return true
			}
		}
	}

	return false
}

// organizationalDomainsMatch checks if two domains share the same organizational domain
func (v *DMARCVerifier) organizationalDomainsMatch(domain1, domain2 string) bool {
	// Simple implementation: check if one is a subdomain of the other
	// or if they share the same base domain

	// Extract organizational domains (simplified)
	org1 := v.getOrganizationalDomain(domain1)
	org2 := v.getOrganizationalDomain(domain2)

	return strings.EqualFold(org1, org2)
}

// getOrganizationalDomain extracts the organizational domain
func (v *DMARCVerifier) getOrganizationalDomain(domain string) string {
	// Simplified implementation
	// In production, this should use the Public Suffix List
	parts := strings.Split(domain, ".")

	// Handle common TLDs
	if len(parts) >= 2 {
		// Check for common two-part TLDs
		tld := parts[len(parts)-1]
		sld := parts[len(parts)-2]

		// Common two-part TLDs
		twoPartTLDs := map[string]bool{
			"co.uk": true, "co.jp": true, "co.kr": true,
			"com.au": true, "com.br": true, "com.cn": true,
			"net.au": true, "org.uk": true, "ac.uk": true,
		}

		if len(parts) >= 3 && twoPartTLDs[sld+"."+tld] {
			// Return the last 3 parts for two-part TLDs
			return strings.Join(parts[len(parts)-3:], ".")
		}

		// Return the last 2 parts for regular TLDs
		return strings.Join(parts[len(parts)-2:], ".")
	}

	return domain
}

// FormatAuthenticationResult formats DMARC result for Authentication-Results header
func (r *DMARCResult) FormatAuthenticationResult() string {
	if r.Domain != "" {
		return fmt.Sprintf("dmarc=%s header.from=%s", r.Result, r.Domain)
	}
	return fmt.Sprintf("dmarc=%s", r.Result)
}

// GetPolicy returns the recommended action based on DMARC policy
func (r *DMARCResult) GetPolicy() string {
	if r.Result != authres.ResultFail {
		return "none"
	}

	switch r.Policy {
	case dmarc.PolicyReject:
		return "reject"
	case dmarc.PolicyQuarantine:
		return "quarantine"
	default:
		return "none"
	}
}

// DMARCReporter handles DMARC aggregate reporting
type DMARCReporter struct {
	logger *zap.SugaredLogger
	domain string
}

// NewDMARCReporter creates a new DMARC reporter
func NewDMARCReporter(domain string) *DMARCReporter {
	return &DMARCReporter{
		logger: logging.Get(),
		domain: domain,
	}
}

// RecordResult records a DMARC verification result for reporting
func (r *DMARCReporter) RecordResult(result *DMARCResult, sourceIP net.IP) {
	// In a full implementation, this would:
	// 1. Store results in a database
	// 2. Generate aggregate reports periodically
	// 3. Send reports to addresses specified in DMARC records

	// For now, just log it
	r.logger.Debugf("DMARC result recorded: domain=%s, result=%s, ip=%s",
		result.Domain, result.Result, sourceIP.String())

	// Update metrics for reporting
	if result.Result == authres.ResultPass {
		metrics.DMARCReportPass.Inc()
	} else {
		metrics.DMARCReportFail.Inc()
	}
}

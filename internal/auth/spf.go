package auth

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/emersion/go-msgauth/authres"
	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/grumpyguvner/gomail/internal/metrics"
	"go.uber.org/zap"
)

// SPFVerifier handles SPF verification for incoming mail
type SPFVerifier struct {
	logger *zap.SugaredLogger
}

// NewSPFVerifier creates a new SPF verifier
func NewSPFVerifier() *SPFVerifier {
	return &SPFVerifier{
		logger: logging.Get(),
	}
}

// VerifyResult represents the result of SPF verification
type SPFResult struct {
	Result authres.ResultValue
	Domain string
	IP     string
	Reason string
}

// Verify performs SPF verification
func (v *SPFVerifier) Verify(ctx context.Context, ip net.IP, heloHost, mailFrom string) (*SPFResult, error) {
	// Extract domain from mail from address
	domain := extractDomain(mailFrom)
	if domain == "" {
		return &SPFResult{
			Result: authres.ResultNone,
			Domain: heloHost,
			IP:     ip.String(),
			Reason: "No domain found in MAIL FROM",
		}, nil
	}

	v.logger.Debugf("Checking SPF for domain=%s, ip=%s", domain, ip.String())

	// Lookup SPF record
	spfRecord, err := lookupSPF(domain)
	if err != nil {
		metrics.SPFLookupErrors.Inc()
		return &SPFResult{
			Result: authres.ResultTempError,
			Domain: domain,
			IP:     ip.String(),
			Reason: fmt.Sprintf("SPF lookup failed: %v", err),
		}, err
	}

	if spfRecord == "" {
		metrics.SPFNone.Inc()
		return &SPFResult{
			Result: authres.ResultNone,
			Domain: domain,
			IP:     ip.String(),
			Reason: "No SPF record found",
		}, nil
	}

	// Check IP against SPF record
	result := checkSPF(spfRecord, ip, domain)

	// Update metrics
	switch result.Result {
	case authres.ResultPass:
		metrics.SPFPass.Inc()
	case authres.ResultFail:
		metrics.SPFFail.Inc()
	case authres.ResultSoftFail:
		metrics.SPFSoftFail.Inc()
	case authres.ResultNeutral:
		metrics.SPFNeutral.Inc()
	}

	v.logger.Infof("SPF verification: domain=%s, ip=%s, result=%s",
		domain, ip.String(), result.Result)

	return result, nil
}

// lookupSPF retrieves the SPF record for a domain
func lookupSPF(domain string) (string, error) {
	// Look up TXT records
	records, err := net.LookupTXT(domain)
	if err != nil {
		return "", fmt.Errorf("TXT lookup failed: %w", err)
	}

	// Find SPF record
	for _, record := range records {
		if strings.HasPrefix(record, "v=spf1") {
			return record, nil
		}
	}

	return "", nil
}

// checkSPF evaluates an IP against an SPF record
func checkSPF(spfRecord string, ip net.IP, domain string) *SPFResult {
	// Parse SPF record
	parts := strings.Fields(spfRecord)

	for _, part := range parts[1:] { // Skip v=spf1
		// Handle all mechanism
		if part == "all" || part == "+all" {
			return &SPFResult{
				Result: authres.ResultPass,
				Domain: domain,
				IP:     ip.String(),
				Reason: "Matched +all",
			}
		}
		if part == "-all" {
			// Check if IP matched any previous mechanism
			// If we get here, it didn't match
			return &SPFResult{
				Result: authres.ResultFail,
				Domain: domain,
				IP:     ip.String(),
				Reason: "Failed -all",
			}
		}
		if part == "~all" {
			return &SPFResult{
				Result: authres.ResultSoftFail,
				Domain: domain,
				IP:     ip.String(),
				Reason: "SoftFail ~all",
			}
		}
		if part == "?all" {
			return &SPFResult{
				Result: authres.ResultNeutral,
				Domain: domain,
				IP:     ip.String(),
				Reason: "Neutral ?all",
			}
		}

		// Handle IP4 mechanism
		if strings.HasPrefix(part, "ip4:") || strings.HasPrefix(part, "+ip4:") {
			ipStr := strings.TrimPrefix(strings.TrimPrefix(part, "+"), "ip4:")
			if checkIP4(ipStr, ip) {
				return &SPFResult{
					Result: authres.ResultPass,
					Domain: domain,
					IP:     ip.String(),
					Reason: fmt.Sprintf("Matched %s", part),
				}
			}
		}
		if strings.HasPrefix(part, "-ip4:") {
			ipStr := strings.TrimPrefix(part, "-ip4:")
			if checkIP4(ipStr, ip) {
				return &SPFResult{
					Result: authres.ResultFail,
					Domain: domain,
					IP:     ip.String(),
					Reason: fmt.Sprintf("Failed %s", part),
				}
			}
		}

		// Handle IP6 mechanism
		if strings.HasPrefix(part, "ip6:") || strings.HasPrefix(part, "+ip6:") {
			ipStr := strings.TrimPrefix(strings.TrimPrefix(part, "+"), "ip6:")
			if checkIP6(ipStr, ip) {
				return &SPFResult{
					Result: authres.ResultPass,
					Domain: domain,
					IP:     ip.String(),
					Reason: fmt.Sprintf("Matched %s", part),
				}
			}
		}

		// Handle MX mechanism
		if part == "mx" || part == "+mx" {
			if checkMX(domain, ip) {
				return &SPFResult{
					Result: authres.ResultPass,
					Domain: domain,
					IP:     ip.String(),
					Reason: "Matched MX",
				}
			}
		}

		// Handle A mechanism
		if part == "a" || part == "+a" {
			if checkA(domain, ip) {
				return &SPFResult{
					Result: authres.ResultPass,
					Domain: domain,
					IP:     ip.String(),
					Reason: "Matched A",
				}
			}
		}

		// Handle include mechanism
		if strings.HasPrefix(part, "include:") {
			includeDomain := strings.TrimPrefix(part, "include:")
			includeRecord, err := lookupSPF(includeDomain)
			if err == nil && includeRecord != "" {
				result := checkSPF(includeRecord, ip, includeDomain)
				if result.Result == authres.ResultPass {
					return &SPFResult{
						Result: authres.ResultPass,
						Domain: domain,
						IP:     ip.String(),
						Reason: fmt.Sprintf("Matched include:%s", includeDomain),
					}
				}
			}
		}
	}

	// Default to neutral if no mechanism matched
	return &SPFResult{
		Result: authres.ResultNeutral,
		Domain: domain,
		IP:     ip.String(),
		Reason: "No mechanism matched",
	}
}

// checkIP4 checks if an IP matches an IPv4 specification
func checkIP4(spec string, ip net.IP) bool {
	// Handle CIDR notation
	if strings.Contains(spec, "/") {
		_, network, err := net.ParseCIDR(spec)
		if err != nil {
			return false
		}
		return network.Contains(ip)
	}

	// Handle single IP
	specIP := net.ParseIP(spec)
	if specIP == nil {
		return false
	}
	return specIP.Equal(ip)
}

// checkIP6 checks if an IP matches an IPv6 specification
func checkIP6(spec string, ip net.IP) bool {
	// Handle CIDR notation
	if strings.Contains(spec, "/") {
		_, network, err := net.ParseCIDR(spec)
		if err != nil {
			return false
		}
		return network.Contains(ip)
	}

	// Handle single IP
	specIP := net.ParseIP(spec)
	if specIP == nil {
		return false
	}
	return specIP.Equal(ip)
}

// checkMX checks if IP matches any MX record for the domain
func checkMX(domain string, ip net.IP) bool {
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return false
	}

	for _, mx := range mxRecords {
		addrs, err := net.LookupHost(mx.Host)
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if net.ParseIP(addr).Equal(ip) {
				return true
			}
		}
	}

	return false
}

// checkA checks if IP matches A/AAAA records for the domain
func checkA(domain string, ip net.IP) bool {
	addrs, err := net.LookupHost(domain)
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		if net.ParseIP(addr).Equal(ip) {
			return true
		}
	}

	return false
}

// extractDomain extracts the domain from an email address
func extractDomain(email string) string {
	// Handle empty sender (bounce messages)
	if email == "" || email == "<>" {
		return ""
	}

	// Remove angle brackets if present
	email = strings.Trim(email, "<>")

	// Extract domain part
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}

	return strings.ToLower(parts[1])
}

// FormatAuthenticationResult formats SPF result for Authentication-Results header
func (r *SPFResult) FormatAuthenticationResult() string {
	return fmt.Sprintf("spf=%s smtp.mailfrom=%s", r.Result, r.Domain)
}

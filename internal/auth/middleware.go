package auth

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/mail"
	"os"
	"strings"

	"github.com/emersion/go-msgauth/authres"
	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/grumpyguvner/gomail/internal/metrics"
	"go.uber.org/zap"
)

// Middleware provides email authentication middleware
type Middleware struct {
	config        *config.Config
	spfVerifier   *SPFVerifier
	dkimVerifier  *DKIMVerifier
	dmarcVerifier *DMARCVerifier
	dkimSigner    *DKIMSigner
	reporter      *DMARCReporter
	logger        *zap.SugaredLogger
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(cfg *config.Config) (*Middleware, error) {
	m := &Middleware{
		config:        cfg,
		spfVerifier:   NewSPFVerifier(),
		dkimVerifier:  NewDKIMVerifier(),
		dmarcVerifier: NewDMARCVerifier(),
		reporter:      NewDMARCReporter(cfg.PrimaryDomain),
		logger:        logging.Get(),
	}

	// Initialize DKIM signer if configured
	if cfg.DKIMEnabled {
		signer, err := m.initDKIMSigner(cfg)
		if err != nil {
			m.logger.Warnf("DKIM signing disabled: %v", err)
		} else {
			m.dkimSigner = signer
			m.logger.Info("DKIM signing enabled")
		}
	}

	return m, nil
}

// initDKIMSigner initializes the DKIM signer
func (m *Middleware) initDKIMSigner(cfg *config.Config) (*DKIMSigner, error) {
	// Read private key from configured path
	privateKeyPath := cfg.DKIMPrivateKeyPath
	if privateKeyPath == "" {
		privateKeyPath = "/etc/mailserver/dkim/private.key"
	}

	privateKeyPEM, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read DKIM private key: %w", err)
	}

	selector := cfg.DKIMSelector
	if selector == "" {
		selector = "default"
	}

	return NewDKIMSigner(cfg.PrimaryDomain, selector, privateKeyPEM)
}

// AuthenticationResult contains all authentication results
type AuthenticationResult struct {
	SPF    *SPFResult
	DKIM   []*DKIMResult
	DMARC  *DMARCResult
	Pass   bool
	Action string // "accept", "quarantine", "reject"
}

// VerifyInbound performs authentication checks on incoming mail
func (m *Middleware) VerifyInbound(ctx context.Context, sourceIP net.IP, heloHost string, mailFrom string, message []byte) (*AuthenticationResult, error) {
	result := &AuthenticationResult{
		Action: "accept", // Default action
	}

	// Parse message headers to get From domain
	msg, err := mail.ReadMessage(bytes.NewReader(message))
	if err != nil {
		m.logger.Warnf("Failed to parse message headers: %v", err)
		return result, nil // Don't reject on parse errors
	}

	fromHeader := msg.Header.Get("From")
	fromDomain := extractDomainFromHeader(fromHeader)

	// 1. SPF Verification
	if m.config.SPFEnabled {
		spfResult, err := m.spfVerifier.Verify(ctx, sourceIP, heloHost, mailFrom)
		if err != nil {
			m.logger.Warnf("SPF verification error: %v", err)
		}
		result.SPF = spfResult
	}

	// 2. DKIM Verification
	if m.config.DKIMEnabled {
		dkimResults, err := m.dkimVerifier.Verify(ctx, message)
		if err != nil {
			m.logger.Warnf("DKIM verification error: %v", err)
		}
		result.DKIM = dkimResults
	}

	// 3. DMARC Verification
	if m.config.DMARCEnabled && fromDomain != "" {
		dmarcResult, err := m.dmarcVerifier.Verify(ctx, fromDomain, result.SPF, result.DKIM)
		if err != nil {
			m.logger.Warnf("DMARC verification error: %v", err)
		}
		result.DMARC = dmarcResult

		// Record for DMARC reporting
		if m.reporter != nil {
			m.reporter.RecordResult(dmarcResult, sourceIP)
		}

		// Determine action based on DMARC policy
		if dmarcResult != nil && dmarcResult.Result == authres.ResultFail {
			switch dmarcResult.GetPolicy() {
			case "reject":
				if m.config.DMARCEnforcement == "strict" {
					result.Action = "reject"
					result.Pass = false
					metrics.EmailsRejected.WithLabelValues("dmarc").Inc()
				} else {
					m.logger.Warnf("DMARC reject policy not enforced (enforcement=%s)",
						m.config.DMARCEnforcement)
				}
			case "quarantine":
				if m.config.DMARCEnforcement != "none" {
					result.Action = "quarantine"
					metrics.EmailsQuarantined.WithLabelValues("dmarc").Inc()
				}
			}
		}
	}

	// Overall pass determination
	result.Pass = m.determineOverallPass(result)

	// Log authentication summary
	m.logAuthenticationSummary(result, sourceIP, fromDomain)

	return result, nil
}

// determineOverallPass determines if the message passes authentication
func (m *Middleware) determineOverallPass(result *AuthenticationResult) bool {
	// DMARC pass supersedes individual SPF/DKIM results
	if result.DMARC != nil && result.DMARC.Result == authres.ResultPass {
		return true
	}

	// If no DMARC, check SPF and DKIM
	spfPass := result.SPF != nil && result.SPF.Result == authres.ResultPass
	dkimPass := false
	for _, dkim := range result.DKIM {
		if dkim.Result == authres.ResultPass {
			dkimPass = true
			break
		}
	}

	// Pass if either SPF or DKIM passes
	return spfPass || dkimPass
}

// logAuthenticationSummary logs a summary of authentication results
func (m *Middleware) logAuthenticationSummary(result *AuthenticationResult, sourceIP net.IP, fromDomain string) {
	var parts []string

	if result.SPF != nil {
		parts = append(parts, fmt.Sprintf("SPF=%s", result.SPF.Result))
	}

	if len(result.DKIM) > 0 {
		dkimParts := make([]string, 0, len(result.DKIM))
		for _, d := range result.DKIM {
			dkimParts = append(dkimParts, string(d.Result))
		}
		parts = append(parts, fmt.Sprintf("DKIM=[%s]", strings.Join(dkimParts, ",")))
	}

	if result.DMARC != nil {
		parts = append(parts, fmt.Sprintf("DMARC=%s", result.DMARC.Result))
	}

	m.logger.Infof("Authentication: ip=%s, from=%s, %s, action=%s",
		sourceIP.String(), fromDomain, strings.Join(parts, ", "), result.Action)
}

// SignOutbound adds DKIM signature to outgoing mail
func (m *Middleware) SignOutbound(ctx context.Context, message []byte) ([]byte, error) {
	if m.dkimSigner == nil {
		return message, nil // DKIM signing not configured
	}

	signedMessage, err := m.dkimSigner.Sign(message)
	if err != nil {
		m.logger.Errorf("DKIM signing failed: %v", err)
		return message, err
	}

	m.logger.Debug("Message signed with DKIM")
	return signedMessage, nil
}

// FormatAuthenticationResults formats all results for Authentication-Results header
func (m *Middleware) FormatAuthenticationResults(result *AuthenticationResult, hostname string) string {
	var parts []string

	// Add SPF result
	if result.SPF != nil {
		parts = append(parts, result.SPF.FormatAuthenticationResult())
	}

	// Add DKIM results
	if len(result.DKIM) > 0 {
		dkimParts := FormatDKIMResults(result.DKIM)
		parts = append(parts, dkimParts...)
	}

	// Add DMARC result
	if result.DMARC != nil {
		parts = append(parts, result.DMARC.FormatAuthenticationResult())
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%s; none", hostname)
	}

	return fmt.Sprintf("%s; %s", hostname, strings.Join(parts, "; "))
}

// extractDomainFromHeader extracts domain from a From header
func extractDomainFromHeader(from string) string {
	// Parse the From header
	addr, err := mail.ParseAddress(from)
	if err != nil {
		// Try to extract domain from raw string
		if idx := strings.LastIndex(from, "@"); idx > 0 {
			domain := from[idx+1:]
			// Clean up domain
			domain = strings.Trim(domain, " <>")
			return strings.ToLower(domain)
		}
		return ""
	}

	// Extract domain from parsed address
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 {
		return ""
	}

	return strings.ToLower(parts[1])
}

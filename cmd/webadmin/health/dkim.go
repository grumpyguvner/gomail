package health

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net"
	"strconv"
	"strings"

	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
)

type DKIMChecker struct {
	logger *logging.Logger
}

func NewDKIMChecker(logger *logging.Logger) *DKIMChecker {
	return &DKIMChecker{logger: logger}
}

func (c *DKIMChecker) Check(domain string) DKIMHealth {
	health := DKIMHealth{
		Status:    "healthy",
		Selectors: []DKIMSelector{},
		Issues:    []string{},
		Score:     100,
	}

	// Common DKIM selectors to check
	commonSelectors := []string{
		"default",
		"mail",
		"dkim",
		"k1",
		"s1",
		"selector1",
		"selector2",
		"google",
		"gmail",
	}

	foundSelectors := 0

	for _, selector := range commonSelectors {
		dkimSelector := c.checkDKIMSelector(domain, selector)
		if dkimSelector.Valid {
			health.Selectors = append(health.Selectors, dkimSelector)
			foundSelectors++
		}
	}

	if foundSelectors == 0 {
		health.Issues = append(health.Issues, "No DKIM records found with common selectors")
		health.Status = "error"
		health.Score = 0
	} else {
		// Validate each selector
		for _, selector := range health.Selectors {
			if !selector.Valid {
				health.Issues = append(health.Issues, "Invalid DKIM record for selector: "+selector.Selector)
				health.Score -= 20
			} else if selector.KeyType != "rsa" {
				health.Issues = append(health.Issues, "Non-RSA key type for selector "+selector.Selector+": "+selector.KeyType)
				health.Score -= 10
			} else if selector.KeySize < 1024 {
				health.Issues = append(health.Issues, "Key size too small for selector "+selector.Selector+": "+strconv.Itoa(selector.KeySize))
				health.Score -= 30
			} else if selector.KeySize < 2048 {
				health.Issues = append(health.Issues, "Key size recommended to be 2048+ bits for selector "+selector.Selector)
				health.Score -= 10
			}
		}

		// Bonus points for multiple selectors
		if foundSelectors > 1 {
			health.Score += 5
		}
	}

	// Ensure score doesn't go below 0 or above 100
	if health.Score < 0 {
		health.Score = 0
	}
	if health.Score > 100 {
		health.Score = 100
	}

	// Update status based on score
	if health.Score >= 80 && health.Status != "error" {
		health.Status = "healthy"
	} else if health.Score >= 50 && health.Status != "error" {
		health.Status = "warning"
	} else if health.Status != "error" {
		health.Status = "warning"
	}

	c.logger.Debug("DKIM check completed",
		"domain", domain,
		"status", health.Status,
		"score", health.Score,
		"selectors_found", foundSelectors,
		"issues", len(health.Issues),
	)

	return health
}

func (c *DKIMChecker) checkDKIMSelector(domain, selector string) DKIMSelector {
	dkimSelector := DKIMSelector{
		Selector: selector,
		Record:   "",
		Valid:    false,
		KeyType:  "",
		KeySize:  0,
	}

	// Construct DKIM DNS query
	dkimDomain := selector + "._domainkey." + domain

	// Look up TXT record
	txtRecords, err := net.LookupTXT(dkimDomain)
	if err != nil {
		c.logger.Debug("DKIM selector not found", "domain", domain, "selector", selector, "error", err)
		return dkimSelector
	}

	// Find DKIM record
	var dkimRecord string
	for _, record := range txtRecords {
		if strings.Contains(record, "k=") && strings.Contains(record, "p=") {
			dkimRecord = record
			break
		}
	}

	if dkimRecord == "" {
		return dkimSelector
	}

	dkimSelector.Record = dkimRecord

	// Parse DKIM record
	c.parseDKIMRecord(dkimRecord, &dkimSelector)

	return dkimSelector
}

func (c *DKIMChecker) parseDKIMRecord(record string, selector *DKIMSelector) {
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

	// Check version (should be DKIM1)
	if v, exists := params["v"]; exists && v != "DKIM1" {
		c.logger.Debug("Invalid DKIM version", "version", v, "selector", selector.Selector)
		return
	}

	// Get key type
	if k, exists := params["k"]; exists {
		selector.KeyType = k
	} else {
		selector.KeyType = "rsa" // Default
	}

	// Get public key
	publicKeyData, exists := params["p"]
	if !exists || publicKeyData == "" {
		c.logger.Debug("No public key in DKIM record", "selector", selector.Selector)
		return
	}

	// Remove whitespace from public key
	publicKeyData = strings.ReplaceAll(publicKeyData, " ", "")
	publicKeyData = strings.ReplaceAll(publicKeyData, "\t", "")
	publicKeyData = strings.ReplaceAll(publicKeyData, "\n", "")
	publicKeyData = strings.ReplaceAll(publicKeyData, "\r", "")

	// Decode base64 public key
	keyBytes, err := base64.StdEncoding.DecodeString(publicKeyData)
	if err != nil {
		c.logger.Debug("Failed to decode DKIM public key", "error", err, "selector", selector.Selector)
		return
	}

	// Parse public key to get key size
	if selector.KeyType == "rsa" {
		keySize, err := c.parseRSAKeySize(keyBytes)
		if err != nil {
			c.logger.Debug("Failed to parse RSA key", "error", err, "selector", selector.Selector)
			return
		}
		selector.KeySize = keySize
	}

	// Check for service type restrictions
	if s, exists := params["s"]; exists && s != "*" && s != "email" {
		c.logger.Debug("DKIM key has service restrictions", "service", s, "selector", selector.Selector)
	}

	// Check for flags
	if t, exists := params["t"]; exists {
		flags := strings.Split(t, ":")
		for _, flag := range flags {
			switch flag {
			case "y":
				c.logger.Debug("DKIM key is in testing mode", "selector", selector.Selector)
			case "s":
				c.logger.Debug("DKIM key requires strict subdomain matching", "selector", selector.Selector)
			}
		}
	}

	selector.Valid = true
}

func (c *DKIMChecker) parseRSAKeySize(keyBytes []byte) (int, error) {
	// Try to parse as PKIX format first
	publicKey, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		// Try to parse as PEM format
		block, _ := pem.Decode(keyBytes)
		if block != nil {
			publicKey, err = x509.ParsePKIXPublicKey(block.Bytes)
		}
		if err != nil {
			return 0, err
		}
	}

	// Check if it's an RSA key
	rsaKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return 0, err
	}

	return rsaKey.N.BitLen(), nil
}

package health

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
)

type SSLChecker struct {
	logger *logging.Logger
}

func NewSSLChecker(logger *logging.Logger) *SSLChecker {
	return &SSLChecker{logger: logger}
}

func (c *SSLChecker) Check(domain string) SSLHealth {
	health := SSLHealth{
		Status:   "healthy",
		Valid:    false,
		Expiry:   time.Time{},
		DaysLeft: 0,
		Issuer:   "",
		Issues:   []string{},
		Score:    100,
	}

	// Common mail ports to check SSL
	ports := []string{"465", "587", "993", "995"}

	for _, port := range ports {
		cert, err := c.getSSLCertificate(domain, port)
		if err != nil {
			c.logger.Debug("Failed to get SSL certificate", "domain", domain, "port", port, "error", err)
			continue
		}

		if cert != nil {
			health.Valid = true
			health.Expiry = cert.NotAfter
			health.DaysLeft = int(time.Until(cert.NotAfter).Hours() / 24)

			if len(cert.Issuer.Organization) > 0 {
				health.Issuer = cert.Issuer.Organization[0]
			} else {
				health.Issuer = cert.Issuer.CommonName
			}

			// Validate certificate
			c.validateCertificate(cert, domain, &health)
			break // Use first valid certificate found
		}
	}

	if !health.Valid {
		health.Issues = append(health.Issues, "No valid SSL certificate found on mail ports")
		health.Status = "error"
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

	c.logger.Debug("SSL check completed",
		"domain", domain,
		"status", health.Status,
		"score", health.Score,
		"valid", health.Valid,
		"days_left", health.DaysLeft,
		"issuer", health.Issuer,
		"issues", len(health.Issues),
	)

	return health
}

func (c *SSLChecker) getSSLCertificate(domain, port string) (*x509.Certificate, error) {
	// Set up TLS connection
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp",
		fmt.Sprintf("%s:%s", domain, port),
		&tls.Config{
			ServerName:         domain,
			InsecureSkipVerify: true, // We want to check the cert even if it's invalid
		},
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Get peer certificates
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no peer certificates")
	}

	return state.PeerCertificates[0], nil
}

func (c *SSLChecker) validateCertificate(cert *x509.Certificate, domain string, health *SSLHealth) {
	if cert == nil {
		health.Issues = append(health.Issues, "No certificate provided")
		health.Score -= 50
		return
	}

	leafCert := cert

	// Check expiry
	now := time.Now()
	if leafCert.NotAfter.Before(now) {
		health.Issues = append(health.Issues, "Certificate has expired")
		health.Status = "error"
		health.Score = 0
		return
	}

	if leafCert.NotBefore.After(now) {
		health.Issues = append(health.Issues, "Certificate is not yet valid")
		health.Status = "error"
		health.Score = 0
		return
	}

	// Check if certificate expires soon
	daysLeft := int(time.Until(leafCert.NotAfter).Hours() / 24)
	if daysLeft < 7 {
		health.Issues = append(health.Issues, "Certificate expires in less than 7 days")
		health.Status = "error"
		health.Score -= 60
	} else if daysLeft < 30 {
		health.Issues = append(health.Issues, "Certificate expires in less than 30 days")
		health.Score -= 30
	} else if daysLeft < 60 {
		health.Issues = append(health.Issues, "Certificate expires in less than 60 days")
		health.Score -= 10
	}

	// Check domain name validation
	domainMatches := false

	// Check CommonName
	if leafCert.Subject.CommonName == domain {
		domainMatches = true
	}

	// Check Subject Alternative Names
	for _, san := range leafCert.DNSNames {
		if san == domain || san == "*."+domain {
			domainMatches = true
			break
		}
	}

	if !domainMatches {
		health.Issues = append(health.Issues, "Certificate does not match domain name")
		health.Score -= 40
	}

	// Check key size and algorithm
	switch leafCert.PublicKeyAlgorithm {
	case x509.RSA:
		// Check RSA key size
		if rsaPubKey, ok := leafCert.PublicKey.(*rsa.PublicKey); ok {
			keySize := rsaPubKey.N.BitLen()
			if keySize < 2048 {
				health.Issues = append(health.Issues, "RSA key size is less than 2048 bits")
				health.Score -= 30
			} else if keySize < 4096 {
				health.Issues = append(health.Issues, "RSA key size is less than 4096 bits (recommended)")
				health.Score -= 5
			}
		}
	case x509.ECDSA:
		// ECDSA is good
	default:
		health.Issues = append(health.Issues, "Unknown public key algorithm")
		health.Score -= 10
	}

	// Check signature algorithm
	switch leafCert.SignatureAlgorithm {
	case x509.SHA1WithRSA, x509.DSAWithSHA1, x509.ECDSAWithSHA1:
		health.Issues = append(health.Issues, "Certificate uses weak SHA-1 signature algorithm")
		health.Score -= 40
	case x509.MD2WithRSA, x509.MD5WithRSA:
		health.Issues = append(health.Issues, "Certificate uses very weak MD2/MD5 signature algorithm")
		health.Score -= 60
	}

	// Check if it's a self-signed certificate
	if leafCert.Issuer.CommonName == leafCert.Subject.CommonName {
		health.Issues = append(health.Issues, "Certificate is self-signed")
		health.Score -= 50
	}

	// Note: We cannot check certificate chain length with just the leaf certificate
	// This would require getting the full certificate chain from the connection

	// Ensure score doesn't go below 0
	if health.Score < 0 {
		health.Score = 0
	}
}

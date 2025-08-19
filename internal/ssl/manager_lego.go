package ssl

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/digitalocean"
	"github.com/go-acme/lego/v4/registration"
	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/logging"
)

// LegoManager handles SSL certificate management using lego
type LegoManager struct {
	config     *config.Config
	certDir    string
	Email      string
	Staging    bool
	AgreeToTOS bool
	client     *lego.Client
	user       *LegoUser
}

// LegoUser implements registration.User
type LegoUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *LegoUser) GetEmail() string {
	return u.Email
}

func (u *LegoUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *LegoUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// NewLegoManager creates a new SSL manager using lego
func NewLegoManager(cfg *config.Config) *LegoManager {
	certDir := "/etc/mailserver/certs"
	
	return &LegoManager{
		config:  cfg,
		certDir: certDir,
	}
}

// setupClient initializes the lego client
func (m *LegoManager) setupClient() error {
	logger := logging.Get()
	
	// Create or load account key
	privateKey, err := m.getOrCreateAccountKey()
	if err != nil {
		return fmt.Errorf("failed to get account key: %w", err)
	}
	
	// Create user
	m.user = &LegoUser{
		Email: m.Email,
		key:   privateKey,
	}
	
	// Create config
	config := lego.NewConfig(m.user)
	config.Certificate.KeyType = certcrypto.RSA2048
	
	// Set CA URL
	if m.Staging {
		config.CADirURL = lego.LEDirectoryStaging
		logger.Info("Using Let's Encrypt staging environment")
	} else {
		config.CADirURL = lego.LEDirectoryProduction
	}
	
	// Create client
	client, err := lego.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create lego client: %w", err)
	}
	
	// Set up DNS provider based on configuration
	if m.config.DOAPIToken != "" {
		// Use DigitalOcean DNS-01 challenge
		logger.Info("Setting up DigitalOcean DNS-01 challenge")
		
		// Set the DO_AUTH_TOKEN environment variable for the provider
		os.Setenv("DO_AUTH_TOKEN", m.config.DOAPIToken)
		
		provider, err := digitalocean.NewDNSProvider()
		if err != nil {
			return fmt.Errorf("failed to create DigitalOcean DNS provider: %w", err)
		}
		
		err = client.Challenge.SetDNS01Provider(provider)
		if err != nil {
			return fmt.Errorf("failed to set DNS-01 provider: %w", err)
		}
		
		logger.Info("DNS-01 challenge configured with DigitalOcean")
	} else {
		// Fall back to HTTP-01 challenge
		logger.Warn("No DigitalOcean API token configured, DNS-01 challenge not available")
		return fmt.Errorf("DNS-01 challenge requires DigitalOcean API token")
	}
	
	m.client = client
	
	// Register if needed
	if m.user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: m.AgreeToTOS})
		if err != nil {
			return fmt.Errorf("failed to register: %w", err)
		}
		m.user.Registration = reg
		logger.Infof("Registered with Let's Encrypt: %s", reg.URI)
	}
	
	return nil
}

// ObtainCertificate obtains a new certificate from Let's Encrypt using DNS-01
func (m *LegoManager) ObtainCertificate() error {
	logger := logging.Get()
	
	// Ensure cert directory exists
	if err := os.MkdirAll(m.certDir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}
	
	// Setup client
	if err := m.setupClient(); err != nil {
		return fmt.Errorf("failed to setup client: %w", err)
	}
	
	// Request certificate
	request := certificate.ObtainRequest{
		Domains: []string{m.config.MailHostname},
		Bundle:  true,
	}
	
	logger.Infof("Requesting certificate for %s using DNS-01 challenge", m.config.MailHostname)
	
	certificates, err := m.client.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("failed to obtain certificate: %w", err)
	}
	
	// Save certificate files
	certPath := filepath.Join(m.certDir, "cert.pem")
	keyPath := filepath.Join(m.certDir, "key.pem")
	
	// Write certificate
	if err := os.WriteFile(certPath, certificates.Certificate, 0644); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}
	
	// Write private key
	if err := os.WriteFile(keyPath, certificates.PrivateKey, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}
	
	logger.Infof("Certificate obtained and saved for %s", m.config.MailHostname)
	logger.Infof("Certificate expires: %s", certificates.Domain)
	
	return nil
}

// RenewCertificate renews an existing certificate
func (m *LegoManager) RenewCertificate() error {
	logger := logging.Get()
	
	// Setup client
	if err := m.setupClient(); err != nil {
		return fmt.Errorf("failed to setup client: %w", err)
	}
	
	// Load existing certificate
	certPath := filepath.Join(m.certDir, "cert.pem")
	keyPath := filepath.Join(m.certDir, "key.pem")
	
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}
	
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key: %w", err)
	}
	
	// Parse certificate to get domains
	cert, err := ParseCertificate(certPEM)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}
	
	// Renew certificate
	certificates, err := m.client.Certificate.Renew(certificate.Resource{
		Domain:      cert.Subject.CommonName,
		Certificate: certPEM,
		PrivateKey:  keyPEM,
	}, true, false, "")
	if err != nil {
		return fmt.Errorf("failed to renew certificate: %w", err)
	}
	
	// Save renewed certificate
	if err := os.WriteFile(certPath, certificates.Certificate, 0644); err != nil {
		return fmt.Errorf("failed to write renewed certificate: %w", err)
	}
	
	if err := os.WriteFile(keyPath, certificates.PrivateKey, 0600); err != nil {
		return fmt.Errorf("failed to write renewed private key: %w", err)
	}
	
	logger.Infof("Certificate renewed for %s", m.config.MailHostname)
	
	return nil
}

// getOrCreateAccountKey gets or creates the account private key
func (m *LegoManager) getOrCreateAccountKey() (crypto.PrivateKey, error) {
	keyPath := filepath.Join(m.certDir, "account.key")
	
	// Try to load existing key
	if keyPEM, err := os.ReadFile(keyPath); err == nil {
		block, _ := pem.Decode(keyPEM)
		if block != nil {
			switch block.Type {
			case "EC PRIVATE KEY":
				return x509.ParseECPrivateKey(block.Bytes)
			case "RSA PRIVATE KEY":
				return x509.ParsePKCS1PrivateKey(block.Bytes)
			case "PRIVATE KEY":
				key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
				if err != nil {
					return nil, err
				}
				switch k := key.(type) {
				case *ecdsa.PrivateKey:
					return k, nil
				default:
					return nil, fmt.Errorf("unknown private key type in PKCS#8")
				}
			}
		}
	}
	
	// Generate new key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	
	// Save key
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})
	
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, fmt.Errorf("failed to save account key: %w", err)
	}
	
	return privateKey, nil
}

// RenewalNeeded checks if certificate renewal is needed
func (m *LegoManager) RenewalNeeded() (bool, error) {
	expires, err := m.ExpirationDate()
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // No certificate, needs obtaining
		}
		return false, err
	}
	
	// Renew if less than 30 days remaining
	daysLeft := int(time.Until(expires).Hours() / 24)
	return daysLeft < 30, nil
}

// ExpirationDate returns the certificate expiration date
func (m *LegoManager) ExpirationDate() (time.Time, error) {
	certPath := filepath.Join(m.certDir, "cert.pem")
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return time.Time{}, err
	}
	
	cert, err := ParseCertificate(certPEM)
	if err != nil {
		return time.Time{}, err
	}
	
	return cert.NotAfter, nil
}

// CertDir returns the certificate directory
func (m *LegoManager) CertDir() string {
	return m.certDir
}
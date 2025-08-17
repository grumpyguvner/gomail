package auth

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"strings"

	"github.com/emersion/go-msgauth/authres"
	"github.com/emersion/go-msgauth/dkim"
	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/grumpyguvner/gomail/internal/metrics"
	"go.uber.org/zap"
)

// DKIMVerifier handles DKIM signature verification
type DKIMVerifier struct {
	logger *zap.SugaredLogger
}

// NewDKIMVerifier creates a new DKIM verifier
func NewDKIMVerifier() *DKIMVerifier {
	return &DKIMVerifier{
		logger: logging.Get(),
	}
}

// DKIMResult represents the result of DKIM verification
type DKIMResult struct {
	Result   authres.ResultValue
	Domain   string
	Selector string
	Reason   string
}

// Verify performs DKIM signature verification
func (v *DKIMVerifier) Verify(ctx context.Context, message []byte) ([]*DKIMResult, error) {
	reader := bytes.NewReader(message)

	// Verify DKIM signatures
	verifications, err := dkim.Verify(reader)
	if err != nil {
		metrics.DKIMVerifyErrors.Inc()
		v.logger.Errorf("DKIM verification error: %v", err)
		return nil, fmt.Errorf("DKIM verification failed: %w", err)
	}

	results := make([]*DKIMResult, 0, len(verifications))

	for _, verification := range verifications {
		result := &DKIMResult{
			Domain:   verification.Domain,
			Selector: "", // Selector info not available in Verification
		}

		if verification.Err == nil {
			result.Result = authres.ResultPass
			result.Reason = "Valid signature"
			metrics.DKIMPass.Inc()
			v.logger.Infof("DKIM pass: domain=%s",
				verification.Domain)
		} else {
			// Determine failure type
			errStr := verification.Err.Error()
			switch {
			case strings.Contains(errStr, "key not found"):
				result.Result = authres.ResultPermError
				result.Reason = "Key not found in DNS"
				metrics.DKIMPermError.Inc()
			case strings.Contains(errStr, "DNS"):
				result.Result = authres.ResultTempError
				result.Reason = fmt.Sprintf("DNS error: %v", verification.Err)
				metrics.DKIMTempError.Inc()
			default:
				result.Result = authres.ResultFail
				result.Reason = fmt.Sprintf("Signature verification failed: %v", verification.Err)
				metrics.DKIMFail.Inc()
			}

			v.logger.Warnf("DKIM fail: domain=%s, error=%v",
				verification.Domain, verification.Err)
		}

		results = append(results, result)
	}

	if len(results) == 0 {
		// No DKIM signatures found
		metrics.DKIMNone.Inc()
		return []*DKIMResult{{
			Result: authres.ResultNone,
			Reason: "No DKIM signatures found",
		}}, nil
	}

	return results, nil
}

// FormatAuthenticationResults formats DKIM results for Authentication-Results header
func FormatDKIMResults(results []*DKIMResult) []string {
	formatted := make([]string, 0, len(results))
	for _, r := range results {
		if r.Domain != "" {
			formatted = append(formatted,
				fmt.Sprintf("dkim=%s header.d=%s header.s=%s",
					r.Result, r.Domain, r.Selector))
		} else {
			formatted = append(formatted, fmt.Sprintf("dkim=%s", r.Result))
		}
	}
	return formatted
}

// DKIMSigner handles DKIM signing for outgoing mail
type DKIMSigner struct {
	logger     *zap.SugaredLogger
	domain     string
	selector   string
	privateKey *rsa.PrivateKey
}

// NewDKIMSigner creates a new DKIM signer
func NewDKIMSigner(domain, selector string, privateKeyPEM []byte) (*DKIMSigner, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing private key")
	}

	// Try parsing as PKCS1
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try parsing as PKCS8
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not an RSA key")
		}
	}

	return &DKIMSigner{
		logger:     logging.Get(),
		domain:     domain,
		selector:   selector,
		privateKey: privateKey,
	}, nil
}

// Sign adds a DKIM signature to an email message
func (s *DKIMSigner) Sign(message []byte) ([]byte, error) {
	reader := bytes.NewReader(message)

	options := &dkim.SignOptions{
		Domain:   s.domain,
		Selector: s.selector,
		Signer:   s.privateKey,
		Hash:     crypto.SHA256,
		HeaderKeys: []string{
			"from", "to", "subject", "date",
			"message-id", "content-type", "mime-version",
		},
	}

	var output bytes.Buffer
	err := dkim.Sign(&output, reader, options)
	if err != nil {
		metrics.DKIMSignErrors.Inc()
		return nil, fmt.Errorf("DKIM signing failed: %w", err)
	}

	metrics.DKIMSigned.Inc()
	s.logger.Debugf("Message signed with DKIM: domain=%s, selector=%s",
		s.domain, s.selector)

	return output.Bytes(), nil
}

// GenerateDKIMKey generates a new DKIM key pair
func GenerateDKIMKey(bits int) (privateKey []byte, publicKey string, err error) {
	// Use dkim-keygen command from go-msgauth
	// This ensures compatibility with the library

	if bits < 1024 {
		bits = 2048 // Minimum recommended size
	}

	// Generate RSA key pair
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	privateKeyBuf := &bytes.Buffer{}
	if err := pem.Encode(privateKeyBuf, privateKeyPEM); err != nil {
		return nil, "", fmt.Errorf("failed to encode private key: %w", err)
	}

	// Generate public key for DNS record
	publicKeyDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	}
	publicKeyBuf := &bytes.Buffer{}
	if err := pem.Encode(publicKeyBuf, publicKeyPEM); err != nil {
		return nil, "", fmt.Errorf("failed to encode public key: %w", err)
	}

	// Format public key for DNS TXT record
	publicKeyStr := publicKeyBuf.String()
	// Remove PEM headers and newlines
	publicKeyStr = strings.ReplaceAll(publicKeyStr, "-----BEGIN PUBLIC KEY-----", "")
	publicKeyStr = strings.ReplaceAll(publicKeyStr, "-----END PUBLIC KEY-----", "")
	publicKeyStr = strings.ReplaceAll(publicKeyStr, "\n", "")

	dnsRecord := fmt.Sprintf("v=DKIM1; k=rsa; p=%s", publicKeyStr)

	return privateKeyBuf.Bytes(), dnsRecord, nil
}

// VerifyFromReader verifies DKIM signatures from an io.Reader
func (v *DKIMVerifier) VerifyFromReader(r io.Reader) ([]*DKIMResult, error) {
	// Read the entire message into memory
	// This is necessary because DKIM verification needs the raw message
	message, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	return v.Verify(context.Background(), message)
}

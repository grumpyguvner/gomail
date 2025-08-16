package mail

import (
	"bufio"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

type EmailData struct {
	Sender         string                 `json:"sender"`
	Recipient      string                 `json:"recipient"`
	ReceivedAt     time.Time              `json:"received_at"`
	Raw            string                 `json:"raw"`
	Subject        string                 `json:"subject,omitempty"`
	MessageID      string                 `json:"message_id,omitempty"`
	Connection     ConnectionInfo         `json:"connection"`
	Authentication AuthenticationMetadata `json:"authentication"`
}

type ConnectionInfo struct {
	ClientAddress  string `json:"client_address,omitempty"`
	ClientHostname string `json:"client_hostname,omitempty"`
	ClientHelo     string `json:"client_helo,omitempty"`
}

type AuthenticationMetadata struct {
	SPF   SPFMetadata   `json:"spf,omitempty"`
	DKIM  DKIMMetadata  `json:"dkim,omitempty"`
	DMARC DMARCMetadata `json:"dmarc,omitempty"`
}

type SPFMetadata struct {
	ClientIP         string `json:"client_ip,omitempty"`
	MailFrom         string `json:"mail_from,omitempty"`
	HeloDomain       string `json:"helo_domain,omitempty"`
	ReceivedSPFHeader string `json:"received_spf_header,omitempty"`
}

type DKIMMetadata struct {
	Signatures  []string `json:"signatures,omitempty"`
	FromDomain  string   `json:"from_domain,omitempty"`
	SignedBy    []string `json:"signed_by,omitempty"`
}

type DMARCMetadata struct {
	FromHeader            string `json:"from_header,omitempty"`
	ReturnPath            string `json:"return_path,omitempty"`
	AuthenticationResults string `json:"authentication_results,omitempty"`
}

func ParseRawEmail(rawEmail string, httpHeaders map[string]string) (*EmailData, error) {
	// Parse email headers
	msg, err := mail.ReadMessage(strings.NewReader(rawEmail))
	if err != nil {
		return nil, fmt.Errorf("failed to parse email: %w", err)
	}

	data := &EmailData{
		Raw:        rawEmail,
		ReceivedAt: time.Now(),
	}

	// Extract basic headers
	header := msg.Header
	data.Subject = header.Get("Subject")
	data.MessageID = header.Get("Message-ID")
	
	// Extract From and To
	if from, err := mail.ParseAddress(header.Get("From")); err == nil {
		data.Sender = from.Address
	} else {
		data.Sender = header.Get("From")
	}
	
	if to := header.Get("To"); to != "" {
		if addr, err := mail.ParseAddress(to); err == nil {
			data.Recipient = addr.Address
		} else {
			data.Recipient = to
		}
	}

	// Extract connection info from HTTP headers
	data.Connection = ConnectionInfo{
		ClientAddress:  httpHeaders["X-Original-Client-Address"],
		ClientHostname: httpHeaders["X-Original-Client-Hostname"],
		ClientHelo:     httpHeaders["X-Original-Helo"],
	}

	// Extract authentication metadata
	data.Authentication = extractAuthenticationMetadata(header, httpHeaders)

	// Override with HTTP headers if present
	if sender := httpHeaders["X-Original-Sender"]; sender != "" {
		data.Sender = sender
	}
	if recipient := httpHeaders["X-Original-Recipient"]; recipient != "" {
		data.Recipient = recipient
	}

	return data, nil
}

func extractAuthenticationMetadata(header mail.Header, httpHeaders map[string]string) AuthenticationMetadata {
	auth := AuthenticationMetadata{}

	// SPF metadata
	auth.SPF = SPFMetadata{
		ClientIP:          httpHeaders["X-Original-Client-Address"],
		MailFrom:          httpHeaders["X-Original-Mail-From"],
		HeloDomain:        httpHeaders["X-Original-Helo"],
		ReceivedSPFHeader: header.Get("Received-SPF"),
	}

	// DKIM metadata
	auth.DKIM = extractDKIMMetadata(header)

	// DMARC metadata
	auth.DMARC = DMARCMetadata{
		FromHeader:            header.Get("From"),
		ReturnPath:            header.Get("Return-Path"),
		AuthenticationResults: header.Get("Authentication-Results"),
	}

	return auth
}

func extractDKIMMetadata(header mail.Header) DKIMMetadata {
	dkim := DKIMMetadata{
		Signatures: []string{},
		SignedBy:   []string{},
	}

	// Extract DKIM-Signature headers (can be multiple)
	for _, sig := range header["Dkim-Signature"] {
		// Handle multi-line DKIM signatures
		cleanSig := strings.ReplaceAll(sig, "\r\n\t", " ")
		cleanSig = strings.ReplaceAll(cleanSig, "\r\n ", " ")
		dkim.Signatures = append(dkim.Signatures, cleanSig)

		// Extract d= parameter (signing domain)
		if d := extractDKIMParam(cleanSig, "d="); d != "" {
			dkim.SignedBy = append(dkim.SignedBy, d)
		}
	}

	// Extract From domain for DKIM alignment
	if from := header.Get("From"); from != "" {
		if addr, err := mail.ParseAddress(from); err == nil {
			parts := strings.Split(addr.Address, "@")
			if len(parts) == 2 {
				dkim.FromDomain = parts[1]
			}
		}
	}

	return dkim
}

func extractDKIMParam(signature, param string) string {
	idx := strings.Index(signature, param)
	if idx == -1 {
		return ""
	}

	start := idx + len(param)
	end := strings.IndexAny(signature[start:], "; \t")
	if end == -1 {
		return strings.TrimSpace(signature[start:])
	}
	return strings.TrimSpace(signature[start : start+end])
}

func FromJSON(data map[string]interface{}) *EmailData {
	email := &EmailData{
		ReceivedAt: time.Now(),
	}

	// Extract basic fields
	if v, ok := data["sender"].(string); ok {
		email.Sender = v
	}
	if v, ok := data["recipient"].(string); ok {
		email.Recipient = v
	}
	if v, ok := data["raw"].(string); ok {
		email.Raw = v
	}
	if v, ok := data["subject"].(string); ok {
		email.Subject = v
	}

	// Extract connection info
	if conn, ok := data["connection"].(map[string]interface{}); ok {
		if v, ok := conn["client_address"].(string); ok {
			email.Connection.ClientAddress = v
		}
		if v, ok := conn["client_hostname"].(string); ok {
			email.Connection.ClientHostname = v
		}
		if v, ok := conn["client_helo"].(string); ok {
			email.Connection.ClientHelo = v
		}
	}

	// Parse raw email if present to extract more metadata
	if email.Raw != "" {
		if parsed, err := ParseRawEmail(email.Raw, nil); err == nil {
			email.Authentication = parsed.Authentication
			if email.Subject == "" {
				email.Subject = parsed.Subject
			}
			if email.MessageID == "" {
				email.MessageID = parsed.MessageID
			}
		}
	}

	return email
}

func ParseHeaders(reader *bufio.Reader) (mail.Header, error) {
	headers := make(mail.Header)
	var currentKey string
	var currentValue strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		// Empty line marks end of headers
		if line == "\r\n" || line == "\n" {
			if currentKey != "" {
				headers[currentKey] = append(headers[currentKey], currentValue.String())
			}
			break
		}

		// Continuation line (starts with space or tab)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			currentValue.WriteString(line)
			continue
		}

		// Save previous header if exists
		if currentKey != "" {
			headers[currentKey] = append(headers[currentKey], currentValue.String())
		}

		// Parse new header
		idx := strings.Index(line, ":")
		if idx > 0 {
			currentKey = strings.TrimSpace(line[:idx])
			currentValue.Reset()
			currentValue.WriteString(strings.TrimSpace(line[idx+1:]))
		}
	}

	return headers, nil
}
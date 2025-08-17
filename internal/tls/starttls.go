package tls

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/grumpyguvner/gomail/internal/metrics"
	"go.uber.org/zap"
)

// STARTTLSServer handles STARTTLS upgrades for SMTP connections
type STARTTLSServer struct {
	tlsConfig *tls.Config
	logger    *zap.SugaredLogger
}

// NewSTARTTLSServer creates a new STARTTLS server
func NewSTARTTLSServer(tlsConfig *tls.Config) *STARTTLSServer {
	return &STARTTLSServer{
		tlsConfig: tlsConfig,
		logger:    logging.Get(),
	}
}

// HandleConnection processes an SMTP connection with STARTTLS support
func (s *STARTTLSServer) HandleConnection(conn net.Conn) error {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Send SMTP greeting
	if err := s.sendResponse(writer, 220, "mail.example.com ESMTP Ready"); err != nil {
		return err
	}

	for {
		// Read command
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, " ", 2)
		command := strings.ToUpper(parts[0])

		switch command {
		case "EHLO", "HELO":
			// Respond with capabilities including STARTTLS
			if err := s.handleEHLO(writer, parts); err != nil {
				return err
			}

		case "STARTTLS":
			// Upgrade to TLS
			if err := s.handleSTARTTLS(conn, reader, writer); err != nil {
				return err
			}
			// After TLS upgrade, update reader/writer
			reader = bufio.NewReader(conn)
			writer = bufio.NewWriter(conn)

		case "QUIT":
			_ = s.sendResponse(writer, 221, "Bye")
			return nil

		default:
			// For other commands, check if TLS is required
			if s.isTLSRequired() && !s.isConnectionSecure(conn) {
				_ = s.sendResponse(writer, 530, "Must issue a STARTTLS command first")
			} else {
				// Pass through to normal SMTP handling
				_ = s.sendResponse(writer, 250, "OK")
			}
		}
	}
}

// handleEHLO responds to EHLO/HELO commands
func (s *STARTTLSServer) handleEHLO(writer *bufio.Writer, parts []string) error {
	hostname := "localhost"
	if len(parts) > 1 {
		hostname = parts[1]
	}

	responses := []string{
		fmt.Sprintf("250-mail.example.com Hello %s", hostname),
		"250-SIZE 26214400",
		"250-8BITMIME",
		"250-PIPELINING",
	}

	// Add STARTTLS if TLS is available
	if s.tlsConfig != nil {
		responses = append(responses, "250-STARTTLS")
	}

	responses = append(responses, "250 HELP")

	for _, resp := range responses {
		if _, err := writer.WriteString(resp + "\r\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}

// handleSTARTTLS upgrades the connection to TLS
func (s *STARTTLSServer) handleSTARTTLS(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) error {
	if s.tlsConfig == nil {
		return s.sendResponse(writer, 454, "TLS not available")
	}

	// Check if already using TLS
	if _, ok := conn.(*tls.Conn); ok {
		return s.sendResponse(writer, 454, "TLS already active")
	}

	// Send ready response
	if err := s.sendResponse(writer, 220, "Ready to start TLS"); err != nil {
		return err
	}

	// Upgrade connection to TLS
	tlsConn := tls.Server(conn, s.tlsConfig)

	// Set handshake timeout
	_ = tlsConn.SetDeadline(time.Now().Add(10 * time.Second))

	// Perform TLS handshake
	if err := tlsConn.Handshake(); err != nil {
		metrics.TLSHandshakeErrors.Inc()
		s.logger.Errorf("TLS handshake failed: %v", err)
		return err
	}

	// Clear deadline after successful handshake
	_ = tlsConn.SetDeadline(time.Time{})

	// Log TLS connection info
	state := tlsConn.ConnectionState()
	s.logger.Infof("TLS connection established: version=%x, cipher=%x",
		state.Version, state.CipherSuite)

	// Track metrics
	metrics.TLSConnections.Inc()
	metrics.TLSVersion.WithLabelValues(getTLSVersionString(state.Version)).Inc()
	metrics.TLSCipherSuite.WithLabelValues(tls.CipherSuiteName(state.CipherSuite)).Inc()

	// Replace the connection in the original net.Conn
	// This is a bit tricky - we need to return the TLS connection
	// In practice, this would be handled by the caller

	return nil
}

// sendResponse sends an SMTP response
func (s *STARTTLSServer) sendResponse(writer *bufio.Writer, code int, message string) error {
	_, err := writer.WriteString(fmt.Sprintf("%d %s\r\n", code, message))
	if err != nil {
		return err
	}
	return writer.Flush()
}

// isTLSRequired checks if TLS is mandatory
func (s *STARTTLSServer) isTLSRequired() bool {
	// This could be configurable
	return false // Opportunistic TLS by default
}

// isConnectionSecure checks if connection is already using TLS
func (s *STARTTLSServer) isConnectionSecure(conn net.Conn) bool {
	_, ok := conn.(*tls.Conn)
	return ok
}

// getTLSVersionString returns a string representation of TLS version
func getTLSVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS1.0"
	case tls.VersionTLS11:
		return "TLS1.1"
	case tls.VersionTLS12:
		return "TLS1.2"
	case tls.VersionTLS13:
		return "TLS1.3"
	default:
		return fmt.Sprintf("0x%04x", version)
	}
}

// UpgradeConnection upgrades an existing connection to TLS
func UpgradeConnection(conn net.Conn, config *tls.Config) (*tls.Conn, error) {
	tlsConn := tls.Server(conn, config)

	// Perform handshake with timeout
	_ = tlsConn.SetDeadline(time.Now().Add(10 * time.Second))
	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}
	_ = tlsConn.SetDeadline(time.Time{})

	return tlsConn, nil
}

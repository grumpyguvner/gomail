package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/mail"
	"github.com/grumpyguvner/gomail/internal/storage"
)

type Server struct {
	config      *config.Config
	httpServer  *http.Server
	listener    net.Listener
	storage     *storage.FileStorage
	metrics     *Metrics
}

type Metrics struct {
	TotalEmails    atomic.Int64
	TotalBytes     atomic.Int64
	LastReceived   atomic.Value // time.Time
	StartTime      time.Time
}

func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize storage
	store, err := storage.NewFileStorage(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	s := &Server{
		config:  cfg,
		storage: store,
		metrics: &Metrics{
			StartTime: time.Now(),
		},
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/mail/inbound", s.requireAuth(s.handleMailInbound))
	mux.HandleFunc("/email", s.requireAuth(s.handleMailInbound)) // Legacy endpoint
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/metrics", s.handleMetrics)

	s.httpServer = &http.Server{
		Handler:           mux,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	var err error

	switch s.config.Mode {
	case "socket":
		// Socket activation mode (systemd)
		if os.Getenv("LISTEN_FDS") == "1" {
			// Use systemd socket
			s.listener, err = net.FileListener(os.NewFile(3, ""))
			if err != nil {
				return fmt.Errorf("failed to get systemd socket: %w", err)
			}
		} else {
			// Fallback to regular TCP
			s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
			if err != nil {
				return fmt.Errorf("failed to listen: %w", err)
			}
		}
	default:
		// Simple mode - standard TCP listener
		s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
	}

	// Start serving
	go func() {
		if err := s.httpServer.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token != s.config.BearerToken {
			http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (s *Server) handleMailInbound(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body with size limit
	body, err := io.ReadAll(io.LimitReader(r.Body, 26214400)) // 25MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Determine content type and parse accordingly
	contentType := r.Header.Get("Content-Type")
	var emailData *mail.EmailData

	switch {
	case strings.Contains(contentType, "message/rfc822"):
		// Raw email format
		emailData, err = mail.ParseRawEmail(string(body), extractHeadersFromRequest(r))
		
	case strings.Contains(contentType, "application/json"):
		// JSON format (legacy)
		var jsonData map[string]interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		emailData = mail.FromJSON(jsonData)
		
	default:
		// Try to detect format
		if body[0] == '{' {
			var jsonData map[string]interface{}
			if err := json.Unmarshal(body, &jsonData); err != nil {
				http.Error(w, "Invalid request format", http.StatusBadRequest)
				return
			}
			emailData = mail.FromJSON(jsonData)
		} else {
			emailData, err = mail.ParseRawEmail(string(body), extractHeadersFromRequest(r))
		}
	}

	if err != nil {
		log.Printf("Failed to parse email: %v", err)
		http.Error(w, "Failed to parse email", http.StatusBadRequest)
		return
	}

	// Store email
	filename, err := s.storage.Store(emailData)
	if err != nil {
		log.Printf("Failed to store email: %v", err)
		http.Error(w, "Failed to store email", http.StatusInternalServerError)
		return
	}

	// Update metrics
	s.metrics.TotalEmails.Add(1)
	s.metrics.TotalBytes.Add(int64(len(body)))
	s.metrics.LastReceived.Store(time.Now())

	// Send response
	response := map[string]interface{}{
		"status":     "success",
		"message_id": filepath.Base(filename),
		"stored_at":  filename,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("Email received: from=%s to=%s size=%d stored=%s",
		emailData.Sender, emailData.Recipient, len(body), filename)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0",
		"uptime":  time.Since(s.metrics.StartTime).Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	var lastReceived interface{} = nil
	if t, ok := s.metrics.LastReceived.Load().(time.Time); ok && !t.IsZero() {
		lastReceived = t.Format(time.RFC3339)
	}

	response := map[string]interface{}{
		"total_emails":    s.metrics.TotalEmails.Load(),
		"total_bytes":     s.metrics.TotalBytes.Load(),
		"last_received":   lastReceived,
		"uptime_seconds":  time.Since(s.metrics.StartTime).Seconds(),
		"start_time":      s.metrics.StartTime.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func extractHeadersFromRequest(r *http.Request) map[string]string {
	headers := make(map[string]string)
	
	// Extract X-Original-* headers from HTTP request
	for key, values := range r.Header {
		if strings.HasPrefix(key, "X-Original-") && len(values) > 0 {
			headers[key] = values[0]
		}
	}
	
	return headers
}
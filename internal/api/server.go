package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/errors"
	"github.com/grumpyguvner/gomail/internal/logging"
	"github.com/grumpyguvner/gomail/internal/mail"
	"github.com/grumpyguvner/gomail/internal/metrics"
	"github.com/grumpyguvner/gomail/internal/middleware"
	"github.com/grumpyguvner/gomail/internal/storage"
	"github.com/grumpyguvner/gomail/internal/validation"
)

type Server struct {
	config          *config.Config
	httpServer      *http.Server
	listener        net.Listener
	listenerMu      sync.RWMutex
	storage         *storage.FileStorage
	metrics         *Metrics
	validator       *validation.EmailValidator
	activeRequests  atomic.Int64
	shutdownStarted atomic.Bool
}

type Metrics struct {
	TotalEmails    atomic.Int64
	TotalBytes     atomic.Int64
	LastReceived   atomic.Value // time.Time
	StartTime      time.Time
	ActiveRequests *atomic.Int64 // Pointer to server's activeRequests
}

func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize storage
	store, err := storage.NewFileStorage(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	s := &Server{
		config:    cfg,
		storage:   store,
		validator: validation.NewEmailValidator(),
	}

	s.metrics = &Metrics{
		StartTime:      time.Now(),
		ActiveRequests: &s.activeRequests,
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/mail/inbound", s.requireAuth(s.handleMailInbound))
	mux.HandleFunc("/email", s.requireAuth(s.handleMailInbound)) // Legacy endpoint
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/metrics", s.handleMetrics)

	// Apply middleware chain
	handler := s.applyMiddleware(mux)

	s.httpServer = &http.Server{
		Handler:        handler,
		ReadTimeout:    time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(cfg.IdleTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	var err error

	switch s.config.Mode {
	case "socket":
		// Socket activation mode (systemd)
		var listener net.Listener
		if os.Getenv("LISTEN_FDS") == "1" {
			// Use systemd socket
			listener, err = net.FileListener(os.NewFile(3, ""))
			if err != nil {
				return fmt.Errorf("failed to get systemd socket: %w", err)
			}
		} else {
			// Fallback to regular TCP
			listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
			if err != nil {
				return fmt.Errorf("failed to listen: %w", err)
			}
		}
		s.listenerMu.Lock()
		s.listener = listener
		s.listenerMu.Unlock()
	default:
		// Simple mode - standard TCP listener
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
		s.listenerMu.Lock()
		s.listener = listener
		s.listenerMu.Unlock()
	}

	// Start serving
	go func() {
		s.listenerMu.RLock()
		listener := s.listener
		s.listenerMu.RUnlock()

		if listener == nil {
			logging.Get().Error("Listener is nil, cannot start server")
			return
		}

		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			logging.Get().Errorf("Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

// GetListener returns the server's listener in a thread-safe way
func (s *Server) GetListener() net.Listener {
	s.listenerMu.RLock()
	defer s.listenerMu.RUnlock()
	return s.listener
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownStart := time.Now()
	s.shutdownStarted.Store(true)

	// Log current state
	activeReqs := s.activeRequests.Load()
	if activeReqs > 0 {
		logging.Get().Infof("Waiting for %d active requests to complete...", activeReqs)
	}

	// Monitor shutdown progress
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if reqs := s.activeRequests.Load(); reqs > 0 {
					logging.Get().Infof("Still waiting for %d active requests...", reqs)
				}
			case <-done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Perform shutdown
	err := s.httpServer.Shutdown(ctx)
	close(done)

	// Record shutdown metrics
	shutdownDuration := time.Since(shutdownStart)
	metrics.ShutdownDuration.Observe(shutdownDuration.Seconds())

	if err != nil {
		if err == context.DeadlineExceeded {
			forcedClose := s.activeRequests.Load()
			if forcedClose > 0 {
				logging.Get().Warnf("Forced shutdown with %d active requests", forcedClose)
				metrics.ShutdownsInitiated.WithLabelValues("forced").Inc()
			}
		}
		return err
	}

	metrics.ShutdownsInitiated.WithLabelValues("graceful").Inc()
	logging.Get().Info("All connections drained successfully")
	return nil
}

func (s *Server) applyMiddleware(handler http.Handler) http.Handler {
	// Apply middlewares in reverse order (innermost first)
	// Request flow: Prometheus -> Timeout -> ActiveRequest -> RateLimit -> RequestID -> ErrorHandler -> Recovery -> handler

	// Track active requests for graceful shutdown
	handler = s.activeRequestsMiddleware(handler)

	handler = middleware.RecoveryMiddleware(handler)
	handler = middleware.ErrorHandlerMiddleware(handler)
	handler = middleware.RequestIDMiddleware(handler)

	// Add rate limiting with configuration
	rate := s.config.RateLimitPerMinute
	if rate <= 0 {
		rate = 60 // Default fallback
	}
	burst := s.config.RateLimitBurst
	if burst <= 0 {
		burst = 10 // Default fallback
	}

	rateLimiter := middleware.NewRateLimiter(rate, burst, 5*time.Minute, logging.Get().Desugar())
	handler = rateLimiter.Middleware(handler)

	// Add timeout middleware if configured
	if s.config.HandlerTimeout > 0 {
		handler = middleware.TimeoutMiddleware(time.Duration(s.config.HandlerTimeout) * time.Second)(handler)
	}

	// Add Prometheus metrics middleware as the outermost layer
	handler = middleware.PrometheusMiddleware(handler)

	return handler
}

func (s *Server) activeRequestsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reject new requests if shutdown has started
		if s.shutdownStarted.Load() {
			middleware.SendErrorResponse(w, errors.UnavailableError("Server is shutting down"))
			return
		}

		s.activeRequests.Add(1)
		defer s.activeRequests.Add(-1)

		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			middleware.SendErrorResponse(w, errors.AuthError("Missing authorization header"))
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token != s.config.BearerToken {
			middleware.SendErrorResponse(w, errors.AuthError("Invalid authorization token"))
			return
		}

		next(w, r)
	}
}

func (s *Server) handleMailInbound(w http.ResponseWriter, r *http.Request) {
	// Create context with handler timeout
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(s.config.HandlerTimeout)*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	// Start timing for processing duration
	start := time.Now()

	if r.Method != http.MethodPost {
		// Method not allowed should return 405, not 400
		err := errors.New(errors.ErrorTypeBadRequest, "Method not allowed")
		err.StatusCode = http.StatusMethodNotAllowed
		middleware.SendErrorResponse(w, err)
		return
	}

	// Read body with size limit
	body, err := io.ReadAll(io.LimitReader(r.Body, 26214400)) // 25MB limit
	if err != nil {
		middleware.SendErrorResponse(w, errors.BadRequestError("Failed to read request body"))
		return
	}

	// Record email size metric
	metrics.EmailSize.Observe(float64(len(body)))

	// Sanitize headers first
	headers := validation.SanitizeHeaders(extractHeadersFromRequest(r))

	// Determine content type and parse accordingly
	contentType := r.Header.Get("Content-Type")
	var emailData *mail.EmailData

	switch {
	case strings.Contains(contentType, "message/rfc822"):
		// Raw email format
		emailData, err = mail.ParseRawEmail(string(body), headers)

	case strings.Contains(contentType, "application/json"):
		// JSON format (legacy)
		var jsonData map[string]interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			middleware.SendErrorResponse(w, errors.ValidationError("Invalid JSON", map[string]string{"error": err.Error()}))
			return
		}
		emailData = mail.FromJSON(jsonData)

	default:
		// Try to detect format
		if len(body) > 0 && body[0] == '{' {
			var jsonData map[string]interface{}
			if err := json.Unmarshal(body, &jsonData); err != nil {
				middleware.SendErrorResponse(w, errors.BadRequestError("Invalid request format"))
				return
			}
			emailData = mail.FromJSON(jsonData)
		} else {
			emailData, err = mail.ParseRawEmail(string(body), headers)
		}
	}

	if err != nil {
		requestID := middleware.GetRequestIDFromRequest(r)
		logging.WithRequestID(requestID).Errorf("Failed to parse email: %v", err)
		metrics.EmailsProcessed.WithLabelValues("error").Inc()
		metrics.EmailProcessingDuration.Observe(time.Since(start).Seconds())
		middleware.SendErrorResponse(w, errors.ValidationError("Failed to parse email", map[string]string{"error": err.Error()}))
		return
	}

	// Validate email data
	if err := s.validator.Validate(emailData); err != nil {
		requestID := middleware.GetRequestIDFromRequest(r)
		logging.WithRequestID(requestID).Errorf("Email validation failed: %v", err)
		metrics.EmailsProcessed.WithLabelValues("rejected").Inc()
		metrics.EmailProcessingDuration.Observe(time.Since(start).Seconds())
		middleware.SendErrorResponse(w, errors.ValidationError("Email validation failed", map[string]string{"error": err.Error()}))
		return
	}

	// Store email
	filename, err := s.storage.Store(emailData)
	if err != nil {
		requestID := middleware.GetRequestIDFromRequest(r)
		logging.WithRequestID(requestID).Errorf("Failed to store email: %v", err)
		metrics.StorageOperations.WithLabelValues("write", "error").Inc()
		metrics.EmailsProcessed.WithLabelValues("error").Inc()
		metrics.EmailProcessingDuration.Observe(time.Since(start).Seconds())
		middleware.SendErrorResponse(w, errors.StorageError("Failed to store email", err))
		return
	}

	// Record successful storage
	metrics.StorageOperations.WithLabelValues("write", "success").Inc()

	// Update metrics
	s.metrics.TotalEmails.Add(1)
	s.metrics.TotalBytes.Add(int64(len(body)))
	s.metrics.LastReceived.Store(time.Now())

	// Record successful email processing
	metrics.EmailsProcessed.WithLabelValues("success").Inc()
	metrics.EmailProcessingDuration.Observe(time.Since(start).Seconds())

	// Send response
	response := map[string]interface{}{
		"status":     "success",
		"message_id": filepath.Base(filename),
		"stored_at":  filename,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		requestID := middleware.GetRequestIDFromRequest(r)
		logging.WithRequestID(requestID).Errorf("Failed to encode response: %v", err)
	}

	requestID := middleware.GetRequestIDFromRequest(r)
	logging.WithRequestID(requestID).Infow("Email received",
		"from", emailData.Sender,
		"to", emailData.Recipient,
		"size", len(body),
		"stored", filename)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0",
		"uptime":  time.Since(s.metrics.StartTime).Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Get().Errorf("Failed to encode response: %v", err)
	}
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
		"active_requests": s.activeRequests.Load(),
		"shutting_down":   s.shutdownStarted.Load(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Get().Errorf("Failed to encode response: %v", err)
	}
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

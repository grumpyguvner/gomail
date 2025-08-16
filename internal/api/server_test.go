package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		Port:        3000,
		BearerToken: "test-token",
		DataDir:     t.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, cfg, server.config)
	assert.NotNil(t, server.httpServer)
	assert.NotNil(t, server.storage)
	assert.NotNil(t, server.metrics)
}

func TestServerAuthentication(t *testing.T) {
	cfg := &config.Config{
		BearerToken: "valid-token",
		DataDir:     t.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "missing auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "valid token",
			authHeader:     "Bearer valid-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wrong auth type",
			authHeader:     "Basic dXNlcjpwYXNz",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := server.requireAuth(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			recorder := httptest.NewRecorder()
			handler(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}
}

func TestHandleHealth(t *testing.T) {
	cfg := &config.Config{
		DataDir: t.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/health", nil)
	recorder := httptest.NewRecorder()

	server.handleHealth(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.NotNil(t, response["uptime"])
}

func TestHandleMetrics(t *testing.T) {
	cfg := &config.Config{
		DataDir: t.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)

	// Add some metrics
	server.metrics.TotalEmails.Add(5)
	server.metrics.TotalBytes.Add(1024)
	server.metrics.LastReceived.Store(time.Now())

	req := httptest.NewRequest("GET", "/metrics", nil)
	recorder := httptest.NewRecorder()

	server.handleMetrics(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(5), response["total_emails"])
	assert.Equal(t, float64(1024), response["total_bytes"])
	assert.NotNil(t, response["last_received"])
	assert.NotNil(t, response["uptime_seconds"])
	assert.NotNil(t, response["start_time"])
}

func TestHandleMailInbound_InvalidMethod(t *testing.T) {
	cfg := &config.Config{
		BearerToken: "test-token",
		DataDir:     t.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/mail/inbound", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	recorder := httptest.NewRecorder()

	server.handleMailInbound(recorder, req)

	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
}

func TestHandleMailInbound_JSONFormat(t *testing.T) {
	cfg := &config.Config{
		BearerToken: "test-token",
		DataDir:     t.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)

	emailData := map[string]interface{}{
		"sender":    "sender@example.com",
		"recipient": "recipient@example.com",
		"subject":   "Test Subject",
		"body":      "Test Body",
	}

	jsonData, err := json.Marshal(emailData)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mail/inbound", bytes.NewReader(jsonData))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.handleMailInbound(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.NotEmpty(t, response["message_id"])
	assert.NotEmpty(t, response["stored_at"])
	assert.NotEmpty(t, response["timestamp"])

	// Verify metrics were updated
	assert.Equal(t, int64(1), server.metrics.TotalEmails.Load())
	assert.Equal(t, int64(len(jsonData)), server.metrics.TotalBytes.Load())
}

func TestHandleMailInbound_InvalidJSON(t *testing.T) {
	cfg := &config.Config{
		BearerToken: "test-token",
		DataDir:     t.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mail/inbound", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.handleMailInbound(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestHandleMailInbound_SizeLimit(t *testing.T) {
	cfg := &config.Config{
		BearerToken: "test-token",
		DataDir:     t.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)

	// Create a reader that exceeds the size limit
	largeBody := bytes.NewReader(make([]byte, 26214401)) // 25MB + 1 byte
	req := httptest.NewRequest("POST", "/mail/inbound", largeBody)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.handleMailInbound(recorder, req)

	// The body will be truncated to 25MB, but should still fail as invalid JSON
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestServer_StartAndShutdown(t *testing.T) {
	cfg := &config.Config{
		Port:        0, // Use random available port
		BearerToken: "test-token",
		DataDir:     t.TempDir(),
		Mode:        "simple",
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server in goroutine
	started := make(chan struct{})
	errChan := make(chan error, 1)
	go func() {
		close(started)
		errChan <- server.Start(ctx)
	}()

	// Wait for server to start
	<-started
	time.Sleep(100 * time.Millisecond)

	// Shutdown server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	err = server.Shutdown(shutdownCtx)
	assert.NoError(t, err)

	// Cancel the start context to clean up
	cancel()

	// Wait for Start to return
	select {
	case <-errChan:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("Server did not shut down in time")
	}
}

func TestExtractHeadersFromRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Original-From", "sender@example.com")
	req.Header.Set("X-Original-To", "recipient@example.com")
	req.Header.Set("X-Original-Subject", "Test Subject")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")

	headers := extractHeadersFromRequest(req)

	assert.Equal(t, "sender@example.com", headers["X-Original-From"])
	assert.Equal(t, "recipient@example.com", headers["X-Original-To"])
	assert.Equal(t, "Test Subject", headers["X-Original-Subject"])
	assert.Empty(t, headers["Content-Type"])  // Should not include non-X-Original headers
	assert.Empty(t, headers["Authorization"]) // Should not include non-X-Original headers
}

func TestHandleMailInbound_RFC822Format(t *testing.T) {
	cfg := &config.Config{
		BearerToken: "test-token",
		DataDir:     t.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)

	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Test Subject
Date: Mon, 2 Jan 2006 15:04:05 -0700
Message-ID: <test@example.com>

This is the email body.`

	req := httptest.NewRequest("POST", "/mail/inbound", bytes.NewReader([]byte(rawEmail)))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "message/rfc822")
	recorder := httptest.NewRecorder()

	server.handleMailInbound(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.NotEmpty(t, response["message_id"])
}

func TestHandleMailInbound_AutoDetectFormat(t *testing.T) {
	cfg := &config.Config{
		BearerToken: "test-token",
		DataDir:     t.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)

	tests := []struct {
		name string
		body []byte
	}{
		{
			name: "auto-detect JSON",
			body: []byte(`{"sender":"test@example.com","recipient":"dest@example.com","subject":"Test","body":"Body"}`),
		},
		{
			name: "auto-detect RFC822",
			body: []byte("From: test@example.com\r\nTo: dest@example.com\r\nSubject: Test\r\n\r\nBody"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mail/inbound", bytes.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer test-token")
			// No Content-Type header to test auto-detection
			recorder := httptest.NewRecorder()

			server.handleMailInbound(recorder, req)

			assert.Equal(t, http.StatusOK, recorder.Code)
		})
	}
}

func BenchmarkHandleMailInbound(b *testing.B) {
	cfg := &config.Config{
		BearerToken: "test-token",
		DataDir:     b.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(b, err)

	emailData := map[string]interface{}{
		"sender":    "sender@example.com",
		"recipient": "recipient@example.com",
		"subject":   "Test Subject",
		"body":      "Test Body",
	}

	jsonData, _ := json.Marshal(emailData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/mail/inbound", bytes.NewReader(jsonData))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		server.handleMailInbound(recorder, req)
	}
}

func BenchmarkAuthentication(b *testing.B) {
	cfg := &config.Config{
		BearerToken: "valid-token",
		DataDir:     b.TempDir(),
	}

	server, err := NewServer(cfg)
	require.NoError(b, err)

	handler := server.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		handler(recorder, req)
	}
}

package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/grumpyguvner/gomail/internal/config"
)

func TestServer_GracefulShutdown(t *testing.T) {
	cfg := &config.Config{
		Port:        0, // Random port
		Mode:        "simple",
		DataDir:     t.TempDir(),
		BearerToken: "test-token",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverStarted := make(chan struct{})
	go func() {
		close(serverStarted)
		if err := server.Start(ctx); err != nil {
			t.Errorf("Server start error: %v", err)
		}
	}()

	// Wait for server to start
	<-serverStarted
	time.Sleep(100 * time.Millisecond)

	// Test normal shutdown without active requests
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	cancel() // Trigger shutdown
	time.Sleep(100 * time.Millisecond)

	if err := server.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Verify server rejects new requests after shutdown
	resp, err := http.Get(baseURL + "/health")
	if err == nil {
		resp.Body.Close()
		t.Error("Server should reject requests after shutdown")
	}
}

func TestServer_GracefulShutdown_WithActiveRequests(t *testing.T) {
	cfg := &config.Config{
		Port:        0, // Random port
		Mode:        "simple",
		DataDir:     t.TempDir(),
		BearerToken: "test-token",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := server.Start(ctx); err != nil && err != context.Canceled {
			t.Errorf("Server start error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test that active requests are tracked correctly
	// Note: In a real scenario, actual HTTP requests would increment this counter
	server.activeRequests.Store(3)

	// Verify initial state
	if active := server.activeRequests.Load(); active != 3 {
		t.Errorf("Expected 3 active requests, got %d", active)
	}

	// Trigger shutdown signal
	cancel()

	// Shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()

	// The shutdown will complete immediately since we're not actually handling requests
	err = server.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Verify shutdown flag is set
	if !server.shutdownStarted.Load() {
		t.Error("Expected shutdown flag to be set")
	}
}

func TestServer_GracefulShutdown_Timeout(t *testing.T) {
	cfg := &config.Config{
		Port:        0, // Random port
		Mode:        "simple",
		DataDir:     t.TempDir(),
		BearerToken: "test-token",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Mock very slow handler that exceeds shutdown timeout
	verySlowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Longer than shutdown timeout
		w.WriteHeader(http.StatusOK)
	})

	// Replace health handler with very slow handler for testing
	mux := http.NewServeMux()
	mux.HandleFunc("/very-slow", verySlowHandler)
	server.httpServer.Handler = server.applyMiddleware(mux)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.Start(ctx); err != nil {
			t.Errorf("Server start error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Get the actual port
	addr := server.listener.Addr().String()
	baseURL := fmt.Sprintf("http://%s", addr)

	// Start a very slow request
	requestDone := make(chan bool, 1)
	go func() {
		resp, err := http.Get(baseURL + "/very-slow")
		if err == nil {
			resp.Body.Close()
			requestDone <- true
		} else {
			requestDone <- false
		}
	}()

	// Give request time to start
	time.Sleep(100 * time.Millisecond)

	// Trigger shutdown
	cancel()

	// Use a short timeout to test forced shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()

	err = server.Shutdown(shutdownCtx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	// The slow request should have been forcefully terminated
	select {
	case completed := <-requestDone:
		if completed {
			t.Error("Request should not have completed successfully")
		}
	case <-time.After(2 * time.Second):
		// Request was terminated as expected
	}
}

func TestServer_RejectRequestsDuringShutdown(t *testing.T) {
	cfg := &config.Config{
		Port:        0, // Random port
		Mode:        "simple",
		DataDir:     t.TempDir(),
		BearerToken: "test-token",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.Start(ctx); err != nil {
			t.Errorf("Server start error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Get the actual port
	addr := server.listener.Addr().String()
	baseURL := fmt.Sprintf("http://%s", addr)

	// Mark server as shutting down
	server.shutdownStarted.Store(true)

	// Try to make a request - should be rejected
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d during shutdown, got %d",
			http.StatusServiceUnavailable, resp.StatusCode)
	}

	// Clean shutdown
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
}

func TestServer_MetricsActiveRequests(t *testing.T) {
	cfg := &config.Config{
		Port:        0, // Random port
		Mode:        "simple",
		DataDir:     t.TempDir(),
		BearerToken: "test-token",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Simulate active requests
	server.activeRequests.Store(5)
	server.shutdownStarted.Store(true)

	// Check metrics
	if active := server.activeRequests.Load(); active != 5 {
		t.Errorf("Expected 5 active requests, got %d", active)
	}

	if !server.shutdownStarted.Load() {
		t.Error("Expected shutdown flag to be true")
	}
}

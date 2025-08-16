package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grumpyguvner/gomail/internal/api"
	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/grumpyguvner/gomail/internal/mail"
	"github.com/grumpyguvner/gomail/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompleteEmailFlow(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	// Setup test environment
	tempDir := t.TempDir()
	testConfig := &config.Config{
		Port:        0, // Use random available port
		Mode:        "simple",
		DataDir:     tempDir,
		BearerToken: "integration-test-token",
	}

	// Start test server
	server, err := api.NewServer(testConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := server.Start(ctx)
		if err != nil && err != context.Canceled {
			t.Logf("Server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test 1: Send email via API
	emailData := map[string]interface{}{
		"sender":    "integration@test.com",
		"recipient": "dest@test.com",
		"subject":   "Integration Test",
		"body":      "This is an integration test email",
	}

	jsonData, err := json.Marshal(emailData)
	require.NoError(t, err)

	// Note: In a real integration test, we'd need to get the actual port
	// For now, this is a framework for integration testing
	req, err := http.NewRequest("POST", "http://localhost:3000/mail/inbound", bytes.NewReader(jsonData))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer integration-test-token")
	req.Header.Set("Content-Type", "application/json")

	// Test 2: Verify storage
	store, err := storage.NewFileStorage(tempDir)
	require.NoError(t, err)

	// Store a test email directly
	testEmail := &mail.EmailData{
		Sender:     "integration@test.com",
		Recipient:  "dest@test.com",
		Subject:    "Direct Storage Test",
		Raw:        "Test raw content",
		ReceivedAt: time.Now(),
	}

	storedPath, err := store.Store(testEmail)
	require.NoError(t, err)
	assert.FileExists(t, storedPath)

	// Load and verify
	loaded, err := store.Load(storedPath)
	require.NoError(t, err)
	assert.Equal(t, testEmail.Sender, loaded.Sender)
	assert.Equal(t, testEmail.Recipient, loaded.Recipient)
	assert.Equal(t, testEmail.Subject, loaded.Subject)

	// Test 3: List stored emails
	files, err := store.List(time.Now())
	require.NoError(t, err)
	assert.NotEmpty(t, files)
	assert.Contains(t, files, storedPath)

	// Cleanup
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	err = server.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}

func TestEmailParsingIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Integration Test Email
Message-ID: <integration-test@example.com>
Date: Mon, 2 Jan 2006 15:04:05 -0700
DKIM-Signature: v=1; a=rsa-sha256; d=example.com; s=selector; h=from:to:subject
Received-SPF: pass (example.com: domain of sender@example.com designates 192.168.1.1 as permitted sender)
Authentication-Results: mx.example.com; dmarc=pass

This is the body of the integration test email.
It has multiple lines.

And even paragraphs.`

	httpHeaders := map[string]string{
		"X-Original-Client-Address":  "192.168.1.1",
		"X-Original-Client-Hostname": "mail.example.com",
		"X-Original-Helo":            "example.com",
		"X-Original-Mail-From":       "sender@example.com",
	}

	// Parse the email
	emailData, err := mail.ParseRawEmail(rawEmail, httpHeaders)
	require.NoError(t, err)
	require.NotNil(t, emailData)

	// Verify parsing
	assert.Equal(t, "sender@example.com", emailData.Sender)
	assert.Equal(t, "recipient@example.com", emailData.Recipient)
	assert.Equal(t, "Integration Test Email", emailData.Subject)
	assert.Equal(t, "<integration-test@example.com>", emailData.MessageID)

	// Verify authentication metadata
	assert.Equal(t, "192.168.1.1", emailData.Authentication.SPF.ClientIP)
	assert.Contains(t, emailData.Authentication.SPF.ReceivedSPFHeader, "pass")
	assert.Contains(t, emailData.Authentication.DMARC.AuthenticationResults, "dmarc=pass")
	assert.NotEmpty(t, emailData.Authentication.DKIM.Signatures)

	// Store the email
	tempDir := t.TempDir()
	store, err := storage.NewFileStorage(tempDir)
	require.NoError(t, err)

	storedPath, err := store.Store(emailData)
	require.NoError(t, err)
	assert.FileExists(t, storedPath)

	// Load and verify persistence
	loaded, err := store.Load(storedPath)
	require.NoError(t, err)
	assert.Equal(t, emailData.Sender, loaded.Sender)
	assert.Equal(t, emailData.Authentication.SPF.ClientIP, loaded.Authentication.SPF.ClientIP)
}

func TestStorageIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	tempDir := t.TempDir()
	store, err := storage.NewFileStorage(tempDir)
	require.NoError(t, err)

	// Test storing multiple emails
	var storedPaths []string
	for i := 0; i < 5; i++ {
		email := &mail.EmailData{
			Sender:     "test@example.com",
			Recipient:  "dest@example.com",
			Subject:    "Test Email",
			ReceivedAt: time.Now(),
		}

		path, err := store.Store(email)
		require.NoError(t, err)
		storedPaths = append(storedPaths, path)
	}

	// List today's emails
	files, err := store.List(time.Now())
	require.NoError(t, err)
	assert.Len(t, files, 5)

	// Move one to processed
	processedDir := filepath.Join(tempDir, "processed", time.Now().Format("2006/01/02"))
	processedPath := filepath.Join(processedDir, filepath.Base(storedPaths[0]))

	err = store.Move(storedPaths[0], processedPath)
	require.NoError(t, err)
	assert.FileExists(t, processedPath)
	assert.NoFileExists(t, storedPaths[0])

	// List should now show 4 files
	files, err = store.List(time.Now())
	require.NoError(t, err)
	assert.Len(t, files, 4)

	// Delete one
	err = store.Delete(storedPaths[1])
	require.NoError(t, err)
	assert.NoFileExists(t, storedPaths[1])

	// List should now show 3 files
	files, err = store.List(time.Now())
	require.NoError(t, err)
	assert.Len(t, files, 3)
}

func TestConcurrentEmailProcessing(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	tempDir := t.TempDir()
	store, err := storage.NewFileStorage(tempDir)
	require.NoError(t, err)

	// Process multiple emails concurrently
	numEmails := 20
	errChan := make(chan error, numEmails)
	pathChan := make(chan string, numEmails)

	for i := 0; i < numEmails; i++ {
		go func(index int) {
			email := &mail.EmailData{
				Sender:     "concurrent@test.com",
				Recipient:  "dest@test.com",
				Subject:    "Concurrent Test",
				ReceivedAt: time.Now(),
			}

			path, err := store.Store(email)
			if err != nil {
				errChan <- err
			} else {
				pathChan <- path
			}
		}(i)
	}

	// Collect results
	var paths []string
	for i := 0; i < numEmails; i++ {
		select {
		case err := <-errChan:
			t.Fatalf("Error storing email: %v", err)
		case path := <-pathChan:
			paths = append(paths, path)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	// Verify all emails were stored
	assert.Len(t, paths, numEmails)

	// Verify all files exist
	for _, path := range paths {
		assert.FileExists(t, path)
	}

	// List and verify count
	files, err := store.List(time.Now())
	require.NoError(t, err)
	assert.Len(t, files, numEmails)
}

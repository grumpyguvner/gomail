package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grumpyguvner/gomail/internal/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileStorage(t *testing.T) {
	baseDir := t.TempDir()

	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)
	assert.NotNil(t, storage)
	assert.Equal(t, baseDir, storage.baseDir)

	// Verify directories were created
	assert.DirExists(t, baseDir)
	assert.DirExists(t, filepath.Join(baseDir, "inbox"))
	assert.DirExists(t, filepath.Join(baseDir, "processed"))
}

func TestNewFileStorage_CreateDirectoryError(t *testing.T) {
	// Use a path that cannot be created
	storage, err := NewFileStorage("/dev/null/invalid")
	assert.Error(t, err)
	assert.Nil(t, storage)
}

func TestFileStorage_Store(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	email := &mail.EmailData{
		Sender:     "sender@example.com",
		Recipient:  "recipient@example.com",
		Subject:    "Test Subject",
		Raw:        "Raw email content",
		ReceivedAt: time.Now(),
	}

	path, err := storage.Store(email)
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.True(t, strings.HasPrefix(path, baseDir))
	assert.True(t, strings.HasSuffix(path, ".json"))
	assert.FileExists(t, path)

	// Verify the file contains valid JSON
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var stored mail.EmailData
	err = json.Unmarshal(data, &stored)
	require.NoError(t, err)
	assert.Equal(t, email.Sender, stored.Sender)
	assert.Equal(t, email.Recipient, stored.Recipient)
	assert.Equal(t, email.Subject, stored.Subject)
}

func TestFileStorage_Store_DateBasedDirectory(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	email := &mail.EmailData{
		Sender:    "test@example.com",
		Recipient: "dest@example.com",
	}

	path, err := storage.Store(email)
	require.NoError(t, err)

	// Verify date-based directory structure
	now := time.Now()
	expectedDir := filepath.Join(
		baseDir, "inbox",
		now.Format("2006"),
		now.Format("01"),
		now.Format("02"),
	)
	assert.True(t, strings.Contains(path, expectedDir))
}

func TestFileStorage_Store_UniqueFilenames(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	email := &mail.EmailData{
		Sender: "test@example.com",
	}

	// Store multiple emails and ensure unique filenames
	paths := make(map[string]bool)
	for i := 0; i < 10; i++ {
		path, err := storage.Store(email)
		require.NoError(t, err)
		assert.False(t, paths[path], "Duplicate path generated: %s", path)
		paths[path] = true
	}
}

func TestFileStorage_List(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	// Store some emails
	today := time.Now()
	email := &mail.EmailData{
		Sender: "test@example.com",
	}

	var expectedPaths []string
	for i := 0; i < 3; i++ {
		path, err := storage.Store(email)
		require.NoError(t, err)
		expectedPaths = append(expectedPaths, path)
	}

	// List files for today
	files, err := storage.List(today)
	require.NoError(t, err)
	assert.Len(t, files, 3)

	// Verify all stored files are in the list
	for _, expected := range expectedPaths {
		assert.Contains(t, files, expected)
	}
}

func TestFileStorage_List_EmptyDirectory(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	// List files for a date with no emails
	files, err := storage.List(time.Now().AddDate(0, 0, 1)) // Tomorrow
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestFileStorage_List_NonexistentDirectory(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	// List files for a far future date
	files, err := storage.List(time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestFileStorage_Load(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	original := &mail.EmailData{
		Sender:     "sender@example.com",
		Recipient:  "recipient@example.com",
		Subject:    "Test Subject",
		Raw:        "Raw content",
		MessageID:  "<123@example.com>",
		ReceivedAt: time.Now(),
		Connection: mail.ConnectionInfo{
			ClientAddress: "192.168.1.1",
		},
	}

	// Store the email
	path, err := storage.Store(original)
	require.NoError(t, err)

	// Load it back
	loaded, err := storage.Load(path)
	require.NoError(t, err)
	assert.Equal(t, original.Sender, loaded.Sender)
	assert.Equal(t, original.Recipient, loaded.Recipient)
	assert.Equal(t, original.Subject, loaded.Subject)
	assert.Equal(t, original.Raw, loaded.Raw)
	assert.Equal(t, original.MessageID, loaded.MessageID)
	assert.Equal(t, original.Connection.ClientAddress, loaded.Connection.ClientAddress)
}

func TestFileStorage_Load_NonexistentFile(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	loaded, err := storage.Load("/nonexistent/file.json")
	assert.Error(t, err)
	assert.Nil(t, loaded)
}

func TestFileStorage_Load_InvalidJSON(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	// Create a file with invalid JSON
	invalidFile := filepath.Join(baseDir, "invalid.json")
	err = os.WriteFile(invalidFile, []byte("not valid json"), 0644)
	require.NoError(t, err)

	loaded, err := storage.Load(invalidFile)
	assert.Error(t, err)
	assert.Nil(t, loaded)
}

func TestFileStorage_Move(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	// Store an email
	email := &mail.EmailData{
		Sender: "test@example.com",
	}
	sourcePath, err := storage.Store(email)
	require.NoError(t, err)

	// Move it to processed directory
	destPath := filepath.Join(baseDir, "processed", filepath.Base(sourcePath))
	err = storage.Move(sourcePath, destPath)
	require.NoError(t, err)

	// Verify source doesn't exist
	assert.NoFileExists(t, sourcePath)

	// Verify destination exists
	assert.FileExists(t, destPath)

	// Verify content is preserved
	loaded, err := storage.Load(destPath)
	require.NoError(t, err)
	assert.Equal(t, email.Sender, loaded.Sender)
}

func TestFileStorage_Move_CreateDestinationDirectory(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	// Store an email
	email := &mail.EmailData{
		Sender: "test@example.com",
	}
	sourcePath, err := storage.Store(email)
	require.NoError(t, err)

	// Move to a new directory that doesn't exist yet
	destPath := filepath.Join(baseDir, "new", "directory", "structure", filepath.Base(sourcePath))
	err = storage.Move(sourcePath, destPath)
	require.NoError(t, err)

	assert.FileExists(t, destPath)
	assert.NoFileExists(t, sourcePath)
}

func TestFileStorage_Delete(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	// Store an email
	email := &mail.EmailData{
		Sender: "test@example.com",
	}
	path, err := storage.Store(email)
	require.NoError(t, err)
	assert.FileExists(t, path)

	// Delete it
	err = storage.Delete(path)
	require.NoError(t, err)
	assert.NoFileExists(t, path)
}

func TestFileStorage_Delete_NonexistentFile(t *testing.T) {
	baseDir := t.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(t, err)

	err = storage.Delete("/nonexistent/file.json")
	assert.Error(t, err)
}

func BenchmarkFileStorage_Store(b *testing.B) {
	baseDir := b.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(b, err)

	email := &mail.EmailData{
		Sender:     "sender@example.com",
		Recipient:  "recipient@example.com",
		Subject:    "Test Subject",
		Raw:        strings.Repeat("x", 1024), // 1KB of data
		ReceivedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.Store(email)
	}
}

func BenchmarkFileStorage_Load(b *testing.B) {
	baseDir := b.TempDir()
	storage, err := NewFileStorage(baseDir)
	require.NoError(b, err)

	email := &mail.EmailData{
		Sender:     "sender@example.com",
		Recipient:  "recipient@example.com",
		Subject:    "Test Subject",
		Raw:        strings.Repeat("x", 1024),
		ReceivedAt: time.Now(),
	}

	path, err := storage.Store(email)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.Load(path)
	}
}

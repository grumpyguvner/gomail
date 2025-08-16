package storage

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grumpyguvner/gomail/internal/mail"
)

type FileStorage struct {
	baseDir string
}

func NewFileStorage(baseDir string) (*FileStorage, error) {
	// Ensure base directories exist
	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "inbox"),
		filepath.Join(baseDir, "processed"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &FileStorage{
		baseDir: baseDir,
	}, nil
}

func (fs *FileStorage) Store(email *mail.EmailData) (string, error) {
	// Create date-based directory structure
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")

	dir := filepath.Join(fs.baseDir, "inbox", year, month, day)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate unique filename
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random ID: %w", err)
	}
	
	filename := fmt.Sprintf("msg_%d_%s.json", now.Unix(), hex.EncodeToString(randomBytes))
	fullPath := filepath.Join(dir, filename)

	// Marshal email data to JSON
	data, err := json.MarshalIndent(email, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal email data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fullPath, nil
}

func (fs *FileStorage) List(date time.Time) ([]string, error) {
	year := date.Format("2006")
	month := date.Format("01")
	day := date.Format("02")

	dir := filepath.Join(fs.baseDir, "inbox", year, month, day)
	
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

func (fs *FileStorage) Load(path string) (*mail.EmailData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var email mail.EmailData
	if err := json.Unmarshal(data, &email); err != nil {
		return nil, fmt.Errorf("failed to unmarshal email data: %w", err)
	}

	return &email, nil
}

func (fs *FileStorage) Move(source, destination string) error {
	// Ensure destination directory exists
	destDir := filepath.Dir(destination)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Move file
	if err := os.Rename(source, destination); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

func (fs *FileStorage) Delete(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
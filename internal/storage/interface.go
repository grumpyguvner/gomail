package storage

import (
	"context"
)

// Storage defines the interface for email storage operations
type Storage interface {
	Store(ctx context.Context, emailID string, data []byte) (string, error)
	Retrieve(ctx context.Context, emailID string) ([]byte, error)
	List(ctx context.Context) ([]string, error)
	Delete(ctx context.Context, emailID string) error
}

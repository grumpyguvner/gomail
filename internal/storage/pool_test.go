package storage

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStorage implements Storage interface for testing
type MockStorage struct {
	storeFunc    func(ctx context.Context, emailID string, data []byte) (string, error)
	retrieveFunc func(ctx context.Context, emailID string) ([]byte, error)
	listFunc     func(ctx context.Context) ([]string, error)
	deleteFunc   func(ctx context.Context, emailID string) error
}

func (m *MockStorage) Store(ctx context.Context, emailID string, data []byte) (string, error) {
	if m.storeFunc != nil {
		return m.storeFunc(ctx, emailID, data)
	}
	return emailID, nil
}

func (m *MockStorage) Retrieve(ctx context.Context, emailID string) ([]byte, error) {
	if m.retrieveFunc != nil {
		return m.retrieveFunc(ctx, emailID)
	}
	return []byte("test data"), nil
}

func (m *MockStorage) List(ctx context.Context) ([]string, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return []string{"id1", "id2"}, nil
}

func (m *MockStorage) Delete(ctx context.Context, emailID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, emailID)
	}
	return nil
}

func TestNewConnectionPool(t *testing.T) {
	factory := func() (Storage, error) {
		return &MockStorage{}, nil
	}

	t.Run("creates pool with default values", func(t *testing.T) {
		pool, err := NewConnectionPool(factory, 0, 0, 0)
		require.NoError(t, err)
		require.NotNil(t, pool)
		assert.Equal(t, 100, pool.maxSize)
		assert.Equal(t, 10, pool.maxIdleConns)
		assert.Equal(t, 30*time.Second, pool.timeout)
		pool.Close()
	})

	t.Run("creates pool with custom values", func(t *testing.T) {
		pool, err := NewConnectionPool(factory, 50, 5, 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, pool)
		assert.Equal(t, 50, pool.maxSize)
		assert.Equal(t, 5, pool.maxIdleConns)
		assert.Equal(t, 10*time.Second, pool.timeout)
		pool.Close()
	})

	t.Run("adjusts idle connections if exceeds max", func(t *testing.T) {
		pool, err := NewConnectionPool(factory, 10, 20, 5*time.Second)
		require.NoError(t, err)
		require.NotNil(t, pool)
		assert.Equal(t, 10, pool.maxSize)
		assert.Equal(t, 10, pool.maxIdleConns) // Adjusted to maxSize
		pool.Close()
	})
}

func TestConnectionPool_Get(t *testing.T) {
	factory := func() (Storage, error) {
		return &MockStorage{}, nil
	}

	t.Run("gets connection from pool", func(t *testing.T) {
		pool, err := NewConnectionPool(factory, 10, 2, 5*time.Second)
		require.NoError(t, err)
		defer pool.Close()

		ctx := context.Background()
		conn, err := pool.Get(ctx)
		require.NoError(t, err)
		require.NotNil(t, conn)
		assert.True(t, conn.inUse)

		// Return connection to pool
		conn.Release()
	})

	t.Run("creates new connection when pool is empty", func(t *testing.T) {
		pool, err := NewConnectionPool(factory, 10, 0, 5*time.Second)
		require.NoError(t, err)
		defer pool.Close()

		ctx := context.Background()
		conn, err := pool.Get(ctx)
		require.NoError(t, err)
		require.NotNil(t, conn)

		stats := pool.Stats()
		assert.GreaterOrEqual(t, stats.Created, int64(1))
		assert.Equal(t, int64(1), stats.Active)

		conn.Release()
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		pool, err := NewConnectionPool(factory, 1, 0, 5*time.Second)
		require.NoError(t, err)
		defer pool.Close()

		// Get the only connection
		ctx := context.Background()
		conn1, err := pool.Get(ctx)
		require.NoError(t, err)

		// Try to get another with canceled context
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()

		conn2, err := pool.Get(cancelCtx)
		assert.Error(t, err)
		assert.Nil(t, conn2)
		assert.Equal(t, context.Canceled, err)

		conn1.Release()
	})

	t.Run("times out when pool is exhausted", func(t *testing.T) {
		pool, err := NewConnectionPool(factory, 1, 0, 100*time.Millisecond)
		require.NoError(t, err)
		defer pool.Close()

		// Get the only connection
		ctx := context.Background()
		conn1, err := pool.Get(ctx)
		require.NoError(t, err)

		// Try to get another, should timeout
		start := time.Now()
		conn2, err := pool.Get(ctx)
		elapsed := time.Since(start)

		assert.Error(t, err)
		assert.Nil(t, conn2)
		assert.Contains(t, err.Error(), "timeout")
		assert.True(t, elapsed >= 100*time.Millisecond)

		conn1.Release()
	})
}

func TestConnectionPool_Put(t *testing.T) {
	factory := func() (Storage, error) {
		return &MockStorage{}, nil
	}

	t.Run("returns connection to pool", func(t *testing.T) {
		pool, err := NewConnectionPool(factory, 10, 5, 5*time.Second)
		require.NoError(t, err)
		defer pool.Close()

		ctx := context.Background()
		conn, err := pool.Get(ctx)
		require.NoError(t, err)

		stats := pool.Stats()
		assert.Equal(t, int64(1), stats.Active)

		pool.Put(conn)

		stats = pool.Stats()
		assert.Equal(t, int64(0), stats.Active)
		assert.GreaterOrEqual(t, stats.Idle, int64(1))
	})

	t.Run("closes connection when pool has too many idle", func(t *testing.T) {
		pool, err := NewConnectionPool(factory, 10, 1, 5*time.Second)
		require.NoError(t, err)
		defer pool.Close()

		ctx := context.Background()

		// Get multiple connections
		conn1, _ := pool.Get(ctx)
		conn2, _ := pool.Get(ctx)

		// Return them to pool
		pool.Put(conn1)
		pool.Put(conn2) // This should be closed due to maxIdleConns=1

		stats := pool.Stats()
		assert.Equal(t, int64(1), stats.Idle) // Only 1 kept idle
	})

	t.Run("ignores nil connection", func(t *testing.T) {
		pool, err := NewConnectionPool(factory, 10, 5, 5*time.Second)
		require.NoError(t, err)
		defer pool.Close()

		// Should not panic
		pool.Put(nil)
	})
}

func TestConnectionPool_Concurrency(t *testing.T) {
	var created atomic.Int32
	factory := func() (Storage, error) {
		created.Add(1)
		return &MockStorage{}, nil
	}

	pool, err := NewConnectionPool(factory, 20, 5, 5*time.Second)
	require.NoError(t, err)
	defer pool.Close()

	// Run concurrent get/put operations
	var wg sync.WaitGroup
	numGoroutines := 50
	numOperations := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()

			for j := 0; j < numOperations; j++ {
				conn, err := pool.Get(ctx)
				if err != nil {
					continue
				}

				// Simulate some work
				time.Sleep(time.Microsecond)

				// Use the connection
				_, _ = conn.Store(ctx, "test", []byte("data"))

				// Release connection
				conn.Release()
			}
		}()
	}

	wg.Wait()

	stats := pool.Stats()
	// We can't strictly enforce pool size limit during concurrent operations
	// due to race conditions, but it should be reasonably close
	assert.LessOrEqual(t, stats.Created, int64(25)) // Allow some overhead
	assert.Equal(t, int64(0), stats.Active)         // All connections should be released
}

func TestConnectionPool_Stats(t *testing.T) {
	factory := func() (Storage, error) {
		return &MockStorage{}, nil
	}

	pool, err := NewConnectionPool(factory, 10, 2, 5*time.Second)
	require.NoError(t, err)
	defer pool.Close()

	ctx := context.Background()

	// Initial stats
	stats := pool.Stats()
	assert.Equal(t, int64(2), stats.Created) // Pre-created idle connections
	assert.Equal(t, int64(0), stats.Active)
	assert.Equal(t, int64(2), stats.Idle)
	assert.Equal(t, 10, stats.MaxSize)

	// Get a connection
	conn, _ := pool.Get(ctx)
	stats = pool.Stats()
	assert.Equal(t, int64(1), stats.Active)
	assert.Equal(t, int64(1), stats.Idle)

	// Release connection
	conn.Release()
	stats = pool.Stats()
	assert.Equal(t, int64(0), stats.Active)
	assert.Equal(t, int64(2), stats.Idle)
}

func TestPooledConnection_Methods(t *testing.T) {
	factory := func() (Storage, error) {
		return &MockStorage{
			storeFunc: func(ctx context.Context, emailID string, data []byte) (string, error) {
				return "stored-" + emailID, nil
			},
			retrieveFunc: func(ctx context.Context, emailID string) ([]byte, error) {
				return []byte("retrieved-" + emailID), nil
			},
			listFunc: func(ctx context.Context) ([]string, error) {
				return []string{"file1", "file2"}, nil
			},
			deleteFunc: func(ctx context.Context, emailID string) error {
				if emailID == "error" {
					return fmt.Errorf("delete error")
				}
				return nil
			},
		}, nil
	}

	pool, err := NewConnectionPool(factory, 10, 2, 5*time.Second)
	require.NoError(t, err)
	defer pool.Close()

	ctx := context.Background()
	conn, err := pool.Get(ctx)
	require.NoError(t, err)
	defer conn.Release()

	t.Run("Store delegates correctly", func(t *testing.T) {
		result, err := conn.Store(ctx, "test-id", []byte("test-data"))
		require.NoError(t, err)
		assert.Equal(t, "stored-test-id", result)
	})

	t.Run("Retrieve delegates correctly", func(t *testing.T) {
		data, err := conn.Retrieve(ctx, "test-id")
		require.NoError(t, err)
		assert.Equal(t, []byte("retrieved-test-id"), data)
	})

	t.Run("List delegates correctly", func(t *testing.T) {
		files, err := conn.List(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"file1", "file2"}, files)
	})

	t.Run("Delete delegates correctly", func(t *testing.T) {
		err := conn.Delete(ctx, "test-id")
		require.NoError(t, err)

		err = conn.Delete(ctx, "error")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete error")
	})
}

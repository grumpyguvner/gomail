package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConnectionPool manages a pool of storage connections
type ConnectionPool struct {
	mu           sync.RWMutex
	connections  chan *PooledConnection
	factory      ConnectionFactory
	maxSize      int
	maxIdleConns int
	timeout      time.Duration

	// Metrics
	created      int64
	active       int64
	idle         int64
	waitCount    int64
	waitDuration time.Duration
}

// PooledConnection wraps a storage connection with pool metadata
type PooledConnection struct {
	storage    Storage
	pool       *ConnectionPool
	createdAt  time.Time
	lastUsedAt time.Time
	inUse      bool
}

// ConnectionFactory creates new storage connections
type ConnectionFactory func() (Storage, error)

// NewConnectionPool creates a new connection pool
func NewConnectionPool(factory ConnectionFactory, maxSize, maxIdleConns int, timeout time.Duration) (*ConnectionPool, error) {
	if maxSize <= 0 {
		maxSize = 100
	}
	if maxIdleConns <= 0 {
		maxIdleConns = 10
	}
	if maxIdleConns > maxSize {
		maxIdleConns = maxSize
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	pool := &ConnectionPool{
		connections:  make(chan *PooledConnection, maxSize),
		factory:      factory,
		maxSize:      maxSize,
		maxIdleConns: maxIdleConns,
		timeout:      timeout,
	}

	// Pre-create minimum idle connections
	for i := 0; i < maxIdleConns; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			// Log error but don't fail pool creation
			continue
		}
		pool.connections <- conn
	}
	pool.idle = int64(maxIdleConns)

	return pool, nil
}

// Get retrieves a connection from the pool
func (p *ConnectionPool) Get(ctx context.Context) (*PooledConnection, error) {
	startTime := time.Now()
	p.mu.Lock()
	p.waitCount++
	p.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case conn := <-p.connections:
		// Got a connection from the pool
		p.mu.Lock()
		conn.inUse = true
		conn.lastUsedAt = time.Now()
		p.idle--
		p.active++
		p.waitDuration += time.Since(startTime)
		p.mu.Unlock()
		return conn, nil
	default:
		// No connections available, try to create a new one
		p.mu.Lock()
		if p.created < int64(p.maxSize) {
			p.mu.Unlock()
			conn, err := p.createConnection()
			if err != nil {
				return nil, err
			}
			p.mu.Lock()
			conn.inUse = true
			conn.lastUsedAt = time.Now()
			p.active++
			p.waitDuration += time.Since(startTime)
			p.mu.Unlock()
			return conn, nil
		}
		p.mu.Unlock()

		// Wait for a connection to become available
		timer := time.NewTimer(p.timeout)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, fmt.Errorf("connection pool timeout after %v", p.timeout)
		case conn := <-p.connections:
			p.mu.Lock()
			conn.inUse = true
			conn.lastUsedAt = time.Now()
			p.idle--
			p.active++
			p.waitDuration += time.Since(startTime)
			p.mu.Unlock()
			return conn, nil
		}
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(conn *PooledConnection) {
	if conn == nil || conn.pool != p {
		return
	}

	p.mu.Lock()
	conn.inUse = false
	p.active--

	// Check if we should keep this connection
	if p.idle < int64(p.maxIdleConns) {
		p.idle++
		p.mu.Unlock()

		select {
		case p.connections <- conn:
			// Successfully returned to pool
		default:
			// Pool is full, close the connection
			p.mu.Lock()
			p.idle--
			p.created--
			p.mu.Unlock()
		}
	} else {
		// Too many idle connections, close this one
		p.created--
		p.mu.Unlock()
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.connections)

	// Drain the channel
	for range p.connections {
		// Connections will be garbage collected
	}

	p.created = 0
	p.active = 0
	p.idle = 0

	return nil
}

// Stats returns pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	avgWait := time.Duration(0)
	if p.waitCount > 0 {
		avgWait = p.waitDuration / time.Duration(p.waitCount)
	}

	return PoolStats{
		Created:     p.created,
		Active:      p.active,
		Idle:        p.idle,
		MaxSize:     p.maxSize,
		WaitCount:   p.waitCount,
		AvgWaitTime: avgWait,
	}
}

// createConnection creates a new storage connection
func (p *ConnectionPool) createConnection() (*PooledConnection, error) {
	storage, err := p.factory()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage connection: %w", err)
	}

	p.mu.Lock()
	p.created++
	p.mu.Unlock()

	return &PooledConnection{
		storage:   storage,
		pool:      p,
		createdAt: time.Now(),
	}, nil
}

// PoolStats contains pool statistics
type PoolStats struct {
	Created     int64
	Active      int64
	Idle        int64
	MaxSize     int
	WaitCount   int64
	AvgWaitTime time.Duration
}

// Storage methods delegation for PooledConnection

// Store delegates to the underlying storage
func (pc *PooledConnection) Store(ctx context.Context, emailID string, data []byte) (string, error) {
	return pc.storage.Store(ctx, emailID, data)
}

// Retrieve delegates to the underlying storage
func (pc *PooledConnection) Retrieve(ctx context.Context, emailID string) ([]byte, error) {
	return pc.storage.Retrieve(ctx, emailID)
}

// List delegates to the underlying storage
func (pc *PooledConnection) List(ctx context.Context) ([]string, error) {
	return pc.storage.List(ctx)
}

// Delete delegates to the underlying storage
func (pc *PooledConnection) Delete(ctx context.Context, emailID string) error {
	return pc.storage.Delete(ctx, emailID)
}

// Release returns the connection to the pool
func (pc *PooledConnection) Release() {
	if pc.pool != nil {
		pc.pool.Put(pc)
	}
}

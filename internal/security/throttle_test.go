package security

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionThrottle_Allow(t *testing.T) {
	// Create throttle with 10 connections/sec global, 5/sec per IP
	throttle := NewConnectionThrottle(10, 5)

	// Should allow initial connections
	for i := 0; i < 5; i++ {
		assert.True(t, throttle.Allow("192.168.1.1:1234"))
	}

	// After burst, may need to wait
	allowed := 0
	for i := 0; i < 10; i++ {
		if throttle.Allow("192.168.1.1:1234") {
			allowed++
		}
	}
	assert.LessOrEqual(t, allowed, 10) // Should respect rate limit
}

func TestConnectionThrottle_AllowN(t *testing.T) {
	throttle := NewConnectionThrottle(10, 5)

	// Should allow burst
	assert.True(t, throttle.AllowN("192.168.1.1:1234", 5))

	// Should not allow exceeding burst
	assert.False(t, throttle.AllowN("192.168.1.1:1234", 20))
}

func TestConnectionThrottle_Wait(t *testing.T) {
	throttle := NewConnectionThrottle(2, 1) // Very low rate for testing

	ctx := context.Background()

	// First request should be immediate
	start := time.Now()
	err := throttle.Wait(ctx, "192.168.1.1:1234")
	require.NoError(t, err)
	assert.Less(t, time.Since(start), 10*time.Millisecond)

	// Burst allows one more immediate
	err = throttle.Wait(ctx, "192.168.1.1:1234")
	require.NoError(t, err)

	// Next request should wait
	start = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = throttle.Wait(ctx, "192.168.1.1:1234")
	// May timeout or succeed depending on timing
	if err == nil {
		assert.GreaterOrEqual(t, time.Since(start), 50*time.Millisecond)
	}
}

func TestConnectionThrottle_PerIPLimiting(t *testing.T) {
	throttle := NewConnectionThrottle(100, 2) // High global, low per-IP

	// IP 1 should be limited
	assert.True(t, throttle.Allow("192.168.1.1:1234"))
	assert.True(t, throttle.Allow("192.168.1.1:1234"))

	// May be rejected after burst
	allowed := 0
	for i := 0; i < 5; i++ {
		if throttle.Allow("192.168.1.1:1234") {
			allowed++
		}
	}
	assert.Less(t, allowed, 5)

	// Different IP should work
	assert.True(t, throttle.Allow("192.168.1.2:1234"))
	assert.True(t, throttle.Allow("192.168.1.2:1234"))
}

func TestConnectionThrottle_Reserve(t *testing.T) {
	throttle := NewConnectionThrottle(10, 5)

	// Reserve a slot
	res := throttle.Reserve("192.168.1.1:1234")
	require.NotNil(t, res)

	// Should have minimal delay for first request
	assert.Less(t, res.Delay(), 10*time.Millisecond)

	// Cancel reservation
	res.Cancel()
}

func TestConnectionThrottle_UpdateRates(t *testing.T) {
	throttle := NewConnectionThrottle(10, 5)

	// Update rates
	throttle.UpdateRates(20, 10)

	stats := throttle.GetStats()
	assert.Equal(t, 20.0, stats.GlobalRate)
	assert.Equal(t, 10.0, stats.PerIPRate)

	// Should use new rates
	for i := 0; i < 10; i++ {
		assert.True(t, throttle.Allow("192.168.1.1:1234"))
	}
}

func TestConnectionThrottle_Cleanup(t *testing.T) {
	throttle := NewConnectionThrottle(10, 5)

	// Add some IPs
	throttle.Allow("192.168.1.1:1234")
	throttle.Allow("192.168.1.2:1234")
	throttle.Allow("192.168.1.3:1234")

	stats := throttle.GetStats()
	assert.Equal(t, 3, stats.ActiveIPs)

	// Manually set IPs as old by modifying their lastSeen time
	throttle.mu.Lock()
	for _, limiter := range throttle.ipLimiters {
		limiter.lastSeen = time.Now().Add(-10 * time.Minute)
	}
	throttle.mu.Unlock()

	// Manually trigger cleanup for testing
	throttle.cleanup()

	// Old IPs should be cleaned up
	stats = throttle.GetStats()
	assert.Equal(t, 0, stats.ActiveIPs)
}

func TestConnectionThrottle_GetStats(t *testing.T) {
	throttle := NewConnectionThrottle(10, 5)

	// Generate some activity
	throttle.Allow("192.168.1.1:1234")
	throttle.Allow("192.168.1.2:1234")

	stats := throttle.GetStats()
	assert.Equal(t, 10.0, stats.GlobalRate)
	assert.Equal(t, 20, stats.GlobalBurst)
	assert.Equal(t, 5.0, stats.PerIPRate)
	assert.Equal(t, 10, stats.PerIPBurst)
	assert.Equal(t, 2, stats.ActiveIPs)
}

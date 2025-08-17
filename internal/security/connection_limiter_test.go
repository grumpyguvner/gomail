package security

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionLimiter_Accept(t *testing.T) {
	limiter := NewConnectionLimiter(2, 5, 1*time.Hour)

	// Test per-IP limit
	assert.True(t, limiter.Accept("192.168.1.1:1234"))
	assert.True(t, limiter.Accept("192.168.1.1:1235"))
	assert.False(t, limiter.Accept("192.168.1.1:1236")) // Should hit per-IP limit

	// Different IP should work
	assert.True(t, limiter.Accept("192.168.1.2:1234"))
	assert.True(t, limiter.Accept("192.168.1.2:1235"))

	// Test total limit
	assert.True(t, limiter.Accept("192.168.1.3:1234"))
	assert.False(t, limiter.Accept("192.168.1.4:1234")) // Should hit total limit
}

func TestConnectionLimiter_Release(t *testing.T) {
	limiter := NewConnectionLimiter(1, 3, 1*time.Hour)

	// Accept a connection
	require.True(t, limiter.Accept("192.168.1.1:1234"))
	assert.False(t, limiter.Accept("192.168.1.1:1235")) // Should hit per-IP limit

	// Release the connection
	limiter.Release("192.168.1.1:1234")

	// Should be able to accept again
	assert.True(t, limiter.Accept("192.168.1.1:1235"))
}

func TestConnectionLimiter_Ban(t *testing.T) {
	limiter := NewConnectionLimiter(5, 10, 100*time.Millisecond)

	// Ban an IP
	limiter.Ban("192.168.1.1", 100*time.Millisecond)

	// Should be banned
	assert.True(t, limiter.IsBanned("192.168.1.1"))
	assert.False(t, limiter.Accept("192.168.1.1:1234"))

	// Wait for ban to expire
	time.Sleep(150 * time.Millisecond)

	// Should no longer be banned
	assert.False(t, limiter.IsBanned("192.168.1.1"))
	assert.True(t, limiter.Accept("192.168.1.1:1234"))
}

func TestConnectionLimiter_AutoBan(t *testing.T) {
	limiter := NewConnectionLimiter(1, 10, 100*time.Millisecond)
	limiter.banThreshold = 3 // Lower threshold for testing

	ip := "192.168.1.1:1234"

	// Accept first connection
	assert.True(t, limiter.Accept(ip))

	// Trigger violations
	for i := 0; i < 3; i++ {
		assert.False(t, limiter.Accept(ip))
	}

	// Should now be auto-banned
	assert.True(t, limiter.IsBanned("192.168.1.1"))
}

func TestConnectionLimiter_Unban(t *testing.T) {
	limiter := NewConnectionLimiter(5, 10, 1*time.Hour)

	// Ban and then unban
	limiter.Ban("192.168.1.1", 1*time.Hour)
	assert.True(t, limiter.IsBanned("192.168.1.1"))

	limiter.Unban("192.168.1.1")
	assert.False(t, limiter.IsBanned("192.168.1.1"))
	assert.True(t, limiter.Accept("192.168.1.1:1234"))
}

func TestConnectionLimiter_GetBannedIPs(t *testing.T) {
	limiter := NewConnectionLimiter(5, 10, 1*time.Hour)

	// Ban multiple IPs
	limiter.Ban("192.168.1.1", 1*time.Hour)
	limiter.Ban("192.168.1.2", 2*time.Hour)

	banned := limiter.GetBannedIPs()
	assert.Len(t, banned, 2)
	assert.Contains(t, banned, "192.168.1.1")
	assert.Contains(t, banned, "192.168.1.2")
}

func TestConnectionLimiter_GetConnectionStats(t *testing.T) {
	limiter := NewConnectionLimiter(2, 5, 1*time.Hour)

	// Accept some connections
	limiter.Accept("192.168.1.1:1234")
	limiter.Accept("192.168.1.1:1235")
	limiter.Accept("192.168.1.2:1234")

	stats := limiter.GetConnectionStats()
	assert.Equal(t, 3, stats.TotalConnections)
	assert.Equal(t, 5, stats.MaxTotal)
	assert.Equal(t, 2, stats.MaxPerIP)
	assert.Equal(t, 2, stats.ConnectionsByIP["192.168.1.1"])
	assert.Equal(t, 1, stats.ConnectionsByIP["192.168.1.2"])
}

func TestConnectionLimiter_Cleanup(t *testing.T) {
	limiter := NewConnectionLimiter(5, 10, 50*time.Millisecond)

	// Ban an IP with short duration
	limiter.Ban("192.168.1.1", 50*time.Millisecond)
	assert.True(t, limiter.IsBanned("192.168.1.1"))

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)
	limiter.cleanup()

	// Should be cleaned up
	assert.False(t, limiter.IsBanned("192.168.1.1"))
}

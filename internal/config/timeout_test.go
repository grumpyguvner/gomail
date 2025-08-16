package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutValidation(t *testing.T) {
	tests := []struct {
		name           string
		readTimeout    int
		writeTimeout   int
		idleTimeout    int
		handlerTimeout int
		expectErrors   []string
	}{
		{
			name:           "valid timeouts",
			readTimeout:    30,
			writeTimeout:   30,
			idleTimeout:    60,
			handlerTimeout: 25,
			expectErrors:   nil,
		},
		{
			name:           "negative read timeout",
			readTimeout:    -1,
			writeTimeout:   30,
			idleTimeout:    60,
			handlerTimeout: 25,
			expectErrors:   []string{"read_timeout: cannot be negative"},
		},
		{
			name:           "negative write timeout",
			readTimeout:    30,
			writeTimeout:   -5,
			idleTimeout:    60,
			handlerTimeout: 25,
			expectErrors:   []string{"write_timeout: cannot be negative"},
		},
		{
			name:           "negative idle timeout",
			readTimeout:    30,
			writeTimeout:   30,
			idleTimeout:    -10,
			handlerTimeout: 25,
			expectErrors:   []string{"idle_timeout: cannot be negative"},
		},
		{
			name:           "negative handler timeout",
			readTimeout:    30,
			writeTimeout:   30,
			idleTimeout:    60,
			handlerTimeout: -1,
			expectErrors:   []string{"handler_timeout: cannot be negative"},
		},
		{
			name:           "excessive read timeout",
			readTimeout:    500,
			writeTimeout:   30,
			idleTimeout:    60,
			handlerTimeout: 25,
			expectErrors:   []string{"read_timeout: unreasonably high timeout (>300s)"},
		},
		{
			name:           "excessive write timeout",
			readTimeout:    30,
			writeTimeout:   400,
			idleTimeout:    60,
			handlerTimeout: 25,
			expectErrors:   []string{"write_timeout: unreasonably high timeout (>300s)"},
		},
		{
			name:           "excessive idle timeout",
			readTimeout:    30,
			writeTimeout:   30,
			idleTimeout:    700,
			handlerTimeout: 25,
			expectErrors:   []string{"idle_timeout: unreasonably high timeout (>600s)"},
		},
		{
			name:           "excessive handler timeout",
			readTimeout:    30,
			writeTimeout:   30,
			idleTimeout:    60,
			handlerTimeout: 350,
			expectErrors:   []string{"handler_timeout: unreasonably high timeout (>300s)"},
		},
		{
			name:           "handler timeout >= read timeout",
			readTimeout:    30,
			writeTimeout:   30,
			idleTimeout:    60,
			handlerTimeout: 30,
			expectErrors:   []string{"handler_timeout: should be less than read_timeout"},
		},
		{
			name:           "handler timeout > read timeout",
			readTimeout:    30,
			writeTimeout:   30,
			idleTimeout:    60,
			handlerTimeout: 35,
			expectErrors:   []string{"handler_timeout: should be less than read_timeout"},
		},
		{
			name:           "multiple timeout errors",
			readTimeout:    -1,
			writeTimeout:   -1,
			idleTimeout:    -1,
			handlerTimeout: -1,
			expectErrors: []string{
				"read_timeout: cannot be negative",
				"write_timeout: cannot be negative",
				"idle_timeout: cannot be negative",
				"handler_timeout: cannot be negative",
			},
		},
		{
			name:           "zero timeouts (disabled)",
			readTimeout:    0,
			writeTimeout:   0,
			idleTimeout:    0,
			handlerTimeout: 0,
			expectErrors:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:           3000,
				Mode:           "simple",
				DataDir:        "/opt/mailserver/data",
				ReadTimeout:    tt.readTimeout,
				WriteTimeout:   tt.writeTimeout,
				IdleTimeout:    tt.idleTimeout,
				HandlerTimeout: tt.handlerTimeout,
			}

			err := cfg.ValidateSchema()
			if len(tt.expectErrors) == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				for _, expectedErr := range tt.expectErrors {
					assert.Contains(t, err.Error(), expectedErr)
				}
			}
		})
	}
}

func TestConnectionPoolValidation(t *testing.T) {
	tests := []struct {
		name           string
		maxConnections int
		maxIdleConns   int
		expectErrors   []string
	}{
		{
			name:           "valid pool settings",
			maxConnections: 100,
			maxIdleConns:   10,
			expectErrors:   nil,
		},
		{
			name:           "negative max connections",
			maxConnections: -1,
			maxIdleConns:   10,
			expectErrors:   []string{"max_connections: cannot be negative"},
		},
		{
			name:           "negative max idle connections",
			maxConnections: 100,
			maxIdleConns:   -5,
			expectErrors:   []string{"max_idle_conns: cannot be negative"},
		},
		{
			name:           "excessive max connections",
			maxConnections: 15000,
			maxIdleConns:   10,
			expectErrors:   []string{"max_connections: unreasonably high (>10000)"},
		},
		{
			name:           "idle exceeds max connections",
			maxConnections: 50,
			maxIdleConns:   100,
			expectErrors:   []string{"max_idle_conns: cannot exceed max_connections"},
		},
		{
			name:           "zero values (defaults)",
			maxConnections: 0,
			maxIdleConns:   0,
			expectErrors:   nil,
		},
		{
			name:           "multiple errors",
			maxConnections: -10,
			maxIdleConns:   -5,
			expectErrors: []string{
				"max_connections: cannot be negative",
				"max_idle_conns: cannot be negative",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:           3000,
				Mode:           "simple",
				DataDir:        "/opt/mailserver/data",
				MaxConnections: tt.maxConnections,
				MaxIdleConns:   tt.maxIdleConns,
			}

			err := cfg.ValidateSchema()
			if len(tt.expectErrors) == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				for _, expectedErr := range tt.expectErrors {
					assert.Contains(t, err.Error(), expectedErr)
				}
			}
		})
	}
}

func TestSchemaValidator_TimeoutValidation(t *testing.T) {
	v := NewSchemaValidator()

	t.Run("validates all timeout fields", func(t *testing.T) {
		v.validateTimeouts(30, 30, 60, 25)
		assert.False(t, v.HasErrors())
		assert.Empty(t, v.Errors())
	})

	t.Run("catches handler timeout issues", func(t *testing.T) {
		v = NewSchemaValidator()
		v.validateTimeouts(30, 30, 60, 45)
		assert.True(t, v.HasErrors())
		assert.Contains(t, v.ErrorMessage(), "handler_timeout: should be less than read_timeout")
	})
}

func TestSchemaValidator_ConnectionPoolValidation(t *testing.T) {
	v := NewSchemaValidator()

	t.Run("validates valid pool settings", func(t *testing.T) {
		v.validateConnectionPool(100, 10)
		assert.False(t, v.HasErrors())
		assert.Empty(t, v.Errors())
	})

	t.Run("catches invalid pool settings", func(t *testing.T) {
		v = NewSchemaValidator()
		v.validateConnectionPool(50, 100)
		assert.True(t, v.HasErrors())
		assert.Contains(t, v.ErrorMessage(), "max_idle_conns: cannot exceed max_connections")
	})
}

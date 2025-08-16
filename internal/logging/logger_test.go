package logging

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		logLevel string
		wantErr  bool
	}{
		{
			name:    "production mode",
			mode:    "production",
			wantErr: false,
		},
		{
			name:    "development mode",
			mode:    "development",
			wantErr: false,
		},
		{
			name:     "with custom log level",
			mode:     "production",
			logLevel: "debug",
			wantErr:  false,
		},
		{
			name:     "with invalid log level still works",
			mode:     "production",
			logLevel: "invalid",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.wantErr {
					assert.NotNil(t, r, "Expected panic but got none")
				} else {
					assert.Nil(t, r, "Unexpected panic: %v", r)
				}
			}()

			if tt.logLevel != "" {
				os.Setenv("MAIL_LOG_LEVEL", tt.logLevel)
				defer os.Unsetenv("MAIL_LOG_LEVEL")
			}

			InitLogger(tt.mode)
			assert.NotNil(t, logger)
		})
	}
}

func TestGet(t *testing.T) {
	InitLogger("production")
	l := Get()
	require.NotNil(t, l)
	assert.IsType(t, &zap.SugaredLogger{}, l)
}

func TestWith(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core).Sugar()
	logger = testLogger

	l := With("key", "value")
	l.Info("test message")

	entries := recorded.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "test message", entries[0].Message)

	found := false
	for _, field := range entries[0].Context {
		if field.Key == "key" && field.String == "value" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find field with key='key' and value='value'")
}

func TestWithRequestID(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core).Sugar()
	logger = testLogger

	l := WithRequestID("test-request-id")
	l.Info("test message")

	entries := recorded.All()
	require.Len(t, entries, 1)

	found := false
	for _, field := range entries[0].Context {
		if field.Key == "request_id" && field.String == "test-request-id" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find request_id field")
}

func TestLogFileConfiguration(t *testing.T) {
	tempFile := "/tmp/test-log.txt"
	os.Setenv("MAIL_LOG_FILE", tempFile)
	defer os.Unsetenv("MAIL_LOG_FILE")
	defer os.Remove(tempFile)

	InitLogger("production")
	logger.Info("test log to file")
	Sync()

	_, err := os.Stat(tempFile)
	assert.NoError(t, err, "Log file should be created")
}

package logging

import (
	internallogging "github.com/grumpyguvner/gomail/internal/logging"
	"go.uber.org/zap"
)

// Logger wraps the internal logging for webadmin
type Logger struct {
	sugar *zap.SugaredLogger
}

// NewLogger creates a new logger instance
func NewLogger(level string, output string) (*Logger, error) {
	// Initialize the internal logger
	internallogging.InitLogger("production")
	
	return &Logger{
		sugar: internallogging.Get(),
	}, nil
}

// Info logs an info message
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

// Error logs an error message
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}
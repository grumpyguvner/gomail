package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

func init() {
	InitLogger("production")
}

func InitLogger(mode string) {
	var config zap.Config

	if mode == "development" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.MessageKey = "message"
		config.EncoderConfig.LevelKey = "level"
	}

	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	if logFile := os.Getenv("MAIL_LOG_FILE"); logFile != "" {
		config.OutputPaths = append(config.OutputPaths, logFile)
		config.ErrorOutputPaths = append(config.ErrorOutputPaths, logFile)
	}

	logLevel := os.Getenv("MAIL_LOG_LEVEL")
	if logLevel != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(logLevel)); err == nil {
			config.Level = zap.NewAtomicLevelAt(level)
		}
	}

	l, err := config.Build()
	if err != nil {
		panic(err)
	}

	logger = l.Sugar()
}

func Get() *zap.SugaredLogger {
	return logger
}

func With(args ...interface{}) *zap.SugaredLogger {
	return logger.With(args...)
}

func WithRequestID(requestID string) *zap.SugaredLogger {
	return logger.With("request_id", requestID)
}

func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}

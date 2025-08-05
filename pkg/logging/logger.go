// Package logging - Pure Zap implementation for FlexCore
package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Global Zap logger instance
var Logger *zap.Logger

// Initialize sets up the global Zap logger
func Initialize(environment, level string) error {
	var config zap.Config

	switch environment {
	case "production":
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "development":
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	default:
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	// Parse level from string
	if level != "" {
		var logLevel zapcore.Level
		if err := logLevel.UnmarshalText([]byte(level)); err == nil {
			config.Level = zap.NewAtomicLevelAt(logLevel)
		}
	}

	// Enterprise logging configuration
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.NameKey = "logger"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.StacktraceKey = "stacktrace"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Build logger
	logger, err := config.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	Logger = logger
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	if Logger == nil {
		// Initialize with default config if not already initialized
		if err := Initialize("development", "info"); err != nil {
			// Fallback to no-op logger if initialization fails
			Logger = zap.NewNop()
		}
	}
	return Logger
}

// F creates a zap.Field for logging (compatibility helper)
func F(key string, value interface{}) zap.Field {
	switch v := value.(type) {
	case string:
		return zap.String(key, v)
	case int:
		return zap.Int(key, v)
	case int64:
		return zap.Int64(key, v)
	case bool:
		return zap.Bool(key, v)
	case error:
		return zap.Error(v)
	default:
		return zap.Any(key, v)
	}
}

// Close flushes and closes the logger
func Close() error {
	if Logger != nil {
		return Logger.Sync()
	}
	return nil
}

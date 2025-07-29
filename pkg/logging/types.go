package logging

import "go.uber.org/zap"

// LoggerInterface represents a logger interface that wraps zap.Logger
type LoggerInterface interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	With(fields ...zap.Field) LoggerInterface
	Sync() error
}

// ZapLogger wraps zap.Logger to implement LoggerInterface
type ZapLogger struct {
	logger *zap.Logger
}

// NewLogger creates a new logger with a given name
func NewLogger(name string) LoggerInterface {
	if Logger == nil {
		// Initialize with default config if not already initialized
		_ = Initialize("development", "info")
	}
	return &ZapLogger{
		logger: Logger.Named(name),
	}
}

// Debug logs a debug message
func (z *ZapLogger) Debug(msg string, fields ...zap.Field) {
	z.logger.Debug(msg, fields...)
}

// Info logs an info message
func (z *ZapLogger) Info(msg string, fields ...zap.Field) {
	z.logger.Info(msg, fields...)
}

// Warn logs a warning message
func (z *ZapLogger) Warn(msg string, fields ...zap.Field) {
	z.logger.Warn(msg, fields...)
}

// Error logs an error message
func (z *ZapLogger) Error(msg string, fields ...zap.Field) {
	z.logger.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func (z *ZapLogger) Fatal(msg string, fields ...zap.Field) {
	z.logger.Fatal(msg, fields...)
}

// With adds fields to the logger
func (z *ZapLogger) With(fields ...zap.Field) LoggerInterface {
	return &ZapLogger{
		logger: z.logger.With(fields...),
	}
}

// Sync flushes the logger
func (z *ZapLogger) Sync() error {
	return z.logger.Sync()
}

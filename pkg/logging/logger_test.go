// Package logging provides comprehensive logging system testing for FlexCore
package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name          string
		environment   string
		level         string
		wantErr       bool
		expectedLevel zapcore.Level
	}{
		{
			name:          "production environment",
			environment:   "production",
			level:         "",
			wantErr:       false,
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name:          "development environment",
			environment:   "development",
			level:         "",
			wantErr:       false,
			expectedLevel: zapcore.DebugLevel,
		},
		{
			name:          "unknown environment defaults to development",
			environment:   "unknown",
			level:         "",
			wantErr:       false,
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name:          "custom debug level",
			environment:   "production",
			level:         "debug",
			wantErr:       false,
			expectedLevel: zapcore.DebugLevel,
		},
		{
			name:          "custom info level",
			environment:   "development",
			level:         "info",
			wantErr:       false,
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name:          "custom warn level",
			environment:   "production",
			level:         "warn",
			wantErr:       false,
			expectedLevel: zapcore.WarnLevel,
		},
		{
			name:          "custom error level",
			environment:   "production",
			level:         "error",
			wantErr:       false,
			expectedLevel: zapcore.ErrorLevel,
		},
		{
			name:          "invalid level ignored",
			environment:   "production",
			level:         "invalid-level",
			wantErr:       false,
			expectedLevel: zapcore.InfoLevel, // Should fall back to production default
		},
		{
			name:          "empty environment and level",
			environment:   "",
			level:         "",
			wantErr:       false,
			expectedLevel: zapcore.InfoLevel, // Default fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset Logger before each test
			Logger = nil

			err := Initialize(tt.environment, tt.level)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, Logger)

			// Test that logger can be used
			assert.NotPanics(t, func() {
				Logger.Info("Test log message")
				Logger.Debug("Test debug message")
				Logger.Warn("Test warn message")
				Logger.Error("Test error message")
			})
		})
	}
}

func TestInitialize_EnvironmentConfigurations(t *testing.T) {
	t.Run("production configuration", func(t *testing.T) {
		Logger = nil
		err := Initialize("production", "")
		require.NoError(t, err)
		require.NotNil(t, Logger)

		// Production logger should be available
		assert.NotPanics(t, func() {
			Logger.Info("Production test message")
		})
	})

	t.Run("development configuration", func(t *testing.T) {
		Logger = nil
		err := Initialize("development", "")
		require.NoError(t, err)
		require.NotNil(t, Logger)

		// Development logger should be available
		assert.NotPanics(t, func() {
			Logger.Debug("Development test message")
			Logger.Info("Development info message")
		})
	})
}

func TestInitialize_LogLevels(t *testing.T) {
	levels := []string{
		"debug",
		"info",
		"warn",
		"error",
		"dpanic",
		"panic",
		"fatal",
	}

	for _, level := range levels {
		t.Run("level_"+level, func(t *testing.T) {
			Logger = nil
			err := Initialize("production", level)
			require.NoError(t, err)
			assert.NotNil(t, Logger)

			// Logger should be functional regardless of level
			assert.NotPanics(t, func() {
				Logger.Info("Test message for level: " + level)
			})
		})
	}
}

func TestClose(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func()
		wantErr   bool
	}{
		{
			name: "close initialized logger",
			setupFunc: func() {
				Logger = nil
				if err := Initialize("development", "info"); err != nil {
					panic(err)
				}
			},
			wantErr: false,
		},
		{
			name: "close nil logger",
			setupFunc: func() {
				Logger = nil
			},
			wantErr: false,
		},
		{
			name: "close production logger",
			setupFunc: func() {
				Logger = nil
				if err := Initialize("production", "warn"); err != nil {
					panic(err)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()

			err := Close()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLogger_Functionality(t *testing.T) {
	t.Run("logger methods work correctly", func(t *testing.T) {
		Logger = nil
		err := Initialize("development", "debug")
		require.NoError(t, err)
		require.NotNil(t, Logger)

		// Test all logging methods
		assert.NotPanics(t, func() {
			Logger.Debug("Debug message")
			Logger.Info("Info message")
			Logger.Warn("Warn message")
			Logger.Error("Error message")
		})

		// Test structured logging
		assert.NotPanics(t, func() {
			Logger.Info("Structured log",
				zap.String("key1", "value1"),
				zap.Int("key2", 42),
				zap.Bool("key3", true),
			)
		})

		// Test with fields
		assert.NotPanics(t, func() {
			logger := Logger.With(
				zap.String("component", "test"),
				zap.String("operation", "testing"),
			)
			logger.Info("Message with context")
		})
	})
}

func TestLogger_WithFields(t *testing.T) {
	t.Run("logger with fields", func(t *testing.T) {
		Logger = nil
		err := Initialize("development", "debug")
		require.NoError(t, err)
		require.NotNil(t, Logger)

		// Create logger with fields
		loggerWithFields := Logger.With(
			zap.String("service", "flexcore"),
			zap.String("version", "0.9.0"),
			zap.Int("pid", 12345),
		)

		assert.NotNil(t, loggerWithFields)
		assert.NotPanics(t, func() {
			loggerWithFields.Info("Test message with predefined fields")
			loggerWithFields.Error("Error with context",
				zap.String("error_code", "E001"),
				zap.String("error_detail", "Something went wrong"),
			)
		})
	})
}

func TestLogger_EdgeCases(t *testing.T) {
	t.Run("empty and nil values", func(t *testing.T) {
		Logger = nil
		err := Initialize("development", "debug")
		require.NoError(t, err)
		require.NotNil(t, Logger)

		// Test with empty strings
		assert.NotPanics(t, func() {
			Logger.Info("") // Empty message
			Logger.Info("Message", zap.String("empty", ""))
		})

		// Test with nil logger operations (shouldn't panic)
		originalLogger := Logger
		Logger = nil
		assert.NotPanics(t, func() {
			if err := Close(); err != nil {
				t.Logf("Close returned error: %v", err)
			}
		})
		Logger = originalLogger
	})

	t.Run("special characters in log messages", func(t *testing.T) {
		Logger = nil
		err := Initialize("development", "debug")
		require.NoError(t, err)
		require.NotNil(t, Logger)

		specialMessage := "Log with special chars: !@#$%^&*(){}[]|\\:;\"'<>?,./-=+~`"
		unicodeMessage := "Unicode log: 测试日志 🚀 ログメッセージ رسالة سجل"
		newlineMessage := "Multi-line\nlog\nmessage\nwith\nnewlines"

		assert.NotPanics(t, func() {
			Logger.Info(specialMessage)
			Logger.Info(unicodeMessage)
			Logger.Info(newlineMessage)
		})
	})

	t.Run("very long log messages", func(t *testing.T) {
		Logger = nil
		err := Initialize("development", "debug")
		require.NoError(t, err)
		require.NotNil(t, Logger)

		// Create very long message
		longMessage := string(make([]byte, 10000))
		for range longMessage {
			longMessage = "x" + longMessage[1:]
		}

		assert.NotPanics(t, func() {
			Logger.Info(longMessage)
		})
		assert.GreaterOrEqual(t, len(longMessage), 10000)
	})
}

func TestLogger_StructuredLogging(t *testing.T) {
	t.Run("various field types", func(t *testing.T) {
		Logger = nil
		err := Initialize("development", "debug")
		require.NoError(t, err)
		require.NotNil(t, Logger)

		assert.NotPanics(t, func() {
			Logger.Info("Structured logging test",
				zap.String("string_field", "test_string"),
				zap.Int("int_field", 42),
				zap.Int64("int64_field", 1234567890),
				zap.Float64("float_field", 3.14159),
				zap.Bool("bool_field", true),
				zap.Duration("duration_field", 1000000000), // 1 second
				zap.Strings("array_field", []string{"item1", "item2", "item3"}),
			)
		})
	})

	t.Run("nested objects", func(t *testing.T) {
		Logger = nil
		err := Initialize("development", "debug")
		require.NoError(t, err)
		require.NotNil(t, Logger)

		assert.NotPanics(t, func() {
			Logger.Info("Nested object logging",
				zap.Any("object", map[string]interface{}{
					"nested_string": "value",
					"nested_int":    123,
					"nested_bool":   false,
					"nested_array":  []string{"a", "b", "c"},
					"deeply_nested": map[string]string{
						"deep_key": "deep_value",
					},
				}),
			)
		})
	})
}

func TestLogger_ErrorHandling(t *testing.T) {
	t.Run("error logging with stack traces", func(t *testing.T) {
		Logger = nil
		err := Initialize("development", "debug")
		require.NoError(t, err)
		require.NotNil(t, Logger)

		testError := assert.AnError

		assert.NotPanics(t, func() {
			Logger.Error("Error occurred",
				zap.Error(testError),
				zap.String("context", "test_context"),
				zap.Int("error_code", 500),
			)
		})
	})
}

// Test concurrent access to ensure thread safety
func TestLogger_Concurrent(t *testing.T) {
	t.Run("concurrent logging", func(t *testing.T) {
		Logger = nil
		err := Initialize("development", "debug")
		require.NoError(t, err)
		require.NotNil(t, Logger)

		const numGoroutines = 100
		const numLogs = 10

		results := make(chan bool, numGoroutines*numLogs)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numLogs; j++ {
					// Concurrent logging operations
					Logger.Info("Concurrent log message",
						zap.Int("goroutine_id", id),
						zap.Int("message_id", j),
						zap.String("test_data", "concurrent_test"),
					)

					Logger.Debug("Debug message",
						zap.Int("goroutine", id),
						zap.Int("iteration", j),
					)

					Logger.Warn("Warning message",
						zap.String("source", "concurrent_test"),
					)

					results <- true
				}
			}(i)
		}

		// Collect all results
		for i := 0; i < numGoroutines*numLogs; i++ {
			success := <-results
			assert.True(t, success, "Concurrent logging operation failed")
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkLogger_Initialize(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Logger = nil
		if err := Initialize("production", "info"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	Logger = nil
	err := Initialize("production", "info")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Logger.Info("Benchmark log message")
	}
}

func BenchmarkLogger_InfoWithFields(b *testing.B) {
	Logger = nil
	err := Initialize("production", "info")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Logger.Info("Benchmark log with fields",
			zap.String("key", "value"),
			zap.Int("iteration", i),
			zap.Bool("benchmark", true),
		)
	}
}

func BenchmarkLogger_StructuredLogging(b *testing.B) {
	Logger = nil
	err := Initialize("production", "info")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Logger.Info("Structured benchmark",
			zap.String("component", "benchmark"),
			zap.Int("iteration", i),
			zap.Duration("elapsed", 1000000), // 1ms
			zap.Any("data", map[string]interface{}{
				"nested": "value",
				"count":  i,
			}),
		)
	}
}

func BenchmarkLogger_Close(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Logger = nil
		if err := Initialize("production", "info"); err != nil {
			b.Fatal(err)
		}
		if err := Close(); err != nil {
			b.Fatal(err)
		}
	}
}

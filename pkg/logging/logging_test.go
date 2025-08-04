package logging

import (
	"os"
	"testing"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name        string
		service     string
		level       string
		expectError bool
	}{
		{
			name:        "Valid initialization",
			service:     "test-service",
			level:       "info",
			expectError: false,
		},
		{
			name:        "Invalid log level falls back to info",
			service:     "test-service",
			level:       "invalid",
			expectError: false,
		},
		{
			name:        "Empty service name",
			service:     "",
			level:       "debug",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Initialize(tt.service, tt.level)
			if (err != nil) != tt.expectError {
				t.Errorf("Initialize() error = %v, expectError %v", err, tt.expectError)
			}
			
			// Verify logger is initialized
			logger := GetLogger()
			if logger == nil {
				t.Error("GetLogger() returned nil after Initialize()")
			}
		})
	}
}

func TestGetLogger(t *testing.T) {
	// Reset global logger
	globalLogger = nil
	
	logger := GetLogger()
	if logger == nil {
		t.Error("GetLogger() should never return nil")
	}
	
	// Test that it returns the same instance
	logger2 := GetLogger()
	if logger != logger2 {
		t.Error("GetLogger() should return the same instance")
	}
}

func TestFieldCreation(t *testing.T) {
	field := F("test_key", "test_value")
	
	if field.Key != "test_key" {
		t.Errorf("Expected key 'test_key', got '%s'", field.Key)
	}
	
	if field.Value != "test_value" {
		t.Errorf("Expected value 'test_value', got '%v'", field.Value)
	}
}

func TestZerologLogger(t *testing.T) {
	// Initialize with a test service
	err := Initialize("test-logger", "debug")
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	
	logger := GetLogger()
	
	// Test that methods don't panic
	t.Run("Debug method", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Debug() panicked: %v", r)
			}
		}()
		logger.Debug("test debug message")
	})
	
	t.Run("Info method", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Info() panicked: %v", r)
			}
		}()
		logger.Info("test info message")
		logger.Info("test info with field", F("key", "value"))
	})
	
	t.Run("Warn method", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Warn() panicked: %v", r)
			}
		}()
		logger.Warn("test warn message")
		logger.Warn("test warn with field", F("key", "value"))
	})
	
	t.Run("Error method", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Error() panicked: %v", r)
			}
		}()
		logger.Error("test error message")
		logger.Error("test error with field", F("key", "value"))
	})
}

func TestLoggingWithEnvironment(t *testing.T) {
	// Test development environment
	os.Setenv("FLEXT_SERVER_ENVIRONMENT", "development")
	defer os.Unsetenv("FLEXT_SERVER_ENVIRONMENT")
	
	err := Initialize("test-dev", "info")
	if err != nil {
		t.Fatalf("Failed to initialize logger for development: %v", err)
	}
	
	logger := GetLogger()
	if logger == nil {
		t.Error("Logger should not be nil in development mode")
	}
}

func TestMultipleFields(t *testing.T) {
	err := Initialize("test-multifield", "info")
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	
	logger := GetLogger()
	
	// Test with multiple fields
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Multiple fields logging panicked: %v", r)
		}
	}()
	
	logger.Info("test message with multiple fields",
		F("string_field", "value"),
		F("int_field", 42),
		F("bool_field", true),
		F("float_field", 3.14),
	)
}

func TestLoggerInterface(t *testing.T) {
	// Verify that zerologLogger implements Logger interface
	var logger Logger = &zerologLogger{}
	
	// This should compile without error, proving interface compliance
	_ = logger
}

// Benchmark tests
func BenchmarkLoggerInfo(b *testing.B) {
	Initialize("benchmark", "info")
	logger := GetLogger()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLoggerInfoWithFields(b *testing.B) {
	Initialize("benchmark", "info")
	logger := GetLogger()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", 
			F("iteration", i),
			F("service", "benchmark"),
		)
	}
}
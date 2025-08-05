// Package config provides comprehensive configuration management testing for FlexCore
package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name    string
		setup   func()
		cleanup func()
		wantErr bool
	}{
		{
			name: "default initialization",
			setup: func() {
				// Clean slate
				V = nil
				Current = nil
			},
			cleanup: func() {},
			wantErr: false,
		},
		{
			name: "with environment variables",
			setup: func() {
				V = nil
				Current = nil
				os.Setenv("FLEXCORE_APP_NAME", "test-app")
				os.Setenv("FLEXCORE_APP_PORT", "9000")
				os.Setenv("FLEXCORE_DATABASE_HOST", "testdb")
			},
			cleanup: func() {
				os.Unsetenv("FLEXCORE_APP_NAME")
				os.Unsetenv("FLEXCORE_APP_PORT")
				os.Unsetenv("FLEXCORE_DATABASE_HOST")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			err := Initialize()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, V)
			assert.NotNil(t, Current)
			assert.NotEmpty(t, Current.App.Name)
			assert.Greater(t, Current.App.Port, 0)
		})
	}
}

func TestSetDefaults(t *testing.T) {
	t.Run("default values are set correctly", func(t *testing.T) {
		V = viper.New()
		setDefaults()

		// Test app defaults
		assert.Equal(t, "flexcore", V.GetString("app.name"))
		assert.Equal(t, "1.0.0", V.GetString("app.version"))
		assert.Equal(t, "development", V.GetString("app.environment"))
		assert.True(t, V.GetBool("app.debug"))
		assert.Equal(t, 8080, V.GetInt("app.port"))

		// Test database defaults
		assert.Equal(t, "memory", V.GetString("database.type"))
		assert.Equal(t, "localhost", V.GetString("database.host"))
		assert.Equal(t, 5432, V.GetInt("database.port"))
		assert.Equal(t, "flexcore", V.GetString("database.name"))
		assert.Equal(t, "postgres", V.GetString("database.user"))
		assert.Equal(t, "", V.GetString("database.password"))
		assert.Equal(t, "disable", V.GetString("database.ssl_mode"))
		assert.Equal(t, 25, V.GetInt("database.max_open_conns"))
		assert.Equal(t, 5, V.GetInt("database.max_idle_conns"))
		assert.Equal(t, 5*time.Minute, V.GetDuration("database.max_lifetime"))
		assert.True(t, V.GetBool("database.auto_migrate"))

		// Test server defaults
		assert.Equal(t, "0.0.0.0", V.GetString("server.host"))
		assert.Equal(t, 8080, V.GetInt("server.port"))
		assert.Equal(t, 15*time.Second, V.GetDuration("server.read_timeout"))
		assert.Equal(t, 15*time.Second, V.GetDuration("server.write_timeout"))
		assert.Equal(t, 60*time.Second, V.GetDuration("server.idle_timeout"))

		// Test logging defaults
		assert.Equal(t, "info", V.GetString("logging.level"))
		assert.Equal(t, "json", V.GetString("logging.format"))
		assert.Equal(t, "stdout", V.GetString("logging.output"))
		assert.True(t, V.GetBool("logging.structured"))
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		wantErr  bool
		errorMsg string
	}{
		{
			name: "valid configuration",
			config: &Config{
				App: struct {
					Name        string `mapstructure:"name"`
					Version     string `mapstructure:"version"`
					Environment string `mapstructure:"environment"`
					Debug       bool   `mapstructure:"debug"`
					Port        int    `mapstructure:"port"`
				}{
					Name: "test-app",
					Port: 8080,
				},
			},
			wantErr: false,
		},
		{
			name: "empty app name",
			config: &Config{
				App: struct {
					Name        string `mapstructure:"name"`
					Version     string `mapstructure:"version"`
					Environment string `mapstructure:"environment"`
					Debug       bool   `mapstructure:"debug"`
					Port        int    `mapstructure:"port"`
				}{
					Name: "",
					Port: 8080,
				},
			},
			wantErr:  true,
			errorMsg: "app.name cannot be empty",
		},
		{
			name: "invalid port - zero",
			config: &Config{
				App: struct {
					Name        string `mapstructure:"name"`
					Version     string `mapstructure:"version"`
					Environment string `mapstructure:"environment"`
					Debug       bool   `mapstructure:"debug"`
					Port        int    `mapstructure:"port"`
				}{
					Name: "test-app",
					Port: 0,
				},
			},
			wantErr:  true,
			errorMsg: "app.port must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			config: &Config{
				App: struct {
					Name        string `mapstructure:"name"`
					Version     string `mapstructure:"version"`
					Environment string `mapstructure:"environment"`
					Debug       bool   `mapstructure:"debug"`
					Port        int    `mapstructure:"port"`
				}{
					Name: "test-app",
					Port: 70000,
				},
			},
			wantErr:  true,
			errorMsg: "app.port must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Current = tt.config
			err := validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigAccessors(t *testing.T) {
	// Setup viper instance
	V = viper.New()
	V.Set("test.string", "test-value")
	V.Set("test.int", 42)
	V.Set("test.bool", true)
	V.Set("test.complex", map[string]interface{}{"nested": "value"})

	t.Run("Get function", func(t *testing.T) {
		value := Get("test.string")
		assert.Equal(t, "test-value", value)

		complexValue := Get("test.complex")
		assert.NotNil(t, complexValue)
	})

	t.Run("GetString function", func(t *testing.T) {
		value := GetString("test.string")
		assert.Equal(t, "test-value", value)

		// Test string conversion
		intAsString := GetString("test.int")
		assert.Equal(t, "42", intAsString)
	})

	t.Run("GetInt function", func(t *testing.T) {
		value := GetInt("test.int")
		assert.Equal(t, 42, value)

		// Test default value for non-existent key
		nonExistent := GetInt("non.existent")
		assert.Equal(t, 0, nonExistent)
	})

	t.Run("GetBool function", func(t *testing.T) {
		value := GetBool("test.bool")
		assert.True(t, value)

		// Test default value for non-existent key
		nonExistent := GetBool("non.existent")
		assert.False(t, nonExistent)
	})

	t.Run("Set function", func(t *testing.T) {
		Set("test.new", "new-value")
		value := GetString("test.new")
		assert.Equal(t, "new-value", value)
	})
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	t.Run("environment variables override defaults", func(t *testing.T) {
		// Set environment variables
		os.Setenv("FLEXCORE_APP_NAME", "env-app")
		os.Setenv("FLEXCORE_APP_PORT", "9999")
		os.Setenv("FLEXCORE_DATABASE_HOST", "env-db-host")
		os.Setenv("FLEXCORE_LOGGING_LEVEL", "debug")
		defer func() {
			os.Unsetenv("FLEXCORE_APP_NAME")
			os.Unsetenv("FLEXCORE_APP_PORT")
			os.Unsetenv("FLEXCORE_DATABASE_HOST")
			os.Unsetenv("FLEXCORE_LOGGING_LEVEL")
		}()

		err := Initialize()
		require.NoError(t, err)

		assert.Equal(t, "env-app", Current.App.Name)
		assert.Equal(t, 9999, Current.App.Port)
		assert.Equal(t, "env-db-host", Current.Database.Host)
		assert.Equal(t, "debug", Current.Logging.Level)
	})
}

func TestConfigStructure(t *testing.T) {
	t.Run("config structure is properly populated", func(t *testing.T) {
		err := Initialize()
		require.NoError(t, err)
		require.NotNil(t, Current)

		// Test App section
		assert.NotEmpty(t, Current.App.Name)
		assert.NotEmpty(t, Current.App.Version)
		assert.NotEmpty(t, Current.App.Environment)
		assert.Greater(t, Current.App.Port, 0)

		// Test Database section
		assert.NotEmpty(t, Current.Database.Type)
		assert.NotEmpty(t, Current.Database.Host)
		assert.Greater(t, Current.Database.Port, 0)
		assert.NotEmpty(t, Current.Database.Name)
		assert.NotEmpty(t, Current.Database.User)
		assert.NotEmpty(t, Current.Database.SSLMode)
		assert.Greater(t, Current.Database.MaxOpenConns, 0)
		assert.Greater(t, Current.Database.MaxIdleConns, 0)
		assert.Greater(t, Current.Database.MaxLifetime, time.Duration(0))

		// Test Server section
		assert.NotEmpty(t, Current.Server.Host)
		assert.Greater(t, Current.Server.Port, 0)
		assert.Greater(t, Current.Server.ReadTimeout, time.Duration(0))
		assert.Greater(t, Current.Server.WriteTimeout, time.Duration(0))
		assert.Greater(t, Current.Server.IdleTimeout, time.Duration(0))

		// Test Logging section
		assert.NotEmpty(t, Current.Logging.Level)
		assert.NotEmpty(t, Current.Logging.Format)
		assert.NotEmpty(t, Current.Logging.Output)
	})
}

func TestWatch(t *testing.T) {
	t.Run("watch function can be called", func(t *testing.T) {
		err := Initialize()
		require.NoError(t, err)

		// This should not panic
		assert.NotPanics(t, func() {
			Watch()
		})
	})
}

func TestConfigEdgeCases(t *testing.T) {
	t.Run("handling of special values", func(t *testing.T) {
		V = viper.New()
		setDefaults()

		// Test zero values
		V.Set("test.zero_int", 0)
		V.Set("test.empty_string", "")
		V.Set("test.false_bool", false)

		assert.Equal(t, 0, V.GetInt("test.zero_int"))
		assert.Equal(t, "", V.GetString("test.empty_string"))
		assert.False(t, V.GetBool("test.false_bool"))
	})

	t.Run("handling of nested configuration", func(t *testing.T) {
		V = viper.New()
		V.Set("deeply.nested.config.value", "deep-value")

		value := V.GetString("deeply.nested.config.value")
		assert.Equal(t, "deep-value", value)
	})

	t.Run("handling of complex data types", func(t *testing.T) {
		V = viper.New()
		complexValue := map[string]interface{}{
			"array":  []string{"item1", "item2", "item3"},
			"number": 42,
			"nested": map[string]string{"key": "value"},
		}
		V.Set("complex", complexValue)

		retrieved := V.Get("complex")
		assert.Equal(t, complexValue, retrieved)
	})
}

func TestConfigValidation_EdgeCases(t *testing.T) {
	t.Run("boundary port values", func(t *testing.T) {
		tests := []struct {
			name    string
			port    int
			wantErr bool
		}{
			{"port 1 (minimum valid)", 1, false},
			{"port 65535 (maximum valid)", 65535, false},
			{"port 0 (invalid)", 0, true},
			{"port -1 (invalid)", -1, true},
			{"port 65536 (invalid)", 65536, true},
			{"port 80 (common)", 80, false},
			{"port 443 (common)", 443, false},
			{"port 8080 (common)", 8080, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				Current = &Config{
					App: struct {
						Name        string `mapstructure:"name"`
						Version     string `mapstructure:"version"`
						Environment string `mapstructure:"environment"`
						Debug       bool   `mapstructure:"debug"`
						Port        int    `mapstructure:"port"`
					}{
						Name: "test-app",
						Port: tt.port,
					},
				}

				err := validate()
				if tt.wantErr {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "app.port must be between 1 and 65535")
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

// Test concurrent access to ensure thread safety
func TestConfig_Concurrent(t *testing.T) {
	t.Run("concurrent config access", func(t *testing.T) {
		err := Initialize()
		require.NoError(t, err)

		const numGoroutines = 100
		const numOperations = 10

		results := make(chan bool, numGoroutines*numOperations)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					// Concurrent read operations
					name := GetString("app.name")
					port := GetInt("app.port")
					debug := GetBool("app.debug")

					// Concurrent write operations
					Set(fmt.Sprintf("test.goroutine_%d_%d", id, j), fmt.Sprintf("value_%d_%d", id, j))

					// Verify read operations succeeded
					success := name != "" && port > 0
					results <- success

					// Use debug to avoid unused variable
					_ = debug
				}
			}(i)
		}

		// Collect all results
		for i := 0; i < numGoroutines*numOperations; i++ {
			success := <-results
			assert.True(t, success, "Concurrent operation failed")
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkConfig_Initialize(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		V = nil
		Current = nil
		_ = Initialize()
	}
}

func BenchmarkConfig_GetString(b *testing.B) {
	err := Initialize()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetString("app.name")
	}
}

func BenchmarkConfig_GetInt(b *testing.B) {
	err := Initialize()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetInt("app.port")
	}
}

func BenchmarkConfig_Set(b *testing.B) {
	err := Initialize()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Set(fmt.Sprintf("benchmark.key_%d", i), fmt.Sprintf("value_%d", i))
	}
}

func BenchmarkConfig_Validate(b *testing.B) {
	err := Initialize()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validate()
	}
}

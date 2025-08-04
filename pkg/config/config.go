// Package config - Pure Viper implementation for FlexCore
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/flext-sh/flexcore/pkg/logging"
	"github.com/spf13/viper"
)

const (
	// Default configuration constants
	defaultAppPort            = 8080
	defaultDatabasePort       = 5432
	defaultMaxOpenConns       = 25
	defaultMaxIdleConns       = 5
	defaultMaxLifetimeMinutes = 5

	// Server timeout constants
	defaultServerPort          = 8080
	defaultReadTimeoutSeconds  = 15
	defaultWriteTimeoutSeconds = 15
	defaultIdleTimeoutSeconds  = 60
)

// Global Viper instance
var V *viper.Viper

// Global configuration instance
var Current *Config

// Config structure for type-safe access
type Config struct {
	App struct {
		Name        string `mapstructure:"name"`
		Version     string `mapstructure:"version"`
		Environment string `mapstructure:"environment"`
		Debug       bool   `mapstructure:"debug"`
		Port        int    `mapstructure:"port"`
	} `mapstructure:"app"`

	Database struct {
		Type         string        `mapstructure:"type"`
		Host         string        `mapstructure:"host"`
		Port         int           `mapstructure:"port"`
		Name         string        `mapstructure:"name"`
		User         string        `mapstructure:"user"`
		Password     string        `mapstructure:"password"`
		SSLMode      string        `mapstructure:"ssl_mode"`
		MaxOpenConns int           `mapstructure:"max_open_conns"`
		MaxIdleConns int           `mapstructure:"max_idle_conns"`
		MaxLifetime  time.Duration `mapstructure:"max_lifetime"`
		AutoMigrate  bool          `mapstructure:"auto_migrate"`
	} `mapstructure:"database"`

	Server struct {
		Host         string        `mapstructure:"host"`
		Port         int           `mapstructure:"port"`
		ReadTimeout  time.Duration `mapstructure:"read_timeout"`
		WriteTimeout time.Duration `mapstructure:"write_timeout"`
		IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	} `mapstructure:"server"`

	Logging struct {
		Level      string `mapstructure:"level"`
		Format     string `mapstructure:"format"`
		Output     string `mapstructure:"output"`
		Structured bool   `mapstructure:"structured"`
	} `mapstructure:"logging"`
}

// Initialize sets up Viper with defaults and loads config
func Initialize() error {
	V = viper.New()

	// Set configuration sources
	V.SetConfigName("config")
	V.SetConfigType("yaml")
	V.AddConfigPath(".")
	V.AddConfigPath("./configs")
	V.AddConfigPath("./configs/development")
	V.AddConfigPath("./configs/production")
	V.AddConfigPath("/etc/flexcore/")

	// Environment variables (12-factor app)
	V.SetEnvPrefix("FLEXCORE")
	V.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	V.AutomaticEnv()

	// Set enterprise defaults
	setDefaults()

	// Read config file and unmarshal
	if err := V.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	Current = &Config{}
	if err := V.Unmarshal(Current); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	return validate()
}

// setDefaults sets default configuration values
func setDefaults() {
	// App defaults
	V.SetDefault("app.name", "flexcore")
	V.SetDefault("app.version", "1.0.0")
	V.SetDefault("app.environment", "development")
	V.SetDefault("app.debug", true)
	V.SetDefault("app.port", defaultAppPort)

	// Database defaults
	V.SetDefault("database.type", "memory")
	V.SetDefault("database.host", "localhost")
	V.SetDefault("database.port", defaultDatabasePort)
	V.SetDefault("database.name", "flexcore")
	V.SetDefault("database.user", "postgres")
	V.SetDefault("database.password", "")
	V.SetDefault("database.ssl_mode", "disable")
	V.SetDefault("database.max_open_conns", defaultMaxOpenConns)
	V.SetDefault("database.max_idle_conns", defaultMaxIdleConns)
	V.SetDefault("database.max_lifetime", defaultMaxLifetimeMinutes*time.Minute)
	V.SetDefault("database.auto_migrate", true)

	// Server defaults
	V.SetDefault("server.host", "0.0.0.0")
	V.SetDefault("server.port", defaultServerPort)
	V.SetDefault("server.read_timeout", defaultReadTimeoutSeconds*time.Second)
	V.SetDefault("server.write_timeout", defaultWriteTimeoutSeconds*time.Second)
	V.SetDefault("server.idle_timeout", defaultIdleTimeoutSeconds*time.Second)

	// Logging defaults
	V.SetDefault("logging.level", "info")
	V.SetDefault("logging.format", "json")
	V.SetDefault("logging.output", "stdout")
	V.SetDefault("logging.structured", true)
}

// validate validates the configuration
func validate() error {
	if Current.App.Name == "" {
		return fmt.Errorf("app.name cannot be empty")
	}

	if Current.App.Port <= 0 || Current.App.Port > 65535 {
		return fmt.Errorf("app.port must be between 1 and 65535")
	}

	return nil
}

// Watch enables configuration hot reloading
func Watch() {
	V.WatchConfig()
	// In a real implementation, you'd have a callback to update Current
}

// Get returns a configuration value by key
func Get(key string) interface{} {
	return V.Get(key)
}

// GetString returns a string configuration value
func GetString(key string) string {
	return V.GetString(key)
}

// GetInt returns an integer configuration value
func GetInt(key string) int {
	return V.GetInt(key)
}

// GetBool returns a boolean configuration value
func GetBool(key string) bool {
	return V.GetBool(key)
}

// Set sets a configuration value
func Set(key string, value interface{}) {
	V.Set(key, value)
}

// CommandLineFlags represents command line flags for FlexCore applications
type CommandLineFlags struct {
	Environment string
	LogLevel    string
	Help        bool
	Version     bool
}

// InitializeApplicationWithFlags initializes application with command line flags
// This eliminates 32 lines of code duplication between flexcore and server main.go
func InitializeApplicationWithFlags(flags CommandLineFlags) error {
	// Initialize configuration
	if err := Initialize(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// Override environment if provided
	if flags.Environment != "" {
		V.Set("app.environment", flags.Environment)
		Current.App.Environment = flags.Environment
	}

	// Determine log level
	logLevel := flags.LogLevel
	if logLevel == "" {
		if Current.App.Debug {
			logLevel = "debug"
		} else {
			logLevel = "info"
		}
	}

	// Initialize logging - we need to import the logging package
	return initializeLogging(Current.App.Environment, logLevel)
}

// initializeLogging initializes logging with given environment and level
func initializeLogging(environment, logLevel string) error {
	// Initialize logging
	if err := logging.Initialize(environment, logLevel); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	// Enable config hot reloading
	Watch()

	return nil
}

// Package config provides advanced configuration management using Viper
package config

import (
	"strings"
	"sync"
	"time"

	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/patterns"
	"github.com/flext/flexcore/shared/result"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Manager provides configuration management with hot-reloading
type Manager struct {
	viper         *viper.Viper
	mu            sync.RWMutex
	watchers      []ConfigWatcher
	validators    []ConfigValidator
	configFile    string
	configType    string
	envPrefix     string
	remoteConfig  *RemoteConfig
}

// ConfigWatcher is called when configuration changes
type ConfigWatcher func(oldConfig, newConfig interface{})

// ConfigValidator validates configuration
type ConfigValidator func(config interface{}) error

// RemoteConfig represents remote configuration settings
type RemoteConfig struct {
	Provider string // consul, etcd, firestore
	Endpoint string
	Path     string
	SecretKeyring string
}

// NewManager creates a new configuration manager
func NewManager(options ...patterns.Option[Manager]) result.Result[*Manager] {
	manager := &Manager{
		viper:      viper.New(),
		watchers:   make([]ConfigWatcher, 0),
		validators: make([]ConfigValidator, 0),
		configType: "yaml",
		envPrefix:  "FLEXCORE",
	}

	if err := patterns.Apply(manager, options...); err != nil {
		return result.Failure[*Manager](errors.Wrap(err, "failed to apply options"))
	}

	// Configure viper
	manager.viper.SetConfigType(manager.configType)
	manager.viper.SetEnvPrefix(manager.envPrefix)
	manager.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	manager.viper.AutomaticEnv()

	// Set config file if provided
	if manager.configFile != "" {
		manager.viper.SetConfigFile(manager.configFile)
	}

	// Setup remote config if provided
	if manager.remoteConfig != nil {
		if err := manager.setupRemoteConfig(); err != nil {
			return result.Failure[*Manager](err)
		}
	}

	return result.Success(manager)
}

// Config file option
func WithConfigFile(path string) patterns.Option[Manager] {
	return func(m *Manager) error {
		m.configFile = path
		return nil
	}
}

// Config type option
func WithConfigType(configType string) patterns.Option[Manager] {
	return func(m *Manager) error {
		m.configType = configType
		return nil
	}
}

// Environment prefix option
func WithEnvPrefix(prefix string) patterns.Option[Manager] {
	return func(m *Manager) error {
		m.envPrefix = prefix
		return nil
	}
}

// Remote config option
func WithRemoteConfig(config *RemoteConfig) patterns.Option[Manager] {
	return func(m *Manager) error {
		m.remoteConfig = config
		return nil
	}
}

// Load loads configuration from all sources
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load from file if exists
	if m.configFile != "" {
		if err := m.viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return errors.Wrap(err, "failed to read config file")
			}
		}
	}

	// Load from remote if configured
	if m.remoteConfig != nil {
		if err := m.viper.ReadRemoteConfig(); err != nil {
			return errors.Wrap(err, "failed to read remote config")
		}
	}

	// Validate configuration
	if err := m.validate(); err != nil {
		return err
	}

	return nil
}

// WatchConfig enables hot-reloading of configuration
func (m *Manager) WatchConfig() error {
	if m.configFile == "" {
		return errors.ValidationError("no config file to watch")
	}

	m.viper.WatchConfig()
	m.viper.OnConfigChange(func(e fsnotify.Event) {
		m.handleConfigChange()
	})

	return nil
}

// WatchRemoteConfig watches remote configuration for changes
func (m *Manager) WatchRemoteConfig(interval time.Duration) error {
	if m.remoteConfig == nil {
		return errors.ValidationError("no remote config to watch")
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := m.viper.WatchRemoteConfig(); err != nil {
				// Log error but continue watching
			}
		}
	}()

	return nil
}

// Get retrieves a value by key
func (m *Manager) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.Get(key)
}

// GetString retrieves a string value
func (m *Manager) GetString(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetString(key)
}

// GetInt retrieves an int value
func (m *Manager) GetInt(key string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetInt(key)
}

// GetBool retrieves a bool value
func (m *Manager) GetBool(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetBool(key)
}

// GetDuration retrieves a duration value
func (m *Manager) GetDuration(key string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetDuration(key)
}

// GetStringSlice retrieves a string slice
func (m *Manager) GetStringSlice(key string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetStringSlice(key)
}

// GetStringMap retrieves a string map
func (m *Manager) GetStringMap(key string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetStringMap(key)
}

// Unmarshal unmarshals configuration into a struct
func (m *Manager) Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.Unmarshal(rawVal, opts...)
}

// UnmarshalKey unmarshals a specific key into a struct
func (m *Manager) UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.UnmarshalKey(key, rawVal, opts...)
}

// Set sets a value for a key
func (m *Manager) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.viper.Set(key, value)
}

// SetDefault sets a default value for a key
func (m *Manager) SetDefault(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.viper.SetDefault(key, value)
}

// IsSet checks if a key is set
func (m *Manager) IsSet(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.IsSet(key)
}

// AllKeys returns all configuration keys
func (m *Manager) AllKeys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.AllKeys()
}

// AllSettings returns all settings as a map
func (m *Manager) AllSettings() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.AllSettings()
}

// AddWatcher adds a configuration change watcher
func (m *Manager) AddWatcher(watcher ConfigWatcher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watchers = append(m.watchers, watcher)
}

// AddValidator adds a configuration validator
func (m *Manager) AddValidator(validator ConfigValidator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validators = append(m.validators, validator)
}

// validate runs all validators
func (m *Manager) validate() error {
	config := m.viper.AllSettings()
	
	for _, validator := range m.validators {
		if err := validator(config); err != nil {
			return errors.Wrap(err, "config validation failed")
		}
	}
	
	return nil
}

// handleConfigChange handles configuration changes
func (m *Manager) handleConfigChange() {
	m.mu.RLock()
	oldConfig := m.viper.AllSettings()
	m.mu.RUnlock()

	// Re-validate configuration
	if err := m.validate(); err != nil {
		// Log error but don't crash
		return
	}

	m.mu.RLock()
	newConfig := m.viper.AllSettings()
	watchers := make([]ConfigWatcher, len(m.watchers))
	copy(watchers, m.watchers)
	m.mu.RUnlock()

	// Notify watchers
	for _, watcher := range watchers {
		watcher(oldConfig, newConfig)
	}
}

// setupRemoteConfig configures remote configuration provider
func (m *Manager) setupRemoteConfig() error {
	if m.remoteConfig == nil {
		return nil
	}

	var err error
	switch m.remoteConfig.Provider {
	case "consul":
		err = m.viper.AddRemoteProvider("consul", m.remoteConfig.Endpoint, m.remoteConfig.Path)
	case "etcd":
		err = m.viper.AddRemoteProvider("etcd", m.remoteConfig.Endpoint, m.remoteConfig.Path)
	case "firestore":
		err = m.viper.AddRemoteProvider("firestore", m.remoteConfig.Endpoint, m.remoteConfig.Path)
	default:
		return errors.ValidationError("unsupported remote provider: " + m.remoteConfig.Provider)
	}

	if err != nil {
		return errors.Wrap(err, "failed to add remote provider")
	}

	if m.remoteConfig.SecretKeyring != "" {
		m.viper.SetConfigType("json")
		if err := m.viper.AddSecureRemoteProvider(
			m.remoteConfig.Provider,
			m.remoteConfig.Endpoint,
			m.remoteConfig.Path,
			m.remoteConfig.SecretKeyring,
		); err != nil {
			return errors.Wrap(err, "failed to add secure remote provider")
		}
	}

	return nil
}

// ConfigBuilder provides a fluent interface for building configurations
type ConfigBuilder struct {
	manager *Manager
	config  map[string]interface{}
}

// NewConfigBuilder creates a new configuration builder
func NewConfigBuilder(manager *Manager) *ConfigBuilder {
	return &ConfigBuilder{
		manager: manager,
		config:  make(map[string]interface{}),
	}
}

// With adds a key-value pair
func (b *ConfigBuilder) With(key string, value interface{}) *ConfigBuilder {
	b.config[key] = value
	return b
}

// WithDefaults adds multiple default values
func (b *ConfigBuilder) WithDefaults(defaults map[string]interface{}) *ConfigBuilder {
	for k, v := range defaults {
		b.manager.SetDefault(k, v)
	}
	return b
}

// FromFile loads configuration from a file
func (b *ConfigBuilder) FromFile(path string) *ConfigBuilder {
	b.manager.configFile = path
	b.manager.viper.SetConfigFile(path)
	return b
}

// FromEnv enables environment variable loading
func (b *ConfigBuilder) FromEnv(prefix string) *ConfigBuilder {
	b.manager.envPrefix = prefix
	b.manager.viper.SetEnvPrefix(prefix)
	b.manager.viper.AutomaticEnv()
	return b
}

// Build applies the configuration
func (b *ConfigBuilder) Build() error {
	for k, v := range b.config {
		b.manager.Set(k, v)
	}
	return b.manager.Load()
}

// Typed configuration wrapper
type TypedConfig[T any] struct {
	manager *Manager
	key     string
}

// NewTypedConfig creates a typed configuration wrapper
func NewTypedConfig[T any](manager *Manager, key string) *TypedConfig[T] {
	return &TypedConfig[T]{
		manager: manager,
		key:     key,
	}
}

// Get retrieves the typed configuration
func (tc *TypedConfig[T]) Get() result.Result[T] {
	var config T
	if err := tc.manager.UnmarshalKey(tc.key, &config); err != nil {
		return result.Failure[T](errors.Wrap(err, "failed to unmarshal config"))
	}
	return result.Success(config)
}

// Watch watches for configuration changes
func (tc *TypedConfig[T]) Watch(handler func(old, new T)) {
	tc.manager.AddWatcher(func(oldConfig, newConfig interface{}) {
		var oldTyped, newTyped T
		
		// Extract old config
		if oldMap, ok := oldConfig.(map[string]interface{}); ok {
			if oldValue, exists := oldMap[tc.key]; exists {
				// Simple type assertion - in production use proper unmarshaling
				if typed, ok := oldValue.(T); ok {
					oldTyped = typed
				}
			}
		}
		
		// Extract new config
		if newMap, ok := newConfig.(map[string]interface{}); ok {
			if newValue, exists := newMap[tc.key]; exists {
				// Simple type assertion - in production use proper unmarshaling
				if typed, ok := newValue.(T); ok {
					newTyped = typed
				}
			}
		}
		
		handler(oldTyped, newTyped)
	})
}

// Common configuration structures

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Database        string        `mapstructure:"database"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	TLS             *TLSConfig    `mapstructure:"tls"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// GetDatabaseConfig retrieves database configuration
func (m *Manager) GetDatabaseConfig() result.Result[DatabaseConfig] {
	var config DatabaseConfig
	if err := m.UnmarshalKey("database", &config); err != nil {
		return result.Failure[DatabaseConfig](errors.Wrap(err, "failed to get database config"))
	}
	return result.Success(config)
}

// GetServerConfig retrieves server configuration
func (m *Manager) GetServerConfig() result.Result[ServerConfig] {
	var config ServerConfig
	if err := m.UnmarshalKey("server", &config); err != nil {
		return result.Failure[ServerConfig](errors.Wrap(err, "failed to get server config"))
	}
	return result.Success(config)
}

// GetLogConfig retrieves logging configuration
func (m *Manager) GetLogConfig() result.Result[LogConfig] {
	var config LogConfig
	if err := m.UnmarshalKey("logging", &config); err != nil {
		return result.Failure[LogConfig](errors.Wrap(err, "failed to get log config"))
	}
	return result.Success(config)
}
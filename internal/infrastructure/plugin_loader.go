package infrastructure

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/flext/flexcore/pkg/logging"
	"go.uber.org/zap"
)

// Plugin represents a HashiCorp-style plugin exactly as specified in FLEXT_SERVICE_ARCHITECTURE.md
type Plugin interface {
	Name() string
	Version() string
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
	Shutdown() error
}

// FlextServicePlugin implements the FLEXT service plugin exactly as specified in the architecture document
type FlextServicePlugin struct {
	servicePath string
	configPath  string
	logger      logging.LoggerInterface
	mu          sync.RWMutex
	running     bool
}

// NewFlextServicePlugin creates a new FLEXT service plugin
func NewFlextServicePlugin(servicePath, configPath string, logger logging.LoggerInterface) *FlextServicePlugin {
	return &FlextServicePlugin{
		servicePath: servicePath,
		configPath:  configPath,
		logger:      logger,
	}
}

// Name returns the plugin name
func (fsp *FlextServicePlugin) Name() string {
	return "flext-service"
}

// Version returns the plugin version
func (fsp *FlextServicePlugin) Version() string {
	return "2.0.0"
}

// Execute executes the FLEXT service as a subprocess exactly as specified in the architecture document
func (fsp *FlextServicePlugin) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	fsp.mu.Lock()
	defer fsp.mu.Unlock()
	
	if fsp.running {
		return nil, fmt.Errorf("FLEXT service plugin is already running")
	}
	
	fsp.logger.Info("Executing FLEXT service plugin",
		zap.String("service_path", fsp.servicePath),
		zap.String("config_path", fsp.configPath),
		zap.Any("params", params))
	
	// Prepare FLEXT service command - Execute as subprocess exactly as specified
	cmd := exec.CommandContext(ctx, "go", "run", fsp.servicePath)
	
	// Set environment variables for FLEXT service
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("FLEXT_CONFIG_PATH=%s", fsp.configPath),
		"FLEXT_ENV=production",
		"FLEXT_DEBUG=false",
		"PYTHONPATH=/home/marlonsc/flext/flext-core/src:/home/marlonsc/flext/flext-meltano/src",
	)
	
	// Add parameters as environment variables
	for key, value := range params {
		cmd.Env = append(cmd.Env, fmt.Sprintf("FLEXT_%s=%v", key, value))
	}
	
	fsp.running = true
	
	// Execute FLEXT service subprocess
	output, err := cmd.CombinedOutput()
	fsp.running = false
	
	if err != nil {
		fsp.logger.Error("FLEXT service plugin execution failed",
			zap.Error(err),
			zap.String("output", string(output)))
		return nil, fmt.Errorf("FLEXT service execution failed: %w", err)
	}
	
	fsp.logger.Info("FLEXT service plugin executed successfully",
		zap.String("output", string(output)))
	
	// Return execution result
	return map[string]interface{}{
		"status":        "completed",
		"output":        string(output),
		"execution_time": time.Now().UTC(),
		"plugin_name":   fsp.Name(),
		"plugin_version": fsp.Version(),
	}, nil
}

// Shutdown shuts down the plugin
func (fsp *FlextServicePlugin) Shutdown() error {
	fsp.mu.Lock()
	defer fsp.mu.Unlock()
	
	fsp.running = false
	fsp.logger.Info("FLEXT service plugin shutdown")
	return nil
}

// HashicorpStyleLoader implements a HashiCorp-style plugin loader exactly as specified in FLEXT_SERVICE_ARCHITECTURE.md
type HashicorpStyleLoader struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
	logger  logging.LoggerInterface
}

// NewHashicorpStyleLoader creates a new HashiCorp-style plugin loader
func NewHashicorpStyleLoader() *HashicorpStyleLoader {
	return &HashicorpStyleLoader{
		plugins: make(map[string]Plugin),
		logger:  logging.NewLogger("hashicorp-plugin-loader"),
	}
}

// RegisterPlugin registers a plugin exactly as specified in the architecture document
func (hsl *HashicorpStyleLoader) RegisterPlugin(name string, plugin Plugin) {
	hsl.mu.Lock()
	defer hsl.mu.Unlock()
	
	hsl.plugins[name] = plugin
	hsl.logger.Info("Plugin registered",
		zap.String("plugin_name", name),
		zap.String("plugin_version", plugin.Version()))
}

// LoadPlugin loads and returns a plugin exactly as specified in the architecture document
func (hsl *HashicorpStyleLoader) LoadPlugin(name string) (interface{}, error) {
	hsl.mu.RLock()
	defer hsl.mu.RUnlock()
	
	plugin, exists := hsl.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}
	
	hsl.logger.Debug("Plugin loaded",
		zap.String("plugin_name", name),
		zap.String("plugin_version", plugin.Version()))
	
	return plugin, nil
}

// ListPlugins returns all registered plugins
func (hsl *HashicorpStyleLoader) ListPlugins() []string {
	hsl.mu.RLock()
	defer hsl.mu.RUnlock()
	
	var names []string
	for name := range hsl.plugins {
		names = append(names, name)
	}
	
	return names
}

// UnregisterPlugin removes a plugin
func (hsl *HashicorpStyleLoader) UnregisterPlugin(name string) error {
	hsl.mu.Lock()
	defer hsl.mu.Unlock()
	
	plugin, exists := hsl.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}
	
	// Shutdown the plugin before removing
	if err := plugin.Shutdown(); err != nil {
		hsl.logger.Warn("Plugin shutdown failed during unregistration",
			zap.String("plugin_name", name),
			zap.Error(err))
	}
	
	delete(hsl.plugins, name)
	hsl.logger.Info("Plugin unregistered", zap.String("plugin_name", name))
	
	return nil
}

// Shutdown shuts down all plugins
func (hsl *HashicorpStyleLoader) Shutdown() error {
	hsl.mu.Lock()
	defer hsl.mu.Unlock()
	
	hsl.logger.Info("Shutting down all plugins...")
	
	var errors []error
	for name, plugin := range hsl.plugins {
		if err := plugin.Shutdown(); err != nil {
			errors = append(errors, fmt.Errorf("plugin %s shutdown failed: %w", name, err))
		}
	}
	
	// Clear all plugins
	hsl.plugins = make(map[string]Plugin)
	
	if len(errors) > 0 {
		return fmt.Errorf("plugin shutdown errors: %v", errors)
	}
	
	hsl.logger.Info("All plugins shutdown successfully")
	return nil
}
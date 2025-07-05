// Dynamic Plugin System - HashiCorp go-plugin Integration
package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
)

// Plugin interface that all plugins must implement
type Plugin interface {
	Name() string
	Version() string
	Initialize(config map[string]interface{}) error
	Execute(ctx context.Context, input interface{}) (interface{}, error)
	Health() PluginHealth
	Shutdown() error
}

// PluginHealth represents plugin health status
type PluginHealth struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

// PluginInfo contains plugin metadata
type PluginInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author"`
	License     string                 `json:"license"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Status      string                 `json:"status"`
	LoadedAt    time.Time              `json:"loaded_at"`
	Health      PluginHealth           `json:"health"`
}

// PluginRegistry manages dynamic plugin loading and execution
type PluginRegistry struct {
	plugins     map[string]*LoadedPlugin
	pluginDir   string
	mu          sync.RWMutex
	handshake   plugin.HandshakeConfig
	pluginMap   map[string]plugin.Plugin
}

// LoadedPlugin represents a loaded plugin instance
type LoadedPlugin struct {
	Info     PluginInfo
	Client   *plugin.Client
	Plugin   Plugin
	Config   map[string]interface{}
	LoadedAt time.Time
}

// PluginRPC implements the RPC interface for plugins
type PluginRPC struct {
	client *plugin.Client
	plugin Plugin
}

// PluginRPCServer implements the RPC server for plugins
type PluginRPCServer struct {
	Impl Plugin
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry(pluginDir string) *PluginRegistry {
	handshake := plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "FLEXCORE_PLUGIN",
		MagicCookieValue: "flexcore-plugin-v1",
	}

	pluginMap := map[string]plugin.Plugin{
		"flexcore": &PluginRPC{},
	}

	return &PluginRegistry{
		plugins:   make(map[string]*LoadedPlugin),
		pluginDir: pluginDir,
		handshake: handshake,
		pluginMap: pluginMap,
	}
}

// DiscoverPlugins discovers all available plugins in the plugin directory
func (r *PluginRegistry) DiscoverPlugins() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var pluginPaths []string

	err := filepath.Walk(r.pluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for plugin binaries (exclude directories and non-executables)
		if !info.IsDir() && (info.Mode()&0111) != 0 {
			// Check if it's a valid plugin by trying to get its info
			if r.isValidPlugin(path) {
				pluginPaths = append(pluginPaths, path)
			}
		}

		return nil
	})

	return pluginPaths, err
}

// LoadPlugin loads a plugin from the given path
func (r *PluginRegistry) LoadPlugin(pluginPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create plugin client
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: r.handshake,
		Plugins:         r.pluginMap,
		Cmd:             exec.Command(pluginPath),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
			plugin.ProtocolGRPC,
		},
	})

	// Connect to the plugin
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to connect to plugin: %w", err)
	}

	// Get the plugin interface
	raw, err := rpcClient.Dispense("flexcore")
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to dispense plugin: %w", err)
	}

	pluginImpl := raw.(Plugin)

	// Initialize the plugin
	config := make(map[string]interface{})
	if err := pluginImpl.Initialize(config); err != nil {
		client.Kill()
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}

	// Create plugin info
	pluginInfo := PluginInfo{
		Name:     pluginImpl.Name(),
		Version:  pluginImpl.Version(),
		Status:   "loaded",
		LoadedAt: time.Now(),
		Health:   pluginImpl.Health(),
	}

	// Store loaded plugin
	loadedPlugin := &LoadedPlugin{
		Info:     pluginInfo,
		Client:   client,
		Plugin:   pluginImpl,
		Config:   config,
		LoadedAt: time.Now(),
	}

	r.plugins[pluginInfo.Name] = loadedPlugin

	log.Printf("Plugin loaded successfully: %s v%s", pluginInfo.Name, pluginInfo.Version)
	return nil
}

// UnloadPlugin unloads a plugin
func (r *PluginRegistry) UnloadPlugin(pluginName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	loadedPlugin, exists := r.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginName)
	}

	// Shutdown the plugin
	if err := loadedPlugin.Plugin.Shutdown(); err != nil {
		log.Printf("Warning: plugin shutdown error: %v", err)
	}

	// Kill the plugin client
	loadedPlugin.Client.Kill()

	// Remove from registry
	delete(r.plugins, pluginName)

	log.Printf("Plugin unloaded: %s", pluginName)
	return nil
}

// ExecutePlugin executes a plugin with given input
func (r *PluginRegistry) ExecutePlugin(ctx context.Context, pluginName string, input interface{}) (interface{}, error) {
	r.mu.RLock()
	loadedPlugin, exists := r.plugins[pluginName]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	return loadedPlugin.Plugin.Execute(ctx, input)
}

// ListPlugins returns information about all loaded plugins
func (r *PluginRegistry) ListPlugins() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var pluginInfos []PluginInfo
	for _, loadedPlugin := range r.plugins {
		// Update health status
		loadedPlugin.Info.Health = loadedPlugin.Plugin.Health()
		pluginInfos = append(pluginInfos, loadedPlugin.Info)
	}

	return pluginInfos
}

// GetPlugin returns information about a specific plugin
func (r *PluginRegistry) GetPlugin(pluginName string) (*PluginInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	loadedPlugin, exists := r.plugins[pluginName]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}

	// Update health status
	loadedPlugin.Info.Health = loadedPlugin.Plugin.Health()
	return &loadedPlugin.Info, nil
}

// ReloadPlugin reloads a plugin
func (r *PluginRegistry) ReloadPlugin(pluginName string) error {
	// Get the original plugin path (simplified for demo)
	pluginPath := filepath.Join(r.pluginDir, pluginName)

	// Unload the plugin
	if err := r.UnloadPlugin(pluginName); err != nil {
		return fmt.Errorf("failed to unload plugin: %w", err)
	}

	// Load the plugin again
	if err := r.LoadPlugin(pluginPath); err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}

	return nil
}

// HealthCheck performs health check on all plugins
func (r *PluginRegistry) HealthCheck() map[string]PluginHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health := make(map[string]PluginHealth)
	for name, loadedPlugin := range r.plugins {
		health[name] = loadedPlugin.Plugin.Health()
	}

	return health
}

// Shutdown shuts down all plugins
func (r *PluginRegistry) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errors []error

	for name, loadedPlugin := range r.plugins {
		if err := loadedPlugin.Plugin.Shutdown(); err != nil {
			errors = append(errors, fmt.Errorf("plugin %s shutdown error: %w", name, err))
		}
		loadedPlugin.Client.Kill()
	}

	// Clear the registry
	r.plugins = make(map[string]*LoadedPlugin)

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

// isValidPlugin checks if a binary is a valid FlexCore plugin
func (r *PluginRegistry) isValidPlugin(path string) bool {
	// Create a temporary client to test the plugin
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: r.handshake,
		Plugins:         r.pluginMap,
		Cmd:             exec.Command(path, "--version"),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
		},
	})
	defer client.Kill()

	// Try to connect (with timeout)
	done := make(chan bool, 1)
	go func() {
		_, err := client.Client()
		done <- (err == nil)
	}()

	select {
	case valid := <-done:
		return valid
	case <-time.After(5 * time.Second):
		return false
	}
}

// Plugin configuration
func (r *PluginRegistry) ConfigurePlugin(pluginName string, config map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	loadedPlugin, exists := r.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginName)
	}

	// Update configuration
	loadedPlugin.Config = config

	// Re-initialize with new config
	return loadedPlugin.Plugin.Initialize(config)
}

// CreateDemoPlugins creates demo plugins for testing
func (r *PluginRegistry) CreateDemoPlugins() error {
	// Demo plugin 1: Data Transformer
	demoPlugin1 := &DemoPlugin{
		name:    "data-transformer",
		version: "1.0.0",
		pluginType: "transformer",
	}

	// Demo plugin 2: Data Validator
	demoPlugin2 := &DemoPlugin{
		name:    "data-validator",
		version: "1.0.0",
		pluginType: "validator",
	}

	// Demo plugin 3: Event Processor
	demoPlugin3 := &DemoPlugin{
		name:    "event-processor",
		version: "1.0.0",
		pluginType: "processor",
	}

	// Register demo plugins directly (simulating loaded plugins)
	r.mu.Lock()
	defer r.mu.Unlock()

	plugins := []*DemoPlugin{demoPlugin1, demoPlugin2, demoPlugin3}

	for _, p := range plugins {
		// Initialize demo plugin
		config := map[string]interface{}{
			"demo": true,
			"mode": "development",
		}
		p.Initialize(config)

		pluginInfo := PluginInfo{
			Name:        p.Name(),
			Version:     p.Version(),
			Description: fmt.Sprintf("Demo %s plugin", p.pluginType),
			Author:      "FlexCore Team",
			License:     "MIT",
			Type:        p.pluginType,
			Config:      config,
			Status:      "loaded",
			LoadedAt:    time.Now(),
			Health:      p.Health(),
		}

		loadedPlugin := &LoadedPlugin{
			Info:     pluginInfo,
			Client:   nil, // Demo plugins don't have clients
			Plugin:   p,
			Config:   config,
			LoadedAt: time.Now(),
		}

		r.plugins[p.Name()] = loadedPlugin
	}

	return nil
}

// DemoPlugin implements Plugin interface for demonstration
type DemoPlugin struct {
	name       string
	version    string
	pluginType string
	config     map[string]interface{}
	initialized bool
}

func (p *DemoPlugin) Name() string {
	return p.name
}

func (p *DemoPlugin) Version() string {
	return p.version
}

func (p *DemoPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	p.initialized = true
	return nil
}

func (p *DemoPlugin) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	// Demo execution logic based on plugin type
	switch p.pluginType {
	case "transformer":
		return p.transform(input), nil
	case "validator":
		return p.validate(input), nil
	case "processor":
		return p.process(input), nil
	default:
		return input, nil
	}
}

func (p *DemoPlugin) Health() PluginHealth {
	status := "healthy"
	if !p.initialized {
		status = "not_initialized"
	}

	return PluginHealth{
		Status:    status,
		Timestamp: time.Now(),
		Details: map[string]string{
			"type":        p.pluginType,
			"initialized": fmt.Sprintf("%v", p.initialized),
		},
	}
}

func (p *DemoPlugin) Shutdown() error {
	p.initialized = false
	return nil
}

func (p *DemoPlugin) transform(input interface{}) interface{} {
	return map[string]interface{}{
		"original": input,
		"transformed": true,
		"transformation_type": "demo",
		"timestamp": time.Now(),
	}
}

func (p *DemoPlugin) validate(input interface{}) interface{} {
	return map[string]interface{}{
		"input": input,
		"valid": true,
		"validation_rules": []string{"demo_rule_1", "demo_rule_2"},
		"validated_at": time.Now(),
	}
}

func (p *DemoPlugin) process(input interface{}) interface{} {
	return map[string]interface{}{
		"input": input,
		"processed": true,
		"processing_steps": []string{"step1", "step2", "step3"},
		"processed_at": time.Now(),
	}
}

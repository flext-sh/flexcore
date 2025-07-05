// Package plugins provides REAL HashiCorp go-plugin system for FlexCore
package plugins

import (
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/flext/flexcore/pkg/errors"
	"github.com/flext/flexcore/pkg/result"
)

// init registers types for gob encoding/decoding
func init() {
	// Register common types for map[string]interface{} serialization
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register([]map[string]interface{}{})

	// Register common primitive types
	gob.Register(string(""))
	gob.Register(int(0))
	gob.Register(int64(0))
	gob.Register(float64(0))
	gob.Register(bool(false))

	// Register plugin-specific types
	gob.Register(PluginInfo{})
}

// FlexCorePlugin defines the interface for FlexCore plugins
type FlexCorePlugin interface {
	// Initialize the plugin with configuration
	Initialize(ctx context.Context, config map[string]interface{}) error

	// Execute the main plugin functionality
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

	// GetInfo returns plugin metadata
	GetInfo() PluginInfo

	// Health check for the plugin
	HealthCheck(ctx context.Context) error

	// Cleanup resources
	Cleanup() error
}

// PluginInfo contains plugin metadata
type PluginInfo struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	Type        string            `json:"type"` // "extractor", "loader", "transformer", "processor"
	Capabilities []string         `json:"capabilities"`
	Config      map[string]string `json:"config"`
}

// PluginMetadata is an alias for PluginInfo for backward compatibility
type PluginMetadata = PluginInfo

// ExtractorPlugin interface for data extraction plugins
type ExtractorPlugin interface {
	FlexCorePlugin
	Extract(ctx context.Context, config map[string]interface{}) ([]map[string]interface{}, error)
}

// ExtractorGRPCPlugin implements gRPC plugin interface for extractors
type ExtractorGRPCPlugin struct {
	plugin.Plugin
	Impl ExtractorPlugin
}

func (p *ExtractorGRPCPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &FlexCorePluginRPCServer{Impl: p.Impl}, nil
}

func (p *ExtractorGRPCPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &FlexCorePluginRPC{client: c}, nil
}

// RealPluginManager manages REAL HashiCorp go-plugin instances
type RealPluginManager struct {
	plugins      map[string]*RealPluginInstance
	pluginPaths  map[string]string
	mu           sync.RWMutex
	isRunning    bool

	// Plugin configuration
	handshakeConfig plugin.HandshakeConfig
	pluginMap       map[string]plugin.Plugin

	// Plugin discovery
	pluginDirectory string
	autoDiscovery   bool
}

// PluginManager is an alias for RealPluginManager
type PluginManager = RealPluginManager

// RealPluginInstance represents a running plugin instance
type RealPluginInstance struct {
	Name           string
	Path           string
	Client         *plugin.Client
	Plugin         FlexCorePlugin
	Info           PluginInfo
	Status         string // "running", "stopped", "error"
	StartedAt      time.Time
	LastHeartbeat  time.Time
	ErrorCount     int64
	ExecutionCount int64
}

// FlexCorePluginRPC implements the RPC interface for plugins
type FlexCorePluginRPC struct {
	client *rpc.Client
	Impl   FlexCorePlugin // For backward compatibility
}

// PluginRPC is an alias for FlexCorePluginRPC for backward compatibility
type PluginRPC = FlexCorePluginRPC

// FlexCorePluginRPCServer implements the RPC server for plugins
type FlexCorePluginRPCServer struct {
	Impl FlexCorePlugin
}

// NewPluginRPCClient creates a new RPC client for plugins
func NewPluginRPCClient(client *rpc.Client) *FlexCorePluginRPC {
	return &FlexCorePluginRPC{client: client}
}

// NewPluginManager creates a REAL HashiCorp plugin manager
func NewPluginManager(pluginDirectory string) *RealPluginManager {
	return NewRealPluginManager(pluginDirectory)
}

// NewRealPluginManager creates a REAL HashiCorp plugin manager
func NewRealPluginManager(pluginDirectory string) *RealPluginManager {
	// Define handshake configuration
	handshakeConfig := plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "FLEXCORE_PLUGIN",
		MagicCookieValue: "flexcore-plugin-magic-cookie",
	}

	// Define plugin map
	pluginMap := map[string]plugin.Plugin{
		"flexcore": &FlexCorePluginImpl{},
	}

	return &RealPluginManager{
		plugins:         make(map[string]*RealPluginInstance),
		pluginPaths:     make(map[string]string),
		handshakeConfig: handshakeConfig,
		pluginMap:       pluginMap,
		pluginDirectory: pluginDirectory,
		autoDiscovery:   true,
	}
}

// Start starts the plugin manager
func (rpm *RealPluginManager) Start(ctx context.Context) error {
	rpm.mu.Lock()
	defer rpm.mu.Unlock()

	if rpm.isRunning {
		return errors.ValidationError("plugin manager is already running")
	}

	rpm.isRunning = true

	// Discover plugins if auto-discovery is enabled
	if rpm.autoDiscovery {
		if err := rpm.discoverPlugins(); err != nil {
			return fmt.Errorf("failed to discover plugins: %w", err)
		}
	}

	// Start health check monitoring
	go rpm.healthCheckLoop(ctx)

	return nil
}

// LoadPlugin loads a plugin from an executable file
func (rpm *RealPluginManager) LoadPlugin(ctx context.Context, name, executablePath string) result.Result[*RealPluginInstance] {
	rpm.mu.Lock()
	defer rpm.mu.Unlock()

	// Check if plugin is already loaded
	if _, exists := rpm.plugins[name]; exists {
		return result.Failure[*RealPluginInstance](errors.ValidationError("plugin already loaded"))
	}

	// Create plugin client
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: rpm.handshakeConfig,
		Plugins:         rpm.pluginMap,
		Cmd:            exec.Command(executablePath),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
			plugin.ProtocolGRPC,
		},
	})

	// Connect to plugin
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return result.Failure[*RealPluginInstance](fmt.Errorf("failed to connect to plugin: %w", err))
	}

	// Get plugin interface
	raw, err := rpcClient.Dispense("flexcore")
	if err != nil {
		client.Kill()
		return result.Failure[*RealPluginInstance](fmt.Errorf("failed to dispense plugin: %w", err))
	}

	flexcorePlugin, ok := raw.(FlexCorePlugin)
	if !ok {
		client.Kill()
		return result.Failure[*RealPluginInstance](errors.ValidationError("invalid plugin interface"))
	}

	// Initialize plugin
	if err := flexcorePlugin.Initialize(ctx, map[string]interface{}{
		"name": name,
		"environment": "production",
	}); err != nil {
		client.Kill()
		return result.Failure[*RealPluginInstance](fmt.Errorf("failed to initialize plugin: %w", err))
	}

	// Get plugin info
	info := flexcorePlugin.GetInfo()

	// Create plugin instance
	instance := &RealPluginInstance{
		Name:           name,
		Path:           executablePath,
		Client:         client,
		Plugin:         flexcorePlugin,
		Info:           info,
		Status:         "running",
		StartedAt:      time.Now(),
		LastHeartbeat:  time.Now(),
		ErrorCount:     0,
		ExecutionCount: 0,
	}

	rpm.plugins[name] = instance
	rpm.pluginPaths[name] = executablePath

	return result.Success[*RealPluginInstance](instance)
}

// ExecutePlugin executes a plugin with given input
func (rpm *RealPluginManager) ExecutePlugin(ctx context.Context, name string, input map[string]interface{}) result.Result[map[string]interface{}] {
	rpm.mu.RLock()
	instance, exists := rpm.plugins[name]
	rpm.mu.RUnlock()

	if !exists {
		return result.Failure[map[string]interface{}](errors.ValidationError("plugin not found"))
	}

	if instance.Status != "running" {
		return result.Failure[map[string]interface{}](errors.ValidationError("plugin is not running"))
	}

	// Execute plugin
	output, err := instance.Plugin.Execute(ctx, input)
	if err != nil {
		instance.ErrorCount++
		return result.Failure[map[string]interface{}](fmt.Errorf("plugin execution failed: %w", err))
	}

	instance.ExecutionCount++
	instance.LastHeartbeat = time.Now()

	return result.Success[map[string]interface{}](output)
}

// GetPlugin retrieves a plugin instance
func (rpm *RealPluginManager) GetPlugin(name string) result.Result[*RealPluginInstance] {
	rpm.mu.RLock()
	defer rpm.mu.RUnlock()

	instance, exists := rpm.plugins[name]
	if !exists {
		return result.Failure[*RealPluginInstance](errors.ValidationError("plugin not found"))
	}

	return result.Success[*RealPluginInstance](instance)
}

// ListPlugins lists all loaded plugins
func (rpm *RealPluginManager) ListPlugins() []*RealPluginInstance {
	rpm.mu.RLock()
	defer rpm.mu.RUnlock()

	instances := make([]*RealPluginInstance, 0, len(rpm.plugins))
	for _, instance := range rpm.plugins {
		instances = append(instances, instance)
	}

	return instances
}

// UnloadPlugin unloads a plugin
func (rpm *RealPluginManager) UnloadPlugin(name string) error {
	rpm.mu.Lock()
	defer rpm.mu.Unlock()

	instance, exists := rpm.plugins[name]
	if !exists {
		return errors.ValidationError("plugin not found")
	}

	// Cleanup plugin
	if err := instance.Plugin.Cleanup(); err != nil {
		// Log error but continue with unloading
	}

	// Kill plugin process
	instance.Client.Kill()
	instance.Status = "stopped"

	// Remove from registry
	delete(rpm.plugins, name)
	delete(rpm.pluginPaths, name)

	return nil
}

// discoverPlugins discovers plugins in the plugin directory
func (rpm *RealPluginManager) discoverPlugins() error {
	if rpm.pluginDirectory == "" {
		return nil
	}

	// Check if directory exists
	if _, err := os.Stat(rpm.pluginDirectory); os.IsNotExist(err) {
		return nil // Directory doesn't exist, no plugins to discover
	}

	// Read directory
	entries, err := os.ReadDir(rpm.pluginDirectory)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}

	// Look for executable files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if isExecutable(rpm.pluginDirectory + "/" + name) {
			rpm.pluginPaths[name] = rpm.pluginDirectory + "/" + name
		}
	}

	return nil
}

// healthCheckLoop performs periodic health checks on plugins
func (rpm *RealPluginManager) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rpm.performHealthChecks(ctx)
		}
	}
}

// performHealthChecks checks health of all running plugins
func (rpm *RealPluginManager) performHealthChecks(ctx context.Context) {
	rpm.mu.RLock()
	instances := make([]*RealPluginInstance, 0, len(rpm.plugins))
	for _, instance := range rpm.plugins {
		instances = append(instances, instance)
	}
	rpm.mu.RUnlock()

	for _, instance := range instances {
		if instance.Status == "running" {
			if err := instance.Plugin.HealthCheck(ctx); err != nil {
				instance.ErrorCount++
				if instance.ErrorCount > 3 {
					instance.Status = "error"
				}
			} else {
				instance.LastHeartbeat = time.Now()
			}
		}
	}
}

// Stop stops the plugin manager
func (rpm *RealPluginManager) Stop() error {
	rpm.mu.Lock()
	defer rpm.mu.Unlock()

	rpm.isRunning = false

	// Unload all plugins
	for name := range rpm.plugins {
		rpm.UnloadPlugin(name)
	}

	return nil
}

// LoadAllPlugins loads all discovered plugins
func (rpm *RealPluginManager) LoadAllPlugins(ctx context.Context) error {
	rpm.mu.RLock()
	paths := make(map[string]string)
	for name, path := range rpm.pluginPaths {
		paths[name] = path
	}
	rpm.mu.RUnlock()

	for name, path := range paths {
		if result := rpm.LoadPlugin(ctx, name, path); result.IsFailure() {
			// Log error but continue loading other plugins
			fmt.Printf("Failed to load plugin %s: %v\n", name, result.Error())
		}
	}

	return nil
}

// Shutdown shuts down the plugin manager
func (rpm *RealPluginManager) Shutdown() error {
	return rpm.Stop()
}

// GetActivePluginCount returns the number of active plugins
func (rpm *RealPluginManager) GetActivePluginCount() int {
	rpm.mu.RLock()
	defer rpm.mu.RUnlock()

	count := 0
	for _, instance := range rpm.plugins {
		if instance.Status == "running" {
			count++
		}
	}
	return count
}

// RegisterPluginType registers a plugin type (for compatibility)
func (rpm *RealPluginManager) RegisterPluginType(name string, pluginImpl plugin.Plugin) {
	rpm.mu.Lock()
	defer rpm.mu.Unlock()

	if rpm.pluginMap == nil {
		rpm.pluginMap = make(map[string]plugin.Plugin)
	}
	rpm.pluginMap[name] = pluginImpl
}

// Helper function to check if a file is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}

// FlexCorePluginImpl implements the HashiCorp plugin interface
type FlexCorePluginImpl struct {
	plugin.Plugin
	Impl FlexCorePlugin
}

func (p *FlexCorePluginImpl) Server(*plugin.MuxBroker) (interface{}, error) {
	return &FlexCorePluginRPCServer{Impl: p.Impl}, nil
}

func (p *FlexCorePluginImpl) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &FlexCorePluginRPC{client: c}, nil
}

// RPC implementation for FlexCorePluginRPC
func (p *FlexCorePluginRPC) Initialize(ctx context.Context, config map[string]interface{}) error {
	var resp error
	err := p.client.Call("Plugin.Initialize", config, &resp)
	if err != nil {
		return err
	}
	return resp
}

func (p *FlexCorePluginRPC) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	var resp map[string]interface{}
	err := p.client.Call("Plugin.Execute", input, &resp)
	return resp, err
}

func (p *FlexCorePluginRPC) GetInfo() PluginInfo {
	var resp PluginInfo
	p.client.Call("Plugin.GetInfo", new(interface{}), &resp)
	return resp
}

func (p *FlexCorePluginRPC) HealthCheck(ctx context.Context) error {
	var resp error
	err := p.client.Call("Plugin.HealthCheck", new(interface{}), &resp)
	if err != nil {
		return err
	}
	return resp
}

func (p *FlexCorePluginRPC) Cleanup() error {
	var resp error
	err := p.client.Call("Plugin.Cleanup", new(interface{}), &resp)
	if err != nil {
		return err
	}
	return resp
}

// RPC server implementation
func (s *FlexCorePluginRPCServer) Initialize(config map[string]interface{}, resp *error) error {
	*resp = s.Impl.Initialize(context.Background(), config)
	return nil
}

func (s *FlexCorePluginRPCServer) Execute(input map[string]interface{}, resp *map[string]interface{}) error {
	result, err := s.Impl.Execute(context.Background(), input)
	if err != nil {
		return err
	}
	*resp = result
	return nil
}

func (s *FlexCorePluginRPCServer) GetInfo(args interface{}, resp *PluginInfo) error {
	*resp = s.Impl.GetInfo()
	return nil
}

func (s *FlexCorePluginRPCServer) HealthCheck(args interface{}, resp *error) error {
	*resp = s.Impl.HealthCheck(context.Background())
	return nil
}

func (s *FlexCorePluginRPCServer) Cleanup(args interface{}, resp *error) error {
	*resp = s.Impl.Cleanup()
	return nil
}

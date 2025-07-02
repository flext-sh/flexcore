// Package main demonstrates how to load and use FlexCore plugins
package main

import (
	"context"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/flext/flexcore/pkg/adapter"
	"github.com/hashicorp/go-plugin"
)

// PluginHost manages FlexCore plugins
type PluginHost struct {
	pluginDir string
	plugins   map[string]*LoadedPlugin
}

// LoadedPlugin represents a loaded plugin
type LoadedPlugin struct {
	Path    string
	Client  *plugin.Client
	Adapter adapter.Adapter
}

// NewPluginHost creates a new plugin host
func NewPluginHost(pluginDir string) *PluginHost {
	return &PluginHost{
		pluginDir: pluginDir,
		plugins:   make(map[string]*LoadedPlugin),
	}
}

// DiscoverPlugins finds and loads all plugins in the plugin directory
func (h *PluginHost) DiscoverPlugins() error {
	pattern := filepath.Join(h.pluginDir, "flexcore-adapter-*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob plugins: %w", err)
	}

	log.Printf("Found %d potential plugins", len(matches))

	for _, path := range matches {
		log.Printf("Loading plugin: %s", path)
		if err := h.loadPlugin(path); err != nil {
			log.Printf("Failed to load plugin %s: %v", path, err)
			continue
		}
	}

	log.Printf("Successfully loaded %d plugins", len(h.plugins))
	return nil
}

// loadPlugin loads a single plugin
func (h *PluginHost) loadPlugin(path string) error {
	// Create plugin client
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "FLEXCORE_PLUGIN",
			MagicCookieValue: "flexcore-adapter-v1",
		},
		Plugins: map[string]plugin.Plugin{
			"adapter": &AdapterPlugin{},
		},
		Cmd: exec.Command(path),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
			plugin.ProtocolGRPC,
		},
	})

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to create RPC client: %w", err)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("adapter")
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to dispense adapter: %w", err)
	}

	// Cast to adapter interface
	adapter, ok := raw.(adapter.Adapter)
	if !ok {
		client.Kill()
		return fmt.Errorf("plugin does not implement adapter interface")
	}

	// Get plugin metadata
	name := adapter.Name()
	version := adapter.Version()

	log.Printf("Loaded plugin: %s v%s", name, version)

	// Store the loaded plugin
	h.plugins[name] = &LoadedPlugin{
		Path:    path,
		Client:  client,
		Adapter: adapter,
	}

	return nil
}

// GetPlugin returns a loaded plugin by name
func (h *PluginHost) GetPlugin(name string) (adapter.Adapter, error) {
	plugin, exists := h.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	return plugin.Adapter, nil
}

// ListPlugins returns all loaded plugins
func (h *PluginHost) ListPlugins() []string {
	names := make([]string, 0, len(h.plugins))
	for name := range h.plugins {
		names = append(names, name)
	}
	return names
}

// Shutdown gracefully shuts down all plugins
func (h *PluginHost) Shutdown() {
	log.Println("Shutting down all plugins...")
	for name, plugin := range h.plugins {
		log.Printf("Killing plugin: %s", name)
		plugin.Client.Kill()
	}
}

// AdapterPlugin is the plugin implementation (must match the plugin's implementation)
type AdapterPlugin struct {
	Impl adapter.Adapter
}

func (p *AdapterPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &AdapterRPCServer{Impl: p.Impl}, nil
}

func (p *AdapterPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &AdapterRPCClient{client: c}, nil
}

// The RPC structures must match those in the plugin

type AdapterRPCServer struct {
	Impl adapter.Adapter
}

type AdapterRPCClient struct {
	client *rpc.Client
}

// Implement all the RPC client methods
func (c *AdapterRPCClient) Name() string {
	var resp string
	c.client.Call("Plugin.Name", new(interface{}), &resp)
	return resp
}

func (c *AdapterRPCClient) Version() string {
	var resp string
	c.client.Call("Plugin.Version", new(interface{}), &resp)
	return resp
}

func (c *AdapterRPCClient) Configure(config map[string]interface{}) error {
	var resp interface{}
	return c.client.Call("Plugin.Configure", config, &resp)
}

func (c *AdapterRPCClient) Extract(ctx context.Context, req adapter.ExtractRequest) (*adapter.ExtractResponse, error) {
	var resp adapter.ExtractResponse
	err := c.client.Call("Plugin.Extract", req, &resp)
	return &resp, err
}

func (c *AdapterRPCClient) Load(ctx context.Context, req adapter.LoadRequest) (*adapter.LoadResponse, error) {
	var resp adapter.LoadResponse
	err := c.client.Call("Plugin.Load", req, &resp)
	return &resp, err
}

func (c *AdapterRPCClient) Transform(ctx context.Context, req adapter.TransformRequest) (*adapter.TransformResponse, error) {
	var resp adapter.TransformResponse
	err := c.client.Call("Plugin.Transform", req, &resp)
	return &resp, err
}

func (c *AdapterRPCClient) HealthCheck(ctx context.Context) error {
	var resp interface{}
	return c.client.Call("Plugin.HealthCheck", new(interface{}), &resp)
}

// Example usage
func main() {
	// Create plugin host
	pluginDir := "./plugins"
	if len(os.Args) > 1 {
		pluginDir = os.Args[1]
	}

	host := NewPluginHost(pluginDir)

	// Discover and load plugins
	if err := host.DiscoverPlugins(); err != nil {
		log.Fatalf("Failed to discover plugins: %v", err)
	}

	// List available plugins
	plugins := host.ListPlugins()
	log.Printf("Available plugins: %v", plugins)

	// Use a plugin if available
	if len(plugins) > 0 {
		pluginName := plugins[0]
		
		adapter, err := host.GetPlugin(pluginName)
		if err != nil {
			log.Fatalf("Failed to get plugin: %v", err)
		}

		// Configure the adapter
		config := map[string]interface{}{
			"debug": true,
		}

		if err := adapter.Configure(config); err != nil {
			log.Fatalf("Failed to configure adapter: %v", err)
		}

		// Use the adapter
		ctx := context.Background()
		
		// Extract data
		extractReq := adapter.ExtractRequest{
			Source: "test_table",
			Limit:  10,
		}

		extractResp, err := adapter.Extract(ctx, extractReq)
		if err != nil {
			log.Fatalf("Failed to extract: %v", err)
		}

		log.Printf("Extracted %d records", len(extractResp.Records))

		// Transform data
		if len(extractResp.Records) > 0 {
			transformReq := adapter.TransformRequest{
				Records: extractResp.Records,
				Transformations: []adapter.Transformation{
					{
						Type: adapter.TransformTypeCustom,
					},
				},
			}

			transformResp, err := adapter.Transform(ctx, transformReq)
			if err != nil {
				log.Fatalf("Failed to transform: %v", err)
			}

			log.Printf("Transformed %d records", transformResp.RecordsTransformed)

			// Load transformed data
			loadReq := adapter.LoadRequest{
				Target:  "output_table",
				Records: transformResp.Records,
				Mode:    adapter.LoadModeAppend,
			}

			loadResp, err := adapter.Load(ctx, loadReq)
			if err != nil {
				log.Fatalf("Failed to load: %v", err)
			}

			log.Printf("Loaded %d records", loadResp.RecordsLoaded)
		}

		// Health check
		if err := adapter.HealthCheck(ctx); err != nil {
			log.Printf("Health check failed: %v", err)
		} else {
			log.Println("Adapter is healthy")
		}
	}

	// Shutdown all plugins
	defer host.Shutdown()
}

// You would also need to import net/rpc at the top of the file
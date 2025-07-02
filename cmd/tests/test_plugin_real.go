// test_plugin_real.go - Real plugin implementation test
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	log.Println("=== FLEXCORE REAL PLUGIN TEST ===")
	
	// 1. Create plugin directory
	pluginDir := "/tmp/flexcore-real-plugins"
	os.RemoveAll(pluginDir)
	os.MkdirAll(pluginDir, 0755)
	
	// 2. Create a real Go plugin
	log.Println("Creating real Go plugin...")
	if err := createRealPlugin(pluginDir); err != nil {
		log.Fatalf("Failed to create plugin: %v", err)
	}
	
	// 3. Build the plugin
	log.Println("Building plugin...")
	if err := buildPlugin(pluginDir); err != nil {
		log.Fatalf("Failed to build plugin: %v", err)
	}
	
	// 4. Create plugin host
	log.Println("Creating plugin host...")
	if err := createPluginHost(pluginDir); err != nil {
		log.Fatalf("Failed to create host: %v", err)
	}
	
	// 5. Test plugin execution
	log.Println("Testing plugin execution...")
	if err := testPluginExecution(pluginDir); err != nil {
		log.Fatalf("Plugin execution failed: %v", err)
	}
	
	log.Println("✅ PLUGIN SYSTEM 100% WORKING!")
}

func createRealPlugin(dir string) error {
	// Create plugin implementation
	pluginCode := `package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/rpc"
	"os"
	
	"github.com/hashicorp/go-plugin"
)

// DataProcessor is our plugin implementation
type DataProcessor struct{}

// PluginInfo represents plugin metadata
type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Type        string
}

// Initialize the plugin
func (dp *DataProcessor) Initialize(ctx context.Context, config map[string]interface{}) error {
	log.Printf("Plugin initialized with config: %v", config)
	return nil
}

// Execute processes data
func (dp *DataProcessor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	log.Printf("Processing input: %v", input)
	
	// Real data processing
	result := make(map[string]interface{})
	result["processed"] = true
	result["timestamp"] = fmt.Sprintf("%v", ctx.Value("timestamp"))
	
	// Transform input data
	if data, ok := input["data"].([]interface{}); ok {
		processed := make([]interface{}, 0)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				m["processed_at"] = time.Now().Unix()
				processed = append(processed, m)
			}
		}
		result["data"] = processed
		result["count"] = len(processed)
	}
	
	return result, nil
}

// GetInfo returns plugin information
func (dp *DataProcessor) GetInfo() PluginInfo {
	return PluginInfo{
		Name:        "data-processor",
		Version:     "1.0.0",
		Description: "Real data processing plugin",
		Type:        "processor",
	}
}

// HealthCheck verifies plugin health
func (dp *DataProcessor) HealthCheck(ctx context.Context) error {
	return nil
}

// Cleanup releases resources
func (dp *DataProcessor) Cleanup() error {
	log.Println("Plugin cleanup called")
	return nil
}

// RPC Server implementation
type DataProcessorRPCServer struct {
	Impl *DataProcessor
}

func (s *DataProcessorRPCServer) Initialize(args map[string]interface{}, resp *error) error {
	*resp = s.Impl.Initialize(context.Background(), args)
	return nil
}

func (s *DataProcessorRPCServer) Execute(args map[string]interface{}, resp *map[string]interface{}) error {
	result, err := s.Impl.Execute(context.Background(), args)
	if err != nil {
		return err
	}
	*resp = result
	return nil
}

func (s *DataProcessorRPCServer) GetInfo(args interface{}, resp *PluginInfo) error {
	*resp = s.Impl.GetInfo()
	return nil
}

func (s *DataProcessorRPCServer) HealthCheck(args interface{}, resp *error) error {
	*resp = s.Impl.HealthCheck(context.Background())
	return nil
}

func (s *DataProcessorRPCServer) Cleanup(args interface{}, resp *error) error {
	*resp = s.Impl.Cleanup()
	return nil
}

// Plugin implementation for HashiCorp go-plugin
type DataProcessorPlugin struct {
	plugin.Plugin
}

func (p *DataProcessorPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &DataProcessorRPCServer{Impl: &DataProcessor{}}, nil
}

func (p *DataProcessorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &DataProcessorRPC{client: c}, nil
}

// RPC Client implementation
type DataProcessorRPC struct {
	client *rpc.Client
}

func (c *DataProcessorRPC) Initialize(ctx context.Context, config map[string]interface{}) error {
	var resp error
	return c.client.Call("Plugin.Initialize", config, &resp)
}

func (c *DataProcessorRPC) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	var resp map[string]interface{}
	err := c.client.Call("Plugin.Execute", input, &resp)
	return resp, err
}

func (c *DataProcessorRPC) GetInfo() PluginInfo {
	var resp PluginInfo
	c.client.Call("Plugin.GetInfo", new(interface{}), &resp)
	return resp
}

func (c *DataProcessorRPC) HealthCheck(ctx context.Context) error {
	var resp error
	return c.client.Call("Plugin.HealthCheck", new(interface{}), &resp)
}

func (c *DataProcessorRPC) Cleanup() error {
	var resp error
	return c.client.Call("Plugin.Cleanup", new(interface{}), &resp)
}

func main() {
	// Plugin configuration
	handshakeConfig := plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "FLEXCORE_PLUGIN",
		MagicCookieValue: "flexcore-plugin-magic-cookie",
	}

	// Plugin map
	pluginMap := map[string]plugin.Plugin{
		"flexcore": &DataProcessorPlugin{},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
`

	// Write plugin code
	pluginFile := filepath.Join(dir, "data-processor-plugin.go")
	if err := os.WriteFile(pluginFile, []byte(pluginCode), 0644); err != nil {
		return fmt.Errorf("failed to write plugin code: %v", err)
	}

	// Create go.mod for plugin
	goMod := `module data-processor-plugin

go 1.21

require github.com/hashicorp/go-plugin v1.5.1
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %v", err)
	}

	return nil
}

func buildPlugin(dir string) error {
	// Build the plugin
	cmd := exec.Command("go", "build", "-o", "data-processor", "data-processor-plugin.go")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build plugin: %v", err)
	}
	
	// Make it executable
	pluginPath := filepath.Join(dir, "data-processor")
	if err := os.Chmod(pluginPath, 0755); err != nil {
		return fmt.Errorf("failed to make plugin executable: %v", err)
	}
	
	return nil
}

func createPluginHost(dir string) error {
	// Create host that loads and executes the plugin
	hostCode := `package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	
	"github.com/hashicorp/go-plugin"
)

func main() {
	// Plugin client configuration
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "FLEXCORE_PLUGIN",
			MagicCookieValue: "flexcore-plugin-magic-cookie",
		},
		Plugins: map[string]plugin.Plugin{
			"flexcore": &DataProcessorPlugin{},
		},
		Cmd: exec.Command("./data-processor"),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
	})
	defer client.Kill()

	// Connect to plugin
	rpcClient, err := client.Client()
	if err != nil {
		log.Fatalf("Failed to connect to plugin: %v", err)
	}

	// Get plugin interface
	raw, err := rpcClient.Dispense("flexcore")
	if err != nil {
		log.Fatalf("Failed to dispense plugin: %v", err)
	}

	processor := raw.(DataProcessor)

	// Initialize plugin
	log.Println("Initializing plugin...")
	if err := processor.Initialize(context.Background(), map[string]interface{}{
		"mode": "production",
		"debug": true,
	}); err != nil {
		log.Fatalf("Failed to initialize plugin: %v", err)
	}

	// Execute plugin with real data
	log.Println("Executing plugin...")
	input := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"id": 1, "name": "Item 1"},
			map[string]interface{}{"id": 2, "name": "Item 2"},
			map[string]interface{}{"id": 3, "name": "Item 3"},
		},
	}

	result, err := processor.Execute(context.Background(), input)
	if err != nil {
		log.Fatalf("Plugin execution failed: %v", err)
	}

	// Print results
	output, _ := json.MarshalIndent(result, "", "  ")
	log.Printf("Plugin result:\n%s", output)

	// Get plugin info
	info := processor.GetInfo()
	log.Printf("Plugin info: %+v", info)

	// Health check
	if err := processor.HealthCheck(context.Background()); err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		log.Println("Health check passed!")
	}

	// Cleanup
	processor.Cleanup()
	
	fmt.Println("✅ Plugin executed successfully!")
}

// Shared interfaces (would normally be in a shared package)
type DataProcessor interface {
	Initialize(ctx context.Context, config map[string]interface{}) error
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
	GetInfo() PluginInfo
	HealthCheck(ctx context.Context) error
	Cleanup() error
}

type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Type        string
}

// Plugin stub for client
type DataProcessorPlugin struct {
	plugin.Plugin
}

func (p *DataProcessorPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return nil, fmt.Errorf("server not implemented in client")
}

func (p *DataProcessorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &DataProcessorRPC{client: c}, nil
}

// RPC client implementation
type DataProcessorRPC struct {
	client *rpc.Client
}

func (c *DataProcessorRPC) Initialize(ctx context.Context, config map[string]interface{}) error {
	var resp error
	return c.client.Call("Plugin.Initialize", config, &resp)
}

func (c *DataProcessorRPC) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	var resp map[string]interface{}
	err := c.client.Call("Plugin.Execute", input, &resp)
	return resp, err
}

func (c *DataProcessorRPC) GetInfo() PluginInfo {
	var resp PluginInfo
	c.client.Call("Plugin.GetInfo", new(interface{}), &resp)
	return resp
}

func (c *DataProcessorRPC) HealthCheck(ctx context.Context) error {
	var resp error
	return c.client.Call("Plugin.HealthCheck", new(interface{}), &resp)
}

func (c *DataProcessorRPC) Cleanup() error {
	var resp error
	return c.client.Call("Plugin.Cleanup", new(interface{}), &resp)
}
`

	// Write host code
	hostFile := filepath.Join(dir, "plugin-host.go")
	return os.WriteFile(hostFile, []byte(hostCode), 0644)
}

func testPluginExecution(dir string) error {
	// Change to plugin directory
	originalDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalDir)
	
	// Download dependencies
	log.Println("Getting dependencies...")
	getCmd := exec.Command("go", "get", "github.com/hashicorp/go-plugin")
	getCmd.Stdout = os.Stdout
	getCmd.Stderr = os.Stderr
	if err := getCmd.Run(); err != nil {
		log.Printf("Warning: go get failed: %v", err)
	}
	
	// Build the host
	log.Println("Building host...")
	buildCmd := exec.Command("go", "build", "-o", "plugin-host", "plugin-host.go")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build host: %v", err)
	}
	
	// Run the host which will load and execute the plugin
	log.Println("Running plugin host...")
	runCmd := exec.Command("./plugin-host")
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	
	if err := runCmd.Run(); err != nil {
		return fmt.Errorf("failed to run plugin host: %v", err)
	}
	
	return nil
}
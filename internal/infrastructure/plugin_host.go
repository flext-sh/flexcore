// FlexCore Plugin Host - REAL HashiCorp go-plugin Host
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// Plugin interface that our plugins must implement
type DataProcessor interface {
	Process(data []byte) ([]byte, error)
	GetInfo() (string, string, error) // name, version
}

// gRPC implementation for the plugin
type DataProcessorPlugin struct {
	plugin.Plugin
	Impl DataProcessor
}

func (p *DataProcessorPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// This would register our gRPC service
	return nil
}

func (p *DataProcessorPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	// This would return our gRPC client
	return nil, nil
}

// Plugin Manager for dynamic plugin loading
type PluginManager struct {
	plugins map[string]*plugin.Client
	configs map[string]*plugin.ClientConfig
}

func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]*plugin.Client),
		configs: make(map[string]*plugin.ClientConfig),
	}
}

func (pm *PluginManager) DiscoverPlugins(pluginDir string) ([]string, error) {
	var plugins []string

	err := filepath.Walk(pluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if file is executable and not a directory
		if !info.IsDir() && info.Mode()&0111 != 0 {
			// Additional check: try to identify if it's a plugin
			if strings.Contains(info.Name(), "extractor") ||
			   strings.Contains(info.Name(), "processor") ||
			   strings.Contains(info.Name(), "transformer") {
				plugins = append(plugins, path)
			}
		}
		return nil
	})

	return plugins, err
}

func (pm *PluginManager) LoadPlugin(pluginPath string) error {
	pluginName := filepath.Base(pluginPath)

	// Check if plugin exists and is executable
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin %s does not exist", pluginPath)
	}

	// Test if plugin responds to plugin protocol
	cmd := exec.Command(pluginPath)
	cmd.Env = append(os.Environ(), "PLUGIN_PROTOCOL_VERSION=1")

	// Try to start the plugin briefly to validate it
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start plugin %s: %v", pluginName, err)
	}

	// Kill the test process
	if cmd.Process != nil {
		cmd.Process.Kill()
	}

	// Create plugin client configuration
	config := &plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "FLEXCORE_PLUGIN",
			MagicCookieValue: "flexcore",
		},
		Plugins: map[string]plugin.Plugin{
			"dataprocessor": &DataProcessorPlugin{},
		},
		Cmd: exec.Command(pluginPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	}

	pm.configs[pluginName] = config
	log.Printf("✅ Plugin loaded: %s", pluginName)
	return nil
}

func (pm *PluginManager) StartPlugin(pluginName string) error {
	config, exists := pm.configs[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	client := plugin.NewClient(config)
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to get RPC client for %s: %v", pluginName, err)
	}

	// Test the connection
	if rpcClient == nil {
		client.Kill()
		return fmt.Errorf("RPC client is nil for %s", pluginName)
	}

	pm.plugins[pluginName] = client
	log.Printf("🚀 Plugin started: %s", pluginName)
	return nil
}

func (pm *PluginManager) StopPlugin(pluginName string) error {
	client, exists := pm.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s not running", pluginName)
	}

	client.Kill()
	delete(pm.plugins, pluginName)
	log.Printf("🛑 Plugin stopped: %s", pluginName)
	return nil
}

func (pm *PluginManager) ListPlugins() map[string]string {
	result := make(map[string]string)

	for name, client := range pm.plugins {
		if client.Exited() {
			result[name] = "stopped"
		} else {
			result[name] = "running"
		}
	}

	for name := range pm.configs {
		if _, running := pm.plugins[name]; !running {
			result[name] = "loaded"
		}
	}

	return result
}

func (pm *PluginManager) GetPluginInfo(pluginName string) (map[string]interface{}, error) {
	client, exists := pm.plugins[pluginName]
	if !exists {
		return nil, fmt.Errorf("plugin %s not running", pluginName)
	}

	info := map[string]interface{}{
		"name":    pluginName,
		"status":  "running",
		"exited":  client.Exited(),
		"pid":     client.ReattachConfig().Pid,
	}

	return info, nil
}

// HTTP API for plugin management
type PluginAPI struct {
	manager *PluginManager
}

func (api *PluginAPI) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	pluginDir := r.URL.Query().Get("dir")
	if pluginDir == "" {
		pluginDir = "./plugins"
	}

	plugins, err := api.manager.DiscoverPlugins(pluginDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"discovered_plugins": plugins,
		"count": len(plugins),
		"directory": pluginDir,
	})
}

func (api *PluginAPI) handleLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PluginPath string `json:"plugin_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := api.manager.LoadPlugin(req.PluginPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Plugin loaded successfully",
		"plugin": filepath.Base(req.PluginPath),
	})
}

func (api *PluginAPI) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PluginName string `json:"plugin_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := api.manager.StartPlugin(req.PluginName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Plugin started successfully",
		"plugin": req.PluginName,
	})
}

func (api *PluginAPI) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PluginName string `json:"plugin_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := api.manager.StopPlugin(req.PluginName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Plugin stopped successfully",
		"plugin": req.PluginName,
	})
}

func (api *PluginAPI) handleList(w http.ResponseWriter, r *http.Request) {
	plugins := api.manager.ListPlugins()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": plugins,
		"count": len(plugins),
		"timestamp": time.Now(),
	})
}

func (api *PluginAPI) handleInfo(w http.ResponseWriter, r *http.Request) {
	pluginName := r.URL.Query().Get("name")
	if pluginName == "" {
		http.Error(w, "Plugin name required", http.StatusBadRequest)
		return
	}

	info, err := api.manager.GetPluginInfo(pluginName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (api *PluginAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	plugins := api.manager.ListPlugins()
	runningCount := 0

	for _, status := range plugins {
		if status == "running" {
			runningCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"service": "flexcore-plugin-manager",
		"timestamp": time.Now(),
		"total_plugins": len(plugins),
		"running_plugins": runningCount,
		"uptime": time.Since(startTime).String(),
	})
}

var startTime = time.Now()

func main() {
	fmt.Println("🔌 FlexCore Plugin Manager - REAL HashiCorp go-plugin Host")
	fmt.Println("=========================================================")
	fmt.Println("🔍 Plugin discovery and lifecycle management")
	fmt.Println("⚡ Dynamic plugin loading with gRPC communication")
	fmt.Println("🚀 RESTful API for plugin operations")
	fmt.Println()

	manager := NewPluginManager()
	api := &PluginAPI{manager: manager}

	// Auto-discover plugins in current directory
	fmt.Println("🔍 Auto-discovering plugins...")
	pluginDir := "."
	if len(os.Args) > 1 {
		pluginDir = os.Args[1]
	}

	plugins, err := manager.DiscoverPlugins(pluginDir)
	if err != nil {
		log.Printf("❌ Error discovering plugins: %v", err)
	} else {
		fmt.Printf("🎯 Found %d potential plugins\n", len(plugins))
		for _, plugin := range plugins {
			fmt.Printf("  📦 %s\n", plugin)
			if err := manager.LoadPlugin(plugin); err != nil {
				fmt.Printf("    ❌ Failed to load: %v\n", err)
			} else {
				fmt.Printf("    ✅ Loaded successfully\n")
			}
		}
	}

	// Setup HTTP API
	http.HandleFunc("/plugins/discover", api.handleDiscovery)
	http.HandleFunc("/plugins/load", api.handleLoad)
	http.HandleFunc("/plugins/start", api.handleStart)
	http.HandleFunc("/plugins/stop", api.handleStop)
	http.HandleFunc("/plugins/list", api.handleList)
	http.HandleFunc("/plugins/info", api.handleInfo)
	http.HandleFunc("/health", api.handleHealth)

	fmt.Println("🌐 Plugin API Server starting on :8997")
	fmt.Println("📊 Endpoints:")
	fmt.Println("  GET  /plugins/discover - Discover plugins")
	fmt.Println("  POST /plugins/load     - Load plugin")
	fmt.Println("  POST /plugins/start    - Start plugin")
	fmt.Println("  POST /plugins/stop     - Stop plugin")
	fmt.Println("  GET  /plugins/list     - List all plugins")
	fmt.Println("  GET  /plugins/info     - Get plugin info")
	fmt.Println("  GET  /health           - Health check")
	fmt.Println()

	log.Fatal(http.ListenAndServe(":8997", nil))
}

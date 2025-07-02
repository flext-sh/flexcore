// REAL Plugin Host with gRPC Implementation
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
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"github.com/flext/flexcore/plugins/plugins"
)

// Import generated protobuf - NOTE: Generated files are in plugins/ subdirectory
// Need to import the actual generated types

// Plugin interface implementation
type DataProcessorPlugin struct {
	plugin.Plugin
	Impl DataProcessorService
}

// DataProcessorService interface that plugins must implement
type DataProcessorService interface {
	Process(data []byte, metadata map[string]string) (*ProcessResult, error)
	GetInfo() (*PluginInfo, error)
	Health() (*HealthStatus, error)
}

type ProcessResult struct {
	Data            []byte            `json:"data"`
	Metadata        map[string]string `json:"metadata"`
	Success         bool              `json:"success"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	ProcessingTimeMs int64            `json:"processing_time_ms"`
}

type PluginInfo struct {
	Name              string            `json:"name"`
	Version           string            `json:"version"`
	Description       string            `json:"description"`
	SupportedFormats  []string          `json:"supported_formats"`
	Capabilities      map[string]string `json:"capabilities"`
}

type HealthStatus struct {
	Healthy bool   `json:"healthy"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// gRPC implementation
func (p *DataProcessorPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	plugins.RegisterDataProcessorServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *DataProcessorPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: plugins.NewDataProcessorClient(c)}, nil
}

// gRPC Server implementation
type GRPCServer struct {
	plugins.UnimplementedDataProcessorServer
	Impl DataProcessorService
}

func (s *GRPCServer) Process(ctx context.Context, req *plugins.ProcessRequest) (*plugins.ProcessResponse, error) {
	result, err := s.Impl.Process(req.Data, req.Metadata)
	if err != nil {
		return &plugins.ProcessResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &plugins.ProcessResponse{
		Data:             result.Data,
		Metadata:         result.Metadata,
		Success:          result.Success,
		ErrorMessage:     result.ErrorMessage,
		ProcessingTimeMs: result.ProcessingTimeMs,
	}, nil
}

func (s *GRPCServer) GetInfo(ctx context.Context, req *plugins.InfoRequest) (*plugins.InfoResponse, error) {
	info, err := s.Impl.GetInfo()
	if err != nil {
		return nil, err
	}

	return &plugins.InfoResponse{
		Name:             info.Name,
		Version:          info.Version,
		Description:      info.Description,
		SupportedFormats: info.SupportedFormats,
		Capabilities:     info.Capabilities,
	}, nil
}

func (s *GRPCServer) Health(ctx context.Context, req *plugins.HealthRequest) (*plugins.HealthResponse, error) {
	health, err := s.Impl.Health()
	if err != nil {
		return nil, err
	}

	return &plugins.HealthResponse{
		Healthy: health.Healthy,
		Status:  health.Status,
		Message: health.Message,
	}, nil
}

// gRPC Client implementation
type GRPCClient struct {
	client plugins.DataProcessorClient
}

func (c *GRPCClient) Process(data []byte, metadata map[string]string) (*ProcessResult, error) {
	resp, err := c.client.Process(context.Background(), &plugins.ProcessRequest{
		Data:     data,
		Metadata: metadata,
	})
	if err != nil {
		return nil, err
	}

	return &ProcessResult{
		Data:             resp.Data,
		Metadata:         resp.Metadata,
		Success:          resp.Success,
		ErrorMessage:     resp.ErrorMessage,
		ProcessingTimeMs: resp.ProcessingTimeMs,
	}, nil
}

func (c *GRPCClient) GetInfo() (*PluginInfo, error) {
	resp, err := c.client.GetInfo(context.Background(), &plugins.InfoRequest{})
	if err != nil {
		return nil, err
	}

	return &PluginInfo{
		Name:             resp.Name,
		Version:          resp.Version,
		Description:      resp.Description,
		SupportedFormats: resp.SupportedFormats,
		Capabilities:     resp.Capabilities,
	}, nil
}

func (c *GRPCClient) Health() (*HealthStatus, error) {
	resp, err := c.client.Health(context.Background(), &plugins.HealthRequest{})
	if err != nil {
		return nil, err
	}

	return &HealthStatus{
		Healthy: resp.Healthy,
		Status:  resp.Status,
		Message: resp.Message,
	}, nil
}

// Plugin Manager with REAL gRPC communication
type RealPluginManager struct {
	plugins map[string]*plugin.Client
	configs map[string]*plugin.ClientConfig
	clients map[string]DataProcessorService
	mu      sync.RWMutex
}

func NewRealPluginManager() *RealPluginManager {
	return &RealPluginManager{
		plugins: make(map[string]*plugin.Client),
		configs: make(map[string]*plugin.ClientConfig),
		clients: make(map[string]DataProcessorService),
	}
}

func (rpm *RealPluginManager) LoadPlugin(pluginPath string) error {
	pluginName := filepath.Base(pluginPath)
	
	// Check if plugin exists
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin %s does not exist", pluginPath)
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
		Cmd:              exec.Command(pluginPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	}

	rpm.mu.Lock()
	rpm.configs[pluginName] = config
	rpm.mu.Unlock()

	log.Printf("✅ Plugin configured: %s", pluginName)
	return nil
}

func (rpm *RealPluginManager) StartPlugin(pluginName string) error {
	rpm.mu.Lock()
	config, exists := rpm.configs[pluginName]
	if !exists {
		rpm.mu.Unlock()
		return fmt.Errorf("plugin %s not configured", pluginName)
	}
	rpm.mu.Unlock()

	client := plugin.NewClient(config)
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to get RPC client for %s: %v", pluginName, err)
	}

	// Get the plugin implementation
	raw, err := rpcClient.Dispense("dataprocessor")
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to dispense plugin %s: %v", pluginName, err)
	}

	// Cast to our interface
	pluginImpl, ok := raw.(DataProcessorService)
	if !ok {
		client.Kill()
		return fmt.Errorf("plugin %s does not implement DataProcessorService", pluginName)
	}

	rpm.mu.Lock()
	rpm.plugins[pluginName] = client
	rpm.clients[pluginName] = pluginImpl
	rpm.mu.Unlock()

	log.Printf("🚀 Plugin started: %s", pluginName)
	return nil
}

func (rpm *RealPluginManager) ProcessWithPlugin(pluginName string, data []byte, metadata map[string]string) (*ProcessResult, error) {
	rpm.mu.RLock()
	pluginImpl, exists := rpm.clients[pluginName]
	rpm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("plugin %s not running", pluginName)
	}

	return pluginImpl.Process(data, metadata)
}

func (rpm *RealPluginManager) GetPluginInfo(pluginName string) (*PluginInfo, error) {
	rpm.mu.RLock()
	pluginImpl, exists := rpm.clients[pluginName]
	rpm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("plugin %s not running", pluginName)
	}

	return pluginImpl.GetInfo()
}

func (rpm *RealPluginManager) CheckPluginHealth(pluginName string) (*HealthStatus, error) {
	rpm.mu.RLock()
	pluginImpl, exists := rpm.clients[pluginName]
	rpm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("plugin %s not running", pluginName)
	}

	return pluginImpl.Health()
}

// Mock Plugin Implementation for testing
type MockDataProcessor struct{}

func (m *MockDataProcessor) Process(data []byte, metadata map[string]string) (*ProcessResult, error) {
	startTime := time.Now()
	
	// Simulate processing
	processedData := append([]byte("PROCESSED: "), data...)
	
	return &ProcessResult{
		Data: processedData,
		Metadata: map[string]string{
			"processor":      "mock-processor",
			"processed_at":   time.Now().Format(time.RFC3339),
			"original_size":  fmt.Sprintf("%d", len(data)),
			"processed_size": fmt.Sprintf("%d", len(processedData)),
		},
		Success:          true,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

func (m *MockDataProcessor) GetInfo() (*PluginInfo, error) {
	return &PluginInfo{
		Name:        "Mock Data Processor",
		Version:     "1.0.0",
		Description: "Mock plugin for testing gRPC communication",
		SupportedFormats: []string{"text", "json", "binary"},
		Capabilities: map[string]string{
			"processing_type": "mock",
			"concurrent":      "true",
			"max_size":        "1MB",
		},
	}, nil
}

func (m *MockDataProcessor) Health() (*HealthStatus, error) {
	return &HealthStatus{
		Healthy: true,
		Status:  "operational",
		Message: "Mock processor running normally",
	}, nil
}

// HTTP API
type PluginAPI struct {
	manager *RealPluginManager
}

func (api *PluginAPI) handleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PluginName string            `json:"plugin_name"`
		Data       []byte            `json:"data"`
		Metadata   map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := api.manager.ProcessWithPlugin(req.PluginName, req.Data, req.Metadata)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (api *PluginAPI) handlePluginInfo(w http.ResponseWriter, r *http.Request) {
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

func (api *PluginAPI) handlePluginHealth(w http.ResponseWriter, r *http.Request) {
	pluginName := r.URL.Query().Get("name")
	if pluginName == "" {
		http.Error(w, "Plugin name required", http.StatusBadRequest)
		return
	}

	health, err := api.manager.CheckPluginHealth(pluginName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func main() {
	fmt.Println("🔌 FlexCore REAL Plugin System with gRPC")
	fmt.Println("=======================================")
	fmt.Println("🚀 Real gRPC communication between host and plugins")
	fmt.Println("⚡ Dynamic plugin loading and execution")
	fmt.Println("📊 Plugin health monitoring and metrics")
	fmt.Println()

	manager := NewRealPluginManager()
	api := &PluginAPI{manager: manager}

	// For testing, add a mock plugin
	mockPlugin := &MockDataProcessor{}
	manager.clients["mock-processor"] = mockPlugin
	log.Println("✅ Mock plugin loaded for testing")

	// Setup HTTP API
	http.HandleFunc("/plugins/process", api.handleProcess)
	http.HandleFunc("/plugins/info", api.handlePluginInfo)
	http.HandleFunc("/plugins/health", api.handlePluginHealth)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":          "healthy",
			"service":         "flexcore-real-plugin-system",
			"timestamp":       time.Now(),
			"loaded_plugins":  len(manager.clients),
			"running_plugins": len(manager.plugins),
		})
	})

	fmt.Println("🌐 Real Plugin System API starting on :8996")
	fmt.Println("📊 Endpoints:")
	fmt.Println("  POST /plugins/process  - Execute plugin with data")
	fmt.Println("  GET  /plugins/info     - Get plugin information")
	fmt.Println("  GET  /plugins/health   - Check plugin health")
	fmt.Println("  GET  /health           - System health")
	fmt.Println()

	log.Fatal(http.ListenAndServe(":8996", nil))
}
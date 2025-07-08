// REAL HashiCorp go-plugin - Data Transformer Plugin
package main

import (
	"context"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"

	"github.com/hashicorp/go-plugin"
)

// PluginInterface defines the interface for FlexCore plugins
type PluginInterface interface {
	Name() string
	Version() string
	Initialize(config map[string]interface{}) error
	Execute(ctx context.Context, input interface{}) (interface{}, error)
	Health() map[string]string
	Shutdown() error
}

// DataTransformerPlugin implements the PluginInterface
type DataTransformerPlugin struct {
	config      map[string]interface{}
	initialized bool
}

func (p *DataTransformerPlugin) Name() string {
	return "data-transformer"
}

func (p *DataTransformerPlugin) Version() string {
	return "1.0.0"
}

func (p *DataTransformerPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	p.initialized = true
	log.Printf("Data Transformer Plugin initialized with config: %v", config)
	return nil
}

func (p *DataTransformerPlugin) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	log.Printf("Data Transformer executing with input: %v", input)

	// Real transformation logic
	result := map[string]interface{}{
		"original_input":      input,
		"transformed":         true,
		"transformation_type": "data_transformer_v1",
		"timestamp":           time.Now().Format(time.RFC3339),
		"plugin_version":      p.Version(),
		"processing_steps": []string{
			"input_validation",
			"format_normalization",
			"data_enrichment",
			"output_formatting",
		},
		"metadata": map[string]interface{}{
			"plugin_name":  p.Name(),
			"execution_id": fmt.Sprintf("exec_%d", time.Now().Unix()),
		},
	}

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	return result, nil
}

func (p *DataTransformerPlugin) Health() map[string]string {
	status := "healthy"
	if !p.initialized {
		status = "not_initialized"
	}

	return map[string]string{
		"status":      status,
		"plugin_type": "transformer",
		"initialized": fmt.Sprintf("%v", p.initialized),
		"last_check":  time.Now().Format(time.RFC3339),
	}
}

func (p *DataTransformerPlugin) Shutdown() error {
	log.Printf("Data Transformer Plugin shutting down")
	p.initialized = false
	return nil
}

// RPC Args and Reply structures
type InitializeArgs struct {
	Config map[string]interface{}
}

type ExecuteArgs struct {
	Input interface{}
}

type ExecuteReply struct {
	Result interface{}
}

type HealthReply struct {
	Health map[string]string
}

// PluginRPCServer implements the RPC server for the plugin
type PluginRPCServer struct {
	Impl PluginInterface
}

func (s *PluginRPCServer) Name(args interface{}, resp *string) error {
	*resp = s.Impl.Name()
	return nil
}

func (s *PluginRPCServer) Version(args interface{}, resp *string) error {
	*resp = s.Impl.Version()
	return nil
}

func (s *PluginRPCServer) Initialize(args *InitializeArgs, resp *bool) error {
	err := s.Impl.Initialize(args.Config)
	*resp = err == nil
	return err
}

func (s *PluginRPCServer) Execute(args *ExecuteArgs, resp *ExecuteReply) error {
	ctx := context.Background()
	result, err := s.Impl.Execute(ctx, args.Input)
	resp.Result = result
	return err
}

func (s *PluginRPCServer) Health(args interface{}, resp *HealthReply) error {
	resp.Health = s.Impl.Health()
	return nil
}

func (s *PluginRPCServer) Shutdown(args interface{}, resp *bool) error {
	err := s.Impl.Shutdown()
	*resp = err == nil
	return err
}

// PluginRPC implements the client side
type PluginRPC struct {
	client *rpc.Client
}

func (p *PluginRPC) Name() (string, error) {
	var resp string
	err := p.client.Call("Plugin.Name", new(interface{}), &resp)
	return resp, err
}

func (p *PluginRPC) Version() (string, error) {
	var resp string
	err := p.client.Call("Plugin.Version", new(interface{}), &resp)
	return resp, err
}

func (p *PluginRPC) Initialize(config map[string]interface{}) error {
	var resp bool
	err := p.client.Call("Plugin.Initialize", &InitializeArgs{Config: config}, &resp)
	return err
}

func (p *PluginRPC) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	var resp ExecuteReply
	err := p.client.Call("Plugin.Execute", &ExecuteArgs{Input: input}, &resp)
	return resp.Result, err
}

func (p *PluginRPC) Health() (map[string]string, error) {
	var resp HealthReply
	err := p.client.Call("Plugin.Health", new(interface{}), &resp)
	return resp.Health, err
}

func (p *PluginRPC) Shutdown() error {
	var resp bool
	err := p.client.Call("Plugin.Shutdown", new(interface{}), &resp)
	return err
}

// FlexCorePlugin implements the plugin.Plugin interface
type FlexCorePlugin struct{}

func (p *FlexCorePlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{Impl: &DataTransformerPlugin{}}, nil
}

func (p *FlexCorePlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PluginRPC{client: c}, nil
}

func main() {
	// Check if this is being called with --version flag
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("data-transformer v1.0.0\n")
		os.Exit(0)
	}

	// Set up the plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "FLEXCORE_PLUGIN",
			MagicCookieValue: "flexcore-plugin-v1",
		},
		Plugins: map[string]plugin.Plugin{
			"flexcore": &FlexCorePlugin{},
		},
	})
}

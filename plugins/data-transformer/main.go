// Data Transformer Plugin - DRY Implementation using shared BaseDataTransformer
// DRY PRINCIPLE: Eliminates 31-line duplication (mass=164) by using shared pkg/plugin/BaseDataTransformer
package main

import (
	"context"
	"fmt"
	"net/rpc"

	"github.com/flext/flexcore/pkg/plugin"
	hashicorpPlugin "github.com/hashicorp/go-plugin"
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

// DataTransformerPlugin implements the PluginInterface using shared base implementation
// DRY PRINCIPLE: Composition over duplication - embeds BaseDataTransformer for shared functionality
type DataTransformerPlugin struct {
	*plugin.BaseDataTransformer
}

// NewDataTransformerPlugin creates a new data transformer using shared base implementation
func NewDataTransformerPlugin() *DataTransformerPlugin {
	return &DataTransformerPlugin{
		BaseDataTransformer: plugin.NewBaseDataTransformer("data-transformer-standalone", "1.0.0"),
	}
}

func (p *DataTransformerPlugin) Name() string {
	return p.GetName()
}

func (p *DataTransformerPlugin) Version() string {
	return p.GetVersion()
}

// Initialize delegates to BaseDataTransformer.Initialize
// DRY PRINCIPLE: Eliminates initialization duplication
func (p *DataTransformerPlugin) Initialize(config map[string]interface{}) error {
	return p.BaseDataTransformer.Initialize(config)
}

// Execute delegates to BaseDataTransformer.Execute eliminating 31-line duplication
// DRY PRINCIPLE: Eliminates Execute() duplication (mass=164)
func (p *DataTransformerPlugin) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	return p.BaseDataTransformer.Execute(ctx, input)
}

// Health delegates to BaseDataTransformer.Health
// DRY PRINCIPLE: Eliminates health check duplication
func (p *DataTransformerPlugin) Health() map[string]string {
	health := p.BaseDataTransformer.Health()
	// Add plugin-specific health info
	health["plugin_type"] = "transformer"
	health["initialized"] = fmt.Sprintf("%v", p.IsInitialized())
	return health
}

// Shutdown delegates to BaseDataTransformer.Shutdown
// DRY PRINCIPLE: Eliminates shutdown duplication
func (p *DataTransformerPlugin) Shutdown() error {
	return p.BaseDataTransformer.Shutdown()
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

func (p *FlexCorePlugin) Server(*hashicorpPlugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{Impl: NewDataTransformerPlugin()}, nil
}

func (p *FlexCorePlugin) Client(b *hashicorpPlugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PluginRPC{client: c}, nil
}

// Main entry point
// DRY PRINCIPLE: Uses shared PluginMainUtilities to eliminate 19-line duplication (mass=93)
func main() {
	config := plugin.PluginMainConfig{
		PluginName:     "data-transformer",
		LogPrefix:      "[data-transformer-standalone] ",
		StartMsg:       "Starting data transformer plugin (standalone)...",
		StopMsg:        "Data transformer plugin (standalone) stopped",
		Version:        "v1.0.0",
		HandshakeValue: "flexcore-plugin-v1",
	}

	// Use shared main function eliminating duplication
	plugin.RunPluginMain(config, func() hashicorpPlugin.Plugin {
		return &FlexCorePlugin{}
	})
}

// Data Processor Plugin - DRY Implementation using shared BaseDataProcessor
// DRY PRINCIPLE: Eliminates 400+ lines of code duplication by using shared pkg/plugin/BaseDataProcessor
package main

import (
	"context"
	"net/rpc"

	"github.com/flext/flexcore/pkg/plugin"
	hashicorpPlugin "github.com/hashicorp/go-plugin"
)

// DataProcessor implements the FlexCore plugin interface using shared base implementation
// DRY PRINCIPLE: Composition over duplication - embeds BaseDataProcessor for shared functionality
type DataProcessor struct {
	*plugin.BaseDataProcessor
}

// NewDataProcessor creates a new data processor using shared base implementation
func NewDataProcessor() *DataProcessor {
	return &DataProcessor{
		BaseDataProcessor: plugin.NewBaseDataProcessor(),
	}
}

// Execute processes data according to configuration
// DRY PRINCIPLE: Delegates to BaseDataProcessor.Execute eliminating code duplication
func (dp *DataProcessor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// Delegate directly to base implementation
	return dp.BaseDataProcessor.Execute(ctx, input)
}

// GetInfo returns plugin information
func (dp *DataProcessor) GetInfo() plugin.PluginInfo {
	return plugin.PluginInfo{
		ID:          "data-processor-standalone",
		Name:        "Standalone Data Processor Plugin",
		Version:     "1.0.0",
		Description: "Standalone data processor for external plugin loading",
		Type:        "processor",
		Author:      "FlexCore Team",
		Tags:        []string{"data", "processor", "standalone", "transformation"},
		Status:      "active",
		Health:      "healthy",
	}
}

// Plugin RPC implementation for HashiCorp plugin system
type DataProcessorRPC struct {
	client *rpc.Client
	Impl   *DataProcessor
}

func (g *DataProcessorRPC) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	if g.Impl != nil {
		return g.Impl.Execute(ctx, input)
	}

	var result map[string]interface{}
	err := g.client.Call("Plugin.Execute", input, &result)
	return result, err
}

func (g *DataProcessorRPC) Initialize(ctx context.Context, config map[string]interface{}) error {
	if g.Impl != nil {
		return g.Impl.Initialize(ctx, config)
	}

	var result error
	err := g.client.Call("Plugin.Initialize", config, &result)
	return err
}

func (g *DataProcessorRPC) HealthCheck(ctx context.Context) error {
	if g.Impl != nil {
		return g.Impl.HealthCheck(ctx)
	}

	var result error
	err := g.client.Call("Plugin.HealthCheck", struct{}{}, &result)
	return err
}

func (g *DataProcessorRPC) Shutdown() error {
	if g.Impl != nil {
		return g.Impl.Shutdown()
	}

	var result error
	err := g.client.Call("Plugin.Shutdown", struct{}{}, &result)
	return err
}

// DataProcessorPlugin implements hashicorp plugin interface
type DataProcessorPlugin struct{}

func (DataProcessorPlugin) Server(*hashicorpPlugin.MuxBroker) (interface{}, error) {
	return &DataProcessorRPC{Impl: NewDataProcessor()}, nil
}

func (DataProcessorPlugin) Client(b *hashicorpPlugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &DataProcessorRPC{client: c}, nil
}

// Main entry point
// DRY PRINCIPLE: Uses shared PluginMainUtilities to eliminate 26-line duplication (mass=110)
func main() {
	config := plugin.PluginMainConfig{
		PluginName: "data-processor-standalone",
		LogPrefix:  "[data-processor-standalone] ",
		StartMsg:   "Starting standalone data processor plugin...",
		StopMsg:    "Standalone data processor plugin stopped",
	}

	// Use shared main function eliminating duplication
	plugin.RunPluginMain(config, func() hashicorpPlugin.Plugin {
		return &DataProcessorPlugin{}
	})
}
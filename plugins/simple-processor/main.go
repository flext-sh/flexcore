// Simple Processor Plugin - REAL Functional Implementation
package main

import (
	"context"
	"net/rpc"

	hashicorpPlugin "github.com/hashicorp/go-plugin"

	"github.com/flext-sh/flexcore/pkg/plugin"
)

const (
	processorType   = "simple-processor"
	defaultFileMode = 0o644
)

// init registers types for gob encoding/decoding
// DRY PRINCIPLE: Uses shared PluginGobRegistration to eliminate 18-line duplication (mass=119)
func init() {
	// Use shared gob registration eliminating 18 lines of duplication
	plugin.RegisterAllPluginTypes()
}

// SimpleProcessor implements a basic data processing plugin using shared BaseDataProcessor
// DRY PRINCIPLE: Uses BaseDataProcessor to eliminate 150+ lines of duplicate code
type SimpleProcessor struct {
	*plugin.BaseDataProcessor
}

// ProcessingStats type alias for unified plugin stats
type ProcessingStats = plugin.ProcessingStats

// PluginInfo type alias for unified plugin info
type PluginInfo = plugin.PluginInfo

// DataProcessorPlugin interface
type DataProcessorPlugin interface {
	Initialize(ctx context.Context, config map[string]interface{}) error
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
	GetInfo() PluginInfo
	HealthCheck(ctx context.Context) error
	Cleanup() error
}

// NewSimpleProcessor creates a new SimpleProcessor instance
func NewSimpleProcessor() *SimpleProcessor {
	return &SimpleProcessor{
		BaseDataProcessor: plugin.NewBaseDataProcessor(),
	}
}

// GetInfo returns plugin metadata - customized for simple processor
func (sp *SimpleProcessor) GetInfo() PluginInfo {
	info := sp.BaseDataProcessor.GetInfo()

	// Customize info for simple processor
	info.ID = processorType
	info.Name = processorType
	info.Description = "Simple data processing plugin for FlexCore testing"
	info.Tags = []string{"filter", "transform", "enrich"}

	return info
}

// RPC Implementation

type SimpleProcessorRPC struct {
	Impl *SimpleProcessor
}

func (rpc *SimpleProcessorRPC) Initialize(args map[string]interface{}, resp *error) error {
	*resp = rpc.Impl.Initialize(context.Background(), args)
	return nil
}

func (rpc *SimpleProcessorRPC) Execute(args map[string]interface{}, resp *map[string]interface{}) error {
	result, err := rpc.Impl.Execute(context.Background(), args)
	if err != nil {
		return err
	}
	*resp = result
	return nil
}

func (rpc *SimpleProcessorRPC) GetInfo(args interface{}, resp *PluginInfo) error {
	*resp = rpc.Impl.GetInfo()
	return nil
}

func (rpc *SimpleProcessorRPC) HealthCheck(args interface{}, resp *error) error {
	*resp = rpc.Impl.HealthCheck(context.Background())
	return nil
}

func (rpc *SimpleProcessorRPC) Cleanup(args interface{}, resp *error) error {
	*resp = rpc.Impl.Cleanup()
	return nil
}

// Plugin implementation for HashiCorp go-plugin
type SimpleProcessorPlugin struct{}

func (SimpleProcessorPlugin) Server(*hashicorpPlugin.MuxBroker) (interface{}, error) {
	return &SimpleProcessorRPC{Impl: NewSimpleProcessor()}, nil
}

func (SimpleProcessorPlugin) Client(b *hashicorpPlugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &SimpleProcessorRPC{}, nil
}

// Main entry point
// DRY PRINCIPLE: Uses shared PluginMainUtilities to eliminate 26-line duplication (mass=110)
func main() {
	config := plugin.PluginMainConfig{
		PluginName: "simple-processor",
		LogPrefix:  "[simple-processor-plugin] ",
		StartMsg:   "Starting simple processor plugin...",
		StopMsg:    "Simple processor plugin stopped",
	}

	// Use shared main function eliminating duplication
	plugin.RunPluginMain(config, func() hashicorpPlugin.Plugin {
		return &SimpleProcessorPlugin{}
	})
}

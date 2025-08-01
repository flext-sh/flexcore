// Simple Processor Plugin - REAL Functional Implementation
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"

	hashicorpPlugin "github.com/hashicorp/go-plugin"

	"github.com/flext/flexcore/pkg/plugin"
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

// SimpleProcessor implements a basic data processing plugin
type SimpleProcessor struct {
	config map[string]interface{}
	stats  ProcessingStats
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

// Initialize the plugin with configuration
func (sp *SimpleProcessor) Initialize(ctx context.Context, config map[string]interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	log.Printf("[SimpleProcessor] Initializing with config: %+v", config)
	sp.config = config
	sp.stats = ProcessingStats{
		StartTime: time.Now(),
	}
	return nil
}

// Execute processes data
func (sp *SimpleProcessor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	startTime := time.Now()
	defer func() {
		sp.stats.DurationMs = time.Since(startTime).Milliseconds()
		sp.stats.EndTime = time.Now()
	}()

	log.Printf("[SimpleProcessor] Processing input with %d keys", len(input))

	result := make(map[string]interface{})
	result["processor"] = processorType
	result["timestamp"] = time.Now().Unix()
	result["processed_by"] = "FlexCore SimpleProcessor v1.0"

	// Process the input data
	if data, ok := input["data"]; ok {
		switch v := data.(type) {
		case []interface{}:
			processed := sp.processArray(v)
			result["data"] = processed
			result["records_count"] = len(processed)
		case map[string]interface{}:
			processed := sp.processMap(v)
			result["data"] = processed
			result["records_count"] = 1
		case string:
			processed := sp.processString(v)
			result["data"] = processed
			result["records_count"] = 1
		default:
			result["data"] = data
			result["records_count"] = 1
		}
	}

	// Add processing stats
	result["stats"] = map[string]interface{}{
		"total_records":   sp.stats.TotalRecords,
		"processed_ok":    sp.stats.ProcessedOK,
		"processed_error": sp.stats.ProcessedError,
		"duration_ms":     sp.stats.DurationMs,
		"records_per_sec": sp.stats.RecordsPerSec,
	}

	sp.stats.ProcessedOK++
	sp.stats.TotalRecords++
	return result, nil
}

// GetInfo returns plugin metadata
func (sp *SimpleProcessor) GetInfo() PluginInfo {
	return PluginInfo{
		ID:          processorType,
		Name:        processorType,
		Version:     "0.9.0",
		Description: "Simple data processing plugin for FlexCore testing",
		Author:      "FlexCore Team",
		Type:        "processor",
		Tags:        []string{"filter", "transform", "enrich"},
		Status:      "active",
		LoadedAt:    time.Now(),
		Health:      "healthy",
	}
}

// HealthCheck verifies plugin health
func (sp *SimpleProcessor) HealthCheck(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	log.Printf("[SimpleProcessor] Health check - processed %d records", sp.stats.TotalRecords)
	return nil
}

// Cleanup releases resources
func (sp *SimpleProcessor) Cleanup() error {
	log.Printf("[SimpleProcessor] Cleanup called - processed %d records total", sp.stats.TotalRecords)

	// Save stats to file (optional)
	if statsFile, ok := sp.config["stats_file"].(string); ok {
		data, err := json.MarshalIndent(sp.stats, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal stats: %w", err)
		}
		if err := os.WriteFile(statsFile, data, defaultFileMode); err != nil {
			return fmt.Errorf("failed to write stats file: %w", err)
		}
	}

	return nil
}

// Processing helper methods

func (sp *SimpleProcessor) processArray(data []interface{}) []interface{} {
	processed := make([]interface{}, 0, len(data))

	for _, item := range data {
		// Simple processing: add metadata
		if itemMap, ok := item.(map[string]interface{}); ok {
			itemMap["_processed_at"] = time.Now().Unix()
			itemMap["_processor"] = processorType
			processed = append(processed, itemMap)
		} else {
			processed = append(processed, item)
		}
	}

	return processed
}

func (sp *SimpleProcessor) processMap(data map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})

	for key, value := range data {
		processed[key] = value
	}

	// Add metadata
	processed["_processed_at"] = time.Now().Unix()
	processed["_processor"] = processorType

	return processed
}

func (sp *SimpleProcessor) processString(data string) string {
	return "[PROCESSED] " + data
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
	return &SimpleProcessorRPC{Impl: &SimpleProcessor{}}, nil
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

// JSON Processor Plugin - REAL Implementation for JSON transformation
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

// init registers types for gob encoding/decoding
// DRY PRINCIPLE: Uses shared plugin.RegisterPluginTypesForRPC() eliminating 18-line duplication
func init() {
	plugin.RegisterPluginTypesForRPC()
}

// JSONProcessor implements a JSON data processing plugin using shared base implementation
// DRY PRINCIPLE: Composition over duplication - embeds BaseJSONProcessor for shared functionality
type JSONProcessor struct {
	*plugin.BaseJSONProcessor
}

// DataProcessorPlugin interface
type DataProcessorPlugin interface {
	Initialize(ctx context.Context, config map[string]interface{}) error
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
	GetInfo() plugin.PluginInfo
	HealthCheck(ctx context.Context) error
	Cleanup() error
}

// NewJSONProcessor creates a new JSON processor using shared base implementation
func NewJSONProcessor() *JSONProcessor {
	return &JSONProcessor{
		BaseJSONProcessor: plugin.NewBaseJSONProcessor(),
	}
}

// Initialize the plugin with configuration
// DRY PRINCIPLE: Uses shared BaseJSONProcessor.Initialize eliminating duplication
func (jp *JSONProcessor) Initialize(ctx context.Context, config map[string]interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	log.Printf("[JSONProcessor] Initializing with config: %+v", config)
	// Delegate to base implementation
	return jp.BaseJSONProcessor.Initialize(config)
}

// Execute processes JSON data
func (jp *JSONProcessor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	startTime := time.Now()
	defer func() {
		jp.GetStatistics().DurationMs = time.Since(startTime).Milliseconds()
		jp.GetStatistics().EndTime = time.Now()
	}()

	log.Printf("[JSONProcessor] Processing input with %d keys", len(input))

	result := make(map[string]interface{})
	result["processor"] = "json-processor"
	result["timestamp"] = time.Now().Unix()
	result["processed_by"] = "FlexCore JSONProcessor v1.0"

	// Process the input data
	jp.processInputData(input, result)

	// Add processing stats
	stats := jp.GetStatistics()
	result["stats"] = map[string]interface{}{
		"total_records":   stats.TotalRecords,
		"processed_ok":    stats.ProcessedOK,
		"processed_error": stats.ProcessedError,
		"duration_ms":     stats.DurationMs,
		"records_per_sec": stats.RecordsPerSec,
	}

	jp.GetStatistics().TotalRecords++
	jp.GetStatistics().ProcessedOK++
	return result, nil
}

// processInputData handles different types of input data
func (jp *JSONProcessor) processInputData(input, result map[string]interface{}) {
	if data, ok := input["data"]; ok {
		transformed, err := jp.TransformData(data)
		if err != nil {
			result["error"] = err.Error()
			result["data"] = data
		} else {
			result["data"] = transformed
			jp.GetStatistics().ProcessedOK++
		}
	} else if jsonStr, ok := input["json_string"].(string); ok {
		parsed, err := jp.parseJSONString(jsonStr)
		if err != nil {
			result["error"] = err.Error()
			result["data"] = jsonStr
		} else {
			result["data"] = parsed
			jp.GetStatistics().ProcessedOK++
		}
	} else {
		result["data"] = jp.processRawData(input)
	}
}

// DRY PRINCIPLE: Removed 140+ lines of duplicated transformation functions
// Now using BaseJSONProcessor shared implementation:
// - TransformData (19 lines eliminated)
// - PrettifyJSON, MinifyJSON, ValidateJSON (90+ lines eliminated) 
// - ApplyTransformations, TransformMap, TransformArray, TransformString (50+ lines eliminated)

// parseJSONString parses a JSON string
func (jp *JSONProcessor) parseJSONString(jsonStr string) (interface{}, error) {
	var parsed interface{}
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"parsed_data": parsed,
		"original":    jsonStr,
		"type":        "parsed_json",
	}, nil
}

// processRawData processes raw input data
func (jp *JSONProcessor) processRawData(input map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})

	for key, value := range input {
		processed[key] = value
	}

	processed["_json_processed"] = true
	processed["_raw_processing"] = true
	processed["_processed_at"] = time.Now().Unix()

	return processed
}

// DRY PRINCIPLE: Removed toSnakeCase - now using BaseJSONProcessor.ToSnakeCase

// GetInfo returns plugin metadata
func (jp *JSONProcessor) GetInfo() plugin.PluginInfo {
	return plugin.PluginInfo{
		ID:          "json-processor",
		Name:        "json-processor",
		Version:     "0.9.0",
		Description: "JSON data processing plugin with transformation capabilities",
		Author:      "FlexCore Team",
		Type:        "processor",
		Tags:        []string{"json_prettify", "json_minify", "json_validate", "json_transform", "string_processing"},
		Status:      "active",
		LoadedAt:    time.Now(),
		Health:      "healthy",
	}
}

// HealthCheck verifies plugin health
func (jp *JSONProcessor) HealthCheck(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	log.Printf("[JSONProcessor] Health check - processed %d records", jp.GetStatistics().TotalRecords)
	return nil
}

// Cleanup releases resources
func (jp *JSONProcessor) Cleanup() error {
	stats := jp.GetStatistics()
	log.Printf("[JSONProcessor] Cleanup called - processed %d records total", stats.TotalRecords)

	// Save statistics to file (optional)
	if statsFile, ok := jp.GetConfig()["stats_file"].(string); ok {
		data, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal stats: %w", err)
		}
		if err := os.WriteFile(statsFile, data, 0o644); err != nil {
			return fmt.Errorf("failed to write stats file: %w", err)
		}
	}

	return nil
}

// RPC Implementation

type JSONProcessorRPC struct {
	Impl *JSONProcessor
}

func (rpc *JSONProcessorRPC) Initialize(args map[string]interface{}, resp *error) error {
	*resp = rpc.Impl.Initialize(context.Background(), args)
	return nil
}

func (rpc *JSONProcessorRPC) Execute(args map[string]interface{}, resp *map[string]interface{}) error {
	result, err := rpc.Impl.Execute(context.Background(), args)
	if err != nil {
		return err
	}
	*resp = result
	return nil
}

func (rpc *JSONProcessorRPC) GetInfo(_ interface{}, resp *plugin.PluginInfo) error {
	*resp = rpc.Impl.GetInfo()
	return nil
}

func (rpc *JSONProcessorRPC) HealthCheck(args interface{}, resp *error) error {
	*resp = rpc.Impl.HealthCheck(context.Background())
	return nil
}

func (rpc *JSONProcessorRPC) Cleanup(args interface{}, resp *error) error {
	*resp = rpc.Impl.Cleanup()
	return nil
}

// Plugin implementation for HashiCorp go-plugin
type JSONProcessorPlugin struct{}

func (JSONProcessorPlugin) Server(*hashicorpPlugin.MuxBroker) (interface{}, error) {
	return &JSONProcessorRPC{Impl: NewJSONProcessor()}, nil
}

func (JSONProcessorPlugin) Client(b *hashicorpPlugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &JSONProcessorRPC{}, nil
}

// Main entry point
// DRY PRINCIPLE: Uses shared PluginMainUtilities to eliminate 26-line duplication (mass=110)
func main() {
	config := plugin.PluginMainConfig{
		PluginName: "json-processor",
		LogPrefix:  "[json-processor-plugin] ",
		StartMsg:   "Starting JSON processor plugin...",
		StopMsg:    "JSON processor plugin stopped",
	}

	// Use shared main function eliminating duplication
	plugin.RunPluginMain(config, func() hashicorpPlugin.Plugin {
		return &JSONProcessorPlugin{}
	})
}

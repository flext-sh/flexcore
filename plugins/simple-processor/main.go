// Simple Processor Plugin - REAL Functional Implementation
package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"log"
	"net/rpc"
	"os"
	"time"

	"github.com/hashicorp/go-plugin"
)

// init registers types for gob encoding/decoding
func init() {
	// Register types for RPC serialization
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register([]map[string]interface{}{})

	// Register primitive types
	gob.Register(string(""))
	gob.Register(int(0))
	gob.Register(int64(0))
	gob.Register(float64(0))
	gob.Register(bool(false))
	gob.Register(time.Time{})

	// Register plugin types
	gob.Register(PluginInfo{})
	gob.Register(ProcessingStats{})
}

// SimpleProcessor implements a basic data processing plugin
type SimpleProcessor struct {
	config map[string]interface{}
	stats  ProcessingStats
}

// ProcessingStats tracks plugin performance
type ProcessingStats struct {
	RecordsProcessed int64     `json:"records_processed"`
	RecordsFiltered  int64     `json:"records_filtered"`
	ProcessingTimeMs int64     `json:"processing_time_ms"`
	LastProcessedAt  time.Time `json:"last_processed_at"`
}

// PluginInfo contains plugin metadata
type PluginInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Type        string   `json:"type"`
	Capabilities []string `json:"capabilities"`
}

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
	log.Printf("[SimpleProcessor] Initializing with config: %+v", config)
	sp.config = config
	sp.stats = ProcessingStats{}
	return nil
}

// Execute processes data
func (sp *SimpleProcessor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	startTime := time.Now()
	defer func() {
		sp.stats.ProcessingTimeMs = time.Since(startTime).Milliseconds()
		sp.stats.LastProcessedAt = time.Now()
	}()

	log.Printf("[SimpleProcessor] Processing input with %d keys", len(input))

	result := make(map[string]interface{})
	result["processor"] = "simple-processor"
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
		"records_processed":   sp.stats.RecordsProcessed,
		"records_filtered":    sp.stats.RecordsFiltered,
		"processing_time_ms":  sp.stats.ProcessingTimeMs,
		"last_processed_at":   sp.stats.LastProcessedAt.Unix(),
	}

	sp.stats.RecordsProcessed++
	return result, nil
}

// GetInfo returns plugin metadata
func (sp *SimpleProcessor) GetInfo() PluginInfo {
	return PluginInfo{
		Name:        "simple-processor",
		Version:     "1.0.0",
		Description: "Simple data processing plugin for FlexCore testing",
		Author:      "FlexCore Team",
		Type:        "processor",
		Capabilities: []string{
			"filter",
			"transform",
			"enrich",
		},
	}
}

// HealthCheck verifies plugin health
func (sp *SimpleProcessor) HealthCheck(ctx context.Context) error {
	log.Printf("[SimpleProcessor] Health check - processed %d records", sp.stats.RecordsProcessed)
	return nil
}

// Cleanup releases resources
func (sp *SimpleProcessor) Cleanup() error {
	log.Printf("[SimpleProcessor] Cleanup called - processed %d records total", sp.stats.RecordsProcessed)

	// Save stats to file (optional)
	if statsFile, ok := sp.config["stats_file"].(string); ok {
		data, _ := json.MarshalIndent(sp.stats, "", "  ")
		os.WriteFile(statsFile, data, 0644)
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
			itemMap["_processor"] = "simple-processor"
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
	processed["_processor"] = "simple-processor"

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

func (SimpleProcessorPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &SimpleProcessorRPC{Impl: &SimpleProcessor{}}, nil
}

func (SimpleProcessorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &SimpleProcessorRPC{}, nil
}

// Main entry point
func main() {
	log.SetPrefix("[simple-processor-plugin] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	log.Println("Starting simple processor plugin...")

	// Handshake configuration
	handshakeConfig := plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "FLEXCORE_PLUGIN",
		MagicCookieValue: "flexcore-plugin-magic-cookie",
	}

	// Plugin map
	pluginMap := map[string]plugin.Plugin{
		"flexcore": &SimpleProcessorPlugin{},
	}

	// Serve the plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}

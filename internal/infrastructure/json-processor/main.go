// JSON Processor Plugin - REAL Implementation for JSON transformation
package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"log"
	"net/rpc"
	"os"
	"strings"
	"time"

	"github.com/flext/flexcore/pkg/plugin"
	hashicorpPlugin "github.com/hashicorp/go-plugin"
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

// JSONProcessor implements a JSON data processing plugin
type JSONProcessor struct {
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
func (jp *JSONProcessor) Initialize(ctx context.Context, config map[string]interface{}) error {
	log.Printf("[JSONProcessor] Initializing with config: %+v", config)
	jp.config = config
	jp.stats = ProcessingStats{
		StartTime: time.Now(),
	}
	return nil
}

// Execute processes JSON data
func (jp *JSONProcessor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	startTime := time.Now()
	defer func() {
		jp.stats.DurationMs = time.Since(startTime).Milliseconds()
		jp.stats.EndTime = time.Now()
	}()

	log.Printf("[JSONProcessor] Processing input with %d keys", len(input))

	result := make(map[string]interface{})
	result["processor"] = "json-processor"
	result["timestamp"] = time.Now().Unix()
	result["processed_by"] = "FlexCore JSONProcessor v1.0"

	// Process the input data
	if data, ok := input["data"]; ok {
		transformed, err := jp.transformData(data)
		if err != nil {
			result["error"] = err.Error()
			result["data"] = data
		} else {
			result["data"] = transformed
			jp.stats.ProcessedOK++
		}
	} else if jsonStr, ok := input["json_string"].(string); ok {
		// Parse JSON string
		parsed, err := jp.parseJSONString(jsonStr)
		if err != nil {
			result["error"] = err.Error()
			result["data"] = jsonStr
		} else {
			result["data"] = parsed
			jp.stats.ProcessedOK++
		}
	} else {
		// Process raw input
		result["data"] = jp.processRawData(input)
	}

	// Add processing stats
	result["stats"] = map[string]interface{}{
		"total_records":   jp.stats.TotalRecords,
		"processed_ok":    jp.stats.ProcessedOK,
		"processed_error": jp.stats.ProcessedError,
		"duration_ms":     jp.stats.DurationMs,
		"records_per_sec": jp.stats.RecordsPerSec,
	}

	jp.stats.TotalRecords++
	jp.stats.ProcessedOK++
	return result, nil
}

// transformData applies JSON transformations
func (jp *JSONProcessor) transformData(data interface{}) (interface{}, error) {
	operation := "prettify"
	if op, ok := jp.config["operation"].(string); ok {
		operation = op
	}

	switch operation {
	case "prettify":
		return jp.prettifyJSON(data)
	case "minify":
		return jp.minifyJSON(data)
	case "validate":
		return jp.validateJSON(data)
	case "transform":
		return jp.applyTransformations(data)
	default:
		return data, nil
	}
}

// prettifyJSON formats JSON data nicely
func (jp *JSONProcessor) prettifyJSON(data interface{}) (interface{}, error) {
	// Convert to JSON and back for pretty formatting
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return data, err
	}

	return map[string]interface{}{
		"pretty_json": string(jsonBytes),
		"original":    data,
		"formatted":   true,
	}, nil
}

// minifyJSON compacts JSON data
func (jp *JSONProcessor) minifyJSON(data interface{}) (interface{}, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return data, err
	}

	return map[string]interface{}{
		"minified_json": string(jsonBytes),
		"original":      data,
		"size_bytes":    len(jsonBytes),
	}, nil
}

// validateJSON checks if data is valid JSON
func (jp *JSONProcessor) validateJSON(data interface{}) (interface{}, error) {
	_, err := json.Marshal(data)

	return map[string]interface{}{
		"is_valid": err == nil,
		"data":     data,
		"error": func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
	}, nil
}

// applyTransformations applies custom transformations
func (jp *JSONProcessor) applyTransformations(data interface{}) (interface{}, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		return jp.transformMap(v), nil
	case []interface{}:
		return jp.transformArray(v), nil
	case string:
		return jp.transformString(v), nil
	default:
		return data, nil
	}
}

// transformMap applies transformations to a map
func (jp *JSONProcessor) transformMap(data map[string]interface{}) map[string]interface{} {
	transformed := make(map[string]interface{})

	for key, value := range data {
		// Transform key to snake_case if configured
		if jp.config["snake_case_keys"] == true {
			key = jp.toSnakeCase(key)
		}

		// Transform value recursively
		switch v := value.(type) {
		case map[string]interface{}:
			transformed[key] = jp.transformMap(v)
		case []interface{}:
			transformed[key] = jp.transformArray(v)
		case string:
			transformed[key] = jp.transformString(v)
		default:
			transformed[key] = value
		}
	}

	// Add metadata
	transformed["_json_processed"] = true
	transformed["_processed_at"] = time.Now().Unix()

	return transformed
}

// transformArray applies transformations to an array
func (jp *JSONProcessor) transformArray(data []interface{}) []interface{} {
	transformed := make([]interface{}, len(data))

	for i, item := range data {
		switch v := item.(type) {
		case map[string]interface{}:
			transformed[i] = jp.transformMap(v)
		case []interface{}:
			transformed[i] = jp.transformArray(v)
		case string:
			transformed[i] = jp.transformString(v)
		default:
			transformed[i] = item
		}
	}

	return transformed
}

// transformString applies string transformations
func (jp *JSONProcessor) transformString(data string) string {
	if jp.config["trim_strings"] == true {
		data = strings.TrimSpace(data)
	}

	if jp.config["lowercase_strings"] == true {
		data = strings.ToLower(data)
	}

	return data
}

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

// toSnakeCase converts camelCase to snake_case
func (jp *JSONProcessor) toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && (r >= 'A' && r <= 'Z') {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// GetInfo returns plugin metadata
func (jp *JSONProcessor) GetInfo() PluginInfo {
	return PluginInfo{
		ID:          "json-processor",
		Name:        "json-processor",
		Version:     "1.0.0",
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
	log.Printf("[JSONProcessor] Health check - processed %d records", jp.stats.TotalRecords)
	return nil
}

// Cleanup releases resources
func (jp *JSONProcessor) Cleanup() error {
	log.Printf("[JSONProcessor] Cleanup called - processed %d records total", jp.stats.TotalRecords)

	// Save statistics to file (optional)
	if statsFile, ok := jp.config["stats_file"].(string); ok {
		data, _ := json.MarshalIndent(jp.stats, "", "  ")
		os.WriteFile(statsFile, data, 0644)
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

func (rpc *JSONProcessorRPC) GetInfo(args interface{}, resp *PluginInfo) error {
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
	return &JSONProcessorRPC{Impl: &JSONProcessor{}}, nil
}

func (JSONProcessorPlugin) Client(b *hashicorpPlugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &JSONProcessorRPC{}, nil
}

// Main entry point
func main() {
	log.SetPrefix("[json-processor-plugin] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	log.Println("Starting JSON processor plugin...")

	// Handshake configuration
	handshakeConfig := hashicorpPlugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "FLEXCORE_PLUGIN",
		MagicCookieValue: "flexcore-plugin-magic-cookie",
	}

	// Plugin map
	pluginMap := map[string]hashicorpPlugin.Plugin{
		"flexcore": &JSONProcessorPlugin{},
	}

	// Serve the plugin
	hashicorpPlugin.Serve(&hashicorpPlugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}

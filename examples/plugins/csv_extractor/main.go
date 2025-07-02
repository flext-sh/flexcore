// Package main implements a CSV extractor plugin
package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/rpc"
	"os"
	"strings"

	"github.com/flext/flexcore/infrastructure/plugins"
	"github.com/flext/flexcore/shared/result"
	"github.com/hashicorp/go-plugin"
)

// CSVExtractor implements the ExtractorPlugin interface
type CSVExtractor struct {
	delimiter rune
	hasHeader bool
}

// Init initializes the CSV extractor
func (e *CSVExtractor) Init(config map[string]interface{}) error {
	// Set default delimiter
	e.delimiter = ','
	e.hasHeader = true

	// Parse config
	if delim, ok := config["delimiter"].(string); ok && len(delim) > 0 {
		e.delimiter = rune(delim[0])
	}
	if header, ok := config["has_header"].(bool); ok {
		e.hasHeader = header
	}

	return nil
}

// Execute executes the extraction (same as Extract for compatibility)
func (e *CSVExtractor) Execute(ctx context.Context, input interface{}) result.Result[interface{}] {
	if source, ok := input.(string); ok {
		extractResult := e.Extract(ctx, source)
		if extractResult.IsSuccess() {
			return result.Success[interface{}](extractResult.Value())
		}
		return result.Failure[interface{}](extractResult.Error())
	}
	return result.Failure[interface{}](fmt.Errorf("input must be a string (file path)"))
}

// Extract extracts data from CSV file
func (e *CSVExtractor) Extract(ctx context.Context, source string) result.Result[[]interface{}] {
	file, err := os.Open(source)
	if err != nil {
		return result.Failure[[]interface{}](fmt.Errorf("failed to open CSV file: %w", err))
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = e.delimiter

	records, err := reader.ReadAll()
	if err != nil {
		return result.Failure[[]interface{}](fmt.Errorf("failed to read CSV: %w", err))
	}

	if len(records) == 0 {
		return result.Success([]interface{}{})
	}

	var data []interface{}
	var headers []string

	if e.hasHeader && len(records) > 0 {
		headers = records[0]
		records = records[1:]
	}

	for _, record := range records {
		if e.hasHeader {
			row := make(map[string]interface{})
			for i, value := range record {
				if i < len(headers) {
					row[headers[i]] = strings.TrimSpace(value)
				}
			}
			data = append(data, row)
		} else {
			row := make([]interface{}, len(record))
			for i, value := range record {
				row[i] = strings.TrimSpace(value)
			}
			data = append(data, row)
		}
	}

	return result.Success(data)
}

// GetSchema returns the schema for CSV files
func (e *CSVExtractor) GetSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "tabular",
		"format": "csv",
		"supports_header": true,
		"configurable_delimiter": true,
		"output_format": map[string]interface{}{
			"with_header": "array of objects with string keys",
			"without_header": "array of arrays",
		},
	}
}

// GetMetadata returns plugin metadata
func (e *CSVExtractor) GetMetadata() plugins.PluginMetadata {
	return plugins.PluginMetadata{
		Name:        "csv-extractor",
		Version:     "1.0.0",
		Description: "Extracts data from CSV files with configurable delimiters",
		Author:      "FlexCore Team",
		Type:        "extractor",
		Capabilities: []string{
			"file_extraction",
			"csv_parsing",
			"configurable_delimiter",
			"header_detection",
		},
	}
}

// Cleanup cleans up resources
func (e *CSVExtractor) Cleanup() error {
	// Nothing to clean up for this plugin
	return nil
}

// Shutdown cleans up resources (kept for backward compatibility)
func (e *CSVExtractor) Shutdown() error {
	return e.Cleanup()
}

// This is the implementation of plugin.Plugin so we can serve/consume this
type CSVExtractorPlugin struct {
	// Impl Injection
	Impl plugins.ExtractorPlugin
}

func (p *CSVExtractorPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &plugins.PluginRPC{Impl: p.Impl}, nil
}

func (p *CSVExtractorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return plugins.NewPluginRPCClient(c), nil
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "FLEXCORE_PLUGIN",
	MagicCookieValue: "flexcore_plugin_v1",
}

// pluginMap is the map of plugins we can dispense.
var pluginMap = map[string]plugin.Plugin{
	"plugin": &CSVExtractorPlugin{},
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"plugin": &CSVExtractorPlugin{Impl: &CSVExtractor{}},
		},
	})
}
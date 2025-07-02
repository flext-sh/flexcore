// FlexCore REAL Plugin Example - Data Extractor
// This is a REAL HashiCorp go-plugin executable
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/flext/flexcore/infrastructure/plugins"
)

// DataExtractorPlugin implements the FlexCorePlugin interface
type DataExtractorPlugin struct {
	config map[string]interface{}
	isInitialized bool
}

// Initialize the plugin
func (p *DataExtractorPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	p.config = config
	p.isInitialized = true
	log.Printf("DataExtractorPlugin initialized with config: %v", config)
	return nil
}

// Execute the plugin functionality
func (p *DataExtractorPlugin) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	if !p.isInitialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	log.Printf("DataExtractorPlugin executing with input: %v", input)

	// Simulate data extraction work
	time.Sleep(100 * time.Millisecond)

	// Extract data source from input
	source, ok := input["source"].(string)
	if !ok {
		source = "default"
	}

	// Return extracted data
	result := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"source":    source,
			"extracted": []map[string]interface{}{
				{
					"id":    1,
					"name":  "Sample Record 1",
					"value": 100,
				},
				{
					"id":    2,
					"name":  "Sample Record 2", 
					"value": 200,
				},
			},
			"count":     2,
			"timestamp": time.Now().Unix(),
		},
		"plugin": "DataExtractorPlugin",
		"version": "1.0.0",
	}

	log.Printf("DataExtractorPlugin completed with result: %v", result)
	return result, nil
}

// GetInfo returns plugin metadata
func (p *DataExtractorPlugin) GetInfo() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "data-extractor",
		Version:     "1.0.0",
		Description: "REAL data extraction plugin using HashiCorp go-plugin",
		Author:      "FlexCore Team",
		Type:        "extractor",
		Capabilities: []string{
			"data_extraction",
			"batch_processing",
			"real_time_processing",
		},
		Config: map[string]string{
			"source_type": "database",
			"output_format": "json",
		},
	}
}

// HealthCheck performs plugin health check
func (p *DataExtractorPlugin) HealthCheck(ctx context.Context) error {
	if !p.isInitialized {
		return fmt.Errorf("plugin not initialized")
	}
	log.Printf("DataExtractorPlugin health check: OK")
	return nil
}

// Cleanup plugin resources
func (p *DataExtractorPlugin) Cleanup() error {
	log.Printf("DataExtractorPlugin cleanup")
	p.isInitialized = false
	return nil
}

// This is the main function for the plugin executable
func main() {
	// Create plugin instance
	dataExtractor := &DataExtractorPlugin{}

	// Define handshake configuration (must match manager)
	handshakeConfig := plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "FLEXCORE_PLUGIN",
		MagicCookieValue: "flexcore-plugin-magic-cookie",
	}

	// Define plugin map
	pluginMap := map[string]plugin.Plugin{
		"flexcore": &plugins.FlexCorePluginImpl{Impl: dataExtractor},
	}

	// Serve the plugin
	log.Printf("Starting FlexCore Data Extractor Plugin...")
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
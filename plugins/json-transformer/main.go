// JSON Transformer Plugin - Real executable plugin using HashiCorp go-plugin
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/flext/flexcore/infrastructure/plugins"
	"github.com/flext/flexcore/shared/result"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// JSONTransformer implements the TransformerPlugin interface
type JSONTransformer struct {
	logger          hclog.Logger
	transformations []string
	config          map[string]interface{}
}

// Init initializes the plugin with configuration
func (t *JSONTransformer) Init(config map[string]interface{}) error {
	t.logger.Info("Initializing JSON transformer", "config", config)
	
	t.config = config
	t.transformations = []string{
		"clean",
		"normalize",
		"validate",
		"convert-types",
		"map-fields",
		"filter",
		"aggregate",
		"extract-fields",
	}
	
	t.logger.Info("JSON transformer initialized successfully")
	return nil
}

// Execute executes the plugin with given input (generic interface method)
func (t *JSONTransformer) Execute(ctx context.Context, input interface{}) result.Result[interface{}] {
	t.logger.Debug("Execute called", "input", input)
	
	// Use Transform method
	transformResult := t.Transform(ctx, input)
	if transformResult.IsFailure() {
		return result.Failure[interface{}](transformResult.Error())
	}

	return result.Success[interface{}](transformResult.Value())
}

// GetMetadata returns plugin metadata
func (t *JSONTransformer) GetMetadata() plugins.PluginMetadata {
	return plugins.PluginMetadata{
		Name:        "json-transformer",
		Version:     "1.0.0",
		Description: "JSON data transformer plugin with multiple transformation capabilities",
		Author:      "FlexCore",
		Type:        plugins.TransformerType,
		Capabilities: []string{
			"clean-data",
			"normalize-fields",
			"validate-schema",
			"type-conversion",
			"field-mapping",
			"data-filtering",
			"aggregation",
			"field-extraction",
		},
	}
}

// Shutdown gracefully shuts down the plugin
func (t *JSONTransformer) Shutdown() error {
	t.logger.Info("Shutting down JSON transformer")
	return nil
}

// GetInfo returns plugin information
func (t *JSONTransformer) GetInfo() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "json-transformer",
		Version:     "1.0.0",
		Description: "JSON data transformer plugin with multiple transformation capabilities",
		Type:        plugins.TransformerType,
		Capabilities: []string{
			"clean-data",
			"normalize-fields",
			"validate-schema",
			"type-conversion",
			"field-mapping",
			"data-filtering",
			"aggregation",
			"field-extraction",
		},
	}
}

// Configure configures the plugin with transformation parameters
func (t *JSONTransformer) Configure(config map[string]interface{}) result.Result[bool] {
	t.logger.Info("Configuring JSON transformer", "config", config)

	t.config = config

	// Get transformations list
	if transformsInterface, exists := config["transformations"]; exists {
		if transforms, ok := transformsInterface.([]interface{}); ok {
			t.transformations = make([]string, len(transforms))
			for i, transform := range transforms {
				if transformStr, ok := transform.(string); ok {
					t.transformations[i] = transformStr
				}
			}
		} else if transforms, ok := transformsInterface.([]string); ok {
			t.transformations = transforms
		}
	}

	if len(t.transformations) == 0 {
		t.transformations = []string{"clean", "normalize", "validate"}
	}

	t.logger.Info("Configured transformations", "transformations", t.transformations)
	return result.Success(true)
}

// Transform transforms the input data according to configured transformations
func (t *JSONTransformer) Transform(ctx context.Context, data interface{}) result.Result[interface{}] {
	t.logger.Info("Starting data transformation", "transformations", t.transformations)

	// Convert data to JSON-compatible format
	jsonData, err := t.normalizeToJSON(data)
	if err != nil {
		return result.Failure[interface{}](fmt.Errorf("failed to normalize data: %w", err))
	}

	// Apply each transformation in sequence
	transformedData := jsonData
	for _, transformation := range t.transformations {
		var err error
		transformedData, err = t.applyTransformation(transformation, transformedData)
		if err != nil {
			t.logger.Error("Transformation failed", "transformation", transformation, "error", err)
			return result.Failure[interface{}](fmt.Errorf("transformation '%s' failed: %w", transformation, err))
		}
		t.logger.Debug("Applied transformation", "transformation", transformation)
	}

	t.logger.Info("Data transformation completed successfully")
	return result.Success(transformedData)
}

// GetTransformations returns available transformations
func (t *JSONTransformer) GetTransformations() []string {
	return []string{
		"clean",        // Remove null values, trim strings
		"normalize",    // Normalize field names and values
		"validate",     // Validate against schema
		"convert",      // Type conversions
		"map",          // Field mapping
		"filter",       // Data filtering
		"aggregate",    // Data aggregation
		"extract",      // Extract nested fields
		"flatten",      // Flatten nested objects
		"enrich",       // Add computed fields
	}
}

// Health checks plugin health
func (t *JSONTransformer) Health() result.Result[bool] {
	// Simple health check - verify we can process sample data
	sampleData := map[string]interface{}{
		"test": "value",
		"null_field": nil,
	}

	_, err := t.applyTransformation("clean", sampleData)
	if err != nil {
		return result.Failure[bool](fmt.Errorf("health check failed: %w", err))
	}

	return result.Success(true)
}


// Private transformation methods

func (t *JSONTransformer) normalizeToJSON(data interface{}) (interface{}, error) {
	// Convert to JSON and back to ensure proper format
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return result, nil
}

func (t *JSONTransformer) applyTransformation(transformation string, data interface{}) (interface{}, error) {
	switch transformation {
	case "clean":
		return t.transformClean(data)
	case "normalize":
		return t.transformNormalize(data)
	case "validate":
		return t.transformValidate(data)
	case "convert":
		return t.transformConvert(data)
	case "map":
		return t.transformMap(data)
	case "filter":
		return t.transformFilter(data)
	case "aggregate":
		return t.transformAggregate(data)
	case "extract":
		return t.transformExtract(data)
	case "flatten":
		return t.transformFlatten(data)
	case "enrich":
		return t.transformEnrich(data)
	default:
		return data, fmt.Errorf("unknown transformation: %s", transformation)
	}
}

func (t *JSONTransformer) transformClean(data interface{}) (interface{}, error) {
	return t.cleanValue(data), nil
}

func (t *JSONTransformer) cleanValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case map[string]interface{}:
		cleaned := make(map[string]interface{})
		for key, val := range v {
			cleanedVal := t.cleanValue(val)
			if cleanedVal != nil {
				cleaned[key] = cleanedVal
			}
		}
		return cleaned
	case []interface{}:
		var cleaned []interface{}
		for _, val := range v {
			cleanedVal := t.cleanValue(val)
			if cleanedVal != nil {
				cleaned = append(cleaned, cleanedVal)
			}
		}
		return cleaned
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil
		}
		return trimmed
	default:
		return v
	}
}

func (t *JSONTransformer) transformNormalize(data interface{}) (interface{}, error) {
	return t.normalizeValue(data), nil
}

func (t *JSONTransformer) normalizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		normalized := make(map[string]interface{})
		for key, val := range v {
			normalizedKey := t.normalizeFieldName(key)
			normalized[normalizedKey] = t.normalizeValue(val)
		}
		return normalized
	case []interface{}:
		var normalized []interface{}
		for _, val := range v {
			normalized = append(normalized, t.normalizeValue(val))
		}
		return normalized
	case string:
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return v
	}
}

func (t *JSONTransformer) normalizeFieldName(fieldName string) string {
	// Convert to lowercase and replace spaces/special chars with underscores
	normalized := strings.ToLower(fieldName)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	normalized = re.ReplaceAllString(normalized, "_")
	normalized = strings.Trim(normalized, "_")
	return normalized
}

func (t *JSONTransformer) transformValidate(data interface{}) (interface{}, error) {
	// Get required fields from config
	requiredFields, _ := t.config["required_fields"].([]interface{})
	
	switch v := data.(type) {
	case map[string]interface{}:
		for _, field := range requiredFields {
			if fieldStr, ok := field.(string); ok {
				if _, exists := v[fieldStr]; !exists {
					return nil, fmt.Errorf("required field missing: %s", fieldStr)
				}
			}
		}
		return v, nil
	case []interface{}:
		var validated []interface{}
		for _, item := range v {
			if validatedItem, err := t.transformValidate(item); err == nil {
				validated = append(validated, validatedItem)
			}
		}
		return validated, nil
	default:
		return data, nil
	}
}

func (t *JSONTransformer) transformConvert(data interface{}) (interface{}, error) {
	return t.convertTypes(data), nil
}

func (t *JSONTransformer) convertTypes(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		converted := make(map[string]interface{})
		for key, val := range v {
			converted[key] = t.convertTypes(val)
		}
		return converted
	case []interface{}:
		var converted []interface{}
		for _, val := range v {
			converted = append(converted, t.convertTypes(val))
		}
		return converted
	case string:
		// Try to convert string to appropriate type
		if v == "" {
			return v
		}
		
		// Try boolean
		if v == "true" || v == "false" {
			return v == "true"
		}
		
		// Try integer
		if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
			return intVal
		}
		
		// Try float
		if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
			return floatVal
		}
		
		// Try timestamp
		if timeVal, err := time.Parse(time.RFC3339, v); err == nil {
			return timeVal.Unix()
		}
		
		return v
	default:
		return v
	}
}

func (t *JSONTransformer) transformMap(data interface{}) (interface{}, error) {
	mapping, _ := t.config["field_mapping"].(map[string]interface{})
	if len(mapping) == 0 {
		return data, nil
	}

	switch v := data.(type) {
	case map[string]interface{}:
		mapped := make(map[string]interface{})
		for key, val := range v {
			newKey := key
			if mappedKey, exists := mapping[key]; exists {
				if mappedKeyStr, ok := mappedKey.(string); ok {
					newKey = mappedKeyStr
				}
			}
			mapped[newKey] = val
		}
		return mapped, nil
	default:
		return data, nil
	}
}

func (t *JSONTransformer) transformFilter(data interface{}) (interface{}, error) {
	filterConfig, _ := t.config["filter"].(map[string]interface{})
	if len(filterConfig) == 0 {
		return data, nil
	}

	switch v := data.(type) {
	case []interface{}:
		var filtered []interface{}
		for _, item := range v {
			if t.matchesFilter(item, filterConfig) {
				filtered = append(filtered, item)
			}
		}
		return filtered, nil
	default:
		if t.matchesFilter(data, filterConfig) {
			return data, nil
		}
		return nil, nil
	}
}

func (t *JSONTransformer) matchesFilter(item interface{}, filter map[string]interface{}) bool {
	itemMap, ok := item.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expectedValue := range filter {
		if actualValue, exists := itemMap[field]; !exists || !reflect.DeepEqual(actualValue, expectedValue) {
			return false
		}
	}
	return true
}

func (t *JSONTransformer) transformAggregate(data interface{}) (interface{}, error) {
	// Simple aggregation - count items
	switch v := data.(type) {
	case []interface{}:
		return map[string]interface{}{
			"count": len(v),
			"items": v,
		}, nil
	default:
		return data, nil
	}
}

func (t *JSONTransformer) transformExtract(data interface{}) (interface{}, error) {
	extractFields, _ := t.config["extract_fields"].([]interface{})
	if len(extractFields) == 0 {
		return data, nil
	}

	switch v := data.(type) {
	case map[string]interface{}:
		extracted := make(map[string]interface{})
		for _, field := range extractFields {
			if fieldStr, ok := field.(string); ok {
				if value, exists := v[fieldStr]; exists {
					extracted[fieldStr] = value
				}
			}
		}
		return extracted, nil
	default:
		return data, nil
	}
}

func (t *JSONTransformer) transformFlatten(data interface{}) (interface{}, error) {
	return t.flattenValue(data, ""), nil
}

func (t *JSONTransformer) flattenValue(value interface{}, prefix string) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		flattened := make(map[string]interface{})
		for key, val := range v {
			newKey := key
			if prefix != "" {
				newKey = prefix + "_" + key
			}
			
			if nested, ok := val.(map[string]interface{}); ok {
				flatNested := t.flattenValue(nested, newKey).(map[string]interface{})
				for flatKey, flatVal := range flatNested {
					flattened[flatKey] = flatVal
				}
			} else {
				flattened[newKey] = val
			}
		}
		return flattened
	default:
		return value
	}
}

func (t *JSONTransformer) transformEnrich(data interface{}) (interface{}, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		enriched := make(map[string]interface{})
		for key, val := range v {
			enriched[key] = val
		}
		// Add computed fields
		enriched["_processed_at"] = time.Now().Unix()
		enriched["_transformer"] = "json-transformer"
		return enriched, nil
	default:
		return data, nil
	}
}

// Plugin handshake and serve
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "FLEXCORE_PLUGIN",
	MagicCookieValue: "json-transformer",
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	transformer := &JSONTransformer{
		logger: logger,
	}

	var pluginMap = map[string]plugin.Plugin{
		"transformer": &plugins.TransformerGRPCPlugin{Impl: transformer},
	}

	logger.Debug("Starting json-transformer plugin")

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}